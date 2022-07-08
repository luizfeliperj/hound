package utf8

import (
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func convToUTF8(dir string, paths []string) error {
	var err error
	args := []string{
		"vim", "-n", "-T", "builtin_dumb",
		"+silent argdo se nobomb | se fileencoding=utf-8 | w", "-c", ":q",
	}

	cmd := exec.Command(args[0], append(args[1:], paths...)...)
	cmd.Dir = dir

	if cmd.Stdin, err = os.Open(os.DevNull); err != nil {
		log.Print("utf8 Open Stdin:", err)
		return err
	}

	if cmd.Stderr, err = os.OpenFile(os.DevNull, os.O_WRONLY, 0644); err != nil {
		log.Print("utf8 Open Stderr:", err)
		return err
	}

	if out, err := cmd.Output(); err != nil {
		log.Print("utf8 Run:", err, string(out))
		return err
	}

	return nil
}

func WalkForSourceFiles(dir string) error {
	var paths []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		switch strings.ToLower(filepath.Ext(path)) {
		case ".h":
			fallthrough
		case ".c":
			fallthrough
		case ".sh":
			fallthrough
		case ".cu":
			fallthrough
		case ".cc":
			fallthrough
		case ".hpp":
			fallthrough
		case ".cpp":
			fallthrough
		case ".f":
			fallthrough
		case ".f90":
			fallthrough
		case ".f77":
			if p, err := filepath.Rel(dir, path); err == nil {
				paths = append(paths, p)
			} else {
				return err
			}
		}

		return nil
	})

	if err == nil {
		convToUTF8(dir, paths)
		return nil
	}

	return err
}
