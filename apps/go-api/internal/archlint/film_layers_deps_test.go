package archlint

// film_layers_deps_test.go — LES CINQ COUCHES DU DECODEUR DE FILM, ET LE SENS UNIQUE DE LEURS
// DEPENDANCES (item 2.5.0 du PLAN_DECODEUR_FILM_2026-09-13 ; ADR 0034 D-1).
//
// # POURQUOI CE RATCHET EXISTE, ET POURQUOI AVANT LE PREMIER `git mv`
//
// ADR 0034 D-1 pose cinq paquets sous `internal/games/halo_infinite/film/` et UN SEUL SENS :
//
//	source -> profile -> grammar -> facts -> replay
//
// Le lot 2.5 les a mis en place par deplacements purs. Mais un ratchet pose APRES le
// deplacement n aurait rien garde PENDANT le deplacement, qui est justement le moment ou l on
// casse des choses : la methode du lot E etait « ratchets de dependance poses AVANT le premier
// `git mv` ». Celui-ci valait donc DEJA sur l arborescence d avant, ou les couches etaient des
// paquets aux noms d alors (`filmdec`, `killsource`, `objectiveevents`, `filmsource`), et il a
// guide 2.5 : son allowlist ETAIT la liste des coupes a faire, et elle s est videe a mesure.
//
// # LA TABLE COUCHE -> PAQUETS
//
// `couchesDuDecodeur` ci-dessous nomme, pour chaque paquet, sa couche. C est le seul endroit a
// changer le jour ou un paquet du decodeur nait ou demenage.
//
// # LES REGLES, EN QUATRE LIGNES
//
//	R1 (le sens)  un paquet de couche N n importe aucun paquet de couche M > N.
//	              Jamais l inverse du sens ci-dessus, jamais un saut de deux couches vers le HAUT.
//	R2 (le lieu)  aucun paquet du decodeur (tout ce qui vit sous `film/`, plus tout paquet classe
//	              dans une couche) n importe `internal/analysis` ni `internal/analysis/*`.
//	R3 (peuplement) une couche declaree porte au moins un paquet ; une couche vide ne garde rien.
//	R4 (le chargement) `replay`, la couche de PUBLICATION, ne CHARGE pas le film : elle le
//	              RECOIT. Aucun appel a un chargeur de `source` dans ses fichiers de production.
//
// LES QUATRE AXES SONT STRICTS SAUF R4. R1 n a jamais eu d exception ; la tolerance de LIEU (R2)
// a ete supprimee au lot 2.5.a avec sa derniere entree, celle de COUCHE VIDE (R3) au lot 2.5.b,
// celle d ARETE au lot 2.5.e-d. R4 nait au lot 2.5.e-d avec une allowlist DATEE de quatre
// entrees, et son critere de retrait est celui, deja ecrit, des enveloppes D2.
//
// # L HISTOIRE DE CE RATCHET, EN TROIS MESURES
//
// A la pose (2026-09-17) : R1 deja tenue, zero import vers le haut — le point remarquable du
// graphe d alors. Ce qui etait a l envers etait le LIEU : quatre paquets de couche vivaient hors
// du decodeur, ou en dependaient (R2, 10 aretes), plus 3 paquets hors lieu et 1 couche vide.
// A la cloture du lot 2.5.b (2026-09-16) : 3 aretes, 0 paquet hors lieu, 0 couche vide.
// A la cloture du lot 2.5.e (2026-09-17) : ZERO arete, et les cinq couches sont en place —
// `source`, `profile`, `grammar` et `facts` sous `film/internal/`, `replay` exportee parce
// qu elle publie le contrat public.
//
// # CE QUE CE RATCHET NE GARDE PAS, ET QUI LE GARDE
//
//   - « Personne hors de `source` ne lit `chunk[i][j]` » (ADR 0034 D-2) : c est le ratchet
//     `no_raw_film_bytes_outside_source_test.go`, pose a l item 2.4.3. Pas ici.
//   - « `internal/analysis/` n importe pas un paquet de titre » (D9) : c est
//     `no_title_package_in_analysis_test.go`, avec sa propre allowlist datee de cinq entrees.
//     Les doubler serait la copie de garde-rail que CLAUDE.md regle 6 interdit.
//   - L inaccessibilite des couches internes depuis l exterieur du decodeur : c est le
//     COMPILATEUR, une fois les paquets sous `film/internal/` (ADR 0030, « the compiler first,
//     ratchets second »). Ce ratchet ne garde que ce que le compilateur ne sait pas dire : les
//     dependances entre couches SOEURS.
//
// # POURQUOI LES FICHIERS DE PRODUCTION SEULS (PAS LES `_test.go`)
//
// Un paquet de test EXTERNE (`package x_test`) n est importe par personne : il ne ferme aucun
// cycle et ne contraint aucune couche. Il porte au contraire les preuves d EQUIVALENCE entre
// deux couches (`filmsource/source_test.go` compare les deux marcheurs de paquets) — c est
// nommement la raison pour laquelle `filmsource_leaf_test.go` exclut deja les `_test.go`.
// Interdire ces imports interdirait la preuve. La frontiere des tests, elle, est tenue la ou
// elle a un sens : cote `analysis/`, par D9, qui parse tests compris.
//
// # LES 12 ARETES DE LA NOTE DE PREPARATION, ET CE QU IL EN RESTE
//
// `.ai/PREPARATION_M2_PAS_4_A_6_2026-09-17.md` §2.2 listait 12 aretes a casser ; ce ratchet en a
// porte 8, en a ajoute 2 que la note n avait pas comptees (elle regardait le SENS, ce ratchet
// regarde aussi le LIEU), et en a laisse 4 a D9. TOUTES SONT TOMBEES, chacune dans le commit du
// deplacement qui la resolvait — les trois dernieres au lot 2.5.e-a, avec la descente de
// `ParseHighlightEvents` en `grammar` et du catalogue d armes en `games/weapons/filmshell`.
//
// # L ORDRE DES COMMITS DE 2.5, ET POURQUOI IL N ETAIT PAS CELUI DE LA NOTE
//
// MESURE DU 2026-09-16, a l entree du lot : deplacer `source` EN PREMIER (ordre §2.7 de la
// note) faisait rougir D9 sur DIX-SEPT fichiers. La cause est mecanique : `objectiveevents` et
// `weaponv3` vivaient sous `internal/analysis/` et importaient `source` ; le jour ou `source`
// descend sous `film/`, ces imports deviennent « `internal/analysis/` importe un paquet de
// titre ». La facade de 2.4 etait nee dans `internal/analysis/filmsource` precisement pour
// l eviter (V15 (1)) — la note ne l avait pas reporte sur l ordre des commits de 2.5.
//
// L ordre suivi a donc ete celui qu imposait la dependance : les paquets de couche quittent
// `internal/analysis/` AVANT la couche `source`.
//
//	2.5.d.2  objectiveevents -> film/facts/objectives   (vide analysis/ de la couche facts)
//	2.5.c    weaponv3 dissous, grammaire d analysis/ descendue, filmdec -> film/grammar
//	2.5.a    filmsource -> film/source                  (plus aucun consommateur dans analysis/)
//	2.5.d.1  killsource -> film/facts/killsource, fallback -> film/facts/fallback
//	2.5.b    la couche profile (EXTRACTION, pas `git mv` : la donnee descend, la detection reste)
//	2.5.e    facade `film/decfilm`, bascule film/<couche> -> film/internal/<couche>, ratchet STRICT
//
// # MUTATIONS QUI DOIVENT LE FAIRE ROUGIR
//
//   - faire importer `grammar` par `film/internal/profile` (par exemple en y ramenant
//     `DetectI0LayoutOf`) : `profile` (rang 1) -> `grammar` (rang 2) est un import VERS LE HAUT,
//     R1 rougit. C est la regle que le lot 2.5.b a rendue tenable, et celle qu une rechute
//     romprait en premier ;
//   - classer `film/internal/grammar` en `replay` : `killsource` (facts) -> `grammar` devient un
//     import vers le haut, R1 rougit ;
//   - faire importer `internal/analysis` par une couche : R2 rougit, et il n y a plus de table
//     ou l inscrire ;
//   - ajouter un `source.LoadDir(...)` dans un fichier de production de `replay` : R4 rougit ;
//   - retirer une entree de `chargementsToleresDansReplay` sans porter son enveloppe : la
//     violation rougit. Y ajouter une entree sans violation reelle : « entree perimee, la
//     retirer ».

