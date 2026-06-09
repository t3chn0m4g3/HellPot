FROM golang:1.26.4 AS build
WORKDIR /go/src/app

COPY go.* .
RUN go mod download

COPY . .

RUN go vet -v ./...
RUN go test -v ./...
ARG VERSION=0.60
RUN CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X main.version=$VERSION" \
    -o /out/HellPot \
    ./cmd/HellPot


FROM scratch
LABEL org.opencontainers.image.source="https://github.com/t3chn0m4g3/hellpot"

COPY --from=build /out/HellPot /app
COPY --from=build /go/src/app/docker_config.toml /config
EXPOSE 8080
USER 65532:65532
ENTRYPOINT ["/app", "-c", "/config"]
