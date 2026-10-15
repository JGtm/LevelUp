//go:build research

package main

// mpp_declare.go — LE DRAPEAU `-mpp-declare`.
//
// La carte se mesure alors sous le decoupage MPP que la grammaire resout pour chaque film
// ([grammar.FilmContext.ResolutionMPP]) : celui de sa version de format, ou celui que le film
// declare par la taille d etat de creation de ses objets. C est la regle que la cuisson applique
// (`grammar.FilmContext.PoserLaCarteEtLeDecoupage`), y compris le chemin d un film qui ne declare rien, dont
// le contexte garde son decoupage. Le decoupage retenu et sa provenance s ecrivent par film dans
// `mpp_declare.tsv`.

import (
	"bufio"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// fichierMPPDeclare : le journal du drapeau, dans le repertoire du rapport.
const fichierMPPDeclare = "mpp_declare.tsv"

// journalMPP ecrit, par film, le decoupage MPP pose et sa provenance. Nil quand le drapeau n est
// pas leve : ses methodes ne font alors rien.
type journalMPP struct {
	f *os.File
	w *bufio.Writer
}

// ouvrirJournalMPP cree le journal dans `sortie` quand `actif`, et rend nil sinon.
func ouvrirJournalMPP(sortie string, actif bool) (*journalMPP, error) {
	if !actif {
		return nil, nil
	}
	f, err := os.Create(filepath.Join(sortie, fichierMPPDeclare)) //nolint:gosec // repertoire du rapport, verifie par preparerSortie
	if err != nil {
		return nil, fmt.Errorf("journal MPP : %w", err)
	}
	j := &journalMPP{f: f, w: bufio.NewWriter(f)}
	if _, err := j.w.WriteString("film\tformat\tdecoupage\tprovenance\trecords\tdiscordants\tpose\n"); err != nil {
		return nil, errors.Join(fmt.Errorf("journal MPP : %w", err), f.Close())
	}
	return j, nil
}

// poser pose sur le contexte du film `id` le decoupage que la grammaire resout, et l ecrit.
func (j *journalMPP) poser(fc *grammar.FilmContext, id string) error {
	if j == nil {
		return nil
	}
	res := fc.ResolutionMPP()
	pose := res.Decide()
	if pose {
		fc.PoserMPP(res.Widths)
	}
	fmt.Printf("%s  mpp=%s (%s)  pose=%v\n", id, res.Widths, res.Provenance, pose)
	_, err := fmt.Fprintf(j.w, "%s\t%d\t%s\t%s\t%d\t%d\t%v\n", id, res.FormatVersion, res.Widths,
		res.Provenance, res.Declaration.Records, res.Declaration.Discordants, pose)
	return err
}

// fermer vide et ferme le journal.
func (j *journalMPP) fermer() error {
	if j == nil {
		return nil
	}
	if err := j.w.Flush(); err != nil {
		return errors.Join(err, j.f.Close())
	}
	return j.f.Close()
}
