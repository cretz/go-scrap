# go-scrap [![GoDoc](https://godoc.org/github.com/cretz/go-scrap?status.svg)](https://godoc.org/github.com/cretz/go-scrap)

go-scrap is a Go wrapper around the Rust [scrap](https://github.com/quadrupleslap/scrap) library. It supports reasonably
fast capturing of raw screen pixels. The library dependency is only at compile time and statically compiled into the
binary. It works on Windows, Linux, and macOS.

## Building

Obtain the library, e.g. use `go get` with `-d` to not install yet:

    go get -d github.com/cretz/go-scrap

Now, the Rust subproject `scrap-sys` must be compiled which is glue between the Go library and the Rust library. With
Rust installed, this can is done by running the following in the `scrap-sys/` subdirectory:

    cargo build --release

Note: On Windows this must use the same `gcc` that Cgo would. Go does not support MSVC-compiled libraries
[yet](https://github.com/golang/go/issues/20982). The easiest way to ensure this is with `rustup` by running
`rustup default stable-x86_64-pc-windows-gnu` before building.

Note: On Linux this needs the X11 XCB libraries with the Shm and RandR extensions. On Ubuntu (18.04+ since RandR must
be >= 1.12) they are packages named `libx11-xcb-dev`, `libxcb-shm0-dev`, and `libxcb-randr0-dev` respectively.

Now that the dependency is built, the library can be built. For example, take a screenshot:

    go run ./example/screenshot

See the [Godoc](https://godoc.org/github.com/cretz/go-scrap) for more documentation and examples.