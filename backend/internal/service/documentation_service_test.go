package service

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestBundledDocumentationIsValidAndComplete(t *testing.T) {
	canonical, err := ValidateDocumentationContent(defaultDocumentationContent)
	require.NoError(t, err)
	var content DocumentationContent
	require.NoError(t, json.Unmarshal(canonical, &content))
	require.Equal(t, DocumentationSchemaVersion, content.SchemaVersion)
	require.Len(t, content.Tutorials, 5)
	require.Equal(t, []string{"quick-start", "desktop", "clients", "api", "recharge"}, []string{
		content.Tutorials[0].ID,
		content.Tutorials[1].ID,
		content.Tutorials[2].ID,
		content.Tutorials[3].ID,
		content.Tutorials[4].ID,
	})
}

func TestValidateDocumentationContentRejectsUnsafeOrIncompleteContent(t *testing.T) {
	valid := `{
      "schema_version":1,
      "tutorials":[{
        "id":"quick-start","tab_label":"新手教程","icon":"image","description":"创建密钥说明。",
        "steps":[{"title":"打开控制台","description":"进入控制台查看。","image":{"src":"/api/v1/public/documentation/assets/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa","alt":"控制台截图","caption":"操作位置"}}]
      }]
    }`
	_, err := ValidateDocumentationContent(json.RawMessage(valid))
	require.NoError(t, err)

	tests := map[string]string{
		"wrong schema": replaceOnce(valid, `"schema_version":1`, `"schema_version":2`),
		"duplicate id": `{
          "schema_version":1,
          "tutorials":[
            {"id":"quick-start","tab_label":"新手教程","icon":"key","description":"第一组说明。","steps":[{"title":"第一步","description":"第一步说明。"}]},
            {"id":"quick-start","tab_label":"重复教程","icon":"key","description":"第二组说明。","steps":[{"title":"第二步","description":"第二步说明。"}]}
          ]
        }`,
		"unsafe url":            replaceOnce(valid, `/api/v1/public/documentation/assets/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa`, `javascript:alert(1)`),
		"raw html":              replaceOnce(valid, `进入控制台查看。`, `<script>alert(1)</script>`),
		"english-only label":    replaceOnce(valid, `新手教程`, `Tutorial`),
		"unknown field":         replaceOnce(valid, `"description":"创建密钥说明。",`, `"description":"创建密钥说明。","html":"<b>x</b>",`),
		"unsupported note tone": replaceOnce(valid, `"image":{`, `"note":{"tone":"danger","text":"危险提示"},"image":{`),
	}
	for name, input := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := ValidateDocumentationContent(json.RawMessage(input))
			require.Error(t, err)
		})
	}
}

func TestDocumentationPublicSnapshotHasStableETagAndSeed(t *testing.T) {
	repo := &documentationRepositoryStub{}
	svc := NewDocumentationService(repo)

	first, firstETag, err := svc.PublicSnapshot(context.Background())
	require.NoError(t, err)
	second, secondETag, err := svc.PublicSnapshot(context.Background())
	require.NoError(t, err)
	require.Equal(t, int64(1), first.Version)
	require.JSONEq(t, string(first.Content), string(second.Content))
	require.Equal(t, firstETag, secondETag)
	require.Equal(t, 1, repo.ensureSeedCalls)
}

func TestIncompleteDraftCanBeSavedButCannotBePublished(t *testing.T) {
	repo := &documentationRepositoryStub{}
	svc := NewDocumentationService(repo)
	incomplete := json.RawMessage(`{"schema_version":1,"tutorials":[]}`)

	state, err := svc.SaveDraft(context.Background(), incomplete, nil)
	require.NoError(t, err)
	require.NotNil(t, state.Draft)
	require.JSONEq(t, string(incomplete), string(state.Draft.Content))

	_, err = svc.Publish(context.Background(), nil)
	require.ErrorIs(t, err, ErrDocumentationInvalid)
}

func TestCodeBlocksMayContainEscapedHTMLExamples(t *testing.T) {
	raw := json.RawMessage(`{
      "schema_version":1,
      "tutorials":[{
        "id":"html-example","tab_label":"代码教程","icon":"code","description":"展示安全代码示例。",
        "steps":[{"title":"查看代码","description":"代码只按文本展示。","code":{"label":"HTML","value":"<button type=\"button\">示例</button>"}}]
      }]
    }`)
	_, err := ValidateDocumentationContent(raw)
	require.NoError(t, err)
}

func TestDocumentationCustomSVGIconIsValidatedAndCanonicalized(t *testing.T) {
	valid := json.RawMessage(`{
      "schema_version":1,
      "tutorials":[{
        "id":"custom-icon","tab_label":"自定义图标","icon":"help",
        "icon_svg":"<svg xmlns=\"http://www.w3.org/2000/svg\" xmlns:xlink=\"http://www.w3.org/1999/xlink\" viewBox=\"0 0 24 24\"><defs><linearGradient id=\"ice\"><stop offset=\"0\" stop-color=\"#22d3ee\"/><stop offset=\"1\" stop-color=\"#2563eb\"/></linearGradient><path id=\"star\" d=\"M12 2L15 9H22L16 13L18 21L12 16L6 21L8 13L2 9H9Z\"/></defs><use xlink:href=\"#star\" fill=\"url(#ice)\"/></svg>",
        "description":"上传任意安全的 SVG 图标。",
        "steps":[{"title":"上传图标","description":"选择本地 SVG 文件。"}]
      }]
    }`)

	canonical, err := ValidateDocumentationContent(valid)
	require.NoError(t, err)
	var content DocumentationContent
	require.NoError(t, json.Unmarshal(canonical, &content))
	require.Contains(t, content.Tutorials[0].IconSVG, "linearGradient")
	require.Equal(t, "help", content.Tutorials[0].Icon)
}

