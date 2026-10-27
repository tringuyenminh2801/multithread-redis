package pool

import (
	"errors"
	"io"
	"net"
	"testing"
	"time"
)

const testTimeout = 2 * time.Second
const bufferSize = 2 * 1024

// roundTrip sends a message over client and returns everything the server
// writes back before it closes the connection.
func roundTrip(t *testing.T, client net.Conn) string {
	t.Helper()

	client.SetDeadline(time.Now().Add(testTimeout))

	if _, err := client.Write([]byte("PING")); err != nil {
		t.Fatalf("write to server: %v", err)
	}

	reply, err := io.ReadAll(client)
	if err != nil {
		t.Fatalf("read from server: %v", err)
	}

	return string(reply)
}

// blockingConn reports on started when a worker begins reading from it and
// then holds that worker until release is closed.
type blockingConn struct {
	net.Conn
	started chan<- struct{}
	release <-chan struct{}
}

func (c blockingConn) Read([]byte) (int, error) {
	c.started <- struct{}{}
	<-c.release
	return 0, io.EOF
}

func (c blockingConn) Close() error { return nil }

func (c blockingConn) RemoteAddr() net.Addr { return &net.TCPAddr{} }

func TestHandleConnectionRepliesAndCloses(t *testing.T) {
	server, client := net.Pipe()
	defer client.Close()

	go handleConnection(server, bufferSize)

	if got, want := roundTrip(t, client), "Thank you!"; got != want {
		t.Errorf("reply = %q, want %q", got, want)
	}
}

func TestHandleConnectionClosesOnReadError(t *testing.T) {
	server, client := net.Pipe()
	client.Close()

	done := make(chan struct{})
	go func() {
		handleConnection(server, bufferSize)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(testTimeout):
		t.Fatal("handleConnection did not return after the client disconnected")
	}

	if _, err := server.Write([]byte("x")); !errors.Is(err, io.ErrClosedPipe) {
		t.Errorf("write after handleConnection error = %v, want io.ErrClosedPipe", err)
	}
}

func TestPoolHandlesEverySubmittedConnection(t *testing.T) {
	const connCount = 10

	pool := New(3, connCount, bufferSize)
	pool.Start()

	clients := make([]net.Conn, connCount)
	for i := range clients {
		server, client := net.Pipe()
		defer client.Close()

		clients[i] = client
		pool.Submit(server)
	}

	for i, client := range clients {
		if got, want := roundTrip(t, client), "Thank you!"; got != want {
			t.Errorf("connection %d: reply = %q, want %q", i, got, want)
		}
	}
}

func TestPoolLimitsConcurrentConnections(t *testing.T) {
	const poolSize = 2
	const connCount = 6

	pool := New(poolSize, connCount, bufferSize)
	pool.Start()

	started := make(chan struct{}, connCount)
	release := make(chan struct{})
	t.Cleanup(func() { close(release) })

	for i := 0; i < connCount; i++ {
		pool.Submit(blockingConn{started: started, release: release})
	}

	for i := 0; i < poolSize; i++ {
		select {
		case <-started:
		case <-time.After(testTimeout):
			t.Fatalf("only %d of %d workers picked up a connection", i, poolSize)
		}
	}

	// Every worker is now held by a blockingConn, so nothing else may start.
	select {
	case <-started:
		t.Errorf("more than %d connections handled at once", poolSize)
	case <-time.After(100 * time.Millisecond):
	}
}
