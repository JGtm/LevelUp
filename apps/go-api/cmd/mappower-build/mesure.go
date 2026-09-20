package main

// mesure.go — l'ORCHESTRATION de la passe de mesure : choisir les cartes, accumuler les
// deux sources (eliminations en base, occupation dans les artefacts), ecrire les sorties.
//
// UNE SEULE PASSE SUR `kill_positions_latest`, TOUTES CARTES A LA FOIS. La vue porte
// 138 382 lignes ; une requete par carte les relirait autant de fois qu'il y a de cartes
// mesurees, pour un filtre que le registre deja charge sait appliquer en memoire.

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"levelup/go-api/internal/analysis/powerpos"
	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
)

// Cible est une (carte, axe) mesuree.
type Cible struct {
	Carte  string
	Axe    string
	Matchs map[string]MatchCorpus
	Acc    *powerpos.Accumulateur

	// Variantes : barycentre des positions de tueur PAR NOM AFFICHE, pour controler que le
	// rabotage des suffixes de variante agrege bien deux fois la meme carte
	// (cf. controle_variantes.go).
	Variantes map[string]*statVariante
	// EcartVariantesM est le resultat de ce controle, repris au rapport de mesure.
	EcartVariantesM float64

	// Scorees / Positions : le resultat du reglage choisi (--reglage) applique a la carte.
	// Calcules une fois et partages entre le CSV, les images et le rapport.
	Scorees   []powerpos.CelluleScoree
	Positions []powerpos.Position

	// Rangs : l'index des rangs par (match, joueur) partage par toutes les cibles, pour
	// que le rapport dise la couverture du rang carte par carte.
	Rangs *Rangs

	// Compteurs de la passe, journalises et repris au rapport.
	KillsLus         int
	KillsRangConnu   int // parmi les kills lus, ceux dont le tueur a un rang en base
	KillsExclus      int // kills ecartes parce que leur variante est exclue (--exclure-variantes)
	ArtefactsLus     int
	ArtefactsEnEchec int
	PistesSansIssue  int
	PointsLus        int
}

// reglageDe rend le reglage nomme par --reglage. Le nom est valide a la lecture des
// options ; un nom inconnu ici serait une faute de programmation, pas d'usage.
func reglageDe(nom string) powerpos.Reglage {
	if nom == "v2" {
		return powerpos.ReglageV2()
	}
	return powerpos.ReglageV1()
}

// LanceMesure execute la passe complete.
func LanceMesure(res *title.PathResolver, opts options, corpus *Corpus) error {
	cibles := choisitCibles(corpus, opts)
	if len(cibles) == 0 {
		return fmt.Errorf("aucune carte retenue (--cartes / --top ne designent rien du corpus)")
	}
	index := map[string]*Cible{}
	for _, c := range cibles {
		for id := range c.Matchs {
			index[id] = c
		}
	}
	slog.Info("mappower: mesure lancee", "cartes", len(cibles), "matchs", len(index),
		"reglage", opts.reglage)

	chemin := res.SharedDBPath(opts.titleSlug)
	db, release, err := duckdb.OpenReadForQuery(chemin)
	if err != nil {
		return fmt.Errorf("base partagee illisible (%s) : %w", chemin, err)
	}
	defer release()

	// Les rangs se lisent AVANT l'accumulation : la ponderation se calibre sur les
	// quantiles du corpus, et un accumulateur ne peut pas changer de bareme en cours de
	// route sans rendre deux cellules incomparables.
	rangs := ChargeRangs(db, index)
	for _, c := range cibles {
		c.Acc = powerpos.NouvelAccumulateurPondere(tactical.GrilleParDefaut(), rangs.Ponderation)
		c.Rangs = rangs
	}
	if err := accumuleKills(db, index, rangs, opts.exclure); err != nil {
		return err
	}
	if err := accumulePresence(db, cibles); err != nil {
		return err
	}
	if err := os.MkdirAll(opts.sortie, 0o755); err != nil {
		return fmt.Errorf("creation du dossier de sortie : %w", err)
	}
	return ecrisSorties(res, opts, cibles)
}

// choisitCibles retient les cartes demandees (--cartes) ou les N plus fournies (--top).
// Le PvE est exclu d'office : ses kills sont diriges contre des vagues d'IA.
func choisitCibles(corpus *Corpus, opts options) []*Cible {
	voulues := map[string]bool{}
	for _, nom := range opts.cartes {
		voulues[strings.ToLower(nom)] = true
	}
	var out []*Cible
	for _, e := range corpus.Cartes {
		if e.Axe == AxePvE {
			continue
		}
		selon := len(voulues) > 0 && voulues[e.Carte]
		parRang := len(voulues) == 0 && len(out) < opts.top
		if !selon && !parRang {
			continue
		}
		// L'accumulateur n'est PAS construit ici : il attend la ponderation par rang, qui
		// se mesure sur le corpus une fois la base ouverte (cf. LanceMesure).
		cible := &Cible{Carte: e.Carte, Axe: e.Axe, Matchs: map[string]MatchCorpus{}}
		for _, m := range corpus.MatchsDe(e.Carte, e.Axe) {
			cible.Matchs[m.MatchID] = m
		}
		out = append(out, cible)
	}
	for nom := range voulues {
		if !presente(out, nom) {
			slog.Warn("mappower: carte demandee absente du corpus", "carte", nom)
		}
	}
	return out
}

// presente dit si une carte figure parmi les cibles.
func presente(cibles []*Cible, carte string) bool {
	for _, c := range cibles {
		if c.Carte == carte {
			return true
		}
	}
	return false
}

