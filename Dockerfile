FROM golang:1.22 AS builder
WORKDIR /src
COPY go.mod .
COPY go.sum .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /sokoni ./cmd/sokoni

FROM gcr.io/distroless/base-debian12
WORKDIR /app
COPY --from=builder /sokoni ./sokoni
ENTRYPOINT ["/app/sokoni"]
CMD ["scheduler"]

