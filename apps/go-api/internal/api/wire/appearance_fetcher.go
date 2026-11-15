// Package wire — appearance_fetcher.go : l'implémentation PROD de service.AppearanceFetcher. Un
// client Halo par jeu de jetons, borné par le budget de débit partagé du compte propriétaire (aligné
// sur CareerFetcherFactoryFromTokens).
package wire

import (
	"context"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite/haloadapter"
	ratebudget "levelup/go-api/internal/platform/ratebudget"
	"levelup/go-api/internal/service"
)

// haloAppearanceFetcher adapte *haloadapter.Client à service.AppearanceFetcher. Il retient aussi les
// jetons : DiagnoseNameplate est une fonction LIBRE (pas une méthode du client) qui exige spartan et
// clearance.
type haloAppearanceFetcher struct {
	client         *haloadapter.Client
	spartanToken   string
	clearanceToken string
}

func (f *haloAppearanceFetcher) FetchAppearanceInputs(ctx context.Context, xuid string) (*domain.AppearanceInputs, error) {
	in, err := f.client.FetchAppearanceInputs(ctx, xuid)
	if err != nil || in == nil {
		return nil, err
	}
	return &domain.AppearanceInputs{
		ServiceTag:        in.ServiceTag,
		BannerImagePath:   in.BannerImagePath,
		BackdropImagePath: in.BackdropImagePath,
		EmblemPath:        in.EmblemPath,
		EmblemConfigID:    in.EmblemConfigID,
	}, nil
}

func (f *haloAppearanceFetcher) DiagnoseNameplate(ctx context.Context, emblemPath string, cfg int64) domain.AppearanceDiagnosis {
	return haloadapter.DiagnoseNameplate(ctx, emblemPath, cfg, f.spartanToken, f.clearanceToken)
}

func (f *haloAppearanceFetcher) DiagnoseCustomizationImage(ctx context.Context, inventoryPath string) domain.AppearanceDiagnosis {
	return f.client.DiagnoseCustomizationImage(ctx, inventoryPath)
}

// newHaloAppearanceFetcher retourne la fabrique PROD d'AppearanceFetcher : un client Halo par jeu de
// jetons, borné par le budget de débit partagé du compte propriétaire (ForXUID), aligné sur le flow
// live carrière.
func newHaloAppearanceFetcher(requestsPerSecond int) service.AppearanceFetcherFactory {
	return func(spartanToken, clearanceToken, ownerXUID string) service.AppearanceFetcher {
		client := haloadapter.NewClient(spartanToken, clearanceToken, requestsPerSecond)
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
