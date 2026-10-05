package grammar

// vue_a_execution_test.go — les vecteurs du lot VA, etape V1 : les charges gardees par une valeur
// d execution (Script, biped_throw_initiate), les positions a index des genres 5 et 6, la tete lue a
// l identique, et un vrai paquet. Ecrits d apres les ecrivains du jeu (`FUN_142eec4d8`,
// `FUN_14104fc8c`, `FUN_1407ec560`) et avec les IMMEDIATS de leurs lecteurs, jamais les constantes du
// portage.

import (
	"math/rand/v2"
	"os"
	"slices"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// teteDAvant est le TEMOIN D AVANT LE LOT VA : la vue A rangee, la tete et la route que
// `rangerLaTete`, `consumeVueA` et `teteDe` rendaient a `87cdfa761` — la continuation et le genre du
// premier message, jamais plus.
func teteDAvant(pay []byte) (lecture.VueA, teteDeTrame, bool) {
	br := LecteurSur(pay)
	br.Skip(1)
	frameLen := len(pay) * 8
	a := lecture.VueA{Debut: 1}
	porte := false
	if placeDisponible(br, frameLen, 1) {
		if !br.ReadBit() {
			porte = true
		} else if placeDisponible(br, frameLen, LargeurGenreVueA) {
			a.Genres = append(a.Genres, uint8(br.ReadBits(LargeurGenreVueA)))
		}
	}
	a.Bits, a.Etat = uint32(br.BitPos()-1), etatDeVue(porte) //nolint:gosec // position d un payload
	tete := teteDuPayload(pay)
	switch {
	case a.Etat == lecture.VueTerminee:
		tete = teteDeTrame{}
	case a.Etat == lecture.VueArretee && len(a.Genres) == 1:
		tete = teteDeTrame{liste: true, genre: int(a.Genres[0])}
	}
	return a, tete, a.Etat == lecture.VueArretee
}

// TestLaTeteEstLueALIdentique : sur des payloads quelconques et sous tous les profils (film sans
// table, film recent, film ancien a 21 genres, Script declare), la tete donnee aux canaux et la route
// vers la localisation sont celles d avant le lot ; la vue A rangee l est aussi quand le film ne la
// rend pas lisible, et commence par la meme tete sinon. MUTATIONS — la tete prise au dernier genre ;
// la route « liste annoncee » reduite a `VueArretee` : ROUGE.
func TestLaTeteEstLueALIdentique(t *testing.T) {
	recent := grammaireRecente()
	recent.script = scriptAvecPrefixe
	grammaires := []grammaireDeLaVueA{{}, recent, grammaireDeTest(vueAPrefixe, 21)}
	gen := rand.New(rand.NewPCG(5605, 36772)) //nolint:gosec // vecteurs de test deterministes
	for n := range 20000 {
		pay := make([]byte, 1+gen.IntN(12))
		for i := range pay {
			pay[i] = byte(gen.UintN(256))
		}
		avant, tete, route := teteDAvant(pay)
		for k, g := range grammaires {
			p := lecture.Paquet{Payload: pay}
			rangerLaTete(&p, ProfilDeBalayageParDefaut(), g)
			if teteDe(&p) != tete || listeAnnoncee(&p.VueA) != route {
				t.Fatalf("payload %d %x, profil %d : tete %+v route %v, avant %+v %v", n, pay, k, teteDe(&p),
					listeAnnoncee(&p.VueA), tete, route)
			}
			if p.VueA.Debut != avant.Debut || len(avant.Genres) == 1 && p.VueA.Genres[0] != avant.Genres[0] {
				t.Fatalf("payload %d %x, profil %d : vue A %+v, avant %+v", n, pay, k, p.VueA, avant)
			}
			if k == 0 && (p.VueA.Bits != avant.Bits || p.VueA.Etat != avant.Etat || !slices.Equal(p.VueA.Genres, avant.Genres)) {
				t.Fatalf("payload %d %x, film sans table : vue A %+v, avant %+v", n, pay, p.VueA, avant)
			}
		}
	}
}

// ecrireScript ecrit un message Script comme `FUN_142eec4d8` : W(15) si `prefixe`, W(13), W(10) n,
// puis n bits.
func (w *bitWriter) ecrireScript(prefixe bool, compteur uint64, n int) {
	w.ecrireEnTeteDeMessage(15)
	if prefixe {
		w.bits(compteur, 15)
	}
	w.bits(148, 13)
	w.bits(uint64(n), 10) //nolint:gosec // n < 1024
	w.bits(0x2a, n)
}

// ecrireEvenementJoueurCourt ecrit un `PlayerGameEventSmall` (genre 82) sans propriete : R(32),
// R(8), compte R(3) nul, sous-sac absent, 32 x R(1).
func (w *bitWriter) ecrireEvenementJoueurCourt() {
	w.ecrireEnTeteDeMessage(82)
	w.bits(0x12345678, 32)
	w.bits(7, 8)
	w.bits(0, 3)
	w.bit(0)
	w.bits(0xffffffff, 32)
}

// TestLeScriptSuitLaSimulationDeLEnregistreur : l ecrivain du Script ecrit le prefixe R(15) quand la
// simulation de l enregistreur n est pas dist-client ; le lecteur porte le lit selon ce que le film
// declare, et la fin de la vue A tombe au bit pres apres un message 82 et le terminateur. Sans
// options de partie lues, le Script ne se lit pas. MUTATION — le prefixe lu sur la regle inversee :
// ROUGE.
func TestLeScriptSuitLaSimulationDeLEnregistreur(t *testing.T) {
	for _, c := range []struct {
		nom     string
		prefixe bool
		etat    etatDuScript
	}{
		{"enregistreur dist-server (3)", true, scriptAvecPrefixe},
		{"enregistreur dist-client (2)", false, scriptSansPrefixe},
	} {
		var w bitWriter
		w.bit(1)
		w.ecrireScript(c.prefixe, 7, 20)
		w.ecrireEvenementJoueurCourt()
		w.bit(0)
		fin := w.n
		w.bits(0x5555, 16)
		g := grammaireRecente()
		g.script = c.etat
		if a := lireSous(w.buf, g); !a.Porte || a.Fin != fin || !slices.Equal(a.Genres, []int{15, 82}) {
			t.Errorf("%s : %+v, attendu portee jusqu au bit %d, genres [15 82]", c.nom, a, fin)
		}
		g.script = scriptInconnu
		if a := lireSous(w.buf, g); a.Porte || a.Fin != 9 {
			t.Errorf("%s, simulation non declaree : %+v, attendu arretee apres le genre 15", c.nom, a)
		}
	}
}

// TestLesOptionsDePartieSeLisentEnTeteDuCorps : `FUN_1407ec560` ecrit au premier bit du corps de
// `chunk_00` game_mode W(3) puis game_simulation W(3). MUTATIONS — la simulation lue au premier
// champ, ou sur deux bits : ROUGE.
func TestLesOptionsDePartieSeLisentEnTeteDuCorps(t *testing.T) {
	var w bitWriter
	w.bits(0x1f, 13) // ce qui precede le corps
	w.bits(2, 3)     // game_mode
	w.bits(5, 3)     // game_simulation (valeur hors enumeration : seul son placement compte ici)
	w.bits(0, 8)
	if sim, ok := lireSimulationDeLEnregistreur(w.buf, 13); !ok || sim != 5 {
		t.Fatalf("simulation %d (%v), attendu 5", sim, ok)
	}
	if _, ok := lireSimulationDeLEnregistreur(w.buf[:2], 13); ok {
		t.Fatal("options lues dans un corps qui ne tient pas dans le tampon")
	}
}

// TestLesBobinesDeclarentLeurTableEtLeurSimulation : sur les sept bobines par build, la classe de la
// table des genres (egale sur HI_1_12_0 et HI_1_13_0, prefixe sur HI_1_9_0 a HI_1_11_0, illisible sur
// HI_1_8_0 et HI_1_4_1 dont une version differe) et la simulation de l enregistreur (dist-server,
// donc le prefixe du Script).
func TestLesBobinesDeclarentLeurTableEtLeurSimulation(t *testing.T) {
	attendu := map[string]classeDeLaVueA{"a521164d": vueAIllisible, "60ae07c4": vueAIllisible,
		"11de8353": vueAPrefixe, "111fa685": vueAPrefixe, "e5adf7b2": vueAPrefixe, "bcb6d393": vueAEgale,
		"fb1a1a72": vueAEgale}
	for _, b := range bobinesIdentite() {
		id, err := ReadFilmIdentity(bobineChunk00(t, b.film))
		if err != nil {
			t.Fatalf("%s : %v", b.film, err)
		}
		if classe, cardinal := classeDesGenres(id.TypeVersions); classe != attendu[b.film] ||
			classe != vueAIllisible && cardinal != b.types {
			t.Errorf("%s (%s) : classe %d, cardinal %d ; attendu %d", b.film, b.build, classe, cardinal, attendu[b.film])
		}
		if !id.OptionsDePartieLues || id.SimulationDeLEnregistreur != 3 {
			t.Errorf("%s (%s) : simulation %d (%v), attendu 3 (dist-server)", b.film, b.build,
				id.SimulationDeLEnregistreur, id.OptionsDePartieLues)
		}
	}
}

// TestLeLancerLitSaChargeSelonK : `FUN_14104fc8c` ecrit k = W(1) ; k = 0 : W(3) ; k = 1 : W(1) porte
// puis W(32) si elle vaut 1, et W(4) ; puis W(1) et W(5) s il vaut 0. MUTATION — la polarite de k
// inversee : ROUGE.
func TestLeLancerLitSaChargeSelonK(t *testing.T) {
	for _, c := range []struct {
		nom    string
		ecrire func(w *bitWriter)
	}{
		{"k = 0, queue a 0", func(w *bitWriter) { w.bit(0); w.bits(5, 3); w.bit(0); w.bits(0x1f, 5) }},
		{"k = 0, queue a 1", func(w *bitWriter) { w.bit(0); w.bits(5, 3); w.bit(1) }},
		{"k = 1, porte a 1", func(w *bitWriter) { w.bit(1); w.bit(1); w.bits(0xdeadbeef, 32); w.bits(9, 4); w.bit(1) }},
		{"k = 1, porte a 0", func(w *bitWriter) { w.bit(1); w.bit(0); w.bits(9, 4); w.bit(0); w.bits(3, 5) }},
	} {
		var w bitWriter
		c.ecrire(&w)
		fin := w.n
		w.bits(0xffff, 16)
		br := LecteurSur(w.buf)
		if !chargeLancerInitie(br) || br.BitPos() != fin {
			t.Errorf("%s : fin %d, attendu %d", c.nom, br.BitPos(), fin)
		}
	}
}

// carteDeTest est une entree de catalogue dont la region jouee est la 1, sur deux bits d index.
func carteDeTest() profile.MapQuantEntry {
	return profile.MapQuantEntry{Min: [3]float32{-100, -50, -10}, Max: [3]float32{100, 50, 10},
		AxisWidths: [3]uint{14, 13, 11}, Region: 1, RegionIndexBits: 2}
}

// ecrirePositionDeNiveau ecrit une position de `FUN_14076e524` : la porte (1 : table DEFAUT), sinon
// l index puis trois axes aux largeurs `w`.
func (w *bitWriter) ecrirePositionDeNiveau(porte bool, index uint64, larg [3]uint) {
	if porte {
		w.bit(1)
	} else {
		w.bit(0)
		w.bits(index, 2)
	}
	for _, x := range larg {
		w.bits(0, int(x))
	}
}

// TestLesPositionsAIndexSeLisentSurLaRegionJouee : la detonation (genre 5, niveau 0xf) et l impact
// (genre 6, niveau 0xc) lisent leur position par `FUN_14076e524` : porte posee, la table DEFAUT du
// build ; index de la region jouee, la loi `FUN_140be9b88` sur les bornes de l entree de catalogue ;
// un autre index, ou un film lu sans carte, arrete la lecture — rien n est devine. MUTATIONS —
// niveaux 0xf / 0xc lus a 0x10 ; un index d une autre region lu : ROUGE.
func TestLesPositionsAIndexSeLisentSurLaRegionJouee(t *testing.T) {
	e := carteDeTest()
	bornes := [3][2]float32{{-100, 100}, {-50, 50}, {-10, 10}}
	type cas struct {
		nom    string
		porte  bool
		index  uint64
		tables tablesDePosition
		lue    bool
	}
	for _, genre := range []struct {
		nom    string
		niveau int
		charge func(*Lecteur) bool
		avant  func(w *bitWriter)
		apres  func(w *bitWriter)
	}{
		{"detonation", 0xf, chargeDetonation,
			func(w *bitWriter) { w.bits(0, 6); w.bit(1); w.bit(0) }, // FUN_140809454, variante, FUN_14080d69c
			func(w *bitWriter) { w.bits(0, 0x13+5+1+9); w.bit(0); w.bit(0); w.bits(0, 8+2) }},
		{"impact", 0xc, chargeImpact,
			func(w *bitWriter) { w.bit(1); w.bits(0, 7+7+0x13) }, // variante, deux scalaires, direction
			func(w *bitWriter) { w.bits(0, 0x13+9+1) }},
	} {
		for _, c := range []cas{
			{"porte posee", true, 0, tablesDeLaRegionJouee(e), true},
			{"index de la region jouee", false, 1, tablesDeLaRegionJouee(e), true},
			{"index d une autre region", false, 2, tablesDeLaRegionJouee(e), false},
			{"film lu sans carte", false, 1, tablesDeLaRegionJouee(profile.MapQuantEntry{}), false},
		} {
			larg := profile.LargeursAxeParDefautDuBuild(genre.niveau)
			if !c.porte {
				larg = profile.LargeursAxeDuNiveau(bornes, genre.niveau)
			}
			var w bitWriter
			genre.avant(&w)
			w.ecrirePositionDeNiveau(c.porte, c.index, larg)
			genre.apres(&w)
			fin := w.n
			w.bits(0xffff, 16)
			br := LecteurSur(w.buf)
			br.vueA = grammaireRecente()
			br.vueA.positions = c.tables
			if lue := genre.charge(br); lue != c.lue || lue && br.BitPos() != fin {
				t.Errorf("%s, %s : lue %v (fin %d), attendu %v (fin %d)", genre.nom, c.nom, lue, br.BitPos(), c.lue, fin)
			}
		}
	}
}

// TestUnVraiPaquetDeQuaranteScripts : `bcb6d393` (HI_1_12_0), chunk 1, paquet 204, sous ce que son
// `chunk_00` declare : la vue A est quarante messages Script dont le prefixe compte de 0 a 39, puis le
// terminateur ; sa fin est le bit 5 605 (`testdata/vue_a_bcb6d393_1_204.PROVENANCE.txt`).
func TestUnVraiPaquetDeQuaranteScripts(t *testing.T) {
	pay, err := os.ReadFile("testdata/vue_a_bcb6d393_1_204.bin")
	if err != nil {
		t.Fatal(err)
	}
	id, err := ReadFilmIdentity(bobineChunk00(t, "bcb6d393"))
	if err != nil {
		t.Fatal(err)
	}
	bal := ProfilDeBalayageParDefaut()
	bal.Grammaire.ControleDeCorruption = id.ControleDeCorruption
	var g grammaireDeLaVueA
	g.classe, g.genres = classeDesGenres(id.TypeVersions)
	g.script = scriptSansPrefixe
	if id.SimulationDeLEnregistreur != simulationDistClient {
		g.script = scriptAvecPrefixe
	}
	a := lireLaVueA(pay, 1, bal, g)
	if !a.Porte || a.Fin != 5605 || len(a.Genres) != 40 || slices.ContainsFunc(a.Genres, func(g int) bool { return g != 15 }) {
		t.Fatalf("vue A : portee %v, fin %d, %d genres %v ; attendu quarante Script jusqu au bit 5605", a.Porte,
			a.Fin, len(a.Genres), a.Genres)
	}
	br := LecteurSur(pay)
	br.SetBitPos(1)
	for i := range 40 {
		if !br.ReadBit() || br.ReadBits(LargeurGenreVueA) != 15 || br.ReadBits(3) != 0 {
			t.Fatalf("message %d : en-tete inattendu au bit %d", i, br.BitPos())
		}
		if c := br.ReadBits(15); c != uint64(i) { //nolint:gosec // i < 40
			t.Fatalf("message %d : compteur %d", i, c)
		}
		br.Skip(13)
		br.Skip(int(br.ReadBits(10)))
		if id.ControleDeCorruption && br.ReadBit() {
			br.Skip(32)
		}
	}
	if br.ReadBit() || br.BitPos() != 5605 {
		t.Fatalf("terminateur attendu au bit 5604, lecture au bit %d", br.BitPos())
	}
}
