FROM golang:1.25

WORKDIR /app

ENV TZ=Asia/Shanghai

RUN mkdir -p /app/configs
COPY gateway /app/gateway

ENTRYPOINT ["/app/gateway"]