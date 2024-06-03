package crdt

import (
	"errors"
	"fmt"
	"log"
	"net"
	"sort"
	"sync"
	"time"
	"bufio"
)

const (
	rootID  = "root"
	trashID = "trash"
	nilID   = ""
)

type treeNode struct {
	id       string
	parent   *treeNode
	children []*treeNode
}

type Tree struct {
	sync.Mutex
	id, time    uint64 // Lamport clock
	Counter     uint64
	replicas    []*Replica
	nodes       map[string]*treeNode
	history     []LogMove
	root, trash *treeNode
}

func NewTree(id uint64, port string, replicaIPs []string) *Tree {
	tree := Tree{}
	tree.id = id
	tree.time = 0
	tree.nodes = make(map[string]*treeNode)
	tree.history = make([]LogMove, 0)

	tree.nodes[rootID] = &treeNode{
		id:       rootID,
		parent:   nil,
		children: make([]*treeNode, 0),
	}
	tree.root = tree.nodes[rootID]

	tree.nodes[trashID] = &treeNode{
		id:       trashID,
		parent:   nil,
		children: make([]*treeNode, 0),
	}
	tree.trash = tree.nodes[trashID]

	go tree.listen(port)
	time.Sleep(5*time.Second) // para que las demas replicas inicien

	tree.replicas = make([]*Replica, len(replicaIPs))
	for i := range tree.replicas {
		tree.replicas[i] = NewReplica(replicaIPs[i])
	}

	return &tree
}

func (this *Tree) listen(port string) {
	ln, err := net.Listen("tcp", ":"+port)
	if err != nil {
		log.Fatal(err)
	}

	for {
		conn, err := ln.Accept()
		if err != nil {
			return
		}

		go this.handleConnection(conn)
	}
}

func (this *Tree) handleConnection(conn net.Conn) {
	defer conn.Close()
	// esta cosa es magica, TCP y bufio mi dios
	reader := bufio.NewReader(conn)
	for {
		msg, err := reader.ReadBytes('}')
		if err != nil {
			return
		}

		// log.Println("RECV:", string(msg))
		go this.applyRecvMove(MoveFromBytes(msg))
	}
}

func (this *Tree) applyRecvMove(op Move) {
	this.Lock()
	defer this.Unlock()

	if this.time < op.Timestamp+1 {
		this.time = op.Timestamp+1
	}

	this.apply(op)
}

// checkear si node1 es descendiente de node2
func (this *Tree) isDescendant(node1, node2 *treeNode) bool {
	curr := node1
	for curr != nil {
		if curr == node2 {
			return true
		} else {
			curr = curr.parent
		}
	}

	return false
}

// cambiar el puntero de posicion
func (this *Tree) movePtr(node, newParent *treeNode) {
	if node.parent != nil {
		for i, child := range node.parent.children {
			if child == node {
				node.parent.children[i] = node.parent.children[len(node.parent.children)-1]
				node.parent.children = node.parent.children[:len(node.parent.children)-1]
			}
		}
	}

	node.parent = newParent
	if newParent != nil {
		newParent.children = append(newParent.children, node)
	}
}

// aplicar la operacion move
// aca esta la magia, debe revertir ops del historial, aplicar la nueva op
// guardar la nueva op en el historial y reaplicar las ops del historial
// ignorando las ops invalidas
func (this *Tree) apply(move Move) {
	// Creando registro en el historial
	this.history = append(this.history, LogMove{
		ReplicaID: move.ReplicaID,
		Timestamp: move.Timestamp,
		NewParent: move.NewParent,
		Node:      move.Node,
	})
	
	// Revirtiendo registros con un timestamp mayor
	i := len(this.history) - 1
	for i > 0 && LogMoveBefore(this.history[i], this.history[i-1]) {
		this.history[i], this.history[i-1] = this.history[i-1], this.history[i]
		this.revert(i)
		i--
	}

	// Creando un nodo implícito
	_, ok := this.nodes[move.Node]
	if !ok {
		this.nodes[move.Node] = &treeNode{
			id:       move.Node,
			parent:   nil,
			children: make([]*treeNode, 0),
		}
	}

	// Aplicando la operacion y reaplicando operaciones revertidas
	for i < len(this.history) {
		this.reapply(i)
		i++
	}

	// Transmision de actualizacion a otras replicas
	if move.ReplicaID == this.id {
		msg := MoveToBytes(move)
		for i := range this.replicas {
			this.replicas[i].Send(msg)
		}
	}
	
	this.time++
	this.Counter++
}

