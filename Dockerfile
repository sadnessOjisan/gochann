FROM golang:1.21 as builder

ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64
WORKDIR /build

COPY go.mod ./
COPY go.sum ./
RUN go mod download

COPY . .

RUN go build -o main

#
# Deploy
#
# hadolint ignore=DL3007
FROM gcr.io/distroless/static-debian11:latest

ENV TZ=Asia/Tokyo

WORKDIR /

COPY --from=builder /build/main /main
COPY --from=builder /build/template /template

USER nonroot

EXPOSE 8080

CMD [ "/main" ]