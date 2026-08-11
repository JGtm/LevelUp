package duckdb

// replay_map_repo.go — QUELLE CARTE A ÉTÉ JOUÉE, pour un match donné.
//
// POURQUOI CE REPO EXISTE. L'artefact de rejeu 2D ne porte AUCUNE identité de carte : il est
// écrit hors ligne à partir des seuls chunks du film, qui ne nomment pas la carte. Le fond de
// carte, lui, est indexé par MODULE (`ridgeline.png`), et la seule chaîne qui relie un nom
// affiché à un module est le catalogue de bornes (`filmdec.NormalizeMapName` +
// `map_quant_bounds.json`). Il manque donc le premier maillon — match -> nom de carte — et il
// n'existe qu'en base.
//
// POURQUOI PAS LE NOM DE L'EN-TÊTE DE LA MATCH VIEW. `MapUI` est LOCALISÉ (FR quand la page
// est en français) alors que le catalogue est indexé par nom canonique EN. Résoudre côté
// client reviendrait à chercher « Cliffhanger » avec la clé française du jour.

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"
)

// ErrMatchMapUnknown signale qu'aucun nom de carte n'est connu pour ce match.
var ErrMatchMapUnknown = errors.New("duckdb: carte inconnue pour ce match")

// ReplayMapRepo résout la carte d'un match (registre partagé + traductions d'assets).
type ReplayMapRepo struct {
	shared   SharedReader
	metadata *DB
}

// NewReplayMapRepo crée le repo. `metadataDB` peut être nil : la résolution retombe alors
// sur le libellé brut du registre.
func NewReplayMapRepo(shared SharedReader, metadataDB *DB) *ReplayMapRepo {
	return &ReplayMapRepo{shared: shared, metadata: metadataDB}
}

// MapNamesForMatch retourne les noms de carte CANDIDATS d'un match, du plus fiable au moins
// fiable, sans doublon.
//
// POURQUOI PLUSIEURS ET PAS UN SEUL. Deux sources existent, et aucune n'est bonne partout :
// `asset_translations` donne un nom canonique même quand `match_registry.map_name` porte un
// UUID brut (cas documenté en tête de MatchMetaRaw), mais sa cascade de langues peut rendre
// un libellé traduit quand l'anglais manque — et le catalogue de modules, lui, est indexé en
// anglais. Rendre les deux laisse l'appelant essayer la seconde quand la première ne résout
// rien, au lieu d'échouer sur un libellé qui n'était pas la bonne clé.
//
// ErrMatchMapUnknown quand rien n'est exploitable : l'appelant dégrade (pas de fond de
// carte), il ne devine pas.
func (r *ReplayMapRepo) MapNamesForMatch(ctx context.Context, matchID string) ([]string, error) {
	if r == nil || r.shared == nil {
		return nil, ErrMatchMapUnknown
	}
	db, release, err := r.shared.Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("replay map: lecteur shared indisponible: %w", err)
	}
	defer release()

	var rawName, assetID sql.NullString
	err = db.QueryRowContext(ctx,
		`SELECT map_name, map_id FROM match_registry WHERE match_id = ?`, matchID,
	).Scan(&rawName, &assetID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrMatchMapUnknown
	}
	if err != nil {
		return nil, fmt.Errorf("replay map: lecture match_registry: %w", err)
	}

	var out []string
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		for _, existing := range out {
			if strings.EqualFold(existing, s) {
				return
			}
		}
		out = append(out, s)
	}
	if id := strings.TrimSpace(assetID.String); id != "" && r.metadata != nil {
		meta := NewMetadataRepoFromDB(r.metadata)
		name, _, ok, resErr := meta.ResolveAssetName(ctx, "map", id, PreferredLangsForLocale("en"))
		if resErr != nil {
			slog.WarnContext(ctx, "replay map: résolution du nom d'asset échouée",
				"err", resErr, "match_id", matchID, "map_id", id)
		} else if ok {
			add(name)
		}
	}
	add(rawName.String)
	if len(out) == 0 {
		return nil, ErrMatchMapUnknown
	}
	return out, nil
}
