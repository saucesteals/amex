package amex

import (
	"context"
)

func (a *API) DeleteVirtualCard(ctx context.Context, accountToken, virtualToken string) error {
	type Args struct {
		VirtualToken string `json:"virtualToken"`
		AccountToken string `json:"accountToken"`
	}

	res, err := a.callFunction(ctx, "DeleteVirtualCard.v1", Args{
		VirtualToken: virtualToken,
		AccountToken: accountToken,
	})
	if err != nil {
		return err
	}

	if err := res.Error(); err != nil {
		return err
	}

	return nil
}
