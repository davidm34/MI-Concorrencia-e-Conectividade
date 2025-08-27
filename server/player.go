package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"sync"
)

type PlayerData struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Password string `json:"password"`
	Duel     bool   `json:"duel"`
	Cards    []Card `json:"cards"`
}

type Player struct {
	PlayerData
	Conn net.Conn `json:"-"`
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

func NewPlayerManager(file string) *PlayerManager {
	// Garante que o arquivo existe e é válido
	pm := &PlayerManager{file: file}
	pm.ensureFileExists()
	return pm
}

func (pm *PlayerManager) ensureFileExists() {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if _, err := os.Stat(pm.file); os.IsNotExist(err) {
		// Cria arquivo com array vazio
		emptyData := []PlayerData{}
		pm.writePlayerData(emptyData)
	}
}

func (pm *PlayerManager) readPlayerData() ([]PlayerData, error) {
	file, err := os.ReadFile(pm.file)
	if err != nil {
		return nil, fmt.Errorf("erro ao ler arquivo: %w", err)
	}
	
	// Verifica se o arquivo está vazio
	if len(file) == 0 {
		return []PlayerData{}, nil
	}
	
	var players []PlayerData
	if err := json.Unmarshal(file, &players); err != nil {
		return nil, fmt.Errorf("erro ao decodificar JSON: %w", err)
	}
	return players, nil
}

func (pm *PlayerManager) writePlayerData(players []PlayerData) error {
	data, err := json.MarshalIndent(players, "", "  ")
	if err != nil {
		return fmt.Errorf("erro ao serializar jogadores: %w", err)
	}

	tempFile := pm.file + ".tmp"
	err = os.WriteFile(tempFile, data, 0644)
	if err != nil {
		return fmt.Errorf("erro ao escrever arquivo temporário: %w", err)
	}

	err = os.Rename(tempFile, pm.file)
	if err != nil {
		return fmt.Errorf("erro ao renomear arquivo: %w", err)
	}

	return nil
}

// Criar jogador
func (pm *PlayerManager) AddPlayer(conn net.Conn, name string, password string) (*Player, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	playersData, err := pm.readPlayerData()
	if err != nil {
		return nil, fmt.Errorf("erro ao ler jogadores: %w", err)
	}

	// Verifica se jogador já existe
	for _, p := range playersData {
		if p.Name == name {
			return nil, fmt.Errorf("jogador já existe")
		}
	}

	newPlayerData := PlayerData{
		ID:       len(playersData),
		Name:     name,
		Password: password,
		Duel:     false,
		Cards:    nil,
	}

	playersData = append(playersData, newPlayerData)

	if err := pm.writePlayerData(playersData); err != nil {
		return nil, fmt.Errorf("erro ao salvar jogadores: %w", err)
	}

	// Cria o Player completo com a conexão
	newPlayer := &Player{
		PlayerData: newPlayerData,
		Conn:       conn,
	}

	fmt.Printf("Jogador adicionado: %s \n", newPlayer.Name)
	return newPlayer, nil
}

// Listar jogadores (retorna PlayerData)
func (pm *PlayerManager) ListPlayers() ([]PlayerData, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	return pm.readPlayerData()
}

// Buscar jogador (retorna Player com conexão se disponível)
func (pm *PlayerManager) GetPlayer(name string) (*Player, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	playersData, err := pm.readPlayerData()
	if err != nil {
		return nil, err
	}

	for i := range playersData {
		if playersData[i].Name == name {
			return &Player{
				PlayerData: playersData[i],
				Conn:       nil, // Conexão será setada posteriormente
			}, nil
		}
	}
	return nil, fmt.Errorf("jogador %s não encontrado", name)
}

func (pm *PlayerManager) Verify_Login(name string, password string) (bool, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	playersData, err := pm.readPlayerData()
	if err != nil {
		return false, err
	}

	for _, p := range playersData {
		if p.Name == name && p.Password == password {
			return true, nil
		}
	}
	return false, nil
}

// Atualizar dados do jogador (mantém a conexão se existir)
func (pm *PlayerManager) UpdatePlayer(player *Player) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	playersData, err := pm.readPlayerData()
	if err != nil {
		return err
	}

	found := false
	for i := range playersData {
		if playersData[i].ID == player.ID {
			// Atualiza os dados mas mantém a conexão original
			playersData[i] = player.PlayerData
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("jogador %d não encontrado", player.ID)
	}

	return pm.writePlayerData(playersData)
}

// Remover jogador
func (pm *PlayerManager) RemovePlayer(id int) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	playersData, err := pm.readPlayerData()
	if err != nil {
		return err
	}

	newPlayers := []PlayerData{}
	found := false
	for _, p := range playersData {
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

	return pm.writePlayerData(newPlayers)
}

func (p *Player) ToData() PlayerData {
	return p.PlayerData
}

func (p *Player) LoadData(data PlayerData) {
	p.PlayerData = data
	
}