func TestDocumentationCustomSVGIconRejectsActiveOrExternalContent(t *testing.T) {
	base := `{
      "schema_version":1,
      "tutorials":[{
        "id":"custom-icon","tab_label":"自定义图标","icon":"help","icon_svg":%s,
        "description":"上传任意安全的 SVG 图标。",
        "steps":[{"title":"上传图标","description":"选择本地 SVG 文件。"}]
      }]
    }`

	tests := map[string]string{
		"non svg root":     `<html></html>`,
		"doctype":          `<!DOCTYPE svg><svg xmlns="http://www.w3.org/2000/svg"></svg>`,
		"script":           `<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`,
		"event handler":    `<svg xmlns="http://www.w3.org/2000/svg" onload="alert(1)"></svg>`,
		"foreign object":   `<svg xmlns="http://www.w3.org/2000/svg"><foreignObject><div>HTML</div></foreignObject></svg>`,
		"external href":    `<svg xmlns="http://www.w3.org/2000/svg"><use href="https://evil.example/icon.svg#x"/></svg>`,
		"data reference":   `<svg xmlns="http://www.w3.org/2000/svg"><use href="data:image/svg+xml;base64,PHN2Zy8+"/></svg>`,
		"external css url": `<svg xmlns="http://www.w3.org/2000/svg"><path fill="url(https://evil.example/fill.svg)"/></svg>`,
		"inline style":     `<svg xmlns="http://www.w3.org/2000/svg"><path style="fill:red"/></svg>`,
		"animation":        `<svg xmlns="http://www.w3.org/2000/svg"><animate attributeName="href" values="#a;#b"/></svg>`,
	}
	for name, iconSVG := range tests {
		t.Run(name, func(t *testing.T) {
			raw, err := json.Marshal(iconSVG)
			require.NoError(t, err)
			_, err = ValidateDocumentationContent(json.RawMessage(fmt.Sprintf(base, raw)))
			require.ErrorIs(t, err, ErrDocumentationInvalid)
		})
	}
}

func replaceOnce(input, old, replacement string) string {
	result := []byte(input)
	needle := []byte(old)
	for i := 0; i+len(needle) <= len(result); i++ {
		match := true
		for j := range needle {
			if result[i+j] != needle[j] {
				match = false
				break
			}
		}
		if match {
			out := make([]byte, 0, len(result)-len(needle)+len(replacement))
			out = append(out, result[:i]...)
			out = append(out, replacement...)
			out = append(out, result[i+len(needle):]...)
			return string(out)
		}
	}
	return input
}

type documentationRepositoryStub struct {
	ensureSeedCalls int
	state           DocumentationAdminState
}

func (r *documentationRepositoryStub) EnsureSeed(_ context.Context, content json.RawMessage) error {
	r.ensureSeedCalls++
	if r.state.Published == nil {
		now := time.Unix(1_700_000_000, 0).UTC()
		r.state = DocumentationAdminState{
			Draft:     &DocumentationSnapshot{Content: append(json.RawMessage(nil), content...), Version: 1, UpdatedAt: now},
			Published: &DocumentationSnapshot{Content: append(json.RawMessage(nil), content...), Version: 1, UpdatedAt: now},
		}
	}
	return nil
}

func (r *documentationRepositoryStub) GetState(context.Context) (*DocumentationAdminState, error) {
	return &r.state, nil
}

func (r *documentationRepositoryStub) SaveDraft(_ context.Context, content json.RawMessage, _ *int64) (*DocumentationAdminState, error) {
	now := time.Unix(1_700_000_100, 0).UTC()
	version := int64(1)
	if r.state.Draft != nil {
		version = r.state.Draft.Version + 1
	}
	r.state.Draft = &DocumentationSnapshot{Content: append(json.RawMessage(nil), content...), Version: version, UpdatedAt: now}
	return &r.state, nil
}
func (r *documentationRepositoryStub) Publish(context.Context, *int64) (*DocumentationAdminState, error) {
	return &r.state, nil
}
func (r *documentationRepositoryStub) ListRevisions(context.Context) ([]DocumentationRevision, error) {
	return nil, nil
}
func (r *documentationRepositoryStub) RestoreRevision(context.Context, int64, *int64) (*DocumentationAdminState, error) {
	return &r.state, nil
}
func (r *documentationRepositoryStub) GetPublished(context.Context) (*DocumentationSnapshot, error) {
	if r.state.Published == nil {
		return nil, ErrDocumentationNotFound
	}
	return r.state.Published, nil
}
func (r *documentationRepositoryStub) SaveAsset(context.Context, *DocumentationAsset) error {
	return nil
}
func (r *documentationRepositoryStub) GetAsset(context.Context, string) (*DocumentationAsset, error) {
	return nil, ErrDocumentationNotFound
}
