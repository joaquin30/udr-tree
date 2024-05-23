package net

import (
	"log"
	"net"
	"udr-tree/queue"
)

type Replica struct {
	queue     *queue.Queue
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
		queue:     queue.New(),
		conn:      conn,
		connected: true,
		closed:    false,
	}
	go updateReplica(&replica)
	return &replica
}

func (this *Replica) Send(move string) {
	this.queue.PushBack(move)
}

func (this *Replica) Close() {
	this.closed = true
	err := this.conn.Close()
	if err != nil {
		log.Fatal(err)
	}
}

func updateReplica(replica *Replica) {
	for {
		if replica.closed {
			break
		} else if !replica.connected || replica.queue.Len() == 0 {
			continue
		}
		_, err := replica.conn.Write(replica.queue.Front().([]byte))
		if err == nil {
			replica.queue.PopFront()
		}
	}
}
