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

	message = strings.TrimSpace(message)
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
    reader := bufio.NewReader(conn)

    name, err := ReadPlayer(conn, reader)
    if err != nil {
        conn.Close()
        return
    }

    player, err := manager.AddPlayer(conn, name)
    if err != nil {
        conn.Write([]byte("Erro: " + err.Error() + "\n"))
        conn.Close()
        return
    }

    room := rooms.AddPlayerRoom(player)
    go HandlePlayer(player, room, manager, rooms)
}


func main() {
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Erro ao iniciar servidor:", err)
		os.Exit(1)
	}
	defer ln.Close()

	serverUdp, err := net.ListenPacket("udp", ":8081")
	if err != nil {
		fmt.Println("Erro ao iniciar servidor UDP:", err)
		os.Exit(1)
	}
	defer serverUdp.Close()

	buffer := make([]byte, 1024)

	fmt.Println("Servidor TCP rodando na porta 8080...")

	for {
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Erro ao aceitar conexão:", err)
			continue
		}
		// Calculo de Ping
		n, clientAddr, err := serverUdp.ReadFrom(buffer)
		if err != nil {
			fmt.Println("Erro ao ler:", err)
			continue
		}

		msg := string(buffer[:n])
		fmt.Printf("Recebido de %v: %s\n", clientAddr, msg)

		// ecoa de volta
		_, err = serverUdp.WriteTo([]byte(msg), clientAddr)
		if err != nil {
			fmt.Println("Erro ao enviar resposta:", err)
		}


		go handleConnection(conn)
	}
}
