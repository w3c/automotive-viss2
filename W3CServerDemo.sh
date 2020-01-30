#!/bin/bash  

usage() {
    echo "usage: $0 start|stop|restart" >&2
}

startme() {
    screen -d -m -S serverCore bash -c 'cd server-core  && go build && ./server-core'
    screen -d -m -S serviceMgr bash -c 'go run service_mgr.go'
    screen -d -m -S wsMgr bash -c 'go run ws_mgr.go'
    screen -d -m -S httpMgr bash -c 'go run http_mgr.go'
    screen -d -m -S w3cDemoUI bash -c 'go run demoHttpServer.go'
}

stopme() {
    screen -X -S w3cDemoUI quit
    screen -X -S httpMgr quit
    screen -X -S wsMgr quit
    screen -X -S serviceMgr quit
    screen -X -S serverCore quit
    #screen -wipe
}

update() {
   reg='(?<=:\/\/).*(?=:)'
   ips=$(hostname -I | xargs)
   perl -pi -e 's/'$reg'/'$ips'/g' w3cDemo/test.html
   perl -pi -e 's/'$reg'/'$ips'/g' w3cDemo/js/ws-w3c.js
   perl -pi -e 's/'$reg'/'$ips'/g' w3cDemo/js/rest-w3c.js
}

if [ $# -ne 1 ]
then
usage $0
exit 1
fi

case "$1" in 
    start)   update; startme ;;
    stop)    stopme ;;
    restart) stopme; update; startme ;;
    *) usage
       exit 1
       ;;
esac