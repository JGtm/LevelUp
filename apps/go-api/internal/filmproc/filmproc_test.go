package filmproc

// filmproc_test.go — les invariants de l'executeur borne.
//
// CE QU'ILS PROTEGENT : la sentinelle declenche AU-DELA du plafond et une seule fois ; elle ne
// declenche PAS quand on la desarme ; le desarmement est idempotent (un appelant qui le differe
// ET l'appelle sur le chemin nominal ne doit pas paniquer) ; le code de sortie inconnu est une
// MORT SUBITE et jamais un succes ; la ligne de protocole du pic est reconnue et n'est jamais
// relayee dans le journal.
//
// LA MEMOIRE REELLE N'EST JAMAIS SOLLICITEE : la sonde est injectee. Un test qui allouerait
// vraiment des gibioctets pour observer la coupure serait exactement le comportement que ce
// paquet existe pour empecher.

import (
	"strings"
	"testing"
	"time"
)

// newGuardForTest arme une sentinelle sur une sonde controlee, sans toucher au tas reel.
func newGuardForTest(hard uint64, probe func() uint64, onExceeded func(uint64)) *Guard {
	return newGuardForTestPeriode(hard, time.Millisecond, probe, onExceeded)
}

// newGuardForTestPeriode : la meme chose, mais la PERIODE est choisie par le test — une
// periode assez longue pour qu'aucun tick ne survienne isole ce que la sentinelle rend AVANT
// son premier echantillonnage.
func newGuardForTestPeriode(hard uint64, periode time.Duration, probe func() uint64, onExceeded func(uint64)) *Guard {
	g := &Guard{hardLimit: hard, stop: make(chan struct{}), probe: probe}
	go g.watch(periode, onExceeded)
	return g
}

// LE PIC INCLUT L'INSTANT PRESENT. L'echantillonnage est a 250 ms : un processus qui meurt
// avant le premier tick (verrou refuse, builder indisponible, toute sortie tres precoce) n'a
// JAMAIS ete echantillonne. Sans re-mesure a l'appel, il emet un pic de ZERO et le recap de
// la passe imprime « (pic inconnu) » la ou l'ancienne sentinelle (picObserve, supprimee au
// lot v2 G.1) rendait toujours un chiffre vivant — constat C4 de la revue R1.
func TestPeakReechantillonneALAppel(t *testing.T) {
	g := newGuardForTestPeriode(1<<40, time.Hour, func() uint64 { return 4242 }, nil)
	defer g.Disarm()
	if p := g.Peak(); p != 4242 {
		t.Errorf("pic rendu avant le premier echantillonnage = %d, attendu 4242 "+
			"(la mesure doit inclure l'instant present)", p)
	}
}

func TestGuardDeclencheAuDelaDuPlafond(t *testing.T) {
	declenche := make(chan uint64, 4)
	g := newGuardForTest(1000, func() uint64 { return 5000 }, func(pic uint64) {
		declenche <- pic
	})
	defer g.Disarm()
	select {
	case pic := <-declenche:
		if pic != 5000 {
			t.Errorf("pic rendu = %d, attendu 5000", pic)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("la sentinelle n'a pas declenche au-dela du plafond dur")
	}
	// UNE SEULE FOIS : la surveillance s'arrete au declenchement, elle ne rappelle pas en
	// boucle pendant que l'appelant applique sa doctrine d'arret.
	select {
	case <-declenche:
		t.Error("onExceeded appele plus d'une fois")
	case <-time.After(50 * time.Millisecond):
	}
}

func TestGuardNeDeclenchePasSousLePlafond(t *testing.T) {
	declenche := make(chan uint64, 1)
	g := newGuardForTest(1_000_000, func() uint64 { return 42 }, func(pic uint64) {
		declenche <- pic
	})
	defer g.Disarm()
	select {
	case <-declenche:
		t.Fatal("declenchement sous le plafond dur")
	case <-time.After(100 * time.Millisecond):
	}
	if p := g.Peak(); p != 42 {
		t.Errorf("pic observe = %d, attendu 42 — la sentinelle doit tenir le pic meme sans declencher", p)
	}
}

// UN PLAFOND A ZERO DESARME : c'est l'echappatoire de l'operateur, et elle doit etre sure.
func TestGuardPlafondNulNeDeclencheJamais(t *testing.T) {
	declenche := make(chan uint64, 1)
	g := newGuardForTest(0, func() uint64 { return 1 << 40 }, func(pic uint64) {
		declenche <- pic
	})
	defer g.Disarm()
	select {
	case <-declenche:
		t.Fatal("declenchement alors que le plafond est desarme")
	case <-time.After(100 * time.Millisecond):
	}
}

// LE DESARMEMENT EST IDEMPOTENT : sans cela, un appelant qui le differe ET l'appelle sur le
// chemin nominal fermerait deux fois le meme canal — panique.
func TestGuardDesarmementIdempotent(t *testing.T) {
	g := newGuardForTest(1<<40, func() uint64 { return 0 }, nil)
	g.Disarm()
	g.Disarm() // ne doit pas paniquer
}

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
	pic := r.relay(src)
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
