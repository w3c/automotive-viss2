/******** Peter Winzell (c), 11/27/23 *********************************************/

package atServer

import (
	"bytes"
	"crypto/rsa"
	"encoding/json"
	"github.com/w3c/automotive-viss2/utils"
	"log"
	_ "log"
	"net/http"
	"strconv"
	"testing"
)

//Prequisites: AGT Server and Vissv2 Server should be up and running (0.0.0.0 in docker)
type AGTResponse struct {
	Token string `json:"token"`
}

const LT_TEST_POP = "eyJ0eXAiOiJkcG9wK2p3dCIsImFsZyI6IkVTMjU2IiwiandrIjp7Imt0eSI6IkVDIiwidXNlIjoic2lnbiIsImNydiI6IlAtMjU2IiwieCI6InRKeDJkcjJKOUZVN1loT21yME9jbTQ2dXMycFFjWTNRcnAxV0RGVTFfYWsiLCJ5IjoiVWdRQnhIRjVUX0xoT28tVmM4RGlmU3NlallKUVd0QTQ2ei1lbmFlazRyVSJ9fQ.eyJhdWQiOiJ2aXNzdjIvYWd0cyIsImlhdCI6IjE2NTUyOTI5MTkiLCJqdGkiOiIzZTZjNmNlMy00YmUyLTQwZmUtYjc5Yi00MzQ2YjBjNmY2MjkifQ.MqM57OE-m1hwyT63aHqHhMu9aMScQBEWQ3B-iG670zvlHIqyvbyVuEB-UhFVdi_pAscSII9FSROhzB9nrWM5sA"
const agt_posttesturl = "http://0.0.0.0:7500/agts"
const SHORT_TERM_TOKEN_LENGTH = 14400
const LONG_TERM_TOKEN_LENGTH = 345600

func getShortTermAGTResponse() (*http.Response, error) {
	body := []byte(`{"vin":"GEO001","context":"Independent+OEM+Cloud","proof":"ABC","key":"DEF"}`)

	r, err := http.NewRequest("POST", agt_posttesturl, bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}
	r.Header.Add("Content-Type", "application/json")
	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
		panic(err)
	}
	return res, err
}

func getLongTermAGTResponse() (*http.Response, error) {
	var privKey *rsa.PrivateKey
	err := utils.ImportRsaKey(PRIV_KEY_DIRECTORY, &privKey)
	if err != nil {
		return nil, err
	}

	popToken := utils.PopToken{}
	token, err := popToken.GenerateToken(privKey) // generate a proof-of-possession-token to get a LT token.

	agtP := agtPayload{
		Vin:     "GEO001",
		Context: "Independent+OEM+Cloud",
		Proof:   "ABC",
		Key:     popToken.Jwk.Thumb, // Thumb print need to match.
	}

	body, err := json.Marshal(agtP) // POST a acces grant token request

	r, err := http.NewRequest("POST", agt_posttesturl, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("PoP", token) // add the Pop as a proof of possesion.

	client := &http.Client{}

	return client.Do(r)
}

func TestShortTermTokenAccess(t *testing.T) {

	/*body := []byte(`{"vin":"GEO001","context":"Independent+OEM+Cloud","proof":"ABC","key":"DEF"}`)

	r, err := http.NewRequest("POST", agt_posttesturl, bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}
	r.Header.Add("Content-Type", "application/json")
	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
		panic(err)
	}*/

	res, err := getShortTermAGTResponse()
	if err != nil {
		t.Error(err)
	}

	defer res.Body.Close()

	post := &AGTResponse{}
	derr := json.NewDecoder(res.Body).Decode(post)
	if derr != nil {
		panic(derr)
	}

	if res.StatusCode != http.StatusCreated {
		t.Error("status code expected to be 201 , got: ", res.StatusCode)
	}

	log.Printf("got token = %s", post.Token)
	var Agt utils.ExtendedJwt
	err = Agt.DecodeFromFull(post.Token) // parsing the JWT token
	vin := Agt.PayloadClaims["vin"]
	ctx := Agt.PayloadClaims["clx"]
	iat, err := strconv.Atoi(Agt.PayloadClaims["iat"])
	exp, err := strconv.Atoi(Agt.PayloadClaims["exp"])

	//test that it is correct length
	if (exp - iat) != SHORT_TERM_TOKEN_LENGTH {
		t.Error("short term token error: ", exp-iat)
	}
	//test vin and context
	if vin != "GEO001" {
		t.Error("Vin does not match => ", vin)
	}
	if ctx != "Independent+OEM+Cloud" {
		t.Error("roles fails to match => ", ctx)
	}

}

const PRIV_KEY_DIRECTORY = "../../agt_server/agt_private_key.rsa"

//TODO this should reside in utils somwhere , duplicate code.
type agtPayload struct {
	Vin     string `json:"vin"`
	Context string `json:"context"`
	Proof   string `json:"proof"`
	//Key     utils.JsonWebKey `json:"key"`
	Key string `json:"key"`
}

func TestLongTermTokenAccess(t *testing.T) {

	var privKey *rsa.PrivateKey
	err := utils.ImportRsaKey(PRIV_KEY_DIRECTORY, &privKey)
	if err != nil {
		t.Error(err)
		return
	}
	popToken := utils.PopToken{}
	token, err := popToken.GenerateToken(privKey) // generate a proof-of-possession-token to get a LT token.

	agtP := agtPayload{
		Vin:     "GEO001",
		Context: "Independent+OEM+Cloud",
		Proof:   "ABC",
		Key:     popToken.Jwk.Thumb, // Thumb print need to match.
	}

	body, err := json.Marshal(agtP) // POST a acces grant token request

	r, err := http.NewRequest("POST", agt_posttesturl, bytes.NewBuffer(body))
	if err != nil {
		panic(err)
	}
	r.Header.Add("Content-Type", "application/json")
	r.Header.Add("PoP", token) // add the Pop as a proof of possesion.

	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
		panic(err)
	}

	defer res.Body.Close()

	post := &AGTResponse{}
	derr := json.NewDecoder(res.Body).Decode(post)
	if derr != nil {
		panic(derr)
	}

	if res.StatusCode != http.StatusCreated {
		t.Error("status code expected to be 201 , got: ", res.StatusCode)
	}

	log.Printf("got token = %s", post.Token)
	var Agt utils.ExtendedJwt
	err = Agt.DecodeFromFull(post.Token) // parsing the JWT token
	vin := Agt.PayloadClaims["vin"]
	ctx := Agt.PayloadClaims["clx"]
	iat, err := strconv.Atoi(Agt.PayloadClaims["iat"])
	exp, err := strconv.Atoi(Agt.PayloadClaims["exp"])

	//test that it is correct length
	if (exp - iat) != LONG_TERM_TOKEN_LENGTH {
		t.Error("short term token error: ", exp-iat)
	}
	//test vin and context
	if vin != "GEO001" {
		t.Error("Vin does not match => ", vin)
	}
	if ctx != "Independent+OEM+Cloud" {
		t.Error("roles fails to match => ", ctx)
	}

}
