package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net"
	"os"
	"strings"
//	"strconv"
        "time"
)

const IpModel = 0 // IpModel = [0,1,2] = [localhost,extIP,envVarIP]
const IpEnvVarName = "GEN2MODULEIP"

func GetServerIP() string {
	if value, ok := os.LookupEnv(IpEnvVarName); ok {
		Info.Println("ServerIP:", value)
		return value
	}
	Error.Printf("Environment variable %s is not set defaulting to localhost.", IpEnvVarName)
	return "localhost" //fallback
}

func GetModelIP(ipModel int) string {
	if ipModel == 0 {
		return "localhost"
	}
	if ipModel == 2 {
		if value, ok := os.LookupEnv(IpEnvVarName); ok {
			Info.Println("Host IP:", value)
			return value
		}
		Error.Printf("Environment variable %s error.", IpEnvVarName)
		return "localhost" //fallback
	}
	conn, err := net.Dial("udp", "8.8.8.8:80")
	if err != nil {
		Error.Fatal(err.Error())
	}
	defer conn.Close()

	localAddr := conn.LocalAddr().(*net.UDPAddr)
	Info.Println("Host IP:", localAddr.IP)

	return localAddr.IP.String()
}

func ExtractPayload(request string, rMap *map[string]interface{}) {
	decoder := json.NewDecoder(strings.NewReader(request))
	err := decoder.Decode(rMap)
	if err != nil {
		Error.Printf("extractPayload: JSON decode failed for request:%s\n", request)
	}
}

func UrlToPath(url string) string {
	var path string = strings.TrimPrefix(strings.Replace(url, "/", ".", -1), ".")
	return path[:]
}

func PathToUrl(path string) string {
	var url string = strings.Replace(path, ".", "/", -1)
	return "/" + url
}

func GenerateHmac(input string, key string) string { //not a correct JWT signature?
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(input))
	return string(mac.Sum(nil))
}

func VerifyTokenSignature(token string, key string) bool { // compatible with result from generateHmac()
	delimiter := strings.LastIndex(token, ".")
	if delimiter == -1 {
		return false
	}
	message := token[:delimiter]
	messageMAC := token[delimiter+1:]
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(message))
	expectedMAC := mac.Sum(nil)
	if strings.Compare(messageMAC, base64.RawURLEncoding.EncodeToString(expectedMAC)) == 0 {
		return true
	}
	return false
}

func ExtractFromToken(token string, claim string) string { // TODO remove white space sensitivity
	delimiter1 := strings.Index(token, ".")
	delimiter2 := strings.Index(token[delimiter1+1:], ".") + delimiter1 + 1
	header := token[:delimiter1]
	payload := token[delimiter1+1 : delimiter2]
	decodedHeaderByte, _ := base64.RawURLEncoding.DecodeString(header)
	decodedHeader := string(decodedHeaderByte)
	claimIndex := strings.Index(decodedHeader, claim)
	if claimIndex != -1 {
		startIndex := claimIndex + len(claim) + 2
		endIndex := strings.Index(decodedHeader[startIndex:], ",") + startIndex // ...claim":abc,...  or ...claim":"abc",... or See next line
		if endIndex == startIndex-1 {                                           // ...claim":abc}  or ...claim":"abc"}
			endIndex = len(decodedHeader) - 1
		}
		if string(decodedHeader[endIndex-1]) == `"` {
			endIndex--
		}
		if string(decodedHeader[startIndex]) == `"` {
			startIndex++
		}
		return decodedHeader[startIndex:endIndex]
	}
	decodedPayloadByte, _ := base64.RawURLEncoding.DecodeString(payload)
	decodedPayload := string(decodedPayloadByte)
	claimIndex = strings.Index(decodedPayload, claim)
	if claimIndex != -1 {
		startIndex := claimIndex + len(claim) + 2
		endIndex := strings.Index(decodedPayload[startIndex:], ",") + startIndex // ...claim":abc,...  or ...claim":"abc",... or See next line
		if endIndex == startIndex-1 {                                            // ...claim":abc}  or ...claim":"abc"}
			endIndex = len(decodedPayload) - 1
		}
		if string(decodedPayload[endIndex-1]) == `"` {
			endIndex--
		}
		if string(decodedPayload[startIndex]) == `"` {
			startIndex++
		}
		return decodedPayload[startIndex:endIndex]
	}
	return ""
}

