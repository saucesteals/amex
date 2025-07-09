package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/saucesteals/amex"
)

func create(ctx context.Context, profile *Profile, api *amex.API, card amex.EligibleCard) error {
	count, err := strconv.Atoi(ask("Enter number of cards to create"))
	if err != nil {
		return fmt.Errorf("invalid number of cards: %w", err)
	}

	name := ask("Enter card name")
	if name == "" {
		name = "Token"
	}

	limit := ask("Enter spending limit per card in dollars (e.g. 100.00)")
	if limitFloat, err := strconv.ParseFloat(limit, 64); err != nil {
		return fmt.Errorf("invalid spending limit: %w", err)
	} else {
		limit = fmt.Sprintf("%.2f", limitFloat)
	}

	now := time.Now()

	days, err := strconv.Atoi(ask("Enter number of days the cards will be active (e.g. 30)"))
	if err != nil {
		return fmt.Errorf("invalid number of days: %w", err)
	}

	today := now.Format("2006-01-02")
	endDate := now.AddDate(0, 0, days).Format("2006-01-02")

	writer, err := NewCardWriter(profile, card, name)
	if err != nil {
		return fmt.Errorf("create card writer: %w", err)
	}
	defer writer.Close()

	var assessmentToken string
	for i := 0; i < count; i++ {
		vcc, err := api.CreateVirtualCard(ctx, assessmentToken, amex.CreateVirtualCardArgs{
			AccountToken: card.AccountToken,
			CardNickname: name,
			CardUseType:  amex.CardUseTypeMultiUse,
			SpendingLimit: amex.SpendingLimit{
				Currency: "USD",
				Amount:   limit,
			},
			SpendingLimitRenewalType: amex.SpendingLimitRenewalTypeNever,
			SpendingLimitRenewalSchedule: []amex.SpendingLimitRenewalSchedule{
				{
					StartDate: today,
					EndDate:   endDate,
				},
			},
		})
		if err != nil {
			var stepUpErr amex.StepUpRequiredError
			if errors.As(err, &stepUpErr) {
				log.Info("Step up required")
				err = api.StepUp(ctx, card.AccountKey, stepUpErr.AssessmentToken)
				if err != nil {
					return fmt.Errorf("step up: %w", err)
				}

				log.Info("Step up successful")
				assessmentToken = stepUpErr.AssessmentToken
				i--

				continue
			}

			log.Error("create virtual card", "error", err)
			continue
		}

		log.Info(fmt.Sprintf("(%d/%d) Virtual card created", i+1, count), "card", vcc.VirtualCardNumber)
		if err := writer.Write(vcc); err != nil {
			return fmt.Errorf("write card: %w", err)
		}
	}

	log.Info("Created cards", "count", count, "path", writer.GetPath())

	return nil
}
