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

type filterDef_t struct {
	name     string
	operator string
	value    string
}

type SubscriptionState struct {
	subscriptionId int
	mgrId          int
	clientId       int
	requestId      string
	path           string
	filterList     []filterDef_t
	latestValue    string
	timestamp      string
}

var hostIp string

var errorResponseMap = map[string]interface{}{
	"MgrId":     0,
	"ClientId":  0,
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
	for i := 0; i < len(subscriptionList); i++ {
		if subscriptionList[i].subscriptionId == subscriptionId {
			return i
		}
	}
	return -1
}

func checkRangeChangeFilter(filterList []filterDef_t, latestVal string, currentVal string) bool {
        latestValue, _ := strconv.Atoi(latestVal)
        currentValue, _ := strconv.Atoi(currentVal)
	for i := range filterList {
		result := evaluateFilter(filterList[i], latestValue, currentValue)
		if result == false {
			return false
		}
	}
	return true
}

func evaluateFilter(filter filterDef_t, latestValue int, currentValue int) bool {
	if filter.name == "$range" {
		if filter.operator == "gt" {
			filterValue, _ := strconv.Atoi(filter.value)
			if currentValue > filterValue {
				return true
			}
			return false
		} else { // "lt"
			filterValue, _ := strconv.Atoi(filter.value)
			if currentValue < filterValue {
				return true
			}
			return false
		}
	}
	if filter.name == "$change" {
		if filter.operator == "gt" { // abs(current-latest) > filter
			filterValue, _ := strconv.Atoi(filter.value)
			if currentValue > latestValue+filterValue {
				return true
                        }
			if currentValue < latestValue-filterValue {
				return true
			}
			return false
		} else if filter.operator == "lt" {  // abs(current-latest) < filter
			filterValue, _ := strconv.Atoi(filter.value)
			if currentValue < latestValue+filterValue {
				return true
			}
			if currentValue > latestValue-filterValue {
				return true
			}
			return false
		} else { // "neq"
			if currentValue != latestValue {
				return true
			}
			return false
		}
	}
	return false
}

func checkSubscription(subscriptionChannel chan int, backendChannel chan string, subscriptionList []SubscriptionState) {
	var subscriptionMap = make(map[string]interface{})
	subscriptionMap["action"] = "subscription"
	select {
	case subscriptionId := <-subscriptionChannel: // $interval triggered
		subscriptionState := subscriptionList[getSubcriptionStateIndex(subscriptionId, subscriptionList)]
		subscriptionMap["subscriptionId"] = strconv.Itoa(subscriptionState.subscriptionId)
		subscriptionMap["MgrId"] = subscriptionState.mgrId
		subscriptionMap["ClientId"] = subscriptionState.clientId
		subscriptionMap["requestId"] = subscriptionState.requestId
		subscriptionMap["value"], subscriptionMap["timestamp"]  = getVehicleData(subscriptionState.path)
 	                       backendChannel <- utils.FinalizeMessage(subscriptionMap)
	default:
		// check $range, $change trigger points
		for i := range subscriptionList {
	               currentValue, timeStamp := getVehicleData(subscriptionList[i].path)
			doTrigger := checkRangeChangeFilter(subscriptionList[i].filterList, subscriptionList[i].latestValue, currentValue)
			if doTrigger == true {
				subscriptionState := subscriptionList[i]
				subscriptionMap["subscriptionId"] = strconv.Itoa(subscriptionState.subscriptionId)
				subscriptionMap["MgrId"] = subscriptionState.mgrId
				subscriptionMap["ClientId"] = subscriptionState.clientId
				subscriptionMap["requestId"] = subscriptionState.requestId
				subscriptionMap["value"] = currentValue
				subscriptionMap["timestamp"]  = timeStamp
  			        subscriptionList[i].latestValue = subscriptionMap["value"].(string)
				backendChannel <- utils.FinalizeMessage(subscriptionMap)
			}
		}
	}
}

func updateState(path string, subscriptionState *SubscriptionState) {

}

func processOneFilter(filter string, filterList *[]filterDef_t) int {
	filterDef := filterDef_t{}
	if strings.Contains(filter, "$interval") == true {
		filterDef.name = "$interval"
	} else if strings.Contains(filter, "$range") == true {
		filterDef.name = "$range"
	} else if strings.Contains(filter, "$change") == true {
		filterDef.name = "$change"
	} else {
		return 0
	}
	valueStart := strings.Index(filter, "EQ")
	if valueStart != -1 {
		filterDef.operator = "eq"
	} else {
		valueStart = strings.Index(filter, "GT")
		if valueStart != -1 {
			filterDef.operator = "gt"
		} else {
			valueStart = strings.Index(filter, "LT")
			if valueStart != -1 {
				filterDef.operator = "lt"
			}
		}
	}
	filterDef.value = filter[valueStart+2:]
	*filterList = append(*filterList, filterDef)
	utils.Info.Printf("processOneFilter():filter.name=%s, filter.operator=%s, filter.value=%s\n", filterDef.name, filterDef.operator, filterDef.value)
        return 1
}

func processFilters(path string, filterList *[]filterDef_t) int {
	utils.Info.Printf("Service-mgr: Entering processFilters().Filter=%s", path)
	queryDelim := strings.Index(path, "?")
	query := path[queryDelim+1:]
	if queryDelim == -1 {
		return 0
	}
	numOfFilters := strings.Count(query, "AND") + 1
	utils.Info.Printf("processFilters():#filter=%d\n", numOfFilters)
	filterStart := 0
        processedFilters := 0
	for i := 0; i < numOfFilters; i++ {
		filterEnd := strings.Index(query[filterStart:], "AND")
		if filterEnd == -1 {
			filterEnd = len(query)
		}
		filter := query[filterStart:filterEnd]
		processedFilters += processOneFilter(filter, filterList)
		filterStart = filterEnd + 3 //len(AND)=3
	}
        return processedFilters
}

