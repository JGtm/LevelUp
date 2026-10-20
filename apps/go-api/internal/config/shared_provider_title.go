// Package config — shared_provider_title.go : provider (lecture ET écriture B-swap) du
// shared d'un titre, pour les moteurs et recalculs qui écrivent dans la base partagée.
package config

import (
	"log/slog"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb/sharedprovider"
)

// SharedProviderForTitle rend le provider B-swap du shared DU TITRE, jamais celui d'un
// autre. cfg.SharedProvider est le provider du titre PAR DÉFAUT uniquement : le passer à un
// moteur d'un autre titre (WithSharedProvider) ferait écrire et lire la base partagée du
// mauvais titre, AcquireWriter ignorant le chemin que le moteur croit viser.
//
//   - titre vide ou par défaut : cfg.SharedProvider (le provider du boot, tel quel) ;
//   - autre titre : le provider du Manager pour le chemin de CE titre (config.SharedDBPath,
//     base de fixture en démo), le même objet à chaque appel ;
//   - nil quand rien ne peut servir ce titre : Manager absent (mode legacy, le moteur ouvre
//     alors sa base directement sous bail) ou ouverture en échec (journalisée). Jamais un
//     repli sur le provider d'un autre titre.
func (cfg *AppConfig) SharedProviderForTitle(titleSlug string) sharedprovider.Provider {
	if titleSlug == "" || title.IsDefaultSlug(titleSlug) {
		return cfg.SharedProvider
	}
	if cfg.SharedManager == nil {
		return nil
	}
	path := SharedDBPath(cfg, titleSlug)
	p, err := cfg.SharedManager.For(path, cfg.UserTimezone)
	if err != nil {
		slog.Warn("SharedProviderForTitle: ouverture du provider du titre échouée, moteur en mode legacy",
			"title", titleSlug, "path", path, "err", err)
		return nil
	}
	return p
}
