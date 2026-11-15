package service

// spawn_model_poids.go — LECTURE DU FICHIER DES POIDS DU MODELE DE REAPPARITION.
//
// Un fichier absent, illisible ou d'une autre revision n'arrete rien : l'appelant sert alors la
// prevision sans modele (ou rien), et ce repli est JOURNALISE et COMPTE a chaque lecture — un
// modele qui disparait ne doit pas se voir seulement par ses effets.

import (
	"context"
	"encoding/json"
	"errors"
	"io/fs"
	"log/slog"
	"os"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/observability"
)

// CompteurSpawnModelRepli compte les lectures qui n'ont pas pu rendre un fichier de poids.
const CompteurSpawnModelRepli = "spawn_model_poids_repli_total"

// ChargerPoidsReapparition lit le fichier de poids d'un titre. Faux : pas de modele (absent,
// illisible, ou d'une revision que ce binaire ne sait pas lire).
func ChargerPoidsReapparition(ctx context.Context, titleSlug, path string) (domain.PoidsReapparition, bool) {
	raw, err := os.ReadFile(path)
	if err != nil {
		raison := "lecture"
		if errors.Is(err, fs.ErrNotExist) {
			raison = "absent"
		}
		return replierPoids(ctx, titleSlug, path, raison, err)
	}
	var p domain.PoidsReapparition
	if err := json.Unmarshal(raw, &p); err != nil {
		return replierPoids(ctx, titleSlug, path, "format", err)
	}
	if p.Revision != domain.SpawnModelRevision {
		return replierPoids(ctx, titleSlug, path, "revision", nil)
	}
	return p, true
}

func replierPoids(ctx context.Context, titleSlug, path, raison string, err error) (domain.PoidsReapparition, bool) {
	slog.WarnContext(ctx, "prevision de reapparition : poids indisponibles, repli sans modele",
		"titleSlug", titleSlug, "path", path, "raison", raison, "err", err)
	observability.IncCounterT(titleSlug, CompteurSpawnModelRepli)
	return domain.PoidsReapparition{}, false
}
