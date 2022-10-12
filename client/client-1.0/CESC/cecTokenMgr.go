/**
* (C) 2022 Geotab
*
* All files and artifacts in the repository at https://github.com/w3c/automotive-viss2
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package main

import (
	"encoding/json"
//	"io/ioutil"
//	"bytes"
//	"os"
	"os/exec"

//	"crypto/tls"
//	"crypto/x509"
//	"net/http"

//	"strconv"
//	"strings"
	"time"

	"github.com/w3c/automotive-viss2/utils"
)

//var digClientCert tls.Certificate // not used
//var digCaCert []byte  // not used?
//const digCaCertFileName = "transport_sec/Amazon_Root_CA_1.crt"

type TokenData struct {
	Token string `json:"TokenString"`
	Expires string `json:"Expires"`
	Created string `json:"Created"`
}
var digData TokenData
var refreshDigData TokenData

type UserCred struct {
	Email string
	Password string
}

func initTokenMgr(tokenChan chan string) {  // keeps the DIG token fresh, provides new over tokenChan
	reinitChan := make(chan bool)
	for {
		go runTokenMgr(tokenChan, reinitChan)
		select {
			case <- reinitChan:
				continue
		}
	}
}

func runTokenMgr(tokenChan chan string, reinitChan chan bool) {
	var userCred UserCred
	var reRun bool

	userCred.Email = "mail address for client account"  // obtained from cloud provider registration
	userCred.Password = "password for client account"   // obtained from cloud provider registration
	digData, refreshDigData = getNewTokens(userCred)
utils.Info.Printf("DIG bearertoken = %s", digData.Token)
	tokenChan <- digData.Token	
	for {
		sleepUntilRefresh(digData.Expires)
		digData, refreshDigData, reRun = getRefreshedTokens(digData.Token, refreshDigData.Token)
		if reRun {
			reinitChan <- true
			break
		}
		tokenChan <- digData.Token
		utils.Info.Printf("DIG token refreshed")
	}
}

func getNewTokens(userCred UserCred) (TokenData, TokenData) {  // aquire tokens from DIG authentication API
	payload := `{"username":"` + userCred.Email + `","password":"` + userCred.Password + `"}`

	url := "https://dig.geotab.com:443/authentication/authenticate"
	curl := exec.Command("curl", "--header", "content-type:application/json", "--data", payload, url)
	out, err := curl.Output()
	    if err != nil {
	    utils.Error.Printf("curl error=%s", err)
	    utils.Error.Printf("getNewTokens:Payload related to the error = %s ", payload)
	    return digData, refreshDigData
	}
	return extractResponseData(string(out))


/*	req, err := http.NewRequest("POST", url, bytes.NewBuffer([]byte(payload)))
	if err != nil {
		utils.Error.Printf("getNewTokens: Error creating request=%s.", err)
		return digData, refreshDigData
	}

	// Set headers
	req.Header.Set("Access-Control-Allow-Origin", "*")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", "dig.geotab.com:443")

	digCaCert, err := ioutil.ReadFile(digCaCertFileName)
	if err != nil {
		utils.Error.Printf("getNewTokens: Error reading CA cert file= %s ", err)
		return digData, refreshDigData
	}

	digCaCertPool := x509.NewCertPool()
	digCaCertPool.AppendCertsFromPEM(digCaCert)

	t := &http.Transport{
		TLSClientConfig: &tls.Config{
			Certificates: []tls.Certificate{digClientCert},
			RootCAs:      digCaCertPool,
		},
	}

	client := &http.Client{Transport: t, Timeout: 15 * time.Second}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		utils.Error.Printf("getNewTokens: Error in issuing request= %s ", err)
		return digData, refreshDigData
	}
	defer resp.Body.Close()

	switch resp.StatusCode {
		case 202:
			body, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				utils.Error.Printf("getNewTokens: Error in reading response= %s ", err)
				return digData, refreshDigData
			}
			utils.Info.Printf("getNewTokens: Response= %s ", string(body))
			return extractResponseData(string(body))
		case 400:
			utils.Error.Printf("Bad request error = %s ", err)
		case 401:
			utils.Error.Printf("Payload too large error = %s ", err)
		case 413:
			utils.Error.Printf("Unauthorized error = %s ", err)
		default:
			utils.Error.Printf("Unknown error = %s ", err)
	}
	utils.Error.Printf("Payload related to the error = %s ", payload)
	return digData, refreshDigData*/

}

func extractResponseData(response string) (TokenData, TokenData) {
//utils.Info.Printf("DIG token response = %s", response)
    var responseMap map[string]interface{}
    err := json.Unmarshal([]byte(response), &responseMap)
    if err != nil {
	utils.Error.Printf("extractResponseData:error unmarshal response=%s", response)
	return digData, refreshDigData
    }
    switch vv := responseMap["Data"].(type) {
      case map[string]interface{}:
//        utils.Info.Println(jsonList, "is a map:")
  	return extractTokensData(vv)
      default:
        utils.Info.Println(vv, "is of an unknown type")
    }
    return digData, refreshDigData
}

func extractTokensData(dataMap map[string]interface{}) (TokenData, TokenData) {
	for k, v := range dataMap {
		if k == "BearerToken" {
			switch vv := v.(type) {
				case map[string]interface{}:
					digData.Token, digData.Expires, digData.Created = extractTokenData(vv)
				default:
					utils.Info.Println(vv, "is of an unknown type")
			}
		}
		if k == "RefreshToken" {
			switch vv := v.(type) {
				case map[string]interface{}:
					refreshDigData.Token, refreshDigData.Expires, refreshDigData.Created = extractTokenData(vv)
				default:
					utils.Info.Println(vv, "is of an unknown type")
			}
		}
	}
	return digData, refreshDigData
}

func extractTokenData(tokenMap map[string]interface{}) (string, string, string) {
	var token string
	var expires string
	var created string
	for k, v := range tokenMap {
		if k == "TokenString" {
			token = v.(string)
		}
		if k == "Expires" {
			expires = v.(string)
		}
		if k == "Created" {
			created = v.(string)
		}
	}
	return token, expires, created
}

func getRefreshedTokens(digToken string, refreshToken string) (TokenData, TokenData, bool) {  // aquire tokens from DIG refresh token API
	payload := `{"BearerToken":"` + digData.Token + `","RefreshToken":"` + refreshDigData.Token + `"}`

	url := "https://dig.geotab.com:443/authentication/refresh-token"
	curl := exec.Command("curl", "--header", "content-type:application/json", "--data", payload, url)
	out, err := curl.Output()
	    if err != nil {
	    utils.Error.Printf("curl error=%s", err)
	    utils.Error.Printf("getRefreshedTokens:Payload related to the error = %s ", payload)
	    return digData, refreshDigData, true  // rerun
	}
	bToken, rToken := extractResponseData(string(out))
	return bToken, rToken, false  //no rerun
}

func sleepUntilRefresh(expiryTime string) { // sleep until 5 mins before expiry
        expTime, err := time.Parse(time.RFC3339, expiryTime)
        if (err != nil) {
	    utils.Error.Printf("expiry time parsing error=%s", err)
	    time.Sleep(30 * time.Minute)  // how long to retry?
        }
        expTime = expTime.Add(-5*time.Minute)
        timeToSleep := expTime.Sub(time.Now().UTC())
	time.Sleep(timeToSleep)
}

