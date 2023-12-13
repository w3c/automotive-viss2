/******** Peter Winzell  11/27/23 *********************************************/
/******** (C) Volvo Cars, 2023 **********/

package atServer

import (
	"bytes"
	"context"
	"crypto/rsa"
	"encoding/json"
	"fmt"
	pb "github.com/w3c/automotive-viss2/grpc_pb"
	"github.com/w3c/automotive-viss2/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	_ "log"
	"net/http"
	"strconv"
	"testing"
	"time"
)

//Prequisites: AGT Server and Vissv2 Server should be up and running (0.0.0.0 in docker)
type TResponse struct {
	Token string `json:"token"`
}

const agt_posttesturl = "http://0.0.0.0:7500/agts"
const at_url = "http://0.0.0.0:8600/ats"
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

	post := &TResponse{}
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

	post := &TResponse{}
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
	Token   string `json:"token"`
	Purpose string `json:"purpose"`
	// Pop     string `json:string "pop"`
}

func getAtToken(agttoken string) (*http.Response, error) {

	atReq := &atRequest{
		Token:   agttoken,
		Purpose: "fuel-status",
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

func parseATToken(res http.Response, t *testing.T) *TResponse {

	defer res.Body.Close()

	post := &TResponse{}
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

func getShToken(t *testing.T) string {
	res_ag, err := getShortTermAGTResponse() // Get Access Grant Token
	if err != nil {
		t.Error(err)
	}

	res, err := getAtToken(parseATToken(*res_ag, t).Token) // Get Access token

	if err != nil {
		t.Error(err)
	}
	if res.StatusCode != http.StatusCreated {
		t.Error("status code expected to be 201 , got: ", res.StatusCode)
	} else {
		attokenpost := &TResponse{}
		derr := json.NewDecoder(res.Body).Decode(attokenpost)

		if derr != nil {
			t.Error(derr)
		}
		log.Println(attokenpost.Token)
		return attokenpost.Token
	}
	return ""
}

// Viss server must be up and running
func TestAtTokenAccess_ST(t *testing.T) {
	getShToken(t)
}

func TestAtTokenAccess_LT(t *testing.T) {
	res_ag, err := getLongTermAGTResponse() // Get Access Grant Token
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
		attokenpost := &TResponse{}
		derr := json.NewDecoder(res.Body).Decode(attokenpost)

		if derr != nil {
			t.Error(derr)
		}
		log.Println(attokenpost.Token)
	}

}

// Test actual requests against server, server and a feeder for south-bound must be running.
// gRPC get request with credentials
func getGRPCServerConnection() (*grpc.ClientConn, error) {
	var connection *grpc.ClientConn
	connection, err := grpc.Dial("0.0.0.0"+":8887", grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return nil, err
	}
	return connection, nil
}

func getVISSClient(connection *grpc.ClientConn) pb.VISSv2Client {
	client := pb.NewVISSv2Client(connection)
	return client
}

func getgRPCGetRequest() []string {
	commandList := make([]string, 4)
	commandList[0] = `{"action":"get","path":"Vehicle/Body/Lights","filter":{"type":"paths","parameter":"*"},"requestId":"235"}`
	commandList[1] = `{"action":"get","path":"Vehicle/Speed","requestId":"236"}`
	return commandList
}

func infuseTokengRPCGetRequest(at_token string) []string {

	commandList := make([]string, 2)
	str := fmt.Sprintf(`{"action":"get","path":"Vehicle/Body/Lights","filter":{"type":"paths","parameter":"*"},,"authorization":"%s","requestId":"235"}`, at_token)
	commandList[0] = str
	str = fmt.Sprintf(`{"action":"get","path":"Vehicle/Speed","authorization":"%s","requestId":"236"}`, at_token)
	commandList[1] = str

	return commandList
}

func TestGetAccessControlST(t *testing.T) {

	utils.InitLog("servercore-log.txt", "./logs", false, "Info")

	AT := getShToken(t)
	if AT != "" {
		grpcConnectiontion, err := getGRPCServerConnection()
		if err != nil {
			t.Error(err)
		}
		defer grpcConnectiontion.Close()
		vissClient := getVISSClient(grpcConnectiontion)

		ctx, cancel := context.WithTimeout(context.Background(), time.Second)
		defer cancel()

		request := infuseTokengRPCGetRequest(AT)
		pbRequest := utils.GetRequestJsonToPb(request[1], utils.PB_LEVEL1)
		pbResponse, err := vissClient.GetRequest(ctx, pbRequest)

		if err != nil {
			t.Error(err)
			return
		}
		vssResponse := utils.GetResponsePbToJson(pbResponse, utils.PB_LEVEL1)
		t.Log(vssResponse)
	} else {
		t.Error("AT token not delivered")
	}
}

func TestGetAccessControlLT(t *testing.T) {
	t.Error("not implemented yet")
}
