package main

import (
	"log"
	goroutinepool "multithreadredis/GoroutinePool"
	"net"
)

const LocalAddress = "localhost:8080"
const BuffSize = 2 * 1024 // 2KB
const PoolSize int = 2
const QueueSize int = 1

func main() {
	listener, err := net.Listen("tcp", LocalAddress)

	if err != nil {
		log.Fatalf("Failed to listen to port %s. Error: %v", LocalAddress, err)
	}

	defer listener.Close()

	log.Println("Server is listening on port", LocalAddress)

	workerPool := goroutinepool.NewGoroutinePool(PoolSize, QueueSize)
	workerPool.Start()

	for {
		conn, err := listener.Accept()

		if err != nil {
			log.Printf("Failed to accept connection: %v", err)
			continue
		}

		workerPool.Submit(conn)
	}
}