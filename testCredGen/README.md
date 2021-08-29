The testCredGen script can be used to generate credentials for server and client(s) communication over HTTPS or WSS. 

It uses openSSL for the generation, so this package must be installed on the computer if not there already.
The script contains the following dummy data that is supplied to openSSL for credential population. 
This can be modified if so desired.

COUNTRY="DK"                # 2 letter country-code
STATE="Zealand"             # state or province name
LOCALITY="Helsingor"        # Locality Name (e.g. city)
ORGNAME="Example Inc"       # Organization Name (eg, company)
ORGUNIT="VISSv2-dev"        # Organizational Unit Name (eg. section)
CAEMAIL="ca@example.com"    # certificate's email address
SRVEMAIL="srv@example.com"  # certificate's email address
CLTEMAIL="clt@example.com"  # certificate's email address
# optional extra details
CHALLENGE=""                # challenge password
COMPANY=""                  # company name


The script must first be started for generation of the CA credentials, as they are needed for the server and client generation. 
$ ./testCredGen ca
The script, or rather openSSL, will ask for a password for the CA credentials. 
This will be asked for again in the generation of the server and client credentials, so make a note of it.

Before the generation of the server credentials, the SAN entry in the file server/server.ext likely needs an update. 
If your environment does not launch the VISSv2 server with localhost as the IP address of the computer, 
you need to declare the computer IP address being used, by updating the row:
IP.1 = 192.168.x.x
with the correct IP address. 
If localhost is defined, this row might need to be removed instead.

You can then generate the server credentials
$ ./testCredGen server
and then credentials for a client
$ ./testCredGen client

The CA credentials 
ca/Root.CA.crt
ca/Root.CA.key
must then be copied to the ./server/transport_sec/ca directory,
and the server credentials
server/server.crt
server/server.key
must be copied to the ./server/transport_sec/server directory.
To switch the server from using the unsecure HTTP and WS transport protocols to the secure versions HTTPS and WSS, 
the config parameter "transportSec" in the ./server/transport_sec/transportSec.json file must be changed from "no" to "yes". 

If the HTTPS or WSS transport manager crashes, you might find the following in the log:
listen tcp :443: bind: permission denied
To resolve that, this link may help.
https://superuser.com/questions/710253/allow-non-root-process-to-bind-to-port-80-and-443/892391#892391

A client for testing this can be found in the https://github.com/GENIVI/ccs-w3c-client repo, where the ovds/client/ccs-client.go
can be configured to use HTTPS/WSS towards the VISSv2 server. 
It is then necessary to copy the CA credentials (see above) and client credentials
client/client.ctr
client/client.key
to the corresponding ca and client directories in that repo, 
and change the "transportSec" parameter in the corresponding transportSec.json file to "yes".



