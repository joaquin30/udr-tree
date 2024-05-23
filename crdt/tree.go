package crdt

import (
	"fmt"
	"github.com/google/uuid"
	"log"
	"sync"
	"time"
)

const (
	rootUUID  = "4b16e69d-80e4-446e-bdd1-bd838d021718"
	trashUUID = "1f8535a6-e1dc-4256-8b60-c7cfe509993f"
)

type treeNode struct {
	id       uuid.UUID
	name     string
	parent   *treeNode
	children []*treeNode
}

type Tree struct {
	nodes                map[uuid.UUID]treeNode
	history              []LogMove
	root, trash, current *treeNode
	replicas             []*crdt.Replica
	closed               bool
	clock                LamportClock
	mtx                  sync.RWMutex
}

func NewTree(port string, replicaIPs []string) *Tree {
	tree := Tree{}
	rootID, _ := uuid.FromString(rootUUID)
	trashID, _ := uuid.FromString(trashUUID)
	tree.nodes[rootID] = treeNode{rootID}
	tree.root = &tree.nodes[rootID]
	tree.nodes[trashID] = treeNode{trashID}
	tree.trash = &tree.nodes[trashID]
	tree.current = tree.root
	tree.closed = false
	tree.clock = LamportClock{uuid.New(), 0}
	go tree.startListener(port)
	time.Sleep(5 * time.Second) // para que las demas replicas inicien
	tree.replicas = make([]*crdt.Replica, len(replicaIPs))
	for i := range tree.replicas {
		tree.replicas[i] = crdt.NewReplica(replicaIPs[i])
	}
	return &tree
}

func (this *Tree) startListener(port string) {
	ln, err = net.Listen("tcp", ":"+port)
	defer ln.Close()
	if err != nil {
		log.Fatal(error)
	}
	for {
		if tree.closed {
			break
		}
		conn, err := ln.Accept()
		if err != nil {
			log.Fatal(error)
		}
		go this.handleConnection(conn)
	}
}

func (this *Tree) handleConnection(conn net.Conn) {
	defer conn.Close()
	var buffer []byte
	for {
		if this.closed {
			break
		}
		len, err := conn.Read(buffer)
		// leer buffer manejar error
		// apply operation move in tree
		this.Apply(crdt.MoveFromBytes(buffer))
	}
}

// checkear si node1 es descendiente de node2
func (this *Tree) isDescendant(node1, node2 *treeNode) bool {
	this.mtx.RLock()
	defer this.mtx.RUnlock()
	if node2 == this.root {return true}
	if node1 == this.root {return false}
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
	this.mtx.RLock()
	defer this.mtx.RUnlock()
	var curr *treeNode = this.root
    if path[0] == "." || path[0] == ".." {
		curr = this.current
	}


	for i, dir := range path {
        fmt.Println("Processing directory:", dir)
		if dir == "." {
			if i == len(path)-1{
				return curr
			}
			continue
		}

		if dir == ".." {
			if curr.parent == nil {return nil}
			curr = curr.parent
			if i == len(path)-1{
				return curr
			}
			continue
		}

		for _, child := range curr.children {
			if child.name == dir {
				curr = child
				if i == len(path)-1{
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
	this.mtx.Lock()
	defer this.mtx.Unlock()

}

// add name .
func (this *Tree) Add(name string, parent []string) error {

}

// mv ./asd ../asda/as asd2
func (this *Tree) Move(node string, newParent []string, newName string) error {

}

// rm ./../asd
func (this *Tree) Remove(node []string) error {

}

// cambiar current_node a la path si existe
// cd ./../asd
func (this *Tree) ChangeDir(path []string) error {

}

// imprimir arbol como "cmd tree"
func (this *Tree) Print() {
	this.mtx.RLock()
	defer this.mtx.RUnlock()

}

func (this *Tree) Disconnect() {
	for i := range tree.replicas {
		this.replicas[i].Connected = false
	}
}

func (this *Tree) Connect() {
	for i := range tree.replicas {
		this.replicas[i].Connected = true
	}
}

func (this *Tree) Close() {
	this.closed = true
	for i := range tree.replicas {
		this.replicas[i].Close()
	}
}
