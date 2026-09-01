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
)

// ingestFn est la couture qui rend la DERIVE testable. Sans elle, exercer le chemin
// « source derivee » exigerait de fabriquer un `.mvar` synthetique, et la garde restait donc
// non cablee : la supprimer ne faisait tomber aucun test. La variable n'existe que pour cela,
// et le code de production ne la reassigne jamais.
var ingestFn = ingest

// addSpawnPointsOnly charge le catalogue existant et n'y ajoute que les points d'apparition.
// Cinq parametres, et pas sept : `res` et `titleSlug` etaient portes sans etre lus — le chemin
// du catalogue est deja resolu par l'appelant et passe en `outPath`.
func addSpawnPointsOnly(ctx context.Context, objectifs *replay.MapObjectivesCatalog,
	dumps *dumpIndex, outPath string, dryRun bool,
) {
	cat, err := replay.LoadMapWeaponPads(outPath)
	if err != nil {
		fail(ctx, "catalogue des socles existant", err)
	}
	var ajoutees, points, sautees, sansDump, retirees, aZeroPoint int
	var detailSautees []string
	for mapID, entry := range cat.Maps {
		e, ok := objectifs.Maps[mapID]
		if !ok {
			sansDump++
			continue
		}
		path, base, ok := dumps.resolve(mapID, e)
		if !ok {
			// SANS DUMP, ON N'EFFACE RIEN. Des points etablis lors d'une passe precedente ne
			// sont pas CONTREDITS par l'absence de fichier aujourd'hui ; les effacer
			// detruirait des donnees valides des qu'on relance le generateur sur un depot
			// partiel. Seule la DERIVE contredit, et seule elle efface.
			sansDump++
			continue
		}
		neuf, _, err := ingestFn(mapID, e, path, base)
		derive := ""
		switch {
		case err != nil:
			derive = "variante illisible"
			slog.WarnContext(ctx, "mapopads: variante illisible", "map_id", mapID, "err", err)
		// LE VERROU, EN TROIS TERMES ET PAS UN SEUL.
		//
		// `memesSocles` seul ne verifie RIEN sur une carte sans socle : deux listes vides sont
		// egales, donc Corpo (0 socle) passait le verrou sans qu'aucune donnee soit comparee.
		// `objects_n` et `level_id` sont precisement les signaux qui ont DETECTE la derive de
		// Deadlock (462 objets au catalogue, 410 au telechargement) : ils comparent le
		// FICHIER, pas seulement ce qu'on en a extrait.
		case entry.ObjectsN != neuf.ObjectsN:
			derive = "objects_n"
		case entry.LevelID != neuf.LevelID:
			derive = "level_id"
		case !memesSocles(entry.Pads, neuf.Pads):
			derive = "socles"
		}
		if derive != "" {
			sautees++
			detailSautees = append(detailSautees, mapID+" ("+derive+")")
			// ON EFFACE LES POINTS DE LA PASSE PRECEDENTE, et c'est le coeur du correctif.
			//
			// Sans cet effacement, une carte acceptee hier et derivee aujourd'hui GARDAIT ses
			// points : le catalogue aurait alors publie des points issus d'une source qu'il
			// vient lui-meme de declarer non concordante — et la note aurait compte cette carte
			// parmi les « sans points » alors qu'elle en portait. Le mensonge etait double.
			if entry.SpawnPoints != nil {
				entry.SpawnPoints = nil
				cat.Maps[mapID] = entry
				retirees++
			}
			continue
		}
		entry.SpawnPoints = neuf.SpawnPoints
		cat.Maps[mapID] = entry
		ajoutees++
		n := 0
		if neuf.SpawnPoints != nil {
			n = len(*neuf.SpawnPoints)
		}
		points += n
		if n == 0 {
			aZeroPoint++
		}
	}
	slog.InfoContext(ctx, "mapopads: ajout des points d'apparition",
		"cartes_enrichies", ajoutees, "points", points,
		"cartes_acceptees_a_zero_point", aZeroPoint,
		"cartes_sautees_source_derivee", sautees,
		"cartes_dont_points_retires", retirees, "cartes_sans_dump", sansDump)
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
		itoaSimple(points) + " points sur " + itoaSimple(ajoutees) + " carte(s) ACCEPTEES, " +
		"dont " + itoaSimple(aZeroPoint) + " a zero point (cle `spawn_points` presente et " +
		"vide : la carte n'en porte aucun, ce n'est pas un trou). " +
		itoaSimple(sautees) + " carte(s) SAUTEES pour source derivee (objects_n, level_id ou " +
		"socles differents du catalogue), dont " + itoaSimple(retirees) + " dont les points " +
		"d'une passe precedente ont ete RETIRES. " + itoaSimple(sansDump) + " carte(s) sans " +
		".mvar au depot local. Pour une carte sautee ou sans dump, la cle `spawn_points` est " +
		"ABSENTE : les points ne sont pas ETABLIS, ce qui ne se confond pas avec `[]`. Le trou " +
		"est VOULU et visible ; il se comble par la CLI ou le sync, jamais pendant une cuisson."
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
