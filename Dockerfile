FROM golang:alpine AS build

RUN apk add --update --no-cache nodejs ca-certificates && apk add --no-cache npm && update-ca-certificates

WORKDIR /go/src/github.com/nicolasparada/nakama

COPY . .

RUN cd web/app && npm i && npm run build && cd ../..

RUN go mod download && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-s -w" -o /go/bin/nakama ./cmd/nakama

FROM scratch

ARG DISABLE_DEV_LOGIN=true
ENV DISABLE_DEV_LOGIN=$DISABLE_DEV_LOGIN

ARG EMBED_STATIC=true
ENV EMBED_STATIC=$EMBED_STATIC

ARG EXEC_SCHEMA=true
ENV EXEC_SCHEMA=$EXEC_SCHEMA

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /go/bin/nakama /usr/bin/nakama

EXPOSE 3000
ENTRYPOINT [ "nakama" ]
