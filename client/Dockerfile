FROM golang:1.24-alpine AS builder


WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

WORKDIR /app/client
RUN go build -o client .


FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/client/client .

CMD [ "./client" ]