import (
	"sort"
	"strings"
	"testing"
)

const (
	// prefixeModuleFilm : le prefixe du module. Un import qui commence par la est un import DU
	// DEPOT ; on le ramene a un chemin relatif a `apps/go-api` pour parler le meme langage que
	// les tables ci-dessous.
	prefixeModuleFilm = "levelup/go-api/"
	// racineDecodeurFilm : l arborescence du decodeur, relative a `apps/go-api`.
	racineDecodeurFilm = "internal/games/halo_infinite/film"
	// racineAnalysisFilm : le paquet title-agnostic dont aucune couche ne doit dependre.
	racineAnalysisFilm = "internal/analysis"
	// plancherFichiersCouches : LE PLANCHER CONTRE UN BALAYAGE MUET. 380 fichiers `.go` de
	// production mesures le 2026-09-17 dans les paquets surveilles (353 sous `film/`, 27 sous
	// `analysis/`) ; un balayage qui en rend nettement moins n a pas trouve l arborescence et ne
	// garde plus rien. Il doit ECHOUER bruyamment, pas rendre vert sur du vide.
	plancherFichiersCouches = 300
)

// coucheFilm : une couche de l ADR 0034 D-1, avec son RANG dans le sens unique. Rang -1 = le
// paquet ne porte aucune couche (catalogue, cache, feuille de libelles) : il n est alors soumis
// qu a la regle de lieu.
type coucheFilm struct {
	nom  string
	rang int
}

