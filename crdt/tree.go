package crdt

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"log"
	"net"
	"sync"
	"time"
)

var rootID, _ = uuid.Parse("4b16e69d-80e4-446e-bdd1-bd838d021718")
var trashID, _ = uuid.Parse("1f8535a6-e1dc-4256-8b60-c7cfe509993f")
var nilID, _ = uuid.Parse("00000000-0000-0000-0000-000000000000")

type treeNode struct {
	id       uuid.UUID
	name     string
	parent   *treeNode
	children []*treeNode
}

type Tree struct {
	sync.RWMutex
	nodes                map[uuid.UUID]*treeNode
	history              []LogMove
	root, trash, current *treeNode
	replicas             []*Replica
	closed               bool
	clock                LamportClock
}

func NewTree(port string, replicaIPs []string) *Tree {
	tree := Tree{}

	tree.nodes[rootID] = &treeNode{
		id:       rootID,
		name:     "root",
		parent:   nil,
		children: nil,
	}
	tree.root = tree.nodes[rootID]

	tree.nodes[trashID] = &treeNode{
		id:       trashID,
		name:     ".trash",
		parent:   tree.root,
		children: []*treeNode{tree.root},
	}
	tree.trash = tree.nodes[trashID]

	tree.current = tree.root
	tree.closed = false
	tree.clock = LamportClock{ID: uuid.New(), time: 0}

	go tree.listen(port)
	time.Sleep(5 * time.Second) // para que las demas replicas inicien

	tree.replicas = make([]*Replica, len(replicaIPs))
	for i := range tree.replicas {
		tree.replicas[i] = NewReplica(replicaIPs[i])
	}

	return &tree
}

func (this *Tree) listen(port string) {
	ln, err := net.Listen("tcp", ":"+port)
	defer ln.Close()

	if err != nil {
		log.Fatal(err)
	}

	for {
		if this.closed {
			break
		}

		conn, err := ln.Accept()
		if err != nil {
			log.Fatal(err)
		}

		go this.handleConnection(conn)
	}
}

func (this *Tree) handleConnection(conn net.Conn) {
	defer conn.Close()

	var buffer [256]byte
	for {
		if this.closed {
			break
		}

		sz, _ := conn.Read(buffer[:])
		op := MoveFromBytes(buffer[:sz])
		this.clock.Sync(op.Timestamp)
		this.Apply(op)
	}
}

// checkear si node1 es descendiente de node2
func (this *Tree) isDescendant(node1, node2 *treeNode) bool {
	this.RLock()
	defer this.RUnlock()
	if node2 == this.root {
		return true
	}
	if node1 == this.root {
		return false
	}
	var curr *treeNode = node1
	for {
		if curr == this.root {
			break
		}
		if node2 == curr.parent {
			return true
		}
		curr = curr.parent
	}
	return false
}

// dar el puntero que corresponde al path o dar nulo si no existe
func (this *Tree) getNode(path []string) *treeNode {
	this.RLock()
	defer this.RUnlock()
	var curr *treeNode = this.root
	if path[0] == "." || path[0] == ".." {
		curr = this.current
	}

	for i, dir := range path {
		fmt.Println("Processing directory:", dir)
		if dir == "." {
			if i == len(path)-1 {
				return curr
			}
			continue
		}

		if dir == ".." {
			if curr.parent == nil {
				return nil
			}
			curr = curr.parent
			if i == len(path)-1 {
				return curr
			}
			continue
		}

		for _, child := range curr.children {
			if child.name == dir {
				curr = child
				if i == len(path)-1 {
					return curr
				}
				break
			}
		}
	}
	return nil
}

// cambiar el puntero de posicion
// solo es llamado por apply
func (this *Tree) movePtr(node, newParent *treeNode) {

}

