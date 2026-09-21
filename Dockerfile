FROM golang:1.24.0-bookworm@sha256:b970e6d47c09fdd34179acef5c4fecaf6410f0b597a759733b3cbea04b4e604a AS toolchain
CMD []

ENV GOTOOLCHAIN=local

WORKDIR /workspace

FROM toolchain AS test

ENV HOME=/tmp/skills-reconcile-home
ENV XDG_CONFIG_HOME=/tmp/skills-reconcile-xdg/config
ENV XDG_STATE_HOME=/tmp/skills-reconcile-xdg/state
ENV XDG_CACHE_HOME=/tmp/skills-reconcile-xdg/cache

COPY . ./

RUN test "$(go env GOVERSION)" = "go1.24.0"
RUN unformatted="$(gofmt -l .)" \
    && test -z "$unformatted"
RUN go test ./... \
    && go vet ./... \
    && CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" \
        -o /usr/local/bin/skills-reconcile ./cmd/skills-reconcile

FROM scratch AS runtime

COPY --from=test /usr/local/bin/skills-reconcile /usr/local/bin/skills-reconcile

ENTRYPOINT ["/usr/local/bin/skills-reconcile"]
