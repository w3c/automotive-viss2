#!/bin/bash  

startme() {
    screen -d -m -S serverCore bash -c 'cd server-core  && go build && ./server-core'
    screen -d -m -S serviceMgr bash -c 'go run service_mgr.go'
    screen -d -m -S wsMgr bash -c 'go run ws_mgr.go'
    screen -d -m -S httpMgr bash -c 'go run http_mgr.go'
}

stopme() {
    screen -X -S httpMgr quit
    screen -X -S wsMgr quit
    screen -X -S serviceMgr quit
    screen -X -S serverCore quit
    #screen -wipe
}

configureme() {
    #ln -s <absolute-path-to-dir-of-git-root>/W3C_VehicleSignalInterfaceImpl/server/Go/server-1.0 $GOPATH/src/server-1.0
}

if [ $1 = startme ]
then
startme
fi

if [ $1 = stopme ]
then
stopme
fi

if [ $1 = configureme ]
then
configureme
fi

