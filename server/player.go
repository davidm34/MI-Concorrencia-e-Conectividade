package main

import (
	"fmt"
	"net"
	"sync"
)

type Card struct {
	Name   string
	Damage int
	Rarity string
}

type Player struct {
	ID    int
	Name  string
	Conn  net.Conn
	Duel  bool
	SelectionRound bool
	Cards []Card
	GameInput chan string 
}

type PlayerManager struct {
	mu      sync.Mutex
	players []Player
}

func NewPlayerManager() *PlayerManager {
	return &PlayerManager{
		players: []Player{},
	}
}

// Criar jogador
func (pm *PlayerManager) AddPlayer(conn net.Conn, name string) (*Player, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	// Verifica se já existe
	for _, p := range pm.players {
		if p.Name == name {
			return nil, fmt.Errorf("jogador já existe")
		}
	}

	newPlayer := Player{
		ID:    len(pm.players),
		Name:  name,
		Conn:  conn,
		Duel:  false,
		SelectionRound: false,
		Cards: []Card{},
		GameInput: make(chan string),
	}

	pm.players = append(pm.players, newPlayer)
	fmt.Printf("Jogador adicionado: %s\n", newPlayer.Name)
	return &pm.players[len(pm.players)-1], nil
}


