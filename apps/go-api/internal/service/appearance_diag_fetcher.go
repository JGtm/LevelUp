// Package service — appearance_diag_fetcher.go : implémentation PROD de
// AppearanceFetcher (Lot F). Un client Halo par jeu de tokens, borné par le rate
// budget partagé du compte propriétaire (aligné sur CareerFetcherFactoryFromTokens).
package service

import (
	"context"

	ratebudget "levelup/go-api/internal/platform/ratebudget"
	haloclient "levelup/go-api/internal/sync/haloclient"
)

// haloAppearanceFetcher adapte *haloclient.HaloAPIClient à AppearanceFetcher.
// Il retient aussi les tokens : DiagnoseNameplate est une fonction LIBRE (pas une
// méthode du client) qui exige spartan + clearance.
type haloAppearanceFetcher struct {
	client         *haloclient.HaloAPIClient
	spartanToken   string
	clearanceToken string
}

func (f *haloAppearanceFetcher) FetchAppearanceInputs(ctx context.Context, xuid string) (*haloclient.AppearanceInputs, error) {
	return f.client.FetchAppearanceInputs(ctx, xuid)
}

func (f *haloAppearanceFetcher) DiagnoseNameplate(ctx context.Context, emblemPath string, cfg int64) haloclient.AppearanceDiagnosis {
	return haloclient.DiagnoseNameplate(ctx, emblemPath, cfg, f.spartanToken, f.clearanceToken)
}

func (f *haloAppearanceFetcher) DiagnoseCustomizationImage(ctx context.Context, inventoryPath string) haloclient.AppearanceDiagnosis {
	return f.client.DiagnoseCustomizationImage(ctx, inventoryPath)
}

// NewHaloAppearanceFetcher retourne la fabrique PROD d'AppearanceFetcher :
// construit un client Halo par jeu de tokens, borné par le rate budget partagé du
// compte propriétaire (ForXUID), aligné sur le flow live carrière.
func NewHaloAppearanceFetcher(requestsPerSecond int) AppearanceFetcherFactory {
	return func(spartanToken, clearanceToken, ownerXUID string) AppearanceFetcher {
		client := haloclient.NewHaloAPIClient(spartanToken, clearanceToken, requestsPerSecond)
		if ownerXUID != "" {
			client = client.WithLimiter(ratebudget.ForXUID(ownerXUID, float64(requestsPerSecond)))
		}
		return &haloAppearanceFetcher{
			client:         client,
			spartanToken:   spartanToken,
			clearanceToken: clearanceToken,
		}
	}
}
