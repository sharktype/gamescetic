FROM golang:1.27-alpine AS builder
WORKDIR /app

RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY tools/ ./tools/
RUN go install -C ./tools github.com/pressly/goose/v3/cmd/goose

COPY . .

ENV CGO_ENABLED=0

RUN go build -o main .

FROM golang:1.27-alpine AS runner
WORKDIR /app

COPY --from=builder /app/main .
COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY sql/migrations/ ./sql/migrations/

EXPOSE 8888

RUN adduser -D gouser
USER gouser

ENTRYPOINT ["./main"]
