**DOCKER**

**(C) 2023 Volvo Cars**<br>

The server can also be built and launched using docker and docker-compose please follow below links to install docker and docker-compose

https://docs.docker.com/install/linux/docker-ce/ubuntu/
https://docs.docker.com/compose/install/

The file docker-compose-rl.yml builds and runs  a variant of the feeder(feeder-rl) which is configured and built to interface the remotive labs cloud.
The Remotive cloud have recorded vehicle data which we can play back to a cloud version of their data broker. We have an
interface written in Go - https://github.com/petervolvowinz/viss-rl-interfaces -  that we have integrated into the WAII feeder application. The docker compose version should be from 3.8.

Dockerfile and docker-compose-rl.yml are located in the project root.

To build and run the docker example see below:

```bash
$ docker compose -f docker-compose-rl.yml build 
...
 => => exporting layers                                                                                                                                                                                              0.1s
 => => writing image sha256:f392918448ece4b26b5c430ffc53c5f2da8872d030c11a22b42360dbf9676934                                                                                                                         0.0s
 => => naming to docker.io/library/automotive-viss2-feeder  
```

```bash
$ docker compose -f docker-compose-rl.yml up
  ✔ Container container_volumes  Created                                                                                                                                                                              0.0s 
 ✔ Container vissv2server       Created                                                                                                                                                                              0.0s 
 ✔ Container app_redis          Created                                                                                                                                                                              0.0s 
 ✔ Container feeder             Recreated                                                                                                                                                                            0.1s 
Attaching to app_redis, container_volumes, feeder, vissv2server  
```
to stop use

```bash
$ docker-compose down
✔ Container vissv2server        Stopped                                                                                                                                                                              0.2s 
 ✔ Container app_redis          Stopped                                                                                                                                                                              0.2s 
 ✔ Container feeder             Stopped                                                                                                                                                                              0.1s 
 ✔ Container container_volumes  Stopped 
```

if server needs to be rebuilt due to src code modifications

```bash
$ docker-compose up -d --force-recreate --build

```


**Access control**

If you want to run the server with access control, we need to copy the access grant token server's public key and make
the key available in the container. These keys will be generated at the AGT server startup if not present.

If we are not using access control servers comment this row in the _vissv2server_ section of the Dockerfile in the project
root.

```
COPY --from=builder /build/server/agt_server/agt_public_key.rsa .
```
