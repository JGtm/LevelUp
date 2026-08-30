package filmproc

// runner.go — LE LANCEUR PARENT/ENFANT : un film, un processus, borne et efface.
//
// LE PARENT PLANIFIE ET NE DECODE RIEN. Pour chaque film retenu il re-execute SON PROPRE
// BINAIRE sur ce seul film, sequentiellement, et attend sa mort. La RAM du film est rendue a
// l'OS par CONSTRUCTION — pas par la grace du GC — et un film-bombe ne peut plus emporter que
// LUI-MEME.
//
// LE CODE DE SORTIE EST LE PROTOCOLE. L'enfant ne remonte pas une erreur : il rend un CODE, que
// le parent traduit en categorie. Tout code HORS protocole (crash, OOM systeme, tue par l'OS)
// est une MORT SUBITE — comptee, journalisee, et LA BOUCLE CONTINUE. C'est tout l'objet du
// patron : le film suivant ne doit rien devoir a la sante du precedent.
//
// LE PARENT IMPOSE LA RACINE DU DEPOT. `LEVELUP_REPO_ROOT` est REECRIT dans l'environnement de
// l'enfant avec la racine que le parent a resolue. Sans cela l'enfant re-detecterait la sienne
// depuis son executable — et sous `go run`, l'executable est dans un dossier temporaire : deux
// processus de la meme passe pourraient ecrire dans deux arborescences differentes.
//
// LA PRIORITE CPU BASSE EST UNE EXIGENCE, PAS UN REGLAGE (leçon du 2026-08-26). Ces boucles
// tournent sur le POSTE DE TRAVAIL de l'utilisateur pendant qu'il s'en sert. Un enfant a
// priorite normale qui decode un film BTB rend la machine inutilisable meme quand sa memoire
// est bornee. L'enfant nait donc en priorite BASSE (cf. lowPriority, par plateforme).

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Codes de sortie du protocole parent/enfant.
//
// ILS COMMENCENT A 10 pour ne JAMAIS collider avec ce que produisent le runtime et le paquet
// `flag` : 1 (erreur rendue par main), 2 (`flag.ExitOnError` et `fatal error` du runtime). Un
// code inconnu n'est donc jamais pris pour une issue metier — il est traite comme une mort
// subite, ce qui est le comportement sur.
const (
	CodeOK          = 0
	CodeSkipped     = 10 // l'enfant a refuse ce film pour une raison VOULUE (hors catalogue...)
	CodeFailed      = 11 // le film est la, le traitement a echoue
	CodePreparation = 12 // l'enfant n'a meme pas pu commencer (builder, cache, ...)
	CodeMemory      = 13 // plafond memoire depasse — la sentinelle a tranche
)

// EnvRepoRoot : la variable d'environnement qui fixe la racine du depot.
const EnvRepoRoot = "LEVELUP_REPO_ROOT"

// Issue : la categorie d'un film, deduite du code de sortie de son enfant.
type Issue int

const (
	IssueOK Issue = iota
	IssueSkipped
	IssueFailed
	IssuePreparation
	IssueMemory
	IssueSuddenDeath
)

// String rend le libelle de journal d'une issue.
func (i Issue) String() string {
	switch i {
	case IssueOK:
		return "ok"
	case IssueSkipped:
		return "ecarte"
	case IssueFailed:
		return "echec"
	case IssuePreparation:
		return "preparation"
	case IssueMemory:
		return "memoire"
	default:
		return "mort subite"
	}
}

// IssueForCode traduit un code de sortie en categorie.
//
// LE DEFAUT EST « MORT SUBITE », ET C'EST LA REGLE DE SURETE : un enfant tue par l'OS, un
// `fatal error` du runtime ou un code qu'on n'a pas prevu ne doivent jamais passer pour un
// succes ni pour un echec ordinaire.
func IssueForCode(code int) Issue {
	switch code {
	case CodeOK:
		return IssueOK
	case CodeSkipped:
		return IssueSkipped
	case CodeFailed:
		return IssueFailed
	case CodePreparation:
		return IssuePreparation
	case CodeMemory:
		return IssueMemory
	default:
		return IssueSuddenDeath
	}
}

// peakMarker : la ligne de protocole par laquelle l'enfant rend le pic memoire qu'il s'est
// mesure.
//
// L'ENFANT SE MESURE LUI-MEME parce que Windows ne le dit pas au parent : sur cette plateforme
// `os.ProcessState.SysUsage()` ne porte que des temps, aucun compteur memoire. Le parent
// INTERCEPTE cette ligne et ne la relaie pas : l'operateur lit un journal, pas un protocole.
const peakMarker = "__levelup_pic_octets__="

// EmitPeak ecrit la ligne de protocole du pic memoire. Appelee par l'ENFANT.
func EmitPeak(peak uint64) { fmt.Printf("%s%d\n", peakMarker, peak) }