var (
	horsCoucheFilm = coucheFilm{nom: "hors-couche", rang: -1}
	coucheSource   = coucheFilm{nom: "source", rang: 0}
	coucheProfile  = coucheFilm{nom: "profile", rang: 1}
	coucheGrammar  = coucheFilm{nom: "grammar", rang: 2}
	coucheFacts    = coucheFilm{nom: "facts", rang: 3}
	coucheReplay   = coucheFilm{nom: "replay", rang: 4}
)

// couchesDuDecodeur — LA TABLE : pour chaque paquet d AUJOURD HUI, sa couche CIBLE. Chemins
// relatifs a `apps/go-api`, en slash. C est le seul endroit que 2.5 aura a changer.
//
// TOUT paquet portant du `.go` de production sous `film/` doit figurer ici : un paquet neuf non
// classe fait rougir `TestCouchesDuDecodeurSontPeupleesEtALeurPlace`, ce qui est le bon sens de
// la faute (classer un paquet coute une ligne, l oublier ouvrirait un trou).
var couchesDuDecodeur = map[string]coucheFilm{
	// --- LA FACADE (lot 2.5.e, 2026-09-16). Elle RE-EXPORTE la surface que les paquets hors du
	// decodeur citaient, et elle ne decode pas une ligne : elle se classe donc HORS COUCHE,
	// comme les catalogues de libelles et l outillage d empreinte. Elle importe les quatre
	// couches internes — c est son travail — et R1 ne la contraint pas (rang -1) ; R2 si, et
	// elle la tient : aucun import d `internal/analysis`.
	//
	// ELLE VIT DANS SON PROPRE REPERTOIRE, ET C EST MESURE : le brief du lot la voulait a la
	// RACINE de `film/`, donc en `package film`. Mesure du 2026-09-17 a la compilation :
	// l identifiant `film` est LE nom du film charge dans 45 fichiers consommateurs
	// (`film *source.Film`, `film.Chunk(i)`, `film.Meta()`) — un paquet du meme nom y est
	// SHADOWE, et les 30 fichiers qui ont besoin des deux ne compilent pas. Renommer la variable
	// du domaine partout serait un changement de contenu massif dans un lot de deplacements ;
	// aliaser l import dans la moitie des fichiers laisserait deux orthographes pour un meme
	// paquet. Le repertoire `decfilm/` fait coincider le nom du paquet et celui du dossier, sans
	// alias et sans collision.
	"internal/games/halo_infinite/film/decfilm": horsCoucheFilm,
	// --- LES INSTRUMENTS DE RETRO-INGENIERIE (lot 3.2 / 3.3, rentres sous le decodeur a la
	// reconciliation du lot 2.5.e, 2026-09-17). Ils vivaient dans `apps/go-api/tools/film_re/` et
	// LISAIENT la grammaire (`grammar`, `source`, `profile`) : le jour ou les couches sont passees
	// sous `film/internal/`, le compilateur les a refuses — ce qui est exactement ce que la
	// frontiere existe pour dire. Ils rentrent donc DANS le decodeur plutot que de faire ouvrir
	// une porte pour eux.
	//
	// ILS SE CLASSENT HORS COUCHE, comme la facade et les catalogues de libelles : ils ne
	// decodent rien POUR LA PRODUCTION et ne publient rien. Tous leurs fichiers portent
	// `//go:build research`, donc ils ne sont ni compiles ni executes par defaut ; le lecteur
	// d imports de ce ratchet, lui, n honore pas les tags de build et les voit — c est voulu, un
	// instrument qui remonterait vers `internal/analysis` doit rougir comme le reste.
	//
	// CE QUI EST RESTE DANS `tools/film_re/` : `doc.go` et `ecrivains_bloquants.go`, qui
	// n importent RIEN du decodeur. La coupe suit la dependance, pas le repertoire d origine.
	"internal/games/halo_infinite/film/research/grenadeids":     horsCoucheFilm,
	"internal/games/halo_infinite/film/research/cmd_grenadeids": horsCoucheFilm,
	// Lot 3.7 (2026-09-17) : l instrument de la reapparition. Il ne lit AUCUN film — il lit
	// `HaloInfinite.exe` et rejoue la chaine descripteur -> ecrivain de
	// `NOTE_3_6_METHODE_DESCRIPTEURS` quand Ghidra n est pas disponible. Il n importe donc
	// aucune couche du decodeur, et se classe hors couche comme ses voisins.
	"internal/games/halo_infinite/film/research/reapparition":     horsCoucheFilm,
	"internal/games/halo_infinite/film/research/cmd_reapparition": horsCoucheFilm,

	// --- source : charger, decompresser, decouper, lire l en-tete, tenir le lecteur de bits.
	// `source` est une FEUILLE sans aucun import du depot (ratchet `filmsource_leaf_test.go`)
	// et passe sous `film/internal/source` au lot 2.5.a (decision V5 du plan).
	"internal/games/halo_infinite/film/internal/source": coucheSource,

	// --- profile : la table de profil, les catalogues et les types de VALEUR. PEUPLEE LE
	// 2026-09-16 (lot 2.5.b) : le paquet `film/profile` porte desormais ce qui vivait dans
	// `grammar` sous les noms `profile.go`, `profile_table.go`, `build_profile.go`,
	// `map_bounds.go`, `i0_layout.go`, `mpp_widths.go`. Ce N ETAIT PAS un `git mv` : la
	// DETECTION (le balayage qui PRODUIT un decoupage i0, la resolution depuis un film, le
	// controle du calibrage) est restee en `grammar` et rend un type de `profile` — c est
	// l inversion de dependance qui a leve le blocage mesure au §4 D2 du plan.
	"internal/games/halo_infinite/film/internal/profile": coucheProfile,

	// --- grammar : decodeurs de records et de composants, fonctions pures de (profil, bits).
	"internal/games/halo_infinite/film/internal/grammar": coucheGrammar,
	// `weaponv3` etait un QUATRIEME lecteur de bits, pose sous `analysis/` : `ResolveXuidToPI` est
	// de la grammaire de film. DESCENDU LE 2026-09-16 (lot 2.5.c) sous `film/grammar/weaponv3`,
	// par `git mv` PUR — il DEVAIT quitter `internal/analysis/` avant que la couche `source` n y
	// descende (sinon D9 rougit). Sa DISSOLUTION reste a faire : le resolveur rejoint le corps de
	// `grammar`, le catalogue d armes (3 symboles, toujours dans `internal/analysis/weapon_data.go`)
	// remonte en `games/weapons` — c est la seule chose qui fermera l arete `weaponv3 -> analysis`
	// ci-dessous.
	"internal/games/halo_infinite/film/internal/grammar/weaponv3": coucheGrammar,
	// DESCENDUS LE 2026-09-16 (lot 2.5.e, decision V15 (4)) d `internal/analysis` racine : c est
	// de la grammaire de film qui vivait dans le paquet title-agnostic. `weaponscan` porte les
	// deux balayages d armes du flux de replication (Formula A, evenement de tir au marqueur
	// universel) ; `positions` decode les positions joueurs des cadres d etat complet. Les deux
	// LISENT des bits, donc ils sont de la couche `grammar` ; ce sont des SOUS-PAQUETS parce que
	// `grammar.FireEvent` et le `FireEvent` de `weaponscan` designent deux records differents.
	"internal/games/halo_infinite/film/internal/grammar/weaponscan": coucheGrammar,
	"internal/games/halo_infinite/film/internal/grammar/positions":  coucheGrammar,

	// --- facts : de la chronologie brute aux faits du match (vies, identite, tirs, morts,
	// objectifs, equipement, vehicules), chacun avec ses compteurs de couverture.
	// La RACINE de l arbre des faits ne porte qu une chose : `facts.Rev`, la revision de la
	// couche (lot 2.6.1, descendue de `sync/killcollector`). Elle se classe `facts` — c est la
	// couche qu elle date — et n importe rien : une constante n a pas de dependance.
	"internal/games/halo_infinite/film/internal/facts":            coucheFacts,
	"internal/games/halo_infinite/film/internal/facts/killsource": coucheFacts,
	// `fallback` (le REGISTRE des replis, D10 bis) etait une feuille de `replay` ; il descend en
	// `facts/fallback` au lot 2.5.d.1 (2026-09-16) et se classe DESORMAIS `facts`, avec son
	// arborescence. Le reclassement ne change aucune arete : la feuille n importe rien du depot,
	// et `replay -> facts` reste descendant.
	"internal/games/halo_infinite/film/internal/facts/fallback": coucheFacts,
	// DESCENDU LE 2026-09-16 (lot 2.5.d.2) d `internal/analysis/objectiveevents` : le paquet vit
	// desormais sous `film/facts/`, et passera sous `film/internal/facts/` au dernier commit du lot.
	"internal/games/halo_infinite/film/internal/facts/objectives": coucheFacts,

	// --- replay : publie le document versionne. NE DECODE RIEN (ADR 0034 D-1).
	"internal/games/halo_infinite/film/replay": coucheReplay,
	// `mapvar` est une feuille de `replay` (variantes de carte), sans aucun import du depot.
	"internal/games/halo_infinite/film/replay/mapvar": coucheReplay,

	// --- hors couches : ni decodage, ni publication. Sans rang, R1 ne les contraint pas ; R2 (le
	// lieu) si, parce qu ils vivent sous `film/`.
	// `damagetag`, `killicon`, `medalname` : catalogues de LIBELLES et d ASSETS (tags de degats,
	// icones de kill, noms de medailles). Ils ne lisent aucun octet de film ; la note de
	// preparation §2.2 les range en « feuilles hors couches, inchangees ».
	"internal/games/halo_infinite/film/damagetag": horsCoucheFilm,
	"internal/games/halo_infinite/film/killicon":  horsCoucheFilm,
	"internal/games/halo_infinite/film/medalname": horsCoucheFilm,
	// `filmcache` : le CACHE DISQUE des films (chemins, index JSON). C est de l infrastructure de
	// stockage local au-dessus de `source`, pas une etape de decodage — meme rangement par la
	// note §2.2, et il reste hors de `film/internal/` pour rester accessible a ses appelants.
	"internal/games/halo_infinite/film/filmcache": horsCoucheFilm,
	// `revision` : L OUTILLAGE D EMPREINTE partage par les revisions de couche (lot 2.6.0,
	// 2026-09-17). Il ne lit AUCUN octet de film — il hache des octets de SOURCE — donc il n est
	// ni une etape de decodage ni une publication : meme rangement que les trois catalogues de
	// libelles ci-dessus. C est une feuille sans aucun import du depot, et elle reste hors de
	// `film/internal/` DELIBEREMENT : `sync/killcollector` doit pouvoir l importer tant que la
	// constante de revision des faits y vit (elle descend en `facts/` au lot 2.6.1, note de
	// preparation §3.1). Le classer dans une couche serait faux dans les deux sens — il ne
	// depend d aucune couche, et les quatre couches l importeront toutes.
	"internal/games/halo_infinite/film/revision": horsCoucheFilm,
	// `types` : LES TYPES DE CONTRAT qui traversent les frontieres de couche (lot 2.6.2,
	// 2026-09-16, etendu a la grammaire le 2026-09-17). Feuille SANS AUCUN import du depot
	// (`film_types_leaf_test.go`), importable par les cinq couches — donc sans rang : lui en
	// donner un serait faux dans les deux sens (il ne depend d aucune couche, et TROIS d entre
	// elles le nomment deja : `source`, `grammar` et `facts` ; `profile` ne produit aucun type de
	// contrat). C est exactement le rangement de `revision` ci-dessus, pour la meme raison : il
	// ne lit aucun octet de film et ne publie rien, il DECLARE des formes.
	"internal/games/halo_infinite/film/types": horsCoucheFilm,
}

