# Stage 1: Build the Go binary with the headless build tag
FROM golang:1.22-alpine AS builder

WORKDIR /build

# Copy Go module files first for layer caching
COPY go/go.mod go/go.sum ./
RUN go mod download

# Copy the rest of the source
COPY go/ .

# Build headless binary (no systray, no X11 dependency)
RUN CGO_ENABLED=0 GOOS=linux go build -tags headless -o clipstreamer .

# Stage 2: Minimal runtime image
FROM alpine:latest

# yt-dlp requires python3; ffmpeg is needed for some clip formats
RUN apk add --no-cache python3 py3-pip ffmpeg && \
    pip3 install --no-cache-dir --break-system-packages yt-dlp

WORKDIR /app
COPY --from=builder /build/clipstreamer .

# All persistent data (config.json, Twitch_Clips/, logs) goes to /data.
# DATA_DIR tells the binary to use /data as its working directory instead
# of the binary directory — so the GUI can save config without needing
# a pre-existing config.json.
ENV DATA_DIR=/data
RUN mkdir /data
VOLUME /data

EXPOSE 42069

CMD ["./clipstreamer"]
