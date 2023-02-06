package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	gogit "github.com/google/go-github/v48/github"
	"github.com/lestrrat-go/jwx/v2/jwt"
	"go.uber.org/zap"

	"github.com/moensch/buildkite-github-token-server/api"
	"github.com/moensch/buildkite-github-token-server/internal/contextvalues"
	"github.com/moensch/buildkite-github-token-server/internal/github"
	"github.com/moensch/buildkite-github-token-server/internal/repoparser"
)

// handleToken reponds to /token requests
func (srv *Server) handleToken(w http.ResponseWriter, r *http.Request) {
	logger := contextvalues.GetLogger(r.Context())
	// parse input
	var input api.TokenRequest
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		srv.handleError(w, r, err, "cannot read input", http.StatusBadRequest)
		return
	}

	// get keyset for JWT validation
	keyset, err := srv.jwkCache.Get()
	if err != nil {
		srv.handleError(w, r, err, "", http.StatusInternalServerError)
		return
	}

	requestToken := r.Header.Get("X-Buildkite-OIDC-Token")
	splitToken := strings.Split(requestToken, "Bearer")
	if len(splitToken) != 2 {
		srv.handleError(w, r, nil, "invalid token", http.StatusForbidden)
		return
	}

	// parse the token, this validates:
	//  * expiry
	//  * not before
	//  * audience
	//  * issuer
	verifiedToken, err := jwt.Parse([]byte(strings.TrimSpace(splitToken[1])), jwt.WithKeySet(keyset), jwt.WithAudience(jwtAudience), jwt.WithIssuer(jwtIssuer))
	if err != nil {
		srv.handleError(w, r, err, "cannot verify token", http.StatusForbidden)
		return
	}

	// ensure custom claims are set
	jobID, exists := verifiedToken.Get("job_id")
	if !exists {
		srv.handleError(w, r, nil, "missing job_id", http.StatusBadRequest)
		return
	}
	organizationSlug, exists := verifiedToken.Get("organization_slug")
	if !exists {
		srv.handleError(w, r, nil, "missing organization_id", http.StatusBadRequest)
		return
	}
	pipelineSlug, exists := verifiedToken.Get("pipeline_slug")
	if !exists {
		srv.handleError(w, r, nil, "missing pipeline_slug", http.StatusBadRequest)
		return
	}
	repoStrings := make([]string, len(input.Repositories))
	for i, r := range input.Repositories {
		repoStrings[i] = r.HTTPS()
	}

	reqLogger := logger.With(
		zap.String("job_id", jobID.(string)),
		zap.String("organization_slug", organizationSlug.(string)),
		zap.String("pipeline_slug", pipelineSlug.(string)),
		zap.Strings("repositories", repoStrings),
	)
	reqLogger.Info("processing token request",
		zap.String("access_level", string(input.AccessLevel)),
	)

	for _, repo := range input.Repositories {
		allow, err := srv.allowRepoAccess(reqLogger, organizationSlug.(string), pipelineSlug.(string), repo)
		if err != nil {
			srv.handleError(w, r, err, "error checking repository access", http.StatusInternalServerError)
			return
		}
		if !allow {
			srv.handleError(w, r, nil, fmt.Sprintf("not allowed to acccess repo %s", repo.HTTPS()), http.StatusForbidden)
			return
		}
	}

	// Repos permitted, mint the token
	// TODO nasty
	token, err := srv.githubAppClients[input.Repositories[0].Host].CreateInstallationToken(input.Repositories, &gogit.InstallationPermissions{
		Metadata:     gogit.String("read"),
		Contents:     gogit.String(string(input.AccessLevel)),
		PullRequests: gogit.String(string(input.AccessLevel)),
	})
	if err != nil {
		srv.handleError(w, r, err, "cannot issue access token", http.StatusInternalServerError)
		return
	}

	jsonResp, err := json.Marshal(api.TokenResponse{
		Token:     token.GetToken(),
		ExpiresAt: token.GetExpiresAt(),
		RequestID: contextvalues.GetRequestID(r.Context()),
	})
	if err != nil {
		srv.handleError(w, r, err, "cannot parse response", http.StatusInternalServerError)
		return
	}
	reqLogger.Info("issued token")
	_, _ = w.Write(jsonResp)
}

func (srv *Server) allowRepoAccess(logger *zap.Logger, organizationSlug string, pipelineSlug string, requestedRepo repoparser.RepositoryName) (bool, error) {
	if strings.HasSuffix(requestedRepo.Repo, "-buildkite-plugin") {
		// Always allow access to buildkite plugin repos
		logger.Info("permit access to buildkite plugin repo",
			zap.String("repository", requestedRepo.HTTPS()),
		)
		return true, nil
	}

	// Check if the requested repo is associated with this pipeline
	repo, err := srv.buildkite.GetPipelineRepo(organizationSlug, pipelineSlug)
	if err != nil {
		return false, err
	}
	buildkitePipelineRepo, err := repoparser.ExtractOrgRepoFromURL(repo)
	if err != nil {
		return false, fmt.Errorf("cannot parse repo %s: %s", repo, err)
	}

	if buildkitePipelineRepo.Equals(requestedRepo) {
		// Requested repo matches repo associated with pipeline that issued the token
		logger.Info("permit access to repo associated with pipeline",
			zap.String("repository", requestedRepo.HTTPS()),
		)
		return true, nil
	}

	// Do we have a github client for this git host?
	if _, ok := srv.githubAppClients[requestedRepo.Host]; !ok {
		return false, fmt.Errorf("no github client for %s", requestedRepo.Host)
	}
	// Check if the requested repo has a gitops.yaml file pointing back to our origin repo
	contents, status, err := srv.githubAppClients[requestedRepo.Host].GetContents(requestedRepo.Org, requestedRepo.Repo, "gitops.yaml")
	if err != nil {
		if status == http.StatusNotFound {
			// Repo does not have a gitops.yaml file, deny access
			//log.Printf("WARNING: repo %s does not have a gitops.yaml", requestedRepo.HTTPS())
			return false, nil
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
	permitted := gitops.RepositoryPermitted(buildkitePipelineRepo)
	if permitted {
		logger.Info("permit access per gitops.yaml",
			zap.String("repository", requestedRepo.HTTPS()),
		)
	}
	return permitted, nil
}
