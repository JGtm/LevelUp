// Package duckdb — temoin_bascule_concordance_test.go : la moitie « source de degat » du
// temoin de bascule (A0.2) et les deux mesures de concordance graphe / kill feed
// (A0.4 et A0.5, decision D13 du plan).
//
// # Ce que ce fichier ajoute au temoin
//
//  1. la ventilation par classe telle que la NOUVELLE chaine la produira : classes
//     melee/grenade servies par les compteurs API (D4), tout le reste lu dans la source
//     de degat, residu = total API moins la somme ;
//  2. A0.4 — les tags reellement observes en base qui obtiennent une IMAGE sans obtenir
//     de CLE (ou l'inverse). Le rejeu 2D sait alors nommer un kill que le graphe classerait
//     « Non attribue » : c'est la divergence que D13 exige de mesurer ;
//  3. A0.5 — pour les tags qui obtiennent une cle, l'ecart entre le nom affiche par le
//     kill feed (`damagetag.Lookup(...).Name`) et le libelle du registre
//     (`metadata.weapons.name`).
//
// Aucune de ces mesures ne CORRIGE quoi que ce soit : donner une cle de registre aux
// vehicules et aux tourelles est explicitement hors perimetre (D13).
package duckdb

import (
	"context"
	"sort"
	"testing"

	"levelup/go-api/internal/domain"
	"levelup/go-api/internal/games/halo_infinite"
	"levelup/go-api/internal/games/halo_infinite/film/damagetag"
	"levelup/go-api/internal/games/halo_infinite/film/killicon"
)

// temoinTag : un tag de source de degat observe en base, et ce que les deux tables en font.
type temoinTag struct {
	tag    uint32
	morts  int
	image  string // sprite du kill feed ; "" = aucune image
	cle    string // weapon_key du registre ; "" = aucune cle
	classe damagetag.Class
	nom    string // nom affiche par le kill feed
}

