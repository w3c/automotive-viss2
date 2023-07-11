/**
* (C) 2020 Geotab Inc
*
* All files and artifacts in the repository at https://github.com/w3c/automotive-viss2
* are licensed under the provisions of the license provided by the LICENSE file in this repository.
*
**/

package atServer

import (
	"crypto/rsa"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	gomodel "github.com/COVESA/vss-tools/binary/go_parser/datamodel"
	golib "github.com/COVESA/vss-tools/binary/go_parser/parserlib"
	"github.com/google/uuid"
	"github.com/w3c/automotive-viss2/utils"
)

var VSSTreeRoot *gomodel.Node_t

// set to MAXFOUNDNODES in cparserlib.h
const MAXFOUNDNODES = 1500

const GAP = 3      // Used for PoP check
const LIFETIME = 5 // Used for PoP check

const theAtSecret = "averysecretkeyvalue2" //not shared
const AGT_PUB_KEY_DIRECTORY = "atServer/agt_public_key.rsa"
const PORT = 8600
const AT_DURATION = 1 * 60 * 60 // 1 hour

var agtKey *rsa.PublicKey

var jtiCache map[string]struct{} // PoPs JTIs that must be refused to not be reused

type NoScopePayload struct {
	Context string `json:"context"`
}

type AtValidatePayload struct {
	Token      string   `json:"token"`
	Paths      []string `json:"paths"`
	Action     string   `json:"action"`
	Validation string   `json:"validation"`
}

type AtGenPayload struct {
	Token   string `json:"token"`
	Purpose string `json:"purpose"`
	Pop     string `json:"pop"`
	Agt     utils.ExtendedJwt
	PopTk   utils.PopToken
}

/***** No needed, as we use utils.JsonWebToken
type AgToken struct {
	Vin      string `json:"vin"`
	Iat      int    `json:"iat"`
	Exp      int    `json:"exp"`
	Context  string `json:"clx"`
	Key      string `json:"pub"`
	Audience string `json:"aud"`
	JwtId    string `json:"jti"`
}
*****/
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
	Path       string
	Permission string
}

var scopeList map[string]interface{}

var sList []ScopeElement

type ScopeElement struct {
	Context  []ContextElement
	NoAccess []string
}

func initVssFile() bool {
	filePath := "vss_vissv2.binary"
	VSSTreeRoot = golib.VSSReadTree(filePath)

	if VSSTreeRoot == nil {
		utils.Error.Println("Tree file not found")
		return false
	}

	return true
}

// Initializes AGT Server public key for AGT checking
func initAgtKey() {
	err := utils.ImportRsaPubKey(AGT_PUB_KEY_DIRECTORY, &agtKey)
	if err != nil {
		utils.Error.Printf("Error importing AGT key: %s", fmt.Sprintf("%v", err))
		return
	}
	utils.Info.Printf("AGT key imported correctly.")
}

