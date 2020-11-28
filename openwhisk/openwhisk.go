package openwhisk

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

// Arguments:
// - hostname
// - auth key
// - action name
// - param
// - value
func CreateFunction(hostname, auth, action string) error {

	log.Println("Startup")

	cl := &http.Client{}

	endpoint := makeEndpointString(hostname, action)

	reqBody := strings.NewReader("{\"address\":\"172.17.0.1\", \"port\":\"9082\"}")
	req, err := http.NewRequest("POST", endpoint, reqBody)
	if err != nil {
		log.Fatalf("Error creating new POST request!\nExit from application")
	}

	req.Header.Add("Authorization", fmt.Sprintf("Basic %s", auth))
	req.Header.Add("Content-Type", "application/json")

	log.Println("Before POST request")
	resp, err := cl.Do(req)
	if err != nil {
		return fmt.Errorf("Error sending POST request to %s\n", endpoint)
	}

	if resp.StatusCode != 200 && resp.StatusCode != 202 {
		return fmt.Errorf("Error StatusCode different by 200: %d\n", resp.StatusCode)
	}

	text, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal("Error reading response to POST request\nExit from application")
	}

	log.Printf("POST request response: %s", string(text))

	return nil
}

func makeEndpointString(hostname, actionName string) string {
	return fmt.Sprintf("http://%s/api/v1/namespaces/_/actions/%s?blocking=false",
		hostname, actionName)
}
