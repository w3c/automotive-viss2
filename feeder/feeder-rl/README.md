**(C) 2023 Ford Motor Company**<br>
**(C) 2023 Volvo Cars**<br>

A feeder version that interfaces the remotive labs signal broker : https://demo.remotivelabs.com/orgs/remotidemo.
It is assuming VSS translated signals, which means that proprietary signals are already translated by the broker.

The feeder needs to setup the broker adress/url together with an api-key provided by remotiveLabs. If tls is used for a more 
secure communication a client cert can be generated. 

The signal filter specifies which signals we would like to filter out from them stream. It is expected by the broker.


```json
{
"tls": "yes",
"cert_path_name": "certificate.pem",
"name_spaces": ["vss"],
"broker_url": "<broker url>",
"port":"443",
"client_id" : "volvo-go-client",
"api_key": "<api-key>",
"vss_tree_path": "../vss/vss-flat-json/normalized-json/vss_n.json",
"signalfilter": ["Vehicle.Speed","Vehicle.VehicleIdentification.Model","Vehicle.VehicleIdentification.VIN"]
}
```


The feeder should be started with the redisDb, which presumes that the vissv2server use redis https://redis.io for state storage.
```
    feeder-rl --dataprovider, remotive, --rdb, /tmp/docker/redisDB.sock,--fch,/tmp/docker/server-feeder-channel.sock
```

The file VehicleVssMapData.json is used if the feeder is executed with
```
--dataprovider sim
```

The docker-compose file is located in docker/viss-docker-rl folder.
The Docker file is located the project root:  *Dockerfile.rlserver*

To build and run.
```
cd docker/viss-docker-rl
docker compose -f docker-compose-rl.yml build
docker compose -f docker-compose-rl.yml up
```


