package main

// forge.go — LA PASSE FORGE du catalogue de callouts : les zones nommées d'une carte
// communautaire, lues dans son `map.mvar`.
//
// LA CHAÎNE, et ce qui la distingue de la passe native :
//
//	<cache>/<map_id>.mvar                      variante téléchargée (forge_fetch.go)
//	  -> mapvar.Parse                          arbre Bond de la variante
//	  -> himap.ZonesNommeesForge               objets TypeIDZoneNommee -> polygones monde,
//	                                           râteliers (palettes d'objets) écartés
//	  -> jointure des libellés par STRING_ID   callouts_i18n.csv (csv.go, index parSID)
//	  -> classement grandes/fines              classify.go, pas desserré (pasDeClassement)
//	  -> entrée sous MapsByID[map_id]
//
// AUCUN DÉCOUPAGE : la provenance reste `mvar`. Une carte Forge n'a pas de tag levl à
// confronter, et son fond publié ne borne pas ses zones (mesure Isolation du 2026-08-27 :
// rogner au maillage tient 25/25 ancres d'objectif, rogner aux zones en perd une — donc
// les zones ne couvrent pas tout le terrain joué).
//
// UNE CARTE N'ENTRE AU CATALOGUE QUE SI AU MOINS UNE DE SES ZONES PORTE UN LIBELLÉ.
// C'est la seule règle de seuil, et elle se justifie par ce que l'utilisateur voit : la
// bascule du rejeu s'appelle « Zones nommées ». Une carte dont AUCUNE zone n'a de texte
// afficherait un calque entièrement muet sous ce nom-là — du bruit, pas une information.
// En revanche, dès qu'une zone est nommée, TOUTES les zones de la carte sont publiées,
// muettes comprises : leur géométrie est mesurée, et le rendu saute simplement le libellé
// vide (calloutsLayer.ts, drawLabels). On n'invente jamais un nom de repli.

import (
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/analysis/replay"
	"levelup/go-api/internal/analysis/replay/mapvar"
	"levelup/go-api/internal/himap"
)

// pasClassementMax : nombre de cellules visé sur le plus grand côté d'une zone lors du
// classement grandes/fines. Le pas natif (0,25 m) ferait exploser le raster sur un canevas
// Forge de 500 m ; à 200 cellules la mesure de recouvrement garde le même sens (elle
// compare des SURFACES, pas des contours) pour un coût borné.
const pasClassementMax = 200

// forgeStats porte le décompte d'une passe, pour la mesure de couverture.
type forgeStats struct {
	Cartes        int
	Zones         int
	ZonesNommees  int
	SansZone      []string
	SansLibelle   []string
	Illisibles    []string
	SidsDistincts map[uint32]bool
	SidsResolus   map[uint32]bool
}

func nouvellesStats() *forgeStats {
	return &forgeStats{SidsDistincts: map[uint32]bool{}, SidsResolus: map[uint32]bool{}}
}

// construitPasseForge lit chaque variante du cache et rend la table `maps_by_id`.
//
// Une variante illisible ou absente est COMPTÉE et journalisée, jamais fatale : le
// catalogue partiel d'une carte vaut mieux qu'un catalogue vide pour toutes.
func construitPasseForge(cibles []carteUGC, cache string, labels libellesParStringID,
	stats *forgeStats) map[string]replay.MapCalloutsEntry {
	out := map[string]replay.MapCalloutsEntry{}
	for _, c := range cibles {
		chemin := filepath.Join(cache, c.MapID+".mvar")
		blob, err := os.ReadFile(chemin)
		if err != nil {
			stats.Illisibles = append(stats.Illisibles, c.Nom+" (variante absente)")
			continue
		}
		entry, n, err := entreeForge(blob, labels, stats)
		if err != nil {
			slog.Warn("carte Forge non traitée", "nom", c.Nom, "map_id", c.MapID, "err", err)
			stats.Illisibles = append(stats.Illisibles, c.Nom+" ("+err.Error()+")")
			continue
		}
		switch {
		case len(entry.Zones) == 0:
			stats.SansZone = append(stats.SansZone, c.Nom)
		case n == 0:
			stats.SansLibelle = append(stats.SansLibelle, c.Nom)
		default:
			out[c.MapID] = entry
			stats.Cartes++
			stats.Zones += len(entry.Zones)
			stats.ZonesNommees += n
			slog.Info("carte Forge lue", "nom", c.Nom, "map_id", c.MapID,
				"zones", len(entry.Zones), "nommees", n)
		}
	}
	return out
}

// entreeForge construit l'entrée d'une carte à partir des octets de sa variante. Le second
// retour est le nombre de zones qui portent un libellé.
func entreeForge(blob []byte, labels libellesParStringID, stats *forgeStats) (replay.MapCalloutsEntry, int, error) {
	v, err := mapvar.Parse(blob)
	if err != nil {
		return replay.MapCalloutsEntry{}, 0, fmt.Errorf("variante illisible : %w", err)
	}
	entry, n := entreeDepuisZones(himap.ZonesNommeesForge(v.Objects), labels, stats)
	return entry, n, nil
}

// entreeDepuisZones assemble l'entrée du catalogue à partir des zones déjà extraites : c'est
// ici que se jouent la jointure des libellés, le classement grandes/fines et l'ordre stable.
func entreeDepuisZones(zs []himap.ZoneNommee, labels libellesParStringID,
	stats *forgeStats) (replay.MapCalloutsEntry, int) {
	entry := replay.MapCalloutsEntry{
		Provenance: replay.CalloutsProvenanceMvar,
		Zones:      make([]replay.CalloutZone, 0, len(zs)),
	}
	var formes []shapedPoly
	for _, z := range zs {
		formes = append(formes, shapedPoly{vi: z.Index, poly: z.Contour})
	}
	grandes := classifyBigAvecPas(formes, pasDeClassement(formes))
	nommees := 0
	for _, z := range zs {
		stats.SidsDistincts[z.StringID] = true
		lbl, connu := labels[z.StringID]
		if connu {
			nommees++
			stats.SidsResolus[z.StringID] = true
		}
		entry.Zones = append(entry.Zones, replay.CalloutZone{
			VolumeIndex: z.Index,
			EN:          lbl.en,
			FR:          lbl.fr,
			X:           z.Pos[0],
			Y:           z.Pos[1],
			Z:           z.Pos[2],
			ZBottom:     z.ZBas,
			ZTop:        z.ZHaut,
			Big:         grandes[z.Index],
			Polygon:     z.Contour,
		})
	}
	sort.Slice(entry.Zones, func(i, j int) bool {
		return entry.Zones[i].VolumeIndex < entry.Zones[j].VolumeIndex
	})
	return entry, nommees
}

// pasDeClassement choisit le pas du raster : celui de la chaîne native tant qu'il tient
// dans `pasClassementMax` cellules sur le plus grand côté, desserré au-delà.
func pasDeClassement(zones []shapedPoly) float64 {
	cote := 0.0
	for _, z := range zones {
		b := bbox(z.poly)
		cote = math.Max(cote, math.Max(b[2]-b[0], b[3]-b[1]))
	}
	if pas := cote / pasClassementMax; pas > classifyCell {
		return pas
	}
	return classifyCell
}
