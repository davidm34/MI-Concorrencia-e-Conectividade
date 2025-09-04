package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
)


func main() {
	conn, err := net.Dial("tcp", "server:8080")
	if err != nil {
		fmt.Println("Erro ao conectar:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Conectado ao servidor!")

	reader := bufio.NewReader(os.Stdin)
	// serverReader := bufio.NewReader(conn)

	fmt.Print("Digite seu nome: ")
	name, _ := reader.ReadString('\n')
	conn.Write([]byte(name))

	



	// Loop principal para enviar mensagens APÓS o menu aparecer
	for {
		// Lê do terminal local
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		
		// Envia para o servidor
		conn.Write([]byte(text + "\n"))
	}


}


// // Loop de leitura (escuta mensagens do cliente)
// func (conn net.Conn) readerLoop() {
// 	reader := bufio.NewReader(conn)

// 	for {
// 		msg, err := reader.ReadString('\n')
// 		if err != nil {
// 			fmt.Printf("Cliente %s desconectado: %v\n", err) 
// 			conn.Close()
// 			return
// 		}
// 		msg = strings.TrimSpace(msg)
// 		fmt.Printf("[RECV] %s: %s\n")

// 		// aqui você pode repassar a mensagem para a lógica do jogo/sala
// 	}
// }