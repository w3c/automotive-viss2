package main

import (
    "crypto/hmac"
    "crypto/sha256"
    "net/http"
    "encoding/json"
    "encoding/base64"
    "io/ioutil"
    "strings"

    "github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/utils"
)

const theSecretKey = "averysecretkeyvalue"


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
				response := <- serverChannel
				utils.Info.Printf("atServer:POST response=%s", response)
                                if (len(response) == 12) {
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
	utils.Info.Printf("initAtServer():"+utils.HostIP+":8600/atserver")
	atServerHandler := makeAtServerHandler(serverChannel)
	muxServer.HandleFunc("/atserver", atServerHandler)
	utils.Error.Fatal(http.ListenAndServe(utils.HostIP+":8600", muxServer))
}

func generateHmac(input string, key string) string {  //not a correct JWT signature?
    mac := hmac.New(sha256.New, []byte(key))
    mac.Write([]byte(input))
    return string(mac.Sum(nil))
}

func verifyTokenSignature(token string) bool {  // compatible with result from generateHmac()
        delimiter := strings.LastIndex(token, ".")
        message := token[:delimiter]
        messageMAC := token[delimiter+1:]
	mac := hmac.New(sha256.New, []byte(theSecretKey))
	mac.Write([]byte(message))
	expectedMAC := mac.Sum(nil)
        if (strings.Compare(messageMAC, base64.RawURLEncoding.EncodeToString(expectedMAC)) == 0) {
            return true
        }
        return false
}

func extractFromToken(token string, claim string) string {  // TODO remove white space sensitivity
    delimiter1 := strings.Index(token, ".")
    delimiter2 := strings.Index(token[delimiter1+1:], ".") + delimiter1 + 1
    header := token[:delimiter1]
    payload := token[delimiter1+1:delimiter2]
    decodedHeaderByte, _ := base64.RawURLEncoding.DecodeString(header)
    decodedHeader:= string(decodedHeaderByte)
    claimIndex := strings.Index(decodedHeader, claim)
    if (claimIndex != -1) {
        startIndex := claimIndex+len(claim)+2
        endIndex := strings.Index(decodedHeader[startIndex:], ",") + startIndex // ...claim":abc,...  or ...claim":"abc",... or See next line
        if (endIndex == startIndex-1) {  // ...claim":abc}  or ...claim":"abc"}
            endIndex = len(decodedHeader) - 1
        }
        if (string(decodedHeader[endIndex-1]) == `"`) {
            endIndex--
        } 
        if (string(decodedHeader[startIndex]) == `"`) {
            startIndex++
        }
        return decodedHeader[startIndex:endIndex]
    }
    decodedPayloadByte, _ := base64.RawURLEncoding.DecodeString(payload)
    decodedPayload := string(decodedPayloadByte)
    claimIndex = strings.Index(decodedPayload, claim)
    if (claimIndex != -1) {
        startIndex := claimIndex+len(claim)+2
        endIndex := strings.Index(decodedPayload[startIndex:], ",") + startIndex // ...claim":abc,...  or ...claim":"abc",... or See next line
        if (endIndex == startIndex-1) {  // ...claim":abc}  or ...claim":"abc"}
            endIndex = len(decodedPayload) - 1
        }
        if (string(decodedPayload[endIndex-1]) == `"`) {
            endIndex--
        } 
        if (string(decodedPayload[startIndex]) == `"`) {
            startIndex++
        }
        return decodedPayload[startIndex:endIndex]
    }
    return ""
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
        if (verifyTokenSignature(payload.Token) == false) {
            utils.Error.Printf("generateAt:invalid signature=%s", payload.Token)
            return ""
        }
        jwtHeader := `{"alg":"HS256","typ":"JWT"}`
        uid := extractFromToken(payload.Token, "uid")
        iss := extractFromToken(payload.Token, "aud")
//utils.Info.Printf("generateAt: uid=%s, iss=%s", uid, iss)
        jwtPayload := `{"exp":1609459199,"aud":"Gen2","scp":"` + payload.Scope + `","jti":"5967e93f-40f9-5f39-893e-cc0da890db2e","uid":"` + uid + `","iss":"` + iss  + `","sigurl":"w3.org/gen2/user/pub/` + uid  + `"}`
	utils.Info.Printf("generateAt:jwtHeader=%s", jwtHeader)
	utils.Info.Printf("generateAt:jwtPayload=%s", jwtPayload)
        encodedJwtHeader := base64.RawURLEncoding.EncodeToString([]byte(jwtHeader))
        encodedJwtPayload := base64.RawURLEncoding.EncodeToString([]byte(jwtPayload))
	utils.Info.Printf("generateAt:encodedJwtHeader=%s", encodedJwtHeader)
        jwtSignature := generateHmac(encodedJwtHeader + "." + encodedJwtPayload, theSecretKey)
        encodedJwtSignature := base64.RawURLEncoding.EncodeToString([]byte(jwtSignature))
        return encodedJwtHeader + "." + encodedJwtPayload + "." + encodedJwtSignature
}

func main() {

	serverChan := make(chan string)
        muxServer := http.NewServeMux()

//	utils.InitLog("atserver-log.txt", "./logs")
	utils.InitLog("atserver-log.txt")
	utils.HostIP = utils.GetOutboundIP()

        go initAtServer(serverChan, muxServer)

	for {
		select {
		case request := <-serverChan:
	utils.Info.Printf("main loop:request received")
			response:= generateAt(request)
			utils.Info.Printf("atServer response=%s", response)
                        serverChan <- `{"token":"` + response + `"}`
		}
	}
}

