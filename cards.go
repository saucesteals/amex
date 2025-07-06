package amex

import "context"

type EligibleCard struct {
	AccountToken          string  `json:"accountToken"`
	AccountKey            string  `json:"accountKey"`
	AccountNumberLastFive string  `json:"accountNumberLastFive"`
	Status                string  `json:"status"`
	Rank                  int     `json:"rank"`
	Product               Product `json:"product"`
}

type DigitalAsset struct {
	AssetURL       string `json:"assetUrl"`
	AssetDimension string `json:"assetDimension"`
}

type Product struct {
	InitialArrangementCode string         `json:"initialArrangementCode"`
	ProductName            string         `json:"productName"`
	AssetID                string         `json:"assetId"`
	DigitalAsset           []DigitalAsset `json:"digitalAsset"`
}

func (a *API) ReadEligibleCards(ctx context.Context) ([]EligibleCard, error) {
	type ReadEligibleCardsArgs struct {
		Status          string   `json:"status"`
		AssetDimensions []string `json:"assetDimensions"`
	}

	res, err := a.callFunction(ctx, "ReadEligibleCards.v1", ReadEligibleCardsArgs{
		Status:          "ACTIVE",
		AssetDimensions: []string{"160x101"},
	})
	if err != nil {
		return nil, err
	}

	var eligibleCards []EligibleCard
	if err := res.Bind(&eligibleCards); err != nil {
		return nil, err
	}

	return eligibleCards, nil
}