// LA TOLERANCE D ARETE A ETE SUPPRIMEE LE 2026-09-17 (lot 2.5.e-d), AVEC SA DERNIERE ENTREE.
// `areteToleree` / `aretesTolerees` dataient le sursis d une arete hors regle, et la table ETAIT
// la liste de travail du lot 2.5 : elle s est videe a mesure. Les trois dernieres sont tombees
// au 2.5.e-a, chacune par le deplacement qu elle annoncait :
//
//	killsource -> analysis   `ParseHighlightEvents` est descendue en `grammar`
//	                         (`highlight_events.go`) et les quatre symboles du temps fort sont
//	                         lus chez `domain/highlightevent` — le pont transitoire de 2.5.h est
//	                         supprime ;
//	replay -> analysis       memes symboles, meme coupe ;
//	weaponv3 -> analysis     le catalogue d armes a quitte `internal/analysis/weapon_data.go`
//	                         pour `games/weapons/filmshell`, feuille sans aucun import du depot
//	                         (et non `games/weapons` lui-meme, qui tire `database/sql` et
//	                         `internal/migration` — les mettre dans les dependances du decodeur
//	                         aurait ete un autre import a rebours).
//
// LE MECANISME PART AVEC ELLES, comme la tolerance de LIEU au 2.5.a et celle de COUCHE VIDE au
// 2.5.b : une table vide qu on garde « au cas ou » invite a la remplir, alors qu une arete hors
// regle re-devient une DECISION a ecrire. LES QUATRE AXES DU RATCHET SONT DESORMAIS STRICTS —
// R1 (le sens) n a jamais eu d exception, R2 (le lieu) n en a plus depuis le 2.5.a, R3 (le
// peuplement) depuis le 2.5.b, et R1/R2 n ont plus AUCUNE table. Seule R4, posee ci-dessous,
// porte encore une allowlist, et elle est datee avec son critere de retrait.

