package main

// cmd_replay_parc_vies_comparer.go — `levelup replay-parc-vies --comparer <avant> <apres>` : CE
// QU'UN CORRECTIF DU NOMMAGE CHANGE SUR LE PARC, piste par piste.
//
// Un correctif des vies sans identite n'est recevable que s'il NOMME sans rien defaire ailleurs :
// aucune vie nommee avant ne doit perdre son nom, aucun nom ne doit changer sans explication.
// La comparaison apparie les pistes d'un meme film par (slot, bornes) ; une piste dont les bornes
// ont bouge est rapprochee de celles du meme slot qui la recouvrent. Elle n'ouvre aucune base et
// ne decode rien : elle relit les fichiers `identites/<id>.tsv` des deux passes.

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

// pisteRelue : une ligne d'un fichier d'identites.
type pisteRelue struct {
	slot             uint32
	debut, fin, team int
	identite         string
}

func (p pisteRelue) cle() string { return fmt.Sprintf("%d:%d-%d", p.slot, p.debut, p.fin) }

// bilanComparaison : les comptes d'une comparaison, et les lignes a lire par film.
type bilanComparaison struct {
	films, sansNomAvant, sansNomApres            int
	nommees, perdues, changees, equipes, decoupe int
	lignes                                       []string
}

// comparerParcVies ecrit le bilan de deux passes sur `w`.
func comparerParcVies(w io.Writer, avant, apres string) error {
	ia, err := issuesDuParc(filepath.Join(avant, "films.tsv"))
	if err != nil {
		return err
	}
	ib, err := issuesDuParc(filepath.Join(apres, "films.tsv"))
	if err != nil {
		return err
	}
	var bilan bilanComparaison
	ids := make([]string, 0, len(ia))
	for id := range ia {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		if ia[id] != "ok" || ib[id] != "ok" {
			if ib[id] != ia[id] {
				bilan.lignes = append(bilan.lignes, fmt.Sprintf("%s\tissue %q -> %q", id, ia[id], ib[id]))
			}
			continue
		}
		pa, err := lirePistes(filepath.Join(avant, "identites", id+".tsv"))
		if err != nil {
			return err
		}
		pb, err := lirePistes(filepath.Join(apres, "identites", id+".tsv"))
		if err != nil {
			return err
		}
		bilan.films++
		comparerFilm(&bilan, id, pa, pb)
	}
	if _, err := fmt.Fprintf(w, "films compares %d ; sans identite %d -> %d ; nommees %d ; perdues %d ; "+
		"noms changes %d ; equipes changees %d ; pistes redecoupees %d\n", bilan.films,
		bilan.sansNomAvant, bilan.sansNomApres, bilan.nommees, bilan.perdues, bilan.changees,
		bilan.equipes, bilan.decoupe); err != nil {
		return fmt.Errorf("bilan : %w", err)
	}
	for _, l := range bilan.lignes {
		if _, err := fmt.Fprintln(w, l); err != nil {
			return fmt.Errorf("bilan : %w", err)
		}
	}
	return nil
}

// comparerFilm apparie les pistes d'un film et accumule ses changements.
func comparerFilm(bilan *bilanComparaison, id string, pa, pb []pisteRelue) {
	parCle := map[string]pisteRelue{}
	for _, p := range pa {
		parCle[p.cle()] = p
		if p.identite == "" {
			bilan.sansNomAvant++
		}
	}
	for _, q := range pb {
		if q.identite == "" {
			bilan.sansNomApres++
		}
		p, ok := parCle[q.cle()]
		if !ok {
			bilan.decoupe++
			for _, r := range recouvrantes(pa, q) {
				if r.identite != q.identite {
					bilan.lignes = append(bilan.lignes, fmt.Sprintf("%s\tredecoupee %s %q -> %s %q",
						id, r.cle(), r.identite, q.cle(), q.identite))
				}
			}
			continue
		}
		noterChangement(bilan, id, p, q)
	}
}

// noterChangement classe le changement d'une piste appariee a l'identique.
func noterChangement(bilan *bilanComparaison, id string, p, q pisteRelue) {
	switch {
	case p.identite == q.identite:
		if p.team != q.team {
			bilan.equipes++
			bilan.lignes = append(bilan.lignes, fmt.Sprintf("%s\tequipe %s %q %d -> %d", id, q.cle(),
				q.identite, p.team, q.team))
		}
		return
	case p.identite == "":
		bilan.nommees++
		bilan.lignes = append(bilan.lignes, fmt.Sprintf("%s\tnommee %s -> %q (equipe %d)", id, q.cle(),
			q.identite, q.team))
	case q.identite == "":
		bilan.perdues++
		bilan.lignes = append(bilan.lignes, fmt.Sprintf("%s\tPERDUE %s %q", id, q.cle(), p.identite))
	default:
		bilan.changees++
		bilan.lignes = append(bilan.lignes, fmt.Sprintf("%s\tCHANGEE %s %q -> %q", id, q.cle(),
			p.identite, q.identite))
	}
}

// recouvrantes rend les pistes du meme slot dont les bornes recouvrent celles de `q`.
func recouvrantes(pa []pisteRelue, q pisteRelue) []pisteRelue {
	var out []pisteRelue
	for _, p := range pa {
		if p.slot == q.slot && p.debut <= q.fin && q.debut <= p.fin {
			out = append(out, p)
		}
	}
	return out
}

// issuesDuParc relit films.tsv : film -> issue.
func issuesDuParc(tsv string) (map[string]string, error) {
	raw, err := os.ReadFile(tsv) //nolint:gosec // chemin de l'operateur
	if err != nil {
		return nil, fmt.Errorf("films.tsv : %w", err)
	}
	out := map[string]string{}
	for _, l := range strings.Split(string(raw), "\n") {
		c := strings.Split(l, "\t")
		if len(c) >= 4 {
			out[c[0]] = c[3]
		}
	}
	return out, nil
}

// lirePistes relit un fichier d'identites.
func lirePistes(chemin string) ([]pisteRelue, error) {
	f, err := os.Open(chemin) //nolint:gosec // chemin de l'operateur
	if err != nil {
		return nil, fmt.Errorf("identites : %w", err)
	}
	defer func() { _ = f.Close() }()
	var out []pisteRelue
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		c := strings.Split(sc.Text(), "\t")
		if len(c) < 5 {
			continue
		}
		slot, e1 := strconv.ParseUint(c[0], 10, 32)
		debut, e2 := strconv.Atoi(c[1])
		fin, e3 := strconv.Atoi(c[2])
		team, e4 := strconv.Atoi(c[3])
		if e1 != nil || e2 != nil || e3 != nil || e4 != nil {
			return nil, fmt.Errorf("identites %s : ligne illisible %q", chemin, sc.Text())
		}
		out = append(out, pisteRelue{slot: uint32(slot), debut: debut, fin: fin, team: team, identite: c[4]})
	}
	if err := sc.Err(); err != nil {
		return nil, fmt.Errorf("identites %s : %w", chemin, err)
	}
	return out, nil
}
