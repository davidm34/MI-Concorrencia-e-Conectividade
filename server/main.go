package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"io"
	"strings"
)

var manager = NewPlayerManager("players.json")
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

    // Lê o tipo de operação (login ou cadastro)
    login, err := ReadPlayer(conn, reader)
    if err != nil {
        fmt.Printf("Erro ao ler operação: %v\n", err)
        return
    }

    var player *Player
    var room *Room

    switch login {
    case "0": // Login
        name, err := ReadPlayer(conn, reader)
        if err != nil {
            fmt.Printf("Erro ao ler nome: %v\n", err)
            return
        }
        password, err := ReadPlayer(conn, reader)
        if err != nil {
            fmt.Printf("Erro ao ler senha: %v\n", err)
            return
        }

        ok, err := manager.Verify_Login(name, password)
        if err != nil {
            conn.Write([]byte("Erro no login.\n"))
            fmt.Printf("Erro no login de %s: %v\n", name, err)
            return
        }
        
        if ok {
            conn.Write([]byte("Login realizado com sucesso.\n"))            
            fmt.Printf("Jogador %s logou.\n", name)

            // Atualiza conexão do jogador
            player, err = manager.GetPlayer(name)
            if err != nil {
                conn.Write([]byte("Erro ao carregar jogador.\n"))
                fmt.Printf("Erro ao carregar jogador %s: %v\n", name, err)
                return
            }
            
            player.Conn = conn  // Atualiza conexão
            room = rooms.AddPlayerRoom(player)
            
        } else {
            conn.Write([]byte("Login falhou.\n"))
            fmt.Printf("Login falhou para %s\n", name)
            return
        }

        go HandlePlayer(player, room)

    case "1": // Cadastro
        name, err := ReadPlayer(conn, reader)
        if err != nil {
            fmt.Printf("Erro ao ler nome (cadastro): %v\n", err)
            return
        }
        password, err := ReadPlayer(conn, reader)
        if err != nil {
            fmt.Printf("Erro ao ler senha (cadastro): %v\n", err)
            return
        }

        player, err = manager.AddPlayer(conn, name, password)
        if err != nil {
            conn.Write([]byte("Erro no cadastro.\n"))
            fmt.Printf("Erro no cadastro de %s: %v\n", name, err)
            return
        }
        
        conn.Write([]byte("Cadastro realizado com sucesso.\n"))
        room = rooms.AddPlayerRoom(player)

    default:
        conn.Write([]byte("Opção inválida. Digite 0 ou 1.\n"))
        fmt.Printf("Opção inválida recebida: %s\n", login)
        return
    }

    fmt.Printf("Iniciando HandlePlayer para %s\n", player.Name)
    go HandlePlayer(player, room)

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
