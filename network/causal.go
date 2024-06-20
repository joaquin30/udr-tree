/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package network

import (
	"net"
	"sync"

	// "log"
	"github.com/vmihailenco/msgpack/v5"
)

type CausalConn struct {
	toSend    chan []byte
	toApply   chan []byte
	exit      chan bool
	connected bool
	wg        sync.WaitGroup
	conn      net.Conn
}

func NewCausalConn(tree CRDTTree, serverIP string) *CausalConn {
	replica := CausalConn{
		toSend:    make(chan []byte, 100000),
		toApply:   make(chan []byte, 100000),
		exit:      make(chan bool),
		connected: true,
	}

	conn, err := net.Dial("tcp", serverIP)
	if err != nil {
		panic(err)
	}

	replica.conn = conn
	go replica.receiveOperations()
	go replica.processToSend()
	go replica.processToApply(tree)
	return &replica
}

func (conn *CausalConn) Send(data []byte) {
	conn.toSend <- data
}

func (conn *CausalConn) Disconnect() {
	if conn.connected {
		conn.connected = false
		conn.exit <- true
	}
}

func (conn *CausalConn) Connect() {
	if !conn.connected {
		conn.connected = true
		go conn.processToSend()
	}
}

func (conn *CausalConn) Close() {
	close(conn.toSend)
	conn.wg.Wait()
}

func (conn *CausalConn) processToSend() {
	conn.wg.Add(1)
	defer conn.wg.Done()

	for {
		select {
		case data, ok := <-conn.toSend:
			if !ok {
				return
			}

			conn.conn.Write(data)

		case <-conn.exit:
			return
		}
	}
}

func (conn *CausalConn) receiveOperations() {
	dec := msgpack.NewDecoder(conn.conn)
	for {
		data, err := dec.DecodeRaw()
		if err != nil {
			panic(err)
		}

		//log.Println("RECV: " + string(data))
		conn.toApply <- data
	}
}

func (conn *CausalConn) processToApply(tree CRDTTree) {
	for {
		select {
		case data := <-conn.toApply:
			tree.ApplyRemoteOperation(data)
		}
	}
}
