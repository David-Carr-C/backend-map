FROM docker.io/golang:1.23.2

WORKDIR /app

COPY go.mod go.sum .

RUN go mod download

COPY . .

RUN go build -o udp-server ./cmd/udp/main.go

EXPOSE 9095/udp

CMD ["./udp-server"]
