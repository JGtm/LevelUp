package filmproc

// runner_child_test.go — LE LANCEUR EXERCE SUR UN VRAI PROCESSUS.
//
// POURQUOI CE FICHIER EXISTE (PLAN_CUISSON_PERF item 5.4, 2026-09-03). Le lanceur etait teste
// PAR MORCEAUX : la traduction d'un code en issue, la reconnaissance du marqueur de pic, le
// relais ligne a ligne. Aucun cas ne prouvait que les trois se rejoignent SUR UN PROCESSUS —
// c'est-a-dire que le code de sortie d'un enfant reel arrive bien jusqu'a `Result.Issue`, que
// son pic traverse le tube, et que son journal ressort chez le parent. C'est precisement le
// contrat sur lequel la passe `backfill-replay` s'est alignee en abandonnant sa propre copie du
// motif : il devait etre verifie de bout en bout avant qu'un second appelant en depende.
//
// LE BINAIRE DE TEST EST SON PROPRE ENFANT, exactement comme le binaire de production l'est
// (`filmproc.NewRunner` re-execute `os.Executable()`). [TestAideEnfantFilmproc] n'est pas un
// test : c'est le CORPS de l'enfant, inerte tant que le drapeau ci-dessous n'est pas passe.

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"testing"
)

// codeAideEnfant : le code de sortie que l'enfant doit rendre. Negatif = ce processus n'est pas
// un enfant, et [TestAideEnfantFilmproc] ne fait rien.
var codeAideEnfant = flag.Int("filmproc-aide-code", -1,
	"usage interne des tests : code de sortie de l'enfant re-execute")

// picAideEnfant : le pic que l'enfant annonce par la ligne de protocole.
const picAideEnfant = 4242

// TestAideEnfantFilmproc EST L'ENFANT. Il ecrit sur les DEUX flux (le parent les fusionne),
// emet sa ligne de protocole, et meurt sur le code demande.
func TestAideEnfantFilmproc(t *testing.T) {
	if *codeAideEnfant < 0 {
		t.Skip("pas un enfant : ce cas ne s'execute que re-execute par TestRun*")
	}
	fmt.Println("journal de l'enfant sur la sortie standard")
	fmt.Fprintln(os.Stderr, "journal de l'enfant sur l'erreur standard")
	EmitPeak(picAideEnfant)
	os.Exit(*codeAideEnfant)
}

// argsEnfantDeTest : la ligne de commande qui transforme ce binaire de test en enfant.
func argsEnfantDeTest(code int) []string {
	return []string{"-test.run=^TestAideEnfantFilmproc$", fmt.Sprintf("-filmproc-aide-code=%d", code)}
}

// TestRunRendLIssueDUnEnfantReel — LE PROTOCOLE, DE BOUT EN BOUT. Chaque code du protocole est
// rendu par un VRAI processus, et le parent doit le traduire en la bonne categorie.
func TestRunRendLIssueDUnEnfantReel(t *testing.T) {
	cas := []struct {
		code int
		veut Issue
	}{
		{CodeOK, IssueOK},
		{CodeSkipped, IssueSkipped},
		{CodeFailed, IssueFailed},
		{CodePreparation, IssuePreparation},
		{CodeMemory, IssueMemory},
		// HORS PROTOCOLE = MORT SUBITE. 1 est le code d'une erreur rendue par `main` : c'est
		// celui qu'un enfant produirait s'il oubliait le protocole, et il ne doit jamais passer
		// pour un succes.
		{1, IssueSuddenDeath},
		{42, IssueSuddenDeath},
	}
	for _, c := range cas {
		var journal strings.Builder
		r, err := NewRunner(t.TempDir(), &journal)
		if err != nil {
			t.Fatalf("NewRunner: %v", err)
		}
		res := r.Run(context.Background(), argsEnfantDeTest(c.code))
		if res.Code != c.code || res.Issue != c.veut {
			t.Errorf("enfant sorti en %d : Result{Code:%d, Issue:%v}, attendu {%d, %v}",
				c.code, res.Code, res.Issue, c.code, c.veut)
		}
		if res.Err != nil {
			t.Errorf("code %d : Err = %v — un code de sortie n'est pas un echec de lancement",
				c.code, res.Err)
		}
		if res.Peak != picAideEnfant {
			t.Errorf("code %d : pic relaye = %d, attendu %d — le protocole du tube est rompu",
				c.code, res.Peak, picAideEnfant)
		}
		if res.Dur <= 0 {
			t.Errorf("code %d : duree nulle — le cycle ne saurait pas ce que l'enfant a coute", c.code)
		}
		got := journal.String()
		if strings.Contains(got, peakMarker) {
			t.Errorf("code %d : la ligne de protocole a fuit dans le journal :\n%s", c.code, got)
		}
		// LES DEUX FLUX ARRIVENT : l'enfant journalise en slog (erreur standard) et ecrit son
		// recap en fmt (sortie standard). En perdre un rendrait la moitie d'une passe muette.
		for _, veut := range []string{"sortie standard", "erreur standard"} {
			if !strings.Contains(got, veut) {
				t.Errorf("code %d : le journal de l'enfant a perdu %q :\n%s", c.code, veut, got)
			}
		}
	}
}

