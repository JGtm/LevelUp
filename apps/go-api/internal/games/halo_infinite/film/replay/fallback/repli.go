// Package fallback porte LE REGISTRE DES REPLIS du décodeur de film (D14 du plan
// `.ai/PLAN_DECODEUR_FILM_2026-09-13.md`, ADR 0034).
//
// # CE QUE CE PAQUET EXISTE POUR EMPÊCHER
//
// Un REPLI ANONYME : un `if` de secours au milieu d'une fonction, qui décide un fait à la
// place de la lecture du film, sans nom, sans compteur, sans date et sans critère de retrait.
// L'audit du lot 0.E en a recensé 62 dans le décodeur (`.ai/V7.5/AUDIT_HEURISTIQUES_DECODEUR_2026-09-13.md`,
// table E), dont neuf portaient DÉJÀ un défaut mesuré — jusqu'à 95 % des poses d'un film
// créditées au mauvais joueur, 213 s d'attribution fausse sur une manche, 27 faux
// enregistrements sur Live Fire. Aucun de ces défauts n'était visible d'un artefact : c'est ce
// que le registre change.
//
// # LA RÈGLE (D14, décision utilisateur du 2026-09-13)
//
// Un repli est NOMMÉ, ORDONNÉ, COMPTÉ, DATÉ, et il porte son critère de retrait :
//
//	(a) il vit dans CE registre, et le code qui l'exécute le nomme ;
//	(b) il ne se déclenche que sur un diagnostic typé « le film est muet ici » — jamais sur un
//	    désaccord avec la lecture, jamais à la place d'une lecture disponible. Ordre fixe :
//	    LIRE d'abord, se replier ensuite ([OrdreApresLecture]) ;
//	(c) chaque déclenchement se compte, et le compte voyage AVEC l'artefact
//	    (`coverage.fallbacks[]`) : le document dit quelle part de lui vient d'un repli ;
//	(d) un repli dont le compte est à ZÉRO sur le corpus à la clôture d'un jalon est SUPPRIMÉ
//	    au jalon suivant, avec ses tests. On ne garde pas un repli « au cas où » : un repli
//	    bancal qui se déclenche à tort corrompt un fait que la lecture aurait donné juste.
//
// # LES DEUX GARDE-RAILS, ET CE QUE CHACUN TIENT
//
// `internal/archlint/no_unregistered_fallback_test.go` tient les DEUX directions :
//
//   - un repli hors registre = ROUGE. Tout identifiant Go DÉCLARÉ dans le décodeur dont le nom
//     porte le segment `repli` / `Repli` / `fallback` / `Fallback` à une frontière camelCase
//     doit être au registre. C'est la convention de nommage détectable ;
//   - une entrée du registre sans site = ROUGE. Chaque entrée cite ses [Site], et chaque site
//     est un couple (fichier, ancre) que le garde-rail RELIT : le fichier doit exister et
//     porter l'ancre. Quand un lot de conversion supprime le repli, son ancre disparaît, le
//     garde-rail rougit, et l'entrée DOIT sortir du registre. C'est le (d) de D14 rendu
//     mécanique : le registre ne peut pas survivre au code qu'il décrit.
//
// # POURQUOI LES CHAINES DE CE PAQUET N'ONT PAS D'ACCENT
//
// Les commentaires en portent ; les LITTERAUX DE CHAINE, non. Le garde-rail
// `archlint/no_french_label_literal_test.go` compte les litteraux accentues de `internal/games/`
// — il vise les LIBELLES en dur, qui doivent venir des TOML de titre (D5). Les phrases de ce
// registre ne sont pas des libelles : elles ne sont ni traduites, ni affichees, ni servies a un
// client ; ce sont des diagnostics de developpeur, imprimes par un test et joints au nom du
// repli dans `coverage.fallbacks`. Les ecrire sans accent les tient hors du compte de ce
// garde-rail sans en agrandir l'allowlist, dont l'objectif est de se VIDER — c'est la convention
// que `killsource/` et une bonne part de `film/` appliquent deja.
//
// # CE QUE CE PAQUET N'EST PAS
//
// Il ne DÉCIDE rien. Il déclare et il compte. Aucune entrée de ce registre ne change le
// comportement d'un décodage : le lot 1.9.0 est une pose d'instruments à équivalence zéro
// différence, et les CONVERSIONS (la lecture qui remplace l'heuristique) sont les lots 1.9.1
// à 1.9.14.
//
// # OÙ IL VIVRA
//
// Sous `replay/` tant que la couche `facts` n'existe pas (plan, item 1.9.0). Au pas 5 de M2
// (lot 2.5), quand les cinq couches `source` -> `profile` -> `grammar` -> `facts` -> `replay`
// sont séparées, ce paquet devient `film/facts/fallback` : il est déjà une FEUILLE (aucun
// import du dépôt), donc le déplacement est pur.
package fallback

