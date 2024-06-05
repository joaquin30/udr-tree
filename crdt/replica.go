package crdt

import (
	"sync"
	"context"
	"net/url"
	"os"
	"os/signal"
	"syscall"
	"log"
	"strconv"

	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
)

const (
	topic = "_crdt_tree_ucsp"
)

type ReplicaConn struct {
	queue     chan []byte
	exit      chan bool
	connected bool
	wg        sync.WaitGroup
	ctx       context.Context
	stop      context.CancelFunc
	conn      *autopaho.ConnectionManager
}

func NewReplicaConn(tree *Tree, serverIP string) *ReplicaConn {
	replica := ReplicaConn{
		queue: make(chan []byte, 1000),
		exit: make(chan bool),
		connected: true,
	}
	// App will run until cancelled by user (e.g. ctrl-c)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	// We will connect to the Eclipse test server (note that you may see messages that other users publish)
	u, err := url.Parse(serverIP)
	if err != nil {
		panic(err)
	}

	cliCfg := autopaho.ClientConfig{
		ServerUrls: []*url.URL{u},
		
		KeepAlive:  20, // Keepalive message should be sent every 20 seconds
		
		// CleanStartOnInitialConnection defaults to false. Setting this to true will clear the session on the first connection.
		CleanStartOnInitialConnection: true,
		
		// SessionExpiryInterval - Seconds that a session will survive after disconnection.
		// It is important to set this because otherwise, any queued messages will be lost if the connection drops and
		// the server will not queue messages while it is down. The specific setting will depend upon your needs
		// (60 = 1 minute, 3600 = 1 hour, 86400 = one day, 0xFFFFFFFE = 136 years, 0xFFFFFFFF = don't expire)
		SessionExpiryInterval: 60,
		
		OnConnectionUp: func(cm *autopaho.ConnectionManager, connAck *paho.Connack) {
			log.Println("mqtt connection up")
			// Subscribing in the OnConnectionUp callback is recommended (ensures the subscription is reestablished if
			// the connection drops)
			if _, err := cm.Subscribe(context.Background(), &paho.Subscribe{
				Subscriptions: []paho.SubscribeOptions{
					{Topic: topic, QoS: 1, NoLocal: true},
				},
			}); err != nil {
				log.Printf("failed to subscribe (%s). This is likely to mean no messages will be received.", err)
			}
			log.Println("mqtt subscription made")
		},
		
		OnConnectError: func(err error) { log.Printf("error whilst attempting connection: %s\n", err) },
		
		// eclipse/paho.golang/paho provides base mqtt functionality, the below config will be passed in for each connection
		ClientConfig: paho.ClientConfig{
			// If you are using QOS 1/2, then it's important to specify a client id (which must be unique)
			ClientID: "crdt_"+strconv.Itoa(int(tree.id)),
			
			// OnPublishReceived is a slice of functions that will be called when a message is received.
			// You can write the function(s) yourself or use the supplied Router
			OnPublishReceived: []func(paho.PublishReceived) (bool, error){
				func(pr paho.PublishReceived) (bool, error) {
					// fmt.Printf("received message on topic %s; body: %s (retain: %t)\n", pr.Packet.Topic, pr.Packet.Payload, pr.Packet.Retain)
					go tree.applyRecvMove(MoveFromBytes(pr.Packet.Payload))
					return true, nil
				}},
				
			OnClientError: func(err error) { log.Printf("client error: %s\n", err) },
			
			OnServerDisconnect: func(d *paho.Disconnect) {
				if d.Properties != nil {
					log.Printf("server requested disconnect: %s\n", d.Properties.ReasonString)
				} else {
					log.Printf("server requested disconnect; reason code: %d\n", d.ReasonCode)
				}
			},
		},
	}

	// starts process; will reconnect until context cancelled
	conn, err := autopaho.NewConnection(ctx, cliCfg)
	if err != nil {
		panic(err)
	}
	
	// Wait for the connection to come up
	if err = conn.AwaitConnection(ctx); err != nil {
		panic(err)
	}

	replica.ctx  = ctx
	replica.stop = stop
	replica.conn = conn
	go replica.update()
	return &replica
}

func (this *ReplicaConn) Send(move Move) {
	this.queue <- MoveToBytes(move)
}

func (this *ReplicaConn) Disconnect() {
	if this.connected {
		this.connected = false
		this.exit <- true
	}
}

func (this *ReplicaConn) Connect() {
	if !this.connected {
		this.connected = true
		go this.update()
	}
}

func (this *ReplicaConn) Close() {
	close(this.queue)
	this.wg.Wait()
	// this.stop()
}

func (this *ReplicaConn) update() {
	this.wg.Add(1)
	defer this.wg.Done()
	
	for {
		select {
		case msg, ok := <-this.queue:
			if !ok {
				return
			}
			
			// Publish a test message (use PublishViaQueue if you don't want to wait for a response)
			if _, err := this.conn.Publish(this.ctx, &paho.Publish{
				QoS:     1,
				Topic:   topic,
				Payload: msg,
			}); err != nil {
				if this.ctx.Err() == nil {
					panic(err) // Publish will exit when context cancelled or if something went wrong
				}
			}

		case <-this.exit:
			return
		}
	}
}
