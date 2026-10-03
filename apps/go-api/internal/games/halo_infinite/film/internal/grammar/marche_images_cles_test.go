package grammar

// marche_images_cles_test.go — LA PHASE IMAGES-CLES RANGE CE QUE LA MARCHE LIT (ADR 0037 IR-1,
// IR-4) : sur les bobines par build, la fermeture que la structure porte est celle que
// [KeyframeClosure] mesure, archetype par archetype, et chaque paquet tient les invariants de la
// structure.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// bobineParBuild charge la mini-bobine par build `court` du rejeu.
func bobineParBuild(t *testing.T, court string) (string, *source.Film) {
	t.Helper()
	dir := filepath.Join("..", "..", "replay", "testdata", "minifilm_"+court)
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("bobine absente (%s) : %v", dir, err)
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	return dir, film
}

// TestLaPhaseDesImagesClesPorteLaFermetureDeKeyframeClosure : sur les sept bobines par build, les
// preuves des records de la structure rendent, archetype par archetype, les comptes de
// [KeyframeClosure] — fermes, bornes, composant bloquant le plus frequent. Chaque paquet tient les
// invariants de la structure. MUTATION — prouver un record dont la traversee depasse la frontiere
// (`tr.EndBit >= b.Want` dans [preuveDeLEtatComplet]) : les fermes montent, ROUGE.
func TestLaPhaseDesImagesClesPorteLaFermetureDeKeyframeClosure(t *testing.T) {
	for _, court := range closureMiniFilms() {
		dir, film := bobineParBuild(t, court)
		want, err := fermetureMemo(dir)
		if err != nil {
			t.Fatal(err)
		}
		fc := NewFilmContext(film)
		reg, err := fc.Registry()
		if err != nil {
			t.Fatalf("%s : registre : %v", court, err)
		}
		f := fermetureDeLaStructure{stats: map[uint32]KeyframeClosureStat{}, bloquants: map[uint32]map[string]int{}}
		paquets := 0
		for p, err := range fc.ImagesCles() {
			if err != nil {
				t.Fatalf("%s : %v", court, err)
			}
			paquets++
			verifierLeRecordDEtatComplet(t, court, p)
			f.accumuler(reg, p)
		}
		got := f.rendre()
		if paquets == 0 || !reflect.DeepEqual(got, want) {
			t.Errorf("%s : %d paquet(s) ; la structure rend\n %v\n KeyframeClosure\n %v", court, paquets, got, want)
		}
	}
}

// fermetureDeLaStructure recompte, depuis les preuves de la structure, ce que [KeyframeClosure]
// mesure : les records bornes (tous sauf le dernier de chaque paquet), les fermes, et le composant
// qui arrete le plus de records bornes.
type fermetureDeLaStructure struct {
	stats     map[uint32]KeyframeClosureStat
	bloquants map[uint32]map[string]int
}

// accumuler compte les records bornes d un paquet.
func (f *fermetureDeLaStructure) accumuler(reg *Registry, p *lecture.Paquet) {
	for i := 0; i+1 < len(p.Records); i++ {
		r := p.Records[i]
		ti := uint32(r.TI) //nolint:gosec // archetype d une ancre, jamais negatif
		s := f.stats[ti]
		s.Total++
		switch {
		case r.Desync >= 0:
			if f.bloquants[ti] == nil {
				f.bloquants[ti] = map[string]int{}
			}
			f.bloquants[ti][nomComposantBloquant(reg, int(r.TI), int(r.Desync))]++
		case r.Preuve == lecture.PreuveFerme:
			s.Closed++
		}
		f.stats[ti] = s
	}
}

// rendre rend les comptes, chaque archetype avec son composant le plus bloquant.
func (f *fermetureDeLaStructure) rendre() map[uint32]KeyframeClosureStat {
	for ti, parComposant := range f.bloquants {
		s := f.stats[ti]
		s.Blocking = composantLePlusBloquant(parComposant)
		f.stats[ti] = s
	}
	return f.stats
}

