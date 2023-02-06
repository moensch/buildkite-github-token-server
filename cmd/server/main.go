package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"

	"github.com/lestrrat-go/jwx/v2/jwk"
	"go.uber.org/zap"

	"github.com/moensch/buildkite-github-token-server/internal/buildkite"
	"github.com/moensch/buildkite-github-token-server/internal/config"
	"github.com/moensch/buildkite-github-token-server/internal/github"
	"github.com/moensch/buildkite-github-token-server/internal/repoparser"
	"github.com/moensch/buildkite-github-token-server/internal/server"
)

const (
	jwksURL     = "https://agent.buildkite.com/.well-known/jwks"
	jwtAudience = "https://buildkite.com/twilio-sandbox"
	jwtIssuer   = "https://agent.buildkite.com"
)

var (
	jwkCache *jwk.Cache
	myctx    context.Context
	bkClient *buildkite.Client
	ghClient *github.Client
)

func allowRepoAccess(organizationSlug string, pipelineSlug string, requestedRepo repoparser.RepositoryName) (bool, error) {
	if strings.HasSuffix(requestedRepo.Repo, "-buildkite-plugin") {
		// Always allow access to buildkite plugin repos
		return true, nil
	}

	// Check if the requested repo is associated with this pipeline
	repo, err := bkClient.GetPipelineRepo(organizationSlug, pipelineSlug)
	if err != nil {
		return false, err
	}
	buildkitePipelineRepo, err := repoparser.ExtractOrgRepoFromURL(repo)
	if err != nil {
		return false, fmt.Errorf("cannot parse repo %s: %s", repo, err)
	}

	if buildkitePipelineRepo.Equals(requestedRepo) {
		// Requested repo matches repo associated with pipeline that issued the token
		return true, nil
	}

	// Check if the requested repo has a gitops.yaml file pointing back to our origin repo
	contents, status, err := ghClient.GetContents(requestedRepo.Org, requestedRepo.Repo, "gitops.yaml")
	if err != nil {
		if status == http.StatusNotFound {
			// Repo does not have a gitops.yaml file, allow access
			log.Printf("WARNING: repo %s does not have a gitops.yaml - anyone can write to it", requestedRepo.HTTPS())
			return true, nil
		}

		// Other error getting gitops.yaml
		return false, err
	}

	// Repo has a gitops.yaml, parse it
	gitops, err := github.GitOpsFromString(contents)
	if err != nil {
		return false, fmt.Errorf("cannot parse gitops.yaml in %s: %s", requestedRepo.HTTPS(), err)
	}

	// Return true if the repo associated with the pipeline that made the request
	// is listed as permitted in the requested repo
	return gitops.RepositoryPermitted(buildkitePipelineRepo), nil

}

func main() {
	rootLogger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatalf("cannot initialize root logger: %s", err)
	}
	cfg, err := config.NewConfig("./config.yaml")
	if err != nil {
		rootLogger.Fatal("cannot load config", zap.Error(err))
	}
	rootLogger.Info("loaded configuration", zap.String("filepath", "./config.yaml"))

	srv := server.New(*cfg)
	err = srv.Initialize()
	if err != nil {
		rootLogger.Error("error initializing server",
			zap.Error(err),
		)
		os.Exit(1)
	}

	// blocking call with log.Fatalf inside
	srv.Serve()
}
