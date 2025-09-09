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
	Actions map[*Player]Card
	Cards [2]Card
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
		Actions: make(map[*Player]Card),
		Cards: [2]Card{},
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

    // reader := bufio.NewReader(p.Conn)
	p.Duel = true
	p.GameInput = make(chan string)


	player0Winner := 0	
	player1Winner := 0
	for { 

		for i, c := range p.Cards {
        p.Conn.Write([]byte(fmt.Sprintf("Carta nº: %d\nNome: %s\nDano: %d\nRaridade: %s\n\n",
            i, c.Name, c.Damage, c.Rarity)))

		}

		p.Conn.Write([]byte("Digite o número da carta que deseja jogar: \n"))
		choiceStr := <-p.GameInput 
		choice, err := strconv.Atoi(choiceStr) 

		if err != nil || choice < 0 || choice >= len(p.Cards) { 
			p.Conn.Write([]byte("Escolha inválida, tente novamente.\n")) 
		} else { 			
			chosenCard := p.Cards[choice]			
			r.Broadcast(p, fmt.Sprintf("%s escolheu uma carta!\n", p.Name))

			for i := 0; i < 2; i++ {
				if r.Players[i].ID == p.ID {
					p.SelectionRound = true
					p.Conn.Write([]byte(fmt.Sprintf("Você escolheu: %s (Dano: %d)\n", chosenCard.Name, chosenCard.Damage))) 
				} 
			} 

			for (!r.Players[0].SelectionRound && r.Players[1].SelectionRound) || (r.Players[0].SelectionRound && !r.Players[1].SelectionRound) {
				p.Conn.Write([]byte("Aguardando a jogada do adversário... \n"))
				time.Sleep(4 * time.Second)
			} 
			

			if r.Players[0].ID == p.ID {
				r.Cards[0] = chosenCard
			} else {
				r.Cards[1] = chosenCard
			}

			if r.Cards[0].Damage > r.Cards[1].Damage {
				p.Conn.Write([]byte("\nJogador: " + r.Players[0].Name + " Vencedor da Rodada\n\n"))
				player0Winner++
			} else if r.Cards[1].Damage > r.Cards[0].Damage {
				p.Conn.Write([]byte("\nJogador: " + r.Players[1].Name + " Vencedor da Rodada\n\n"))
				player1Winner++
			} else {
				p.Conn.Write([]byte("Rodada Empatada!\n"))
			}

			
			p.Cards = append(p.Cards[:choice], p.Cards[choice+1:]...)

			for len(p.Cards) == 0 {
				p.Conn.Write([]byte("Jogo Finalizado! \n"))
				if player0Winner > player1Winner {
					p.Conn.Write([]byte("\nJogador: " + r.Players[0].Name + " Vencedor da Partida\n\n"))
				} else if player1Winner > player0Winner {
					p.Conn.Write([]byte("\nJogador: " + r.Players[1].Name + " Vencedor da Partida\n\n"))
				} else {
					p.Conn.Write([]byte("\nPartida Empatada!\n\n"))
				}
				time.Sleep(10 * time.Second)
			}
			
			
			r.Players[0].SelectionRound = false
			r.Players[1].SelectionRound = false
		} 
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
        // Sertanejo
        {"Pablo", 100, "Lendário"},
        {"Gusttavo Lima", 95, "Lendário"},
        {"Ana Castela", 90, "Lendário"},
        {"Chitãozinho e Xororó", 85, "Épico"},
        {"Henrique e Juliano", 80, "Épico"},
        {"Zezé Di Camargo & Luciano", 75, "Épico"},
        {"Luan Santana", 70, "Épico"},
        {"Maiara e Maraisa", 65, "Raro"},
        {"Daniel", 60, "Raro"},
        {"Sérgio Reis", 55, "Raro"},
        {"Michel Teló", 50, "Raro"},
        {"Roberta Miranda", 45, "Raro"},
        {"Almir Sater", 40, "Comum"},
        {"Gino & Geno", 35, "Comum"},
        {"João Carreiro & Capataz", 30, "Comum"},
        {"Edson & Hudson", 25, "Comum"},
        {"Cezar & Paulinho", 20, "Comum"},
        {"Rick & Renner", 15, "Comum"},
        {"Matogrosso & Mathias", 10, "Comum"},
        {"Milionário & José Rico", 5, "Comum"},

        // Pagode
        {"Alexandre Pires", 100, "Lendário"},
        {"Belo", 95, "Lendário"},
        {"Zeca Pagodinho", 90, "Épico"},
        {"Ferrugem", 85, "Épico"},
        {"Sorriso Maroto", 80, "Épico"},
        {"Péricles", 75, "Raro"},
        {"Dilsinho", 70, "Raro"},
        {"Thiaguinho", 65, "Raro"},
        {"Molejo", 60, "Comum"},
        {"Pixote", 55, "Comum"},
        
        // MPB
        {"Caetano Veloso", 100, "Lendário"},
        {"Gilberto Gil", 95, "Lendário"},
        {"Elis Regina", 90, "Lendário"},
        {"Gal Costa", 85, "Épico"},
        {"Djavan", 80, "Épico"},
        {"Cássia Eller", 75, "Épico"},
        {"Lenine", 70, "Raro"},
        {"Seu Jorge", 65, "Raro"},
        {"Maria Rita", 60, "Raro"},
        {"Tom Jobim", 110, "Lendário"},
    }
}