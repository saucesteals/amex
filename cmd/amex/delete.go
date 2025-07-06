package main

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/saucesteals/amex"
)

func delete(ctx context.Context, api *amex.API, card amex.EligibleCard) error {
	nameFilter := strings.ToLower(ask("Enter name filter (optional)"))

	minDayAgeString := ask("Enter minimum age in days (optional)")
	if minDayAgeString == "" {
		minDayAgeString = "0"
	}

	minDayAge, err := strconv.Atoi(minDayAgeString)
	if err != nil {
		return fmt.Errorf("invalid minimum day age: %w", err)
	}

	var virtualCards []amex.BillingAccountVirtualCard
	for i := 1; ; i++ {
		response, err := api.ListVirtualCards(ctx, card.AccountToken, i, 100)
		if err != nil {
			return fmt.Errorf("list virtual cards: %w", err)
		}

		for _, billingAccount := range response.BillingAccts {
			for _, virtualCard := range billingAccount.VirtualCards {
				if virtualCard.CurrentUsageStatus != "Active" {
					continue
				}

				if nameFilter != "" && !strings.Contains(strings.ToLower(virtualCard.Name), nameFilter) {
					continue
				}

				if minDayAge > 0 {
					tokenCreatedAt, err := time.ParseInLocation("2006-01-02", virtualCard.DateCreated, time.Local)
					if err != nil {
						return fmt.Errorf("parse card created at: %w", err)
					}

					dayAge := int(time.Since(tokenCreatedAt).Hours() / 24)
					if dayAge < minDayAge {
						log.Info("Skipping card", "card", virtualCard.VirtualCardLastFive, "age", dayAge)
						continue
					}
				}

				virtualCards = append(virtualCards, virtualCard)
			}
		}

		if response.TotalPages == i {
			break
		}
	}

	for i, virtualCard := range virtualCards {
		fmt.Printf("%d. %s (%s)\n", i+1, virtualCard.Name, virtualCard.VirtualCardLastFive)
	}

	log.Info("Found cards", "count", len(virtualCards))
	if len(virtualCards) == 0 {
		return nil
	}

	confirm := ask("Are you sure you want to delete these cards? (y/n)")
	if confirm != "y" {
		return nil
	}

	for _, virtualCard := range virtualCards {
		fmt.Printf("Deleting %s... ", virtualCard.Name)
		err := api.DeleteVirtualCard(ctx, card.AccountToken, virtualCard.EncryptedVirtualCardNumber)
		if err != nil {
			fmt.Println("Failed")
			log.Error("Failed to delete card", "error", err)
			continue
		}

		fmt.Println("Done")
	}

	return nil
}
