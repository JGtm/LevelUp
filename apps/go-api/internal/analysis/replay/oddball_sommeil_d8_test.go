package replay

// oddball_sommeil_d8_test.go — D8 VOLET 1 : LE SOMMEIL DE L'OBJET EXISTE-T-IL ?
//
// LE PROTOCOLE EST ECRIT ET COMMITE AVANT CE FICHIER (`.ai/V7.5/PLAN_OBJECTIFS_ETAT_VIVANT_
// 2026-08.md`, section « D8 »). Ce qui suit l'applique.
//
// # L'HYPOTHESE, ET D'OU ELLE VIENT
//
// D7 a mesure que l'objet est A L'ARRET quand il se tait, et que le lacher est capte a 74/62 %
// quand le ramassage ne l'est qu'a 45/48 %. Une asymetrie de ~26 points sur un canal symetrique.
// L'explication qui colle : un objet immobile CESSE d'etre replique. Le ramassage n'aurait alors
// pas lieu a l'instant du silence, mais PLUS TARD — quand un joueur passe sur le lieu de repos.
//
// # CE QUE CE VOLET CORRIGE DE MA PROPRE SONDE
//
// D7 classait les naissances dans un ORDRE de priorite, ce qui masquait les recouvrements et
// rendait sa troisieme colonne ininterpretable. Ici chaque naissance est notee sur les TROIS
// attributs A LA FOIS, et les recouvrements sortent tels quels.
//
// REGIME : gardes `ATT_FILM` + `ODDBALL_FILM`, UN FILM PAR PROCESSUS, lecture seule, AUCUNE base.

import (
	"math"
	"os"
	"sort"
	"testing"

	"levelup/go-api/internal/filmproc"
)

const (
	// d8SurPlaceM : distance en deca de laquelle une naissance est « au LIEU EXACT du silence
	// precedent ». Un metre — l'objet se reveille ou il s'est endormi, pas « dans le coin ».
	d8SurPlaceM = 1.0
	// d8PartTraverseeMin / d8DelaiMedianMinMS : le critere de « sommeil etabli », recopie du
	// protocole. Le point de comparaison des 70 % est DEJA MESURE : 45,5 % et 47,8 % a l'instant
	// du silence seul (D6/D7), et elargir a +/- 10 images ne le bouge pas.
	d8PartTraverseeMin = 0.70
	d8DelaiMedianMinMS = 1000
	// d8PontMinimum : part des slots de bipede que le pont doit NOMMER pour qu'un film soit
	// exploitable. Le corpus separe 89,7 / 86,1 / 87,5 % contre 10,7 % — un facteur huit.
	d8PontMinimum = 0.50
)

// TestOddballSommeilD8 — VOLET 1. Un film par processus.
func TestOddballSommeilD8(t *testing.T) {
	root := attRequireRoot(t)
	id := os.Getenv(d4FilmEnv)
	if id == "" {
		t.Skipf("mesure non demandee : %s vide", d4FilmEnv)
	}
	g := filmproc.Arm("d8-sommeil", filmproc.MeasureLimitGiB, func(peak uint64) {
		t.Errorf("PLAFOND MEMOIRE DEPASSE %s : %.2f Gio", id, float64(peak)/(1<<30))
	})
	defer func() {
		g.Disarm()
		t.Logf("%s : pic memoire observe %.2f Gio", id, float64(g.Peak())/(1<<30))
	}()

	e, ok := d8Charge(t, root, id)
	if !ok {
		return
	}
	d8Naissances(t, id, e)
	d8Traversees(t, id, e)
}

// d8Etat porte ce qu'un film rend une fois lu (regle des 5 parametres).
type d8Etat struct {
	vies   []flagFreeLife
	socles []PointObjective
	tracks map[uint32]slotTrack
	pont   objBridge
}

