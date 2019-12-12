ARG GO_VERSION=1.13
ARG DEFAULT_APP_NAME="servercore"
ARG DEFAULT_VSSTREE_NAME="vss_rel_1.0.cnative"
ARG DEFAULT_APP_PATH="/app"
ARG DEFAULT_SRCDIR="server/server-core/"


FROM golang:${GO_VERSION}-alpine AS dev
ARG DEFAULT_APP_NAME
ARG DEFAULT_APP_PATH
ARG DEFAULT_SRCDIR

ENV APP_NAME=${DEFAULT_APP_NAME} \
    APP_PATH=${DEFAULT_APP_PATH}

WORKDIR ${APP_PATH}

COPY ${DEFAULT_SRCDIR} .
COPY go.mod go.sum ./
#RUN ln -s utils/ ./utils
#RUN go get -u ./...

#install gcc and std lib
RUN apk add --no-cache gcc musl-dev
#compile servercore
RUN go build -o ${APP_NAME} .

FROM alpine
ARG DEFAULT_APP_NAME
ARG DEFAULT_APP_PATH
ARG DEFAULT_VSSTREE_NAME

ENV APP_NAME=${DEFAULT_APP_NAME} \
    APP_PATH=${DEFAULT_APP_PATH} \
    VSSTREE=${DEFAULT_VSSTREE_NAME}

WORKDIR ${APP_PATH}
COPY --from=dev ${APP_PATH}/${APP_NAME} .
COPY --from=dev ${APP_PATH}/${VSSTREE} .

#transportRegPort, transportDataPort [8100-]
#EXPOSE 8081
#EXPOSE 8100

#serviceRegPort, serviceDataPort [8200-]
#EXPOSE 8082
#EXPOSE 8200

ENTRYPOINT ./${APP_NAME}
CMD ""
