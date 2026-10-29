package profile

// classement_registre.go — LA CLASSIFICATION D UNE EMPREINTE DE REGISTRE LUE DANS UN FILM :
// `connue` / `presumee` / `inconnue`, DEFINIE EN UN SEUL ENDROIT (lot 3.1.1, item 3.2.1).
//
// # LE DEFAUT QUE CE FICHIER FERME
//
// Jusqu au lot 2.6.3, le decodeur ne connaissait qu UNE empreinte de registre — une constante de
// paquet, celle du build de reference. Tout le reste sortait `inconnue`, ce qui melangeait
// « grammaire d un autre build, connue du depot » et « grammaire jamais vue ». Sur les 17
// temoins du corpus gate, CINQ empreintes parfaitement mesurees et ecrites au catalogue
// sortaient `inconnue` : `HI_1_8_0` / `HI_1_9_0`, `HI_1_10_0`, `HI_1_11_0`, `HI_1_4_1`.
//
// La table des empreintes (registre_empreintes.go) recopie desormais le catalogue, et ce fichier
// en tire les TROIS cas. Le statut publie (`coverage.decoder.registry.status`) reste une CHAINE,
// exactement pour que ce troisieme etat n oblige aucun consommateur a changer de forme.
//
// # LES TROIS CAS, ET LA FRONTIERE ENTRE LES DEUX DERNIERS
//
//	connue    l empreinte lue est celle que la table ATTEND pour CETTE clef. C est l accord
//	          complet : la grammaire des composants est celle que le depot a mesuree sur ce
//	          build-la.
//	presumee  l empreinte lue est AU CATALOGUE, mais pas comme valeur attendue de cette clef.
//	          Deux regimes tombent ici, et ils disent la meme chose : soit la clef a une
//	          empreinte attendue et le film en porte une autre — connue d une clef VOISINE —,
//	          soit la clef n a AUCUNE empreinte attendue (build neuf) et le film porte une
//	          grammaire que le depot connait deja. Dans les deux cas la grammaire des
//	          composants est LUE quelque part au depot : le film se decode sous une grammaire
//	          etablie, mais pas sous celle que sa clef annonce.
//	inconnue  l empreinte lue n est NULLE PART au catalogue. C est l evenement « le jeu a
//	          change de grammaire de composants », et c est le seul des trois qui demande une
//	          mesure neuve (`docs/RUNBOOK_FILM_PROFILES.md` §4.5).
//
// # CE QUE LA CLASSIFICATION NE FAIT PAS
//
// Elle ne refuse AUCUN film et ne change AUCUNE largeur : c est un signal, pas une porte. La
// porte, elle, est la CLEF ([CleConnue]) et non l empreinte — D2 (3.2) l a mesure : trois paires
// de clefs partagent leur empreinte, donc une empreinte egale n autorise aucune conclusion sur
// les tailles de structures, qui different entre ces memes builds.

import "strconv"

// StatutRegistre classe une empreinte LUE DANS UN FILM contre la table des empreintes.
//
// CE N EST PAS [StatutCatalogue], et les confondre est le piege de ces deux tables : celui-la
// dit jusqu ou la MESURE d une ligne porte, celui-ci ce qu un film VAUT contre elle.
type StatutRegistre string

// Les trois statuts, et rien d autre. Les valeurs sont celles que l artefact publie
// (`coverage.decoder.registry.status`, schema 61) : les changer est une montee de schema.
const (
	// StatutRegistreConnue : l empreinte lue est celle que la table attend pour cette clef.
	StatutRegistreConnue StatutRegistre = "connue"
	// StatutRegistrePresumee : l empreinte lue est au catalogue, mais sous une AUTRE clef — ou
	// la clef du film n a pas d empreinte attendue.
	StatutRegistrePresumee StatutRegistre = "presumee"
	// StatutRegistreInconnue : l empreinte lue est absente du catalogue.
	StatutRegistreInconnue StatutRegistre = "inconnue"
)

