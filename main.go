package main

import (
	"encoding/json"
	"flag"
	"go/build"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/golang/glog"
	"github.com/integralist/go-findroot/find"
	fsnotify "gopkg.in/fsnotify.v1"
)

func findProjectBase(l Package) (importPath string, dir string, err error) {
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

func getDeps() ([]string, error) {
	cmd := exec.Command("go", "list", "-json")
	b, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var l Package
	if err := json.Unmarshal(b, &l); err != nil {
		return nil, err
	}

	baseImportPath, baseDir, err := findProjectBase(l)
	if err != nil {
		return nil, err
	}

	glog.Infof("import path: %s", baseImportPath)
	glog.Infof("base dir: %s", baseDir)

	var res []string
	for _, d := range l.Deps {
		if strings.HasPrefix(d, baseImportPath) && !strings.Contains(d, "/vendor/") {
			res = append(res, filepath.Join(baseDir, strings.TrimPrefix(d, baseImportPath)))
		}
	}
	return res, nil
}

func wait() error {
	deps, err := getDeps()
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

	deps = append(deps, ".")
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
