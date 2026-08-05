// cmd/mapobj-build — catalog.go : structure et écriture du catalogue figé.
package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

// catalogSchemaVersion — incrémenter à tout changement incompatible de forme.
//
//	v2 (2026-08-02) : chaque objectif SURFACIQUE porte un champ `shape`
//	(famille, demi-extents en mètres, orientation, brut). Les objectifs
//	PONCTUELS n'en portent pas — un consommateur qui ne trouve pas `shape`
//	affiche un point, jamais un disque par défaut.
const catalogSchemaVersion = 2

// catalog est le document figé sous data/titles/{slug}/reference/map_objectives.json.
type catalog struct {
	SchemaVersion int                   `json:"schema_version"`
	TitleSlug     string                `json:"title_slug"`
	GeneratedAt   time.Time             `json:"generated_at"`
	Maps          map[string]*mapEntry  `json:"maps"` // clé = map_id (asset id UGC)
	Notes         map[string]string     `json:"notes,omitempty"`
	Coverage      map[string]coverStats `json:"coverage,omitempty"`
}

type mapEntry struct {
	MapID      string               `json:"map_id"`
	VersionID  string               `json:"version_id"`
	PublicName string               `json:"public_name"`
	MvarFile   string               `json:"mvar_file"`  // ex. ctf_breaker.mvar
	Module     string               `json:"module"`     // ex. ctf_breaker (dossier de niveau)
	LevelID    int32                `json:"level_id"`   // root[1][0][0] du .mvar
	ObjectsN   int                  `json:"objects_n"`  // objets placés dans la variante
	FetchedAt  time.Time            `json:"fetched_at"` // date de l'appel réseau figé
	Objectives []mapvar.Objective   `json:"objectives"`
	Unresolved map[string]int       `json:"unresolved_labels,omitempty"`
	Names      []string             `json:"names,omitempty"`
	Bounds     map[string][]float64 `json:"bounds,omitempty"` // min/max XYZ des objets
}

type coverStats struct {
	Objectives int `json:"objectives"`
	Unresolved int `json:"unresolved_label_hashes"`
}

func newCatalog(titleSlug string) *catalog {
	return &catalog{
		SchemaVersion: catalogSchemaVersion,
		TitleSlug:     titleSlug,
		GeneratedAt:   time.Now().UTC(),
		Maps:          map[string]*mapEntry{},
		Coverage:      map[string]coverStats{},
		Notes: map[string]string{
			"source":     "variantes de carte UGC (.mvar), Bond CompactBinary v2",
			"repere":     "positions en repère monde du .mvar, mètres, non transformées",
			"labels":     "murmur3_x86_32(seed=0) du nom snake_case du label de mode",
			"team_index": "index d'équipe brut du fichier ; -1 = absent",
			"shape": "demi-extents en mètres ; ABSENT sur un objectif ponctuel — " +
				"afficher un point, jamais un disque par défaut. Orientation " +
				"obligatoire : ignorer forward se trompe de 31 % sur une zone tournée. " +
				"raw = les entiers 16.16 du fichier, conservés pour recalculer sans ré-extraire",
			"regeneration": "go run ./cmd/mapobj-build --player <GT> --map-id <uuid> ; " +
				"hors ligne : --refresh-from <dossier de .mvar>",
		},
	}
}

// addVariant enregistre une variante parsée dans le catalogue.
func (c *catalog) addVariant(e *mapEntry, v *mapvar.Variant) {
	e.LevelID = v.LevelID
	e.ObjectsN = len(v.Objects)
	e.Objectives = v.Objectives()
	e.Names = v.Names
	if u := v.UnresolvedLabels(); len(u) > 0 {
		e.Unresolved = map[string]int{}
		for h, n := range u {
			e.Unresolved[fmt.Sprintf("%d", h)] = n
		}
	}
	e.Bounds = boundsOf(v)
	c.Maps[e.MapID] = e
	c.Coverage[e.MapID] = coverStats{
		Objectives: len(e.Objectives),
		Unresolved: len(e.Unresolved),
	}
}

func boundsOf(v *mapvar.Variant) map[string][]float64 {
	if len(v.Objects) == 0 {
		return nil
	}
	mn := []float64{v.Objects[0].Pos.X, v.Objects[0].Pos.Y, v.Objects[0].Pos.Z}
	mx := append([]float64(nil), mn...)
	for _, o := range v.Objects {
		p := []float64{o.Pos.X, o.Pos.Y, o.Pos.Z}
		for i := 0; i < 3; i++ {
			if p[i] < mn[i] {
				mn[i] = p[i]
			}
			if p[i] > mx[i] {
				mx[i] = p[i]
			}
		}
	}
	return map[string][]float64{"min": mn, "max": mx}
}

// mergeInto fusionne le catalogue existant (s'il y en a un) pour ne pas perdre
// les cartes déjà figées lors d'un run partiel.
func (c *catalog) mergeExisting(path string) error {
	raw, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	var prev catalog
	if err := json.Unmarshal(raw, &prev); err != nil {
		return fmt.Errorf("catalogue existant illisible: %w", err)
	}
	if prev.SchemaVersion != catalogSchemaVersion {
		return fmt.Errorf("catalogue existant en schema_version=%d, attendu %d — régénérer entièrement",
			prev.SchemaVersion, catalogSchemaVersion)
	}
	for id, e := range prev.Maps {
		if _, exists := c.Maps[id]; !exists {
			c.Maps[id] = e
			c.Coverage[id] = prev.Coverage[id]
		}
	}
	return nil
}

// write sérialise le catalogue de façon atomique (temporaire + rename).
func (c *catalog) write(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	// Tri déterministe des objectifs pour un diff git lisible.
	for _, e := range c.Maps {
		sort.SliceStable(e.Objectives, func(i, j int) bool {
			if e.Objectives[i].Role != e.Objectives[j].Role {
				return e.Objectives[i].Role < e.Objectives[j].Role
			}
			return e.Objectives[i].ObjectIdx < e.Objectives[j].ObjectIdx
		})
	}
	buf, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}
