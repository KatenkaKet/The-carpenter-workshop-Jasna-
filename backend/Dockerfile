FROM golang:1.26-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/jasna ./cmd/server

FROM alpine:3.23
RUN apk add --no-cache ca-certificates tzdata && addgroup -S jasna && adduser -S -G jasna jasna
COPY --from=build /out/jasna /usr/local/bin/jasna
USER jasna
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/jasna"]
