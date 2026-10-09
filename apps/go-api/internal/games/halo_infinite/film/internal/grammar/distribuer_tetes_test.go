package grammar

// distribuer_tetes_test.go — LA TETE DE CHAQUE TRAME, LUE UNE FOIS (lot 2.3 du plan de l etape 2) :
// la vue A rangee, la passe des tetes, et l egalite avec la tete lue hors de la structure.

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// TestRangerLaTete : les formes de la vue A rangee — liste vide (vue terminee sur la continuation),
// un message d un film dont la vue A ne se lit pas au-dela de sa tete (vue arretee apres son genre),
// une tete qui ne tient pas dans le payload (vue arretee sans genre, et la lecture tolerante
// d aujourd hui pour les canaux), et la MEME tete d un film recent, dont la vue A se lit jusqu a son
// terminateur (lot VA) : vue terminee, tous ses genres, et la tete donnee aux canaux INCHANGEE.
// MUTATIONS — ranger la vue A sans ses genres ; la tete prise au dernier genre au lieu du premier :
// ROUGE.
func TestRangerLaTete(t *testing.T) {
	// config 1 ; zoom (genre 21) : continuation, R(7), trois gardes fermees, R(2) ; ramassage
	// (genre 9) : R(3), porte de FUN_14080d69c fermee ; terminateur.
	var deux bitWriter
	deux.bit(1)
	deux.ecrireEnTeteDeMessage(21)
	deux.bits(1, 2)
	deux.ecrireEnTeteDeMessage(9)
	deux.bits(5, 3)
	deux.bit(0)
	deux.bit(0)
	cas := []struct {
		nom     string
		payload []byte
		g       grammaireDeLaVueA
		vueA    lecture.VueA
		tete    teteDeTrame
	}{
		{"liste vide", []byte{0x80, 0x00}, grammaireRecente(), lecture.VueA{Debut: 1, Bits: 1, Etat: lecture.VueTerminee},
			teteDeTrame{}},
		// config 1, continuation 1, genre 36 = 0b0100100 : 0xD2 puis le bit de poids faible du genre
		{"un message", []byte{0xD2, 0x00}, grammaireDeLaVueA{},
			lecture.VueA{Debut: 1, Bits: 8, Etat: lecture.VueArretee, Genres: []uint8{36}}, teteDeTrame{liste: true, genre: 36}},
		{"tete tronquee", []byte{0xD2}, grammaireRecente(), lecture.VueA{Debut: 1, Bits: 1, Etat: lecture.VueArretee},
			teteDuPayload([]byte{0xD2})},
		{"zoom puis ramassage, film sans table", deux.buf, grammaireDeLaVueA{},
			lecture.VueA{Debut: 1, Bits: 8, Etat: lecture.VueArretee, Genres: []uint8{21}}, teteDeTrame{liste: true, genre: 21}},
		{"zoom puis ramassage, film recent", deux.buf, grammaireRecente(),
			lecture.VueA{Debut: 1, Bits: uint32(deux.n - 1), Etat: lecture.VueTerminee, Genres: []uint8{21, 9}},
			teteDeTrame{liste: true, genre: 21}},
	}
	for _, c := range cas {
		p := lecture.Paquet{Payload: c.payload}
		rangerLaTete(&p, ProfilDeBalayageParDefaut(), c.g)
		if p.VueA.Debut != c.vueA.Debut || p.VueA.Bits != c.vueA.Bits || p.VueA.Etat != c.vueA.Etat ||
			!reflect.DeepEqual(append([]uint8(nil), p.VueA.Genres...), append([]uint8(nil), c.vueA.Genres...)) {
			t.Errorf("%s : vue A %+v, attendu %+v", c.nom, p.VueA, c.vueA)
		}
		if got := teteDe(&p); got != c.tete {
			t.Errorf("%s : tete %+v, attendu %+v", c.nom, got, c.tete)
		}
	}
}

// canalDeTeteTemoin retient la tete de chaque trame qu on lui donne.
type canalDeTeteTemoin struct {
	vues  []teteVue
	clos  int
	bilan BilanDeMarche
}

