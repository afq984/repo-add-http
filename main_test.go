package main

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func makepkg(t *testing.T, pkgname, pkgver string) string {
	dir := t.TempDir()
	err := ioutil.WriteFile(
		filepath.Join(dir, "PKGBUILD"),
		[]byte(fmt.Sprintf(`pkgname=%s
pkgver=%s
pkgrel=1
arch=("any")
`,
			pkgname, pkgver)),
		0644,
	)
	if err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("makepkg")
	cmd.Dir = dir
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		t.Fatalf("makepkg failed: %v", err)
	}
	cmd = exec.Command("makepkg", "--packagelist")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = dir
	err = cmd.Run()
	if err != nil {
		t.Fatalf("makepkg --packagelist failed: %v", err)
	}
	return strings.TrimSpace(stdout.String())
}

func newServerTest(t *testing.T) *server {
	dir := t.TempDir()
	s, err := newServer(filepath.Join(dir, "test.db.tar"))
	if err != nil {
		t.Fatalf("cannot create test server: %v", err)
	}
	return s
}

func TestServerBasic(t *testing.T) {
	s := newServerTest(t)
	_, err := os.Stat(s.repoDB)
	if err != nil {
		log.Fatalf("cannot stat newly created repo db: %v", err)
	}
}

func TestMakepkgBasic(t *testing.T) {
	pkg := makepkg(t, "pacman", "1.0.0")
	_, err := os.Stat(pkg)
	if err != nil {
		log.Fatalf("cannot stat newly created package: %v", err)
	}
}
