package ecuFeeder

import (
//	"io"
	"time"
	"encoding/json"
	"io/ioutil"
	"strconv"

	"database/sql"
	_ "modernc.org/sqlite"
//	_ "github.com/mattn/go-sqlite3"
	"github.com/go-redis/redis"
	"github.com/w3c/automotive-viss2/utils"
)

var feederListFile string
var serverPathListFile string

var redisUdsSocket string
var redisImpl bool  // true = redis impl; false=sqlite impl; of state storage
var dbHandle *sql.DB
var dbErr error
var redisClient *redis.Client

type PathList struct {
	LeafPaths []string
}
var pathList PathList

func getServiceDataList(feederMap map[string]interface{}) [][]string {
	var serviceDataList [][]string
	for k, v := range feederMap {
		switch vv := v.(type) {
		case []interface{}:
//			utils.Info.Println(k, "is an array:, len=", strconv.Itoa(len(vv)))
			serviceDataList = make([][]string, len(vv))
			serviceDataList = extractServiceDataList(serviceDataList, vv)
		case map[string]interface{}:
//			utils.Info.Println(k, "is a map:")
			serviceDataList = make([][]string, 1)
			serviceDataList[0] = extractServiceData(vv)
		default:
			utils.Warning.Println(k, "is of an unknown type")
		}
	}
	return serviceDataList
}

func extractServiceDataList(serviceDataList [][]string, serviceList []interface{}) [][]string {
	i := 0
	for _, v := range serviceList {
		serviceDataList[i] = extractServiceData(v.(map[string]interface{}))
		i++
	}
	return serviceDataList
}

func extractServiceData(serviceElem map[string]interface{}) []string {
	var serviceData []string
	var serviceName string
	for k, v := range serviceElem {
		switch vv := v.(type) {
		case string:
//			utils.Info.Println(k, "is a string=", vv)
			if (k == "feederService") {
			    serviceName = vv
			} else { // k == "vssPaths"
			serviceData = make([]string, 2)
			serviceData[1] = vv
			}
		case []interface{}:
//			utils.Info.Println(k, "is an array:, len=", strconv.Itoa(len(vv)))
			serviceData = make([]string, len(vv)+1)
			for i := 0 ; i < len(vv) ; i++ {
//			    utils.Info.Println(i, "is a string=", vv[i])
			    serviceData[i+1] = vv[i].(string)
			}
		default:
			utils.Warning.Println(k, "is of an unknown type")
		}
	}
	serviceData[0] = serviceName
	return serviceData
}

func getVerifiedSignalList(ecuServiceData []string, vssPathList PathList) []string {
	verifiedSignalList := make([]string, len(ecuServiceData)-1)
	verifiedIndex := 0
	for i := 0 ; i < len(ecuServiceData)-1 ; i++ {
	    if (signalVerified(ecuServiceData[i+1], vssPathList) == true) {
	        verifiedSignalList[verifiedIndex] = ecuServiceData[i+1]
	        verifiedIndex++
	    }
	}
	return verifiedSignalList[:verifiedIndex]
}

func signalVerified(vssSignal string, vssPathList PathList) bool {
	for i := 0 ; i < len(vssPathList.LeafPaths) ; i++ {
	    if (vssSignal == vssPathList.LeafPaths[i]) {
	        return true
	    }
	}
	utils.Error.Printf("Signal path=%s is not in the VSS tree", vssSignal)
	return false
}

func jsonToStructList(jsonList string, list interface{}) {
	err := json.Unmarshal([]byte(jsonList), list)
	if err != nil {
		utils.Error.Printf("Error unmarshal json=%s, err=%s", jsonList, err)
		return
	}
}

func createPathList(fname string) PathList {
	var pathList PathList
	pathList.LeafPaths = nil
	data, err := ioutil.ReadFile(fname)
	if err != nil {
		utils.Error.Printf("Error reading %s: %s", fname, err)
		return pathList
	}
	jsonToStructList(string(data), &pathList)
	return pathList
}

/*func createRedisClient() *redis.Client {
	var redisClient *redis.Client
	redisClient = redis.NewClient(&redis.Options{
	    Network:  "unix",
	    Addr:     redisUdsSocket,
	    Password: "",
	    DB:       1,
	})
	err := redisClient.Ping().Err()
	if err != nil {
		utils.Error.Printf("Could not initialise redis DB, err = %s", err)
		return nil
	}
	return redisClient
}*/

func writeStateStorage(vssPath string, val string, ts string) {
	if (redisImpl == true) {
		writeRedisDp(redisClient, vssPath, val, ts)
	} else {
		writeSqliteDp(dbHandle, vssPath, val, ts)
	}
}

func writeRedisDp(redisClient *redis.Client, vssPath string, val string, ts string) {
	dp := `{"val":"` + val + `", "ts":"` + ts + `"}`
	err := redisClient.Set(vssPath, dp, time.Duration(0)).Err()
	if err != nil {
		utils.Error.Printf("Feeder failed to write %s in statestorage. Err=%s", vssPath, err)
	}
}

