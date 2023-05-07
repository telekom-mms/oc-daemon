package ocrunner

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Authenticate is an OpenConnect authentication runner
type Authenticate struct {
	// path definitions
	// TODO: get from config?
	Certificate string
	Key         string
	CA          string
	XMLProfile  string
	Script      string
	Server      string
	User        string
	Password    string

	Command *exec.Cmd
	Login   LoginInfo

	// Env are extra environment variables set during execution
	Env []string
}

// Authenticate runs OpenConnect in authentication mode
func (r *Authenticate) Authenticate() {
	// create openconnect command:
	//
	// openconnect \
	//   --protocol=anyconnect \
	//   --certificate="$CLIENT_CERT" \
	//   --sslkey="$PRIVATE_KEY" \
	//   --cafile="$CA_CERT" \
	//   --xmlconfig="$XML_CONFIG" \
	//   --script="$SCRIPT" \
	//   --authenticate \
	//   --quiet \
	//   "$SERVER"
	//
	certificate := fmt.Sprintf("--certificate=%s", r.Certificate)
	sslKey := fmt.Sprintf("--sslkey=%s", r.Key)
	caFile := fmt.Sprintf("--cafile=%s", r.CA)
	xmlConfig := fmt.Sprintf("--xmlconfig=%s", r.XMLProfile)
	script := fmt.Sprintf("--script=%s", r.Script)
	user := fmt.Sprintf("--user=%s", r.User)

	parameters := []string{
		"--protocol=anyconnect",
		certificate,
		sslKey,
		xmlConfig,
		script,
		"--authenticate",
		"--quiet",
		"--no-proxy",
	}
	if r.CA != "" {
		parameters = append(parameters, caFile)
	}
	if r.User != "" {
		parameters = append(parameters, user)
	}
	if r.Password != "" {
		// read password from stdin and switch to non-interactive mode
		parameters = append(parameters, "--passwd-on-stdin")
		parameters = append(parameters, "--non-inter")
	}
	parameters = append(parameters, r.Server)

	r.Command = exec.Command("openconnect", parameters...)

	// run command: allow user input, show stderr, buffer stdout
	var b bytes.Buffer
	r.Command.Stdin = os.Stdin
	if r.Password != "" {
		r.Command.Stdin = bytes.NewBufferString(r.Password)
	}
	r.Command.Stdout = &b
	r.Command.Stderr = os.Stderr
	r.Command.Env = append(os.Environ(), r.Env...)
	if err := r.Command.Run(); err != nil {
		// TODO: handle failed program start?
		log.WithError(err).Error("OC-Runner executing authenticate error")
		return
	}

	// parse login info, cookie from command line in buffer:
	//
	// COOKIE=3311180634@13561856@1339425499@B315A0E29D16C6FD92EE...
	// HOST=10.0.0.1
	// CONNECT_URL='https://vpnserver.example.com'
	// FINGERPRINT=469bb424ec8835944d30bc77c77e8fc1d8e23a42
	// RESOLVE='vpnserver.example.com:10.0.0.1'
	//
	s := b.String()
	for _, line := range strings.Fields(s) {
		r.Login.ParseLine(line)
	}
}

// NewAuthenticate returns a new Authenticate
func NewAuthenticate() *Authenticate {
	return &Authenticate{}
}
