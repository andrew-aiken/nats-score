FROM public.ecr.aws/docker/library/golang:1.26.2 AS builder
WORKDIR /app

COPY go.mod go.sum /app/

RUN go mod download

COPY pkg/ /app/pkg/
COPY cmd/ /app/cmd/

RUN CGO_ENABLED=0 GOOS=linux go build -o /agent /app/cmd/agent


# https://github.com/GoogleContainerTools/distroless
FROM gcr.io/distroless/static-debian13

WORKDIR /

COPY --chown=root:nonroot --chmod=050 --from=builder /agent /agent

USER nonroot:nonroot

ENTRYPOINT ["/agent"]
