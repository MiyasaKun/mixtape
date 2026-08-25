FROM oven/bun:1.3.14-alpine AS frontend-deps
WORKDIR /src/template

COPY template/package.json template/bun.lock ./
RUN bun install --frozen-lockfile

FROM frontend-deps AS frontend-builder
COPY template ./
RUN bun run build

FROM golang:1.27-alpine AS backend-builder
WORKDIR /src

RUN apk add --no-cache ca-certificates

COPY go.mod go.sum ./
RUN go mod download

COPY . .
COPY --from=frontend-builder /src/template/build ./template/build
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/mixtape .

FROM alpine:3.24 AS runner
WORKDIR /app

RUN apk add --no-cache ca-certificates ffmpeg yt-dlp

ENV NODE_ENV=production
ENV PORT=:3000

COPY --from=backend-builder /out/mixtape ./mixtape
COPY --from=backend-builder /src/docs ./docs
# config.Load expects a dotenv file; runtime environment variables override it.
COPY .env.example .env

EXPOSE 3000
HEALTHCHECK --interval=30s --timeout=5s --start-period=40s --retries=3 \
  CMD wget -q -O - http://127.0.0.1:3000/api/health || exit 1

USER nobody
ENTRYPOINT ["/app/mixtape"]
