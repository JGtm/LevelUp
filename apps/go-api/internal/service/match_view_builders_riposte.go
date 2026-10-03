// Package service — match_view_builders_riposte.go : le bloc « Riposte » de l'onglet
// Combat (décision D22-2, 2026-09-21).
//
// « La mort de X a été vengée dans les 5 s, par Y » n'était rendu nulle part sur la page,
// alors que la matière y est déjà : Nemesis / Souffre-douleur donne un SOLDE de duel (ni
// fenêtre de temps, ni ordre des événements), et le graphe des assistances un autre sujet.
//
// ─── AUCUNE REQUÊTE NOUVELLE, ET AUCUN N+1 ────────────────────────────────────────────
//
// Les deux entrées sont DÉJÀ chargées par la vue match : les paires killer→victim de Q20
// (`match_kill_events_latest`, une ligne par mort avec son instant) et le scoreboard, qui
// porte le `team_id` de chaque joueur. Le camp vient donc du tableau des scores, la même
// source que tous les autres blocs de la page — le demander au journal des morts en aurait
// donné une seconde.
//
// ─── DES COMPTES, JAMAIS UN TAUX ──────────────────────────────────────────────────────
//
// Décision D21 : un taux sur 11 morts est du bruit affiché avec deux décimales. Ce bloc ne
// publie que des comptes exhaustifs du match et les couples nommés. La seule réserve est
// la couverture du film — sans lignes de journal, pas de bloc (nil), jamais une section
// qui disparaît sans rien dire.
package service

import (
	"sort"

	"levelup/go-api/internal/analysis/coordination"
	"levelup/go-api/internal/domain"
)

// riposteMatchKey — la clé de match utilisée pour appeler `coordination.Echanges`.
//
// Q20 SCOPE DÉJÀ SA REQUÊTE PAR match_id et laisse donc `KVPairRaw.MatchID` VIDE sur le
// chemin mono-match (cf. sa doctrine). Les grouper sous une clé explicite rend le calcul
// indépendant de ce détail : si un jour la lecture peuplait le champ, rien ne changerait
// — et rien ne traverse de frontière de match, puisqu'il n'y en a qu'un.
const riposteMatchKey = "match"

// buildMatchRiposte assemble le bloc `combat_tab.riposte`.
//
// nil quand le match ne porte aucune ligne de journal : il n'y a pas d'ordre des morts,
// donc rien à dire. C'est le MÊME arbitrage d'émission que buildAssistPairs.
func buildMatchRiposte(
	kvPairs []domain.KVPairRaw, scoreboard []domain.ScoreboardRaw,
) *domain.MatchRiposteBlock {
	if len(kvPairs) == 0 {
		return nil
	}
	equipes := domain.EquipesParMatch{riposteMatchKey: campsDuScoreboard(scoreboard)}
	kills := make([]domain.KillEvent, 0, len(kvPairs))
	nomVictime := make(map[string]string, len(kvPairs))
	for _, kv := range kvPairs {
		kills = append(kills, domain.KillEvent{
			MatchID:    riposteMatchKey,
			KillerXUID: kv.KillerXUID,
			VictimXUID: kv.VictimXUID,
			TimeMs:     kv.TimeMS,
		})
		// Le gamertag de la victime est NOT NULL au DDL : c'est lui qui nomme une victime
		// BOT, dont le xuid est vide. Il sert de repli quand le scoreboard ne connaît pas
		// l'identité (joueur parti avant la fin).
		if kv.VictimXUID != "" && nomVictime[kv.VictimXUID] == "" {
			nomVictime[kv.VictimXUID] = kv.VictimGT
		}
	}
	morts := coordination.Echanges(kills, equipes).Morts
	noms := nomsDuScoreboard(scoreboard)
	out := &domain.MatchRiposteBlock{
		FenetreMs:      coordination.FenetreEchangeMs,
		MeasuredDeaths: len(kvPairs),
		Deaths:         mortsRiposte(morts, equipes[riposteMatchKey], noms, nomVictime),
	}
	out.Players = joueursRiposte(morts, equipes[riposteMatchKey], noms, scoreboard)
	return out
}

