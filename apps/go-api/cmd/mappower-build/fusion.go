package main

// fusion.go — LE MODE `--fusion` : combiner, cellule par cellule, la passe empirique (CSV
// de `--mesure --reglage v2`) et la cuisson geometrique (CSV de `cmd/mapgeo-build`), puis
// selectionner et publier `positions_fusion.json` (meme forme que les autres fichiers de
// positions : le harnais de verdict le juge avec le meme oeil), une planche par carte et un
// rapport. Item 2bis.D du plan, seconde moitie (2026-09-20).
//
// SANS BASE. Les identites des cartes (module, map_id) sont relues dans les deux fichiers
// de positions deja produits ; les scores empiriques sont REJOUES sur le CSV avec le
// reglage fige `powerpos.ReglageV2()` (celui que la passe a serialise — le rejeu est prouve
// fidele au verdict v2, section 7) ; les scores geometriques sont lus tels quels (colonne
// `score`, `geo.ReglageGeoV1`). La fusion elle-meme est `fusion.ReglageFusionV1`.
//
// Les cartes fusionnees sont celles du fichier geometrique : une carte sans geometrie
// cuite n'a rien a fusionner et n'est pas publiee ici (elle reste jugee « empirique seul »).

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"levelup/go-api/internal/analysis/powerpos"
	"levelup/go-api/internal/analysis/powerpos/fusion"
	"levelup/go-api/internal/analysis/tactical"
	"levelup/go-api/internal/domain/title"
)

// Noms des fichiers d'entree et de sortie du mode fusion.
const (
	positionsGeoNom    = "positions_geo.json"
	positionsV2Nom     = "positions_v2.json"
	positionsFusionNom = "positions_fusion.json"
	rapportFusionNom   = "_rapport.md"
	// axeFusion : la geometrie n'a pas d'axe ; les cartes cuites sont des cartes d'arene et
	// la passe empirique les a mesurees sur cet axe.
	axeFusion = "arene"
)

// SortiePositionsFusion est le document ecrit dans `positions_fusion.json`.
type SortiePositionsFusion struct {
	SchemaVersion int    `json:"schema_version"`
	TitleSlug     string `json:"title_slug"`
	GenereLe      string `json:"genere_le"`
	// Reglage : la fusion. Ses cles ne sont PAS celles du reglage empirique — le harnais
	// de verdict, qui lit tout fichier de positions comme un reglage empirique, n'y trouve
	// pas d'hysteresis et ne tente pas de rejouer un diagnostic v2 dessus.
	Reglage fusion.Reglage `json:"reglage"`
	// ReglageEmpirique / ReglageGeometrique : les noms des reglages FIGES en entree.
	ReglageEmpirique   string `json:"reglage_empirique"`
	ReglageGeometrique string `json:"reglage_geometrique"`
	// SourcesEmpirique / SourcesGeometrique : les dossiers lus.
	SourceEmpirique   string `json:"source_empirique"`
	SourceGeometrique string `json:"source_geometrique"`

	Cartes []SortieCarteFusion `json:"cartes"`
}

// SortieCarteFusion porte une carte fusionnee et ses positions.
type SortieCarteFusion struct {
	SortieCartePos
	CellulesGeo    int `json:"cellules_geo"`
	CellulesEmp    int `json:"cellules_emp"`
	CellulesFusion int `json:"cellules_fusion"`
	NoeudsNonFinis int `json:"noeuds_non_finis"`
	// SeuilAmorce / SeuilCroissance : les seuils de l'hysteresis appliques a la carte.
	SeuilAmorce     float64 `json:"seuil_amorce"`
	SeuilCroissance float64 `json:"seuil_croissance"`

	PositionsFusion []SortiePositionFusion `json:"positions_fusion"`
}

// SortiePositionFusion complete une position par l'origine de son score.
type SortiePositionFusion struct {
	ID       string  `json:"id"`
	GeoMoyen float64 `json:"geo_moyen"`
	EmpMoyen float64 `json:"emp_moyen"`
	SansEmp  int     `json:"sans_emp"`
	SansGeo  int     `json:"sans_geo"`
	ZMin     float64 `json:"z_min"`
	ZMax     float64 `json:"z_max"`
}

// carteFusionnee est l'etat d'une carte pendant la passe.
type carteFusionnee struct {
	Identite  SortieCartePos
	Cellules  []fusion.Cellule
	Positions []powerpos.Position
	Bilans    []fusion.Bilan
	Diag      powerpos.DiagnosticSelection
	Sortie    SortieCarteFusion
}