// parsePeak reconnait la ligne de protocole. Appelee par le PARENT.
func parsePeak(line string) (uint64, bool) {
	rest, ok := strings.CutPrefix(strings.TrimSpace(line), peakMarker)
	if !ok {
		return 0, false
	}
	v, err := strconv.ParseUint(rest, 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// Result : ce que le parent retient d'un enfant.
type Result struct {
	Code  int
	Issue Issue
	Dur   time.Duration
	// Peak vaut 0 quand l'enfant est mort avant d'avoir pu se mesurer.
	Peak uint64
	// Err est un echec de LANCEMENT, pas un echec du film.
	Err error
}

// Runner : le lanceur de processus enfants d'une boucle sur des films.
type Runner struct {
	exe string
	env []string
	out io.Writer
}

// NewRunner prepare le lanceur : le binaire courant, et un environnement dont la racine du
// depot est celle que le parent a resolue.
func NewRunner(repoRoot string, out io.Writer) (*Runner, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("executable courant introuvable (re-execution impossible): %w", err)
	}
	if out == nil {
		out = os.Stdout
	}
	return &Runner{exe: exe, env: childEnv(repoRoot), out: out}, nil
}

// childEnv recopie l'environnement en y IMPOSANT la racine du depot.
func childEnv(repoRoot string) []string {
	raw := os.Environ()
	out := make([]string, 0, len(raw)+1)
	for _, kv := range raw {
		if strings.HasPrefix(strings.ToUpper(kv), EnvRepoRoot+"=") {
			continue
		}
		out = append(out, kv)
	}
	return append(out, EnvRepoRoot+"="+repoRoot)
}

// Run lance UN enfant, relaie sa sortie et rend son issue.
//
// LA SORTIE EST LUE JUSQU'A EOF AVANT `Wait` : c'est l'ordre impose par os/exec, et c'est aussi
// ce qui garantit que le journal de l'enfant est COMPLET dans celui du parent, y compris les
// dernieres lignes d'un enfant qui meurt.
func (r *Runner) Run(ctx context.Context, args []string) Result {
	start := time.Now()
	cmd := exec.CommandContext(ctx, r.exe, args...)
	cmd.Env = r.env
	lowPriority(cmd) // la machine doit rester utilisable pendant la boucle

	read, write, err := os.Pipe()
	if err != nil {
		return Result{Issue: IssueSuddenDeath, Err: fmt.Errorf("tube de sortie: %w", err)}
	}
	// UN SEUL TUBE POUR LES DEUX FLUX : l'enfant journalise en slog (stderr) et ecrit son recap
	// en fmt (stdout) ; les fusionner preserve leur entrelacement reel.
	cmd.Stdout, cmd.Stderr = write, write

	if err := cmd.Start(); err != nil {
		_, _ = read.Close(), write.Close()
		return Result{Issue: IssueSuddenDeath, Err: fmt.Errorf("lancement: %w", err),
			Dur: time.Since(start)}
	}
	// Le parent relache SA copie du bout ecrivain : sans cela le scanner ne verrait jamais
	// l'EOF, meme apres la mort de l'enfant.
	_ = write.Close()
	peak := r.relay(read)
	_ = read.Close()

	res := Result{Peak: peak, Dur: time.Since(start)}
	res.Code, res.Err = exitCode(cmd.Wait())
	res.Issue = IssueForCode(res.Code)
	if res.Err != nil {
		res.Issue = IssueSuddenDeath
	}
	return res
}

// exitCode extrait le code de sortie d'un `Wait`. Une erreur NON `ExitError` (le processus n'a
// pas pu etre attendu) est rendue telle quelle : ce n'est pas un code.
func exitCode(werr error) (int, error) {
	if werr == nil {
		return CodeOK, nil
	}
	var ee *exec.ExitError
	if errors.As(werr, &ee) {
		return ee.ExitCode(), nil
	}
	return -1, werr
}

// relay recopie la sortie de l'enfant vers celle du parent, en INTERCEPTANT les lignes de
// protocole. Rend le pic memoire si l'enfant a eu le temps de le rendre.
func (r *Runner) relay(src io.Reader) uint64 {
	var peak uint64
	sc := bufio.NewScanner(src)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		line := sc.Text()
		if v, ok := parsePeak(line); ok {
			peak = v
			continue // protocole : jamais relaye
		}
		// L'ERREUR D'ECRITURE EST ECARTEE, ET C'EST LE SEUL ENDROIT OU C'EST LEGITIME : cette
		// sortie EST le canal de rapport du parent. S'il est rompu, un log de l'incident partirait
		// vers le meme tube casse. Le relais continue de vider le tube de l'enfant — s'arreter ici
		// bloquerait l'enfant sur son ecriture suivante, ce qui serait pire que la ligne perdue.
		_, _ = fmt.Fprintln(r.out, line)
	}
	return peak
}
