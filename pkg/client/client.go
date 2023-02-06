package client

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strings"

	"github.com/moensch/buildkite-github-token-server/api"
)

type Client struct {
	ServerURL string
	client    *http.Client
}

func NewClient(server string) *Client {
	return &Client{
		ServerURL: server,
	}
}

var (
	BuildkiteTokenGetter = getBuildkiteOIDCToken
)

func (c *Client) GetToken(req *api.TokenRequest) (*api.TokenResponse, *api.HTTPError, error) {
	jsonBody, err := json.Marshal(req)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot prepare request: %w", err)
	}

	request, err := http.NewRequest("POST", fmt.Sprintf("%s/token", c.ServerURL), bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, nil, fmt.Errorf("cannot create http request: %w", err)
	}

	// Set auth header
	buildkiteToken, err := BuildkiteTokenGetter()
	if err != nil {
		var eErr *exec.ExitError
		if errors.As(err, &eErr) {
			return nil, nil, fmt.Errorf("cannot get buildkite OIDC token: %s / %w", eErr.Stderr, err)
		}
		return nil, nil, fmt.Errorf("cannot get buildkite OIDC token: %w", err)
	}
	request.Header.Set("X-Buildkite-OIDC-Token", fmt.Sprintf("Bearer %s", buildkiteToken))

	client := &http.Client{}
	response, err := client.Do(request)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot get github access token: %w", err)
	}
	defer response.Body.Close()
	if err != nil {
		return nil, nil, fmt.Errorf("http response error: %w", err)
	}

	// Read the body
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot read response body: %w", err)
	}

	// Process error responses
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		//log.Printf("token service responded with: %s", response.Status)
		// See if we can parse the error response
		errResp := &api.HTTPError{}
		err = json.Unmarshal(body, errResp)
		if err != nil {
			return nil, errResp, fmt.Errorf("token service responded with error")
		} else {
			// Just print the raw response body
			return nil, nil, fmt.Errorf("token server responded with: %s / %s", response.Status, body)
		}
	}

	// Sucess
	resp := &api.TokenResponse{}
	err = json.Unmarshal(body, resp)
	if err != nil {
		return nil, nil, fmt.Errorf("cannot process server response: %w", err)
	}

	return resp, nil, nil
}

// getBuildkiteOIDCToken invokes buildkite-agent to obtain an OIDC token
// we could do this by calling the agent API directly, but since it's not a publicly
// documented API, this feels safer
func getBuildkiteOIDCToken() (string, error) {
	out, err := exec.Command("buildkite-agent", "oidc", "request-token").Output()
	return strings.TrimSpace(string(out)), err
}
