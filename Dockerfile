# syntax=docker/dockerfile:1.7
FROM golang:1.26.5-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/ssb ./cmd/ssb

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=build /out/ssb /ssb
USER nonroot:nonroot
EXPOSE 8080
ENTRYPOINT ["/ssb"]
CMD ["serve"]
