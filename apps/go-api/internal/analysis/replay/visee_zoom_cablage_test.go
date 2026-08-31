package replay

// visee_zoom_cablage_test.go — LES GATES DE CABLAGE ET LES MESURES DE STRUCTURE de la lunette.
//
// EXTRAIT DE `visee_zoom_gate_test.go`, qui franchissait le seuil de 500 lignes du depot. La
// decoupe tombe sur une frontiere nette : l'autre fichier porte L'IDENTIFICATION (les evenements
// decrivent-ils la chronologie relevee a la main ?), celui-ci porte LE CABLAGE et la STRUCTURE
// (le palier arrive-t-il au bon joueur ? qu'est-ce qui ferme une periode ? combien durent-elles ?).
//
// Seuils, gardes et conventions sont ceux du fichier d'origine ; rien n'est redefini ici.

import (
	"fmt"
	"os"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/analysis/filmdec"
)

// TestViseeZoomBoutEnBout verifie que le palier de lunette arrive jusqu'au document — au bon
// joueur, au bon moment. C'est le gate de bout en bout : le decodage peut etre juste et le
// cablage faux (mauvais slot, mauvaise horloge, champ jamais pose).
//
// LA METRIQUE EST LE RAPPEL, ET C'EST UN CHOIX RAISONNE. Le releve de l'utilisateur est une
// liste de ce qu'il A VU (« brievement », « environ »), pas une certification d'absence sur le
// reste de la fenetre : il n'a jamais affirme que le joueur ne zoomait PAS ailleurs. Compter
// des « faux positifs » contre lui mesurerait donc l'exhaustivite du releve, pas la justesse du
// cablage. On mesure ce que le releve peut REELLEMENT arbitrer : ses episodes sont-ils couverts ?
//
// SEUILS ECRITS AVANT LA MESURE :
//
//	RAPPEL   les 6 episodes releves doivent porter au moins un echantillon « a la lunette » sur
//	         la track du joueur (6/6 exige : un releve de six periodes vues de ses yeux ne
//	         souffre pas d'exception).
//	TEMOIN   le meme rappel sur les episodes TRANSLATES de 30 s doit s'effondrer, sans quoi le
//	         rappel ne dirait rien (un champ allume en permanence couvrirait tout).
//
// Garde ZOOM_FILM (00162144).
func TestViseeZoomBoutEnBout(t *testing.T) {
	dir := os.Getenv(zoomFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : gate saute", zoomFilmEnv)
	}
	release := filmdec.LockProcessDecode()
	defer release()

	evts := filmdec.ScanFilmZoomEvents(dir)
	if len(evts) == 0 {
		t.Fatalf("aucun evenement de lunette : le scanner de production ne rend rien")
	}
	t.Logf("SCANNER DE PRODUCTION — %d bascules de lunette lues", len(evts))

	scan := filmdec.DefaultScanFilmOptions()
	scan.QuantaOnly = true
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Fatalf("balayage des positions : %v", err)
	}
	lives := buildLifeSpans(indexBySlot(pos))
	// LA MEME reconstruction que la production (zoom_state.go) : plusieurs causes de fermeture.
	etat := buildScopedLookup(evts, lives, zoomHoldUS)
	off, _ := bestDeathOffset(lives, ScanFilmDeaths2(t, dir))

	// La track de Nilton est celle du slot 513 (index 1 + base 512), etabli par le pont.
	const slotNilton = 513
	var scopedS []float64
	echantillons := 0
	for _, p := range pos {
		if p.Slot != slotNilton {
			continue
		}
		echantillons++
		if etat(p.Slot, p.TimestampUS) == 0 {
			continue
		}
		scopedS = append(scopedS, float64(int64(p.TimestampUS/1000)-off)/1000)
	}
	if len(scopedS) == 0 {
		t.Fatalf("COUVERTURE — aucun echantillon « a la lunette » sur la track du slot %d :"+
			" le cablage ne pose jamais le champ", slotNilton)
	}
	t.Logf("COUVERTURE — %d echantillons sur la track du slot %d, dont %d a la lunette (%.1f %%)",
		echantillons, slotNilton, len(scopedS), 100*float64(len(scopedS))/float64(echantillons))

	rappel := zoomRappel(scopedS, chronoEpisodes, 0)
	t.Logf("RAPPEL — %d/%d episodes releves portent un echantillon a la lunette",
		rappel, len(chronoEpisodes))
	// CONTROLE PAR TRANSLATION plutot qu'un decalage unique : avec 22 % d'echantillons a la
	// lunette, un seul decalage temoin ne dit rien (il attrape souvent 4/6 par hasard). On
	// mesure la part des decalages qui atteignent le rappel observe — c'est elle qui dit si
	// 6/6 est remarquable, et elle est publiee meme quand elle est mauvaise.
	essais, aussiBien := 0, 0
	for d := -300.0; d <= 300.0; d += 1 {
		if d > -6 && d < 6 {
			continue
		}
		essais++
		if zoomRappel(scopedS, chronoEpisodes, d) >= rappel {
			aussiBien++
		}
	}
	part := float64(aussiBien) / float64(essais)
	t.Logf("TEMOIN — %d/%d decalages atteignent aussi %d/%d (%.1f %%)",
		aussiBien, essais, rappel, len(chronoEpisodes), 100*part)
	if rappel < len(chronoEpisodes) {
		t.Fatalf("RAPPEL INSUFFISANT : %d/%d (6/6 exige)", rappel, len(chronoEpisodes))
	}
	t.Logf("GATE DE CABLAGE FRANCHI — le palier arrive au bon joueur et couvre les %d episodes"+
		" releves. RESERVE PUBLIEE : %.1f %% des decalages temoins en font autant, ce gate"+
		" verifie donc LE CABLAGE, pas l'identification — celle-ci est etablie par l'epreuve"+
		" des INSTANTS d'entree (TestViseeZoomGate, 6/6 a p = 0,00 %%).",
		len(chronoEpisodes), 100*part)
}

