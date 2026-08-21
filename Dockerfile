# Frontend Build
FROM public.ecr.aws/docker/library/node:26.5.0-alpine3.24 AS frontend
WORKDIR /app

COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --ignore-scripts && \
    npm cache clean --force

COPY frontend/index.html frontend/vite.config.ts frontend/tsconfig.json ./
COPY frontend/public ./public
COPY frontend/src ./src

RUN npm run build


# Server build
FROM docker.io/library/golang:1.26.5 AS server
WORKDIR /go/src/app

COPY server/go.mod server/go.sum ./
RUN go mod download

COPY server/cmd/ ./cmd
COPY server/internal/ ./internal

COPY --from=frontend /app/dist ./internal/static/dist

RUN CGO_ENABLED=0 go build -o /go/bin/score ./cmd/cli/main.go


FROM gcr.io/distroless/static-debian13:nonroot
USER nonroot:nonroot

COPY --from=server --chown=nonroot:nonroot --chmod=500 /go/bin/score /

ENTRYPOINT ["/score"]
CMD [ "server", "start" ]
