//go:build gui && !cgo
// +build gui,!cgo

package main

import "fmt"

func main() {
	fmt.Println("duplica-scan-gui requires cgo. Enable CGO and install a C toolchain (e.g. MSYS2 mingw-w64 gcc).")
}