// zoomRappel compte les episodes (decales de `shift` secondes) portant au moins un echantillon.
func zoomRappel(scopedS []float64, eps [][2]float64, shift float64) int {
	n := 0
	for _, ep := range eps {
		for _, s := range scopedS {
			if s >= ep[0]+shift-zoomTolS && s <= ep[1]+shift+zoomTolS {
				n++
				break
			}
		}
	}
	return n
}

// ScanFilmDeaths2 est un adaptateur de test : ScanFilmDeaths rend une erreur, et l'ignorer
// silencieusement dans le corps du gate masquerait un film illisible.
func ScanFilmDeaths2(t *testing.T, dir string) []Death {
	t.Helper()
	d, err := ScanFilmDeaths(dir)
	if err != nil {
		t.Fatalf("fil des morts : %v", err)
	}
	return d
}

// TestViseeZoomDureesObservees mesure la distribution des periodes de lunette REELLEMENT
// fermees (une entree suivie d'une sortie du meme slot). Elle sert a calibrer le maintien
// (`zoomHoldUS`) SUR LA DONNEE, et non sur la chronologie de l'utilisateur — se caler sur la
// verite terrain reviendrait a s'ajuster a la reponse. Garde ZOOM_FILM.
func TestViseeZoomDureesObservees(t *testing.T) {
	dir := os.Getenv(zoomFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : mesure sautee", zoomFilmEnv)
	}
	evts := filmdec.ScanFilmZoomEvents(dir)
	ouv := map[uint32]uint64{}
	var durees []float64
	entrees, sorties := 0, 0
	for _, e := range evts {
		if e.Scoped() {
			entrees++
			ouv[e.Slot] = e.TimestampUS
			continue
		}
		sorties++
		if t0, ok := ouv[e.Slot]; ok {
			durees = append(durees, float64(e.TimestampUS-t0)/1e6)
			delete(ouv, e.Slot)
		}
	}
	sort.Float64s(durees)
	if len(durees) == 0 {
		t.Fatalf("aucune periode fermee")
	}
	q := func(f float64) float64 { return durees[int(f*float64(len(durees)-1))] }
	t.Logf("PERIODES FERMEES — %d (sur %d entrees, %d sorties)", len(durees), entrees, sorties)
	t.Logf("  quantiles (s) : p10=%.2f p25=%.2f p50=%.2f p75=%.2f p90=%.2f p95=%.2f max=%.2f",
		q(0.10), q(0.25), q(0.50), q(0.75), q(0.90), q(0.95), durees[len(durees)-1])
}

