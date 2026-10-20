package filmprofile

// empreinte_registre.go — LES EMPREINTES DU REGISTRE ECS, PAR CLEF ECRITE DU FILM.
//
// # CE QUE C EST
//
// Le registre ECS d un film (`chunk_00`) porte les NOMS des composants et leur ORDRE : c est
// lui qui route toute la grammaire du decodeur. Il est bit-a-bit identique d un film a l autre
// DANS UN MEME BUILD, et il change d un build a l autre. Son empreinte — un FNV-1a 64 bits sur
// les entrees nommees — est donc l identite de la grammaire des composants du film.
//
// Le decodeur ne connaissait qu UNE empreinte, en constante de paquet : celle du build de
// reference. Tout ce qui n etait pas elle sortait en une ligne de journal « registre INCONNU »,
// dedupliquee par processus, qui melangeait « grammaire d un autre build, connue du depot » et
// « grammaire jamais vue ». Cette section-ci est la TABLE qui manquait : une empreinte par
// clef ecrite, avec ce qui l a mesuree.
//
// # POURQUOI UNE SECTION A PART, ET PAS DES LIGNES D `entries`
//
// Deux raisons, mesurees toutes les deux :
//
//  1. UNE EMPREINTE N EST PAS UNE PHRASE. `Entree.Valeur` est une chaine « ecrite pour etre lue
//     par un humain » ; un test de conformite ne peut mordre dessus que par analyse de texte.
//     Une empreinte, un nombre de blocs et un nombre de slots sont des GRANDEURS : elles se
//     comparent valeur contre valeur (question Q3 de la note de preparation M3, 2026-09-16).
//  2. `entries` EST LA TABLE DU LOT 2.1, LIGNE POUR LIGNE. `TestCatalogueConformeALaTableDuLot21`
//     tient les deux egales tant qu elles coexistent : une ligne ajoutee a `entries` sans la
//     meme ligne dans `film/profile/profile_table.go` rougit — expres. La table du lot 2.1 ne
//     porte aucune empreinte de registre, et ce lot-ci ne touche aucun paquet du film.
//
// # LA CLEF EST CE QUE LE FILM ECRIT — ET CERTAINS FILMS N ECRIVENT PAS DE BUILD
//
// Les clefs admises ici sont `build=<id>` et `majeure=<n>`, et elles seules :
//
//	build=<id>    le cas normal : la chaine en clair de la section 2 de `chunk_00`.
//	majeure=<n>   les films dont `chunk_00` ne porte AUCUNE section d identification (mesure :
//	              5 films du cache, versions majeures 31 et 33). Ils n ecrivent pas de build ;
//	              la version majeure est alors la seule clef qu ils portent.
//
// `toutes` et `format=` sont REFUSEES : une empreinte de registre est une propriete du BUILD DU
// JEU qui a ecrit le film, jamais un invariant du depot ni une propriete du format de bits.
//
// # DEUX FAITS MESURES QU IL FAUT LIRE AVANT DE S ETONNER
//
//	L EMPREINTE N EST PAS UNIQUE. Trois paires de clefs partagent la leur : HI_1_8_0 et
//	HI_1_9_0 ; HI_1_12_0 et HI_1_13_0 ; HI_1_4_1 et les films de majeure 33 sans section. La
//	frontiere du registre n est donc PAS celle des builds — l unicite validee ici porte sur la
//	CLEF, jamais sur l empreinte.
//
//	UNE CLEF `majeure` PEUT SELECTIONNER UN FILM QUI ECRIT UN BUILD. `majeure=33` couvre les
//	films sans section de cette version, mais aussi `a521164d`, qui ecrit `HI_1_4_1` : les deux
//	entrees portent la meme empreinte, il n y a pas de contradiction, et
//	[Catalogue.EmpreinteRegistrePour] rend de toute facon la plus specifique — celle du build.

import (
	"fmt"
	"strings"
	"time"
)