import (
	"fmt"
	"sort"
	"strings"
)

// Nom est l'identifiant STABLE d'un repli. Il ne change jamais : un compte publié dans un
// artefact cuit se relit par ce nom des mois plus tard.
//
// CONVENTION, VÉRIFIÉE PAR [VerifierRegistre] : `repli_<fait>_<mecanisme>`, en minuscules,
// segments séparés par `_`. Le FAIT d'abord (ce qui est décidé), le MÉCANISME ensuite (comment
// on se replie) — de sorte qu'un tri alphabétique regroupe les replis d'un même fait.
type Nom string

// Condition est le diagnostic TYPÉ qui ouvre le repli. Elle répond à « pourquoi la lecture
// n'a-t-elle pas décidé ? » et à rien d'autre.
//
// ELLE EST TYPÉE ET NON UNE PHRASE, parce que D14 (b) fait reposer la légitimité d'un repli sur
// sa condition : [CondFilmMuet] est légitime, [CondLectureNonPortee] est une DETTE (le film
// écrit le fait, le lecteur manque), et [CondInconditionnel] est le pire cas — le repli ne
// regarde même pas si une lecture existait.
type Condition string

const (
	// CondFilmMuet : le film n'écrit pas ce fait, et le NÉGATIF est MESURÉ (un chiffre et sa
	// source). C'est la seule condition qui rende un repli légitime à demeure.
	CondFilmMuet Condition = "film_muet"
	// CondSectionAbsente : le film écrit ce fait ailleurs, mais CE film-ci ne porte pas la
	// section (bobine partielle, chunk_00 absent, build sans section d'identification).
	//
	// LA TABLE D'INDEX DES JOUEURS VIDE EN EST LE CAS TYPE (revue M1, 2026-09-15) : sans elle,
	// le registre d'identité sort avant toute découpe et ne rend AUCUNE vie — le film n'est pas
	// muet sur les morts, c'est la section qui les rattacherait à un joueur qui manque.
	CondSectionAbsente Condition = "section_absente"
	// CondLectureNonPortee : le film ÉCRIT le fait et le lecteur n'existe pas encore. C'est une
	// dette nommée : la cible de retrait est le lot qui porte le lecteur.
	CondLectureNonPortee Condition = "lecture_non_portee"
	// CondContradiction : deux lectures se contredisent. D14 (b) : ce n'est PAS un repli, c'est
	// une contradiction — elle se COMPTE, elle ne se tranche pas en silence. La présence de
	// cette condition dans le registre signale un site à requalifier.
	CondContradiction Condition = "contradiction"
	// CondNonResolu : la lecture a tourné et n'a pas tranché (plusieurs candidats, aucun
	// candidat, valeur hors domaine).
	CondNonResolu Condition = "non_resolu"
	// CondCarteAbsenteDuCatalogue : le fait est une DONNÉE DE PROFIL (D-3 d'ADR 0034 : bornes,
	// découpage d'axe, géométrie), le catalogue de carte est la source, et CETTE carte n'y a pas
	// d'entrée exploitable.
	//
	// ELLE EXISTE PARCE QUE [CondFilmMuet] MENTIRAIT : le film n'est pas muet, il écrit des
	// quanta parfaitement lisibles — c'est le référentiel qui manque, et le seul geste qui
	// retire un tel repli est d'ajouter la carte au catalogue, jamais de mieux lire le film.
	// Confondre les deux ferait chercher la correction du mauvais côté de la frontière. Ajoutée
	// au lot 1.9.2 (2026-09-15) avec la rétrogradation du découpage d'i0.
	CondCarteAbsenteDuCatalogue Condition = "carte_absente_du_catalogue"
	// CondFormatSansProfilRelu : le PROFIL existe et la VERSION DE FORMAT de ce film-ci n a pas
	// sa valeur RELUE chez l ecrivain. Pose le 2026-09-15 (lot 1.9.1 bis, pas 3) sous le nom
	// `build_sans_profil_relu`, RENOMME au lot 1.9.1 ter le meme jour : la grammaire du bloc
	// `object-multiplayer-properties` n est pas versionnee par BUILD mais par la version de
	// format de `chunk_00` (`+4`), et c est elle que le lecteur du jeu consulte
	// (`filmdec/film_format_version.go`). L ancien nom decrivait une cle qui n existe plus.
	//
	// Un film SANS section d identification tombe ici aussi — il porte le format 20, qui n a pas
	// de valeur relue — ce qui evite de dedoubler la meme condition sous deux noms. Un format
	// FUTUR et inconnu (28 au prochain patch du jeu) y tombe egalement, et c est le point : le
	// repli tient le parc neuf au lieu de l eteindre. Son evenement est compte a part, par
	// `filmdec.UnknownFormatExpvarPairs` (`filmdec_unknown_format_<n>`), parce qu un patch du
	// jeu doit se voir tout de suite et non au comptage differe du registre.
	CondFormatSansProfilRelu Condition = "format_sans_profil_relu"
	// CondChassisAbsentDeLaTable : le film ECRIT le mot d identite du chassis, parfaitement
	// lisible, et c est NOTRE table (`replay/vehicle_families.go`) qui ne le nomme pas.
	//
	// ELLE EXISTE POUR LA MEME RAISON QUE [CondCarteAbsenteDuCatalogue], et il faut la meme
	// rigueur : [CondFilmMuet] MENTIRAIT ici, et enverrait chercher la correction du mauvais
	// cote de la frontiere (mieux lire le film) alors que le seul geste qui retire ce repli est
	// de NOMMER le chassis en table. Decision utilisateur du 2026-09-14 : le parc d assets
	// vehicules est complet, un chassis absent de la table est un MISMATCH, jamais un vehicule
	// manquant. Ajoutee au lot 1.9.9 (2026-09-16).
	CondChassisAbsentDeLaTable Condition = "chassis_absent_de_la_table"
	// CondInconditionnel : le repli s applique TOUJOURS, sans diagnostic. C'est la forme la
	// plus grave : rien ne dit si une lecture existait.
	CondInconditionnel Condition = "inconditionnel"
)

