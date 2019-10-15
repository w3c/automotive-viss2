package utils

import (
    "net"
    "encoding/json"
    "strings"
)

/* set to true if localhost to be returned */
const isClientLocal = false

// Get preferred outbound ip of this machine, or sets it to localhost
func GetOutboundIP() string {
    if (isClientLocal == true) {
        return "localhost"
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


