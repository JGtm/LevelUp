package main

// presence.go — L'OCCUPATION PAR EQUIPE, depuis les artefacts de rejeu.
//
// C'est le seul signal qui dise « tenir » plutot que « tirer ». Il coute cher : il n'existe
// que pour les matchs dont le film a ete decode (92 artefacts au 2026-09-20, contre ~2 800
// matchs porteurs de positions de kill).
//
// DEUX PIEGES MESURES SUR LES ARTEFACTS, ET CE QU'ON EN FAIT :
//
//   - LES POINTS D'UNE PISTE NE SONT PAS EQUIDISTANTS (t = 0, 13, 26, 49, 51, 60...).
//     Compter les points ferait peser une zone ou le decodeur a echantillonne serre autant
//     qu'une zone tenue. La duree d'un point est donc l'ECART A SON SUIVANT, converti en
//     millisecondes par `frameIntervalMs`. Le DERNIER point d'une piste n'a pas de suivant :
//     il compte pour un intervalle median, jamais pour zero (une piste d'un seul point
//     disparaitrait) ni pour la fin du match (une piste qui s'arrete a la mort du joueur
//     occuperait alors sa cellule de mort jusqu'a la fin).
//   - UNE PISTE EST UNE VIE, PAS UN JOUEUR : le meme xuid porte plusieurs pistes. C'est sans
//     consequence ici (on somme des durees), mais ca interdit de compter les pistes comme
//     des joueurs.
//
// L'ISSUE VIENT DE LA BASE, PAS DU FILM. Le film porte le designateur d'equipe, jamais le
// resultat ; `match_participants.outcome` par xuid est la seule source de « qui a gagne ».
// Une piste dont le xuid n'est pas au tableau du match (bot, xuid non resolu) est ECARTEE et
// COMPTEE — la ranger d'office du cote des perdants fabriquerait le signal qu'on mesure.

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis/powerpos"
	"levelup/go-api/internal/domain"
)

// intervalleParDefautMS est la cadence retenue quand l'artefact ne declare pas
// `frameIntervalMs` (artefacts anterieurs au champ). 100 ms est la valeur portee par les
// artefacts du parc qui le declarent ; l'employer en repli garde les durees comparables
// entre cartes, et le compteur `artefacts_sans_cadence` dit combien de matchs en dependent.
const intervalleParDefautMS = 100.0

// artefactRejeu : la seule part du document de rejeu que ce programme lit.
type artefactRejeu struct {
	MatchID         string        `json:"matchId"`
	FrameIntervalMS int           `json:"frameIntervalMs"`
	Tracks          []pisteRejeu2 `json:"tracks"`
}

// pisteRejeu2 est UNE VIE : le meme xuid en porte plusieurs dans un match.
type pisteRejeu2 struct {
	Team   int          `json:"team"`
	XUID   string       `json:"xuid"`
	Points []pointPiste `json:"points"`
}

