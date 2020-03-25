/**
* (C) 2020 Geotab Inc
*
* All files and artifacts in the repository at https://github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package main

import (
    "net/http"
    "encoding/json"
    "encoding/base64"
    "io/ioutil"

    "github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/utils"
)

const theAgtSecret = "averysecretkeyvalue1"  // shared with at-server


func makeAgtServerHandler(serverChannel chan string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		utils.Info.Printf("agtServer:url=%s", req.URL.Path)
		if req.URL.Path != "/agtserver" {
			http.Error(w, "404 url path not found.", 404)
		} else if req.Method != "POST" {
			http.Error(w, "400 bad request method.", 400)
		} else {
                        bodyBytes, err := ioutil.ReadAll(req.Body)
                        if err != nil {
                                http.Error(w, "400 request unreadable.", 400)
                        } else {
				utils.Info.Printf("agtServer:received POST request=%s\n", string(bodyBytes))
				serverChannel <- string(bodyBytes)
				response := <- serverChannel
				utils.Info.Printf("agtServer:POST response=%s", response)
                                if (len(response) == 0) {
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

func initAgtServer(serverChannel chan string, muxServer *http.ServeMux) {
	utils.Info.Printf("initAtServer(): :7500/agtserver")
	agtServerHandler := makeAgtServerHandler(serverChannel)
	muxServer.HandleFunc("/agtserver", agtServerHandler)
	utils.Error.Fatal(http.ListenAndServe(":7500", muxServer))
}

func generateAgt(input string) string {
	type Payload struct {
		UserId string
		Vin string
	}
	var payload Payload
	err := json.Unmarshal([]byte(input), &payload)
	if err != nil {
            utils.Error.Printf("generateAgt:error input=%s", input)
            return ""
	}
        jwtHeader := `{"alg":"HS256","typ":"JWT","iss":"oem.com/gen2/backend","sigurl":"oem.com/gen2/backend/pub","iat":1577847600,"exp":1593561599,"jti":"4667e93f-40f9-5f39-893e-cc0da890db3f"}`
        jwtPayload := `{"uid":"` + payload.UserId + `","aud":"` + payload.Vin + `"}`
	utils.Info.Printf("generateAgt:jwtHeader=%s", jwtHeader)
	utils.Info.Printf("generateAgt:jwtPayload=%s", jwtPayload)
        encodedJwtHeader := base64.RawURLEncoding.EncodeToString([]byte(jwtHeader))
        encodedJwtPayload := base64.RawURLEncoding.EncodeToString([]byte(jwtPayload))
	utils.Info.Printf("generateAgt:encodedJwtHeader=%s", encodedJwtHeader)
        jwtSignature := utils.GenerateHmac(encodedJwtHeader + "." + encodedJwtPayload, theAgtSecret)
        encodedJwtSignature := base64.RawURLEncoding.EncodeToString([]byte(jwtSignature))
        return encodedJwtHeader + "." + encodedJwtPayload + "." + encodedJwtSignature
}

func main() {

	utils.InitLog("agtserver-log.txt", "./logs")
	serverChan := make(chan string)
        muxServer := http.NewServeMux()

        go initAgtServer(serverChan, muxServer)

	for {
		select {
		case request := <-serverChan:
	utils.Info.Printf("main loop:request received")
			response:= generateAgt(request)
			utils.Info.Printf("agtServer response=%s", response)
                        serverChan <- `{"token":"` + response + `"}`
		}
	}
}

