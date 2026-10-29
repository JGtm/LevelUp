package haloadapter

// endpoints.go — les adresses de la brique construites depuis le résolveur de titre (MT-01 /
// PMT-1). La source de vérité est la section [endpoints] de `config/titles/{slug}/constants.toml`,
// câblée au boot dans `games.SetDefaultEndpointResolver` ; la brique ne la lit pas, elle reçoit
// des `infinite.Endpoints`. Sans résolveur câblé (CLI hors boot, tests), ou pour une clé que le
// titre ne déclare pas, l'adresse par défaut de la brique sert (celle de Halo Infinite).

import (
	"context"
	"log/slog"
	"strings"

	"levelup/go-api/internal/games"
	"levelup/go-api/internal/haloapi/infinite"
)

// EndpointsDepuisResolveur construit les adresses de la brique pour le titre `slug` : le segment
// de jeu des chemins (« hi », « h5 ») et l'hôte de chacune des clés `cles` lus dans `res`. Une clé
// que le titre ne déclare pas est journalisée (`endpoint_missing`, transition : la validation du
// boot l'empêche pour un titre actif) et garde l'adresse par défaut. `res` nil : adresses par
// défaut. Les clés propres à la brique (hôte des stats sans port, site public) ne sont pas lues
// dans le résolveur.
func EndpointsDepuisResolveur(ctx context.Context, res games.EndpointResolver, slug string, cles ...infinite.Cle) infinite.Endpoints {
	return adressesDuTitre(ctx, res, slug, true, cles...)
}

// adressesDuTitre : [EndpointsDepuisResolveur], `avertir` décidant du journal d'une clé absente.
func adressesDuTitre(ctx context.Context, res games.EndpointResolver, slug string, avertir bool, cles ...infinite.Cle) infinite.Endpoints {
	e := infinite.DefaultEndpoints().AvecPrefixeJeu(games.GamePrefixFromResolver(res, slug))
	if res == nil {
		return e
	}
	for _, cle := range cles {
		cleTitre, ok := cleDuResolveur(cle)
		if !ok {
			continue
		}
		if hote, ok := res.HostFor(slug, cleTitre); ok {
			e = e.Avec(cle, hote)
			continue
		}
		if avertir {
			slog.WarnContext(ctx, "endpoint_missing", "title", slug, "endpoint_key", string(cleTitre))
		}
	}
	return e
}

// cleDuResolveur rend la clé du résolveur de titre d'une clé de la brique (mêmes chaînes pour les
// huit clés de `constants.toml`) ; false pour une clé propre à la brique.
func cleDuResolveur(cle infinite.Cle) (games.EndpointKey, bool) {
	switch cle {
	case infinite.CleStats, infinite.CleGameCMS, infinite.CleEconomy, infinite.CleSkill,
		infinite.CleUGCFilm, infinite.CleDiscoveryUGC, infinite.CleChallenges, infinite.CleNameplate:
		return games.EndpointKey(cle), true
	}
	return "", false
}

// surcharges : les hôtes d'instance (tests) qui priment sur le résolveur, par clé.
type surcharges map[infinite.Cle]string

// restantes rend les clés de `cles` sans surcharge (celles à lire dans le résolveur).
func (s surcharges) restantes(cles []infinite.Cle) []infinite.Cle {
	out := make([]infinite.Cle, 0, len(cles))
	for _, c := range cles {
		if strings.TrimSpace(s[c]) == "" {
			out = append(out, c)
		}
	}
	return out
}

// appliquer pose les surcharges non vides sur `e`.
func (s surcharges) appliquer(e infinite.Endpoints) infinite.Endpoints {
	for c, h := range s {
		if strings.TrimSpace(h) != "" {
			e = e.Avec(c, h)
		}
	}
	return e
}
