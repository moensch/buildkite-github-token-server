package repoparser

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/gobwas/glob"
	giturls "github.com/whilp/git-urls"
)

type RepositoryName struct {
	Host string
	Org  string
	Repo string
}

func (r *RepositoryName) MarshalJSON() ([]byte, error) {
	return []byte("\"" + r.HTTPS() + "\""), nil
}

// Return org/repo for the repository
func (r RepositoryName) String() string {
	return fmt.Sprintf("%s/%s", r.Org, r.Repo)
}

// HTTPS returns the full https:// URI for the repository
func (r RepositoryName) HTTPS() string {
	return fmt.Sprintf("https://%s/%s/%s.git", r.Host, r.Org, r.Repo)
}

// GIT returns the full git+ssh style URI for the repository
func (r RepositoryName) GIT() string {
	return fmt.Sprintf("git@%s:%s/%s.git", r.Host, r.Org, r.Repo)
}

// Equals returns true if the provided repository has the same host, org, and repo name
func (r RepositoryName) Equals(repo RepositoryName) bool {
	return r.Host == repo.Host && r.Org == repo.Org && r.Repo == repo.Repo
}

// Matches returns true if the provided repository matches a glob-style repo definition
func (r RepositoryName) Matches(repo RepositoryName) bool {
	if r.Host != repo.Host {
		// No glob match on host, sorry
		return false
	}

	g, err := glob.Compile(r.Org)
	if err != nil {
		// ignore errors for now
		return false
	}
	if !g.Match(repo.Org) {
		return false
	}

	g, err = glob.Compile(r.Repo)
	if err != nil {
		// ignore errors for now
		return false
	}
	if !g.Match(repo.Repo) {
		return false
	}

	return true
}

// ExtractOrgRepoFromURL returns the GitHub org and repo string from nearly any GitHub URL
func ExtractOrgRepoFromURL(githubURL string) (RepositoryName, error) {
	ret := RepositoryName{}
	if strings.Count(githubURL, "/") == 2 {
		// The input has a hostname, org, and repo part, but no protocol part
		// Add https as the protocol to make git-urls parse it correctly
		githubURL = fmt.Sprintf("https://%s", githubURL)
	}
	prURL, err := giturls.Parse(githubURL)
	if err != nil {
		return ret, err
	}

	prURLPathParts := strings.Split(strings.TrimPrefix(prURL.Path, "/"), "/")
	if len(prURLPathParts) < 2 {
		return ret, fmt.Errorf("invalid github URL")
	}
	org := prURLPathParts[0]
	repo := prURLPathParts[1]

	ret.Host = prURL.Host
	ret.Org = org
	// strip .git suffix
	ret.Repo = strings.TrimSuffix(repo, filepath.Ext(repo))

	if ret.Host == "" {
		ret.Host = "github.com"
	}

	return ret, nil
}
