/**
* (C) 2019 Geotab Inc
* (C) 2019 Volvo Cars
*
* All files and artifacts in the repository at https://github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package main

import (
	"bytes"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"
	"time"
	"os"
	"sync"
        "database/sql"
        _ "github.com/mattn/go-sqlite3"
	"github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/utils"
)

// one muxServer component for service registration, one for the data communication

type RegRequest struct {
	Rootnode string
}

type RegResponse struct {
	Portnum int
	Urlpath string
}

type SubscriptionState struct {
	subscriptionId  int
	routerId        string
	path            []string
	filterList      []utils.FilterObject
	latestDataPoint string
}

var subscriptionId int
var killCurveLogicId int = -1 //mutex m shall be used for read/write
var m sync.Mutex

type CLPack struct {
    DataPack string
    SubscriptionId int
}

var hostIp string

var errorResponseMap = map[string]interface{}{
	"RouterId":  "0?0",
	"action":    "unknown",
	"requestId": "XX",
	"error":     `{"number":AA, "reason": "BB", "message": "CC"}`,
	"timestamp": "yy",
}

var db *sql.DB
var dbErr error
var isStateStorage = false

var dummyValue int  // dummy value returned when nothing better is available. Counts from 0 to 999, wrap around, updated every 50 msec

func registerAsServiceMgr(regRequest RegRequest, regResponse *RegResponse) int {
	url := "http://" + hostIp + ":8082/service/reg"
	utils.Info.Printf("ServerCore URL %s", url)

	data := []byte(`{"Rootnode": "` + regRequest.Rootnode + `"}`)

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(data))
	if err != nil {
		utils.Error.Fatal("registerAsServiceMgr: Error creating request. ", err)
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Host", hostIp+":8082")

	// Set client timeout
	client := &http.Client{Timeout: time.Second * 10}

	// Validate headers are attached
	utils.Info.Println(req.Header)

	// Send request
	resp, err := client.Do(req)
	if err != nil {
		utils.Error.Fatal("registerAsServiceMgr: Error reading response. ", err)
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

func initDataServer(muxServer *http.ServeMux, dataChannel chan string, backendChannel chan string, regResponse RegResponse) {
	serviceDataHandler := makeServiceDataHandler(dataChannel, backendChannel)
	muxServer.HandleFunc(regResponse.Urlpath, serviceDataHandler)
	utils.Info.Printf("initDataServer: URL:%s, Portno:%d\n", regResponse.Urlpath, regResponse.Portnum)
	utils.Error.Fatal(http.ListenAndServe(":"+strconv.Itoa(regResponse.Portnum), muxServer))
}

var subscriptionTicker [100]*time.Ticker
var tickerIndexList [100]int // implicitly initialized with zeroes

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
	subscriptionTicker[index] = time.NewTicker(time.Duration(interval) * 1000 * time.Millisecond) // interval in seconds
	go func() {
		for range subscriptionTicker[index].C {
			subscriptionChannel <- subscriptionId
		}
	}()
}

func deactivateInterval(subscriptionId int) {
	subscriptionTicker[deallocateTicker(subscriptionId)].Stop()
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

func checkRangeChangeFilter(filterList []utils.FilterObject, latestDataPoint string, currentDataPoint string) bool {
    for i := 0 ; i < len(filterList) ; i++ {
        if (filterList[i].OpType == "paths" || filterList[i].OpValue == "time-based" || filterList[i].OpValue == "curve-logic") {
            continue
        }
        if (filterList[i].OpValue == "range") {
            return evaluateRangeFilter(filterList[i].OpExtra, getDPValue(currentDataPoint))
        }
        if (filterList[i].OpValue == "change") {
            return evaluateChangeFilter(filterList[i].OpExtra, getDPValue(latestDataPoint), getDPValue(currentDataPoint))
        }
    }
    return false
}

func getDPValue(dp string) string { // {“value”:”Y”, “ts”:”Z”}
    type DataPoint struct {
        Value string  `json:"value"`
        Ts string     `json:"ts"`
    }
    var dataPoint DataPoint
    err := json.Unmarshal([]byte(dp), &dataPoint)
    if (err != nil) {
        utils.Error.Printf("getDPValue: Unmarshal error=%s", err)
        return ""
    }
    return dataPoint.Value
}

func evaluateRangeFilter(opExtra string, currentValue string) bool {
//utils.Info.Printf("evaluateRangeFilter: opExtra=%s", opExtra)
    type ChangeFilter struct {
        LogicOp  string  `json:"logic-op"`
        Boundary string  `json:"boundary"`
    }
    var changeFilter []ChangeFilter
    var err error
    if (strings.Contains(opExtra, "[") == false) {
        changeFilter = make([]ChangeFilter, 1)
        err = json.Unmarshal([]byte(opExtra), &(changeFilter[0]))
    } else {
        err = json.Unmarshal([]byte(opExtra), &changeFilter)
    }
    if (err != nil) {
        utils.Error.Printf("evaluateChangeFilter: Unmarshal error=%s", err)
        return false
    }
    evaluation := true
    for i := 0 ; i < len(changeFilter) ; i++ {
        evaluation = evaluation && compareValues(changeFilter[i].LogicOp, changeFilter[i].Boundary, currentValue, "0")  // currVal - 0 logic-op boundary
    }
    return evaluation
}

func evaluateChangeFilter(opExtra string, latestValue string, currentValue string) bool {
//utils.Info.Printf("evaluateChangeFilter: opExtra=%s", opExtra)
    type ChangeFilter struct {
        LogicOp string  `json:"logic-op"`
        Diff    string  `json:"diff"`
    }
    var changeFilter ChangeFilter
    err := json.Unmarshal([]byte(opExtra), &changeFilter)
    if (err != nil) {
        utils.Error.Printf("evaluateChangeFilter: Unmarshal error=%s", err)
        return false
    }
    return compareValues(changeFilter.LogicOp, latestValue, currentValue, changeFilter.Diff)
}

func compareValues(logicOp string, latestValue string, currentValue string, diff string) bool {
    if (utils.AnalyzeValueType(latestValue) != utils.AnalyzeValueType(currentValue)) {
        utils.Error.Printf("compareValues: Incompatible types, latVal=%s, curVal=%s", latestValue, currentValue)
        return false
    }
    switch utils.AnalyzeValueType(latestValue) {
        case 0: fallthrough  // string
        case 2: // bool
//utils.Info.Printf("compareValues: value type=bool OR string")
          switch logicOp {
            case "eq": return currentValue == latestValue
            case "ne": return currentValue != latestValue
          }
          return false
        case 1:  // int
//utils.Info.Printf("compareValues: value type=integer, cv=%s, lv=%s, diff=%s", currentValue, latestValue, diff)
          curVal, err := strconv.Atoi(currentValue)
          if (err != nil) {
              return false
          }
          latVal, err := strconv.Atoi(latestValue)
          if (err != nil) {
              return false
          }
          diffVal, err := strconv.Atoi(diff)
          if (err != nil) {
              return false
          }
//utils.Info.Printf("compareValues: value type=integer, cv=%d, lv=%d, diff=%d, logicOp=%s", curVal, latVal, diffVal, logicOp)
         switch logicOp {
            case "eq": return curVal-diffVal == latVal
            case "ne": return curVal-diffVal != latVal
            case "gt": return curVal-diffVal > latVal
            case "gte": return curVal-diffVal >= latVal
            case "lt": return curVal-diffVal < latVal
            case "lte": return curVal-diffVal <= latVal
          }
          return false
        case 3: // float
//utils.Info.Printf("compareValues: value type=float")
          f64Val, err := strconv.ParseFloat(currentValue, 32)
          if (err != nil) {
              return false
          }
          curVal := float32(f64Val)
          f64Val, err = strconv.ParseFloat(latestValue, 32)
          if (err != nil) {
              return false
          }
          latVal := float32(f64Val)
          f64Val, err = strconv.ParseFloat(diff, 32)
          if (err != nil) {
              return false
          }
          diffVal := float32(f64Val)
          switch logicOp {
            case "eq": return curVal-diffVal == latVal
            case "ne": return curVal-diffVal != latVal
            case "gt": return curVal-diffVal > latVal
            case "gte": return curVal-diffVal >= latVal
            case "lt": return curVal-diffVal < latVal
            case "lte": return curVal-diffVal <= latVal
          }
          return false
    }
    return false
}

func addDataPackage(incompleteMessage string, dataPack string) string {
    if (strings.Contains(dataPack, "[") == false) {
        return incompleteMessage[:len(incompleteMessage)-1] + ", \"data\":\"" + dataPack + "\"" + "}"
    } else {
        return incompleteMessage[:len(incompleteMessage)-1] + ", \"data\":" + dataPack + "}"
    }
}

func checkSubscription(subscriptionChannel chan int, CLChan chan CLPack, backendChannel chan string, subscriptionList []SubscriptionState) {
	var subscriptionMap = make(map[string]interface{})
	subscriptionMap["action"] = "subscription"
	select {
	case subscriptionId := <-subscriptionChannel: // interval notification triggered
		subscriptionState := subscriptionList[getSubcriptionStateIndex(subscriptionId, subscriptionList)]
		subscriptionMap["subscriptionId"] = strconv.Itoa(subscriptionState.subscriptionId)
		subscriptionMap["RouterId"] = subscriptionState.routerId
 	        backendChannel <- addDataPackage(utils.FinalizeMessage(subscriptionMap), getDataPack(subscriptionState.path, ""))
 	case clPack := <-CLChan: // curve logic notification
		subscriptionState := subscriptionList[getSubcriptionStateIndex(clPack.SubscriptionId, subscriptionList)]
		subscriptionMap["subscriptionId"] = strconv.Itoa(subscriptionState.subscriptionId)
		subscriptionMap["RouterId"] = subscriptionState.routerId
 	        backendChannel <- addDataPackage(utils.FinalizeMessage(subscriptionMap), clPack.DataPack)
	default:
		// check if range or change notification triggered
		for i := range subscriptionList {
	               triggerDataPoint := getVehicleData(subscriptionList[i].path[0], "")
			doTrigger := checkRangeChangeFilter(subscriptionList[i].filterList, subscriptionList[i].latestDataPoint, triggerDataPoint)
			if doTrigger == true {
				subscriptionState := subscriptionList[i]
				subscriptionMap["subscriptionId"] = strconv.Itoa(subscriptionState.subscriptionId)
				subscriptionMap["RouterId"] = subscriptionState.routerId
  			        subscriptionList[i].latestDataPoint = triggerDataPoint
				backendChannel <- addDataPackage(utils.FinalizeMessage(subscriptionMap), getDataPack(subscriptionList[i].path, ""))
			}
		}
	}
}

func deactivateSubscription(subscriptionList []SubscriptionState, subscriptionId string) (int, []SubscriptionState) {
	id, _ := strconv.Atoi(subscriptionId)
	index := getSubcriptionStateIndex(id, subscriptionList)
        if (index == -1) {
            return -1, subscriptionList
        }
utils.Info.Printf("deactivateSubscription: getOpType(subscriptionList[index].filterList, time-based)=%d", getOpType(subscriptionList[index].filterList, "time-based"))
utils.Info.Printf("deactivateSubscription: getOpType(subscriptionList[index].filterList, curve-logic)=%d", getOpType(subscriptionList[index].filterList, "curve-logic"))
        if (getOpType(subscriptionList[index].filterList, "time-based") == true) {
	    deactivateInterval(subscriptionList[index].subscriptionId)
	} else if (getOpType(subscriptionList[index].filterList, "curve-logic") == true) {
	    m.Lock()
	    killCurveLogicId = subscriptionList[index].subscriptionId
utils.Info.Printf("deactivateSubscription: killCurveLogicId set to %d", killCurveLogicId)
	    m.Unlock()
	}
	//remove from list
	subscriptionList[index] = subscriptionList[len(subscriptionList)-1] // Copy last element to index i.
	subscriptionList = subscriptionList[:len(subscriptionList)-1] // Truncate slice.
        return 1, subscriptionList
}

func getOpType(filterList []utils.FilterObject, opType string) bool {
    for i := 0; i < len(filterList); i++ {
        if filterList[i].OpValue == opType {
	    return true
	}
    }
    return false
}

func getIntervalPeriod(opExtra string) int {  // {"period":"X"}
    type IntervalData struct {
        Period string  `json:"period"`
    }
    var intervalData IntervalData
    err := json.Unmarshal([]byte(opExtra), &intervalData)
    if (err != nil) {
	utils.Error.Printf("getIntervalPeriod: Unmarshal failed, err=%s", err)
        return -1
    }
    period, err := strconv.Atoi(intervalData.Period)
    if (err != nil) {
	utils.Error.Printf("getIntervalPeriod: Invalid period=%s", period)
        return -1
    }
    return period
}

func getCurveLogicParams(opExtra string) (int, int) {  // {"max-err": "X", "buf-size":"Y"}
    type CLData struct {
        MaxErr string   `json:"max-err"`
        BufSize string  `json:"buf-size"`
    }
    var cLData CLData
    err := json.Unmarshal([]byte(opExtra), &cLData)
    if (err != nil) {
	utils.Error.Printf("getIntervalPeriod: Unmarshal failed, err=%s", err)
        return 0, 0
    }
    maxErr, err := strconv.Atoi(cLData.MaxErr)
    if (err != nil) {
	utils.Error.Printf("getIntervalPeriod: MaxErr invalid integer, maxErr=%s", cLData.MaxErr)
        maxErr = 0
    }
    bufSize, err := strconv.Atoi(cLData.BufSize)
    if (err != nil) {
	utils.Error.Printf("getIntervalPeriod: BufSize invalid integer, BufSize=%s", cLData.BufSize)
        maxErr = 0
    }
    return maxErr, bufSize
}

func activateIfIntervalOrCL(filterList []utils.FilterObject, subscriptionChan chan int, CLChan chan CLPack, subscriptionId int, paths []string) {
	for i := 0; i < len(filterList); i++ {
		if filterList[i].OpValue == "time-based" {
			interval := getIntervalPeriod(filterList[i].OpExtra)
			utils.Info.Printf("interval activated, period=%d", interval)
			if (interval >0) {
			    activateInterval(subscriptionChan, subscriptionId, interval)
			}
			break
		}
		if filterList[i].OpValue == "curve-logic" {
			go curveLogicServer(CLChan, subscriptionId, filterList[i].OpExtra, paths, &m)
			break
		}
	}
}

func curveLogicServer(CLChan chan CLPack, subscriptionId int, opExtra string, paths []string, m *sync.Mutex) {
    maxError, bufSize := getCurveLogicParams(opExtra)
    type DataPoint struct {
        Value int //TODO: set data type according to VSS data type
        Ts    string
    }
    clBuf := make([]DataPoint, bufSize)
    bufIndex := 0
    utils.Info.Printf("Curve logic activated with max error=%d, buffer size=%d", maxError, bufSize)
    for {
        // TODO: load buffer with new dp from statestorage
        clBuf[bufIndex].Value = 0
        clBuf[bufIndex].Ts = ""
        bufIndex++
        m.Lock()
        if (killCurveLogicId == subscriptionId) {
            m.Unlock()
            break
        }
        m.Unlock()
        if (bufIndex == bufSize) {
            // TODO: run CL algo, send resulting dataPack on CLChan
	    simulateClResults(paths, subscriptionId, CLChan)  // simulate by having bufIndex counter, send dummy dataPack
            bufIndex = 0
        }
	time.Sleep(200 * time.Millisecond)  // 200 ms is appropriate for simulation case, probably less in non-sim (configuration data from vehicle system?)
    }
    utils.Info.Printf("Curve logic de-activated for subscriptionId=%d", subscriptionId)
}

func simulateClResults(paths []string, subscriptionId int, CLChan chan CLPack) {  //TODO: replace with real CL impl
    dp := `[{“value”:”1”, “ts”:”2020-12-31T23:59:40Z”}, {“value”:”2”, “ts”:”2020-12-31T23:59:56Z”}, {“value”:”3”, “ts”:”2020-12-31T23:59:59Z”}]`
    data := ""
    if (len(paths) > 1) {
        data = "["
    }
    for i := 0 ; i < len(paths) ; i++ {
        data += `{"path":"` + paths[i] + `", "dp":` + dp + "}, "
    }
    data = data[:len(data)-2]
    if (len(paths) > 1) {
        data = "]"
    }
    var clPack CLPack
    clPack.DataPack = data
//    clPack.DataPack = `{"data":` + data + `}`
    clPack.SubscriptionId = subscriptionId
    utils.Info.Printf("simulateClResults:dataPack=%s", clPack.DataPack)
    CLChan <- clPack
}

func getVehicleData(path string, filter string) string { // returns {“value”:”Y”, “ts”:”Z”}
    if (len(filter) > 0) {
    }
    if (isStateStorage == true) {
	rows, err := db.Query("SELECT `value`, `timestamp` FROM VSS_MAP WHERE `path`=?", path)
	if err != nil {
            return `{“value”:”` + strconv.Itoa(dummyValue) + `”, “ts”:”` + utils.GetRfcTime() + `”}`
	}
	value := ""
	timestamp := ""

	rows.Next()
	err = rows.Scan(&value, &timestamp)
	if err != nil {
            return `{“value”:”` + strconv.Itoa(dummyValue) + `”, “ts”:”` + utils.GetRfcTime() + `”}`
	}
	rows.Close()
        return `{“value”:”` + value + `”, “ts”:”` + timestamp + `”}`
    } else {
            return `{“value”:”` + strconv.Itoa(dummyValue) + `”, “ts”:”` + utils.GetRfcTime() + `”}`
    }
}

func setVehicleData(path string, value string) string {
    if (isStateStorage == true) {
	stmt, err := db.Prepare("UPDATE VSS_MAP SET value=?, timestamp=? WHERE `path`=?")
	if err != nil {
                utils.Error.Printf("Could not prepare for statestorage updating, err = %s", err)
		return ""
	}

       ts := utils.GetRfcTime()
	_, err = stmt.Exec(value, ts, path[1:len(path)-1])  // remove quotes surrounding path
	if err != nil {
                utils.Error.Printf("Could not update statestorage, err = %s", err)
		return ""
	}
	stmt.Close()
	return ts
    }
    return ""
}

func unpackPaths(paths string) []string {
    var pathArray []string
    if (strings.Contains(paths, "[") == true) {
        err := json.Unmarshal([]byte(paths), &pathArray)
        if (err != nil) {
            return nil
        }
    } else {
        pathArray = make([]string, 1)
	pathArray[0] = paths[1:len(paths)-1]
   }
   return pathArray
}

func historyServer(muxServer *http.ServeMux, historyChan chan string) {
    histCtrlChannel := make(chan string)
    go initHistoryControlServer(histCtrlChannel, muxServer)
    for {
	select {
	  case histCtrlReq := <-histCtrlChannel: // history config request
	    histCtrlChannel <- processHistoryCtrl(histCtrlReq)
	  case histGetReq := <-historyChan: // history get request
	    historyChan <- processHistoryGet(histGetReq)
          default:
            checkForNewData(/*historyList*/)
	    time.Sleep(50 * time.Millisecond)
	}
    }
}

