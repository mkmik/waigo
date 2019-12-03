package main

import (
	"flag"
	"path/filepath"
	"sort"
	"strings"

	"github.com/golang/glog"
	"golang.org/x/tools/go/packages"
	fsnotify "gopkg.in/fsnotify.v1"
)

var (
	pattern = flag.String("p", "./...", "package pattern")
)

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

func wait(pattern string) error {
	deps, err := getDeps(pattern)
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

	if err := wait(*pattern); err != nil {
		glog.Fatal(err)
	}
}