// LA TOLERANCE DE LIEU A ETE SUPPRIMEE LE 2026-09-16 (lot 2.5.a), AVEC SA DERNIERE ENTREE.
// `paquetHorsLieuTolere` / `paquetsHorsLieuToleres` dataient le sursis d un paquet de couche
// vivant hors de `film/` ; les trois qui en avaient un y sont rentres (`objectiveevents` au
// 2.5.d.2, `weaponv3` au 2.5.c, `filmsource` au 2.5.a). La regle R2 (le lieu) n a plus AUCUNE
// exception, et le mecanisme qui les portait est supprime avec elles : une table vide dont
// personne ne lit plus les champs est du code mort, et un sursis re-devient une DECISION a
// ecrire, pas une ligne a remplir.

// LA TOLERANCE DE COUCHE VIDE A ETE SUPPRIMEE LE 2026-09-16 (lot 2.5.b), AVEC SA DERNIERE
// ENTREE. `couchesVidesTolerees` datait le sursis d une couche declaree que aucun paquet ne
// portait ; la seule qui en avait un, `profile`, est peuplee par ce lot. R3 (le peuplement) n a
// donc plus AUCUNE exception, et le mecanisme qui les portait part avec elles — meme geste, meme
// raison qu au lot 2.5.a pour la tolerance de LIEU : une table vide qu on garde « au cas ou »
// invite a la remplir, alors qu une couche declaree sans paquet est une DECISION a ecrire.
// TROISIEME ET DERNIER AXE DU RATCHET A PASSER STRICT : R1 (le sens) n a jamais eu d exception,
// R2 (le lieu) n en a plus depuis le 2.5.a, R3 n en a plus depuis celui-ci. Seule
// `aretesTolerees` subsiste, et elle ne porte plus que les trois aretes de la dissolution de
// `weaponv3` et de la descente de `analysis.ParseHighlightEvents`.

