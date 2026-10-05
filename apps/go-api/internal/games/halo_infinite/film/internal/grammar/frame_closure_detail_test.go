package grammar

// frame_closure_detail_test.go — LA CARTE V2 (`frame_closure_detail.go`) : sa marche est celle de
// [FrameClosure] (meme carte, champ a champ, sur les bobines du depot), et son detail dit juste
// sur des paquets synthetiques (sortie de la vue B, eid rejete, vue C, reste, mode borne).

import (
	"os"
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// TestFrameClosureDetailleeRendLaCarteDeFrameClosure : LE GARDE-FOU DE LA RECOPIE DU PILOTAGE.
// Sur chaque bobine du ratchet qui porte un registre, la carte rendue par
// [FrameClosureDetaillee] est IDENTIQUE a celle de [FrameClosure] ; et le detail se recoupe avec
// elle (un detail par paquet, les paquets « hors cadre » et leurs records en jeu). Les deux
// branches du pilotage d un paquet a liste d evenements sont exercees sur chaque bobine qui en porte :
// au moins une liste LOCALISEE et une NON localisee ([listesDeLaBobine]).
func TestFrameClosureDetailleeRendLaCarteDeFrameClosure(t *testing.T) {
	utiles := usagesProduitDeLaTable(t)
	mesurees := 0
	for _, bo := range frameClosureBobines() {
		if _, err := os.Stat(bo.dir); err != nil {
			t.Fatalf("bobine absente (%s) : %v", bo.dir, err)
		}
		film, err := source.LoadDir(bo.dir, nil)
		if err != nil {
			t.Fatalf("LoadDir %s : %v", bo.dir, err)
		}
		if _, ok := FilmRegistryChunk(film); !ok {
			continue
		}
		want, err := FrameClosure(contexteDeBobine(film), utiles)
		if err != nil {
			t.Fatalf("FrameClosure %s : %v", bo.nom, err)
		}
		var details []PaquetDeCarte
		got, err := FrameClosureDetaillee(contexteDeBobine(film), utiles,
			func(d PaquetDeCarte) { details = append(details, d) })
		if err != nil {
			t.Fatalf("FrameClosureDetaillee %s : %v", bo.nom, err)
		}
		if !reflect.DeepEqual(got, want) {
			t.Fatalf("%s : la carte detaillee differe de FrameClosure\n detaillee %+v\n reference %+v",
				bo.nom, got, want)
		}
		recouperLeDetail(t, bo.nom, got, details)
		loc, nonLoc := listesDeLaBobine(details)
		t.Logf("%s : %d liste(s) d evenements localisee(s), %d non localisee(s)", bo.nom, loc, nonLoc)
		if want.Paquets > 0 && (loc == 0 || nonLoc == 0) {
			t.Errorf("%s : %d liste(s) localisee(s), %d non localisee(s) : les deux branches du pilotage doivent etre exercees", bo.nom, loc, nonLoc)
		}
		if want.Paquets > 0 {
			mesurees++
		}
	}
	if mesurees < 2 {
		t.Fatalf("%d bobine(s) a trames delta mesuree(s), attendu au moins 2 (ks_000d5950, ks_e5adf7b2)", mesurees)
	}
}

// recouperLeDetail verifie que le detail rend les memes comptes que la carte.
func recouperLeDetail(t *testing.T, nom string, r FrameClosureReport, details []PaquetDeCarte) {
	t.Helper()
	if len(details) != r.Paquets {
		t.Errorf("%s : %d details pour %d paquets", nom, len(details), r.Paquets)
	}
	fermes, horsCadre, enJeuHorsCadre := 0, 0, 0
	for _, d := range details {
		if d.Fermee {
			fermes++
		}
		if d.VueCAtteinte && d.Sortie.EstUnRejet() != (d.Invariant == InvariantSortieParRejet) {
			t.Errorf("%s : chunk %d paquet %d, sortie %q et regle %q en desaccord", nom, d.Chunk, d.Index, d.Sortie, d.Invariant)
		}
		if d.Sortie == SortieVueBAutre {
			t.Errorf("%s : chunk %d paquet %d, sortie de vue B « autre »", nom, d.Chunk, d.Index)
		}
		if d.Cause != CauseHorsCadre {
			continue
		}
		horsCadre++
		enJeuHorsCadre += d.UtilesEnJeu
		if !d.VueCAtteinte || !d.VueC.Porte {
			t.Errorf("%s : paquet hors cadre sans vue C lue jusqu a son terminateur : %+v", nom, d)
		}
		if d.Sortie != SortieVueBTerminateur && !d.Sortie.EstUnRejet() {
			t.Errorf("%s : paquet hors cadre sorti de la vue B par %q", nom, d.Sortie)
		}
	}
	b := r.Bloquants[CauseHorsCadre]
	if fermes != r.PaquetsFermes || horsCadre != b.Paquets || enJeuHorsCadre != b.UtilesEnJeu {
		t.Errorf("%s : detail fermes %d / hors cadre %d / en jeu %d, carte %d / %d / %d", nom,
			fermes, horsCadre, enJeuHorsCadre, r.PaquetsFermes, b.Paquets, b.UtilesEnJeu)
	}
}

// marcherPaquetDetaille marche UN payload depuis `debut` par la marche par rangs
// ([lireTrameParRangs]), le classe et en remplit le detail, comme
// [marcheDetaillee.detaillerLaTrame].
func (md *marcheDetaillee) marcherPaquetDetaille(pay []byte, w *World, debut int, d *PaquetDeCarte) {
	avant := compteDesAnticipations(md.cfg.Obs)
	br := LecteurSur(pay)
	br.poserCadre(md.cfg)
	var l lectureDeTrame
	lireTrameParRangs(br, pay, w, md.cfg, departDeTrame{bit: debut}, &l)
	md.detaillerLaMarche(&l, debut, pay, d)
	d.Anticipations = compteDesAnticipations(md.cfg.Obs) - avant
}

// marcherUnDetaille marche un payload depuis la tete du paquet et rend son detail.
func marcherUnDetaille(t *testing.T, w *World, pay []byte) PaquetDeCarte {
	t.Helper()
	md := marcheDetaillee{mesureDesTrames: nouvelleMesureDesTrames(w.Reg, nil, cadreDeCarte())}
	d := PaquetDeCarte{Bits: len(pay) * 8, DebutVueB: -1, FinVueB: -1}
	md.marcherPaquetDetaille(pay, w, DefaultPacketPreambleBits, &d)
	return d
}

// enteteDelta13 ecrit l EN-TETE seul d un record DELTA (prefixe, slot de 13 bits, tag 1).
func (w *bitWriter) enteteDelta13(slot uint32) {
	w.bit(1)
	w.bits(uint64(slot), 13)
	w.bits(1, 2)
}

// TestFrameClosureDetaillee_SortieParTerminateur : une vue B close par son terminateur, une vue C
// a une entree ; le paquet ferme, puis, suivi d un octet non nul, il est « hors cadre » avec un
// reste de huit bits.
func TestFrameClosureDetaillee_SortieParTerminateur(t *testing.T) {
	var bw bitWriter
	bw.teteDePaquet()
	bw.deltaMasque13(123)
	bw.finDeVueB()
	bw.vueCUneEntree()
	ferme := append([]byte(nil), bw.buf...)

	d := marcherUnDetaille(t, mondeDeCarte(), ferme)
	if !d.Fermee || d.Cause != "" || d.Sortie != SortieVueBTerminateur || d.RecordsLus != 1 {
		t.Fatalf("paquet ferme : %+v, attendu ferme, sortie par terminateur, un record", d)
	}
	if !d.VueCAtteinte || d.VueC.Vide || len(d.VueC.Entrees) != 1 || d.DernierLu != "DELTA ti=4 sans composant" {
		t.Errorf("vue C %+v, dernier lu %q : attendu une entree, DELTA ti=4 sans composant", d.VueC, d.DernierLu)
	}
	d = marcherUnDetaille(t, mondeDeCarte(), append(append([]byte(nil), ferme...), 0xff))
	if d.Fermee || d.Cause != CauseHorsCadre || d.Sortie != SortieVueBTerminateur {
		t.Fatalf("octet de trop : %+v, attendu hors cadre apres un terminateur", d)
	}
	if reste := d.Bits - d.Curseur; reste < 8 || reste > 15 {
		t.Errorf("reste %d bit(s), attendu de 8 a 15 (l octet de trop et le bourrage)", reste)
	}
}

// TestFrameClosureDetaillee_SortieParRejetHorsDatum : un en-tete DELTA d un slot que le monde ne
// lie pas arrete la vue B (garde vive) ; le curseur reste a la fin de l en-tete, l eid COMPLET est
// relu, et la vue C qui suit est lue de la. Le paquet se ferme au bit pres mais n est PAS ferme :
// `FUN_142f2e174` n ecrit un DELTA que pour une entite que le lecteur a deja (sortie par rejet,
// `ecrivain_invariants.go`). MUTATION : relire l eid un bit trop tot — l eid rendu change, ROUGE ;
// retirer la regle de la sortie par rejet — le paquet est ferme, ROUGE.
func TestFrameClosureDetaillee_SortieParRejetHorsDatum(t *testing.T) {
	var bw bitWriter
	bw.teteDePaquet()
	bw.deltaMasque13(123)
	bw.enteteDelta13(500)
	bw.bit(0) // la vue C : son terminateur seul
	d := marcherUnDetaille(t, mondeDeCarte(), bw.buf)
	if d.Sortie != SortieVueBRejetHorsDatum || d.EIDRejete != 1<<30|500 {
		t.Fatalf("sortie %q, eid %#x : attendu rejet hors datum de 0x%x", d.Sortie, d.EIDRejete, uint32(1<<30|500))
	}
	if d.Fermee || !d.FermeeAuBit || !d.VueC.Vide || d.FinVueB != 2+(1+13+2+1+1+3)+(1+13+2) {
		t.Errorf("detail %+v : attendu ferme au bit pres, non ferme, vue C vide, fin de vue B a la fin de l en-tete rejete", d)
	}
	if d.Invariant != InvariantSortieParRejet || d.Cause != InvariantSortieParRejet.String() {
		t.Errorf("regle %s, cause %q : attendu la sortie par rejet", d.Invariant, d.Cause)
	}
}

// TestFrameClosureDetaillee_ModeBorne : un record dont le corps lit au-dela du payload est compte
// debordant, avec son composant.
func TestFrameClosureDetaillee_ModeBorne(t *testing.T) {
	var bw bitWriter
	bw.teteDePaquet()
	bw.deltaMasque13(123, 0)
	w := mondeDeCarte("object-scale-component")
	d := marcherUnDetaille(t, w, bw.buf)
	if d.RecordsLus != 1 || d.RecordsDebordants != 1 || d.ComposantsDebordants != 1 {
		t.Fatalf("detail %+v : attendu un record et un composant debordants", d)
	}
	if d.Fermee || d.Sortie != SortieVueBOuverte {
		t.Errorf("detail %+v : attendu un paquet ouvert, non ferme", d)
	}
}

// listesDeLaBobine compte les paquets a liste d evenements localisee et non localisee d un detail.
func listesDeLaBobine(details []PaquetDeCarte) (localisees, nonLocalisees int) {
	for _, d := range details {
		switch {
		case d.ListeLocalisee:
			localisees++
		case d.ListeNonLocalisee:
			nonLocalisees++
		}
	}
	return localisees, nonLocalisees
}