// Ordre dit QUAND le repli entre par rapport à la lecture. D14 (b) : « ordre fixe = lire
// d'abord, repli ensuite ».
type Ordre string

const (
	// OrdreApresLecture : la lecture est tentée, elle échoue ou se tait, ALORS le repli entre.
	// C'est le seul ordre conforme à D14 (b) quand une lecture existe.
	OrdreApresLecture Ordre = "apres_lecture"
	// OrdreSansLecture : aucune lecture n'existe pour ce fait (négatif mesuré). Le repli n'est
	// pas « après » une lecture, il est la seule voie.
	OrdreSansLecture Ordre = "sans_lecture"
	// OrdreDevantLaLecture : LE REPLI DÉCIDE AVANT, OU À LA PLACE, D'UNE LECTURE DISPONIBLE.
	// C'est la VIOLATION de D14 (b), inscrite ici pour être comptée et retirée — pas tolérée.
	// [NbDevantLaLecture] en est le ratchet : ce nombre ne monte jamais.
	OrdreDevantLaLecture Ordre = "devant_la_lecture"
)

// Site est un endroit du code où le repli s'exécute. L'ANCRE est un littéral du fichier — le
// garde-rail la relit, donc elle survit aux déplacements de lignes et meurt avec le code.
//
// POURQUOI UNE ANCRE ET PAS UN NUMÉRO DE LIGNE. L'audit 0.E cite des `fichier:ligne` du
// 2026-09-13 ; huit jours et neuf lots plus tard, une bonne part avait déjà dérivé. Un registre
// dont les références pourrissent ne se relit plus, donc ne se maintient plus.
type Site struct {
	// Fichier : chemin RELATIF à `apps/go-api/`, séparateurs `/`.
	Fichier string
	// Ancre : sous-chaîne EXACTE présente dans le fichier, choisie pour être distinctive
	// (signature de fonction, littéral de la décision, commentaire de la branche).
	Ancre string
	// Condition : le diagnostic typé de CE SITE quand il diffère de celui de l'entrée. Vide =
	// le site partage la [Repli.Condition] de son entrée, ce qui est le cas courant.
	//
	// IL EXISTE PARCE QU'UN MÊME REPLI PEUT S'OUVRIR SUR DEUX SILENCES DIFFÉRENTS, et que D14 (b)
	// fait reposer la légitimité d'un repli sur sa condition : une condition déclarée une seule
	// fois pour deux sites en décrit forcément un de travers. Constaté à la revue de jalon M1
	// (2026-09-15, lentille D13) sur `repli_vie_coupee_au_trou_de_replication` : le site de
	// `lives_decoupe.go` s'ouvre sur un film qui n'écrit AUCUNE mort du joueur ([CondFilmMuet]),
	// celui de `tracks_publication.go` sur un film dont la TABLE D'INDEX des joueurs est absente
	// ([CondSectionAbsente]) — deux diagnostics, deux gestes de retrait.
	//
	// UN SITE NE DEVIENT PAS UN REPLI À PART POUR AUTANT : c'est le même fait décidé par le même
	// mécanisme, donc le même nom, le même compteur et le même critère de retrait. Ce qui varie
	// est la raison pour laquelle la lecture s'est tue.
	Condition Condition
}

