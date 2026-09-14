package main

// cmd_backfill_pad_tiers.go — sous-commande `levelup backfill-pad-tiers`.
//
// ELLE REMPLIT `shared.match_pad_pickups_by_tier` POUR LES ARTEFACTS DEJA CUITS. Le fil de
// l eau ne couvre que les artefacts construits par les cycles a venir (etape post-sync, cf.
// internal/sync/replayartifacts/padtiers.go) : le corpus existant passe par cette commande.
//
// # ELLE NE DECODE AUCUN FILM, ET ELLE N EN RE-CUIT AUCUN
//
// La source est l ARTEFACT de rejeu deja range (`data/cache/replays/{slug}/{short8}.json`),
// dont les socles (`weaponPads`, `padPickups`) et les equipements de depart (`loadouts`) sont
// publies depuis le schema 30. Aucune recuisson n est necessaire — et aucune n est declenchee :
// cuire des artefacts en lot est INTERDIT (bombe RAM, verrou `filmproc.AcquireSolo`). Cette
// passe ne fait que LIRE et ECRIRE, un artefact a la fois.
//
// C EST LE MEME MOTIF QUE `backfill-flag-grabs-net`, et pour la meme raison : la mesure se
// fait A LA LECTURE.
//
// # DEUX PORTES, ET L UNE N EST PAS UNE CAPABILITY
//
//	`film.weapon_tiers`                         le titre sait lire ces socles — sans elle,
//	                                            RIEN n est produit ;
//	`[weapon_tiers].random_start_mode_prefixes` ses modes a departs aleatoires — son absence
//	                                            n eteint rien, elle fait seulement qu aucun
//	                                            mode n est tenu pour aleatoire.
//
// # ELLE EST REPRENABLE, ET LA CLE EST LA PRESENCE EN BASE
//
// Un match dont la vue `match_pad_pickups_by_tier_latest` porte deja au moins une ligne est
// saute, sauf `--force`. ⚠ UNE REFERENCE DE CARTES COMPLETEE EXIGE `--force` : les lignes deja
// ecrites portent l ANCIEN croisement (des prises en `non_classe` qui deviendraient
// `terrain` ou `puissance`), et la reprise ne les reverrait jamais. Le bilan imprime les
// socles confirmes pour que ce point se verifie au lieu de se supposer.
//
// Usage (SERVEUR ARRETE — `OpenReadWrite` echoue si le lock est tenu ; meme precondition que
// backfill-flag-grabs-net, y compris pour --dry-run qui joue les migrations) :
//
//	levelup backfill-pad-tiers --dry-run        # une ligne par match, aucune ecriture
//	levelup backfill-pad-tiers --match <uuid>   # un seul match
//	levelup backfill-pad-tiers                  # tout le corpus, reprenable
//	levelup backfill-pad-tiers --force          # re-ecrit meme ce qui est deja en base

import (
	"context"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"time"

	"levelup/go-api/internal/config"
	titlePkg "levelup/go-api/internal/domain/title"
	"levelup/go-api/internal/games"
	"levelup/go-api/internal/games/halo_infinite/film/replay"
	"levelup/go-api/internal/games/mappings"
	"levelup/go-api/internal/persist"
	"levelup/go-api/internal/platform/duckdb"
	"levelup/go-api/internal/sync/replayartifacts"
)

// padTiersOptions : les reglages de la passe.
type padTiersOptions struct {
	titleSlug string
	limit     int
	force     bool
	dryRun    bool
	match     string
}

// bilanPadTiersBackfill : ce que la passe a fait — et pourquoi elle a saute ce qu elle a saute.
//
// `sansPrise` N EST PAS UN ECHEC : c est un artefact anterieur au schema 30 (aucun ramasseur
// nomme) ou un film sans humain au roster. Le distinguer de `sansArtefact` et d `echecs` est ce
// qui permet de lire une sortie de passe sans se demander si quelque chose s est mal passe.
type bilanPadTiersBackfill struct {
	ecrits, dejaEnBase, sansArtefact, sansPrise, echecs int
	totalLignes, totalPrises, sansReference             int
}