// prefixeEmpreinte / chiffresEmpreinte : une empreinte FNV-1a 64 bits s ecrit `0x` suivi de
// SEIZE chiffres hexadecimaux minuscules — la forme que produit `fmt.Sprintf("0x%016x")`. La
// largeur est validee : une empreinte tronquee (`0x36ca8c3d`) ne serait egale a aucune valeur
// lue et passerait pour une grammaire inconnue.
const (
	prefixeEmpreinte  = "0x"
	chiffresEmpreinte = 16
)

// StatutEmpreinte dit ce que le depot SAIT de la population couverte par la clef.
//
// CE N EST PAS LA PROVENANCE, et les confondre est le piege de cette table. La provenance dit
// d ou vient la LIGNE (relue chez l ecrivain / mesuree sur temoin / presumee) ; le statut dit
// jusqu ou la mesure PORTE :
//
//	connue    l empreinte est lue sur au moins deux films distincts de la clef, ou sur le seul
//	          film que le cache en porte — la population est couverte.
//	presumee  un seul film de la clef a ete lu alors que le cache en porte d autres : rien ne
//	          contredit la valeur, rien ne prouve encore qu elle vaut pour toute la clef.
//
// Le troisieme etat que l item 3.2.1 demande — `inconnue` — n a pas sa place ici : il qualifie
// une empreinte LUE DANS UN FILM et absente de cette table. C est une classification a
// l execution, elle nait avec le volet code du lot (publication dans `coverage.decoder`).
type StatutEmpreinte string

// Les deux statuts, et rien d autre.
const (
	StatutConnue   StatutEmpreinte = "connue"
	StatutPresumee StatutEmpreinte = "presumee"
)

// EstConnu dit si le statut est l un des deux.
func (s StatutEmpreinte) EstConnu() bool {
	return s == StatutConnue || s == StatutPresumee
}

// EmpreinteRegistre est UNE empreinte de registre ECS, pour UNE clef ecrite du film.
type EmpreinteRegistre struct {
	// Cle est la clef ECRITE que cette empreinte couvre (`build=HI_1_11_0`, `majeure=31`).
	Cle string `json:"key"`
	// Empreinte est le FNV-1a 64 bits des entrees nommees, `0x` + 16 chiffres minuscules.
	Empreinte string `json:"fingerprint"`
	// Blocs est le nombre de blocs d archetype que porte le registre.
	Blocs int `json:"blocks"`
	// SlotsNommes est le nombre d entrees NOMMEES — le denominateur de l alerte.
	SlotsNommes int `json:"namedSlots"`
	// Statut dit jusqu ou la mesure porte (cf. [StatutEmpreinte]).
	Statut StatutEmpreinte `json:"status"`
	// Source est la provenance de la ligne, comme pour toute entree saisie.
	Source Provenance `json:"provenance"`
	// Temoins sont les films sur lesquels l empreinte a ete LUE, en forme courte (8 hex). Leur
	// NOMBRE est le nombre de films temoins : il n est pas recopie a cote, une seconde ecriture
	// du meme compte divergerait au premier temoin ajoute.
	Temoins []string `json:"witnesses"`
	// Preuve est l instrument, le journal ou l oracle qui a rendu la valeur.
	Preuve string `json:"proof"`
	// Date est le jour de la mesure, `AAAA-MM-JJ`.
	Date string `json:"date"`
}

