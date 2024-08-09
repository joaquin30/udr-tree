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
		time.Sleep(120 * time.Second)
		os.Exit(0)
	}()
	GenerateLoad(tree, ops)
	time.Sleep(10 * time.Second)
	log.Println("\n\tTotal time:", tree.LocalSum+tree.RemoteSum,
		"\n\tLocal operations:", tree.LocalCnt,
		"\n\tRemote operations:", tree.RemoteCnt,
		"\n\tAvg delay for local operation:", tree.LocalSum/time.Duration(tree.LocalCnt),
		"\n\tAvg delay for remote operation:", tree.RemoteSum/time.Duration(tree.RemoteCnt),
		"\n\tAvg undo and redos for remote operation:", tree.UndoRedoCnt/tree.RemoteCnt,
		"\n\tAvg packet size:", tree.PacketSzSum/(tree.LocalCnt+tree.RemoteCnt), "bytes")
}

func GenerateNodes(tree *crdt.Tree) {
	for t := 0; t < 333; t++ {
		i := rand.Intn(len(nodes))
		name := uuid.New().String()
		nodes = append(nodes, name)
		tree.Add(name, nodes[i])
	}

	time.Sleep(10 * time.Second)
	nodes = tree.GetNames()
	tree.LocalCnt = 0
	tree.LocalSum = 0
	tree.RemoteCnt = 0
	tree.RemoteSum = 0
	tree.PacketSzSum = 0
	log.Println("Generated nodes:", len(nodes))
}

func GenerateLoad(tree *crdt.Tree, ops int) {
	for t := 0; t < ops; t++ {
		// Se hacen las operaciones con un
		// 33.33% de probabilidad cada una
		x := rand.Intn(3)
		if len(nodes) <= 1 || x == 0 {
			i := rand.Intn(len(nodes))
			name := uuid.New().String()
			nodes = append(nodes, name)
			log.Println("add start", t)
			tree.Add(name, nodes[i])
			log.Println("add end", t)
		} else if x == 1 {
			i := rand.Intn(len(nodes)-1) + 1
			j := rand.Intn(len(nodes)-1) + 1
			log.Println("mv start", t)
			tree.Move(nodes[i], nodes[j])
			log.Println("mv end", t)
		} else {
			i := rand.Intn(len(nodes)-1) + 1
			log.Println("rm start", t)
			tree.Remove(nodes[i])
			log.Println("rm end", t)
			nodes[i] = nodes[len(nodes)-1]
			nodes = nodes[:len(nodes)-1]
		}
	}
}
