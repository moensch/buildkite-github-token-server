package github

import (
	"reflect"
	"testing"
)

func TestGitOpsFromString(t *testing.T) {
	type args struct {
		contents string
	}
	tests := []struct {
		name    string
		args    args
		want    GitOps
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GitOpsFromString(tt.args.contents)
			if (err != nil) != tt.wantErr {
				t.Errorf("GitOpsFromString() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GitOpsFromString() = %v, want %v", got, tt.want)
			}
		})
	}
}
