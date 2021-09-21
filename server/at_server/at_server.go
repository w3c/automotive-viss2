/**
* (C) 2020 Geotab Inc
*
* All files and artifacts in the repository at https://github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
	"unsafe"

	"github.com/MEAE-GOT/W3C_VehicleSignalInterfaceImpl/utils"
	"github.com/akamensky/argparse"
)

// #include <stdlib.h>
// #include <stdint.h>
// #include <stdio.h>
// #include <stdbool.h>
// #include "cparserlib.h"
import "C"

var VSSTreeRoot C.long

// set to MAXFOUNDNODES in cparserlib.h
const MAXFOUNDNODES = 1500

type searchData_t struct { // searchData_t defined in cparserlib.h
	path            [512]byte // cparserlib.h: #define MAXCHARSPATH 512; typedef char path_t[MAXCHARSPATH];
	foundNodeHandle int64     // defined as long in cparserlib.h
}

const theAgtSecret = "averysecretkeyvalue1" //shared with agt-server
const theAtSecret = "averysecretkeyvalue2"  //not shared

type NoScopePayload struct {
	Context string `json:"context"`
}

type AtValidatePayload struct {
	Token      string `json:"token"`
	Paths      []string `json:"paths"`
	Action     string `json:"action"`
	Validation string `json:"validation"`
}

type AtGenPayload struct {
	Token   string `json:"token"`
	Purpose string `json:"purpose"`
	Pop     string `json:"pop"`
}

type AgToken struct {
	Vin      string `json:"vin"`
	Iat      int    `json:"iat"`
	Exp      int    `json:"exp"`
	Context  string `json:"clx"`
	Key      string `json:"pub"`
	Audience string `json:"aud"`
	JwtId    string `json:"jti"`
}

var purposeList map[string]interface{}

var pList []PurposeElement

type PurposeElement struct {
	Short   string
	Long    string
	Context []ContextElement
	Access  []AccessElement
}

type ContextElement struct {
	Actor [3]RoleElement // User, App, Device
}

type RoleElement struct {
	Role []string
}

type AccessElement struct {
	Path string
	Mode string
}

var scopeList map[string]interface{}

var sList []ScopeElement

type ScopeElement struct {
	Context  []ContextElement
	NoAccess []string
}

func initVssFile() bool {
	filePath := "vss_vissv2.binary"
	cfilePath := C.CString(filePath)
	VSSTreeRoot = C.VSSReadTree(cfilePath)
	C.free(unsafe.Pointer(cfilePath))

	if VSSTreeRoot == 0 {
		utils.Error.Println("Tree file not found")
		return false
	}

	return true
}

func makeAtServerHandler(serverChannel chan string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		utils.Info.Printf("atServer:url=%s", req.URL.Path)
		if req.URL.Path != "/atserver" {
			http.Error(w, "404 url path not found.", 404)
		} else if req.Method != "POST" {
			http.Error(w, "400 bad request method.", 400)
		} else {
			bodyBytes, err := ioutil.ReadAll(req.Body)
			if err != nil {
				http.Error(w, "400 request unreadable.", 400)
			} else {
				utils.Info.Printf("atServer:received POST request=%s", string(bodyBytes))
				serverChannel <- string(bodyBytes)
				response := <-serverChannel
				utils.Info.Printf("atServer:POST response=%s", response)
				if len(response) == 0 {
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

func initAtServer(serverChannel chan string, muxServer *http.ServeMux) {
	utils.Info.Printf("initAtServer(): :8600/atserver")
	atServerHandler := makeAtServerHandler(serverChannel)
	muxServer.HandleFunc("/atserver", atServerHandler)
	utils.Error.Fatal(http.ListenAndServe(":8600", muxServer))
}

func generateResponse(input string) string {
	if strings.Contains(input, "purpose") == true {
		return accessTokenResponse(input)
	} else if strings.Contains(input, "context") == true {
		return noScopeResponse(input)
	} else {
		return tokenValidationResponse(input)
	}
}

func getPathLen(path string) int {
	for i := 0; i < len(path); i++ {
		if path[i] == 0x00 { // the path buffer defined in searchData_t is initiated with all zeros
			return i
		}
	}
	return len(path)
}

func validateRequestAccess(scope string, action string, paths []string) int {
	numOfPaths := len(paths)
	var pathSubList []string
	for i := 0; i < numOfPaths; i++ {
		numOfWildcardPaths := 1
		if strings.Contains(paths[i], "*") == true {
			searchData := [MAXFOUNDNODES]searchData_t{}
			// call int VSSSearchNodes(char* searchPath, long rootNode, int maxFound, searchData_t* searchData, bool anyDepth, bool leafNodesOnly, int listSize, noScopeList_t* noScopeList, int* validation);
			cpath := C.CString(paths[i])
			numOfWildcardPaths := int(C.VSSSearchNodes(cpath, VSSTreeRoot, MAXFOUNDNODES, (*C.struct_searchData_t)(unsafe.Pointer(&searchData)), true, true, 0, nil, nil))
			C.free(unsafe.Pointer(cpath))
			pathSubList = make([]string, numOfWildcardPaths)
			for j := 0; j < numOfWildcardPaths; j++ {
				pathLen := getPathLen(string(searchData[j].path[:]))
				pathSubList[j] = string(searchData[j].path[:pathLen])
			}
		} else {
			pathSubList = make([]string, 1)
			pathSubList[0] = paths[i]
		}
		for j := 0; j < numOfWildcardPaths; j++ {
			status := validateScopeAndAccessMode(scope, action, pathSubList[j])
			if status != 0 {
				return status
			}
		}
	}
	return 0
}

func validateScopeAndAccessMode(scope string, action string, path string) int {
	for i := 0; i < len(pList); i++ {
		if pList[i].Short == scope {
			for j := 0; j < len(pList[i].Access); j++ {
				if pList[i].Access[j].Path == path {
					if action == "set" && pList[i].Access[j].Mode == "read-only" {
						return -16
					} else {
						return 0
					}
				}
			}
		}
	}
	return -8
}

func matchingContext(index int, context string) bool { // identical to checkAuthorization(), using sList instead of pList
	for i := 0; i < len(sList[index].Context); i++ {
		actorValid := [3]bool{false, false, false}
		for j := 0; j < len(sList[index].Context[i].Actor); j++ {
			if j > 2 {
				return false // only three subactors supported
			}
			for k := 0; k < len(sList[index].Context[i].Actor[j].Role); k++ {
				if getActorRole(j, context) == sList[index].Context[i].Actor[j].Role[k] {
					actorValid[j] = true
					break
				}
			}
		}
		if actorValid[0] == true && actorValid[1] == true && actorValid[2] == true {
			return true
		}
	}
	return false
}

func synthesizeNoScope(index int) string {
	if len(sList[index].NoAccess) == 1 {
		return `"` + sList[index].NoAccess[0] + `"`
	}
	noScope := "["
	for i := 0; i < len(sList[index].NoAccess); i++ {
		noScope += `"` + sList[index].NoAccess[i] + `", `
	}
	return noScope[:len(noScope)-2] + "]"
}

func getNoAccessScope(context string) string {
	for i := 0; i < len(sList); i++ {
		if matchingContext(i, context) == true {
			return synthesizeNoScope(i)
		}
	}
	return `""`
}

func noScopeResponse(input string) string {
	var payload NoScopePayload
	err := json.Unmarshal([]byte(input), &payload)
	if err != nil {
		utils.Error.Printf("noScopeResponse:error input=%s", input)
		return `{"no_access":""}`
	}
	res := getNoAccessScope(payload.Context)
	utils.Info.Printf("getNoAccessScope result=%s", res)
	return `{"no_access":` + res + `}`
}

func tokenValidationResponse(input string) string {
	var inputMap map[string]interface{}
	err := json.Unmarshal([]byte(input), &inputMap)
	if err != nil {
		utils.Error.Printf("tokenValidationResponse:error input=%s", input)
		return `{"validation":"-128"}`
	}
	var atValidatePayload AtValidatePayload
	extractAtValidatePayloadLevel1(inputMap, &atValidatePayload)
	if utils.VerifyTokenSignature(atValidatePayload.Token, theAtSecret) == false {
		utils.Info.Printf("tokenValidationResponse:invalid signature=%s", atValidatePayload.Token)
		return `{"validation":"-2"}`
	}
	scope := utils.ExtractFromToken(atValidatePayload.Token, "scp")
	res := validateRequestAccess(scope, atValidatePayload.Action, atValidatePayload.Paths)
	if res != 0 {
		utils.Info.Printf("validateRequestAccess fails with result=%d", res)
		return `{"validation":"` + strconv.Itoa(res) + `"}`
	}
	return `{"validation":"0"}`
}

func extractAtValidatePayloadLevel1(atValidateMap map[string]interface{}, atValidatePayload *AtValidatePayload) {
	for k, v := range atValidateMap {
		switch vv := v.(type) {
		case []interface{}:
			utils.Info.Println(k, "is an array:, len=", strconv.Itoa(len(vv)))
			extractAtValidatePayloadLevel2(vv, atValidatePayload)
		case string:
			utils.Info.Println(k, "is a string:")
			if (k == "token") {
			    atValidatePayload.Token = v.(string)
			} else if (k == "action") {
			    atValidatePayload.Action = v.(string)
			} else if (k == "validation") {
			    atValidatePayload.Validation = v.(string)
			} else if (k == "paths") {
			    atValidatePayload.Paths = make([]string, 1)
			    atValidatePayload.Paths[0] = v.(string)
			}
		default:
			utils.Info.Println(k, "is of an unknown type")
		}
	}
}

func extractAtValidatePayloadLevel2(pathList []interface{}, atValidatePayload *AtValidatePayload) {
	atValidatePayload.Paths = make([]string, len(pathList))
	i := 0
	for k, v := range pathList {
		switch v.(type) {
		case string:
			utils.Info.Println(k, "is a string:")
			atValidatePayload.Paths[i] = v.(string)
		default:
			utils.Info.Println(k, "is of an unknown type")
		}
		i++
	}
}

func accessTokenResponse(input string) string {
	var payload AtGenPayload
	err := json.Unmarshal([]byte(input), &payload)
	if err != nil {
		utils.Error.Printf("accessTokenResponse:error input=%s", input)
		return `{"error": "Client request malformed"}`
	}
	agToken, errResp := extractTokenPayload(payload.Token)
	if len(errResp) > 0 {
		return errResp
	}
	ok, errResponse := validateRequest(payload, agToken)
	if ok == true {
		return generateAt(payload, agToken.Context)
	}
	return errResponse
}

func validateTokenTimestamps(iat int, exp int) bool {
	now := time.Now()
	if now.Before(time.Unix(int64(iat), 0)) == true {
		return false
	}
	if now.After(time.Unix(int64(exp), 0)) == true {
		return false
	}
	return true
}

func validatePurpose(purpose string, context string) bool { // TODO: learn how to code to parse the purpose list, then use it to validate the purpose
	valid := false
	for i := 0; i < len(pList); i++ {
		//utils.Info.Printf("validatePurpose:purposeList[%d].Short=%s", i, pList[i].Short)
		if pList[i].Short == purpose {
			//utils.Info.Printf("validatePurpose:purpose match=%s", pList[i].Short)
			valid = checkAuthorization(i, context)
			if valid == true {
				break
			}
		}
	}
	return valid
}

func checkAuthorization(index int, context string) bool {
	//utils.Info.Printf("checkAuthorization:context=%s, len(pList[index].Context)=%d", context, len(pList[index].Context))
	for i := 0; i < len(pList[index].Context); i++ {
		actorValid := [3]bool{false, false, false}
		//utils.Info.Printf("checkAuthorization:len(pList[index].Context[%d].Actor)=%d", i, len(pList[index].Context[i].Actor))
		for j := 0; j < len(pList[index].Context[i].Actor); j++ {
			if j > 2 {
				return false // only three subactors supported
			}
			for k := 0; k < len(pList[index].Context[i].Actor[j].Role); k++ {
				//utils.Info.Printf("checkAuthorization:getActorRole(%d, context)=%s vs pList[index].Context[%d].Actor[%d].Role[%d])=%s", j, getActorRole(j, context), i, j, k, pList[index].Context[i].Actor[j].Role[k])
				if getActorRole(j, context) == pList[index].Context[i].Actor[j].Role[k] {
					actorValid[j] = true
					break
				}
			}
		}
		if actorValid[0] == true && actorValid[1] == true && actorValid[2] == true {
			return true
		}
	}
	return false
}

func getActorRole(actorIndex int, context string) string {
	delimiter1 := strings.Index(context, "+")
	if actorIndex == 0 {
		return context[:delimiter1]
	}
	delimiter2 := strings.Index(context[delimiter1+1:], "+")
	if actorIndex == 1 {
		return context[delimiter1+1 : delimiter1+1+delimiter2]
	}
	return context[delimiter1+1+delimiter2+1:]
}

func decodeTokenPayload(token string) string {
	delim1 := strings.Index(token, ".")
	delim2 := delim1 + 1 + strings.Index(token[delim1+1:], ".")
	pload := token[delim1+1 : delim1+1+delim2]
	payload, _ := base64.RawURLEncoding.DecodeString(pload)
	//utils.Info.Printf("decodeTokenPayload:payload=%s", string(payload))
	return string(payload)
}

func extractTokenPayload(token string) (AgToken, string) {
	var agToken AgToken
	tokenPayload := decodeTokenPayload(token)
	err := json.Unmarshal([]byte(tokenPayload), &agToken)
	if err != nil {
		utils.Error.Printf("extractTokenPayload:token payload=%s, error=%s", tokenPayload, err)
		return agToken, `{"error": "AG token malformed"}`
	}
	return agToken, ""
}

func checkVin(vin string) bool {
	return true // should be checked with VIN in tree
}

func validateRequest(payload AtGenPayload, agToken AgToken) (bool, string) {
	if checkVin(agToken.Vin) == false {
		utils.Info.Printf("validateRequest:incorrect VIN=%s", agToken.Vin)
		return false, `{"error": "Incorrect vehicle identifiction"}`
	}
	if utils.VerifyTokenSignature(payload.Token, theAgtSecret) == false {
		utils.Info.Printf("validateRequest:invalid signature=%s", payload.Token)
		return false, `{"error": "AG token signature validation failed"}`
	}
	if validateTokenTimestamps(agToken.Iat, agToken.Exp) == false {
		utils.Info.Printf("validateRequest:invalid token timestamps, iat=%d, exp=%d", agToken.Iat, agToken.Exp)
		return false, `{"error": "AG token timestamp validation failed"}`
	}
	if len(agToken.Key) != 0 && payload.Pop != "GHI" { // PoP should be a signed timestamp
		utils.Info.Printf("validateRequest:Proof of possession of key pair failed")
		return false, `{"error": "Proof of possession of key pair failed"}`
	}
	if validatePurpose(payload.Purpose, agToken.Context) == false {
		utils.Info.Printf("validateRequest:invalid purpose=%s, context=%s", payload, agToken.Context)
		return false, `{"error": "Purpose validation failed"}`
	}
	return true, ""
}

func generateAt(payload AtGenPayload, context string) string {
	uuid, err := exec.Command("uuidgen").Output()
	if err != nil {
		utils.Error.Printf("generateAt:Error generating uuid, err=%s", err)
		return `{"error": "Internal error"}`
	}
	uuid = uuid[:len(uuid)-1] // remove '\n' char
	iat := int(time.Now().Unix())
	exp := iat + 1*60*60 // 1 hour
	jwtHeader := `{"alg":"ES256","typ":"JWT"}`
	jwtPayload := `{"iat":` + strconv.Itoa(iat) + `,"exp":` + strconv.Itoa(exp) + `,"scp":"` + payload.Purpose + `"` + `,"clx":"` + context +
		`","aud": "w3.org/gen2","jti":"` + string(uuid) + `"}`
	utils.Info.Printf("generateAt:jwtHeader=%s", jwtHeader)
	utils.Info.Printf("generateAt:jwtPayload=%s", jwtPayload)
	encodedJwtHeader := base64.RawURLEncoding.EncodeToString([]byte(jwtHeader))
	encodedJwtPayload := base64.RawURLEncoding.EncodeToString([]byte(jwtPayload))
	utils.Info.Printf("generateAt:encodedJwtHeader=%s", encodedJwtHeader)
	jwtSignature := utils.GenerateHmac(encodedJwtHeader+"."+encodedJwtPayload, theAtSecret)
	encodedJwtSignature := base64.RawURLEncoding.EncodeToString([]byte(jwtSignature))
	return `{"token":"` + encodedJwtHeader + "." + encodedJwtPayload + "." + encodedJwtSignature + `"}`
}

func initPurposelist() {
	data, err := ioutil.ReadFile("purposelist.json")
	if err != nil {
		utils.Error.Printf("Error reading purposelist.json\n")
		os.Exit(-1)
	}
	err = json.Unmarshal([]byte(data), &purposeList)
	if err != nil {
		utils.Error.Printf("initPurposelist:error data=%s, err=%s", data, err)
		os.Exit(-1)
	}
	extractPurposeElementsLevel1(purposeList)
}

func extractPurposeElementsLevel1(purposeList map[string]interface{}) {
	for k, v := range purposeList {
		switch vv := v.(type) {
		case []interface{}:
			utils.Info.Println(k, "is an array:, len=", strconv.Itoa(len(vv)))
			extractPurposeElementsLevel2(vv)
		case map[string]interface{}:
			utils.Info.Println(k, "is a map:")
			extractPurposeElementsLevel3(0, vv)
		default:
			utils.Info.Println(k, "is of an unknown type")
		}
	}
}

func extractPurposeElementsLevel2(purposeList []interface{}) {
	pList = make([]PurposeElement, len(purposeList))
	i := 0
	for k, v := range purposeList {
		switch vv := v.(type) {
		case map[string]interface{}:
			utils.Info.Println(k, "is a map:")
			extractPurposeElementsLevel3(i, vv)
		default:
			utils.Info.Println(k, "is of an unknown type")
		}
		i++
	}
}

func extractPurposeElementsLevel3(index int, purposeElem map[string]interface{}) {
	for k, v := range purposeElem {
		switch vv := v.(type) {
		case string:
			utils.Info.Println(k, "is string", vv)
			if k == "short" {
				pList[index].Short = vv
			} else {
				pList[index].Long = vv
			}
		case []interface{}:
			utils.Info.Println(k, "is an array:, len=", strconv.Itoa(len(vv)))
			if k == "contexts" {
				pList[index].Context = make([]ContextElement, len(vv))
				extractPurposeElementsL4ContextL1(index, vv)
			} else {
				pList[index].Access = make([]AccessElement, len(vv))
				extractPurposeElementsL4SignalAccessL1(index, vv)
			}
		case map[string]interface{}:
			utils.Info.Println(k, "is a map:")
			if k == "contexts" {
				pList[index].Context = make([]ContextElement, 1)
				extractPurposeElementsL4ContextL2(0, index, vv)
			} else {
				pList[index].Access = make([]AccessElement, 1)
				extractPurposeElementsL4SignalAccessL2(0, index, vv)
			}
		default:
			utils.Info.Println(k, "is of an unknown type")
		}
	}
}

func extractPurposeElementsL4ContextL1(index int, contextElem []interface{}) {
	for k, v := range contextElem {
		switch vv := v.(type) {
		case map[string]interface{}:
			utils.Info.Println(k, "is a map:")
			extractPurposeElementsL4ContextL2(k, index, vv)
		default:
			utils.Info.Println(k, "is of an unknown type")
		}
	}
}

func extractPurposeElementsL4ContextL2(k int, index int, contextElem map[string]interface{}) {
	for i, u := range contextElem {
		utils.Info.Println(i, u)
		switch vvv := u.(type) {
		case string:
			if i == "user" {
				pList[index].Context[k].Actor[0].Role = make([]string, 1)
				pList[index].Context[k].Actor[0].Role[0] = u.(string)
			} else if i == "app" {
				pList[index].Context[k].Actor[1].Role = make([]string, 1)
				pList[index].Context[k].Actor[1].Role[0] = u.(string)
			} else {
				pList[index].Context[k].Actor[2].Role = make([]string, 1)
				pList[index].Context[k].Actor[2].Role[0] = u.(string)
			}
		case []interface{}:
			m := 0
			for l, t := range vvv {
				utils.Info.Println(l, t)
				switch t.(type) {
				case string:
					if i == "user" {
						if m == 0 {
							pList[index].Context[k].Actor[0].Role = make([]string, len(vvv))
						}
						pList[index].Context[k].Actor[0].Role[m] = t.(string)
					} else if i == "app" {
						if m == 0 {
							pList[index].Context[k].Actor[1].Role = make([]string, len(vvv))
						}
						pList[index].Context[k].Actor[1].Role[m] = t.(string)
					} else {
						if m == 0 {
							pList[index].Context[k].Actor[2].Role = make([]string, len(vvv))
						}
						pList[index].Context[k].Actor[2].Role[m] = t.(string)
					}
				default:
					utils.Info.Println(k, "is of an unknown type")
				}
				m++
			}
		default:
			utils.Info.Println(k, "is of an unknown type")
		}
	}
}

func extractPurposeElementsL4SignalAccessL1(index int, accessElem []interface{}) {
	for k, v := range accessElem {
		switch vv := v.(type) {
		case map[string]interface{}:
			utils.Info.Println(k, "is a map:")
			extractPurposeElementsL4SignalAccessL2(k, index, vv)
		default:
			utils.Info.Println(k, "is of an unknown type")
		}
	}
}

func extractPurposeElementsL4SignalAccessL2(k int, index int, accessElem map[string]interface{}) {
	for i, u := range accessElem {
		utils.Info.Println(i, u)
		if i == "path" {
			pList[index].Access[k].Path = u.(string)
		} else {
			pList[index].Access[k].Mode = u.(string)
		}
	}
}

func initScopeList() {
	data, err := ioutil.ReadFile("scopelist.json")
	if err != nil {
		utils.Info.Printf("scopelist.json not found")
		return
	}
	err = json.Unmarshal([]byte(data), &scopeList)
	if err != nil {
		utils.Error.Printf("initScopeList:error data=%s, err=%s", data, err)
		os.Exit(-1)
	}
	extractScopeElementsLevel1(scopeList)
}

func extractScopeElementsLevel1(scopeList map[string]interface{}) {
	for k, v := range scopeList {
		switch vv := v.(type) {
		case []interface{}:
			utils.Info.Println(k, "is an array:, len=", strconv.Itoa(len(vv)))
			extractScopeElementsLevel2(vv)
		case map[string]interface{}:
			utils.Info.Println(k, "is a map:")
			extractScopeElementsLevel3(0, vv)
		default:
			utils.Info.Println(k, "is of an unknown type")
		}
	}
}

func extractScopeElementsLevel2(scopeList []interface{}) {
	sList = make([]ScopeElement, len(scopeList))
	i := 0
	for k, v := range scopeList {
		switch vv := v.(type) {
		case map[string]interface{}:
			utils.Info.Println(k, "is a map:")
			extractScopeElementsLevel3(i, vv)
		default:
			utils.Info.Println(k, "is of an unknown type")
		}
		i++
	}
}

func extractScopeElementsLevel3(index int, scopeElem map[string]interface{}) {
	for k, v := range scopeElem {
		switch vv := v.(type) {
		case string:
			sList[index].NoAccess = make([]string, 1)
			sList[index].NoAccess[0] = vv
		case []interface{}:
			utils.Info.Println(k, "is an array:, len=", strconv.Itoa(len(vv)))
			if k == "contexts" {
				sList[index].Context = make([]ContextElement, len(vv))
				extractScopeElementsL4ContextL1(index, vv)
			} else {
				sList[index].NoAccess = make([]string, len(vv))
				extractScopeElementsL4NoAccessL1(index, vv)
			}
		case map[string]interface{}:
			utils.Info.Println(k, "is a map:")
			sList[index].Context = make([]ContextElement, 1)
			extractScopeElementsL4ContextL2(0, index, vv)
		default:
			utils.Info.Println(k, "is of an unknown type")
		}
	}
}

func extractScopeElementsL4ContextL1(index int, contextElem []interface{}) {
	for k, v := range contextElem {
		switch vv := v.(type) {
		case map[string]interface{}:
			utils.Info.Println(k, "is a map:")
			extractScopeElementsL4ContextL2(k, index, vv)
		default:
			utils.Info.Println(k, "is of an unknown type")
		}
	}
}

func extractScopeElementsL4ContextL2(k int, index int, contextElem map[string]interface{}) {
	for i, u := range contextElem {
		utils.Info.Println(i, u)
		switch vvv := u.(type) {
		case string:
			if i == "user" {
				sList[index].Context[k].Actor[0].Role = make([]string, 1)
				sList[index].Context[k].Actor[0].Role[0] = u.(string)
			} else if i == "app" {
				sList[index].Context[k].Actor[1].Role = make([]string, 1)
				sList[index].Context[k].Actor[1].Role[0] = u.(string)
			} else {
				sList[index].Context[k].Actor[2].Role = make([]string, 1)
				sList[index].Context[k].Actor[2].Role[0] = u.(string)
			}
		case []interface{}:
			m := 0
			for l, t := range vvv {
				utils.Info.Println(l, t)
				switch t.(type) {
				case string:
					if i == "user" {
						if m == 0 {
							sList[index].Context[k].Actor[0].Role = make([]string, len(vvv))
						}
						sList[index].Context[k].Actor[0].Role[m] = t.(string)
					} else if i == "app" {
						if m == 0 {
							sList[index].Context[k].Actor[1].Role = make([]string, len(vvv))
						}
						sList[index].Context[k].Actor[1].Role[m] = t.(string)
					} else {
						if m == 0 {
							sList[index].Context[k].Actor[2].Role = make([]string, len(vvv))
						}
						sList[index].Context[k].Actor[2].Role[m] = t.(string)
					}
				default:
					utils.Info.Println(k, "is of an unknown type")
				}
				m++
			}
		default:
			utils.Info.Println(k, "is of an unknown type")
		}
	}
}

func extractScopeElementsL4NoAccessL1(index int, noAccessElem []interface{}) {
	for k, v := range noAccessElem {
		switch vv := v.(type) {
		case string:
			utils.Info.Println(vv)
			sList[index].NoAccess[k] = vv
		default:
			utils.Info.Println(k, "is of an unknown type")
		}
	}
}

func main() {
	// Create new parser object
	parser := argparse.NewParser("print", "AT Server")
	// Create string flag
	logFile := parser.Flag("", "logfile", &argparse.Options{Required: false, Help: "outputs to logfile in ./logs folder"})
	logLevel := parser.Selector("", "loglevel", []string{"trace", "debug", "info", "warn", "error", "fatal", "panic"}, &argparse.Options{
		Required: false,
		Help:     "changes log output level",
		Default:  "info"})

	// Parse input
	err := parser.Parse(os.Args)
	if err != nil {
		fmt.Print(parser.Usage(err))
	}

	serverChan := make(chan string)
	muxServer := http.NewServeMux()

	utils.InitLog("atserver-log.txt", "./logs", *logFile, *logLevel)
	initPurposelist()
	initScopeList()
	initVssFile()

	go initAtServer(serverChan, muxServer)

	for {
		select {
		case request := <-serverChan:
			response := generateResponse(request)
			utils.Info.Printf("atServer response=%s", response)
			serverChan <- response
		}
	}
}
