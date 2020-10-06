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
    "time"
    "os"
    "os/exec"
    "strconv"
    "strings"

    "github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/utils"
)

const theAgtSecret = "averysecretkeyvalue1" //shared with agt-server
const theAtSecret = "averysecretkeyvalue2"  //not shared

type Payload struct {
    Token string    `json:"token"`
    Purpose string  `json:"purpose"`
    Pop string      `json:"pop"`
}

type AgToken struct {
    Vin string       `json:"vin"`
    Iat int          `json:"iat"`
    Exp int          `json:"exp"`
    Context string   `json:"clx"`
    Key string       `json:"pub"`
    Audience string  `json:"aud"`
    JwtId string     `json:"jti"`
}

var purposeList map[string]interface{}

var pList []PurposeElement

type PurposeElement struct {
    Short string
    Long string
    Context []ContextElement
    Access []AccessElement
}

type ContextElement struct {
    Actor [3]RoleElement   // User, App, Device
}

type RoleElement struct {
    Role []string
}

type AccessElement struct {
    Path string
    Mode string
}


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

func initAtServer(serverChannel chan string, muxServer *http.ServeMux) {
	utils.Info.Printf("initAtServer(): :8600/atserver")
	atServerHandler := makeAtServerHandler(serverChannel)
	muxServer.HandleFunc("/atserver", atServerHandler)
	utils.Error.Fatal(http.ListenAndServe(":8600", muxServer))
}

func generateResponse(input string) string {
	var payload Payload
	err := json.Unmarshal([]byte(input), &payload)
	if err != nil {
            utils.Error.Printf("generateResponse:error input=%s", input)
            return `{"error": "Client request malformed"}`
	}
        agToken, errResp := extractTokenPayload(payload.Token)
        if (len(errResp) > 0) {
            return errResp
        }
	ok, errResponse := validateRequest(payload, agToken)
	if (ok == true) {
	    return generateAt(payload, agToken.Context)
	}
	return errResponse
}

func validateTokenTimestamps(iat int, exp int) bool {
        now := time.Now()
        if (now.Before(time.Unix(int64(iat), 0)) == true) {
            return false
        }
        if (now.After(time.Unix(int64(exp), 0)) == true) {
            return false
        }
        return true
}

func validatePurpose(purpose string, context string) bool { // TODO: learn how to code to parse the purpose list, then use it to validate the purpose
    valid := false
    for i := 0 ; i < len(pList) ; i++ {
utils.Info.Printf("validatePurpose:purposeList[%d].Short=%s", i, pList[i].Short)
        if (pList[i].Short == purpose) {
utils.Info.Printf("validatePurpose:purpose match=%s", pList[i].Short)
            valid = checkAuthorization(i, context)
        }
    }
    return valid
}

func checkAuthorization(index int, context string) bool {
    for i := 0 ; i < len(pList[index].Context) ; i++ {
        actorValid := [3]bool{false, false, false}
        for j := 0 ; j < len(pList[index].Context[i].Actor) ; j++ {
            if (j > 2) {
                return false  // only three subactors supported
            }
            for k := 0 ; k < len(pList[index].Context[i].Actor[j].Role) ; k++ {
                if (getActorRole(j, context) == pList[index].Context[i].Actor[j].Role[k]) {
                    actorValid[j] = true
                }
            }
        }
        if (actorValid[0] == true && actorValid[1] == true && actorValid[2] == true) {
            return true
        }
    }
    return false
}

func getActorRole(actorIndex int, context string) string {
    delimiter1 := strings.Index(context, "+")
    if (actorIndex == 0) {
        return context[:delimiter1]
    }
    delimiter2 := strings.Index(context[delimiter1+1:], "+")
    if (actorIndex == 1) {
        return context[delimiter1+1:delimiter1+1+delimiter2]
    }
    return context[delimiter1+1+delimiter2+1:]
}

func decodeTokenPayload(token string) string {
    delim1 := strings.Index(token, ".")
    delim2 := delim1 + 1 +strings.Index(token[delim1+1:], ".")
    pload := token[delim1+1:delim1+1+delim2]
    payload, _ := base64.RawURLEncoding.DecodeString(pload)
utils.Info.Printf("decodeTokenPayload:payload=%s", string(payload))
    return string(payload)
}

func extractTokenPayload(token string) (AgToken, string) {
	var agToken AgToken
	tokenPayload := decodeTokenPayload(token)
	err := json.Unmarshal([]byte(tokenPayload), &agToken)
	if err != nil {
            utils.Error.Printf("extractTokenPayload:token payload=%s, error=%s", tokenPayload, err)
            return agToken, `{"error": "AG token malformed"}`
	}
	return agToken, ""
}

