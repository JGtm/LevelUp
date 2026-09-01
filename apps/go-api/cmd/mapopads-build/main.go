// cmd/mapopads-build — construit le catalogue figé des EMPLACEMENTS DE SOCLE par carte
// (socles d'arme et de power-up), HORS LIGNE, depuis le dépôt local de variantes `.mvar`.
//
// Usage :
//
//	go run ./cmd/mapopads-build --from <dossier de .mvar> [--title <slug>] [--dry-run]
//
// Sortie : data/titles/{slug}/reference/map_weapon_pads.json (via PathResolver).
// AUCUN appel réseau, AUCUNE base : les `.mvar` sont déjà sur disque, et le map_id de
// chacun se lit dans le catalogue d'objectifs déjà figé.
//
// POURQUOI UN FRÈRE DE cmd/mapobj-build ET NON UNE EXTENSION. Les trois raisons sont des
// faits de code, pas un goût :
//
//  1. mapobj-build est un producteur RÉSEAU (un appel UGC par carte) dont le mode hors
//     ligne `--refresh-from` relit un dépôt à SA disposition (`<dir>/{map_id}/{fichier}`,
//     ou le fichier à plat). Le dépôt de mesure, lui, nomme ses fichiers
//     `{carte}_{fichier}.mvar` : aucun des deux chemins de mapobj-build ne sait le lire.
//  2. Son type `catalog` — `write`, `mergeExisting`, `carryOver`, `catalogSchemaVersion` —
//     est lié à UN document, `map_objectives.json`. Y loger un second document, avec sa
//     propre version de schéma et sa propre politique de report, dédoublerait chacune de
//     ces quatre fonctions dans un fichier déjà à 179 lignes.
//  3. Ce producteur-ci CONSOMME la sortie de l'autre : `map_objectives.json` est le seul
//     index qui relie un map_id à un nom de fichier `.mvar`. Le frère lit donc le grand
//     frère ; il ne duplique pas sa chaîne réseau.
//
// CE QUE LE CATALOGUE NE FAIT PAS. Il ne dit pas si un socle est ALLUMÉ : le fichier de
// carte POSE, le mode ALLUME (Cliffhanger porte 17 socles, en rend 10 en CTF et ZÉRO en
// Super Fiesta). Le croisement avec les socles du match est OBLIGATOIRE en aval
// (internal/analysis/replay/map_weapon_pads.go).
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sort"
	"time"

	"levelup/go-api/internal/analysis/replay"
	titlePkg "levelup/go-api/internal/domain/title"
)

