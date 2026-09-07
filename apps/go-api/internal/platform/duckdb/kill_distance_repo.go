// Package duckdb — kill_distance_repo.go : implémentation DuckDB du loader
// « distance par arme, par joueur » pour UN match (POC LOT G.3, 2026-08-30,
// plan .ai/PLAN_RETOURS_UTILISATEUR_2026-08-29.md §3bis DEC-8).
//
// Source : la jointure « mort mesurée » — `match_kill_events_latest` × la table
// de positions — qui ne s'écrit plus ici : elle vit dans kill_measured.go depuis
// le lot 3 du plan .ai/PLAN_DUELS_PORTEE_2026-09-06.md (règle n°6 du dépôt : à la
// troisième copie on centralise ET on migre les copies). Ce fichier n'apporte
// donc plus que SA clause de portée (`killDistanceWhere`) et sa résolution
// d'armes. Les gardes (règle ART n°2 — vues `_latest` jamais les tables brutes,
// `publishable`, unanimité sur `source_tag` comme Q21b) sont celles de l'helper.
//
// Ce repo reste de la même famille que KillSourceClassRepo (kills-hors-arme) et
// Q21b/Q21c (kill feed du rejeu) : même classificateur injecté
// (port.KillSourceClassifier).
//
// CE QUE CE REPO AJOUTE PAR RAPPORT À KillSourceClassRepo : celui-là résout
// UNIQUEMENT les sources HORS ARSENAL (anti-double-comptage avec
// weapon_kills, cf. son en-tête) — ce repo-ci veut TOUTES les armes (arme à
// feu du registre comme hors arsenal), parce qu'il ne recoupe jamais avec
// weapon_kills : chaque ligne qu'il émet porte déjà sa propre mesure de
// distance, il n'y a rien à additionner par-dessus une autre voie de comptage.
//
// PÉRIMÈTRE FERMÉ (cadrage utilisateur, DEC-8) : un seul match_id, jamais un
// agrégat multi-matchs ; le TUEUR seulement, jamais l'assistant (ni son arme,
// ni sa distance) ; aucune colonne de distance stockée — la distance (hypot 3D)
// se calcule ICI, à la lecture, jamais en base (doctrine G.0 : « on stocke une
// mesure, pas une résolution améliorable »).
package duckdb

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"time"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/port"
)

// KillDistanceRepo implémente port.KillDistanceRepository.
type KillDistanceRepo struct {
	pdb *PlayerDB
	// classifier traduit une source de dégât en clé de registre. nil = ce
	// titre n'en fournit pas -> le repo rend zéro ligne (état nominal, même
	// doctrine que KillSourceClassRepo).
	classifier port.KillSourceClassifier
}

// NewKillDistanceRepo crée un KillDistanceRepo lié à un PlayerDB.
//
// classifier peut être nil : voir l'en-tête du fichier.
func NewKillDistanceRepo(pdb *PlayerDB, classifier port.KillSourceClassifier) *KillDistanceRepo {
	return &KillDistanceRepo{pdb: pdb, classifier: classifier}
}

// killDistanceAgg : compteur intermédiaire par (xuid, weapon_key).
type killDistanceAgg struct {
	kills int
	sum   float64
	min   float64
	max   float64
}