// TestCouchesDuDecodeurRespectentLeSensEtLeLieu : aucune arete a rebours, aucune dependance du
// decodeur vers `internal/analysis` — hors les aretes datees ci-dessus.
func TestCouchesDuDecodeurRespectentLeSensEtLeLieu(t *testing.T) {
	graphe, fichiers := balayerPaquetsDuDecodeur(t)
	if fichiers < plancherFichiersCouches {
		t.Fatalf("balayage muet : %d fichiers .go de production vus dans %d paquets, plancher "+
			"%d. L arborescence a bouge ou un paquet a disparu de `couchesDuDecodeur` — ce "+
			"ratchet ne garde plus rien et doit echouer bruyamment.",
			fichiers, len(graphe), plancherFichiersCouches)
	}
	var violations []string
	for _, de := range clesTrieesFilm(graphe) {
		for _, vers := range graphe[de] {
			motifs := motifsDeViolationDeCouche(de, vers)
			if len(motifs) == 0 {
				continue
			}
			violations = append(violations,
				cleArete(de, vers)+"\n      "+strings.Join(motifs, "\n      "))
		}
	}
	if len(violations) == 0 {
		return
	}
	t.Errorf("le decodeur de film a %d arete(s) hors regle (ADR 0034 D-1) :\n  %s\n"+
		"Le sens est source -> profile -> grammar -> facts -> replay, jamais l inverse, et "+
		"aucune couche ne depend d `internal/analysis`.\n"+
		"QUOI FAIRE : passer la donnee en ARGUMENT depuis la couche du dessus, ou faire "+
		"descendre le symbole dans la couche qui le produit. IL N Y A PLUS DE TABLE OU "+
		"INSCRIRE UN SURSIS : l allowlist d aretes a ete supprimee au lot 2.5.e-d avec sa "+
		"derniere entree, et une arete hors regle re-devient une DECISION a ecrire.",
		len(violations), strings.Join(violations, "\n  "))
}

