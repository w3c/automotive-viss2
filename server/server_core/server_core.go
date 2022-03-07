/**
* (C) 2020 Mitsubishi Electrics Automotive
* (C) 2019 Geotab Inc
* (C) 2019 Volvo Cars
*
* All files and artifacts in the repository at https://github.com/MEAE-GOT/WAII
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package main

import (
	//   "fmt"

	"fmt"
	"log"
	"os"

	"github.com/akamensky/argparse"
	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"

	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	gomodel "github.com/COVESA/vss-tools/binary/go_parser/datamodel"
	golib "github.com/COVESA/vss-tools/binary/go_parser/parserlib"
	"github.com/MEAE-GOT/WAII/utils"
)

/*
* Core-server main tasks:
    - server for transportmgr registrations
    - server for servicemgr registrations
    - server in transportmgr data channel requests
    - client in servicemgr data channel requests
    - router hub for request-response messages
    - request message path verification
    - request message access restriction control
    - service discovery response synthesis
*/

var VSSTreeRoot *gomodel.Node_t

// set to MAXFOUNDNODES in cparserlib.h
const MAXFOUNDNODES = 1500

// add element to both channels if support for new transport protocol is added
var transportDataChan = []chan string{
	make(chan string),
	make(chan string),
	make(chan string),
}

var backendChan = []chan string{
	make(chan string),
	make(chan string),
	make(chan string),
}

var serviceDataPortNum int = 8200 // port number interval [8200-]

// add element if support for new service manager is added
var serviceDataChan = []chan string{
	make(chan string),
	//	make(chan string),
}

/** muxServer[0] is assigned to transport registration server,
*   muxServer[1] is assigned to service registration server,
*   of the following the first part is assigned for transport data servers,
*   and the second part is assigned for service data servers
**/
var muxServer = []*http.ServeMux{
	http.NewServeMux(), // 0 = transport reg
	http.NewServeMux(), // 1 = service reg
	http.NewServeMux(), // 2 = transport data
	http.NewServeMux(), // 3 = transport data
	http.NewServeMux(), // 4 = transport data
	http.NewServeMux(), // 5 = service data
}

type RouterTable_t struct {
	mgrId    int
	mgrIndex int
}

var routerTable []RouterTable_t

var errorResponseMap = map[string]interface{}{
	"RouterId":  0,
	"action":    "unknown",
	"requestId": "XXX",
	"error":     `{"number":AAA, "reason": "BBB", "message": "CCC"}`,
	"ts":        1234,
}

func routerTableAdd(mgrId int, mgrIndex int) {
	var tableElement RouterTable_t
	tableElement.mgrId = mgrId
	tableElement.mgrIndex = mgrIndex
	routerTable = append(routerTable, tableElement)
}

func extractMgrId(routerId string) int { // "RouterId" : "mgrId?clientId"
	delim := strings.Index(routerId, "?")
	mgrId, _ := strconv.Atoi(routerId[:delim])
	return mgrId
}

func routerTableSearchForMgrIndex(routerId string) int {
	mgrId := extractMgrId(routerId)
	for _, element := range routerTable {
		if element.mgrId == mgrId {
			return element.mgrIndex
		}
	}
	return -1
}

func getRouterId(response string) string { // "RouterId" : "mgrId?clientId",
	afterRouterIdKey := strings.Index(response, "RouterId")
	if afterRouterIdKey == -1 {
		return ""
	}
	afterRouterIdKey += 8 + 1 // points to after quote
	routerIdValStart := utils.NextQuoteMark([]byte(response), afterRouterIdKey) + 1
	routerIdValStop := utils.NextQuoteMark([]byte(response), routerIdValStart)
	utils.Info.Printf("getRouterId: %s", response[routerIdValStart:routerIdValStop])
	return response[routerIdValStart:routerIdValStop]
}

func frontendServiceDataComm(dataConn *websocket.Conn, request string) {
	err := dataConn.WriteMessage(websocket.TextMessage, []byte(request))
	if err != nil {
		utils.Error.Print("Service datachannel write error:", err)
	}
}

