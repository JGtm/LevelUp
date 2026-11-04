package memguard

// memguard_test.go — les invariants de la sentinelle memoire.
//
// CE QU'ILS PROTEGENT : la sentinelle declenche AU-DELA du plafond et une seule fois ; elle ne
// declenche PAS quand on la desarme ; le desarmement est idempotent (un appelant qui le differe
// ET l'appelle sur le chemin nominal ne doit pas paniquer).
//
// LA MEMOIRE REELLE N'EST JAMAIS SOLLICITEE : la sonde est injectee. Un test qui allouerait
// vraiment des gibioctets pour observer la coupure serait exactement le comportement que ce
// paquet existe pour empecher.

import (
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
