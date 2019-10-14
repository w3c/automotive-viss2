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

