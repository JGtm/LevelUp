//go:build research

package divergence

// comparer.go : LA COMPARAISON D UN PAQUET. Les deux lectures sont confrontees dans l ordre du flux :
// vue A (genres des messages, debut de la vue B), vue B (records, puis composants), vue C (tours).
// Le PREMIER desaccord est rendu, avec le lecteur en cause.
//
// LE LECTEUR EN CAUSE. Quand le debut d un element differe alors que tout ce qui le precede
// coincide, c est la LARGEUR de l element precedent qui differe : le lecteur en cause est le dernier
// composant du record precedent (ou le genre du message precedent, ou le tour precedent). Quand le
// debut coincide et que la lecture elle-meme differe (genre, index, type, liste de composants, arret
// de notre marche sur un rejet), le lecteur en cause est cet element-la.
//
// LES POSITIONS sont en bits depuis le debut de la charge utile, bit 0 = drapeau initial, pour les
// deux lectures (LISEZMOI de la sonde ; `lecture` : « une position se compte en bits depuis le
// debut du payload »). Un record d entites nous donne `Debut` = le premier bit de son en-tete =
// `op_bit` du temoin (le champ d index du temoin est 1 bit apres pour une mise a jour, 3 pour une
// creation ou une suppression).

import (
	"fmt"
	"strings"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar/lecture"
)

const sansBit = -1

// desaccord : le premier point ou les deux lectures cessent de coincider. vue vide = aucun.
type desaccord struct {
	vue, lecteur       string
	bitNous, bitTemoin int
	luNous, luTemoin   string
	// record : le rang du record de la vue B ou le desaccord a ete trouve. decalage : le debut d un composant
	// differe alors que la liste des composants coincide.
	record   int
	decalage bool
}

func (d desaccord) aucun() bool { return d.vue == "" }

// contexteFilm : ce que la comparaison d un paquet sait du film.
type contexteFilm struct {
	idLowBits int
	// crees : les entites que le temoin a CREEES dans le film avant le paquet courant,
	// (index, generation) -> (chunk, paquet) de la creation.
	crees map[[2]int][2]int
}

// paquetNous : ce que notre lecture dit d un paquet : la structure et le verdict de la carte.
type paquetNous struct {
	p *lecture.Paquet
	verdictNous
}

// comparer rend le premier desaccord du paquet, vue par vue. Quand ce desaccord est un DECALAGE de
// frontiere entre deux composants d un record et que le record suivant recommence au meme bit des deux
// cotes, la largeur totale du record coincide : le decalage est RECUPERE, et la comparaison continue
// au record suivant pour rendre `suite`, le premier desaccord qui ne se rattrape pas.
func comparer(n paquetNous, t *paquetTemoin, cx *contexteFilm) (premier, suite desaccord, recupere bool) {
	if d := comparerVueA(n, t); !d.aucun() {
		return d, desaccord{}, false
	}
	d := comparerVueB(n, t, cx, 0)
	if d.aucun() {
		return comparerVueC(n, t), desaccord{}, false
	}
	if !d.decalage || !recordSuivantCoincide(n.p, t, cx, d.record) {
		return d, desaccord{}, false
	}
	suite = comparerVueB(n, t, cx, d.record+1)
	if suite.aucun() {
		suite = comparerVueC(n, t)
	}
	return d, suite, true
}

// recordSuivantCoincide : le record i+1 commence au meme bit chez les deux, ou le record i est le
// dernier des deux et la sortie de la vue B coincide.
func recordSuivantCoincide(p *lecture.Paquet, t *paquetTemoin, cx *contexteFilm, i int) bool {
	if i+1 < min(len(p.Records), len(t.ops)) {
		return int(p.Records[i+1].Debut) == t.ops[i+1].opBit
	}
	return i+1 == len(p.Records) && i+1 == len(t.ops) && sortieDeLaVueB(p, t, cx).aucun()
}

func genresDuTemoin(t *paquetTemoin) string {
	g := make([]string, 0, 8)
	for i, e := range t.evts {
		if i >= 7 {
			g = append(g, "...")
			break
		}
		g = append(g, fmt.Sprint(e.genre))
	}
	return fmt.Sprintf("%d evenements [%s]", len(t.evts), strings.Join(g, " "))
}

func genresNous(gs []uint8) string {
	g := make([]string, 0, 8)
	for i, v := range gs {
		if i >= 7 {
			g = append(g, "...")
			break
		}
		g = append(g, fmt.Sprint(v))
	}
	return fmt.Sprintf("%d genres [%s]", len(gs), strings.Join(g, " "))
}

