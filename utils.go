package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
)

// makeRequest is a generic function to make HTTP GET requests to the Dune API.
//
// Type Parameters:
// - T: The type of the expected response structure.
//
// Parameters:
// - url string: The URL to send the request to.
//
// Returns:
// - *T: A pointer to the parsed response of type T.
// - error: An error if the request or parsing fails.
func makeRequest[T any](url string) (*T, error) {
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("X-Dune-Api-Key", DuneAPIKey)
	fmt.Println("Making request to", url, DuneAPIKey)

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	var result T
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	return &result, nil
}
