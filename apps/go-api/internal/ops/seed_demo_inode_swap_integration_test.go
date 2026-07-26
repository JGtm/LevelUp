//go:build integration

// Package ops — seed_demo_inode_swap_integration_test.go : publication de la metadata
// démo par ÉCHANGE D'INODE, rejouée avec un VRAI second processus détenteur.
//
// Scénario de production (fenêtre de déploiement du 2026-07-26) : le seed republie
// data/demo/warehouse/metadata.duckdb pendant que l'ANCIEN conteneur `levelup-demo`
// tient encore le fichier ouvert avec son verrou DuckDB, puis la phase Prestige
// (insertDemoMilestonesEarned) fait un ATTACH READ_ONLY sur le chemin republié. Ce
// sont DEUX PROCESSUS : le détenteur (conteneur démo) et le lecteur (seed).
//
// POURQUOI UN SOUS-PROCESSUS, ET NON UN HANDLE LOCAL — CHEMIN OU INODE ?
// La première version de ce test simulait le détenteur avec un `sql.Open` gardé
// ouvert DANS le processus de test : elle échouait en CI (run 30197117063) sur
// « Binder Error: Unique file handle conflict: Cannot attach "demometa" - the
// database file ".../metadata.duckdb" is already attached by database "metadata" »,
// c'est-à-dire pour une raison qui NE PEUT PAS se produire en prod. Mesuré le
// 2026-07-26 (DuckDB 1.5.4 via duckdb-go/v2, sonde jetable + log CI) :
//
//   - le dédoublonnage porte sur le CHEMIN CANONIQUE, à l'échelle du processus : deux
//     instances DuckDB distinctes du même processus refusent d'ouvrir/attacher le même
//     chemin, et le même chemin orthographié autrement (segment « ./ », nom court 8.3)
//     produit le même refus en citant le chemin normalisé — DuckDB normalise avant de
//     comparer ;
//   - il ne porte PAS sur l'identité du fichier : un lien dur (même inode, autre
//     chemin) ne déclenche pas ce refus — c'est un autre mécanisme qui répond, avec
//     un autre message (mesuré sous Windows : le partage de fichier de l'OS) ;
//   - et le refus observé en CI est arrivé APRÈS un copyMetadataFile réussi, donc sur
//     un inode NEUF publié sous l'ancien chemin.
//
// Conclusion : simuler le détenteur dans le processus de test est structurellement
// impossible — l'échange d'inode n'y change rien, le chemin suffit à déclencher le
// refus. Le détenteur doit donc être un vrai second processus (idiome os/exec de la
// stdlib : le binaire de test se ré-exécute en processus assistant).
//
// NB pattern : les deux autres ré-exécutions de binaire de test du dépôt
// (media_kill_brutal_test.go, platform/duckdb/pool_wal_recovery_subprocess_test.go)
// sont des enfants « écris puis meurs » lus en une fois par CombinedOutput. Celui-ci
// est un enfant LONGUE VIE avec poignée de main (READY / ordre de fin / réponse) : il
// n'y a pas de noyau commun factorisable au-delà de l'appel exec.Command(os.Args[0]).
package ops

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	_ "github.com/duckdb/duckdb-go/v2"
)

// holdDBPathEnv porte le chemin de la base que le processus assistant doit tenir
// ouverte. Absente = TestHelperProcessHoldDuckDB est inerte (exécution normale du
// paquet).
const holdDBPathEnv = "LEVELUP_TEST_HOLD_DB"

// Protocole ligne-à-ligne parent ↔ assistant, sur le stdout de l'enfant.
const (
	holdReadyMarker = "READY"
	holdOldPrefix   = "OLD "
)

// holdTimeout borne la vie de l'assistant : un enfant qui ne répond pas ne doit
// JAMAIS bloquer la CI (le contexte le tue, ses tuyaux se ferment, les lectures du
// parent rendent la main).
const holdTimeout = 30 * time.Second