// comparerVueA : les genres des messages, puis la fin de la vue A (= le debut de la vue B).
func comparerVueA(n paquetNous, t *paquetTemoin) desaccord {
	p := n.p
	if n.cause == grammar.CauseListeNonLocalisee {
		return desaccord{vue: "A", lecteur: "A : liste d'evenements non localisee", bitNous: sansBit,
			bitTemoin: t.entityStart, luNous: "debut de la vue B introuvable", luTemoin: genresDuTemoin(t)}
	}
	g := p.VueA.Genres
	m := min(len(g), len(t.evts))
	for i := 0; i < m; i++ {
		if int(g[i]) != t.evts[i].genre {
			return desaccord{vue: "A", lecteur: lecteurVueA(g, i), bitNous: sansBit, bitTemoin: t.evts[i].bit,
				luNous:   fmt.Sprintf("genre %d (message %d)", g[i], i),
				luTemoin: fmt.Sprintf("genre %d (message %d)", t.evts[i].genre, i)}
		}
	}
	if p.VueB.Sortie == lecture.SortieNonAtteinte && p.VueA.Etat == lecture.VueArretee && len(g) > 0 && len(g) <= len(t.evts) {
		i := len(g) - 1
		return desaccord{vue: "A", lecteur: fmt.Sprintf("A : genre %d non porte", g[i]), bitNous: sansBit,
			bitTemoin: t.evts[i].bit, luNous: fmt.Sprintf("arret sur le genre %d", g[i]),
			luTemoin: fmt.Sprintf("genre %d lu", t.evts[i].genre)}
	}
	if p.VueB.Sortie == lecture.SortieNonAtteinte {
		return desaccord{vue: "A", lecteur: "A : vue B non atteinte (" + n.cause + ")", bitNous: sansBit,
			bitTemoin: t.entityStart, luNous: genresNous(g), luTemoin: genresDuTemoin(t)}
	}
	if len(g) != len(t.evts) && p.VueA.Etat == lecture.VueTerminee {
		return desaccord{vue: "A", lecteur: lecteurVueA(g, len(g)), bitNous: int(p.VueB.Debut), bitTemoin: t.entityStart,
			luNous: genresNous(g), luTemoin: genresDuTemoin(t)}
	}
	if t.entityStart < 0 {
		return desaccord{vue: "A", lecteur: "A : le temoin ne termine pas la vue des evenements", bitNous: int(p.VueB.Debut),
			bitTemoin: sansBit, luNous: genresNous(g), luTemoin: "event_stop=" + t.eventStop}
	}
	if int(p.VueB.Debut) != t.entityStart {
		return desaccord{vue: "A", lecteur: lecteurVueA(g, len(g)), bitNous: int(p.VueB.Debut), bitTemoin: t.entityStart,
			luNous:   fmt.Sprintf("vue B au bit %d (debut par %d)", p.VueB.Debut, p.Debut),
			luTemoin: genresDuTemoin(t)}
	}
	return desaccord{}
}

// lecteurVueA nomme le lecteur d un desaccord a la position du message i : la largeur du message
// precedent (son genre), ou la tete quand c est le premier.
func lecteurVueA(g []uint8, i int) string {
	if i == 0 || len(g) == 0 {
		return "A : tete de la vue (premier message)"
	}
	return fmt.Sprintf("A : largeur du message genre %d", g[min(i, len(g))-1])
}

// comparerVueC : les tours de la vue de controle, puis sa fin. Sans vue C chez le temoin (arret de
// sa marche avant elle), il n y a rien a confronter.
func comparerVueC(n paquetNous, t *paquetTemoin) desaccord {
	p := n.p
	if t.stop != "aucun" {
		return desaccord{}
	}
	j := 0
	for i, e := range p.VueC.Entrees {
		prochain := sansBit
		if j < len(t.msgs) {
			prochain = t.msgs[j].bit
		}
		if e.Kind != 0 {
			return desaccord{vue: "C", lecteur: fmt.Sprintf("C : kind %d non porte", e.Kind), bitNous: int(e.Debut),
				bitTemoin: prochain, luNous: fmt.Sprintf("tour %d : kind %d, arret", i, e.Kind),
				luTemoin: fmt.Sprintf("%d messages lus, prochain a %d", len(t.msgs), prochain)}
		}
		if j >= len(t.msgs) {
			return desaccord{vue: "C", lecteur: lecteurVueC(p.VueC.Entrees, i), bitNous: int(e.Debut), bitTemoin: sansBit,
				luNous: fmt.Sprintf("tour %d : joueur %d", i, e.Index), luTemoin: "aucun message de plus"}
		}
		m := t.msgs[j]
		if int(e.Debut) != m.bit || int(e.Index) != m.joueur {
			return desaccord{vue: "C", lecteur: lecteurVueC(p.VueC.Entrees, i), bitNous: int(e.Debut), bitTemoin: m.bit,
				luNous:   fmt.Sprintf("tour %d : joueur %d", i, e.Index),
				luTemoin: fmt.Sprintf("message %d : joueur %d", j, m.joueur)}
		}
		j++
	}
	fin := int(p.VueC.Debut + p.VueC.Bits)
	if j < len(t.msgs) {
		m := t.msgs[j]
		return desaccord{vue: "C", lecteur: lecteurVueC(p.VueC.Entrees, len(p.VueC.Entrees)), bitNous: fin, bitTemoin: m.bit,
			luNous:   fmt.Sprintf("vue C finie au bit %d (%d tours)", fin, len(p.VueC.Entrees)),
			luTemoin: fmt.Sprintf("message %d : joueur %d", j, m.joueur)}
	}
	if p.VueC.Etat == lecture.VueTerminee && t.ctrlFin >= 0 && fin != t.ctrlFin {
		return desaccord{vue: "C", lecteur: lecteurVueC(p.VueC.Entrees, len(p.VueC.Entrees)) + " / terminateur",
			bitNous: fin, bitTemoin: t.ctrlFin, luNous: "fin de la vue C", luTemoin: "fin de la vue C"}
	}
	return desaccord{}
}

// lecteurVueC : le tour precedent le point de desaccord.
func lecteurVueC(es []lecture.EntreeVueC, i int) string {
	if i == 0 || len(es) == 0 {
		return "C : debut de la vue (apres la vue B)"
	}
	return fmt.Sprintf("C : largeur du tour kind %d", es[min(i, len(es))-1].Kind)
}
