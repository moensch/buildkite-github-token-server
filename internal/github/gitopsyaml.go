package github

import (
	"fmt"

	"gopkg.in/yaml.v2"

	"github.com/moensch/buildkite-github-token-server/internal/repoparser"
)

type rawGitOps struct {
	ProtectedDestinations []string `yaml:"protectedDestinations"`
	Repositories          []string `yaml:"repos"`
}

type GitOps struct {
	ProtectedDestinations []string
	Repositories          []repoparser.RepositoryName
}

// UnmarshalYAML is a custom unmarshaller that ensures all "repositories" point to a valid repository name
func (g *GitOps) UnmarshalYAML(unmarshal func(interface{}) error) error {
	raw := rawGitOps{}
	err := unmarshal(&raw)
	if err != nil {
		return err
	}

	g.ProtectedDestinations = raw.ProtectedDestinations
	g.Repositories = make([]repoparser.RepositoryName, len(raw.Repositories))
	for idx, repoString := range raw.Repositories {
		repo, err := repoparser.ExtractOrgRepoFromURL(repoString)
		if err != nil {
			return fmt.Errorf("cannot parse repo %s: %w", repoString, err)
		}
		g.Repositories[idx] = repo
	}
	return nil
}

func GitOpsFromString(contents string) (GitOps, error) {
	gitops := &GitOps{}
	err := yaml.Unmarshal([]byte(contents), gitops)
	return *gitops, err
}

// RepositoryPermitted returns true if a given repository is in the permitted list
func (g GitOps) RepositoryPermitted(repo repoparser.RepositoryName) bool {
	for _, r := range g.Repositories {
		if r.Matches(repo) {
			return true
		}
	}
	return false
}
