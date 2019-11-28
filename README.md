# waigo

Waits until any file belonging to the current project and required by the Go package in the current directory changes.

# Install

```
$ (cd /; GO111MODULE=on go get github.com/mkmik/waigo@master)
```

# Usage

```
$ cd myproject/pkg/somepkg
$ while waigo; do go test; done
```

Now you can edit sources in this package *and* in other related packages
in your project and have the test for the current package re-run if anything changes.

Handy in combination with [arepa](https://github.com/mkmik/arepa):

```sh
; cat ~/bin/ago
#!/bin/bash

exec arepa -t waigo go "$@"
```
