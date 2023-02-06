package repoparser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractOrgRepoFromURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected RepositoryName
		wantErr  bool
	}{
		{
			name:     "pluginRepo",
			input:    "github.com/myorg/fancy-buildkite-plugin#ab8c2e7",
			expected: RepositoryName{Host: "github.com", Org: "myorg", Repo: "fancy-buildkite-plugin"},
			wantErr:  false,
		},
		{
			name:     "internalPluginRepo",
			input:    "ssh://git@ghes.mycompany.com/someorg/docker-buildkite-plugin.git#v1.10.0",
			expected: RepositoryName{Host: "ghes.mycompany.com", Org: "someorg", Repo: "docker-buildkite-plugin"},
			wantErr:  false,
		},
		{
			name:     "noProtocol",
			input:    "github.com/myorg/cool-buildkite-plugin",
			expected: RepositoryName{Host: "github.com", Org: "myorg", Repo: "cool-buildkite-plugin"},
			wantErr:  false,
		},
		{
			name:     "sshURI",
			input:    "git@github.com:myorg/podinfo.git",
			expected: RepositoryName{Host: "github.com", Org: "myorg", Repo: "podinfo"},
			wantErr:  false,
		},
		{
			name:     "sshURINoExtension",
			input:    "git@github.com:myorg/podinfo",
			expected: RepositoryName{Host: "github.com", Org: "myorg", Repo: "podinfo"},
			wantErr:  false,
		},
		{
			name:     "httpsURI",
			input:    "https://github.com/myorg/podinfo.git",
			expected: RepositoryName{Host: "github.com", Org: "myorg", Repo: "podinfo"},
			wantErr:  false,
		},
		{
			name:     "httpsURINoExtension",
			input:    "https://github.com/myorg/podinfo",
			expected: RepositoryName{Host: "github.com", Org: "myorg", Repo: "podinfo"},
			wantErr:  false,
		},
		{
			name:     "SimpleOrgRepo",
			input:    "myorg/somerepo",
			expected: RepositoryName{Host: "github.com", Org: "myorg", Repo: "somerepo"},
			wantErr:  false,
		},
		{
			name:     "SimpleHostOrgRepo",
			input:    "myhost.com/myorg/somerepo",
			expected: RepositoryName{Host: "myhost.com", Org: "myorg", Repo: "somerepo"},
			wantErr:  false,
		},
		{
			name:     "SimpleHostOrgRepoWildcard",
			input:    "myhost.com/myorg/*",
			expected: RepositoryName{Host: "myhost.com", Org: "myorg", Repo: "*"},
			wantErr:  false,
		},
		{
			name:     "SimpleOrgRepoWithExtension",
			input:    "myorg/somerepo.git",
			expected: RepositoryName{Host: "github.com", Org: "myorg", Repo: "somerepo"},
			wantErr:  false,
		},
		{
			name:     "WildCardRepo",
			input:    "myorg/*",
			expected: RepositoryName{Host: "github.com", Org: "myorg", Repo: "*"},
			wantErr:  false,
		},
		{
			name:     "WildCardRepoPartial",
			input:    "myorg/myservice*",
			expected: RepositoryName{Host: "github.com", Org: "myorg", Repo: "myservice*"},
			wantErr:  false,
		},
		{
			name:     "WildCardOrg",
			input:    "*/*",
			expected: RepositoryName{Host: "github.com", Org: "*", Repo: "*"},
			wantErr:  false,
		},
		{
			name:     "WildCardPartialRepo",
			input:    "myorg/foo-*.git",
			expected: RepositoryName{Host: "github.com", Org: "myorg", Repo: "foo-*"},
			wantErr:  false,
		},
		/*
			// this is incorrectly parsed as Repo: "foo-"
			{
				name:     "WildCardSSH",
				input:    "git@ghes.mycompany.com:myorg/foo-*.git",
				expected: RepositoryName{Host: "ghes.mycompany.com", Org: "myorg", Repo: "foo-*"},
				wantErr:  false,
			},
		*/
		{
			name:     "WildCardHTTPS",
			input:    "https://ghes.mycompany.com/myorg/foo-*.git",
			expected: RepositoryName{Host: "ghes.mycompany.com", Org: "myorg", Repo: "foo-*"},
			wantErr:  false,
		},
		{
			name:     "WildCardHTTPSFull",
			input:    "https://ghes.mycompany.com/*/*.git",
			expected: RepositoryName{Host: "ghes.mycompany.com", Org: "*", Repo: "*"},
			wantErr:  false,
		},
		{
			name:    "onlyRepo",
			input:   "somerepo",
			wantErr: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo, err := ExtractOrgRepoFromURL(test.input)
			if test.wantErr {
				require.Error(t, err, "expected function to error but did not")
			} else {
				require.NoError(t, err, "expected function to NOT error but it did")

				t.Logf("expect org '%s' to match '%s'", repo.Org, test.expected.Org)
				t.Logf("expect repo '%s' to match '%s'", repo.Repo, test.expected.Repo)
				t.Logf("expect host '%s' to match '%s'", repo.Host, test.expected.Host)
				assert.Equal(t, test.expected.Org, repo.Org, "org should match expected")
				assert.Equal(t, test.expected.Repo, repo.Repo, "repo should match expected")
				assert.Equal(t, test.expected.Host, repo.Host, "host should match expected")
			}
		})
	}
}

