package source

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/diskcache"

	"github.com/ekristen/distillery/pkg/asset"
	"github.com/ekristen/distillery/pkg/clients/gitlab"
	"github.com/ekristen/distillery/pkg/provider"
)

type GitLab struct {
	provider.Provider

	client *gitlab.Client

	Owner   string
	Repo    string
	Version string

	Release *gitlab.Release
}

func (s *GitLab) GetSource() string {
	return "gitlab"
}
func (s *GitLab) GetOwner() string {
	return s.Owner
}
func (s *GitLab) GetRepo() string {
	return s.Repo
}
func (s *GitLab) GetApp() string {
	return fmt.Sprintf("%s/%s", s.Owner, s.Repo)
}
func (s *GitLab) GetID() string {
	return fmt.Sprintf("%s/%s/%s", s.GetSource(), s.GetOwner(), s.GetRepo())
}

func (s *GitLab) GetDownloadsDir() string {
	return filepath.Join(s.Options.Config.GetDownloadsPath(), s.GetSource(), s.GetOwner(), s.GetRepo(), s.Version)
}

func (s *GitLab) sourceRun(ctx context.Context) error {
	cacheFile := filepath.Join(s.Options.Config.GetMetadataPath(), fmt.Sprintf("cache-%s", s.GetID()))

	s.client = gitlab.NewClient(httpcache.NewTransport(diskcache.New(cacheFile)).Client())
	token := s.Options.Settings["gitlab-token"].(string)
	if token != "" {
		s.client.SetToken(token)
	}

	if s.Version == provider.VersionLatest {
		release, err := s.client.GetLatestRelease(ctx, fmt.Sprintf("%s/%s", s.Owner, s.Repo))
		if err != nil {
			return err
		}

		s.Version = release.TagName
		s.Release = release
	} else {
		release, err := s.client.GetRelease(ctx, fmt.Sprintf("%s/%s", s.Owner, s.Repo), s.Version)
		if err != nil {
			return err
		}

		s.Release = release
	}

	if s.Release == nil {
		return fmt.Errorf("no release found for %s version %s", s.GetApp(), s.Version)
	}

	for _, a := range s.Release.Assets.Links {
		s.Assets = append(s.Assets, &GitLabAsset{
			Asset:  asset.New(filepath.Base(a.URL), "", s.GetOS(), s.GetArch(), s.Version),
			GitLab: s,
			Link:   a,
		})
	}

	return nil
}

func (s *GitLab) Run(ctx context.Context) error {
	if err := s.sourceRun(ctx); err != nil {
		return err
	}

	if err := s.Discover([]string{s.Repo}, s.Version); err != nil {
		return err
	}

	if err := s.CommonRun(ctx); err != nil {
		return err
	}

	return nil
}
