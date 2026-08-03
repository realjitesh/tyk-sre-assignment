FROM golang:1.19-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o tyk-sre-assignment .

FROM alpine:3.20

WORKDIR /app

COPY --from=builder /app/tyk-sre-assignment .

EXPOSE 8080

ENTRYPOINT ["./tyk-sre-assignment"]