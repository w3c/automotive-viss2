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
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/utils"
	"github.com/akamensky/argparse"
)

const theAgtSecret = "averysecretkeyvalue1" // shared with at-server

type Payload struct {
	Vin     string `json:"vin"`
	Context string `json:"context"`
	Proof   string `json:"proof"`
	Key     string `json:"key"`
}

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
				response := <-serverChannel
				utils.Info.Printf("agtServer:POST response=%s", response)
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

func initAgtServer(serverChannel chan string, muxServer *http.ServeMux) {
	utils.Info.Printf("initAtServer(): :7500/agtserver")
	agtServerHandler := makeAgtServerHandler(serverChannel)
	muxServer.HandleFunc("/agtserver", agtServerHandler)
	utils.Error.Fatal(http.ListenAndServe(":7500", muxServer))
}

func generateResponse(input string) string {
	var payload Payload
	err := json.Unmarshal([]byte(input), &payload)
	if err != nil {
		utils.Error.Printf("generateResponse:error input=%s", input)
		return `{"error": "Client request malformed"}`
	}
	if authenticateClient(payload) == true {
		return generateAgt(payload)
	}
	return `{"error": "Client authentication failed"}`
}

func checkUserRole(userRole string) bool {
	if userRole != "OEM" && userRole != "Dealer" && userRole != "Independent" && userRole != "Owner" && userRole != "Driver" && userRole != "Passenger" {
		return false
	}
	return true
}

func checkAppRole(appRole string) bool {
	if appRole != "OEM" && appRole != "Third party" {
		return false
	}
	return true
}

func checkDeviceRole(deviceRole string) bool {
	if deviceRole != "Vehicle" && deviceRole != "Nomadic" && deviceRole != "Cloud" {
		return false
	}
	return true
}

func checkRoles(context string) bool {
	if strings.Count(context, "+") != 2 {
		return false
	}
	delimiter1 := strings.Index(context, "+")
	delimiter2 := strings.Index(context[delimiter1+1:], "+")
	if checkUserRole(context[:delimiter1]) == false || checkAppRole(context[delimiter1+1:delimiter1+1+delimiter2]) == false || checkDeviceRole(context[delimiter1+1+delimiter2+1:]) == false {
		return false
	}
	return true

}

func authenticateClient(payload Payload) bool {
	if checkRoles(payload.Context) == true && payload.Proof == "ABC" { // a bit too simple validation...
		return true
	}
	return false
}

func generateAgt(payload Payload) string {
	uuid, err := exec.Command("uuidgen").Output()
	if err != nil {
		utils.Error.Printf("generateAgt:Error generating uuid, err=%s", err)
		return `{"error": "Internal error"}`
	}
	uuid = uuid[:len(uuid)-1] // remove '\n' char
	iat := int(time.Now().Unix())
	exp := iat + 4*60*60 // 4 hours
	if len(payload.Key) != 0 {
		exp = iat + 7*24*60*60 // 1 week
	}
	jwtHeader := `{"alg":"ES256","typ":"JWT"}`
	jwtPayload := `{"vin":"` + payload.Vin + `", "iat":` + strconv.Itoa(iat) + `, "exp":` + strconv.Itoa(exp) + `, "clx":"` + payload.Context + `"`
	if len(payload.Key) != 0 {
		jwtPayload += `, "pub": "` + payload.Key + `"`
	}
	jwtPayload += `, "aud": "w3.org/gen2", "jti":"` + string(uuid) + `"}`
	utils.Info.Printf("generateAgt:jwtHeader=%s", jwtHeader)
	utils.Info.Printf("generateAgt:jwtPayload=%s", jwtPayload)
	encodedJwtHeader := base64.RawURLEncoding.EncodeToString([]byte(jwtHeader))
	encodedJwtPayload := base64.RawURLEncoding.EncodeToString([]byte(jwtPayload))
	utils.Info.Printf("generateAgt:encodedJwtHeader=%s", encodedJwtHeader)
	jwtSignature := utils.GenerateHmac(encodedJwtHeader+"."+encodedJwtPayload, theAgtSecret)
	encodedJwtSignature := base64.RawURLEncoding.EncodeToString([]byte(jwtSignature))
	return `{"token":"` + encodedJwtHeader + "." + encodedJwtPayload + "." + encodedJwtSignature + `"}`
}

func main() {
	// Create new parser object
	parser := argparse.NewParser("print", "AGT Server")
	// Create string flag
	logFile := parser.Flag("", "logfile", &argparse.Options{Required: false, Help: "outputs to logfile in ./logs folder"})
	logLevel := parser.Selector("", "loglevel", []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}, &argparse.Options{
		Required: false,
		Help:     "changes log output level",
		Default:  "info"})

	// Parse input
	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
	}

	utils.InitLog("agtserver-log.txt", "./logs", *logFile, *logLevel)
	serverChan := make(chan string)
	muxServer := http.NewServeMux()

	go initAgtServer(serverChan, muxServer)

	for {
		select {
		case request := <-serverChan:
			response := generateResponse(request)
			utils.Info.Printf("agtServer response=%s", response)
			serverChan <- response
		}
	}
}
