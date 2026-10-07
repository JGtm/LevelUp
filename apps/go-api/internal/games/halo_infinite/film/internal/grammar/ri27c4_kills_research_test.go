//go:build research

package grammar

// ri27c4_kills_research_test.go — LA MESURE DE 2.7.c4 (plan de l etape 2 de la representation
// intermediaire) : les messages de kill que killsource prend — ceux que la vue A lit, puis ceux que le
// rattrapage retrouve dans les trames ou sa lecture n est pas etablie — contre ceux que sa recherche
// bit a bit d avant trouvait dans toute trame a evenements (depuis le bit 1, `gate15` tranche par film
// par la meme regle). Les trames sont MARCHEES (le canal des morts accompagne la mesure, comme dans
// [LireLaMarcheDeKillsource]) : le debut de la vue B est pose. Aucun fichier de production n est
// touche.
//
// Chaque kill de l une ou l autre lecture est range avec sa trame et sa place :
//
//	commun_vue_A / commun_rattrape          trouve par les deux lectures, a la meme position (champs
//	                                        egaux, sinon le suffixe `_champs_differents`)
//	ancien_seul_lecture_etablie_dans_la_vue_A_lue
//	ancien_seul_lecture_etablie_apres_le_terminateur
//	                                        la recherche d avant seule, dans une trame dont la vue B
//	                                        commence a la fin de la vue A lue : dans l etendue lue, hors
//	                                        du debut d un message (decouverte 30), ou apres elle
//	ancien_seul_apres_le_debut_de_la_vue_B  la recherche d avant seule, au-dela de la fenetre du
//	                                        rattrapage
//	ancien_seul_dans_la_fenetre             la recherche d avant seule, dans la fenetre du rattrapage
//	                                        (anomalie)
//	ancien_seul_sans_liste                  la recherche d avant seule, dans une trame dont la vue A
//	                                        n annonce aucune liste (anomalie)
//	nouveau_vue_A / nouveau_rattrape        la lecture neuve seule ; la colonne `chaine` dit la longueur
//	                                        de la chaine d evenements que la recherche d avant exigeait
//	                                        (3)
//
// Une ligne `K` : film, position du chunk, rang, horodatage, debut du message, classe, victime, tueur,
// assistant, deux parts, drapeau, chaine, etat de la vue A, fin de la vue A, debut de la vue B (et
// comment il a ete trouve).
//
//	RI27C_FILMS=<id,...> RI27C_RACINE=<film_chunks> RI27C_OUT=<dossier> [RI27C_CARTES=<id=Carte;...>] \
//	  go test -tags=research -count=1 -run '^TestRI27c4Kills$' -timeout 120m \
//	  ./internal/games/halo_infinite/film/internal/grammar/

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sort"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// ri27c4Canal confronte, trame par trame, les deux lectures des messages de kill.
type ri27c4Canal struct {
	fc         *FilmContext
	court      string
	gate15     bool
	rattrapage rattrapageDesKills
	lignes     []string
	classes    map[string]int
}

func (c *ri27c4Canal) Interets() []Interet { return nil }
func (c *ri27c4Canal) Clore(BilanDeMarche) {}

// ri27c4Kill est un message de kill d une des deux lectures.
type ri27c4Kill struct {
	k         lecture.MessageDeKill
	rattrape  bool
	chaine    int
	apparie   bool
	doublonDe int
}

// Tete confronte les deux lectures de la trame marchee `p`.
func (c *ri27c4Canal) Tete(p *lecture.Paquet) {
	var neufs []ri27c4Kill
	for _, k := range p.VueA.Kills {
		neufs = append(neufs, ri27c4Kill{k: k, doublonDe: -1})
	}
	for _, k := range c.rattrapage.rattraper(p) {
		neufs = append(neufs, ri27c4Kill{k: k.Kill, rattrape: true, chaine: k.Chaine, doublonDe: -1})
	}
	var anciens []ri27c4Kill
	if source.BitAt(p.Payload, 1) != 0 {
		entiere := fenetreDeRecherche{depuis: 1, jusqua: len(p.Payload) * 8}
		ks, _ := killsAvecArrets(p.Payload, entiere, c.gate15)
		for _, k := range ks {
			anciens = append(anciens, ri27c4Kill{k: k.Kill, chaine: k.Chaine, doublonDe: -1})
		}
	}
	marquerLesDoublons(anciens)
	marquerLesDoublons(neufs)
	parDebut := map[uint32]int{}
	for i := range neufs {
		parDebut[neufs[i].k.Debut] = i
	}
	for i := range anciens {
		a := &anciens[i]
		classe := ""
		if j, ok := parDebut[a.k.Debut]; ok {
			n := &neufs[j]
			n.apparie = true
			classe = map[bool]string{false: "commun_vue_A", true: "commun_rattrape"}[n.rattrape]
			if n.k != a.k {
				classe += "_champs_differents"
			}
		} else {
			classe = "ancien_seul_" + ri27c4Place(p, a.k.Debut)
		}
		c.ranger(p, a, classe)
	}
	for i := range neufs {
		n := &neufs[i]
		if n.apparie {
			continue
		}
		n.chaine = ri27c4ChaineDAvant(p.Payload, n.k.Debut, c.gate15)
		c.ranger(p, n, map[bool]string{false: "nouveau_vue_A", true: "nouveau_rattrape"}[n.rattrape])
	}
}

