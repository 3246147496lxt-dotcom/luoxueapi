package service

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

type skillZIPTestEntry struct {
	name string
	data []byte
	mode os.FileMode
}

func buildSkillTestZIP(t *testing.T, entries ...skillZIPTestEntry) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for _, entry := range entries {
		header := &zip.FileHeader{Name: entry.name, Method: zip.Deflate}
		if entry.mode != 0 {
			header.SetMode(entry.mode)
		}
		file, err := writer.CreateHeader(header)
		require.NoError(t, err)
		_, err = file.Write(entry.data)
		require.NoError(t, err)
	}
	require.NoError(t, writer.Close())
	return buffer.Bytes()
}

func validSkillMD(name string) []byte {
	return []byte("---\nname: " + name + "\ndescription: A useful test skill.\n---\n\n# Instructions\n\nDo the safe thing.\n")
}

func requireSkillArchiveCode(t *testing.T, err error, code string) {
	t.Helper()
	var validationErr *SkillArchiveValidationError
	require.ErrorAs(t, err, &validationErr)
	require.False(t, validationErr.Report.Valid)
	require.NotEmpty(t, validationErr.Report.Errors)
	require.Equal(t, code, validationErr.Report.Errors[0].Code)
}

func TestValidateSkillArchiveHappyPathNormalizesAndStripsFrontmatter(t *testing.T) {
	raw := buildSkillTestZIP(t,
		skillZIPTestEntry{name: "demo-skill/SKILL.md", data: validSkillMD("demo-skill")},
		skillZIPTestEntry{name: "demo-skill/scripts/check.sh", data: []byte("#!/bin/sh\necho ok\n"), mode: 0o755},
	)

	first, err := ValidateSkillArchive(raw, "demo-skill")
	require.NoError(t, err)
	second, err := ValidateSkillArchive(raw, "demo-skill")
	require.NoError(t, err)
	require.True(t, first.ValidationReport.Valid)
	require.Empty(t, first.ValidationReport.Errors)
	require.Equal(t, "demo-skill", first.ManifestName)
	require.Equal(t, "# Instructions\n\nDo the safe thing.", first.SkillMD)
	require.Equal(t, first.SHA256, second.SHA256)
	require.Equal(t, first.PackageData, second.PackageData)
	require.Len(t, first.FileManifest, 2)
	require.NotEmpty(t, first.ValidationReport.Warnings)
}

func TestValidateSkillArchiveRejectsSecurityBoundaries(t *testing.T) {
	tests := []struct {
		name  string
		entry skillZIPTestEntry
		code  string
		slug  string
	}{
		{name: "traversal", entry: skillZIPTestEntry{name: "demo-skill/../evil.txt", data: []byte("x")}, code: "UNSAFE_PATH", slug: "demo-skill"},
		{name: "symlink", entry: skillZIPTestEntry{name: "demo-skill/link", data: []byte("target"), mode: os.ModeSymlink | 0o777}, code: "SYMLINK", slug: "demo-skill"},
		{name: "binary", entry: skillZIPTestEntry{name: "demo-skill/blob.dat", data: []byte{0xff, 0x00, 0x01}}, code: "BINARY_FILE", slug: "demo-skill"},
		{name: "mac metadata", entry: skillZIPTestEntry{name: "demo-skill/__MACOSX/junk", data: []byte("x")}, code: "OS_METADATA", slug: "demo-skill"},
		{name: "nested archive", entry: skillZIPTestEntry{name: "demo-skill/payload.zip", data: []byte("not really a zip")}, code: "NESTED_ARCHIVE", slug: "demo-skill"},
		{name: "private key", entry: skillZIPTestEntry{name: "demo-skill/secret.txt", data: []byte("-----BEGIN ENCRYPTED PRIVATE KEY-----\nsecret")}, code: "PRIVATE_KEY", slug: "demo-skill"},
		{name: "wrong top level", entry: skillZIPTestEntry{name: "other/file.txt", data: []byte("x")}, code: "TOP_LEVEL_DIRECTORY", slug: "demo-skill"},
		{name: "non NFC", entry: skillZIPTestEntry{name: "demo-skill/cafe\u0301.txt", data: []byte("x")}, code: "UNICODE_NORMALIZATION", slug: "demo-skill"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := buildSkillTestZIP(t,
				skillZIPTestEntry{name: "demo-skill/SKILL.md", data: validSkillMD("demo-skill")},
				tt.entry,
			)
			_, err := ValidateSkillArchive(raw, tt.slug)
			requireSkillArchiveCode(t, err, tt.code)
		})
	}
}

func TestValidateSkillArchiveRejectsNameMismatchAndFileLimit(t *testing.T) {
	raw := buildSkillTestZIP(t, skillZIPTestEntry{name: "demo-skill/SKILL.md", data: validSkillMD("other-skill")})
	_, err := ValidateSkillArchive(raw, "demo-skill")
	requireSkillArchiveCode(t, err, "MANIFEST_NAME_MISMATCH")

	large := bytes.Repeat([]byte("a"), int(SkillArchiveMaxFileBytes)+1)
	raw = buildSkillTestZIP(t,
		skillZIPTestEntry{name: "demo-skill/SKILL.md", data: validSkillMD("demo-skill")},
		skillZIPTestEntry{name: "demo-skill/large.txt", data: large},
	)
	_, err = ValidateSkillArchive(raw, "demo-skill")
	requireSkillArchiveCode(t, err, "FILE_SIZE")
}

