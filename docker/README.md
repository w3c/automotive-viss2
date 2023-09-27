### Using Docker-Compose to launch a W3CServer instance

The W3C server can also be built and launched using docker and docker-compose please follow below links to install docker and docker-compose

https://docs.docker.com/install/linux/docker-ce/ubuntu/
https://docs.docker.com/compose/install/

to launch use

```bash
$ docker-compose up -d
Starting w3c_vehiclesignalinterfaceimpl_servercore_1 ... done
Starting w3c_vehiclesignalinterfaceimpl_servicemgr_1 ... done
Starting w3c_vehiclesignalinterfaceimpl_wsmgr_1      ... done
Starting w3c_vehiclesignalinterfaceimpl_httpmgr_1    ... done
```
to stop use 

```bash
$ docker-compose down
Stopping w3c_vehiclesignalinterfaceimpl_servicemgr_1 ... done
Stopping w3c_vehiclesignalinterfaceimpl_httpmgr_1    ... done
Stopping w3c_vehiclesignalinterfaceimpl_wsmgr_1      ... done
Stopping w3c_vehiclesignalinterfaceimpl_servercore_1 ... done
Removing w3c_vehiclesignalinterfaceimpl_servicemgr_1 ... done
Removing w3c_vehiclesignalinterfaceimpl_httpmgr_1    ... done
Removing w3c_vehiclesignalinterfaceimpl_wsmgr_1      ... done
Removing w3c_vehiclesignalinterfaceimpl_servercore_1 ... done
```

if server needs to be rebuilt due to src code modifications

```bash
$ docker-compose up -d --force-recreate --build
```
all log files are stored in a docker volume to find the local path for viewing the logs use the docker inspect command to find the Mountpoint
```bash
$ docker volume ls
DRIVER              VOLUME NAME
local               w3c_vehiclesignalinterfaceimpl_logdata
$ docker volume inspect w3c_vehiclesignalinterfaceimpl_logdata
[
    {
        "CreatedAt": "2019-12-12T15:32:13+01:00",
        "Driver": "local",
        "Labels": {
            "com.docker.compose.project": "w3c_vehiclesignalinterfaceimpl",
            "com.docker.compose.version": "1.25.1-rc1",
            "com.docker.compose.volume": "logdata"
        },
        "Mountpoint": "/var/lib/docker/volumes/w3c_vehiclesignalinterfaceimpl_logdata/_data",
        "Name": "w3c_vehiclesignalinterfaceimpl_logdata",
        "Options": null,
        "Scope": "local"
    }
]

```
