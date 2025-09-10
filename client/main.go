package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"
)

func main() {
	conn, err := net.Dial("tcp", "server:8080")
	if err != nil {
		fmt.Println("Erro ao conectar:", err)
		return
	}
	
	defer conn.Close()

	fmt.Println("Conectado ao servidor!")

	// Conecta ao servidor
	connUdp, err := net.Dial("udp", "server:8081")
	if err != nil {
		fmt.Println("Erro ao conectar:", err)
		return
	}
	defer conn.Close()

	Udp(connUdp)	

	reader := bufio.NewReader(os.Stdin)

	// Pega nome do jogador
	fmt.Print("Digite seu nome: ")
	name, _ := reader.ReadString('\n')
	conn.Write([]byte(name))

	// Goroutine para ouvir mensagens do servidor
	go func() {
		serverReader := bufio.NewReader(conn)
		for {
			msg, err := serverReader.ReadString('\n')
			if err != nil {
				fmt.Println("Conexão com o servidor encerrada.")
				os.Exit(0)
			}
			fmt.Print(msg) // mostra mensagens recebidas
		}
	}()

	// Loop de envio de mensagens (input do jogador)
	for {
		text, _ := reader.ReadString('\n')
		text = strings.TrimSpace(text)
		if text == "" {
			continue
		}
		conn.Write([]byte(text + "\n"))
	}
}


func Udp(connUdp net.Conn){

	buffer := make([]byte, 1024)

	message := "Ping: "

	// Marca o tempo antes de enviar
	start := time.Now()
	_ , errUdp := connUdp.Write([]byte(message))
	if errUdp != nil {
		fmt.Println("Erro ao enviar:", errUdp)
	}

	// Aguarda resposta
	n, err := connUdp.Read(buffer)
	if err != nil {
		fmt.Println("Erro ao ler resposta:", err)
	}
	elapsed := time.Since(start)

	fmt.Printf("Servidor respondeu: %s | RTT: %v\n", string(buffer[:n]), elapsed)


}