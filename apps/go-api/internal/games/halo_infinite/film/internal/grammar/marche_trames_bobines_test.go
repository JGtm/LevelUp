package grammar

// marche_trames_bobines_test.go — LA MARCHE DES TRAMES SUR LES BOBINES DU DEPOT : chaque paquet
// tient les invariants de la structure de lecture (ADR 0037 IR-1, IR-4, IR-6), la marche rend les
// memes comptes que ses consommateurs, et elle s arrete quand son consommateur s arrete.

import (
	"errors"
	"fmt"
	"os"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// TestLaStructureDesBobinesTientSesInvariants : sur chaque bobine du depot qui porte des trames
// delta sous un registre, chaque paquet de la marche tient les invariants de la structure (records
// et composants contigus et emboites dans leur vue, tours de vue C contigus, verdict accorde a la
// vue C et aux preuves), et la marche rend les memes comptes que la carte de fermeture et que les
// etats de mouvement, qui la consomment.
func TestLaStructureDesBobinesTientSesInvariants(t *testing.T) {
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
		paquets, fermes, nonLocalises := 0, 0, 0
		for p, err := range contexteDeBobine(film).Trames(nil) {
			if err != nil {
				t.Fatalf("%s : %v", bo.nom, err)
			}
			paquets++
			if p.Fermeture.Verdict == lecture.VerdictFerme {
				fermes++
			}
			if p.Debut == lecture.DebutNonLocalise {
				nonLocalises++
			}
			verifierLePaquet(t, bo.nom, p)
		}
		carte, err := FrameClosure(contexteDeBobine(film), nil)
		if err != nil {
			t.Fatalf("FrameClosure %s : %v", bo.nom, err)
		}
		if paquets != carte.Paquets || fermes != carte.PaquetsFermes {
			t.Errorf("%s : la marche rend %d paquets dont %d fermes, la carte %d dont %d", bo.nom,
				paquets, fermes, carte.Paquets, carte.PaquetsFermes)
		}
		verifierLesEtatsDeMouvement(t, bo.nom, film, paquets, nonLocalises)
		if paquets > 0 {
			mesurees++
		}
	}
	if mesurees < 2 {
		t.Fatalf("%d bobine(s) a trames delta, attendu au moins 2 (ks_000d5950, ks_e5adf7b2)", mesurees)
	}
}

// verifierLesEtatsDeMouvement : les etats de mouvement consomment la meme marche — chaque paquet
// rendu est compte, marche ou liste non localisee.
func verifierLesEtatsDeMouvement(t *testing.T, nom string, film *source.Film, paquets, nonLocalises int) {
	t.Helper()
	m, err := ScanMarcheDesTrames(contexteDeBobine(film))
	if err != nil {
		t.Fatalf("ScanMarcheDesTrames %s : %v", nom, err)
	}
	st := m.MovementStateStats
	if st.Absent {
		return // les compteurs de marche sont remis a zero pour un film sans etats de mouvement
	}
	if st.Packets+st.EventPacketsUnlocated != paquets || st.EventPacketsUnlocated != nonLocalises {
		t.Errorf("%s : les etats de mouvement comptent %d paquets marches et %d listes non localisees, "+
			"la marche %d paquets dont %d non localises", nom, st.Packets, st.EventPacketsUnlocated, paquets, nonLocalises)
	}
}

// verifierLePaquet verifie les invariants d UN paquet de la structure.
func verifierLePaquet(t *testing.T, nom string, p *lecture.Paquet) {
	t.Helper()
	ou := func() string { return fmt.Sprintf("%s chunk %d paquet %d", nom, p.Chunk, p.Index) }
	f := p.Fermeture
	if f.Longueur != uint32(len(p.Payload)*8) {
		t.Errorf("%s : longueur %d pour un payload de %d octets", ou(), f.Longueur, len(p.Payload))
	}
	if p.Debut == lecture.DebutNonLocalise {
		if f.Verdict != lecture.VerdictQueueOpaque || f.Queue.Cause != lecture.CauseListeNonLocalisee || len(p.Records) != 0 {
			t.Errorf("%s : liste non localisee rendue %+v avec %d record(s)", ou(), f, len(p.Records))
		}
		return
	}
	verifierLesRecords(t, ou, p)
	verifierLaVueC(t, ou, p)
	switch f.Verdict {
	case lecture.VerdictFerme:
		if p.VueC.Etat != lecture.VueTerminee || p.VueB.Sortie != lecture.SortieTerminateur || f.Longueur-f.Consommes >= 8 {
			t.Errorf("%s : ferme avec la vue B %+v, la vue C %d, %d bits consommes sur %d", ou(), p.VueB,
				p.VueC.Etat, f.Consommes, f.Longueur)
		}
	case lecture.VerdictRefuse:
		if p.VueC.Etat != lecture.VueTerminee || f.Queue != (lecture.QueueOpaque{}) {
			t.Errorf("%s : refuse avec la vue C %d et la queue %+v", ou(), p.VueC.Etat, f.Queue)
		}
	case lecture.VerdictQueueOpaque:
		if f.Queue.Cause == lecture.CauseAucune || f.Queue.Debut > f.Consommes {
			t.Errorf("%s : queue opaque %+v, %d bits consommes", ou(), f.Queue, f.Consommes)
		}
	default:
		t.Errorf("%s : verdict non rendu", ou())
	}
	for i, r := range p.Records {
		ferme := f.Verdict == lecture.VerdictFerme
		if (r.Preuve == lecture.PreuveFerme) != ferme || r.Preuve == lecture.PreuveNonRenseignee {
			t.Errorf("%s : record %d preuve %d dans un paquet de verdict %d", ou(), i, r.Preuve, f.Verdict)
		}
	}
}

