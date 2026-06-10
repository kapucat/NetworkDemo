FROM golang:1.26.4-alpine

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build -o hook-runner-server .

EXPOSE 8080

CMD ["./hook-runner-server"]