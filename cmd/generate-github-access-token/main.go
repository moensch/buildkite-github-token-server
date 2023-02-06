package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/moensch/buildkite-github-token-server/api"
	"github.com/moensch/buildkite-github-token-server/internal/repoparser"
	"github.com/moensch/buildkite-github-token-server/pkg/client"
)

var Version = "dev"

func main() {
	log.Printf("generate-github-access-token %s starting up", Version)

	repoFlag := flag.String("repositories", "", "repositories to request access to (can be multiple, comma separated)")
	accessFlag := flag.String("access", "read", "access level ('read' or 'write')")
	flag.Parse()
	if *repoFlag == "" {
		log.Print("no repositories specified")
		os.Exit(0)
	}

	repositories := make([]repoparser.RepositoryName, 0)
	// Process and validate input repos
	var lastHostOrg string
	for _, repo := range strings.Split(*repoFlag, ",") {
		parsedRepo, err := repoparser.ExtractOrgRepoFromURL(repo)
		if err != nil {
			log.Fatalf("unable to parse repository '%s': %s", repo, err.Error())
		}
		currentHostOrg := fmt.Sprintf("%s/%s", parsedRepo.Host, parsedRepo.Org)

		if lastHostOrg != "" && currentHostOrg != lastHostOrg {
			log.Fatalf("unable to generate access tokens spanning multiple organizations. Got %s, but already seen %s", currentHostOrg, lastHostOrg)
		}
		repositories = append(repositories, parsedRepo)
		lastHostOrg = currentHostOrg
	}

	req := &api.TokenRequest{
		Repositories: repositories,
		AccessLevel:  api.AccessLevel(*accessFlag),
	}

	client := client.NewClient("http://localhost:8080")
	resp, errorResp, err := client.GetToken(req)
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

	fmt.Printf("%s\n", resp.Token)
}