func main() {
	var (
		from      = flag.String("from", "", "dossier des .mvar dumpés (obligatoire)")
		titleSlug = flag.String("title", titlePkg.DefaultSlug, "slug du titre")
		dryRun    = flag.Bool("dry-run", false, "ne pas écrire le catalogue")
		// AJOUT SEUL — le mode de livraison des points d'apparition, et la raison d'etre du
		// flag est une NON-REGRESSION, pas un confort.
		//
		// Les `.mvar` servis par l'UGC DERIVENT : neuf cartes du catalogue rendent aujourd'hui
		// un fichier different de celui qui l'a bati le 2026-08-19 (Deadlock 462 objets au
		// catalogue, 410 au telechargement d'aujourd'hui). Une regeneration complete
		// reecrirait donc leurs socles d'ARME — or ces socles alimentent des chemins livres
		// (datation des occupations, tableau de la page match).
		//
		// Ce mode charge le catalogue EXISTANT, recalcule les socles pour VERIFIER qu'ils
		// retombent a l'identique, et n'ecrit QUE `spawn_points`. Une carte dont les socles ne
		// retombent pas est SAUTEE et COMPTEE : le trou se voit, il ne se comble pas en
		// douce.
		addOnly = flag.Bool("only-add-spawn-points", false,
			"ne pas reecrire les socles : charger le catalogue existant et n'y ajouter que "+
				"les points d'apparition, en sautant toute carte dont les socles auraient change")
	)
	flag.Parse()

	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelInfo})))
	ctx := context.Background()
	if *from == "" {
		fail(ctx, "arguments", fmt.Errorf("--from <dossier de .mvar> est obligatoire"))
	}
	root, err := titlePkg.FindRepoRoot()
	if err != nil {
		fail(ctx, "racine du dépôt", err)
	}
	res := titlePkg.NewPathResolver(root)

	// L'INDEX DES CARTES vient du catalogue d'objectifs : c'est la seule table qui relie un
	// map_id à un fichier de variante. Une carte qui n'y est pas n'a pas de map_id
	// atteignable hors ligne — elle est ABSENTE du catalogue des socles, jamais devinée.
	objectifs, err := replay.LoadMapObjectives(res.MapObjectivesPath(*titleSlug))
	if err != nil {
		fail(ctx, "catalogue d'objectifs", err)
	}
	dumps, err := newDumpIndex(*from)
	if err != nil {
		fail(ctx, "dépôt de .mvar", err)
	}
	slog.InfoContext(ctx, "mapopads: sources lues",
		"cartes_catalogue", len(objectifs.Maps), "fichiers_dumpes", dumps.count(), "dossier", *from)

	outPath := res.MapWeaponPadsPath(*titleSlug)
	if *addOnly {
		addSpawnPointsOnly(ctx, res, *titleSlug, objectifs, dumps, outPath, *dryRun)
		return
	}
	cat := newPadsCatalog(*titleSlug)
	build(ctx, cat, objectifs, dumps)

	if *dryRun {
		slog.InfoContext(ctx, "mapopads: dry-run, rien écrit", "cartes", len(cat.Maps))
		return
	}
	if err := writeCatalog(cat, outPath); err != nil {
		fail(ctx, "écriture du catalogue", err)
	}
	slog.InfoContext(ctx, "mapopads: catalogue écrit", "path", outPath, "cartes", len(cat.Maps))
}

