//go:build research

package divergence

// film.go : UN film. Notre marche (structure de lecture, paquet par paquet) et le flux du temoin
// avancent ensemble ; les paquets s apparient par (chunk, rang du paquet dans le chunk, tous types
// confondus), verifies par la taille de la charge utile et l horodatage. Les paquets de moins de 2
// octets ne sont pas marches par le temoin (LISEZMOI) : ils sont comptes, pas compares.

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

const entetePaquets = "chunk\tpaquet\ttaille\tfenetre_ok\tnous_ferme\ttemoin_ferme\ttemoin_reprises\ttemoin_arret\tcause_nous\t" +
	"vue\tbit_nous\tbit_temoin\tlu_nous\tlu_temoin\tlecteur_en_cause\tdecalage_recupere\t" +
	"suite_vue\tsuite_bit_nous\tsuite_bit_temoin\tsuite_lu_nous\tsuite_lu_temoin\tsuite_lecteur\n"

func oui(b bool) string {
	if b {
		return "oui"
	}
	return "non"
}

// traiterUnFilm compare nos paquets a ceux du temoin et ecrit divergence_paquets.tsv du film.
func traiterUnFilm(cfg config, id, build string, g *agregat) error {
	fc, err := ouvrirFilm(cfg.racine, id, cfg.niveau, cfg.carte)
	if err != nil {
		return err
	}
	verdicts, err := lireLesVerdicts(fc)
	if err != nil {
		return err
	}
	tem, err := ouvrirTemoin(filepath.Join(cfg.temoin, id+".tsv"))
	if err != nil {
		return err
	}
	defer tem.fermer()
	dir := filepath.Join(cfg.sortie, id)
	if err := os.MkdirAll(dir, 0o755); err != nil { //nolint:gosec // dossier de sortie de l operateur
		return err
	}
	out, err := os.Create(filepath.Join(dir, "divergence_paquets.tsv")) //nolint:gosec // idem
	if err != nil {
		return err
	}
	defer func() { _ = out.Close() }()
	w := bufio.NewWriterSize(out, 1<<20)
	defer func() { _ = w.Flush() }()
	_, _ = w.WriteString(entetePaquets)

	f := &statsFilm{id: id, build: build, classement: map[cleClassement]int{}, decalages: map[string]int{}, nousSeulsParTemoin: map[string]int{}}
	cx := &contexteFilm{idLowBits: fc.EnTete().IDLowBits.Valeur, crees: map[[2]int][2]int{}}
	var cur *paquetTemoin
	suivant := func() error {
		var err error
		cur, err = tem.suivant()
		return err
	}
	if err := suivant(); err != nil {
		return err
	}
	for p, err := range fc.Trames(grammar.NouvelleObservation()) {
		if err != nil {
			return err
		}
		if len(p.Payload) < 2 {
			f.nonMarches++
			continue
		}
		for cur != nil && (cur.chunk < p.Chunk || cur.chunk == p.Chunk && cur.paquet < p.Index) {
			f.temoinSeul++
			cx.appliquer(cur)
			if err := suivant(); err != nil {
				return err
			}
		}
		v := verdicts[cleDePaquet{p.Chunk, p.Index}]
		n := paquetNous{p: p, verdictNous: v}
		if cur == nil || cur.chunk != p.Chunk || cur.paquet != p.Index {
			f.sansTemoin++
			fmt.Fprintf(w, "%d\t%d\t%d\t-\t%s\t-\t-\t-\t%s\tsans temoin\t-\t-\t-\t-\t-\t-\t-\t-\t-\t-\t-\t-\n", p.Chunk, p.Index, len(p.Payload), oui(v.fermee), v.cause)
			continue
		}
		fenetreOK := cur.taille == len(p.Payload) && cur.tsUS == p.TS
		d, suite, recupere := comparer(n, cur, cx)
		f.compter(n, cur, d, suite, recupere, fenetreOK, g)
		ecrireLigne(w, n, cur, d, suite, recupere, fenetreOK)
		if eff := effectif(d, suite, recupere); strings.HasPrefix(eff.lecteur, "B : sortie par rejet") && len(f.exemplesRejet) < exemplesParFilm {
			f.exemplesRejet = append(f.exemplesRejet, exempleRejet(id, n, eff))
		}
		cx.appliquer(cur)
		if err := suivant(); err != nil {
			return err
		}
	}
	for cur != nil {
		f.temoinSeul++
		if err := suivant(); err != nil {
			return err
		}
	}
	if err := w.Flush(); err != nil {
		return err
	}
	return f.ecrire(dir, g)
}

// appliquer range les creations et suppressions du paquet dans la table des entites du temoin.
func (cx *contexteFilm) appliquer(t *paquetTemoin) {
	for _, o := range t.ops {
		switch o.op {
		case "create":
			cx.crees[[2]int{o.index, o.gen}] = [2]int{t.chunk, t.paquet}
		case "delete":
			delete(cx.crees, [2]int{o.index, o.gen})
		}
	}
}

func ecrireLigne(w *bufio.Writer, n paquetNous, t *paquetTemoin, d, suite desaccord, recupere, fenetreOK bool) {
	arret := t.stop
	if t.repriseDepart >= 0 {
		arret += "+reprise:" + t.repriseCause
	}
	fmt.Fprintf(w, "%d\t%d\t%d\t%s\t%s\t%s\t%d\t%s\t%s\t%s\t%s\t%s\n", n.p.Chunk, n.p.Index, len(n.p.Payload),
		oui(fenetreOK), oui(n.fermee), oui(t.ferme()), t.reprises, arret, n.cause, colonnes(d), oui(recupere), colonnes(suite))
}

// colonnes : vue, bit nous, bit temoin, lu nous, lu temoin, lecteur (tirets quand il n y a pas de desaccord).
func colonnes(d desaccord) string {
	if d.aucun() {
		return "-\t-\t-\t-\t-\t-"
	}
	return fmt.Sprintf("%s\t%d\t%d\t%s\t%s\t%s", d.vue, d.bitNous, d.bitTemoin, nettoyer(d.luNous), nettoyer(d.luTemoin), nettoyer(d.lecteur))
}

// effectif : le premier desaccord qui ne se rattrape pas.
func effectif(d, suite desaccord, recupere bool) desaccord {
	if recupere {
		return suite
	}
	return d
}

func nettoyer(s string) string {
	return strings.NewReplacer("\t", " ", "\n", " ", "\r", " ").Replace(s)
}
