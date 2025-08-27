package main

import (
	"bufio"
	"fmt"
	"sync"
)

type Room struct {
    ID      int
    Players []*Player
}

type RoomManager struct {
    mu          sync.Mutex
    rooms       []*Room
    nextID      int
    waitingPool []*Player  // Pool de jogadores esperando
}

func NewRoomManager() *RoomManager {
    return &RoomManager{
        rooms:       []*Room{},
        waitingPool: []*Player{},
        nextID:      1,
    }
}

func (rm *RoomManager) AddPlayerRoom(p *Player) *Room {
    rm.mu.Lock()
    defer rm.mu.Unlock()

    fmt.Printf("Adicionando jogador %s. Salas existentes: %d\n", p.Name, len(rm.rooms))

    // Primeiro tenta encontrar uma sala com vaga
    for i := range rm.rooms {
        if len(rm.rooms[i].Players) < 2 {
            rm.rooms[i].Players = append(rm.rooms[i].Players, p)
            fmt.Printf("Jogador %s adicionado à sala %d\n", p.Name, rm.rooms[i].ID)
            
            // Se a sala ficou completa
            if len(rm.rooms[i].Players) == 2 {
                fmt.Printf("Sala %d completa! Iniciando jogo...\n", rm.rooms[i].ID)
            }
            return rm.rooms[i]
        }
    }

    // Se não encontrou sala, cria uma nova
    room := &Room{
        ID:      rm.nextID,
        Players: []*Player{p},
    }
    rm.nextID++
    rm.rooms = append(rm.rooms, room)
    fmt.Printf("Nova sala %d criada para jogador %s. Esperando adversário...\n", room.ID, p.Name)
    return room
}

// broadcast dentro da sala
func (r *Room) Broadcast(sender *Player, msg string) {
	for _, p := range r.Players {
		if p != sender {
			p.Conn.Write([]byte(sender.Name + ": " + msg))
		}
	}
}

// exemplo de loop para ler mensagens do jogador e redirecionar para sala
func HandlePlayer(p *Player, room *Room) {
	reader := bufio.NewReader(p.Conn)
	for {
		msg, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Jogador saiu:", p.Name)
			return
		}
		room.Broadcast(p, msg)
	}
}
