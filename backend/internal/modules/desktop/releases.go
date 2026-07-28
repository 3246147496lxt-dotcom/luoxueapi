package desktop

import (
	"context"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/mod/semver"
)

var strictSemverPattern = regexp.MustCompile(`^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)(?:-[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?(?:\+[0-9A-Za-z-]+(?:\.[0-9A-Za-z-]+)*)?$`)
var releaseSHA256Pattern = regexp.MustCompile(`^[a-fA-F0-9]{64}$`)

type ReleaseManifest struct {
	Version   string `json:"version"`
	Notes     string `json:"notes"`
	PubDate   string `json:"pub_date"`
	URL       string `json:"url"`
	Signature string `json:"signature"`
}

type PublicReleaseManifest struct {
	Version      string `json:"version"`
	InstallerURL string `json:"installer_url"`
	SHA256       string `json:"sha256,omitempty"`
	Notes        string `json:"notes"`
	PubDate      string `json:"pub_date"`
}

func (s *Service) releaseChannelForUser(userID int64) string {
	for _, configuredUserID := range s.desktopConfig.InternalUserIDs {
		if configuredUserID == userID {
			return ReleaseChannelInternal
		}
	}
	return ReleaseChannelStable
}

func (s *Service) FindRelease(
	ctx context.Context,
	subject DesktopSubject,
	target, arch, currentVersion string,
) (*ReleaseManifest, error) {
	if !subject.HasScope(ScopeProfileRead) {
		return nil, ErrDesktopScope
	}
	target = strings.TrimSpace(target)
	arch = strings.TrimSpace(arch)
	currentVersion = strings.TrimSpace(currentVersion)
	if target != ReleaseTargetDarwin || arch != ReleaseArchUniversal || !validStrictSemver(currentVersion) {
		return nil, ErrReleaseParameters
	}
	channel, err := s.repository.GetReleaseChannel(ctx, subject.DeviceID, subject.UserID, subject.TokenVersion)
	if err != nil {
		return nil, err
	}
	if channel != ReleaseChannelStable && channel != ReleaseChannelInternal {
		return nil, ErrReleaseParameters
	}

	var selected *ReleaseManifest
	for _, candidate := range s.desktopConfig.Releases {
		artifactURL := strings.TrimSpace(candidate.URL)
		signature := strings.TrimSpace(candidate.Signature)
		if !candidate.Enabled || candidate.Channel != channel ||
			candidate.Target != target || candidate.Arch != arch ||
			!validStrictSemver(candidate.Version) ||
			semver.Compare("v"+candidate.Version, "v"+currentVersion) <= 0 ||
			!validTrustedReleaseURL(artifactURL) ||
			signature == "" || len(signature) > 8192 ||
			len(candidate.Notes) > 10000 {
			continue
		}
		pubDate, parseErr := time.Parse(time.RFC3339, candidate.PubDate)
		if parseErr != nil {
			continue
		}
		if selected == nil || semver.Compare("v"+candidate.Version, "v"+selected.Version) > 0 {
			selected = &ReleaseManifest{
				Version: candidate.Version, Notes: candidate.Notes,
				PubDate: pubDate.UTC().Format(time.RFC3339), URL: artifactURL,
				Signature: signature,
			}
		}
	}
	return selected, nil
}

func (s *Service) PublicLatestRelease() *PublicReleaseManifest {
	var selected *PublicReleaseManifest
	for _, candidate := range s.desktopConfig.Releases {
		sha256Value := strings.TrimSpace(candidate.SHA256)
		installerURL := strings.TrimSpace(candidate.InstallerURL)
		if !candidate.Enabled || candidate.Channel != ReleaseChannelStable ||
			candidate.Target != ReleaseTargetDarwin || candidate.Arch != ReleaseArchUniversal ||
			!validStrictSemver(candidate.Version) || !validTrustedReleaseURL(installerURL) ||
			(sha256Value != "" && !releaseSHA256Pattern.MatchString(sha256Value)) || len(candidate.Notes) > 10000 {
			continue
		}
		pubDate, err := time.Parse(time.RFC3339, candidate.PubDate)
		if err != nil {
			continue
		}
		if selected == nil || semver.Compare("v"+candidate.Version, "v"+selected.Version) > 0 {
			selected = &PublicReleaseManifest{
				Version: candidate.Version, InstallerURL: installerURL,
				SHA256: strings.ToLower(sha256Value), Notes: candidate.Notes,
				PubDate: pubDate.UTC().Format(time.RFC3339),
			}
		}
	}
	return selected
}

func validStrictSemver(value string) bool {
	return strictSemverPattern.MatchString(value) && semver.IsValid("v"+value)
}

func validTrustedReleaseURL(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	return err == nil && parsed.Scheme == "https" && parsed.Host != "" && parsed.User == nil && parsed.Fragment == ""
}
