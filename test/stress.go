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
	numClients    = 50  // número de clientes simulados
	numMessages   = 100 // número de mensagens que cada cliente envia
)

func tcpClient(id int, wg *sync.WaitGroup) {
	defer wg.Done()

	conn, err := net.Dial("tcp", serverAddrTCP)
	if err != nil {
		fmt.Printf("[Client %d] Erro ao conectar TCP: %v\n", id, err)
		return
	}
	defer conn.Close()

	name := fmt.Sprintf("Jogador%d\n", id)
	conn.Write([]byte(name))

	reader := bufio.NewReader(conn)

	// goroutine para ler mensagens do servidor
	go func() {
		for {
			msg, err := reader.ReadString('\n')
			if err != nil {
				return
			}
			_ = msg // não printa para não poluir
		}
	}()

	// envia mensagens
	for i := 0; i < numMessages; i++ {
		msg := fmt.Sprintf("Mensagem %d do Cliente %d\n", i, id)
		_, err := conn.Write([]byte(msg))
		if err != nil {
			fmt.Printf("[Client %d] Erro ao enviar: %v\n", id, err)
			return
		}
		time.Sleep(10 * time.Millisecond) // pequeno delay
	}
}

func udpClient(id int, wg *sync.WaitGroup) {
	defer wg.Done()

	conn, err := net.Dial("udp", serverAddrUDP)
	if err != nil {
		fmt.Printf("[Client %d] Erro ao conectar UDP: %v\n", id, err)
		return
	}
	defer conn.Close()

	buffer := make([]byte, 1024)

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
		time.Sleep(50 * time.Millisecond)
	}
}

func main() {
	var wg sync.WaitGroup

	fmt.Println("Iniciando teste de stress...")

	// dispara vários clientes TCP
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go tcpClient(i, &wg)
	}

	// dispara vários clientes UDP
	for i := 0; i < numClients; i++ {
		wg.Add(1)
		go udpClient(i, &wg)
	}

	wg.Wait()
	fmt.Println("Teste de stress finalizado!")
}
