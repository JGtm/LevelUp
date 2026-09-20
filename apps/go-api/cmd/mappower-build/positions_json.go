package main

// positions_json.go — la SORTIE LISIBLE PAR MACHINE des positions retenues (etape 2 du
// chantier, 2026-09-20).
//
// POURQUOI CE FICHIER EXISTE. La passe de mesure ecrivait le detail par CELLULE (CSV), des
// planches PNG, et un rapport en prose ou les positions n'apparaissaient que par leur
// centre. Le verdict contre l'oracle a besoin des POLYGONES : « la position calculee
// recouvre-t-elle >= 30 % de la zone nommee attendue ? » ne se repond pas avec un centre.
//
// CE FICHIER NE CALCULE RIEN. Il SERIALISE `Cible.Positions`, deja produit par
// `powerpos.Selectionne` avec le reglage FIGE (`powerpos.ReglageV1`). Aucun seuil, aucune
// formule, aucun filtre ne vit ici — c'est la condition pour que le verdict de l'etape 2
// porte sur le reglage fige et sur rien d'autre.
//
// LES CLES D'IDENTITE SONT TOUTES PUBLIEES, ET C'EST VOLONTAIRE. L'oracle designe une carte
// par son `carte_cle` : le MODULE installe pour une carte native (`ctf_aquarius`), le
// MAP_ID de l'asset UGC pour une carte Forge (`f1cc3b4e-...`). Une carte republiee au fil
// des saisons porte PLUSIEURS map_id (Solitude et sa variante classee) et l'oracle n'en
// nomme qu'un. Publier le module ET tous les map_id du corpus laisse le lecteur rattacher
// sans deviner.

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sort"
	"time"

	"levelup/go-api/internal/analysis/powerpos"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/halo_infinite/film/decfilm"
)

// PositionsSchemaVersion est la version de forme du fichier. Elle n'a rien a voir avec le
// `SchemaVersion` du futur catalogue de reference (etape 3) : ceci est une SORTIE DE
// MESURE, versionnee pour que le test de recherche refuse un fichier d'une autre forme.
const PositionsSchemaVersion = 1

// SortiePositions est le document ecrit dans `positions.json`.
type SortiePositions struct {
	SchemaVersion int    `json:"schema_version"`
	TitleSlug     string `json:"title_slug"`
	// GenereLe date la passe : un verdict se relit en sachant sur quelle cuisson il porte.
	GenereLe string           `json:"genere_le"`
	Reglage  powerpos.Reglage `json:"reglage"`
	Cartes   []SortieCartePos `json:"cartes"`
}

// SortieCartePos porte une (carte, axe) et ses positions.
type SortieCartePos struct {
	Carte string `json:"carte"`
	Axe   string `json:"axe"`
	// Module : le module installe, vide pour une carte Forge.
	Module string `json:"module"`
	// MapIDDominant : le map_id le plus frequent du corpus de la carte.
	MapIDDominant string `json:"map_id_dominant"`
	// MapIDs : TOUS les map_id vus dans le corpus, tries. Une carte republiee en porte
	// plusieurs et l'oracle n'en nomme qu'un.
	MapIDs []string `json:"map_ids"`

	Matchs            int     `json:"matchs"`
	CellulesScorables int     `json:"cellules_scorables"`
	EcartVariantesM   float64 `json:"ecart_variantes_m"`

	Positions []SortiePosition `json:"positions"`
}

// SortiePosition est une position retenue, telle que `powerpos.Selectionne` l'a rendue.
type SortiePosition struct {
	// ID : identifiant stable dans le fichier (carte, axe, rang par score decroissant).
	ID       string       `json:"id"`
	Rang     int          `json:"rang"`
	Polygone [][2]float64 `json:"polygone"`

	ScoreMoyen float64 `json:"score_moyen"`
	ScoreMax   float64 `json:"score_max"`

	NbCellules int `json:"nb_cellules"`
	// NbCellulesMesurees : celles qui portent un score (les autres ont ete ajoutees par la
	// fermeture morphologique du reglage v2 ; en v1 les deux comptes sont egaux).
	NbCellulesMesurees int     `json:"nb_cellules_mesurees"`
	AireM2             float64 `json:"aire_m2"`
	CentreX            float64 `json:"centre_x"`
	CentreY            float64 `json:"centre_y"`

	Kills     int `json:"kills"`
	Morts     int `json:"morts"`
	MatchsPos int `json:"matchs"`
}

