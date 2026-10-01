//go:build research

package main

// v2_rapport.go — LES SORTIES DE LA CARTE V2 : sept TSV ecrits film par film, et les cumuls par
// build que `v2_resume.go` ecrit a la fin. Les TSV et le resume du mode `fermeture` ne changent
// pas : la carte v2 les ecrit A L IDENTIQUE (memes colonnes, memes valeurs) et AJOUTE ceux-ci.
//
//	fermeture_sorties_vueB.tsv        (film, sortie de la vue B) sur tous les paquets qui l atteignent
//	fermeture_hors_cadre.tsv          (film, sortie, vue C, reste) des paquets « hors cadre »
//	fermeture_hors_cadre_dernier.tsv  (film, sortie, dernier composant lu) des memes paquets
//	fermeture_rejets.tsv              (film, sortie par rejet, sort du paquet, etat au bloc de
//	                                  type 1, naissance)
//	fermeture_entrees.tsv             le denominateur des entrees de controle utiles, par film
//	fermeture_chunk3.tsv              le compte declare du chunk des temps forts, par film
//	fermeture_borne.tsv               le mode borne, par film

import (
	"cmp"
	"errors"
	"fmt"
	"maps"
	"os"
	"slices"
	"strings"
)

// cumulV2 agrege les mesures v2 d un build (ou du corpus).
type cumulV2 struct {
	films                     int
	sorties                   map[string]*sortieStat
	horsCadre                 map[cleHorsCadre]*compte
	dernier                   map[cleDernier]*compte
	rejets                    map[cleRejet]*compte
	entrees                   entreesV2
	borne                     borneV2
	sansBloc, blocsIllisibles int
}

func nouveauCumulV2() *cumulV2 {
	return &cumulV2{sorties: map[string]*sortieStat{}, horsCadre: map[cleHorsCadre]*compte{},
		dernier: map[cleDernier]*compte{}, rejets: map[cleRejet]*compte{}}
}

// filmV2 est ce que le resume garde d un film (tables par film).
type filmV2 struct {
	id, build string
	m         *mesureV2
}

// rapportV2 porte les TSV de la carte v2 et ses cumuls.
type rapportV2 struct {
	sorties, horsCadre, dernier, rejets, entrees, chunk3, borne *os.File
	parBuild                                                    map[string]*cumulV2
	corpus                                                      *cumulV2
	films                                                       []filmV2
}

// ouvrirRapportV2 cree les sept TSV de la carte v2.
func ouvrirRapportV2(dir string) (*rapportV2, error) {
	r := &rapportV2{parBuild: map[string]*cumulV2{}, corpus: nouveauCumulV2()}
	tsv := []struct {
		f           **os.File
		nom, entete string
	}{
		{&r.sorties, "fermeture_sorties_vueB.tsv", "film\tbuild\tsortie_vueB\tpaquets\tfermes\thors_cadre\tautres_causes\tutiles_en_jeu_hors_cadre"},
		{&r.horsCadre, "fermeture_hors_cadre.tsv", "film\tbuild\tsortie_vueB\tvue_c\treste\tpaquets\tutiles_en_jeu"},
		{&r.dernier, "fermeture_hors_cadre_dernier.tsv", "film\tbuild\tsortie_vueB\tdernier_lu\tpaquets\tutiles_en_jeu"},
		{&r.rejets, "fermeture_rejets.tsv", "film\tbuild\tsortie_vueB\tpaquet\tetat_bloc_type1\tnaissance\tpaquets\tutiles_en_jeu"},
		{&r.entrees, "fermeture_entrees.tsv", "film\tbuild\tpaquets\tpaquets_non_fermes\tvues_c_fermees\t" +
			"entrees_fermees\tentrees_utiles_fermees\tmoyenne_utiles_par_vue_c_fermee\tdenominateur_estime\t" +
			"part_estimee\tentrees_utiles_lues_hors_fermeture"},
		{&r.chunk3, "fermeture_chunk3.tsv", "film\tbuild\tchunk\tmesure\tpaquets_type9\tdeclares\ttrouves\t" +
			"ecart\tkills\tdeaths\tmedailles\tmodes\tautres\tfil_des_morts\trefus"},
		{&r.borne, "fermeture_borne.tsv", "film\tbuild\trecords_lus\trecords_debordants\tcomposants_debordants\t" +
			"neufs_propres_debordants\tpaquets_avec_debordement\tterminateurs_au_dela\tchunks_consultes_sans_bloc_type1\tblocs_type1_illisibles"},
	}
	for _, t := range tsv {
		f, err := creerTSV(dir, t.nom, t.entete)
		if err != nil {
			return nil, errors.Join(err, r.fermer())
		}
		*t.f = f
	}
	return r, nil
}

