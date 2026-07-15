# syntax=docker/dockerfile:1
# check=skip=InvalidDefaultArgInFrom

# GO_VERSION comes from go.mod through scripts/docker_build.sh.
ARG GO_VERSION
FROM golang:${GO_VERSION}-bookworm AS build
WORKDIR /src

ARG TAILWIND_VERSION
ARG TAILWIND_LINUX_X64_SHA256
ARG TAILWIND_LINUX_ARM64_SHA256
ARG HTMX_VERSION
ARG HTMX_SSE_VERSION
ARG ALPINE_VERSION
ARG TEMPL_VERSION
ARG SQLC_VERSION

RUN : "${TAILWIND_VERSION:?missing build arg. Use: mise run docker-build}" \
    && : "${HTMX_VERSION:?missing build arg. Use: mise run docker-build}" \
    && : "${HTMX_SSE_VERSION:?missing build arg. Use: mise run docker-build}" \
    && : "${ALPINE_VERSION:?missing build arg. Use: mise run docker-build}" \
    && : "${TEMPL_VERSION:?missing build arg. Use: mise run docker-build}" \
    && : "${SQLC_VERSION:?missing build arg. Use: mise run docker-build}"

RUN apt-get update \
    && apt-get install -y --no-install-recommends ca-certificates curl \
    && rm -rf /var/lib/apt/lists/*

COPY go.mod go.sum ./
RUN go mod download

RUN go install github.com/a-h/templ/cmd/templ@${TEMPL_VERSION} \
    && go install github.com/sqlc-dev/sqlc/cmd/sqlc@${SQLC_VERSION}

RUN case "$(uname -m)" in \
      x86_64|amd64) arch=x64; expected_sha="${TAILWIND_LINUX_X64_SHA256}" ;; \
      aarch64|arm64) arch=arm64; expected_sha="${TAILWIND_LINUX_ARM64_SHA256}" ;; \
      *) echo "unsupported arch $(uname -m)" >&2; exit 1 ;; \
    esac \
    && curl -fsSL -o /usr/local/bin/tailwindcss "https://github.com/tailwindlabs/tailwindcss/releases/download/${TAILWIND_VERSION}/tailwindcss-linux-${arch}" \
    && if [ -n "$expected_sha" ]; then echo "${expected_sha}  /usr/local/bin/tailwindcss" | sha256sum -c -; fi \
    && chmod +x /usr/local/bin/tailwindcss

COPY . .
RUN HTMX_VERSION=${HTMX_VERSION} HTMX_SSE_VERSION=${HTMX_SSE_VERSION} ALPINE_VERSION=${ALPINE_VERSION} ./scripts/vendor_browser_js.sh
RUN sqlc generate
RUN templ generate
RUN tailwindcss -i ./web/assets/css/input.css -o ./web/assets/css/app.css --minify
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/server ./cmd/server \
    && CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags='-s -w' -o /out/migrate ./cmd/migrate

FROM gcr.io/distroless/static-debian12:nonroot
ARG TAILWIND_VERSION
ARG HTMX_VERSION
ARG HTMX_SSE_VERSION
ARG ALPINE_VERSION
ARG TEMPL_VERSION
ARG SQLC_VERSION
ARG GOOSE_VERSION

ENV HTMX_VERSION=${HTMX_VERSION}
ENV HTMX_SSE_VERSION=${HTMX_SSE_VERSION}
ENV ALPINE_VERSION=${ALPINE_VERSION}
ENV TAILWIND_VERSION=${TAILWIND_VERSION}
ENV TEMPL_VERSION=${TEMPL_VERSION}
ENV SQLC_VERSION=${SQLC_VERSION}
ENV GOOSE_VERSION=${GOOSE_VERSION}

WORKDIR /
COPY --from=build /out/server /server
COPY --from=build /out/migrate /migrate
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/server"]
