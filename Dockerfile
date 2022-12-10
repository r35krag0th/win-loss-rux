ARG ldflags
ARG app_version

FROM    golang:1.19-alpine3.15 AS build
WORKDIR /go/src/github.com/r35krag0th/win-loss-rux
COPY    . .
RUN     GO111MODULE=on CGO_ENABLED=0 go build \
          -o bin/win-loss \
        -ldflags "-X=github.com/r35krag0th/win-loss-rux/v1/version.Version=${app_version}"

FROM    ghcr.io/r35krag0th/alpine-with-utils:3.15

RUN     adduser -D app
COPY    --from=build /go/src/github.com/r35krag0th/win-loss-rux/bin/* /usr/bin
USER    app
ENTRYPOINT ["win-loss"]

