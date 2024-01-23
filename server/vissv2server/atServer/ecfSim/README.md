# External Consent Framework Simulator
This ECF simulator exposes a simple UI in the terminal window it runs in, after being built and started.<br>
$ go build<br>
$ ./ecfSim

The protocol between the VISSv2 AT server and and ECF is described in the tutorial chapter <a href="https://w3c.github.io/automotive-viss2/server/access-control-servers/">VISSv2 Access Control Servers</a>.
When a consent request message is received from the VISSv2 AT servera question is shown which response to the request message that should be used (Ok response/Error response).
After responding to that the simulator asks whether the answer to the consent request should be answered with YES or NO, or if the answer shold be delayed.

If NO, the simulator sends a consent reply message to the server.

If YES, the simulator  sends a consent reply message to the server, and then asks whether the consent shall be cancelled at a later point. If the answer to this is Yes,
then the simulator asks for how many seconds it should wait from now until it sends a consent callcellation message to the server.

If delayed, the simulator asks for how many seconds it should delay the consent reply message. When this time period is elapsed, the simulator will ask which consent reply to send (YES/NO),
before sending it.

This simple UI is repeated for every consent request message that is received from the server.

There can only be one delayed consent reply at a time, delaying a second before the first is elapsed leads to the first one being lost.
