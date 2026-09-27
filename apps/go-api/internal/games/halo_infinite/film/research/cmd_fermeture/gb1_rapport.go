//go:build research

package main

// gb1_rapport.go — LES SORTIES DU MODE `gb1` : trois TSV bruts ecrits film par film, et la
// section GB-1 du resume Markdown.
//
//	gb1_films.tsv      une ligne par film : creations par generation, vies, vies sans position
//	                   (filtre prod / generation vivante), en-tetes bruts et orphelins par tag,
//	                   durees (film, `durationMs` publie reconstitue, vivante)
//	gb1_vies.tsv       une ligne par (film, slot, generation) connue
//	gb1_orphelins.tsv  une ligne par (film, slot, tag) d en-tete brut qui n est aucune vie connue
//
// Les definitions sont celles de gb1.go. Les chiffres sont colles, sans interpretation : le
// verdict (critere de poursuite du J5.0) appartient au rapport de mesure du superviseur.

import (
	"cmp"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"
)

// syntheseGB1 : les comptes derives d une mesure, communs au TSV et au resume.
type syntheseGB1 struct {
	vies, viesGen2, slotsMultiVies            int
	creationSeule, imageCleSeule, deuxSources int
	horsBande                                 int
	sansProd, gen2SansProd                    int
	sansVivante, gen2AvecVivante              int
	entetes, orphelins, orphelinsGardes       int
	dureeFilm, dureeDelta                     int64
	dureePubliee, dureeVivante                int64
	finTronquee, finTronqueeVivante           int64
}

// synthetiser calcule les comptes derives d une mesure.
func synthetiser(m mesureGB1) syntheseGB1 {
	s := syntheseGB1{vies: len(m.vies)}
	parSlot := map[uint32]int{}
	for k, v := range m.vies {
		parSlot[k.slot]++
		s.compterVie(k, v)
	}
	for _, n := range parSlot {
		if n > 1 {
			s.slotsMultiVies++
		}
	}
	for t := range nombreDeTags {
		s.entetes += m.entetesParTag[t]
		s.orphelins += m.orphelinsParTag[t]
		s.orphelinsGardes += m.gardesParTag[t]
	}
	s.dureeFilm, s.dureeDelta = ecartMS(m.film.fin, m.film.debut), ecartMS(m.delta.fin, m.delta.debut)
	s.dureePubliee, s.dureeVivante = m.prod.dureeMS(), m.vivant.dureeMS()
	if m.prod.fin > 0 {
		s.finTronquee = ecartMS(m.delta.fin, m.prod.fin)
	}
	if m.vivant.fin > 0 {
		s.finTronqueeVivante = ecartMS(m.delta.fin, m.vivant.fin)
	}
	return s
}

// compterVie ajoute UNE vie aux comptes.
func (s *syntheseGB1) compterVie(k cleDeVie, v *vieGB1) {
	gen2 := k.gen >= 2
	if gen2 {
		s.viesGen2++
	}
	switch {
	case v.creations > 0 && v.imagesCles > 0:
		s.deuxSources++
	case v.creations > 0:
		s.creationSeule++
	default:
		s.imageCleSeule++
	}
	if v.horsBande {
		s.horsBande++
	}
	if v.prod == 0 {
		s.sansProd++
		if gen2 {
			s.gen2SansProd++
		}
	}
	if v.vives == 0 {
		s.sansVivante++
	} else if gen2 {
		s.gen2AvecVivante++
	}
}

// rapportGB1 porte les fichiers du mode et les mesures gardees pour le resume.
type rapportGB1 struct {
	films, vies, orphelins *os.File
	lignes                 []ligneResumeGB1
}

// ligneResumeGB1 : ce que le resume garde d un film (les vies et orphelins sont liberes).
type ligneResumeGB1 struct {
	id, build, statut string
	s                 syntheseGB1
	creations         [nombreDeTags]int
	slotsGen2         int
	orphProd          int
}

