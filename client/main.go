package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

type Response struct {
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
}

func main() {
	for {
		start := time.Now()

		resp, err := http.Get("http://server:8080/ping")
		if err != nil {
			log.Println("Erro ao conectar no servidor:", err)
			time.Sleep(5 * time.Second)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		
		var data Response
		if err := json.Unmarshal(body, &data); err != nil {
			log.Println("Erro ao decodificar resposta:", err)
			time.Sleep(5 * time.Second)
			continue
		}

		rtt := time.Since(start)

		fmt.Printf(
			"Mensagem: %s | Servidor recebeu em: %s | RTT: %v\n",
			data.Message,
			data.Timestamp.Format(time.RFC3339Nano),
			rtt,
		)

		time.Sleep(5 * time.Second)
	}
}
