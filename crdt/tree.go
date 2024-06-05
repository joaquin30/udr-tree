package crdt

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
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
	id, time  uint64 // Lamport clock
	nodes     map[string]*treeNode
	history   []LogMove
	root      *treeNode
	trash     *treeNode
	conn      *ReplicaConn
	// ops    []Move
	LocalSum  time.Duration
	LocalCnt  uint64
	RemoteSum time.Duration
	RemoteCnt uint64
}

func NewTree(id int, serverIP string) *Tree {
	tree := Tree{}
	tree.id = uint64(id)
	tree.time = 0
	tree.nodes = make(map[string]*treeNode)
	tree.history = make([]LogMove, 0)

	tree.nodes[rootID] = &treeNode{
		id:       rootID,
		parent:   nil,
		children: []*treeNode{},
	}
	tree.root = tree.nodes[rootID]

	tree.nodes[trashID] = &treeNode{
		id:       trashID,
		parent:   nil,
		children: []*treeNode{},
	}
	tree.trash = tree.nodes[trashID]
	
	tree.conn = NewReplicaConn(&tree, serverIP)
	return &tree
}

func (this *Tree) applyRecvMove(move Move) {
	this.Lock()
	defer this.Unlock()
	
	if this.time < move.Timestamp+1 {
		this.time = move.Timestamp+1
	}
	
	this.apply(move)
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
	// this.ops = append(this.ops, move)
	
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
		this.LocalCnt++
		this.LocalSum += time.Now().Sub(move.Time)
		this.conn.Send(move)
	} else {
		this.RemoteCnt++
		this.RemoteSum += time.Now().Sub(move.Time)
	}
	
	this.time++
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

	/* sort.Slice(this.ops, func(i, j int) bool {
		if this.ops[i].Timestamp == this.ops[j].Timestamp {
			return this.ops[i].ReplicaID < this.ops[j].ReplicaID
		}
		
		return this.ops[i].Timestamp < this.ops[j].Timestamp
	})
	
	for _, v := range this.ops {
		fmt.Println(v)
	}*/

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
	this.conn.Disconnect()
}

func (this *Tree) Connect() {
	this.conn.Connect()
}

func (this *Tree) Close() {
	this.conn.Close()
}
