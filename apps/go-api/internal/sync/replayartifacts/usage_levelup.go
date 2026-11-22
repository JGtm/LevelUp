package replayartifacts

// usage_levelup.go — LES TROIS MESURES DE LEVELUP SUR LE FILM : bris de bouclier, recharges
// complètes (médaille Cible difficile), vengeances (médaille Vengeur). UNE SEULE FONCTION
// ([AttacherMesuresLevelUp]) pour le fil de l'eau (usage.go) ET le rattrapage
// (`levelup backfill-usage-summary`) : les deux écrivent le même résumé, à l'octet près.
//
// Le calcul vit dans `analysis/shieldcycle` (bris et recharges, entrée = pistes du document, lues par
// `replayshield`) et `analysis/revenge` (vengeances, entrée = fil des morts + équipes). Ce fichier
// ne fait que les JOINDRE aux lignes du résumé et dire ce qui est « non mesuré » (NULL).
//
// # NON MESURÉ, JAMAIS ZÉRO
//
//   - Bris et recharges : un match dont aucune piste ne porte une mesure `Sh` est non mesuré pour
//     tous ses joueurs (mode sans bouclier, rejeu sans la mesure). Dans un match mesuré, un joueur
//     qui a au moins une piste est mesuré, zéro compris (son bouclier n'a jamais bougé) ; un joueur
//     sans piste reste NULL.
//   - Vengeances : non mesurées sans fil des morts lu (`Frags.Read` faux) ou sans équipes lues
//     (les trahisons ne s'écarteraient pas) ; sinon un compte, zéro compris, pour chaque ligne.
//
// Un joueur sans ligne de résumé qui a au moins une mesure non nulle reçoit une ligne neuve (comme
// `replay.AttachShieldlessKills`) ; un joueur à zéro partout n'en reçoit pas : il se lit zéro
// parce que son match est mesuré par ses autres lignes.

import (
	"cmp"
	"context"
	"database/sql"
	"fmt"
	"slices"
	"strings"

	"levelup/go-api/internal/analysis/revenge"
	"levelup/go-api/internal/analysis/shieldcycle"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/replayshield"

	"github.com/JGtm/halo-infinite-film-decoder/replay"
)

// UsageLevelUpRev est la révision du calcul des trois colonnes (bris, recharges, vengeances),
// rangée dans `match_usage_films.levelup_rev` et comparée par `levelup backfill-usage-summary` :
// un match dont la passe courante porte une autre révision est re-dérivé. À CHANGER dès qu'une
// règle de `analysis/shieldcycle` ou `analysis/revenge`, ou leur jointure ici, change de résultat.
const UsageLevelUpRev = "lu1"

// EntreesMesuresUsage : ce que les mesures lisent en base, pour UN match, avant tout writer.
type EntreesMesuresUsage struct {
	// Frags : le fil des morts du match (Read faux = non lu).
	Frags replay.ShieldlessKillsInput
	// Equipes : l'équipe de chaque xuid (`match_participants.team_id`). Vide = non lue.
	Equipes map[string]int
}

// AttacherMesuresLevelUp calcule les mesures de LevelUp d'un match, ajoute au résumé les lignes
// de joueur qui en manquent (ordre des xuid) et rend ce que le persister range à côté du résumé.
// Un document nil ne produit rien (révision comprise : la passe reste re-dérivable).
func AttacherMesuresLevelUp(s *replay.UsageSummary, doc *replay.ReplayDocument, in EntreesMesuresUsage) persist.UsageLevelUp {
	if s == nil || doc == nil {
		return persist.UsageLevelUp{}
	}
	m := mesurerMatch(doc, in)
	out := persist.UsageLevelUp{Rev: UsageLevelUpRev, Players: map[string]persist.UsageLevelUpPlayer{}}
	seen := make(map[string]bool, len(s.Players))
	for i := range s.Players {
		x := s.Players[i].XUID
		seen[x] = true
		out.Players[x] = m.joueur(x)
	}
	added := false
	for _, x := range m.candidats() {
		if seen[x] {
			continue
		}
		if v := m.joueur(x); aUneValeurNonNulle(v) {
			out.Players[x] = v
			s.Players = append(s.Players, replay.UsagePlayerSummary{XUID: x})
			added = true
		}
	}
	if added {
		slices.SortFunc(s.Players, func(a, b replay.UsagePlayerSummary) int { return cmp.Compare(a.XUID, b.XUID) })
	}
	return out
}

