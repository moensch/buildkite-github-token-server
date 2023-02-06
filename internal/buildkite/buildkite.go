package buildkite

import (
	"context"
	"fmt"

	graphql "github.com/shurcooL/graphql"
	"golang.org/x/oauth2"
)

const (
	graphQLEndpoint = "https://graphql.buildkite.com/v1"
)

// Client is a buildkite client for querying agent metrics and the buildkite GraphQL API
type Client struct {
	GraphQL *graphql.Client
}

// NewClient instantiates a new client
func NewClient(token string) *Client {
	src := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	httpClient := oauth2.NewClient(context.Background(), src)
	return &Client{
		GraphQL: graphql.NewClient(graphQLEndpoint, httpClient),
	}
}

func (c *Client) GetPipelineRepo(organizationSlug string, pipelineSlug string) (repo string, err error) {
	var q struct {
		Pipeline struct {
			Repository struct {
				URL *string `graphql:"url"`
			} `graphql:"repository"`
		} `graphql:"pipeline(slug:$slug)"`
	}

	variables := map[string]interface{}{
		"slug": fmt.Sprintf("%s/%s", organizationSlug, pipelineSlug),
	}

	err = c.GraphQL.Query(context.Background(), &q, variables)

	if err != nil {
		return "", fmt.Errorf("error fetching pipeline repo from Buildkite for %s: %w", variables["slug"], err)
	}

	if q.Pipeline.Repository.URL == nil {
		// not found
		return "", fmt.Errorf("pipeline %s does not exist", variables["slug"])
	}
	return *q.Pipeline.Repository.URL, nil
}
