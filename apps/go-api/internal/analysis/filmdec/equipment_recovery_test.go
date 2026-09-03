package filmdec

// equipment_recovery_test.go — les règles qui DÉCIDENT de la récupération gatée : la
// construction des fenêtres, le témoin de compteur (acceptation/refus), et l'ordre de bits
// du masque dense figé par P1.0. Le balayage lui-même exige un film (non versionné) : il est
// couvert par le test gaté sur pièces (transloc_exemption_film_test.go) et par la validation
// Dynasty du plan.

import (
	"reflect"
	"testing"
)

// emiss fabrique une émission stricte minimale pour les fenêtres.
func emiss(slot uint32, ts uint64, chunk int, counter uint32) abilityEmission {
	return abilityEmission{Slot: slot, TimestampUS: ts, Chunk: chunk, Counter: counter}
}

func TestFenetresDeRecuperationSurLesSautsDeCompteur(t *testing.T) {
	bySlot := map[uint32][]abilityEmission{
		535: {emiss(535, 1000, 8, 5), emiss(535, 5000, 10, 7)}, // saut c5 -> c7 : 1 manquante
		600: {emiss(600, 2000, 3, 5), emiss(600, 2500, 3, 6)},  // chaîne saine : rien
		601: {emiss(601, 100, 1, 6), emiss(601, 900, 2, 1)},    // saut c6 -> c1 : 2 manquantes
		602: {emiss(602, 400, 2, 7)},                           // tête hors norme : fenêtre de tête
		603: {emiss(603, 50, 1, 7)},                            // tête hors norme, naissance APRÈS : refusée
	}
	born := func(slot uint32) (uint64, bool) {
		switch slot {
		case 602:
			return 100, true
		case 603:
			return 60, true // naissance postérieure à la première émission : fenêtre vide
		}
		return 0, false
	}
	wins := buildEquipRecoveryWindows(bySlot, born)
	if len(wins) != 3 {
		t.Fatalf("%d fenêtre(s), attendu 3 (deux sauts + une tête) : %+v", len(wins), wins)
	}
	byKey := map[uint32]equipRecoveryWindow{}
	for _, w := range wins {
		byKey[w.slot] = w
	}
	if w := byKey[535]; w.miss != 1 || w.fromC != 5 || w.toC != 7 || w.tsMin != 1000 ||
		w.tsMax != 5000 || w.chunkMin != 8 || w.chunkMax != 10 || w.head {
		t.Errorf("fenêtre 535 inattendue : %+v", w)
	}
	if w := byKey[601]; w.miss != 2 || w.fromC != 6 || w.toC != 1 {
		t.Errorf("fenêtre 601 inattendue (le saut par-dessus le repli compte modulo 8) : %+v", w)
	}
	if w := byKey[602]; !w.head || w.fromC != equipRecoveryHeadCounter || w.miss != 2 ||
		w.tsMin != 100 || w.chunkMin != 1 || w.chunkMax != 2 {
		t.Errorf("fenêtre de tête 602 inattendue (première c7 : {c5, c6} manquent) : %+v", w)
	}
	// Sans témoin de naissance, AUCUNE fenêtre de tête n'est bâtie : la récupération de tête
	// exige le témoin, comme le classement des réapparitions.
	if got := buildEquipRecoveryWindows(map[uint32][]abilityEmission{
		602: bySlot[602],
	}, nil); len(got) != 0 {
		t.Errorf("fenêtre de tête bâtie SANS témoin de naissance : %+v", got)
	}
}

// recCand fabrique un candidat retrouvé.
func recCand(slot uint32, ts uint64, counter uint32, rank, off int) equipRecovered {
	return equipRecovered{
		abilityEmission: abilityEmission{Slot: slot, TimestampUS: ts, Counter: counter, Rank: rank},
		off:             off,
	}
}

func TestTemoinDeCompteurAccepteLeComblementExact(t *testing.T) {
	w := equipRecoveryWindow{slot: 535, fromC: 5, toC: 7, miss: 1, tsMin: 0, tsMax: 10_000}
	w.cands = []equipRecovered{recCand(535, 4000, 6, 11, 0)}
	got := acceptEquipRecovery(&w)
	if len(got) != 1 || got[0].Counter != 6 || got[0].Rank != 11 {
		t.Fatalf("le candidat au compteur prédit doit être accepté : %+v", got)
	}
}

