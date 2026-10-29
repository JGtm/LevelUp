package haloadapter

// season_provider.go — calendriers des saisons (SeasonCalendar.json) et des saisons CSR
// (CsrSeasonCalendar.json), GameCMS.

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/haloapi/infinite"
)

// FetchSeasonCalendar récupère le calendrier des saisons. Jetons lus dans le contexte ; filet
// 401/403 : un jeton jugé valide par son expiration peut avoir été périmé par une rotation juste
// avant l'appel (jeton de session de la requête) — un seul nouvel essai, avec le jeton rafraîchi
// par le magasin (jamais de recapture) ; sans xuid dans le contexte, pas d'essai.
func (p *Provider) FetchSeasonCalendar(ctx context.Context, titleID string) ([]domain.SeasonCalendar, []byte, error) {
	if ctxkeys.HaloTokens(ctx) == nil {
		return nil, nil, fmt.Errorf("FetchSeasonCalendar: tokens absents du contexte")
	}
	api := p.api(ctx, surfaceProviderGet, infinite.RetryProvider(), infinite.CleGameCMS)
	type lu struct {
		dto  *infinite.SeasonCalendarDTO
		body []byte
	}
	got, err := retryOnAuth(ctx, func(c context.Context) (lu, error) {
		dto, body, e := api.FetchSeasonCalendar(c, authDe(ctxkeys.HaloTokens(c)))
		return lu{dto: dto, body: body}, e
	})
	if err != nil {
		return nil, got.body, err
	}
	hash := ContentHash(got.body)
	fetchedAt := time.Now().UTC()
	source := infinite.SeasonCalendarURL(api.Endpoints())
	seasons := make([]domain.SeasonCalendar, 0, len(got.dto.Seasons))
	for i, s := range got.dto.Seasons {
		sc := domain.SeasonCalendar{
			TitleID:     titleID,
			SeasonID:    fmt.Sprintf("s%d", i+1),
			Name:        s.SeasonMetadataPath,
			FetchedAt:   fetchedAt,
			ContentHash: hash,
			SourceURL:   source,
		}
		sc.StartDate, sc.EndDate = datesDeSaison(s.StartDate, s.EndDate)
		seasons = append(seasons, sc)
	}
	return seasons, got.body, nil
}

// FetchCSRSeasonCalendar récupère le calendrier des saisons CSR. Jetons lus dans le contexte.
func (p *Provider) FetchCSRSeasonCalendar(ctx context.Context, titleID string) ([]domain.CSRSeasonCalendar, []byte, error) {
	tokens := ctxkeys.HaloTokens(ctx)
	if tokens == nil {
		return nil, nil, fmt.Errorf("FetchCSRSeasonCalendar: tokens absents du contexte")
	}
	api := p.api(ctx, surfaceProviderGet, infinite.RetryProvider(), infinite.CleGameCMS)
	dto, body, err := api.FetchCSRSeasonCalendar(ctx, authDe(tokens))
	if err != nil {
		return nil, body, err
	}
	hash := ContentHash(body)
	fetchedAt := time.Now().UTC()
	source := infinite.CSRSeasonCalendarURL(api.Endpoints())
	seasons := make([]domain.CSRSeasonCalendar, 0, len(dto.Seasons))
	for i, s := range dto.Seasons {
		cs := domain.CSRSeasonCalendar{
			TitleID:     titleID,
			SeasonID:    fmt.Sprintf("csr-s%d", i+1),
			Name:        s.CsrSeasonFilePath,
			FetchedAt:   fetchedAt,
			ContentHash: hash,
			SourceURL:   source,
		}
		cs.StartDate, cs.EndDate = datesDeSaison(s.StartDate, s.EndDate)
		seasons = append(seasons, cs)
	}
	return seasons, body, nil
}

// datesDeSaison lit les dates RFC 3339 d'une saison : début (zéro si illisible), fin (nil si
// absente ou illisible).
func datesDeSaison(debut, fin string) (time.Time, *time.Time) {
	var start time.Time
	if t, err := time.Parse(time.RFC3339, debut); err == nil {
		start = t
	}
	if fin == "" {
		return start, nil
	}
	if t, err := time.Parse(time.RFC3339, fin); err == nil {
		return start, &t
	}
	return start, nil
}

// ContentHash retourne le SHA-256 hexadécimal du contenu brut.
func ContentHash(b []byte) string {
	h := sha256.Sum256(b)
	return hex.EncodeToString(h[:])
}
