package main

import (
	"udr-tree/crdt"
	"log"
	"os"
	"bufio"
	"strings"
	"regexp"
	"errors"
)

func isValidName(name string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, name)
	return matched
}

func isValidPath(path string) bool {
	matched, _ := regexp.MatchString(`^(\.)?\/([a-zA-Z0-9_-]+|\.\.)(\/([a-zA-Z0-9_-]+|\.\.))*$`, path)
	return matched
}

func main() {
	helpMessage := `
	<name>: string sin "/" ni espacios
	<path>: ./../<name>/<name> (relativo) o /<name>/<name>/<name> (absoluto)

	- cd <path>: moverse a un nodo
	- add <name> <path>: crear un nuevo nodo en el path con ese nombre
	- rm <path>: eliminar nodo
	- mv <path> <path> <name>: mover nodo a ser hijo de otro nodo con un nuevo nombre
	- print: mostrar todo el sistema de archivos como arbol (similar a tree en CMD)
	- connect: conectarse al servidor
	- disconnect: desconectarse del servidor
	- quit: salir de la app
	- help: mostrar este mensaje
	`
	args := os.Args
	// os.argv[1]: puerto propio
	// os.argv[2:]: array de "ip:port" de las replicas
	tree := crdt.NewTree(args[1], args[2:])

	// para leer linea por linea
	// aca validar entrada del usuario y ejecutar operaciones
	// separar paths por "/"
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		// cmd es un array de strings separados por espacios
		cmd := strings.Fields(scanner.Text())
		
		// manejar los comandos, imprimir errores
		var err error
		switch cmd[0] {
			case "cd":
				if isValidPath(cmd[1]) {
					err = tree.ChangeDir(strings.Split(cmd[1], "/"))
				} else {
					err = errors.New("Invalid path")
				}
			case "add":
				if isValidName(cmd[1]) && isValidPath(cmd[2]) {
					err = tree.Add(cmd[1], strings.Split(cmd[2], "/"))
				} else {
					err = errors.New("Invalid name or path")
				}
			case "rm":
				if isValidPath(cmd[1]) {
					err = tree.Remove(strings.Split(cmd[1], "/"))
				} else {
					err = errors.New("Invalid path")
				}
			case "mv":
				if isValidPath(cmd[1]) && isValidPath(cmd[2]) && isValidName(cmd[3]) {
					err = tree.Move(cmd[1], strings.Split(cmd[2], "/"), cmd[3]) // tree.Move(node, parent)
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
				log.Println(helpMessage)
			default:
				err = errors.New("Command not found")
		}

		if err != nil {
			log.Println(err)
		}
	}
}