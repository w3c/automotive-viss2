# (C) 2023 Volvo Cars
#
# All files and artifacts in the repository at https://github.com/w3c/automotive-vissv2
# are licensed under the provisions of the license provided by the LICENSE file in this repository.
ARG GO_VERSION=1.18
ARG VSSTREE_NAME="vss_vissv2.binary"
ARG BUILD_IMAGE="golang:latest"
ARG RUNTIME_IMAGE="debian:bullseye-slim"

#----------------------Builder-----------------------
FROM ${BUILD_IMAGE} AS builder
ARG VSSTREE_NAME
WORKDIR /build

#add bin folder to store the compiled files
RUN mkdir bin

#copy the content of the server and utils dir and .mod/.sum files to builder
COPY redis/redis.conf ./etc/
COPY client/ ./client
COPY feeder/ ./feeder
COPY server/ ./server
COPY grpc_pb/ ./grpc_pb
COPY protobuf/ ./protobuf
COPY utils ./utils
COPY go.mod go.sum ./

RUN ls -a etc/

#copy cert info from testCredGen to path expected by w3c server
COPY testCredGen/ca transport_sec/ca
COPY testCredGen/server transport_sec/server
COPY testCredGen/client transport_sec/client

#clean up unused dependencies
#RUN go mod tidy
#compile all projects and place the executables in the bin folder
RUN go build -v -o ./bin ./...

#----------------------runtime-----------------------
FROM ${RUNTIME_IMAGE} AS runtime
RUN apt-get update && apt-get upgrade -y
RUN apt-get update && apt-get install -y net-tools iproute2 iputils-ping
RUN apt-get autoclean -y
RUN apt-get autoremove -y
COPY --from=builder /build/transport_sec/ ../transport_sec/.

FROM redis
USER root
#RUN apt-get update && apt-get install -y redis-server
COPY --from=builder /build/etc/redis.conf /etc/.
EXPOSE 6379
ENTRYPOINT ["/usr/bin/redis-server", "/etc/redis.conf" ]

FROM golang:1.21-bookworm as feeder
USER root
WORKDIR /app
COPY --from=builder /build/bin/feeder .
COPY --from=builder /build/feeder/certificate.pem .
COPY --from=builder /build/feeder/config.json .
COPY --from=builder /build/feeder/VehicleVssMapData.json .
ENTRYPOINT ["/app/feeder"]


FROM golang:1.21-bookworm as vissv2server
USER root
RUN mkdir transport_sec
WORKDIR /app
RUN mkdir /app/atServer
COPY --from=builder /build/bin/vissv2server .
COPY --from=builder /build/server/transport_sec/transportSec.json ../transport_sec/transportSec.json
COPY --from=builder /build/server/vissv2server/atServer/purposelist.json atServer/purposelist.json
COPY --from=builder /build/server/vissv2server/atServer/scopelist.json atServer/scopelist.json
COPY --from=builder /build/server/vissv2server/feeder-registration.json .
COPY --from=builder /build/server/vissv2server/vss_vissv2.binary .


ENTRYPOINT ["/app/vissv2server","-s","redis"]



