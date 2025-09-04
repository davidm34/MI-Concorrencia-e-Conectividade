package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"io"
	"strings"
)

var manager = NewPlayerManager()
var rooms = NewRoomManager()


func ReadPlayer(conn net.Conn, reader *bufio.Reader) (string, error) {
	message, err := reader.ReadString('\n')
	if err != nil {
		if err == io.EOF {
			fmt.Println("Jogador desconectou.")
		} else {
			fmt.Println("Erro ao ler do cliente:", err)
		}
		return "", err
	}

	message = strings.TrimSpace(message) // Remove '\n' e espaços extras
	if message != "" {
		_, err = conn.Write([]byte("Servidor recebeu: " + message + "\n"))
		if err != nil {
			fmt.Println("Erro ao enviar para o cliente:", err)
			return "", err
		}
	}
	return message, nil
}

func handleConnection(conn net.Conn) {
    defer conn.Close()
    reader := bufio.NewReader(conn)
    name, _ := ReadPlayer(conn, reader)
    player, _ := manager.AddPlayer(conn, name)
    if player != nil {
        room := rooms.AddPlayerRoom(player)
        go HandlePlayer(player, room)
    }

    

}


func main() {
	// Cria servidor na porta 8080
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Erro ao iniciar servidor:", err)
		os.Exit(1)
	}
	defer ln.Close()

	fmt.Println("Servidor TCP rodando na porta 8080...")

	for {
		// Aceita nova conexão
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Erro ao aceitar conexão:", err)
			continue
		}

		// Trata cliente em goroutine
		go handleConnection(conn)
	}
}
