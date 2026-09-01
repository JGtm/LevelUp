package main

// add_only.go — LE MODE « AJOUT SEUL » : livrer les points d'apparition sans toucher aux socles.
//
// Il existe parce que la source DERIVE. Neuf des 72 cartes du catalogue rendent aujourd'hui un
// `.mvar` different de celui qui l'a bati : regenerer en bloc reecrirait leurs socles d'ARME,
// qui alimentent des chemins livres. Ce mode rend la non-regression STRUCTURELLE plutot que
// verifiee apres coup — il ne peut pas ecrire un socle, le code ne le lui permet pas.

import (
	"context"
	"encoding/json"
	"log/slog"
	"strconv"
	"time"

	"levelup/go-api/internal/analysis/replay"
	titlePkg "levelup/go-api/internal/domain/title"
)

// addSpawnPointsOnly charge le catalogue existant et n'y ajoute que les points d'apparition.
func addSpawnPointsOnly(ctx context.Context, res *titlePkg.PathResolver, titleSlug string,
	objectifs *replay.MapObjectivesCatalog, dumps *dumpIndex, outPath string, dryRun bool,
) {
	cat, err := replay.LoadMapWeaponPads(outPath)
	if err != nil {
		fail(ctx, "catalogue des socles existant", err)
	}
	var ajoutees, points, sautees, sansDump int
	var detailSautees []string
	for mapID, entry := range cat.Maps {
		e, ok := objectifs.Maps[mapID]
		if !ok {
			sansDump++
			continue
		}
		path, base, ok := dumps.resolve(mapID, e)
		if !ok {
			sansDump++
			continue
		}
		neuf, _, err := ingest(mapID, e, path, base)
		if err != nil {
			slog.WarnContext(ctx, "mapopads: variante illisible, carte sautee",
				"map_id", mapID, "err", err)
			sautees++
			continue
		}
		// LE VERROU. Si les socles recalculs ne retombent pas EXACTEMENT sur ceux du
		// catalogue, la source a derive : on ne sait pas si les points d'apparition
		// decrivent la meme carte que les socles publies. On saute, et on le dit.
		if !memesSocles(entry.Pads, neuf.Pads) {
			sautees++
			detailSautees = append(detailSautees, mapID)
			continue
		}
		entry.SpawnPoints = neuf.SpawnPoints
		cat.Maps[mapID] = entry
		ajoutees++
		points += len(neuf.SpawnPoints)
	}
	slog.InfoContext(ctx, "mapopads: ajout des points d'apparition",
		"cartes_enrichies", ajoutees, "points", points,
		"cartes_sautees_source_derivee", sautees, "cartes_sans_dump", sansDump)
	if len(detailSautees) > 0 {
		slog.WarnContext(ctx, "mapopads: cartes SAUTEES — leur .mvar ne redonne plus les "+
			"memes socles qu'au catalogue ; leurs points d'apparition ne sont PAS ecrits",
			"n", len(detailSautees), "map_ids", detailSautees)
	}
	// LE FICHIER SE DOCUMENTE LUI-MEME. `generated_at` n'est PAS touche : il date les SOCLES,
	// et les socles n'ont pas bouge. Une note dit ce qui a ete ajoute, quand, et combien de
	// cartes sont restees sans points — sans quoi un lecteur du catalogue prendrait les neuf
	// trous pour des cartes sans point d'apparition.
	if cat.Notes == nil {
		cat.Notes = map[string]string{}
	}
	cat.Notes["spawn_points"] = "Points d'apparition d'objet ramassable NON-ARME, ajoutes le " +
		time.Now().UTC().Format("2006-01-02") + " par --only-add-spawn-points. " +
		itoaSimple(points) + " points sur " + itoaSimple(ajoutees) + " cartes. " +
		itoaSimple(sautees) + " carte(s) SANS points : leur .mvar servi par l'UGC ne redonne " +
		"plus les memes socles qu'au catalogue, donc on ne sait pas s'il decrit la meme " +
		"carte — le trou est VOULU et visible, il ne se comble pas pendant une cuisson."
	if dryRun {
		slog.InfoContext(ctx, "mapopads: dry-run, rien ecrit")
		return
	}
	if err := writeCatalog(cat, outPath); err != nil {
		fail(ctx, "ecriture du catalogue", err)
	}
	slog.InfoContext(ctx, "mapopads: catalogue ecrit", "path", outPath, "cartes", len(cat.Maps))
}

// itoaSimple evite d'importer fmt pour trois entiers dans une note.
func itoaSimple(v int) string { return strconv.Itoa(v) }

// memesSocles compare deux listes de socles par leur serialisation JSON — la MEME comparaison
// que celle du gate de non-regression, pour qu'un ecart invisible a l'oeil ne passe pas.
func memesSocles(a, b []replay.MapWeaponPadSpot) bool {
	ja, err1 := json.Marshal(a)
	jb, err2 := json.Marshal(b)
	if err1 != nil || err2 != nil {
		return false
	}
	return string(ja) == string(jb)
}
