package main

import (
	"os"
	"udr-tree/crdt"
	"log"
	"time"
	"math/rand"
	"errors"
	"strconv"
	"github.com/google/uuid"
)

const (
	MaxN = 100
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal(errors.New("USE: ./udr-tree [id] [server_ip]"))
	}
	
	id, err := strconv.Atoi(os.Args[1])
	if err != nil {
		panic(err)
	}
	
	log.SetPrefix("CRDT "+os.Args[1]+" ")
	tree := crdt.NewTree(id, os.Args[2])
	time.Sleep(5*time.Second)
	GenerateLoad(tree)
	tree.Print()
	log.Println("CRDT finalized.")
}

/*
0: add 20%
1: remove 20%
2: move 50%
3: disconnect 10%
*/

func GenerateLoad(tree *crdt.Tree) {
	start := time.Now()
	log.Println("Start")
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
				// log.Println("Disconnected")
				tree.Disconnect()
			} else {
				// log.Println("Connected")
				tree.Connect()
			}
			
			connected = !connected
		}
	}
	
	if !connected {
		// log.Println("Connected")
		tree.Connect()
	}
	
	tree.Close()
	log.Println("Duration:", time.Now().Sub(start))
	time.Sleep(60*time.Second)
}
