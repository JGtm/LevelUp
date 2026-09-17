package killsource

// label.go — DU TAG A L ETIQUETTE.
//
// Le tag est un EFFET, PAS UNE ARME : la relation est MANY-TO-ONE par effet. Le meme fusil porte
// quatre tags distincts (standard, variante d un mode, deux effets). L etiquette honnete est donc
// << source (effet) >>, et c est pour cela que `Detail` accompagne toujours `Display`.
//
// LA TABLE EST EMBARQUEE ET PRECALCULEE (paquet `damagetag`, `go:embed`). Ce fichier ne fait que
// la consulter et appliquer UNE regle d affichage.
//
// LA REGLE D AFFICHAGE, dictee par l utilisateur : *<< les armes sans nom, on dira que c est
// Autres pour le moment >>*. Elle est volontairement simple, et son cout est connu — un encodage
// existe quelque part pour les 206 tags sans source remontee, ce n est pas la priorite du lot.
//
//	nom propre present ET etiquette publiable  -> le nom propre
//	tout le reste                              -> [LabelOther]
//
// CE QUE LA REGLE NE FAIT PAS PERDRE : `Class`, `Status`, `Detail` et `Reserve` restent remplis a
// cote. Un consommateur qui veut afficher << MELEE >> plutot que << Autres >>, ou traduire, a
// tout ce qu il lui faut sans que ce paquet ait a porter un libelle traduit — regle du depot :
// aucun libelle FR/EN en dur cote Go.
//
// DEUX LIGNES DE LA TABLE SONT MARQUEES `AMBIGU`, ET C EST MESURE : ce sont des effets GENERIQUES
// qui traversent plusieurs entrees de la liste des grenades ; publier un nom y serait affirmatif
// et faux. `Publishable()` rend `false`, donc elles sortent en [LabelOther]. Le critere n est PAS
// code en dur sur ces deux tags : le generateur ENUMERE les entrees atteintes et marque AMBIGU
// des qu il y en a plus d une — il attrapera donc les cas d une saison future.

import "levelup/go-api/internal/games/halo_infinite/film/damagetag"

// sourceTruthOf : la verite SOURCE d un tag et d une categorie.
func sourceTruthOf(tag uint32, cat int) SourceTruth {
	s := SourceTruth{
		Tag:      tag,
		Display:  LabelOther,
		Class:    damagetag.ClassInconnu,
		Status:   damagetag.StatusInconnu,
		Category: Category(cat),
	}
	l, ok := damagetag.Lookup(tag)
	if !ok {
		return s
	}
	s.Class, s.Status, s.Detail, s.Reserve = l.Class, l.Status, l.Detail, l.Reserve
	if l.Name != "" && l.Publishable() {
		s.Display, s.Named = l.Name, true
	}
	return s
}

// isCatalogued : le tag est-il une entree connue du groupe `jpt!` ? C est le test T4 du scan, et
// c est aussi le test de PLAUSIBILITE des compteurs de sante. Un seul point d appel vers le
// paquet de donnees, pour qu il n existe jamais deux definitions du catalogue.
func isCatalogued(tag uint32) bool { return damagetag.IsDamageEffect(tag) }

// CatalogueProvenance : d ou viennent les tables embarquees, pour qu une sortie puisse porter sa
// date de generation. La peremption d une table se paie en COUVERTURE, silencieusement.
func CatalogueProvenance() damagetag.Provenance { return damagetag.Source() }

// CatalogueSize : nombre d ids `jpt!` embarques.
func CatalogueSize() int { return damagetag.Size() }