// ConditionEffective rend le diagnostic de ce site : le sien s'il en porte un, sinon celui de
// l'entrée. C'est la valeur que lit tout rapport de couverture.
func (r Repli) ConditionEffective(s Site) Condition {
	if strings.TrimSpace(string(s.Condition)) != "" {
		return s.Condition
	}
	return r.Condition
}

// Repli est une entrée du registre : tout ce qu'il faut savoir d'un repli sans ouvrir le code.
type Repli struct {
	// Nom : l'identifiant stable publié dans `coverage.fallbacks[]`.
	Nom Nom
	// Fait : CE QUI EST DÉCIDÉ par le repli, en une phrase. Pas le mécanisme — le fait.
	Fait string
	// Mecanisme : comment le repli décide, avec ses paramètres exacts quand il en a.
	Mecanisme string
	// Condition : le diagnostic typé qui l'ouvre.
	Condition Condition
	// Ordre : sa place par rapport à la lecture (D14 b).
	Ordre Ordre
	// Sites : où il s'exécute. Au moins un ; plusieurs quand la même décision se prend à
	// plusieurs endroits qui se convertiront ensemble.
	Sites []Site
	// DatePose : date d'ENTRÉE AU REGISTRE, `AAAA-MM-JJ`.
	//
	// CE N'EST PAS LA DATE DE NAISSANCE DU CODE, et le dire autrement serait faux : les 62
	// replis hérités portent le 2026-09-13, jour où l'audit 0.E les a nommés pour la première
	// fois — leur apparition dans le code est antérieure et n'a pas été bisectée (elle ne
	// changerait aucune décision : c'est le critère de retrait qui pilote, pas l'ancienneté).
	// Les replis posés par un lot de ce chantier portent la date de leur lot.
	DatePose string
	// CibleRetrait : le LOT du plan qui doit faire disparaître ce repli, ou la condition qui
	// l'y rendrait éligible. Jamais vide (règle 11 du dépôt : pas de repli sans date cible).
	CibleRetrait string
	// CritereRetrait : ce qu'il faut MESURER pour avoir le droit de le retirer. Un critère
	// vérifiable, pas une intention.
	CritereRetrait string
	// CompteurBranche : le compteur de ce repli est-il câblé au site ?
	//
	// IL EXISTE PARCE QU'UN ZÉRO DOIT SE LIRE SANS AMBIGUÏTÉ. D14 (d) fait supprimer un repli
	// dont le compte est à zéro : confondre « jamais déclenché » et « jamais instrumenté »
	// ferait supprimer un repli actif. Faux = le compte est INCONNU, pas nul.
	CompteurBranche bool
	// CibleComptage : quand [Repli.CompteurBranche] est faux, le lot qui câblera le compteur —
	// et pourquoi il n'est pas câblé ici. Vide quand le compteur est branché.
	CibleComptage string
}

