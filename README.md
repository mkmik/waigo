# waigo

Waits until any file belonging to the current project and required by the Go package in the current directory changes.

# Install

```
$ go get -u github.com/mkmik/waigo
```

# Usage

```
$ cd myproject/pkg/somepkg
$ while waigo; do go test; done
```

Now you can edit sources in this package *and* in other related packages
in your project and have the test for the current package re-run if anything changes.
