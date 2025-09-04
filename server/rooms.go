package main

import (
	"bufio"
	"fmt"
	"sync"
    "time"
    "strings"
)

type Room struct {
    mu sync.Mutex
    ID      int
    Players []*Player
    Actions map[int]bool
}

type RoomManager struct {
    mu          sync.Mutex
    rooms       []*Room
    nextID      int
    waitingPool []*Player  
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

    for i := range rm.rooms {

       if len(rm.rooms[i].Players) < 2 {
            rm.rooms[i].Players = append(rm.rooms[i].Players, p)
            fmt.Printf("Jogador %s adicionado à sala %d\n", p.Name, rm.rooms[i].ID)
            
            if len(rm.rooms[i].Players) == 2 {
                fmt.Printf("Sala %d completa! Iniciando jogo...\n", rm.rooms[i].ID)
            }
            return rm.rooms[i]
        }
    }

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

func HandlePlayer(p *Player, room *Room) {
    reader := bufio.NewReader(p.Conn)
    p.Duel = true
    
    room.mu.Lock()
    room.Actions[p.ID] = true
    currentActions := len(room.Actions)
    totalPlayers := len(room.Players)
    room.mu.Unlock()

    if currentActions < totalPlayers || totalPlayers < 2 {
        p.Conn.Write([]byte("Aguardando o segundo jogador...\n"))
    }
    
    for {
        room.mu.Lock()
        currentActions = len(room.Actions)
        totalPlayers = len(room.Players)
        bothReady := currentActions >= 2 && totalPlayers == 2

        room.mu.Unlock()
        
        if bothReady {
            break
        }
        time.Sleep(100 * time.Millisecond) 
    }
    
    // AMBOS ESTÃO PRONTOS - Mostra o menu
    p.Conn.Write([]byte("\nDigite [0] se você quer enviar uma mensagem: \nDigite [1] se deseja jogar: \n"))
    
    var choice string
    var err error
    // Lê a escolha do jogador
    choice, err = reader.ReadString('\n')
    for err != nil {
        choice, err = reader.ReadString('\n')
    }
    choice = strings.TrimSpace(choice)

    switch choice {
    case "0":
        // Modo chat
        p.Conn.Write([]byte("Modo chat ativado. Digite suas mensagens:\n"))
        for {
            msg, err := reader.ReadString('\n')
            if err != nil {
                fmt.Println("Jogador saiu:", p.Name)
                return
            }
            room.Broadcast(p, msg)
        }
        
    case "1":
        // Inicia o jogo
        p.Conn.Write([]byte("Iniciando jogo...\n"))
        go Game(p, room)
        
    default:
        p.Conn.Write([]byte("Opção inválida. Use 0 ou 1.\n"))
    }
}

func Game(p *Player, room *Room){

   for _, v := range p.Cards {
        fmt.Print("Suas Cartas: \n", v.Name)
   }


}