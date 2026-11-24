FROM golang:1.26.4-alpine AS build
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
COPY *.sql ./
RUN apk add --no-cache gcc musl-dev && CGO_ENABLED=1 go test -race ./...
RUN go test ./... && go vet ./... && CGO_ENABLED=0 go build -trimpath -o /server .
FROM scratch
COPY --from=build /server /server
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/server"]
