FROM golang:1.26 AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 go build -o /build/server  ./cmd
RUN CGO_ENABLED=0 go build -o /build/migrate  ./scripts/migrate.go
RUN CGO_ENABLED=0 go build -o /build/seed     ./scripts/seed.go

# ----

FROM alpine:3.20 AS runtime

RUN apk add --no-cache postgresql-client

COPY --from=build /build/server  /server
COPY --from=build /build/migrate /migrate
COPY --from=build /build/seed    /seed
COPY entrypoint.sh /entrypoint.sh

RUN chmod +x /entrypoint.sh

ENTRYPOINT ["/entrypoint.sh"]
