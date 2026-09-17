// Package watcher — daemon_profile_gate.go : porte « profil suivi » du daemon
// (ADR 0035 D3).
//
// Extrait de daemon.go pour ne pas grossir un fichier déjà au-dessus du seuil de
// 500 lignes (CLAUDE.md règle 5).
package watcher

import (
	"context"
	"errors"
	"log/slog"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/domain/title"
)

// ErrPlayerNotTracked est rendue par AddPlayer quand le couple (gamertag, titre)
// demandé n'a pas de profil suivi dans db_profiles.json (ADR 0035 D3). Le caller
// ne doit PAS la traiter comme une panne : c'est un refus légitime, dont la seule
// sortie est la création du profil (wizard de mise en place).
var ErrPlayerNotTracked = errors.New("watcher_daemon: joueur sans profil suivi — tracking refusé")

// WithProfileGate pose la porte « profil suivi » sur AddPlayer ET sur le
// coordinateur de sync que ce daemon possède — les deux portes que le chemin
// watcher franchit (prise en charge live, puis soumission d'un sync). Le
// Coordinator étant construit dans NewDaemon et n'étant exposé que derrière
// l'interface SyncGate, c'est ici le point de câblage unique des deux.
//
// À appeler AVANT Start (la porte est lue sans verrou). Gate nil = porte ouverte.
func (d *Daemon) WithProfileGate(g domain.ProfileGate) *Daemon {
	d.profileGate = g
	d.coordinator.WithProfileGate(g)
	return d
}

// checkProfileGate rend ErrPlayerNotTracked si le joueur n'a pas de profil suivi
// pour son titre. Sans profil : pas de poller — donc pas de détection de match,
// pas de sync, pas de player DB créée. C'est le maillon qui manquait le
// 2026-07-23, où le SSO ajoutait au watcher un compte qu'aucun profil ne
// déclarait. initPlayers n'en a pas besoin : sa liste est déjà filtrée par
// SyncablePlayers en amont.
//
// Le titre est normalisé AVANT d'interroger la porte, comme le fait playerKey :
// sinon un titre vide (les callers qui s'en remettent au défaut, dont le SSO)
// serait lu « tous les titres » par le chargeur de profils, et le profil d'un
// AUTRE jeu ouvrirait le suivi de celui-ci.
func (d *Daemon) checkProfileGate(ctx context.Context, p domain.PlayerSummary, evID string) error {
	gateTitle := p.TitleSlug
	if gateTitle == "" {
		gateTitle = title.DefaultSlug
	}
	if d.profileGate == nil || d.profileGate(ctx, gateTitle, p.XUID) {
		return nil
	}
	slog.WarnContext(ctx, "watcher_daemon: AddPlayer refusé — joueur sans profil suivi",
		"gamertag", p.Gamertag, "xuid", p.XUID, "title_slug", gateTitle, "event", evID)
	return ErrPlayerNotTracked
}
