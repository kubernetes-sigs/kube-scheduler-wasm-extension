FROM golang:1.21 AS build-env

ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0
ENV GO111MODULE=on

WORKDIR /go/src/kube-scheduler-wasm-extension

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -v -o ./bin/kube-scheduler-wasm-extension ./cmd/scheduler

FROM alpine:3.19.1

COPY --from=build-env /go/src/kube-scheduler-wasm-extension/bin/kube-scheduler-wasm-extension /kube-scheduler-wasm-extension
RUN chmod a+x /kube-scheduler-wasm-extension

CMD ["/kube-scheduler-wasm-extension"]