// marquerLesDoublons note, pour chaque kill d une trame, le premier kill de memes champs qui le
// precede (la regle de dedoublonnage de killsource).
func marquerLesDoublons(ks []ri27c4Kill) {
	for i := range ks {
		for j := range i {
			a, b := ks[i].k, ks[j].k
			a.Debut, b.Debut = 0, 0
			if a == b {
				ks[i].doublonDe = j
				break
			}
		}
	}
}

// ri27c4Place dit ou tombe, par rapport a la lecture de la vue A de sa trame et a la fenetre du
// rattrapage, un kill que seule la recherche d avant trouve.
func ri27c4Place(p *lecture.Paquet, debut uint32) string {
	if !listeAnnoncee(&p.VueA) {
		return "sans_liste"
	}
	if p.Debut == lecture.DebutParVueA {
		if debut < p.VueA.Debut+p.VueA.Bits {
			return "lecture_etablie_dans_la_vue_A_lue"
		}
		return "lecture_etablie_apres_le_terminateur"
	}
	if f, _ := fenetreDuRattrapage(p); int(debut) >= f.jusqua {
		return "apres_le_debut_de_la_vue_B"
	}
	return "dans_la_fenetre"
}

// ri27c4ChaineDAvant rend la longueur de chaine que la recherche d avant mesurait derriere le kill
// qui commence au bit `debut`, -1 quand elle ne l ouvrait pas (boucle de presence illisible, kill
// implausible).
func ri27c4ChaineDAvant(pl []byte, debut uint32, gate15 bool) int {
	x := int(debut) + 1
	if !estAncreDeKill(pl, x) {
		return -1
	}
	r := nouveauCurseurEv(pl, x+LargeurGenreVueA)
	if !evPresence(r, GenreJoueurTue) {
		return -1
	}
	k, fin := lireLeKillDeLaChaine(pl, r.pos())
	if !killPlausible(k, fin) {
		return -1
	}
	n, _ := evChainLenAvecVerdict(pl, fin, gate15, maxChainProbe)
	return n
}

// ranger ecrit la ligne d un kill et le compte de sa classe.
func (c *ri27c4Canal) ranger(p *lecture.Paquet, k *ri27c4Kill, classe string) {
	if k.doublonDe >= 0 {
		classe += "_doublon"
	}
	c.classes[classe]++
	m := k.k
	c.lignes = append(c.lignes, fmt.Sprintf("K\t%s\t%d\t%d\t%d\t%d\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d",
		c.court, filmChunkPos(c.fc.Film(), p.Chunk), p.Index, p.TS, m.Debut, classe, m.Victime, m.Tueur,
		m.Assistant, m.PartDuTueur, m.PartDeLAssistant, m.Drapeau, k.chaine, p.VueA.Etat,
		p.VueA.Debut+p.VueA.Bits, p.VueB.Debut, p.Debut))
}

func TestRI27c4Kills(t *testing.T) {
	films, racine, sortie := ri27cEnv(t)
	var lignes []string
	for _, court := range films {
		fc := ri27cContexte(t, filepath.Join(racine, court), court, true)
		reg, err := fc.Registry()
		if err != nil {
			t.Fatalf("%s : registre : %v", court, err)
		}
		c := &ri27c4Canal{fc: fc, court: court, gate15: trancherGate15(fc.Film()),
			rattrapage: rattrapageDesKills{film: fc.Film()}, classes: map[string]int{}}
		if err := Distribuer(fc, nouveauCanalDesMorts(reg), c); err != nil {
			t.Fatalf("%s : distribution : %v", court, err)
		}
		noms := make([]string, 0, len(c.classes))
		for n := range c.classes {
			noms = append(noms, n)
		}
		sort.Strings(noms)
		for _, n := range noms {
			lignes = append(lignes, fmt.Sprintf("S\t%s\t%s\t%d", court, n, c.classes[n]))
		}
		lignes = append(lignes, c.lignes...)
		t.Logf("%s : gate15=%v, gate15 du rattrapage=%v (tranche=%v) ; %v", court, c.gate15,
			c.rattrapage.gate15, c.rattrapage.tranche, c.classes)
		runtime.GC()
	}
	ri27cEcrire(t, filepath.Join(sortie, "kills_c4.tsv"), lignes)
}
