package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/moensch/buildkite-github-token-server/api"
	"github.com/moensch/buildkite-github-token-server/internal/gitcredentials"
	"github.com/moensch/buildkite-github-token-server/internal/repoparser"
	"github.com/moensch/buildkite-github-token-server/pkg/client"
)

var Version = "dev"

func main() {
	log.Printf("git-credential-github-app %s starting up", Version)

	// We only do something if the action was 'get', this credential helper does not act
	// as a store
	// https://git-scm.com/docs/gitcredentials#_custom_helpers
	// > If it does not support the requested operation (e.g., a read-only store or generator), it should silently
	// > ignore the request.
	// > If a helper receives any other operation, it should silently ignore the request. This leaves room for future
	// > operations to be added (older helpers will just ignore the new requests).
	switch gitcredentials.GetCredentialAction() {
	case "get":
		issueGitHubToken()
	default:
		log.Printf("ignoring action %s", gitcredentials.GetCredentialAction())
		os.Exit(0)
	}
}

func issueGitHubToken() {
	// Process input from git-credentials
	options, err := gitcredentials.ReadInput(os.Stdin)
	if err != nil {
		log.Fatalf("unable to process git-credentials input: %s", err.Error())
	}

	log.Printf("requesting access for: protocol=%s, host=%s, path=%s", options.Protocol, options.Host, options.Path)

	if !strings.HasPrefix(options.Protocol, "http") {
		log.Fatalf("only http and https protocols are supported, not %s", options.Protocol)
	}

	// Validate input
	if options.Host == "" {
		log.Fatalf("git-credentials did not pass `host` field")
	}
	if options.Path == "" {
		log.Fatalf("git-credentials did not pass `path` field - is `useHttpPath` set?")
	}

	repoURI := fmt.Sprintf("%s://%s/%s", options.Protocol, options.Host, options.Path)
	repo, err := repoparser.ExtractOrgRepoFromURL(repoURI)
	if err != nil {
		log.Fatalf("cannot process repo %s: %s", repoURI, err.Error())
	}

	client := client.NewClient("http://localhost:8080")
	resp, errorResp, err := client.GetToken(&api.TokenRequest{
		Repositories: []repoparser.RepositoryName{repo},
		AccessLevel:  api.AccessLevelWrite,
	})
	if err != nil {
		log.Printf("server returned error")
		if errorResp != nil {
			log.Printf("server request ID: %s", errorResp.RequestID)
			log.Printf("server message: %s", errorResp.Message)
		} else {
			log.Printf("%s", err)
		}
		os.Exit(1)
	}

	log.Printf("server request ID: %s", resp.RequestID)
	log.Printf("token expires at: %s", resp.ExpiresAt)

	gitcredentials.SendOutput(gitcredentials.GitCredentialResponse{
		Username: "x-access-token",
		Password: resp.Token,
		Quit:     true,
	}, os.Stdout)
}