func backendServiceDataComm(dataConn *websocket.Conn, backendChannel []chan string, serviceIndex int) {
	for {
		_, response, err := dataConn.ReadMessage()
		utils.Info.Printf("Server core: Response from service mgr:%s", string(response))
		if err != nil {
			utils.Error.Println("Service datachannel read error:", err)
			response = []byte(utils.FinalizeMessage(errorResponseMap)) // needs improvement
		}
//utils.Info.Printf("RESPONSE=%s", string(response))
		mgrIndex := routerTableSearchForMgrIndex(getRouterId(string(response)))
		utils.Info.Printf("mgrIndex=%d", mgrIndex)
		backendChannel[mgrIndex] <- string(response)
	}
}

func frontendWSDataSession(conn *websocket.Conn, transportDataChannel chan string) {
	defer conn.Close()
	for {
		_, msg, err := conn.ReadMessage()
		if err != nil {
			utils.Error.Print("read error data WS protocol.", err)
			break
		}

		utils.Info.Printf("%s request: %s", conn.RemoteAddr(), string(msg))
		transportDataChannel <- string(msg) // send request to server hub
	}
}

func backendWSDataSession(conn *websocket.Conn, backendChannel chan string) {
	defer conn.Close()
	for {
		message := <-backendChannel

		utils.Info.Printf("%s Transport mgr server: message= %s", conn.RemoteAddr(), message)
		err := conn.WriteMessage(websocket.TextMessage, []byte(message))
		if err != nil {
			utils.Error.Print("write error data WS protocol.", err)
			break
		}
	}
}

func initVssFile() bool {
	filePath := "vss_vissv2.binary"
	VSSTreeRoot = golib.VSSReadTree(filePath)

	if VSSTreeRoot == nil {
		//		utils.Error.Println("Tree file not found")
		return false
	}

	return true
}

func searchTree(rootNode *gomodel.Node_t, path string, anyDepth bool, leafNodesOnly bool, listSize int, noScopeList []string, validation *int) (int, []golib.SearchData_t) {
	utils.Info.Printf("searchTree(): path=%s, anyDepth=%t, leafNodesOnly=%t", path, anyDepth, leafNodesOnly)
	if len(path) > 0 {
		var searchData []golib.SearchData_t
		var matches int
		searchData, matches = golib.VSSsearchNodes(path, rootNode, MAXFOUNDNODES, anyDepth, leafNodesOnly, listSize, noScopeList, validation)
		//		searchData, matches = golib.VSSsearchNodes(path, rootNode, MAXFOUNDNODES, anyDepth, leafNodesOnly, validation)
		return matches, searchData
	}
	return 0, nil
}

func getPathLen(path string) int {
	for i := 0; i < len(path); i++ {
		if path[i] == 0x00 { // the path buffer defined in searchData_t is initiated with all zeros
			return i
		}
	}
	return len(path)
}

func getTokenErrorMessage(index int) string {
	switch index {
	case 0:
		return "Token missing. "
	case 1:
		return "Invalid token signature. "
	case 2:
		return "Invalid purpose scope. "
	case 3:
		return "Insufficient access mode permission. "
	case 4:
		return "Invalid issued at time. "
	case 5:
		return "Token expired. "
	case 6:
		return ""
	case 7:
		return "Internal error. "
	}
	return "Unknown error. "
}

func setTokenErrorResponse(reqMap map[string]interface{}, errorCode int) {
	errMsg := ""
	bitValid := 1
	for i := 0; i < 8; i++ {
		if errorCode&bitValid == bitValid {
			errMsg += getTokenErrorMessage(i)
		}
		bitValid = bitValid << 1
	}
	utils.SetErrorResponse(reqMap, errorResponseMap, "400", "Bad Request", errMsg)
}

func accessTokenServerValidation(token string, paths string, action string, validation int) int {
	hostIp := utils.GetServerIP()
	url := "http://" + hostIp + ":8600/atserver"
	utils.Info.Printf("accessTokenServerValidation::url = %s", url)

	data := []byte(`{"token":"` + token + `","paths":` + paths + `,"action":"` + action + `","validation":"` + strconv.Itoa(validation) + `"}`)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		utils.Error.Print("accessTokenServerValidation: Error reading request. ", err)
		return -128
	}

	// Set headers
	req.Header.Set("Access-Control-Allow-Origin", "*")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", hostIp+":8600")

	// Set client timeout
	client := &http.Client{Timeout: time.Second * 10}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		utils.Error.Print("accessTokenServerValidation: Error reading response. ", err)
		return -128
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		utils.Error.Print("Error reading response. ", err)
		return -128
	}

	bdy := string(body)
	frontIndex := strings.Index(bdy, "validation") + 13 // point to first char of int-string
	endIndex := nextQuoteMark(bdy[frontIndex:]) + frontIndex
	atsValidation, err := strconv.Atoi(bdy[frontIndex:endIndex])
	if err != nil {
		utils.Error.Print("Error reading validation. ", err)
		return -128
	}
	return atsValidation
}

