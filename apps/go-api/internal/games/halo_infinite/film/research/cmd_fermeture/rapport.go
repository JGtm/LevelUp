//go:build research

package main

// rapport.go — LES SORTIES : trois TSV bruts ecrits film par film (un arret en cours de corpus
// garde ce qui est mesure), et un resume Markdown ecrit a la fin, par build.
//
//	fermeture_films.tsv       une ligne par film
//	fermeture_archetypes.tsv  une ligne par (film, archetype)
//	fermeture_bloquants.tsv   une ligne par (film, cause d arret)
//	fermeture_resume.md       par build : paquets fermes par vue, records utiles fermes ; puis le
//	                          classement des bloquants du corpus
//
// Les chiffres sont colles, sans interpretation : les verdicts appartiennent a la note de
// mesure (J4.0.5), pas a l instrument.

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
)

// mesureFilm est ce qu UN film a rendu.
type mesureFilm struct {
	id, build string
	carte     grammar.FrameClosureReport
	// pic : l empreinte memoire maximale observee par la sentinelle pendant ce film.
	pic   uint64
	duree time.Duration
	// largeursLues : les largeurs d axe ont ete detectees dans le film (sinon : profil par defaut).
	largeursLues bool
}

// cumulBuild agrege les films d un build.
type cumulBuild struct {
	films, paquets, fermes, listes int
	atteints, vuesFermees          [grammar.NombreDeVues]int
	utiles, utilesFermes, entrees  int
	picMax                         uint64
}

// cumulBloquant agrege une cause d arret sur le corpus.
type cumulBloquant struct {
	ti             int
	composant      string
	paquets, enJeu int
	index, builds  map[string]bool
	statut, usage  string
	exemple        string
}

// rapport porte les fichiers ouverts et les cumuls.
type rapport struct {
	dir                          string
	tab                          tableECS
	films, archetypes, bloquants *os.File
	parBuild                     map[string]*cumulBuild
	parBloquant                  map[string]*cumulBloquant
	mesures, echecs              int
}

// ouvrirRapport cree les trois TSV et ecrit leurs en-tetes.
func ouvrirRapport(dir string, tab tableECS) (*rapport, error) {
	r := &rapport{dir: dir, tab: tab, parBuild: map[string]*cumulBuild{},
		parBloquant: map[string]*cumulBloquant{}}
	var err error
	if r.films, err = creerTSV(dir, "fermeture_films.tsv", "film\tbuild\tlargeurs_lues\tpaquets\t"+
		"paquets_fermes\tlistes_non_localisees\tvueA_atteints\tvueA_fermes\tvueB_atteints\t"+
		"vueB_fermes\tvueC_atteints\tvueC_fermes\tutiles\tutiles_fermes\tentrees_controle_fermees\t"+
		"premier_bloquant\tpic_octets\tduree_ms"); err != nil {
		return nil, err
	}
	if r.archetypes, err = creerTSV(dir, "fermeture_archetypes.tsv", "film\tbuild\tti\tneufs\t"+
		"neufs_fermes\tdeltas\tdeltas_fermes\tutiles\tutiles_fermes\tbloquant"); err != nil {
		return nil, errors.Join(err, r.fermer())
	}
	if r.bloquants, err = creerTSV(dir, "fermeture_bloquants.tsv", "film\tbuild\tcause\tti\tindex\t"+
		"composant\tstatut\tusage_produit\tpaquets\tutiles_en_jeu"); err != nil {
		return nil, errors.Join(err, r.fermer())
	}
	return r, nil
}

// creerTSV cree un fichier du rapport et y ecrit l en-tete.
func creerTSV(dir, nom, entete string) (*os.File, error) {
	f, err := os.Create(filepath.Join(dir, nom)) //nolint:gosec // repertoire verifie hors de data/
	if err != nil {
		return nil, fmt.Errorf("creation de %s : %w", nom, err)
	}
	if _, err := fmt.Fprintln(f, entete); err != nil {
		return nil, errors.Join(fmt.Errorf("ecriture de %s : %w", nom, err), f.Close())
	}
	return f, nil
}

