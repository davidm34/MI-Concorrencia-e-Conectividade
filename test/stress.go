package main

import (
	"bufio"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"
)

const (
	serverAddrTCP = "server:8080"
	serverAddrUDP = "server:8081"
	numClients    = 6 // número de clientes simulados
)

var readyWG sync.WaitGroup
var doneWG sync.WaitGroup
var startCh = make(chan struct{})

func tcpClient(id int) {
	defer doneWG.Done()

	conn, err := net.Dial("tcp", serverAddrTCP)
	if err != nil {
		fmt.Printf("[Client %d] Erro ao conectar TCP: %v\n", id, err)
		return
	}
	defer conn.Close()

	reader := bufio.NewReader(conn)

	// avisa que este cliente está pronto
	readyWG.Done()

	// aguarda sinal global para começar
	<-startCh

	// 1) Envia o nome
	name := fmt.Sprintf("Jogador%d\n", id)
	_, _ = conn.Write([]byte(name))

	// Lógica do jogo: lê e responde
	for {
		// Lê a próxima mensagem do servidor
		msg, err := reader.ReadString('\n')
		if err != nil {
			fmt.Printf("[Client %d] Conexão encerrada pelo servidor: %v\n", id, err)
			return
		}

		// Verifica se o servidor está pedindo uma jogada
		if strings.Contains(msg, "Digite o número da carta que deseja jogar:") {
			conn.Write([]byte("0\n"))
		} else if strings.Contains(msg, "Jogo Finalizado!") {
			// Se o jogo terminou, o cliente pode se desconectar
			return
		}
	}
}

func udpClient(id int) {
	defer doneWG.Done()

	conn, err := net.Dial("udp", serverAddrUDP)
	if err != nil {
		fmt.Printf("[Client %d] Erro ao conectar UDP: %v\n", id, err)
		return
	}
	defer conn.Close()

	buffer := make([]byte, 1024)

	// avisa que este cliente está pronto
	readyWG.Done()

	// aguarda sinal para começar
	<-startCh

	message := fmt.Sprintf("Ping-from Client-%d", id)

	start := time.Now()
	_, err0 := conn.Write([]byte(message))
	if err0 != nil {
		fmt.Printf("[Client %d] Erro ao enviar UDP: %v\n", id, err0)
		return
	}

	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Printf("[Client %d] Erro ao ler resposta UDP: %v\n", id, err)
		return
	}
	elapsed := time.Since(start)

	if strings.TrimSpace(string(buffer[:n])) != message {
		fmt.Printf("[Client %d] Erro: mensagem diferente recebida!\n", id)
	}

	fmt.Printf("[Client %d] RTT: %v\n", id, elapsed)
	
}

func main() {
	fmt.Println("Iniciando teste de stress...")

	readyWG.Add(numClients * 2)
	doneWG.Add(numClients * 2)

	for i := 0; i < numClients; i++ {
		go tcpClient(i)
	}

	for i := 0; i < numClients; i++ {
		go udpClient(i)
	}

	readyWG.Wait()

	fmt.Println("Todos os clientes conectados. Enviando mensagens simultaneamente...")

	close(startCh)

	doneWG.Wait()
	fmt.Println("Teste de stress finalizado!")
}