package grammar

// canal_des_kills_test.go — LES MESSAGES DE KILL QUE LA MARCHE REND A KILLSOURCE : ceux de la vue A,
// relus par la chaine du rattrapage, et le rattrapage borne aux trames ou la lecture de la vue A n est
// pas etablie.

import (
	"slices"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// TestLesKillsDeLaVueASeRelisentParLaChaine : sur la bobine de la marche ([bobineMarcheDir]), chaque
// message de kill que la vue A range est a sa position — une continuation a 1, le genre 85 — et la
// grammaire de la chaine du rattrapage relit les memes champs a la fin de sa boucle de presence : les
// deux lecteurs du message s accordent sur le flux reel. MUTATIONS — la victime et le tueur
// intervertis dans [chargeJoueurTue] ; la position du message decalee d un bit : ROUGES.
func TestLesKillsDeLaVueASeRelisentParLaChaine(t *testing.T) {
	film := bobineDeLaMarche(t)
	l, err := LireLaMarcheDeKillsource(NewFilmContext(film))
	if err != nil {
		t.Fatal(err)
	}
	trames := map[[2]int]types.Packet{}
	for _, p := range film.AllPackets() {
		trames[[2]int{p.Chunk, p.Index}] = p
	}
	lus := 0
	for _, k := range l.Kills {
		if k.Rattrape {
			continue
		}
		lus++
		p, ok := trames[[2]int{k.PositionDuChunk, k.Index}]
		if !ok || p.TS != k.TS {
			t.Fatalf("kill %+v : trame absente ou differente", k)
		}
		x := int(k.Kill.Debut) + 1
		if !estAncreDeKill(p.Payload, x) {
			t.Fatalf("kill %+v : pas de continuation suivie du genre 85 a %d", k, x)
		}
		r := nouveauCurseurEv(p.Payload, x+LargeurGenreVueA)
		if !evPresence(r, GenreJoueurTue) {
			t.Fatalf("kill %+v : boucle de presence illisible", k)
		}
		relu, fin := lireLeKillDeLaChaine(p.Payload, r.pos())
		relu.Debut = k.Kill.Debut
		if fin < 0 || relu != k.Kill {
			t.Errorf("kill %+v : la chaine relit %+v", k.Kill, relu)
		}
	}
	if lus == 0 {
		t.Fatalf("aucun kill lu par la vue A sur %d : la bobine ne prouve rien", len(l.Kills))
	}
}

// ecrireUnKill ecrit un message de kill complet : continuation, genre 85, trois gardes fermees, puis
// victime 3, tueur 7, deux parts et un assistant absent.
func (w *bitWriter) ecrireUnKill() {
	w.bit(1)
	w.bits(GenreJoueurTue, LargeurGenreVueA)
	w.bits(0, 3)
	w.bit(0)
	w.bits(3, 5)
	w.bit(0)
	w.bits(7, 5)
	w.bits(60, 32)
	w.bit(0)
	w.bit(1)
	w.bits(40, 32)
}

// ecrireTroisVides ecrit trois messages de genre 3 (corps vide, trois gardes fermees) puis le
// terminateur de la vue A : la chaine qui valide un message de kill qui les precede.
func (w *bitWriter) ecrireTroisVides() {
	for range 3 {
		w.bit(1)
		w.bits(3, LargeurGenreVueA)
		w.bits(0, 3)
	}
	w.bit(0)
}

// TestLeRattrapageCherchePartoutOuLaLectureNEstPasEtablie : le rattrapage ne cherche que dans une
// trame dont la vue B ne commence pas a la fin de sa vue A lue — lecture arretee, ou terminateur que
// la marche ne retient pas —, du premier genre de la vue A au debut de la vue B (la fin du payload
// quand la liste n est pas localisee), et ne rend pas un message que la vue A a lu. MUTATIONS — le
// rattrapage d une trame dont la vue B commence a la fin de la vue A ; la recherche au-dela du debut
// de la vue B ; un message de la vue A rendu une seconde fois : ROUGES.
func TestLeRattrapageCherchePartoutOuLaLectureNEstPasEtablie(t *testing.T) {
	// Apres le terminateur de la vue A (bit `e`), une vue B qui porte par hasard la forme d un kill
	// enchaine, suivie de rembourrage.
	var w bitWriter
	w.bit(1) // bit de configuration
	w.ecrireUnKill()
	w.ecrireTroisVides()
	e := w.n
	w.ecrireUnKill()
	w.ecrireTroisVides()
	w.bits(0, 16)
	premier := lecture.MessageDeKill{Debut: 1, Victime: 3, Tueur: 7, PartDuTueur: 60,
		Assistant: lecture.RefAbsente, PartDeLAssistant: 40}
	second := premier
	second.Debut = uint32(e) //nolint:gosec // position dans un payload de test
	for _, c := range []struct {
		nom     string
		g       grammaireDeLaVueA
		debut   lecture.DebutDeVueB
		vueB    int
		attendu []lecture.MessageDeKill
	}{
		{"film sans table, liste non localisee : jusqu a la fin du payload", grammaireDeLaVueA{},
			lecture.DebutNonLocalise, 0, []lecture.MessageDeKill{premier, second}},
		{"film sans table, vue B localisee apres le terminateur : avant elle seulement",
			grammaireDeLaVueA{}, lecture.DebutParSignature, e, []lecture.MessageDeKill{premier}},
		{"vue A lue, la vue B commence a sa fin : rien", grammaireRecente(), lecture.DebutParVueA, e, nil},
		{"vue A lue, terminateur non retenu, vue B a sa fin : le kill de la vue A n est pas rendu",
			grammaireRecente(), lecture.DebutParChaine, e, nil},
		{"vue A lue, terminateur non retenu, vue B plus loin : le second", grammaireRecente(),
			lecture.DebutParFermeture, w.n - 16, []lecture.MessageDeKill{second}},
	} {
		var p lecture.Paquet
		p.Payload = w.buf
		rangerLaTete(&p, ProfilDeBalayageParDefaut(), c.g)
		p.Debut, p.VueB.Debut = c.debut, uint32(c.vueB) //nolint:gosec // position dans un payload de test
		r := rattrapageDesKills{tranche: true}
		var lus []lecture.MessageDeKill
		for _, k := range r.rattraper(&p) {
			if k.Chaine != 3 {
				t.Errorf("%s : chaine de %d pour %+v, attendu 3", c.nom, k.Chaine, k.Kill)
			}
			lus = append(lus, k.Kill)
		}
		if !slices.Equal(lus, c.attendu) {
			t.Errorf("%s : rattrapes %+v, attendu %+v (vue A : %+v)", c.nom, lus, c.attendu, p.VueA.Kills)
		}
	}
}
