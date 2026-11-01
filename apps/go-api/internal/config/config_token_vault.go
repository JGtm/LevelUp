// config_token_vault.go — le chiffreur des fichiers de jetons, dérivé du profil résolu
// (ADR 0045 D-11). Seul appelant de tokencipher.InstallForProfile : le serveur et les outils en
// ligne de commande passent tous par le chargement de la configuration, donc par ce choix unique.
package config

import "levelup/go-api/internal/platform/auth/tokencipher"

// installTokenCipher pose le chiffreur du processus d'après la capacité TokenVault de cfg. Le
// dernier profil résolu l'emporte : un chargement hébergé ou de développement remet `plain`.
// L'indisponibilité du coffre n'est jamais une erreur de chargement (les tests du profil desktop
// tournent aussi hors Windows) : elle se lit par tokencipher.ProcessError, que cmd/server refuse.
func installTokenCipher(cfg *AppConfig) {
	tokencipher.InstallForProfile(cfg.Capabilities.TokenVault)
}