func verifyToken(token string, action string, paths string, validation int) int {
	validateResult := accessTokenServerValidation(token, paths, action, validation)
	iatStr := utils.ExtractFromToken(token, "iat")
	iat, err := strconv.Atoi(iatStr)
	if err != nil {
		utils.Error.Print("Error reading iat. ", err)
		return -128
	}
	now := time.Now()
	if now.Before(time.Unix(int64(iat), 0)) == true {
		validateResult -= 16
	}
	expStr := utils.ExtractFromToken(token, "exp")
	exp, err := strconv.Atoi(expStr)
	if err != nil {
		utils.Error.Print("Error reading exp. ", err)
		return -128
	}
	if now.After(time.Unix(int64(exp), 0)) == true {
		validateResult -= 32
	}
	return validateResult
}

func nextQuoteMark(message string) int { // strings.Index() seems to have a problem with finding a quote char
	for i := 0; i < len(message); i++ {
		if message[i] == '"' {
			return i
		}
	}
	return -1
}

// nativeCnodeDef.h: nodeTypes_t;
func nodeTypesToString(nodeType int) string {
	switch nodeType {
	case 1:
		return "sensor"
	case 2:
		return "actuator"
	case 3:
		return "attribute"
	case 4:
		return "branch"
	default:
		return ""
	}
}

// nativeCnodeDef.h: nodeDatatypes_t
func nodeDataTypesToString(nodeType int) string {
	switch nodeType {
	case 1:
		return "int8"
	case 2:
		return "uint8"
	case 3:
		return "int16"
	case 4:
		return "uint16"
	case 5:
		return "int32"
	case 6:
		return "uint32"
	case 7:
		return "double"
	case 8:
		return "float"
	case 9:
		return "boolean"
	case 10:
		return "string"
	default:
		return ""
	}
}

func jsonifyTreeNode(nodeHandle *gomodel.Node_t, jsonBuffer string, depth int, maxDepth int) string {
	if depth >= maxDepth {
		return jsonBuffer
	}
	depth++
	var newJsonBuffer string
	nodeName := golib.VSSgetName(nodeHandle)
	newJsonBuffer += `"` + nodeName + `":{`
	nodeType := int(golib.VSSgetType(nodeHandle))
	utils.Info.Printf("nodeType=%d", nodeType)
	newJsonBuffer += `"type":` + `"` + nodeTypesToString(nodeType) + `",`
	nodeDescr := golib.VSSgetDescr(nodeHandle)
	newJsonBuffer += `"description":` + `"` + nodeDescr + `",`
	nodeNumofChildren := golib.VSSgetNumOfChildren(nodeHandle)
	switch nodeType {
	case 4: // branch
	case 1: // sensor
		fallthrough
	case 2: // actuator
		fallthrough
	case 3: // attribute
		// TODO Look for other metadata, unit, enum, ...
		nodeDatatype := golib.VSSgetDatatype(nodeHandle)
		newJsonBuffer += `"datatype:"` + `"` + nodeDataTypesToString(int(nodeDatatype)) + `",`
	default:
		return ""

	}
	if depth < maxDepth {
		if nodeNumofChildren > 0 {
			newJsonBuffer += `"children":` + "{"
		}
		for i := 0; i < nodeNumofChildren; i++ {
			childNode := golib.VSSgetChild(nodeHandle, i)
			newJsonBuffer += jsonifyTreeNode(childNode, jsonBuffer, depth, maxDepth)
		}
		if nodeNumofChildren > 0 {
			newJsonBuffer = newJsonBuffer[:len(newJsonBuffer)-1] // remove comma after curly bracket
			newJsonBuffer += "}"
		}
	}
	if newJsonBuffer[len(newJsonBuffer)-1] == ',' && newJsonBuffer[len(newJsonBuffer)-2] != '}' {
		newJsonBuffer = newJsonBuffer[:len(newJsonBuffer)-1]
	}
	newJsonBuffer += "},"
	return jsonBuffer + newJsonBuffer
}

