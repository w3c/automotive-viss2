/**
* (C) 2021 Geotab Inc
*
* All files and artifacts in the repository at https://github.com/UlfBj/ccs-w3c-client
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package main

import (
    "sort"
    "strconv"
    "time"
    "sync"
    "encoding/json"
    "io/ioutil"

    "github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/utils"
    _ "github.com/mattn/go-sqlite3"
)

type CLPack struct {
	DataPack       string
	SubscriptionId int
}

var CLChannel chan CLPack

var closeClSubId int = -1
var mcloseClSubId = &sync.Mutex{}

type RingElement struct {
	Value string
	Timestamp string
}

type RingBuffer struct {
    bufSize int
    RingElem []RingElement
    Head int
    Tail int
}

type CLBufElement struct {
	Value float64
	Timestamp int64
}

const MAXCLBUFSIZE = 240   // something large...
const MAXCLSESSIONS = 100  // This value depends on the HW memory and performance
var numOfClSessions int = 0

func createRingBuffer(bufSize int) RingBuffer {
    var aRingBuffer RingBuffer
    aRingBuffer.bufSize = bufSize
    aRingBuffer.Head = 0
    aRingBuffer.Tail = 0
    aRingBuffer.RingElem = make([]RingElement, bufSize)
    return aRingBuffer
}

func getRingHead(aRingBuffer *RingBuffer) int {
    return aRingBuffer.Head
}

func setRingTail(aRingBuffer *RingBuffer, tail int) {
    aRingBuffer.Tail = aRingBuffer.Head - tail
}

func writeRing(aRingBuffer *RingBuffer, value string, timestamp string) {
//utils.Info.Printf("writeRing: value=%s, ts=%s\n", value, timestamp)
    aRingBuffer.RingElem[aRingBuffer.Head].Value = value
    aRingBuffer.RingElem[aRingBuffer.Head].Timestamp = timestamp
    aRingBuffer.Head++
    if (aRingBuffer.Head == aRingBuffer.bufSize) {
        aRingBuffer.Head = 0
    }
}

func readRing(aRingBuffer *RingBuffer, headOffset int) (string, string) {
    currentHead := aRingBuffer.Head - (headOffset + 1)   // Head points to next to write to
    if (currentHead < 0) {
        currentHead += aRingBuffer.bufSize
    }
//utils.Info.Printf("value=%s,timestamp=%s, currentHead=%d,", aRingBuffer.RingElem[currentHead].Value, aRingBuffer.RingElem[currentHead].Timestamp, currentHead)
    return aRingBuffer.RingElem[currentHead].Value, aRingBuffer.RingElem[currentHead].Timestamp
}

func getNumOfPopulatedRingElements(aRingBuffer *RingBuffer) int {
    head := aRingBuffer.Head
    tail := aRingBuffer.Tail
    if (head < tail) {
        head += aRingBuffer.bufSize
    }
    return head - tail
}

type Dim2Elem struct {
    path1 string
    path2 string
}

type Dim3Elem struct {
    path1 string
    path2 string
    path3 string
}

type SignalDimensionLists struct {
    dim2List []Dim2Elem
    dim3List []Dim3Elem
}

type PathDimElem struct {
    Dim int
    Id int
    Populated bool
}

func populateDimLists(paths []string) ([]string, []string, []string) {  // TODO: read signaldimensions.json, populate 1dimList, 2dimList, 3dimList accordingly
    var dim1List []string
    var dim2List []Dim2Elem
    var dim3List []Dim3Elem

    signalDimensionList := readSignalDimensions("signaldimension.json")
    pathDimList := analyzeSignalDimensions(paths, signalDimensionList)

    for i := 0 ; i < len(paths) ; i++ {
        if (pathDimList[i].Dim == 1) {
            dim1List = append(dim1List, paths[i])
        } else if (pathDimList[i].Dim == 2 && pathDimList[i].Populated == false) {
            for j := i+1 ; j < len(paths) ; j++ {
                if (pathDimList[j].Dim == 2 && pathDimList[j].Id == pathDimList[i].Id) {
                    var dim2Elem Dim2Elem
                    dim2Elem.path1 = paths[i]
                    dim2Elem.path2 = paths[j]
                    dim2List = append(dim2List, dim2Elem)
                    pathDimList[j].Populated = true
                }
            }
            
        } else if (pathDimList[i].Dim == 3 && pathDimList[i].Populated == false) {
            for j := i+1 ; j < len(paths) ; j++ {
                if (pathDimList[j].Dim == 3 && pathDimList[j].Id == pathDimList[j].Id) {
                    for k := j+1 ; k < len(paths) ; k++ {
                        if (pathDimList[k].Dim == 3 && pathDimList[k].Id == pathDimList[i].Id) {
                            var dim3Elem Dim3Elem
                            dim3Elem.path1 = paths[i]
                            dim3Elem.path2 = paths[j]
                            dim3Elem.path3 = paths[k]
                            dim3List = append(dim3List, dim3Elem)
                            pathDimList[j].Populated = true
                            pathDimList[k].Populated = true
                        }
                    }
                }
            }
        }
    }
    return dim1List, nil, nil
}

func jsonToStructList(jsonList string) SignalDimensionLists {
        var signalDimensionLists SignalDimensionLists
	err := json.Unmarshal([]byte(jsonList), &signalDimensionLists)
	if err != nil {
		utils.Error.Printf("Error unmarshal signal dimension list=%s\n", err)
//		return 
	}
	return signalDimensionLists
}

func readSignalDimensions(fname string) SignalDimensionLists {
	data, err := ioutil.ReadFile(fname)
	if err != nil {
		utils.Error.Printf("Error reading signal dimension file=%s\n", err)
//		return
	}
	return jsonToStructList(string(data))
}

func analyzeSignalDimensions(paths []string, signalDimensionList SignalDimensionLists) []PathDimElem {
    pathDimList := make([]PathDimElem, len(paths))
    dim2Id := 0
    dim3Id := 0
    for i := 0 ; i < len(paths) ; i++ {
        pathDimList[i].Dim = 1
        pathDimList[i].Id = -1
        pathDimList[i].Populated = false
    }
    for i := 0 ; i < len(paths) ; i++ {
        if (is2dim(paths[i], 1, signalDimensionList.dim2List) == true) {
            for j := i+1 ; j < len(paths) ; j++ {
                if (is2dim(paths[j], 2, signalDimensionList.dim2List) == true) {
                    pathDimList[i].Dim = 2
                    pathDimList[i].Id = dim2Id
                    pathDimList[j].Dim = 2
                    pathDimList[j].Id = dim2Id
                    dim2Id++
                }
            }
        } else if (is3dim(paths[i], 1, signalDimensionList.dim3List) == true) {
            for j := i+1 ; j < len(paths) ; j++ {
                if (is3dim(paths[j], 2, signalDimensionList.dim3List) == true) {
                    for k := j+1 ; k < len(paths) ; k++ {
                        if (is3dim(paths[k], 3, signalDimensionList.dim3List) == true) {
                            pathDimList[i].Dim = 3
                            pathDimList[i].Id = dim3Id
                            pathDimList[j].Dim = 3
                            pathDimList[j].Id = dim3Id
                            pathDimList[k].Dim = 3
                            pathDimList[k].Id = dim3Id
                            dim3Id++
                        } else {
                            pathDimList[i].Dim = 2
                            pathDimList[i].Id = dim2Id
                            pathDimList[j].Dim = 2
                            pathDimList[j].Id = dim2Id
                            dim2Id++
                        }
                    }
                }
            }
        }
    }
    return pathDimList
}

func is2dim(path string, index int, dim2List []Dim2Elem) bool {
    var listPath string
    for i := 0 ; i < len(dim2List) ; i++ {
        if (index == 1) {
            listPath = dim2List[i].path1
        } else if (index == 2) {
            listPath = dim2List[i].path2
        } else {
            return false
        }
        if (listPath == path) {
            return true
        }
    }
    return false
}

func is3dim(path string, index int, dim3List []Dim3Elem) bool {
    var listPath string
    for i := 0 ; i < len(dim3List) ; i++ {
        if (index == 1) {
            listPath = dim3List[i].path1
        } else if (index == 2) {
            listPath = dim3List[i].path2
        } else if (index == 3) {
            listPath = dim3List[i].path3
        } else {
            return false
        }
        if (listPath == path) {
            return true
        }
    }
    return false
}

func curveLogicServer(clChan chan CLPack, subscriptionId int, opExtra string, paths []string) {
	maxError, bufSize := getCurveLogicParams(opExtra)
	if (bufSize > MAXCLBUFSIZE) {
	    bufSize = MAXCLBUFSIZE
	}
	dim1List, dim2List, _ := populateDimLists(paths)
//utils.Info.Printf("len(dim1List)=%d, len(dim2List)=%d", len(dim1List), len(dim2List))
	for i := 0 ; i < len(dim1List) ; i++ {
	    if (numOfClSessions > MAXCLSESSIONS) {
	        utils.Error.Printf("Curve logic: All resources are utilized.")
	        break
	    }
//	    returnInitialDp(clChan, subscriptionId, dim1List[i], 1) //TODO: Very first dp at start of subscribe should be returned? 
	    go clCapture1dim(clChan, subscriptionId, dim1List[i], bufSize, maxError)
	    numOfClSessions++
	}
	for i := 0 ; i < len(dim2List) ; i++ {
	    if (numOfClSessions > MAXCLSESSIONS) {
	        utils.Error.Printf("Curve logic: All resources are utilized.")
	        break
	    }
//	    returnInitialDp(clChan, subscriptionId, dim2List[i], 2) //TODO: Very first dp at start of subscribe should be returned? 
//	    go clCapture2dim(clChan, subscriptionId, dim2List[i], bufSize, maxError)
	    numOfClSessions++
	}
}

func clCapture1dim(clChan chan CLPack, subscriptionId int, path string, bufSize int, maxError float64) {
    aRingBuffer := createRingBuffer(bufSize+1)  // logic requires buffer to have a size of one larger than needed
    var dpMap = make(map[string]interface{})
    closeClSession := false
    for {
	time.Sleep(500 * time.Millisecond)  // TODO:Should be configurable and set to less than sample freq of signal.
	mcloseClSubId.Lock()
	if (closeClSubId == subscriptionId) {
	    closeClSession = true
	}
	mcloseClSubId.Unlock()
        dp := getVehicleData(path)
	utils.ExtractPayload(dp, &dpMap)
	_, ts := readRing(&aRingBuffer, 0)  // read latest written
//utils.Info.Printf("ts=%s", ts)
	if (ts != dpMap["ts"].(string)) {
	    writeRing(&aRingBuffer, dpMap["value"].(string), dpMap["ts"].(string))
	}
	currentBufSize := getNumOfPopulatedRingElements(&aRingBuffer)
	if (currentBufSize == bufSize) || (closeClSession == true) {
//utils.Info.Printf("clAnalyze1dim called for path=%s", path)
	    dp, updatedTail := clAnalyze1dim(&aRingBuffer, currentBufSize, maxError)
            var clPack CLPack
            clPack.DataPack = `{"path":"`+ path + `","data":` + dp + "}"
            clPack.SubscriptionId = subscriptionId
            clChan <- clPack
	    setRingTail(&aRingBuffer, updatedTail)
	}
	if (closeClSession == true) {
	    break
	}
    }
    // TODO:send final notification with last dp?
}

func clAnalyze1dim(aRingBuffer *RingBuffer, bufSize int, maxError float64) (string, int) {  // [{"value":"X","ts":"Y"},..{}] ; square brackets optional
    clBuffer := make([]CLBufElement, bufSize)  // array holds transformed value/ts pairs, from latest to first captured
    for i := 0 ; i < bufSize ; i++ {
        val, ts := readRing(aRingBuffer, i)
        value, err := strconv.ParseFloat(val, 64)
	if err != nil {
		utils.Error.Printf("Curve log failed to convert value=%s to float err=%s", val, err)
		return "", -1
	}
        clBuffer[i].Value = (float64)(value)
        t, err := time.Parse(time.RFC3339, ts)
	if err != nil {
		utils.Error.Printf("Curve log failed to convert time to Unix time err=%s", err)
		return "", -1
	}
        clBuffer[i].Timestamp = t.Unix()
    }
    savedIndex := clReduction(clBuffer, 0, bufSize-1, maxError)
    dataPoint := ""
    updatedTail := 0
    if (savedIndex != nil) {
        sort.Sort(sort.Reverse(sort.IntSlice(savedIndex)))
        updatedTail = savedIndex[len(savedIndex)-1]
        if (len(savedIndex) > 1) {
            dataPoint += "["
        }
        for i := 0 ; i < len(savedIndex) ; i++ {
//utils.Info.Printf("clAnalysis1dim:savedIndex[%d]=%d", i, savedIndex[i])
            val, ts := readRing(aRingBuffer, savedIndex[i])
            dataPoint += `{"value":"` + val + `","ts":"` + ts + `"},`
        }
        dataPoint = dataPoint[:len(dataPoint)-1]
        if (len(savedIndex) > 1) {
            dataPoint += "]"
        }
    } else {
            val, ts := readRing(aRingBuffer, 0)  // return latest sample (= head sample)
            dataPoint += `{"value":"` + val + `","ts":"` + ts + `"}`
    }
    return dataPoint, updatedTail
}

func clReduction(clBuffer []CLBufElement, firstIndex int, lastIndex int, maxError float64) []int {
//utils.Info.Printf("clReduction:firstIndex=%d, lastIndex=%d, maxError=%f, ", firstIndex, lastIndex, maxError)
    if (lastIndex - firstIndex <= 1) {
        return nil
    }
    var maxMeasuredError float64 = 0.0
    indexOfMaxMeasuredError := firstIndex
    var measuredError float64
    
    linearSlope := (clBuffer[lastIndex].Value - clBuffer[firstIndex].Value) / (float64)(clBuffer[lastIndex].Timestamp - clBuffer[firstIndex].Timestamp)
    
    for i := 0 ; i <= lastIndex - firstIndex ; i++ {
        measuredError = clBuffer[firstIndex+i].Value - (clBuffer[firstIndex].Value + linearSlope * (float64)(clBuffer[firstIndex+i].Timestamp - clBuffer[firstIndex].Timestamp))
        if (measuredError < 0) {
            measuredError = -measuredError
        }
        if (measuredError > maxMeasuredError) {
            maxMeasuredError = measuredError
            indexOfMaxMeasuredError = firstIndex + i
        }
    }
    
    if (maxMeasuredError > maxError) {
        var savedIndex1, savedIndex2 []int
        savedIndex1 = append(savedIndex1, clReduction(clBuffer, firstIndex, indexOfMaxMeasuredError, maxError)...)
        savedIndex2 = append(savedIndex2, clReduction(clBuffer, indexOfMaxMeasuredError, lastIndex, maxError)...)
        savedIndex1 = append(savedIndex1, savedIndex2...)
        return append(savedIndex1, indexOfMaxMeasuredError)
    }
    return nil
}

func clCapture2dim(clChan chan CLPack, subscriptionId int, path string, bufSize int, maxError float64) {
}

