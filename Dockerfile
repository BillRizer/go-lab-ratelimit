FROM golang:1.24-alpine as builder

WORKDIR /app

COPY .env .
COPY . .

RUN go mod tidy

RUN go build -o main .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /root/

COPY --from=builder /app/main .
COPY --from=builder /app/.env .

EXPOSE 8080

CMD ["./main"]