func writeSqliteDp(sqlDb *sql.DB, vssPath string, val string, ts string) {
	stmt, err := sqlDb.Prepare("UPDATE VSS_MAP SET c_value=?, c_ts=? WHERE `path`=?")
	if err != nil {
		utils.Error.Printf("Could not prepare for statestorage updating, err = %s", err)
		return
	}
	defer stmt.Close()

	_, err = stmt.Exec(val, ts, vssPath)
	if err != nil {
		utils.Error.Printf("Could not update statestorage, err = %s", err)
		return
	}
}

func initServices() {
	data, err := ioutil.ReadFile(feederListFile)
	if err != nil {
		utils.Error.Printf("Error reading %s", feederListFile)
		return
	}
	var feederMap map[string]interface{}
	err = json.Unmarshal([]byte(data), &feederMap)
	if err != nil {
		utils.Error.Printf("Feederlist unmarshal error=%s", err)
		return
	}
	ecuServiceDataList := getServiceDataList(feederMap)
	if (len(ecuServiceDataList) == 0) {
		utils.Error.Printf("Failed to find any services. Feeder terminates.")
		return
	}

	for i, _ := range ecuServiceDataList {
	    switch ecuServiceDataList[i][0] {
	        case "GpsLocation":
	        	signalList := getVerifiedSignalList(ecuServiceDataList[i], pathList)
	        	if (len(signalList) > 0) {
	        	    go locationInterface(signalList)
	        	}
	        case "Ignition":
	        	signalList := getVerifiedSignalList(ecuServiceDataList[i], pathList)
	        	if (len(signalList) > 0) {
	        	    go ignitionInterface(signalList)
	        	}
/*	        case "VehicleIdentificationNumber":
	        	signalList := getVerifiedSignalList(ecuServiceDataList[i], pathList)
	        	if (len(signalList) > 0) {
	        	    go vehicleIdentificationNumberInterface(signalList)
	        	}*/ // activate when testing on vehicle
	    }
	}
}

var fakeLat float64
var fakeLong float64
func locationInterface(signalList []string) {  // real method commented below, this is for testing only
	utils.Info.Printf("Waiting for incoming Vehicle GPS data")
	fakeLat = 56.020362896401075
	fakeLong = 12.599945897013853
	for {
		ts := utils.GetRfcTime()

		if (len(signalList[0]) > 0) {
		    fakeLat += 0.0000000000001
		    lat := strconv.FormatFloat(fakeLat, 'f', -1, 64)
		    writeStateStorage(signalList[0], lat, ts)
		}
		if (len(signalList[1]) > 0) {
		    fakeLong += 0.0000000000001
		    long := strconv.FormatFloat(fakeLong, 'f', -1, 64)
		    writeStateStorage(signalList[1], long, ts)
		}
utils.Info.Printf("Feeder:GPS data written")
		time.Sleep(1111 * time.Millisecond)
	}
}


func ignitionInterface(signalList []string) {
/*	redisClient := createRedisClient()
	if redisClient == nil {
		return
	}*/

}

func vehicleIdentificationNumberInterface(signalList []string) {
/*	redisClient := createRedisClient()
	if redisClient == nil {
		return
	}*/

}

func EcuFeederInit(stateStorageType string, udsPath string, redisSocketFile string, dbFile string, target string) {
	if (target == "ecu") {
		feederListFile = "feederServicesList.json"
		serverPathListFile = "vsspathlist.json"
	} else {
		feederListFile = "feeder/ecuFeeder/feederServicesList.json"
		serverPathListFile = "../vsspathlist.json"  //found in the viss2/server directory
	}

	switch stateStorageType {
	    case "sqlite":
		if utils.FileExists(dbFile) {
		    dbHandle, dbErr = sql.Open("sqlite", dbFile)
		    if dbErr != nil {
			utils.Error.Printf("Could not open state storage file = %s, err = %s", dbFile, dbErr)
			return
		    } else {
		        utils.Info.Printf("SQLite state storage initialised.")
		    }
		} else {
			utils.Error.Printf("Could not find state storage file = %s", dbFile)
		}
		redisImpl = false
	    case "redis":
		redisClient = redis.NewClient(&redis.Options{
		    Network:  "unix",
		    Addr:     udsPath + redisSocketFile,
		    Password: "",
		    DB:       1,
		})
		err := redisClient.Ping().Err()
		if err != nil {
			utils.Error.Printf("Could not initialise redis DB, err = %s", err)
			return
		    } else {
		        utils.Info.Printf("Redis state storage initialised.")
		    }
		redisUdsSocket = udsPath + redisSocketFile
		redisImpl = true
	    default:
		utils.Error.Printf("Unknown state storage type = %s", stateStorageType)
		return
	}

	pathList = createPathList(serverPathListFile)
	if pathList.LeafPaths == nil {
		utils.Error.Printf("Error reading vsspathlist file=%s", serverPathListFile)
		return
	}

	initServices()
	utils.Info.Printf("ecuFeeder started...")

	for {
		time.Sleep(time.Duration(2) * time.Second)  // implement heart beat here by updating ts of some signal?
	}
}

