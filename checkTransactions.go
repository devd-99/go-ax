package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
)

type ContractTransaction struct {
	TxHash    string `json:"tx_hash"`
	Sender    string `json:"sender"`
	BlockTime string `json:"block_time"`
	Success   bool   `json:"success"`
	EthValue  string `json:"eth_value"`
}

type Result struct {
	BlockNumber int    `json:"block_number"`
	TxHash      string `json:"tx_hash"`
	Sender      string `json:"sender"`
	Success     bool   `json:"success"`
}

type DuneContractTxResponse struct {
	Data struct {
		Rows []ContractTransaction `json:"rows"`
	} `json:"data"`
}

// handleCheckTransaction handles the /check-transaction endpoint.
// It checks if the specified wallet has interacted with a specific contract. THIS: -> (0xa8F5dCC3035089111a9435FF25546c922a7c713A)
//
// Parameters:
// - w http.ResponseWriter: The response writer to send the HTTP response.
// - r *http.Request: The HTTP request containing the wallet address.
//
// Expected JSON request body:
//
//	{
//	  "walletAddress": "0x1234...5678"     // The Ethereum wallet address to check
//	}
//
// Returns JSON response:
//
//	{
//	  "wallet_interacted": true/false     // Boolean indicating if the wallet interacted with the contract
//	}
func handleCheckTransaction(w http.ResponseWriter, r *http.Request) {
	fmt.Println("Checking transaction")
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	contractTxResp, err := queryDuneContractTransactions()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	// fmt.Println("contractTxResp")
	// fmt.Println(contractTxResp)

	result, err := checkWalletInteraction(req.WalletAddress, *contractTxResp)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	//send result to Prover circuit
	//getting errors so not implemented. casting code available in prepareReq.go

	json.NewEncoder(w).Encode(map[string]bool{"wallet_interacted": result})
}

// queryDuneContractTransactions retrieves the transactions for a specific contract from the Dune API.
//
// Returns:
// - *DuneContractTxResponse: The response containing the contract transactions.
// - error: An error if the query fails.
func queryDuneContractTransactions() (*[]Result, error) {
	url := "https://api.dune.com/api/v1/query/4072279/results?limit=1000"
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-DUNE-API-KEY", "nAAC4sDHqo3cvmemQCrDpnXlEgjr27Ip")

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	// Convert body to JSON
	var jsonBody map[string]interface{}
	if err := json.Unmarshal(body, &jsonBody); err != nil {
		fmt.Println("Error parsing JSON:", err)
		return nil, err
	}

	// Print JSON
	jsonString, err := json.MarshalIndent(jsonBody, "", "  ")
	if err != nil {
		fmt.Println("Error converting JSON to string:", err)
		return nil, err
	}
	fmt.Println(string(jsonString))

	var resultsArray []Result
	if results, ok := jsonBody["result"].(map[string]interface{}); ok {
		if rows, ok := results["rows"].([]interface{}); ok {
			for _, row := range rows {
				rowMap := row.(map[string]interface{})
				result := Result{
					BlockNumber: int(rowMap["block_number"].(float64)),
					TxHash:      rowMap["tx_hash"].(string),
					Sender:      rowMap["sender"].(string),
					Success:     rowMap["success"].(bool),
				}
				resultsArray = append(resultsArray, result)
			}
		}
	}

	// Print results
	for _, result := range resultsArray {
		fmt.Printf("BlockNumber: %d, TxHash: %s, Sender: %s, Success: %t\n", result.BlockNumber, result.TxHash, result.Sender, result.Success)
	}

	fmt.Println("done")

	// var contractTxResp DuneContractTxResponse
	// if err := json.Unmarshal(body, &contractTxResp); err != nil {
	// 	return nil, err
	// }

	return &resultsArray, nil
}

// checkWalletInteraction checks if the specified wallet has interacted with the contract.
//
// Parameters:
// - walletAddress string: The wallet address to check.
// - transactions []ContractTransaction: The list of contract transactions to analyze.
//
// Returns:
// - bool: True if the wallet has interacted with the contract, false otherwise.
func checkWalletInteraction(walletAddress string, transactions []Result) (bool, error) {
	currentBlockNumber, err := queryCurrentBlock2()
	if err != nil {
		return false, err
	}

	for _, tx := range transactions {
		if tx.BlockNumber >= currentBlockNumber-100 && strings.EqualFold(tx.Sender, walletAddress) && tx.Success {
			return true, nil
		}
	}
	return false, nil
}

func queryCurrentBlock2() (int, error) {
	url := DuneCurrentBlockURL
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return 0, err
	}
	req.Header.Add("X-DUNE-API-KEY", DuneAPIKey)

	res, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer res.Body.Close()

	var currentBlockResp CurrentBlockResponse
	if err := json.NewDecoder(res.Body).Decode(&currentBlockResp); err != nil {
		return 0, err
	}

	if len(currentBlockResp.Result.Rows) == 0 {
		return 0, fmt.Errorf("no rows in current block response")
	}

	return int(currentBlockResp.Result.Rows[0].LatestBlockNumber), nil
}
