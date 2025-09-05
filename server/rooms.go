package main

import (
	"bufio"
	"fmt"
	"sync"
	"time"
	"math/rand"
	"strings"
	"strconv"
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
			msg = strings.TrimSpace(msg)
			if p.Duel {
				// manda escolha para um canal do game
				p.GameInput <- msg
			} else {
				// caso contrário, é mensagem de chat
				room.Broadcast(p, msg)
			}
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
	Game(room, p)
}


func Game(r *Room, p *Player) {
    
    DrawCards(p)

    p.Conn.Write([]byte("Cartas Sorteadas! Escolha uma carta nesse turno\n\n"))

    for i, c := range p.Cards {
        p.Conn.Write([]byte(fmt.Sprintf("Carta nº: %d\nNome: %s\nDano: %d\nRaridade: %s\n\n",
            i, c.Name, c.Damage, c.Rarity)))

	}
	p.Conn.Write([]byte("Digite o número da carta que deseja jogar: \n"))

    // reader := bufio.NewReader(p.Conn)
	p.Duel = true
	p.GameInput = make(chan string)
	
    for {

		choiceStr := <-p.GameInput 
		choice, err := strconv.Atoi(choiceStr)
		if err != nil || choice < 0 || choice >= len(p.Cards) {
			p.Conn.Write([]byte("Escolha inválida, tente novamente.\n"))
		} else {
			chosenCard := p.Cards[choice]
			p.Conn.Write([]byte(fmt.Sprintf("Você escolheu: %s (Dano: %d)\n",
				chosenCard.Name, chosenCard.Damage)))

			r.Broadcast(p, fmt.Sprintf("%s escolheu uma carta!\n", p.Name))
			
		}

    }
}


func DrawCards(p *Player){

	var cards []Card = []Card{
        {"Dragão Negro", 100, "Raro"},
        {"Guerreiro Valente", 50, "Comum"},
        {"Mago Arcano", 75, "Épico"},
        {"Arqueiro Élfico", 60, "Raro"},
        {"Cavaleiro da Luz", 85, "Lendário"},
        {"Feiticeira das Sombras", 70, "Épico"},
        {"Goblin Ladrão", 30, "Comum"},
        {"Dragão de Fogo", 95, "Raro"},
        {"Paladino Sagrado", 80, "Épico"},
        {"Ninja Silencioso", 65, "Raro"},
        {"Troll da Montanha", 90, "Épico"},
        {"Fada Curandeira", 45, "Comum"},
        {"Demônio do Abismo", 110, "Lendário"},
        {"Lobo Selvagem", 40, "Comum"},
        {"Sereia Encantadora", 55, "Raro"},
        {"Gigante de Pedra", 120, "Lendário"},
        {"Elfo da Floresta", 50, "Comum"},
        {"Vampiro Noturno", 85, "Épico"},
        {"Quimera Mística", 105, "Lendário"},
        {"Espadachim Ágil", 60, "Raro"},
    }

    p.Conn.Write([]byte("Começando os Sorteios das Cartas...\n"))
    time.Sleep(2 * time.Second)


	rand.Seed(time.Now().UnixNano())
    for i := 0; i < 3; i++ {
		pos := rand.Intn(len(cards))
		for cards[pos].Name == " " {
			pos = rand.Intn(len(cards))
		}
		p.Cards = append(p.Cards, cards[pos])
		cards = append(cards[:pos], cards[pos+1:]...)
    }

}