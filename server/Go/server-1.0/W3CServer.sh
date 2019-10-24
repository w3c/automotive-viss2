#!/bin/bash  



startme() {
    echo "starting ..."
    screen -d -m -S serverCore bash -c 'cd server-core  && go build && ./server-core'
    screen -d -m -S serviceMgr bash -c 'go build service_mgr.go managerdata.go managerhandlers.go  && ./service_mgr'
    screen -d -m -S wsMgr bash -c 'go build ws_mgr.go managerdata.go managerhandlers.go  && ./ws_mgr'
    screen -d -m -S httpMgr bash -c 'go build http_mgr.go managerdata.go managerhandlers.go  && ./http_mgr'
}

stopme() {
    screen -X -S httpMgr quit
    screen -X -S wsMgr quit
    screen -X -S serviceMgr quit
    screen -X -S serverCore quit
    #screen -wipe
}

configureme() {
    ln -s <absolute-path-to-dir-of-git-root>/W3C_VehicleSignalInterfaceImpl/server/Go/server-1.0/vendor/utils $GOPATH/src/utils
}

if [ "$1" = "startme" ]
then
startme
fi

if [ "$1" = "stopme" ]
then
stopme
fi

if [ "$1" = "configureme" ]
then
configureme
fi