// LoadMatch charge les distances mesurées par (xuid, weapon_key) pour un match.
//
// Jamais de scan complet : matchID est obligatoire (une lecture sans filtre
// balaierait shared.kill_positions/match_kill_events en entier).
func (r *KillDistanceRepo) LoadMatch(ctx context.Context, matchID string) ([]domain.MatchKillDistancePlayer, error) {
	if matchID == "" {
		return nil, fmt.Errorf("KillDistanceRepo.LoadMatch: matchID vide")
	}
	if r.classifier == nil {
		slog.DebugContext(ctx, "KillDistanceRepo: no classifier for title, nothing to load",
			"match_id", matchID)
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	measured, err := r.queryMeasuredKills(ctx, matchID)
	if err != nil {
		if isTableNotFoundErr(err) {
			// La table est NOMMEE par la constante du proprietaire de la jointure, jamais par un
			// litteral : le garde-rail kill_measured_guard_test.go interdit ces noms hors de
			// kill_measured.go, et un nom recopie ici survivrait a un renommage la-bas.
			slog.DebugContext(ctx, "KillDistanceRepo: positions/kill-feed table missing",
				"match_id", matchID, "table", string(positionsAtKill))
			return nil, games.ErrCapabilityNotSupported
		}
		slog.ErrorContext(ctx, "KillDistanceRepo: query failed", "match_id", matchID, "err", err)
		return nil, fmt.Errorf("KillDistanceRepo.LoadMatch: %w", err)
	}
	if len(measured) == 0 {
		return nil, nil
	}
	return r.resolveRows(ctx, measured), nil
}

// killDistanceWhere : la portée de CE lecteur — un seul match, jamais un scan.
// Le reste de la jointure (gardes `publishable`, d'unanimité et d'unicité du
// frag, NULL-checks, groupement) vit dans kill_measured.go, partagé avec
// WeaponRangeRepo.
//
// GARDE D'UNICITÉ DU FRAG (2026-09-06, lot 3) : deux morts au même (match,
// tueur, instant) — MÊME à la même arme — ne publient plus rien. Le POC les
// mesurait avant, sur une position arbitraire, faute de savoir laquelle des
// deux victimes la ligne de positions plaçait. Impact mesuré sur le corpus :
// NUL (0 groupe multi-victimes sur 138 293 événements). Changement statué et
// accepté au résidu 4.0c du plan .ai/PLAN_DUELS_PORTEE_2026-09-06.md.
const killDistanceWhere = `e.match_id = ?`

// killDistanceFragScope : le MÊME match, borné dans la sous-requête `fragSolo`.
// Sans lui, DuckDB balaie la vue entière pour juger l'unicité du frag (résidu
// 4.0a). Ses paramètres sont liés AVANT ceux de killDistanceWhere.
const killDistanceFragScope = `s.match_id = ?`

// killDistanceQueryFor compose la lecture d'UN match : le texte SQL ET ses
// paramètres, dans l'ordre du texte.
//
// UN SEUL SITE DE COMPOSITION, ET C'EST LUI QUE LE TEST DE PLAN EXERCE. La
// tentation est d'écrire ce couple deux fois — ici et dans
// kill_measured_scope_test.go — mais un test qui RECOMPOSE la requête à partir
// des constantes ne juge plus le lecteur de production : ramener le scope de la
// sous-requête à `TRUE` à cet endroit le laisserait vert (constat F1 de la revue
// adversariale du lot 4, 2026-09-06). Le test appelle donc cette fonction.
//
// matchID est lié DEUX FOIS : la sous-requête `fragSolo` d'abord (elle précède
// la clause WHERE dans le texte SQL), la portée externe ensuite.
func killDistanceQueryFor(matchID string) (string, []any) {
	return measuredKillsQuery(positionsAtKill, killDistanceFragScope, killDistanceWhere),
		[]any{matchID, matchID}
}

// queryMeasuredKills lit les morts mesurées du match via l'helper canonique.
func (r *KillDistanceRepo) queryMeasuredKills(ctx context.Context, matchID string) ([]killMeasured, error) {
	db, release, err := r.pdb.SharedReadDB().Get(ctx)
	if err != nil {
		return nil, fmt.Errorf("shared reader: %w", err)
	}
	defer release()

	q, args := killDistanceQueryFor(matchID)
	return queryMeasuredKills(ctx, db, q, args, "KillDistanceRepo("+matchID+")")
}

// resolveRows traduit source_tag -> weapon_key (classificateur), agrège par
// (xuid, weapon_key), puis habille le résultat de son libellé.
func (r *KillDistanceRepo) resolveRows(ctx context.Context, measured []killMeasured) []domain.MatchKillDistancePlayer {
	type key struct {
		xuid, weaponKey string
	}
	agg := map[key]*killDistanceAgg{}
	keysSeen := map[string]bool{}
	weaponKeys := make([]string, 0)

	for _, m := range measured {
		wk, ok := r.classifier.KillSourceRegistryKey(m.sourceTag)
		if !ok {
			continue // source hors registre : cette mort n'entre pas dans le POC (jamais de devinette)
		}
		if !keysSeen[wk] {
			keysSeen[wk] = true
			weaponKeys = append(weaponKeys, wk)
		}
		k := key{xuid: m.killerXUID, weaponKey: wk}
		a, exists := agg[k]
		if !exists {
			a = &killDistanceAgg{min: m.distanceM, max: m.distanceM}
			agg[k] = a
		}
		a.kills++
		a.sum += m.distanceM
		if m.distanceM < a.min {
			a.min = m.distanceM
		}
		if m.distanceM > a.max {
			a.max = m.distanceM
		}
	}
	if len(agg) == 0 {
		return nil
	}

	meta := resolveWeaponKeyLabelsAny(ctx, r.pdb.Metadata, r.pdb.TitleSlug, weaponKeys)

	byPlayer := map[string][]domain.MatchKillDistanceWeapon{}
	for k, a := range agg {
		m := meta[k.weaponKey]
		byPlayer[k.xuid] = append(byPlayer[k.xuid], domain.MatchKillDistanceWeapon{
			WeaponKey:     k.weaponKey,
			Label:         m.label,
			LabelEN:       m.labelEN,
			MeasuredKills: a.kills,
			AvgDistanceM:  a.sum / float64(a.kills),
			MinDistanceM:  a.min,
			MaxDistanceM:  a.max,
		})
	}

	out := make([]domain.MatchKillDistancePlayer, 0, len(byPlayer))
	for xuid, weapons := range byPlayer {
		sortKillDistanceWeapons(weapons)
		out = append(out, domain.MatchKillDistancePlayer{XUID: xuid, Weapons: weapons})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].XUID < out[j].XUID })
	return out
}

// sortKillDistanceWeapons rend la sortie DÉTERMINISTE (map d'agrégation non
// ordonnée) : kills mesurés décroissants, puis weapon_key — jamais ambigu.
func sortKillDistanceWeapons(weapons []domain.MatchKillDistanceWeapon) {
	sort.Slice(weapons, func(i, j int) bool {
		if weapons[i].MeasuredKills != weapons[j].MeasuredKills {
			return weapons[i].MeasuredKills > weapons[j].MeasuredKills
		}
		return weapons[i].WeaponKey < weapons[j].WeaponKey
	})
}
