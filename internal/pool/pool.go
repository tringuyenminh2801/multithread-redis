package pool

import (
	"log/slog"
	"net"
	"sync/atomic"

	"multithreadredis/internal/protocol"
)

type Job struct {
	conn net.Conn
}

type Worker struct {
	id         int
	bufferSize int
	jobs       <-chan Job
	stats      *stats
}

// stats is updated by workers and read by the metrics endpoint.
type stats struct {
	active  atomic.Int64
	handled atomic.Uint64
}

func (w Worker) handleJobs() {
	for job := range w.jobs {
		w.stats.active.Add(1)
		slog.Debug("connection started", "worker", w.id, "remote", job.conn.RemoteAddr().String())

		handleConnection(job.conn, w.bufferSize)

		slog.Debug("connection finished", "worker", w.id, "remote", job.conn.RemoteAddr().String())
		w.stats.active.Add(-1)
		w.stats.handled.Add(1)
	}
}

func handleConnection(conn net.Conn, bufferSize int) {
	defer conn.Close()

	buf := make([]byte, bufferSize)
	i := 0

	for {
		n, err := conn.Read(buf)

		if err != nil {
			slog.Debug("read from connection failed", "error", err)

			return
		}

		command, nextI, err := protocol.Decode(buf[i:n])

		if err == nil {
			slog.Debug("received", "type", string(command.Type), "data", command.Data)
		}

		if nextI == n {
			break
		}

		i = nextI
	}

	conn.Write([]byte("Thank you!"))
}

type Pool struct {
	jobs       chan Job
	workers    []Worker
	BufferSize int
	stats      stats
}

func New(poolSize, queueSize, bufferSize int) *Pool {
	jobs := make(chan Job, queueSize)
	workers := make([]Worker, poolSize)

	return &Pool{
		jobs:       jobs,
		workers:    workers,
		BufferSize: bufferSize,
	}
}

func (p *Pool) Start() {
	for i := range p.workers {
		worker := Worker{id: i, jobs: p.jobs, bufferSize: p.BufferSize, stats: &p.stats}
		go worker.handleJobs()
		p.workers[i] = worker
	}
}

func (p *Pool) Submit(conn net.Conn) {
	p.jobs <- Job{conn}
}

// Size is the number of workers.
func (p *Pool) Size() int { return len(p.workers) }

// Active is the number of connections being handled right now.
func (p *Pool) Active() int64 { return p.stats.active.Load() }

// Queued is the number of accepted connections waiting for a free worker.
func (p *Pool) Queued() int { return len(p.jobs) }

// Handled is the total number of connections that have been fully handled.
func (p *Pool) Handled() uint64 { return p.stats.handled.Load() }
