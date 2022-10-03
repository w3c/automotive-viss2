#!/bin/bash

services=(vissv2server agt_server)
#services=(agt_server)
#services=(vissv2server)

usage() {
	#    echo "usage: $0 startme|stopme|configureme" >&2
	echo "usage: $0 startme|stopme" >&2
	echo "usage: Optional parameter: sqlite|redis|none" >&2
}

startme() {
	for service in "${services[@]}"; do
		echo "Starting $service"
		mkdir -p logs
		screen -S $service -dm bash -c "pushd server/$service && go build && mkdir -p logs && ./$service &> ./logs/$service-log.txt && popd"
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

if [ $# -ne 1 ] && [ $# -ne 2 ]
then
	usage $0
	exit 1
fi

case "$1" in 
	startme)
		stopme
		startme $# $2;;
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
