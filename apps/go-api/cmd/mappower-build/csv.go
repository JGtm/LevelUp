package main

// csv.go — la SORTIE de mesure : une ligne par cellule peuplee, toutes les grandeurs
// brutes, aucune derivee. Les ratios se recalculent depuis ces colonnes ; les y figer
// interdirait d'essayer une autre formule sans re-lire la base.

import (
	"encoding/csv"
	"fmt"
	"log/slog"
	"os"
	"strconv"

	"levelup/go-api/internal/analysis/powerpos"
)

// colonnesCSV est l'en-tete, dans l'ordre.
//
// Les douze dernieres colonnes sont les signaux v2 (2026-09-20) : les SOMMES angulaires
// brutes (cos, sin, n — additives, la dispersion se recalcule depuis elles, cf.
// powerpos/angles.go) et les comptes au poids du rang du tueur.
var colonnesCSV = []string{
	"col", "lig", "centre_x", "centre_y",
	"kills_depuis", "morts_dedans", "portee_mediane_m", "denivele_median_m",
	"matchs_kills", "matchs_presence", "matchs_distincts",
	"occupation_gagnants_ms", "occupation_perdants_ms",
	"dir_sortante_cos", "dir_sortante_sin", "dir_sortante_n",
	"dir_entrante_cos", "dir_entrante_sin", "dir_entrante_n",
	"kills_ponderes", "morts_ponderees",
	"kills_rang_connu", "morts_rang_connu", "kills_tueur_fort", "morts_tueur_fort",
}

// EcrisCSV ecrit les cellules d'une carte.
func EcrisCSV(chemin string, cellules []powerpos.Cellule) error {
	f, err := os.Create(chemin)
	if err != nil {
		return fmt.Errorf("creation du CSV (%s) : %w", chemin, err)
	}
	defer func() {
		if errFerme := f.Close(); errFerme != nil {
			slog.Error("mappower: fermeture du CSV", "err", errFerme, "path", chemin)
		}
	}()

	w := csv.NewWriter(f)
	if err := w.Write(colonnesCSV); err != nil {
		return fmt.Errorf("ecriture de l'en-tete (%s) : %w", chemin, err)
	}
	for _, c := range cellules {
		if err := w.Write(ligneDe(c)); err != nil {
			return fmt.Errorf("ecriture d'une cellule (%s) : %w", chemin, err)
		}
	}
	w.Flush()
	if err := w.Error(); err != nil {
		return fmt.Errorf("vidage du CSV (%s) : %w", chemin, err)
	}
	return nil
}

// ligneDe formate une cellule. Les flottants sont ecrits a trois decimales : le pas de la
// grille est de 0,5 m, le millimetre est deja au-dela de ce que la mesure sait dire.
func ligneDe(c powerpos.Cellule) []string {
	return []string{
		strconv.Itoa(c.Col), strconv.Itoa(c.Lig),
		f3(c.CentreX), f3(c.CentreY),
		strconv.Itoa(c.KillsDepuis), strconv.Itoa(c.MortsDedans),
		f3(c.PorteeMedianeM), f3(c.DeniveleMedianM),
		strconv.Itoa(c.MatchsKills), strconv.Itoa(c.MatchsPresence), strconv.Itoa(c.MatchsDistincts),
		f3(c.OccupationGagnantsMS), f3(c.OccupationPerdantsMS),
		f3(c.DirSortante.Cos), f3(c.DirSortante.Sin), strconv.Itoa(c.DirSortante.N),
		f3(c.DirEntrante.Cos), f3(c.DirEntrante.Sin), strconv.Itoa(c.DirEntrante.N),
		f3(c.KillsPonderes), f3(c.MortsPonderees),
		strconv.Itoa(c.KillsRangConnu), strconv.Itoa(c.MortsRangConnu),
		strconv.Itoa(c.KillsTueurFort), strconv.Itoa(c.MortsTueurFort),
	}
}

func f3(v float64) string { return strconv.FormatFloat(v, 'f', 3, 64) }
