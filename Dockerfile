# syntax=docker/dockerfile:1.7
# =============================================================================
# New Vision Gaming Arcade (nvg-arcade) — Self-Contained 1-Click Cloud Run + STIG
# =============================================================================
# Builds directly from a clean git checkout (1-Click "Run on Google Cloud" &
# `gcloud run deploy --source .`). For SI-7 signed-bundle startup attestation
# in the 9-gate Release Software Factory, see Dockerfile.attested.
FROM golang:1.27.4-bookworm AS builder

WORKDIR /src
COPY go.mod ./
COPY app/ ./app/

ENV CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GOPROXY=off \
    GOSUMDB=off \
    GOFLAGS="-mod=readonly" \
    GOFIPS140=v1.0.0 \
    SOURCE_DATE_EPOCH=1735689600 \
    TZ=UTC \
    LC_ALL=C

RUN go test -count=1 ./app/...
RUN go build -trimpath -buildvcs=false -ldflags="-s -w -buildid=" -o /out/nvg-arcade ./app/cmd/nvg-arcade

FROM scratch AS runtime

LABEL org.opencontainers.image.title="nvg-arcade" \
      org.opencontainers.image.description="New Vision Gaming Classic Games Suite (Tic-Tac-Toe & Mexican Train Dominoes — GoTH + A2A v1.0 + A2UI v1.0 + Firebase Cloud Run)" \
      gov.aegis.crypto.fips140="v1.0.0" \
      gov.aegis.memory_safety="go-no-unsafe-no-cgo"

COPY --from=builder --chmod=0555 /out/nvg-arcade /nvg-arcade

ENV GODEBUG="fips140=on" \
    PORT="8080" \
    ADS_ENABLED="false"

USER 65532:65532
EXPOSE 8080

ENTRYPOINT ["/nvg-arcade"]
CMD ["--require-fips"]
