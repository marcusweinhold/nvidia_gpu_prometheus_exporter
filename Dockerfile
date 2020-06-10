FROM golang:1.14 AS builder
WORKDIR /go
COPY go.mod go.sum main.go ./
ENV CGO=0
ENV GOPATH=""
RUN go build

FROM ubuntu:18.04
COPY --from=builder /go/nvidia_gpu_prometheus_exporter /
ENV NVIDIA_VISIBLE_DEVICES=all
EXPOSE 9445
ENTRYPOINT ["/nvidia_gpu_prometheus_exporter"]
