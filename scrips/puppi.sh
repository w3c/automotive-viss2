#!/bin/bash

usage() {
    echo "usage: $0 startme|stopme" >&2
}

startme() {
    stopme #stoping zombie screens before starting new ones
    echo "starting ..."
    screen -dmS servercore ./servercore
    sleep 5s
    screen -dmS servicemgr ./servicemgr
    screen -dmS wsmgr bash /wsmgr
    screen -dmS httpmgr ./http_mgr
    screen -dmS agtserver ./agt-server
    screen -dmS atserver ./at-server
}

stopme() {
    screen -X -S atserver quit
    screen -X -S agtserver quit
    screen -X -S httpmgr quit
    screen -X -S wsmgr quit
    screen -X -S servicemgr quit
    screen -X -S servercore quit
}

if [ $# -ne 1 ]
    then
        usage $0
        exit 1
fi

case "$1" in 
    startme) startme ;;
    stopme) stopme ;;
    *) usage
        exit 1
        ;;
esac
