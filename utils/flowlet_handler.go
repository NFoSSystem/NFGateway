package utils

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
)

func SendToEndpoint(packet *Buffer, endpoint string) (string, error) {
	valMap := url.Values{}
	valMap.Set("payload", string(packet.Buff()))
	defer packet.Release()

	res, err := http.PostForm(endpoint, valMap)
	if err != nil {
		return "", fmt.Errorf("Error received trying to send packet to %s", endpoint)
	}

	log.Println("Message sent to %s", endpoint)

	if res.StatusCode != 200 {
		fmt.Printf("Received HTTP Status %s", res.Status)
	}

	if data, err := ioutil.ReadAll(res.Body); err == nil {
		return string(data), nil
	}

	return "", fmt.Errorf("Error reading output from HTTP response")
}
