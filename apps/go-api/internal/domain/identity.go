// Package domain — identity.go : types de l'annuaire des joueurs (ADR 0035).
//
// L'identité d'un joueur est éparpillée sur quatre registres de cycles de vie
// différents — le compte (`data/auth/users.json`), les credentials
// (`data/auth/watcher_tokens/{xuid}.json`), le profil de suivi
// (`db_profiles.json`) et le suivi live (le daemon watcher) — et la SEULE clé
// qui les relie est le xuid (ADR 0035 D1). Le gamertag et le slug sont des
// valeurs d'affichage et une composante de chemin, jamais une clé de jointure.
package domain

import "context"

// ProfileGate répond « ce couple (titre, xuid) est-il un profil SUIVI ? ».
//
// Suivi = déclaré dans `db_profiles.json` pour ce titre, non `auth_only`, et
// `sync_enabled != false` — exactement le filtre de SyncablePlayers. C'est la
// porte que franchissent le coordinateur de sync (`sync.Coordinator.Submit`),
// le daemon watcher (`watcher.Daemon.AddPlayer`) et le SSO Xbox avant de
// notifier le watcher (ADR 0035 D3).
//
// POURQUOI (incident du 2026-07-23) : un compte Xbox inconnu s'est connecté par
// SSO sur une instance non verrouillée. Le compte et ses credentials ont été
// créés, le watcher l'a pris en charge, et deux heures plus tard un sync a créé
// une player DB et écrit 25 matchs dans l'entrepôt partagé — alors qu'aucun
// profil n'existait. Tout ce qui vient ensuite (notifications post-sync,
// Prestige, ownership, scheduler, pages admin) résout un joueur par
// `db_profiles.json` : ce joueur n'existait pour personne.
//
// Un compte sans profil est un état VALIDE et INERTE. Sa seule sortie est le
// wizard de mise en place (ou une invitation, cf. le plan frère des invitations).
//
// La recherche se fait par XUID, jamais par gamertag : un gamertag se renomme,
// un xuid non.
type ProfileGate func(ctx context.Context, titleSlug, xuid string) bool
