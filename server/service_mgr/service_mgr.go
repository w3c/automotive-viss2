/**
* (C) 2021 Mitsubishi Electrics Automotive
* (C) 2019 Geotab Inc
* (C) 2019 Volvo Cars
*
* All files and artifacts in the repository at https://github.com/MEAE-GOT/WAII
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"

	//	"sync"
	"time"

	"github.com/MEAE-GOT/WAII/utils"
	"github.com/akamensky/argparse"
	_ "github.com/mattn/go-sqlite3"
)

// one muxServer component for service registration, one for the data communication

type RegRequest struct {
	Rootnode string
}

type SubscriptionState struct {
	subscriptionId      int
	SubscriptionThreads int //only used by subs that spawn multiple threads that return notifications
	routerId            string
	path                []string
	filterList          []utils.FilterObject
	latestDataPoint     string
}

var subscriptionId int

type HistoryList struct {
	Path      string
	Frequency int
	BufSize   int
	Status    int
	BufIndex  int // points to next empty buffer element
	Buffer    []string
}

var historyList []HistoryList
var historyAccessChannel chan string

//var hostIp string

var errorResponseMap = map[string]interface{}{
	"RouterId":  "0?0",
	"action":    "unknown",
	"requestId": "XX",
	"error":     `{"number":AA, "reason": "BB", "message": "CC"}`,
	"ts":        "yy",
}

var db *sql.DB
var dbErr error
var isStateStorage = false

var dummyValue int // dummy value returned when nothing better is available. Counts from 0 to 999, wrap around, updated every 50 msec

func registerAsServiceMgr(hostIp string, regRequest RegRequest, regResponse *utils.SvcRegResponse) int {
	url := "http://" + hostIp + ":8081/service/reg"
	utils.Info.Printf("ServerCore URL %s", url)

	data := []byte(`{"Rootnode": "` + regRequest.Rootnode + `"}`)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		utils.Error.Fatal("registerAsServiceMgr: Error creating request. ", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", hostIp+":8081")

	// Set client timeout
	client := &http.Client{Timeout: time.Second * 10}

	// Validate headers are attached
	utils.Info.Println(req.Header)

	// Send request loop until connection succeeds
	var resp *http.Response
	for {
		resp, err = client.Do(req)
		if err != nil {
			utils.Error.Error("registerAsServiceMgr: Error reading response. ", err)
			time.Sleep(1 * time.Second)
		} else {
			break
		}
	}
	defer resp.Body.Close()

	utils.Info.Println("response Status:", resp.Status)
	utils.Info.Println("response Headers:", resp.Header)

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		utils.Error.Fatal("Error reading response. ", err)
	}
	utils.Info.Printf("%s\n", body)

	err = json.Unmarshal(body, regResponse)
	if err != nil {
		utils.Error.Fatal("Service mgr: Error JSON decoding of response. ", err)
	}

	utils.Info.Printf("registerAsServiceMgr: response: port:%d, path=%s\n", regResponse.Portnum, regResponse.Urlpath)

	if regResponse.Portnum <= 0 {
		utils.Warning.Printf("Service registration denied.\n")
		return 0
	}
	return 1
}

func makeServiceDataHandler(dataChannel chan string, backendChannel chan string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		if req.Header.Get("Upgrade") == "websocket" {
			utils.Info.Printf("we are upgrading to a websocket connection.\n")
			utils.Upgrader.CheckOrigin = func(r *http.Request) bool { return true }
			conn, err := utils.Upgrader.Upgrade(w, req, nil)
			if err != nil {
				utils.Error.Printf("upgrade: %s", err)
				return
			}
			go utils.FrontendWSdataSession(conn, dataChannel, backendChannel)
			go utils.BackendWSdataSession(conn, backendChannel)
		} else {
			utils.Warning.Printf("Client must set up a Websocket session.\n")
		}
	}
}

func initDataServer(muxServer *http.ServeMux, dataChannel chan string, backendChannel chan string, regResponse utils.SvcRegResponse) {
	serviceDataHandler := makeServiceDataHandler(dataChannel, backendChannel)
	muxServer.HandleFunc(regResponse.Urlpath, serviceDataHandler)
	utils.Info.Printf("initDataServer: URL:%s, Portno:%d\n", regResponse.Urlpath, regResponse.Portnum)
	utils.Error.Fatal(http.ListenAndServe(":"+strconv.Itoa(regResponse.Portnum), muxServer))
}

const MAXTICKERS = 255 // total number of active subscription and history tickers
var subscriptionTicker [MAXTICKERS]*time.Ticker
var historyTicker [MAXTICKERS]*time.Ticker
var tickerIndexList [MAXTICKERS]int

func allocateTicker(subscriptionId int) int {
	for i := 0; i < len(tickerIndexList); i++ {
		if tickerIndexList[i] == 0 {
			tickerIndexList[i] = subscriptionId
			return i
		}
	}
	return -1
}

func deallocateTicker(subscriptionId int) int {
	for i := 0; i < len(tickerIndexList); i++ {
		if tickerIndexList[i] == subscriptionId {
			tickerIndexList[i] = 0
			return i
		}
	}
	return -1
}

func activateInterval(subscriptionChannel chan int, subscriptionId int, interval int) {
	index := allocateTicker(subscriptionId)
	if index == -1 {
		utils.Error.Printf("activateInterval: No available ticker.")
		return
	}
	subscriptionTicker[index] = time.NewTicker(time.Duration(interval) * time.Millisecond) // interval in milliseconds
	go func() {
		for range subscriptionTicker[index].C {
			subscriptionChannel <- subscriptionId
		}
	}()
}

func deactivateInterval(subscriptionId int) {
	subscriptionTicker[deallocateTicker(subscriptionId)].Stop()
}

func activateHistory(historyChannel chan int, signalId int, frequency int) {
	index := allocateTicker(signalId)
	if index == -1 {
		utils.Error.Printf("activateHistory: No available ticker.")
		return
	}
	historyTicker[index] = time.NewTicker(time.Duration((3600*1000)/frequency) * time.Millisecond) // freq in cycles per hour
	go func() {
		for range historyTicker[index].C {
			historyChannel <- signalId
		}
	}()
}

func deactivateHistory(signalId int) {
	historyTicker[deallocateTicker(signalId)].Stop()
}

func getSubcriptionStateIndex(subscriptionId int, subscriptionList []SubscriptionState) int {
	//utils.Info.Printf("getSubcriptionStateIndex: subscriptionId=%d, len(subscriptionList)=%d", subscriptionId, len(subscriptionList))
	for i := 0; i < len(subscriptionList); i++ {
		if subscriptionList[i].subscriptionId == subscriptionId {
			return i
		}
	}
	return -1
}

func setSubscriptionListThreads(subscriptionList []SubscriptionState, subThreads SubThreads) []SubscriptionState {
	index := getSubcriptionStateIndex(subThreads.SubscriptionId, subscriptionList)
	subscriptionList[index].SubscriptionThreads = subThreads.NumofThreads
	return subscriptionList
}

func checkRangeChangeFilter(filterList []utils.FilterObject, latestDataPoint string, currentDataPoint string) bool {
	for i := 0; i < len(filterList); i++ {
		if filterList[i].Type == "paths" || filterList[i].Type == "timebased" || filterList[i].Type == "curvelog" {
			continue
		}
		if filterList[i].Type == "range" {
			return evaluateRangeFilter(filterList[i].Value, getDPValue(currentDataPoint))
		}
		if filterList[i].Type == "change" {
			return evaluateChangeFilter(filterList[i].Value, getDPValue(latestDataPoint), getDPValue(currentDataPoint))
		}
	}
	return false
}

func getDPValue(dp string) string {
	value, _ := unpackDataPoint(dp)
	return value
}

func getDPTs(dp string) string {
	_, ts := unpackDataPoint(dp)
	return ts
}

func unpackDataPoint(dp string) (string, string) { // {"value":"Y", "ts":"Z"}
	type DataPoint struct {
		Value string `json:"value"`
		Ts    string `json:"ts"`
	}
	var dataPoint DataPoint
	err := json.Unmarshal([]byte(dp), &dataPoint)
	if err != nil {
		utils.Error.Printf("unpackDataPoint: Unmarshal failed for dp=%s, error=%s", dp, err)
		return "", ""
	}
	return dataPoint.Value, dataPoint.Ts
}

func evaluateRangeFilter(opValue string, currentValue string) bool {
	//utils.Info.Printf("evaluateRangeFilter: opValue=%s", opValue)
	type RangeFilter struct {
		LogicOp  string `json:"logic-op"`
		Boundary string `json:"boundary"`
	}
	var rangeFilter []RangeFilter
	var err error
	if strings.Contains(opValue, "[") == false {
		rangeFilter = make([]RangeFilter, 1)
		err = json.Unmarshal([]byte(opValue), &(rangeFilter[0]))
	} else {
		err = json.Unmarshal([]byte(opValue), &rangeFilter)
	}
	if err != nil {
		utils.Error.Printf("evaluateRangeFilter: Unmarshal error=%s", err)
		return false
	}
	evaluation := true
	for i := 0; i < len(rangeFilter); i++ {
		evaluation = evaluation && compareValues(rangeFilter[i].LogicOp, rangeFilter[i].Boundary, currentValue, "0") // currVal - 0 logic-op boundary
	}
	return evaluation
}

func evaluateChangeFilter(opValue string, latestValue string, currentValue string) bool {
	//utils.Info.Printf("evaluateChangeFilter: opValue=%s", opValue)
	type ChangeFilter struct {
		LogicOp string `json:"logic-op"`
		Diff    string `json:"diff"`
	}
	var changeFilter ChangeFilter
	err := json.Unmarshal([]byte(opValue), &changeFilter)
	if err != nil {
		utils.Error.Printf("evaluateChangeFilter: Unmarshal error=%s", err)
		return false
	}
	return compareValues(changeFilter.LogicOp, latestValue, currentValue, changeFilter.Diff)
}

func compareValues(logicOp string, latestValue string, currentValue string, diff string) bool {
	latestValueType := utils.AnalyzeValueType(latestValue)
	if latestValueType != utils.AnalyzeValueType(currentValue) {
		utils.Error.Printf("compareValues: Incompatible types, latVal=%s, curVal=%s", latestValue, currentValue)
		return false
	}
	switch latestValueType {
	case 0:
		fallthrough // string
	case 2: // bool
		if diff != "0" {
		    utils.Error.Printf("compareValues: invalid parameter for boolean type")
		    return false
		}
		switch logicOp {
		case "eq":
			return currentValue == latestValue
		case "ne":
			return currentValue != latestValue  // true->false OR false->true
		case "gt":
			return latestValue == "false" && currentValue != latestValue  // false->true
		case "lt":
			return latestValue == "true" && currentValue != latestValue  // true->false
		}
		return false
	case 1: // int
		curVal, err := strconv.Atoi(currentValue)
		if err != nil {
			return false
		}
		latVal, err := strconv.Atoi(latestValue)
		if err != nil {
			return false
		}
		diffVal, err := strconv.Atoi(diff)
		if err != nil {
			return false
		}
		//utils.Info.Printf("compareValues: value type=integer, cv=%d, lv=%d, diff=%d, logicOp=%s", curVal, latVal, diffVal, logicOp)
		switch logicOp {
		case "eq":
			return curVal-diffVal == latVal
		case "ne":
			return curVal-diffVal != latVal
		case "gt":
			return curVal-diffVal > latVal
		case "gte":
			return curVal-diffVal >= latVal
		case "lt":
			return curVal-diffVal < latVal
		case "lte":
			return curVal-diffVal <= latVal
		}
		return false
	case 3: // float
		f64Val, err := strconv.ParseFloat(currentValue, 32)
		if err != nil {
			return false
		}
		curVal := float32(f64Val)
		f64Val, err = strconv.ParseFloat(latestValue, 32)
		if err != nil {
			return false
		}
		latVal := float32(f64Val)
		f64Val, err = strconv.ParseFloat(diff, 32)
		if err != nil {
			return false
		}
		diffVal := float32(f64Val)
		//utils.Info.Printf("compareValues: value type=float, cv=%d, lv=%d, diff=%d, logicOp=%s", curVal, latVal, diffVal, logicOp)
		switch logicOp {
		case "eq":
			return curVal-diffVal == latVal
		case "ne":
			return curVal-diffVal != latVal
		case "gt":
			return curVal-diffVal > latVal
		case "gte":
			return curVal-diffVal >= latVal
		case "lt":
			return curVal-diffVal < latVal
		case "lte":
			return curVal-diffVal <= latVal
		}
		return false
	}
	return false
}

func addPackage(incompleteMessage string, packName string, packValue string) string {
	return incompleteMessage[:len(incompleteMessage)-1] + ", \"" + packName + "\":" + packValue + "}"
}

func checkSubscription(subscriptionChannel chan int, CLChan chan CLPack, backendChannel chan string, subscriptionList []SubscriptionState) []SubscriptionState {
	var subscriptionMap = make(map[string]interface{})
	subscriptionMap["action"] = "subscription"
	subscriptionMap["ts"] = utils.GetRfcTime()
	select {
	case subscriptionId := <-subscriptionChannel: // interval notification triggered
		subscriptionState := subscriptionList[getSubcriptionStateIndex(subscriptionId, subscriptionList)]
		subscriptionMap["subscriptionId"] = strconv.Itoa(subscriptionState.subscriptionId)
		subscriptionMap["RouterId"] = subscriptionState.routerId
		backendChannel <- addPackage(utils.FinalizeMessage(subscriptionMap), "data", getDataPack(subscriptionState.path, nil))
	case clPack := <-CLChan: // curve logging notification
		index := getSubcriptionStateIndex(clPack.SubscriptionId, subscriptionList)
		//subscriptionState := subscriptionList[index]
		subscriptionList[index].SubscriptionThreads--
		if clPack.SubscriptionId == closeClSubId && subscriptionList[index].SubscriptionThreads == 0 {
			subscriptionList = removeFromsubscriptionList(subscriptionList, index)
			closeClSubId = -1
		}
		subscriptionMap["subscriptionId"] = strconv.Itoa(subscriptionList[index].subscriptionId)
		subscriptionMap["RouterId"] = subscriptionList[index].routerId
		backendChannel <- addPackage(utils.FinalizeMessage(subscriptionMap), "data", clPack.DataPack)
	default:
		// check if range or change notification triggered
		for i := range subscriptionList {
			triggerDataPoint := getVehicleData(subscriptionList[i].path[0])
			doTrigger := checkRangeChangeFilter(subscriptionList[i].filterList, subscriptionList[i].latestDataPoint, triggerDataPoint)
			subscriptionList[i].latestDataPoint = triggerDataPoint
			if doTrigger == true {
				subscriptionState := subscriptionList[i]
				subscriptionMap["subscriptionId"] = strconv.Itoa(subscriptionState.subscriptionId)
				subscriptionMap["RouterId"] = subscriptionState.routerId
//				subscriptionList[i].latestDataPoint = triggerDataPoint
				backendChannel <- addPackage(utils.FinalizeMessage(subscriptionMap), "data", getDataPack(subscriptionList[i].path, nil))
			}
		}
	}
	return subscriptionList
}

func deactivateSubscription(subscriptionList []SubscriptionState, subscriptionId string) (int, []SubscriptionState) {
	id, _ := strconv.Atoi(subscriptionId)
	index := getSubcriptionStateIndex(id, subscriptionList)
	if index == -1 {
		return -1, subscriptionList
	}
	if getOpType(subscriptionList[index].filterList, "timebased") == true {
		deactivateInterval(subscriptionList[index].subscriptionId)
	} else if getOpType(subscriptionList[index].filterList, "curvelog") == true {
		mcloseClSubId.Lock()
		closeClSubId = subscriptionList[index].subscriptionId
		utils.Info.Printf("deactivateSubscription: closeClSubId set to %d", closeClSubId)
		mcloseClSubId.Unlock()
	}
	if getOpType(subscriptionList[index].filterList, "curvelog") == false {
		subscriptionList = removeFromsubscriptionList(subscriptionList, index)
	}
	return 1, subscriptionList
}

func removeFromsubscriptionList(subscriptionList []SubscriptionState, index int) []SubscriptionState {
	subscriptionList[index] = subscriptionList[len(subscriptionList)-1] // Copy last element to index i.
	subscriptionList = subscriptionList[:len(subscriptionList)-1]       // Truncate slice.
	return subscriptionList
}

func getOpType(filterList []utils.FilterObject, opType string) bool {
	for i := 0; i < len(filterList); i++ {
		if filterList[i].Type == opType {
			return true
		}
	}
	return false
}

func getIntervalPeriod(opValue string) int { // {"period":"X"}
	type IntervalData struct {
		Period string `json:"period"`
	}
	var intervalData IntervalData
	err := json.Unmarshal([]byte(opValue), &intervalData)
	if err != nil {
		utils.Error.Printf("getIntervalPeriod: Unmarshal failed, err=%s", err)
		return -1
	}
	period, err := strconv.Atoi(intervalData.Period)
	if err != nil {
		utils.Error.Printf("getIntervalPeriod: Invalid period=%s", period)
		return -1
	}
	return period
}

func getCurveLoggingParams(opValue string) (float64, int) { // {"maxerr": "X", "bufsize":"Y"}
	type CLData struct {
		MaxErr  string `json:"maxerr"`
		BufSize string `json:"bufsize"`
	}
	var cLData CLData
	err := json.Unmarshal([]byte(opValue), &cLData)
	if err != nil {
		utils.Error.Printf("getIntervalPeriod: Unmarshal failed, err=%s", err)
		return 0.0, 0
	}
	maxErr, err := strconv.ParseFloat(cLData.MaxErr, 64)
	if err != nil {
		utils.Error.Printf("getIntervalPeriod: MaxErr invalid integer, maxErr=%s", cLData.MaxErr)
		maxErr = 0.0
	}
	bufSize, err := strconv.Atoi(cLData.BufSize)
	if err != nil {
		utils.Error.Printf("getIntervalPeriod: BufSize invalid integer, BufSize=%s", cLData.BufSize)
		maxErr = 0.0
	}
	return maxErr, bufSize
}

func activateIfIntervalOrCL(filterList []utils.FilterObject, subscriptionChan chan int, CLChan chan CLPack, subscriptionId int, paths []string) {
	for i := 0; i < len(filterList); i++ {
		if filterList[i].Type == "timebased" {
			interval := getIntervalPeriod(filterList[i].Value)
			utils.Info.Printf("interval activated, period=%d", interval)
			if interval > 0 {
				activateInterval(subscriptionChan, subscriptionId, interval)
			}
			break
		}
		if filterList[i].Type == "curvelog" {
			go curveLoggingServer(CLChan, threadsChan, subscriptionId, filterList[i].Value, paths)
			break
		}
	}
}

func getVehicleData(path string) string { // returns {"value":"Y", "ts":"Z"}
	if isStateStorage == true {
		rows, err := db.Query("SELECT `value`, `timestamp` FROM VSS_MAP WHERE `path`=?", path)
		if err != nil {
			return `{"value":"` + strconv.Itoa(dummyValue) + `", "ts":"` + utils.GetRfcTime() + `"}`
		}
		defer rows.Close()
		value := ""
		timestamp := ""

		rows.Next()
		err = rows.Scan(&value, &timestamp)
		if err != nil {
			return `{"value":"` + strconv.Itoa(dummyValue) + `", "ts":"` + utils.GetRfcTime() + `"}`
		}
		return `{"value":"` + value + `", "ts":"` + timestamp + `"}`
	} else {
		return `{"value":"` + strconv.Itoa(dummyValue) + `", "ts":"` + utils.GetRfcTime() + `"}`
	}
}

func setVehicleData(path string, value string) string {
	if isStateStorage == true {
		stmt, err := db.Prepare("UPDATE VSS_MAP SET value=?, timestamp=? WHERE `path`=?")
		if err != nil {
			utils.Error.Printf("Could not prepare for statestorage updating, err = %s", err)
			return ""
		}
		defer stmt.Close()

		ts := utils.GetRfcTime()
		_, err = stmt.Exec(value, ts, path[1:len(path)-1]) // remove quotes surrounding path
		if err != nil {
			utils.Error.Printf("Could not update statestorage, err = %s", err)
			return ""
		}
		return ts
	}
	return ""
}

func unpackPaths(paths string) []string {
	var pathArray []string
	if strings.Contains(paths, "[") == true {
		err := json.Unmarshal([]byte(paths), &pathArray)
		if err != nil {
			return nil
		}
	} else {
		pathArray = make([]string, 1)
		pathArray[0] = paths[1 : len(paths)-1]
	}
	return pathArray
}

func createHistoryList(vss_data []byte) bool {
	type PathList struct {
		LeafPaths []string
	}

	var pathList PathList
	err := json.Unmarshal(vss_data, &pathList)
	if err != nil {
		utils.Error.Printf("Error unmarshal json, err=%s\n", err)
		return false
	}

	utils.Info.Printf("createHistoryList: len(data.Vsspathlist)=%d, len(pathList.LeafPaths)=%d", len(vss_data), len(pathList.LeafPaths))

	historyList = make([]HistoryList, len(pathList.LeafPaths))
	for i := 0; i < len(pathList.LeafPaths); i++ {
		historyList[i].Path = pathList.LeafPaths[i]
		historyList[i].Frequency = 0
		historyList[i].BufSize = 0
		historyList[i].Status = 0
		historyList[i].BufIndex = 0
		historyList[i].Buffer = nil
	}
	return true
}

func historyServer(historyAccessChan chan string, udsPath string, vss_data []byte) {
	listExists := createHistoryList(vss_data) // file is created by core-server at startup
	histCtrlChannel := make(chan string)
	go initHistoryControlServer(histCtrlChannel, udsPath)
	historyChannel := make(chan int)
	for {
		select {
		case signalId := <-historyChannel:
			captureHistoryValue(signalId)
		case histCtrlReq := <-histCtrlChannel: // history config request
			histCtrlChannel <- processHistoryCtrl(histCtrlReq, historyChannel, listExists)
		case getRequest := <-historyAccessChan: // history get request
			response := ""
			if listExists == true {
				response = processHistoryGet(getRequest)
			}
			historyAccessChan <- response
		default:
			time.Sleep(50 * time.Millisecond)
		}
	}
}

func processHistoryCtrl(histCtrlReq string, historyChan chan int, listExists bool) string {
	if listExists == false {
		utils.Error.Printf("processHistoryCtrl:Path list not found")
		return "500 Internal Server Error"
	}
	var requestMap = make(map[string]interface{})
	utils.MapRequest(histCtrlReq, &requestMap)
	if requestMap["action"] == nil || requestMap["path"] == nil {
		utils.Error.Printf("processHistoryCtrl:Missing command param")
		return "400 Bad Request"
	}
	index := getHistoryListIndex(requestMap["path"].(string))
	switch requestMap["action"].(string) {
	case "create":
		if requestMap["buf-size"] == nil {
			utils.Error.Printf("processHistoryCtrl:Buffer size missing")
			return "400 Bad Request"
		}
		bufSize, err := strconv.Atoi(requestMap["buf-size"].(string))
		if err != nil {
			utils.Error.Printf("processHistoryCtrl:Buffer size malformed=%s", requestMap["buf-size"].(string))
			return "400 Bad Request"
		}
		historyList[index].BufSize = bufSize
		historyList[index].Buffer = make([]string, bufSize)
	case "start":
		if requestMap["frequency"] == nil {
			utils.Error.Printf("processHistoryCtrl:Frequency missing")
			return "400 Bad Request"
		}
		freq, err := strconv.Atoi(requestMap["frequency"].(string))
		if err != nil {
			utils.Error.Printf("processHistoryCtrl:Frequeny malformed=%s", requestMap["frequency"].(string))
			return "400 Bad Request"
		}
		historyList[index].Frequency = freq
		historyList[index].Status = 1
		activateHistory(historyChan, index, freq)
	case "stop":
		historyList[index].Status = 0
		deactivateHistory(index)
	case "delete":
		if historyList[index].Status != 0 {
			utils.Error.Printf("processHistoryCtrl:History recording must first be stopped")
			return "409 Conflict"
		}
		historyList[index].Frequency = 0
		historyList[index].BufSize = 0
		historyList[index].BufIndex = 0
		historyList[index].Buffer = nil
	default:
		utils.Error.Printf("processHistoryCtrl:Unknown command:action=%s", requestMap["action"].(string))
		return "400 Bad Request"
	}
	return "200 OK"
}

func getHistoryListIndex(path string) int {
	for i := 0; i < len(historyList); i++ {
		if historyList[i].Path == path {
			return i
		}
	}
	return -1
}

func getCurrentUtcTime() time.Time {
	return time.Now().UTC()
}

func convertFromIsoTime(isoTime string) (time.Time, error) {
	time, err := time.Parse(time.RFC3339, isoTime)
	return time, err
}

func processHistoryGet(request string) string { // {"path":"X", "period":"Y"}
	var requestMap = make(map[string]interface{})
	utils.MapRequest(request, &requestMap)
	index := getHistoryListIndex(requestMap["path"].(string))
	currentTs := getCurrentUtcTime()
	periodTime, _ := convertFromIsoTime(requestMap["period"].(string))
	oldTs := currentTs.Add(time.Hour*(time.Duration)((24*periodTime.Day()+periodTime.Hour())*(-1)) -
		time.Minute*(time.Duration)(periodTime.Minute()) - time.Second*(time.Duration)(periodTime.Second())).UTC()
	var matches int
	for matches = 0; matches < historyList[index].BufIndex; matches++ {
		storedTs, _ := convertFromIsoTime(getDPTs(historyList[index].Buffer[matches]))
		if storedTs.Before(oldTs) {
			break
		}
	}
	return historicDataPack(index, matches)
}

func historicDataPack(index int, matches int) string {
	dp := ""
	if matches > 1 {
		dp += "["
	}
	for i := 0; i < matches; i++ {
		dp += `{"value":"` + getDPValue(historyList[index].Buffer[i]) + `", "ts":"` + getDPTs(historyList[index].Buffer[i]) + `"}, `
	}
	if matches > 0 {
		dp = dp[:len(dp)-2]
	}
	if matches > 1 {
		dp += "]"
	}
	return dp
}

func captureHistoryValue(signalId int) {
	dp := getVehicleData(historyList[signalId].Path)
	utils.Info.Printf("captureHistoryValue:Captured historic dp = %s", dp)
	newTs := getDPTs(dp)
	latestTs := ""
	if historyList[signalId].BufIndex > 0 {
		latestTs = getDPTs(historyList[signalId].Buffer[historyList[signalId].BufIndex-1])
	}
	if newTs != latestTs && historyList[signalId].BufIndex < historyList[signalId].BufSize-1 {
		historyList[signalId].Buffer[historyList[signalId].BufIndex] = dp
		utils.Info.Printf("captureHistoryValue:Saved historic dp in buffer element=%d", historyList[signalId].BufIndex)
		historyList[signalId].BufIndex++
	}
}

func initHistoryControlServer(histCtrlChan chan string, udsPath string) {
	os.Remove(udsPath)
	l, err := net.Listen("unix", udsPath)
	if err != nil {
		utils.Error.Printf("HistCtrlServer:Listen failed, err = %s", err)
		return
	}

	for {
		conn, err := l.Accept()
		if err != nil {
			utils.Error.Printf("HistCtrlServer:Accept failed, err = %s", err)
			return
		}

		go historyControlServer(conn, histCtrlChan)
	}
}

func historyControlServer(conn net.Conn, histCtrlChan chan string) {
	buf := make([]byte, 512)
	for {
		nr, err := conn.Read(buf)
		if err != nil {
			utils.Error.Printf("HistCtrlServer:Read failed, err = %s", err)
			conn.Close() // assuming client hang up
			return
		}

		data := buf[:nr]
		utils.Info.Printf("HistCtrlServer:Read:data = %s", string(data))
		histCtrlChan <- string(data)
		resp := <-histCtrlChan
		_, err = conn.Write([]byte(resp))
		if err != nil {
			utils.Error.Printf("HistCtrlServer:Write failed, err = %s", err)
			return
		}
	}
}

func getDataPack(pathArray []string, filterList []utils.FilterObject) string {
	dataPack := ""
	if len(pathArray) > 1 {
		dataPack += "["
	}
	getHistory := false
	getDomain := false
	period := ""
	domain := ""
	if filterList != nil {
		for i := 0; i < len(filterList); i++ {
			if filterList[i].Type == "history" {
				period = filterList[i].Value
				utils.Info.Printf("Historic data request, period=%s", period)
				getHistory = true
				break
			} else if (filterList[i].Type == "dynamic-metadata") {
				domain = filterList[i].Value
				utils.Info.Printf("Dynamic metadata request, domain=%s", domain)
				getDomain = true
				break
			}
		}
	}
	var dataPoint string
	var request string
	for i := 0; i < len(pathArray); i++ {
		if getHistory == true {
			request = `{"path":"` + pathArray[i] + `", "period":"` + period + `"}`
			historyAccessChannel <- request
			dataPoint = <-historyAccessChannel
			if len(dataPoint) == 0 {
				return ""
			}
		} else if (getDomain == true) {
			dataPoint = getMetadataDomainDp(domain, pathArray[i])
		} else {
			dataPoint = getVehicleData(pathArray[i])
		}
		dataPack += `{"path":"` + pathArray[i] + `", "dp":` + dataPoint + "}, "
	}
	dataPack = dataPack[:len(dataPack)-2]
	if len(pathArray) > 1 {
		dataPack += "]"
	}
	return dataPack
}

func getVssPathList(host string, port int, path string) []byte {
	url := "http://" + host + ":" + strconv.Itoa(port) + path
	utils.Info.Printf("url = %s", url)

	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		utils.Error.Fatal("getVssPathList: Error creating request:: ", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", host+":"+strconv.Itoa(port))

	// Set client timeout
	client := &http.Client{Timeout: time.Second * 10}

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		utils.Error.Fatal("getVssPathList: Error reading response:: ", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		utils.Error.Fatal("getVssPathList::response Status: ", resp.Status)
	}

	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		utils.Error.Fatal("getVssPathList::Error reading response. ", err)
	}

	utils.Info.Printf("getVssPathList fetched %d bytes", len(data))
	return data
}

func getMetadataDomainDp(domain string, path string) string {
    value := ""
    switch domain {
        case "samplerate": value = getSampleRate(path)
        case "availability": value = getAvailability(path)
        case "validate": value = getValidation(path)
        default: value = "Unknown domain"
    } 
    return `{"value":"` + value + `","ts":"` + utils.GetRfcTime() + `"}`
}

func getSampleRate(path string) string {
    return "X Hz"  //dummy return
}

func getAvailability(path string) string {
    return "available"  //dummy return
}

func getValidation(path string) string {
    return "read-write"  //dummy return
}

func main() {
	// Create new parser object
	parser := argparse.NewParser("print", "Service Manager service")
	// Create string flag
	logFile := parser.Flag("", "logfile", &argparse.Options{Required: false, Help: "outputs to logfile in ./logs folder"})
	logLevel := parser.Selector("", "loglevel", []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}, &argparse.Options{
		Required: false,
		Help:     "changes log output level",
		Default:  "info"})
	udsPath := parser.String("", "uds", &argparse.Options{
		Required: false,
		Help:     "Set UDS path and file",
		Default:  "/tmp/vissv2/histctrlserver.sock"})
	dbFile := parser.String("", "dbfile", &argparse.Options{
		Required: false,
		Help:     "statestorage database filename",
		Default:  "statestorage.db"})

	// Parse input
	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
	}

	//os.Remove("/tmp/vissv2/histctrlserver.sock")
	//listExists := createHistoryList("../vsspathlist.json") // file is created by core-server at startup

	utils.InitLog("service-mgr-log.txt", "./logs", *logFile, *logLevel)
	if utils.FileExists(*dbFile) {
		db, dbErr = sql.Open("sqlite3", *dbFile)
		if dbErr != nil {
			utils.Error.Printf("Could not open DB file = %s, err = %s", *dbFile, dbErr)
			os.Exit(1)
		}
		defer db.Close()
		isStateStorage = true
	}

	var regResponse utils.SvcRegResponse
	dataChan := make(chan string)
	backendChan := make(chan string)
	regRequest := RegRequest{Rootnode: "Vehicle"}
	subscriptionChan := make(chan int)
	historyAccessChannel = make(chan string)
	CLChannel = make(chan CLPack, 5) // allow some buffering...
	subscriptionList := []SubscriptionState{}
	subscriptionId = 1 // do not start with zero!

	var serverCoreIP string = utils.GetModelIP(2)

	if registerAsServiceMgr(serverCoreIP, regRequest, &regResponse) == 0 {
		return
	}

	vss_data := getVssPathList(serverCoreIP, 8081, "/vsspathlist")
	go initDataServer(utils.MuxServer[1], dataChan, backendChan, regResponse)
	go historyServer(historyAccessChannel, *udsPath, vss_data)
	dummyTicker := time.NewTicker(47 * time.Millisecond)

	for {
		select {
		case request := <-dataChan: // request from server core
			utils.Info.Printf("Service manager: Request from Server core:%s\n", request)
			// TODO: interact with underlying subsystem to get the value
			var requestMap = make(map[string]interface{})
			var responseMap = make(map[string]interface{})
			utils.MapRequest(request, &requestMap)
			responseMap["RouterId"] = requestMap["RouterId"]
			responseMap["action"] = requestMap["action"]
			responseMap["requestId"] = requestMap["requestId"]
			responseMap["ts"] = utils.GetRfcTime()
			switch requestMap["action"] {
			case "set":
				if strings.Contains(requestMap["path"].(string), "[") == true {
					utils.SetErrorResponse(requestMap, errorResponseMap, "400", "Forbidden request", "Set request must only address a single end point.")
					dataChan <- utils.FinalizeMessage(errorResponseMap)
					break
				}
				ts := setVehicleData(requestMap["path"].(string), requestMap["value"].(string))
				if len(ts) == 0 {
					utils.SetErrorResponse(requestMap, errorResponseMap, "400", "Internal error", "Underlying system failed to update.")
					dataChan <- utils.FinalizeMessage(errorResponseMap)
					break
				}
				responseMap["ts"] = ts
				dataChan <- utils.FinalizeMessage(responseMap)
			case "get":
				pathArray := unpackPaths(requestMap["path"].(string))
				if pathArray == nil {
					utils.Error.Printf("Unmarshal of path array failed.")
					utils.SetErrorResponse(requestMap, errorResponseMap, "400", "Internal error.", "Unmarshal failed on array of paths.")
					dataChan <- utils.FinalizeMessage(errorResponseMap)
					break
				}
				var filterList []utils.FilterObject
				if requestMap["filter"] != nil && requestMap["filter"] != "" {
					utils.UnpackFilter(requestMap["filter"], &filterList)
					if len(filterList) == 0 {
						utils.Error.Printf("Request filter malformed.")
						utils.SetErrorResponse(requestMap, errorResponseMap, "400", "Bad request", "Request filter malformed.")
						dataChan <- utils.FinalizeMessage(errorResponseMap)
						break
					}
					if (filterList[0].Type == "dynamic-metadata" && filterList[0].Value == "server_capabilities") {
				    	    metadataPack := `{"filter":["paths","timebased","change","range","curvelog","history","dynamic-metadata","static-metadata"],"access_ctrl":["short_term","long_term","signalset_claim"],"transport_protocol":["https","wss","mqtts"]}`
				    	    dataChan <- addPackage(utils.FinalizeMessage(responseMap), "metadata", metadataPack)
				    	    break
				        }
				}
				dataPack := getDataPack(pathArray, filterList)
				if len(dataPack) == 0 {
					utils.Info.Printf("No historic data available")
					utils.SetErrorResponse(requestMap, errorResponseMap, "404", "Not found", "Historic data not available.")
					dataChan <- utils.FinalizeMessage(errorResponseMap)
					break
				}
				dataChan <- addPackage(utils.FinalizeMessage(responseMap), "data", dataPack)
			case "subscribe":
				var subscriptionState SubscriptionState
				subscriptionState.subscriptionId = subscriptionId
				subscriptionState.routerId = requestMap["RouterId"].(string)
				subscriptionState.path = unpackPaths(requestMap["path"].(string))
				if requestMap["filter"] == nil || requestMap["filter"] == "" {
					utils.SetErrorResponse(requestMap, errorResponseMap, "400", "Filter missing.", "")
					dataChan <- utils.FinalizeMessage(errorResponseMap)
					break
				}
				utils.UnpackFilter(requestMap["filter"], &(subscriptionState.filterList))
				if len(subscriptionState.filterList) == 0 {
					utils.SetErrorResponse(requestMap, errorResponseMap, "400", "Invalid filter.", "See VISSv2 specification.")
					dataChan <- utils.FinalizeMessage(errorResponseMap)
				}
				subscriptionState.latestDataPoint = getVehicleData(subscriptionState.path[0])
				subscriptionList = append(subscriptionList, subscriptionState)
				responseMap["subscriptionId"] = strconv.Itoa(subscriptionId)
				activateIfIntervalOrCL(subscriptionState.filterList, subscriptionChan, CLChannel, subscriptionId, subscriptionState.path)
				subscriptionId++ // not to be incremented elsewhere
				dataChan <- utils.FinalizeMessage(responseMap)
			case "unsubscribe":
				if requestMap["subscriptionId"] != nil {
					status := -1
					subscriptId, ok := requestMap["subscriptionId"].(string)
					if ok == true {
						status, subscriptionList = deactivateSubscription(subscriptionList, subscriptId)
						if status != -1 {
							responseMap["subscriptionId"] = subscriptId
							dataChan <- utils.FinalizeMessage(responseMap)
							break
						}
					requestMap["subscriptionId"] = subscriptId
					}
				}
				utils.SetErrorResponse(requestMap, errorResponseMap, "400", "Unsubscribe failed.", "Incorrect or missing subscription id.")
				dataChan <- utils.FinalizeMessage(errorResponseMap)
			default:
				utils.SetErrorResponse(requestMap, errorResponseMap, "400", "Unknown action.", "")
				dataChan <- utils.FinalizeMessage(errorResponseMap)
			} // switch
		case <-dummyTicker.C:
			dummyValue++
			if dummyValue > 999 {
				dummyValue = 0
			}
		case subThreads := <-threadsChan:
			subscriptionList = setSubscriptionListThreads(subscriptionList, subThreads)
		default:
			subscriptionList = checkSubscription(subscriptionChan, CLChannel, backendChan, subscriptionList)
			time.Sleep(50 * time.Millisecond)
		} // select
	} // for
}