// TestCouchesDuDecodeurSontPeupleesEtALeurPlace : chaque paquet sous `film/` est classe, chaque
// paquet de couche vit sous `film/`, et chaque couche declaree porte au moins un paquet.
func TestCouchesDuDecodeurSontPeupleesEtALeurPlace(t *testing.T) {
	racineAPI := apiRootDepuisIci(t)
	for _, rel := range repertoiresDeProductionSousFilm(t, racineAPI) {
		if _, classe := couchesDuDecodeur[rel]; !classe {
			t.Errorf("%s porte du code de production sous %s mais n est dans aucune couche : "+
				"l ajouter a `couchesDuDecodeur` (une couche de l ADR 0034 D-1, ou "+
				"`horsCoucheFilm` s il ne decode ni ne publie rien).", rel, racineDecodeurFilm)
		}
	}
	verifierLieuDesCouches(t)
	verifierPeuplementDesCouches(t)
}

// verifierLieuDesCouches : un paquet de couche vit sous `film/`, ou bien son sursis est date.
func verifierLieuDesCouches(t *testing.T) {
	t.Helper()
	for _, rel := range clesTrieesFilm(couchesDuDecodeur) {
		c := couchesDuDecodeur[rel]
		if c.rang < 0 || souscheminDeFilm(rel, racineDecodeurFilm) {
			continue
		}
		t.Errorf("%s porte la couche %q mais ne vit pas sous %s : une couche du decodeur vit "+
			"DANS le decodeur (ADR 0034 D-1), sans exception — la tolerance datee a ete "+
			"supprimee au lot 2.5.a avec sa derniere entree. La deplacer.",
			rel, c.nom, racineDecodeurFilm)
	}
}

// verifierPeuplementDesCouches : une couche sans paquet ne garde rien (R3).
func verifierPeuplementDesCouches(t *testing.T) {
	t.Helper()
	peuplees := map[string]bool{}
	for _, c := range couchesDuDecodeur {
		peuplees[c.nom] = true
	}
	toutes := []coucheFilm{coucheSource, coucheProfile, coucheGrammar, coucheFacts, coucheReplay}
	for _, c := range toutes {
		if peuplees[c.nom] {
			continue
		}
		t.Errorf("la couche %q ne porte aucun paquet : un rang sans paquet ne garde rien. "+
			"Classer le paquet qui la porte — la tolerance datee a ete supprimee au lot 2.5.b "+
			"avec sa derniere entree, une couche declaree porte donc un paquet, sans exception.",
			c.nom)
	}
}

// ── R4 : LA COUCHE DE PUBLICATION NE CHARGE PAS LE FILM ────────────────────────────────────
//
// POURQUOI CETTE REGLE (lot 2.5.e-d, V19 (3)). `replay` PUBLIE le document versionne : il recoit
// des faits et les met en forme. Un `source.LoadDir` dans `replay` est donc un SAUT de trois
// couches vers le bas, et surtout le retour du defaut que le lot 1 de PLAN_CUISSON_PERF a
// ferme — le film relu et redecompresse par chaque etage pour son propre compte (~94 % du temps
// de cuisson avant correction). Le film se charge UNE fois, en amont, et circule par valeur.
//
// CE QUE LA REGLE N EST PAS. Elle n interdit pas a `replay` de NOMMER `*source.Film` : recevoir
// un film charge est exactement ce qu on veut. Quatre de ses huit fichiers de production qui
// importent `source` ne font que cela, et ils sont en regle.
//
// POURQUOI ELLE N EST PAS PLUS LARGE. La formulation d origine de V19 (3) — « une couche
// n importe que la couche IMMEDIATEMENT inferieure » — n est PAS takeable en l etat : mesure du
// 2026-09-17, `replay` importe `grammar` dans 59 fichiers de production et `profile` dans
// plusieurs autres, `facts` importe `profile`. La couper demanderait de faire transiter par
// `facts` tout ce que `replay` lit de la grammaire : un chantier de contenu, pas un
// deplacement. Consigne au §4 du plan.
var chargeursDeSource = map[string]bool{
	"LoadDir": true, "Load": true, "LoadIntoDir": true, "DirSource": true, "MemoryChunks": true,
}

