package main

import (
	"os"
	"strconv"
	"udr-tree/crdt"
	"log"
	"time"
	"math/rand"
	"errors"
	"github.com/google/uuid"
)

const (
	MaxN = 1000
)

func main() {
	if len(os.Args) < 3 {
		log.Fatal(errors.New("USE: ./udr-tree [id] [port] [ip1] [ip2] ..."))
	}

	id, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	tree := crdt.NewTree(uint64(id), os.Args[2], os.Args[3:])
	GenerateLoad(tree, id)
	tree.Print()
	log.Println("CRDT", id, "finalized.")
}

/*
0: add 20 %
1: remove 20 %
2: move 50 %
3: disconnect 10%
*/

func GenerateLoad(tree *crdt.Tree, id int) {
	start := time.Now()
	nodes := []string{"root"}
	for t := 0; t < MaxN; t++ {
		i := rand.Intn(len(nodes))
		name := uuid.New().String()
		nodes = append(nodes, name)
		tree.Add(name, nodes[i])
	}
	
	connected := true
	for t := 0; t < MaxN; t++ {
		x := rand.Intn(10)
		if x < 2 {
			i := rand.Intn(len(nodes))
			name := uuid.New().String()
			nodes = append(nodes, name)
			tree.Add(name, nodes[i])
		} else if x < 4 {
			i := rand.Intn(len(nodes))
			tree.Remove(nodes[i])
			nodes[i], nodes[len(nodes)-1] = nodes[len(nodes)-1], nodes[i]
			nodes = nodes[:len(nodes)-1]
		} else if x < 8 {
			i := rand.Intn(len(nodes))
			j := rand.Intn(len(nodes))
			tree.Move(nodes[i], nodes[j])
		} else {
			if connected {
				connected = false
				tree.Disconnect()
			} else {
				connected = true
				tree.Connect()
			}
		}
	}
	
	tree.Connect()
	tree.Close()
	log.Println("Time CRDT", id, time.Now().Sub(start))
	time.Sleep(10*time.Second)
	log.Println("Operations CRDT", id, tree.Counter)
}
