package goroutinepool

import (
	"fmt"
	"net"
)

type Job struct {
	conn net.Conn
}

type Worker struct {
	id int
	jobs <-chan Job
}

func (w Worker) handleJobs() {
	for job := range w.jobs {
		fmt.Printf("[Worker %d] Started handling connection from %s\n", w.id, job.conn.RemoteAddr())
		handleConnection(job.conn)
		fmt.Printf("[Worker %d] Finished handling connection from %s\n", w.id, job.conn.RemoteAddr())
	}
}

func handleConnection(conn net.Conn) {
	defer conn.Close()

	buf := make([]byte, 2 * 1024)

	n, err := conn.Read(buf)

	if err != nil {
		fmt.Printf("Error reading from connection socket: %v\n", err)

		return
	}

	fmt.Printf("Received: %s", string(buf[:n]))

	conn.Write([]byte("Thank you!"))
}

type Pool struct {
	jobs chan Job
	workers []Worker
}

func NewGoroutinePool(poolSize, queueSize int) *Pool {
	jobs := make(chan Job, queueSize)
	workers := make([]Worker, poolSize)

	for i := range workers {
		worker := Worker{id: i, jobs: jobs}
		go worker.handleJobs()
		workers[i] = worker
	}

	return &Pool{
		jobs: jobs,
		workers: workers,
	}
}

func (p *Pool) Start() {
	for i := range p.workers {
		worker := Worker{id: i, jobs: p.jobs}
		go worker.handleJobs()
		p.workers[i] = worker
	}
}

func (p *Pool) Submit(conn net.Conn) {
	p.jobs <- Job{conn}
}