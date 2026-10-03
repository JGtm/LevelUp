package replaynotify

import (
	"fmt"
	"testing"
	"time"
)

// t0 : instant de référence des tests. AUCUN appel à time.Now nulle part dans ce fichier —
// c'est tout l'intérêt de l'horloge injectée : les scénarios de fenêtre s'exécutent en
// microsecondes et sont rejouables à l'identique.
var t0 = time.Date(2026, 8, 26, 12, 0, 0, 0, time.UTC)

func ev(slug, match string) Event { return Event{TitleSlug: slug, MatchID: match} }

// TestGrouper_PremierEvenementArmeEtRienNeSortAvantEcheance — l'invariant de base :
// le message ne part pas à l'artefact, il part à l'échéance.
func TestGrouper_PremierEvenementArmeEtRienNeSortAvantEcheance(t *testing.T) {
	g := New(10 * time.Minute)
	g.Add(t0, ev("halo_infinite", "aaaa1111"))

	for _, d := range []time.Duration{0, time.Second, 5 * time.Minute, 10*time.Minute - time.Nanosecond} {
		if lots := g.Due(t0.Add(d)); len(lots) != 0 {
			t.Fatalf("à T+%s : %d lot(s) sorti(s), attendu 0 — la fenêtre n'est pas échue", d, len(lots))
		}
	}
	if lots := g.Due(t0.Add(10 * time.Minute)); len(lots) != 1 {
		t.Fatalf("à l'échéance exacte : %d lot(s), attendu 1", len(lots))
	}
}

// TestGrouper_UnSeulLotPourTouteLaFenetre — N artefacts dans la fenêtre = UN message.
func TestGrouper_UnSeulLotPourTouteLaFenetre(t *testing.T) {
	g := New(10 * time.Minute)
	g.Add(t0, ev("halo_infinite", "aaaa1111"))
	g.Add(t0.Add(2*time.Minute), ev("halo_infinite", "bbbb2222"))
	g.Add(t0.Add(9*time.Minute), ev("halo_infinite", "cccc3333"))

	lots := g.Due(t0.Add(10 * time.Minute))
	if len(lots) != 1 {
		t.Fatalf("lots = %d, attendu 1 (un seul message pour toute la fenêtre)", len(lots))
	}
	l := lots[0]
	if l.TitleSlug != "halo_infinite" || l.Total != 3 || l.Omitted != 0 {
		t.Errorf("lot = %+v, attendu titre halo_infinite, total 3, omis 0", l)
	}
	// Ordre d'arrivée conservé : le lecteur doit retrouver la chronologie.
	want := []string{"aaaa1111", "bbbb2222", "cccc3333"}
	for i, id := range want {
		if l.MatchIDs[i] != id {
			t.Errorf("MatchIDs[%d] = %q, attendu %q", i, l.MatchIDs[i], id)
		}
	}
}

// TestGrouper_DedupSansDecalerEcheance — LE PIÈGE : un match re-cuit en boucle ne doit ni
// compter deux fois, ni repousser le message.
func TestGrouper_DedupSansDecalerEcheance(t *testing.T) {
	g := New(10 * time.Minute)
	g.Add(t0, ev("halo_infinite", "aaaa1111"))
	g.Add(t0.Add(4*time.Minute), ev("halo_infinite", "aaaa1111"))
	g.Add(t0.Add(9*time.Minute), ev("halo_infinite", "aaaa1111"))

	lots := g.Due(t0.Add(10 * time.Minute))
	if len(lots) != 1 {
		t.Fatalf("lots = %d, attendu 1 — l'échéance a été repoussée par une re-cuisson", len(lots))
	}
	if lots[0].Total != 1 || len(lots[0].MatchIDs) != 1 {
		t.Errorf("lot = %+v, attendu 1 match distinct", lots[0])
	}
}

// TestGrouper_FenetreDesarmeeApresFlush — après le message, tout repart de zéro : le
// prochain artefact arme une fenêtre NEUVE (il ne sort pas immédiatement).
func TestGrouper_FenetreDesarmeeApresFlush(t *testing.T) {
	g := New(10 * time.Minute)
	g.Add(t0, ev("halo_infinite", "aaaa1111"))
	if len(g.Due(t0.Add(10*time.Minute))) != 1 {
		t.Fatal("premier lot non sorti")
	}
	if lots := g.Due(t0.Add(20 * time.Minute)); len(lots) != 0 {
		t.Fatalf("%d lot(s) fantôme(s) après flush, attendu 0", len(lots))
	}

	t1 := t0.Add(20 * time.Minute)
	g.Add(t1, ev("halo_infinite", "bbbb2222"))
	if lots := g.Due(t1.Add(9 * time.Minute)); len(lots) != 0 {
		t.Fatalf("la nouvelle fenêtre a rendu %d lot(s) avant échéance", len(lots))
	}
	lots := g.Due(t1.Add(10 * time.Minute))
	if len(lots) != 1 || lots[0].Total != 1 || lots[0].MatchIDs[0] != "bbbb2222" {
		t.Fatalf("second lot = %+v, attendu le seul bbbb2222", lots)
	}
}

