/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package main

import (
	"errors"
	"log"
	"math/rand"
	"os"
	"strconv"
	"time"
	"udr-tree/crdt"

	"github.com/google/uuid"
)

const (
	total = 120
)

var (
	nodes = []string{"root"}
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

	log.SetPrefix("CRDT " + os.Args[1] + " ")
	tree := crdt.NewTree(id, os.Args[2])
	GenerateNodes(tree)
	go func() {
		time.Sleep(total * time.Second)
		log.Println("\n\tSeconds running:", total,
			// "\n\tOperations per second:", (tree.LocalCnt+tree.RemoteCnt)/total,
			"\n\tAvg delay for local operation:", tree.LocalSum/time.Duration(tree.LocalCnt),
			"\n\tAvg delay for remote operation:", tree.RemoteSum/time.Duration(tree.RemoteCnt),
			"\n\tAvg undo and redos for remote operation:", tree.UndoRedoCnt/tree.RemoteCnt,
			"\n\tAvg bandwidth consumption:", tree.PacketSzSum/total, "bps")
		os.Exit(0)
	}()
	GenerateLoad(tree, ops)
}

func GenerateNodes(tree *crdt.Tree) {
	for t := 0; t < 33; t++ {
		i := rand.Intn(len(nodes))
		name := uuid.New().String()
		nodes = append(nodes, name)
		tree.Add(name, nodes[i])
	}

	time.Sleep(5 * time.Second)
	nodes = tree.GetNames()
	tree.LocalCnt = 0
	tree.LocalSum = 0
	tree.RemoteCnt = 0
	tree.RemoteSum = 0
	tree.PacketSzSum = 0
	log.Printf("Generated %d nodes\n", len(nodes))
}

func GenerateLoad(tree *crdt.Tree, ops int) {
	rate := time.Second / time.Duration(ops)
	previous := time.Now()
	var delay time.Duration
	for {
		current := time.Now()
		delay += current.Sub(previous)
		previous = current

		for delay >= rate {
			// Se hacen las operaciones con un
			// 33.33% de probabilidad cada una
			x := rand.Intn(3)
			if len(nodes) <= 1 || x == 0 {
				i := rand.Intn(len(nodes))
				name := uuid.New().String()
				nodes = append(nodes, name)
				tree.Add(name, nodes[i])
			} else if x == 1 {
				i := rand.Intn(len(nodes)-1) + 1
				j := rand.Intn(len(nodes)-1) + 1
				tree.Move(nodes[i], nodes[j])
			} else {
				i := rand.Intn(len(nodes)-1) + 1
				tree.Remove(nodes[i])
				nodes[i] = nodes[len(nodes)-1]
				nodes = nodes[:len(nodes)-1]
			}

			delay -= rate
		}
	}
}
