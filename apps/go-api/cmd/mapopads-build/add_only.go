package main

// add_only.go — LE MODE « AJOUT SEUL » : livrer les points d'apparition sans toucher aux socles.
//
// Il existe parce que la source DERIVE. Seize des 72 cartes du catalogue rendent aujourd'hui
// un `.mvar` different de celui qui l'a bati : regenerer en bloc reecrirait leurs socles d'ARME,
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
// bilanAjout compte ce qu'une passe d'ajout a fait, pour le journal et pour la note.
type bilanAjout struct {
	ajoutees, points, sautees, sansDump, retirees, aZeroPoint int
	detailSautees                                             []string
}

// deriveDe rend la RAISON pour laquelle la source d'une carte ne concorde plus avec le
// catalogue, ou la chaine vide si tout concorde.
//
// LE VERROU EST EN TROIS TERMES ET PAS UN SEUL. `memesSocles` seul ne verifie RIEN sur une
// carte sans socle : deux listes vides sont egales, donc une carte a zero socle passait le
// verrou sans qu'aucune donnee soit comparee. `objects_n` et `level_id` sont precisement les
// signaux qui ont DETECTE la derive de Deadlock (462 objets au catalogue, 410 au
// telechargement) : ils comparent le FICHIER, pas seulement ce qu'on en a extrait.
func deriveDe(entry, neuf replay.MapWeaponPadsEntry) string {
	switch {
	case entry.ObjectsN != neuf.ObjectsN:
		return "objects_n"
	case entry.LevelID != neuf.LevelID:
		return "level_id"
	case !memesSocles(entry.Pads, neuf.Pads):
		return "socles"
	}
	return ""
}

// retirerPointsPerimes efface les points d'une carte dont la source a derive, et dit s'il y
// avait quelque chose a effacer.
//
// C'EST LE COEUR DU CORRECTIF. Sans cet effacement, une carte acceptee hier et derivee
// aujourd'hui GARDAIT ses points : le catalogue publiait alors des points issus d'une source
// qu'il venait lui-meme de declarer non concordante, et la note comptait cette carte parmi les
// « sans points » alors qu'elle en portait. Le mensonge etait double.
func retirerPointsPerimes(cat *replay.MapWeaponPadsCatalog, mapID string,
	entry replay.MapWeaponPadsEntry,
) bool {
	if entry.SpawnPoints == nil {
		return false
	}
	entry.SpawnPoints = nil
	cat.Maps[mapID] = entry
	return true
}

// ajouterPointsDUneCarte traite UNE carte : elle recoit ses points, ou elle est sautee.
func ajouterPointsDUneCarte(ctx context.Context, cat *replay.MapWeaponPadsCatalog, mapID string,
	e replay.MapObjectivesEntry, dumps *dumpIndex, b *bilanAjout,
) {
	entry := cat.Maps[mapID]
	path, base, ok := dumps.resolve(mapID, e)
	if !ok {
		// SANS DUMP, ON N'EFFACE RIEN. Des points etablis lors d'une passe precedente ne sont
		// pas CONTREDITS par l'absence de fichier aujourd'hui ; les effacer detruirait des
		// donnees valides des qu'on relance le generateur sur un depot partiel. Seule la
		// DERIVE contredit, et seule elle efface.
		b.sansDump++
		return
	}
	neuf, _, err := ingestFn(mapID, e, path, base)
	derive := ""
	if err != nil {
		derive = "variante illisible"
		slog.WarnContext(ctx, "mapopads: variante illisible", "map_id", mapID, "err", err)
	} else {
		derive = deriveDe(entry, neuf)
	}
	if derive != "" {
		b.sautees++
		b.detailSautees = append(b.detailSautees, mapID+" ("+derive+")")
		if retirerPointsPerimes(cat, mapID, entry) {
			b.retirees++
		}
		return
	}
	entry.SpawnPoints = neuf.SpawnPoints
	cat.Maps[mapID] = entry
	b.ajoutees++
	n := 0
	if neuf.SpawnPoints != nil {
		n = len(*neuf.SpawnPoints)
	}
	b.points += n
	if n == 0 {
		b.aZeroPoint++
	}
}

