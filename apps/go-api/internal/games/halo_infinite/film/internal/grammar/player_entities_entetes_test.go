package grammar

import "testing"

// player_entities_entetes_test.go — L'ABSENCE D'UN JOUEUR SE PROUVE PAR L'EN-TETE EXACT, QUEL QUE
// SOIT LE CHEMIN PAR LEQUEL LA MARCHE L'A PERDU (constat DFIX-R2 de la revue adverse du lot D-fix).
//
// Trois payloads synthetiques, un par chemin que la premiere version du lot laissait MUET : le
// joueur gere D (slot 15, `ti 9`) est dans l'image-cle, la marche ne le lit pas, et sa liste
// d'ecartes ne le porte pas. ROUGE AVANT (regle « ecartes + illisibles ») : aucun doute, l'absence
// de D passait pour prouvee. VERT APRES : l'en-tete exact de D est trouve, le doute se note.

const (
	enteteSlotJoueur = 15 // le joueur gere que la marche perd
	enteteTIObjet    = 3  // l'archetype des records qui l'entourent
)

// kfSentinelles ferme la table.
func kfSentinelles(w *bitWriter) {
	for i := 0; i < 2100; i++ {
		w.bits(kfSent, 32)
	}
}

// payloadSautSurUneFausseAncre : A(10) et B(11) de meme largeur apprennent la largeur de leur
// archetype ; C(12) est plus long, et une fausse ancre (slot 20) se trouve EXACTEMENT a la largeur
// apprise dans son corps : le saut la prend, et D, qui suit C, est perdu.
func payloadSautSurUneFausseAncre() []byte {
	w := &bitWriter{}
	w.bit(0)
	kfEcrireRecord(w, 1, 10, enteteTIObjet, 100)
	kfEcrireRecord(w, 1, 11, enteteTIObjet, 100)
	// C : 64 bits d'en-tete puis 44 + 32 bits jusqu'a son corps ; la largeur apprise est 176
	// (44 + 32 + 100) — la fausse ancre est posee a 100 bits dans le corps de C.
	kfEcrireRecord(w, 1, 12, enteteTIObjet, 100)
	w.bits(1<<30|20, 32)
	w.bits(enteteTIObjet, 32)
	w.bits(0, 236)
	kfEcrireRecord(w, 1, enteteSlotJoueur, managedPlayerTypeIndex, 300)
	kfSentinelles(w)
	return w.buf
}

// payloadFauxVoisin : dans le corps de D, un faux voisin (slot 11, generation 1) du record A :
// la recherche rend le premier `slot+1` de la fenetre et enjambe D, candidat vu avant lui.
func payloadFauxVoisin() []byte {
	w := &bitWriter{}
	w.bit(0)
	kfEcrireRecord(w, 1, 10, enteteTIObjet, 100)
	kfEcrireRecord(w, 1, enteteSlotJoueur, managedPlayerTypeIndex, 100)
	w.bits(1<<30|11, 32)
	w.bits(enteteTIObjet, 32)
	w.bits(0, 200)
	kfSentinelles(w)
	return w.buf
}

// payloadElectionHorsFenetre : apres A, la fenetre (reduite ici a 5 000 bits, plus que les 2 048
// sentinelles qui closent la table) ne porte qu'une fausse ancre de generation 2 (slot 50) ; D est
// au-dela. L'election prend la fausse ancre, et D (slot 15 < 50) n'est plus jamais un candidat.
func payloadElectionHorsFenetre() []byte {
	w := &bitWriter{}
	w.bit(0)
	kfEcrireRecord(w, 1, 10, enteteTIObjet, 100)
	w.bits(0, 300)
	w.bits(2<<30|50, 32)
	w.bits(enteteTIObjet, 32)
	w.bits(0, 8000)
	kfEcrireRecord(w, 1, enteteSlotJoueur, managedPlayerTypeIndex, 100)
	kfSentinelles(w)
	return w.buf
}