func countPathSegments(path string) int {
	return strings.Count(path, ".") + 1
}

func getNoScopeList(tokenContext string) ([]string, int) {
	// call ATS to get list
	hostIp := utils.GetServerIP()
	url := "http://" + hostIp + ":8600/atserver"

	data := []byte(`{"context":"` + tokenContext + `"}`)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		utils.Error.Print("getNoScopeList: Error reading request. ", err)
		return nil, 0
	}

	// Set headers
	req.Header.Set("Access-Control-Allow-Origin", "*")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", hostIp+":8600")

	// Set client timeout
	client := &http.Client{Timeout: time.Second * 10}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		utils.Error.Print("getNoScopeList: Error sending request. ", err)
		return nil, 0
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		utils.Error.Print("getNoScopeList:Error reading response. ", err)
		return nil, 0
	}
	var noScopeMap map[string]interface{}
	err = json.Unmarshal(body, &noScopeMap)
	if err != nil {
		utils.Error.Printf("initPurposelist:error data=%s, err=%s", data, err)
		return nil, 0
	}
	return extractNoScopeElementsLevel1(noScopeMap)
}

func extractNoScopeElementsLevel1(noScopeMap map[string]interface{}) ([]string, int) {
	for k, v := range noScopeMap {
		switch vv := v.(type) {
		case string:
			utils.Info.Println(k, "is string", vv)
			noScopeList := make([]string, 1)
			noScopeList[0] = vv
			return noScopeList, 1
		case []interface{}:
			utils.Info.Println(k, "is an array:, len=", strconv.Itoa(len(vv)))
			return extractNoScopeElementsLevel2(vv)
		default:
			utils.Info.Println(k, "is of an unknown type")
		}
	}
	return nil, 0
}

func extractNoScopeElementsLevel2(noScopeMap []interface{}) ([]string, int) {
	noScopeList := make([]string, len(noScopeMap))
	i := 0
	for k, v := range noScopeMap {
		switch vv := v.(type) {
		case string:
			utils.Info.Println(k, "is string", vv)
			noScopeList[i] = vv
		default:
			utils.Info.Println(k, "is of an unknown type")
		}
		i++
	}
	return noScopeList, i
}

func synthesizeJsonTree(path string, depth int, tokenContext string) string {
	var jsonBuffer string
	var searchData []golib.SearchData_t
	var matches int
	noScopeList, numOfListElem := getNoScopeList(tokenContext)
	//utils.Info.Printf("noScopeList[0]=%s", noScopeList[0])
	matches, searchData = searchTree(VSSTreeRoot, path, false, false, numOfListElem, noScopeList, nil)
	if matches < countPathSegments(path) {
		return ""
	}
	subTreeRoot := searchData[matches-1].NodeHandle
	utils.Info.Printf("synthesizeJsonTree:subTreeRoot-name=%s", golib.VSSgetName(subTreeRoot))
	if depth == 0 {
		depth = 100
	}
	jsonBuffer = jsonifyTreeNode(subTreeRoot, jsonBuffer, 0, depth)
	if len(jsonBuffer) > 0 {
		return "{" + jsonBuffer[:len(jsonBuffer)-1] + "}" // remove comma
	}
	return ""
}

func getTokenContext(reqMap map[string]interface{}) string {
	if reqMap["authorization"] != nil {
		return utils.ExtractFromToken(reqMap["authorization"].(string), "clx")
	}
	return ""
}

func validRequest(request string, action string) bool {
	switch action {
	case "get":
		return isValidGetParams(request)
	case "set":
		return isValidSetParams(request)
	case "subscribe":
		return isValidSubscribeParams(request)
	case "unsubscribe":
		return isValidUnsubscribeParams(request)
	}
	return false
}

func isValidGetParams(request string) bool {
	if strings.Contains(request, "path") == false {
		return false
	}
	if strings.Contains(request, "filter") == true {
		return isValidGetFilter(request)
	}
	return true
}

