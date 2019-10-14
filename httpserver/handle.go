package httpserver

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
)

// HandleOption customizes the behavior of a Handle.
type HandleOption interface {
	apply(*Handle)
}

type handleOptionFunc func(*Handle)

func (f handleOptionFunc) apply(h *Handle) { f(h) }

// ListenFunc is an option for Handle that allows changing how it listens for
// incoming connections.
func ListenFunc(f func(string, string) (net.Listener, error)) HandleOption {
	return handleOptionFunc(func(h *Handle) {
		h.listenFunc = f
	})
}

// DefaultListenFunc builds a net.Listener with the given network and address.
// This function is the default value for ListenFunc.
func DefaultListenFunc(network, address string) (net.Listener, error) {
	ln, err := net.Listen(network, address)

	// keep-alive on all TCP connections. net/http's ListenAndServe and
	// ListenAndServeTLS do this by default but not Server.Serve(..).
	if tcpListener, ok := ln.(*net.TCPListener); ok {
		ln = tcpKeepAliveListener{tcpListener}
	}

	return ln, err
}

func newDialer() dialer { return new(net.Dialer) }

// Changes how we build dialers.
//
// This is an unexported option used for testing only.
func newDialerFunc(f func() dialer) HandleOption {
	return handleOptionFunc(func(h *Handle) {
		h.newDialerFunc = f
	})
}

// Handle is a reference to an HTTP server. It provides clean startup and
// shutdown for net/http HTTP servers.
type Handle struct {
	// HTTP server provided by the user.
	srv *http.Server

	// Listener we're listening on (if any). This is nil if Start hasn't been
	// called yet.
	ln net.Listener

	// errCh will be filled with the error returned by http.Server.Serve.
	errCh chan error

	// Function used to create net.Listeners. Defaults to net.Listen.
	listenFunc func(string, string) (net.Listener, error)

	// Function used to build dialers. Defaults to newDialer.
	newDialerFunc func() dialer
}

// NewHandle builds a Handle to the given HTTP server. You can use the
// returned Handle to start the server and access information about the
// running server.
//
// Handle must be used for all server operations from this point onwards.
// Starting or stopping the http.Server directly will lead to undefined
// behavior.
//
// Note that Handle is not thread-safe. You must not call procedures on Handle
// concurrently.
func NewHandle(srv *http.Server, opts ...HandleOption) *Handle {
	h := &Handle{
		srv:           srv,
		listenFunc:    DefaultListenFunc,
		newDialerFunc: newDialer,
	}

	for _, opt := range opts {
		opt.apply(h)
	}

	return h
}

// Addr returns the address on which the HTTP server is listening. This can be
// used to determine the address of the server if it was started on an
// OS-assigned port (":0").
//
// Returns nil if the server hasn't been started yet.
func (h *Handle) Addr() net.Addr {
	if h.ln == nil {
		return nil
	}
	return h.ln.Addr()
}

// Start starts the HTTP server for this Handle in a separate goroutine and
// blocks until the server is ready to accept requests or the provided context
// finishes.
//
// The server is started on the address defined on Server.Addr, defaulting to
// an OS-assigned port (":0") if Server.Addr is empty.
//
//   h := httpserver.NewHandle(&http.Server{Handler: myHandler})
//   err := h.Start(ctx)
//
// Note that because the server is started in a separate goroutine, this
// method is safe to use as-is inside Fx Lifecycle hooks.
//
//   fx.Hook{
//     OnStart: handle.Start,
//     OnStop: handle.Shutdown,
//   }
func (h *Handle) Start(ctx context.Context) error {
	if h.ln != nil {
		return errors.New("server is already running")
	}

	// http.Server defaults to ":http" if Addr is empty. For our purposes,
	// ":0" is more desirable since we almost never listen on port 80.
	addr := h.srv.Addr
	if addr == "" {
		addr = ":0"
	}

	// Most errors that occur when starting an http.Server are actually Listen
	// errors. If we encounter one of those, we can abort immediately.
	ln, err := h.listenFunc("tcp", addr)
	if err != nil {
		return fmt.Errorf("error starting HTTP server on %q: %v", addr, err)
	}

	errCh := make(chan error, 1)
	go func() {
		// Serve blocks until it encounters an error or until the server shuts
		// down, so we need to call it in a separate goroutine. Errors here
		// (apart from http.ErrServerClosed) are rare.
		err := h.srv.Serve(ln)
		errCh <- err

		// Close the channel so that if shutdown is called on this Handle
		// again, it doesn't wait on the channel indefinitely.
		close(errCh)
	}()

	// We wait until the server is ready to process requests.
	//
	// We would normally be able to return after starting the listener but
	// that introduces a very annoying race condition:
	//
	// Consider,
	//
	//   err := h.Start(..)
	//   h.Shutdown(ctx)
	//
	// If srv.Shutdown gets invoked before the goroutine that is calling
	// srv.Serve has transitioned the server to the running state,
	// srv.Shutdown will return right away but srv.Serve will run forever.
	d := h.newDialerFunc()
	if err := waitUntilAvailable(ctx, d, ln.Addr().String()); err != nil {
		select {
		case err := <-errCh:
			// If the server failed to start up, errCh probably has a more
			// helpful error.
			return fmt.Errorf("error starting HTTP server: %v", err)
		default:
			// Kill the listener if we failed to start the server up.
			//
			// We don't need to do this for the errCh path because having a
			// value in errCh indicates that Serve finished running, and Serve
			// always closes the listener.
			ln.Close()
			return wrapNetErr(err, "error waiting for server to start up")
		}
	}

	h.errCh = errCh
	h.ln = ln
	return nil
}

// Shutdown initiates a graceful shutdown of the HTTP server. The provided
// context controls how long we are willing to wait for the server to shut
// down. Shutdown will block until the server has shut down completely or
// until the context finishes.
func (h *Handle) Shutdown(ctx context.Context) error {
	if err := h.srv.Shutdown(ctx); err != nil {
		return err
	}

	if err := <-h.errCh; err != http.ErrServerClosed {
		return err
	}

	h.ln = nil
	return nil
}
