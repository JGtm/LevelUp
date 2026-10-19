package ops

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	titlePkg "levelup/go-api/internal/domain/title"
)

// DemoReplayMatchIDs rend les match_id des rejeux figés (`replay_matches`) de TOUS les
// manifestes de démo `config/demo/<gamertag>/*.json` qui déclarent le titre titleSlug, dans
// l'ordre stable (manifestes triés par chemin, puis ordre du manifeste), sans doublon (casse
// ignorée). C'est la liste que `levelup backfill-replay --demo` recuit en local : aucun
// identifiant n'est recopié à la main.
//
// Refus (erreur) : un manifeste présent mais invalide (cf. LoadDemoManifest), aucun manifeste
// pour ce titre, ou des manifestes sans aucun rejeu figé. Une liste vide ne se rend jamais en
// succès : l'appelant s'en servirait comme d'un filtre, et un filtre vide laisserait tout passer.
func DemoReplayMatchIDs(pr *titlePkg.PathResolver, titleSlug string) ([]string, error) {
	motif := pr.DemoManifestPath("*", "*")
	chemins, err := filepath.Glob(motif)
	if err != nil {
		return nil, fmt.Errorf("manifestes démo (%s): %w", motif, err)
	}
	sort.Strings(chemins)
	var ids []string
	vus := map[string]bool{}
	manifestes := 0
	for _, chemin := range chemins {
		m, found, err := LoadDemoManifest(chemin)
		if err != nil {
			return nil, err
		}
		if !found || m.TitleSlug != titleSlug {
			continue
		}
		manifestes++
		for _, id := range replayPickMatchIDs(m.Corpus.ReplayMatches) {
			if bas := strings.ToLower(id); !vus[bas] {
				vus[bas] = true
				ids = append(ids, id)
			}
		}
	}
	if manifestes == 0 {
		return nil, fmt.Errorf("aucun manifeste de démo ne déclare le titre %q (%s)", titleSlug, motif)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("les %d manifeste(s) de démo du titre %q ne listent aucun rejeu figé (replay_matches)",
			manifestes, titleSlug)
	}
	return ids, nil
}