// TestCopyMetadataFile_AttachAfterSwapWhileHeld rejoue la fenêtre de déploiement du
// 2026-07-26 : le seed republie la metadata démo pendant qu'un AUTRE PROCESSUS tient
// encore le fichier ouvert avec son verrou DuckDB. Après le remplacement d'inode,
// l'ATTACH READ_ONLY que fait la phase Prestige (insertDemoMilestonesEarned) doit
// réussir, et l'ancien détenteur doit continuer à lire sa base (son inode, délié du
// chemin, reste intact pour lui).
//
// MORDANT — ce que ce test attrape. La forme fautive (celle d'avant cfc341b4f)
// écrivait EN PLACE, sur le MÊME inode : `os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|
// os.O_TRUNC, 0o644)` suivi de `io.Copy(out, in)`. Ce qu'elle produit :
//
//   - Linux (cible du gate CI) : l'ouverture O_TRUNC réussit — le verrou DuckDB est
//     consultatif, il n'empêche pas write(2) — et tronque l'inode que l'assistant
//     tient, donc la démo sert une base invalide pendant toute la fenêtre. L'ATTACH
//     READ_ONLY du parent porte ensuite sur cet inode TOUJOURS VERROUILLÉ : DuckDB
//     refuse avec « Could not set lock on file … Conflicting lock is held in … (PID
//     n) » — exactement le symptôme prod (milestone_earned démo vide, avalé par le
//     best-effort). Le test échoue sur l'assertion ATTACH. C'est l'assertion
//     porteuse ; la relecture OLD de l'assistant (fichier tronqué puis réécrit sous
//     lui) échouerait probablement aussi, mais son gestionnaire de tampons peut
//     servir la valeur en cache : on ne compte pas sur elle.
//   - Windows (mesuré 2026-07-26) : l'ouverture O_TRUNC échoue immédiatement
//     (« ce fichier est utilisé par un autre processus » — DuckDB n'accorde pas le
//     partage en écriture), donc copyMetadataFile rend « create dst: … » et le test
//     échoue dès sa première assertion.
//
// Le mordant Linux est un raisonnement, pas une exécution : la preuve d'exécution
// vient du gate CI (le test ne tourne pas sous Windows, cf. skip ci-dessous). Le
// mordant Windows, lui, a été mesuré en levant temporairement le skip et en
// restaurant la forme O_TRUNC : échec sur « create dst », comme décrit.
func TestCopyMetadataFile_AttachAfterSwapWhileHeld(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Scénario Linux-only ASSUMÉ : le déploiement visé (conteneur démo) et le
		// gate de référence (CI) le sont, et les sémantiques de partage de fichier
		// de Windows ne sont PAS celles du scénario prod. Mesuré le 2026-07-26
		// (Windows 11, Go 1.26, DuckDB 1.5.4) en levant temporairement ce skip : le
		// scénario passe, mais par un autre chemin — os.Remove d'une base tenue
		// réussit (suppression à sémantique POSIX), tandis qu'un autre processus ne
		// peut même pas l'OUVRIR EN LECTURE (partage refusé), là où Linux laisse
		// lire et fait arbitrer le verrou consultatif de DuckDB. Le test resterait
		// donc vert ici en vérifiant autre chose, et rougirait au moindre
		// changement de ces drapeaux de partage (Go ou DuckDB) sans qu'aucun bug
		// n'ait été introduit. Recette pour rejouer localement : remplacer cette
		// garde par `if false`.
		t.Skip("scénario Linux-only : sémantiques de partage de fichier Windows non représentatives du conteneur démo")
	}
	ctx := context.Background()
	dir := t.TempDir()
	src := filepath.Join(dir, "src_meta.duckdb")
	dst := filepath.Join(dir, "metadata.duckdb")

	// Metadata source (nouvelle génération). Le processus de test n'ouvre JAMAIS dst :
	// il ne peut pas être à la fois le détenteur et le lecteur (cf. en-tête).
	srcDB, err := sql.Open("duckdb", src)
	if err != nil {
		t.Fatal(err)
	}
	mustExec(t, srcDB, `CREATE TABLE career_ranks (rank_id INTEGER, rank_name VARCHAR)`)
	mustExec(t, srcDB, `INSERT INTO career_ranks VALUES (1, 'Recruit'), (2, 'Iron')`)
	if err := srcDB.Close(); err != nil {
		t.Fatal(err)
	}

	// « Ancien conteneur démo » : second processus qui tient dst ouverte (verrou
	// DuckDB posé, table écrite, CHECKPOINT fait) pendant toute la publication.
	holder := startDuckDBHolder(t, dst)

	if err := copyMetadataFile(src, dst); err != nil {
		t.Fatalf("copyMetadataFile pendant que la démo tient le fichier: %v", err)
	}

	// Lecteur suivant du seed (phase Prestige) : ATTACH READ_ONLY sur le chemin.
	probe, err := sql.Open("duckdb", filepath.Join(dir, "probe.duckdb"))
	if err != nil {
		t.Fatal(err)
	}
	defer probe.Close()
	if _, err := probe.ExecContext(ctx, fmt.Sprintf("ATTACH '%s' AS demometa (READ_ONLY)", dst)); err != nil {
		t.Fatalf("ATTACH READ_ONLY après remplacement d'inode: %v", err)
	}
	defer func() { _, _ = probe.ExecContext(ctx, "DETACH demometa") }()
	var n int
	if err := probe.QueryRowContext(ctx, `SELECT COUNT(*) FROM demometa.career_ranks`).Scan(&n); err != nil {
		t.Fatalf("lecture metadata publiée: %v", err)
	}
	if n != 2 {
		t.Errorf("career_ranks publiées = %d, want 2", n)
	}

	// L'ancien conteneur n'a pas vu sa base tronquée sous lui : son inode est intact.
	// C'est l'assistant qui relit — il est le seul à détenir ce handle.
	if held := holder.stopAndReadOldGeneration(t); held != 1 {
		t.Errorf("ancienne_generation relue par le détenteur = %d, want 1", held)
	}
}

