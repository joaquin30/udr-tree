package main

import (
	"os"
	"strconv"
	"udr-tree/crdt"
	"fmt"
	"log"
	"time"
	"math/rand"
	"errors"
	"github.com/google/uuid"
)

func main() {
	if len(os.Args) < 4 {
		log.Fatal(errors.New("USE: ./test [ops] [id] [port] [ip1] [ip2] ..."))
	}

	ops, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
	
	id, err := strconv.Atoi(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}

	tree := crdt.NewTree(uint64(id), os.Args[3], os.Args[4:])
	go func() {
		time.Sleep(time.Minute)
		os.Exit(0);
	}()
	
	GenerateLoad(tree, ops, id)
}

/*
0: add 20%
1: remove 20%
2: move 60%
*/

func GenerateLoad(tree *crdt.Tree, ops, id int) {
	nodes := []string{"root"}
	
	for t := 0; t < 1000; t++ {
		i := rand.Intn(len(nodes))
		name := uuid.New().String()
		nodes = append(nodes, name)
		tree.Add(name, nodes[i])
	}
	
	log.Println("Finished creating tree", id)
	time.Sleep(5*time.Second)
	cnt := 0
	
	go func() {
		for range time.Tick(time.Second) {
			fmt.Println(cnt)
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
		
		i := rand.Intn(len(nodes))
		j := rand.Intn(len(nodes))
		tree.Move(nodes[i], nodes[j])
		cnt++
	}
}

