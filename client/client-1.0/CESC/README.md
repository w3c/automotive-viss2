**(C) 2022 Geotab Inc**<br>

# Cloud Edge Server Client (CESC)

The CESC is deployed in the cloud and implements a server that listens for connects from in-vehicle based clients like the VEC client, 
and a client that forwards the received veicle data to a cloud backend via an API that the backend exposes for this ingestion.
The two components, the server and the client, communicate the vehicle data via an OVDS structured SQLite DB. 
For OVDS info see <a href="https://github.com/COVESA/ccs-components/tree/master/ovds/server">Open Vehicle Data Set</a><br>

The in-vehicle VEC requests vehicle data from the in-vehicle VISSv2 server, and then pushes the data to the CESC in the cloud.<br>

The CESC listens for HTTPS(/HTTP) requests on port 4443 (8000 if TLS is not configured), where the payload of the POST requests are expected to have the following JSON based format:<br>
{"vin":"XXX","data":YYY}<br>
where XXX is the vehicle identification that the VISSv2 server responds to on a request for the VSS path "Vehicle.VehicleIdentification.VIN", 
and YYY is one of the four possible data objects that is specified in the W3C VISSv2 specification.<br>

If the CESC is started with the command argument "-h" the following is shown:<br>
<pre>
usage: print [-h|--help] [--logfile] [--loglevel
             (trace|debug|info|warn|error|fatal|panic)]

             Cloud Edge Server Client

Arguments:

  -h  --help      Print help information
      --logfile   outputs to logfile in ./logs folder
      --loglevel  changes log output level. Default: info
</pre>

The vehicle data is temporarily stored in an OVDS DB named ovdsCESC.db, until either a maximum of new data points have been stored, or a timer is triggered.<br>
The backend API that is implemented in this version of the CESC is the Geotab Data Intake Gateway (DIG) API, that is described in this document:<br>
<a href="https://docs.google.com/document/d/1XFHQ1s-um6HcW3qPRNiKX7bj-_X-O--4Fj4_j_An8U0/edit?usp=sharing">Data Intake Gateway (DIG) API</a>