func runBackfillPadTiers(cfg *config.AppConfig, args []string) error {
	fs := flag.NewFlagSet("backfill-pad-tiers", flag.ExitOnError)
	o := padTiersOptions{}
	fs.StringVar(&o.titleSlug, "title", titlePkg.DefaultSlug, "slug du titre")
	fs.IntVar(&o.limit, "limit", 0, "borne le nombre de matchs projetes (0 = tous)")
	fs.BoolVar(&o.force, "force", false,
		"re-ecrire meme les matchs deja presents en base (OBLIGATOIRE apres un ajout a la reference des cartes)")
	fs.BoolVar(&o.dryRun, "dry-run", false, "projeter et imprimer une ligne par match sans rien ecrire")
	fs.StringVar(&o.match, "match", "", "ne traiter que ce match (identifiant complet du registre)")
	if err := fs.Parse(args); err != nil {
		return err
	}

	ref, reg, err := portePadTiers(cfg, o.titleSlug)
	if err != nil || ref == nil {
		return err
	}

	ctx := context.Background()
	pr := titlePkg.NewPathResolver(cfg.RepoRoot)
	sharedPath := pr.SharedDBPath(o.titleSlug)
	if _, err := os.Stat(sharedPath); err != nil {
		return fmt.Errorf("shared_matches introuvable (%s): %w", sharedPath, err)
	}
	handle, err := duckdb.OpenReadWrite(sharedPath)
	if err != nil {
		return fmt.Errorf("open shared RW (%s): %w (serveur arrete ?)", sharedPath, err)
	}
	defer handle.Close()
	db := handle.SQLDb()

	// La table et sa vue doivent exister AVANT de lire ou d ecrire — cette commande tourne
	// SERVEUR ARRETE, donc rien n a joue les migrations pour elle.
	if err := migrerSchemaPartage(db, o.titleSlug); err != nil {
		return err
	}

	candidats, err := candidatsPadTiers(ctx, db, o)
	if err != nil {
		return err
	}
	dejaEcrits := map[string]bool{}
	if !o.force {
		if dejaEcrits, err = matchsDejaProjetesPadTiers(ctx, db); err != nil {
			return err
		}
	}
	identites, err := replayartifacts.IdentitesDesMatchs(ctx, db, candidats)
	if err != nil {
		return err
	}
	debut := time.Now()
	b := projeterCorpusPadTiers(ctx, db, pr, o, ref, reg, identites, candidats, dejaEcrits)
	fmt.Printf("niveaux d armes : %d ecrits, %d deja en base, %d sans artefact, "+
		"%d sans prise nommable, %d echecs — %s\n",
		b.ecrits, b.dejaEnBase, b.sansArtefact, b.sansPrise, b.echecs,
		time.Since(debut).Round(time.Second))
	if b.totalLignes > 0 {
		fmt.Printf("totaux du corpus projete : %d lignes, %d prises, %d matchs SANS carte a la reference\n",
			b.totalLignes, b.totalPrises, b.sansReference)
	}
	return nil
}

// portePadTiers franchit les DEUX portes du titre. Rend (nil, nil, nil) quand le titre ne
// declare pas la grandeur : ce n est pas une erreur, c est une passe vide qui le dit a l ecran.
func portePadTiers(cfg *config.AppConfig, titleSlug string) (
	*replayartifacts.ReferenceEmplacements, *mappings.RegulationSet, error,
) {
	// LE GATE EST UNE CAPABILITY, JAMAIS UN SLUG (ratchet no_slug_comparison_test.go).
	caps, err := capabilitesDuTitre(cfg, titleSlug)
	if err != nil {
		return nil, nil, err
	}
	if !caps.Has(games.CapFilmWeaponTiers) {
		fmt.Printf("le titre %s ne declare pas %s : rien a projeter (passe vide)\n",
			titleSlug, string(games.CapFilmWeaponTiers))
		return nil, nil, nil
	}
	ref, err := replayartifacts.ChargerReferenceEmplacements(cfg.RepoRoot, titleSlug)
	if err != nil {
		return nil, nil, fmt.Errorf("reference des emplacements du titre %s: %w", titleSlug, err)
	}
	reg, err := replayartifacts.ReglesDepartsAleatoires(cfg.RepoRoot, titleSlug)
	if err != nil {
		return nil, nil, fmt.Errorf("regulation.toml du titre %s: %w", titleSlug, err)
	}
	if len(reg.RandomStartModePrefixes()) == 0 {
		// Ce n est PAS une panne — mais l operateur doit le savoir : aucun mode ne sera tenu
		// pour aleatoire, donc le niveau « base » sera produit partout.
		fmt.Printf("le titre %s ne declare aucun mode a departs aleatoires "+
			"([weapon_tiers].random_start_mode_prefixes) : le niveau « base » sera produit sur tous les modes\n",
			titleSlug)
	}
	return ref, reg, nil
}

// candidatsPadTiers : les matchs a examiner, dans un ordre stable — `--match` seul, sinon tout
// le registre (c est l artefact qui filtre ensuite : pas d artefact, pas de projection).
func candidatsPadTiers(ctx context.Context, db *sql.DB, o padTiersOptions) ([]string, error) {
	if o.match != "" {
		return []string{o.match}, nil
	}
	return matchsDuRegistre(ctx, db, 0)
}

// matchsDejaProjetesPadTiers : la cle de reprise, lue sur la VUE `_latest` (ADR 0026 — jamais
// la table brute, qui servirait les lignes d une passe deja supplantee).
func matchsDejaProjetesPadTiers(ctx context.Context, db *sql.DB) (map[string]bool, error) {
	rows, err := db.QueryContext(ctx, `SELECT DISTINCT match_id FROM match_pad_pickups_by_tier_latest`)
	if err != nil {
		return nil, fmt.Errorf("matchs deja projetes (niveaux d armes): %w", err)
	}
	defer func() { _ = rows.Close() }()
	out := map[string]bool{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("matchs deja projetes (niveaux d armes, scan): %w", err)
		}
		out[id] = true
	}
	return out, rows.Err()
}