// build parcourt les cartes du catalogue d'objectifs, DANS L'ORDRE DES map_id, et ingère
// celles dont le `.mvar` est au dépôt local.
//
// UNE CARTE SANS FICHIER EST ABSENTE, ET ELLE SE DIT : un emplacement deviné vaudrait moins
// que rien du tout, puisque le calque n'a aucun moyen de distinguer un socle inventé d'un
// socle mesuré une fois qu'il est publié.
func build(ctx context.Context, cat *replay.MapWeaponPadsCatalog,
	objectifs *replay.MapObjectivesCatalog, dumps *dumpIndex) {
	ids := make([]string, 0, len(objectifs.Maps))
	for id := range objectifs.Maps {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	var absents, melanges, sansSocle []string
	totalPads, mixtes := 0, 0
	for _, id := range ids {
		e := objectifs.Maps[id]
		path, base, ok := dumps.resolve(id, e)
		if !ok {
			absents = append(absents, fmt.Sprintf("%s (%s)", id, e.MvarFile))
			continue
		}
		entry, mixed, err := ingest(id, e, path, base)
		if err != nil {
			// Échec documenté, jamais contourné (même arbitrage que mapobj-build) : une
			// carte absente vaut mieux qu'un socle au mauvais endroit.
			slog.ErrorContext(ctx, "mapopads: variante non ingérée",
				"map_id", id, "file", base, "err", err)
			absents = append(absents, fmt.Sprintf("%s (%s, parse)", id, base))
			continue
		}
		if len(entry.Pads) == 0 {
			// MESURÉE ET VIDE, ce qui n'est pas la même chose qu'absente : sur une carte
			// Forge, le CANEVAS ne porte aucun socle et le rack les porte tous. L'entrée
			// reste publiée pour que le silence se distingue de l'ignorance.
			sansSocle = append(sansSocle, base)
		}
		if mixed > 0 {
			melanges = append(melanges, fmt.Sprintf("%s:%d", base, mixed))
		}
		mixtes += mixed
		totalPads += len(entry.Pads)
		cat.Maps[id] = entry
	}
	slog.InfoContext(ctx, "mapopads: extraction terminée",
		"cartes", len(cat.Maps), "emplacements", totalPads, "cartes_sans_socle", len(sansSocle),
		"emplacements_familles_melangees", mixtes)
	if len(absents) > 0 {
		slog.WarnContext(ctx, "mapopads: cartes sans .mvar au dépôt — absentes du catalogue",
			"n", len(absents), "detail", absents)
	}
	if len(sansSocle) > 0 {
		slog.InfoContext(ctx, "mapopads: cartes MESUREES sans aucun socle",
			"n", len(sansSocle), "detail", sansSocle)
	}
	if len(melanges) > 0 {
		// Deux objets de familles DIFFÉRENTES à moins d'un mètre : la famille publiée
		// est celle du représentant. Le dire plutôt que le taire.
		slog.WarnContext(ctx, "mapopads: emplacements a familles melangees",
			"n", mixtes, "detail", melanges)
	}
}

// newPadsCatalog pose l'en-tête du document, notes comprises.
func newPadsCatalog(titleSlug string) *replay.MapWeaponPadsCatalog {
	return &replay.MapWeaponPadsCatalog{
		SchemaVersion: replay.MapWeaponPadsSchemaVersion,
		TitleSlug:     titleSlug,
		GeneratedAt:   time.Now().UTC(),
		Maps:          map[string]replay.MapWeaponPadsEntry{},
		Notes:         padsNotes(),
	}
}

// padsNotes est ce que le document dit de lui-même. Il se lit hors de tout code : ces
// notes sont sa documentation embarquée, comme celles de map_objectives.json.
func padsNotes() map[string]string {
	return map[string]string{
		"source": "variantes de carte UGC (.mvar) déjà dumpées, Bond CompactBinary v2 ; " +
			"map_id joint depuis map_objectives.json",
		"repere": "positions en repère monde du .mvar, mètres, non transformées — " +
			"le même repère que les positions joueur du rejeu",
		"type_id": "type brut de l'objet, en hexadécimal : 0x5F379533, 0x6253CFC0, " +
			"0x5E86D110 — les trois seuls types de socle mesurés (32/32 appariés à " +
			"moins d'un mètre sur trois cartes, médiane 0,01 m)",
		"family": "INFÉRENCE mesurée par corrélation avec les armes observées, publiée " +
			"À CÔTÉ du type_id brut et jamais à sa place : power (épée, marteau, SPNKr, " +
			"Cindershot, S7 Sniper), rack (Bulldog, Disruptor, Mangler, Commando, " +
			"Vestige, BR75, Sentinel Beam), powerup",
		"objects": "nombre d'objets du fichier fusionnés dans l'emplacement (moins d'un " +
			"mètre) : deux déclarations voisines sont UN socle, pas deux",
		"allumage": "CE CATALOGUE NE DIT PAS SI UN SOCLE EST ALLUMÉ. Le fichier de carte " +
			"POSE, le mode ALLUME : Cliffhanger porte 17 socles, en rend 10 en CTF et " +
			"ZÉRO en Super Fiesta. Le servir brut afficherait des socles fantômes — le " +
			"croisement avec les socles du match est obligatoire (map_weapon_pads.go)",
		"pads_vides": "une carte présente avec `pads: []` a été MESURÉE sans socle ; une " +
			"carte absente du document n'a simplement pas de .mvar au dépôt local",
		"regeneration": "go run ./cmd/mapopads-build --from <dossier de .mvar>",
	}
}

// writeCatalog sérialise le catalogue de façon atomique (temporaire + rename).
func writeCatalog(cat *replay.MapWeaponPadsCatalog, path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	buf, err := json.MarshalIndent(cat, "", "  ")
	if err != nil {
		return err
	}
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, buf, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func fail(ctx context.Context, what string, err error) {
	slog.ErrorContext(ctx, "mapopads: échec", "etape", what, "err", err)
	os.Exit(1)
}
