package daemon

import (
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/telekom-mms/oc-daemon/internal/daemoncfg"
)

// TestPrepareFolders tests prepareFolders.
func TestPrepareFolders(t *testing.T) {
	// create temp dir and config
	dir := t.TempDir()
	cfg := daemoncfg.NewConfig()

	// set files: config, socket, xml-profile, pid file
	conf := filepath.Join(dir, "conf")
	sock := filepath.Join(dir, "sock")
	prof := filepath.Join(dir, "prof")
	pidf := filepath.Join(dir, "pidf")

	cfg.Config = conf
	cfg.SocketServer.SocketFile = sock
	cfg.OpenConnect.XMLProfile = prof
	cfg.OpenConnect.PIDFile = pidf

	// test
	if err := prepareFolders(cfg); err != nil {
		t.Error(err)
	}
}

// TestRun tests run.
func TestRun(t *testing.T) {
	// test invalid arg
	if err := run([]string{"test", "-invalid"}); err == nil || err == flag.ErrHelp {
		t.Errorf("invalid argument should return error, got: %v", err)
	}

	// test with "-version"
	if err := run([]string{"test", "-version"}); err != nil {
		t.Errorf("version should not return error, got: %v", err)
	}

	// test with "-help"
	if err := run([]string{"test", "-help"}); err != flag.ErrHelp {
		t.Errorf("help should return ErrHelp, got: %v", err)
	}

	// return error in osMkdirAll, so daemon start stops at preprareFolders
	osMkdirAll = func(string, fs.FileMode) error {
		return errors.New("test error")
	}
	defer func() { osMkdirAll = os.MkdirAll }()

	// set temp dir and config file
	dir := t.TempDir()
	cfg := filepath.Join(dir, "config")

	// not existing config
	if err := run([]string{"test", "-verbose", "-config", cfg}); err == nil {
		t.Errorf("start should return error")
	}

	// invalid config
	if err := os.WriteFile(cfg, []byte(`{
	"Executables": {
		"IP": "",
		"Nft": "",
		"Resolvectl": "",
		"Sysctl": ""
	}
}
	`), 0600); err != nil {
		t.Fatal(err)
	}

	if err := run([]string{"test", "-verbose", "-config", cfg}); err == nil {
		t.Errorf("start should return error")
	}

	// not existing command list templates file
	tmplsFile := filepath.Join(dir, "tmpls")
	tmplsConf := fmt.Sprintf(`{
	"CommandLists": {
		"TemplatesFile": "%s"
	}
}
	`, tmplsFile)
	if err := os.WriteFile(cfg, []byte(tmplsConf), 0600); err != nil {
		t.Fatal(err)
	}

	if err := run([]string{"test", "-verbose", "-config", cfg}); err == nil {
		t.Errorf("start should return error")
	}

	// empty command list templates file
	if err := os.WriteFile(tmplsFile, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"test", "-verbose", "-config", cfg}); err == nil {
		t.Errorf("start should return error")
	}

	// not existing command lists file
	cmdListsFile := filepath.Join(dir, "cmd-lists")
	cmdListsConf := fmt.Sprintf(`{
	"CommandLists": {
		"ListsFile": "%s"
	}
}
	`, cmdListsFile)
	if err := os.WriteFile(cfg, []byte(cmdListsConf), 0600); err != nil {
		t.Fatal(err)
	}

	if err := run([]string{"test", "-verbose", "-config", cfg}); err == nil {
		t.Errorf("start should return error")
	}

	// minimum command lists file
	if err := os.WriteFile(cmdListsFile, []byte("[]"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := run([]string{"test", "-verbose", "-config", cfg}); err == nil {
		t.Errorf("start should return error")
	}

	// not existing executables
	if err := os.WriteFile(cfg, []byte(fmt.Sprintf(`{
	"Executables": {
		"IP": "%s"
	}
}
	`, filepath.Join(dir, "does-not-exist"))), 0600); err != nil {
		t.Fatal(err)
	}

	if err := run([]string{"test", "-verbose", "-config", cfg}); err == nil {
		t.Errorf("start should return error")
	}
}
