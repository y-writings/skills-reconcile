FROM node:22.20.0-bookworm-slim@sha256:b21fe589dfbe5cc39365d0544b9be3f1f33f55f3c86c87a76ff65a02f8f5848e AS node

FROM golang:1.24.0-bookworm@sha256:b970e6d47c09fdd34179acef5c4fecaf6410f0b597a759733b3cbea04b4e604a AS toolchain
CMD []

COPY --from=node /usr/local/bin/node /usr/local/bin/node
COPY --from=node /usr/local/lib/node_modules /usr/local/lib/node_modules
RUN ln -s ../lib/node_modules/npm/bin/npm-cli.js /usr/local/bin/npm \
    && ln -s ../lib/node_modules/npm/bin/npx-cli.js /usr/local/bin/npx

WORKDIR /opt/skills
COPY tools/skills/package.json tools/skills/package-lock.json ./
RUN npm ci --ignore-scripts

ENV GOTOOLCHAIN=local
ENV SKILLS_RECONCILE_EXECUTABLE=/opt/skills/node_modules/.bin/skills

WORKDIR /workspace

FROM toolchain AS test

COPY . ./

RUN test "$(go env GOVERSION)" = "go1.24.0" \
    && test "$(node --version)" = "v22.20.0" \
    && skills_version="$($SKILLS_RECONCILE_EXECUTABLE --version)" \
    && test "$skills_version" = "1.5.23"
RUN unformatted="$(gofmt -l .)" \
    && test -z "$unformatted"
RUN go test ./... \
    && go vet ./... \
    && go build -trimpath -o /usr/local/bin/skills-reconcile ./cmd/skills-reconcile \
    && skills-reconcile --help >/dev/null

FROM toolchain AS development

CMD ["bash"]
