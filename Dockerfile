FROM golang:1.25.7-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags='-s -w' -o /out/ci-capsule ./cmd/ci-capsule

FROM scratch
COPY --from=build /out/ci-capsule /ci-capsule
USER 65532:65532
ENTRYPOINT ["/ci-capsule"]
