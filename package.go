package main

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
