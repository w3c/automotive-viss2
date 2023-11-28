/******** Peter Winzell (c), 11/27/23 *********************************************/

package atServer

import (
	"bytes"
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

const TEST = "eyJhbGciOiJSUzI1NiIsInR5cCI6IkpXVCJ9.eyJ2aW4iOiJHRU8wMDEiLCJpYXQiOiIxNzAxMDk1MDY0IiwiZXhwIjoiMTcwMTEwOTQ2NCIsImNseCI6IkluZGVwZW5kZW50K09FTStDbG91ZCIsImF1ZCI6Inczb3JnL2dlbjIiLCJqdGkiOiI0RjUwNjhBNC1COTVBLTRDQ0MtODM0OS01MzMwMkRCQkRDMUYifQ.JAEo_geyETITEdzA0ozg53OpoFmjhjGr1NSMhUnKJLdFKJltFgg4BtAfKJcRk_aWiL_7a4DYbRqpYgA0wVaMKlNsEPtPq9Zi_-ZaEOtA7CkywcnoYJQX844zMs_WkgZsY3biY8zybTvvS-owio8LAzPbBi9uzJlNu99707l6gidifigV5iFeaOfAe1EPXB0JWhQCYC0pIYwEq9hYfk9D77ELE4EBr77c-rVKIYRsKVfZFqUUlQ9alHU5u3u9bqd0wxBjPdWyPpZu5FbWXnGUabnE1bqumcIX-ZYIebdAlBvqcFBMkjaJQt7FNEXfOIW7qHYxObX7I_3iExwuaW0Xfw"
const agt_posttesturl = "http://0.0.0.0:7500/agts"
const SHORT_TERM_TOKEN_LENGTH = 14400

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
		t.Error("Vin does not match => ", vin)
	}
}

func TestLongTermTokenAccess(t *testing.T) {

}
