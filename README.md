# MI-Concorrencia-e-Conectividade

Jogo de cartas online multiplayer focado em duelos táticos e na coleção de cartas, onde os jogadores devem interagir em um ambiente compartilhado. Este jogo será baseado em um servidor centralizado que gerenciará parte da lógica, o estado dos jogadores e a comunicação entre eles.

## Como começar

Siga os passos abaixo para baixar e executar o projeto em sua máquina.

### Clonar o repositório

Crie uma pasta e abra seu terminal na pasta e execute o seguinte comando para clonar o repositório:

```bash
git clone https://github.com/davidm34/mi-concorrencia-e-conectividade.git
```

## Executando o Código com Docker

O projeto utiliza Docker Compose para orquestrar os serviços de `server`, `client` e `test`.

### Executar toda a aplicação

Para iniciar todos os serviços (servidor, cliente e teste) e construir as imagens, use o seguinte comando:

```bash
docker compose up --build
```

### Jogar com dois jogadores

Para interagir com cada jogador, você precisa acessar o terminal de cada contêiner cliente em um terminal separado.

Para iniciar um jogo de dois jogadores, inicie o servidor e duas instâncias do cliente. O comando a seguir fará isso:

```bash
docker compose up --build server client --scale client=2
```
1. No primeiro terminal, execute o seguinte comando para acessar o contêiner do Jogador 1:
```bash
docker exec -it mi-concorrencia-e-conectividade-client-1 /bin/sh
```

2. No segundo terminal, execute o seguinte comando para acessar o contêiner do Jogador 2:
```bash
docker exec -it mi-concorrencia-e-conectividade-client-2 /bin/sh
```

Para começar a execução do cliente, você precisa rodar o seguinte comando nos dois terminais:
```bash
go run main.go
```
