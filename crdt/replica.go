package crdt

import (
	"log"
	"net"
	"sync"
	// "time"
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
		// se necesita mucho espacio en queue para evitar locks
		// ya que si se queda sin espacio bloquea hasta tenerlo
		queue:     make(chan []byte, 1000000),
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
		this.exit <- true
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
			sz, err := this.conn.Write(msg)
			if err != nil || sz != len(msg) {
				this.wg.Done()
				return
			}
			
			// Sin usar bufio en tree.go se necesita esta linea
			// Ya que los mensajes TCP pueden juntarse, ya que estan basado en streams
			// Pero esa linea hace que el programa sea 40 veces mas lento !!!!!
			
			// TCP framing
			// time.Sleep(10*time.Millisecond)

		case <-this.exit:
			return
		}
	}
}
