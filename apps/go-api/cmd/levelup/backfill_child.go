package main

// backfill_child.go — LE RUNNER PARENT/ENFANT des passes de cuisson de films.
//
// # POURQUOI CE FICHIER EXISTE : LE 2026-08-20, UNE PASSE A SATURE LA MACHINE
//
// `backfill-replay --only-existing` (29 films) a cuit QUATRE petits films (8 a 13 chunks,
// tous valides) puis s'est effondree sur le cinquieme : plus de six heures de spirale GC,
// puis `runtime.preemptM: duplicatehandle failed; errno=1450` — ERROR_NO_SYSTEM_RESOURCES,
// le runtime Go n'obtenait plus meme un handle de thread de Windows. Ce n'est pas UN film
// qui est trop gros : c'est le PROCESSUS UNIQUE qui empile les pics de tous les films et ne
// rend jamais rien a l'OS.
//
// # LA REGLE (doctrine machine D17) : UN FILM = UN PROCESSUS
//
// Le parent PLANIFIE (enumeration, tri par cout, sauts) et ne decode RIEN. Pour chaque film
// retenu il re-execute SON PROPRE BINAIRE sur ce seul film, sequentiellement, et attend sa
// mort. La RAM du film est rendue a l'OS par CONSTRUCTION — pas par la grace du GC — et un
// film-bombe ne peut plus emporter que LUI-MEME.
//
// # LE CODE DE SORTIE EST LE PROTOCOLE
//
// L'enfant ne remonte pas une erreur : il rend un CODE, que le parent traduit en categorie
// de recap. Tout code HORS protocole (crash, OOM systeme, tue par l'OS) est une MORT SUBITE
// — comptee, journalisee, et LA PASSE CONTINUE. C'est tout l'objet du lot : le film suivant
// ne doit rien devoir a la sante du precedent.
//
// # LE PARENT IMPOSE LA RACINE DU DEPOT A L'ENFANT
//
// `LEVELUP_REPO_ROOT` est REECRIT dans l'environnement de l'enfant avec la racine que le
// parent a resolue. Sans cela l'enfant re-detecterait la sienne depuis son executable (et
// sous `go run`, l'executable est dans un dossier temporaire) : deux processus de la meme
// passe pourraient ecrire dans deux arborescences differentes.

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Codes de sortie du protocole parent/enfant.
//
// Ils commencent a 10 pour ne JAMAIS collider avec ce que produisent le runtime et le
// paquet `flag` : 1 (erreur rendue par main), 2 (`flag.ExitOnError` et `fatal error` du
// runtime). Un code inconnu n'est donc jamais pris pour une issue metier — il est traite
// comme une mort subite, ce qui est le comportement sur.
const (
	codeEnfantOK             = 0
	codeEnfantHorsCatalogue  = 10 // carte sans bornes de dequantification — echec VOULU
	codeEnfantErreurDecodage = 11 // le film est la, le decodage a echoue
	codeEnfantPreparation    = 12 // l'enfant n'a meme pas pu commencer (builder, cache, ...)
	codeEnfantMemoire        = 13 // plafond memoire depasse — la sentinelle a tue l'enfant
)

// cleRepoRoot : la variable d'environnement qui fixe la racine du depot (cf. en-tete).
const cleRepoRoot = "LEVELUP_REPO_ROOT"

// issueEnfant : la categorie de recap d'un film, deduite du code de sortie de son enfant.
type issueEnfant int

const (
	issueOK issueEnfant = iota
	issueHorsCatalogue
	issueErreurDecodage
	issuePreparation
	issueMemoire
	issueMortSubite
)

// issuePourCode traduit un code de sortie en categorie.
//
// LE DEFAUT EST « MORT SUBITE », ET C'EST LA REGLE DE SURETE DU LOT : un enfant tue par
// l'OS, un `fatal error` du runtime ou un code qu'on n'a pas prevu ne doivent jamais passer
// pour un succes ni pour une erreur de decodage ordinaire.
func issuePourCode(code int) issueEnfant {
	switch code {
	case codeEnfantOK:
		return issueOK
	case codeEnfantHorsCatalogue:
		return issueHorsCatalogue
	case codeEnfantErreurDecodage:
		return issueErreurDecodage
	case codeEnfantPreparation:
		return issuePreparation
	case codeEnfantMemoire:
		return issueMemoire
	default:
		return issueMortSubite
	}
}

// marqueurPicMemoire : la ligne de protocole par laquelle l'enfant rend le pic memoire
// qu'il s'est mesure.
//
// L'ENFANT SE MESURE LUI-MEME parce que Windows ne le dit pas au parent : sur cette
// plateforme `os.ProcessState.SysUsage()` ne porte que des temps (creation, sortie, noyau,
// utilisateur), aucun compteur memoire. Le parent INTERCEPTE cette ligne et ne la relaie
// pas : l'operateur lit un journal, pas un protocole.
const marqueurPicMemoire = "__levelup_pic_octets__="

// emettrePicMemoire ecrit la ligne de protocole du pic memoire. Appelee par l'ENFANT.
func emettrePicMemoire(pic uint64) {
	fmt.Printf("%s%d\n", marqueurPicMemoire, pic)
}

