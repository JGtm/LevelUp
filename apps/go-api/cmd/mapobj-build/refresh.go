// cmd/mapobj-build — refresh.go : régénération HORS LIGNE du catalogue.
//
// Pourquoi ce mode existe : ajouter un champ au contrat (la forme de zone, v2)
// ne doit pas obliger à re-solliciter l'UGC pour 34 cartes. Les `.mvar` sont
// déjà sur disque ; leurs noms de fichier sont ceux que le catalogue a
// enregistrés au moment du gel. On les re-parse et on ne remplace que ce qui
// vient du parse — les métadonnées issues du réseau (map_id, version_id,
// public_name, fetched_at) sont PRÉSERVÉES telles quelles.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"levelup/go-api/internal/analysis/replay/mapvar"
)

// loadAnyVersion lit un catalogue existant SANS exiger la version courante :
// c'est précisément une migration de schéma que ce mode réalise.
func loadAnyVersion(path string) (*catalog, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var prev catalog
	if err := json.Unmarshal(raw, &prev); err != nil {
		return nil, fmt.Errorf("catalogue existant illisible: %w", err)
	}
	return &prev, nil
}

// refreshOffline re-parse chaque carte du catalogue depuis un dossier de
// `.mvar` locaux. Une carte dont le fichier manque est CONSERVÉE telle quelle
// et signalée : un objectif périmé vaut mieux qu'un objectif effacé, mais le
// silence ne vaut rien.
func refreshOffline(ctx context.Context, dir, outPath, titleSlug string, dryRun bool) error {
	prev, err := loadAnyVersion(outPath)
	if err != nil {
		return err
	}
	slog.InfoContext(ctx, "mapobj: catalogue lu",
		"path", outPath, "schema", prev.SchemaVersion, "cartes", len(prev.Maps))

	cat := newCatalog(titleSlug)
	var missing, failed []string
	ids := make([]string, 0, len(prev.Maps))
	for id := range prev.Maps {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	shaped, total := 0, 0
	for _, id := range ids {
		e := prev.Maps[id]
		if e.MvarFile == "" {
			missing = append(missing, id+" (aucun mvar_file)")
			carryOver(cat, prev, id)
			continue
		}
		path := filepath.Join(dir, e.MvarFile)
		buf, err := os.ReadFile(path)
		if err != nil {
			slog.WarnContext(ctx, "mapobj: .mvar local absent, carte conservée en l'état",
				"map_id", id, "file", e.MvarFile, "err", err)
			missing = append(missing, e.MvarFile)
			carryOver(cat, prev, id)
			continue
		}
		v, err := mapvar.Parse(buf)
		if err != nil {
			slog.ErrorContext(ctx, "mapobj: parse local en échec",
				"map_id", id, "file", e.MvarFile, "err", err)
			failed = append(failed, e.MvarFile)
			carryOver(cat, prev, id)
			continue
		}
		// Métadonnées réseau préservées, données de parse remplacées.
		fresh := &mapEntry{
			MapID: e.MapID, VersionID: e.VersionID, PublicName: e.PublicName,
			MvarFile: e.MvarFile, Module: e.Module, FetchedAt: e.FetchedAt,
		}
		cat.addVariant(fresh, v)
		for _, ob := range fresh.Objectives {
			total++
			if ob.Shape != nil {
				shaped++
			}
		}
		slog.InfoContext(ctx, "mapobj: carte régénérée hors ligne",
			"map_id", id, "file", e.MvarFile,
			"objectifs", len(fresh.Objectives), "objets", fresh.ObjectsN)
	}

	// Les cartes reportées d'un schéma antérieur ne sont PAS comptées dans `total` :
	// elles n'ont pas été re-parsées. Les annoncer à part évite de lire « sans_forme: 0 »
	// comme « tout est migré ».
	carried := 0
	for _, e := range cat.Maps {
		if e != nil && e.CarriedFromSchema > 0 {
			carried++
		}
	}
	slog.InfoContext(ctx, "mapobj: régénération terminée",
		"cartes", len(cat.Maps), "objectifs", total, "avec_forme", shaped,
		"sans_forme", total-shaped, "cartes_non_migrees", carried)
	if len(missing) > 0 {
		slog.WarnContext(ctx, "mapobj: .mvar manquants — cartes non rafraîchies",
			"n", len(missing), "detail", strings.Join(missing, " | "))
	}
	if len(failed) > 0 {
		return fmt.Errorf("%d carte(s) en échec de parse: %s", len(failed), strings.Join(failed, " | "))
	}
	if dryRun {
		slog.InfoContext(ctx, "mapobj: dry-run, rien écrit", "cartes", len(cat.Maps))
		return nil
	}
	return cat.write(outPath)
}

// carryOver recopie une carte non rafraîchie, avec sa couverture.
//
// Elle la MARQUE quand elle vient d'un schéma antérieur : la carte n'a pas été migrée,
// et le document qui l'accueille s'annonce pourtant en schéma courant. Sans ce marqueur,
// une zone v1 (aucun `shape`) est indiscernable d'un objectif ponctuel v2 — le champ
// `carried_from_schema` de mapEntry porte le pourquoi.
//
// Un report depuis le schéma COURANT n'est pas marqué : il n'y a rien à distinguer. Un
// marqueur déjà posé par un refresh antérieur est conservé tel quel — il ne tombe que le
// jour où la carte est réellement re-parsée (addVariant repart d'une entrée neuve).
func carryOver(cat *catalog, prev *catalog, id string) {
	e := prev.Maps[id]
	if e != nil && prev.SchemaVersion < catalogSchemaVersion {
		clone := *e
		clone.CarriedFromSchema = prev.SchemaVersion
		e = &clone
	}
	cat.Maps[id] = e
	cat.Coverage[id] = prev.Coverage[id]
}
