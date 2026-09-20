package main

// rangs.go — LE RANG DU TUEUR AU MOMENT DU MATCH, et la ponderation qui en decoule
// (item 2bis.B du plan des positions de force, D10a).
//
// # OU VIT LE RANG D'UN JOUEUR QUELCONQUE POUR UN MATCH DONNE
//
// Recensement du 2026-09-20 sur le depot : il n'existe QU'UNE source par-match ET
// par-joueur, la vue `match_csrs_latest` de la base PARTAGEE, de cle `(match_id, xuid)`.
// Les autres candidates ne repondent pas a la question :
//
//   - `match_skill_rank_latest` vit dans la base JOUEUR et n'a PAS de colonne `xuid` :
//     une ligne = un match, et c'est implicitement le rang du proprietaire de la base. Elle
//     ne peut donc rien dire du rang d'un adversaire, qui est precisement ce qu'il faut
//     pour ponderer les kills d'un corpus multi-joueurs.
//   - `player_csr_snapshots_latest` est un etat par playlist et par saison, sans `match_id`.
//   - `player_skill_state_v2_latest` (partagee) est l'etat COURANT par (xuid, groupe de
//     playlist) : elle donne le rang d'aujourd'hui, pas celui du jour du match.
//   - `match_participants` ne porte aucun CSR : son `rank` est le classement de fin de
//     partie, et ses `team_mmr` / `enemy_mmr` sont des moyennes d'EQUIPE.
//
// LA COUVERTURE EST DONC STRUCTURELLEMENT PARTIELLE : `match_csrs` n'est alimentee que par
// le payload skill des matchs CLASSES. Dans une playlist sociale, aucun joueur du lobby n'a
// de rang en base. Le chiffre mesure est ecrit au rapport, carte par carte.
//
// Lecture par la VUE `_latest` uniquement (regle ART n°2, ADR 0026), sur la connexion deja
// ouverte en `OpenReadForQuery`. Aucune ecriture.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"strings"

	"levelup/go-api/internal/analysis/powerpos"
)

// Rangs indexe le rang competitif par (match, joueur) et porte la ponderation qui s'en
// deduit.
type Rangs struct {
	parCle map[string]float64
	// matchsConnus : les matchs du corpus pour lesquels au moins un rang est en base.
	matchsConnus map[string]struct{}

	// Ponderation : les bornes mesurees sur TOUT le corpus du titre (p10 / mediane / p90
	// des rangs vus), et non sur les seules cartes mesurees : le poids d'un kill ne doit
	// pas dependre de la liste passee a --cartes (mesure du 2026-09-20 : p10 1 241 sur
	// trois cartes, 1 248 sur douze — un meme kill aurait pese differemment selon la
	// passe).
	Ponderation powerpos.Ponderation

	// Lignes : lignes retenues (rang connu sur un match des cartes mesurees).
	Lignes int
	// Matchs : matchs des cartes mesurees pour lesquels au moins un rang est connu.
	Matchs int
	// LignesCorpus : lignes de la vue, toutes cartes du titre — l'assiette des quantiles.
	LignesCorpus int
	// P10 / P50 / P90 : les quantiles mesures sur LignesCorpus, repris au rapport.
	P10, P50, P90 float64
}

// poidsBas et poidsHaut encadrent la rampe de ponderation. Symetriques autour de 1,0, qui
// est aussi le poids d'un rang INCONNU : la ponderation penche le corpus sans le deplacer
// en moyenne, et une playlist sociale (ou personne n'a de rang) n'est ni favorisee ni
// effacee.
const (
	poidsBas  = 0.5
	poidsHaut = 1.5
)

