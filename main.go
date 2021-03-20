package main

import (
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sync"

	"github.com/spf13/pflag"
)

var (
	cRepoAdd string
	cListen  string
	cRepoDB  string
)

func init() {
	pflag.StringVar(&cRepoAdd, "repo-add", "repo-add", "path to repo-add")
	pflag.StringVar(&cListen, "listen", "127.8.5.45:8545", "listen address")
	pflag.StringVar(&cRepoDB, "db", "", "repo db path")
}

type server struct {
	repoDir string
	repoDB  string
	dbMux   sync.Mutex
}

var _ http.Handler = &server{}

func newServer(repoDB string) (*server, error) {
	s := &server{
		repoDir: filepath.Dir(repoDB),
		repoDB:  repoDB,
	}
	return s, os.MkdirAll(s.repoDir, 0755)
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "PUT" {
		s.handlePut(w, r)
	} else if r.URL.Path == "/ping" {
		io.WriteString(w, "pong\n")
	} else {
		errorStatus(w, 404)
	}
}

func errorStatus(w http.ResponseWriter, code int) {
	http.Error(w, http.StatusText(code), code)
}

func (s *server) handlePut(w http.ResponseWriter, r *http.Request) {
	defer io.Copy(ioutil.Discard, r.Body)

	pkgname := path.Base(r.URL.Path)
	if len(pkgname) > 80 {
		http.Error(w, "package name too long", http.StatusBadRequest)
		return
	}
	pkgpath := filepath.Join(s.repoDir, pkgname)
	log.Printf("receiving package %s", pkgname)

	f, err := os.OpenFile(pkgpath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0755)
	if os.IsExist(err) {
		log.Print("package already exists")
		io.WriteString(w, "package already exists\n")
		return
	}
	if err != nil {
		log.Printf("open failed: %v", err)
		errorStatus(w, http.StatusInternalServerError)
		return
	}

	cleanup := func() {
		err := os.Remove(pkgpath)
		if err != nil {
			log.Printf("failed to remove %s", pkgpath)
		}
	}

	_, err = io.Copy(f, r.Body)
	if err != nil {
		log.Printf("error in io.Copy: %v", err)
		errorStatus(w, http.StatusInternalServerError)
		cleanup()
		return
	}

	err = f.Close()
	if err != nil {
		log.Printf("failed to close file: %v", err)
		errorStatus(w, http.StatusInternalServerError)
		cleanup()
		return
	}

	cmd := exec.Command(cRepoAdd, "--new", "--remove", s.repoDB, pkgpath)
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	s.dbMux.Lock()
	defer s.dbMux.Unlock()
	log.Printf("running %v", cmd)
	err = cmd.Run()
	if err != nil {
		log.Printf("repo-add failed: %v", err)
		errorStatus(w, http.StatusInternalServerError)
		cleanup()
		return
	}

	log.Printf("successfully added %s", pkgname)
	io.WriteString(w, "ok\n")
}

func main() {
	pflag.Parse()

	_, err := exec.LookPath(cRepoAdd)
	if err != nil {
		log.Fatal(err)
	}

	s, err := newServer(cRepoDB)
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("starting server at %s", cListen)
	err = http.ListenAndServe(cListen, s)
	if err != nil {
		log.Fatal(err)
	}
}