// ouvrirRapportGB1 cree les trois TSV du mode et ecrit leurs en-tetes.
func ouvrirRapportGB1(dir string) (*rapportGB1, error) {
	r := &rapportGB1{}
	var err error
	if r.films, err = creerTSV(dir, "gb1_films.tsv", enteteFilmsGB1()); err != nil {
		return nil, err
	}
	if r.vies, err = creerTSV(dir, "gb1_vies.tsv", "film\tslot\tgen\tcreations\tpremiere_creation_us\t"+
		"images_cles\thors_bande\tpositions_prod\tentetes_bruts\tpositions_vivantes\tpremier_us\tdernier_us"); err != nil {
		return nil, errors.Join(err, r.fermer())
	}
	if r.orphelins, err = creerTSV(dir, "gb1_orphelins.tsv",
		"film\tslot\ttag\tentetes\tgardes\tpremier_us\tdernier_us"); err != nil {
		return nil, errors.Join(err, r.fermer())
	}
	return r, nil
}

// enteteFilmsGB1 rend l en-tete de `gb1_films.tsv`.
func enteteFilmsGB1() string {
	cols := []string{"film", "build", "statut", "creations"}
	cols = append(cols, parTag("creations_gen")...)
	cols = append(cols, "creations_repetees", "slots_crees", "slots_crees_gen2plus",
		"vies", "vies_gen2plus", "slots_a_plusieurs_vies", "vies_creation_seule",
		"vies_image_cle_seule", "vies_deux_sources", "vies_hors_bande",
		"positions_prod", "orphelins_prod", "vies_sans_position_prod", "vies_gen2plus_sans_position_prod",
		"entetes_bruts")
	cols = append(cols, parTag("entetes_tag")...)
	cols = append(cols, parTag("vivantes_tag")...)
	cols = append(cols, parTag("orphelins_tag")...)
	cols = append(cols, parTag("orphelins_gardes_tag")...)
	cols = append(cols, "vies_sans_position_vivante", "vies_gen2plus_avec_position_vivante",
		"duree_film_ms", "duree_delta_ms", "duree_publiee_ms", "duree_vivante_ms", "fin_tronquee_ms",
		"fin_tronquee_vivante_ms", "pic_octets", "duree_mesure_ms")
	return strings.Join(cols, "\t")
}

// parTag rend les quatre noms de colonne d une grandeur par tag.
func parTag(prefixe string) []string {
	out := make([]string, nombreDeTags)
	for t := range out {
		out[t] = fmt.Sprintf("%s%d", prefixe, t)
	}
	return out
}

// ajouter ecrit les lignes d un film et le garde pour le resume.
func (r *rapportGB1) ajouter(m mesureGB1) error {
	s := synthetiser(m)
	vals := []any{m.id, m.build, m.statut, m.creations}
	vals = append(vals, entiers(m.creationsParGen)...)
	vals = append(vals, m.creationsRepetees, m.slotsCrees, m.slotsCreesGen2,
		s.vies, s.viesGen2, s.slotsMultiVies, s.creationSeule, s.imageCleSeule, s.deuxSources, s.horsBande,
		m.positionsProd, m.orphelinsProd, s.sansProd, s.gen2SansProd, s.entetes)
	vals = append(vals, entiers(m.entetesParTag)...)
	vals = append(vals, entiers(m.vivesParTag)...)
	vals = append(vals, entiers(m.orphelinsParTag)...)
	vals = append(vals, entiers(m.gardesParTag)...)
	vals = append(vals, s.sansVivante, s.gen2AvecVivante, s.dureeFilm, s.dureeDelta, s.dureePubliee, s.dureeVivante,
		s.finTronquee, s.finTronqueeVivante, m.pic, m.duree.Milliseconds())
	if _, err := fmt.Fprintln(r.films, joindre(vals)); err != nil {
		return err
	}
	if err := r.ecrireVies(m); err != nil {
		return err
	}
	if err := r.ecrireOrphelins(m); err != nil {
		return err
	}
	r.lignes = append(r.lignes, ligneResumeGB1{id: m.id, build: m.build, statut: m.statut, s: s,
		creations: m.creationsParGen, slotsGen2: m.slotsCreesGen2, orphProd: m.orphelinsProd})
	return nil
}

// ecrireVies ecrit une ligne par vie connue, dans l ordre (slot, generation).
func (r *rapportGB1) ecrireVies(m mesureGB1) error {
	for _, k := range clesDeVieTriees(m.vies) {
		v := m.vies[k]
		if _, err := fmt.Fprintf(r.vies, "%s\t%d\t%d\t%d\t%d\t%d\t%t\t%d\t%d\t%d\t%d\t%d\n", m.id, k.slot, k.gen,
			v.creations, v.premiereCreationUS, v.imagesCles, v.horsBande, v.prod, v.brut, v.vives,
			v.premierUS, v.dernierUS); err != nil {
			return err
		}
	}
	return nil
}