func TestRepositoryName_Matches(t *testing.T) {
	type fields struct {
		Host string
		Org  string
		Repo string
	}
	type args struct {
		repo RepositoryName
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "wildcard-match",
			fields: fields{Host: "github.com", Org: "*", Repo: "*"},
			args:   args{repo: RepositoryName{Host: "github.com", Org: "foobar", Repo: "baz"}},
			want:   true,
		},
		{
			name:   "disallow-host-wildcard",
			fields: fields{Host: "*", Org: "*", Repo: "*"},
			args:   args{repo: RepositoryName{Host: "randomhost.com", Org: "foobar", Repo: "baz"}},
			want:   false,
		},
		{
			name:   "host-mismatch",
			fields: fields{Host: "github.com", Org: "*", Repo: "*"},
			args:   args{repo: RepositoryName{Host: "foobargithub.com", Org: "foobar", Repo: "baz"}},
			want:   false,
		},
		{
			name:   "wildcard-match-repo",
			fields: fields{Host: "github.com", Org: "twilio", Repo: "*"},
			args:   args{repo: RepositoryName{Host: "github.com", Org: "twilio", Repo: "baz"}},
			want:   true,
		},
		{
			name:   "wildcard-match-repo-partial",
			fields: fields{Host: "github.com", Org: "twilio", Repo: "some-*"},
			args:   args{repo: RepositoryName{Host: "github.com", Org: "twilio", Repo: "some-thing"}},
			want:   true,
		},
		{
			name:   "wildcard-mismatch-repo-partial",
			fields: fields{Host: "github.com", Org: "twilio", Repo: "some-*"},
			args:   args{repo: RepositoryName{Host: "github.com", Org: "twilio", Repo: "other-thing"}},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := RepositoryName{
				Host: tt.fields.Host,
				Org:  tt.fields.Org,
				Repo: tt.fields.Repo,
			}
			if got := r.Matches(tt.args.repo); got != tt.want {
				t.Errorf("RepositoryName.Matches() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRepositoryName_Equals(t *testing.T) {
	type fields struct {
		Host string
		Org  string
		Repo string
	}
	type args struct {
		repo RepositoryName
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		{
			name:   "all-match",
			fields: fields{Host: "github.com", Org: "myorg", Repo: "somerepo"},
			args:   args{repo: RepositoryName{Host: "github.com", Org: "myorg", Repo: "somerepo"}},
			want:   true,
		},
		{
			name:   "some-match",
			fields: fields{Host: "github.com", Org: "myorg", Repo: "somerepo"},
			args:   args{repo: RepositoryName{Host: "github.com", Org: "myorg", Repo: "otherrepo"}},
			want:   false,
		},
		{
			name:   "none-match",
			fields: fields{Host: "github.com", Org: "myorg", Repo: "somerepo"},
			args:   args{repo: RepositoryName{Host: "othergithub.com", Org: "otherorg", Repo: "otherrepo"}},
			want:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := RepositoryName{
				Host: tt.fields.Host,
				Org:  tt.fields.Org,
				Repo: tt.fields.Repo,
			}
			if got := r.Equals(tt.args.repo); got != tt.want {
				t.Errorf("RepositoryName.Equals() = %v, want %v", got, tt.want)
			}
		})
	}
}
