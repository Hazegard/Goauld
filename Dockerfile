FROM golang:1.25.1-alpine3.22 AS init

RUN apk add go alpine-sdk upx
RUN go install github.com/goreleaser/goreleaser/v2@v2.7.0
RUN go install mvdan.cc/garble@latest

WORKDIR /app

COPY go.mod /app/go.mod
COPY go.sum /app/go.sum
RUN go mod download

FROM init AS build

ARG COMPRESS=1
COPY . /app
WORKDIR /app

ENV PATH="$PATH:/root/go/bin"
RUN go run ./scripts/build/ --gen-age-key=false --gen-access-token=false --id agent
RUN go run ./scripts/build/ --gen-age-key=false --gen-access-token=false --id server --goos linux --goarch amd64

RUN if [[ "$COMPRESS" == 1 ]]; then \
      for binary in output/agent/*; do \
        if [[ "$binary" != *darwin* && "$binary" != *windows-arm* ]]; then \
          if [[ "$binary" == *.exe ]]; then \
            n="${binary%.exe}"; \
            new="${n}_compressed.exe"; \
          else \
            new="${binary}_compressed"; \
          fi; \
          upx "$binary" --best -o "$new"; \
        else \
          echo "Skipping $binary (darwin/windows-arm build)"; \
        fi; \
      done; \
    fi

FROM alpine:3.22 AS run

COPY --from=build --chmod=755 /app/output/server/*_linux-amd64 /app/server
COPY --from=build /app/output/agent/* /app/build_binaries/
COPY --from=build /app/entrypoint.sh /app/entrypoint.sh

WORKDIR /app
CMD /app/entrypoint.sh