func isValidGetFilter(request string) bool { // paths, history,static-metadata, dynamic-metadata supported
	if strings.Contains(request, "paths") == true {
		if strings.Contains(request, "value") == true {
			return true
		}
	}
	if strings.Contains(request, "history") == true {
		if strings.Contains(request, "value") == true {
			return true
		}
	}
	if strings.Contains(request, "static-metadata") == true {
		if strings.Contains(request, "value") == true {
			return true
		}
	}
	if strings.Contains(request, "dynamic-metadata") == true {
		if strings.Contains(request, "value") == true {
			return true
		}
	}
	return false
}

func isValidSetParams(request string) bool {
	return strings.Contains(request, "path") && strings.Contains(request, "value")
}

func isValidSubscribeParams(request string) bool {
	if strings.Contains(request, "path") == false {
		return false
	}
	if strings.Contains(request, "filter") == true {
		return isValidSubscribeFilter(request)
	}
	return true
}

func isValidSubscribeFilter(request string) bool { // paths, history, timebased, range, change, curvelog, static-metadata, dynamic-metadata supported
	if isValidGetFilter(request) == true {
		return true
	}
	if strings.Contains(request, "timebased") == true {
		if strings.Contains(request, "value") == true && strings.Contains(request, "period") == true {
			return true
		}
	}
	if strings.Contains(request, "range") == true {
		if strings.Contains(request, "value") == true && strings.Contains(request, "logic-op") == true &&
			strings.Contains(request, "boundary") == true {
			return true
		}
	}
	if strings.Contains(request, "change") == true {
		if strings.Contains(request, "value") == true && strings.Contains(request, "logic-op") == true &&
			strings.Contains(request, "diff") == true {
			return true
		}
	}
	if strings.Contains(request, "curvelog") == true {
		if strings.Contains(request, "value") == true && strings.Contains(request, "maxerr") == true &&
			strings.Contains(request, "bufsize") == true {
			return true
		}
	}
	return false
}

func isValidUnsubscribeParams(request string) bool {
	return strings.Contains(request, "subscriptionId")
}

func serveRequest(request string, tDChanIndex int, sDChanIndex int) {
	var requestMap = make(map[string]interface{})
	if utils.MapRequest(request, &requestMap) != 0 {
		utils.Error.Printf("serveRequest():invalid JSON format=%s", request)
		utils.SetErrorResponse(requestMap, errorResponseMap, "400", "invalid request syntax", "See VISSv2 spec and JSON RFC for valid request syntax.")
		backendChan[tDChanIndex] <- utils.FinalizeMessage(errorResponseMap)
		return
	}
	if requestMap["action"] == nil || validRequest(request, requestMap["action"].(string)) == false {
		utils.Error.Printf("serveRequest():invalid action params=%s", requestMap["action"])
		utils.SetErrorResponse(requestMap, errorResponseMap, "400", "invalid request syntax", "Request parameter invalid.")
		backendChan[tDChanIndex] <- utils.FinalizeMessage(errorResponseMap)
		return
	}
	if requestMap["path"] != nil && strings.Contains(requestMap["path"].(string), "*") == true {
		utils.Error.Printf("serveRequest():path contained wildcard=%s", requestMap["path"])
		utils.SetErrorResponse(requestMap, errorResponseMap, "400", "invalid request syntax", "Wildcard must be in filter expression.")
		backendChan[tDChanIndex] <- utils.FinalizeMessage(errorResponseMap)
		return
	}
	if requestMap["path"] != nil {
		requestMap["path"] = utils.UrlToPath(requestMap["path"].(string)) // replace slash with dot
	}
	if requestMap["action"] == "set" && requestMap["filter"] != nil {
		utils.Error.Printf("serveRequest():Set request combined with filtering.")
		utils.SetErrorResponse(requestMap, errorResponseMap, "400", "invalid request", "Set request must not contain filtering.")
		backendChan[tDChanIndex] <- utils.FinalizeMessage(errorResponseMap)
		return
	}
	if requestMap["action"] == "unsubscribe" {
		serviceDataChan[sDChanIndex] <- request
		return
	}
	issueServiceRequest(requestMap, tDChanIndex, sDChanIndex)
}