// coucheQuiNeChargePas : le paquet soumis a R4, relatif a `apps/go-api`.
//
// `grammar` n y est PAS, et c est mesure : il porte une quarantaine d enveloppes D2
// (`ScanFilmXxx(dir)`) qui chargent le film pour les instruments et les tests. Elles sont
// gardees ailleurs, par la regle 3 de `no_film_reread_test.go`, qui interdit a la PRODUCTION de
// les appeler. Les doubler ici serait la copie de garde-rail que CLAUDE.md regle 6 interdit.
const coucheQuiNeChargePas = "internal/games/halo_infinite/film/replay"

// chargementToleré : une enveloppe D2 de `replay` qui charge encore le film elle-meme.
type chargementTolere struct {
	fichier   string // chemin relatif a `apps/go-api`
	enveloppe string // la fonction qui charge
	pose      string // date de mise en table
	retrait   string // le critere mesurable de retrait
}

// chargementsToleresDansReplay — LES QUATRE ENVELOPPES D2 MESUREES LE 2026-09-17.
//
// Toutes les quatre sont des `ScanFilmXxx(dir)` declarees HORS PRODUCTION dans leur propre
// godoc : la cuisson appelle leur jumelle qui prend un `*source.Film` deja charge. Les
// supprimer est un changement de CONTENU (elles ont une soixantaine d appelants — tests de
// recherche et `cmd/diag_deaths`), hors d un lot de deplacements : elles portent donc le
// critere de retrait DEJA ECRIT par `no_film_reread_test.go` pour toute la famille D2.
var chargementsToleresDansReplay = []chargementTolere{
	{
		fichier:   "internal/games/halo_infinite/film/replay/deaths_source.go",
		enveloppe: "ScanFilmDeaths", pose: "2026-09-17",
		retrait: "avec la famille D2 : quand `grep -r 'ScanFilm[A-Za-z]*(' --include=*.go` ne " +
			"rend plus que leurs definitions (critere de `no_film_reread_test.go`, lot 6)",
	},
	{
		fichier:   "internal/games/halo_infinite/film/replay/inventory_decode.go",
		enveloppe: "ScanFilmKeyframeInventory", pose: "2026-09-17",
		retrait: "idem",
	},
	{
		fichier:   "internal/games/halo_infinite/film/replay/origin.go",
		enveloppe: "ScanFilmClockOrigin", pose: "2026-09-17",
		retrait: "idem",
	},
	{
		fichier:   "internal/games/halo_infinite/film/replay/player_index.go",
		enveloppe: "ScanFilmPlayerIndices", pose: "2026-09-17",
		retrait: "idem",
	},
}

// TestCoucheDePublicationNeChargePasLeFilm : R4.
func TestCoucheDePublicationNeChargePasLeFilm(t *testing.T) {
	sites := chargementsDeSourceDansReplay(t)
	tolere := map[string]bool{}
	for _, c := range chargementsToleresDansReplay {
		tolere[c.fichier] = true
	}
	var violations []string
	for _, s := range sites {
		if tolere[s.fichier] {
			continue
		}
		violations = append(violations, s.fichier+" : "+s.detail)
	}
	if len(violations) == 0 {
		return
	}
	sort.Strings(violations)
	t.Errorf("la couche de PUBLICATION charge le film elle-meme (%d site(s)) :\n  %s\n"+
		"`replay` RECOIT un `*source.Film` ; il ne l ouvre pas. Le film se charge UNE fois par "+
		"cuisson (lot 1 de PLAN_CUISSON_PERF : ~94 %% du temps de cuisson avant correction), en "+
		"amont, et circule par valeur.",
		len(violations), strings.Join(violations, "\n  "))
}

// TestAllowlistDeChargementNEstPasPerimee : une entree qui ne decrit plus un chargement reel se
// RETIRE, dans le commit meme qui la resout.
func TestAllowlistDeChargementNEstPasPerimee(t *testing.T) {
	vivants := map[string]bool{}
	for _, s := range chargementsDeSourceDansReplay(t) {
		vivants[s.fichier] = true
	}
	for _, c := range chargementsToleresDansReplay {
		if strings.TrimSpace(c.retrait) == "" {
			t.Errorf("`chargementsToleresDansReplay` cite %s sans critere de retrait : une "+
				"tolerance sans cible est une dette anonyme.", c.fichier)
		}
		if !vivants[c.fichier] {
			t.Errorf("`chargementsToleresDansReplay` cite %s (%s, pose %s), qui ne charge plus "+
				"le film : entree perimee, la retirer.", c.fichier, c.enveloppe, c.pose)
		}
	}
}
