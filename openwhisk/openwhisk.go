package openwhisk

import (
	"fmt"
	"io/ioutil"
	"net/http"
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
func CreateFunction(hostname, auth, action string) error {
	cl := &http.Client{}

	endpoint := makeEndpointString(hostname, action)

	Counter++
	addrPtr := Counter

	var paramStr string = "{\"address\":\"172.17.0.1\", \"port\":\"9082\", \"abcd\":\"efgh\"}"
	if action == "nat" {
		leasedPorts := utils.CPMap.AssignPorts(&addrPtr)
		leasedPortsString := nflib.GetStringFromPortSlice(leasedPorts)
		paramStr = "{\"address\":\"172.17.0.1\", \"port\":\"9082\", \"leasedPorts\":\"" +
			leasedPortsString + "\"}"
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
