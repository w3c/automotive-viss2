# External Vehicle Interface Client Feeder
The EVIC feeder enables the Vehicle Interface client to be implemented as a separate process that is communicating with the feeder over a Websocket protocol, see the figure below.
![EVIC feeder tech stack](pics/VISSv2-tech-stack-EVIC-feeder.jpg?pct=50)<br>

A scenario where this could be of interest is if an implementation of an interface client exists, written in another language then the language of the feeder, the Go language.
This was the case when it was developed, a CAN driver interface was available in Python, which then was extened with the Websocket interface to the EVIC feeder.
Websocket was chosen as th IPC because the developer was familiar with a Python based Websocket library. It is currently not TLS protected,
which should be added before using it for more then demo purposes.
A simple EVIC simulator exists that can be used to verify the communication between the feeder and the external interface client.
