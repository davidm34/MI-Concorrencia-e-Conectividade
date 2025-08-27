package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
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
	fmt.Print(response)

	// --- SE ENTRAR NA SALA ---
	fmt.Println("Agora você está em uma sala! Digite mensagens:")

	// goroutine para ouvir servidor
	go func() {
		for {
			msg, err := serverReader.ReadString('\n')
			if err != nil {
				fmt.Println("Servidor desconectado.")
				os.Exit(0)
			}
			fmt.Print(msg)
		}
	}()

	// loop para enviar mensagens
	for {
		text, _ := reader.ReadString('\n')
		conn.Write([]byte(text))
	}
}
