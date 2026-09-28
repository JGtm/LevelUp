// Package duckdb — killsource_category_kills_repo.go : les frags par CATEGORIE de source
// de degat (objet explosif du decor, chute et environnement), lus dans le film.
//
// # Pourquoi une seconde lecture de la meme requete
//
// Le lecteur par arme (`LoadWeaponKillsAggregated`) ne remonte qu'une source qui porte une
// cle de registre (decision D7 du 2026-09-01 : on ne devine pas). Or la plupart des objets
// explosifs du decor n'en ont pas : leurs frags y restent « Non attribue ». Les « Outils de
// destruction » de l'Escouade (decision D8 du plan du 2026-09-26) les nomment pourtant,
// d'apres la CATEGORIE de la source, que le titre lit avec certitude dans ses classes. Cette
// lecture-ci rend ces categories ; la « Repartition des frags » (fragdist) n'en depend pas
// et reste byte-identique.
//
// Meme requete, meme vue `_latest` (regle ART n2), meme credit (`feed_killer_xuid`) : le
// parcours est partage avec le lecteur par arme (forEachSourceTally).
package duckdb

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// categoryTally : compteur intermediaire par (xuid, categorie, cle de registre eventuelle).
type categoryTally struct {
	xuid      string
	category  string
	weaponKey string
}

// LoadKillSourceCategoryKills rend les frags par (joueur, categorie de source), avec la cle
// de registre de la source quand elle en porte une (port.KillSourceCategoryRow).
//
// Titre sans categoriseur (interface optionnelle du classificateur) ou vue absente →
// games.ErrCapabilityNotSupported : l'appelant degrade sans ces lignes.
func (r *KillSourceWeaponKillsRepo) LoadKillSourceCategoryKills(
	ctx context.Context,
	slug string,
	filters port.WeaponKillFilters,
) ([]port.KillSourceCategoryRow, error) {
	if err := filters.Validate(); err != nil {
		return nil, fmt.Errorf("KillSourceWeaponKillsRepo.LoadKillSourceCategoryKills: %w", err)
	}
	if r.categorizer == nil || r.classifier == nil {
		slog.DebugContext(ctx, "KillSourceWeaponKillsRepo: aucun categoriseur de source pour ce titre",
			"slug", slug, "match_count", len(filters.MatchIDs))
		return nil, games.ErrCapabilityNotSupported
	}

	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	tally := map[categoryTally]int{}
	err := r.forEachSourceTally(ctx, filters, func(xuid string, sourceTag uint32, kills int) {
		category, ok := r.categorizer.KillSourceCategory(sourceTag)
		if !ok {
			return
		}
		key, _ := r.classifier.KillSourceRegistryKey(sourceTag) // "" : source sans cle
		tally[categoryTally{xuid: xuid, category: category, weaponKey: key}] += kills
	})
	if err != nil {
		if isTableNotFoundErr(err) {
			slog.DebugContext(ctx, "KillSourceWeaponKillsRepo: match_kill_events_latest absente",
				"slug", slug, "match_count", len(filters.MatchIDs))
			return nil, games.ErrCapabilityNotSupported
		}
		slog.ErrorContext(ctx, "KillSourceWeaponKillsRepo: lecture des categories echouee",
			"slug", slug, "match_count", len(filters.MatchIDs), "err", err)
		return nil, fmt.Errorf("KillSourceWeaponKillsRepo.LoadKillSourceCategoryKills: %w", err)
	}

	out := make([]port.KillSourceCategoryRow, 0, len(tally))
	for k, kills := range tally {
		out = append(out, port.KillSourceCategoryRow{
			XUID: k.xuid, Category: k.category, WeaponKey: k.weaponKey, Kills: kills,
		})
	}
	// Sortie deterministe (la map d'agregation ne l'est pas).
	sort.Slice(out, func(i, j int) bool {
		if out[i].XUID != out[j].XUID {
			return out[i].XUID < out[j].XUID
		}
		if out[i].Category != out[j].Category {
			return out[i].Category < out[j].Category
		}
		return out[i].WeaponKey < out[j].WeaponKey
	})
	return out, nil
}
