package grammar

// marche_images_cles_test.go — LA PHASE IMAGES-CLES RANGE CE QUE LA MARCHE LIT (ADR 0037 IR-1,
// IR-4, IR-6) : sur les bobines par build, chaque paquet tient les invariants de la structure, et
// chaque ancre elue par le repli est marquee. La fermeture par archetype que la carte en tire
// ([KeyframeClosure]) est tenue par son golden (`keyframe_closure_ratchet_test.go`).

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// bobineParBuild charge la mini-bobine par build `court` du rejeu.
func bobineParBuild(t *testing.T, court string) *source.Film {
	t.Helper()
	dir := filepath.Join("..", "..", "replay", "testdata", "minifilm_"+court)
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("bobine absente (%s) : %v", dir, err)
	}
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("LoadDir %s : %v", dir, err)
	}
	return film
}

// TestLaPhaseDesImagesClesTientSesInvariants : sur les sept bobines par build, chaque paquet
// d image-cle tient les invariants de la structure ([verifierLeRecordDEtatComplet]), et chaque
// ancre que le repli a elue — une par election comptee par la marche — est marquee.
// MUTATION — ne plus marquer l ancre elue (`Elue: elue && pos < 0` dans `marcherLaTable`) : ROUGE.
func TestLaPhaseDesImagesClesTientSesInvariants(t *testing.T) {
	electionsVues := 0
	for _, court := range closureMiniFilms() {
		fc := NewFilmContext(bobineParBuild(t, court))
		marche := fc.MarcheDImageCle()
		paquets, elues, elections := 0, 0, 0
		for p, err := range fc.ImagesCles() {
			if err != nil {
				t.Fatalf("%s : %v", court, err)
			}
			paquets++
			verifierLeRecordDEtatComplet(t, court, p)
			for _, r := range p.Records {
				if r.Liaison == lecture.LiaisonImageCleElue {
					elues++
				}
			}
			_, st := marche.RecordsStats(p.Payload)
			elections += st.Elections
		}
		if paquets == 0 || elues != elections {
			t.Errorf("%s : %d paquet(s), %d ancre(s) marquee(s) elue(s) pour %d election(s) de la marche",
				court, paquets, elues, elections)
		}
		electionsVues += elections
	}
	if electionsVues == 0 {
		t.Fatal("aucune election sur les sept bobines : la marque d election n est pas exercee")
	}
}

// verifierLeRecordDEtatComplet verifie les invariants d UN paquet d image-cle : des records d etat
// complet tries par bit, lies par l image-cle (de proche en proche ou par election), leur rang de
// vue egal a la tete de leur identite ; un record ferme finit sur le record suivant ; ses
// composants sont dans son etendue, contigus, et un composant infranchissable est le dernier et en
// porte la desynchronisation ; aucun verdict de paquet.
func verifierLeRecordDEtatComplet(t *testing.T, nom string, p *lecture.Paquet) {
	t.Helper()
	ou := fmt.Sprintf("%s chunk %d paquet %d", nom, p.Chunk, p.Index)
	if p.Type != PacketTypeKeyframe || p.Fermeture != (lecture.Fermeture{}) || p.Debut != lecture.DebutNonRenseigne {
		t.Errorf("%s : type %d, fermeture %+v, debut %d : attendu une image-cle sans verdict", ou, p.Type, p.Fermeture, p.Debut)
	}
	arene := uint32(0)
	for i, r := range p.Records {
		lie := r.Liaison == lecture.LiaisonImageCle || r.Liaison == lecture.LiaisonImageCleElue
		if r.Genre != lecture.GenreEtatComplet || !lie || uint32(r.Vue) != r.Vie.Gen {
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
	fc := NewFilmContext(bobineParBuild(t, "fb1a1a72"))
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
	sansRegistre := bobineParBuild(t, "000d5950") // la bobine historique, sans `chunk_00`
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
