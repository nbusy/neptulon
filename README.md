Neptulon
========

[![Build Status](https://travis-ci.org/nbusy/neptulon.svg?branch=master)](https://travis-ci.org/nbusy/neptulon) [![GoDoc](https://godoc.org/github.com/nbusy/neptulon?status.svg)](https://godoc.org/github.com/nbusy/neptulon)

Neptulon is a socket framework with middleware support. Framework core is built on listener and context objects. Each message on each connection creates a context which is then passed on to the registered middleware for handling. Client server communication is full-duplex bidirectional.

Framework core is a small ~1000 SLOC codebase which makes it easy to fork, specialize, and maintain for specific purposes, if you need to.

TLS Only
--------

Currently we only support TLS for communication. Raw TCP/UDP and DTLS support is planned for future iterations.

JSON-RPC 2.0
------------

[neptulon/jsonrpc](jsonrpc) package contains JSON-RPC 2.0 implementation on top of Neptulon. You can see a basic example below.

Example
-------

Following example creates a TLS listener with JSON-RPC 2.0 protocol and starts listening for 'ping' requests and replies with a typical 'pong'.

```go
nep, _ := neptulon.NewServer(cert, privKey, nil, "127.0.0.1:3000", true)
rpc, _ := jsonrpc.NewServer(nep)
route, _ := jsonrpc.NewRouter(rpc)

route.Request("ping", func(ctx *jsonrpc.ReqCtx) {
	ctx.Res = "pong"
})

nep.Run()
```

Users
-----

[Devastator](https://github.com/nbusy/devastator) mobile messaging server is written entirely using the Neptulon framework. It uses JSON-RPC 2.0 package over Neptulon to act as the server part of a mobile messaging app. You can visit its repo to see a complete use case of Neptulon framework.

Testing
-------

All the tests can be executed with `GORACE="halt_on_error=1" go test -race -cover ./...` command. Optionally you can add `-v` flag to observe all connection logs.

License
-------

[MIT](LICENSE)
