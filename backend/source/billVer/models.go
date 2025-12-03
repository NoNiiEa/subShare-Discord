package billver

import (
	"github.com/NoNiiEa/subShare-Discord/source/group"
)

type SlipVerificationResult struct {
	IsValid bool `json:"is_valid"`
	MatchedAmount float64 `json:"matched_amount"`
	Method group.PaymentMethod `json:"method"`
	Account string `json:"account"`
	RawResponse []byte `json:"raw_response"`
}

type easySlipResponse struct {
	Status int `json:"status"`
	Data   struct {
		Payload     string `json:"payload"`
		TransRef    string `json:"transRef"`
		Date        string `json:"date"`
		CountryCode string `json:"countryCode"`

		Amount struct {
			Amount float64 `json:"amount"`
			Local  struct {
				Amount   float64 `json:"amount"`
				Currency string  `json:"currency"`
			} `json:"local"`
		} `json:"amount"`

		Fee float64 `json:"fee"`

		Ref1 string `json:"ref1"`
		Ref2 string `json:"ref2"`
		Ref3 string `json:"ref3"`

		Sender struct {
			Bank struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Short string `json:"short"`
			} `json:"bank"`
			Account struct {
				Name struct {
					Th string `json:"th"`
					En string `json:"en"`
				} `json:"name"`

				Bank *struct {
					Type    string `json:"type"`    // "BANKAC" | "TOKEN" | "DUMMY"
					Account string `json:"account"`
				} `json:"bank,omitempty"`

				Proxy *struct {
					Type    string `json:"type"`    // "NATID" | "MSISDN" | "EWALLETID" | "EMAIL" | "BILLERID"
					Account string `json:"account"`
				} `json:"proxy,omitempty"`
			} `json:"account"`
		} `json:"sender"`

		Receiver struct {
			Bank struct {
				ID    string `json:"id"`
				Name  string `json:"name"`
				Short string `json:"short"`
			} `json:"bank"`
			Account struct {
				Name struct {
					Th string `json:"th"`
					En string `json:"en"`
				} `json:"name"`

				Bank *struct {
					Type    string `json:"type"`    // "BANKAC" | "TOKEN" | "DUMMY"
					Account string `json:"account"`
				} `json:"bank,omitempty"`

				Proxy *struct {
					Type    string `json:"type"`    // "NATID" | "MSISDN" | "EWALLETID" | "EMAIL" | "BILLERID"
					Account string `json:"account"`
				} `json:"proxy,omitempty"`
			} `json:"account"`

			MerchantID string `json:"merchantId"`
		} `json:"receiver"`
	} `json:"data"`
}

type SubmitBillProofRequest struct {
	BillID     int64   `json:"bill_id"`
	MemberID   string  `json:"member_id"`
	AmountPaid float64 `json:"amount_paid"` // user-claimed, optional
	ImageBytes []byte  `json:"-"`
	FileName   string  `json:"-"` // "slip.jpg"
}