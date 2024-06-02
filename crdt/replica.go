package crdt

import (
	"log"
	"net"
	"time"
	"sync"
)

type Replica struct {
	queue     chan []byte
	exit      chan bool
	connected bool
	conn      net.Conn
	wg        sync.WaitGroup
}

func NewReplica(serverIP string) *Replica {
	conn, err := net.Dial("tcp", serverIP)
	if err != nil {
		log.Fatal(err)
	}

	replica := Replica{
		queue:     make(chan []byte, 4096),
		exit:      make(chan bool),
		connected: true,
		conn:      conn,
	}
	replica.wg.Add(1)
	go replica.update()
	return &replica
}

func (this *Replica) Send(move Move) {
	this.queue <- MoveToBytes(move)
}

func (this *Replica) Disconnect() {
	if this.connected {
		this.connected = false
		this.exit <- false
	}
}

func (this *Replica) Connect() {
	if !this.connected {
		this.connected = true
		go this.update()
	}
}

func (this *Replica) Close() {
	this.Connect()
	close(this.queue)
	this.wg.Wait()
	close(this.exit)
	this.conn.Close()
}

func (this *Replica) update() {
	for {
		select {
		case msg, ok := <-this.queue:
			if !ok {
				this.wg.Done()
				return
			}
			
			// log.Println("SEND:", string(msg))
			_, err := this.conn.Write(msg)
			if err != nil {
				this.wg.Done()
				return
			}
			
			// para evitar errores con tcp
			time.Sleep(10*time.Millisecond)

		case <-this.exit:
			return
		}
	}
}
