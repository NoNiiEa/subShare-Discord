package okslip

import "time"

type CheckSlipRequest struct {
	Url string `json:"url"`
	Log bool `json:"log"`
}

type Response struct {
	Success bool `json:"success"`
	Data    Data `json:"data"`
}

type Data struct {
	Success           bool      `json:"success"`
	Message           string    `json:"message"`
	Language          string    `json:"language,omitempty"`
	ReceivingBank     string    `json:"receivingBank"`
	SendingBank       string    `json:"sendingBank"`
	TransRef          string    `json:"transRef"`
	TransDate         string    `json:"transDate"`      
	TransTime         string    `json:"transTime"`      
	TransTimestamp    time.Time `json:"transTimestamp"`
	Sender            Party     `json:"sender"`
	Receiver          Party     `json:"receiver"`
	Amount            float64   `json:"amount"`
	PaidLocalAmount   float64   `json:"paidLocalAmount,omitempty"`
	PaidLocalCurrency string    `json:"paidLocalCurrency,omitempty"`
	CountryCode       string    `json:"countryCode"`
	TransFeeAmount    float64   `json:"transFeeAmount,omitempty"`
	Ref1              string    `json:"ref1,omitempty"`
	Ref2              string    `json:"ref2,omitempty"`
	Ref3              string    `json:"ref3,omitempty"`
	ToMerchantID      string    `json:"toMerchantId,omitempty"`
}

type Party struct {
	DisplayName string  `json:"displayName"`
	Name        string  `json:"name"`
	Proxy       Proxy   `json:"proxy"`
	Account     Account `json:"account"`
}

type Proxy struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

type Account struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}