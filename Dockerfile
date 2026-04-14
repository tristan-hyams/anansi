FROM golang:1.26-alpine AS build

WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true
COPY . .
RUN CGO_ENABLED=0 go build -o /anansi ./cmd/anansi

FROM alpine:3.23
RUN apk add --no-cache ca-certificates
COPY --from=build /anansi /usr/local/bin/anansi
ENTRYPOINT ["anansi"]
