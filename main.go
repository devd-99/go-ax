package main

import (
	"fmt"
	"net/http"
)

const (
	DuneTransactionsURL = "https://api.dune.com/api/beta/transactions/%s?chain_ids=11155111"
	DuneBalanceURL      = "https://api.dune.com/api/beta/balance/%s?chain_ids=11155111"
	DuneContractTxURL   = "https://api.dune.com/api/v1/query/4072279/results?limit=1000"
	DuneCurrentBlockURL = "https://api.dune.com/api/v1/query/4074150/results?limit=1000"
	DuneAPIKey          = "nAAC4sDHqo3cvmemQCrDpnXlEgjr27Ip"
)

func main() {
	http.HandleFunc("/check-wallet", handleCheckWallet)
	http.HandleFunc("/check-transaction", handleCheckTransaction)
	fmt.Println("Server running on :8080")
	http.ListenAndServe(":8080", nil)
}
