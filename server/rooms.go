package main

import (
	"bufio"
	"fmt"
	"sync"
	"time"
)

type Room struct {
	mu      sync.Mutex
	ID      int
	Players []*Player
	Actions map[int]bool
}

type RoomManager struct {
	mu     sync.Mutex
	rooms  []*Room
	nextID int
}

func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms:  []*Room{},
		nextID: 1,
	}
}

func (rm *RoomManager) AddPlayerRoom(p *Player) *Room {
	rm.mu.Lock()
	defer rm.mu.Unlock()

	for i := range rm.rooms {
		if len(rm.rooms[i].Players) < 2 {
			rm.rooms[i].Players = append(rm.rooms[i].Players, p)
			fmt.Printf("Jogador %s adicionado à sala %d\n", p.Name, rm.rooms[i].ID)

			if len(rm.rooms[i].Players) == 2 {
				fmt.Printf("Sala %d completa! Iniciando jogo...\n", rm.rooms[i].ID)
				for _, pl := range rm.rooms[i].Players {
					pl.Conn.Write([]byte("A partida começou! Boa sorte!\n"))
				}
			}
			return rm.rooms[i]
		}
	}

	// Cria sala nova
	room := &Room{
		ID:      rm.nextID,
		Players: []*Player{p},
		Actions: make(map[int]bool),
	}
	rm.nextID++
	rm.rooms = append(rm.rooms, room)

	fmt.Printf("Nova sala %d criada para jogador %s. Esperando adversário...\n", room.ID, p.Name)
	return room
}

func (r *Room) Broadcast(sender *Player, msg string) {
	for _, p := range r.Players {
		if p != sender {
			p.Conn.Write([]byte(sender.Name + ": " + msg))
		}
	}
}

func (r *Room) RemovePlayer(p *Player) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for i, player := range r.Players {
		if player == p {
			r.Players = append(r.Players[:i], r.Players[i+1:]...)
			break
		}
	}

	for _, player := range r.Players {
		player.Conn.Write([]byte(fmt.Sprintf("%s saiu da sala.\n", p.Name)))
	}
}

func HandlePlayer(p *Player, room *Room) {
	p.Duel = true

	// Goroutine de leitura contínua
	go func() {
		reader := bufio.NewReader(p.Conn)
		for {
			msg, err := reader.ReadString('\n')
			if err != nil {
				fmt.Printf("Jogador %s desconectou.\n", p.Name)
				room.RemovePlayer(p)
				return
			}
			room.Broadcast(p, msg)
		}
	}()

	// Espera os dois jogadores estarem prontos
	for {
		room.mu.Lock()
		totalPlayers := len(room.Players)
		room.mu.Unlock()

		if totalPlayers == 2 {
			break
		}
		time.Sleep(100 * time.Millisecond)
	}

	p.Conn.Write([]byte("Bem-vindo ao servidor!\n"))
}
