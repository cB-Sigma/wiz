// Test-only module: github.com/cB-Sigma/wiz's client against the fake
// bulbs of pywizlight's test suite, in Docker (see interop_test.go). Not
// part of the published module.
module github.com/cB-Sigma/wiz/interop

go 1.22

replace github.com/cB-Sigma/wiz => ../

require github.com/cB-Sigma/wiz v0.0.0

require github.com/cB-Sigma/lanbcast v0.2.0 // indirect
