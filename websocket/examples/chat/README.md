# Chat Example

This application shows how to use use the
[websocket](https://btcsuite/websocket) package and
[jQuery](http://jquery.com) to implement a simple web chat application.

**NOTE:** This is a fork/vendoring of http://gorilla/websocket
The following documentation has been modified to point at this fork for
convenience.

## Running the example

The example requires a working Go development environment. The [Getting
Started](http://golang.org/doc/install) page describes how to install the
development environment.

Once you have Go up and running, you can download, build and run the example
using the following commands.

    $ go get btcsuite/websocket
    $ cd `go list -f '{{.Dir}}' btcsuite/websocket/examples/chat`
    $ go run *.go

