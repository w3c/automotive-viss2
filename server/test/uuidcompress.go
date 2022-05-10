/**
* (C) 2020 Mitsubishi Electrics Automotive
* (C) 2019 Geotab Inc
* (C) 2019 Volvo Cars
*
* All files and artifacts in the repository at https://github.com/w3c/automotive-viss2
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"strings"
)


func urlToPath(url string) string{
	return strings.ReplaceAll(url,"/",".")
}

type UuidListElem struct {
	Path string  `json:"path"`
	Uuid string  `json:"uuid"`
}

type UuidList struct {
	Object []UuidListElem `json:"leafuuids"`
}

var uuidList UuidList

func jsonToStructList(jsonList string, elements int) int {
	err := json.Unmarshal([]byte(jsonList), &uuidList)
	if err != nil {
		fmt.Printf("Error unmarshal json=%s\n", err)
		return 0
	}
	return elements
}

func createUuidList(fname string) int {
	data, err := ioutil.ReadFile(fname)
	if err != nil {
		fmt.Printf("Error reading %s: %s\n", fname, err)
		return 0
	}
	elements := strings.Count(string(data), "{") - 1
	return jsonToStructList(string(data), elements)
}

func isUuidMatch(uuid1 string, uuid2 string, uuidlen int) bool {
    if (strings.Compare(uuid1[:uuidlen], uuid2[:uuidlen]) == 0) {
        return true
    }
    return false        
}

func main() {
    var fname string
    fmt.Printf("UUID list file name: ")
    fmt.Scanf("%s", &fname)

    numOfUuids := createUuidList(fname)
    fmt.Printf("UUID list elements=%d\n", numOfUuids)
    fmt.Printf("UUID list elements=%d\n", len(uuidList.Object))
    var i int
    uuidlen := 1
    uuidMatch := false
    for uuidMatch == false {
        for i = 0 ; i < numOfUuids-1 ; i++ {
            for j := i+1 ; j < numOfUuids-1 ; j++ {
                
                if (isUuidMatch(uuidList.Object[j].Uuid, uuidList.Object[i].Uuid, uuidlen) == true) {
                    uuidMatch = true
                    break
                }
            }
            if (uuidMatch == true) {
                break
            }
        }
        if (uuidMatch == true) {
            uuidlen++
            uuidMatch = false
        } else {
            break
        }
    }
    fmt.Printf("Minimum UUID length for uniqueness is %d\n", uuidlen)
}
