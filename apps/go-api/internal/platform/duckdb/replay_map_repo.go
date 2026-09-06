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

	"levelup/go-api/internal/port"
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

// MapKeysForMatch retourne les identités de carte d'un match : son map_id (asset UGC, clé
// du fond d'une carte Forge) et ses noms CANDIDATS, du plus fiable au moins fiable, sans
// doublon.
//
// POURQUOI PLUSIEURS NOMS ET PAS UN SEUL. Deux sources existent, et aucune n'est bonne
// partout : `asset_translations` donne un nom canonique même quand
// `match_registry.map_name` porte un UUID brut (cas documenté en tête de MatchMetaRaw),
// mais sa cascade de langues peut rendre un libellé traduit quand l'anglais manque — et le
// catalogue de modules, lui, est indexé en anglais. Rendre les deux laisse l'appelant
// essayer la seconde quand la première ne résout rien, au lieu d'échouer sur un libellé qui
// n'était pas la bonne clé.
//
// ErrMatchMapUnknown quand rien n'est exploitable — ni map_id ni nom : l'appelant dégrade
// (pas de fond de carte), il ne devine pas. Un map_id SANS nom reste exploitable : c'est la
// clé du fond des cartes Forge, et `map_name` est NULL sur certains matchs du registre.
func (r *ReplayMapRepo) MapKeysForMatch(ctx context.Context, matchID string) (port.MatchMapKeys, error) {
	if r == nil || r.shared == nil {
		return port.MatchMapKeys{}, ErrMatchMapUnknown
	}
	db, release, err := r.shared.Get(ctx)
	if err != nil {
		return port.MatchMapKeys{}, fmt.Errorf("replay map: lecteur shared indisponible: %w", err)
	}
	defer release()

	var rawName, assetID, pairName sql.NullString
	err = db.QueryRowContext(ctx,
		`SELECT map_name, map_id, pair_name FROM match_registry WHERE match_id = ?`, matchID,
	).Scan(&rawName, &assetID, &pairName)
	if errors.Is(err, sql.ErrNoRows) {
		return port.MatchMapKeys{}, ErrMatchMapUnknown
	}
	if err != nil {
		return port.MatchMapKeys{}, fmt.Errorf("replay map: lecture match_registry: %w", err)
	}

	// pair_name voyage BRUT (il peut être un UUID) : c'est le service qui le normalise —
	// la clé des rôles d'objectif du rejeu (lot 4), lue dans la même ligne que la carte.
	return r.assemblerIdentites(ctx, identitesBrutes{
		MapID:    assetID.String,
		RawName:  rawName.String,
		PairName: pairName.String,
		Ref:      matchID,
	})
}

// MapKeysForMap retourne les identités de carte à partir du SEUL map_id — la surface qui
// raisonne par CARTE (grille de l'onglet Tactique) n'a aucun match sous la main.
//
// La cascade est celle de MapKeysForMatch, à une nuance près : le nom BRUT ne peut plus
// venir de la ligne du match, il est cherché sur n'importe quel match du registre portant
// ce map_id. `map_name` est NULL sur une partie du registre — d'où le filtre : une ligne
// muette ne vaut pas moins qu'une autre, elle n'apporte simplement rien.
//
// PAS DE PairName : il qualifie un MATCH (le mode joué), jamais une carte.
//
// Un map_id absent du registre reste exploitable si le catalogue d'assets le nomme : la
// carte peut n'avoir jamais été jouée par CE joueur tout en existant dans le titre.
func (r *ReplayMapRepo) MapKeysForMap(ctx context.Context, mapID string) (port.MatchMapKeys, error) {
	mapID = strings.TrimSpace(mapID)
	if r == nil || r.shared == nil || mapID == "" {
		return port.MatchMapKeys{}, ErrMatchMapUnknown
	}
	db, release, err := r.shared.Get(ctx)
	if err != nil {
		return port.MatchMapKeys{}, fmt.Errorf("replay map: lecteur shared indisponible: %w", err)
	}
	defer release()

	var rawName sql.NullString
	err = db.QueryRowContext(ctx,
		`SELECT map_name FROM match_registry WHERE map_id = ? AND map_name IS NOT NULL LIMIT 1`, mapID,
	).Scan(&rawName)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return port.MatchMapKeys{}, fmt.Errorf("replay map: lecture match_registry par carte: %w", err)
	}
	return r.assemblerIdentites(ctx, identitesBrutes{MapID: mapID, RawName: rawName.String, Ref: mapID})
}

// identitesBrutes porte ce que la base a rendu, avant résolution des noms.
type identitesBrutes struct {
	MapID    string
	RawName  string
	PairName string
	// Ref n'entre dans AUCUNE clé : c'est la référence journalisée (match_id ou map_id)
	// quand la résolution d'un nom d'asset échoue.
	Ref string
}

// assemblerIdentites construit les MatchMapKeys : map_id, puis les noms candidats du plus
// fiable (catalogue d'assets, en anglais — la langue du catalogue de modules) au moins
// fiable (libellé brut du registre), sans doublon.
//
// UNE SEULE COPIE, et c'est délibéré : les deux entrées (par match, par carte) ne diffèrent
// que par la façon d'obtenir la ligne brute. Recopier la cascade donnerait deux définitions
// de « quels noms désignent cette carte », qui divergeraient au premier ajustement.
func (r *ReplayMapRepo) assemblerIdentites(ctx context.Context, brut identitesBrutes) (port.MatchMapKeys, error) {
	keys := port.MatchMapKeys{
		MapID:    strings.TrimSpace(brut.MapID),
		PairName: strings.TrimSpace(brut.PairName),
	}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		for _, existing := range keys.Names {
			if strings.EqualFold(existing, s) {
				return
			}
		}
		keys.Names = append(keys.Names, s)
	}
	if keys.MapID != "" && r.metadata != nil {
		meta := NewMetadataRepoFromDB(r.metadata)
		name, _, ok, resErr := meta.ResolveAssetName(ctx, "map", keys.MapID, PreferredLangsForLocale("en"))
		if resErr != nil {
			slog.WarnContext(ctx, "replay map: résolution du nom d'asset échouée",
				"err", resErr, "ref", brut.Ref, "map_id", keys.MapID)
		} else if ok {
			add(name)
		}
	}
	add(brut.RawName)
	if keys.MapID == "" && len(keys.Names) == 0 {
		return port.MatchMapKeys{}, ErrMatchMapUnknown
	}
	return keys, nil
}
