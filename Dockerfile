FROM golang:alpine AS build

RUN apk update && apk add --no-cache ca-certificates && update-ca-certificates

WORKDIR /go/src/nicolasparada/nakama

COPY go.mod .
COPY go.sum .
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /bin/nakama

FROM scratch

ARG EMBED_STATIC=true
ENV EMBED_STATIC=$EMBED_STATIC

ARG EXEC_SCHEMA=true
ENV EXEC_SCHEMA=$EXEC_SCHEMA

COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=build /bin/nakama /bin/nakama

EXPOSE 3000
ENTRYPOINT [ "/bin/nakama" ]