func SetErrorResponse(reqMap map[string]interface{}, errRespMap map[string]interface{}, number string, reason string, message string) {
	if reqMap["MgrId"] != nil {
		errRespMap["MgrId"] = reqMap["MgrId"]
	}
	if reqMap["ClientId"] != nil {
		errRespMap["ClientId"] = reqMap["ClientId"]
	}
	if reqMap["action"] != nil {
		errRespMap["action"] = reqMap["action"]
	}
	if reqMap["requestId"] != nil {
		errRespMap["requestId"] = reqMap["requestId"]
	}
	errRespMap["error"] = `{"number":` + number + `,"reason":"` + reason + `","message":"` + message + `"}`
        errRespMap["timestamp"] = GetRfcTime()
}

func FinalizeMessage(responseMap map[string]interface{}) string {
	response, err := json.Marshal(responseMap)
	if err != nil {
		Error.Print("Server core-FinalizeMessage: JSON encode failed. ", err)
		return `{"error":{"number":400,"reason":"JSON marshal error","message":""}}` //???
	}
	return string(response)
}

func GetRfcTime() string {
    withTimeZone := time.Now().Format(time.RFC3339)   // 2020-05-01T15:34:35+02:00
    if (withTimeZone[len(withTimeZone)-6] == '+') {
        return withTimeZone[:len(withTimeZone)-6] + "Z"
    } else {
        return withTimeZone
    }
}

func nextQuoteMark(message []byte, offset int) int {
    for i := offset ; i < len(message) ; i++ {
        if (message[i] == '"') {
            return i
        }
    }
    return offset
}

func getMessageToken(message []byte, offset int) []byte {
    var token []byte
    if (message[offset] == '"') {
        offset2 := nextQuoteMark(message, offset+1)
        token = message[offset:offset2+1]
    } else {
        token = []byte(string(message[offset]))
    }
    return token
}

func getKwListIndex(token string) byte {
    var i byte
    listLen := byte(len(kwList.Kw))
    for i = 0 ; i < listLen ; i++ {
Info.Printf("kwList.Kw[%d]=%s, token=%s", i, kwList.Kw[i], token)
        if (kwList.Kw[i] == token[1:len(token)-1]) {
            return i
        }
    }
    return 255
}

func DecompressMessage(message []byte) []byte {
    var message2 []byte
    message2 = message // for testing of compress...
    return message2
}

func CompressMessage(message []byte) []byte {
    var message2 []byte
    if (len(kwList.Kw) == 0) {
        jsonToStructList(keywordlist)
    }
    for offset := 0 ; offset < len(message) ; {
        token := getMessageToken(message, offset)
Info.Printf("Token=%s, len=%d", string(token), len(token))
        offset += len(token)
        if (len(token) == 1) {
            message2 = append(message2, token...)
        } else {
            listIndex := getKwListIndex(string(token))
Info.Printf("listIndex=%d", listIndex)
            listLen := byte(len(kwList.Kw))
            if (listIndex < listLen) {
                listIndex += 16
                index := make([]byte, 1)
                index[0] = listIndex
Info.Printf("index=%d, index[0]=%d", index, index[0])
                message2 = append(message2, index...)  
            } else if (listIndex == KEYWORDLISTINDEXTS) {
                message2 = append(message2, token...)   // this should not be, only for temp testing => previousToken = token, tested on when listIndex < 0...
//                previousToken = token
            } else {
                message2 = append(message2, token...)
            }
        }
    }
    Info.Printf("Compressed message:%s, len(message)=%d", message2, len(message2))
    return message2
}

/*
* The keywordlist shall contain all keys used in JSON payloads, and also all "constant" key values.
  If the list is extended, the keys must be placed before the constant key values in the list, 
* and the KEYWORDLISTDELIM must be updated to the number of elements in the list that are keys.
*/
var keywordlist string = `{"keywords":["action", "path", "value", "timestamp", "requestId", "subscriptionId", "filter", "authorization", "get", "set", "subscribe", "unsubscribe", "subscription"]}`

const KEYWORDLISTDELIM = 8  // must be set to the number of keywordlist elements that are keys
const KEYWORDLISTINDEXTS = 3  // must be set to the list index of the "timestamp" element

type KwList struct {
	Kw []string `json:"keywords"`
}

var kwList KwList

func jsonToStructList(jsonList string) int {
	err := json.Unmarshal([]byte(jsonList), &kwList)
	if err != nil {
		Error.Printf("Error unmarshal json=%s\n", err)
		return 0
	}
Info.Printf("jsonToStructList():len(kwList.Kw)=%d", len(kwList.Kw))
	return len(kwList.Kw)
}


