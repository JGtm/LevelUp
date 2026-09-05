package main

// cmd_replay_facts_export.go — sous-commande `levelup replay-facts-export`.
//
// ELLE EXPORTE, ELLE NE DECODE RIEN. Pour chaque match nomme (forme courte ou complete), elle
// ecrit ce que la base sait du match et que le film ne dit pas — les FAITS (lignes de match,
// scores des deux camps, variante, map_id ; cf. domain.MatchFacts) et les identites de carte
// candidates, dans l'ordre que ResolveMapEntry essaie — dans un fichier JSON de la forme que
// `cmd/replay-build --facts` lit deja (`loadFacts`).
//
// # POURQUOI ELLE EXISTE (PLAN_CUISSON_PERF §4, item 0.2)
//
// Le harnais d'equivalence de la cuisson (`cmd/replay-equiv`) rejoue la construction HORS
// LIGNE, un film par processus, sans base. Sans les faits, zones, actions d'objectif,
// VIP/crane/bombe, socles et points d'apparition sont court-circuites (replaybuild/zones.go,
// matchfacts.go) et l'equivalence ne les verrait pas. Les faits sont donc FIGES une fois, dans
// testdata, a cote des digests qu'ils conditionnent.
//
// # UNE SEULE FORME DE RESOLUTION, CELLE DU BACKFILL
//
// Le registre et les cartes candidates viennent de `registreParShort`, les faits de
// `ReplayFactsRepo` — exactement ce que l'enfant de `backfill-replay` lit pour cuire. Une
// troisieme ecriture des memes requetes divergerait (regle des deux copies du depot).
//
// # LECTURE SEULE, ET UN ECHEC FRANC PLUTOT QU'UN FICHIER VIDE
//
// `OpenReadForQuery` (jamais `OpenReadOnly` force). Un serveur qui tient la base en ECRITURE
// fait echouer l'ouverture (« File is already open ») : la commande echoue alors, et des faits
// vides pour un match present au registre echouent aussi — un fichier de faits vides rendrait
// l'equivalence vacuante sans le dire, ce qui est pire qu'aucun fichier.
//
// Usage :
//
//	levelup replay-facts-export --out <dossier> [--title halo_infinite] <short8|match_id>...

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/port"
	"levelup/go-api/internal/replaybuild"
)

// La forme ecrite est `replaybuild.FactsFile` — les faits du match A PLAT (la forme exacte que
// `cmd/replay-build --facts` desserialise en port.MatchFacts), plus deux champs que ce lecteur
// ignore : l'identite complete du match et ses cartes candidates. Elle vit dans `replaybuild`
// parce que c'est LUI qui la relit (harnais d'equivalence) : une copie ici derivait en silence.

// runReplayFactsExport ecrit un <short8>.facts.json par match demande.
func runReplayFactsExport(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("replay-facts-export", flag.ContinueOnError)
	out := fs.String("out", "", "dossier de sortie des <short8>.facts.json (obligatoire)")
	titleSlug := fs.String("title", titlePkg.DefaultSlug, "slug du titre")
	if err := fs.Parse(args); err != nil {
		return err
	}
	ids := fs.Args()
	if *out == "" || len(ids) == 0 {
		return errors.New("usage : levelup replay-facts-export --out <dossier> [--title slug] <short8|match_id>...")
	}
	ctx := context.Background()
	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	registre, err := registreParShort(ctx, cfg, pr, *titleSlug)
	if err != nil {
		return err
	}
	db, release, err := duckdb.OpenReadForQuery(pr.SharedDBPath(*titleSlug))
	if err != nil {
		return fmt.Errorf("open shared RO : %w (serveur en ecriture ? l'arreter le temps de l'export)", err)
	}
	defer release()
	var repo port.ReplayFactsRepo = duckdb.NewReplayFactsRepo(db)
	if err := os.MkdirAll(*out, 0o750); err != nil {
		return fmt.Errorf("dossier de sortie : %w", err)
	}
	for _, id := range ids {
		short := titlePkg.FilmShortMatchID(id)
		entry, ok := registre[short]
		if !ok {
			return fmt.Errorf("%s : match absent du registre", id)
		}
		facts, err := repo.FactsForMatch(ctx, entry.matchID)
		if err != nil {
			return fmt.Errorf("%s : faits illisibles : %w", short, err)
		}
		if facts.Empty() {
			return fmt.Errorf("%s : faits VIDES pour un match present au registre — export refuse", short)
		}
		blob, err := json.MarshalIndent(
			replaybuild.FactsFile{MatchFacts: facts, MatchID: entry.matchID, MapNames: entry.mapNames}, "", "  ")
		if err != nil {
			return fmt.Errorf("%s : serialisation : %w", short, err)
		}
		path := filepath.Join(*out, short+".facts.json")
		if err := os.WriteFile(path, append(blob, '\n'), 0o600); err != nil {
			return fmt.Errorf("%s : ecriture : %w", short, err)
		}
		fmt.Printf("  %s : %d joueur(s), variante %q, cartes %v -> %s\n",
			short, len(facts.Players), facts.GameVariantName, entry.mapNames, path)
	}
	return nil
}
