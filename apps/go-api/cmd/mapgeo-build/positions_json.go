package main

// positions_json.go — LA SORTIE LISIBLE PAR MACHINE, de la MEME FORME que le
// `positions.json` de la voie empirique (cmd/mappower-build) : une carte, ses cles
// d'identite, ses positions avec polygone monde, score, aire. La fusion de 2bis.D et le
// verdict lisent les deux fichiers avec le meme oeil.
//
// Ce fichier NE CALCULE RIEN : il serialise `Cuite.Positions`.

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"time"

	"levelup/go-api/internal/analysis/powerpos/geo"
)

// PositionsGeoSchemaVersion versionne la forme du fichier.
const PositionsGeoSchemaVersion = 1

// SortiePositionsGeo est le document ecrit dans `positions_geo.json`.
type SortiePositionsGeo struct {
	SchemaVersion int              `json:"schema_version"`
	TitleSlug     string           `json:"title_slug"`
	GenereLe      string           `json:"genere_le"`
	Parametres    geo.Parametres   `json:"parametres"`
	Reglage       geo.Reglage      `json:"reglage"`
	Cartes        []SortieCarteGeo `json:"cartes"`
}

// AxeGeo : l'axe de mesure publie. La geometrie ne distingue pas arene et BTB (elle n'a pas
// de corpus) ; les six cartes cuites sont des cartes d'arene, et le harnais de verdict lit
// cet axe pour nommer ses fichiers.
const AxeGeo = "arene"

// SortieCarteGeo porte une carte et ses positions. Les cles `carte`, `axe`, `module`,
// `map_id_dominant`, `map_ids` et celles des positions sont CELLES du `positions.json`
// empirique : le harnais de verdict (`cmd/mappower-build`, tag `research`) lit les deux
// fichiers avec le meme lecteur.
type SortieCarteGeo struct {
	Carte  string `json:"carte"`
	Axe    string `json:"axe"`
	Module string `json:"module"`
	// MapIDDominant : le premier map_id du module (la geometrie n'a pas de corpus pour en
	// designer un plus frequent).
	MapIDDominant string   `json:"map_id_dominant"`
	MapIDs        []string `json:"map_ids"`

	Noeuds    int     `json:"noeuds"`
	Cibles    int     `json:"cibles"`
	Rayons    int     `json:"rayons"`
	DureeS    float64 `json:"duree_s"`
	Frontiere bool    `json:"frontiere_appliquee"`

	Positions []SortiePositionGeo `json:"positions"`
}

// SortiePositionGeo est une position retenue.
type SortiePositionGeo struct {
	ID       string       `json:"id"`
	Rang     int          `json:"rang"`
	Polygone [][2]float64 `json:"polygone"`

	ScoreMoyen float64 `json:"score_moyen"`
	ScoreMax   float64 `json:"score_max"`

	NbCellules int     `json:"nb_cellules"`
	NbNoeuds   int     `json:"nb_noeuds"`
	AireM2     float64 `json:"aire_m2"`
	CentreX    float64 `json:"centre_x"`
	CentreY    float64 `json:"centre_y"`
	ZMin       float64 `json:"z_min"`
	ZMax       float64 `json:"z_max"`

	// Variables : moyennes normalisees, pour lire POURQUOI la position est retenue.
	Variables geo.Variables `json:"variables"`
}

// EcrisPositionsJSON serialise les positions de toutes les cartes cuites.
func EcrisPositionsJSON(chemin, titleSlug string, p geo.Parametres, r geo.Reglage, cuites []*Cuite) error {
	doc := SortiePositionsGeo{
		SchemaVersion: PositionsGeoSchemaVersion, TitleSlug: titleSlug,
		GenereLe: time.Now().UTC().Format(time.RFC3339), Parametres: p, Reglage: r,
	}
	for _, c := range cuites {
		doc.Cartes = append(doc.Cartes, sortieDe(c))
	}
	blob, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("serialisation : %w", err)
	}
	if err := os.WriteFile(chemin, append(blob, '\n'), 0o644); err != nil {
		return fmt.Errorf("ecriture (%s) : %w", chemin, err)
	}
	slog.Info("mapgeo: positions ecrites", "path", chemin, "cartes", len(doc.Cartes))
	return nil
}

// sortieDe projette une carte cuite.
func sortieDe(c *Cuite) SortieCarteGeo {
	out := SortieCarteGeo{
		Carte: c.Cible.Carte, Axe: AxeGeo, Module: c.Cible.Module, MapIDs: c.Cible.MapIDs,
		Noeuds: len(c.Resultat.Noeuds), Cibles: c.Bilan.Cibles, Rayons: c.Bilan.Rayons,
		DureeS: c.DureeTotale.Seconds(), Frontiere: c.FrontiereAppliquee,
	}
	if len(c.Cible.MapIDs) > 0 {
		out.MapIDDominant = c.Cible.MapIDs[0]
	}
	for i, p := range c.Positions {
		out.Positions = append(out.Positions, SortiePositionGeo{
			ID: fmt.Sprintf("%s__%s__geo__%d", nomDeFichier(c.Cible.Carte), AxeGeo, i+1), Rang: i + 1,
			Polygone: p.Polygone, ScoreMoyen: p.ScoreMoyen, ScoreMax: p.ScoreMax,
			NbCellules: len(p.Cellules), NbNoeuds: len(p.Noeuds), AireM2: p.AireM2,
			CentreX: p.CentreX, CentreY: p.CentreY, ZMin: p.ZMin, ZMax: p.ZMax, Variables: p.Norm,
		})
	}
	return out
}