// duckdbHolder : poignée du parent sur le processus assistant qui tient la base.
type duckdbHolder struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout *bufio.Reader
	seen   []string // lignes déjà consommées, pour les diagnostics d'échec
}

// startDuckDBHolder lance le binaire de test comme processus assistant sur path et
// rend la main quand celui-ci a signalé READY (base ouverte, écrite, checkpointée).
// L'enfant est tué et récolté par t.Cleanup sur TOUS les chemins de sortie.
func startDuckDBHolder(t *testing.T, path string) *duckdbHolder {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), holdTimeout)
	cmd := exec.CommandContext(ctx, os.Args[0],
		"-test.run=^TestHelperProcessHoldDuckDB$",
		"-test.count=1",
		"-test.timeout="+holdTimeout.String(),
	)
	cmd.Env = append(os.Environ(), holdDBPathEnv+"="+path)
	// Contexte expiré : après le kill, ne pas attendre indéfiniment la fermeture des
	// tuyaux (un enfant bloqué en écriture suffirait à figer Wait).
	cmd.WaitDelay = 5 * time.Second
	// Les échecs de test de l'enfant passent par son stdout (harness testing) et sont
	// donc capturés par le protocole ; seuls ses panics sortent sur stderr.
	cmd.Stderr = os.Stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		t.Fatalf("tuyau stdin assistant: %v", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		t.Fatalf("tuyau stdout assistant: %v", err)
	}
	if err := cmd.Start(); err != nil {
		cancel()
		t.Fatalf("lancement de l'assistant %s: %v", os.Args[0], err)
	}
	t.Cleanup(func() {
		cancel() // tue l'enfant s'il vit encore (échec en cours de scénario)
		_ = stdin.Close()
		_ = cmd.Wait() // erreur ignorée : Wait a pu être appelé par le test
	})

	h := &duckdbHolder{cmd: cmd, stdin: stdin, stdout: bufio.NewReader(stdout)}
	if _, ok := h.waitFor(holdReadyMarker); !ok {
		t.Fatalf("l'assistant n'a pas signalé %s (base non tenue) — sortie:\n%s",
			holdReadyMarker, strings.Join(h.seen, "\n"))
	}
	return h
}

// waitFor consomme le stdout de l'assistant jusqu'à la ligne préfixée par prefix.
// Rend (ligne, false) sur fin de flux — l'enfant est mort ou a été tué par le
// contexte, ce qui garantit que cette lecture rend toujours la main.
func (h *duckdbHolder) waitFor(prefix string) (string, bool) {
	for {
		raw, err := h.stdout.ReadString('\n')
		line := strings.TrimRight(raw, "\r\n")
		if line != "" {
			h.seen = append(h.seen, line)
		}
		if strings.HasPrefix(line, prefix) {
			return line, true
		}
		if err != nil {
			return line, false
		}
	}
}

