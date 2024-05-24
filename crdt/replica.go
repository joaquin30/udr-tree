package crdt

import (
	"github.com/enriquebris/goconcurrentqueue"
	"log"
	"net"
)

type Replica struct {
	queue     *goconcurrentqueue.FIFO
	conn      net.Conn
	closed    bool
	connected bool
}

func NewReplica(serverIP string) *Replica {
	conn, err := net.Dial("tcp", serverIP)
	if err != nil {
		log.Fatal(err)
	}

	replica := Replica{
		queue:     goconcurrentqueue.NewFIFO(),
		conn:      conn,
		connected: true,
		closed:    false,
	}

	go replica.update()
	return &replica
}

func (this *Replica) Send(move Move) {
	this.queue.Enqueue(MoveToBytes(move))
}

func (this *Replica) Close() {
	this.closed = true
	this.conn.Close()
}

func (this *Replica) update() {
	for {
		if this.closed {
			break
		} else if !this.connected {
			continue
		}

		item, _ := this.queue.DequeueOrWaitForNextElement()
		this.conn.Write(item.([]byte))
	}
}
