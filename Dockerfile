FROM golang:1.26.2-alpine AS builder

WORKDIR /llama-gateway

RUN apk add --no-cache ca-certificates

ENV CGO_ENABLED=0

COPY . .

RUN go build -o llama-gateway

FROM ubuntu:24.04 AS llama-cpp

RUN apt-get update && \
    apt-get install -y ca-certificates curl tar && \
    rm -rf /var/lib/apt/lists/*

WORKDIR /work

ENV LLAMA_CPP_TAG=b8833
ENV LLAMA_CPP_CHECKSUM=8262b45a82436aefd994f16461d99a02cd1ddf0bb343ef0153186a69229667c7

RUN echo "Downloading llama.cpp version ${LLAMA_CPP_TAG}..." && \
    echo "Expected SHA256 checksum: ${LLAMA_CPP_CHECKSUM}" && \
    echo "Downloading from:  https://github.com/ggml-org/llama.cpp/releases/download/${LLAMA_CPP_TAG}/llama-${LLAMA_CPP_TAG}-bin-ubuntu-x64.tar.gz" && \
    curl -Lo llama.tar.gz "https://github.com/ggml-org/llama.cpp/releases/download/${LLAMA_CPP_TAG}/llama-${LLAMA_CPP_TAG}-bin-ubuntu-x64.tar.gz" && \
    echo "${LLAMA_CPP_CHECKSUM}  llama.tar.gz" > checksum.txt && \
    cat checksum.txt && \
    sha256sum -c checksum.txt && \
    tar -xzf llama.tar.gz --strip-components=1 && \
    rm llama.tar.gz

FROM ubuntu:24.04

COPY ./config/config.yaml /etc/llama-gateway/config.yaml
COPY --from=builder /llama-gateway/llama-gateway /usr/local/bin/
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt
COPY --from=llama-cpp /work /opt/llama.cpp

EXPOSE 8080

CMD ["/usr/local/bin/llama-gateway"]
