# (C) 2021 Mitsubishi Electric Automotive Europe B.V.
#
# All files and artifacts in the repository at https://github.com/MEAE-GOT/WAII
# are licensed under the provisions of the license provided by the LICENSE file in this repository.

#ARG GO_VERSION=1.16
ARG VSSTREE_NAME="vss_vissv2.binary"
ARG BUILD_IMAGE="golang:latest" 
#${GO_VERSION}-bullseye"
ARG RUNTIME_IMAGE="debian:bullseye-slim"

#----------------------Builder-----------------------
FROM ${BUILD_IMAGE} AS builder
ARG VSSTREE_NAME
WORKDIR /build
#2021-09-03:: not used since moved away from alpine in favor of debian,
    #install gcc and std lib
    #RUN apk add --no-cache gcc musl-dev git

#add bin folder to store the compiled files
RUN mkdir bin

#copy the content of the server and utils dir and .mod/.sum files to builder
COPY server/ ./server
COPY grpc_pb/ ./grpc_pb
COPY protobuf/ ./protobuf
COPY utils ./utils
COPY go.mod go.sum ./

#copy cert info from testCredGen to path expected by w3c server 
COPY testCredGen/ca transport_sec/ca
COPY testCredGen/server transport_sec/server
COPY testCredGen/client transport_sec/client

#remove these since these arent currently buildable and shouldnt be included
RUN rm -rf test
RUN rm -rf signal_broker
#RUN rm hist_ctrl_client.go

#clean up unused dependencies
#RUN go mod tidy
#compile all projects and place the executables in the bin folder
RUN go build -v -o ./bin ./...

# 2021-09-03:: add and compile the ccs-w3c ovds client 
RUN git clone https://github.com/GENIVI/ccs-w3c-client.git
RUN cd ccs-w3c-client/ovds && go mod tidy && go build -v -o /build/bin ./...
#RUN echo $(ls -lah bin)
#----------------------DONE with builder-----------------------


#----------------------runtime-----------------------
FROM ${RUNTIME_IMAGE} AS runtime
RUN apt-get update && apt-get upgrade -y
RUN apt-get update && apt-get install -y net-tools iproute2 iputils-ping
RUN apt-get autoclean -y
RUN apt-get autoremove -y
COPY --from=builder /build/transport_sec/ ../transport_sec/.
#----------------------DONE with runtime-----------------------


#----------------------server_core-----------------------
FROM runtime AS server_core
ARG VSSTREE_NAME
WORKDIR /app
COPY --from=builder /build/bin/server_core .
COPY --from=builder /build/server_core/${VSSTREE_NAME} .
RUN ["/bin/bash","-c","/app/server_core --dryrun"]
ENTRYPOINT ["/app/server_core"]
#----------------------DONE with server_core-----------------------

#----------------------at_server-----------------------
FROM runtime AS at_server
ARG VSSTREE_NAME
WORKDIR /app
COPY --from=builder /build/bin/at_server .
COPY --from=builder /build/at_server/${VSSTREE_NAME} .
#copy *.json (purpose/scope) maybe these should be moved to
#config folder and mounted so that they can be changed without
#rebuilding the docker image
COPY --from=builder /build/at_server/purposelist.json .
COPY --from=builder /build/at_server/scopelist.json .
ENTRYPOINT ["/app/at_server"]
#----------------------DONE with at_server-----------------------

#----------------------agt_server-----------------------
FROM runtime AS agt_server
WORKDIR /app
COPY --from=builder /build/bin/agt_server .
ENTRYPOINT ["/app/agt_server"]
#----------------------DONE with agt_server-----------------------

#----------------------service_mgr-----------------------
FROM runtime AS service_mgr
WORKDIR /app
RUN mkdir -p /tmp/vissv2/
COPY --from=builder /build/bin/service_mgr .
#this copy can be problematic since it generated by server_core at runtime
#20210219 fix added in service_mgr to handle the missing file more gracefully :)
ENTRYPOINT ["/app/service_mgr -uds &> ./logs/$service-log.txt"]
#----------------------DONE with service_mgr-----------------------

#----------------------http_mgr-----------------------
FROM runtime AS http_mgr
WORKDIR /app
COPY --from=builder /build/bin/http_mgr .
ENTRYPOINT ["/app/http_mgr"]
#----------------------DONE with http_mgr-----------------------

#----------------------ws_mgr-----------------------
FROM runtime AS ws_mgr
WORKDIR /app
COPY --from=builder /build/bin/ws_mgr .
ENTRYPOINT ["/app/ws_mgr"]
#----------------------DONE with ws_mgr-----------------------

#----------------------mqtt_mgr-----------------------
FROM runtime AS mqtt_mgr
WORKDIR /app
COPY --from=builder /build/bin/mqtt_mgr .
ENTRYPOINT ["/app/mqtt_mgr"]
#----------------------DONE with mqtt_mgr-----------------------

#----------------------tls_client-----------------------
FROM runtime AS tls_client
WORKDIR /app
COPY --from=builder /build/bin/client .
COPY --from=server_core /vsspathlist.json .
#2021-09-06 transport_sec path is different for ccs-client so 
#adding a symbolic link
RUN ln -s ../transport_sec/ transport_sec
ENTRYPOINT ["/app/client"]
#----------------------DONE with tls_client-----------------------