func checkVin(vin string) bool {
    return true    // should be checked with VIN in tree
}

func validateRequest(payload Payload, agToken AgToken) (bool, string) {
        if (checkVin(agToken.Vin) == false) {
            utils.Info.Printf("validateRequest:incorrect VIN=%s", agToken.Vin)
	    return false, `{"error": "Incorrect vehicle identifiction"}`
        }
        if (utils.VerifyTokenSignature(payload.Token, theAgtSecret) == false) {
            utils.Info.Printf("validateRequest:invalid signature=%s", payload.Token)
	    return false, `{"error": "AG token signature validation failed"}`
        }
        if (validateTokenTimestamps(agToken.Iat, agToken.Exp) == false) {
            utils.Info.Printf("validateRequest:invalid token timestamps, iat=%d, exp=%d", agToken.Iat, agToken.Exp)
	    return false, `{"error": "AG token timestamp validation failed"}`
        }
        if (len(agToken.Key) != 0 && payload.Pop != "GHI") {  // PoP should be a signed timestamp
            utils.Info.Printf("validateRequest:Proof of possession of key pair failed")
	    return false, `{"error": "Proof of possession of key pair failed"}`
        }
        if (validatePurpose(payload.Purpose, agToken.Context) == false) {
            utils.Info.Printf("validateRequest:invalid purpose=%s, context=%s", payload, agToken.Context)
	    return false, `{"error": "Purpose validation failed"}`
        }
        return true, ""
}

func generateAt(payload Payload, context string) string{
	uuid, err := exec.Command("uuidgen").Output()
        if err != nil {
            utils.Error.Printf("generateAt:Error generating uuid, err=%s", err)
            return `{"error": "Internal error"}`
        }
        uuid = uuid[:len(uuid)-1]  // remove '\n' char
        iat := int(time.Now().Unix())
        exp := iat + 1*60*60  // 1 hour
        jwtHeader := `{"alg":"ES256","typ":"JWT"}`
        jwtPayload := `{"iat":` + strconv.Itoa(iat) + `,"exp":` + strconv.Itoa(exp) + `"scp":"` + payload.Purpose + `"` + `"clx":"` + context + 
        `", "aud": "w3.org/gen2", "jti":"` + string(uuid) + `"}`
	utils.Info.Printf("generateAt:jwtHeader=%s", jwtHeader)
	utils.Info.Printf("generateAt:jwtPayload=%s", jwtPayload)
        encodedJwtHeader := base64.RawURLEncoding.EncodeToString([]byte(jwtHeader))
        encodedJwtPayload := base64.RawURLEncoding.EncodeToString([]byte(jwtPayload))
	utils.Info.Printf("generateAt:encodedJwtHeader=%s", encodedJwtHeader)
        jwtSignature := utils.GenerateHmac(encodedJwtHeader + "." + encodedJwtPayload, theAtSecret)
        encodedJwtSignature := base64.RawURLEncoding.EncodeToString([]byte(jwtSignature))
        return encodedJwtHeader + "." + encodedJwtPayload + "." + encodedJwtSignature
}

func initPurposelist() {
	data, err := ioutil.ReadFile("purposelist.json")
	if err != nil {
		utils.Error.Printf("Error reading purposelist.json\n")
		os.Exit(-1)
	}
	err = json.Unmarshal([]byte(data), &purposeList)
	if err != nil {
            utils.Error.Printf("initPurposelist:error data=%s, err=%s", data, err)
	    os.Exit(-1)
	}
	extractPurposeElementsLevel1(purposeList)
}

func extractPurposeElementsLevel1(purposeList map[string]interface{}) {
    for k, v := range purposeList {
        switch vv := v.(type) {
          case []interface{}:
            utils.Info.Println(k, "is an array:")
  	    extractPurposeElementsLevel2(vv)
          case map[string]interface{}:
            utils.Info.Println(k, "is a map:")
            extractPurposeElementsLevel3(vv)
          default:
            utils.Info.Println(k, "is of an unknown type")
        }
    }
}

func extractPurposeElementsLevel2(purposeListElem []interface{}) {
    for k, v := range purposeListElem {
        switch vv := v.(type) {
          case map[string]interface{}:
            utils.Info.Println(k, "is a map:")
            extractPurposeElementsLevel3(vv)
          default:
            utils.Info.Println(k, "is of an unknown type")
        }
    }
}

