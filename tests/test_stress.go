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

func main() {
	if len(os.Args) != 4 {
		log.Fatal(errors.New("USE: ./test [id] [server_ip] [ops]"))
	}

	id, err := strconv.Atoi(os.Args[1])
	if err != nil {
		panic(err)
	}
	

	ops, err := strconv.Atoi(os.Args[3])
	if err != nil {
		panic(err)
	}
	

	log.SetPrefix("CRDT "+os.Args[1]+" ")
	tree := crdt.NewTree(id, os.Args[2])
	time.Sleep(5*time.Second)
	go func() {
		time.Sleep(time.Minute)
		log.Println("Operations per second:", (tree.LocalCnt + tree.RemoteCnt) / 60)
		log.Println("Avg delay for local operation:", tree.LocalSum / time.Duration(tree.LocalCnt))
		log.Println("Avg delay for remote operation:", tree.RemoteSum / time.Duration(tree.RemoteCnt))
		os.Exit(0)
	}()
	
	GenerateLoad(tree, ops)
}

/*
0: add 20%
1: remove 20%
2: move 60%
*/

func GenerateLoad(tree *crdt.Tree, ops int) {
	nodes := []string{"root"}
	
	for t := 0; t < 100; t++ {
		i := rand.Intn(len(nodes))
		name := uuid.New().String()
		nodes = append(nodes, name)
		tree.Add(name, nodes[i])
	}
	
	log.Println("Finished creating tree")
	time.Sleep(10*time.Second)
	cnt := 0
	
	go func() {
		for range time.Tick(time.Second) {
			cnt = 0
		}
	}()
	
	for {
		// Aca antes habia un time.Sleep
		// Pero no se porque reduce demasiado el rendimiento del bucle
		// Permitiendo solo 65 ops / sec
		if cnt >= ops {
			continue
		}
		
		x := rand.Intn(10)
		if x < 2 {
			i := rand.Intn(len(nodes))
			name := uuid.New().String()
			nodes = append(nodes, name)
			if tree.Add(name, nodes[i]) == nil {
				cnt++
			}
		} else if x < 4 {
			i := rand.Intn(len(nodes))
			if tree.Remove(nodes[i]) == nil {
				cnt++
			}
			
			nodes[i], nodes[len(nodes)-1] = nodes[len(nodes)-1], nodes[i]
			nodes = nodes[:len(nodes)-1]
		} else {
			i := rand.Intn(len(nodes))
			j := rand.Intn(len(nodes))
			if tree.Move(nodes[i], nodes[j]) == nil {
				cnt++
			}
		}
	}
}

