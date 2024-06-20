/*
 * This Source Code Form is subject to the terms of the Mozilla Public
 * License, v. 2.0. If a copy of the MPL was not distributed with this
 * file, You can obtain one at https://mozilla.org/MPL/2.0/.
 */

package crdt

import (
	"log"
	"time"

	"github.com/google/uuid"
	"github.com/vmihailenco/msgpack/v5"
)

type Operation struct {
	// Para omitir campos en blanco
	_msgpack struct{} `msgpack:",omitempty"`

	ReplicaID uint64
	Timestamp uint64
	NewParent uuid.UUID
	Node      uuid.UUID
	Name      string
	time      time.Time
}

// Se usa MessagePack para serializar las operaciones
func OperationFromBytes(data []byte) Operation {
	var op Operation
	err := msgpack.Unmarshal(data, &op)
	if err != nil {
		log.Println("error decoding MessagePack")
		log.Println(string(data))
		log.Fatal(err)
	}

	return op
}

func OperationToBytes(op Operation) []byte {
	data, err := msgpack.Marshal(op)
	if err != nil {
		log.Fatal(err)
	}

	return data
}

type LogOperation struct {
	ReplicaID uint64
	Timestamp uint64
	OldParent uuid.UUID
	NewParent uuid.UUID
	Node      uuid.UUID
	ignored   bool
}

func LogOperationBefore(log1, log2 LogOperation) bool {
	if log1.Timestamp == log2.Timestamp {
		return log1.ReplicaID < log2.ReplicaID
	}

	return log1.Timestamp < log2.Timestamp
}

// https://cplusplus.com/reference/algorithm/upper_bound/
func HistoryUpperBound(history []LogOperation, time uint64) int {
	start := 0
	cnt := len(history)
	for cnt > 0 {
		step := cnt / 2
		i := start + step
		if history[i].Timestamp <= time {
			start = i + 1
			cnt -= step + 1
		} else {
			cnt = step
		}
	}

	return start
}

func Max(x, y uint64) uint64 {
	if x >= y {
		return x
	}

	return y
}

func Min(x, y uint64) uint64 {
	if x <= y {
		return x
	}

	return y
}