func deactivateSubscription(subscriptionList []SubscriptionState, subscriptionId string) (int, []SubscriptionState) {
	id, _ := strconv.Atoi(subscriptionId)
	index := getSubcriptionStateIndex(id, subscriptionList)
        if (index == -1) {
            return -1, subscriptionList
        }
	deactivateInterval(subscriptionList[index].subscriptionId)
	//remove from list
	subscriptionList[index] = subscriptionList[len(subscriptionList)-1] // Copy last element to index i.
	//    subscriptionList[len(subscriptionList)-1] = ""   // Erase last element (write zero value).
	subscriptionList = subscriptionList[:len(subscriptionList)-1] // Truncate slice.
        return 1, subscriptionList
}

func getIndexForInterval(filterList []filterDef_t) int {
	for i := 0; i < len(filterList); i++ {
		if filterList[i].name == "$interval" {
			return i
		}
	}
	return -1
}

func getVehicleData(path string) (string, string) {
    if (isStateStorage == true) {
	rows, err := db.Query("SELECT `value`, `timestamp` FROM VSS_MAP WHERE `path`=?", path)
	if err != nil {
            return strconv.Itoa(dummyValue), utils.GetRfcTime()
	}
	value := ""
	timestamp := ""

	rows.Next()
	err = rows.Scan(&value, &timestamp)
	if err != nil {
            return strconv.Itoa(dummyValue), utils.GetRfcTime()
	}
	rows.Close()
	return value, timestamp
    } else {
	if (dummyValue%10 == 0) {// Return array type instead. Must be represented as string due to server core inability to handle it otherwise...
	    dummyArray := `["` + strconv.Itoa(dummyValue) + "\",\"" + strconv.Itoa(dummyValue+1) + "\",\"" + strconv.Itoa(dummyValue+2) + "\"]"
	    return dummyArray, utils.GetRfcTime()
	}
        return strconv.Itoa(dummyValue), utils.GetRfcTime()
    }
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
                utils.Error.Printf("Could not open DB file = %s, err = %s\n", os.Args[1], dbErr)
                os.Exit(1)
            }
            defer db.Close()
            isStateStorage = true
        }

	hostIp = utils.GetModelIP(2)
	var regResponse RegResponse
	dataChan := make(chan string)
	backendChan := make(chan string)
	regRequest := RegRequest{Rootnode: "Vehicle"}
	subscriptionChan := make(chan int)
	subscriptionList := []SubscriptionState{}
	subscriptionId := 1 // do not start with zero!

	if registerAsServiceMgr(regRequest, &regResponse) == 0 {
		return
	}
	go initDataServer(utils.MuxServer[1], dataChan, backendChan, regResponse)
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
			responseMap["MgrId"] = requestMap["MgrId"]
			responseMap["ClientId"] = requestMap["ClientId"]
			responseMap["action"] = requestMap["action"]
			responseMap["requestId"] = requestMap["requestId"]
			switch requestMap["action"] {
			case "get":
		               responseMap["value"], responseMap["timestamp"]  = getVehicleData(requestMap["path"].(string))
 		               dataChan <- utils.FinalizeMessage(responseMap)
			case "set":
				// TODO: interact with underlying subsystem to set the value
			        dataChan <- utils.FinalizeMessage(responseMap)
			case "subscribe":
				var subscriptionState SubscriptionState
				subscriptionState.subscriptionId = subscriptionId
				subscriptionState.mgrId = int(requestMap["MgrId"].(float64))
				subscriptionState.clientId = int(requestMap["ClientId"].(float64))
				subscriptionState.requestId = requestMap["requestId"].(string)
				subscriptionState.path = requestMap["path"].(string)
				subscriptionState.filterList = []filterDef_t{}
					utils.Info.Printf("filter=%s", requestMap["filter"])
                                if (requestMap["filter"] == nil || requestMap["filter"] == "") {
		                        utils.SetErrorResponse(requestMap, errorResponseMap, "400", "Filter missing.", "")
			                dataChan <- utils.FinalizeMessage(errorResponseMap)
                                        break
                                }
				filters := processFilters("?"+requestMap["filter"].(string), &(subscriptionState.filterList))
                                if (filters == 0) {
		                    utils.SetErrorResponse(requestMap, errorResponseMap, "400", "Unsupported filter.", "See Gen2 Core documentation.")
			            dataChan <- utils.FinalizeMessage(errorResponseMap)
                                }
				subscriptionState.latestValue, subscriptionState.timestamp = getVehicleData(subscriptionState.path)
				subscriptionList = append(subscriptionList, subscriptionState)
				responseMap["subscriptionId"] = strconv.Itoa(subscriptionId)
				filterIndex := getIndexForInterval(subscriptionState.filterList)
				utils.Info.Printf("filterIndex=%d", filterIndex)
				if filterIndex != -1 {
					interval, err := strconv.Atoi(subscriptionState.filterList[filterIndex].value)
					utils.Info.Printf("interval=%d", interval)
					if err == nil {
						activateInterval(subscriptionChan, subscriptionId, interval)
					}
				}
				subscriptionId++
			        dataChan <- utils.FinalizeMessage(responseMap)
			case "unsubscribe":
                                if requestMap["subscriptionId"] != nil {
                                        status := -1
				        if subscriptionIdStr, ok := requestMap["subscriptionId"].(string); ok {
					        if ok == true {
						        status, subscriptionList = deactivateSubscription(subscriptionList, subscriptionIdStr)
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
			checkSubscription(subscriptionChan, backendChan, subscriptionList)
			time.Sleep(10 * time.Millisecond)
		} // select
	} // for
}
