package haloadapter

// compare_provider.go — stats d'un joueur distant (comparaison joueur contre joueur) depuis son
// service record matchmade : `/{jeu}/players/{joueur}/Matchmade/servicerecord` (schéma et chemin
// confirmés via le wrapper Grunt, Stats_GetPlayerServiceRecord). L'hôte est celui des stats SANS
// `:443` (clé `infinite.CleStatsSansPort`, jamais lue dans le résolveur de titre).

import (
	"context"
	"fmt"

	"levelup/go-api/internal/ctxkeys"
	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/duration"
	"levelup/go-api/internal/haloapi/infinite"
)

// FetchRemoteStats retourne les stats normalisées d'un joueur (sans les médailles). Conservé pour
// la comparaison ; délègue à FetchServiceRecord.
func (p *Provider) FetchRemoteStats(ctx context.Context, gamertag, titleSlug string) (*domain.NormalizedPlayerStats, error) {
	rec, err := p.FetchServiceRecord(ctx, gamertag, titleSlug)
	if err != nil {
		return nil, err
	}
	if rec == nil {
		return nil, nil
	}
	s := rec.Stats
	return &s, nil
}

// FetchServiceRecord lit le service record matchmade CUMULATIF d'un joueur et rend ses stats
// normalisées (temps de jeu compris) et ses médailles agrégées, en un appel. Jetons lus dans le
// contexte ; filet 401/403 (nouvel essai unique sur jeton rafraîchi).
func (p *Provider) FetchServiceRecord(ctx context.Context, gamertag, titleSlug string) (*domain.RemoteServiceRecord, error) {
	return retryOnAuth(ctx, func(c context.Context) (*domain.RemoteServiceRecord, error) {
		return p.fetchServiceRecordOnce(c, gamertag, titleSlug)
	})
}

func (p *Provider) fetchServiceRecordOnce(ctx context.Context, gamertag, titleSlug string) (*domain.RemoteServiceRecord, error) {
	tokens := ctxkeys.HaloTokens(ctx)
	if tokens == nil {
		return nil, fmt.Errorf("FetchServiceRecord: tokens absents du contexte")
	}
	dto, err := p.api(ctx, surfaceProviderGet, infinite.RetryProvider()).GetServiceRecord(ctx, authDe(tokens), gamertag)
	if err != nil {
		return nil, err
	}
	return serviceRecordDepuisDTO(dto, gamertag, titleSlug), nil
}

// serviceRecordDepuisDTO convertit le service record brut en valeurs de domaine : moyennes par
// match, KDR, FDA NET par match (ADR 0006 : ((frags + assists/3) − morts) / matchs, jamais un
// quotient par les morts), précision, médailles non nulles.
func serviceRecordDepuisDTO(dto *infinite.ServiceRecordDTO, gamertag, titleSlug string) *domain.RemoteServiceRecord {
	cs := dto.CoreStats
	stats := domain.NormalizedPlayerStats{
		TitleSlug:         titleSlug,
		Gamertag:          gamertag,
		Matches:           dto.MatchesCompleted,
		TimePlayedSeconds: duration.SecondsInt64(dto.TimePlayed),
	}
	if dto.MatchesCompleted > 0 {
		n := float64(dto.MatchesCompleted)
		stats.WinRate = float64(dto.Wins) / n
		stats.KillsPerGame = float64(cs.Kills) / n
		stats.DeathsPerGame = float64(cs.Deaths) / n
		stats.AssistsPerGame = float64(cs.Assists) / n
		stats.DamagePerGame = cs.DamageDealt / n
		stats.DamageTakenPerGame = cs.DamageTaken / n
		if cs.Deaths > 0 {
			stats.KDR = float64(cs.Kills) / float64(cs.Deaths)
		}
		stats.KDA = (float64(cs.Kills) + float64(cs.Assists)/3.0 - float64(cs.Deaths)) / n
		if cs.ShotsFired > 0 {
			stats.Accuracy = cs.ShotsHit / cs.ShotsFired
		}
	}
	medals := make([]domain.RemoteMedalCount, 0, len(cs.Medals))
	for _, m := range cs.Medals {
		if m.NameID == 0 || m.Count <= 0 {
			continue
		}
		medals = append(medals, domain.RemoteMedalCount{NameID: m.NameID, Count: m.Count})
	}
	return &domain.RemoteServiceRecord{
		Stats:            stats,
		Medals:           medals,
		SeasonIDs:        dto.Subqueries.SeasonIDs,
		PlaylistAssetIDs: dto.Subqueries.PlaylistAssetIDs,
	}
}

// FetchSeasonServiceRecord rend le nombre de matchs matchmade complétés par un joueur sur UNE
// saison (`seasonID` = chemin CMS, ex. « Seasons/Season7.json »), filtré classé / non classé si
// `isRanked` est donné (nil : total de la saison). (0, nil) si la saison n'a pas de donnée.
func (p *Provider) FetchSeasonServiceRecord(ctx context.Context, gamertag, seasonID string, isRanked *bool) (int, error) {
	return retryOnAuth(ctx, func(c context.Context) (int, error) {
		return p.fetchSeasonServiceRecordOnce(c, gamertag, seasonID, isRanked)
	})
}

func (p *Provider) fetchSeasonServiceRecordOnce(ctx context.Context, gamertag, seasonID string, isRanked *bool) (int, error) {
	tokens := ctxkeys.HaloTokens(ctx)
	if tokens == nil {
		return 0, fmt.Errorf("FetchSeasonServiceRecord: tokens absents du contexte")
	}
	dto, err := p.api(ctx, surfaceProviderGet, infinite.RetryProvider()).GetSeasonServiceRecord(ctx, authDe(tokens), gamertag, seasonID, isRanked)
	if err != nil {
		return 0, err
	}
	return dto.MatchesCompleted, nil
}