// stopAndReadOldGeneration donne l'ordre de fin à l'assistant (fermeture de son
// stdin), lit le nombre de lignes qu'il relit dans SA base, puis exige une sortie
// propre. Variante retenue plutôt qu'un simple « code de sortie 0 » : un code de
// sortie ne distingue pas « relecture OK » de « base vide » ni de « harness sorti
// avant la relecture », et le parent doit pouvoir afficher le compte fautif.
func (h *duckdbHolder) stopAndReadOldGeneration(t *testing.T) int {
	t.Helper()
	if err := h.stdin.Close(); err != nil {
		t.Fatalf("fermeture du stdin de l'assistant: %v", err)
	}
	line, ok := h.waitFor(holdOldPrefix)
	if !ok {
		t.Fatalf("l'assistant n'a pas relu sa table (%s absent) — sortie:\n%s",
			strings.TrimSpace(holdOldPrefix), strings.Join(h.seen, "\n"))
	}
	n, err := strconv.Atoi(strings.TrimSpace(strings.TrimPrefix(line, holdOldPrefix)))
	if err != nil {
		t.Fatalf("réponse illisible de l'assistant %q: %v", line, err)
	}
	if err := h.cmd.Wait(); err != nil {
		t.Fatalf("sortie de l'assistant: %v — sortie:\n%s", err, strings.Join(h.seen, "\n"))
	}
	return n
}

// TestHelperProcessHoldDuckDB n'est un test que par commodité : c'est le PROCESSUS
// ASSISTANT de TestCopyMetadataFile_AttachAfterSwapWhileHeld (idiome os/exec de la
// stdlib), inerte hors de ce scénario. Il joue l'ancien conteneur démo — il tient une
// base DuckDB ouverte — et parle un protocole ligne-à-ligne avec son parent :
//
//	stdout READY   : base ouverte, écrite, checkpointée → le parent peut republier ;
//	stdin  EOF     : ordre de fin (le parent a fini ses assertions) ;
//	stdout OLD <n> : lignes relues dans SA base après la republication.
func TestHelperProcessHoldDuckDB(t *testing.T) {
	path := os.Getenv(holdDBPathEnv)
	if path == "" {
		t.Skip("processus assistant du scénario détenteur — inerte sans " + holdDBPathEnv)
	}
	ctx := context.Background()
	db, err := sql.Open("duckdb", path)
	if err != nil {
		t.Fatalf("ASSISTANT open %s: %v", path, err)
	}
	defer db.Close()
	// Connexion explicitement retenue : le verrou DuckDB ne tient que tant qu'une
	// connexion vit, et un pool sql.DB est libre de fermer ses connexions inactives.
	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatalf("ASSISTANT conn: %v", err)
	}
	defer conn.Close()
	for _, q := range []string{
		`CREATE TABLE ancienne_generation (n INTEGER)`,
		`INSERT INTO ancienne_generation VALUES (42)`,
		// CHECKPOINT : pas de WAL en suspens qui brouillerait l'assertion.
		`CHECKPOINT`,
	} {
		if _, err := conn.ExecContext(ctx, q); err != nil {
			t.Fatalf("ASSISTANT %q: %v", q, err)
		}
	}
	holderSay(t, holdReadyMarker)

	// Attente de l'ordre de fin : le parent republie la metadata et fait ses
	// assertions pendant que ce processus tient toujours le fichier.
	if _, err := io.Copy(io.Discard, os.Stdin); err != nil {
		t.Fatalf("ASSISTANT lecture de stdin: %v", err)
	}

	// Relecture de SA base : ancien inode, délié du chemin par la republication.
	var n int
	if err := conn.QueryRowContext(ctx, `SELECT COUNT(*) FROM ancienne_generation`).Scan(&n); err != nil {
		t.Fatalf("ASSISTANT relecture de l'ancienne génération: %v", err)
	}
	holderSay(t, holdOldPrefix+strconv.Itoa(n))
}

// holderSay écrit une ligne de protocole sur le stdout de l'assistant. os.Stdout et
// non t.Log : la sortie du harness testing est tamponnée jusqu'à la fin du test,
// alors que le parent a besoin de READY AVANT que l'enfant ne termine.
func holderSay(t *testing.T, line string) {
	t.Helper()
	if _, err := fmt.Fprintln(os.Stdout, line); err != nil {
		t.Fatalf("ASSISTANT écriture de %q: %v", line, err)
	}
	_ = os.Stdout.Sync() // no-op sur un tuyau ; documente l'intention
}
