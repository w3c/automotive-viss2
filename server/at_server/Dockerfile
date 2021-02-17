#editable variables
ARG GO_VERSION=1.13
ARG DEFAULT_APP_NAME="at_server"
ARG DEFAULT_VSSTREE_NAME="vss_gen2.cnative"
ARG DEFAULT_APP_PATH="/app"
ARG DEFAULT_SRCDIR="server/${DEFAULT_APP_NAME}/"

#dont edit below this line
FROM golang:${GO_VERSION}-alpine AS dev
ARG DEFAULT_APP_NAME
ARG DEFAULT_APP_PATH
ARG DEFAULT_SRCDIR

ENV APP_NAME=${DEFAULT_APP_NAME} \
    APP_PATH=${DEFAULT_APP_PATH}

WORKDIR ${APP_PATH}
COPY ${DEFAULT_SRCDIR} .
COPY go.mod go.sum ./

#install gcc and std lib
RUN apk add --no-cache gcc musl-dev

#compile atserver
RUN go build -o ${APP_NAME} .

FROM alpine AS at-server
ARG DEFAULT_APP_NAME
ARG DEFAULT_APP_PATH
ARG DEFAULT_VSSTREE_NAME

ENV APP_NAME=${DEFAULT_APP_NAME} \
    APP_PATH=${DEFAULT_APP_PATH} \
    VSSTREE=${DEFAULT_VSSTREE_NAME}

WORKDIR ${APP_PATH}
#copy binary
COPY --from=dev ${APP_PATH}/${APP_NAME} .
COPY --from=dev ${APP_PATH}/${VSSTREE} .

#copy *.json (purpose/scope) maybe these should be moved to
#config folder and mounted so that they can be changed without
#rebuilding the container
COPY --from=dev ${APP_PATH}/purposelist.json .
COPY --from=dev ${APP_PATH}/scopelist.json .

ENTRYPOINT ./${APP_NAME}
CMD ""
