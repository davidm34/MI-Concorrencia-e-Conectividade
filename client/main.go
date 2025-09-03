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
	serverReader := bufio.NewReader(conn)

	// --- LOGIN / CADASTRO ---
	fmt.Print("[0] Login\n[1] Cadastro\nEscolha: ")
	option, _ := reader.ReadString('\n')
	conn.Write([]byte(option))

	fmt.Print("Digite seu nome: ")
	name, _ := reader.ReadString('\n')
	conn.Write([]byte(name))

	fmt.Print("Digite sua senha: ")
	pass, _ := reader.ReadString('\n')
	conn.Write([]byte(pass))

	// recebe resposta do servidor
	response, _ := serverReader.ReadString('\n')
	fmt.Println(response)

	// --- AGORA ESCUTA O SERVIDOR PARA SABER QUANDO MOSTRAR O MENU ---

	// Loop principal para enviar mensagens APÓS o menu aparecer
	for {
		// Lê do terminal local
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		
		// Envia para o servidor
		conn.Write([]byte(text + "\n"))
	}


}
