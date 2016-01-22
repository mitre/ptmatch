# ptmatch
===============================
A patient matching test harness to support PCOR

Environment
-----------

This project currently uses Go 1.5 and is built using the Go toolchain.

To install Go, follow the instructions found at the [Go Website](http://golang.org/doc/install).

Following standard Go practices, you should clone this project to:

    $GOPATH/src/github.com/ ... TBD

To get all of the dependencies for this project, run:

    go get

    and, to retrieve test dependencies,

    go get -t

in this directory.

To run all of the tests for this project, run:

    go test ./...

in this directory.

This project also requires MongoDB 3.0.* or higher. To install MongoDB, refer to the
[MongoDB installation guide](http://docs.mongodb.org/manual/installation/).

To start the application, simply run server.go:

    go run main.go
