package main

// facts.go — LES FAITS DU MATCH, PAR LA MEME PORTE QUE LE BALAYAGE DU PARC.
//
// # POURQUOI UN SOUS-PROCESSUS, ET PAS UN IMPORT DIRECT
//
// `levelup replay-facts-export` ouvre la base partagee (`OpenReadForQuery`, CGO/DuckDB) et
// resout le registre des cartes candidates (`registreParShort`, non exporte : package `main`
// de `cmd/levelup`). Une troisieme copie de cette resolution DIVERGERAIT au premier champ
// ajoute (regle du depot, "une seule ecriture des memes requetes") ; ce fichier invoque donc
// L'OUTIL CANONIQUE tel quel, exactement comme la methode du balayage
// (.ai/V7.5/V2/BALAYAGE_PARC_2026-09-06.md §9). Ce gate reste lui-meme compilable SANS CGO
// (`go build ./cmd/replay-corpus-gate` marche a vide) : seule CETTE etape, en sous-processus,
// exige CGO/gcc — comme `go-api-test-gamefiles` exige le jeu pour SA seule etape.
//
// # LECTURE SEULE SUR LE PARC
//
// `LEVELUP_REPO_ROOT=parcRoot` : l'export lit `parcRoot/data/titles/{slug}/warehouse/...` en
// RO (`OpenReadForQuery`, jamais `OpenReadOnly` force) et n'ecrit que les `<short8>.facts.json`
// demandes, dans `factsDir` (une racine temporaire, jamais le parc).
//
// # EXPORT PAR TEMOIN, PAS EN UN SEUL LOT (CORPUS-R1 C4, 2026-09-07)
//
// Une invocation UNIQUE portant tous les ids faisait echouer TOUT le sous-processus (exit 2)
// au premier id inconnu du registre — AVANT toute cuisson, y compris pour les temoins valides
// du meme manifeste. Chaque id est desormais exporte par une invocation SEPAREE : un id
// inconnu ne fait echouer QUE lui (`slog.Warn`, jamais fatal), son `<short8>.facts.json` reste
// simplement absent — `orchestrate.go` le detecte alors comme un temoin ABSENT (jamais une
// ERREUR), et `verifierCouverture` (report.go, C3) en fait un plancher de couverture explicite
// plutot qu'un silence.

// # LA BASE TENUE EN ECRITURE N'EST PAS UNE ABSENCE (D2 (cloture M1), 2026-09-17)
//
// Au gate du lot 2.1, `levelup replay-facts-export` a echoue sur DEUX temoins (`c75f33b8`,
// `111fa685`) avec « open shared RO … utilise par un autre processus (serveur en ecriture ?
// reessayer) » — le serveur local, relance 16 min plus tot, tenait le fichier a cet instant
// precis ; les douze autres exports, quelques secondes avant ou apres, ont reussi. Le gate a
// fini « 2/14 absent(s) » sur un alea de quelques secondes, et le pilote a du le rejouer avec
// un manifeste REDUIT ecrit a la main.
//
// Deux reponses, ici et dans manifest.go :
//
//	LE REESSAI     un echec qui porte le marqueur de base tenue est RETENTE, 3 fois, 2 s
//	               d'ecart. Un echec PERMANENT (id inconnu du registre, faits vides) n'est
//	               JAMAIS retente : attendre six secondes pour re-poser une question dont la
//	               reponse ne peut pas changer ne fait que rallonger un gate de 25 min.
//	LE REJEU CIBLE `--temoins a,b` rejoue les seuls temoins nommes (manifest.go), sans
//	               fabriquer un manifeste reduit dont personne ne sait plus, ensuite, qu'il
//	               n'etait pas le corpus.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"strings"
	"time"
)

// marqueurBaseTenue : LE marqueur d'un echec TRANSITOIRE de l'export, MIROIR DU TEXTE DE
// `duckdb.ErrBaseTenueEnEcriture` (lot 2.10.4).
//
// Depuis ce lot, l'indication n'est plus collee a la main par `cmd/levelup` : elle vient de la
// SENTINELLE que rend `internal/platform/duckdb` quand une ouverture echoue sur un verrou d'un
// autre processus — donc seulement quand la base est VRAIMENT tenue, jamais sur un fichier
// absent. En processus, un appelant ecrit `errors.Is(err, duckdb.ErrBaseTenueEnEcriture)`.
//
// CE GATE N'EST PAS DANS CE PROCESSUS : il EXECUTE `levelup` et lit son `stderr`, ou
// `errors.Is` n'a aucun sens — il ne lui reste que le texte. Il le MIROITE ici plutot que
// d'importer `internal/platform/duckdb`, qui embarquerait le pilote DuckDB (CGO) dans un
// binaire de gate qui n'ouvre aucune base. L'egalite des deux est tenue par
// `archlint.TestMarqueurDuGateEgaleLaSentinelleBaseTenue`, qui lit la definition de la
// sentinelle : sans ce garde-rail, un reformulage desarmerait le reessai en silence, et le
// gate reperdrait des temoins sur un alea de quelques secondes.
const marqueurBaseTenue = "serveur en ecriture"