// revierte un logmove si no ha sido ignorado
func (this *Tree) revert(i int) {
	if this.history[i].ignored {
		return
	}

	nodePtr := this.nodes[this.history[i].Node]
	if this.history[i].OldParent == nilID {
		this.movePtr(nodePtr, nil)
	} else {
		parentPtr := this.nodes[this.history[i].OldParent]
		this.movePtr(nodePtr, parentPtr)
	}
}

// reaplica un logmove o lo ignora
func (this *Tree) reapply(i int) {
	nodePtr := this.nodes[this.history[i].Node]
	parentPtr, ok := this.nodes[this.history[i].NewParent]
	this.history[i].ignored = !ok || this.isDescendant(parentPtr, nodePtr)
	if this.history[i].ignored {
		return
	}

	if nodePtr.parent == nil {
		this.history[i].OldParent = nilID
	} else {
		this.history[i].OldParent = nodePtr.parent.id
	}

	this.movePtr(nodePtr, parentPtr)
}

func (this *Tree) Add(name, parent string) error {
	this.Lock()
	defer this.Unlock()

	if _, ok := this.nodes[name]; ok {
		return errors.New("Name already exists")
	}

	parentPtr, ok := this.nodes[parent]
	if !ok {
		return errors.New("Parent does not exist")
	}

	op := Move{
		ReplicaID: this.id,
		Timestamp: this.time,
		NewParent: parentPtr.id,
		Node:      name,
		Time:      time.Now(),
	}
	this.apply(op)
	return nil
}

func (this *Tree) Move(node, newParent string) error {
	this.Lock()
	defer this.Unlock()

	nodePtr := this.nodes[node]
	parentPtr := this.nodes[newParent]
	if nodePtr == nil {
		return errors.New("Node does not exist")
	} else if parentPtr == nil {
		return errors.New("Parent does not exist")
	} else if nodePtr == this.root {
		return errors.New("Cannot move root")
	} else if this.isDescendant(parentPtr, nodePtr) {
		return errors.New("Cannot move node to one of its decendants")
	}

	op := Move{
		ReplicaID: this.id,
		Timestamp: this.time,
		NewParent: parentPtr.id,
		Node:      nodePtr.id,
		Time:      time.Now(),
	}
	this.apply(op)
	return nil
}

func (this *Tree) Remove(node string) error {
	this.Lock()
	defer this.Unlock()

	nodePtr := this.nodes[node]
	if nodePtr == nil {
		return errors.New("Node does not exist")
	} else if nodePtr == this.root {
		return errors.New("Cannot remove root")
	}

	op := Move{
		ReplicaID: this.id,
		Timestamp: this.time,
		NewParent: this.trash.id,
		Node:      nodePtr.id,
		Time:      time.Now(),
	}
	this.apply(op)
	return nil
}

// imprimir arbol como "cmd tree"
func (this *Tree) Print() {
	this.Lock()
	defer this.Unlock()

	fmt.Println(rootID)
	printNode(this.root, "")
}

func printNode(node *treeNode, prefix string) {
	sort.Slice(node.children, func(i, j int) bool {
		return node.children[i].id < node.children[j].id
	})

	for index, child := range node.children {
		if index == len(node.children)-1 {
			fmt.Println(prefix+"└──", child.id)
			printNode(child, prefix+"    ")
		} else {
			fmt.Println(prefix+"├──", child.id)
			printNode(child, prefix+"│   ")
		}
	}
}

func (this *Tree) Disconnect() {
	for i := range this.replicas {
		this.replicas[i].Disconnect()
	}
}

func (this *Tree) Connect() {
	for i := range this.replicas {
		this.replicas[i].Connect()
	}
}

func (this *Tree) Close() {
	for i := range this.replicas {
		this.replicas[i].Close()
	}
}
