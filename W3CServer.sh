#!/bin/bash

services=(server_core service_mgr at_server agt_server http_mgr ws_mgr mqtt_mgr)

usage() {
	#    echo "usage: $0 startme|stopme|configureme" >&2
	echo "usage: $0 startme|stopme" >&2
}

startme() {
	for service in "${services[@]}"; do
		echo "Starting $service"
		mkdir -p logs
		if [ $service == "service_mgr" ]; then
      screen -S $service -dm bash -c "pushd server/$service && go build && mkdir -p logs && ./$service &> ./logs/$service-log.txt && popd"
		else
			screen -S $service -dm bash -c "pushd server/$service && go build && mkdir -p logs && ./$service &> ./logs/$service-log.txt && popd"
		fi
	done
	screen -list
}

stopme() {
	for service in "${services[@]}"; do
		echo "Stopping $service"
		screen -X -S $service quit
		#killall -9 $service	
	done
	sleep 1
	screen -wipe
}

#configureme() {
#ln -s <absolute-path-to-dir-of-git-root>/WAII/server/Go/server-1.0/vendor/utils $GOPATH/src/utils
#}

if [ $# -ne 1 ]
then
	usage $0
	exit 1
fi

case "$1" in 
	startme)
		stopme
		startme ;;
	stopme)
		stopme
		;;
	#configureme)
		#	configureme
		#	;; 
	*)
		usage
		exit 1
		;;
esac