// ajouter ecrit les lignes d un film et l ajoute aux cumuls.
func (r *rapport) ajouter(m mesureFilm) error {
	c := m.carte
	v := c.Vues
	if _, err := fmt.Fprintf(r.films, "%s\t%s\t%t\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%s\t%d\t%d\n",
		m.id, m.build, m.largeursLues, c.Paquets, c.PaquetsFermes, c.ListesNonLocalisees,
		v[grammar.VueMessages].Atteints, v[grammar.VueMessages].Fermes,
		v[grammar.VueEntites].Atteints, v[grammar.VueEntites].Fermes,
		v[grammar.VueControle].Atteints, v[grammar.VueControle].Fermes,
		c.Utiles.Records, c.Utiles.RecordsFermes, c.Utiles.EntreesDeControleFermees,
		c.BloquantPrincipal(), m.pic, m.duree.Milliseconds()); err != nil {
		return err
	}
	if err := r.ecrireArchetypes(m); err != nil {
		return err
	}
	if err := r.ecrireBloquants(m); err != nil {
		return err
	}
	r.cumulerBuild(m)
	r.mesures++
	return nil
}

// ecrireArchetypes ecrit une ligne par archetype du film, dans l ordre des index.
func (r *rapport) ecrireArchetypes(m mesureFilm) error {
	for _, ti := range clesTriees(m.carte.Archetypes) {
		a := m.carte.Archetypes[ti]
		if _, err := fmt.Fprintf(r.archetypes, "%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%s\n", m.id, m.build,
			ti, a.Neufs, a.NeufsFermes, a.Deltas, a.DeltasFermes, a.Utiles, a.UtilesFermes,
			a.Blocking); err != nil {
			return err
		}
	}
	return nil
}

// ecrireBloquants ecrit une ligne par cause d arret du film et la cumule.
func (r *rapport) ecrireBloquants(m mesureFilm) error {
	for _, cause := range nomsTries(m.carte.Bloquants) {
		b := m.carte.Bloquants[cause]
		statut, usage := "", ""
		if b.Composant != "" {
			statut, usage = r.tab.decrire(b.TI, b.Composant)
		}
		if _, err := fmt.Fprintf(r.bloquants, "%s\t%s\t%s\t%d\t%d\t%s\t%s\t%s\t%d\t%d\n", m.id, m.build,
			cause, b.TI, b.Index, b.Composant, statut, usage, b.Paquets, b.UtilesEnJeu); err != nil {
			return err
		}
		r.cumulerBloquant(m.build, cause, b, statut, usage)
	}
	return nil
}

// cumulerBuild ajoute un film aux cumuls de son build.
func (r *rapport) cumulerBuild(m mesureFilm) {
	cb := r.parBuild[m.build]
	if cb == nil {
		cb = &cumulBuild{}
		r.parBuild[m.build] = cb
	}
	c := m.carte
	cb.films++
	cb.paquets += c.Paquets
	cb.fermes += c.PaquetsFermes
	cb.listes += c.ListesNonLocalisees
	for v := range cb.atteints {
		cb.atteints[v] += c.Vues[v].Atteints
		cb.vuesFermees[v] += c.Vues[v].Fermes
	}
	cb.utiles += c.Utiles.Records
	cb.utilesFermes += c.Utiles.RecordsFermes
	cb.entrees += c.Utiles.EntreesDeControleFermees
	cb.picMax = max(cb.picMax, m.pic)
}

// cumulerBloquant agrege une cause sur le corpus. Un COMPOSANT se cumule par (archetype, nom) et
// non par index : l index d un composant change d un build a l autre, son nom non.
func (r *rapport) cumulerBloquant(build, cause string, b grammar.FrameBlockerStat, statut, usage string) {
	cle := cause
	if b.Composant != "" {
		cle = fmt.Sprintf("ti=%d %s", b.TI, grammar.CleComposant(b.TI, b.Composant).Nom)
	}
	cb := r.parBloquant[cle]
	if cb == nil {
		cb = &cumulBloquant{ti: b.TI, composant: b.Composant, index: map[string]bool{},
			builds: map[string]bool{}, statut: statut, usage: usage, exemple: cause}
		r.parBloquant[cle] = cb
	}
	cb.paquets += b.Paquets
	cb.enJeu += b.UtilesEnJeu
	cb.builds[build] = true
	if b.Index >= 0 {
		cb.index[fmt.Sprintf("i%d", b.Index)] = true
	}
}

// fermer ferme les trois TSV.
func (r *rapport) fermer() error {
	var errs []error
	for _, f := range []*os.File{r.films, r.archetypes, r.bloquants} {
		if f != nil {
			errs = append(errs, f.Close())
		}
	}
	return errors.Join(errs...)
}