func extractPurposeElementsLevel3(purposeElem map[string]interface{}) {
    i := 0
    pList = make([]PurposeElement, len(purposeElem))
    for k, v := range purposeElem {
        switch vv := v.(type) {
          case string:
            utils.Info.Println(k, "is string", vv)
            if (k == "short") {
                pList[i].Short = vv
            } else {
                pList[i].Long = vv
            }
          case []interface{}:
            utils.Info.Println(k, "is an array:, len=",strconv.Itoa(len(vv)))
            if (k == "contexts") {
                pList[i].Context = make([]ContextElement, len(vv))
                extractPurposeElementsL4ContextL1(i, vv)
            } else {
                pList[i].Access = make([]AccessElement, len(vv))
                extractPurposeElementsL4SignalAccessL1(i, vv)
            }
          case map[string]interface{}:
            utils.Info.Println(k, "is a map:")
            if (k == "contexts") {
                pList[i].Context = make([]ContextElement, 1)
                extractPurposeElementsL4ContextL2(0, i, vv)
            } else {
                pList[i].Access = make([]AccessElement, 1)
                extractPurposeElementsL4SignalAccessL2(0, i, vv)
            }
          default:
            utils.Info.Println(k, "is of an unknown type")
        }
        i++
    }
}

func extractPurposeElementsL4ContextL1(index int, contextElem []interface{}) {
    for k, v := range contextElem {
        switch vv := v.(type) {
          case map[string]interface{}:
            utils.Info.Println(k, "is a map:")
            extractPurposeElementsL4ContextL2(k, index, vv)
          default:
            utils.Info.Println(k, "is of an unknown type")
        }
    }
}

func extractPurposeElementsL4ContextL2(k int, index int, contextElem map[string]interface{}) {
    for i, u := range contextElem {
        utils.Info.Println(i, u)
        switch vvv := u.(type) {
          case string:
            if (i == "user") {
                pList[index].Context[k].Actor[0].Role = make([]string, 1)
                pList[index].Context[k].Actor[0].Role[0] = u.(string)
            } else if (i == "app") {
                pList[index].Context[k].Actor[1].Role = make([]string, 1)
                pList[index].Context[k].Actor[1].Role[0] = u.(string)
            } else {
                pList[index].Context[k].Actor[2].Role = make([]string, 1)
                pList[index].Context[k].Actor[2].Role[0] = u.(string)
            }
          case []interface{}:
            m := 0
            for l, t := range vvv {
              utils.Info.Println(l, t)
              switch t.(type) {
                case string:
                  if (i == "user") {
                      if (m == 0) {
                          pList[index].Context[k].Actor[0].Role = make([]string, len(vvv))
                      }
                      pList[index].Context[k].Actor[0].Role[m] = t.(string)
                  } else if (i == "app") {
                      if (m == 0) {
                          pList[index].Context[k].Actor[1].Role = make([]string, len(vvv))
                      }
                      pList[index].Context[k].Actor[1].Role[m] = t.(string)
                  } else {
                      if (m == 0) {
                          pList[index].Context[k].Actor[2].Role = make([]string, len(vvv))
                      }
                      pList[index].Context[k].Actor[2].Role[m] = t.(string)
                  }
                default:
                  utils.Info.Println(k, "is of an unknown type")
              }
              m++
            }
          default:
            utils.Info.Println(k, "is of an unknown type")
        }
    }
}

func extractPurposeElementsL4SignalAccessL1(index int, accessElem []interface{}) {
    for k, v := range accessElem {
        switch vv := v.(type) {
          case map[string]interface{}:
            utils.Info.Println(k, "is a map:")
            extractPurposeElementsL4SignalAccessL2(k, index, vv)
          default:
            utils.Info.Println(k, "is of an unknown type")
        }
    }
}

func extractPurposeElementsL4SignalAccessL2(k int, index int, accessElem map[string]interface{}) {
    for i, u := range accessElem {
        utils.Info.Println(i, u)
        if (i == "path") {
            pList[index].Access[k].Path = u.(string)
        } else {
            pList[index].Access[k].Mode = u.(string)
        }
    }
}

func main() {

	serverChan := make(chan string)
        muxServer := http.NewServeMux()

	utils.InitLog("atserver-log.txt", "./logs")
	initPurposelist()

        go initAtServer(serverChan, muxServer)

	for {
		select {
		case request := <-serverChan:
 		        response := generateResponse(request)
			 utils.Info.Printf("atServer response=%s", response)
                        serverChan <- response
		}
	}
}