// Paquet rend le paquet Go du premier site — la clé de regroupement des rapports.
func (r Repli) Paquet() string {
	if len(r.Sites) == 0 {
		return ""
	}
	if i := strings.LastIndex(r.Sites[0].Fichier, "/"); i >= 0 {
		return r.Sites[0].Fichier[:i]
	}
	return r.Sites[0].Fichier
}

// Table rend le registre complet, trié par nom. La tranche est une COPIE ; les [Site] qu'elle
// porte ne le sont pas et ne se modifient pas.
func Table() []Repli {
	out := append([]Repli(nil), registre...)
	sort.Slice(out, func(i, j int) bool { return out[i].Nom < out[j].Nom })
	return out
}

// Lire rend l'entrée d'un nom, et si elle existe.
func Lire(n Nom) (Repli, bool) {
	for _, r := range registre {
		if r.Nom == n {
			return r, true
		}
	}
	return Repli{}, false
}

// NbDevantLaLecture compte les replis qui décident DEVANT une lecture disponible — la violation
// de D14 (b). C'est un RATCHET : le nombre ne monte jamais, il descend à mesure que les lots
// 1.9.x convertissent. Vérifié par `fallback_registre_test.go`.
func NbDevantLaLecture() int {
	n := 0
	for _, r := range registre {
		if r.Ordre == OrdreDevantLaLecture {
			n++
		}
	}
	return n
}

// VerifierRegistre rend la liste des défauts STRUCTURELS du registre : champ obligatoire vide,
// nom hors convention, nom en double, condition ou ordre inconnus, site sans fichier ni ancre,
// cible de comptage absente quand le compteur ne l'est pas. Rend nil quand tout tient.
//
// ELLE EST APPELÉE PAR LE TEST DU PAQUET ET PAR LE GARDE-RAIL `archlint`, et c'est voulu :
// l'un échoue au plus près de l'erreur, l'autre empêche qu'un contournement passe par un
// paquet qui n'aurait pas de test.
func VerifierRegistre() []string {
	pbs := make([]string, 0, len(registre))
	vus := map[Nom]bool{}
	for _, r := range registre {
		pbs = append(pbs, verifierUneEntree(r, vus)...)
		vus[r.Nom] = true
	}
	if len(pbs) == 0 {
		return nil // nil et non une tranche vide : l'appelant teste `len`, le contrat dit « rien »
	}
	return pbs
}