// campsDuScoreboard rend xuid -> camp. Un joueur sans xuid (bot) ou sans `team_id` (FFA,
// ligne incomplète) n'a pas de camp : il ne peut alors ni venger ni être vengé, et lui en
// deviner un fabriquerait des échanges.
func campsDuScoreboard(scoreboard []domain.ScoreboardRaw) map[string]int {
	out := make(map[string]int, len(scoreboard))
	for _, s := range scoreboard {
		if s.XUID == "" || s.TeamID == nil {
			continue
		}
		out[s.XUID] = *s.TeamID
	}
	return out
}

// nomsDuScoreboard rend xuid -> gamertag. Source unique des noms du bloc, comme pour les
// paires d'assistance : le film écrit le gamertag capté à l'enregistrement, le scoreboard
// sert celui de l'API (alias compris), et mélanger les deux affiche un joueur sous deux
// orthographes.
func nomsDuScoreboard(scoreboard []domain.ScoreboardRaw) map[string]string {
	out := make(map[string]string, len(scoreboard))
	for _, s := range scoreboard {
		if s.XUID != "" && s.Gamertag != "" {
			out[s.XUID] = s.Gamertag
		}
	}
	return out
}

// mortsRiposte projette chaque mort suivie en ligne publiable.
func mortsRiposte(
	morts []domain.MortSuivie, camps map[string]int, noms, nomVictime map[string]string,
) []domain.MatchRiposteDeath {
	out := make([]domain.MatchRiposteDeath, 0, len(morts))
	for _, m := range morts {
		d := domain.MatchRiposteDeath{
			VictimXUID:     m.VictimeXUID,
			VictimGamertag: nomOuRepli(noms, nomVictime, m.VictimeXUID),
			KillerXUID:     m.TueurXUID,
			TimeMs:         m.TimeMs,
			Vengeable:      m.Vengeable,
			Avenged:        m.Vengee,
		}
		if team, ok := camps[m.VictimeXUID]; ok {
			t := team
			d.VictimTeamID = &t
		}
		if m.Vengee {
			delai := m.DelaiMs
			d.DelaiMs = &delai
			d.AvengerXUID = m.VengeurXUID
			d.AvengerGamertag = noms[m.VengeurXUID]
		}
		out = append(out, d)
	}
	return out
}

// nomOuRepli : le scoreboard d'abord, le nom écrit par le journal ensuite. Jamais un xuid
// recopié dans un champ de nom — le front a son masque « Joueur #### ».
func nomOuRepli(noms, repli map[string]string, xuid string) string {
	if gt := noms[xuid]; gt != "" {
		return gt
	}
	return repli[xuid]
}

// joueursRiposte compte, par joueur, ses morts vengées par son camp et les ripostes qu'il
// a portées.
//
// LES JOUEURS SANS AUCUN DES DEUX ÉVÉNEMENTS SORTENT QUAND MÊME, à zéro, dès qu'ils sont
// au scoreboard : les deux graphes du bloc doivent montrer TOUT le camp, sinon un joueur
// qui n'a ni vengé ni été vengé disparaîtrait de la ligne d'en face et changerait la
// hauteur apparente des autres. L'ordre est celui du scoreboard.
func joueursRiposte(
	morts []domain.MortSuivie, camps map[string]int, noms map[string]string,
	scoreboard []domain.ScoreboardRaw,
) []domain.MatchRiposteePlayer {
	vengees := map[string]int{}
	portees := map[string]int{}
	for _, m := range morts {
		if !m.Vengee {
			continue
		}
		vengees[m.VictimeXUID]++
		portees[m.VengeurXUID]++
	}
	out := make([]domain.MatchRiposteePlayer, 0, len(scoreboard))
	for _, s := range scoreboard {
		if s.XUID == "" {
			continue
		}
		j := domain.MatchRiposteePlayer{
			XUID:          s.XUID,
			Gamertag:      nomOuRepli(noms, nil, s.XUID),
			DeathsAvenged: vengees[s.XUID],
			Ripostes:      portees[s.XUID],
		}
		if team, ok := camps[s.XUID]; ok {
			t := team
			j.TeamID = &t
		}
		out = append(out, j)
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if (a.TeamID == nil) != (b.TeamID == nil) {
			return a.TeamID != nil
		}
		if a.TeamID != nil && *a.TeamID != *b.TeamID {
			return *a.TeamID < *b.TeamID
		}
		return a.XUID < b.XUID
	})
	return out
}