// ChargeRangs lit `match_csrs_latest` — TOUTE la vue pour l'assiette des quantiles, et
// n'indexe que les matchs des cartes mesurees — puis en deduit la ponderation. L'absence
// de la vue n'est pas fatale : la mesure se poursuit a ponderation neutre, et le fait est
// journalise — un signal absent ne doit pas priver des autres.
func ChargeRangs(db *sql.DB, index map[string]*Cible) *Rangs {
	r := &Rangs{parCle: map[string]float64{}, matchsConnus: map[string]struct{}{},
		Ponderation: powerpos.PonderationNeutre()}
	const q = `SELECT match_id, xuid, rating_value FROM match_csrs_latest
		WHERE rating_value IS NOT NULL AND rating_value > 0`
	rows, err := db.QueryContext(context.Background(), q)
	if err != nil {
		slog.Warn("mappower: rangs par match illisibles — ponderation neutre", "err", err)
		return r
	}
	defer rows.Close()

	matchs := map[string]struct{}{}
	var corpus []float64
	for rows.Next() {
		var matchID, xuid string
		var valeur float64
		if err := rows.Scan(&matchID, &xuid, &valeur); err != nil {
			slog.Error("mappower: lecture d'un rang par match", "err", err)
			return r
		}
		if xuid == "" {
			continue
		}
		corpus = append(corpus, valeur)
		if index[matchID] == nil {
			continue
		}
		r.parCle[cleRang(matchID, xuid)] = valeur
		matchs[matchID] = struct{}{}
		r.Lignes++
	}
	if err := rows.Err(); err != nil {
		slog.Error("mappower: parcours des rangs par match", "err", err)
		return r
	}
	r.Matchs, r.LignesCorpus = len(matchs), len(corpus)
	r.matchsConnus = matchs
	r.etablitPonderation(corpus)
	return r
}

// etablitPonderation mesure les quantiles du corpus et en fait la rampe de poids. Sous
// deux rangs distincts, la ponderation reste neutre : une rampe sur une seule valeur
// n'aurait aucun sens.
func (r *Rangs) etablitPonderation(valeurs []float64) {
	if len(valeurs) < 2 {
		slog.Warn("mappower: pas assez de rangs pour ponderer", "rangs", len(valeurs))
		return
	}
	r.P10 = powerpos.Quantile(valeurs, 0.10)
	r.P50 = powerpos.Quantile(valeurs, 0.50)
	r.P90 = powerpos.Quantile(valeurs, 0.90)
	if r.P90 <= r.P10 {
		slog.Warn("mappower: rangs du corpus trop plats pour ponderer", "p10", r.P10, "p90", r.P90)
		return
	}
	r.Ponderation = powerpos.Ponderation{
		RangBas: r.P10, RangHaut: r.P90, RangMedian: r.P50,
		PoidsBas: poidsBas, PoidsHaut: poidsHaut, PoidsInconnu: 1,
	}
	slog.Info("mappower: ponderation par rang etablie", "lignes_cartes_mesurees", r.Lignes,
		"matchs_cartes_mesurees", r.Matchs, "lignes_corpus", r.LignesCorpus,
		"p10", r.P10, "p50", r.P50, "p90", r.P90)
}

// MatchConnu dit si au moins un rang est connu pour ce match.
func (r *Rangs) MatchConnu(matchID string) bool {
	if r == nil {
		return false
	}
	_, ok := r.matchsConnus[matchID]
	return ok
}

// De rend le rang du joueur pour ce match, ou nil s'il est inconnu.
func (r *Rangs) De(matchID, xuid string) *float64 {
	if r == nil || xuid == "" {
		return nil
	}
	v, ok := r.parCle[cleRang(matchID, xuid)]
	if !ok {
		return nil
	}
	return &v
}

// cleRang assemble la cle d'index. Le separateur est un caractere que ni un identifiant de
// match ni un xuid ne portent.
func cleRang(matchID, xuid string) string {
	var b strings.Builder
	b.Grow(len(matchID) + len(xuid) + 1)
	b.WriteString(matchID)
	b.WriteByte('|')
	b.WriteString(xuid)
	return b.String()
}

// Resume rend une ligne lisible pour le rapport de mesure.
func (r *Rangs) Resume() string {
	if r == nil || r.Lignes == 0 {
		return "aucun rang par match disponible sur le corpus mesure"
	}
	return fmt.Sprintf("%d rangs sur %d matchs des cartes mesurees ; rampe mesuree sur les"+
		" %d rangs de tout le titre : p10 %.0f, mediane %.0f, p90 %.0f",
		r.Lignes, r.Matchs, r.LignesCorpus, r.P10, r.P50, r.P90)
}
