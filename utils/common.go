package utils

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net"
	"os"
	"strings"
	"io/ioutil"
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

func decompressPath(uuidCompressed []byte, uuidLen int) []byte {
    for i := 0 ; i < len(uuidList.Object) ; i++ {
        if (string(uuidCompressed[:uuidLen]) == uuidList.Object[i].Uuid[:uuidLen]) {
            path := make([]byte, 1)
            path[0] = '"'
            path = append(path, []byte(uuidList.Object[i].Path)...)
            quoteByte := make([]byte, 1)
            quoteByte[0] = '"'
            path = append(path, quoteByte...)
            return path
        }
    }
    Error.Printf("Compressed UUID=%s could not be found.", string(uuidCompressed[:uuidLen]))
    return []byte(`"Unknown path"`)
}

func expandTsItem(tsItem byte,index int) []byte { //yyyy-mm-ddThh:mm:ss<.ssss>Z  TODO: support for subsec
    expandedItem := make([]byte, 3)
    expandedItem[0] = tsItem/10 + '0'
    expandedItem[1] = tsItem%10 + '0'
    if (index < 2) {
        expandedItem[2] = '-'
    } else if (index == 2) {
        expandedItem[2] = 'T'
    } else if (index > 2 && index < 5) {
        expandedItem[2] = ':'
    } else {
        expandedItem[2] = 'Z'
    }
    return expandedItem
}

func decompressTs(tsCompressed []byte, tsLen int) []byte {
    tsUncompressed := make([]byte, 3)
    tsUncompressed[0] = '"'
    tsUncompressed[1] = '2'
    tsUncompressed[2] = '0'  // needs to be updated year 2100...
    for i := 0 ; i < tsLen ; i++ {
        tsUncompressed = append(tsUncompressed, expandTsItem(tsCompressed[i],i)...)
    }
    quoteByte := make([]byte, 1)
    quoteByte[0] = '"'
    tsUncompressed = append(tsUncompressed, quoteByte...)
    return tsUncompressed
}

func readCompressedMessage(message []byte, offset int, uuidLen int) ([]byte, int) {
    var unCompressedToken []byte
    noCompressByte := make([]byte, 1)
    bytesRead := 1
    if (message[offset] > 127) {
        noCompressByte[0] = '"'  // quote
        unCompressedToken = append(unCompressedToken, noCompressByte...)
        unCompressedToken = append(unCompressedToken, []byte(kwList.Kw[message[offset]-128])...)
        unCompressedToken = append(unCompressedToken, noCompressByte...)
        if (message[offset]-128 == KEYWORDLISTINDEXPATH) {
            noCompressByte[0] = message[offset+1]  // colon
            unCompressedToken = append(unCompressedToken, noCompressByte...)
            unCompressedToken = append(unCompressedToken, decompressPath(message[offset+2:], uuidLen)...)
            bytesRead += 1 + uuidLen
        } else if (message[offset]-128 == KEYWORDLISTINDEXTS) {
            noCompressByte[0] = message[offset+1]  // colon
            unCompressedToken = append(unCompressedToken, noCompressByte...)
            unCompressedToken = append(unCompressedToken, decompressTs(message[offset+2:], 6)...)
            bytesRead += 1 + 6
        }
    } else {
        noCompressByte[0] = message[offset]
        unCompressedToken = append(unCompressedToken, noCompressByte...)
    }
    return unCompressedToken, bytesRead
}

func DecompressMessage(message []byte, uuidLen int) []byte {
    var message2 []byte
    if (len(kwList.Kw) == 0) {
        jsonToStructList(keywordlist, &kwList)
    }
    if (len(uuidList.Object) == 0) {
        numOfUuids := createUuidList("../uuidlist.txt") // assuming that the file is in the server directory...
        Info.Printf("UUID list elements=%d\n", numOfUuids)
    }
    for offset := 0 ; offset < len(message) ; {
        uncompressedToken, compressedLen := readCompressedMessage(message, offset, uuidLen)
        offset += compressedLen
        message2 = append(message2, uncompressedToken...)
    }
    return message2
}

func readUncompressedMessage(message []byte, offset int) []byte {
    var token []byte
    if (message[offset] == '"') {
        offset2 := nextQuoteMark(message, offset+1)
//        offset2 := strings.Index(string(message[offset+1:]), "\"")
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
//Info.Printf("kwList.Kw[%d]=%s, token=%s", i, kwList.Kw[i], token)
        if (kwList.Kw[i] == token[1:len(token)-1]) {
            return i
        }
    }
    return 255
}

func compressPath(path []byte, uuidLen int) []byte {
    for i := 0 ; i < len(uuidList.Object) ; i++ {
//Info.Printf("%s == %s", uuidList.Object[i].Path, string(path[1:len(path)-1]))
        if (uuidList.Object[i].Path == string(path[1:len(path)-1])) {
            return []byte(uuidList.Object[i].Uuid[:uuidLen])
        }
    }
    return path
}