func TestTemoinDeCompteurRefuseLeBruit(t *testing.T) {
	cases := []struct {
		nom   string
		w     equipRecoveryWindow
		cands []equipRecovered
		want  int // émissions acceptées
	}{
		{
			nom:   "hors prédiction : ignoré",
			w:     equipRecoveryWindow{slot: 5, fromC: 5, toC: 7, miss: 1},
			cands: []equipRecovered{recCand(5, 100, 3, 9, 0)},
			want:  0,
		},
		{
			nom: "deux candidats pour le même compteur prédit : fenêtre rejetée entière",
			w:   equipRecoveryWindow{slot: 5, fromC: 5, toC: 7, miss: 1},
			cands: []equipRecovered{
				recCand(5, 100, 6, 9, 0), recCand(5, 200, 6, 11, 4),
			},
			want: 0,
		},
		{
			nom: "ordre temporel contre ordre des compteurs : rejeté",
			w:   equipRecoveryWindow{slot: 5, fromC: 5, toC: 0, miss: 2},
			cands: []equipRecovered{
				recCand(5, 100, 7, 9, 0), recCand(5, 200, 6, 11, 4),
			},
			want: 0,
		},
		{
			nom: "comblement du milieu seul : éclaterait le saut en deux, refusé",
			w:   equipRecoveryWindow{slot: 5, fromC: 5, toC: 1, miss: 3}, // prédits {6, 7, 0}
			cands: []equipRecovered{
				recCand(5, 100, 7, 9, 0),
			},
			want: 0,
		},
		{
			nom: "préfixe partiel : accepté, le trou résiduel reste UN saut",
			w:   equipRecoveryWindow{slot: 5, fromC: 5, toC: 1, miss: 3},
			cands: []equipRecovered{
				recCand(5, 100, 6, 9, 0),
			},
			want: 1,
		},
		{
			nom: "préfixe + suffixe : accepté, un seul trou au milieu",
			w:   equipRecoveryWindow{slot: 5, fromC: 5, toC: 1, miss: 3},
			cands: []equipRecovered{
				recCand(5, 100, 6, 9, 0), recCand(5, 300, 0, AbilitySetNoRank, 8),
			},
			want: 2,
		},
		{
			nom: "comblement complet d'un double manque",
			w:   equipRecoveryWindow{slot: 5, fromC: 6, toC: 1, miss: 2}, // prédits {7, 0}
			cands: []equipRecovered{
				recCand(5, 100, 7, 9, 0), recCand(5, 300, 0, AbilitySetNoRank, 8),
			},
			want: 2,
		},
		{
			nom: "tête de vie : le suffixe ancré sur la première émission est accepté",
			w: equipRecoveryWindow{
				slot: 5, fromC: equipRecoveryHeadCounter, toC: 7, miss: 2, head: true,
			}, // prédits {5, 6}
			cands: []equipRecovered{recCand(5, 100, 6, 11, 0)},
			want:  1,
		},
	}
	for _, c := range cases {
		t.Run(c.nom, func(t *testing.T) {
			w := c.w
			w.tsMax = 10_000
			w.cands = c.cands
			if got := acceptEquipRecovery(&w); len(got) != c.want {
				t.Errorf("%d émission(s) acceptée(s), attendu %d : %+v", len(got), c.want, got)
			}
		})
	}
}

