package assets

import (
	"context"
	"errors"
	"fmt"
)

// ChainFetcher est un fetcher composé de plusieurs fetchers ordonnés.
// Tente chaque fetcher dans l'ordre ; passe au suivant en cas d'ErrNotFound.
// Utilisé pour le fallback médaille individuelle → spritesheet.
type ChainFetcher struct {
	fetchers []Fetcher
}

// NewChainFetcher crée un ChainFetcher avec les fetchers donnés.
func NewChainFetcher(fetchers ...Fetcher) *ChainFetcher {
	return &ChainFetcher{fetchers: fetchers}
}

// Supports retourne true si au moins un fetcher de la chaîne supporte le Kind.
func (c *ChainFetcher) Supports(k Kind) bool {
	for _, f := range c.fetchers {
		if f.Supports(k) {
			return true
		}
	}
	return false
}

// Fetch tente chaque fetcher dans l'ordre.
// Retourne le premier succès, ou la dernière erreur si tous échouent.
func (c *ChainFetcher) Fetch(ctx context.Context, ref Ref) (Payload, error) {
	var lastErr error
	for _, f := range c.fetchers {
		if !f.Supports(ref.Kind) {
			continue
		}
		p, err := f.Fetch(ctx, ref)
		if err == nil {
			return p, nil
		}
		if !errors.Is(err, ErrNotFound) {
			// Erreur non-404 : on arrête la chaîne.
			return nil, err
		}
		lastErr = err
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, fmt.Errorf("%w: no fetcher for kind %s", ErrUnsupportedKind, ref.Kind)
}

// SpritesheetFallbackFetcher est un Fetcher spécial pour KindMedalImage.
// Il essaie d'abord l'URL individuelle (GameCMSFetcher) ;
// en cas d'ErrNotFound, il retourne l'URL du spritesheet comme URLPayload.
// Cela permet au frontend de découper via sprite_index.
type SpritesheetFallbackFetcher struct {
	primary Fetcher
	baseURL string
}

// NewSpritesheetFallbackFetcher crée un fetcher avec fallback spritesheet.
func NewSpritesheetFallbackFetcher(primary Fetcher, gamecmsBaseURL string) *SpritesheetFallbackFetcher {
	if gamecmsBaseURL == "" {
		gamecmsBaseURL = defaultGameCMSBase
	}
	return &SpritesheetFallbackFetcher{primary: primary, baseURL: gamecmsBaseURL}
}

// Supports retourne true uniquement pour KindMedalImage.
func (f *SpritesheetFallbackFetcher) Supports(k Kind) bool {
	return k == KindMedalImage
}

// Fetch essaie d'abord l'URL individuelle, puis retourne l'URL spritesheet si 404.
func (f *SpritesheetFallbackFetcher) Fetch(ctx context.Context, ref Ref) (Payload, error) {
	if ref.Kind != KindMedalImage {
		return nil, ErrUnsupportedKind
	}
	p, err := f.primary.Fetch(ctx, ref)
	if err == nil {
		return p, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return nil, err
	}
	// Fallback : URL spritesheet.
	spritesheetURL := fmt.Sprintf(
		"%s/hi/Progression/file/medals/sprites/%s.png",
		f.baseURL, ref.TitleID,
	)
	return URLPayload{
		URL:         spritesheetURL,
		ContentType: "image/png",
	}, nil
}