func processHistoryCtrl(histCtrlReq string) string {
    return "ok"  // TODO
}

func processHistoryGet(histGetReq string) string {
    return ""  // TODO
}

func checkForNewData() {
}

func initHistoryControlServer(histCtrlChan chan string, muxServer *http.ServeMux) {
	utils.Info.Printf("initHistoryControlServer(): :8989/histctrlserver")
	histCtrlServerHandler := makeHistCtrlServerHandler(histCtrlChan)
	muxServer.HandleFunc("/agtserver", histCtrlServerHandler)
	utils.Error.Fatal(http.ListenAndServe(":8989", muxServer))
}

func makeHistCtrlServerHandler(histCtrlChan chan string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		utils.Info.Printf("histCtrlServer:url=%s", req.URL.Path)
		if req.URL.Path != "/histctrlserver" {
			http.Error(w, "404 url path not found.", 404)
		} else if req.Method != "POST" {
			http.Error(w, "400 bad request method.", 400)
		} else {
                        bodyBytes, err := ioutil.ReadAll(req.Body)
                        if err != nil {
                                http.Error(w, "400 request unreadable.", 400)
                        } else {
				utils.Info.Printf("agtServer:received POST request=%s\n", string(bodyBytes))
				histCtrlChan <- string(bodyBytes)
				response := <- histCtrlChan
				utils.Info.Printf("agtServer:POST response=%s", response)
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

func getDataPack(pathArray []string, filter string) string {
    dataPack := ""
    if (len(pathArray) > 1) {
        dataPack += "["
    }
    for i := 0 ; i < len(pathArray) ; i++ {
        dataPoint  := getVehicleData(pathArray[i], filter)
        dataPack += `{“path”:”` + pathArray[i] + `”, “dp”:` + dataPoint + "}, "
    }
    dataPack = dataPack[:len(dataPack)-2]
    if (len(pathArray) > 1) {
        dataPack += "]"
    }
    return dataPack
}

func main() {
	utils.InitLog("service-mgr-log.txt", "./logs")
	dbFile := "statestorage.db"
        if (len(os.Args) == 2) {
            dbFile = os.Args[1]
        }
        if (utils.FileExists(dbFile) == true) {
 	    db, dbErr = sql.Open("sqlite3", dbFile)
	    if dbErr != nil {
                utils.Error.Printf("Could not open DB file = %s, err = %s", os.Args[1], dbErr)
                os.Exit(1)
            }
//            defer db.Close()
            isStateStorage = true
        }

	hostIp = utils.GetModelIP(2)
	var regResponse RegResponse
	dataChan := make(chan string)
	backendChan := make(chan string)
	regRequest := RegRequest{Rootnode: "Vehicle"}
	subscriptionChan := make(chan int)
	historyChannel := make(chan string)
	CLChannel := make(chan CLPack)
	subscriptionList := []SubscriptionState{}
	subscriptionId = 1 // do not start with zero!

	if registerAsServiceMgr(regRequest, &regResponse) == 0 {
		return
	}
	go initDataServer(utils.MuxServer[1], dataChan, backendChan, regResponse)
	go historyServer(utils.MuxServer[2], historyChannel)
	dummyTicker := time.NewTicker(47 * time.Millisecond)
	utils.Info.Printf("initDataServer() done\n")
	for {
		select {
		case request := <-dataChan: // request from server core
			utils.Info.Printf("Service manager: Request from Server core:%s\n", request)
			// TODO: interact with underlying subsystem to get the value
			var requestMap = make(map[string]interface{})
			var responseMap = make(map[string]interface{})
			utils.ExtractPayload(request, &requestMap)
			responseMap["RouterId"] = requestMap["RouterId"]
			responseMap["action"] = requestMap["action"]
			responseMap["requestId"] = requestMap["requestId"]
			switch requestMap["action"] {
			case "set":
                                if (strings.Contains(requestMap["path"].(string), "[") == true) {
		                        utils.SetErrorResponse(requestMap, errorResponseMap, "400", "Forbidden request", "Set request must only address a single end point.")
			                dataChan <- utils.FinalizeMessage(errorResponseMap)
                                       break
                                }
				ts := setVehicleData(requestMap["path"].(string), requestMap["value"].(string))
				if (len(ts) == 0) {
		                        utils.SetErrorResponse(requestMap, errorResponseMap, "400", "Internal error", "Underlying system failed to update.")
			                dataChan <- utils.FinalizeMessage(errorResponseMap)
                                       break
				}
				responseMap["timestamp"] = ts
			        dataChan <- utils.FinalizeMessage(responseMap)
			case "get":
		            pathArray := unpackPaths(requestMap["path"].(string))
                           if (pathArray == nil) {
				    utils.Error.Printf("Unmarshal of path array failed.")
		                   utils.SetErrorResponse(requestMap, errorResponseMap, "400", "Internal error.", "Unmarshal failed on array of paths.")
	                           dataChan <- utils.FinalizeMessage(errorResponseMap)
	                           break
                           }
			    filter := ""
			    if (requestMap["filter"] != nil) {
				filterData, err := json.Marshal(requestMap["filter"])	
				if err != nil {
				    utils.Error.Printf("Marshal of filter failed.")
		                   utils.SetErrorResponse(requestMap, errorResponseMap, "400", "Internal error.", "Marshal failed on filter data.")
	                           dataChan <- utils.FinalizeMessage(errorResponseMap)
	                           break
				}	
				filter = string(filterData)
			    }
	                   dataChan <- addDataPackage(utils.FinalizeMessage(responseMap), getDataPack(pathArray, filter))
			case "subscribe":
				var subscriptionState SubscriptionState
				subscriptionState.subscriptionId = subscriptionId
				subscriptionState.routerId = requestMap["RouterId"].(string)
				subscriptionState.path = unpackPaths(requestMap["path"].(string))
                               if (requestMap["filter"] == nil || requestMap["filter"] == "") {
		                        utils.SetErrorResponse(requestMap, errorResponseMap, "400", "Filter missing.", "")
			                dataChan <- utils.FinalizeMessage(errorResponseMap)
                                       break
                               }
				utils.UnpackFilter(requestMap["filter"], &(subscriptionState.filterList))
                               if (len(subscriptionState.filterList) == 0) {
		                    utils.SetErrorResponse(requestMap, errorResponseMap, "400", "Invalid filter.", "See VISSv2 specification.")
			            dataChan <- utils.FinalizeMessage(errorResponseMap)
                               }
				subscriptionState.latestDataPoint = getVehicleData(subscriptionState.path[0], "")
				subscriptionList = append(subscriptionList, subscriptionState)
				responseMap["subscriptionId"] = strconv.Itoa(subscriptionId)
				activateIfIntervalOrCL(subscriptionState.filterList, subscriptionChan, CLChannel, subscriptionId, subscriptionState.path)
				subscriptionId++  // not to be incremented elsewhere
			        dataChan <- utils.FinalizeMessage(responseMap)
			case "unsubscribe":
                                if requestMap["subscriptionId"] != nil {
                                        status := -1
				        if subscriptId, ok := requestMap["subscriptionId"].(string); ok {
					        if ok == true {
						        status, subscriptionList = deactivateSubscription(subscriptionList, subscriptId)
					        }
                                                if (status != -1) {
			                            dataChan <- utils.FinalizeMessage(responseMap)
                                                    break
                                                }
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
		default:
			checkSubscription(subscriptionChan, CLChannel, backendChan, subscriptionList)
			time.Sleep(50 * time.Millisecond)
		} // select
	} // for
}
