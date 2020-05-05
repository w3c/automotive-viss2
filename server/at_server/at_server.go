/**
* (C) 2020 Geotab Inc
*
* All files and artifacts in the repository at https://github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package main

import (
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/basanjeev/W3C_VehicleSignalInterfaceImpl/utils"
)

const theAgtSecret = "averysecretkeyvalue1" //shared with agt-server
const theAtSecret = "averysecretkeyvalue2"  //not shared

func makeAtServerHandler(serverChannel chan string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		utils.Info.Printf("atServer:url=%s", req.URL.Path)
		if req.URL.Path != "/atserver" {
			http.Error(w, "404 url path not found.", 404)
		} else if req.Method != "POST" {
			http.Error(w, "400 bad request method.", 400)
		} else {
			bodyBytes, err := ioutil.ReadAll(req.Body)
			if err != nil {
				http.Error(w, "400 request unreadable.", 400)
			} else {
				utils.Info.Printf("atServer:received POST request=%s\n", string(bodyBytes))
				serverChannel <- string(bodyBytes)
				response := <-serverChannel
				utils.Info.Printf("atServer:POST response=%s", response)
				if len(response) == 0 {
					http.Error(w, "400 bad input.", 400)
				} else {
					w.Header().Set("Access-Control-Allow-Origin", "*")
					//				    w.Header().Set("Content-Type", "application/json")
					w.Write([]byte(response))
				}
			}
		}
	}
}

func initAtServer(serverChannel chan string, muxServer *http.ServeMux) {
	utils.Info.Printf("initAtServer(): :8600/atserver")
	atServerHandler := makeAtServerHandler(serverChannel)
	muxServer.HandleFunc("/atserver", atServerHandler)
	utils.Error.Fatal(http.ListenAndServe(":8600", muxServer))
}

func generateAt(input string) string { // TODO validate AGT header fields (iat, exp,..), create dynamic AT payload fields (exp, jti)
	type Payload struct {
		Scope string
		Token string
	}
	var payload Payload
	err := json.Unmarshal([]byte(input), &payload)
	if err != nil {
		utils.Error.Printf("generateAt:error input=%s", input)
		return ""
	}
	if utils.VerifyTokenSignature(payload.Token, theAgtSecret) == false {
		utils.Error.Printf("generateAt:invalid signature=%s", payload.Token)
		return ""
	}
	jwtHeader := `{"alg":"HS256","typ":"JWT"}`
	uid := utils.ExtractFromToken(payload.Token, "uid")
	iss := utils.ExtractFromToken(payload.Token, "aud")
	//utils.Info.Printf("generateAt: uid=%s, iss=%s", uid, iss)
	jwtPayload := `{"exp":1609459199,"aud":"Gen2","scp":"` + payload.Scope + `","jti":"5967e93f-40f9-5f39-893e-cc0da890db2e","uid":"` + uid + `","iss":"` + iss + `","sigurl":"w3.org/gen2/user/pub/` + uid + `"}`
	utils.Info.Printf("generateAt:jwtHeader=%s", jwtHeader)
	utils.Info.Printf("generateAt:jwtPayload=%s", jwtPayload)
	encodedJwtHeader := base64.RawURLEncoding.EncodeToString([]byte(jwtHeader))
	encodedJwtPayload := base64.RawURLEncoding.EncodeToString([]byte(jwtPayload))
	utils.Info.Printf("generateAt:encodedJwtHeader=%s", encodedJwtHeader)
	jwtSignature := utils.GenerateHmac(encodedJwtHeader+"."+encodedJwtPayload, theAtSecret)
	encodedJwtSignature := base64.RawURLEncoding.EncodeToString([]byte(jwtSignature))
	return encodedJwtHeader + "." + encodedJwtPayload + "." + encodedJwtSignature
}

func main() {

	serverChan := make(chan string)
	muxServer := http.NewServeMux()

	utils.InitLog("atserver-log.txt", "./logs")

	go initAtServer(serverChan, muxServer)

	for {
		select {
		case request := <-serverChan:
			var response string
			utils.Info.Printf("main loop:received request=%s", request)
			if strings.Contains(request, "scope") == true {
				response = generateAt(request)
				if len(response) > 0 {
					response = `{"token":"` + response + `"}`
				}
			} else {
				response = `{"signature":"false"}`
				type Payload struct {
					Token string
				}
				var payload Payload
				err := json.Unmarshal([]byte(request), &payload)
				if err == nil {
					if utils.VerifyTokenSignature(payload.Token, theAtSecret) == true {
						response = `{"signature":"true"}`
					}
				}
			}
			utils.Info.Printf("atServer response=%s", response)
			serverChan <- response
		}
	}
}
