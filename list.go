package amex

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"

	http "github.com/saucesteals/fhttp"

	"github.com/google/uuid"
)

type BillingAccountVirtualCard struct {
	Name                       string            `json:"-"`
	VirtualCardReferenceID     string            `json:"virtual_card_reference_id"`
	VirtualCardID              string            `json:"virtual_card_id"`
	VirtualCardType            string            `json:"virtual_card_type"`
	VirtualCardMaintainedBy    string            `json:"virtual_card_maintained_by"`
	LimitStartDate             string            `json:"limit_start_date"`
	LimitEndDate               string            `json:"limit_end_date"`
	DateCreated                string            `json:"date_created"`
	CurrentAmount              float64           `json:"current_amount"`
	BillingCurrency            string            `json:"billing_currency"`
	OriginalAmount             float64           `json:"original_amount"`
	CurrentUsageStatus         string            `json:"current_usage_status"`
	UserDefinedFields          map[string]string `json:"user_defined_fields"`
	AccountingFields           map[string]string `json:"accounting_fields"`
	EncryptedVirtualCardNumber string            `json:"encrypted_virtual_card_number"`
	VirtualCardLastFive        string            `json:"virtual_card_last_five"`
}

type BillingAccount struct {
	BillingAccountID string                      `json:"billing_account_id"`
	VirtualCards     []BillingAccountVirtualCard `json:"virtual_cards"`
}

type ListVirtualCardsResponse struct {
	Status struct {
		Code            string `json:"code"`
		ShortMessage    string `json:"short_message"`
		DetailedMessage string `json:"detailed_message"`
	} `json:"status"`
	CurrentPage         int              `json:"current_page"`
	TotalPages          int              `json:"total_pages"`
	TotalRecordsCount   int              `json:"total_records_count"`
	IssuingCardLastFive string           `json:"issuing_card_last_five"`
	CompanyID           string           `json:"company_id"`
	BillingAccts        []BillingAccount `json:"billing_accts"`
}

func (a *API) ListVirtualCards(ctx context.Context, accountToken string, page int, maxRecordsPerPage int) (*ListVirtualCardsResponse, error) {
	type SortBy struct {
		SortKey   string `json:"sort_key"`
		SortOrder string `json:"sort_order"`
	}

	type Payload struct {
		PartnerID             string   `json:"partner_id"`
		EncryptedAccountToken string   `json:"encrypted_account_token"`
		MaxRecordsPerPage     int      `json:"max_records_per_page"`
		PageNumber            int      `json:"page_number"`
		SortBy                []SortBy `json:"sort_by"`
	}

	payload, err := json.Marshal(Payload{
		PartnerID:             "600003120SharedVC",
		EncryptedAccountToken: accountToken,
		MaxRecordsPerPage:     maxRecordsPerPage,
		PageNumber:            page,
		SortBy:                []SortBy{{SortKey: "accounting_field_7", SortOrder: "desc"}},
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://apigw.americanexpress.com/payments/digital/v1/vpayment/acct_mgmt/accounts/virtual_card_search", bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "gzip, deflate, br, zstd")
	req.Header.Set("Accept-Language", "en-US,en;q=0.8")
	req.Header.Set("Cache-Control", "no-cache")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("One-Data-Correlation-Id", uuid.NewString())
	req.Header.Set("Origin", "https://www.americanexpress.com")
	req.Header.Set("Pragma", "no-cache")
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-site")
	req.Header.Set("Sec-Gpc", "1")
	req.Header.Set("X-Amex-Developer-App-Name", "600003120SharedVC")

	res, err := a.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	var response ListVirtualCardsResponse
	if err := json.NewDecoder(bytes.NewReader(body)).Decode(&response); err != nil {
		return nil, err
	}

	if response.Status.Code != "0000" {
		return nil, fmt.Errorf("status code: %s, message: %s", response.Status.Code, response.Status.DetailedMessage)
	}

	for i, billingAccount := range response.BillingAccts {
		for j, virtualCard := range billingAccount.VirtualCards {
			name, ok := virtualCard.AccountingFields["accounting_field_7"]
			if !ok {
				return nil, fmt.Errorf("no name found for card %s", virtualCard.VirtualCardLastFive)
			}

			response.BillingAccts[i].VirtualCards[j].Name = name
		}
	}

	return &response, nil
}
