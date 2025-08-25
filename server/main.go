package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
)

var manager = NewPlayerManager("players.json")

func Login(conn net.Conn, name string){
	fmt.Print("Login Realizado com Sucesso! \n")
	reader := bufio.NewReader(conn)
	for {
		message, err := reader.ReadString('\n')
		if err != nil {
			fmt.Println("Jogador saiu:", name)
			return
		}
	fmt.Printf("[%s]: %s", name, message)
	conn.Write([]byte("Servidor recebeu: " + message))
	}	
}

func handleConnection(conn net.Conn) {
    var login int
	var name, password string
    defer conn.Close()

	
    fmt.Print("[0] Digite se deseja fazer Login\n[1] Digite se deseja fazer cadastro: \n")
    fmt.Scan(&login)
	fmt.Print("Digite seu nome: \n")
	fmt.Scan(&name) 
	fmt.Print("Digite sua senha: \n")
	fmt.Scan(&password) 
	if (login == 0){		
		var add_player bool = manager.Verify_Login(name, password)
		if (add_player) {
			Login(conn, name)
		} else {
			fmt.Print("Usuário não encontrado! \n")
			
		}
	} else {
		player, err := manager.AddPlayer(conn, name, password)
		if err != nil {
			fmt.Print("Erro ao fazer cadastro")
		}
		Login(conn, player.Name)
	}

   
}


func main() {
	// Cria servidor na porta 8080
	ln, err := net.Listen("tcp", ":8080")
	if err != nil {
		fmt.Println("Erro ao iniciar servidor:", err)
		os.Exit(1)
	}
	defer ln.Close()

	fmt.Println("Servidor TCP rodando na porta 8080...")

	for {
		// Aceita nova conexão
		conn, err := ln.Accept()
		if err != nil {
			fmt.Println("Erro ao aceitar conexão:", err)
			continue
		}

		// Trata cliente em goroutine
		go handleConnection(conn)
	}
}
