## Rate Limiter em Go com Redis
Este é um projeto que implementa um Rate Limiter em Go, utilizando o Redis como mecanismo de persistência para controlar o número de requisições feitas por IP ou por token de acesso. O objetivo é limitar o número de requisições por segundo e bloquear usuários ou tokens que excedam o limite.

## Funcionalidade
O Rate Limiter possui duas formas de limitação:

**Limitação por IP**: Limita o número de requisições feitas por um único endereço IP dentro de um intervalo de tempo definido (por padrão, 5 requisições por segundo).

**Limitação por Token**: Limita o número de requisições feitas por um token de acesso específico (por padrão, 10 requisições por segundo).

## Comportamento:
Quando um IP ou token excede o limite de requisições, a resposta será um erro HTTP 429 com a mensagem: "You have reached the maximum number of requests or actions allowed within a certain time frame".

- O Redis armazena o estado das requisições e a contagem de acessos para garantir que o limite seja aplicado de maneira eficiente.

- A chave que armazena a contagem de requisições expira após 1 segundo, garantindo que a verificação seja feita a cada segundo.

## Estrutura do Projeto
main.go: Contém a lógica principal do Rate Limiter e o middleware para interagir com o servidor HTTP.

**Dockerfile**: Configura o ambiente para rodar a aplicação Go dentro de um container.

**docker-compose.yml**: Configura o Docker Compose para subir tanto a aplicação quanto o Redis.

**.env**: Arquivo de configuração para as variáveis de ambiente.

## Como Configurar
```
git clone git@github.com:BillRizer/go-lab-ratelimit.git
cd rate-limiter-go
```
Crie um arquivo `.env` na raiz do projeto
``` 
REDIS_ADDR=localhost:6379          # Endereço do Redis
RATE_LIMIT_IP=5                    # Limite de requisições por segundo por IP
RATE_LIMIT_TOKEN=10                # Limite de requisições por segundo por Token
BLOCK_TIME_SECONDS=300                     # Tempo de bloqueio em segundos (5 minutos)
```

Execute o projeto
```
docker-compose up --build
```

Após isso, o servidor estará rodando na porta `8080`.

---

## Testando a Aplicação
Você pode testar a aplicação fazendo requisições para http://localhost:8080.

Exemplo com curl:
Para testar a limitação por IP (sem um token), basta acessar diretamente o endpoint:
```
curl http://localhost:8080
```

usando token no cabeçalho da requisição:
```
curl -H "API_KEY: seu-token-aqui" http://localhost:8080
```

Caso exceda o maximo de requisicoes permitidas, recebera esta resposta
```
{
  "error": "You have reached the maximum number of requests or actions allowed within a certain time frame"
}
```

# Explicação da Lógica do Rate Limiter
1. Mecanismo de Limitação por IP:
Cada IP é associado a uma chave única no Redis (com o prefixo rate_limiter:ip:), e o contador de requisições é incrementado a cada requisição feita por esse IP. Caso o número de requisições por segundo ultrapasse o limite configurado, a resposta será o erro 429.

2. Mecanismo de Limitação por Token:
Se a requisição contiver um cabeçalho API_KEY com um token de acesso válido, a limitação será aplicada com base nesse token. Cada token é associado a uma chave única no Redis (com o prefixo rate_limiter:token:), e o contador de requisições é incrementado de forma semelhante ao caso do IP.

3. Estratégia de Bloqueio:
Se o limite de requisições for ultrapassado, o IP ou token será bloqueado por um tempo determinado (definido pela variável BLOCK_TIME, em segundos). Durante esse período, as requisições serão rejeitadas.

4. Persistência no Redis:
A contagem de requisições é armazenada no Redis e é associada ao IP ou token. As chaves no Redis expiram a cada 1 segundo, garantindo que o limite de requisições por segundo seja corretamente validado. Após a expiração, a contagem é zerada, permitindo novas requisições.

## Testes Automatizados
Para executar os testes automatizados, utilize:
```
go test
```