// reessaisExport / delaiEntreReessais : le reessai BORNE — trois tentatives, deux secondes
// d'ecart (D2 (cloture M1)). Borne, parce qu'un serveur qu'on vient de relancer tient la base
// quelques secondes, pas quelques minutes : au-dela, le temoin est ABSENT et le dire vite vaut
// mieux que retarder un gate de 25 min.
const (
	reessaisExport     = 3
	delaiEntreReessais = 2 * time.Second
)

// dormeur attend, ou rend l'erreur d'annulation du contexte. Injecte pour que le test du
// reessai ne dorme pas six secondes.
type dormeur func(ctx context.Context, d time.Duration) error

// dormirContexte est LE dormeur reel : il rend la main tout de suite si le gate est interrompu
// (Ctrl-C sur une passe de 25 min) au lieu de laisser courir l'attente.
func dormirContexte(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

// estBaseTenue dit si l'erreur d'export est celle d'une base momentanement tenue en ecriture —
// le SEUL cas qui merite un reessai.
func estBaseTenue(err error) bool {
	return err != nil && strings.Contains(err.Error(), marqueurBaseTenue)
}

// exporterAvecReessai appelle `tenter` jusqu'a `reessaisExport` fois, et n'attend QU'ENTRE deux
// tentatives d'un echec transitoire. Un echec permanent rend tout de suite ; un contexte annule
// rend la derniere erreur sans attendre de plus.
func exporterAvecReessai(ctx context.Context, tenter func() error, dormir dormeur) error {
	var dernier error
	for essai := 1; essai <= reessaisExport; essai++ {
		dernier = tenter()
		if dernier == nil {
			return nil
		}
		if !estBaseTenue(dernier) {
			return dernier
		}
		if essai == reessaisExport {
			break
		}
		slog.Warn("replay-corpus-gate: base partagee tenue en ecriture — nouvel essai de l'export",
			"essai", essai, "essais", reessaisExport, "attente", delaiEntreReessais, "err", dernier)
		if err := dormir(ctx, delaiEntreReessais); err != nil {
			return errors.Join(dernier, err)
		}
	}
	return dernier
}

// exporterUnFait exporte les faits d'UN match — extrait en type nommé pour permettre un test
// unitaire de la boucle de continuation (exportFactsAvec) SANS vrai sous-processus CGO.
type exporterUnFait func(id string) error

// exportParams regroupe les chemins fixes d'un export — un struct plutot qu'une signature a
// plus de 5 parametres une fois `ctx` ajoute (CLAUDE.md n°5).
type exportParams struct {
	GoAPIDir  string // depuis ou lancer `go run` (celui qui porte cmd/levelup)
	ParcRoot  string
	TitleSlug string
	FactsDir  string
}

// exportFacts ecrit un `<short8>.facts.json` PAR id dans `p.FactsDir`, en lecture seule sur la
// base partagee de `p.ParcRoot`.
func exportFacts(ctx context.Context, p exportParams, ids []string) error {
	if len(ids) == 0 {
		return nil
	}
	if err := os.MkdirAll(p.FactsDir, 0o750); err != nil {
		return fmt.Errorf("dossier des faits : %w", err)
	}
	exportFactsAvec(func(id string) error {
		return exporterAvecReessai(ctx, func() error { return exportUnFait(ctx, p, id) }, dormirContexte)
	}, ids)
	return nil
}

// exportFactsAvec applique `exporter` a chaque id, EN CONTINUANT apres un echec — le coeur du
// correctif C4, teste independamment du sous-processus reel (facts_test.go).
func exportFactsAvec(exporter exporterUnFait, ids []string) {
	for _, id := range ids {
		if err := exporter(id); err != nil {
			slog.Warn("replay-corpus-gate: export des faits impossible pour ce temoin — ignore, "+
				"les autres temoins du manifeste continuent",
				"temoin", id, "err", err)
		}
	}
}

// exportUnFait invoque `levelup replay-facts-export` pour UN SEUL id.
func exportUnFait(ctx context.Context, p exportParams, id string) error {
	cmd := exec.CommandContext(ctx, "go", "run", "./cmd/levelup", "replay-facts-export", //nolint:gosec // id du manifeste + chemins internes
		"--out", p.FactsDir, "--title", p.TitleSlug, id)
	cmd.Dir = p.GoAPIDir
	cmd.Env = append(os.Environ(),
		"LEVELUP_REPO_ROOT="+p.ParcRoot,
		"CGO_ENABLED=1",
	)
	var stderr, stdout bytes.Buffer
	cmd.Stderr, cmd.Stdout = &stderr, &stdout
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("export des faits (levelup replay-facts-export %s) : %w\nstdout:\n%s\nstderr:\n%s",
			id, err, stdout.String(), stderr.String())
	}
	return nil
}
