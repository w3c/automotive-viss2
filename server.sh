#!/bin/bash
HTTP_SVR=0
WS_SVR=1
CORE_SVR=2
SVCMGR_SVR=3
AGT_SVR=4
ATT_SVR=5
declare -a svrfolders=("server/http_mgr"
                        "server/ws_mgr"
                        "server/server_core" 
                        "server/service_mgr"
                        "client/client-1.0"
                        "server/at_server")

declare -a svrbin=("http_mgr" 
                        "ws_mgr" 
                        "server_core" 
                        "service_mgr"
                        "agt_server"
                        "at_server")

declare -a svrname=("HTTP SERVER"
                        "WEB SOCKET SERVER"
                        "SERVER CORE"
                        "SERVICE MANAGER"
                        "ACCESS GRANT SERVER"
                        "ACCESS TOKEN SERVER")

usage()
{
    echo 'Launch Server'

    echo "1. HTTP SERVER"
    echo "2. WEB SOCKET SERVER"
    echo "3. SERVER CORE"
    echo "4. SERVICE MANAGER"
    echo "5. ACCESS GRANT SERVER"
    echo "6. ACCESS TOKEN SERVER"
}

if [ $# -ne 1 ];
then
    usage
else
    pushd ${svrfolders[$1 - 1]}
    echo ${svrfolders[$1 - 1]}
    pwd
    ${svrbin[$1 - 1]}
    echo ${svrbin[$1 - 1]}
    echo ${svrname[$1 - 1]} script executed.
fi
