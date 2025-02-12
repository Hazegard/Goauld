FROM alpine:3.21 AS build

RUN apk add go alpine-sdk
RUN go install github.com/goreleaser/goreleaser/v2@latest

COPY . /app

WORKDIR /app
ENV PATH="$PATH:/root/go/bin"
RUN go run ./scripts/build/ --gen-age-key=false --gen-access-token=false

# FROM alpine:3.21 AS run

# COPY --from=build --chmod=644 /app/output/server/*_linux-amd64 /app/server
# COPY --from=build /app/output/agent/* /app/binaries/
