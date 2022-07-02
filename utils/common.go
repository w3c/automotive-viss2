/**
* (C) 2021 Geotab Inc
*
* All files and artifacts in the repository at https://github.com/w3c/automotive-viss2
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net"
	"os"
	"strconv"
	"strings"
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

func MapRequest(request string, rMap *map[string]interface{}) int {
	decoder := json.NewDecoder(strings.NewReader(request))
	err := decoder.Decode(rMap)
	if err != nil {
		Error.Printf("extractPayload: JSON decode failed for request:%s\n", request)
		return -1
	}
	return 0
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
	if reqMap["RouterId"] != nil {
		errRespMap["RouterId"] = reqMap["RouterId"]
	}
	if reqMap["action"] != nil {
		errRespMap["action"] = reqMap["action"]
	}
	if reqMap["requestId"] != nil {
		errRespMap["requestId"] = reqMap["requestId"]
	}
	if reqMap["subscriptionId"] != nil {
		errRespMap["subscriptionId"] = reqMap["subscriptionId"]
	}
	errMap := map[string]interface{}{
	    "number" : number,
	    "reason" : reason,
	    "message": message,
	}
	errRespMap["error"] = errMap
        errRespMap["ts"] = GetRfcTime()
}

func FinalizeMessage(responseMap map[string]interface{}) string {
	response, err := json.Marshal(responseMap)
	if err != nil {
		Error.Print("Server core-FinalizeMessage: JSON encode failed. ", err)
		return `{"error":{"number":400,"reason":"JSON marshal error","message":""}}` //???
	}
	return string(response)
}

func AddKeyValue(message string, key string, value string) string { // to avoid Marshal() to reformat using \"
	if len(value) > 0 {
		if value[0] == '{' {
			return message[:len(message)-1] + ", \"" + key + "\":" + value + "}"
		}
		return message[:len(message)-1] + ", \"" + key + "\":\"" + value + "\"}"
	}
	return message
}

func GetRfcTime() string {
	withTimeZone := time.Now().Format(time.RFC3339) // 2020-05-01T15:34:35+02:00
	if withTimeZone[len(withTimeZone)-6] == '+' {
		return withTimeZone[:len(withTimeZone)-6] + "Z"
	} else {
		return withTimeZone
	}
}

func FileExists(filename string) bool {
	info, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return false
	}
	return !info.IsDir()
}

type FilterObject struct {
	Type  string
	Value string
}

func UnpackFilter(filter interface{}, fList *[]FilterObject) { // See VISSv CORE, Filtering chapter for filter structure
	switch vv := filter.(type) {
	case []interface{}:
		Info.Println(filter, "is an array:, len=", strconv.Itoa(len(vv)))
		*fList = make([]FilterObject, len(vv))
		unpackFilterLevel1(vv, fList)
	case map[string]interface{}:
		Info.Println(filter, "is a map:")
		*fList = make([]FilterObject, 1)
		unpackFilterLevel2(0, vv, fList)
	default:
		Info.Println(filter, "is of an unknown type")
	}
}

func unpackFilterLevel1(filterArray []interface{}, fList *[]FilterObject) {
	i := 0
	for k, v := range filterArray {
		switch vv := v.(type) {
		case map[string]interface{}:
			Info.Println(k, "is a map:")
			unpackFilterLevel2(i, vv, fList)
		default:
			Info.Println(k, "is of an unknown type")
		}
		i++
	}
}

func unpackFilterLevel2(index int, filterExpression map[string]interface{}, fList *[]FilterObject) {
	for k, v := range filterExpression {
		switch vv := v.(type) {
		case string:
			Info.Println(k, "is string", vv)
			if k == "type" {
				(*fList)[index].Type = vv
			} else if k == "value" {
				(*fList)[index].Value = vv
			}
		case []interface{}:
			Info.Println(k, "is an array:, len=", strconv.Itoa(len(vv)))
			arrayVal, err := json.Marshal(vv)
			if err != nil {
				Error.Print("UnpackFilter(): JSON array encode failed. ", err)
			} else if k == "value" {
				(*fList)[index].Value = string(arrayVal)
			}
		case map[string]interface{}:
			Info.Println(k, "is a map:")
			opValue, err := json.Marshal(vv)
			if err != nil {
				Error.Print("UnpackFilter(): JSON map encode failed. ", err)
			} else {
				(*fList)[index].Value = string(opValue)
			}
		default:
			Info.Println(k, "is of an unknown type")
		}
	}
}

