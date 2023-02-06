package gitcredentials

import (
	"bytes"
	"io"
	"reflect"
	"strings"
	"testing"
)

func TestReadInput(t *testing.T) {
	type args struct {
		from io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    *GitCredentialOptions
		wantErr bool
	}{
		{
			name: "happyPath",
			args: args{from: strings.NewReader("host=github.com\npath=myorg/foorepo.git\nprotocol=https\nusername=foo\npassword=bar\n")},
			want: &GitCredentialOptions{
				Host:     "github.com",
				Path:     "myorg/foorepo.git",
				Username: "foo",
				Protocol: "https",
				Password: "bar",
			},
		},
		{
			name:    "No value",
			args:    args{from: strings.NewReader("host=github.com\npath\n")},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ReadInput(tt.args.from)
			if (err != nil) != tt.wantErr {
				t.Errorf("ReadInput() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ReadInput() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSendOutput(t *testing.T) {
	type args struct {
		resp GitCredentialResponse
	}
	tests := []struct {
		name   string
		args   args
		wantTo string
	}{
		{
			name:   "Happy",
			args:   args{GitCredentialResponse{Username: "foo", Password: "bar"}},
			wantTo: "username=foo\npassword=bar\n",
		},
		{
			name:   "WithQuit",
			args:   args{GitCredentialResponse{Username: "foo", Password: "bar", Quit: true}},
			wantTo: "username=foo\npassword=bar\nquit=true\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			to := &bytes.Buffer{}
			SendOutput(tt.args.resp, to)
			if gotTo := to.String(); gotTo != tt.wantTo {
				t.Errorf("SendOutput() = %v, want %v", gotTo, tt.wantTo)
			}
		})
	}
}
