package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"udr-tree/crdt"
)

func main() {
	helpMessage := `COMMANDS
  add [name] [parent]	Add new node [name] to be child of [parent]
  rm [node]		Remove [node]
  mv [node] [parent]	Move [node] to be child of [parent]
  print			Show tree
  connect		Connect to other replicas
  disconnect		Disconnect from other replicas
  quit			Close app
  help			Show this message`

	if len(os.Args) < 3 {
		log.Fatal(errors.New("USE: ./udr-tree [id] [port] [ip1] [ip2] ..."))
	}

	id, err := strconv.Atoi(os.Args[1])
	if err != nil {
		log.Fatal(err)
	}

	tree := crdt.NewTree(uint64(id), os.Args[2], os.Args[3:])

	
	fmt.Print("> ")
	// para leer linea por linea
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		// cmd es un array de strings separados por espacios
		cmd := strings.Fields(scanner.Text())
		if len(cmd) == 0 {
			fmt.Print("> ")
			continue
		}

		// manejar los comandos, imprimir errores
		var err error
		switch cmd[0] {
		case "add":
			if len(cmd) >= 3 {
				err = tree.Add(cmd[1], cmd[2])
			} else {
				err = errors.New("Invalid command")
			}
		case "rm":
			if len(cmd) >= 2 {
				err = tree.Remove(cmd[1])
			} else {
				err = errors.New("Invalid command")
			}
		case "mv":
			if len(cmd) >= 3 {
				err = tree.Move(cmd[1], cmd[2])
			} else {
				err = errors.New("Invalid command")
			}
		case "print":
			tree.Print()
		case "connect":
			tree.Connect()
		case "disconnect":
			tree.Disconnect()
		case "quit":
			tree.Close()
			return
		case "help":
			fmt.Println(helpMessage)
		default:
			err = errors.New("Command not found")
		}

		if err != nil {
			log.Println(err)
		}

		fmt.Print("> ")
	}
}
