//go:build research

package divergence

// temoin.go : lecture en FLUX du TSV du temoin libfilmshell (sonde `oracle-filmshell`, plan marche
// complete A0.6). Un film pese 50 a 350 Mo : une ligne a la fois, un paquet a la fois, jamais le
// fichier entier en memoire. Les colonnes et les positions de bit sont celles du LISEZMOI de la
// sonde : `bit` est RELATIF a la charge utile du paquet, bit 0 = drapeau initial.

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
)

// evtTemoin : un evenement de la vue A (bit du champ de type, apres le drapeau de continuation).
type evtTemoin struct{ bit, genre int }

// opTemoin : une entree de la vue B. `opBit` est le debut de l entree (bits d operation compris) ;
// `bit` le champ d index. `compo` et `blocs` : la liste des composants et le bit de debut de chacun.
type opTemoin struct {
	op         string // create, update, delete
	opBit, bit int
	index, gen int
	typeEntite string
	compo      []int
	blocBit    []int // bit du premier bit de donnees de chaque composant, meme ordre que compo
	initAbsent bool
}

// msgTemoin : un message de la vue C rendu par la bibliotheque (bit du drapeau [more:1]).
type msgTemoin struct{ bit, joueur int }

// paquetTemoin : tout ce que le temoin a rendu d UN paquet.
type paquetTemoin struct {
	chunk, paquet, tick int
	rang0, taille       int
	tsUS                uint64
	eventStop, stop     string
	reprises            int
	repriseCause        string // cause de la PREMIERE reprise (`repriseN=<cause>@<depart>><reprise>`)
	repriseDepart       int    // bit de depart de la premiere reprise, -1 sans reprise
	entityStart, end    int    // -1 : `-` dans le TSV
	ctrlValide          bool
	ctrlFin             int
	ctrlFinDernierOctet bool
	evts                []evtTemoin
	ops                 []opTemoin
	msgs                []msgTemoin
}

// ferme : le temoin a lu le paquet jusqu a son dernier octet sans arret de la marche.
func (p *paquetTemoin) ferme() bool { return p.ctrlValide && p.ctrlFinDernierOctet }

// lecteurTemoin rend les paquets du TSV dans l ordre du fichier.
type lecteurTemoin struct {
	f       *os.File
	r       *bufio.Reader
	attente []string // ligne P deja lue du paquet suivant
	ligne   int
}

func ouvrirTemoin(chemin string) (*lecteurTemoin, error) {
	f, err := os.Open(chemin) //nolint:gosec // chemin de l operateur
	if err != nil {
		return nil, err
	}
	l := &lecteurTemoin{f: f, r: bufio.NewReaderSize(f, 1<<20)}
	if _, err := l.lireLigne(); err != nil { // en-tete de colonnes
		_ = f.Close()
		return nil, fmt.Errorf("temoin %s : en-tete illisible : %w", chemin, err)
	}
	return l, nil
}

func (l *lecteurTemoin) fermer() { _ = l.f.Close() }

// lireLigne rend les colonnes de la ligne suivante ; io.EOF a la fin.
func (l *lecteurTemoin) lireLigne() ([]string, error) {
	s, err := l.r.ReadString('\n')
	if err != nil && (err != io.EOF || s == "") {
		return nil, err
	}
	l.ligne++
	return strings.Split(strings.TrimRight(s, "\r\n"), "\t"), nil
}

// suivant rend le paquet suivant du fichier, nil a la fin.
func (l *lecteurTemoin) suivant() (*paquetTemoin, error) {
	cols := l.attente
	l.attente = nil
	for cols == nil {
		c, err := l.lireLigne()
		if err == io.EOF {
			return nil, nil
		}
		if err != nil {
			return nil, err
		}
		if len(c) >= 15 && c[6] == "P" {
			cols = c
		}
	}
	p, err := nouveauPaquetTemoin(cols)
	if err != nil {
		return nil, fmt.Errorf("ligne %d : %w", l.ligne, err)
	}
	for {
		c, err := l.lireLigne()
		if err == io.EOF {
			return p, nil
		}
		if err != nil {
			return nil, err
		}
		if len(c) < 15 {
			continue
		}
		if c[6] == "P" {
			l.attente = c
			return p, nil
		}
		if err := p.ajouter(c); err != nil {
			return nil, fmt.Errorf("ligne %d : %w", l.ligne, err)
		}
	}
}

func atoi(s string) int {
	n, err := strconv.Atoi(s)
	if err != nil {
		return -1
	}
	return n
}

// champs decoupe un detail `cle=valeur;cle=valeur`.
func champs(detail string) map[string]string {
	m := map[string]string{}
	for _, kv := range strings.Split(detail, ";") {
		if k, v, ok := strings.Cut(kv, "="); ok {
			m[k] = v
		}
	}
	return m
}

func nouveauPaquetTemoin(c []string) (*paquetTemoin, error) {
	d := champs(c[14])
	p := &paquetTemoin{
		chunk: atoi(c[1]), paquet: atoi(c[2]), tick: atoi(c[5]),
		rang0: atoi(d["rang0"]), taille: atoi(d["taille"]),
		eventStop: d["event_stop"], stop: d["stop"],
		entityStart: atoi(d["entity_start"]), end: atoi(d["end"]),
		ctrlValide:          d["ctrl_valide"] == "oui",
		ctrlFin:             atoi(d["ctrl_fin"]),
		ctrlFinDernierOctet: d["ctrl_fin_dernier_octet"] == "oui",
	}
	if us, err := strconv.ParseUint(d["ts_us"], 10, 64); err == nil {
		p.tsUS = us
	}
	p.reprises, p.repriseDepart = atoi(d["reprises"]), -1
	if v, ok := d["reprise0"]; ok {
		cause, pos, _ := strings.Cut(v, "@")
		depart, _, _ := strings.Cut(pos, ">")
		p.repriseCause, p.repriseDepart = cause, atoi(depart)
	}
	if p.taille < 0 || p.chunk < 0 || p.paquet < 0 {
		return nil, fmt.Errorf("ligne P incomplete : %q", strings.Join(c, "\t"))
	}
	return p, nil
}

func (p *paquetTemoin) ajouter(c []string) error {
	bit := atoi(c[7])
	switch c[6] {
	case "A":
		p.evts = append(p.evts, evtTemoin{bit: bit, genre: atoi(c[8])})
	case "B":
		d := champs(c[14])
		o := opTemoin{op: c[8], bit: bit, opBit: atoi(d["op_bit"]), index: atoi(c[9]), gen: atoi(c[10]),
			typeEntite: c[11], initAbsent: d["init"] == "absent"}
		if c[12] != "" {
			for _, s := range strings.Split(c[12], ",") {
				o.compo = append(o.compo, atoi(s))
			}
		}
		if c[13] != "" {
			for _, s := range strings.Split(c[13], ",") {
				_, b, _ := strings.Cut(s, ":")
				o.blocBit = append(o.blocBit, atoi(b))
			}
		}
		for len(o.blocBit) < len(o.compo) { // blocs non rendus apres un echec de lecture du temoin
			o.blocBit = append(o.blocBit, -1)
		}
		p.ops = append(p.ops, o)
	case "C":
		d := champs(c[14])
		p.msgs = append(p.msgs, msgTemoin{bit: bit, joueur: atoi(d["joueur"])})
	}
	return nil
}
