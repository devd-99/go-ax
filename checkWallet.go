package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"math/big"
	"net/http"
	"sort"
	"strconv"
	"strings"
)

type Request struct {
	Threshold     string `json:"threshold"`
	WalletAddress string `json:"walletAddress"`
}

type Transaction struct {
	Hash            string `json:"hash"`
	Value           string `json:"value"`
	From            string `json:"from"`
	To              string `json:"to"`
	TransactionType string `json:"transaction_type"`
	BlockTime       string `json:"block_time"`
	BlockNumber     int64  `json:"block_number"`
}

type DuneTransactionsResponse struct {
	Transactions []Transaction `json:"transactions"`
}

type Balance struct {
	Amount string `json:"amount"`
}

type CurrentBlockResponse struct {
	Result struct {
		Rows []struct {
			LatestBlockNumber int64 `json:"latest_block_number"`
		} `json:"rows"`
	} `json:"result"`
}

// handleCheckWallet handles the /check-wallet endpoint.
// It checks if the wallet balance has crossed a specified threshold within the last 3 blocks.
//
// Parameters:
// - w http.ResponseWriter: The response writer to send the HTTP response.
// - r *http.Request: The HTTP request containing the wallet address and threshold.
//
// Expected JSON request body:
//
//	{
//	  "threshold": "1000000000000000000",  // The threshold value as a string (in wei)
//	  "walletAddress": "0x1234...5678"     // The Ethereum wallet address to check
//	}
//
// Returns JSON response:
//
//	{
//	  "threshold_crossed": true/false     // Boolean indicating if the threshold was crossed
//	}
func handleCheckWallet(w http.ResponseWriter, r *http.Request) {
	var req Request
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	balanceResp, err := queryDuneBalance(req.WalletAddress)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	transactionsResp, err := queryDuneTransactions(req.WalletAddress)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// logging the transactions
	// for _, tx := range transactionsResp.Transactions {
	// 	fmt.Printf("Hash: %s\n", tx.Hash)
	// 	fmt.Printf("Value: %s\n", tx.Value)
	// 	fmt.Printf("From: %s\n", tx.From)
	// 	fmt.Printf("To: %s\n", tx.To)
	// 	fmt.Printf("Transaction Type: %s\n", tx.TransactionType)
	// 	fmt.Printf("Block Time: %s\n", tx.BlockTime)
	// 	fmt.Printf("Block Number: %d\n", tx.BlockNumber)
	// 	fmt.Println("-----")
	// }

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	threshold, _ := new(big.Int).SetString(req.Threshold, 10)

	fmt.Println(*balanceResp)
	fmt.Println(threshold)

	result := checkThresholdCrossing(*balanceResp, threshold, transactionsResp.Transactions)

	json.NewEncoder(w).Encode(map[string]bool{"threshold_crossed": result})
}

// queryDuneBalance retrieves the balance of a given wallet address from the Dune API.
//
// Parameters:
// - address string: The wallet address to query.
//
// Returns:
// - *DuneBalanceResponse: The response containing the wallet balance.
// - error: An error if the query fails.
func queryDuneBalance(address string) (*Balance, error) {
	url := fmt.Sprintf("https://api.dune.com/api/beta/balance/%s?chain_ids=11155111", address)
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-Dune-Api-Key", DuneAPIKey)

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	// fmt.Println(string(body))

	// get the balance from the response body, from the balances.amount field
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	balances, ok := result["balances"].([]interface{})
	if !ok || len(balances) == 0 {
		return nil, fmt.Errorf("no balances found in response")
	}

	balanceMap, ok := balances[0].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid balance format")
	}

	amount, ok := balanceMap["amount"].(string)
	if !ok {
		return nil, fmt.Errorf("amount not found or not a string")
	}

	balanceResp := Balance{Amount: amount}
	return &balanceResp, nil
}