// fermer ferme les TSV ouverts.
func (r *rapportV2) fermer() error {
	return fermerTSV(r.sorties, r.horsCadre, r.dernier, r.rejets, r.entrees, r.chunk3, r.borne)
}

// ajouter ecrit les lignes v2 d un film et le cumule.
func (r *rapportV2) ajouter(id, build string, m *mesureV2) error {
	ecrire := []func(id, build string, m *mesureV2) error{r.ecrireSorties, r.ecrireHorsCadre,
		r.ecrireDernier, r.ecrireRejets, r.ecrireEntrees, r.ecrireChunk3, r.ecrireBorne}
	for _, e := range ecrire {
		if err := e(id, build, m); err != nil {
			return err
		}
	}
	cb := r.parBuild[build]
	if cb == nil {
		cb = nouveauCumulV2()
		r.parBuild[build] = cb
	}
	cb.cumuler(m)
	r.corpus.cumuler(m)
	r.films = append(r.films, filmV2{id: id, build: build, m: m})
	return nil
}

func (r *rapportV2) ecrireSorties(id, build string, m *mesureV2) error {
	for _, s := range slices.Sorted(maps.Keys(m.sorties)) {
		v := m.sorties[s]
		if _, err := fmt.Fprintf(r.sorties, "%s\t%s\t%s\t%d\t%d\t%d\t%d\t%d\n", id, build, s, v.paquets,
			v.fermes, v.horsCadre, v.autres, v.enJeuHorsCadre); err != nil {
			return err
		}
	}
	return nil
}

func (r *rapportV2) ecrireHorsCadre(id, build string, m *mesureV2) error {
	for _, k := range clesHorsCadre(m.horsCadre) {
		v := m.horsCadre[k]
		if _, err := fmt.Fprintf(r.horsCadre, "%s\t%s\t%s\t%s\t%s\t%d\t%d\n", id, build, k.sortie, k.vueC,
			k.reste, v.paquets, v.enJeu); err != nil {
			return err
		}
	}
	return nil
}

func (r *rapportV2) ecrireDernier(id, build string, m *mesureV2) error {
	for _, k := range clesDernier(m.dernier) {
		v := m.dernier[k]
		if _, err := fmt.Fprintf(r.dernier, "%s\t%s\t%s\t%s\t%d\t%d\n", id, build, k.sortie, k.dernier,
			v.paquets, v.enJeu); err != nil {
			return err
		}
	}
	return nil
}

func (r *rapportV2) ecrireRejets(id, build string, m *mesureV2) error {
	for _, k := range clesRejets(m.rejets) {
		if _, err := fmt.Fprintf(r.rejets, "%s\t%s\t%s\t%s\t%s\t%s\t%d\t%d\n", id, build, k.sortie, k.paquet,
			k.etat, k.naissance, m.rejets[k].paquets, m.rejets[k].enJeu); err != nil {
			return err
		}
	}
	return nil
}

func (r *rapportV2) ecrireEntrees(id, build string, m *mesureV2) error {
	e := m.entrees
	_, err := fmt.Fprintf(r.entrees, "%s\t%s\t%d\t%d\t%d\t%d\t%d\t%.3f\t%.0f\t%s\t%d\n", id, build,
		e.paquets, e.nonFermes, e.vuesCFermees, e.entreesFermees, e.utilesFermees, e.moyenne(),
		e.denominateurEstime(), partEstimee(e), e.utilesLuesHorsFermeture)
	return err
}

func (r *rapportV2) ecrireChunk3(id, build string, m *mesureV2) error {
	c := m.chunk3
	_, err := fmt.Fprintf(r.chunk3, "%s\t%s\t%d\t%t\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%s\n", id, build,
		c.chunk, c.mesure, c.paquets9, c.declares, c.trouves, c.trouves-c.declares, c.kills, c.deaths,
		c.medailles, c.modes, c.autres, c.filDesMorts, cellule(c.refus))
	return err
}