// lirePicMemoire reconnait la ligne de protocole. Appelee par le PARENT.
func lirePicMemoire(ligne string) (uint64, bool) {
	reste, ok := strings.CutPrefix(strings.TrimSpace(ligne), marqueurPicMemoire)
	if !ok {
		return 0, false
	}
	v, err := strconv.ParseUint(reste, 10, 64)
	if err != nil {
		return 0, false
	}
	return v, true
}

// resultatEnfant : ce que le parent retient d'un enfant.
type resultatEnfant struct {
	code      int
	issue     issueEnfant
	duree     time.Duration
	picOctets uint64 // 0 = non rendu (l'enfant est mort avant de pouvoir se mesurer)
	err       error  // echec de LANCEMENT (pas un echec du film)
}

// runnerEnfant : le lanceur de processus enfants d'une passe.
type runnerEnfant struct {
	exe    string
	env    []string
	sortie io.Writer
}

// nouveauRunnerEnfant prepare le lanceur : le binaire courant, et un environnement dont la
// racine du depot est celle que le parent a resolue (cf. en-tete).
func nouveauRunnerEnfant(repoRoot string, sortie io.Writer) (*runnerEnfant, error) {
	exe, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("executable courant introuvable (re-execution impossible): %w", err)
	}
	return &runnerEnfant{exe: exe, env: envEnfant(repoRoot), sortie: sortie}, nil
}

// envEnfant recopie l'environnement en y IMPOSANT la racine du depot.
func envEnfant(repoRoot string) []string {
	brut := os.Environ()
	out := make([]string, 0, len(brut)+1)
	for _, kv := range brut {
		if strings.HasPrefix(strings.ToUpper(kv), cleRepoRoot+"=") {
			continue
		}
		out = append(out, kv)
	}
	return append(out, cleRepoRoot+"="+repoRoot)
}

// executer lance UN enfant, relaie sa sortie et rend son issue.
//
// LA SORTIE EST LUE JUSQU'A EOF AVANT `Wait` : c'est l'ordre impose par os/exec, et c'est
// aussi ce qui garantit que le journal de l'enfant est COMPLET dans celui du parent, y
// compris les dernieres lignes d'un enfant qui meurt.
func (r *runnerEnfant) executer(ctx context.Context, args []string) resultatEnfant {
	debut := time.Now()
	cmd := exec.CommandContext(ctx, r.exe, args...)
	cmd.Env = r.env

	lecture, ecriture, err := os.Pipe()
	if err != nil {
		return resultatEnfant{issue: issueMortSubite, err: fmt.Errorf("tube de sortie: %w", err)}
	}
	// UN SEUL TUBE POUR LES DEUX FLUX : l'enfant journalise en slog (stderr) et ecrit son
	// recap en fmt (stdout) ; les fusionner preserve leur entrelacement reel.
	cmd.Stdout, cmd.Stderr = ecriture, ecriture

	if err := cmd.Start(); err != nil {
		_, _ = lecture.Close(), ecriture.Close()
		return resultatEnfant{issue: issueMortSubite, err: fmt.Errorf("lancement: %w", err),
			duree: time.Since(debut)}
	}
	// Le parent relache SA copie du bout ecrivain : sans cela le scanner ne verrait jamais
	// l'EOF, meme apres la mort de l'enfant.
	_ = ecriture.Close()
	pic := r.relayer(lecture)
	_ = lecture.Close()

	res := resultatEnfant{picOctets: pic, duree: time.Since(debut)}
	res.code, res.err = codeDeSortie(cmd.Wait())
	res.issue = issuePourCode(res.code)
	if res.err != nil {
		res.issue = issueMortSubite
	}
	return res
}

// codeDeSortie extrait le code de sortie d'un `Wait`. Une erreur NON `ExitError` (le
// processus n'a pas pu etre attendu) est rendue telle quelle : ce n'est pas un code.
func codeDeSortie(werr error) (int, error) {
	if werr == nil {
		return codeEnfantOK, nil
	}
	var sortie *exec.ExitError
	if errors.As(werr, &sortie) {
		return sortie.ExitCode(), nil
	}
	return -1, werr
}

// relayer recopie la sortie de l'enfant vers celle du parent, en INTERCEPTANT les lignes de
// protocole. Rend le pic memoire si l'enfant a eu le temps de le rendre.
func (r *runnerEnfant) relayer(source io.Reader) uint64 {
	var pic uint64
	sc := bufio.NewScanner(source)
	// Les lignes de journal d'un decodage peuvent etre longues (chemins Windows + compteurs) :
	// le plafond par defaut de bufio (64 Kio) casserait le relais sur une ligne trop longue.
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for sc.Scan() {
		ligne := sc.Text()
		if v, ok := lirePicMemoire(ligne); ok {
			pic = v
			continue
		}
		fmt.Fprintln(r.sortie, ligne)
	}
	if err := sc.Err(); err != nil {
		slog.Error("relais de la sortie de l'enfant interrompu", "err", err)
	}
	return pic
}

// listeDrapeau : un drapeau REPETABLE (`--map-name A --map-name B`).
//
// Un drapeau unique a separateur serait plus court, mais les identites de carte sont des
// libelles libres : aucun separateur n'est sur. La repetition, elle, l'est.
type listeDrapeau []string

func (l *listeDrapeau) String() string { return strings.Join(*l, ", ") }

func (l *listeDrapeau) Set(v string) error {
	*l = append(*l, v)
	return nil
}
