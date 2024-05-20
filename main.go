package main

import (
	"udr-tree/replica"
	"log"
	"os"
	"bufio"
	"strings"
)

/*
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
*/

func main() {
	// os.argv[1]: puerto propio
	// os.argv[2:]: array de "ip:port" de las replicas
	tree = crdt.NewTree(os.argv[1], os.argv[2:])
	
	// para leer linea por linea
	// aca validar entrada del usuario y ejecutar operaciones
	// separar paths por "/"
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		// cmd es un array de strings separados por espacios
		cmd := strings.Fields(scanner.Text())
		
		// manejar los comandos, imprimir errores
		var err: Error = nil
		if .. {
			err = tree.Add(node, parent)
		} else if {
			err = tree.Move(node, parent)
		} else if{
			err = tree.Delete(node)
			
			tree.Print
			tree.Disconect
			tree.Connect
			tree.Close 
			tree.ChangeDir
			
		} else {
			err = Error{"Command not found"}
		}
			
		if err != nil {
			log.PrintLn(err)
		}
	}
}
