package utils

import (
	"encoding/json"
	"net"
        "os"
	"strings"
        "crypto/hmac"
        "crypto/sha256"
        "encoding/base64"
)

const IpModel = 0  // IpModel = [0,1,2] = [localhost,extIP,envVarIP]
const IpEnvVarName = "GEN2MODULEIP"

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
 	    return "localhost"  //fallback
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

func GenerateHmac(input string, key string) string {  //not a correct JWT signature?
    mac := hmac.New(sha256.New, []byte(key))
    mac.Write([]byte(input))
    return string(mac.Sum(nil))
}

func VerifyTokenSignature(token string, key string) bool {  // compatible with result from generateHmac()
        delimiter := strings.LastIndex(token, ".")
        if (delimiter == -1) {
            return false
        }
        message := token[:delimiter]
        messageMAC := token[delimiter+1:]
	mac := hmac.New(sha256.New, []byte(key))
	mac.Write([]byte(message))
	expectedMAC := mac.Sum(nil)
        if (strings.Compare(messageMAC, base64.RawURLEncoding.EncodeToString(expectedMAC)) == 0) {
            return true
        }
        return false
}

func ExtractFromToken(token string, claim string) string {  // TODO remove white space sensitivity
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


