package httpserver

import (
	"context"
	"fmt"
	"net"
)

var _invalidHTTPRequestLine = []byte("INVALID\n\n")

// Subset of the net.Dialer API that we care about.
type dialer interface {
	DialContext(context.Context, string, string) (net.Conn, error)
}

var _ dialer = (*net.Dialer)(nil)

// waitUntilAvailable uses the given dialer to connect to the HTTP server at
// the provided address and waits until the server is ready to accept requests
// or the given context times out.
//
// This works by sending an invalid request line to the server and waiting for
// a response. The request line is the "GET /index.html HTTP/1.1" part of an
// HTTP request. Instead of sending a valid one which could end up calling the
// user-provided request handler, we send one that will be rejected by the
// HTTP server implementation without crashing.
func waitUntilAvailable(ctx context.Context, d dialer, addr string) error {
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return wrapNetErr(err, "failed to dial to %q", addr)
	}
	defer conn.Close()

	if deadline, ok := ctx.Deadline(); ok {
		// DialContext applies the timeout only to establishing the
		// connection. Here we're applying the same deadline to the rest of
		// this TCP conversation.
		if err := conn.SetDeadline(deadline); err != nil {
			return fmt.Errorf("failed to set connection deadline to %v: %v", deadline, err)
		}
	}

	if _, err := conn.Write(_invalidHTTPRequestLine); err != nil {
		return wrapNetErr(err, "failed to write request to server")
	}

	// Once we receive a single byte from the server, we know that the server
	// is processing HTTP requests.
	var out [1]byte
	if _, err := conn.Read(out[:]); err != nil {
		return wrapNetErr(err, "failed to read response from server")
	}

	return nil
}

// Similar to fmt.Errorf except net.Error timeouts are translated to
// context.DeadlineExceeded.
func wrapNetErr(err error, msg string, args ...interface{}) error {
	if err == nil {
		return nil
	}

	if ne, ok := err.(net.Error); ok && ne.Timeout() {
		return context.DeadlineExceeded
	}

	if len(args) > 0 {
		msg = fmt.Sprintf(msg, args...)
	}

	return fmt.Errorf("%s: %v", msg, err)
}