// noteDuCatalogue redige la note que le fichier porte sur lui-meme.
//
// `generated_at` n'est PAS touche : il date les SOCLES, et les socles n'ont pas bouge. Sans
// cette note, un lecteur du catalogue prendrait les cartes sans cle pour des cartes sans point.
func noteDuCatalogue(b bilanAjout) string {
	return "Points d'apparition d'objet ramassable NON-ARME, ajoutes le " +
		time.Now().UTC().Format("2006-01-02") + " par --only-add-spawn-points. " +
		itoaSimple(b.points) + " points sur " + itoaSimple(b.ajoutees) + " carte(s) ACCEPTEES, " +
		"dont " + itoaSimple(b.aZeroPoint) + " a zero point (cle `spawn_points` presente et " +
		"vide : la carte n'en porte aucun, ce n'est pas un trou). " +
		itoaSimple(b.sautees) + " carte(s) SAUTEES pour source derivee (objects_n, level_id ou " +
		"socles differents du catalogue) : leur cle `spawn_points` est RETIREE — " +
		itoaSimple(b.retirees) + " l'ont effectivement perdue lors de cette passe, les autres " +
		"n'en avaient pas. " + itoaSimple(b.sansDump) + " carte(s) sans .mvar au depot local : " +
		"leur cle est CONSERVEE telle quelle, l'absence de fichier ne contredit rien. " +
		"Une cle ABSENTE veut dire « points NON ETABLIS » et ne se confond pas avec `[]`. " +
		"Le trou est VOULU et visible ; il se comble par la CLI ou le sync, jamais pendant " +
		"une cuisson."
}

// addSpawnPointsOnly charge le catalogue existant et n'y ajoute que les points d'apparition.
//
// Cinq parametres, et pas sept : `res` et `titleSlug` etaient portes sans etre lus — le chemin
// du catalogue est deja resolu par l'appelant et passe en `outPath`.
func addSpawnPointsOnly(ctx context.Context, objectifs *replay.MapObjectivesCatalog,
	dumps *dumpIndex, outPath string, dryRun bool,
) {
	cat, err := replay.LoadMapWeaponPads(outPath)
	if err != nil {
		fail(ctx, "catalogue des socles existant", err)
	}
	var b bilanAjout
	for mapID := range cat.Maps {
		e, ok := objectifs.Maps[mapID]
		if !ok {
			b.sansDump++
			continue
		}
		ajouterPointsDUneCarte(ctx, cat, mapID, e, dumps, &b)
	}
	slog.InfoContext(ctx, "mapopads: ajout des points d'apparition",
		"cartes_enrichies", b.ajoutees, "points", b.points,
		"cartes_acceptees_a_zero_point", b.aZeroPoint,
		"cartes_sautees_source_derivee", b.sautees,
		"cartes_dont_points_retires", b.retirees, "cartes_sans_dump", b.sansDump)
	if len(b.detailSautees) > 0 {
		slog.WarnContext(ctx, "mapopads: cartes SAUTEES — leur .mvar ne redonne plus les "+
			"memes socles qu'au catalogue ; leur cle `spawn_points` est RETIREE",
			"n", len(b.detailSautees), "map_ids", b.detailSautees)
	}
	if cat.Notes == nil {
		cat.Notes = map[string]string{}
	}
	cat.Notes["spawn_points"] = noteDuCatalogue(b)
	if dryRun {
		slog.InfoContext(ctx, "mapopads: dry-run, rien ecrit")
		return
	}
	if err := writeCatalog(cat, outPath); err != nil {
		fail(ctx, "ecriture du catalogue", err)
	}
	slog.InfoContext(ctx, "mapopads: catalogue ecrit", "path", outPath, "cartes", len(cat.Maps))
}

// itoaSimple evite d importer fmt pour quelques entiers dans une note.
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
