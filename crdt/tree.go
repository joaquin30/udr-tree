/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package crdt

import (
	"errors"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"
	"udr-tree/network"

	"github.com/google/uuid"
)

const (
	rootName    = "root"
	NumReplicas = 3
)

var (
	rootID  = uuid.MustParse("5568143b-b80b-452d-ba1b-f9d333a06e7a")
	trashID = uuid.MustParse("cf0008e9-9845-465b-8c9b-fa4b31b7f3bd")
	nilID   = uuid.MustParse("00000000-0000-0000-0000-000000000000")
)

type treeNode struct {
	id       uuid.UUID
	name     string
	parent   *treeNode
	children []*treeNode
}

func (node treeNode) Debug() {
	fmt.Print(node.id.String())
	fmt.Print(" ")
	if node.parent == nil {
		fmt.Print("nil")
	} else {
		fmt.Print(node.parent.id.String())
	}

	sort.Slice(node.children, func(i, j int) bool {
		return node.children[i].id.String() < node.children[j].id.String()
	})

	fmt.Print(" [")
	for i, ptr := range node.children {
		fmt.Print(ptr.id.String())
		if i < len(node.children)-1 {
			fmt.Print(" ")
		}
	}

	fmt.Println("]")
}

type Tree struct {
	sync.Mutex
	id        uint64
	localTime uint64 // lamport clock
	time      [NumReplicas]uint64
	nodes     map[uuid.UUID]*treeNode
	names     map[string]uuid.UUID
	conn      network.ReplicaConn
	history   []LogOperation
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
	tree.nodes[trashID] = &treeNode{id: trashID, name: "__trash"}
	tree.nodes[nilID] = &treeNode{id: nilID, name: "__nil"}

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

func (tree *Tree) exists(id uuid.UUID) bool {
	_, ok := tree.nodes[id]
	return ok
}

// checkear si node1 es descendiente de node2
func (tree *Tree) descendant(id1, id2 uuid.UUID) bool {
	if !tree.exists(id1) || !tree.exists(id2) {
		log.Fatal("descendant: node does not exist")
	}

	curr := tree.nodes[id1]
	node2 := tree.nodes[id2]
	for curr != nil {
		if curr == node2 {
			return true
		} else {
			curr = curr.parent
		}
	}

	return false
}

// checkear si node1 es descendiente de node2
// nota: esta funcion esta comentada en Add Remove y Move
// debido al test de stress, es posible que se elimine un nodo
// bastante cerca al arbol y que el 90% de los nodos ya no sirvan
func (tree *Tree) deleted(id uuid.UUID) bool {
	if !tree.exists(id) {
		log.Fatal("deleted: node does not exist")
	}

	return tree.descendant(id, trashID)
}

// cambiar el puntero de posicion
func (tree *Tree) moveInternal(id, parentId uuid.UUID) {
	if !tree.exists(id) || !tree.exists(parentId) {
		log.Fatal("moveInternal: node does not exist")
	}

	node := tree.nodes[id]
	newParent := tree.nodes[parentId]
	for i, child := range node.parent.children {
		if child.id == node.id {
			node.parent.children[i] = node.parent.children[len(node.parent.children)-1]
			node.parent.children = node.parent.children[:len(node.parent.children)-1]
		}
	}

	node.parent = newParent
	newParent.children = append(newParent.children, node)
}

// aplicar la operacion op
// aca esta la magia, debe revertir ops del historial, aplicar la nueva op
// guardar la nueva op en el historial y reaplicar las ops del historial
// ignorando las ops invalidas
func (tree *Tree) apply(op Operation) {
	undoRedoCnt := uint64(0)
	// Creación de nodo implícito
	if !tree.exists(op.Node) {
		tree.names[op.Name] = op.Node
		tree.nodes[op.Node] = &treeNode{
			id:     op.Node,
			name:   op.Name,
			parent: tree.nodes[nilID],
		}
	}
	// Creando registro en el historial
	tree.history = append(tree.history, LogOperation{
		ReplicaID: op.ReplicaID,
		Timestamp: op.Timestamp,
		NewParent: op.NewParent,
		Node:      op.Node,
	})

	// Revirtiendo registros con un timestamp mayor
	i := len(tree.history) - 1
	for i > 0 && LogOperationBefore(tree.history[i], tree.history[i-1]) {
		tree.history[i], tree.history[i-1] = tree.history[i-1], tree.history[i]
		tree.revert(&tree.history[i])
		i--
		undoRedoCnt++
	}

	// Aplicando la operacion y reaplicando operaciones revertidas
	for i < len(tree.history) {
		tree.reapply(&tree.history[i])
		i++
	}

	if op.ReplicaID == tree.id {
		// Transmision de actualizacion a otras replicas
		tree.LocalCnt++
		tree.LocalSum += time.Since(op.time)
		data := OperationToBytes(op)
		tree.PacketSzSum += uint64(len(data))
		tree.conn.Send(data)
	} else {
		tree.RemoteCnt++
		tree.RemoteSum += time.Since(op.time)
		tree.UndoRedoCnt += undoRedoCnt
	}

	tree.time[op.ReplicaID] = Max(tree.time[op.ReplicaID], op.Timestamp)
	tree.localTime = Max(tree.localTime, op.Timestamp) + 1
}

// revierte un logmove si no ha sido ignorado
func (tree *Tree) revert(op *LogOperation) {
	if op.ignored {
		return
	}

	tree.moveInternal(op.Node, op.OldParent)
}

// reaplica un logmove o lo ignora
func (tree *Tree) reapply(op *LogOperation) {
	op.ignored = !tree.exists(op.NewParent) || tree.descendant(op.NewParent, op.Node)
	if op.ignored {
		return
	}

	op.OldParent = tree.nodes[op.Node].parent.id
	tree.moveInternal(op.Node, op.NewParent)
}

func (tree *Tree) truncateHistory() {
	tree.Lock()
	defer tree.Unlock()

	time := tree.time[0]
	for _, t := range tree.time {
		time = Min(time, t)
	}

	start := HistoryUpperBound(tree.history, time)
	tree.history = tree.history[start:]
}

func (tree *Tree) GetID() int {
	return int(tree.id)
}

func (tree *Tree) ApplyRemoteOperation(data []byte) {
	tree.Lock()
	defer tree.Unlock()

	tree.PacketSzSum += uint64(len(data))
	op := OperationFromBytes(data)
	op.time = time.Now()
	tree.apply(op)
}

func (tree *Tree) Add(name, parent string) error {
	tree.Lock()
	defer tree.Unlock()

	if _, ok := tree.names[name]; ok {
		return errors.New("add: name already exists")
	}

	parentID, ok := tree.names[parent]
	if !ok /* || tree.deleted(parentID) */ {
		return errors.New("add: parent does not exist")
	}

	op := Operation{
		ReplicaID: tree.id,
		Timestamp: tree.localTime,
		NewParent: parentID,
		Node:      uuid.New(),
		Name:      name,
		time:      time.Now(),
	}
	tree.apply(op)
	return nil
}

func (tree *Tree) Move(node, newParent string) error {
	tree.Lock()
	defer tree.Unlock()

	nodeID, ok1 := tree.names[node]
	parentID, ok2 := tree.names[newParent]
	if !ok1 /* || tree.deleted(nodeID) */ {
		return errors.New("move: node does not exist")
	} else if !ok2 /* || tree.deleted(parentID) */ {
		return errors.New("move: parent does not exist")
	} else if nodeID == rootID {
		return errors.New("move: cannot move root")
	} else if tree.descendant(parentID, nodeID) {
		return errors.New("move: cannot move node to one of its decendants")
	} else if parentID == tree.nodes[nodeID].parent.id {
		return errors.New("move: new parent is already the parent of node")
	}

	op := Operation{
		ReplicaID: tree.id,
		Timestamp: tree.localTime,
		NewParent: parentID,
		Node:      nodeID,
		time:      time.Now(),
	}
	tree.apply(op)
	return nil
}

func (tree *Tree) Remove(node string) error {
	tree.Lock()
	defer tree.Unlock()

	nodeID, ok := tree.names[node]
	// se comenta la condicion para evitar problemas en el stress test
	if !ok /* || tree.deleted(nodeID) */ {
		return errors.New("remove: node does not exist")
	} else if nodeID == rootID {
		return errors.New("remove: cannot remove root")
	}

	op := Operation{
		ReplicaID: tree.id,
		Timestamp: tree.localTime,
		NewParent: trashID,
		Node:      nodeID,
		time:      time.Now(),
	}
	tree.apply(op)
	return nil
}

// imprimir tree.nodes de forma ordenada
func (tree *Tree) Debug() {
	tree.Lock()
	defer tree.Unlock()

	fmt.Println(tree.RemoteCnt+tree.LocalCnt, "operations")
	var keys []uuid.UUID
	for k := range tree.nodes {
		keys = append(keys, k)
	}

	sort.Slice(keys, func(i, j int) bool {
		return keys[i].String() < keys[j].String()
	})

	for _, k := range keys {
		tree.nodes[k].Debug()
	}
}

// imprimir arbol de forma bonita
func (tree *Tree) Print() {
	tree.Lock()
	defer tree.Unlock()

	fmt.Println(rootName)
	printInternal(tree.nodes[rootID], "")
}

func printInternal(node *treeNode, prefix string) {
	sort.Slice(node.children, func(i, j int) bool {
		return node.children[i].name < node.children[j].name
	})

	for index, child := range node.children {
		if index == len(node.children)-1 {
			fmt.Println(prefix+"└──", child.name)
			printInternal(child, prefix+"    ")
		} else {
			fmt.Println(prefix+"├──", child.name)
			printInternal(child, prefix+"│   ")
		}
	}
}

func (tree *Tree) Disconnect() {
	tree.conn.Disconnect()
}

func (tree *Tree) Connect() {
	tree.conn.Connect()
}

func (tree *Tree) Close() {
	tree.conn.Close()
}

func (tree *Tree) GetNames() []string {
	tree.Lock()
	defer tree.Unlock()

	return getNamesInternal(tree.nodes[rootID], []string{})
}

func getNamesInternal(node *treeNode, names []string) []string {
	names = append(names, node.name)
	for _, child := range node.children {
		names = getNamesInternal(child, names)
	}

	return names
}
