# MI-Concorrencia-e-Conectividade
Jogo de cartas online multiplayer focado em duelos táticos e na coleção de cartas, onde os jogadores devem interagir em um ambiente compartilhado. Este jogo será baseado em um servidor centralizado que gerenciará parte da lógica, o estado dos jogadores e a comunicação entre eles.


## Executando o código

Para executar o código é necessário três terminais sendo um para o servidor e dois para os clientes.

1. **No primeiro terminal**, conecte-se ao servidor:
   ```bash
   docker compose up --build --scale client=2 
   ```

2. **No segundo terminal**, conecte-se ao primeiro cliente:
   ```bash
   docker exec -it go-docker-communication-client-1 /bin/sh
   ```
3. **No terceiro terminal**, conecte-se ao segundo cliente:
   ```bash
   docker exec -it go-docker-communication-client-2 /bin/sh
   ```
