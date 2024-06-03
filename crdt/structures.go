package crdt

import (
	"log"
	"time"
	"encoding/json"
)

type Move struct {
	ReplicaID uint64
	Timestamp uint64
	NewParent string
	Node      string
	Time 	  time.Time
}

// definir estructura de los bytes

func MoveFromBytes(data []byte) Move {
	var move Move
	err := json.Unmarshal(data, &move)
	if err != nil {
		log.Println("Error parsing JSON")
		log.Println(string(data))
		log.Fatal(err)
	}
	
	return move
}

func MoveToBytes(move Move) []byte {
	bin, err := json.Marshal(move)
	if err != nil {
		log.Fatal(err)
	}
	
	return bin
}

type LogMove struct {
	ReplicaID uint64
	Timestamp uint64
	OldParent string
	NewParent string
	Node      string
	ignored   bool
}

func LogMoveBefore(log1, log2 LogMove) bool {
	if log1.Timestamp == log2.Timestamp {
		return log1.ReplicaID < log2.ReplicaID
	}

	return log1.Timestamp < log2.Timestamp
}