// TestGrouper_DeuxTitresDeuxFenetresIndependantes — chaque titre a SA fenêtre (libellés et
// langue par titre), et l'ordre de sortie est stable malgré le parcours de map.
func TestGrouper_DeuxTitresDeuxFenetresIndependantes(t *testing.T) {
	g := New(10 * time.Minute)
	g.Add(t0, ev("halo_infinite", "aaaa1111"))
	g.Add(t0.Add(5*time.Minute), ev("halo_5", "bbbb2222"))

	// À T+10 seul halo_infinite est échu.
	lots := g.Due(t0.Add(10 * time.Minute))
	if len(lots) != 1 || lots[0].TitleSlug != "halo_infinite" {
		t.Fatalf("à T+10 : %+v, attendu le seul halo_infinite", lots)
	}
	lots = g.Due(t0.Add(15 * time.Minute))
	if len(lots) != 1 || lots[0].TitleSlug != "halo_5" {
		t.Fatalf("à T+15 : %+v, attendu le seul halo_5", lots)
	}

	// Deux titres échus au MÊME tick : ordre stable (tri par slug).
	g2 := New(10 * time.Minute)
	g2.Add(t0, ev("halo_infinite", "aaaa1111"))
	g2.Add(t0, ev("halo_5", "bbbb2222"))
	both := g2.Due(t0.Add(10 * time.Minute))
	if len(both) != 2 || both[0].TitleSlug != "halo_5" || both[1].TitleSlug != "halo_infinite" {
		t.Fatalf("ordre = %+v, attendu [halo_5, halo_infinite] (tri stable)", both)
	}
}

// TestGrouper_ListeTronqueeEtResteCompte — au-delà de MaxListed on tronque, mais le total
// annoncé reste juste (« et N autres »). Une liste non bornée ferait rejeter l'embed.
func TestGrouper_ListeTronqueeEtResteCompte(t *testing.T) {
	g := New(10 * time.Minute)
	const n = MaxListed + 7
	for i := 0; i < n; i++ {
		g.Add(t0, ev("halo_infinite", fmt.Sprintf("match%03d", i)))
	}
	lots := g.Due(t0.Add(10 * time.Minute))
	if len(lots) != 1 {
		t.Fatalf("lots = %d, attendu 1", len(lots))
	}
	l := lots[0]
	if len(l.MatchIDs) != MaxListed {
		t.Errorf("énumérés = %d, attendu %d", len(l.MatchIDs), MaxListed)
	}
	if l.Total != n {
		t.Errorf("total = %d, attendu %d", l.Total, n)
	}
	if l.Omitted != n-MaxListed {
		t.Errorf("omis = %d, attendu %d", l.Omitted, n-MaxListed)
	}
}

// TestGrouper_PlafondMemoireDeborde — au-delà de MaxPending on cesse de MÉMORISER mais on
// continue de COMPTER : sans ce plafond, une instance sans webhook verrait la mémoire
// croître au rythme des artefacts, sans fin.
func TestGrouper_PlafondMemoireDeborde(t *testing.T) {
	g := New(10 * time.Minute)
	const extra = 12
	for i := 0; i < MaxPending+extra; i++ {
		g.Add(t0, ev("halo_infinite", fmt.Sprintf("match%04d", i)))
	}
	_, artefacts := g.Pending()
	if artefacts != MaxPending+extra {
		t.Errorf("Pending() = %d artefacts, attendu %d (le débordement doit rester compté)",
			artefacts, MaxPending+extra)
	}
	lots := g.Due(t0.Add(10 * time.Minute))
	if len(lots) != 1 || lots[0].Total != MaxPending+extra {
		t.Fatalf("lot = %+v, attendu un total de %d", lots, MaxPending+extra)
	}
	if lots[0].Omitted != MaxPending+extra-MaxListed {
		t.Errorf("omis = %d, attendu %d", lots[0].Omitted, MaxPending+extra-MaxListed)
	}
}

// TestGrouper_VideEtEvenementsSansIdentite — rien à envoyer ne produit rien, et un
// événement sans identité est ignoré (l'appelant le journalise avant d'appeler).
func TestGrouper_VideEtEvenementsSansIdentite(t *testing.T) {
	g := New(0) // 0 → DefaultWindow
	if lots := g.Due(t0.Add(time.Hour)); len(lots) != 0 {
		t.Fatalf("groupeur vide : %d lot(s), attendu 0", len(lots))
	}
	g.Add(t0, ev("", "aaaa1111"))
	g.Add(t0, ev("halo_infinite", ""))
	if titres, artefacts := g.Pending(); titres != 0 || artefacts != 0 {
		t.Errorf("Pending() = (%d, %d), attendu (0, 0) — un événement sans identité n'arme rien",
			titres, artefacts)
	}
	if lots := g.Due(t0.Add(DefaultWindow)); len(lots) != 0 {
		t.Fatalf("%d lot(s) issus d'événements sans identité, attendu 0", len(lots))
	}
}
