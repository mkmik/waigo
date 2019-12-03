package main

import (
	"encoding/json"
	"flag"
	"go/build"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/golang/glog"
	"github.com/integralist/go-findroot/find"
	"golang.org/x/tools/go/packages"
	fsnotify "gopkg.in/fsnotify.v1"
)

func findProjectBase() (importPath string, dir string, err error) {
	cmd := exec.Command("go", "list", "-json")
	b, err := cmd.Output()
	if err != nil {
		return "", "", err
	}
	var l Package
	if err := json.Unmarshal(b, &l); err != nil {
		return "", "", err
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

// importsUnder returns the package path of pkg pluse all the package paths
// for all the imported packages recursively, as long as they are children
// of the base import path.
func importsUnder(pkg *packages.Package, base string) []string {
	res := []string{strings.TrimSuffix(pkg.PkgPath, ".test")}
	for _, i := range pkg.Imports {
		if strings.HasPrefix(i.PkgPath, base) {
			res = append(res, importsUnder(i, base)...)
		}
	}
	return res
}

// Dedup sorts and deduplicates a string, returning a trimmed slice.
func dedup(s []string) []string {
	sort.Strings(s)
	n := len(s)
	if n <= 1 {
		return s[0:n]
	}
	j := 1
	for i := 1; i < n; i++ {
		if s[i] != s[i-1] {
			s[j] = s[i]
			j++
		}
	}

	return s[0:j]
}

func getDeps(pattern string) ([]string, error) {
	baseImportPath, baseDir, err := findProjectBase()
	if err != nil {
		return nil, err
	}

	pkgs, err := packages.Load(&packages.Config{
		Mode:  packages.NeedName | packages.NeedImports | packages.NeedDeps,
		Tests: true,
	}, pattern)
	if err != nil {
		return nil, err
	}
	var paths []string
	for _, p := range pkgs {
		paths = append(paths, importsUnder(p, baseImportPath)...)
	}
	paths = dedup(paths)

	glog.Infof("import path: %s", baseImportPath)
	glog.Infof("base dir: %s", baseDir)

	var res []string
	for _, d := range paths {
		if strings.HasPrefix(d, baseImportPath) {
			res = append(res, filepath.Join(baseDir, strings.TrimPrefix(d, baseImportPath)))
		}
	}

	return res, nil
}

func wait() error {
	deps, err := getDeps("./...")
	if err != nil {
		return err
	}
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	defer watcher.Close()

	done := make(chan bool)
	go func() {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return
				}
				if event.Op&fsnotify.Write == fsnotify.Write {
					glog.Info(event)
					select {
					case done <- true:
					default:
					}
				}
			case err, ok := <-watcher.Errors:
				if !ok {
					return
				}
				glog.Errorf("error: %v", err)
			}
		}
	}()

	for _, d := range deps {
		glog.Infof("watching %q", d)
		watcher.Add(d)
	}

	<-done
	return nil
}

func main() {
	flag.Parse()
	defer glog.Flush()

	if err := wait(); err != nil {
		glog.Fatal(err)
	}
}
