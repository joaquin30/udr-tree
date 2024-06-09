package crdt

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"
	"log"
	"udr-tree/network"
	"github.com/google/uuid"
)

const (
	rootName = "root"
	trashName = "trash"
	NumReplicas = 3
)

var (
	rootID = uuid.MustParse("5568143b-b80b-452d-ba1b-f9d333a06e7a")
	trashID = uuid.MustParse("cf0008e9-9845-465b-8c9b-fa4b31b7f3bd")
)

type treeNode struct {
	id       uuid.UUID
	name     string
	parent   *treeNode
	children []*treeNode
}

type Tree struct {
	sync.Mutex
	id          uint64
	localTime   uint64 // lamport clock
	time        [NumReplicas]uint64
	nodes       map[uuid.UUID]*treeNode
	names       map[string]uuid.UUID
	root, trash *treeNode
	conn        network.ReplicaConn
	history     []LogOperation
	// Estadisticas
	LocalSum    time.Duration
	LocalCnt    uint64
	RemoteSum   time.Duration
	RemoteCnt   uint64
	UndoRedoCnt uint64
	PacketSzSum uint64
}

func NewTree(id int, serverIP string) *Tree {
	tree := Tree{}
	tree.id = uint64(id)
	tree.localTime = 1
	tree.nodes = make(map[uuid.UUID]*treeNode)
	tree.names = make(map[string]uuid.UUID)

	if NumReplicas < id {
		panic("Tree CRDT: Invalid ID")
	}

	tree.names[rootName] = rootID
	tree.nodes[rootID] = &treeNode{id: rootID, name: rootName}
	tree.root = tree.nodes[rootID]
	
	tree.names[trashName] = trashID
	tree.nodes[trashID] = &treeNode{id: trashID, name: trashName}
	tree.trash = tree.nodes[trashID]

	tree.conn = network.NewCausalConn(&tree, serverIP)
	// Esperar a que las demas replicas se inicien
	time.Sleep(5 * time.Second)

	// Iniciando corutina que cada 10 segundos limpiará el historial
	go func() {
		for range time.Tick(10 * time.Second) {
			tree.truncateHistory()
		}
	}()

	return &tree
}

func (this *Tree) GetID() int {
	return int(this.id)
}

func (this *Tree) ApplyRemoteOperation(data []byte) {
	this.Lock()
	defer this.Unlock()

	this.PacketSzSum += uint64(len(data))
	op := OperationFromBytes(data)
	op.Time = time.Now()
	this.apply(op)
}

