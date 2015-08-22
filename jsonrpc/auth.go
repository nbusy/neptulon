package jsonrpc

import "log"

// todo: action taken in response to un authenticated req/res/not messages should be configurable on per-item basis,
// as closing the connection always might not be the desired behavior

// todo2: we need to pass client CA cert as a param here which will add it to listener.tls.Config file as client CA cert
// rather than TLS listener always requiring client CA cert w/ constructor

// CertAuth is a TLS certificate authentication middleware for Neptulon JSON-RPC app.
type CertAuth struct {
}

// NewCertAuth creates and registers a new certificate authentication middleware instance with a Neptulon JSON-RPC app.
func NewCertAuth(app *App) (*CertAuth, error) {
	a := CertAuth{}
	app.ReqMiddleware(a.reqMiddleware)
	app.ResMiddleware(a.resMiddleware)
	app.NotMiddleware(a.notMiddleware)
	return &a, nil
}

func (a *CertAuth) reqMiddleware(ctx *ReqContext) {
	if ctx.Conn.UID != "" {
		return
	}

	// if provided, client certificate is verified by the TLS listener so the peerCerts list in the connection is trusted
	certs := ctx.Conn.ConnectionState().PeerCertificates
	if len(certs) == 0 {
		log.Println("Invalid client-certificate authentication attempt:", ctx.Conn.RemoteAddr())
		ctx.Done = true
		ctx.Conn.Close()
		return
	}

	userID := certs[0].Subject.CommonName
	ctx.Conn.UID = userID
	log.Println("Client-certificate authenticated:", ctx.Conn.RemoteAddr(), userID)
}

func (a *CertAuth) resMiddleware(ctx *ResContext) {
	if ctx.Conn.UID != "" {
		return
	}

	ctx.Done = true
	ctx.Conn.Close()
}

func (a *CertAuth) notMiddleware(ctx *NotContext) {
	if ctx.Conn.UID != "" {
		return
	}

	ctx.Done = true
	ctx.Conn.Close()
}
