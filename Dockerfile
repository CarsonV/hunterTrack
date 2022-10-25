# syntax=docker/dockerfile:1
#BUILD
FROM golang:1.19-bullseye as builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./

RUN go build -o main .

#DEPLOY
FROM debian:bullseye-slim

WORKDIR /app

COPY --from=builder /app/main .

USER nonroot:nonroot

CMD ["./main"]
