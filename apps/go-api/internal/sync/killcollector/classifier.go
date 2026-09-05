// Package killcollector — classifier.go : le traducteur « source de degat -> cle du
// registre d'armes » d'un titre, pour les chemins de sync qui en ont besoin HORS de
// l'etape 1.57.
//
// # Pourquoi il vit ici
//
// Ce paquet est deja LE pont entre `internal/sync` et le decodeur de film Halo Infinite
// (`games/halo_infinite/film/*`) : il porte l'import title-specific et la resolution de
// capability qui l'autorise. Le moteur de citations a besoin de la meme chose — traduire
// une source de degat pour compter les frags par arme — et le faire depuis le paquet
// `sync` racine y aurait ramene un import title-specific de plus, sans la garde de
// capability qui va avec.
//
// # La garde est la capability, jamais le slug
//
// Un titre qui ne declare pas `film.kill_source` n'obtient AUCUN traducteur, et l'appelant
// retombe sur sa voie historique. C'est la meme condition que celle qui arme l'etape 1.57.
package killcollector

import (
	"log/slog"

	"levelup/go-api/internal/games"
	halo "levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/port"
)

// ClassifierPourTitre rend le traducteur de source de degat du titre, ou nil.
//
// Nil dans TROIS cas, tous nominaux : `capabilities.toml` illisible, capability
// `film.kill_source` absente, ou titre sans decodeur. L'appelant lit alors sa source
// historique — jamais de panique, jamais de silence (l'echec de lecture se journalise).
func ClassifierPourTitre(repoRoot, slug string) port.KillSourceClassifier {
	caps, err := capabilitiesDuTitre(repoRoot, slug)
	if err != nil {
		slog.Warn("kill source: capabilities illisibles, traducteur non arme",
			"title", slug, "err", err)
		return nil
	}
	if !caps.Has(games.CapFilmKillSource) {
		return nil
	}
	return halo.NewKillSourceRegistry()
}

// capabilitiesDuTitre lit `capabilities.toml` du titre. Sans memorisation : les appelants
// de cette fonction sont des chemins de BACKFILL, appeles une fois par lot, pas une fois
// par match — la memorisation de PostSyncHook sert un cycle, pas un lot.
//
// La recette elle-meme vit dans games.LoadCapabilityMap (centralisee le 2026-09-04,
// regle des <= 2 copies — ce fichier en portait une des trois copies d origine).
func capabilitiesDuTitre(repoRoot, slug string) (games.CapabilityMap, error) {
	return games.LoadCapabilityMap(repoRoot, slug)
}