// TestLAbsenceDUnJoueurPerduParLaMarcheNEstPasProuvee : les trois chemins.
func TestLAbsenceDUnJoueurPerduParLaMarcheNEstPasProuvee(t *testing.T) {
	cas := []struct {
		nom   string
		pay   []byte
		fen   int
		decis func(KeyframeWalkStats) bool
	}{
		{"saut de largeur", payloadSautSurUneFausseAncre(), kfScanFenetreBits,
			func(s KeyframeWalkStats) bool { return s.Sauts == 1 }},
		{"faux voisin", payloadFauxVoisin(), kfScanFenetreBits,
			func(s KeyframeWalkStats) bool { return s.Voisins == 1 && s.Elections == 0 }},
		{"election hors fenetre", payloadElectionHorsFenetre(), 5000,
			func(s KeyframeWalkStats) bool { return s.Elections >= 1 && s.Voisins == 0 && s.Sauts == 0 }},
	}
	for _, c := range cas {
		mp := marcherLaTable(&kfRecherche{buf: c.pay, total: len(c.pay) * 8, maxWin: c.fen})
		lus := map[int]bool{}
		for _, r := range mp.Records {
			if r.Slot == enteteSlotJoueur {
				t.Fatalf("%s : la marche lit D (%+v) — le temoin ne mord plus", c.nom, mp.Records)
			}
			lus[r.Slot] = true
		}
		if !c.decis(mp.Stats) {
			t.Fatalf("%s : la marche n'a pas pris le chemin attendu (%+v, records %+v)", c.nom, mp.Stats, mp.Records)
		}
		for _, e := range mp.Ecartes {
			if e.Slot == enteteSlotJoueur {
				t.Fatalf("%s : D est dans les ecartes — ce chemin n'est pas muet", c.nom)
			}
		}
		a := nouvelAccumulateurDEntites()
		rang := a.ouvrirImageCle(100)
		a.noter(rang, 30, 0, 0) // une autre entite, lue : l'image-cle est porteuse
		a.ouvrirImageCle(200)
		a.noter(1, enteteSlotJoueur, 1, 0) // D lu a l'image-cle suivante : c'est une entite
		a.douterDe(rang, lus, slotsDEntetesExacts(c.pay))
		s := a.publier()
		if s.AbsenceProuvee(enteteSlotJoueur, rang) {
			t.Errorf("%s : l'absence de D a l'image-cle qui le porte passe pour PROUVEE", c.nom)
		}
		for _, e := range s.Entities {
			if e.Slot == enteteSlotJoueur && !s.AtStart(e) {
				t.Errorf("%s : D lu comme ARRIVE plus tard alors que l'image-cle le porte", c.nom)
			}
		}
	}
}

// TestLAbsenceSansEnTeteEstProuvee : le meme payload SANS le record de D prouve son absence — la
// regle ne fabrique pas de doute.
func TestLAbsenceSansEnTeteEstProuvee(t *testing.T) {
	w := &bitWriter{}
	w.bit(0)
	kfEcrireRecord(w, 1, 10, enteteTIObjet, 100)
	kfEcrireRecord(w, 1, 11, managedPlayerTypeIndex, 100)
	kfSentinelles(w)
	got := slotsDEntetesExacts(w.buf)
	if len(got) != 1 || !got[11] {
		t.Fatalf("en-tetes ti=9 %v, attendu {11}", got)
	}
}

// TestLesRecordsTi9DeLaMarcheOntUnEnTeteExact : l'invariant qui fait de la recherche un SUR-ENSEMBLE
// de ce que la marche lit — tout record `ti=9` de la marche, lisible ou non, a un en-tete exact
// (le filtre fort de [kfAnchorFromID] est le meme motif). Sur les sept bobines par build.
func TestLesRecordsTi9DeLaMarcheOntUnEnTeteExact(t *testing.T) {
	for _, b := range bobinesEquipes() {
		fc := NewFilmContext(bobineFilm(t, b.film))
		m := fc.MarcheDImageCle()
		for _, c := range fc.ChunkNumbers() {
			raw, pks, ok := fc.ChunkAt(c)
			if !ok {
				continue
			}
			for _, pk := range pks {
				if pk.Type != PacketTypeKeyframe {
					continue
				}
				pay := pk.Payload(raw)
				entetes := slotsDEntetesExacts(pay)
				for _, r := range m.Records(pay) {
					if r.TI == managedPlayerTypeIndex && !entetes[r.Slot] {
						t.Fatalf("%s morceau %d : record ti=9 slot %d sans en-tete exact", b.film, c, r.Slot)
					}
				}
			}
		}
	}
}
