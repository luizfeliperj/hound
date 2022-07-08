package vcs

import (
	"bytes"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/etsy/hound/utf8"
)

const defaultRepo = "/vcs/repos/cvs"

func init() {
	Register(newCvs, "cvs", "concurrent")
}

type CVSDriver struct {
	Cvsroot string `json:"cvsroot"`
}

func newCvs(b []byte) (Driver, error) {
	var d CVSDriver
	d.Cvsroot = defaultRepo

	if b != nil {
		if err := json.Unmarshal(b, &d); err != nil {
			return nil, err
		}
	}

	return &d, nil
}

func (g *CVSDriver) HeadRev(dir string) (string, error) {
	repo, err := ioutil.ReadFile(
		strings.Join([]string{dir, "CVS", "Repository"}, "/"))
	if err != nil {
		log.Print("repo read", err)
		return "", err
	}

	buf := bytes.Buffer{}
	buf.Write(repo)

	args := []string{
		"cvs",
		"-Q",
		"-d",
		g.Cvsroot,
		"rlog",
		"-r",
		strings.TrimSpace(buf.String()),
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed to headrev %s, see output below\n%sContinuing...",
			buf.String(), string(out))
		return "", err
	}

	h := sha1.New()
	if _, err := io.Copy(h, bytes.NewReader(out)); err != nil {
		log.Print("io.Copy", err)
		return "", err
	}

	var dst [40]byte
	hex.Encode(dst[:], h.Sum(nil))

	buf.Truncate(0)
	buf.Write(dst[:])

	return buf.String(), nil
}

func (g *CVSDriver) Pull(dir string) (string, error) {
	repo, err := ioutil.ReadFile(
		strings.Join([]string{dir, "CVS", "Repository"}, "/"))
	if err != nil {
		log.Print("repo read", err)
		return "", err
	}

	buf := bytes.Buffer{}
	buf.Write(repo)

	args := []string{
		"cvs",
		"-Q",
		"-d",
		g.Cvsroot,
		"update",
		"-r",
		"HEAD",
		".",
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = dir

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed to pull %s, see output below\n%sContinuing...",
			buf.String(), string(out))
		return "", err
	}

	utf8.WalkForSourceFiles(dir)
	return g.HeadRev(dir)
}

func (g *CVSDriver) Clone(dir, url string) (string, error) {
	par, rep := filepath.Split(dir)

	args := []string{
		"cvs",
		"-Q",
		"-d",
		g.Cvsroot,
		"checkout",
		"-r",
		"HEAD",
		"-d",
		rep,
		url,
	}

	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = par

	out, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("Failed to clone %s, see output below\n%sContinuing...", url, string(out))
		return "", err
	}

	utf8.WalkForSourceFiles(dir)
	return g.HeadRev(dir)
}

func (g *CVSDriver) SpecialFiles() []string {
	return []string{
		"CVS",
	}
}
