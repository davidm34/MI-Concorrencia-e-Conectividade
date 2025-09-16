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
	PlayerWins [2]int
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
		PlayerWins: [2]int{},
	}
	rm.nextID++
	rm.rooms = append(rm.rooms, room)

	fmt.Printf("Nova sala %d criada para jogador %s. Esperando adversário...\n", room.ID, p.Name)
	return room
}

func (r *Room) Broadcast(sender *Player, msg string, includeSender bool, prefix bool) {
    for _, p := range r.Players {
        if !includeSender && p == sender {
            continue
        }
        if prefix && sender != nil {
            p.Conn.Write([]byte(sender.Name + ": " + msg))
        } else {
            p.Conn.Write([]byte(msg))
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
				room.Broadcast(p, msg, false, true)
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
			r.Broadcast(p, fmt.Sprintf("%s escolheu uma carta!\n", p.Name), false, true)

			// Colocando a carta selecionada pelo usuario como as cartas da rodada
			if r.Players[0].ID == p.ID {
				r.Cards[0] = p.Cards[choice]
			} else {
				r.Cards[1] = p.Cards[choice]
			}

			// Mostrar a carta selecionada
			for i := 0; i < 2; i++ {
				if r.Players[i].ID == p.ID {
					p.SelectionRound = true
					p.Conn.Write([]byte(fmt.Sprintf("Você escolheu: %s (Dano: %d)\n", chosenCard.Name, chosenCard.Damage))) 
				} 
			} 

			
			// Jogador esperando a jogada do outro jogador
			r.mu.Lock()
			for (!r.Players[0].SelectionRound && r.Players[1].SelectionRound) || (r.Players[0].SelectionRound && !r.Players[1].SelectionRound) {
				p.Conn.Write([]byte("Aguardando a jogada do adversário... \n"))
				time.Sleep(4 * time.Second)
			}
			r.mu.Unlock()

			// remove as cartas selecionadas
			p.Cards = append(p.Cards[:choice], p.Cards[choice+1:]...)


			// Lógica para o Vencedor da Rodada
			if r.Cards[0].Damage > r.Cards[1].Damage {
				r.PlayerWins[0]++
				p.Conn.Write([]byte("\nSala " + strconv.Itoa(r.ID) + ": " + r.Players[0].Name+" Vencedor da Rodada\n\n"))				
				fmt.Println("\nSala " + strconv.Itoa(r.ID) + ": " + r.Players[0].Name+" Vencedor da Rodada\n\n")
			} else if r.Cards[1].Damage > r.Cards[0].Damage {
				r.PlayerWins[1]++				
				p.Conn.Write([]byte("\nSala " + strconv.Itoa(r.ID) + ": " + r.Players[1].Name+" Vencedor da Rodada\n\n"))				
				fmt.Println("\nSala " + strconv.Itoa(r.ID) + ": " + r.Players[0].Name+" Vencedor da Rodada\n\n")
			} else {				
				p.Conn.Write([]byte("\nSala " + strconv.Itoa(r.ID) + ": " +" Rodada Empatada\n\n"))			
				fmt.Println("\nSala " + strconv.Itoa(r.ID) + ": " +" Rodada Empatada\n\n")
			}
			

			
			r.Players[0].SelectionRound = false
			r.Players[1].SelectionRound = false

			// Lógica para fim de jogo
			if len(r.Players[0].Cards) == 0 || len(r.Players[1].Cards) == 0 {
				break
			}
							
		} 
	}

	// Lógica para Fim de Jogo
	p.Conn.Write([]byte("\nJogo Finalizado!\n\n"))
	if r.PlayerWins[0] > r.PlayerWins[1] {
		p.Conn.Write([]byte("\nSala " + strconv.Itoa(r.ID) + ": " + r.Players[0].Name+" Vencedor da Partida\n\n"))
		fmt.Println("\nSala " + strconv.Itoa(r.ID) + ": " + r.Players[0].Name+" Vencedor da Partida\n\n")
	} else if r.PlayerWins[1] > r.PlayerWins[0] {
		p.Conn.Write([]byte("\nSala " + strconv.Itoa(r.ID) + ": " + r.Players[1].Name+" Vencedor da Partida\n\n"))
		fmt.Println("\nSala " + strconv.Itoa(r.ID) + ": " + r.Players[1].Name+" Vencedor da Partida\n\n")
	} else {
		p.Conn.Write([]byte("\nSala " + strconv.Itoa(r.ID) + ": " +" Rodada Empatada\n\n"))			
		fmt.Println("\nSala " + strconv.Itoa(r.ID) + ": " +" Rodada Empatada\n\n")
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
        {"Wesley Safadão", 100, "Lendário"},
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
		// Samba & Pagode
		{"Cartola", 100, "Lendário"},
		{"Chico Buarque", 95, "Lendário"},
		{"Tim Maia", 90, "Lendário"},
		{"Clara Nunes", 85, "Épico"},
		{"Martinho da Vila", 80, "Épico"},
		{"Alcione", 75, "Raro"},
		{"Jorge Aragão", 70, "Raro"},
		{"Arlindo Cruz", 65, "Raro"},
		{"Paulinho da Viola", 60, "Raro"},
		{"Zeca Pagodinho", 55, "Raro"},
        
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
        {"Tom Jobim", 100, "Lendário"},

		// Rock
		{"Cazuza", 100, "Lendário"},
		{"Renato Russo", 95, "Lendário"},
		{"Rita Lee", 90, "Lendário"},
		{"Raul Seixas", 85, "Épico"},
		{"Chorão", 80, "Épico"},
		{"Pitty", 75, "Épico"},
		{"Marcelo D2", 70, "Raro"},
		{"Frejat", 65, "Raro"},
		{"Dinho Ouro Preto", 60, "Raro"},
		{"Herbert Vianna", 55, "Raro"},
		{"Paulo Ricardo", 50, "Comum"},
		{"Supla", 45, "Comum"},
		{"Tico Santa Cruz", 40, "Comum"},
		{"Digão", 35, "Comum"},
		{"Di Ferrero", 30, "Comum"},
		{"Badauí", 25, "Comum"},
		{"Rogério Flausino", 20, "Comum"},
		{"Samuel Rosa", 15, "Comum"},
		{"Lulu Santos", 10, "Comum"},
		{"Tony Bellotto", 5, "Comum"},

		// Forró
		{"Luiz Gonzaga", 100, "Lendário"},
		{"Dominguinhos", 95, "Lendário"},
		{"Alceu Valença", 90, "Épico"},
		{"Geraldo Azevedo", 85, "Épico"},
		{"Flávio José", 80, "Raro"},
		{"Elba Ramalho", 75, "Raro"},
		{"Wesley Safadão", 70, "Comum"},
		{"Solange Almeida", 65, "Comum"},
		{"Mastruz com Leite", 60, "Comum"},
		{"Calcinha Preta", 55, "Comum"},

		// Axé
		{"Ivete Sangalo", 100, "Lendário"},
		{"Daniela Mercury", 95, "Lendário"},
		{"Netinho", 90, "Épico"},
		{"Bell Marques", 85, "Épico"},
		{"Saulo Fernandes", 80, "Raro"},
		{"Claudia Leitte", 75, "Raro"},
		{"Durval Lelys", 70, "Raro"},
		{"Timbalada", 65, "Comum"},
		{"É O Tchan!", 60, "Comum"},
		{"Banda Eva", 55, "Comum"},
    }
}