//go:build research

package replay

// instruments_partages_pads_cluster_research_test.go — synthese des grappes de ground_weapon_pads_cluster_test.go, utilisee par des instruments research (J12.7 lint : inutilises dans le build par defaut). Deplacement pur.

import "fmt"

// gwPadsSpread resume la taille des grappes — le temoin de Notion 11 se lit la : un socle
// recurre, une grappe d'une seule apparition est un lacher isole.
func gwPadsSpread(in []gwPadCluster) string {
	if len(in) == 0 {
		return "aucune grappe"
	}
	sizes := make([]float64, 0, len(in))
	singles, maxN := 0, 0
	for _, c := range in {
		sizes = append(sizes, float64(len(c.TS)))
		if len(c.TS) == 1 {
			singles++
		}
		if len(c.TS) > maxN {
			maxN = len(c.TS)
		}
	}
	return fmt.Sprintf("grappes %d · a une seule apparition %d · mediane %.0f · max %d",
		len(in), singles, gwPadsQuantile(sizes, 0.5), maxN)
}

// gwPadsCycleSummary compte les socles dont le cycle est ETABLI, et donne la mediane des
// medianes — la seule facon honnete de resumer des cycles qui n'ont pas tous la meme stabilite.
func gwPadsCycleSummary(socles []gwPadCluster) string {
	if len(socles) == 0 {
		return "aucun socle : aucun cycle"
	}
	var med []float64
	etablis := 0
	for _, p := range socles {
		c := gwPadsCycle(p.TS)
		if !c.Established {
			continue
		}
		etablis++
		med = append(med, c.MedianS)
	}
	if etablis == 0 {
		return fmt.Sprintf("socles %d · cycle ETABLI 0 · aucun cycle publiable", len(socles))
	}
	return fmt.Sprintf("socles %d · cycle ETABLI %d (%s) · mediane des medianes %.1f s",
		len(socles), etablis, gwPadsPart(etablis, len(socles)), gwPadsQuantile(med, 0.5))
}