// teteVue : ce qu un canal de tete garde d une trame.
type teteVue struct {
	chunk, index int
	vueA         lecture.VueA
	tete         teteDeTrame
	duPayload    teteDeTrame
}

func (*canalDeTeteTemoin) Interets() []Interet { return nil }

func (c *canalDeTeteTemoin) Tete(p *lecture.Paquet) {
	a := p.VueA
	a.Genres = append([]uint8(nil), a.Genres...)
	c.vues = append(c.vues, teteVue{chunk: p.Chunk, index: p.Index, vueA: a, tete: teteDe(p),
		duPayload: teteDuPayload(p.Payload)})
}

func (c *canalDeTeteTemoin) Clore(b BilanDeMarche) { c.clos, c.bilan = c.clos+1, b }

// TestLaPasseDesTetesEstLaTeteDeLaMarche : sur les deux bobines a trames delta, la passe des tetes
// (aucun canal des trames) rend les memes trames, avec la meme vue A, que la marche complete ; la
// tete rangee est celle que la lecture hors structure rend ; la passe ne marche ni les images-cles
// ni les records.
// MUTATION — la marche complete qui ne range plus la tete d une trame a liste (seule la
// continuation, sans genre) : ROUGE.
func TestLaPasseDesTetesEstLaTeteDeLaMarche(t *testing.T) {
	for _, court := range []string{"000d5950", "e5adf7b2"} {
		film := bobineDeTrames(t, court)
		fc := contexteDeBobine(film)
		seule := &canalDeTeteTemoin{}
		if err := Distribuer(fc, seule); err != nil {
			t.Fatalf("%s : passe des tetes : %v", court, err)
		}
		if fc.marches != nil || seule.bilan.Obs != nil || seule.clos != 1 || seule.bilan.ChunksLus == 0 {
			t.Errorf("%s : memoire d ancres %v, observation %v, %d cloture(s), bilan %+v : la passe des "+
				"tetes a marche autre chose que les tetes", court, fc.marches != nil, seule.bilan.Obs, seule.clos,
				seule.bilan)
		}
		avecMarche := &canalDeTeteTemoin{}
		if err := Distribuer(contexteDeBobine(film), &canalDesTramesTemoin{}, avecMarche); err != nil {
			t.Fatalf("%s : marche complete : %v", court, err)
		}
		if len(seule.vues) == 0 || !reflect.DeepEqual(seule.vues, avecMarche.vues) {
			t.Fatalf("%s : %d tete(s) par la passe, %d par la marche, ou differentes", court, len(seule.vues),
				len(avecMarche.vues))
		}
		listes := 0
		for _, v := range seule.vues {
			if v.tete != v.duPayload && v.duPayload.liste {
				t.Fatalf("%s paquet %d:%d : tete %+v, %+v hors structure", court, v.chunk, v.index, v.tete, v.duPayload)
			}
			if v.tete.liste != v.duPayload.liste {
				t.Fatalf("%s paquet %d:%d : continuation %v, %v hors structure", court, v.chunk, v.index,
					v.tete.liste, v.duPayload.liste)
			}
			if v.tete.liste {
				listes++
			}
		}
		if listes == 0 {
			t.Errorf("%s : aucune trame a liste d evenements : la tete n est pas exercee", court)
		}
	}
}

// sourceEnMemoire : les chunks d un film, en memoire et deja decompresses.
type sourceEnMemoire [][]byte

func (s sourceEnMemoire) NumChunks() int { return len(s) }

func (s sourceEnMemoire) Chunk(i int) ([]byte, error) { return s[i], nil }

// chunkDUneTrame : un chunk de donnees qui porte la seule trame delta `pay`, puis son terminateur
// (type 7, taille 0) ; en-tetes de 16 octets [u16 type][2 octets][u32 taille][u64 horodatage].
func chunkDUneTrame(pay []byte) []byte {
	out := make([]byte, 2*packetHeaderSize+len(pay))
	binary.LittleEndian.PutUint16(out, PacketTypeDelta)
	binary.LittleEndian.PutUint32(out[4:], uint32(len(pay))) //nolint:gosec // payload de test
	copy(out[packetHeaderSize:], pay)
	binary.LittleEndian.PutUint16(out[packetHeaderSize+len(pay):], 7) // CHUNK_END
	return out
}