// EmpreinteRegistrePour rend l empreinte que le depot connait pour les clefs lues d un film.
//
// Elle rend la clef la PLUS SPECIFIQUE : un film qui ecrit un build est decrit par son build,
// jamais par sa version majeure (cf. l en-tete). A defaut de build, la premiere clef `majeure`
// qui correspond, dans l ordre du fichier.
//
// Le booleen est faux quand le depot ne sait rien de ce film : c est l etat `inconnue` de la
// classification, et c est une reponse, pas un defaut a combler par une valeur voisine.
func (c *Catalogue) EmpreinteRegistrePour(lues ClesLues) (EmpreinteRegistre, bool) {
	var repli EmpreinteRegistre
	var aRepli bool
	for _, e := range c.EmpreintesRegistre {
		cle, err := ParseCle(e.Cle)
		if err != nil {
			continue // [Catalogue.Valide] a deja refuse ce catalogue a la lecture
		}
		if !cle.Correspond(lues) {
			continue
		}
		if cle.Sorte == SorteBuild {
			return e, true
		}
		if !aRepli {
			repli, aRepli = e, true
		}
	}
	return repli, aRepli
}

// valideEmpreintesRegistre rend toutes les anomalies de la section, d un coup.
//
// La section est FACULTATIVE : un titre dont le decodeur ne lit pas encore de registre n a
// aucune empreinte a poser, et un catalogue sans table d empreintes reste valide. Ce qui ne
// l est pas, c est une ligne a moitie ecrite.
func (c *Catalogue) valideEmpreintesRegistre() []error {
	var anomalies []error
	builds := c.buildsDesEntrees()
	vues := map[string]int{}
	for i, e := range c.EmpreintesRegistre {
		anomalies = append(anomalies, e.valide(i, builds)...)
		if precedent, deja := vues[e.Cle]; deja {
			anomalies = append(anomalies, fmt.Errorf("registryFingerprints[%d] : la clef %q est "+
				"deja posee en registryFingerprints[%d] — deux empreintes pour une meme "+
				"grammaire, aucune ne prime", i, e.Cle, precedent))
			continue
		}
		vues[e.Cle] = i
	}
	return anomalies
}

// buildsDesEntrees rend les builds que la table saisie du catalogue nomme deja.
//
// C est la table des builds du depot : une empreinte posee sur un build qui n y figure pas
// serait une grammaire orpheline — le catalogue dirait connaitre le registre d un build dont
// il ne sait rien d autre.
func (c *Catalogue) buildsDesEntrees() map[string]bool {
	builds := map[string]bool{}
	for _, e := range c.Entrees {
		cle, err := ParseCle(e.Cle)
		if err != nil || cle.Sorte != SorteBuild {
			continue
		}
		builds[cle.Build] = true
	}
	return builds
}

// valide rend les anomalies d UNE empreinte.
func (e EmpreinteRegistre) valide(i int, builds map[string]bool) []error {
	anomalies := e.valideLaClef(i, builds)
	if err := empreinteBienFormee(e.Empreinte); err != nil {
		anomalies = append(anomalies, fmt.Errorf("registryFingerprints[%d] (%q) : %w", i, e.Cle, err))
	}
	if e.Blocs <= 0 {
		anomalies = append(anomalies, fmt.Errorf("registryFingerprints[%d] (%q) : blocks %d — un "+
			"registre sans bloc n a pas ete lu", i, e.Cle, e.Blocs))
	}
	if e.SlotsNommes <= 0 {
		anomalies = append(anomalies, fmt.Errorf("registryFingerprints[%d] (%q) : namedSlots %d — "+
			"le denominateur de l alerte ne peut pas etre nul", i, e.Cle, e.SlotsNommes))
	}
	if !e.Statut.EstConnu() {
		anomalies = append(anomalies, fmt.Errorf("registryFingerprints[%d] (%q) : status %q "+
			"inconnu — %q ou %q, et rien d autre", i, e.Cle, e.Statut, StatutConnue, StatutPresumee))
	}
	if !e.Source.EstConnue() {
		anomalies = append(anomalies, fmt.Errorf("registryFingerprints[%d] (%q) : provenance %q "+
			"inconnue — %q, %q ou %q", i, e.Cle, e.Source, ProvenanceRelue, ProvenanceMesuree,
			ProvenancePresumee))
	}
	anomalies = append(anomalies, e.valideLesTemoins(i)...)
	if e.Preuve == "" {
		anomalies = append(anomalies, fmt.Errorf("registryFingerprints[%d] (%q) : proof vide — "+
			"une empreinte sans l instrument qui l a rendue ne se rejoue pas", i, e.Cle))
	}
	if _, err := time.Parse(formatDeDate, e.Date); err != nil {
		anomalies = append(anomalies, fmt.Errorf("registryFingerprints[%d] (%q) : date %q n est "+
			"pas une date AAAA-MM-JJ", i, e.Cle, e.Date))
	}
	return anomalies
}