// d8Charge lit le film et applique la PRECONDITION DE PONT, ecrite au protocole cette fois.
func d8Charge(t *testing.T, root, id string) (d8Etat, bool) {
	t.Helper()
	vies, socles, ok := d6ViesEtSocles(t, root, id)
	if !ok {
		return d8Etat{}, false
	}
	wr, ok := d6Bornes(t, root, id)
	if !ok {
		return d8Etat{}, false
	}
	dir := objChunkDir(root, id)
	pos, err := d6Positions(dir, wr)
	if err != nil {
		t.Fatalf("%s : positions de bipede illisibles : %v", id, err)
	}
	tracks := indexBySlot(pos)
	pont := objBridgeOf(t, root, id)
	nommes := 0
	for slot := range tracks {
		if _, ok := pont.SlotXUID[slot]; ok {
			nommes++
		}
	}
	part := float64(nommes) / math.Max(float64(len(tracks)), 1)
	t.Logf("%s : %d vie(s) libre(s), %d socle(s), pont %d/%d slot(s) nomme(s) = %.1f %% "+
		"(plancher %.0f %%)", id, len(vies), len(socles), nommes, len(tracks), 100*part,
		100*d8PontMinimum)
	if part < d8PontMinimum {
		t.Logf("NON EXPLOITABLE %s : le pont nomme %.1f %% des slots, sous le plancher de %.0f %%. "+
			"NI POUR NI CONTRE — et ce film ne sera pas compte au denominateur.",
			id, 100*part, 100*d8PontMinimum)
		return d8Etat{}, false
	}
	return d8Etat{vies: vies, socles: socles, tracks: tracks, pont: pont}, true
}

// d8Naissances reclasse les naissances SANS ORDRE DE PRIORITE : chaque naissance est notee sur
// les trois attributs a la fois, et les recouvrements sortent tels quels.
func d8Naissances(t *testing.T, id string, e d8Etat) {
	t.Helper()
	var socle, joueur, surPlace, aucun, socleEtJoueur, surPlaceEtJoueur int
	var dists []float64
	for i, l := range e.vies {
		x, y := l.First()
		auSocle := d6NaitAuSocle(l, e.socles)
		_, dJ, _ := d6PlusProche(e.tracks, e.pont, l.T0US, x, y)
		auJoueur := dJ <= d6RayonRamassageM
		auLieu := false
		if i > 0 {
			px, py := e.vies[i-1].Last()
			d := math.Hypot(float64(x)-float64(px), float64(y)-float64(py))
			dists = append(dists, d)
			auLieu = d <= d8SurPlaceM
		}
		if auSocle {
			socle++
		}
		if auJoueur {
			joueur++
		}
		if auLieu {
			surPlace++
		}
		if auSocle && auJoueur {
			socleEtJoueur++
		}
		if auLieu && auJoueur {
			surPlaceEtJoueur++
		}
		if !auSocle && !auJoueur && !auLieu {
			aucun++
		}
	}
	n := len(e.vies)
	t.Logf("D8.1 %s : %d naissance(s) — AU SOCLE %s · AUX PIEDS D UN JOUEUR %s · "+
		"AU LIEU DU SILENCE %s · AUCUN des trois %s",
		id, n, d6Part(socle, n), d6Part(joueur, n), d6Part(surPlace, n), d6Part(aucun, n))
	t.Logf("D8.1 %s : RECOUVREMENTS (ce que l ordre de priorite de D7 masquait) — "+
		"socle ET joueur %d ; lieu du silence ET joueur %d", id, socleEtJoueur, surPlaceEtJoueur)
	d8Quantiles(t, id, "distance naissance -> silence precedent", dists)
}