// EcrisPositionsJSON serialise les positions de toutes les cibles mesurees.
func EcrisPositionsJSON(chemin string, res *title.PathResolver, titleSlug string,
	cibles []*Cible, reglage powerpos.Reglage) error {
	doc := SortiePositions{
		SchemaVersion: PositionsSchemaVersion,
		TitleSlug:     titleSlug,
		GenereLe:      time.Now().UTC().Format(time.RFC3339),
		Reglage:       reglage,
	}
	for _, c := range cibles {
		doc.Cartes = append(doc.Cartes, sortieDe(res, titleSlug, c))
	}
	blob, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return fmt.Errorf("serialisation des positions : %w", err)
	}
	if err := os.WriteFile(chemin, append(blob, '\n'), 0o644); err != nil {
		return fmt.Errorf("ecriture des positions (%s) : %w", chemin, err)
	}
	slog.Info("mappower: positions ecrites", "path", chemin, "cartes", len(doc.Cartes))
	return nil
}

// sortieDe projette une cible.
func sortieDe(res *title.PathResolver, titleSlug string, c *Cible) SortieCartePos {
	out := SortieCartePos{
		Carte: c.Carte, Axe: c.Axe,
		Module:            moduleDeCarte(res, titleSlug, c.Carte),
		MapIDDominant:     mapIDDominant(c),
		MapIDs:            mapIDsDe(c),
		Matchs:            c.Acc.NbMatchs(),
		CellulesScorables: len(c.Scorees),
		EcartVariantesM:   c.EcartVariantesM,
	}
	for i, p := range c.Positions {
		out.Positions = append(out.Positions, SortiePosition{
			ID:         fmt.Sprintf("%s__%d", nomDeFichier(c.Carte, c.Axe), i+1),
			Rang:       i + 1,
			Polygone:   p.Polygone,
			ScoreMoyen: p.ScoreMoyen, ScoreMax: p.ScoreMax,
			NbCellules: len(p.Cellules), NbCellulesMesurees: p.NbCellulesMesurees, AireM2: p.AireM2,
			CentreX: p.CentreX, CentreY: p.CentreY,
			Kills: p.Kills, Morts: p.Morts, MatchsPos: p.Matchs,
		})
	}
	return out
}

// mapIDsDe rend tous les map_id du corpus de la carte, tries.
func mapIDsDe(c *Cible) []string {
	vus := map[string]bool{}
	for _, m := range c.Matchs {
		if m.MapID != "" {
			vus[m.MapID] = true
		}
	}
	out := make([]string, 0, len(vus))
	for id := range vus {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}

// moduleDeCarte resout le module installe d'une carte par le catalogue de bornes de
// quantification — la meme table que la resolution des zones nommees. Vide pour une carte
// Forge (elle n'a pas de module) comme pour une carte absente du catalogue : dans les deux
// cas le lecteur se rabat sur le map_id.
func moduleDeCarte(res *title.PathResolver, titleSlug, carte string) string {
	quant, err := decfilm.LoadMapQuantCatalog(res.MapQuantBoundsPath(titleSlug))
	if err != nil {
		slog.Warn("mappower: catalogue de bornes illisible — module non resolu", "err", err)
		return ""
	}
	entree, err := quant.Lookup(carte)
	if err != nil {
		return ""
	}
	return entree.Module
}

// LitPositionsJSON relit un fichier de positions (empirique, geometrique ou fusionne : tous
// partagent les cles d'identite et de position) et refuse une autre version de forme. Les
// champs propres a une lignee (reglage, variables, bilans) sont ignores : ce lecteur ne
// sert qu'a retrouver les cartes, leurs identites et leurs polygones.
func LitPositionsJSON(chemin string) (SortiePositions, error) {
	blob, err := os.ReadFile(chemin)
	if err != nil {
		return SortiePositions{}, fmt.Errorf("positions illisibles (%s) : %w", chemin, err)
	}
	var doc SortiePositions
	if err := json.Unmarshal(blob, &doc); err != nil {
		return SortiePositions{}, fmt.Errorf("positions invalides (%s) : %w", chemin, err)
	}
	if doc.SchemaVersion != PositionsSchemaVersion {
		return SortiePositions{}, fmt.Errorf("positions (%s) en version %d, attendu %d",
			chemin, doc.SchemaVersion, PositionsSchemaVersion)
	}
	return doc, nil
}