// aplicar la operacion move
// aca esta la magia, debe revertir ops del historial, aplicar la nueva op
// guardar la nueva op en el historial y reaplicar las ops del historial
// ignorando las ops invalidas
func (this *Tree) Apply(move Move) {
	this.Lock()
	defer this.Unlock()

	this.history = append(this.history, LogMove{
		ReplicaID: move.ReplicaID,
		Timestamp: move.Timestamp,
		NewParent: move.NewParent,
		NewName:   move.NewName,
		Node:      move.Node,
	})
	i := len(this.history) - 1
	for i > 0 && this.history[i].Before(this.history[i-1]) {
		this.history[i], this.history[i-1] = this.history[i-1], this.history[i]
		this.revert(i)
		i -= 1
	}

	_, ok := this.nodes[move.Node]
	if !ok {
		this.nodes[move.Node] = &treeNode{
			id:       move.Node,
			name:     move.NewName,
			parent:   this.nodes[move.NewParent],
			children: nil,
		}
		this.history[i].OldParent = nilID
		this.history[i].OldName = ""
	} else {
		nodePtr := this.nodes[move.Node]
		this.history[i].OldParent = nodePtr.parent.id
		this.history[i].OldName = nodePtr.name
		this.movePtr(nodePtr, this.nodes[move.NewParent])
	}

	i += 1
	for i < len(this.history) {
		this.reapply(i)
	}
}

func (this *Tree) revert(i int) {
	nodePtr := this.nodes[this.history[i].Node]
	if this.history[i].OldParent == nilID {
		this.movePtr(nodePtr, nil)
	} else {
		parentPtr := this.nodes[this.history[i].OldParent]
		this.movePtr(nodePtr, parentPtr)
		nodePtr.name = this.history[i].OldName
	}
}

func (this *Tree) reapply(i int) {
	nodePtr := this.nodes[this.history[i].Node]
	if nodePtr.parent.id == nilID {
		this.history[i].OldParent = nilID
		this.history[i].OldName = ""
	} else {
		this.history[i].OldParent = nodePtr.parent.id
		this.history[i].OldName = nodePtr.name
	}

	parentPtr := this.nodes[this.history[i].NewParent]
	this.movePtr(nodePtr, parentPtr)
	nodePtr.name = this.history[i].NewName
}

func (this *Tree) Add(name string, parent []string) error {
	parentPtr := this.getNode(parent)
	if parentPtr == nil {
		return errors.New("Parent path does not exist")
	}

	op := Move{
		ReplicaID: this.clock.ID,
		Timestamp: this.clock.Now(),
		NewParent: parentPtr.id,
		NewName:   name,
		Node:      uuid.New(),
	}
	this.Apply(op)
	for i := range this.replicas {
		this.replicas[i].Send(op)
	}

	return nil
}

func (this *Tree) Move(node, newParent []string, newName string) error {
	nodePtr := this.getNode(node)
	parentPtr := this.getNode(newParent)
	if nodePtr == nil {
		return errors.New("Node path does not exist")
	}

	if parentPtr == nil {
		return errors.New("Parent path does not exist")
	}

	op := Move{
		ReplicaID: this.clock.ID,
		Timestamp: this.clock.Now(),
		NewParent: parentPtr.id,
		NewName:   newName,
		Node:      nodePtr.id,
	}
	this.Apply(op)
	for i := range this.replicas {
		this.replicas[i].Send(op)
	}

	return nil
}

func (this *Tree) Remove(node []string) error {
	nodePtr := this.getNode(node)
	if nodePtr == nil {
		return errors.New("Node path does not exist")
	}

	op := Move{
		ReplicaID: this.clock.ID,
		Timestamp: this.clock.Now(),
		NewParent: this.trash.id,
		NewName:   nodePtr.name,
		Node:      nodePtr.id,
	}
	this.Apply(op)
	for i := range this.replicas {
		this.replicas[i].Send(op)
	}

	return nil
}

// cambiar current_node a la path si existe
// cd ./../asd
func (this *Tree) ChangeDir(path []string) error {
	return nil
}

// imprimir arbol como "cmd tree"
func (this *Tree) Print() {
	this.RLock()
	defer this.RUnlock()

}

func (this *Tree) Disconnect() {
	for i := range this.replicas {
		this.replicas[i].connected = false
	}
}

func (this *Tree) Connect() {
	for i := range this.replicas {
		this.replicas[i].connected = true
	}
}

func (this *Tree) Close() {
	this.closed = true
	for i := range this.replicas {
		this.replicas[i].Close()
	}
}