// tagsObserves lit les tags de source reellement presents sur le lot, avec leur volume de
// morts, et les confronte aux deux tables embarquees.
func tagsObserves(t *testing.T, b *temoinBase, s temoinScope) []temoinTag {
	t.Helper()
	args := make([]any, 0, len(s.matchIDs)+len(s.xuids))
	for _, id := range s.matchIDs {
		args = append(args, id)
	}
	for _, x := range s.xuids {
		args = append(args, x)
	}
	rows, err := b.pdb.Shared.Query(context.Background(), `
SELECT k.source_tag, COUNT(*)::INTEGER AS morts
FROM match_kill_events_latest k
WHERE k.match_id IN (`+Placeholders(len(s.matchIDs))+`)
  AND k.source_tag IS NOT NULL
  AND k.feed_killer_xuid IS NOT NULL
  AND k.feed_killer_xuid IN (`+Placeholders(len(s.xuids))+`)
GROUP BY k.source_tag`, args...)
	if err != nil {
		t.Fatalf("tags observes : %v", err)
	}
	defer rows.Close()
	var out []temoinTag
	reg := halo_infinite.NewKillSourceRegistry()
	for rows.Next() {
		var tag uint32
		var morts int
		if err := rows.Scan(&tag, &morts); err != nil {
			t.Fatalf("scan tag : %v", err)
		}
		e := temoinTag{tag: tag, morts: morts}
		if ic, ok := killicon.Lookup(tag); ok {
			e.image = ic.Sprite
		}
		if k, ok := reg.KillSourceRegistryKey(tag); ok {
			e.cle = k
		}
		if l, ok := damagetag.Lookup(tag); ok {
			e.classe, e.nom = l.Class, l.Name
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iteration des tags : %v", err)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].morts > out[j].morts })
	return out
}

// registreParCle lit `weapons` pour un lot de cles : classe et NOM du registre. Sans le
// filtre anti-double-comptage de resolveOffArsenalKeys — c'est precisement le filtre que
// la fusion des chemins (D11) fait disparaitre.
func registreParCle(t *testing.T, b *temoinBase, cles []string) map[string][2]string {
	t.Helper()
	out := map[string][2]string{}
	if len(cles) == 0 {
		return out
	}
	args := make([]any, 0, len(cles)+1)
	args = append(args, temoinSlug)
	for _, k := range cles {
		args = append(args, k)
	}
	rows, err := b.pdb.Metadata.Query(context.Background(),
		"SELECT weapon_key, COALESCE(class,''), COALESCE(name,'') FROM weapons"+
			" WHERE title_slug = ? AND weapon_key IN ("+Placeholders(len(cles))+")", args...)
	if err != nil {
		t.Fatalf("registre par cle : %v", err)
	}
	defer rows.Close()
	for rows.Next() {
		var k, c, n string
		if err := rows.Scan(&k, &c, &n); err != nil {
			t.Fatalf("scan registre : %v", err)
		}
		out[k] = [2]string{c, n}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iteration registre : %v", err)
	}
	return out
}

// ventilationNouvelleChaine construit la ventilation par classe que servira la nouvelle
// chaine : compteurs API pour melee et grenade (D4), source de degat pour tout le reste,
// residu = total API moins la somme.
func ventilationNouvelleChaine(t *testing.T, b *temoinBase, s temoinScope,
	counts domain.FragKillTypeCounts,
) domain.FragDistribution {
	t.Helper()
	tags := tagsObserves(t, b, s)
	cles := make([]string, 0, len(tags))
	vu := map[string]bool{}
	for _, e := range tags {
		if e.cle != "" && !vu[e.cle] {
			vu[e.cle] = true
			cles = append(cles, e.cle)
		}
	}
	reg := registreParCle(t, b, cles)

	parClasse := map[string]int{}
	for _, e := range tags {
		if e.cle == "" {
			continue
		}
		meta, ok := reg[e.cle]
		if !ok || meta[0] == "" {
			continue // cle hors registre : reste dans « Non attribue » (D7)
		}
		if meta[0] == domain.FragClassMelee || meta[0] == domain.FragClassGrenade {
			continue // servi par les compteurs API (D4) — jamais deux fois
		}
		parClasse[meta[0]] += e.morts
	}
	if m := counts.Melee + counts.Assassination; m > 0 {
		parClasse[domain.FragClassMelee] = m
	}
	if counts.Grenade > 0 {
		parClasse[domain.FragClassGrenade] = counts.Grenade
	}
	somme := 0
	for _, v := range parClasse {
		somme += v
	}
	if residu := counts.Total - somme; residu > 0 {
		parClasse[domain.FragClassUnattributed] = residu
	}
	classes := make([]domain.FragClassEntry, 0, len(parClasse))
	for c, k := range parClasse {
		classes = append(classes, domain.FragClassEntry{Class: c, Kills: k})
	}
	sort.Slice(classes, func(i, j int) bool { return classes[i].Class < classes[j].Class })
	return domain.FragDistribution{TotalKills: counts.Total, Classes: classes}
}

// ecrireConcordance publie A0.4 (image sans cle, cle sans image) et A0.5 (ecart de nom),
// plus le volume des classes melee/grenade vues par la source de degat — le nombre que la
// decision D4 ecarte, et qu'il faut donc connaitre.
func ecrireConcordance(r *temoinRapport, t *testing.T, b *temoinBase, s temoinScope) {
	tags := tagsObserves(t, b, s)
	cles := make([]string, 0, len(tags))
	vu := map[string]bool{}
	for _, e := range tags {
		if e.cle != "" && !vu[e.cle] {
			vu[e.cle] = true
			cles = append(cles, e.cle)
		}
	}
	reg := registreParCle(t, b, cles)
	ecrireA04(r, tags)
	ecrireA05(r, tags, reg)
	ecrireEcartD4(r, tags, reg)
}

// ecrireA04 : combien de tags obtiennent une image, une cle, et lesquels divergent.
func ecrireA04(r *temoinRapport, tags []temoinTag) {
	var avecImage, avecCle, mortsImageSansCle, mortsCleSansImage, mortsNi int
	var imageSansCle, cleSansImage []temoinTag
	for _, e := range tags {
		if e.image != "" {
			avecImage++
		}
		if e.cle != "" {
			avecCle++
		}
		switch {
		case e.image != "" && e.cle == "":
			imageSansCle = append(imageSansCle, e)
			mortsImageSansCle += e.morts
		case e.image == "" && e.cle != "":
			cleSansImage = append(cleSansImage, e)
			mortsCleSansImage += e.morts
		case e.image == "" && e.cle == "":
			mortsNi += e.morts
		}
	}
	r.ligne("## A0.4 — concordance graphe / kill feed (D13)")
	r.ligne("")
	r.ligne("| grandeur | tags | morts |")
	r.ligne("|---|---:|---:|")
	r.ligne("| tags observes en base | %d | %d |", len(tags), totalMorts(tags))
	r.ligne("| (a) obtiennent une IMAGE | %d | |", avecImage)
	r.ligne("| (b) obtiennent une CLE | %d | |", avecCle)
	r.ligne("| (c) image SANS cle | %d | %d |", len(imageSansCle), mortsImageSansCle)
	r.ligne("| (c) cle SANS image | %d | %d |", len(cleSansImage), mortsCleSansImage)
	r.ligne("| ni image ni cle | %d | %d |", len(tags)-avecImage-len(cleSansImage), mortsNi)
	r.ligne("")
	ecrireTableTags(r, "### Tags rendant une IMAGE sans CLE — le rejeu les nomme, le graphe non", imageSansCle)
	ecrireTableTags(r, "### Tags rendant une CLE sans IMAGE", cleSansImage)
}

// ecrireTableTags detaille un lot de tags divergents : volume, classe et nom.
func ecrireTableTags(r *temoinRapport, titre string, tags []temoinTag) {
	r.ligne("%s", titre)
	r.ligne("")
	if len(tags) == 0 {
		r.ligne("_aucun._")
		r.ligne("")
		return
	}
	r.ligne("| tag | morts | classe damagetag | nom kill feed | image | cle |")
	r.ligne("|---|---:|---|---|---|---|")
	for _, e := range tags {
		r.ligne("| 0x%08x | %d | %s | %s | %s | %s |", e.tag, e.morts, e.classe, e.nom, e.image, e.cle)
	}
	r.ligne("")
}

// ecrireA05 : le NOM que le kill feed affiche contre le libelle du registre.
func ecrireA05(r *temoinRapport, tags []temoinTag, reg map[string][2]string) {
	type ecart struct {
		cle, feed, registre string
		morts               int
	}
	agg := map[string]*ecart{}
	for _, e := range tags {
		if e.cle == "" || e.nom == "" {
			continue
		}
		m, ok := reg[e.cle]
		if !ok || m[1] == "" || m[1] == e.nom {
			continue
		}
		k := e.cle + "|" + e.nom
		if agg[k] == nil {
			agg[k] = &ecart{cle: e.cle, feed: e.nom, registre: m[1]}
		}
		agg[k].morts += e.morts
	}
	lignes := make([]*ecart, 0, len(agg))
	for _, v := range agg {
		lignes = append(lignes, v)
	}
	sort.Slice(lignes, func(i, j int) bool { return lignes[i].morts > lignes[j].morts })
	r.ligne("## A0.5 — ecart de NOM entre le kill feed et le registre (D13)")
	r.ligne("")
	if len(lignes) == 0 {
		r.ligne("_aucun ecart : tous les tags a cle portent le meme nom des deux cotes._")
		r.ligne("")
		return
	}
	r.ligne("| cle de registre | nom kill feed (damagetag) | nom registre (weapons.name) | morts |")
	r.ligne("|---|---|---|---:|")
	for _, l := range lignes {
		r.ligne("| %s | %s | %s | %d |", l.cle, l.feed, l.registre, l.morts)
	}
	r.ligne("")
}

// ecrireEcartD4 chiffre ce que la decision D4 ecarte : les morts dont la source resout vers
// une cle de classe melee ou grenade. Elles sont deja servies par les compteurs API, mais le
// volume doit etre connu — c'est lui qui dit si D4 rend le sunburst faux ou juste.
func ecrireEcartD4(r *temoinRapport, tags []temoinTag, reg map[string][2]string) {
	parCle := map[string]int{}
	for _, e := range tags {
		if e.cle == "" {
			continue
		}
		m, ok := reg[e.cle]
		if !ok {
			continue
		}
		if m[0] == domain.FragClassMelee || m[0] == domain.FragClassGrenade {
			parCle[e.cle+" ("+m[0]+")"] += e.morts
		}
	}
	r.ligne("## Volume ecarte par la decision D4 (melee et grenade servies par l'API)")
	r.ligne("")
	if len(parCle) == 0 {
		r.ligne("_aucune source ne resout vers une cle de classe melee ou grenade._")
		r.ligne("")
		return
	}
	cles := make([]string, 0, len(parCle))
	for k := range parCle {
		cles = append(cles, k)
	}
	sort.Slice(cles, func(i, j int) bool { return parCle[cles[i]] > parCle[cles[j]] })
	r.ligne("| cle de registre (classe) | morts vues par la source de degat |")
	r.ligne("|---|---:|")
	for _, k := range cles {
		r.ligne("| %s | %d |", k, parCle[k])
	}
	r.ligne("")
}

// totalMorts somme les morts d'un lot de tags.
func totalMorts(tags []temoinTag) int {
	n := 0
	for _, e := range tags {
		n += e.morts
	}
	return n
}