func makeAtServerHandler(serverChannel chan string) func(http.ResponseWriter, *http.Request) {
	return func(w http.ResponseWriter, req *http.Request) {
		utils.Info.Printf("atServer:url=%s", req.URL.Path)
		if req.URL.Path != "/ats" {
			http.Error(w, "404 url path not found.", 404)
		} else if req.Method != "POST" {
			//CORS POLICY, necessary for web client
			if req.Method == "OPTIONS" {
				w.Header().Set("Access-Control-Allow-Origin", "*")
				w.Header().Set("Access-Control-Allow-Headers", "PoP")
				w.Header().Set("Access-Control-Allow-Methods", "POST")
				w.Header().Set("Access-Control-Max-Age", "57600")
			} else {
				http.Error(w, "400 bad request method.", 400)
			}
		} else {
			bodyBytes, err := ioutil.ReadAll(req.Body)
			if err != nil {
				http.Error(w, "400 request unreadable.", 400)
			} else {
				utils.Info.Printf("atServer:received POST request=%s", string(bodyBytes))
				serverChannel <- string(bodyBytes) // Sends request to server channel
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

// Initializes the AT Server with the given port
func initAtServer(serverChannel chan string, muxServer *http.ServeMux) {
	utils.Info.Printf("initAtServer(): Starting AT server")
	utils.ReadTransportSecConfig()                        // loads the secure configuration file
	atServerHandler := makeAtServerHandler(serverChannel) // Generates handlers for the AT server
	muxServer.HandleFunc("/ats", atServerHandler)
	// Initializes the AT Server depending on sec configuration
	if utils.SecureConfiguration.TransportSec == "yes" {
		server := http.Server{
			Addr:    ":" + utils.SecureConfiguration.AtsSecPort,
			Handler: muxServer,
			TLSConfig: utils.GetTLSConfig("localhost", "../transport_sec/"+utils.SecureConfiguration.CaSecPath+"Root.CA.crt",
				tls.ClientAuthType(utils.CertOptToInt(utils.SecureConfiguration.ServerCertOpt)), nil),
		}
		utils.Info.Printf("initAtServer():Starting AT Server with TLS on %s/ats", utils.SecureConfiguration.AtsSecPort)
		utils.Info.Printf("HTTPS:CerOpt=%s", utils.SecureConfiguration.ServerCertOpt)
		utils.Error.Fatal(server.ListenAndServeTLS("../transport_sec/"+utils.SecureConfiguration.ServerSecPath+"server.crt",
			"../transport_sec/"+utils.SecureConfiguration.ServerSecPath+"server.key"))
	} else { // No TLSmtvacuc14uma
		utils.Info.Printf("initAtServer():Starting AT Server without TLS on %s/ats", PORT)
		utils.Error.Fatal(http.ListenAndServe(":"+strconv.Itoa(PORT), muxServer))
	}
}

// Generates response depending on the request received
func generateResponse(input string) string {
	if strings.Contains(input, "purpose") { // Purpose request
		return accessTokenResponse(input)
	} else if strings.Contains(input, "context") { // No scope request
		return noScopeResponse(input)
	} else { // AT validation request
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

// Receives a purpose, an action and a set of paths, returns an error code (0 ok)
func validateRequestAccess(purpose string, action string, paths []string) int {
	numOfPaths := len(paths)
	var pathSubList []string
	for i := 0; i < numOfPaths; i++ {
		var searchData []golib.SearchData_t
		numOfWildcardPaths := 1
		if strings.Contains(paths[i], "*") {
			searchData, numOfWildcardPaths = golib.VSSsearchNodes(paths[i], VSSTreeRoot, MAXFOUNDNODES, true, true, 0, nil, nil)
			pathSubList = make([]string, numOfWildcardPaths)
			for j := 0; j < numOfWildcardPaths; j++ {
				pathLen := getPathLen(string(searchData[j].NodePath[:]))
				pathSubList[j] = string(searchData[j].NodePath[:pathLen])
			}
		} else {
			pathSubList = make([]string, 1)
			pathSubList[0] = paths[i]
		}
		for j := 0; j < numOfWildcardPaths; j++ {
			status := validatePurposeAndAccessPermission(purpose, action, pathSubList[j])
			if status != 0 {
				return status
			}
		}
	}
	return 0
}

// Receives a purpose, action and path and checks if the access is allowed
// Returns error code
func validatePurposeAndAccessPermission(purpose string, action string, path string) int {
	for i := 0; i < len(pList); i++ {
		if pList[i].Short == purpose {
			for j := 0; j < len(pList[i].Access); j++ {
				if pList[i].Access[j].Path == path {
					if action == "set" && pList[i].Access[j].Permission == "read-only" {
						return 61
					} else {
						return 0
					}
				}
			}
		}
	}
	return 60
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
		if actorValid[0] && actorValid[1] && actorValid[2] {
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
		if matchingContext(i, context) {
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

// Validates an access token, returns validation message.
// The only validation done is the one regading the Access Token List
func tokenValidationResponse(input string) string {
	var inputMap map[string]interface{}
	err := json.Unmarshal([]byte(input), &inputMap)
	if err != nil {
		utils.Error.Printf("tokenValidationResponse:error input=%s", input)
		return `{"validation":"1"}`
	}
	var atValidatePayload AtValidatePayload
	extractAtValidatePayloadLevel1(inputMap, &atValidatePayload)
	err = utils.VerifyTokenSignature(atValidatePayload.Token, theAtSecret)
	if err != nil {
		utils.Info.Printf("tokenValidationResponse:invalid signature, error= %s, token=%s", err, atValidatePayload.Token)
		return `{"validation":"5"}`
	}
	purpose := utils.ExtractFromToken(atValidatePayload.Token, "scp")
	res := validateRequestAccess(purpose, atValidatePayload.Action, atValidatePayload.Paths)
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
			//			utils.Info.Println(k, "is an array:, len=", strconv.Itoa(len(vv)))
			extractAtValidatePayloadLevel2(vv, atValidatePayload)
		case string:
			//			utils.Info.Println(k, "is a string:")
			if k == "token" {
				atValidatePayload.Token = v.(string)
			} else if k == "action" {
				atValidatePayload.Action = v.(string)
			} else if k == "validation" {
				atValidatePayload.Validation = v.(string)
			} else if k == "paths" {
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
		switch typ := v.(type) {
		case string:
			//			utils.Info.Println(k, "is a string:")
			atValidatePayload.Paths[i] = v.(string)
		default:
			utils.Info.Printf("%d is of an unknown type: %T", k, typ)
		}
		i++
	}
}

// Calls method to check a correct AT request. If all ok, calls AT generator and returns the AT
func accessTokenResponse(input string) string {
	var payload AtGenPayload
	err := json.Unmarshal([]byte(input), &payload) // Unmarshalls the request
	if err != nil {
		utils.Error.Printf("accessTokenResponse:error input=%s", input)
		return `{"error": "Client request malformed"}`
	}
	err = payload.Agt.DecodeFromFull(payload.Token) // Decodes the AGT included in the request
	if err != nil {
		utils.Error.Printf("accessTokenResponse: error decoding token=%s", payload.Token)
		return `{"error":"AGT Malformed"}`
	}
	if payload.Pop != "" { // Checks for POP token and decodes if exists
		err = payload.PopTk.Unmarshal(payload.Pop)
		if err != nil {
			utils.Error.Printf("accessTokenResponse: error decoding pop, error=%s, pop=%s", err, payload.Agt.PayloadClaims["pop"])
			return `{"error":"POP malformed"}`
		}
	}
	valid, errResponse := validateRequest(payload) // Validates the request
	if valid {
		return generateAt(payload) // Generates the at if the request is valid

	}
	return errResponse
}

func validateTokenTimestamps(iat int, exp int) bool {
	now := time.Now()
	if now.Before(time.Unix(int64(iat), 0)) {
		return false
	}
	if now.After(time.Unix(int64(exp), 0)) {
		return false
	}
	return true
}

// *** PURPOSE VALIDATION ***
func validatePurpose(purpose string, context string) bool { // TODO: learn how to code to parse the purpose list, then use it to validate the purpose
	for i := 0; i < len(pList); i++ {
		//utils.Info.Printf("validatePurpose:purposeList[%d].Short=%s", i, pList[i].Short)
		if pList[i].Short == purpose {
			//utils.Info.Printf("validatePurpose:purpose match=%s", pList[i].Short)
			if checkAuthorization(i, context) {
				return true
			}
		}
	}
	return false
}

// Validates the purpose with the context of the client given an index of a purpose in the purpose list
func checkAuthorization(index int, context string) bool {
	//utils.Info.Printf("checkAuthorization:context=%s, len(pList[index].Context)=%d", context, len(pList[index].Context))
	for i := 0; i < len(pList[index].Context); i++ { // Iterates over the different contexts
		actorValid := [3]bool{false, false, false}
		//utils.Info.Printf("checkAuthorization:len(pList[index].Context[%d].Actor)=%d", i, len(pList[index].Context[i].Actor))
		for j := 0; j < len(pList[index].Context[i].Actor); j++ { // Iterates over the actors
			if j > 2 {
				return false // Only three subactors supported
			}
			for k := 0; k < len(pList[index].Context[i].Actor[j].Role); k++ { // Iterates over the roles of the actors
				//utils.Info.Printf("checkAuthorization:getActorRole(%d, context)=%s vs pList[index].Context[%d].Actor[%d].Role[%d])=%s", j, getActorRole(j, context), i, j, k, pList[index].Context[i].Actor[j].Role[k])
				if getActorRole(j, context) == pList[index].Context[i].Actor[j].Role[k] {
					actorValid[j] = true
					break
				}
			}
		}
		if actorValid[0] && actorValid[1] && actorValid[2] {
			return true
		}
	}
	return false
}

// Returns the role of the actor in the context depending on the index (user, app, device)
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

// *** END PURPOSE VALIDATION ***

// Checks vin is included in the list of valid vins: Not implemented
func checkVin(vin string) bool {
	if len(vin) == 0 {
		return true // this can only happen if AG token does not contain VIN, which is OK according to spec
	}
	return true // TODO:should be checked with VIN in tree
}

// Checks if jwt id exist in cache, if it does, return false. If not, it adds it and automatically clear it from cache when it expires
func addCheckJti(jti string) bool {
	if jtiCache == nil { // If map is empty (first time), it doesnt even check, initializes and add
		jtiCache = make(map[string]struct{})
		jtiCache[jti] = struct{}{}
		go deleteJti(jti)
		return true
	}
	if _, ok := jtiCache[jti]; ok { // Check if jti exist in cache
		return false
	}
	// If we get here, it does not exist in cache
	jtiCache[jti] = struct{}{}
	go deleteJti(jti)
	return true
}

func deleteJti(jti string) {
	time.Sleep((GAP + LIFETIME + 5) * time.Second)
	delete(jtiCache, jti)
}

// Validates the Proof of Possession of the client key
func validatePop(payload AtGenPayload) (bool, string) {
	// Check jti
	if !addCheckJti(payload.PopTk.PayloadClaims["jti"]) {
		utils.Error.Printf("validateRequest: JTI used")
		return false, `{"error": "Repeated JTI"}`
	}
	// Check signaure
	if err := payload.PopTk.CheckSignature(); err != nil {
		utils.Info.Printf("validateRequest: Invalid POP signature: %s", err)
		return false, `{"error": "Cannot validate POP signature"}`
	}
	// Check exp: no need, iat will be used instead
	// Check iat
	if ok, cause := payload.PopTk.CheckIat(GAP, LIFETIME); !ok {
		utils.Info.Printf("validateRequest: Invalid POP iat: %s", cause)
		return false, `{"error": "Cannot validate POP iat"}`
	}
	// Check that pub (thumprint) corresponds with pop key
	if ok, _ := payload.PopTk.CheckThumb(payload.Agt.PayloadClaims["pub"]); !ok {
		utils.Info.Printf("validateRequest: PubKey in POP is not same as in AGT")
		return false, `{"error": "Keys in POP and AGToken are not matching"}`
	}
	// Check aud
	if ok, _ := payload.PopTk.CheckAud("vissv2/ats"); !ok {
		utils.Info.Printf("validateRequest: Aud in POP not valid")
		return false, `{"error": "Invalid aud"}`
	}
	//utils.Info.Printf("validateRequest:Proof of possession of key pair failed")
	//return false, `{"error": "Proof of possession of key pair failed"}`
	return true, ""
}

func validateRequest(payload AtGenPayload) (bool, string) {
	if !checkVin(payload.Agt.HeaderClaims["vin"]) {
		utils.Info.Printf("validateRequest:incorrect VIN=%s", payload.Agt.HeaderClaims["vin"])
		return false, `{"error": "Incorrect vehicle identifiction"}`
	}
	// To verify the AG Token signature
	err := payload.Agt.Token.CheckAssymSignature(agtKey)
	if err != nil {
		utils.Info.Printf("validateRequest:invalid signature, error: %s, token:%s", err, payload.Token)
		return false, `{"error": "AG token signature validation failed"}`
	}
	iat, err := strconv.Atoi(payload.Agt.PayloadClaims["iat"])
	if err != nil {
		return false, `{"error": AG token iat timestamp malformed"}`
	}
	exp, err := strconv.Atoi(payload.Agt.PayloadClaims["exp"])
	if err != nil {
		return false, `{"error": "AG token exp timestamp malformed"}`
	}
	if !validateTokenTimestamps(iat, exp) {
		utils.Info.Printf("validateRequest:invalid token timestamps, iat=%d, exp=%d", payload.Agt.PayloadClaims["iat"], payload.Agt.PayloadClaims["exp"])
		return false, `{"error": "AG token timestamp validation failed"}`
	}
	// POP Checking
	if payload.Agt.PayloadClaims["pub"] != "" { // That means the agt is associated with a public key
		if ok, errmsj := validatePop(payload); !ok {
			return ok, errmsj
		}
	}
	if !validatePurpose(payload.Purpose, payload.Agt.PayloadClaims["clx"]) {
		utils.Info.Printf("validateRequest:invalid purpose=%s, context=%s", payload.Purpose, payload.Agt.PayloadClaims["clx"])
		return false, `{"error": "Purpose validation failed"}`
	}
	return true, ""
}

func generateAt(payload AtGenPayload) string {
	unparsedId, err := uuid.NewRandom()
	if err != nil { // Better way to generate uuid than calling an ext program
		utils.Error.Printf("generateAgt:Error generating uuid, err=%s", err)
		return `{"error": "Internal error"}`
	}
	iat := int(time.Now().Unix())
	exp := iat + AT_DURATION // 1 hour
	var jwtoken utils.JsonWebToken
	jwtoken.SetHeader("HS256")
	//jwtoken.AddClaim("vin", AtGenPayload.Agt.Vin)
	jwtoken.AddClaim("iat", strconv.Itoa(iat))
	jwtoken.AddClaim("exp", strconv.Itoa(exp))
	jwtoken.AddClaim("scp", payload.Purpose)
	jwtoken.AddClaim("clx", payload.Agt.PayloadClaims["clx"])
	jwtoken.AddClaim("aud", "w3org/gen2")
	jwtoken.AddClaim("jti", unparsedId.String())
	utils.Info.Printf("generateAt:jwtHeader=%s", jwtoken.GetHeader())
	utils.Info.Printf("generateAt:jwtPayload=%s", jwtoken.GetPayload())
	jwtoken.Encode()
	jwtoken.SymmSign(theAtSecret)
	return `{"token":"` + jwtoken.GetFullToken() + `"}`
}

func initPurposelist() {
	data, err := ioutil.ReadFile("atServer/purposelist.json")
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
			//			utils.Info.Println(k, "is an array:, len=", strconv.Itoa(len(vv)))
			extractPurposeElementsLevel2(vv)
		case map[string]interface{}:
			//			utils.Info.Println(k, "is a map:")
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
			//			utils.Info.Println(k, "is a map:")
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
			//			utils.Info.Println(k, "is string", vv)
			if k == "short" {
				pList[index].Short = vv
			} else {
				pList[index].Long = vv
			}
		case []interface{}:
			//			utils.Info.Println(k, "is an array:, len=", strconv.Itoa(len(vv)))
			if k == "contexts" {
				pList[index].Context = make([]ContextElement, len(vv))
				extractPurposeElementsL4ContextL1(index, vv)
			} else {
				pList[index].Access = make([]AccessElement, len(vv))
				extractPurposeElementsL4SignalAccessL1(index, vv)
			}
		case map[string]interface{}:
			//			utils.Info.Println(k, "is a map:")
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
			//			utils.Info.Println(k, "is a map:")
			extractPurposeElementsL4ContextL2(k, index, vv)
		default:
			utils.Info.Println(k, "is of an unknown type")
		}
	}
}

func extractPurposeElementsL4ContextL2(k int, index int, contextElem map[string]interface{}) {
	for i, u := range contextElem {
		//		utils.Info.Println(i, u)
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
			for _, t := range vvv {
				//				utils.Info.Println(l, t)
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
					//					utils.Info.Println(k, "is of an unknown type")
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
			//			utils.Info.Println(k, "is a map:")
			extractPurposeElementsL4SignalAccessL2(k, index, vv)
		default:
			utils.Info.Println(k, "is of an unknown type")
		}
	}
}

func extractPurposeElementsL4SignalAccessL2(k int, index int, accessElem map[string]interface{}) {
	for i, u := range accessElem {
		//		utils.Info.Println(i, u)
		if i == "path" {
			pList[index].Access[k].Path = u.(string)
		} else {
			pList[index].Access[k].Permission = u.(string)
		}
	}
}

func initScopeList() {
	data, err := ioutil.ReadFile("atServer/scopelist.json")
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
			//			utils.Info.Println(k, "is an array:, len=", strconv.Itoa(len(vv)))
			extractScopeElementsLevel2(vv)
		case map[string]interface{}:
			//			utils.Info.Println(k, "is a map:")
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
			//			utils.Info.Println(k, "is a map:")
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
			//			utils.Info.Println(k, "is an array:, len=", strconv.Itoa(len(vv)))
			if k == "contexts" {
				sList[index].Context = make([]ContextElement, len(vv))
				extractScopeElementsL4ContextL1(index, vv)
			} else {
				sList[index].NoAccess = make([]string, len(vv))
				extractScopeElementsL4NoAccessL1(index, vv)
			}
		case map[string]interface{}:
			//			utils.Info.Println(k, "is a map:")
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
			//			utils.Info.Println(k, "is a map:")
			extractScopeElementsL4ContextL2(k, index, vv)
		default:
			utils.Info.Println(k, "is of an unknown type")
		}
	}
}

func extractScopeElementsL4ContextL2(k int, index int, contextElem map[string]interface{}) {
	for i, u := range contextElem {
		//		utils.Info.Println(i, u)
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
			for _, t := range vvv {
				//				utils.Info.Println(l, t)
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
			//			utils.Info.Println(vv)
			sList[index].NoAccess[k] = vv
		default:
			utils.Info.Println(k, "is of an unknown type")
		}
	}
}

func AtServerInit() {
	serverChan := make(chan string)
	muxServer := http.NewServeMux()

	initPurposelist()
	initScopeList()
	initVssFile()
	initAgtKey()

	go initAtServer(serverChan, muxServer)

	for {
		request := <-serverChan
		response := generateResponse(request)
		utils.Info.Printf("atServer response=%s", response)
		serverChan <- response
	}
}
