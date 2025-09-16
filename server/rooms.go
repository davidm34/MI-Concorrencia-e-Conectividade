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

		// Bossa Nova
		{"João Gilberto", 100, "Lendário"},
		{"Tom Jobim", 95, "Lendário"},
		{"Vinicius de Moraes", 90, "Lendário"},
		{"Astrud Gilberto", 85, "Épico"},
		{"Roberto Menescal", 80, "Épico"},
		{"Carlos Lyra", 75, "Raro"},
		{"Toquinho", 70, "Raro"},
		{"Wanda Sá", 65, "Raro"},
		{"Sérgio Mendes", 60, "Comum"},
		{"Stan Getz", 55, "Comum"},
		{"Miúcha", 50, "Comum"},
		{"Nara Leão", 45, "Comum"},
		{"Lisa Ono", 40, "Comum"},
		{"Bebel Gilberto", 35, "Comum"},
		{"Joyce Moreno", 30, "Comum"},
		{"Pery Ribeiro", 25, "Comum"},
		{"Maria Creuza", 20, "Comum"},
		{"Claudette Soares", 15, "Comum"},
		{"Alaíde Costa", 10, "Comum"},
		{"Os Cariocas", 5, "Comum"},

		// Funk Carioca
		{"Anitta", 100, "Lendário"},
		{"Ludmilla", 95, "Lendário"},
		{"MC Kevinho", 90, "Épico"},
		{"MC Ryan SP", 85, "Épico"},
		{"Livinho", 80, "Épico"},
		{"Kevin o Chris", 75, "Raro"},
		{"Nego do Borel", 70, "Raro"},
		{"Dennis DJ", 65, "Raro"},
		{"MC Poze do Rodo", 60, "Raro"},
		{"Tati Quebra Barraco", 55, "Raro"},
		{"MC G15", 50, "Comum"},
		{"MC Don Juan", 45, "Comum"},
		{"MC Cabelinho", 40, "Comum"},
		{"MC Guimê", 35, "Comum"},
		{"MC Valesca Popozuda", 30, "Comum"},
		{"Bonde do Tigrão", 25, "Comum"},
		{"Furacão 2000", 20, "Comum"},
		{"Os Hawaianos", 15, "Comum"},
		{"Mr. Catra", 10, "Comum"},
		{"MC Fioti", 5, "Comum"},

		// Pop e Pop Rock
		{"Lulu Santos", 100, "Lendário"},
		{"Marina Lima", 95, "Lendário"},
		{"Kid Abelha", 90, "Épico"},
		{"Jota Quest", 85, "Épico"},
		{"Skank", 80, "Épico"},
		{"Capital Inicial", 75, "Raro"},
		{"Paralamas do Sucesso", 70, "Raro"},
		{"Nando Reis", 65, "Raro"},
		{"Marisa Monte", 60, "Raro"},
		{"Titãs", 55, "Raro"},
		{"O Rappa", 50, "Comum"},
		{"Pato Fu", 45, "Comum"},
		{"Los Hermanos", 40, "Comum"},
		{"Ana Carolina", 35, "Comum"},
		{"Vitor Kley", 30, "Comum"},
		{"Manu Gavassi", 25, "Comum"},
		{"Luan Santana", 20, "Comum"},
		{"NX Zero", 15, "Comum"},
		{"Roupa Nova", 10, "Comum"},
		{"RPM", 5, "Comum"},

		// Hip Hop e Rap
		{"Mano Brown", 100, "Lendário"},
		{"Emicida", 95, "Lendário"},
		{"Sabotage", 90, "Lendário"},
		{"Racionais MC's", 85, "Épico"},
		{"Criolo", 80, "Épico"},
		{"Djonga", 75, "Épico"},
		{"Marcelo D2", 70, "Raro"},
		{"Projota", 65, "Raro"},
		{"Filipe Ret", 60, "Raro"},
		{"Costa Gold", 55, "Raro"},
		{"Haikaiss", 50, "Comum"},
		{"Matuê", 45, "Comum"},
		{"Recayd Mob", 40, "Comum"},
		{"Coruja BC1", 35, "Comum"},
		{"Rashid", 30, "Comum"},
		{"Rincon Sapiência", 25, "Comum"},
		{"Kamau", 20, "Comum"},
		{"Flora Matos", 15, "Comum"},
		{"Drik Barbosa", 10, "Comum"},
		{"Baco Exu do Blues", 5, "Comum"},

		// MPB (Música Popular Brasileira)
		{"Milton Nascimento", 100, "Lendário"},
		{"Chico Buarque", 95, "Lendário"},
		{"Maria Bethânia", 90, "Lendário"},
		{"Djavan", 85, "Épico"},
		{"Fagner", 80, "Épico"},
		{"Geraldo Vandré", 75, "Raro"},
		{"Guilherme Arantes", 70, "Raro"},
		{"Tim Maia", 65, "Raro"},
		{"Luiz Melodia", 60, "Raro"},
		{"Ivan Lins", 55, "Raro"},
		{"Eduardo Gudin", 50, "Comum"},
		{"Joyce Moreno", 45, "Comum"},
		{"João Donato", 40, "Comum"},
		{"Alaíde Costa", 35, "Comum"},
		{"Os Mutantes", 30, "Comum"},
		{"Zé Ramalho", 25, "Comum"},
		{"Belchior", 20, "Comum"},
		{"Emílio Santiago", 15, "Comum"},
		{"Jorge Ben Jor", 10, "Comum"},
		{"Clube da Esquina", 5, "Comum"},

		// Sertanejo (continuação)
		{"Marília Mendonça", 100, "Lendário"},
		{"Jorge e Mateus", 95, "Lendário"},
		{"Bruno e Marrone", 90, "Épico"},
		{"João Bosco e Vinicius", 85, "Épico"},
		{"Zé Neto e Cristiano", 80, "Épico"},
		{"Leonardo", 75, "Raro"},
		{"Eduardo Costa", 70, "Raro"},
		{"Gusttavo Lima", 65, "Raro"},
		{"Cristiano Araújo", 60, "Raro"},
		{"Lauana Prado", 55, "Raro"},
		{"Fernando e Sorocaba", 50, "Comum"},
		{"Israel e Rodolffo", 45, "Comum"},
		{"César Menotti e Fabiano", 40, "Comum"},
		{"Hugo e Guilherme", 35, "Comum"},
		{"João Gomes", 30, "Comum"},
		{"Barões da Pisadinha", 25, "Comum"},
		{"Zé Vaqueiro", 20, "Comum"},
		{"Mano Walter", 15, "Comum"},
		{"Tierry", 10, "Comum"},
		{"Washington Brasileiro", 5, "Comum"},

		// Samba e Pagode (continuação)
		{"Dona Ivone Lara", 100, "Lendário"},
		{"Beth Carvalho", 95, "Lendário"},
		{"Cartola", 90, "Lendário"},
		{"Paulinho da Viola", 85, "Épico"},
		{"Martinho da Vila", 80, "Épico"},
		{"Alcione", 75, "Épico"},
		{"Elza Soares", 70, "Raro"},
		{"Jorge Aragão", 65, "Raro"},
		{"Arlindo Cruz", 60, "Raro"},
		{"Fundo de Quintal", 55, "Raro"},
		{"Diogo Nogueira", 50, "Comum"},
		{"Clara Nunes", 45, "Comum"},
		{"Adoniran Barbosa", 40, "Comum"},
		{"Chico Buarque", 35, "Comum"},
		{"Elis Regina", 30, "Comum"},
		{"Wilson das Neves", 25, "Comum"},
		{"Nelson Cavaquinho", 20, "Comum"},
		{"Clementina de Jesus", 15, "Comum"},
		{"João Nogueira", 10, "Comum"},
		{"Zeca Pagodinho", 5, "Comum"},

		// Rock (continuação)
		{"Legião Urbana", 100, "Lendário"},
		{"Titãs", 95, "Lendário"},
		{"Sepultura", 90, "Lendário"},
		{"Engenheiros do Hawaii", 85, "Épico"},
		{"Barão Vermelho", 80, "Épico"},
		{"Plebe Rude", 75, "Épico"},
		{"Cássia Eller", 70, "Raro"},
		{"Pitty", 65, "Raro"},
		{"Fresno", 60, "Raro"},
		{"CPM 22", 55, "Raro"},
		{"Charlie Brown Jr.", 50, "Comum"},
		{"Detonautas", 45, "Comum"},
		{"Raimundos", 40, "Comum"},
		{"O Terno", 35, "Comum"},
		{"Vanguart", 30, "Comum"},
		{"Pitty", 25, "Comum"},
		{"Skank", 20, "Comum"},
		{"Capital Inicial", 15, "Comum"},
		{"O Rappa", 10, "Comum"},
		{"Nação Zumbi", 5, "Comum"},

		// Forró (continuação)
		{"Dominguinhos", 100, "Lendário"},
		{"Luiz Gonzaga", 95, "Lendário"},
		{"Alceu Valença", 90, "Épico"},
		{"Geraldo Azevedo", 85, "Épico"},
		{"Elba Ramalho", 80, "Épico"},
		{"Flávio José", 75, "Raro"},
		{"Calcinha Preta", 70, "Raro"},
		{"Mastruz com Leite", 65, "Raro"},
		{"Falamansa", 60, "Raro"},
		{"Aviões do Forró", 55, "Raro"},
		{"Wesley Safadão", 50, "Comum"},
		{"Solange Almeida", 45, "Comum"},
		{"Limão com Mel", 40, "Comum"},
		{"Dorgival Dantas", 35, "Comum"},
		{"Saia Rodada", 30, "Comum"},
		{"Magníficos", 25, "Comum"},
		{"Gilberto Gil", 20, "Comum"},
		{"Chico César", 15, "Comum"},
		{"Jackson do Pandeiro", 10, "Comum"},
		{"Trio Virgulino", 5, "Comum"},

		// Axé (continuação)
		{"Ivete Sangalo", 100, "Lendário"},
		{"Bell Marques", 95, "Lendário"},
		{"Daniela Mercury", 90, "Lendário"},
		{"Claudia Leitte", 85, "Épico"},
		{"Saulo Fernandes", 80, "Épico"},
		{"É o Tchan!", 75, "Épico"},
		{"Banda Eva", 70, "Raro"},
		{"Durval Lelys", 65, "Raro"},
		{"Netinho", 60, "Raro"},
		{"Harmonia do Samba", 55, "Raro"},
		{"Psirico", 50, "Comum"},
		{"Trio da Huanna", 45, "Comum"},
		{"Léo Santana", 40, "Comum"},
		{"Parangolé", 35, "Comum"},
		{"Araketu", 30, "Comum"},
		{"Timbalada", 25, "Comum"},
		{"Chiclete com Banana", 20, "Comum"},
		{"Babado Novo", 15, "Comum"},
		{"Banda Vingadora", 10, "Comum"},
		{"Olodum", 5, "Comum"},

		// Funk 150 BPM / Baile Funk
		{"MC Kevin", 100, "Lendário"},
		{"MC Marcinho", 95, "Lendário"},
		{"MC Daleste", 90, "Lendário"},
		{"MC Livinho", 85, "Épico"},
		{"MC Hariel", 80, "Épico"},
		{"MC PH", 75, "Raro"},
		{"MC Kevin o Chris", 70, "Raro"},
		{"MC Jottapê", 65, "Raro"},
		{"MC G15", 60, "Raro"},
		{"MC Don Juan", 55, "Raro"},
		{"MC Valesca Popozuda", 50, "Comum"},
		{"MC Bin Laden", 45, "Comum"},
		{"Bonde do Tigrão", 40, "Comum"},
		{"Bonde da Stronda", 35, "Comum"},
		{"MC Guimê", 30, "Comum"},
		{"MC Gago", 25, "Comum"},
		{"Tati Quebra Barraco", 20, "Comum"},
		{"MC Serginho", 15, "Comum"},
		{"MC Carol", 10, "Comum"},
		{"Mr. Catra", 5, "Comum"},

		// Música Regional (Choro, Brega, etc.)
		{"Pixinguinha", 100, "Lendário"},
		{"Jacob do Bandolim", 95, "Lendário"},
		{"Lupicínio Rodrigues", 90, "Lendário"},
		{"Reginaldo Rossi", 85, "Épico"},
		{"Nelson Gonçalves", 80, "Épico"},
		{"Waldick Soriano", 75, "Raro"},
		{"Altemar Dutra", 70, "Raro"},
		{"Ovelha", 65, "Raro"},
		{"Falcão", 60, "Raro"},
		{"Sidney Magal", 55, "Raro"},
		{"Odair José", 50, "Comum"},
		{"Amado Batista", 45, "Comum"},
		{"Fagner", 40, "Comum"},
		{"Wando", 35, "Comum"},
		{"Carlos Alberto", 30, "Comum"},
		{"Evaldo Braga", 25, "Comum"},
		{"Gino e Geno", 20, "Comum"},
		{"Jonas Esticado", 15, "Comum"},
		{"Zé Cantor", 10, "Comum"},
		{"João Gomes", 5, "Comum"},

		// Música Eletrônica e DJ's
		{"Alok", 100, "Lendário"},
		{"Vintage Culture", 95, "Lendário"},
		{"KVSH", 90, "Épico"},
		{"Bruno Be", 85, "Épico"},
		{"Dubdogz", 80, "Épico"},
		{"Illusionize", 75, "Raro"},
		{"Cat Dealers", 70, "Raro"},
		{"Bhaskar", 65, "Raro"},
		{"Chemical Surf", 60, "Raro"},
		{"Liu", 55, "Raro"},
		{"Gabe", 50, "Comum"},
		{"Gustavo Mota", 45, "Comum"},
		{"Pontifexx", 40, "Comum"},
		{"Mojjo", 35, "Comum"},
		{"Groove Delight", 30, "Comum"},
		{"Fancy Inc", 25, "Comum"},
		{"Victor Lou", 20, "Comum"},
		{"Öwnboss", 15, "Comum"},
		{"Doozie", 10, "Comum"},
		{"Doozie", 5, "Comum"},

		// Gospel
		{"Luan Santana", 100, "Lendário"},
		{"Fernanda Brum", 95, "Lendário"},
		{"Aline Barros", 90, "Lendário"},
		{"Gabriela Rocha", 85, "Épico"},
		{"Isadora Pompeo", 80, "Épico"},
		{"Priscilla Alcantara", 75, "Raro"},
		{"Thalles Roberto", 70, "Raro"},
		{"Kleber Lucas", 65, "Raro"},
		{"Diante do Trono", 60, "Raro"},
		{"Bruna Karla", 55, "Raro"},
		{"Eyshila", 50, "Comum"},
		{"Regis Danese", 45, "Comum"},
		{"Damares", 40, "Comum"},
		{"Anderson Freire", 35, "Comum"},
		{"Ton Carfi", 30, "Comum"},
		{"Leonardo Gonçalves", 25, "Comum"},
		{"Oficina G3", 20, "Comum"},
		{"Renascer Praise", 15, "Comum"},
		{"André Valadão", 10, "Comum"},
		{"Jotta A", 5, "Comum"},

		// Clássicos da Música Popular
		{"Roberto Carlos", 100, "Lendário"},
		{"Erasmo Carlos", 95, "Lendário"},
		{"Ney Matogrosso", 90, "Lendário"},
		{"João Gilberto", 85, "Épico"},
		{"Toquinho", 80, "Épico"},
		{"Maria Bethânia", 75, "Raro"},
		{"Gal Costa", 70, "Raro"},
		{"Caetano Veloso", 65, "Raro"},
		{"Gilberto Gil", 60, "Raro"},
		{"Elis Regina", 55, "Raro"},
		{"Tom Jobim", 50, "Comum"},
		{"Vinicius de Moraes", 45, "Comum"},
		{"Chico Buarque", 40, "Comum"},
		{"Alcione", 35, "Comum"},
		{"Martinho da Vila", 30, "Comum"},
		{"Paulinho da Viola", 25, "Comum"},
		{"Pixinguinha", 20, "Comum"},
		{"Cartola", 15, "Comum"},
		{"Zeca Pagodinho", 10, "Comum"},
		{"Tim Maia", 5, "Comum"},

		// Outros gêneros (Trap, Lo-fi, Emo)
		{"Matuê", 100, "Lendário"},
		{"Froid", 95, "Lendário"},
		{"Cynthia Luz", 90, "Épico"},
		{"Kayblack", 85, "Épico"},
		{"Teto", 80, "Épico"},
		{"Derek", 75, "Raro"},
		{"Major RD", 70, "Raro"},
		{"Sidoka", 65, "Raro"},
		{"Veigh", 60, "Raro"},
		{"MC Cabelinho", 55, "Raro"},
		{"Don L", 50, "Comum"},
		{"Filipe Ret", 45, "Comum"},
		{"Mão de Oito", 40, "Comum"},
		{"Ouriço", 35, "Comum"},
		{"L7NNON", 30, "Comum"},
		{"Yunk Vino", 25, "Comum"},
		{"Tasha & Tracie", 20, "Comum"},
		{"Recayd Mob", 15, "Comum"},
		{"Haikaiss", 10, "Comum"},
		{"Baco Exu do Blues", 5, "Comum"},

		// Pop Brasileiro Atual
		{"IZA", 100, "Lendário"},
		{"Vitão", 95, "Lendário"},
		{"Pabllo Vittar", 90, "Lendário"},
		{"Luísa Sonza", 85, "Épico"},
		{"Gloria Groove", 80, "Épico"},
		{"Lagum", 75, "Raro"},
		{"Melim", 70, "Raro"},
		{"Jão", 65, "Raro"},
		{"Silva", 60, "Raro"},
		{"Duda Beat", 55, "Raro"},
		{"Bala Desejo", 50, "Comum"},
		{"Marina Sena", 45, "Comum"},
		{"Ana Vilela", 40, "Comum"},
		{"Vitor Kley", 35, "Comum"},
		{"Gilsons", 30, "Comum"},
		{"Vitor Fernandes", 25, "Comum"},
		{"João Bosco e Vinícius", 20, "Comum"},
		{"Ferrugem", 15, "Comum"},
		{"Dilsinho", 10, "Comum"},
		{"Sorriso Maroto", 5, "Comum"},

		// Mais Sertanejo
		{"Zé Ramalho", 100, "Lendário"},
		{"Daniel", 95, "Lendário"},
		{"Leonardo", 90, "Lendário"},
		{"Bruno e Marrone", 85, "Épico"},
		{"Chitãozinho e Xororó", 80, "Épico"},
		{"Zezé Di Camargo & Luciano", 75, "Épico"},
		{"João Paulo & Daniel", 70, "Raro"},
		{"Rick & Renner", 65, "Raro"},
		{"Gino e Geno", 60, "Raro"},
		{"Trio Parada Dura", 55, "Raro"},
		{"Milionário e José Rico", 50, "Comum"},
		{"Cezar e Paulinho", 45, "Comum"},
		{"João Carreiro e Capataz", 40, "Comum"},
		{"Teodoro e Sampaio", 35, "Comum"},
		{"Léo Canhoto e Robertinho", 30, "Comum"},
		{"Irmãs Galvão", 25, "Comum"},
		{"Liu e Léo", 20, "Comum"},
		{"Tonico e Tinoco", 15, "Comum"},
		{"Sérgio Reis", 10, "Comum"},
		{"Roberta Miranda", 5, "Comum"},

		// Clássicos do Rock Nacional
		{"Raul Seixas", 100, "Lendário"},
		{"Renato Russo", 95, "Lendário"},
		{"Rita Lee", 90, "Lendário"},
		{"Cazuza", 85, "Épico"},
		{"Chorão", 80, "Épico"},
		{"Pitty", 75, "Raro"},
		{"Marcelo Nova", 70, "Raro"},
		{"Dinho Ouro Preto", 65, "Raro"},
		{"Frejat", 60, "Raro"},
		{"Herbert Vianna", 55, "Raro"},
		{"Supla", 50, "Comum"},
		{"Tico Santa Cruz", 45, "Comum"},
		{"Digão", 40, "Comum"},
		{"Di Ferrero", 35, "Comum"},
		{"Badauí", 30, "Comum"},
		{"Rogério Flausino", 25, "Comum"},
		{"Samuel Rosa", 20, "Comum"},
		{"Lulu Santos", 15, "Comum"},
		{"Tony Bellotto", 10, "Comum"},
		{"Paulo Ricardo", 5, "Comum"},

		// Pagode e Samba dos Anos 90
		{"Alexandre Pires", 100, "Lendário"},
		{"Belo", 95, "Lendário"},
		{"Art Popular", 90, "Épico"},
		{"Raça Negra", 85, "Épico"},
		{"Exaltasamba", 80, "Épico"},
		{"Soweto", 75, "Raro"},
		{"SPC", 70, "Raro"},
		{"Molejo", 65, "Raro"},
		{"Katinguelê", 60, "Raro"},
		{"Grupo Revelação", 55, "Raro"},
		{"Travessos", 50, "Comum"},
		{"Negritude Junior", 45, "Comum"},
		{"Os Travessos", 40, "Comum"},
		{"Pixote", 35, "Comum"},
		{"Samba Pura", 30, "Comum"},
		{"Grupo Pirraça", 25, "Comum"},
		{"Netinho de Paula", 20, "Comum"},
		{"Zeca Pagodinho", 15, "Comum"},
		{"Jorge Aragão", 10, "Comum"},
		{"Péricles", 5, "Comum"},

		// Funk Ostentação
		{"MC Guimê", 100, "Lendário"},
		{"MC Livinho", 95, "Lendário"},
		{"MC Rodolfinho", 90, "Épico"},
		{"MC Boy do Charmes", 85, "Épico"},
		{"MC Lon", 80, "Épico"},
		{"MC João", 75, "Raro"},
		{"MC Daleste", 70, "Raro"},
		{"MC Bin Laden", 65, "Raro"},
		{"MC Dede", 60, "Raro"},
		{"MC Pedrinho", 55, "Raro"},
		{"MC Léo da Baixada", 50, "Comum"},
		{"MC Guimê", 45, "Comum"},
		{"MC Pikeno e Menor", 40, "Comum"},
		{"MC Biel", 35, "Comum"},
		{"MC Gui", 30, "Comum"},
		{"MC Brinquedo", 25, "Comum"},
		{"MC Lan", 20, "Comum"},
		{"MC Kevin", 15, "Comum"},
		{"Kevinho", 10, "Comum"},
		{"MC Ryan SP", 5, "Comum"},
	}
}