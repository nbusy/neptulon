package neptulon

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"sync"
)

var (
	ping   = []byte("ping")
	closed = []byte("close")
)

// Listener accepts connections from devices.
type Listener struct {
	debug    bool
	listener net.Listener
	connWG   sync.WaitGroup
	reqWG    sync.WaitGroup
}

// Listen creates a TCP listener with the given PEM encoded X.509 certificate and the private key on the local network address laddr.
// Debug mode logs all server activity.
func Listen(cert, privKey []byte, laddr string, debug bool) (*Listener, error) {
	tlsCert, err := tls.X509KeyPair(cert, privKey)
	pool := x509.NewCertPool()
	ok := pool.AppendCertsFromPEM(cert)
	if err != nil || !ok {
		return nil, fmt.Errorf("failed to parse the certificate or the private key: %v", err)
	}

	conf := tls.Config{
		Certificates: []tls.Certificate{tlsCert},
		ClientCAs:    pool,
		ClientAuth:   tls.VerifyClientCertIfGiven,
	}

	l, err := tls.Listen("tcp", laddr, &conf)
	if err != nil {
		return nil, fmt.Errorf("failed to create TLS listener on network address %v with error: %v", laddr, err)
	}
	log.Printf("Listener created: %v\n", laddr)

	return &Listener{
		debug:    debug,
		listener: l,
	}, nil
}

// Accept waits for incoming connections and forwards the client connect/message/disconnect events to provided handlers in a new goroutine.
// This function blocks and never returns, unless there is an error while accepting a new connection.
func (l *Listener) Accept(handleConn func(conn *Conn), handleMsg func(conn *Conn, msg []byte), handleDisconn func(conn *Conn)) error {
	defer log.Println("Listener closed:", l.listener.Addr())
	for {
		conn, err := l.listener.Accept()
		if err != nil {
			if operr, ok := err.(*net.OpError); ok && operr.Op == "accept" && operr.Err.Error() == "use of closed network connection" {
				return nil
			}
			return fmt.Errorf("error while accepting a new connection from a client: %v", err)
			// todo: it might not be appropriate to break the loop on recoverable errors (like client disconnect during handshake)
			// the underlying fd.accept() does some basic recovery though we might need more: http://golang.org/src/net/fd_unix.go
		}

		// todo: this casting early on doesn't seem necessary and can be removed in a futur iteration not to cause any side effects
		tlsconn, ok := conn.(*tls.Conn)
		if !ok {
			conn.Close()
			return errors.New("cannot cast net.Conn interface to tls.Conn type")
		}

		l.connWG.Add(1)
		log.Println("Client connected:", conn.RemoteAddr())

		c, err := NewConn(tlsconn, 0, 0, 0, l.debug)
		if err != nil {
			return err
		}

		go handleClient(l, c, handleConn, handleMsg, handleDisconn)
	}
}

// handleClient waits for messages from the connected client and forwards the client message/disconnect
// events to provided handlers in a new goroutine.
// This function never returns, unless there is an error while reading from the channel or the client disconnects.
func handleClient(l *Listener, conn *Conn, handleConn func(conn *Conn), handleMsg func(conn *Conn, msg []byte), handleDisconn func(conn *Conn)) error {
	handleConn(conn)

	defer func() {
		conn.Session.error = conn.Close() // todo: handle close error, store the error in conn object and return it to handleMsg/handleErr/handleDisconn or one level up (to server)
		if conn.Session.disconnected {
			log.Println("Client disconnected:", conn.RemoteAddr())
		} else {
			log.Println("Closed client connection:", conn.RemoteAddr())
		}
		handleDisconn(conn)
		l.connWG.Done()
	}()

	for {
		if conn.Session.error != nil {
			// todo: send error message to user, log the error, and close the conn and return
			return conn.Session.error
		}

		n, msg, err := conn.Read()
		if err != nil {
			if err == io.EOF {
				conn.Session.disconnected = true
				break
			}
			if operr, ok := err.(*net.OpError); ok && operr.Op == "read" && operr.Err.Error() == "use of closed network connection" {
				conn.Session.disconnected = true
				break
			}
			log.Fatalln("Errored while reading:", err)
		}

		// shortcut 'ping' and 'close' messages, saves some processing time
		if n == 4 && bytes.Equal(msg, ping) {
			continue // send back pong?
		}
		if n == 5 && bytes.Equal(msg, closed) {
			return conn.Session.error
		}

		l.reqWG.Add(1)
		go func() {
			defer l.reqWG.Done()
			handleMsg(conn, msg)
		}()
	}

	return conn.Session.error
}

// Close closes the listener.
func (l *Listener) Close() error {
	return l.listener.Close()
}
