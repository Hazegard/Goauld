FROM alpine:3.21 AS build

RUN apk add go alpine-sdk
RUN go install github.com/goreleaser/goreleaser/v2@v2.7.0

COPY . /app

WORKDIR /app
ENV PATH="$PATH:/root/go/bin"
RUN go run ./scripts/build/ --gen-age-key=false --gen-access-token=false

FROM alpine:3.21 AS run

COPY --from=build --chmod=755 /app/output/server/*_linux-amd64 /app/server
COPY --from=build /app/output/agent/* /app/build_binaries/
COPY --from=build /app/entrypoint.sh /app/entrypoint.sh

WORKDIR /app
CMD /app/entrypoint.sh