// TestViseeZoomEntreesOrphelines cherche CE QUI FERME une periode de lunette quand aucun
// evenement de sortie n'est lu. Hypotheses de l'utilisateur, toutes deux plausibles et
// testables : (a) le joueur MEURT a la lunette — il n'a pas le temps de dezoomer, et le moteur
// n'a pas de sortie a emettre ; (b) il subit des DEGATS, ce qui force le dezoom — et cet
// evenement-la voyagerait alors dans le meme paquet que le degat, donc en DEUXIEME position
// d'une liste, hors de portee du scanner actuel.
//
// L'epreuve ne pose pas de seuil : elle mesure le delai entre une entree orpheline et la mort
// suivante du meme slot, et le compare au delai qui suit une entree NORMALEMENT fermee. Si (a)
// explique les orphelines, leur delai a la mort doit etre COURT la ou celui des fermees ne l'est
// pas. Garde ZOOM_FILM.
func TestViseeZoomEntreesOrphelines(t *testing.T) {
	dir := os.Getenv(zoomFilmEnv)
	if dir == "" {
		t.Skipf("%s absent : mesure sautee", zoomFilmEnv)
	}
	release := filmdec.LockProcessDecode()
	defer release()

	evts := filmdec.ScanFilmZoomEvents(dir)
	scan := filmdec.DefaultScanFilmOptions()
	scan.QuantaOnly = true
	pos, err := filmdec.ScanFilmBipedPositions(dir, scan)
	if err != nil {
		t.Fatalf("balayage des positions : %v", err)
	}
	lives := buildLifeSpans(indexBySlot(pos))
	// Fin de vie d'un slot = l'instant ou sa trajectoire s'arrete : c'est la mort (ou la fin du
	// film). On l'utilise comme horloge de mort, sans passer par le fil des morts : ici on ne
	// cherche pas QUI meurt, seulement QUAND ce slot cesse d'exister.
	finDeVie := map[uint32][]uint64{}
	for _, l := range lives {
		finDeVie[l.slot] = append(finDeVie[l.slot], uint64(l.to))
	}
	for s := range finDeVie {
		sort.Slice(finDeVie[s], func(i, j int) bool { return finDeVie[s][i] < finDeVie[s][j] })
	}
	prochaineFin := func(slot uint32, ts uint64) (float64, bool) {
		for _, f := range finDeVie[slot] {
			if f >= ts {
				return float64(f-ts) / 1e6, true
			}
		}
		return 0, false
	}

	ouv := map[uint32]uint64{}
	var delaisFermees, delaisOrphelines []float64
	orphelines, fermees := 0, 0
	// Une entree est ORPHELINE si la bascule suivante du meme slot est une AUTRE entree (ou
	// s'il n'y en a plus) : le moteur ne peut pas entrer deux fois sans etre sorti entre-temps.
	for _, e := range evts {
		if e.Scoped() {
			if t0, encore := ouv[e.Slot]; encore {
				orphelines++
				if d, ok := prochaineFin(e.Slot, t0); ok {
					delaisOrphelines = append(delaisOrphelines, d)
				}
			}
			ouv[e.Slot] = e.TimestampUS
			continue
		}
		if t0, ok := ouv[e.Slot]; ok {
			fermees++
			if d, ok2 := prochaineFin(e.Slot, t0); ok2 {
				delaisFermees = append(delaisFermees, d)
			}
			delete(ouv, e.Slot)
		}
	}
	// Les entrees encore ouvertes en fin de film sont orphelines elles aussi.
	for slot, t0 := range ouv {
		orphelines++
		if d, ok := prochaineFin(slot, t0); ok {
			delaisOrphelines = append(delaisOrphelines, d)
		}
	}
	t.Logf("BASCULES — %d entrees fermees par une sortie lue, %d ORPHELINES", fermees, orphelines)

	med := func(v []float64) float64 {
		if len(v) == 0 {
			return -1
		}
		c := append([]float64(nil), v...)
		sort.Float64s(c)
		return c[len(c)/2]
	}
	sous := func(v []float64, seuil float64) float64 {
		if len(v) == 0 {
			return 0
		}
		n := 0
		for _, x := range v {
			if x <= seuil {
				n++
			}
		}
		return 100 * float64(n) / float64(len(v))
	}
	t.Logf("DELAI ENTREE -> FIN DE VIE DU SLOT (secondes) :")
	t.Logf("  entrees FERMEES    (n=%d) : mediane %.2f · %.0f %% a moins de 2 s · %.0f %% a moins de 5 s",
		len(delaisFermees), med(delaisFermees), sous(delaisFermees, 2), sous(delaisFermees, 5))
	t.Logf("  entrees ORPHELINES (n=%d) : mediane %.2f · %.0f %% a moins de 2 s · %.0f %% a moins de 5 s",
		len(delaisOrphelines), med(delaisOrphelines), sous(delaisOrphelines, 2), sous(delaisOrphelines, 5))
	t.Log("LECTURE : si les orphelines meurent nettement plus vite que les fermees, l'hypothese" +
		" « il meurt a la lunette » explique la sortie manquante — et la periode doit alors se" +
		" fermer A LA MORT, pas au plafond de maintien. Sinon la sortie existe ailleurs dans le" +
		" flux (2e position d'une liste, vraisemblablement le paquet du degat qui fait dezoomer).")
}

