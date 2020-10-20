package models

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"multisig/configs"
	"multisig/session"
	"time"

	bot "github.com/MixinNetwork/bot-api-go-client"
	number "github.com/MixinNetwork/go-number"
	"github.com/MixinNetwork/mixin/common"
	"github.com/lib/pq"
	"github.com/timshannon/badgerhold"
)

const (
	PaymentStatePending   = "pending"
	PaymentStatePaid      = "paid"
	PaymentStateDeposited = "deposited"

	WGTAssetID      = "965e5c6e-434c-3fa9-b780-c50f43cd955c"
	WGTMixinAssetID = "b9f49cf777dc4d03bc54cd1367eebca319f8603ea1ce18910d09e2c540c630d8"
)

type Payment struct {
	PaymentID       string         `db:"payment_id"`
	AssetID         string         `db:"asset_id"`
	Amount          string         `db:"amount"`
	Threshold       int64          `db:"threshold"`
	Receivers       pq.StringArray `db:"receivers"`
	Memo            string         `db:"memo"`
	State           string         `db:"state"`
	CodeID          string         `db:"code_id"`
	TransactionHash string         `db:"transaction_hash"`
	RawTransaction  string         `db:"raw_transaction"`
	CreatedAt       time.Time      `db:"created_at"`
}

func CreatedPayment(ctx context.Context, payment *bot.Payment) (*Payment, error) {
	p := &Payment{
		PaymentID: payment.TraceId,
		AssetID:   payment.AssetId,
		Amount:    payment.Amount,
		Threshold: payment.Threshold,
		Receivers: payment.Receivers,
		Memo:      payment.Memo,
		State:     payment.Status,
		CodeID:    payment.CodeId,
		CreatedAt: payment.CreatedAt,
	}

	old, err := findPaymentByID(ctx, p.PaymentID)
	if err == nil || old == nil {
		err = session.Database(ctx).Insert(p)
	}

	if err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	return p, nil
}

func findPaymentByID(ctx context.Context, paymentID string) (*Payment, error) {
	if id, _ := bot.UuidFromString(paymentID); id.String() != paymentID {
		return nil, nil
	}

	var payments []*Payment

	err := session.Database(ctx).Find(&payments, badgerhold.Where("PaymentID").Eq(paymentID))
	var p *Payment
	if len(payments) > 0 {
		p = payments[0]
	}
	return p, err

}

func FindPaymentByMemo(ctx context.Context, memo string) (*Payment, error) {
	if id, _ := bot.UuidFromString(memo); id.String() != memo {
		return nil, nil
	}
	var payments []*Payment

	err := session.Database(ctx).Find(&payments, badgerhold.Where("Memo").Eq(memo).And("State").Eq(PaymentStatePending))
	var p *Payment
	if len(payments) > 0 {
		p = payments[0]
	}
	return p, err
}

func FindPaymentsByState(ctx context.Context, state string, limit int) ([]*Payment, error) {
	var payments []*Payment

	err := session.Database(ctx).Find(&payments, badgerhold.Where("State").Eq(state).Limit(limit))
	if err != nil {
		return nil, session.TransactionError(ctx, err)
	}
	return payments, nil
}

func LoopingPendingPayments(ctx context.Context) error {
	for {
		payments, err := FindPaymentsByState(ctx, PaymentStatePending, 100)
		if err != nil {
			time.Sleep(time.Second)
			session.Logger(ctx).Errorf("FindPaymentsByState %#v", err)
			continue
		}
		for _, payment := range payments {
			botPayment, err := bot.ReadPaymentByCode(ctx, payment.CodeID)
			if err != nil {
				time.Sleep(time.Second)
				session.Logger(ctx).Errorf("ReadPaymentByCode %#v", err)
				continue
			}
			if botPayment.Status == PaymentStatePaid {
				session.Database(ctx).UpdateMatching(&Payment{}, badgerhold.Where("PaymentID").Eq(payment.PaymentID), func(record interface{}) error {
					update, ok := record.(*Payment)
					if !ok {
						err = fmt.Errorf("Record isn't the correct type! Got %T", record)
						return err
					}
					update.State = PaymentStatePaid

					return nil
				})

				if err != nil {
					time.Sleep(time.Second)
					session.Logger(ctx).Errorf("Updated payment %#v", err)
					continue
				}
			}
		}
		if len(payments) < 1 {
			time.Sleep(10 * time.Second)
		}
	}
}