// mesuresMatch : les calculs d'un match, avant leur jointure aux lignes du résumé.
type mesuresMatch struct {
	cycles         map[string][]shieldcycle.Event
	tracked        map[string]bool // xuid ayant au moins une piste dans le document
	vengeances     map[string]int
	revengeMeasure bool
}

func mesurerMatch(doc *replay.ReplayDocument, in EntreesMesuresUsage) *mesuresMatch {
	lives := replayshield.Lives(doc)
	m := &mesuresMatch{cycles: shieldcycle.Cycles(lives), tracked: make(map[string]bool, len(lives))}
	for i := range lives {
		if lives[i].XUID != "" {
			m.tracked[lives[i].XUID] = true
		}
	}
	if in.Frags.Read && len(in.Equipes) > 0 {
		m.revengeMeasure = true
		m.vengeances = revenge.Count(fragsDeVengeance(in.Frags.Kills), in.Equipes)
	}
	return m
}

// joueur rend les mesures d'un xuid ; une mesure impossible reste nil (NULL).
func (m *mesuresMatch) joueur(xuid string) persist.UsageLevelUpPlayer {
	var out persist.UsageLevelUpPlayer
	if len(m.cycles) > 0 && m.tracked[xuid] {
		b, r := shieldcycle.Totals(m.cycles[xuid])
		out.ShieldBreaks, out.ShieldRecharges = &b, &r
	}
	if m.revengeMeasure {
		v := m.vengeances[xuid]
		out.RevengeKills = &v
	}
	return out
}

// candidats : les xuid qui peuvent porter une mesure non nulle (pistes et vengeurs), triés.
func (m *mesuresMatch) candidats() []string {
	set := make(map[string]bool, len(m.tracked)+len(m.vengeances))
	for x := range m.tracked {
		set[x] = true
	}
	for x := range m.vengeances {
		set[x] = true
	}
	out := make([]string, 0, len(set))
	for x := range set {
		out = append(out, x)
	}
	slices.Sort(out)
	return out
}

// aUneValeurNonNulle dit si au moins une des trois mesures vaut plus que zéro.
func aUneValeurNonNulle(m persist.UsageLevelUpPlayer) bool {
	for _, p := range []*int{m.ShieldBreaks, m.ShieldRecharges, m.RevengeKills} {
		if p != nil && *p > 0 {
			return true
		}
	}
	return false
}

// fragsDeVengeance convertit les frags du fil des morts pour `analysis/revenge`.
func fragsDeVengeance(kills []replay.ShieldlessKillRef) []revenge.Frag {
	out := make([]revenge.Frag, len(kills))
	for i, k := range kills {
		out[i] = revenge.Frag{Killer: k.Killer, Victim: k.Victim, TimeMS: k.TimeMS}
	}
	return out
}

// LireEquipesDesMatchs lit, pour un lot de matchs, l'équipe de chaque participant
// (`match_participants.team_id`). Une entrée par match qui a au moins un participant d'équipe
// connue ; un match absent du résultat a ses équipes « non lues ». EXPORTÉE pour le rattrapage,
// qui doit lire les équipes comme le fil de l'eau.
func LireEquipesDesMatchs(ctx context.Context, db *sql.DB, ids []string) (map[string]map[string]int, error) {
	out := make(map[string]map[string]int, len(ids))
	if len(ids) == 0 {
		return out, nil
	}
	args := make([]any, 0, len(ids))
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := db.QueryContext(ctx, `SELECT match_id, xuid, team_id
		FROM match_participants
		WHERE team_id IS NOT NULL AND COALESCE(xuid, '') <> ''
		  AND match_id IN (?`+strings.Repeat(",?", len(ids)-1)+`)`, args...)
	if err != nil {
		return nil, fmt.Errorf("mesures d'usage: lecture des equipes: %w", err)
	}
	defer func() { _ = rows.Close() }()
	for rows.Next() {
		var id, xuid string
		var team int
		if err := rows.Scan(&id, &xuid, &team); err != nil {
			return nil, fmt.Errorf("mesures d'usage: scan des equipes: %w", err)
		}
		if out[id] == nil {
			out[id] = map[string]int{}
		}
		out[id][xuid] = team
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("mesures d'usage: lignes des equipes: %w", err)
	}
	return out, nil
}