func (r *rapportV2) ecrireBorne(id, build string, m *mesureV2) error {
	b := m.borne
	_, err := fmt.Fprintf(r.borne, "%s\t%s\t%d\t%d\t%d\t%d\t%d\t%d\t%d\t%d\n", id, build, b.records,
		b.debordants, b.composants, b.neufs, b.paquets, b.terminateursAuDela, m.sansBloc, m.blocsIllisibles)
	return err
}

// moyenne rend la moyenne d entrees utiles par vue C fermee.
func (e entreesV2) moyenne() float64 {
	if e.vuesCFermees == 0 {
		return 0
	}
	return float64(e.utilesFermees) / float64(e.vuesCFermees)
}

// partEstimee rend `utiles fermees / denominateur estime`, en pourcentage.
func partEstimee(e entreesV2) string {
	d := e.denominateurEstime()
	if d == 0 {
		return "-"
	}
	return strings.Replace(fmt.Sprintf("%.1f %%", 100*float64(e.utilesFermees)/d), ".", ",", 1)
}

// cumuler ajoute les mesures d un film.
func (c *cumulV2) cumuler(m *mesureV2) {
	c.films++
	for k, v := range m.sorties {
		s := c.sorties[k]
		if s == nil {
			s = &sortieStat{}
			c.sorties[k] = s
		}
		s.paquets += v.paquets
		s.fermes += v.fermes
		s.horsCadre += v.horsCadre
		s.autres += v.autres
		s.enJeuHorsCadre += v.enJeuHorsCadre
	}
	sommer(c.horsCadre, m.horsCadre)
	sommer(c.dernier, m.dernier)
	sommer(c.rejets, m.rejets)
	e := &c.entrees
	e.paquets += m.entrees.paquets
	e.nonFermes += m.entrees.nonFermes
	e.vuesCFermees += m.entrees.vuesCFermees
	e.entreesFermees += m.entrees.entreesFermees
	e.utilesFermees += m.entrees.utilesFermees
	e.utilesLuesHorsFermeture += m.entrees.utilesLuesHorsFermeture
	b := &c.borne
	b.records += m.borne.records
	b.debordants += m.borne.debordants
	b.composants += m.borne.composants
	b.neufs += m.borne.neufs
	b.paquets += m.borne.paquets
	b.terminateursAuDela += m.borne.terminateursAuDela
	c.sansBloc += m.sansBloc
	c.blocsIllisibles += m.blocsIllisibles
}

// sommer ajoute une table de comptes a une autre.
func sommer[K comparable](dst, src map[K]*compte) {
	for k, v := range src {
		d := dst[k]
		if d == nil {
			d = &compte{}
			dst[k] = d
		}
		d.paquets += v.paquets
		d.enJeu += v.enJeu
	}
}

// clesHorsCadre rend les cles triees : paquets decroissants, puis l ordre des champs.
func clesHorsCadre(t map[cleHorsCadre]*compte) []cleHorsCadre {
	out := slices.Collect(maps.Keys(t))
	slices.SortFunc(out, func(a, b cleHorsCadre) int {
		return cmp.Or(cmp.Compare(t[b].paquets, t[a].paquets), strings.Compare(a.sortie, b.sortie),
			strings.Compare(a.vueC, b.vueC), strings.Compare(a.reste, b.reste))
	})
	return out
}

// clesDernier rend les cles triees : paquets decroissants, puis l ordre des champs.
func clesDernier(t map[cleDernier]*compte) []cleDernier {
	out := slices.Collect(maps.Keys(t))
	slices.SortFunc(out, func(a, b cleDernier) int {
		return cmp.Or(cmp.Compare(t[b].paquets, t[a].paquets), strings.Compare(a.sortie, b.sortie),
			strings.Compare(a.dernier, b.dernier))
	})
	return out
}

// clesRejets rend les cles triees : paquets decroissants, puis l ordre des champs.
func clesRejets(t map[cleRejet]*compte) []cleRejet {
	out := slices.Collect(maps.Keys(t))
	slices.SortFunc(out, func(a, b cleRejet) int {
		return cmp.Or(cmp.Compare(t[b].paquets, t[a].paquets), strings.Compare(a.sortie, b.sortie),
			strings.Compare(a.paquet, b.paquet), strings.Compare(a.etat, b.etat),
			strings.Compare(a.naissance, b.naissance))
	})
	return out
}