func (this *Tree) truncateHistory() {
	this.Lock()
	defer this.Unlock()

	time := this.time[0]
	for _, t := range this.time {
		time = Min(time, t)
	}
	
	/* fmt.Println(this.time)
	for _, v := range this.history {
		fmt.Println(v)
	}*/
	
	start := HistoryUpperBound(this.history, time)
	this.history = this.history[start:]
	// log.Printf("History truncated to %d\n", len(this.history))
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

// aplicar la operacion op
// aca esta la magia, debe revertir ops del historial, aplicar la nueva op
// guardar la nueva op en el historial y reaplicar las ops del historial
// ignorando las ops invalidas
func (this *Tree) apply(op Operation) {
	undoRedoCnt := uint64(0)
	_, ok := this.nodes[op.Node]
	// Si la operacion crea un nodo no es necesario guardarlo en el historial
	if !ok {
		parentPtr, ok := this.nodes[op.NewParent]
		if !ok {
			log.Fatal("Parent does not exist:", op)
		}
		
		this.names[op.Name] = op.Node
		this.nodes[op.Node] = &treeNode{
			id:       op.Node,
			name:     op.Name,
			parent:   parentPtr,
		}
		parentPtr.children = append(parentPtr.children, this.nodes[op.Node])
	} else {
		// Creando registro en el historial
		this.history = append(this.history, LogOperation{
			ReplicaID: op.ReplicaID,
			Timestamp: op.Timestamp,
			NewParent: op.NewParent,
			Node:      op.Node,
		})

		// Revirtiendo registros con un timestamp mayor
		i := len(this.history) - 1
		for i > 0 && LogOperationBefore(this.history[i], this.history[i-1]) {
			this.history[i], this.history[i-1] = this.history[i-1], this.history[i]
			this.revert(i)
			i--
			undoRedoCnt++
		}

		// Aplicando la operacion y reaplicando operaciones revertidas
		for i < len(this.history) {
			this.reapply(i)
			i++
		}
	}

	// Transmision de actualizacion a otras replicas
	if op.ReplicaID == this.id {
		this.LocalCnt++
		this.LocalSum += time.Now().Sub(op.Time)
		data := OperationToBytes(op)
		this.PacketSzSum += uint64(len(data))
		this.conn.Send(data)
	} else {
		this.RemoteCnt++
		this.RemoteSum += time.Now().Sub(op.Time)
		this.UndoRedoCnt += undoRedoCnt
	}

	this.time[op.ReplicaID] = Max(this.time[op.ReplicaID], op.Timestamp)
	this.localTime = Max(this.localTime, op.Timestamp) + 1
}

// revierte un logmove si no ha sido ignorado
func (this *Tree) revert(i int) {
	if this.history[i].ignored {
		return
	}

	nodePtr := this.nodes[this.history[i].Node]
	parentPtr := this.nodes[this.history[i].OldParent]
	this.movePtr(nodePtr, parentPtr)
}

// reaplica un logmove o lo ignora
func (this *Tree) reapply(i int) {
	nodePtr := this.nodes[this.history[i].Node]
	parentPtr := this.nodes[this.history[i].NewParent]
	this.history[i].ignored = this.isDescendant(parentPtr, nodePtr)
	if this.history[i].ignored {
		return
	}

	this.history[i].OldParent = nodePtr.parent.id
	this.movePtr(nodePtr, parentPtr)
}

func (this *Tree) Add(name, parent string) error {
	this.Lock()
	defer this.Unlock()

	if _, ok := this.names[name]; ok {
		return errors.New("Name already exists")
	}

	parentID, ok := this.names[parent]
	if !ok || this.isDescendant(this.nodes[parentID], this.trash) {
		return errors.New("Parent does not exist")
	}

	op := Operation{
		ReplicaID: this.id,
		Timestamp: this.localTime,
		NewParent: parentID,
		Node:      uuid.New(),
		Name:      name,
		Time:      time.Now(),
	}
	this.apply(op)
	return nil
}

func (this *Tree) Move(node, newParent string) error {
	this.Lock()
	defer this.Unlock()

	nodeID, ok1 := this.names[node]
	parentID, ok2 := this.names[newParent]
	if !ok1 || this.isDescendant(this.nodes[nodeID], this.trash) {
		return errors.New("Node does not exist")
	} else if !ok2 || this.isDescendant(this.nodes[parentID], this.trash) {
		return errors.New("Parent does not exist")
	} else if nodeID == rootID {
		return errors.New("Cannot move root")
	} else if this.isDescendant(this.nodes[parentID], this.nodes[nodeID]) {
		return errors.New("Cannot move node to one of its decendants")
	}

	op := Operation{
		ReplicaID: this.id,
		Timestamp: this.localTime,
		NewParent: parentID,
		Node:      nodeID,
		Time:      time.Now(),
	}
	this.apply(op)
	return nil
}

func (this *Tree) Remove(node string) error {
	this.Lock()
	defer this.Unlock()

	nodeID, ok := this.names[node]
	if !ok || this.isDescendant(this.nodes[nodeID], this.trash) {
		return errors.New("Node does not exist")
	} else if nodeID == rootID {
		return errors.New("Cannot remove root")
	}

	op := Operation{
		ReplicaID: this.id,
		Timestamp: this.localTime,
		NewParent: trashID,
		Node:      nodeID,
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

	fmt.Println(rootName)
	printNode(this.root, "")
}

func printNode(node *treeNode, prefix string) {
	sort.Slice(node.children, func(i, j int) bool {
		return node.children[i].name < node.children[j].name
	})

	for index, child := range node.children {
		if index == len(node.children)-1 {
			fmt.Println(prefix+"└──", child.name)
			printNode(child, prefix+"    ")
		} else {
			fmt.Println(prefix+"├──", child.name)
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
