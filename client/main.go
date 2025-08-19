package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

func main() {
	// Conecta ao servidor na porta 8080
	conn, err := net.Dial("tcp", "server:8080")
	if err != nil {
		fmt.Println("Erro ao conectar:", err)
		return
	}
	defer conn.Close()

	fmt.Println("Conectado ao servidor!")

	reader := bufio.NewReader(os.Stdin)
	serverReader := bufio.NewReader(conn)

	for {
		// Lê entrada do usuário
		fmt.Print("Digite uma mensagem: ")
		text, _ := reader.ReadString('\n')

		// Envia para o servidor
		_, err := conn.Write([]byte(text))
		if err != nil {
			fmt.Println("Erro ao enviar:", err)
			return
		}

		// Recebe resposta do servidor
		message, err := serverReader.ReadString('\n')
		if err != nil {
			fmt.Println("Servidor desconectado.")
			return
		}
		fmt.Print("Resposta do servidor: " + message)
	}
}
