package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/saucesteals/amex"
)

func list(ctx context.Context, api *amex.API, card amex.EligibleCard) error {
	nameFilter := strings.ToLower(ask("Enter name filter (optional)"))

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

	return nil
}