// accumuleKills lit la vue `_latest` en une passe et dispatche chaque elimination vers sa
// carte. Les lignes dont une extremite manque sont ECARTEES PAR LA REQUETE : le paquet pur
// les ecarterait aussi, mais les compter ici comme « ignorees » melangerait un trou de
// donnee connu (6,5 % du corpus, cf. main.go) avec un decodage rate.
//
// Le rang du tueur est joint EN MEMOIRE (`rangs.De`) et non par la requete : la jointure
// SQL sur `match_csrs_latest` doublerait la lecture d'une vue de 138 000 lignes pour un
// index qui tient en memoire, et laisserait a la base le soin de dire « inconnu » (NULL)
// la ou le paquet pur veut un pointeur nil.
func accumuleKills(db *sql.DB, index map[string]*Cible, rangs *Rangs, exclure map[string]bool) error {
	const q = `SELECT match_id, COALESCE(killer_xuid, ''),
			killer_x, killer_y, killer_z, victim_x, victim_y, victim_z
		FROM kill_positions_latest
		WHERE killer_x IS NOT NULL AND killer_y IS NOT NULL AND killer_z IS NOT NULL
		  AND victim_x IS NOT NULL AND victim_y IS NOT NULL AND victim_z IS NOT NULL`
	rows, err := db.QueryContext(context.Background(), q)
	if err != nil {
		return fmt.Errorf("lecture des positions de kill : %w", err)
	}
	defer rows.Close()
	horsCible, exclus, rangConnu, total := 0, 0, 0, 0
	for rows.Next() {
		var k powerpos.KillSample
		var xuid string
		if err := rows.Scan(&k.MatchID, &xuid, &k.KillerX, &k.KillerY, &k.KillerZ,
			&k.VictimX, &k.VictimY, &k.VictimZ); err != nil {
			return fmt.Errorf("lecture d'une position de kill : %w", err)
		}
		cible := index[k.MatchID]
		if cible == nil {
			horsCible++
			continue
		}
		mapName := cible.Matchs[k.MatchID].MapName
		if exclure[strings.ToLower(mapName)] {
			cible.KillsExclus++
			exclus++
			continue
		}
		k.RangTueur = rangs.De(k.MatchID, xuid)
		if k.RangTueur != nil {
			cible.KillsRangConnu++
			rangConnu++
		}
		cible.Acc.AjouteKill(k)
		cible.noteVariante(mapName, k.KillerX, k.KillerY)
		cible.KillsLus++
		total++
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("parcours des positions de kill : %w", err)
	}
	slog.Info("mappower: eliminations accumulees", "kills", total, "hors_cartes_mesurees", horsCible,
		"exclus_variante", exclus, "rang_tueur_connu", rangConnu)
	return nil
}

// ecrisSorties produit un CSV par carte et, sauf --sans-png, les PNG de controle.
func ecrisSorties(res *title.PathResolver, opts options, cibles []*Cible) error {
	reglage := reglageDe(opts.reglage)
	for _, c := range cibles {
		c.EcartVariantesM = c.ControleVariantes()
		cellules := c.Acc.Cellules()
		c.Scorees = powerpos.Score(tactical.GrilleParDefaut(), cellules, reglage)
		c.Positions = powerpos.Selectionne(tactical.GrilleParDefaut(), c.Scorees, reglage)
		nom := nomDeFichier(c.Carte, c.Axe)
		cheminCSV := filepath.Join(opts.sortie, nom+".csv")
		if err := EcrisCSV(cheminCSV, cellules); err != nil {
			return fmt.Errorf("carte %s : %w", c.Carte, err)
		}
		slog.Info("mappower: CSV ecrit", "carte", c.Carte, "axe", c.Axe, "path", cheminCSV,
			"cellules", len(cellules), "matchs", c.Acc.NbMatchs(), "kills_lus", c.KillsLus,
			"kills_exclus", c.KillsExclus, "kills_rang_connu", c.KillsRangConnu,
			"kills_ignores", c.Acc.KillsIgnores(), "artefacts", c.ArtefactsLus,
			"artefacts_en_echec", c.ArtefactsEnEchec, "pistes_sans_issue", c.PistesSansIssue,
			"points_lus", c.PointsLus, "presences_ignorees", c.Acc.PresencesIgnorees(),
			"cellules_scorables", len(c.Scorees), "positions_retenues", len(c.Positions))
		if opts.sansPNG {
			continue
		}
		if err := EcrisPNGDeControle(res, opts, c, cellules, filepath.Join(opts.sortie, nom)); err != nil {
			// Un fond manquant ne doit pas priver de mesure : on le dit et on continue.
			slog.Warn("mappower: PNG de controle non produit", "err", err, "carte", c.Carte)
		}
	}
	// Les POLYGONES des positions retenues, lisibles par machine : le verdict contre
	// l'oracle (etape 2) se prononce sur des recouvrements de surface, que ni le CSV par
	// cellule ni les centres du rapport ne portent. Aucun calcul ici, une serialisation.
	nomPositions := "positions.json"
	if opts.reglage != "v1" {
		nomPositions = "positions_" + opts.reglage + ".json"
	}
	positions := filepath.Join(opts.sortie, nomPositions)
	if err := EcrisPositionsJSON(positions, res, opts.titleSlug, cibles, reglage); err != nil {
		return err
	}
	rapport := filepath.Join(opts.sortie, "_rapport.md")
	if err := EcrisRapport(rapport, cibles, reglage, opts.reglage); err != nil {
		return err
	}
	slog.Info("mappower: rapport de mesure ecrit", "path", rapport, "cartes", len(cibles))
	return nil
}

// nomDeFichier rend un nom de fichier stable pour une (carte, axe).
func nomDeFichier(carte, axe string) string {
	nom := strings.NewReplacer(" ", "_", "'", "", "/", "-", "\\", "-").Replace(carte)
	return nom + "__" + axe
}
