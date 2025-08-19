package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"sync"
)

var manager = NewPlayerManager()

type Player struct {
    ID     int
    Name   string
    Conn   net.Conn
    Status string // ex: "livre", "em_duelo"
}


type PlayerManager struct {
    players map[int]*Player
	mu 		sync.Mutex
    nextID  int
}

func NewPlayerManager() *PlayerManager {
    return &PlayerManager{
        players: make(map[int]*Player),
        nextID:  1,
    }
}

func (pm *PlayerManager) AddPlayer(conn net.Conn, name string) *Player {    
    pm.mu.Lock()
    defer pm.mu.Unlock()

    player := &Player{
        ID:     pm.nextID,
        Name:   name,
        Conn:   conn,
        Status: "livre",
    }

    pm.players[player.ID] = player
    pm.nextID++

    return player
}

func (pm *PlayerManager) RemovePlayer(id int) {
    pm.mu.Lock()
    defer pm.mu.Unlock()
    delete(pm.players, id)
}

func (pm *PlayerManager) ListPlayers() []*Player {
    pm.mu.Lock()
    defer pm.mu.Unlock()

    list := []*Player{}
    for _, p := range pm.players {
        list = append(list, p)
    }
    return list
}


func handleConnection(conn net.Conn) {
    defer conn.Close()

    // cria novo jogador (nome temporário com ID gerado pelo manager)
    player := manager.AddPlayer(conn, fmt.Sprintf("Jogador%d", manager.nextID))
    fmt.Println("Novo jogador conectado:", player.Name, player.Conn.RemoteAddr())

    reader := bufio.NewReader(conn)
    for {
        message, err := reader.ReadString('\n')
        if err != nil {
            fmt.Println("Jogador saiu:", player.Name)
            manager.RemovePlayer(player.ID)
            return
        }

        fmt.Printf("[%s]: %s", player.Name, message)
        conn.Write([]byte("Servidor recebeu: " + message))
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
