package main

// forge.go — LA PASSE FORGE du catalogue de callouts : les zones nommées d'une carte
// communautaire, lues dans son `map.mvar`.
//
// LA CHAÎNE est celle de `mapcatalog.EntreeCalloutsForge`, partagée avec le rattrapage au
// fetch de film (sync/replayartifacts) — variante décodée, zones extraites, libellés joints par
// string_id, classement grandes/fines — ; ce fichier n'y ajoute que la source (le cache des
// variantes téléchargées depuis l'inventaire, forge_fetch.go) et la MESURE de la passe.
//
//	<cache>/<map_id>.mvar   variante téléchargée (forge_fetch.go)
//	  -> mapcatalog.EntreeCalloutsForge
//	  -> verdict de publication (mapcatalog.CouvertureLibelles.Verdict)
//	  -> entrée sous MapsByID[map_id]
//
// La règle de publication (au moins une zone nommée) vit avec la chaîne, dans `mapcatalog` :
// la CLI et le runtime ne peuvent pas la trancher différemment.

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/mapcatalog"
)

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

// compterLibelles verse la couverture d'une carte dans la mesure de la passe.
func (s *forgeStats) compterLibelles(c mapcatalog.CouvertureLibelles) {
	for sid, resolu := range c.StringIDs {
		s.SidsDistincts[sid] = true
		if resolu {
			s.SidsResolus[sid] = true
		}
	}
}

// construitPasseForge lit chaque variante du cache et rend la table `maps_by_id`.
//
// Une variante illisible ou absente est COMPTÉE et journalisée, jamais fatale : le
// catalogue partiel d'une carte vaut mieux qu'un catalogue vide pour toutes.
func construitPasseForge(cibles []carteUGC, cache string, lex mapcatalog.Lexique,
	stats *forgeStats) map[string]replay.MapCalloutsEntry {
	out := map[string]replay.MapCalloutsEntry{}
	for _, c := range cibles {
		chemin := filepath.Join(cache, c.MapID+".mvar")
		blob, err := os.ReadFile(chemin)
		if err != nil {
			stats.Illisibles = append(stats.Illisibles, c.Nom+" (variante absente)")
			continue
		}
		entry, couv, err := mapcatalog.EntreeCalloutsForge(blob, lex)
		if err != nil {
			slog.Warn("carte Forge non traitée", "nom", c.Nom, "map_id", c.MapID, "err", err)
			stats.Illisibles = append(stats.Illisibles, fmt.Sprintf("%s (%v)", c.Nom, err))
			continue
		}
		stats.compterLibelles(couv)
		switch couv.Verdict() {
		case mapcatalog.VerdictSansZone:
			stats.SansZone = append(stats.SansZone, c.Nom)
		case mapcatalog.VerdictSansLibelle:
			stats.SansLibelle = append(stats.SansLibelle, c.Nom)
		case mapcatalog.VerdictPubliable:
			out[c.MapID] = entry
			stats.Cartes++
			stats.Zones += couv.Zones
			stats.ZonesNommees += couv.Nommees
			slog.Info("carte Forge lue", "nom", c.Nom, "map_id", c.MapID,
				"zones", couv.Zones, "nommees", couv.Nommees)
		}
	}
	return out
}
