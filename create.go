package amex

import (
	"context"
	"fmt"

	http "github.com/saucesteals/fhttp"
)

type CardUseType string

var (
	CardUseTypeSingleUse CardUseType = "SINGLE_USE"
	CardUseTypeMultiUse  CardUseType = "MULTI_USE"
)

type SpendingLimitRenewalType string

var (
	SpendingLimitRenewalTypeNever    SpendingLimitRenewalType = "NEVER"
	SpendingLimitRenewalTypeWeekly   SpendingLimitRenewalType = "WEEKLY"
	SpendingLimitRenewalTypeBiWeekly SpendingLimitRenewalType = "BI_WEEKLY"
	SpendingLimitRenewalTypeMonthly  SpendingLimitRenewalType = "MONTHLY"
)

type SpendingLimit struct {
	// Currency (e.g. "USD")
	Currency string `json:"currency"`

	// Amount in dollars (e.g. "20.00")
	Amount string `json:"amount"`
}

type SpendingLimitRenewalSchedule struct {
	// Start date of the renewal period (YYYY-MM-DD)
	StartDate string `json:"startDate"`

	// End date of the renewal period (YYYY-MM-DD)
	EndDate string `json:"endDate"`
}

type CreateVirtualCardArgs struct {
	AccountToken string `json:"accountToken"`
	CardNickname string `json:"cardNickname"`

	CardUseType CardUseType `json:"cardUseType"`

	SpendingLimit SpendingLimit `json:"spendingLimit"`

	SpendingLimitRenewalType SpendingLimitRenewalType `json:"spendingLimitRenewalType"`

	SpendingLimitRenewalSchedule []SpendingLimitRenewalSchedule `json:"spendingLimitRenewalSchedule"`
}

type VirtualCard struct {
	VirtualCardNumber string `json:"virtualCardNumber"`
	VirtualToken      string `json:"virtualToken"`
	CardNickname      string `json:"cardNickname"`
	SecurityCode      string `json:"securityCode"`

	// Expiry date (YYYY-MM)
	ExpiryYearMonth string `json:"expiryYearMonth"`

	// Start date of the card validity period (YYYY-MM-DD)
	StartDate string `json:"startDate"`

	// End date of the card validity period (YYYY-MM-DD)
	EndDate string `json:"endDate"`

	// Token end date (YYYY-MM-DD)
	TokenEndDate string `json:"tokenEndDate"`

	CardMemberFirstName string `json:"cardmemberFirstName"`
	CardMemberLastName  string `json:"cardmemberLastName"`
}

func (a *API) CreateVirtualCard(ctx context.Context, assessmentToken string, args CreateVirtualCardArgs) (VirtualCard, error) {
	res, err := a.callFunction(ctx, "CreateVirtualCard.v1", args, http.Header{
		"one-data-risk-assessment-token": {assessmentToken},
	})
	if err != nil {
		return VirtualCard{}, err
	}

	if err := res.Error(); err != nil {
		if err.Code == "access_denied" {
			return VirtualCard{}, StepUpRequiredError{
				AssessmentToken: err.AssessmentToken,
			}
		}

		return VirtualCard{}, fmt.Errorf("create virtual card: %s", err.Code)
	}

	var virtualCard VirtualCard
	if err := res.Bind(&virtualCard); err != nil {
		return VirtualCard{}, err
	}

	return virtualCard, nil
}
