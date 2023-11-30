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

func TestShortTermTokenAccess(t *testing.T) {

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

const pub_key = "MIIBCgKCAQEAp1CqpdSa9mpi6ERtK1bf8o84GMfyZa+F5j/lCLEwOU2d+12rfp+8\n1BV3xXdnJ+N0nzU3NkUmTdX/Omkd+9si8JcIykmcb4sR3QeVCd+vdIEcBIdoWAT+\nc9j+z2EUmG0rNk1gynxkJbb7Owr4cancNH3kQDWWGveZ9lsDijOI9o4pB4CKdWc0\nRJObz4e4DpzOJT07qnnWVI9WueEoCheZNkDosrb4MmLnHeDEStySNeAuX3BO6CSH\nujcXYwOoA6wKHPtYyvgPeS8SHdpzf1YYBuHoesha4E0O0PacFZV0TTJaY4+nS4Mn\n+sRcdyv5aShkXhsQv9u6Vs1kOcpEZ2gokwIDAQAB"
const priv_key = "MIIEowIBAAKCAQEAp1CqpdSa9mpi6ERtK1bf8o84GMfyZa+F5j/lCLEwOU2d+12r\nfp+81BV3xXdnJ+N0nzU3NkUmTdX/Omkd+9si8JcIykmcb4sR3QeVCd+vdIEcBIdo\nWAT+c9j+z2EUmG0rNk1gynxkJbb7Owr4cancNH3kQDWWGveZ9lsDijOI9o4pB4CK\ndWc0RJObz4e4DpzOJT07qnnWVI9WueEoCheZNkDosrb4MmLnHeDEStySNeAuX3BO\n6CSHujcXYwOoA6wKHPtYyvgPeS8SHdpzf1YYBuHoesha4E0O0PacFZV0TTJaY4+n\nS4Mn+sRcdyv5aShkXhsQv9u6Vs1kOcpEZ2gokwIDAQABAoIBABxKJmBdn0n02P5e\nu3qteLYhgyGlhRWuZNx2hzo+A2Jc/k5HGz0QszPE4Xhw5O84pTpaHBi//mcAvOPa\nbChud+zoDKNvaNTvVbjilE+UE62GOv+FCZ6AUamy0fqsdngDVWAcGzaBa8l4s+fa\nxgEp8EKr2pEEvnmWzeB6qRGP/yN4xe9ajivJQ7PJLDgebYC4na0K9Jg2L5ud9LgO\nftLiJV+jl37TN7qTqK/Dd97w+aoxo3hTyhuVPuXoxaE5YBjN3bzLsQu0rlHkEvHu\ncw9WSPh3JBwFv8Y7PD7pMgqRX+LXKfI1KTc7oF98qZ/s03NI3U6cAFhaFmAQdo6T\nDjrh3OkCgYEA1bKjmW/2B0ESo1jUXygE6sGaiUiWsG0AYTEyJgCr+RPTkHXYMb1w\n5Hy0JbZ5LDop9FWLQSCV2TW6kezoYCpVsfxlyqhTWMhRXgVbTITKwPNhBNrEjal+\nxHjcqp/dHzf05kG87RYffsjoNb/q1TJLoXudcOK7QHb7GrQyGuBPdD8CgYEAyG+J\nBUwmLGlvdWs/9V8MLXbLzgJc8v2tfjVquCtENN/A+RdzW8AokpJ6OsRUafP3Jx5P\nu0trOy+LtdyFMt4e0PBKMrtzS+unL2TpN27D+wqJfj89xVA914VTeETfHHQZJCpY\ndWZe13GNk6GM+ojejLK7AlePiV9l1EZ2wqc65q0CgYAbyNs+kvERJmPO+zi5mpFx\nGHUITnjRPYrkGCpmCIZTn0FNshTG+tOX0aL2mFAO8Q0NaKXvdNYm5LZ6TKw1/Ksh\ntihh/hrAG2OA7v9c5pMaHUrK/8q4hIYn83L1eE2exn7ABWIUDWFQ8bxHaMmWqLBu\nsYzZ5ZDlI9MoOK+fEPUjrwKBgQCUYFTbshJ0QBz9nEZ9mz4FjfKzb3ZlfztmuZ5l\n9cmJJrbQ7vY7zpV6Y6rORDaFNNAaikrVyK/54WmYWEXWcS342FjlE3T3l9xsrlQi\n8AFuns9HwQM2RP9yw0UWPE2534wZBKv1RLIi5PG8fxRBBv9QwqLDyhP8yr00FnGm\nCWwGBQKBgC17FdAVL3vsku9GAWKnjkHq5T+fRgm9fUklot/6toik2x2EzOz9EGmV\nQBnCCnffDcuCdIFb/i6grX7BMfVYGBTcfkF5mLupW4BoVzW3YgJI0BJTVkptkDBC\nMZt6sOS51GhLIDtL1eyEf76Ks4HWcdcd2xUj2MWRFNt+CNReAKR8"
const PRIV_KEY_DIRECTORY = "../../agt_server/agt_private_key.rsa"

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
