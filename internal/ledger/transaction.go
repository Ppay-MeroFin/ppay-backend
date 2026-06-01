package ledger

import (
	"errors"

	"github.com/google/uuid"
)

type ProductType string

const (
	ProductTypeAirtime ProductType = "airtime"
	ProductTypeBundle  ProductType = "bundle"
)

var (
	ErrInvalidProductType = errors.New("invalid product type")
	ErrPhoneRequired      = errors.New("phone number is required")
	ErrNetworkRequired    = errors.New("network is required")
	ErrBundleCodeRequired = errors.New("bundle code is required")
	ErrInvalidAmount      = errors.New("amount must be greater than zero")
	ErrCurrencyRequired   = errors.New("currency is required")
	ErrInvalidAccount     = errors.New("from_account and to_account are required")
)

type TransactionRequest struct {
	ProductType  string    `json:"product_type"`
	PhoneNumber  string    `json:"phone_number"`
	Network      string    `json:"network"`
	BundleCode   *string   `json:"bundle_code,omitempty"`
	BundleName   *string   `json:"bundle_name,omitempty"`
	BundleSizeMB *int64    `json:"bundle_size_mb,omitempty"`
	AmountMinor  int64     `json:"amount_minor"`
	Currency     string    `json:"currency"`
	FromAccount  uuid.UUID `json:"from_account"`
	ToAccount    uuid.UUID `json:"to_account"`
}

type DataBundleTransaction struct {
	ProductType  ProductType
	PhoneNumber  string
	Network      string
	BundleCode   string
	BundleName   string
	BundleSizeMB int64
	AmountMinor  int64
	Currency     string
	FromAccount  uuid.UUID
	ToAccount    uuid.UUID
}

func NewDataBundleTransaction(
	phoneNumber string,
	network string,
	bundleCode string,
	bundleName string,
	bundleSizeMB int64,
	amountMinor int64,
	currency string,
	fromAccount uuid.UUID,
	toAccount uuid.UUID,
) (DataBundleTransaction, error) {
	if phoneNumber == "" {
		return DataBundleTransaction{}, ErrPhoneRequired
	}
	if network == "" {
		return DataBundleTransaction{}, ErrNetworkRequired
	}
	if bundleCode == "" {
		return DataBundleTransaction{}, ErrBundleCodeRequired
	}
	if amountMinor <= 0 {
		return DataBundleTransaction{}, ErrInvalidAmount
	}
	if currency == "" {
		return DataBundleTransaction{}, ErrCurrencyRequired
	}
	if fromAccount == uuid.Nil || toAccount == uuid.Nil {
		return DataBundleTransaction{}, ErrInvalidAccount
	}

	return DataBundleTransaction{
		ProductType:  ProductTypeBundle,
		PhoneNumber:  phoneNumber,
		Network:      network,
		BundleCode:   bundleCode,
		BundleName:   bundleName,
		BundleSizeMB: bundleSizeMB,
		AmountMinor:  amountMinor,
		Currency:     currency,
		FromAccount:  fromAccount,
		ToAccount:    toAccount,
	}, nil
}
