package main

// csv_lecture.go — LA RELECTURE du CSV par cellule ecrit par `EcrisCSV` (csv.go). Extrait du
// harnais de verdict (tag `research`) le 2026-09-20 pour que le mode `--fusion` relise la
// passe empirique sans base : un seul lecteur, utilise par le diagnostic du verdict et par
// la fusion. Il accepte les deux formes ecrites par le depot — treize colonnes (etape 1) et
// vingt-cinq (passe v2) — et verifie l'en-tete nom a nom.

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"strings"

	"levelup/go-api/internal/analysis/powerpos"
)

// nbColonnesCSVV1 : les treize colonnes du CSV de l'etape 1. La passe v2 (2bis.B) en ecrit
// vingt-cinq — les memes treize d'abord, puis les signaux angulaires et ponderes. Le
// lecteur accepte les deux formes : le verdict v1 relit ses CSV d'etape 1, le verdict v2
// ceux de la passe v2, et l'en-tete est verifie sur son prefixe commun.
const nbColonnesCSVV1 = 13

// litCSVCellules relit le CSV par cellule produit par une passe de mesure (v1 ou v2).
func litCSVCellules(chemin string) ([]powerpos.Cellule, error) {
	f, err := os.Open(chemin)
	if err != nil {
		return nil, fmt.Errorf("CSV de cellules illisible (%s) : %w", chemin, err)
	}
	defer func() { _ = f.Close() }()
	lignes, err := csv.NewReader(f).ReadAll()
	if err != nil {
		return nil, fmt.Errorf("CSV de cellules invalide (%s) : %w", chemin, err)
	}
	if len(lignes) == 0 || !enTeteCSVConnu(lignes[0]) {
		return nil, fmt.Errorf("CSV de cellules a l'en-tete inattendu (%s)", chemin)
	}
	out := make([]powerpos.Cellule, 0, len(lignes)-1)
	for _, l := range lignes[1:] {
		out = append(out, celluleDeLigne(l))
	}
	return out, nil
}

// enTeteCSVConnu dit si l'en-tete est celui d'une passe v1 (13 colonnes) ou v2 (toutes),
// nom a nom sur les colonnes presentes.
func enTeteCSVConnu(entete []string) bool {
	if len(entete) != nbColonnesCSVV1 && len(entete) != len(colonnesCSV) {
		return false
	}
	for i, nom := range entete {
		if nom != colonnesCSV[i] {
			return false
		}
	}
	return true
}

// celluleDeLigne reconstruit une cellule depuis sa ligne CSV (memes colonnes, meme ordre
// que `colonnesCSV` — l'en-tete est verifie par l'appelant). Les douze colonnes v2 ne sont
// lues que si la ligne les porte.
func celluleDeLigne(l []string) powerpos.Cellule {
	c := powerpos.Cellule{
		Col: entier(l[0]), Lig: entier(l[1]),
		CentreX: flottant(l[2]), CentreY: flottant(l[3]),
		KillsDepuis: entier(l[4]), MortsDedans: entier(l[5]),
		PorteeMedianeM: flottant(l[6]), DeniveleMedianM: flottant(l[7]),
		MatchsKills: entier(l[8]), MatchsPresence: entier(l[9]), MatchsDistincts: entier(l[10]),
		OccupationGagnantsMS: flottant(l[11]), OccupationPerdantsMS: flottant(l[12]),
	}
	if len(l) < len(colonnesCSV) {
		return c
	}
	c.DirSortante = powerpos.SommeAngulaire{Cos: flottant(l[13]), Sin: flottant(l[14]), N: entier(l[15])}
	c.DirEntrante = powerpos.SommeAngulaire{Cos: flottant(l[16]), Sin: flottant(l[17]), N: entier(l[18])}
	c.KillsPonderes, c.MortsPonderees = flottant(l[19]), flottant(l[20])
	c.KillsRangConnu, c.MortsRangConnu = entier(l[21]), entier(l[22])
	c.KillsTueurFort, c.MortsTueurFort = entier(l[23]), entier(l[24])
	return c
}

func entier(s string) int { v, _ := strconv.Atoi(strings.TrimSpace(s)); return v }
func flottant(s string) float64 {
	v, _ := strconv.ParseFloat(strings.TrimSpace(s), 64)
	return v
}
