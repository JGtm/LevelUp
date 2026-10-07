//go:build research

package grammar

// ri27c_vue_a_research_test.go — LES MESURES DE LA VUE A QUI OUVRENT 2.7.c (plan de l etape 2 de la
// representation intermediaire, item 2.7.c0), suite de `ri27c_killsource_research_test.go` (memes
// variables, meme contexte, meme dossier de sortie). Aucun fichier de production n est touche.
//
//	TestRI27cVueA          la vue A unique de chaque trame a evenements (portee ou non, nombre de
//	                       messages), ses messages de genre 85 avec la position de leur genre, leurs
//	                       champs et l acceptation de leur charge ; ce que la variante de partie du
//	                       film decide de la queue du 85.
//	TestRI27cVueASansQueue la vue A lue avec un genre 85 SANS queue, confrontee au debut de vue B que
//	                       la marche localise sans elle (signature, chaine, fermeture) dans les
//	                       trames fermees.

import (
	"fmt"
	"path/filepath"
	"runtime"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// ri27cKill est un message de genre 85 lu dans la vue A unique.
type ri27cKill struct {
	bitGenre                     int
	victime, tueur, assistant    int
	partTueur, drapeau, partAide uint64
	accepte                      bool
}

// ri27cRef5 lit une reference d entite `FUN_1407f2058` : porte a 1, absente (-1) ; a 0, R(5).
func ri27cRef5(br *Lecteur) int {
	if br.ReadBit() {
		return -1
	}
	return int(br.ReadBits(5))
}

// ri27cChampsDuKill lit les champs d un message de genre 85 dont le genre finit a `finDuGenre` :
// les trois references gardees du descripteur, puis le corps de `FUN_14104bd08` sans sa queue.
func ri27cChampsDuKill(pay []byte, finDuGenre int, bal ProfilDeBalayage, g grammaireDeLaVueA) ri27cKill {
	br := LecteurSur(pay)
	br.PoserProfil(bal)
	br.vueA = g
	br.SetBitPos(finDuGenre)
	domaines, _ := descripteurDuGenre(ri27cGenreKill)
	for _, dom := range domaines {
		if br.ReadBit() {
			readVarWidthInt(br, dom)
		}
	}
	k := ri27cKill{bitGenre: finDuGenre - LargeurGenreVueA}
	k.victime, k.tueur = ri27cRef5(br), ri27cRef5(br)
	k.partTueur, k.drapeau = br.ReadBits(32), br.ReadBits(1)
	k.assistant, k.partAide = ri27cRef5(br), br.ReadBits(32)
	return k
}

// ri27cLireLaVueA est [lireLaVueA], plus les messages de genre 85 : leur position, leurs champs et
// l acceptation de leur charge par la lecture unique.
func ri27cLireLaVueA(pay []byte, bal ProfilDeBalayage, g grammaireDeLaVueA) (FluxVueA, []ri27cKill) {
	br := LecteurSur(pay)
	br.PoserProfil(bal)
	br.vueA = g
	frameLen := len(pay) * 8
	lisible := false
	if frameLen >= 1 {
		br.SetBitPos(0)
		lisible = br.ReadBit() && g.classe != vueAIllisible
	}
	out := FluxVueA{Debut: 1}
	var kills []ri27cKill
	br.SetBitPos(1)
	for placeDisponible(br, frameLen, 1) {
		if !br.ReadBit() {
			out.Porte, out.Vide = true, len(out.Genres) == 0
			break
		}
		if !placeDisponible(br, frameLen, LargeurGenreVueA) {
			break
		}
		genre := int(br.ReadBits(LargeurGenreVueA))
		out.Genres = append(out.Genres, genre)
		finDuGenre := br.BitPos()
		var k *ri27cKill
		if genre == ri27cGenreKill {
			kk := ri27cChampsDuKill(pay, finDuGenre, bal, g)
			k = &kk
		}
		ok := lisible && lireUnMessage(br, genre) && !br.Deborde()
		if k != nil {
			k.accepte = ok
			kills = append(kills, *k)
		}
		if !ok {
			br.SetBitPos(finDuGenre)
			break
		}
	}
	out.Fin = br.BitPos()
	return out, kills
}

// TestRI27cVueA lit la vue A unique de chaque trame a evenements et en rend les messages de genre 85.
func TestRI27cVueA(t *testing.T) {
	films, racine, sortie := ri27cEnv(t)
	for _, court := range films {
		fc := ri27cContexte(t, filepath.Join(racine, court), court, false)
		g, bal := fc.grammaireDeLaVueA(), fc.ProfilDeBalayage()
		v := g.variante
		lignes := []string{fmt.Sprintf("V\t%s\t%v\t%v\t%v\t%d\t%d", court, v.lue, v.moteurUn,
			v.queueDuKillPossible, g.classe, g.genres)}
		for _, c := range fc.ChunkNumbers() {
			data, pks, ok := fc.ChunkAt(c)
			if !ok {
				continue
			}
			for _, pk := range pks {
				if pk.Type != PacketTypeDelta || pk.Size < 1 {
					continue
				}
				pay := pk.Payload(data)
				if source.BitAt(pay, 1) == 0 {
					continue
				}
				a, kills := ri27cLireLaVueA(pay, bal, g)
				lignes = append(lignes, fmt.Sprintf("P\t%s\t%d\t%d\t%d\t%v\t%d\t%d", court, c, pk.Index,
					pk.TimestampUS, a.Porte, len(a.Genres), a.Fin))
				for _, k := range kills {
					lignes = append(lignes, fmt.Sprintf("K\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%v", court, c,
						pk.Index, pk.TimestampUS, k.bitGenre, k.victime, k.tueur, k.partTueur, k.drapeau, k.assistant,
						k.partAide, k.accepte))
				}
			}
		}
		ri27cEcrire(t, filepath.Join(sortie, "vuea_"+court+".tsv"), lignes)
		t.Logf("%s : variante lue %v, queue du 85 possible %v, classe %d", court, v.lue, v.queueDuKillPossible,
			g.classe)
		runtime.GC()
	}
}

// ri27cCanalSansQueue confronte, trame par trame, la fin de la vue A lue avec un genre 85 SANS queue
// (la garde de l ecrivain supposee fausse) au debut de la vue B que la marche a trouve par un
// localisateur (signature, chaine, fermeture), dans les trames fermees : si l ecrivain n a pas ecrit
// de queue, la vue A ainsi lue finit exactement la ou la vue B commence.
type ri27cCanalSansQueue struct {
	fc    *FilmContext
	g     grammaireDeLaVueA
	bal   ProfilDeBalayage
	court string
	// comptes : trames a genre 85, portees sans queue, dont le debut de vue B est localise et le paquet
	// ferme, dont la fin de vue A tombe sur ce debut ; et la meme chose quand le debut est lu
	// (la tete) — temoin du lecteur.
	avec85, portees, localiseesFermees, egales, differentes int
	// chaineVersLeDebut : fin de vue A avant le debut localise, et une chaine de records lisibles
	// ([pasDEssai]) mene de l une a l autre au bit pres ; apres : fin de vue A au-dela du debut localise.
	chaineVersLeDebut, apres int
	m                        *MarcheDistribuee
	lignes                   []string
}

func (c *ri27cCanalSansQueue) Interets() []Interet                          { return nil }
func (c *ri27cCanalSansQueue) Clore(BilanDeMarche)                          {}
func (c *ri27cCanalSansQueue) Brancher(_ *Observation, m *MarcheDistribuee) { c.m = m }

func (c *ri27cCanalSansQueue) Trame(p *lecture.Paquet) {
	if !listeAnnoncee(&p.VueA) || !slicesContient(p.VueA.Genres, ri27cGenreKill) {
		return
	}
	c.avec85++
	a, _ := ri27cLireLaVueA(p.Payload, c.bal, c.g)
	if !a.Porte {
		return
	}
	c.portees++
	if p.Fermeture.Verdict != lecture.VerdictFerme || !debutLocalise(p.Debut) {
		return
	}
	c.localiseesFermees++
	if uint32(a.Fin) == p.VueB.Debut { //nolint:gosec // position dans un payload
		c.egales++
		return
	}
	c.differentes++
	if a.Fin < int(p.VueB.Debut) {
		if ri27cChaineJusqua(p.Payload, a.Fin, int(p.VueB.Debut), c.m) {
			c.chaineVersLeDebut++
		}
	} else {
		c.apres++
	}
	if len(c.lignes) < 40 {
		c.lignes = append(c.lignes, fmt.Sprintf("X\t%s\t%d\t%d\t%d\t%d\t%d", c.court, p.Chunk, p.Index, a.Fin,
			p.VueB.Debut, p.Debut))
	}
}

// slicesContient dit si le genre `g` est parmi les genres lus.
func slicesContient(genres []uint8, g int) bool {
	for _, x := range genres {
		if int(x) == g {
			return true
		}
	}
	return false
}

// TestRI27cVueASansQueue mesure la regle « le 85 n a pas de queue » contre les debuts de vue B que
// la marche localise sans la vue A.
func TestRI27cVueASansQueue(t *testing.T) {
	films, racine, sortie := ri27cEnv(t)
	var lignes []string
	for _, court := range films {
		fc := ri27cContexte(t, filepath.Join(racine, court), court, true)
		g := fc.grammaireDeLaVueA()
		g.variante.lue, g.variante.queueDuKillPossible = true, false
		c := &ri27cCanalSansQueue{fc: fc, g: g, bal: fc.ProfilDeBalayage(), court: court}
		if err := Distribuer(fc, c); err != nil {
			t.Fatalf("%s : distribution : %v", court, err)
		}
		lignes = append(lignes, fmt.Sprintf("S\t%s\t%d\t%d\t%d\t%d\t%d", court, c.avec85, c.portees,
			c.localiseesFermees, c.egales, c.differentes))
		lignes = append(lignes, c.lignes...)
		t.Logf("%s : trames a 85 %d, portees sans queue %d, localisees et fermees %d : fin de vue A = debut de "+
			"vue B %d, differente %d (chaine vers le debut %d, au-dela %d)", court, c.avec85, c.portees, c.localiseesFermees, c.egales,
			c.differentes, c.chaineVersLeDebut, c.apres)
		runtime.GC()
	}
	ri27cEcrire(t, filepath.Join(sortie, "vuea_sans_queue.tsv"), lignes)
}

// ri27cChaineJusqua dit si une chaine de records lisibles ([pasDEssai], cadre et monde de la marche)
// mene de `de` a `a` au bit pres.
func ri27cChaineJusqua(pay []byte, de, a int, m *MarcheDistribuee) bool {
	cfg := m.marche.cfg
	cfg.Obs = nil
	extra := motFacultatifDEnTete(cfg)
	pos := de
	for range 64 {
		if pos == a {
			return true
		}
		if pos > a {
			return false
		}
		suivant, ok := pasDEssai(pay, pos, extra, m.marche.monde, cfg)
		if !ok || suivant <= pos {
			return false
		}
		pos = suivant
	}
	return false
}