// d8Traversees mesure QUAND un joueur passe sur le lieu de repos, et applique le critere.
func d8Traversees(t *testing.T, id string, e d8Etat) {
	t.Helper()
	// LES PAIRES SE RECOMPTENT ICI PLUTOT QUE DE REUTILISER `d7Trous` : cette liste-la SAUTE les
	// paires qui se chevauchent, donc son indice ne correspond plus a celui des vies. S'en servir
	// pour retrouver la vie suivante aurait apparie le mauvais couple, silencieusement.
	var trous []d7Silence
	var fins []uint64
	for i := 0; i+1 < len(e.vies); i++ {
		if e.vies[i+1].T0US <= e.vies[i].T1US {
			continue
		}
		x, y := e.vies[i].Last()
		trous = append(trous, d7Silence{auUS: e.vies[i].T1US, x: x, y: y})
		fins = append(fins, e.vies[i+1].T0US)
	}
	if len(trous) == 0 {
		t.Logf("D8.2 %s : aucun trou confrontable", id)
		return
	}
	traverses := 0
	var delais []float64
	for i, s := range trous {
		if at, ok := d8PremiereTraversee(e, s, fins[i]); ok {
			traverses++
			delais = append(delais, float64(at-s.auUS)/1e6)
		}
	}
	part := float64(traverses) / float64(len(trous))
	med := d8Mediane(delais)
	t.Logf("D8.2 %s : %s trou(s) TRAVERSE(s) pendant leur duree (seuil %.0f %%) ; "+
		"delai silence -> premiere traversee : mediane %.2f s (seuil >= %.1f s)",
		id, d6Part(traverses, len(trous)), 100*d8PartTraverseeMin, med,
		float64(d8DelaiMedianMinMS)/1000)
	d8Quantiles(t, id, "delai silence -> premiere traversee (s)", delais)
	ok1 := part >= d8PartTraverseeMin
	ok2 := med*1000 >= d8DelaiMedianMinMS
	verdict := "SOMMEIL NON ETABLI"
	if ok1 && ok2 {
		verdict = "SOMMEIL ETABLI"
	}
	t.Logf("VERDICT %s : traversees %s, delai %s — %s",
		id, d8Oui(ok1), d8Oui(ok2), verdict)
}

// d8PremiereTraversee rend l'instant du PREMIER passage d'un joueur nomme a portee du lieu de
// repos, strictement pendant le trou.
//
// LE PAS EST L'IMAGE : on ne cherche pas un echantillon exact mais un PASSAGE, et un joueur qui
// traverse est a portee pendant plusieurs images consecutives.
func d8PremiereTraversee(e d8Etat, s d7Silence, finUS uint64) (uint64, bool) {
	for at := s.auUS; at < finUS; at += d7ImageUS {
		if _, d, _ := d6PlusProche(e.tracks, e.pont, at, s.x, s.y); d <= d6RayonRamassageM {
			return at, true
		}
	}
	return 0, false
}

// d8Quantiles publie une distribution. UNE DISTRIBUTION PLUTOT QU'UNE MOYENNE : c'est elle qui
// dit si deux populations se separent, et c'est la seule chose qui rende un seuil defendable.
func d8Quantiles(t *testing.T, id, quoi string, v []float64) {
	t.Helper()
	if len(v) == 0 {
		t.Logf("  %s : %s — aucune mesure", id, quoi)
		return
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	q := func(p float64) float64 { return s[int(float64(len(s)-1)*p)] }
	t.Logf("  %s : %s — n=%d, min %.2f, q25 %.2f, MEDIANE %.2f, q75 %.2f, q90 %.2f, max %.2f",
		id, quoi, len(s), s[0], q(0.25), q(0.50), q(0.75), q(0.90), s[len(s)-1])
}

// d8Mediane rend la mediane, zero sur une serie vide.
func d8Mediane(v []float64) float64 {
	if len(v) == 0 {
		return 0
	}
	s := append([]float64(nil), v...)
	sort.Float64s(s)
	return s[len(s)/2]
}

// d8Oui rend un booleen lisible.
func d8Oui(b bool) string {
	if b {
		return "OUI"
	}
	return "NON"
}
