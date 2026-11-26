# syntax=docker/dockerfile:1.7
# =============================================================================
# New Vision Gaming Arcade (nvg-arcade) — Cloud Run + Firebase Hosting + STIG
# =============================================================================
FROM golang:1.24.3-bookworm AS builder

WORKDIR /src
COPY go.mod ./
COPY spec/ ./spec/
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
      org.opencontainers.image.description="New Vision Gaming Classic Games Suite (GoTH + A2A v1.0 + A2UI v1.0 + Firebase Cloud Run)" \
      gov.aegis.crypto.fips140="v1.0.0" \
      gov.aegis.memory_safety="go-no-unsafe-no-cgo"

COPY --from=builder --chmod=0555 /out/nvg-arcade /nvg-arcade
COPY --chmod=0444 dist/release-bundle.signed.json /etc/nvg/release-bundle.signed.json
COPY --chmod=0444 .factory-state/keys/trust-roots.json /etc/nvg/trust-roots.json

ENV GODEBUG="fips140=on" \
    PORT="8080" \
    ADS_ENABLED="false"

USER 65532:65532
EXPOSE 8080

ENTRYPOINT ["/nvg-arcade"]
CMD [ \
  "--require-fips", \
  "--bundle", "/etc/nvg/release-bundle.signed.json", \
  "--trust-roots", "/etc/nvg/trust-roots.json" \
]