// Etats d un match examine par la passe.
type etatPadTiersMatch int

const (
	niveauxAProjeter etatPadTiersMatch = iota
	niveauxSansArtefact
	niveauxSansPrise
	niveauxEchec
)

// projeterCorpusPadTiers lit (et ecrit, hors --dry-run) les matchs du lot, UN ARTEFACT A LA
// FOIS.
//
// struct ferait un type a usage unique de plus (meme forme que projeterCorpusFlagGrabsNet).
//
//nolint:revive // 9 parametres : la passe a besoin de tout son contexte, et les regrouper en
func projeterCorpusPadTiers(
	ctx context.Context, db *sql.DB, pr *titlePkg.PathResolver, o padTiersOptions,
	ref *replayartifacts.ReferenceEmplacements, reg *mappings.RegulationSet,
	identites map[string]replayartifacts.IdentiteMatchNiveaux,
	candidats []string, dejaEcrits map[string]bool,
) bilanPadTiersBackfill {
	b := bilanPadTiersBackfill{}
	p := persist.NewPadTiersPersister(db)
	projetes := 0
	for _, id := range candidats {
		if o.limit > 0 && projetes >= o.limit {
			break
		}
		if !o.force && dejaEcrits[id] {
			b.dejaEnBase++
			continue
		}
		ident := identites[id]
		batch, etat := lireUnArtefactPadTiers(
			pr.ReplayArtifactPath(o.titleSlug, id), id, ref, ident, reg.HasRandomStarts(ident.PairName))
		switch etat {
		case niveauxSansArtefact:
			b.sansArtefact++
			continue
		case niveauxSansPrise:
			b.sansPrise++
			continue
		case niveauxEchec:
			b.echecs++
			continue
		}
		projetes++
		prises := comptabiliserPasseNiveaux(&b, batch)
		if o.dryRun {
			// UNE LIGNE PAR MATCH, et c est le but de --dry-run : l operateur doit pouvoir
			// CONTROLER ce qui sera ecrit avant un --force, pas seulement un total.
			fmt.Printf("  %-40s lignes=%3d prises=%3d socles=%2d/%2d aleatoire=%-5v carte=%s\n",
				id, len(batch.Rows), prises, batch.PadsConfirmed, batch.PadsTotal,
				batch.RandomStarts, carteOuInconnue(ident.MapID))
			continue
		}
		if err := p.PersistPass(ctx, batch); err != nil {
			// Jamais avale : l echec d UN match n arrete pas la passe, mais il se voit.
			fmt.Printf("  ECHEC %s : %v\n", id, err)
			b.echecs++
			continue
		}
		b.ecrits++
	}
	return b
}

// carteOuInconnue rend le map_id, ou une mention explicite quand le registre ne le porte pas.
func carteOuInconnue(mapID string) string {
	if mapID == "" {
		return "(inconnue)"
	}
	return mapID
}

// comptabiliserPasseNiveaux additionne les totaux du bilan et rend les prises de CE match.
func comptabiliserPasseNiveaux(b *bilanPadTiersBackfill, batch persist.PadTiersBatch) int {
	prises := 0
	for _, r := range batch.Rows {
		prises += r.Pickups
	}
	b.totalLignes += len(batch.Rows)
	b.totalPrises += prises
	if batch.PadsTotal > 0 && batch.PadsConfirmed == 0 {
		b.sansReference++
	}
	return prises
}

// lireUnArtefactPadTiers lit UN artefact et en tire la passe a ecrire, ou dit pourquoi il ne le
// fait pas. L artefact entier est relache a la sortie — seule la passe survit.
//
// LA PROJECTION EST CELLE DU FIL DE L EAU, PAS UNE SECONDE :
// `replayartifacts.ProjeterNiveauxDArmes` est exportee exactement pour ca. Deux projections de
// la meme regle divergeraient au premier changement.
func lireUnArtefactPadTiers(
	path, matchID string, ref *replayartifacts.ReferenceEmplacements,
	ident replayartifacts.IdentiteMatchNiveaux, randomStarts bool,
) (persist.PadTiersBatch, etatPadTiersMatch) {
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return persist.PadTiersBatch{}, niveauxSansArtefact
		}
		fmt.Printf("  ECHEC %s : lecture artefact: %v\n", matchID, err)
		return persist.PadTiersBatch{}, niveauxEchec
	}
	var doc replay.ReplayDocument
	if err := json.Unmarshal(raw, &doc); err != nil {
		fmt.Printf("  ECHEC %s : parse artefact: %v\n", matchID, err)
		return persist.PadTiersBatch{}, niveauxEchec
	}
	batch := replayartifacts.ProjeterNiveauxDArmes(matchID, &doc, ref, ident, randomStarts)
	if batch.MatchID == "" {
		return persist.PadTiersBatch{}, niveauxSansPrise
	}
	return batch, niveauxAProjeter
}
