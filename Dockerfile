FROM golang:1.25 AS build
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -o /out/mcp-data-gateway ./cmd/mcp-data-gateway

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/mcp-data-gateway /mcp-data-gateway
USER nonroot:nonroot
ENTRYPOINT ["/mcp-data-gateway"]
