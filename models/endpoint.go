package models

import (
	"context"
	"fmt"
	"multisig/session"
	"net/http"
	"strconv"
	"time"
)

type Proposal struct {
	Symbol    string `json:"symbol"`
	Amount    string `json:"amount"`
	Rule      string `json:"rule"`
	UserId    string `json:"userId"`
	Status    string `json:"status"` // pending done
	CreatedAt uint64 `json:"created_at"`
}

var passCtx context.Context

func proposalHandler(w http.ResponseWriter, r *http.Request) {
	values := r.URL.Query()

	// wgt or point
	symbol := values.Get("symbol")
	amount := values.Get("amount")
	rule := values.Get("rule")
	userId := values.Get("userId")

	pass := false

	if symbol == "point" {
	} else if symbol == "wgt" {
		if rule == "GrowthLimit" {
			pass = GrowthLimitRule(amount, userId)
		}
	}

	status := "reject"
	if pass {
		status = "pending"
	}

	nowTime := uint64(time.Now().Unix())

	payload := &Proposal{
		Symbol:    symbol,
		Amount:    amount,
		Rule:      rule,
		UserId:    userId,
		Status:    status,
		CreatedAt: nowTime,
	}

	err := session.Database(passCtx).Insert(payload)
	if err != nil {
		pass = false
	}

	fmt.Fprintf(w, strconv.FormatBool(pass))
}

func EndpointRun(ctx context.Context) {
	passCtx = ctx
	http.HandleFunc("/proposal", proposalHandler)
	http.ListenAndServe(":9300", nil)
}
