# File Watch example.

**NOTE:** This is a fork/vendoring of http://gorilla/websocket
The following documentation has been modified to point at this fork for
convenience.

This example sends a file to the browser client for display whenever the file is modified.

    $ go get btcsuite/websocket
    $ cd `go list -f '{{.Dir}}' btcsuite/websocket/examples/filewatch`
    $ go run main.go <name of file to watch>
    # Open http://localhost:8080/ .
    # Modify the file to see it update in the browser.
