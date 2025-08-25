package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
)

type Player struct {
	ID    int    `json:"id"`
	Name  string `json:"name"`
    Password string `json:"password"`
	Conn  net.Conn `json:"-"`
	Duel  bool   `json:"duel"`
	Cards []Card `json:"cards"`
}

type Card struct {
	Name   string `json:"name"`
	Damage int    `json:"damage"`
	Rarity string `json:"rarity"`
}

type PlayerManager struct {
	mu   sync.Mutex
	file string
}

// cria um novo gerenciador ligado ao arquivo
func NewPlayerManager(file string) *PlayerManager {
	return &PlayerManager{file: file}
}

// Função utilitária para ler JSON
func (pm *PlayerManager) readPlayers() ([]Player, error) {
	file, err := os.ReadFile(pm.file)
	if err != nil {
		if os.IsNotExist(err) {
			return []Player{}, nil // se não existe, retorna lista vazia
		}
		return nil, err
	}
	var players []Player
	if err := json.Unmarshal(file, &players); err != nil {
		return nil, err
	}
	return players, nil
}

// Função utilitária para escrever JSON
func (pm *PlayerManager) writePlayers(players []Player) error {
	data, err := json.MarshalIndent(players, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(pm.file, data, 0644)
}

// Criar jogador
func (pm *PlayerManager) AddPlayer(conn net.Conn, name string, password string) (*Player, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	players, err := pm.readPlayers()
	if err != nil {
		return nil, err
	}

	newPlayer := Player{
		ID:    len(players),
		Name:  name,
        Password: password, 
		Conn:  conn,
		Duel:  false,
		Cards: nil,
	}

	players = append(players, newPlayer)

	if err := pm.writePlayers(players); err != nil {
		return nil, err
	}

	fmt.Println("Jogador adicionado:", newPlayer.Name)
	return &newPlayer, nil
}

// Listar jogadores
func (pm *PlayerManager) ListPlayers() ([]Player, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	return pm.readPlayers()
}

// Buscar jogador
func (pm *PlayerManager) GetPlayer(id int) (*Player, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	players, err := pm.readPlayers()
	if err != nil {
		return nil, err
	}
	for _, p := range players {
		if p.ID == id {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("jogador %d não encontrado", id)
}

func (pm *PlayerManager) Verify_Login(name string, password string) (bool){
    pm.mu.Lock()
    defer pm.mu.Unlock()
    players, err := pm.readPlayers()
	if err != nil {
		return false
	}
	for _, p := range players {
		if p.Name == name && p.Password == password {
			return true
		}
	}
	return false

}

// Remover jogador
func (pm *PlayerManager) RemovePlayer(id int) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	players, err := pm.readPlayers()
	if err != nil {
		return err
	}

	newPlayers := []Player{}
	found := false
	for _, p := range players {
		if p.ID != id {
			newPlayers = append(newPlayers, p)
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("jogador %d não encontrado", id)
	}

	// Reatribui IDs para manter consistência
	for i := range newPlayers {
		newPlayers[i].ID = i
	}

	return pm.writePlayers(newPlayers)
}
