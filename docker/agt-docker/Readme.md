**DOCKER**

**(C) 2023 Volvo Cars**<br>

Running the Access Grant token server in a docker container. The docker file: *Dockerfile.agtserver* is located in the
project root and the current setup have the agt server to listen on port 7500.

To build and run the agt docker
container.
```
cd docker/agt-docker
docker compose -f docker-compose-agt.yml build
docker compose -f docker-compose-agt.yml up
```


