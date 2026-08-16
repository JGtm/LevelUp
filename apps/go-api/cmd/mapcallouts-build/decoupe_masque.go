package main

// decoupe_masque.go — LE DÉCOUPAGE UNIVERSEL : rogner les pavés du designer sur le décor
// réellement publié, carte par carte.
//
// LA RÈGLE, et elle n'a pas d'exception : une carte n'est découpée que si son FOND est
// publié (`map_backgrounds/{module}.png` + sidecar de calage). Pas de fond = provenance
// `brut`, polygone du tag servi tel quel. On ne devine jamais un décor.
//
// RIDGELINE RESTE À PART, et c'est mesuré : son découpage vient du dump du POC
// (decoupe.go), qui travaillait par ÉTAGE sur les emprises de `map_structure` — une donnée
// que seules deux cartes possèdent. La chaîne universelle, elle, lit un masque SANS
// altitude (cf. l'en-tête de internal/mapdecoupe) et rogne donc moins. Garder le meilleur
// découpage là où il existe, et le comparer à la chaîne universelle, c'est ce qui fait de
// Ridgeline un ORACLE et pas un miroir.

import (
	"fmt"
	"log/slog"
	"os"
	"sort"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/mapdecoupe"
)

// rapportDecoupe chiffre le découpage d'une carte — de quoi CONSTATER ce que la chaîne a
// fait sans rouvrir le catalogue.
type rapportDecoupe struct {
	Zones     int
	Decoupees int
	// Degenerees nomme les zones qui gardent leur brut faute d'assez de décor sous elles.
	// Chacune est un cas à REGARDER : une zone nommée qui n'a pas de sol dit que le fond a
	// un trou là, pas que la zone n'existe pas.
	Degenerees        []string
	PartGardeeMediane float64
	Sommets           int
}

// decoupeCarte applique le découpage universel à une carte, quand son fond est publié.
//
// Les polygones bruts remplacés sont rangés dans `brut` : le découpage ne perd rien.
func decoupeCarte(e *replay.MapCalloutsEntry, res *title.PathResolver, slug string,
	brut map[string][]replay.CalloutBrutZone) error {
	m, err := chargeMasqueCarte(res.MapBackgroundPath(slug, e.Module), res.MapBackgroundMetaPath(slug, e.Module))
	if err != nil {
		return err
	}
	if m == nil {
		slog.Info("carte sans fond publié — pavés du designer servis tels quels", "module", e.Module)
		return nil
	}
	bruts, rap := decoupeSurMasque(e, m)
	if len(bruts) > 0 {
		brut[e.Module] = bruts
	}
	slog.Info("callouts découpés sur le décor", "module", e.Module,
		"zones_a_forme", rap.Zones, "decoupees", rap.Decoupees,
		"degenerees", len(rap.Degenerees), "part_gardee_mediane", rap.PartGardeeMediane,
		"sommets", rap.Sommets)
	for _, d := range rap.Degenerees {
		// Chaque zone dégénérée est un cas à regarder — elle ne se perd pas dans un compteur.
		slog.Warn("zone gardée BRUTE : pas assez de décor sous elle", "module", e.Module, "zone", d)
	}
	return nil
}

// chargeMasqueCarte rend le masque praticable d'une carte, ou nil si son fond n'est pas
// publié. L'absence de fond est un cas NOMINAL ; une paire présente mais illisible, non.
func chargeMasqueCarte(pngPath, metaPath string) (*mapdecoupe.Masque, error) {
	if _, err := os.Stat(pngPath); err != nil {
		return nil, nil
	}
	if _, err := os.Stat(metaPath); err != nil {
		return nil, fmt.Errorf("fond %s sans sidecar de calage : %w", pngPath, err)
	}
	m, err := mapdecoupe.ChargeMasque(pngPath, metaPath)
	if err != nil {
		return nil, err
	}
	return m.Comble(mapdecoupe.ToleranceParDefaut), nil
}

// decoupeSurMasque rogne les zones d'une carte et rend les polygones bruts à conserver.
//
// Une zone sans forme propre (volume secondaire) n'est pas touchée : elle n'a rien à
// découper et reste interrogeable par `zoneAt`.
func decoupeSurMasque(e *replay.MapCalloutsEntry, m *mapdecoupe.Masque) ([]replay.CalloutBrutZone, rapportDecoupe) {
	rap := rapportDecoupe{}
	opts := mapdecoupe.OptionsParDefaut()
	var bruts []replay.CalloutBrutZone
	var parts []float64
	for i := range e.Zones {
		z := &e.Zones[i]
		if len(z.Polygon) < 3 {
			continue
		}
		rap.Zones++
		r := mapdecoupe.Decoupe(z.Polygon, m, opts)
		if r.Degenere {
			rap.Degenerees = append(rap.Degenerees,
				fmt.Sprintf("%s [vol %d] %.1f m² -> %.1f m²", z.EN, z.VolumeIndex, r.AireBrutM2, r.AireM2))
			continue
		}
		bruts = append(bruts, replay.CalloutBrutZone{VolumeIndex: z.VolumeIndex, Polygon: z.Polygon})
		z.Polygon, z.Parts, z.Holes = r.Contour, r.Parties, r.Trous
		rap.Decoupees++
		parts = append(parts, r.PartGardee())
		rap.Sommets += compteSommets(r)
	}
	rap.PartGardeeMediane = mediane(parts)
	if rap.Decoupees > 0 {
		e.Provenance = replay.CalloutsProvenanceDecoupe
	}
	return bruts, rap
}

// compteSommets totalise les sommets publiés pour une zone (contour, parties, trous).
func compteSommets(r mapdecoupe.Resultat) int {
	n := len(r.Contour)
	for _, p := range r.Parties {
		n += len(p)
	}
	for _, h := range r.Trous {
		n += len(h)
	}
	return n
}

// mediane rend la médiane d'un échantillon (0 si vide). Le tri se fait sur une COPIE :
// l'appelant garde son ordre.
func mediane(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	c := append([]float64(nil), v...)
	sort.Float64s(c)
	return c[len(c)/2]
}