// contexteQuiDeclareLeControle ouvre le contexte d un film dont le registre est celui de la bobine
// 000d5950 et dont le seul chunk de donnees porte la trame `pay`. Son profil est celui que la bobine
// rend, sauf deux champs de sa section d identification : la table des genres native (classe EGALE)
// et le controle de corruption VRAI. Aucune bobine du depot ne declare ce controle : le profil est
// pose avant la premiere derivation, qui le lit comme elle lirait celui du film.
func contexteQuiDeclareLeControle(t *testing.T, pay []byte) *FilmContext {
	t.Helper()
	reg, err := os.ReadFile(filepath.Join("..", "facts", "killsource", "testdata", "minibobine_000d5950",
		"chunk_00.bin"))
	if err != nil {
		t.Fatalf("registre de la bobine : %v", err)
	}
	film, err := source.Load(sourceEnMemoire{reg, chunkDUneTrame(pay)}, nil)
	if err != nil {
		t.Fatalf("film : %v", err)
	}
	raw, _ := FilmRegistryChunk(film)
	id, err := ReadFilmIdentity(raw)
	if err != nil {
		t.Fatalf("identite de la bobine : %v", err)
	}
	id.ControleDeCorruption = true
	id.TypeVersions = make([]uint32, GenresVueA)
	for g := range GenresVueA {
		id.TypeVersions[g] = versionNative(g)
	}
	fc := NewFilmContext(film)
	fc.prof, fc.profLu = profile.Resoudre(profile.ClesDuFilm{RegistrePresent: true, Identite: id,
		IdentiteLue: true}, nil), true
	return fc
}

// TestLaPasseDesTetesLitLeControleDuFilm : un film qui declare le controle de corruption par
// composant ; sa trame porte un zoom (genre 21 : trois gardes fermees, R(2)) suivi du controle
// `R(1) = 1, R(32)` (`FUN_14076cea8`), puis le terminateur. La passe des tetes et la marche lisent
// le controle : la meme vue A, terminee apres le terminateur, de genre 21 seul.
// MUTATION — [FilmContext.profilDeLaVueA] sans le controle du film : la passe lit le bit du controle
// comme la continuation d un second message, de genre 127 (au-dela du cardinal, la vue s arrete) :
// ROUGE.
func TestLaPasseDesTetesLitLeControleDuFilm(t *testing.T) {
	var w bitWriter
	w.bit(1) // configuration
	w.ecrireEnTeteDeMessage(21)
	w.bits(1, 2)           // FUN_14080cb98 : R(2)
	w.bit(1)               // FUN_14076cea8 : R(1)
	w.bits(0xffffffff, 32) // R(32)
	w.bit(0)               // le terminateur
	fin := w.n
	want := lecture.VueA{Debut: 1, Bits: uint32(fin - 1), Etat: lecture.VueTerminee, Genres: []uint8{21}} //nolint:gosec // bits du vecteur

	seule := &canalDeTeteTemoin{}
	if err := Distribuer(contexteQuiDeclareLeControle(t, w.buf), seule); err != nil {
		t.Fatalf("passe des tetes : %v", err)
	}
	avecMarche := &canalDeTeteTemoin{}
	if err := Distribuer(contexteQuiDeclareLeControle(t, w.buf), &canalDesTramesTemoin{}, avecMarche); err != nil {
		t.Fatalf("marche complete : %v", err)
	}
	for nom, c := range map[string]*canalDeTeteTemoin{"passe des tetes": seule, "marche complete": avecMarche} {
		if len(c.vues) != 1 {
			t.Fatalf("%s : %d trame(s), attendu une", nom, len(c.vues))
		}
		if a := c.vues[0].vueA; a.Debut != want.Debut || a.Bits != want.Bits || a.Etat != want.Etat ||
			!reflect.DeepEqual(a.Genres, want.Genres) {
			t.Errorf("%s : vue A %+v, attendu %+v", nom, a, want)
		}
	}
}
