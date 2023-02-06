package github

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/bradleyfalzon/ghinstallation/v2"
	"github.com/google/go-github/v48/github"
	"golang.org/x/oauth2"

	"github.com/moensch/buildkite-github-token-server/internal/config"
	"github.com/moensch/buildkite-github-token-server/internal/repoparser"
)

type Client struct {
	Client     *github.Client
	OrgClients map[string]*OrgClient
	Config     *config.ConfigApplication
}

type OrgClient struct {
	ExpiresAt      time.Time
	Client         *github.Client
	InstallationID int64
}

// NewClientForHost ...
func NewClientForHost(cfg *config.Config, host string) (*Client, error) {
	githubAppHostConfig, err := cfg.AppConfigForHost(host)
	if err != nil {
		log.Fatalf("no github app config found for host %s: %s", host, err.Error())
	}
	client := &Client{
		Config:     githubAppHostConfig,
		OrgClients: make(map[string]*OrgClient),
	}
	itr, err := ghinstallation.NewAppsTransportKeyFromFile(http.DefaultTransport, githubAppHostConfig.AppID, githubAppHostConfig.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize apps transport: %v", err)
	}

	if host == "github.com" {
		// Auth to GitHub.com
		client.Client = github.NewClient(&http.Client{Transport: itr})
	} else {
		// Auth to GitHub Enterprise Server
		client.Client, err = github.NewEnterpriseClient(host, host, &http.Client{Transport: itr})
		if err != nil {
			return nil, err
		}
	}

	// Get all installations
	installations, _, err := client.Client.Apps.ListInstallations(context.TODO(), &github.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("cannot list github app installations: %s", err)
	}
	for _, inst := range installations {
		githubAppHostConfig.Accounts = append(githubAppHostConfig.Accounts, config.ConfigAccount{
			Name:           *inst.Account.Login,
			InstallationID: *inst.ID,
		})
	}

	return client, nil
}

func (c *Client) GetContents(owner string, repo string, path string) (contents string, status int, err error) {
	orgClient, _, err := c.ClientForOrg(owner)
	if err != nil {
		return "", 0, fmt.Errorf("cannot get github client for org %s: %w", owner, err)
	}
	fileContent, _, resp, err := orgClient.Repositories.GetContents(context.TODO(), owner, repo, path, &github.RepositoryContentGetOptions{
		Ref: "HEAD",
	})

	if err != nil {
		if resp != nil {
			return "", resp.StatusCode, err
		}
		return "", 0, err
	}
	contents, err = fileContent.GetContent()
	return contents, 0, err
}

func (c *Client) CreateInstallationToken(repos []repoparser.RepositoryName, permissions *github.InstallationPermissions) (*github.InstallationToken, error) {
	if len(repos) == 0 {
		return nil, fmt.Errorf("must supply at least one repository")
	}
	// Validate all orgs are the same
	repoNames := make([]string, len(repos))

	var lastHostOrg string
	for idx, repo := range repos {
		currentHostOrg := fmt.Sprintf("%s/%s", repo.Host, repo.Org)

		if lastHostOrg != "" && currentHostOrg != lastHostOrg {
			return nil, fmt.Errorf("unable to generate access tokens spanning multiple organizations. Got %s, but already seen %s", currentHostOrg, lastHostOrg)
		}
		repoNames[idx] = repo.Repo
		lastHostOrg = currentHostOrg
	}

	installation, _, err := c.Client.Apps.FindOrganizationInstallation(context.TODO(), repos[0].Org)
	if err != nil {
		return nil, fmt.Errorf("cannot access organization %s: %w", repos[0].Org, err)
	}

	opts := &github.InstallationTokenOptions{
		Repositories: repoNames,
		Permissions:  permissions,
	}

	token, _, err := c.Client.Apps.CreateInstallationToken(context.Background(), *installation.ID, opts)

	return token, err
}

// ClientForOrg returns a GitHub client with read access to contents and metadata for a given org
func (c *Client) ClientForOrg(org string) (*github.Client, int64, error) {
	var installationID int64
	// Check in-memory cache
	if val, ok := c.OrgClients[org]; ok {
		// Found a client in the cache
		if !val.ExpiresAt.Before(time.Now().Add(-time.Second * 30)) {
			// Got more than 30 seconds left on that token/client
			return val.Client, val.InstallationID, nil
		}
		// Despite the client being expired, keep the cached installation ID to save us one API call to github
		installationID = val.InstallationID

		// TODO invalidate cache, multi threading? I think I shouldn't do that
		delete(c.OrgClients, org)
	}

	if installationID == 0 {
		installation, _, err := c.Client.Apps.FindOrganizationInstallation(context.TODO(), org)
		if err != nil {
			return nil, 0, fmt.Errorf("cannot find github app installation for organization %s: %w", org, err)
		}
		installationID = *installation.ID
	}

	// Create new static token so we can access org contents, do not limit to repos
	token, _, err := c.Client.Apps.CreateInstallationToken(context.TODO(), installationID, &github.InstallationTokenOptions{
		Permissions: &github.InstallationPermissions{
			Contents: github.String("read"),
			Metadata: github.String("read"),
		},
	})
	if err != nil {
		return nil, 0, err
	}

	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: *token.Token},
	)

	oauthCli := oauth2.NewClient(context.TODO(), ts)

	c.OrgClients[org] = &OrgClient{
		Client:    github.NewClient(oauthCli),
		ExpiresAt: *token.ExpiresAt,
	}
	return c.OrgClients[org].Client, installationID, nil
}