// ecrireOrphelins ecrit une ligne par (slot, tag) orphelin, dans l ordre (slot, tag).
func (r *rapportGB1) ecrireOrphelins(m mesureGB1) error {
	for _, k := range clesDeVieTriees(m.orphelins) {
		o := m.orphelins[k]
		if _, err := fmt.Fprintf(r.orphelins, "%s\t%d\t%d\t%d\t%d\t%d\t%d\n", m.id, k.slot, k.gen,
			o.entetes, o.gardes, o.premierUS, o.dernierUS); err != nil {
			return err
		}
	}
	return nil
}

// fermer ferme les trois TSV du mode.
func (r *rapportGB1) fermer() error {
	err := fermerTSV(r.films, r.vies, r.orphelins)
	r.films, r.vies, r.orphelins = nil, nil, nil
	return err
}

// ecrireSection ecrit la section GB-1 du resume.
func (r *rapportGB1) ecrireSection(w io.Writer) {
	fmt.Fprintln(w, "## GB-1 — vies du bipede par generation du handle")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "Vie = (slot, generation) connue par un record de creation ou d image-cle. "+
		"Filtre prod = `DefaultScanFilmOptions` (generation 1 avant le lot J5.2, generations vivantes du film depuis), "+
		"en quanta (sans filtre de vitesse), chaque position rattachee a sa vie (slot, tag). "+
		"Generation vivante = en-tetes du meme marcheur filtre leve dont (slot, tag) est une vie "+
		"connue, isolement 15 s par (slot, tag). Orphelin = en-tete brut dont (slot, tag) n est "+
		"aucune vie connue (faux positif potentiel). Durees en ms ; `durationMs` publie reconstitue "+
		"par la formule de `replay` sur les positions prod.")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Film | Build | Statut | Vies | dont gen >= 2 | Slots a plusieurs vies | Creations gen 0/1/2/3 "+
		"| Slots crees gen >= 2 | Sans position (prod) | dont gen >= 2 | Sans position (gen. vivante) "+
		"| Gen >= 2 positionnees (gen. vivante) | Orphelins bruts (gardes) | Orphelins prod "+
		"| durationMs publie | Duree des paquets delta (tous paquets) | Duree vivante |")
	fmt.Fprintln(w, "|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|")
	for _, l := range r.lignes {
		s := l.s
		fmt.Fprintf(w, "| %s | %s | %s | %d | %d | %d | %d/%d/%d/%d | %d | %d | %d | %d | %d | %d (%d) | %d | %d | %d (%d) | %d |\n",
			l.id, l.build, l.statut, s.vies, s.viesGen2, s.slotsMultiVies, l.creations[0], l.creations[1],
			l.creations[2], l.creations[3], l.slotsGen2, s.sansProd, s.gen2SansProd, s.sansVivante,
			s.gen2AvecVivante, s.orphelins, s.orphelinsGardes, l.orphProd, s.dureePubliee, s.dureeDelta, s.dureeFilm,
			s.dureeVivante)
	}
	fmt.Fprintln(w)
}

// clesDeVieTriees rend les cles d une table, dans l ordre (slot, generation).
func clesDeVieTriees[V any](m map[cleDeVie]V) []cleDeVie {
	out := make([]cleDeVie, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	slices.SortFunc(out, func(a, b cleDeVie) int { return cmp.Or(cmp.Compare(a.slot, b.slot), cmp.Compare(a.gen, b.gen)) })
	return out
}

// entiers rend un tableau par tag sous forme de valeurs a joindre.
func entiers(a [nombreDeTags]int) []any {
	out := make([]any, len(a))
	for i, v := range a {
		out[i] = v
	}
	return out
}

// joindre rend les valeurs separees par des tabulations.
func joindre(vals []any) string {
	parts := make([]string, len(vals))
	for i, v := range vals {
		parts[i] = fmt.Sprint(v)
	}
	return strings.Join(parts, "\t")
}
