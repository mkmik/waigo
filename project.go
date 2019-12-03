package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"go/build"
	"os/exec"

	"github.com/integralist/go-findroot/find"
)

// The package struct is defined by the go help list documentation
// We only include fields that we care about.
type Package struct {
	Module *Module // info about package's containing module, if any (can be nil)
}

// The Module struct is defined by the go help list documentation
// We only include fields that we care about.
type Module struct {
	Path string // module path
	Dir  string // directory holding files for this module, if any
}

// findProjectBase discovers the base dir of the current "project"
// trying to learn about the current Go module, and then falling back
// on the current git repo.
func findProjectBase() (importPath string, dir string, err error) {
	cmd := exec.Command("go", "list", "-json", "./...")
	b, err := cmd.Output()
	if err != nil {
		return "", "", err
	}
	var l Package
	if err := json.NewDecoder(bytes.NewReader(b)).Decode(&l); err != nil {
		return "", "", fmt.Errorf("while parsing go list: %w", err)
	}

	if l.Module != nil {
		return l.Module.Path, l.Module.Dir, nil
	}
	root, err := find.Repo()
	if err != nil {
		return "", "", err
	}
	dir = root.Path
	pkg, err := build.Default.ImportDir(dir, build.FindOnly)
	if err != nil {
		return "", "", err
	}
	return pkg.ImportPath, dir, nil
}