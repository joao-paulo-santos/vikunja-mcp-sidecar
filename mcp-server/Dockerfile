FROM golang:1.26-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY *.go ./
RUN CGO_ENABLED=0 go build -o vikunja-mcp .

FROM alpine:3.21
RUN apk add --no-cache ca-certificates
COPY --from=builder /build/vikunja-mcp /usr/local/bin/vikunja-mcp
EXPOSE 3666
ENTRYPOINT ["vikunja-mcp"]
CMD ["-listen", ":3666"]