func LoopingPaidPayments(ctx context.Context) error {
	network := NewMixinNetwork("http://35.234.74.25:8239")
	for {
		payments, err := FindPaymentsByState(ctx, PaymentStatePaid, 100)
		if err != nil {
			time.Sleep(time.Second)
			session.Logger(ctx).Errorf("FindPaymentsByState %#v", err)
			continue
		}
		for _, payment := range payments {
			err = payment.deposit(ctx, network)
			if err != nil {
				time.Sleep(time.Second)
				session.Logger(ctx).Errorf("deposit %#v", err)
				continue
			}
		}
		if len(payments) < 1 {
			time.Sleep(10 * time.Second)
		}
	}
}

func (payment *Payment) deposit(ctx context.Context, network *MixinNetwork) error {
	mixin := configs.AppConfig.Mixin
	input, err := ReadMultisig(ctx, payment.Amount, payment.Memo)
	if err != nil || input == nil {
		return err
	}
	if payment.RawTransaction != input.SignedTx && input.SignedTx != "" {
		payment.RawTransaction = input.SignedTx
		session.Database(ctx).UpdateMatching(&Payment{}, badgerhold.Where("PaymentID").Eq(payment.PaymentID), func(record interface{}) error {
			update, ok := record.(*Payment)
			if !ok {
				err = fmt.Errorf("Record isn't the correct type! Got %T", record)
				return err
			}
			update.RawTransaction = payment.RawTransaction

			return nil
		})
		if err != nil {
			return err
		}
	}
	if payment.RawTransaction == "" {
		var raw = ""
		if input.State == "signed" {
			raw = input.SignedTx
		}
		if raw == "" {
			key, err := bot.ReadGhostKeys(ctx, input.Members, 0, mixin.AppID, mixin.SessionID, mixin.PrivateKey)
			if err != nil {
				return err
			}
			outputs := []*Output{&Output{Mask: key.Mask, Keys: key.Keys, Amount: payment.Amount, Script: "fffe01"}}
			tx := &Transaction{
				Inputs:  []*Input{&Input{Hash: input.TransactionHash, Index: input.OutputIndex}},
				Outputs: outputs,
				Asset:   WGTMixinAssetID,
			}
			data, err := json.Marshal(tx)
			if err != nil {
				return err
			}
			raw, err = buildTransaction(data)
			if err != nil {
				return err
			}
		}
		session.Database(ctx).UpdateMatching(&Payment{}, badgerhold.Where("PaymentID").Eq(payment.PaymentID), func(record interface{}) error {
			update, ok := record.(*Payment)
			if !ok {
				err = fmt.Errorf("Record isn't the correct type! Got %T", record)
				return err
			}
			update.RawTransaction = raw
			return nil
		})
		if err != nil {
			return err
		}
	}
	request, err := bot.CreateMultisig(ctx, "sign", payment.RawTransaction, mixin.AppID, mixin.SessionID, mixin.PrivateKey)

	if err != nil {
		return err
	}

	if request.State == "initial" {
		pin, err := bot.EncryptPIN(ctx, mixin.Pin, mixin.PinToken, mixin.SessionID, mixin.PrivateKey, uint64(time.Now().UnixNano()))
		if err != nil {
			return err
		}
		request, err = bot.SignMultisig(ctx, request.RequestId, pin, mixin.AppID, mixin.SessionID, mixin.PrivateKey)
		if err != nil {
			return err
		}
	}

	if payment.RawTransaction != request.RawTransaction {
		payment.TransactionHash = request.TransactionHash
		payment.RawTransaction = request.RawTransaction
		session.Database(ctx).UpdateMatching(&Payment{}, badgerhold.Where("PaymentID").Eq(payment.PaymentID), func(record interface{}) error {
			update, ok := record.(*Payment)
			if !ok {
				err = fmt.Errorf("Record isn't the correct type! Got %T", record)
				return err
			}
			update.TransactionHash = payment.TransactionHash
			update.RawTransaction = payment.RawTransaction

			return nil
		})
		if err != nil {
			return err
		}
	}

	data, err := hex.DecodeString(payment.RawTransaction)
	if err != nil {
		return err
	}
	var stx common.SignedTransaction
	err = common.MsgpackUnmarshal(data, &stx)

	if err != nil {
		return err
	}
	if len(stx.Signatures) > 0 && len(stx.Signatures[0]) < int(payment.Threshold) {
		return nil
	}
	tx, err := network.GetTransaction(payment.TransactionHash)
	if tx == nil {
		_, err := network.SendRawTransaction(payment.RawTransaction)
		if err != nil {
			return err
		}
	}
	session.Database(ctx).UpdateMatching(&Payment{}, badgerhold.Where("PaymentID").Eq(payment.PaymentID), func(record interface{}) error {
		update, ok := record.(*Payment)
		if !ok {
			err = fmt.Errorf("Record isn't the correct type! Got %T", record)
			return err
		}
		update.State = PaymentStateDeposited
		return nil
	})
	return err
}