func TestSkillArchiveWarningsIgnoreDocumentationURLs(t *testing.T) {
	for _, name := range []string{
		"demo-skill/LICENSE",
		"demo-skill/LICENSE.txt",
		"demo-skill/NOTICE.md",
		"demo-skill/README.md",
		"demo-skill/references/guide.rst",
	} {
		t.Run(name, func(t *testing.T) {
			warnings := skillArchiveWarnings(name, []byte("See https://example.com/reference for details."), 0o644)
			for _, warning := range warnings {
				require.NotEqual(t, "NETWORK_REFERENCE", warning.Code)
			}
		})
	}
}

func TestSkillArchiveWarningsKeepActionableNetworkSignals(t *testing.T) {
	tests := []struct {
		name string
		data string
		mode os.FileMode
	}{
		{name: "demo-skill/scripts/install.sh", data: "curl https://example.com/install.sh", mode: 0o755},
		{name: "demo-skill/README.md", data: "Run: wget https://example.com/archive", mode: 0o644},
		{name: "demo-skill/config.yaml", data: "endpoint: https://example.com/api", mode: 0o644},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings := skillArchiveWarnings(tt.name, []byte(tt.data), tt.mode)
			codes := make([]string, 0, len(warnings))
			for _, warning := range warnings {
				codes = append(codes, warning.Code)
			}
			require.Contains(t, codes, "NETWORK_REFERENCE")
		})
	}
}

type skillGateSettingRepo struct {
	mu     sync.Mutex
	values map[string]string
	err    error
	calls  int
}

func (r *skillGateSettingRepo) Get(_ context.Context, key string) (*Setting, error) {
	value, err := r.GetValue(context.Background(), key)
	if err != nil {
		return nil, err
	}
	return &Setting{Key: key, Value: value}, nil
}
func (r *skillGateSettingRepo) GetValue(_ context.Context, key string) (string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return "", r.err
	}
	value, ok := r.values[key]
	if !ok {
		return "", ErrSettingNotFound
	}
	return value, nil
}
func (r *skillGateSettingRepo) Set(_ context.Context, key, value string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.values == nil {
		r.values = map[string]string{}
	}
	r.values[key] = value
	return r.err
}
func (r *skillGateSettingRepo) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.calls++
	if r.err != nil {
		return nil, r.err
	}
	out := make(map[string]string, len(keys))
	for _, key := range keys {
		out[key] = r.values[key]
	}
	return out, nil
}
func (r *skillGateSettingRepo) SetMultiple(_ context.Context, values map[string]string) error {
	for key, value := range values {
		if err := r.Set(context.Background(), key, value); err != nil {
			return err
		}
	}
	return nil
}
func (r *skillGateSettingRepo) GetAll(context.Context) (map[string]string, error) {
	return r.values, r.err
}
func (r *skillGateSettingRepo) Delete(_ context.Context, key string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.values, key)
	return nil
}

func TestSkillMarketplaceGateFailsClosedCachesAndInvalidates(t *testing.T) {
	repo := &skillGateSettingRepo{values: map[string]string{
		SettingKeySkillMarketplaceEnabled: "true",
		SettingKeyBackendModeEnabled:      "false",
	}}
	settings := NewSettingService(repo, &config.Config{})
	notifications := 0
	settings.SetOnUpdateCallback(func() { notifications++ })
	service := NewSkillMarketService(nil, settings)

	var wait sync.WaitGroup
	results := make(chan bool, 12)
	for range 12 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			results <- service.IsPublicEnabled(context.Background())
		}()
	}
	wait.Wait()
	close(results)
	for enabled := range results {
		require.True(t, enabled)
	}
	require.Equal(t, 1, repo.calls)

	config, err := service.UpdateConfig(context.Background(), false)
	require.NoError(t, err)
	require.False(t, config.Enabled)
	require.Equal(t, 1, notifications)
	require.False(t, service.IsPublicEnabled(context.Background()))
	require.Equal(t, 2, repo.calls)

	repo.mu.Lock()
	repo.values[SettingKeySkillMarketplaceEnabled] = "true"
	repo.values[SettingKeyBackendModeEnabled] = "true"
	repo.mu.Unlock()
	service.gateMu.Lock()
	service.gateExpires = service.gateExpires.Add(-10 * time.Second)
	service.gateMu.Unlock()
	require.False(t, service.IsPublicEnabled(context.Background()))

	repo.mu.Lock()
	repo.err = errors.New("settings unavailable")
	repo.mu.Unlock()
	service.gateMu.Lock()
	service.gateExpires = time.Time{}
	service.gateMu.Unlock()
	require.False(t, service.IsPublicEnabled(context.Background()))
}

func TestSkillArchiveErrorMetadataContainsFullReport(t *testing.T) {
	_, err := ValidateSkillArchive([]byte("not zip"), "demo-skill")
	var validationErr *SkillArchiveValidationError
	require.ErrorAs(t, err, &validationErr)
	appErr := infraerrors.FromError(err)
	require.Equal(t, int32(422), appErr.Code)
	require.Contains(t, appErr.Metadata["validation_report"], `"valid":false`)
	require.True(t, strings.Contains(appErr.Metadata["validation_report"], "INVALID_ZIP"))
}
