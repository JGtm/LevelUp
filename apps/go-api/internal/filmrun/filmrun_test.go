package filmrun

// filmrun_test.go — les invariants du lanceur parent/enfant.
//
// CE QU'ILS PROTEGENT : le code de sortie inconnu est une MORT SUBITE et jamais un succes ; la
// ligne de protocole du pic est reconnue et n'est jamais relayee dans le journal ; les racines
// imposees a l'enfant le sont en une seule occurrence.

import (
	"strings"
	"testing"
)

func TestIssueForCode(t *testing.T) {
	cas := []struct {
		code int
		veut Issue
	}{
		{CodeOK, IssueOK},
		{CodeSkipped, IssueSkipped},
		{CodeFailed, IssueFailed},
		{CodePreparation, IssuePreparation},
		{CodeMemory, IssueMemory},
		// LES CODES QU'ON N'A PAS PREVUS SONT DES MORTS SUBITES. 1 = erreur rendue par main,
		// 2 = flag.ExitOnError et `fatal error` du runtime, -1 = tue par un signal, 137 = OOM
		// killer. Aucun ne doit passer pour un succes ni pour un echec ordinaire.
		{1, IssueSuddenDeath},
		{2, IssueSuddenDeath},
		{-1, IssueSuddenDeath},
		{137, IssueSuddenDeath},
		{99, IssueSuddenDeath},
	}
	for _, c := range cas {
		if got := IssueForCode(c.code); got != c.veut {
			t.Errorf("IssueForCode(%d) = %v, attendu %v", c.code, got, c.veut)
		}
	}
}

// LES CODES DU PROTOCOLE COMMENCENT A 10, et c'est ce qui les rend distinguables de ce que
// produisent le runtime et le paquet flag. Ce cas fige la regle plutot que la convention.
func TestCodesProtocoleHorsDePorteeDuRuntime(t *testing.T) {
	for _, c := range []int{CodeSkipped, CodeFailed, CodePreparation, CodeMemory} {
		if c < 10 {
			t.Errorf("code de protocole %d < 10 : il pourrait etre confondu avec un code du "+
				"runtime (1 = erreur de main, 2 = flag.ExitOnError)", c)
		}
	}
}

func TestParsePeak(t *testing.T) {
	if v, ok := parsePeak(peakMarker + "123456"); !ok || v != 123456 {
		t.Errorf("parsePeak = (%d, %v), attendu (123456, true)", v, ok)
	}
	// Une ligne de journal ordinaire n'est pas du protocole.
	if _, ok := parsePeak("INFO rejeu : couverture par calque"); ok {
		t.Error("une ligne de journal a ete prise pour du protocole")
	}
	// Un marqueur mal forme n'est pas du protocole non plus — mieux vaut le relayer que
	// l'avaler en silence.
	if _, ok := parsePeak(peakMarker + "pas-un-nombre"); ok {
		t.Error("un marqueur illisible a ete accepte")
	}
}

// LE PROTOCOLE NE FUIT JAMAIS DANS LE JOURNAL : l'operateur lit un journal, pas un protocole.
func TestRelayInterceptLeProtocoleEtRelaieLeReste(t *testing.T) {
	var out strings.Builder
	r := &Runner{out: &out}
	src := strings.NewReader("premiere ligne\n" + peakMarker + "4096\nseconde ligne\n")
	pic, _ := r.relay(src)
	if pic != 4096 {
		t.Errorf("pic relaye = %d, attendu 4096", pic)
	}
	got := out.String()
	if strings.Contains(got, peakMarker) {
		t.Errorf("le marqueur de protocole a fuit dans le journal : %q", got)
	}
	for _, veut := range []string{"premiere ligne", "seconde ligne"} {
		if !strings.Contains(got, veut) {
			t.Errorf("ligne %q absente du journal relaye : %q", veut, got)
		}
	}
}

// LA RACINE DU DEPOT EST IMPOSEE A L'ENFANT, en une seule occurrence : sans le retrait
// prealable, l'environnement en porterait deux et la derniere gagnerait par hasard.
func TestChildEnvImposeUneSeuleRacine(t *testing.T) {
	t.Setenv(EnvRepoRoot, "/ancienne/racine")
	env := childEnv("/nouvelle/racine")
	n := 0
	for _, kv := range env {
		if strings.HasPrefix(strings.ToUpper(kv), EnvRepoRoot+"=") {
			n++
			if kv != EnvRepoRoot+"=/nouvelle/racine" {
				t.Errorf("racine transmise = %q, attendu la racine du parent", kv)
			}
		}
	}
	if n != 1 {
		t.Errorf("%d occurrence(s) de %s dans l'environnement de l'enfant, attendu 1", n, EnvRepoRoot)
	}
}

// LES TROIS RACINES SONT IMPOSEES A L'ENFANT, chacune en une seule occurrence ; une racine
// vide est RETIREE (l'enfant retombe sur son defaut, jamais sur une valeur heritee du parent).
func TestChildEnvRootsImposeLesTroisRacines(t *testing.T) {
	t.Setenv(EnvResourcesDir, "/ancienne")
	t.Setenv(EnvDataDir, "/ancienne-donnees")
	t.Setenv(EnvLogsDir, "/ancien-journal")
	env := childEnvRoots(Roots{Resources: "/res", Data: "/donnees"})
	got := map[string][]string{}
	for _, kv := range env {
		name, val, _ := strings.Cut(kv, "=")
		switch strings.ToUpper(name) {
		case EnvRepoRoot, EnvResourcesDir, EnvDataDir, EnvLogsDir:
			got[strings.ToUpper(name)] = append(got[strings.ToUpper(name)], val)
		}
	}
	want := map[string]string{EnvRepoRoot: "/res", EnvResourcesDir: "/res", EnvDataDir: "/donnees"}
	for k, v := range want {
		if len(got[k]) != 1 || got[k][0] != v {
			t.Errorf("%s = %v, attendu une seule occurrence %q", k, got[k], v)
		}
	}
	if len(got[EnvLogsDir]) != 0 {
		t.Errorf("une racine vide ne doit pas etre imposee : %s = %v", EnvLogsDir, got[EnvLogsDir])
	}
}

// Le lanceur historique impose la racine de ressources en plus de LEVELUP_REPO_ROOT, sinon une
// LEVELUP_RESOURCES_DIR heritee l'emporterait dans l'enfant.
func TestChildEnvImposeAussiLaRacineDeRessources(t *testing.T) {
	t.Setenv(EnvResourcesDir, "/heritee")
	t.Setenv(EnvDataDir, "/donnees-heritees")
	env := childEnv("/nouvelle")
	var res, data string
	for _, kv := range env {
		switch {
		case strings.HasPrefix(kv, EnvResourcesDir+"="):
			res = kv
		case strings.HasPrefix(kv, EnvDataDir+"="):
			data = kv
		}
	}
	if res != EnvResourcesDir+"=/nouvelle" {
		t.Errorf("racine de ressources transmise = %q", res)
	}
	if data != EnvDataDir+"=/donnees-heritees" {
		t.Errorf("le lanceur historique laisse les donnees heritees : %q", data)
	}
}
