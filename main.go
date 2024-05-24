package main

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"strings"
	"udr-tree/crdt"
)

func isValidName(name string) bool {
	for _, char := range name {
		if char == '/' {
			return false
		}
	}
	
	return true
}

func isValidPath(path string) bool {
	names := strings.Split(path, "/")
	for i, name := range names {
		if (i > 0 && i < len(names)-1 && len(name) == 0) || !isValidName(name) {
			return false
		}
	}
	
	return true
}

func main() {
	helpMessage := `<name>: string sin "/" ni espacios
<path>: ./../<name>/<name> (relativo) o /<name>/<name>/<name> (absoluto)

- cd <path>: moverse a un nodo
- add <name> <path>: crear un nuevo nodo en el path con ese nombre
- rm <path>: eliminar nodo
- mv <path> <path> <name>: mover nodo a ser hijo de otro nodo con un nuevo nombre
- print: mostrar todo el sistema de archivos como arbol (similar a tree en CMD)
- connect: conectarse al servidor
- disconnect: desconectarse del servidor
- quit: salir de la app
- help: mostrar este mensaje`

	if len(os.args) < 2 {
		log.Fatal(errors.New("Use: ./udr-tree [port] [ip1] [ip2] ..."))
	}
	
	// os.argv[1]: puerto propio
	// os.argv[2:]: array de "ip:port" de las replicas
	tree := crdt.NewTree(os.args[1], os.args[2:])

	// para leer linea por linea
	// aca validar entrada del usuario y ejecutar operaciones
	// separar paths por "/"
	scanner := bufio.NewScanner(os.Stdin)
	fmt.Print("> ")
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
		case "cd":
			if len(cmd) > 1 && isValidPath(cmd[1]) {
				err = tree.ChangeDir(strings.Split(cmd[1], "/"))
			} else {
				err = errors.New("Invalid path")
			}
		case "add":
			if len(cmd) > 2 && isValidName(cmd[1]) && isValidPath(cmd[2]) {
				err = tree.Add(cmd[1], strings.Split(cmd[2], "/"))
			} else {
				err = errors.New("Invalid name or path")
			}
		case "rm":
			if len(cmd) > 1 && isValidPath(cmd[1]) {
				err = tree.Remove(strings.Split(cmd[1], "/"))
			} else {
				err = errors.New("Invalid path")
			}
		case "mv":
			if len(cmd) > 3 && isValidPath(cmd[1]) && isValidPath(cmd[2]) && isValidName(cmd[3]) {
				err = tree.Move(strings.Split(cmd[1], "/"), strings.Split(cmd[2], "/"), cmd[3])
			} else {
				err = errors.New("Invalid path or name")
			}
		case "print":
			tree.Print()
		case "connect":
			tree.Connect()
		case "disconnect":
			tree.Disconnect()
		case "quit":
			tree.Close()
			os.Exit(0)
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
