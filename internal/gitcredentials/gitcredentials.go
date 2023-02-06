package gitcredentials

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"os"
	"strings"
)

type GitCredentialOptions struct {
	Protocol string
	Host     string
	Path     string
	Username string
	Password string
}

// ReadInput reads git-credential options from a stream. See: https://git-scm.com/docs/git-credential#IOFMT
func ReadInput(from io.Reader) (*GitCredentialOptions, error) {
	opts := &GitCredentialOptions{}
	scanner := bufio.NewScanner(from)
	//ln := newLineIterator(from)
	for scanner.Scan() {
		line := scanner.Text()

		lineSplit := strings.SplitN(string(line), "=", 2)
		if len(lineSplit) < 2 {
			return nil, fmt.Errorf("unable to parse git credential input line: '%s' - not enough parameters", line)
		}
		switch lineSplit[0] {
		case "protocol":
			opts.Protocol = lineSplit[1]
		case "host":
			opts.Host = lineSplit[1]
		case "path":
			opts.Path = lineSplit[1]
		case "username":
			opts.Username = lineSplit[1]
		case "password":
			opts.Password = lineSplit[1]
		default:
			// ignoring any other options
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read gitcredential format: %w", err)
	}

	return opts, nil
}

type GitCredentialResponse struct {
	Username string
	Password string
	Quit     bool
}

// SendOutput sends output back to git-credential
//
// https://git-scm.com/docs/gitcredentials#_custom_helpers
// > If a helper outputs a quit attribute with a value of true or 1, no further helpers will be consulted,
// > nor will the user be prompted (if no credential has been provided, the operation will then fail).
func SendOutput(resp GitCredentialResponse, to io.Writer) {
	var output strings.Builder
	output.WriteString(fmt.Sprintf("username=%s\n", resp.Username))
	output.WriteString(fmt.Sprintf("password=%s\n", resp.Password))
	if resp.Quit {
		output.WriteString("quit=true\n")
	}
	to.Write([]byte(output.String()))
}

// GetCredentialAction returns the credential action requested by git-credentials. It is always the last argument passed to the program
// see: https://git-scm.com/docs/gitcredentials#_custom_helpers
func GetCredentialAction() string {
	action := os.Args[len(os.Args)-1]
	if action != "get" && action != "store" && action != "erase" {
		log.Fatalf("unable to process git-credential action '%s'", action)
	}
	return action
}
