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
	numClients    = 4  // número de clientes simulados
	numMessages   = 2 // número de mensagens que cada cliente envia
)

// WaitGroup para sincronizar "todos prontos"
var readyWG sync.WaitGroup

// WaitGroup para aguardar todos terminarem
var doneWG sync.WaitGroup

// canal usado como "barreira" para soltar todos juntos
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
	go func() {
		for {
			msg, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			_ = msg
		}
	}()

	// avisa que este cliente está pronto
	readyWG.Done()

	// aguarda sinal global
	<-startCh

	// 1) Envia o nome
	name := fmt.Sprintf("Jogador%d\n", id)
	_, _ = conn.Write([]byte(name))

	// Delay entre nome e sequência de "0"
	time.Sleep(7 * time.Second) 

	// 2) Envia "0" três vezes com delay
	for i := 0; i < 3; i++ {
		_, err := conn.Write([]byte("0\n"))
		if err != nil {
			fmt.Printf("[Client %d] Erro ao enviar '0': %v\n", id, err)
			return
		}
		time.Sleep(5 * time.Second) // delay entre os "0"
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

	// 🚀 aguarda sinal para começar
	<-startCh

	for i := 0; i < numMessages; i++ {
		message := fmt.Sprintf("Ping-%d from Client-%d", i, id)

		start := time.Now()
		_, err := conn.Write([]byte(message))
		if err != nil {
			fmt.Printf("[Client %d] Erro ao enviar UDP: %v\n", id, err)
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
}

func main() {
	fmt.Println("Iniciando teste de stress...")

	// todos clientes TCP e UDP contam para a barreira
	readyWG.Add(numClients * 2)
	// todos clientes contam para o final
	doneWG.Add(numClients * 2)

	// dispara vários clientes TCP
	for i := 0; i < numClients; i++ {
		go tcpClient(i)
	}

	// dispara vários clientes UDP
	for i := 0; i < numClients; i++ {
		go udpClient(i)
	}

	// espera todos ficarem prontos
	readyWG.Wait()

	fmt.Println("Todos os clientes conectados. Enviando mensagens simultaneamente...")

	// fecha o canal para liberar todos os clientes de uma vez
	close(startCh)

	// espera todos terminarem
	doneWG.Wait()
	fmt.Println("Teste de stress finalizado!")
}