// verifierLeRecordDEtatComplet verifie les invariants d UN paquet d image-cle : des records d etat
// complet tries par bit, lies par l image-cle, leur rang de vue egal a la tete de leur identite ; un
// record ferme finit sur le record suivant ; ses composants sont dans son etendue, contigus, et un
// composant infranchissable est le dernier et en porte la desynchronisation ; aucun verdict de
// paquet.
func verifierLeRecordDEtatComplet(t *testing.T, nom string, p *lecture.Paquet) {
	t.Helper()
	ou := fmt.Sprintf("%s chunk %d paquet %d", nom, p.Chunk, p.Index)
	if p.Type != PacketTypeKeyframe || p.Fermeture != (lecture.Fermeture{}) || p.Debut != lecture.DebutNonRenseigne {
		t.Errorf("%s : type %d, fermeture %+v, debut %d : attendu une image-cle sans verdict", ou, p.Type, p.Fermeture, p.Debut)
	}
	arene := uint32(0)
	for i, r := range p.Records {
		if r.Genre != lecture.GenreEtatComplet || r.Liaison != lecture.LiaisonImageCle || uint32(r.Vue) != r.Vie.Gen {
			t.Errorf("%s : record %d %+v, attendu un etat complet lie par l image-cle", ou, i, r)
		}
		if i+1 < len(p.Records) {
			suivant := p.Records[i+1].Debut
			if suivant <= r.Debut || (r.Preuve == lecture.PreuveFerme && r.Debut+r.Bits != suivant) {
				t.Errorf("%s : record %d [%d, %d) %d, suivant au bit %d", ou, i, r.Debut, r.Debut+r.Bits, r.Preuve, suivant)
			}
		} else if r.Preuve != lecture.PreuveNonProuve {
			t.Errorf("%s : dernier record prouve (%d) sans frontiere", ou, r.Preuve)
		}
		if r.Comps[0] != arene || r.Comps[1] < r.Comps[0] {
			t.Fatalf("%s : record %d, composants %v hors de l ordre de l arene (%d)", ou, i, r.Comps, arene)
		}
		arene = r.Comps[1]
		cs := p.Comps[r.Comps[0]:r.Comps[1]]
		for k, c := range cs {
			if c.Debut < r.Debut || c.Debut+c.Bits > r.Debut+r.Bits || (k+1 < len(cs) && c.Debut+c.Bits != cs[k+1].Debut) {
				t.Errorf("%s : record %d, composant %+v hors de son etendue ou non contigu", ou, i, c)
			}
			if c.Etat == lecture.EtatInfranchissable && (k+1 != len(cs) || r.Desync != int16(c.Index)) {
				t.Errorf("%s : record %d, composant infranchissable %+v qui n arrete pas le record", ou, i, c)
			}
		}
	}
}

// TestLaPhaseDesImagesClesRestaureLeContexte : l iteration pose le decoupage MPP du format du film
// et le restaure a la sortie, arret anticipe compris ; un film sans registre rend son erreur, une
// fois.
func TestLaPhaseDesImagesClesRestaureLeContexte(t *testing.T) {
	_, film := bobineParBuild(t, "fb1a1a72")
	fc := NewFilmContext(film)
	avant := fc.ProfilDeBalayage().MPP
	n := 0
	for _, err := range fc.ImagesCles() {
		if err != nil {
			t.Fatalf("marche : %v", err)
		}
		if n++; n == 2 {
			break
		}
	}
	if n != 2 || fc.ProfilDeBalayage().MPP != avant {
		t.Errorf("%d paquet(s) avant l arret, decoupage MPP %+v apres contre %+v avant", n, fc.ProfilDeBalayage().MPP, avant)
	}
	_, sansRegistre := bobineParBuild(t, "000d5950") // la bobine historique, sans `chunk_00`
	rendus := 0
	for p, err := range NewFilmContext(sansRegistre).ImagesCles() {
		rendus++
		if p != nil || err == nil || errors.Is(err, ErrNoFilmChunk) {
			t.Errorf("film sans registre : paquet %v, erreur %v, attendu l erreur du registre", p, err)
		}
	}
	if rendus != 1 {
		t.Errorf("film sans registre : %d tour(s), attendu 1", rendus)
	}
}
