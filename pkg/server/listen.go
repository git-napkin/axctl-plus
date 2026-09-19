package server

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
)

// ListenAndServe listens on the daemon socket, restricts it to 0600, and
// verifies SO_PEERCRED on every accept before serving JSON-RPC.
func (s *Server) ListenAndServe() error {
	if err := os.MkdirAll(filepath.Dir(s.socketPath), 0o700); err != nil {
		return err
	}
	_ = os.Remove(s.socketPath)
	l, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return err
	}
	_ = os.Chmod(s.socketPath, 0o600)
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			continue
		}
		go s.serveVerified(conn)
	}
}

func (s *Server) serveVerified(conn net.Conn) {
	if err := verifyPeerUID(conn); err != nil {
		fmt.Printf("[axctl] rejected connection: %v\n", err)
		_ = conn.Close()
		return
	}
	s.handleConnection(conn)
}
