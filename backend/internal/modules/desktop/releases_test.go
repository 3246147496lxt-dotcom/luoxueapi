package desktop

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestFindReleaseEnforcesChannelVersionTargetAndTrustedURL(t *testing.T) {
	repo := &memoryRepository{device: Device{
		ID: 41, PublicID: testDevicePublicID, UserID: 7, Status: DeviceStatusActive,
		TokenVersion: 1, ReleaseChannel: ReleaseChannelStable,
	}}
	cfg := &config.Config{}
	cfg.JWT.Secret = "release-test-secret"
	cfg.Desktop.Releases = []config.DesktopReleaseConfig{
		{Version: "0.2.0", Channel: ReleaseChannelInternal, Target: ReleaseTargetDarwin, Arch: ReleaseArchUniversal, Enabled: true, URL: "https://downloads.example.com/internal.tar.gz", Signature: "sig", PubDate: "2026-07-28T00:00:00Z"},
		{Version: "0.3.0", Channel: ReleaseChannelStable, Target: ReleaseTargetDarwin, Arch: ReleaseArchUniversal, Enabled: false, URL: "https://downloads.example.com/disabled.tar.gz", Signature: "sig", PubDate: "2026-07-28T00:00:00Z"},
		{Version: "0.4.0", Channel: ReleaseChannelStable, Target: ReleaseTargetDarwin, Arch: ReleaseArchUniversal, Enabled: true, URL: "http://untrusted.example.com/update.tar.gz", Signature: "sig", PubDate: "2026-07-28T00:00:00Z"},
		{Version: "0.1.5", Channel: ReleaseChannelStable, Target: ReleaseTargetDarwin, Arch: ReleaseArchUniversal, Enabled: true, URL: "https://downloads.example.com/0.1.5.tar.gz", Signature: "sig-15", PubDate: "2026-07-28T01:00:00+08:00"},
		{Version: "0.2.0", Channel: ReleaseChannelStable, Target: ReleaseTargetDarwin, Arch: ReleaseArchUniversal, Enabled: true, URL: "https://downloads.example.com/0.2.0.tar.gz", Signature: "sig-20", PubDate: "2026-07-28T00:00:00Z"},
	}
	svc := NewService(nil, repo, nil, nil, cfg)
	subject := DesktopSubject{DeviceID: 41, UserID: 7, TokenVersion: 1, Scopes: []string{ScopeProfileRead}}

	manifest, err := svc.FindRelease(context.Background(), subject, ReleaseTargetDarwin, ReleaseArchUniversal, "0.1.0")
	require.NoError(t, err)
	require.Equal(t, "0.2.0", manifest.Version)
	require.Equal(t, "https://downloads.example.com/0.2.0.tar.gz", manifest.URL)
	require.Equal(t, "2026-07-28T00:00:00Z", manifest.PubDate)

	manifest, err = svc.FindRelease(context.Background(), subject, ReleaseTargetDarwin, ReleaseArchUniversal, "0.2.0")
	require.NoError(t, err)
	require.Nil(t, manifest)
	_, err = svc.FindRelease(context.Background(), subject, "macos", ReleaseArchUniversal, "0.1.0")
	require.ErrorIs(t, err, ErrReleaseParameters)
	_, err = svc.FindRelease(context.Background(), subject, ReleaseTargetDarwin, ReleaseArchUniversal, "1.0")
	require.ErrorIs(t, err, ErrReleaseParameters)
}

func TestPublicLatestReleaseNeverLeaksInternalOrUpdaterArtifact(t *testing.T) {
	cfg := &config.Config{}
	cfg.JWT.Secret = "release-public-secret"
	cfg.Desktop.Releases = []config.DesktopReleaseConfig{
		{Version: "9.0.0", Channel: ReleaseChannelInternal, Target: ReleaseTargetDarwin, Arch: ReleaseArchUniversal, Enabled: true, InstallerURL: "https://downloads.example.com/internal.dmg", URL: "https://downloads.example.com/internal.tar.gz", PubDate: time.Now().UTC().Format(time.RFC3339)},
		{Version: "0.2.0", Channel: ReleaseChannelStable, Target: ReleaseTargetDarwin, Arch: ReleaseArchUniversal, Enabled: true, InstallerURL: "https://downloads.example.com/app-0.2.0.dmg", URL: "https://downloads.example.com/app-0.2.0.tar.gz", SHA256: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Notes: "Stable", PubDate: "2026-07-28T00:00:00Z"},
	}
	svc := NewService(nil, &memoryRepository{}, nil, nil, cfg)

	manifest := svc.PublicLatestRelease()
	require.NotNil(t, manifest)
	require.Equal(t, "0.2.0", manifest.Version)
	require.Equal(t, "https://downloads.example.com/app-0.2.0.dmg", manifest.InstallerURL)
	require.NotContains(t, manifest.InstallerURL, "internal")
}

func TestApprovePairingAssignsServerControlledInternalChannel(t *testing.T) {
	store := &memoryPairingStore{pairing: Pairing{Status: PairingStatusPending, DeviceName: "Mac", ExpiresAt: time.Now().Add(time.Minute)}}
	repo := &memoryRepository{}
	cfg := &config.Config{}
	cfg.JWT.Secret = "release-channel-secret"
	cfg.Desktop.InternalUserIDs = []int64{7}
	svc := NewService(store, repo, nil, nil, cfg)

	device, err := svc.ApprovePairing(context.Background(), 7, "ABCD-EFGH")
	require.NoError(t, err)
	require.Equal(t, ReleaseChannelInternal, device.ReleaseChannel)
}
