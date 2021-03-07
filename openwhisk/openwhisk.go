package openwhisk

import (
	"crypto/tls"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"faasrouter/utils"

	"bitbucket.org/Manaphy91/nflib"
)

var Counter uint32 = 0

// Arguments:
// - hostname
// - auth key
// - action name
// - param
// - value
func CreateFunction(hostname, auth, action, redisIp string, redisPort int, cntId uint16, repl string) error {
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}

	cl := &http.Client{Transport: tr}

	endpoint := makeEndpointString(hostname, action)

	Counter++
	addrPtr := Counter

	// var paramStr string = "{\"address\":\"172.18.0.1\", \"port\":\"9082\", \"abcd\":\"efgh\"}"
	// if action == "nat" {
	// 	leasedPorts := utils.CPMap.AssignPorts(&addrPtr)
	// 	leasedPortsString := nflib.GetStringFromPortSlice(leasedPorts)
	// 	paramStr = "{\"address\":\"172.18.0.1\", \"port\":\"9082\", \"leasedPorts\":\"" +
	// 		leasedPortsString + "\"}"
	// }

	var paramStr string = "{\"address\":\"172.17.0.1\", \"port\":\"9082\", \"redisIp\":\"" + redisIp + "\"," + "\"redisPort\":" +
		strconv.Itoa(redisPort) + ", \"cntId\":\"" + strconv.Itoa(int(cntId)) + "\", \"repl\":\"" + repl + "\"}"
	if action == "nat" {
		leasedPorts := utils.CPMap.AssignPorts(&addrPtr)
		leasedPortsString := nflib.GetStringFromPortSlice(leasedPorts)
		paramStr = "{\"address\":\"172.17.0.1\", \"port\":\"9082\", \"leasedPorts\":\"" +
			leasedPortsString + "\", \"redisIp\":\"" + redisIp + "\"," + "\"redisPort\":" +
			strconv.Itoa(redisPort) + ", \"cntId\":\"" + strconv.Itoa(int(cntId)) + "\", \"repl\":\"" + repl + "\"}"
	}

	reqBody := strings.NewReader(paramStr)
	req, err := http.NewRequest("POST", endpoint, reqBody)
	if err != nil {
		utils.RLogger.Fatalf("Error creating new POST request!\nExit from application")
	}

	req.Header.Add("Authorization", fmt.Sprintf("Basic %s", auth))
	req.Header.Add("Content-Type", "application/json")

	resp, err := cl.Do(req)
	if err != nil {
		return fmt.Errorf("Error sending POST request to %s\n", endpoint)
	}

	if resp.StatusCode != 200 && resp.StatusCode != 202 {
		text, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			utils.RLogger.Fatal("Error reading response to POST request\nExit from application")
		}
		utils.RLogger.Printf("Content of response: %s\n", text)
		return fmt.Errorf("Error StatusCode different by 200: %d\n", resp.StatusCode)
	}

	text, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		utils.RLogger.Fatal("Error reading response to POST request\nExit from application")
	}

	utils.RLogger.Printf("POST request response: %s", string(text))

	return nil
}

func makeEndpointString(hostname, actionName string) string {
	return fmt.Sprintf("http://%s/api/v1/namespaces/_/actions/%s?blocking=false",
		hostname, actionName)
}
