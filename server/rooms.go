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
	deck []Card
	nextID int
}

func NewRoomManager() *RoomManager {
	return &RoomManager{
		rooms:  []*Room{},
		deck: NewDeck(),
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

func HandlePlayer(p *Player, room *Room, pm *PlayerManager, rm *RoomManager) {

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
	Game(room, p, pm, rm)
}


func Game(r *Room, p *Player, pm *PlayerManager, rm *RoomManager) {
    
	// Sorteio das Cartas
    pm.DrawCards(r, p, rm)

    p.Conn.Write([]byte("Cartas Sorteadas! Escolha uma carta nesse turno\n\n"))

    for i, c := range p.Cards {
        p.Conn.Write([]byte(fmt.Sprintf("Carta nº: %d\nNome: %s\nDano: %d\nRaridade: %s\n\n",
            i, c.Name, c.Damage, c.Rarity)))

	}
	p.Conn.Write([]byte("Digite o número da carta que deseja jogar: \n"))

    // reader := bufio.NewReader(p.Conn)
	p.Duel = true
	p.GameInput = make(chan string)
    plays := 0 
	
	for {
		// enquanto plays < 2, o jogador continua escolhendo cartas
		for plays < 2 {
			choiceStr := <-p.GameInput
			choice, err := strconv.Atoi(choiceStr)

			if err != nil || choice < 0 || choice >= len(p.Cards) {
				p.Conn.Write([]byte("Escolha inválida, tente novamente.\n"))
				continue
			}

			chosenCard := p.Cards[choice]
			plays++ 
			p.Conn.Write([]byte(fmt.Sprintf(
				"Você escolheu: %s (Dano: %d) | Jogada %d de 2\n",
				chosenCard.Name, chosenCard.Damage, plays,
			)))

			r.Broadcast(p, fmt.Sprintf("%s escolheu uma carta!\n", p.Name))
		}

		plays = 0
		p.Conn.Write([]byte("Suas 2 jogadas foram feitas. Próxima rodada!\n"))
	}

}


func (pm *PlayerManager) DrawCards(r *Room, p *Player, rm *RoomManager){
	pm.mu.Lock()
    defer pm.mu.Unlock()

	
	p.Conn.Write([]byte("Começando os Sorteios das Cartas...\n"))
    time.Sleep(2 * time.Second)


	rand.Seed(time.Now().UnixNano())

    for i := 0; i < 3; i++ {
        if len(rm.deck) == 0 {
            break 
        }
        pos := rand.Intn(len(rm.deck))
        p.Cards = append(p.Cards, rm.deck[pos])

        rm.deck = append(rm.deck[:pos], rm.deck[pos+1:]...)
	}
}


func NewDeck() []Card {
	return []Card{
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
}