// LanceFusion execute le mode fusion.
func LanceFusion(res *title.PathResolver, opts options) error {
	geoDoc, err := LitPositionsJSON(filepath.Join(opts.geometrie, positionsGeoNom))
	if err != nil {
		return err
	}
	empDoc, err := LitPositionsJSON(filepath.Join(opts.mesures, positionsV2Nom))
	if err != nil {
		return err
	}
	if err := os.MkdirAll(opts.sortie, 0o755); err != nil {
		return fmt.Errorf("dossier de sortie (%s) : %w", opts.sortie, err)
	}
	reglage := fusion.ReglageFusionV1()
	var cartes []*carteFusionnee
	for _, ident := range geoDoc.Cartes {
		if len(opts.cartes) > 0 && !contient(opts.cartes, ident.Carte) {
			continue
		}
		c, err := fusionneCarte(opts, ident, identiteEmpirique(empDoc, ident.Carte), reglage)
		if err != nil {
			return fmt.Errorf("carte %s : %w", ident.Carte, err)
		}
		if !opts.sansPNG {
			base := filepath.Join(opts.sortie, nomDeFichier(c.Identite.Carte, axeFusion))
			if err := EcrisPNGFusion(res, opts.titleSlug, c, base+"_fusion_score.png"); err != nil {
				slog.Warn("mappower: planche de fusion non produite", "err", err, "carte", ident.Carte)
			}
		}
		cartes = append(cartes, c)
	}
	if len(cartes) == 0 {
		return fmt.Errorf("aucune carte fusionnee (--cartes ne designe rien du fichier geometrique)")
	}
	doc := SortiePositionsFusion{
		SchemaVersion: PositionsSchemaVersion, TitleSlug: opts.titleSlug,
		GenereLe: time.Now().UTC().Format(time.RFC3339), Reglage: reglage,
		ReglageEmpirique: "v2", ReglageGeometrique: "geo_v1",
		SourceEmpirique: opts.mesures, SourceGeometrique: opts.geometrie,
	}
	for _, c := range cartes {
		doc.Cartes = append(doc.Cartes, c.Sortie)
	}
	if err := ecrisJSON(filepath.Join(opts.sortie, positionsFusionNom), doc); err != nil {
		return err
	}
	return EcrisRapportFusion(filepath.Join(opts.sortie, rapportFusionNom), doc, cartes)
}

// fusionneCarte lit les deux CSV d'une carte, fusionne, selectionne et projette la sortie.
func fusionneCarte(opts options, ident, emp SortieCartePos, r fusion.Reglage) (*carteFusionnee, error) {
	g := tactical.GrilleParDefaut()
	noeuds, nonFinis, err := litCSVNoeudsGeo(filepath.Join(opts.geometrie, nomGeo(ident.Carte)+".csv"), g)
	if err != nil {
		return nil, err
	}
	cellules, err := litCSVCellules(filepath.Join(opts.mesures, nomDeFichier(ident.Carte, axeFusion)+".csv"))
	if err != nil {
		return nil, err
	}
	geoCellules := fusion.AgregeCellules(noeuds)
	scorees := powerpos.Score(g, cellules, powerpos.ReglageV2())
	c := &carteFusionnee{Identite: ident}
	c.Identite.Axe = axeFusion
	c.Identite.Matchs, c.Identite.CellulesScorables = emp.Matchs, emp.CellulesScorables
	c.Cellules = fusion.Fusionne(g, geoCellules, scorees, r)
	c.Positions = fusion.Selectionne(g, c.Cellules, r)
	c.Diag = powerpos.Diagnostique(fusion.CommeScorees(c.Cellules), r.Selection)
	for _, p := range c.Positions {
		c.Bilans = append(c.Bilans, fusion.BilanDe(p, c.Cellules))
	}
	c.Sortie = sortieFusionDe(c, len(geoCellules), len(scorees), nonFinis)
	slog.Info("mappower: carte fusionnee", "carte", ident.Carte, "noeuds_geo", len(noeuds),
		"noeuds_non_finis", nonFinis, "cellules_geo", len(geoCellules), "cellules_emp", len(scorees),
		"cellules_fusion", len(c.Cellules), "positions", len(c.Positions),
		"seuil_amorce", c.Diag.SeuilAmorce, "seuil_croissance", c.Diag.SeuilCroissance)
	return c, nil
}

// sortieFusionDe projette une carte fusionnee sur le document.
func sortieFusionDe(c *carteFusionnee, nbGeo, nbEmp, nonFinis int) SortieCarteFusion {
	out := SortieCarteFusion{SortieCartePos: c.Identite, CellulesGeo: nbGeo, CellulesEmp: nbEmp,
		CellulesFusion: len(c.Cellules), NoeudsNonFinis: nonFinis,
		SeuilAmorce: c.Diag.SeuilAmorce, SeuilCroissance: c.Diag.SeuilCroissance}
	out.Positions = nil
	for i, p := range c.Positions {
		id := fmt.Sprintf("%s__fusion__%d", nomDeFichier(c.Identite.Carte, axeFusion), i+1)
		out.Positions = append(out.Positions, SortiePosition{
			ID: id, Rang: i + 1, Polygone: p.Polygone,
			ScoreMoyen: p.ScoreMoyen, ScoreMax: p.ScoreMax,
			NbCellules: len(p.Cellules), NbCellulesMesurees: p.NbCellulesMesurees, AireM2: p.AireM2,
			CentreX: p.CentreX, CentreY: p.CentreY,
			Kills: p.Kills, Morts: p.Morts, MatchsPos: p.Matchs,
		})
		b := c.Bilans[i]
		out.PositionsFusion = append(out.PositionsFusion, SortiePositionFusion{ID: id,
			GeoMoyen: b.GeoMoyen, EmpMoyen: b.EmpMoyen, SansEmp: b.SansEmp, SansGeo: b.SansGeo,
			ZMin: b.ZMin, ZMax: b.ZMax})
	}
	return out
}

