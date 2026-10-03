package main

// provision_title.go — provisioning des warehouses des titres ADDITIONNELS au boot
// (MT-16 / day-one 2e titre). Extrait de main.go le 2026-09-20 avec le correctif
// ci-dessous ; le titre par défaut (halo_infinite) garde son chemin propre.
//
// CORRECTIF DU 2026-09-20 — TOUTES LES BASES SONT TRAITÉES. La boucle retournait à la
// PREMIÈRE erreur : entre le 2026-09-12 et le 2026-09-20, l'échec de la metadata de
// halo_5 (contrainte name_en NOT NULL violée par le seed du registre d'armes) a donc
// privé ce titre de ses migrations shared ET social à chaque boot, sous une seule ligne
// ERROR. Une base en échec ne dit RIEN des suivantes : chaque cible est désormais tentée,
// son échec est logué AU MOMENT où il survient (titre, cible, chemin, err) et toutes les
// erreurs remontent JOINTES. Le titre n'est déclaré provisionné que si tout a réussi.

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"path/filepath"

	"levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games/weapons"
	"levelup/go-api/internal/migration"
	"levelup/go-api/internal/platform/duckdb"
)

// provisionTarget — une DB partagée d'un titre à créer puis migrer.
type provisionTarget struct {
	path string
	kind migration.TargetDB
}

// targetProvisionError — échec d'UNE base. Le type existe pour que l'appelant puisse
// NOMMER les cibles en échec sans parser un message (failedProvisionTargets).
type targetProvisionError struct {
	kind migration.TargetDB
	path string
	err  error
}

func (e *targetProvisionError) Error() string {
	return fmt.Sprintf("%s (%s): %v", e.kind, e.path, e.err)
}

func (e *targetProvisionError) Unwrap() error { return e.err }

// failedProvisionTargets extrait les noms des bases en échec d'une erreur jointe par
// provisionAllTargets. Ordre de la liste de cibles ; vide si err n'en porte aucune.
func failedProvisionTargets(err error) []string {
	if err == nil {
		return nil
	}
	var joined interface{ Unwrap() []error }
	if !errors.As(err, &joined) {
		var single *targetProvisionError
		if errors.As(err, &single) {
			return []string{string(single.kind)}
		}
		return nil
	}
	var out []string
	for _, e := range joined.Unwrap() {
		var te *targetProvisionError
		if errors.As(e, &te) {
			out = append(out, string(te.kind))
		}
	}
	return out
}

// provisionTargetsFor liste les bases partagées d'un titre. La DB PvE n'en fait partie
// que si le titre déclare la capability Firefight (gating par capability, jamais par
// comparaison de slug — archlint no_slug_comparison).
func provisionTargetsFor(pr *title.PathResolver, td *title.TitleDescriptor) []provisionTarget {
	slug := td.Slug
	targets := []provisionTarget{
		{pr.MetadataDBPath(slug), migration.TargetMetadata},
		{pr.SharedDBPath(slug), migration.TargetShared},
		{pr.SharedSocialDBPath(slug), migration.TargetSharedSocial},
	}
	if td.HasCapability(title.CapFirefight) {
		targets = append(targets, provisionTarget{pr.SharedPVEDBPath(slug), migration.TargetSharedPvE})
	}
	return targets
}

// provisionAllTargets applique `apply` à CHAQUE cible, sans jamais s'arrêter au premier
// échec : les bases d'un titre sont INDÉPENDANTES (fichiers distincts, jeux de migrations
// distincts). Chaque échec est logué à l'instant où il survient, puis toutes les erreurs
// sont retournées JOINTES (errors.Join) — nil si tout a réussi.
func provisionAllTargets(ctx context.Context, slug string, targets []provisionTarget,
	apply func(provisionTarget) error) error {
	var errs []error
	for _, t := range targets {
		if err := apply(t); err != nil {
			slog.ErrorContext(ctx, "provisioning: base du titre additionnel en échec (les autres bases sont tout de même traitées)",
				"title", slug, "target", string(t.kind), "path", t.path, "err", err)
			errs = append(errs, &targetProvisionError{kind: t.kind, path: t.path, err: err})
		}
	}
	return errors.Join(errs...)
}

