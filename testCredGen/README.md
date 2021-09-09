The testCredGen script can be used to generate credentials for server and client(s) communication over HTTPS and WSS. 

It uses openSSL for the generation, so this package must be installed on the computer if not there already.
The script contains the following dummy data that is supplied to openSSL for credential population. 
This can be modified if so desired.

COUNTRY="DK"                # 2 letter country-code<br>
STATE="Zealand"             # state or province name<br>
LOCALITY="Helsingor"        # Locality Name (e.g. city)<br>
ORGNAME="Example Inc"       # Organization Name (eg, company)<br>
ORGUNIT="VISSv2-dev"        # Organizational Unit Name (eg. section)<br>
CAEMAIL="ca@example.com"    # certificate's email address<br>
SRVEMAIL="srv@example.com"  # certificate's email address<br>
CLTEMAIL="clt@example.com"  # certificate's email address<br>

optional extra details<br>
CHALLENGE=""                # challenge password<br>
COMPANY=""                  # company name<br>


The script must first be started for generation of the CA credentials, as they are needed for the server and client credential generation. 
$ ./testCredGen ca
The script, or rather openSSL, will ask for a password for the CA credentials. 
This will be asked for again in the generation of the server and client credentials, so make a note of it.

Before the generation of the server credentials, the SAN entry in the file server/server.ext likely needs an update. 
If your environment does not launch the VISSv2 server with localhost as the computer IP address, 
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
and the server credentials:
server/server.crt
server/server.key
must be copied to the ./server/transport_sec/server directory.
To switch the server from using the unsecure HTTP and WS transport protocols to the secure versions HTTPS and WSS, 
the config parameter "transportSec" in the ./server/transport_sec/transportSec.json file must be changed from "no" to "yes". 

When "transportSec" is set to "yes", for testing the "serverCertOpt" key can be set to either of the values:<br>
"NoClientCert"// server does not require client to provide a certificate<br>
"ClientCertNoVerification"// server requires client to have a certificate, but it is not verified<br>
"ClientCertVerification" //server requires client to have a certificate that verifies successfully<br>
The access control model in the VISSv2 standard requires "ClientCertVerification" to be set.

If the HTTPS or WSS transport manager crashes, you might find the following in the log:
listen tcp :443: bind: permission denied
To resolve that, this link may help.
https://superuser.com/questions/710253/allow-non-root-process-to-bind-to-port-80-and-443/892391#892391

A client for testing this can be found in the https://github.com/GENIVI/ccs-w3c-client repo, where the ovds/client/ccs-client.go
can be configured to use HTTPS/WSS towards the VISSv2 server. 
It is then necessary to copy the CA cert (see above) and client cert and key:
client/client.ctr
client/client.key
to the corresponding ca and client directories in that repo, 
and change the "transportSec" parameter in the corresponding transportSec.json file to "yes".