// verifierLesRecords : les records se suivent sans trou depuis le debut de la vue B et finissent
// dans elle ; les composants d un record sont dans son etendue, dans l ordre de l arene, contigus ;
// un composant infranchissable est le dernier de son record et en porte la desynchronisation.
func verifierLesRecords(t *testing.T, ou func() string, p *lecture.Paquet) {
	t.Helper()
	pos, arene := p.VueB.Debut, uint32(0)
	for i, r := range p.Records {
		if r.Debut != pos {
			t.Errorf("%s : record %d commence au bit %d, attendu %d", ou(), i, r.Debut, pos)
		}
		pos = r.Debut + r.Bits
		if r.Comps[0] != arene || r.Comps[1] < r.Comps[0] {
			t.Errorf("%s : record %d, composants %v hors de l ordre de l arene (%d)", ou(), i, r.Comps, arene)
			return
		}
		arene = r.Comps[1]
		cs := p.Comps[r.Comps[0]:r.Comps[1]]
		for k, c := range cs {
			if c.Debut < r.Debut || c.Debut+c.Bits > pos || (k+1 < len(cs) && c.Debut+c.Bits != cs[k+1].Debut) {
				t.Errorf("%s : record %d [%d, %d), composant %+v hors de son etendue ou non contigu", ou(), i, r.Debut, pos, c)
			}
			if c.Etat == lecture.EtatInfranchissable && (k+1 != len(cs) || r.Desync != int16(c.Index)) {
				t.Errorf("%s : record %d, composant infranchissable %+v qui n arrete pas le record (desync %d)", ou(), i, c, r.Desync)
			}
		}
	}
	if fin := p.VueB.Debut + p.VueB.Bits; len(p.Records) > 0 && pos > fin {
		t.Errorf("%s : les records finissent au bit %d, apres la vue B [%d, %d)", ou(), pos, p.VueB.Debut, fin)
	}
	if arene != uint32(len(p.Comps)) {
		t.Errorf("%s : %d composant(s) dans l arene, %d rattache(s) a un record", ou(), len(p.Comps), arene)
	}
}

// verifierLaVueC : les tours de la vue C se suivent sans trou depuis son debut ; une vue terminee
// finit un bit apres son dernier tour (le terminateur).
func verifierLaVueC(t *testing.T, ou func() string, p *lecture.Paquet) {
	t.Helper()
	c := p.VueC
	if c.Etat == lecture.VueNonLue {
		return
	}
	pos := c.Debut
	for k, e := range c.Entrees {
		if e.Debut != pos {
			t.Errorf("%s : tour %d de la vue C au bit %d, attendu %d", ou(), k, e.Debut, pos)
		}
		pos = e.Debut + e.Bits
	}
	if c.Etat == lecture.VueTerminee && pos+1 != c.Debut+c.Bits {
		t.Errorf("%s : vue C [%d, %d) terminee, son dernier tour finit au bit %d", ou(), c.Debut, c.Debut+c.Bits, pos)
	}
}

// TestLaMarcheDesTramesSArreteQuandOnLeLuiDit : la boucle du consommateur qui s arrete arrete la
// marche ; un film sans chunk de donnees rend son erreur, une fois.
func TestLaMarcheDesTramesSArreteQuandOnLeLuiDit(t *testing.T) {
	film, err := source.LoadDir(frameClosureBobines()[len(frameClosureBobines())-1].dir, nil)
	if err != nil {
		t.Fatalf("LoadDir : %v", err)
	}
	n := 0
	for _, err := range contexteDeBobine(film).Trames(nil) {
		if err != nil {
			t.Fatalf("marche : %v", err)
		}
		n++
		if n == 3 {
			break
		}
	}
	if n != 3 {
		t.Fatalf("%d paquet(s) rendu(s) avant l arret, attendu 3", n)
	}
	rendus := 0
	for p, err := range NewFilmContext(&source.Film{}).Trames(nil) {
		rendus++
		if p != nil || !errors.Is(err, ErrNoFilmChunk) {
			t.Errorf("film vide : paquet %v, erreur %v, attendu %v", p, err, ErrNoFilmChunk)
		}
	}
	if rendus != 1 {
		t.Errorf("film vide : %d tour(s), attendu 1", rendus)
	}
}