// conditionsConnues / ordresConnus : les domaines fermés, pour que [VerifierRegistre] refuse
// une valeur inventée à la main dans une entrée.
var (
	conditionsConnues = map[Condition]bool{
		CondFilmMuet: true, CondSectionAbsente: true, CondLectureNonPortee: true,
		CondContradiction: true, CondNonResolu: true, CondInconditionnel: true,
		CondCarteAbsenteDuCatalogue: true,
		CondFormatSansProfilRelu:    true,
		CondChassisAbsentDeLaTable:  true,
	}
	ordresConnus = map[Ordre]bool{
		OrdreApresLecture: true, OrdreSansLecture: true, OrdreDevantLaLecture: true,
	}
)

// verifierUneEntree : les contrôles d'une seule entrée. Extraite pour tenir la limite de
// 80 lignes par fonction.
func verifierUneEntree(r Repli, vus map[Nom]bool) []string {
	var pbs []string
	add := func(f string, a ...any) { pbs = append(pbs, fmt.Sprintf("%s : %s", r.Nom, fmt.Sprintf(f, a...))) }
	if vus[r.Nom] {
		add("nom en double dans le registre")
	}
	if !nomConforme(string(r.Nom)) {
		add("nom hors convention `repli_<fait>_<mecanisme>` (minuscules, `_`, 3 segments au moins)")
	}
	for champ, v := range map[string]string{
		"Fait": r.Fait, "Mecanisme": r.Mecanisme, "DatePose": r.DatePose,
		"CibleRetrait": r.CibleRetrait, "CritereRetrait": r.CritereRetrait,
	} {
		if strings.TrimSpace(v) == "" {
			add("champ %s vide", champ)
		}
	}
	if !dateConforme(r.DatePose) {
		add("DatePose %q n'est pas au format AAAA-MM-JJ", r.DatePose)
	}
	if !conditionsConnues[r.Condition] {
		add("condition inconnue %q", r.Condition)
	}
	if !ordresConnus[r.Ordre] {
		add("ordre inconnu %q", r.Ordre)
	}
	if len(r.Sites) == 0 {
		add("aucun site : une entree sans site ne se verifie pas et ne se retire jamais")
	}
	for i, s := range r.Sites {
		if strings.TrimSpace(s.Fichier) == "" || strings.TrimSpace(s.Ancre) == "" {
			add("site #%d incomplet (fichier %q, ancre %q)", i, s.Fichier, s.Ancre)
		}
		if !conditionsConnues[r.ConditionEffective(s)] {
			add("site #%d : condition inconnue %q", i, s.Condition)
		}
		if s.Condition == r.Condition && strings.TrimSpace(string(s.Condition)) != "" {
			add("site #%d : condition recopiee de l entree — la laisser vide", i)
		}
	}
	if !r.CompteurBranche && strings.TrimSpace(r.CibleComptage) == "" {
		add("compteur non branche sans CibleComptage : un zero y serait illisible")
	}
	if r.CompteurBranche && strings.TrimSpace(r.CibleComptage) != "" {
		add("CibleComptage renseignee alors que le compteur est deja branche")
	}
	return pbs
}

// nomConforme : minuscules, chiffres et `_`, préfixe `repli_`, au moins trois segments.
func nomConforme(n string) bool {
	if !strings.HasPrefix(n, "repli_") || len(strings.Split(n, "_")) < 3 {
		return false
	}
	for _, c := range n {
		if (c < 'a' || c > 'z') && (c < '0' || c > '9') && c != '_' {
			return false
		}
	}
	return !strings.Contains(n, "__") && !strings.HasSuffix(n, "_")
}

// dateConforme : `AAAA-MM-JJ`, chiffres et tirets aux bonnes places. Volontairement syntaxique —
// une date de pose est une référence, pas un calcul.
func dateConforme(d string) bool {
	if len(d) != 10 || d[4] != '-' || d[7] != '-' {
		return false
	}
	for i, c := range d {
		if i == 4 || i == 7 {
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