// TestRunEchecDeLancementEstUneMortSubite — un binaire introuvable n'est pas un film en echec :
// c'est un incident de lancement, et il doit se voir comme tel (Err renseigne, jamais un code).
func TestRunEchecDeLancementEstUneMortSubite(t *testing.T) {
	r := &Runner{exe: filepathInexistant(t), env: childEnv(t.TempDir()), out: &strings.Builder{}}
	res := r.Run(context.Background(), nil)
	if res.Issue != IssueSuddenDeath || res.Err == nil {
		t.Fatalf("Result{Issue:%v, Err:%v}, attendu une mort subite avec erreur de lancement",
			res.Issue, res.Err)
	}
}

// filepathInexistant rend un chemin d'executable qui n'existe pas.
func filepathInexistant(t *testing.T) string {
	t.Helper()
	return t.TempDir() + string(os.PathSeparator) + "binaire-qui-n-existe-pas.exe"
}

// TestRelayNeCoupePasLesLignesLongues — les lignes de journal d'un decodage portent des chemins
// Windows et des compteurs : le plafond par defaut de bufio (64 Kio) les couperait, et une ligne
// coupee en deux se relit comme deux evenements.
func TestRelayNeCoupePasLesLignesLongues(t *testing.T) {
	var out strings.Builder
	r := &Runner{out: &out}
	longue := strings.Repeat("x", 200*1024)
	if pic := r.relay(strings.NewReader(longue + "\n" + peakMarker + "7\n")); pic != 7 {
		t.Fatalf("pic = %d, attendu 7 — le scanner s'est arrete sur la ligne longue", pic)
	}
	if !strings.Contains(out.String(), longue) {
		t.Errorf("la ligne de %d octets a ete tronquee par le relais", len(longue))
	}
}

// TestChildEnvPreserveLeResteDeLEnvironnement — imposer la racine ne doit rien AMPUTER : un
// enfant prive de PATH ou de TEMP ne demarrerait pas, et l'echec ne dirait pas pourquoi.
func TestChildEnvPreserveLeResteDeLEnvironnement(t *testing.T) {
	t.Setenv("LEVELUP_TEMOIN_ENV", "present")
	env := childEnv("/racine")
	if len(env) < len(os.Environ()) {
		t.Fatalf("l'environnement de l'enfant a perdu des entrees : %d < %d", len(env), len(os.Environ()))
	}
	var vu bool
	for _, kv := range env {
		if kv == "LEVELUP_TEMOIN_ENV=present" {
			vu = true
		}
	}
	if !vu {
		t.Error("une variable ordinaire du parent n'a pas ete transmise a l'enfant")
	}
}

// TestParsePeakFormesLimites — le marqueur est reconnu malgre les espaces de bord, et JAMAIS
// quand il est precede d'autre chose : une ligne de journal qui le CITE reste un journal.
func TestParsePeakFormesLimites(t *testing.T) {
	if _, ok := parsePeak("  " + peakMarker + "7 "); !ok {
		t.Error("le marqueur borde d'espaces doit rester reconnu")
	}
	if _, ok := parsePeak("prefixe " + peakMarker + "12"); ok {
		t.Error("un marqueur precede de texte a ete pris pour du protocole")
	}
}
