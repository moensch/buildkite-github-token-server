package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/moensch/buildkite-github-token-server/api"
	"github.com/moensch/buildkite-github-token-server/internal/repoparser"
)

var (
	validOIDCToken = "validtoken"
)

type testTokenServer struct {
	response   interface{}
	statusCode int
}

func (s *testTokenServer) New(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var input api.TokenRequest
		json.NewDecoder(r.Body).Decode(&input)

		requestToken := r.Header.Get("X-Buildkite-OIDC-Token")
		splitToken := strings.Split(requestToken, "Bearer")
		if len(splitToken) != 2 {
			w.WriteHeader(http.StatusForbidden)
			resp := api.HTTPError{
				RequestID: "foo",
				Message:   "invalid token",
			}
			jsonResp, _ := json.Marshal(resp)
			_, _ = w.Write(jsonResp)
			return
		}
		if strings.TrimSpace(splitToken[1]) != validOIDCToken {
			w.WriteHeader(http.StatusForbidden)
			resp := api.HTTPError{
				RequestID: "foo",
				Message:   fmt.Sprintf("bad token, wanted %s, got %s", validOIDCToken, splitToken[1]),
			}
			jsonResp, _ := json.Marshal(resp)
			_, _ = w.Write(jsonResp)
			return
		}

		if r.URL.Path == "/token" {
			w.WriteHeader(s.statusCode)
			jsonResp, _ := json.Marshal(s.response)
			_, _ = w.Write(jsonResp)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func fakeGetBuildkiteOIDCTokenOK() (string, error) {
	return validOIDCToken, nil
}

func fakeGetBuildkiteOIDCTokenEmpty() (string, error) {
	return " ", nil
}

func fakeGetBuildkiteOIDCTokenFail() (string, error) {
	return "", &exec.ExitError{
		Stderr: []byte("something went wrong"),
	}
}

func TestClient_GetToken(t *testing.T) {
	type fields struct {
		ServerURL string
		client    *http.Client
	}
	type args struct {
		req *api.TokenRequest
	}
	tests := []struct {
		name             string
		serverResponse   interface{}
		serverStatusCode int
		request          *api.TokenRequest
		want             *api.TokenResponse
		tokenGetterFunc  func() (string, error)
		wantErr          bool
	}{
		{
			name: "happypath",
			serverResponse: api.TokenResponse{
				Token:     "abcde",
				ExpiresAt: time.Now(),
				RequestID: "foo",
			},
			serverStatusCode: http.StatusOK,
			request: &api.TokenRequest{
				Repositories: []repoparser.RepositoryName{
					{
						Host: "github.com",
						Org:  "myorg",
						Repo: "happypath",
					},
				},
				AccessLevel: api.AccessLevelRead,
			},
			tokenGetterFunc: fakeGetBuildkiteOIDCTokenOK,
			wantErr:         false,
		},
		{
			name: "server error",
			serverResponse: api.HTTPError{
				RequestID: "foo",
				Message:   "this is an error",
			},
			serverStatusCode: http.StatusInternalServerError,
			request: &api.TokenRequest{
				Repositories: []repoparser.RepositoryName{
					{
						Host: "github.com",
						Org:  "myorg",
						Repo: "error",
					},
				},
				AccessLevel: api.AccessLevelRead,
			},
			tokenGetterFunc: fakeGetBuildkiteOIDCTokenOK,
			wantErr:         true,
		},
		{
			name: "forbidden",
			serverResponse: api.HTTPError{
				RequestID: "foo",
				Message:   fmt.Sprintf("not allowed to acccess repo foobarbaz"),
			},
			serverStatusCode: http.StatusForbidden,
			request: &api.TokenRequest{
				Repositories: []repoparser.RepositoryName{
					{
						Host: "github.com",
						Org:  "myorg",
						Repo: "forbidden",
					},
				},
				AccessLevel: api.AccessLevelRead,
			},
			tokenGetterFunc: fakeGetBuildkiteOIDCTokenOK,
			wantErr:         true,
		},
		{
			name: "local token error",
			request: &api.TokenRequest{
				Repositories: []repoparser.RepositoryName{
					{
						Host: "github.com",
						Org:  "myorg",
						Repo: "happypath",
					},
				},
				AccessLevel: api.AccessLevelRead,
			},
			tokenGetterFunc: fakeGetBuildkiteOIDCTokenFail,
			wantErr:         true,
		},
		{
			name: "empty token",
			request: &api.TokenRequest{
				Repositories: []repoparser.RepositoryName{
					{
						Host: "github.com",
						Org:  "myorg",
						Repo: "happypath",
					},
				},
				AccessLevel: api.AccessLevelRead,
			},
			tokenGetterFunc: fakeGetBuildkiteOIDCTokenEmpty,
			wantErr:         true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			func() {
				server := (&testTokenServer{
					response:   tt.serverResponse,
					statusCode: tt.serverStatusCode,
				}).New(t)
				//server := newTestServer(t)
				defer server.Close()
				c := NewClient(server.URL)
				BuildkiteTokenGetter = tt.tokenGetterFunc
				resp, httpErr, err := c.GetToken(tt.request)
				t.Logf("resp: %+v", resp)
				t.Logf("httpErr: %+v", httpErr)
				t.Logf("err: %+v", err)

				if (err != nil) != tt.wantErr {
					t.Errorf("GetToken() error = %v, wantErr %v", err, tt.wantErr)
					return
				}
			}()
		})
	}
}