// ClasserEmpreinteRegistre rend le statut d une empreinte lue dans un film, pour les clefs que
// ce film ECRIT — le build quand il en ecrit un, la version majeure sinon.
//
// C EST LE SEUL ENDROIT OU LES TROIS CAS SE DECIDENT. `replay` le publie, il ne le recalcule
// pas : une seconde table de verite divergerait au premier build ajoute (CLAUDE.md regle 6).
func ClasserEmpreinteRegistre(build string, majeure int, empreinte uint64) StatutRegistre {
	if attendue, ok := EmpreinteRegistreAttendue(build, majeure); ok &&
		attendue.Empreinte == empreinte {
		return StatutRegistreConnue
	}
	if EmpreinteAuCatalogue(empreinte) {
		return StatutRegistrePresumee
	}
	return StatutRegistreInconnue
}

// EmpreinteRegistreAttendue rend l empreinte que la table ATTEND pour les clefs lues d un film.
//
// Elle rend la clef la PLUS SPECIFIQUE : un film qui ecrit un build est decrit par son build,
// jamais par sa version majeure — c est la meme regle que `filmprofile.EmpreinteRegistrePour`,
// dont cette fonction est le pendant cote decodeur. A defaut de build, la premiere clef
// `majeure=` qui correspond, dans l ordre de la table.
//
// Le booleen est faux quand la table ne connait ni le build ni la majeure du film : c est le cas
// d un build NEUF, et c est une reponse — pas un defaut a combler par la valeur d un voisin.
func EmpreinteRegistreAttendue(build string, majeure int) (EmpreinteRegistre, bool) {
	var parMajeure EmpreinteRegistre
	var majeureVue bool
	for _, e := range EmpreintesRegistre() {
		switch {
		case build != "" && e.Cle == clefBuild(build):
			return e, true
		case e.Cle == clefMajeure(majeure) && !majeureVue:
			parMajeure, majeureVue = e, true
		}
	}
	if build != "" {
		// LE BUILD PRIME, ET SON ABSENCE NE SE COMBLE PAS PAR LA MAJEURE : un film qui ECRIT un
		// build que la table ignore n est pas decrit par la clef `majeure=` qui l englobe — les
		// clefs `majeure=` sont celles des films SANS section d identification (cf. l en-tete de
		// registre_empreintes.go). Rendre la moins specifique ici ferait passer un build neuf pour
		// connu.
		return EmpreinteRegistre{}, false
	}
	return parMajeure, majeureVue
}

// EmpreinteAuCatalogue dit si une empreinte figure dans la table, SOUS N IMPORTE QUELLE CLEF.
// C est la frontiere entre `presumee` et `inconnue`.
func EmpreinteAuCatalogue(empreinte uint64) bool {
	for _, e := range EmpreintesRegistre() {
		if e.Empreinte == empreinte {
			return true
		}
	}
	return false
}

// clefBuild / clefMajeure composent les deux formes de clef ecrite. LES LITTERAUX SONT ICI ET
// NULLE PART AILLEURS : la table, la classification et le recensement lisent la meme grammaire
// de clef, et une troisieme copie divergerait au premier prefixe renomme.
func clefBuild(build string) string { return prefixeClefBuild + build }

func clefMajeure(majeure int) string { return prefixeClefMajeure + strconv.Itoa(majeure) }

// Les deux prefixes de clef, ecrits une fois (cf. `docs/RUNBOOK_FILM_PROFILES.md` §1).
const (
	prefixeClefBuild   = "build="
	prefixeClefMajeure = "majeure="
)

// CleEcrite compose la clef SELECTIONNANTE d un film : son build quand il en ecrit un, sa
// version majeure sinon. C est la forme que le catalogue, la table et les journaux emploient.
//
// ELLE EST LA PORTE UNIQUE DE CETTE COMPOSITION : `replay` la journalise et la publie, il ne la
// recompose pas — un `"build=" + build` recopie ailleurs divergerait au premier prefixe renomme.
func CleEcrite(build string, majeure int) string {
	if build != "" {
		return clefBuild(build)
	}
	return clefMajeure(majeure)
}