// queryDuneTransactions retrieves the transactions for a given wallet address from the Dune API.
//
// Parameters:
// - address string: The wallet address to query.
//
// Returns:
// - *DuneTransactionsResponse: The response containing the wallet transactions.
// - error: An error if the query fails.
func queryDuneTransactions(address string) (*DuneTransactionsResponse, error) {
	url := fmt.Sprintf("https://api.dune.com/api/beta/transactions/%s?chain_ids=11155111&limit=100", address)
	method := "GET"

	client := &http.Client{}
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("X-Dune-Api-Key", DuneAPIKey)

	res, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	// fmt.Println(string(body))
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, err
	}

	// fmt.Println(result)

	transactions, ok := result["transactions"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("no transactions found in response")
	}

	var duneTransactionsResponse DuneTransactionsResponse
	for _, tx := range transactions {
		txMap, ok := tx.(map[string]interface{})
		if !ok {
			continue
		}

		// fmt.Println(txMap)

		hash, ok := txMap["hash"].(string)
		if !ok {
			return nil, fmt.Errorf("hash not found or not a string")
		}

		value, ok := txMap["value"].(string)
		if !ok {
			return nil, fmt.Errorf("value not found or not a string")
		}

		from, ok := txMap["from"].(string)
		if !ok {
			return nil, fmt.Errorf("from not found or not a string")
		}

		to, ok := txMap["address"].(string) // Updated key from "to" to "address"
		if !ok {
			return nil, fmt.Errorf("address not found or not a string")
		}

		transactionType, ok := txMap["transaction_type"].(string)
		if !ok {
			return nil, fmt.Errorf("transaction_type not found or not a string")
		}

		blockTime, ok := txMap["block_time"].(string)
		if !ok {
			return nil, fmt.Errorf("block_time not found or not a string")
		}

		blockNumber, ok := txMap["block_number"].(float64)
		if !ok {
			return nil, fmt.Errorf("block_number not found or not a float64")
		}

		transaction := Transaction{
			Hash:            hash,
			Value:           value,
			From:            from,
			To:              to,
			TransactionType: transactionType,
			BlockTime:       blockTime,
			BlockNumber:     int64(blockNumber),
		}
		duneTransactionsResponse.Transactions = append(duneTransactionsResponse.Transactions, transaction)
	}

	// Print transaction details
	// for _, tx := range duneTransactionsResponse.Transactions {
	// 	fmt.Printf("Hash: %s\n", tx.Hash)
	// 	fmt.Printf("Value: %s\n", tx.Value)
	// 	fmt.Printf("From: %s\n", tx.From)
	// 	fmt.Printf("To: %s\n", tx.To)
	// 	fmt.Printf("Transaction Type: %s\n", tx.TransactionType)
	// 	fmt.Printf("Block Time: %s\n", tx.BlockTime)
	// 	fmt.Printf("Block Number: %d\n", tx.BlockNumber)
	// 	fmt.Println("-----")
	// }

	return &duneTransactionsResponse, nil
}

// queryCurrentBlock retrieves the current block number from the Dune API.
//
// Returns:
// - *CurrentBlockResponse: The response containing the current block number.
// - error: An error if the query fails.

// checkThresholdCrossing checks if the wallet balance has crossed the specified threshold
// within the last 3 blocks.
//
// Parameters:
// - currentBalance *big.Int: The current balance of the wallet.
// - threshold *big.Int: The threshold to check against.
// - transactions []Transaction: The list of transactions to analyze.
// - currentBlockNumber int64: The current block number.
//
// Returns:
// - bool: True if the threshold was crossed, false otherwise.
func checkThresholdCrossing(currentBalance Balance, threshold *big.Int, transactions []Transaction) bool {
	// Sort transactions by BlockTime in descending order
	sort.Slice(transactions, func(i, j int) bool {
		return transactions[i].BlockTime > transactions[j].BlockTime
	})

	// Initialize balance with the current balance amount
	balance := new(big.Int)
	balance.SetString(currentBalance.Amount, 10)

	// Print initial balance
	fmt.Printf("Initial Balance: %s\n", balance.String())

	// Query the current block number
	currentBlockNumber, err := queryCurrentBlock2()
	if err != nil {
		fmt.Println("Error querying current block:", err)
		return false
	}
	fmt.Printf("Current Block Number: %d\n", currentBlockNumber)

	// Calculate the minimum block number to consider (current block - 3)
	minBlockNumber := int64(currentBlockNumber) - int64(3)
	fmt.Printf("Minimum Block Number to Consider: %d\n", minBlockNumber)

	// Iterate over the transactions
	for _, tx := range transactions {
		// Convert block number to big.Int
		blockNumber, ok := new(big.Int).SetString(strconv.FormatInt(tx.BlockNumber, 10), 10)
		if !ok {
			fmt.Printf("Invalid Block Number: %d\n", tx.BlockNumber)
			continue
		}

		// Skip transactions older than the minimum block number
		if blockNumber.Int64() < minBlockNumber {
			fmt.Printf("Skipping Transaction: %s (Block Number: %d)\n", tx.Hash, tx.BlockNumber)
			break
		}

		// Convert transaction value from hex to big.Int
		value, _ := new(big.Int).SetString(strings.TrimPrefix(tx.Value, "0x"), 16)
		fmt.Printf("Transaction: %s, Value: %s, Type: %s\n", tx.Hash, value.String(), tx.TransactionType)

		// Adjust balance based on transaction type
		if tx.TransactionType == "Sender" {
			balance.Sub(balance, value)
			fmt.Printf("Balance after sending: %s\n", balance.String())
		} else {
			balance.Add(balance, value)
			fmt.Printf("Balance after receiving: %s\n", balance.String())
		}

		// Check if the threshold was crossed in either direction
		currentBalanceInt := new(big.Int)
		currentBalanceInt.SetString(currentBalance.Amount, 10)
		if (balance.Cmp(threshold) > 0 && currentBalanceInt.Cmp(threshold) <= 0) ||
			(balance.Cmp(threshold) < 0 && currentBalanceInt.Cmp(threshold) >= 0) {
			fmt.Println("Threshold crossed!")
			return true
		}
	}

	fmt.Println("Threshold not crossed.")
	return false
}