func LoopingApprove(ctx context.Context) error {
	network := NewMixinNetwork("http://35.234.74.25:8239")
	for {
		var proposals []*Proposal
		session.Database(passCtx).Find(&proposals, badgerhold.Where("Status").Eq("pending").Limit(1))
		for _, proposal := range proposals {
			approve(ctx, proposal, network)
		}

		time.Sleep(10 * time.Second)
	}
}

func approve(ctx context.Context, proposal *Proposal, network *MixinNetwork) error {
	mixin := configs.AppConfig.Mixin

	input, err := FilterMultisig(ctx, proposal.Amount)
	if err != nil || input == nil {
		return err
	}

	receivers := mixin.Receivers
	receivers = append(receivers, mixin.AppID)

	key, err := bot.ReadGhostKeys(ctx, []string{proposal.UserId}, 0, mixin.AppID, mixin.SessionID, mixin.PrivateKey)
	keyMultisig, err := bot.ReadGhostKeys(ctx, receivers, 1, mixin.AppID, mixin.SessionID, mixin.PrivateKey)
	if err != nil {
		return err
	}
	afterAmount := number.FromString(input.Amount).Sub(number.FromString(proposal.Amount))

	outputs := []*Output{&Output{Mask: key.Mask, Keys: key.Keys, Amount: proposal.Amount, Script: "fffe01"}}
	if afterAmount.String() > "0" {
		outputs = append(outputs, &Output{Mask: keyMultisig.Mask, Keys: keyMultisig.Keys, Amount: afterAmount.String(), Script: "fffe02"})
	}

	tx := &Transaction{
		Inputs:  []*Input{&Input{Hash: input.TransactionHash, Index: input.OutputIndex}},
		Outputs: outputs,
		Asset:   WGTMixinAssetID,
	}
	data, err := json.Marshal(tx)
	if err != nil {
		return err
	}
	raw, err := buildTransaction(data)

	if err != nil {
		return err
	}
	request, err := bot.CreateMultisig(ctx, "sign", raw, mixin.AppID, mixin.SessionID, mixin.PrivateKey)

	if err != nil {
		return err
	}

	dataRaw, err := hex.DecodeString(request.RawTransaction)
	if err != nil {
		return err
	}
	var stx common.SignedTransaction
	err = common.MsgpackUnmarshal(dataRaw, &stx)

	if err != nil {
		return err
	}
	if len(stx.Signatures) > 0 && len(stx.Signatures[0]) < int(request.Threshold) {
		return nil
	}
	txReqeust, err := network.GetTransaction(request.TransactionHash)
	if txReqeust == nil {
		rawTransaction := request.RawTransaction
		if request.RawTransaction != input.SignedTx && input.SignedTx != "" {
			rawTransaction = input.SignedTx
		}
		_, err := network.SendRawTransaction(rawTransaction)
		if err != nil {
			return err
		}
	}

	session.Database(ctx).UpdateMatching(&Proposal{}, badgerhold.Where("CreatedAt").Eq(proposal.CreatedAt), func(record interface{}) error {
		update, ok := record.(*Proposal)
		if !ok {
			err = fmt.Errorf("Record isn't the correct type! Got %T", record)
			return err
		}
		update.Status = "done"
		return nil
	})

	return nil
}

func HandleMessage(ctx context.Context, userID, amount string) (string, error) {
	payment, err := FindPaymentByMemo(ctx, userID)
	if err != nil {
		return "", err
	}
	if payment != nil {
		return payment.CodeID, nil
	}
	mixin := configs.AppConfig.Mixin
	receivers := mixin.Receivers
	receivers = append(receivers, mixin.AppID)
	om := struct {
		Receivers []string `json:"receivers"`
		Threshold int64    `json:"threshold"`
	}{
		receivers, 2,
	}
	pr := &bot.PaymentRequest{
		AssetId:          WGTAssetID,
		Amount:           amount,
		TraceId:          bot.UuidNewV4().String(),
		OpponentMultisig: om,
		Memo:             userID,
	}
	botPayment, err := bot.CreatePaymentRequest(ctx, pr, mixin.AppID, mixin.SessionID, mixin.PrivateKey)
	if err != nil {
		return "", err
	}
	payment, err = CreatedPayment(ctx, botPayment)
	if err != nil {
		return "", err
	}
	return payment.CodeID, nil
}