// TestRecuperationBorneEnQueueDePaquet — REPRO DE LA REVUE (ronde 1, F1 — P0) : un préfixe
// valide en QUEUE de paquet. La boucle d'appel ne garantit que l'en-tête et UN index
// (p+27 <= total), mais un comptage à 7 fait lire les indices jusqu'au bit p+62 — et
// readBitsAt indexe le tampon SANS filet : sans la garde de borne, panic hors limites.
func TestRecuperationBorneEnQueueDePaquet(t *testing.T) {
	pay := make([]byte, 4)     // 32 bits : l'en-tête (21 bits) tient, les 7 index de 6 bits non
	ecrisBits(pay, 0, 1, 1)    // préfixe
	ecrisBits(pay, 1, 13, 535) // slot
	ecrisBits(pay, 14, 2, 1)   // tag = 1 ; bits 16-17 restent 0 (bit16 nul, forme « sans i0 »)
	ecrisBits(pay, 18, 3, 7)   // comptage 7 : la lecture des indices irait jusqu'au bit 62
	last, restore := equipRecoveryHook()
	defer restore()
	if _, _, ok := walkEquipRecoveryAt(abilityScanSetup{}, pay, 0, len(pay)*8, last); ok {
		t.Fatal("un record tronqué en queue de paquet a été accepté")
	}
	// La forme DENSE se borne déjà (i0 + TotalBits > total) : un motif à porte de masque
	// levée en queue de paquet est rejeté par la même sortie, sans lire le masque.
	dense := make([]byte, 4)
	ecrisBits(dense, 0, 1, 1)
	ecrisBits(dense, 1, 13, 535)
	ecrisBits(dense, 14, 2, 1)
	ecrisBits(dense, 17, 1, 1) // porte du masque : R(64) dense, hors du tampon
	if _, _, ok := walkEquipRecoveryAt(abilityScanSetup{}, dense, 0, len(dense)*8, last); ok {
		t.Fatal("un record dense tronqué en queue de paquet a été accepté")
	}
}

// TestMasqueDenseOrdreDeBitsFige verrouille l'ordre mesuré par P1.0 : bit k du flux =
// composant 63−k. Le motif d'essai lève les bits 63, 62, 46, 15 et 5 — qui doivent rendre
// les composants {0, 1, 17, 48, 58}, la famille structurelle des vrais manques.
func TestMasqueDenseOrdreDeBitsFige(t *testing.T) {
	pay := make([]byte, 9)
	for _, comp := range []int{0, 1, 17, 48, 58} {
		bit := 63 - comp // bit k du flux = composant 63−k
		pay[bit/8] |= 1 << (7 - uint(bit%8))
	}
	got := denseMaskIndices(pay, 0)
	if want := []int{0, 1, 17, 48, 58}; !reflect.DeepEqual(got, want) {
		t.Fatalf("masque dense lu %v, attendu %v — l'ordre de bits figé par P1.0 a bougé", got, want)
	}
}

// ecrisBits pose v sur n bits MSB-first à la position at — l'inverse de readBitsAt, pour
// fabriquer des motifs d'essai lisibles.
func ecrisBits(pay []byte, at, n int, v uint32) {
	for i := 0; i < n; i++ {
		if v>>(uint(n-1-i))&1 == 1 {
			pay[(at+i)/8] |= 1 << (7 - uint((at+i)%8))
		}
	}
}

func TestIndicesCroissantsSansI0(t *testing.T) {
	// Deux indices de 6 bits : 17 puis 48 — croissants, premier != 0 : la forme « sans i0 »
	// telle que le masque [17 48] du saut c5 -> c7 de 01e1f945 la porte (rapport R2 §3).
	pay := make([]byte, 2)
	ecrisBits(pay, 0, 6, 17)
	ecrisBits(pay, 6, 6, 48)
	got, ok := ascendingIndices(pay, 0, 2)
	if !ok || !reflect.DeepEqual(got, []int{17, 48}) {
		t.Fatalf("indices lus %v (ok=%v), attendu [17 48]", got, ok)
	}
	// Une liste NON croissante est refusée : la croissance stricte est l'ancre anti-bruit
	// que la forme « sans i0 » CONSERVE.
	desc := make([]byte, 2)
	ecrisBits(desc, 0, 6, 48)
	ecrisBits(desc, 6, 6, 17)
	if _, ok := ascendingIndices(desc, 0, 2); ok {
		t.Fatal("une liste d'indices décroissante a été acceptée")
	}
}

func TestCounterGap(t *testing.T) {
	cases := []struct {
		from, to uint32
		want     int
	}{
		{5, 6, 0}, {7, 0, 0}, {5, 7, 1}, {7, 1, 1}, {1, 5, 3}, {5, 5, 0},
	}
	for _, c := range cases {
		if got := counterGap(c.from, c.to); got != c.want {
			t.Errorf("counterGap(c%d -> c%d) = %d, attendu %d", c.from, c.to, got, c.want)
		}
	}
}
