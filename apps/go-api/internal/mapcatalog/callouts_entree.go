package mapcatalog

// callouts_entree.go — D'UN `.mvar` A L'ENTREE DE CATALOGUE DES ZONES NOMMEES d'une carte
// Forge.
//
// LA CHAINE, la meme pour la CLI de fabrication (`cmd/mapcallouts-build`, passe Forge) et pour
// le rattrapage au fetch de film (`sync/replayartifacts`) :
//
//	octets de la variante  -> mapvar.Parse
//	objets                 -> mapvar.ZonesNommeesForge (polygones monde, rateliers ecartes)
//	string_id du lieu      -> Lexique (libelle joueur EN/FR)
//	contours               -> classement grandes/fines (callouts_classement.go, pas desserre)
//
// AUCUN DECOUPAGE : la provenance reste `mvar`. Une carte Forge n'a pas de tag levl a
// confronter, et son fond publie ne borne pas ses zones.
//
// LA REGLE DE PUBLICATION est ici, en un exemplaire (`CouvertureLibelles.Verdict`) : une
// carte n'entre au catalogue que si AU MOINS UNE de ses zones porte un libelle — la bascule du
// rejeu s'appelle « Zones nommees », un calque entierement muet serait du bruit. Des qu'une
// zone est nommee, TOUTES les zones sont publiees, muettes comprises : leur geometrie est
// mesuree, et le rendu saute le libelle vide. Aucun nom de repli n'est jamais invente.

import (
	"fmt"
	"sort"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/games/halo_infinite/film/replay/mapvar"
)

// VerdictZones dit ce que vaut la lecture d'une variante pour le catalogue.
type VerdictZones string

const (
	// VerdictPubliable : au moins une zone porte un libelle — la carte entre au catalogue.
	VerdictPubliable VerdictZones = "publiable"
	// VerdictSansZone : la variante ne pose aucune zone nommee (carte native republiee en
	// asset, canevas, ratelier ecarte).
	VerdictSansZone VerdictZones = "sans_zone"
	// VerdictSansLibelle : des zones, mais aucun string_id connu du lexique.
	VerdictSansLibelle VerdictZones = "sans_libelle"
)

// CouvertureLibelles mesure la jointure des libelles sur une carte.
type CouvertureLibelles struct {
	// Zones et Nommees : zones publiees, et celles qui portent un libelle.
	Zones, Nommees int
	// StringIDs : chaque string_id de lieu employe par la carte, vrai s'il est resolu par le
	// lexique. C'est la mesure qui dit quoi ajouter au lexique quand un nom manque.
	StringIDs map[uint32]bool
}

// Verdict applique la regle de publication (cf. l'en-tete).
func (c CouvertureLibelles) Verdict() VerdictZones {
	switch {
	case c.Zones == 0:
		return VerdictSansZone
	case c.Nommees == 0:
		return VerdictSansLibelle
	}
	return VerdictPubliable
}

// SansLibelle compte les string_id employes que le lexique ne resout pas.
func (c CouvertureLibelles) SansLibelle() int {
	n := 0
	for _, resolu := range c.StringIDs {
		if !resolu {
			n++
		}
	}
	return n
}

// EntreeCalloutsForge decode la variante et rend l'entree de catalogue de ses zones nommees,
// avec la mesure de ses libelles. L'entree n'a de sens pour le catalogue que si le verdict de
// la couverture est `VerdictPubliable`.
func EntreeCalloutsForge(blob []byte, lex Lexique) (replay.MapCalloutsEntry, CouvertureLibelles, error) {
	v, err := mapvar.Parse(blob)
	if err != nil {
		return replay.MapCalloutsEntry{}, CouvertureLibelles{}, fmt.Errorf("variante illisible : %w", err)
	}
	entry, couv := EntreeCalloutsDepuisZones(mapvar.ZonesNommeesForge(v.Objects), lex)
	return entry, couv, nil
}

// EntreeCalloutsDepuisZones assemble l'entree a partir des zones deja extraites : jointure des
// libelles par string_id, classement grandes/fines, ordre stable par indice d'objet.
func EntreeCalloutsDepuisZones(zs []mapvar.ZoneNommee, lex Lexique) (replay.MapCalloutsEntry, CouvertureLibelles) {
	entry := replay.MapCalloutsEntry{
		Provenance: replay.CalloutsProvenanceMvar,
		Zones:      make([]replay.CalloutZone, 0, len(zs)),
	}
	couv := CouvertureLibelles{Zones: len(zs), StringIDs: make(map[uint32]bool, len(zs))}
	formes := make([]FormeDeZone, 0, len(zs))
	for _, z := range zs {
		formes = append(formes, FormeDeZone{Index: z.Index, Contour: z.Contour})
	}
	grandes := classerGrandesAuPas(formes, pasDeClassementForge(formes))
	for _, z := range zs {
		lbl, connu := lex[z.StringID]
		couv.StringIDs[z.StringID] = connu
		if connu {
			couv.Nommees++
		}
		entry.Zones = append(entry.Zones, replay.CalloutZone{
			VolumeIndex: z.Index,
			EN:          lbl.EN,
			FR:          lbl.FR,
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
	return entry, couv
}
