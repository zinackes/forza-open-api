# Build statique multi-stage → image distroless nonroot.
# Le code généré (internal/oas) est committé : pas d'ogen au build.
# Cross-compilation pilotée par buildx (TARGET* auto-remplis par BuildKit) :
# `docker buildx build --platform linux/arm64 .` pour cibler Hetzner CAX (arm64).
FROM --platform=$BUILDPLATFORM golang:1.26 AS build
ARG TARGETOS
ARG TARGETARCH
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/api ./cmd/api
# Scheduler (cron interne du rafraîchissement playlist) — même image, ENTRYPOINT
# surchargé par le service compose dédié.
RUN CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH \
    go build -trimpath -ldflags="-s -w" -o /out/scheduler ./cmd/scheduler

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=build /out/api /api
COPY --from=build /out/scheduler /scheduler
# Contrat embarqué dans l'image (référence runtime, source de vérité).
COPY --from=build /src/api/openapi.yaml /openapi.yaml
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/api"]