// provisionAdditionalActiveTitles crée + migre les warehouses des titres ADDITIONNELS
// actifs (slug != DefaultSlug) découverts dans le registre piloté par config (MT-16 /
// day-one 2e titre). Le titre par défaut (halo_infinite) est provisionné séparément
// (chemin byte-identique). Chaque échec est logué sans interrompre le boot — un titre
// additionnel cassé ne bloque jamais Halo.
func provisionAdditionalActiveTitles(pr *title.PathResolver, reg *title.Registry) {
	ctx := context.Background()
	for _, td := range reg.Active() {
		// Le titre par défaut (built-in) est provisionné par le chemin Halo
		// byte-identique ailleurs → on saute son descripteur ici (flag sémantique,
		// pas de comparaison de slug — archlint no_slug_comparison).
		if td.IsDefault {
			continue
		}
		if err := provisionAdditionalTitle(pr, td); err != nil {
			slog.ErrorContext(ctx, "provisioning titre additionnel échoué (non-fatal)",
				"title", td.Slug, "targets_failed", failedProvisionTargets(err), "err", err.Error())
			continue
		}
		slog.InfoContext(ctx, "titre additionnel provisionné", "title", td.Slug, "status", string(td.Status))
	}
}

// provisionAdditionalTitle crée le warehouse + applique les migrations des DB partagées
// d'un titre additionnel via RunForTitleDB (jeu de migrations du titre si enregistré via
// RegisterMigrationSet, sinon set Halo en fallback). Toutes les DB sont isolées par chemin
// sous data/titles/<slug>/. TOUTES les bases sont tentées : l'erreur retournée est la
// jonction des échecs (cf. le bandeau de tête du fichier).
func provisionAdditionalTitle(pr *title.PathResolver, td *title.TitleDescriptor) error {
	if err := ensureWarehouseDir(pr, td.Slug); err != nil {
		return fmt.Errorf("warehouse dir: %w", err)
	}
	return provisionAllTargets(context.Background(), td.Slug, provisionTargetsFor(pr, td),
		func(t provisionTarget) error { return provisionOneTarget(pr, td.Slug, t) })
}

// provisionOneTarget ouvre UNE base en écriture, y applique le jeu de migrations du titre
// et, pour la metadata, rejoue les réconciliations idempotentes du registre d'armes.
func provisionOneTarget(pr *title.PathResolver, slug string, t provisionTarget) error {
	db, err := duckdb.OpenReadWrite(t.path)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer db.Close()

	if err := migration.RunForTitleDB(db.SQLDb(), slug, t.kind); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}
	if t.kind != migration.TargetMetadata {
		return nil
	}
	// Auto-guérison du registre d'armes (cross-titre) sur la metadata du titre
	// additionnel : le seed add_weapon_registry étant sauté une fois « done », les
	// lignes ajoutées après coup (buckets non-combat H5…) manqueraient sans ce rejeu
	// idempotent (INSERT OR IGNORE). Comparaison de TargetDB, pas de slug.
	if _, err := weapons.ReconcileRegistry(db.SQLDb(), slug); err != nil {
		return fmt.Errorf("reconcile weapon registry: %w", err)
	}
	// V72-06 : source unique des noms d'armes du titre additionnel (idempotent).
	if _, err := weapons.ReconcileNameLabels(db.SQLDb(), slug,
		filepath.Join(pr.RepoRoot(), "config", "titles", slug, "mappings", "weapon_names.toml")); err != nil {
		return fmt.Errorf("reconcile weapon name labels: %w", err)
	}
	return nil
}