// identiteEmpirique rend la carte du fichier empirique qui porte ce nom (zero si absente :
// la fusion se fait alors sans les comptes de matchs, qui ne sont qu'une provenance).
func identiteEmpirique(doc SortiePositions, carte string) SortieCartePos {
	for _, c := range doc.Cartes {
		if c.Carte == carte && c.Axe == axeFusion {
			return c
		}
	}
	slog.Warn("mappower: carte absente du fichier empirique — provenance sans matchs", "carte", carte)
	return SortieCartePos{}
}

// nomGeo rend le nom de fichier de `cmd/mapgeo-build` pour une carte (minuscules, espaces
// remplaces). C'est la convention de l'autre programme, relue ici pour retrouver ses CSV.
func nomGeo(carte string) string {
	return strings.ReplaceAll(strings.ToLower(strings.TrimSpace(carte)), " ", "_")
}

// contient dit si la liste porte le nom.
func contient(noms []string, nom string) bool {
	for _, n := range noms {
		if strings.EqualFold(strings.TrimSpace(n), nom) {
			return true
		}
	}
	return false
}

// ecrisJSON serialise un document indente.
func ecrisJSON(chemin string, doc any) error {
	blob, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("serialisation (%s) : %w", chemin, err)
	}
	if err := os.WriteFile(chemin, append(blob, '\n'), 0o644); err != nil {
		return fmt.Errorf("ecriture (%s) : %w", chemin, err)
	}
	slog.Info("mappower: fichier ecrit", "path", chemin)
	return nil
}

// EcrisRapportFusion ecrit le rapport de la passe de fusion : les comptes par carte et
// chaque position avec l'origine de son score.
func EcrisRapportFusion(chemin string, doc SortiePositionsFusion, cartes []*carteFusionnee) error {
	var b strings.Builder
	fmt.Fprintf(&b, "# Fusion geometrie x empirique — %s\n\n", doc.GenereLe)
	fmt.Fprintf(&b, "Reglage de fusion : `%+v`\n\nEmpirique : `%s` (%s) ; geometrie : `%s` (%s).\n\n",
		doc.Reglage, doc.ReglageEmpirique, doc.SourceEmpirique, doc.ReglageGeometrique, doc.SourceGeometrique)
	fmt.Fprintln(&b, "| carte | cellules geo | cellules emp | union | noeuds non finis | amorce | croissance | composantes apres fermeture (tailles) | positions |")
	fmt.Fprintln(&b, "|---|---:|---:|---:|---:|---:|---:|---|---:|")
	for _, c := range cartes {
		s := c.Sortie
		fmt.Fprintf(&b, "| %s | %d | %d | %d | %d | %.3f | %.3f | %d : %s | %d |\n", s.Carte, s.CellulesGeo,
			s.CellulesEmp, s.CellulesFusion, s.NoeudsNonFinis, s.SeuilAmorce, s.SeuilCroissance,
			c.Diag.Composantes, strings.Trim(fmt.Sprint(c.Diag.TaillesComposantes), "[]"), len(s.Positions))
	}
	fmt.Fprintln(&b)
	fmt.Fprintln(&b, "| carte | position | score moyen | cellules (mesurees) | aire m2 | centre | z | geo moyen | emp moyen | sans emp | sans geo | matchs |")
	fmt.Fprintln(&b, "|---|---|---:|---|---:|---|---|---:|---:|---:|---:|---:|")
	for _, c := range cartes {
		for i, p := range c.Sortie.Positions {
			f := c.Sortie.PositionsFusion[i]
			fmt.Fprintf(&b, "| %s | `%s` | %.3f | %d (%d) | %.1f | (%.1f, %.1f) | %.1f..%.1f | %.2f | %.2f | %d | %d | %d |\n",
				c.Sortie.Carte, p.ID, p.ScoreMoyen, p.NbCellules, p.NbCellulesMesurees, p.AireM2,
				p.CentreX, p.CentreY, f.ZMin, f.ZMax, f.GeoMoyen, f.EmpMoyen, f.SansEmp, f.SansGeo, p.MatchsPos)
		}
	}
	if err := os.WriteFile(chemin, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("ecriture du rapport (%s) : %w", chemin, err)
	}
	slog.Info("mappower: rapport de fusion ecrit", "path", chemin, "cartes", len(cartes))
	return nil
}
