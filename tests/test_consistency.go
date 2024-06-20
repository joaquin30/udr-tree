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
	MaxN = 200
)

func main() {
	if len(os.Args) != 3 {
		log.Fatal(errors.New("USE: ./udr-tree [id] [server_ip]"))
	}

	id, err := strconv.Atoi(os.Args[1])
	if err != nil {
		panic(err)
	}

	log.SetPrefix("CRDT " + os.Args[1] + " ")
	tree := crdt.NewTree(id, os.Args[2])
	GenerateLoad(tree)
	tree.Debug()
	log.Println("Finalized")
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
	for t := 0; t < 33; t++ {
		i := rand.Intn(len(nodes))
		name := uuid.New().String()
		tree.Add(name, nodes[i])
		nodes = append(nodes, name)
	}

	time.Sleep(5 * time.Second)
	nodes = tree.GetNames()
	cnt := 0
	tree.Disconnect()
	for {
		if cnt >= MaxN {
			break
		}

		if cnt%50 == 0 {
			tree.Connect()
			time.Sleep(time.Second)
			tree.Disconnect()
		}

		cnt++
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
		} else {
			i := rand.Intn(len(nodes))
			j := rand.Intn(len(nodes))
			tree.Move(nodes[i], nodes[j])
		}
	}

	tree.Connect()
	tree.Close()
	log.Println("Duration:", time.Now().Sub(start))
	log.Println("Waiting for eventual consistency")
	time.Sleep(5 * time.Second)
}
