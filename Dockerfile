FROM golang:1.27.3-alpine3.23@sha256:754d5200fc3edae76d3b6598a9a3fac7e74981ee6a5413ab955025f185753ef2 AS build

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/github-actions-self-hosted-runner-exporter ./cmd/github-actions-self-hosted-runner-exporter

FROM alpine:3.24@sha256:be2656061b536c1883c4034790e21d0a09c5aaaf51968b89368a3c0371ea2c9a AS trust
RUN apk add --no-cache ca-certificates

FROM gcr.io/distroless/static-debian12:nonroot@sha256:afa5c872c891853ca7fcf1f12c3edb23f7eeef36189728842dd51042ff57f7ab
COPY --from=trust /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

COPY --from=build /out/github-actions-self-hosted-runner-exporter /github-actions-self-hosted-runner-exporter

USER nonroot:nonroot
EXPOSE 9176
ENTRYPOINT ["/github-actions-self-hosted-runner-exporter"]
