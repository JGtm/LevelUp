// Package duckdb — kill_distance_repo_elevation.go : LA LECTURE AU GRAIN DU FRAG.
//
// Même jointure que `LoadMatch` (kill_distance_repo.go), même scope d'un seul match, mêmes
// gardes — celles de kill_measured.go, qui restent l'unique site de la jointure. Ce qui
// change est la SORTIE : `LoadMatch` agrège par (xuid, arme) pour la carte « Distance des
// frags par arme » ; celle-ci ne rend rien d'autre que les lignes, parce que la carte
// « Dénivelé » (lot Y, décision D24) trace UN POINT PAR FRAG.
//
// POURQUOI DEUX REQUÊTES ET PAS UNE (et pourquoi c'est tenable) : les deux lectures sont
// bornées au même match_id et coûtent ~12 ms chacune sur le corpus (mesure du résidu 4.0a,
// 140 000 événements). Fondre la seconde dans la première aurait fait rendre à `LoadMatch`
// un couple (agrégat, lignes) dont chaque appelant jette une moitié — et aurait lié la carte
// des armes à la carte du dénivelé, qui n'ont ni le même cycle de vie ni la même porte.
// Elles partent en PARALLÈLE dans le groupe de chargement de la vue match : le mur d'attente
// n'en garde qu'une.
package duckdb

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
)

// LoadMatchElevation rend les frags mesurés du match, UN PAR LIGNE, dénivelé PHYSIQUE
// (`killer_z - victim_z`) non signé d'un point de vue — c'est `analysis.BuildMatchElevation`
// qui lui en donne un.
//
// Mêmes dégradations que `LoadMatch` : matchID vide = erreur (jamais un scan), classificateur
// absent = nil, tables absentes = `games.ErrCapabilityNotSupported`, zéro ligne = nil.
func (r *KillDistanceRepo) LoadMatchElevation(
	ctx context.Context, matchID string,
) ([]domain.MatchElevationKillRaw, error) {
	if matchID == "" {
		return nil, fmt.Errorf("KillDistanceRepo.LoadMatchElevation: matchID vide")
	}
	if r.classifier == nil {
		slog.DebugContext(ctx, "KillDistanceRepo: no classifier for title, no elevation",
			"match_id", matchID)
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	measured, err := r.queryMeasuredKills(ctx, matchID)
	if err != nil {
		if isTableNotFoundErr(err) {
			slog.DebugContext(ctx, "KillDistanceRepo: positions/kill-feed table missing (elevation)",
				"match_id", matchID, "table", string(positionsAtKill))
			return nil, games.ErrCapabilityNotSupported
		}
		slog.ErrorContext(ctx, "KillDistanceRepo: elevation query failed",
			"match_id", matchID, "err", err)
		return nil, fmt.Errorf("KillDistanceRepo.LoadMatchElevation: %w", err)
	}
	if len(measured) == 0 {
		return nil, nil
	}
	return r.resolveElevationRows(ctx, measured), nil
}

// resolveElevationRows traduit `source_tag` -> clé de registre -> libellé, une ligne par frag.
//
// Une source HORS REGISTRE est écartée, exactement comme dans `resolveRows` : le point
// porterait une arme inventée dans son infobulle. Une clé SANS LIBELLÉ, elle, est CONSERVÉE
// — la position du point ne dépend pas du mot, et l'UI sait taire un libellé vide.
func (r *KillDistanceRepo) resolveElevationRows(
	ctx context.Context, measured []killMeasured,
) []domain.MatchElevationKillRaw {
	keysSeen := map[string]bool{}
	weaponKeys := make([]string, 0)
	keyOf := make([]string, len(measured))
	for i, m := range measured {
		wk, ok := r.classifier.KillSourceRegistryKey(m.sourceTag)
		if !ok {
			continue
		}
		keyOf[i] = wk
		if !keysSeen[wk] {
			keysSeen[wk] = true
			weaponKeys = append(weaponKeys, wk)
		}
	}
	if len(weaponKeys) == 0 {
		return nil
	}
	meta := resolveWeaponKeyLabelsAny(ctx, r.pdb.Metadata, r.pdb.TitleSlug, weaponKeys)

	out := make([]domain.MatchElevationKillRaw, 0, len(measured))
	for i, m := range measured {
		if keyOf[i] == "" {
			continue
		}
		lbl := meta[keyOf[i]]
		out = append(out, domain.MatchElevationKillRaw{
			KillerXUID:     m.killerXUID,
			KillerGamertag: m.killerGT,
			VictimXUID:     m.victimXUID,
			VictimGamertag: m.victimGT,
			Weapon:         lbl.label,
			WeaponEN:       lbl.labelEN,
			TimeMS:         m.timeMS,
			DistanceM:      m.distanceM,
			DeltaZ:         m.deltaZ,
		})
	}
	return out
}