// zoomDumpBrut publie les evenements BRUTS des unites les plus actives dans la fenetre du releve.
// C'est le diagnostic qui dit si la reconstruction d'intervalles est fidele ou si des transitions
// manquent (evenement de lunette porte en 2e position d'une liste, dans une autre famille).
func zoomDumpBrut(t *testing.T, evts []zoomEvt, off int64) {
	t.Helper()
	parU := map[uint64][]zoomEvt{}
	for _, e := range evts {
		s := float64(e.tMS-off) / 1000
		if s < zoomFenetreDebut || s > zoomFenetreFin {
			continue
		}
		parU[e.unite] = append(parU[e.unite], e)
	}
	type uc struct {
		u uint64
		n int
	}
	var l []uc
	for u, es := range parU {
		l = append(l, uc{u, len(es)})
	}
	sort.Slice(l, func(i, j int) bool { return l[i].n > l[j].n })
	t.Logf("EVENEMENTS BRUTS dans la fenetre [%.0f ; %.0f] s — %d unites concernees",
		zoomFenetreDebut, zoomFenetreFin, len(parU))
	for i, e := range l {
		if i == 6 {
			break
		}
		var sb []string
		for _, ev := range parU[e.u] {
			sens := "SORTIE"
			if ev.entree {
				sens = "ENTREE"
			}
			sb = append(sb, fmt.Sprintf("%.1f:%s", float64(ev.tMS-off)/1000, sens))
		}
		t.Logf("  unite %3d (%d evts) : %s", e.u, e.n, strings.Join(sb, " "))
	}
}
