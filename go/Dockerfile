FROM golang:1.27-alpine AS builder
WORKDIR /app

RUN apk add --no-cache git

ENV CGO_ENABLED=0

COPY go.mod go.sum ./
RUN go mod download

COPY tools/ ./tools/
RUN go install -C ./tools github.com/pressly/goose/v3/cmd/goose

COPY . .

RUN go build -o main .

FROM alpine:3 AS runner
WORKDIR /app

COPY --from=builder /app/main .

COPY --from=builder /go/bin/goose /usr/local/bin/goose

#
# Add to your post-deploy:
#
#   goose -dir ./sql/migrations postgres "$CLIENTS_DB_DSN" up
#

COPY --from=builder /go/bin/goose /usr/local/bin/goose
COPY sql/migrations/ ./sql/migrations/

EXPOSE 9500

RUN adduser -D gouser
USER gouser

ENTRYPOINT ["./main"]
