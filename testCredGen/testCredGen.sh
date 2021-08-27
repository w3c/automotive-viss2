#!/bin/bash

if [ "$#" -ne 1 ]
then
  echo "Usage: Must enter one of ca/server/client"
  exit 1
fi

ROLE=$1

BOLD=$(tput bold)
CLEAR=$(tput sgr0)

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

########################### CA ############################
if [[ "$ROLE" == "ca" ]]
then
echo -e "${BOLD}Generating RSA AES-256 Private Key for Root Certificate Authority${CLEAR}"
openssl genrsa -aes256 -out ca/Root.CA.key 4096
echo -e "${BOLD}Generating Certificate for Root Certificate Authority${CLEAR}"
cat <<EOF | openssl req -x509 -new -nodes -key ca/Root.CA.key -sha256 -days 1825 -out ca/Root.CA.crt
$COUNTRY
$STATE
$LOCALITY
$ORGNAME
$ORGUNIT
$site
$CAEMAIL
$CHALLENGE
$COMPANY
EOF
fi

########################### Server ############################
if [[ "$ROLE" == "server" ]]
then
echo -e "${BOLD}Generating RSA Private Key for Server Certificate${CLEAR}"
openssl genrsa -out server/server.key 4096
echo -e "${BOLD}Generating Certificate Signing Request for Server Certificate${CLEAR}"
cat <<EOF | openssl req -new -key server/server.key -out server/server.csr
$COUNTRY
$STATE
$LOCALITY
$ORGNAME
$ORGUNIT
$site
$SRVEMAIL
$CHALLENGE
$COMPANY
EOF
echo -e "${BOLD}Generating Certificate for Server Certificate${CLEAR}"
openssl x509 -req -in server/server.csr -CA ca/Root.CA.crt -CAkey ca/Root.CA.key -CAcreateserial -out server/server.crt -days 1825 -sha256 -extfile server/server.ext
fi
########################### Client ############################
if [[ "$ROLE" == "client" ]]
then
echo -e "${BOLD}Generating RSA Private Key for Client Certificate${CLEAR}"
openssl genrsa -out client/client.key 4096
echo -e "${BOLD}Generating Certificate Signing Request for Client Certificate${CLEAR}"
cat <<EOF | openssl req -new -key client/client.key -out client/client.csr
$COUNTRY
$STATE
$LOCALITY
$ORGNAME
$ORGUNIT
$site
$CLTEMAIL
$CHALLENGE
$COMPANY
EOF
echo -e "${BOLD}Generating Certificate for Client Certificate${CLEAR}"
openssl x509 -req -in client/client.csr -CA ca/Root.CA.crt -CAkey ca/Root.CA.key -CAcreateserial -out client/client.crt -days 1825 -sha256 -extfile client/client.ext
fi

echo "Done!"
