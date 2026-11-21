# syntax=docker/dockerfile:1.7

ARG GO_VERSION=1.27.0
ARG GO_ALPINE_VERSION=3.24
ARG ALPINE_VERSION=3.24.0

# NOTE: template/build/ must exist in the build context (CI downloads the
# frontend-build artifact there; locally run `cd template && bun run build`
# first — the same prerequisite as `go build`, which embeds it via
# template/template.go's //go:embed all:build).
FROM golang:${GO_VERSION}-alpine${GO_ALPINE_VERSION} AS backend-deps
WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

FROM backend-deps AS backend-builder
ARG TARGETOS
ARG TARGETARCH

COPY . .
RUN CGO_ENABLED=0 GOOS="${TARGETOS:-linux}" GOARCH="${TARGETARCH:-$(go env GOARCH)}" \
    go build -trimpath -ldflags="-s -w" -o /out/mixtape .

FROM alpine:${ALPINE_VERSION} AS runner
WORKDIR /app

RUN apk add --no-cache ca-certificates ffmpeg yt-dlp

ENV NODE_ENV=production
ENV PORT=:3000

COPY --from=backend-builder --chown=nobody:nobody /out/mixtape ./mixtape
COPY --from=backend-builder --chown=nobody:nobody /src/docs ./docs
# config.Load expects a dotenv file; runtime environment variables override it.
COPY --chown=nobody:nobody .env.example .env

EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=5s --start-period=40s --retries=3 \
  CMD wget -q -O - http://127.0.0.1:3000/api/health || exit 1

USER nobody:nobody
ENTRYPOINT ["/app/mixtape"]