// pointPiste est un echantillon de position ; `T` est un indice de FRAME, pas des
// millisecondes.
type pointPiste struct {
	T int     `json:"t"`
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

// accumulePresence lit les artefacts des matchs cibles et alimente l'occupation.
func accumulePresence(db *sql.DB, cibles []*Cible) error {
	matchs := matchsAvecArtefact(cibles)
	if len(matchs) == 0 {
		slog.Warn("mappower: aucun artefact de rejeu pour les cartes mesurees — occupation absente")
		return nil
	}
	issues, err := lisIssues(db, matchs)
	if err != nil {
		return err
	}
	for _, c := range cibles {
		for _, m := range triParID(c.Matchs) {
			if m.Artefact == "" {
				continue
			}
			if err := accumuleUnArtefact(c, m, issues[m.MatchID]); err != nil {
				c.ArtefactsEnEchec++
				slog.Warn("mappower: artefact de rejeu ecarte", "err", err,
					"match_id", m.MatchID, "carte", c.Carte)
				continue
			}
			c.ArtefactsLus++
		}
	}
	return nil
}

// matchsAvecArtefact rend les identifiants de match dont l'artefact est en cache.
func matchsAvecArtefact(cibles []*Cible) []string {
	var out []string
	for _, c := range cibles {
		for _, m := range c.Matchs {
			if m.Artefact != "" {
				out = append(out, m.MatchID)
			}
		}
	}
	sort.Strings(out)
	return out
}

// lisIssues charge l'issue par (match, xuid) pour les matchs donnes.
func lisIssues(db *sql.DB, matchs []string) (map[string]map[string]bool, error) {
	place := make([]string, len(matchs))
	args := make([]any, len(matchs))
	for i, m := range matchs {
		place[i], args[i] = "?", m
	}
	q := `SELECT match_id, xuid, COALESCE(outcome, 0) FROM match_participants
		WHERE match_id IN (` + strings.Join(place, ",") + `)`
	rows, err := db.QueryContext(context.Background(), q, args...)
	if err != nil {
		return nil, fmt.Errorf("lecture des issues par joueur : %w", err)
	}
	defer rows.Close()
	out := map[string]map[string]bool{}
	for rows.Next() {
		var matchID, xuid string
		var outcome int
		if err := rows.Scan(&matchID, &xuid, &outcome); err != nil {
			return nil, fmt.Errorf("lecture d'une issue : %w", err)
		}
		if out[matchID] == nil {
			out[matchID] = map[string]bool{}
		}
		if outcome == domain.OutcomeWin || outcome == domain.OutcomeLoss {
			out[matchID][xuid] = outcome == domain.OutcomeWin
		}
	}
	return out, rows.Err()
}

// accumuleUnArtefact projette les pistes d'un artefact en segments d'occupation.
func accumuleUnArtefact(c *Cible, m MatchCorpus, issues map[string]bool) error {
	blob, err := os.ReadFile(m.Artefact)
	if err != nil {
		return fmt.Errorf("artefact illisible (%s) : %w", m.Artefact, err)
	}
	var doc artefactRejeu
	if err := json.Unmarshal(blob, &doc); err != nil {
		return fmt.Errorf("artefact invalide (%s) : %w", m.Artefact, err)
	}
	if len(issues) == 0 {
		return fmt.Errorf("aucune issue connue au tableau du match %s", m.MatchID)
	}
	cadence := float64(doc.FrameIntervalMS)
	if cadence <= 0 {
		cadence = intervalleParDefautMS
	}
	for _, piste := range doc.Tracks {
		gagnant, connu := issues[piste.XUID]
		if !connu {
			c.PistesSansIssue++
			continue
		}
		accumuleUnePiste(c, m.MatchID, gagnant, cadence, piste)
	}
	return nil
}

// accumuleUnePiste convertit les points d'une piste en segments de duree.
func accumuleUnePiste(c *Cible, matchID string, gagnant bool, cadence float64, piste pisteRejeu2) {
	points := piste.Points
	if len(points) == 0 {
		return
	}
	ecarts := make([]float64, 0, len(points))
	for i := 0; i+1 < len(points); i++ {
		if d := float64(points[i+1].T - points[i].T); d > 0 {
			ecarts = append(ecarts, d)
		}
	}
	// Le dernier point prend l'ecart MEDIAN de sa piste (cf. l'en-tete). Une piste d'un
	// seul point prend une frame.
	dernier := powerpos.Mediane(ecarts)
	if dernier <= 0 {
		dernier = 1
	}
	for i, p := range points {
		frames := dernier
		if i+1 < len(points) {
			if d := float64(points[i+1].T - points[i].T); d > 0 {
				frames = d
			}
		}
		c.Acc.AjoutePresence(powerpos.PresenceSample{
			MatchID: matchID, Team: piste.Team, Gagnant: gagnant,
			X: p.X, Y: p.Y, DurMS: frames * cadence,
		})
		c.PointsLus++
	}
}

// triParID rend les matchs d'une cible dans un ordre stable.
func triParID(matchs map[string]MatchCorpus) []MatchCorpus {
	out := make([]MatchCorpus, 0, len(matchs))
	for _, m := range matchs {
		out = append(out, m)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].MatchID < out[j].MatchID })
	return out
}