func twoToOneByte(twoByte []byte) []byte {
    oneByte := make([]byte, 1)
    oneByte[0] = (twoByte[0]-48)*10 + (twoByte[1]-48)  
    return oneByte
}

func compressTS(ts []byte) []byte {  // ts = "YYYY-MM-DDTHH:MM:SS<.sss...>Z"
    var compressedTs []byte

    compressedTs = append(compressedTs, twoToOneByte(ts[3:5])...)  // year, only last two digits
    compressedTs = append(compressedTs, twoToOneByte(ts[6:8])...)  // month
    compressedTs = append(compressedTs, twoToOneByte(ts[9:11])...)  // day
    compressedTs = append(compressedTs, twoToOneByte(ts[12:14])...)  // hour
    compressedTs = append(compressedTs, twoToOneByte(ts[15:17])...)  // minute
    compressedTs = append(compressedTs, twoToOneByte(ts[18:20])...)  // second
//    subsecond := ts[20:len(ts)-2]  TODO: handle subsecond
    return compressedTs
}

func createUuidList(fname string) int {
	data, err := ioutil.ReadFile(fname)
	if err != nil {
		Error.Printf("Error reading %s: %s", fname, err)
		return 0
	}
	jsonToStructList(string(data), &uuidList)
	return len(uuidList.Object)
}

func CompressMessage(message []byte, uuidLen int) []byte {
    var message2 []byte
    if (len(kwList.Kw) == 0) {
        jsonToStructList(keywordlist, &kwList)
    }
    if (len(uuidList.Object) == 0) {
        numOfUuids := createUuidList("../uuidlist.txt") // assuming that the file is in the server directory...
        Info.Printf("UUID list elements=%d\n", numOfUuids)
    }
    var tokenState byte
    tokenState = 255
    for offset := 0 ; offset < len(message) ; {
        token := readUncompressedMessage(message, offset)
//Info.Printf("Token=%s, len=%d", string(token), len(token))
        offset += len(token)
        if (len(token) == 1) {
            if (token[0] != ' ') {  // remove space
                message2 = append(message2, token...)
            }
        } else {
            listIndex := getKwListIndex(string(token))
//Info.Printf("listIndex=%d", listIndex)
            listLen := byte(len(kwList.Kw))
            if (listIndex < listLen) {
                index := make([]byte, 1)
                index[0] = listIndex + 128   //16 gives printout in wsclient.html without binaryMessage set, 128 does not...
                message2 = append(message2, index...)
                if (listIndex == KEYWORDLISTINDEXTS || listIndex == KEYWORDLISTINDEXPATH) {
                    tokenState = listIndex
                }
            } else {
                if (tokenState == KEYWORDLISTINDEXTS) {
                    message2 = append(message2, compressTS(token)...)
                    tokenState = 255
                } else if (tokenState == KEYWORDLISTINDEXPATH) {
                    message2 = append(message2, compressPath(token, uuidLen)...)
                    tokenState = 255
                } else {
                    message2 = append(message2, token...)
                }
            }
        }
    }
//    Info.Printf("Decompressed message=%s", DecompressMessage(message2, uuidLen))
    Info.Printf("Length of compressed message=%d", len(message2))
/*    for i := 0 ; i < len(message2) ; i++ {
        Info.Printf("mess[%d]=%d,", i, message2[i])
    }*/
    return message2
}

/*
* The keywordlist shall contain all keys used in JSON payloads, and also all "constant" key values.
  If the list is extended, the keys shall be placed before the constant key values in the list, 
* and the constant key values at the end of the list.
* The KEYWORDLISTDELIM must be updated to the correct element numbers.
*/
var keywordlist string = `{"keywords":["action", "path", "value", "timestamp", "requestId", "subscriptionId", "filter", "authorization", "get", "set", "subscribe", "unsubscribe", "subscription"]}`

const KEYWORDLISTINDEXPATH = 1  // must be set to the list index of the "path" element
const KEYWORDLISTINDEXTS = 3  // must be set to the list index of the "timestamp" element

type KwList struct {
	Kw []string `json:"keywords"`
}

var kwList KwList

func jsonToStructList(jsonList string, list interface{}) {
	err := json.Unmarshal([]byte(jsonList), list)
	if err != nil {
		Error.Printf("Error unmarshal json=%s\n", err)
		return
	}
//Info.Printf("jsonToStructList():len(kwList.Kw)=%d", len(kwList.Kw))
//	return len(kwList.Kw)
}

type UuidListElem struct {
	Path string  `json:"path"`
	Uuid string  `json:"uuid"`
}

type UuidList struct {
	Object []UuidListElem `json:"leafuuids"`
}

var uuidList UuidList
