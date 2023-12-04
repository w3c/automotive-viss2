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

const agt_posttesturl = "http://0.0.0.0:7500/agts"
const at_url = "http://localhost:8600/"
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

	res, err := getLongTermAGTResponse()
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

type atRequest struct {
	Token   string `json:string "token"`
	Purpose string `json:string "purpose"`
	Pop     string `json:string "pop"`
}

func getAtToken(agttoken string) (*http.Response, error) {

	atReq := &atRequest{
		Token:   agttoken,
		Purpose: "fuel-status",
		Pop:     "GHI",
	}
	body, err := json.Marshal(atReq)
	if err != nil {
		return nil, err
	}

	r, err := http.NewRequest("POST", at_url, bytes.NewBuffer(body))
	if err != nil {
		return nil, err
	}

	r.Header.Add("Content-Type", "application/json")
	client := &http.Client{}
	res, err := client.Do(r)
	if err != nil {
		return nil, err
	}
	return res, err
}

func parseATToken(res http.Response, t *testing.T) *AGTResponse {

	defer res.Body.Close()

	post := &AGTResponse{}
	derr := json.NewDecoder(res.Body).Decode(post)
	if derr != nil {
		t.Error("could not parse http response")
		return nil
	}

	if res.StatusCode != http.StatusCreated {
		t.Error("status code expected to be 201 , got: ", res.StatusCode)

	}

	return post
}

//
func TestAtTokenAccess_ST(t *testing.T) {
	res_ag, err := getShortTermAGTResponse()
	if err != nil {
		t.Error(err)
	}

	res, err := getAtToken(parseATToken(*res_ag, t).Token)

	if err != nil {
		t.Error(err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Error("status code expected to be 201 , got: ", res.StatusCode)
	} else {
		attokenpost := &AGTResponse{}
		derr := json.NewDecoder(res.Body).Decode(attokenpost)

		if derr != nil {
			t.Error(derr)
		}
		log.Println(attokenpost.Token)
	}

}

func TestAtTokenAccess_LT(t *testing.T) {
	t.Error("not implemented yet")

}

func TestGetAccessControlST(t *testing.T) {
	t.Error("not implemented yet")
}

func TestGetAccessControlLT(t *testing.T) {
	t.Error("not implemented yet")
}
