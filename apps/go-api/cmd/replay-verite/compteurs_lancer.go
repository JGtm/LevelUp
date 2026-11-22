package main

// compteurs_lancer.go — LA LECTURE DES REJEUX ET LE LANCEMENT DU MODE `-compteurs`.

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"path/filepath"
	"sort"

	"levelup/go-api/internal/replayartifact"
)

// lireRejeu lit un rejeu cuit par la porte `replayartifact`.
func lireRejeu(chemin string) (*documentCompteurs, error) {
	blob, err := replayartifact.ReadBytes(chemin)
	if err != nil {
		return nil, err
	}
	var d documentCompteurs
	if err := json.Unmarshal(blob, &d); err != nil {
		return nil, fmt.Errorf("%s : %w", chemin, err)
	}
	return &d, nil
}

// fichiersRejeux developpe les arguments (fichiers `.json.gz` ou dossiers) en une liste triee.
func fichiersRejeux(args []string) ([]string, error) {
	var out []string
	for _, a := range args {
		if replayartifact.Exists(a) {
			out = append(out, a)
			continue
		}
		pris, err := replayartifact.List(a)
		if err != nil {
			return nil, err
		}
		for _, e := range pris {
			out = append(out, filepath.Join(a, e.Name))
		}
	}
	sort.Strings(out)
	if len(out) == 0 {
		return nil, errors.New("aucun rejeu (.json.gz) a lire")
	}
	return out, nil
}

// optionsCompteurs regroupe les entrees du mode.
type optionsCompteurs struct {
	verite, series, apres string
	detail                bool
	rejeux                []string
}

// lancerCompteurs est le mode `-compteurs` : une lecture, ou une comparaison avant / apres.
func lancerCompteurs(o optionsCompteurs, out, errw io.Writer) int {
	if o.verite == "" || len(o.rejeux) == 0 {
		_, _ = fmt.Fprintln(errw, "replay-verite : -compteurs demande -verite <participants.tsv> et des rejeux (fichiers ou dossiers)")
		return codeUsage
	}
	v, err := chargerVerite(o.verite, o.series)
	if err != nil {
		_, _ = fmt.Fprintf(errw, "replay-verite : verite : %v\n", err)
		return codeUsage
	}
	fichiers, err := fichiersRejeux(o.rejeux)
	if err != nil {
		_, _ = fmt.Fprintf(errw, "replay-verite : %v\n", err)
		return codeUsage
	}
	l := &lecture{v: v, apres: o.apres, avant: nouvelleMesure(), apresMes: nouvelleMesure(),
		tr: transitions{gagnes: map[string]*[nbCompteurs]int{}, perdus: map[string]*[nbCompteurs]int{}}}
	for _, f := range fichiers {
		if err := l.lireUn(f); err != nil {
			_, _ = fmt.Fprintf(errw, "replay-verite : %v\n", err)
			return codeUsage
		}
	}
	rendreCompteurs(out, o, l)
	return codeOK
}

// lecture porte l'etat d'une lecture : la verite, les mesures AVANT et APRES, les transitions.
type lecture struct {
	v               verite
	apres           string
	avant, apresMes *mesure
	tr              transitions
}

// lireUn lit un rejeu (et son pendant « apres ») dans les mesures ; un seul document en memoire a la fois.
func (l *lecture) lireUn(f string) error {
	d, err := lireRejeu(f)
	if err != nil {
		return err
	}
	issAvant := l.avant.ajouter(d, l.v)
	if l.apres == "" {
		return nil
	}
	d2, err := lireRejeu(filepath.Join(l.apres, filepath.Base(f)))
	if err != nil {
		return err
	}
	build := buildDe(d2)
	issApres := l.apresMes.ajouter(d2, l.v)
	for xuid, a := range issAvant {
		b, ok := issApres[xuid]
		if !ok {
			continue
		}
		for c := range nbCompteurs {
			l.tr.noter(build, c, a[c], b[c])
		}
	}
	return nil
}

// noter compte un gain ou une perte d'accord large d'un joueur pour un compteur.
func (t transitions) noter(build string, c int, avant, apres issue) {
	if avant == issueSansAPI || avant.large() == apres.large() {
		return
	}
	cible := t.perdus
	if apres.large() {
		cible = t.gagnes
	}
	if cible[build] == nil {
		cible[build] = &[nbCompteurs]int{}
	}
	cible[build][c]++
}