// valideLaClef refuse une clef illisible, une sorte qui n a pas de sens pour un registre, et un
// build absent de la table des builds du catalogue.
func (e EmpreinteRegistre) valideLaClef(i int, builds map[string]bool) []error {
	cle, err := ParseCle(e.Cle)
	if err != nil {
		return []error{fmt.Errorf("registryFingerprints[%d] : %w", i, err)}
	}
	switch cle.Sorte {
	case SorteBuild:
		if !builds[cle.Build] {
			return []error{fmt.Errorf("registryFingerprints[%d] : build %q absent de la table "+
				"des builds du catalogue (aucune entree `build=%s`) — une grammaire posee sur un "+
				"build dont le depot ne sait rien d autre", i, cle.Build, cle.Build)}
		}
	case SorteMajeure:
		if cle.Comparateur != ComparateurEgal || len(cle.Majeures) != 1 {
			return []error{fmt.Errorf("registryFingerprints[%d] (%q) : une empreinte de registre "+
				"se pose sur UNE version majeure precise (`majeure=33`), jamais sur un intervalle "+
				"— le registre change de build en build a l interieur d une meme majeure", i, e.Cle)}
		}
	default:
		return []error{fmt.Errorf("registryFingerprints[%d] (%q) : sorte de clef %q — une "+
			"empreinte de registre est une propriete du BUILD DU JEU : `build=<id>`, ou "+
			"`majeure=<n>` pour les films sans section d identification", i, e.Cle, cle.Sorte)}
	}
	return nil
}

// valideLesTemoins refuse une empreinte que personne n a lue, et un temoin ecrit deux fois.
func (e EmpreinteRegistre) valideLesTemoins(i int) []error {
	if len(e.Temoins) == 0 {
		return []error{fmt.Errorf("registryFingerprints[%d] (%q) : witnesses vide — une empreinte "+
			"se lit sur un film, elle ne se postule pas", i, e.Cle)}
	}
	var anomalies []error
	vus := map[string]bool{}
	for _, t := range e.Temoins {
		switch {
		case strings.TrimSpace(t) == "":
			anomalies = append(anomalies, fmt.Errorf("registryFingerprints[%d] (%q) : temoin vide",
				i, e.Cle))
		case vus[t]:
			anomalies = append(anomalies, fmt.Errorf("registryFingerprints[%d] (%q) : temoin %q "+
				"cite deux fois — le compte des films temoins serait faux", i, e.Cle, t))
		}
		vus[t] = true
	}
	return anomalies
}

// empreinteBienFormee refuse tout ce qui n est pas `0x` + 16 chiffres hexadecimaux minuscules.
func empreinteBienFormee(s string) error {
	if !strings.HasPrefix(s, prefixeEmpreinte) {
		return fmt.Errorf("fingerprint %q : prefixe %q attendu", s, prefixeEmpreinte)
	}
	chiffres := strings.TrimPrefix(s, prefixeEmpreinte)
	if len(chiffres) != chiffresEmpreinte {
		return fmt.Errorf("fingerprint %q : %d chiffre(s) apres %q, %d attendus (FNV-1a 64 bits, "+
			"zeros de tete compris)", s, len(chiffres), prefixeEmpreinte, chiffresEmpreinte)
	}
	for _, r := range chiffres {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return fmt.Errorf("fingerprint %q : %q n est pas un chiffre hexadecimal minuscule",
				s, string(r))
		}
	}
	return nil
}
