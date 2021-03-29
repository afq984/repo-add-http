package main

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http/httptest"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
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

type testServer struct {
	*server
}

func (s *testServer) request(method, target string, body io.Reader) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	s.ServeHTTP(w, httptest.NewRequest(method, target, body))
	return w
}

func newTestServer(t *testing.T) *testServer {
	dir := t.TempDir()
	s, err := newServer(filepath.Join(dir, "test.db.tar"))
	if err != nil {
		t.Fatalf("cannot create test server: %v", err)
	}
	return &testServer{s}
}

func TestServerBasic(t *testing.T) {
	s := newTestServer(t)
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

func TestPing(t *testing.T) {
	s := newTestServer(t)

	resp := s.request("GET", "http://up.example.com/ping", nil)
	assert.Equal(t, 200, resp.Code)
	assert.Equal(t, "pong\n", resp.Body.String())
}
func TestHeadPutHead(t *testing.T) {
	s := newTestServer(t)
	pkg := makepkg(t, "pacman", "1.0.0")
	pkgbin, err := ioutil.ReadFile(pkg)
	if err != nil {
		t.Fatalf("cannot read %s: %v", pkg, err)
	}
	pkgurl := "http://up.example.com/" + path.Base(pkg)

	resp := s.request("HEAD", pkgurl, nil)
	assert.Equal(t, 404, resp.Code)
	assert.Equal(t, "", resp.Body.String())

	resp = s.request("PUT", pkgurl, bytes.NewReader(pkgbin))
	assert.Equal(t, 200, resp.Code)
	assert.Equal(t, "ok\n", resp.Body.String())

	resp = s.request("HEAD", pkgurl, nil)
	assert.Equal(t, 200, resp.Code)
	assert.Equal(t, "", resp.Body.String())

	resp = s.request("PUT", pkgurl, bytes.NewReader(pkgbin))
	assert.Equal(t, 409, resp.Code)
	assert.Equal(t, "package already exists\n", resp.Body.String())
}