func issueServiceRequest(requestMap map[string]interface{}, tDChanIndex int, sDChanIndex int) {
	rootPath := requestMap["path"].(string)
	var searchPath []string
	
	if requestMap["filter"] != nil {
		var filterList []utils.FilterObject
		utils.UnpackFilter(requestMap["filter"], &filterList)
		for i := 0; i < len(filterList); i++ {
			utils.Info.Printf("filterList[%d].Type=%s, filterList[%d].Value=%s", i, filterList[i].Type, i, filterList[i].Value)
			if filterList[i].Type == "paths" {
				if strings.Contains(filterList[i].Value, "[") == true {
					err := json.Unmarshal([]byte(filterList[i].Value), &searchPath)
					if err != nil {
						utils.Error.Printf("Unmarshal filter path array failed.")
						utils.SetErrorResponse(requestMap, errorResponseMap, "400", "Internal error.", "Unmarshall failed on array of paths.")
						backendChan[tDChanIndex] <- utils.FinalizeMessage(errorResponseMap)
						return
					}
					for i := 0; i < len(searchPath); i++ {
						searchPath[i] = rootPath + "." + utils.UrlToPath(searchPath[i]) // replace slash with dot
					}
				} else {
					searchPath = make([]string, 1)
					searchPath[0] = rootPath + "." + filterList[i].Value
				}
				break // only one paths object is allowed
			}
			if filterList[i].Type == "static-metadata" {
				tokenContext := getTokenContext(requestMap)
				if len(tokenContext) == 0 {
					tokenContext = "Undefined+Undefined+Undefined"
				}
				metadata := ""
				metadata = synthesizeJsonTree(requestMap["path"].(string), 0, tokenContext) // TODO: depth setting via filtering?
				if len(metadata) > 0 {
					delete(requestMap, "path")
					delete(requestMap, "filter")
					requestMap["ts"] = utils.GetRfcTime()
					backendChan[tDChanIndex] <- utils.AddKeyValue(utils.FinalizeMessage(requestMap), "metadata", metadata)
					return
				}
				utils.Error.Printf("Metadata not available.")
				utils.SetErrorResponse(requestMap, errorResponseMap, "400", "Bad request", "Metadata not available.")
				backendChan[tDChanIndex] <- utils.FinalizeMessage(errorResponseMap)
				return
			}
			if filterList[i].Type == "dynamic-metadata" && filterList[i].Value == "server_capabilities" {
			    serviceDataChan[sDChanIndex] <- utils.FinalizeMessage(requestMap) // no further verification
			    return
			}
		}
	}
	if requestMap["filter"] == nil || len(searchPath) == 0 {
		searchPath = make([]string, 1)
		searchPath[0] = rootPath
	}
	var searchData []golib.SearchData_t
	var matches int
	totalMatches := 0
	paths := ""
	maxValidation := -1
	for i := 0; i < len(searchPath); i++ {
		anyDepth := true
		validation := -1
		matches, searchData = searchTree(VSSTreeRoot, searchPath[i], anyDepth, true, 0, nil, &validation)
		//utils.Info.Printf("Path=%s, Matches=%d. Max validation from search=%d", searchPath[i], matches, int(validation))
		utils.Info.Printf("Matches=%d. Max validation from search=%d", matches, int(validation))
		for i := 0; i < matches; i++ {
			pathLen := getPathLen(string(searchData[i].NodePath[:]))
			paths += "\"" + string(searchData[i].NodePath[:pathLen]) + "\", "
		}
		totalMatches += matches
		if int(validation) > maxValidation {
			maxValidation = int(validation)
		}
	}
	if totalMatches == 0 {
		utils.SetErrorResponse(requestMap, errorResponseMap, "400", "No signals matching path.", "")
		backendChan[tDChanIndex] <- utils.FinalizeMessage(errorResponseMap)
		return
	}
	if requestMap["action"] == "set" && golib.VSSgetType(searchData[0].NodeHandle) != gomodel.ACTUATOR {
		utils.SetErrorResponse(requestMap, errorResponseMap, "400", "Illegal command", "Only the actuator node type can be set.")
		backendChan[tDChanIndex] <- utils.FinalizeMessage(errorResponseMap)
		return
	}
	paths = paths[:len(paths)-2]
	if totalMatches > 1 {
		paths = "[" + paths + "]"
	}
	switch maxValidation {
	case 0: // validation not required
	case 1:
		fallthrough
	case 2:
		errorCode := 0
		if requestMap["authorization"] == nil {
			errorCode = -1
		} else {
			if requestMap["action"] == "set" || maxValidation == 2 { // no validation for get/subscribe when validation is 1 (write-only)
				errorCode = verifyToken(requestMap["authorization"].(string), requestMap["action"].(string), paths, maxValidation)
			}
		}
		if errorCode < 0 {
			setTokenErrorResponse(requestMap, errorCode)
			backendChan[tDChanIndex] <- utils.FinalizeMessage(errorResponseMap)
			return
		}
	default: // should not be possible...
		utils.SetErrorResponse(requestMap, errorResponseMap, "400", "Access control tag invalid.", "See VISSv2 spec for access control tagging")
		backendChan[tDChanIndex] <- utils.FinalizeMessage(errorResponseMap)
		return
	}
	requestMap["path"] = paths
	serviceDataChan[sDChanIndex] <- utils.FinalizeMessage(requestMap)
}

