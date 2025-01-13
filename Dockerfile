FROM alpine:3.21 as build

RUN apk add go
RUN go install github.com/goreleaser/goreleaser/v2@latest

COPY . /app

WORKDIR /app
RUN go run ./scripts/build.go

FROM alpine:3.21 as run

COPY --from=build /app/output/server/*_linux-amd64 /app/server
COPY --from=build /app/output/agent/* /app/binaries/


