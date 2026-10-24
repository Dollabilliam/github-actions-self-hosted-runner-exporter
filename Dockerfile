FROM golang:1.27.3-alpine3.23@sha256:754d5200fc3edae76d3b6598a9a3fac7e74981ee6a5413ab955025f185753ef2 AS build

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /out/github-actions-self-hosted-runner-exporter ./cmd/github-actions-self-hosted-runner-exporter

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=build /out/github-actions-self-hosted-runner-exporter /github-actions-self-hosted-runner-exporter

USER nonroot:nonroot
EXPOSE 9176
ENTRYPOINT ["/github-actions-self-hosted-runner-exporter"]