type PathList struct {
	LeafPaths []string
}

var pathList PathList

func sortPathList(listFname string) {
	data, err := ioutil.ReadFile(listFname)
	if err != nil {
		utils.Error.Printf("Error reading %s: %s\n", listFname, err)
		return
	}
	err = json.Unmarshal([]byte(data), &pathList)
	if err != nil {
		utils.Error.Printf("Error unmarshal json=%s\n", err)
		return
	}
	sort.Strings(pathList.LeafPaths)
	file, _ := json.Marshal(pathList)
	_ = ioutil.WriteFile(listFname, file, 0644)
}

func createPathListFile(listFname string) {
	// call int VSSGetLeafNodesList(long rootNode, char* leafNodeList);
	golib.VSSGetLeafNodesList(VSSTreeRoot, listFname)
	sortPathList(listFname)
}

func main() {
	// Create new parser object
	parser := argparse.NewParser("print", "Server Core")
	// Create string flag
	logFile := parser.Flag("", "logfile", &argparse.Options{Required: false, Help: "outputs to logfile in ./logs folder"})
	logLevel := parser.Selector("", "loglevel", []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}, &argparse.Options{
		Required: false,
		Help:     "changes log output level",
		Default:  "info"})
	dryRun := parser.Flag("", "dryrun", &argparse.Options{Required: false, Help: "dry run to generate vsspathlist file", Default: false})
	vssJson := parser.String("", "vssJson", &argparse.Options{Required: false, Help: "path and name vssPathlist json file", Default: "../vsspathlist.json"})

	// Parse input
	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
	}

	utils.InitLog("servercore-log.txt", "./logs", *logFile, *logLevel)

	if !initVssFile() {
		utils.Error.Fatal(" Tree file not found")
		return
	}
	createPathListFile(*vssJson) // save in server directory, where transport managers will expect it to be
	if *dryRun {
		utils.Info.Printf("vsspathlist.json created. Job done.")
		return
	}

	serviceIndex := 0 // index assigned to registered services
	router := mux.NewRouter()
	router.HandleFunc("/vsspathlist", pathList.vssPathListHandler).Methods("GET")
	router.HandleFunc("/service/reg", makeServiceRegisterHandler(&serviceIndex, backendChan)).Methods("POST")
	router.HandleFunc("/transport/reg", transportRegisterHandler).Methods("POST")
	router.HandleFunc("/transport/data/{protocol}", makeTransportDataHandler(transportDataChan, backendChan)).
		Headers("Upgrade", "websocket")

	srv := &http.Server{
		Addr:         "0.0.0.0:8081",
		WriteTimeout: time.Second * 15,
		ReadTimeout:  time.Second * 15,
		IdleTimeout:  time.Second * 60,
		Handler:      router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil {
			log.Fatal("ListenAndServe: ", err)
		}
		utils.Info.Printf("Server is listening on %s", srv.Addr)
	}()

	utils.Info.Printf("main():starting loop for channel receptions...")
	for {
		select {
		case request := <-transportDataChan[0]: // request from HTTP/HTTPS mgr
			serveRequest(request, 0, 0)
		case request := <-transportDataChan[1]: // request from WS/WSS mgr
			serveRequest(request, 1, 0)
		case request := <-transportDataChan[2]: // request from MQTT mgr
			serveRequest(request, 2, 0)
			//  case xxx := <- transportDataChan[3]:  // implement when there is a 4th transport protocol mgr
		}
	}
}
