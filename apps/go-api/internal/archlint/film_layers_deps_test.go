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
// Le lot 2.5 les mettra en place par deplacements purs. Mais un ratchet pose APRES le
// deplacement n aurait rien garde PENDANT le deplacement, qui est justement le moment ou l on
// casse des choses : la methode du lot E est « ratchets de dependance poses AVANT le premier
// `git mv` ». Celui-ci vaut donc DEJA sur l arborescence d aujourd hui, ou les couches sont des
// paquets aux noms d avant (`filmdec`, `killsource`, `objectiveevents`, `filmsource`), et il
// guide 2.5 : son allowlist EST la liste des coupes a faire, et elle se vide a mesure.
//
// # LA TABLE COUCHE -> PAQUETS, ET CE QUI CHANGERA EN 2.5
//
// `couchesDuDecodeur` ci-dessous nomme, pour chaque paquet d AUJOURD HUI, sa couche CIBLE.
// Quand 2.5 deplacera les paquets, SEULE CETTE TABLE changera (les chemins), pas les regles.
//
// # LES REGLES, EN TROIS LIGNES
//
//	R1 (le sens)  un paquet de couche N n importe aucun paquet de couche M > N.
//	              Jamais l inverse du sens ci-dessus, jamais un saut de deux couches vers le HAUT.
//	R2 (le lieu)  aucun paquet du decodeur (tout ce qui vit sous `film/`, plus tout paquet classe
//	              dans une couche) n importe `internal/analysis` ni `internal/analysis/*`.
//	R3 (peuplement) une couche declaree porte au moins un paquet ; une couche vide ne garde rien.
//
// Mesure a la pose (2026-09-17, `go list -f` sur les paquets surveilles) : **R1 est deja tenue —
// zero import vers le haut**. C est le point remarquable du graphe actuel : a l INTERIEUR du
// decodeur, tout descend deja. Ce qui est a l envers est le LIEU : quatre paquets de couche
// vivent encore hors du decodeur, ou en dependent (R2, 10 aretes).
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
// # LES 12 ARETES DE LA NOTE DE PREPARATION, ET CE QUE CE RATCHET EN PORTE
//
// `.ai/PREPARATION_M2_PAS_4_A_6_2026-09-17.md` §2.2 liste 12 aretes a casser. Re-mesure du
// 2026-09-17 sur `1e246b209` : ce ratchet en porte 8, en ajoute 2 que la note n avait pas
// comptees, et en laisse 4 a D9. Correspondance, pour que personne ne cherche les manquantes :
//
//	note #1  replay -> objectiveevents        -> aretesTolerees[7]
//	note #2  killsource -> filmsource         -> aretesTolerees[4]
//	note #3  filmdec -> filmsource            -> aretesTolerees[2]
//	note #4  replay -> filmsource             -> aretesTolerees[6]
//	note #5  replay -> weaponv3               -> aretesTolerees[8]
//	note #6  replay -> analysis (racine)      -> aretesTolerees[5]
//	note #7  killsource -> analysis (racine)  -> aretesTolerees[3]
//	note #8  weaponv3 -> analysis (racine)    -> aretesTolerees[10]
//	note #9  sessionusage -> replay           -> D9 (production, hors du decodeur)
//	note #10 filmsource/source_test.go        -> D9 (test)
//	note #11 objectiveevents/*_test.go (x2)   -> D9 (tests)
//	note #12 analysis/weapon_index_equiv_test -> D9 (test)
//	EN PLUS : filmcache -> filmsource         -> aretesTolerees[1]
//	EN PLUS : objectiveevents -> filmsource   -> aretesTolerees[9]
//
// Les deux « en plus » sont reelles et tombent au meme lot que les autres : la note regardait le
// SENS des aretes (celles-la sont dans le bon sens), ce ratchet regarde aussi le LIEU.
//
// # MUTATIONS QUI DOIVENT LE FAIRE ROUGIR
//
//   - classer `film/filmdec` en `replay` : `killsource` (facts) -> `filmdec` devient un import
//     vers le haut, R1 rougit ;
//   - retirer une entree d `aretesTolerees` : l arete correspondante rougit ;
//   - ajouter une entree d allowlist sans violation reelle : « entree perimee, la retirer ».

import (
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
	// --- source : charger, decompresser, decouper, lire l en-tete, tenir le lecteur de bits.
	// `filmsource` est une FEUILLE sans aucun import du depot (ratchet `filmsource_leaf_test.go`)
	// et passe sous `film/internal/source` au lot 2.5.a (decision V5 du plan).
	"internal/analysis/filmsource": coucheSource,

	// --- profile : VIDE AUJOURD HUI, et c est ecrit. La part profil existe (`filmdec/profile.go`,
	// `profile_table.go`, `build_profile.go`, `map_bounds.go`, `i0_layout.go`, `mpp_widths.go`,
	// `slot_band_dense.go`) mais elle vit DANS `filmdec` : au niveau PAQUET — la seule granularite
	// qu un graphe d imports connaisse — `filmdec` vaut donc `grammar` tant que 2.5.b n a pas
	// extrait ces sept fichiers. Voir `couchesVidesTolerees`.

	// --- grammar : decodeurs de records et de composants, fonctions pures de (profil, bits).
	"internal/games/halo_infinite/film/filmdec": coucheGrammar,
	// `weaponv3` est un QUATRIEME lecteur de bits, pose sous `analysis/` : `ResolveXuidToPI` est
	// de la grammaire de film. Le paquet ne se deplace pas en bloc, il se DISSOUT au lot 2.5.c —
	// le resolveur descend en `grammar`, le catalogue d armes (3 symboles) remonte en
	// `games/weapons`, qui est la lingua franca inter-titres.
	"internal/analysis/weaponv3": coucheGrammar,

	// --- facts : de la chronologie brute aux faits du match (vies, identite, tirs, morts,
	// objectifs, equipement, vehicules), chacun avec ses compteurs de couverture.
	"internal/games/halo_infinite/film/killsource": coucheFacts,
	"internal/analysis/objectiveevents":            coucheFacts,

	// --- replay : publie le document versionne. NE DECODE RIEN (ADR 0034 D-1).
	"internal/games/halo_infinite/film/replay": coucheReplay,
	// `fallback` et `mapvar` sont deux feuilles de `replay` (registre des replis, variantes de
	// carte), sans aucun import du depot. Classees `replay` avec leur parent : la note de
	// preparation §2.7 prevoit que `fallback` descende en `facts/fallback` au lot 2.5.d.1 — le
	// reclassement ne changera AUCUNE arete (une feuille n importe rien, et `replay -> facts`
	// reste descendant), il ne touchera que la ligne ci-dessous.
	"internal/games/halo_infinite/film/replay/fallback": coucheReplay,
	"internal/games/halo_infinite/film/replay/mapvar":   coucheReplay,

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
}

// areteToleree : une arete qui viole une regle AUJOURD HUI, avec la date de sa mise en table, le
// lot de 2.5 qui doit la casser et la FORME de la coupe. Ce n est pas un blanc-seing : c est la
// liste de travail de 2.5, et le test refuse toute entree devenue sans objet.
type areteToleree struct {
	de    string // paquet importateur, relatif a apps/go-api
	vers  string // paquet importe
	pose  string // date de mise en table
	lot   string // le lot qui casse l arete
	coupe string // la forme de la coupe
}

// aretesTolerees — LES 10 ARETES MESUREES LE 2026-09-17 sur `1e246b209`
// (`go list -f '{{.ImportPath}} {{.Imports}}'`, production seule ; aucun fichier de production de
// ces paquets ne porte de build tag, la mesure est donc exhaustive). Toutes violent R2 (le lieu) ;
// AUCUNE ne viole R1 (le sens) — a l interieur du decodeur, tout descend deja. Chaque entree
// disparait dans le commit qui fait le deplacement : une entree qui survit a sa violation fait
// rougir `TestAllowlistsDesCouchesNeSontPasPerimees`.
var aretesTolerees = []areteToleree{
	{
		de: "internal/games/halo_infinite/film/filmcache", vers: "internal/analysis/filmsource",
		pose: "2026-09-17", lot: "2.5.a",
		coupe: "le cache ouvre la source ; l import suit `filmsource` sous " +
			"`film/internal/source`, sans autre changement que la ligne d import",
	},
	{
		de: "internal/games/halo_infinite/film/filmdec", vers: "internal/analysis/filmsource",
		pose: "2026-09-17", lot: "2.5.a",
		coupe: "grammar -> source : le sens est deja bon, seul le lieu est faux ; " +
			"`git mv` de `filmsource` puis re-pointage de l import",
	},
	{
		de: "internal/games/halo_infinite/film/killsource", vers: "internal/analysis",
		pose: "2026-09-17", lot: "2.5.c",
		coupe: "4 symboles (`EventTypeDeath`, `EventTypeKill`, `HighlightEvent`, " +
			"`ParseHighlightEvents`) : `ParseHighlightEvents` est de la grammaire de film posee " +
			"dans un paquet title-agnostic ; elle descend en `grammar` avec son type",
	},
	{
		de: "internal/games/halo_infinite/film/killsource", vers: "internal/analysis/filmsource",
		pose: "2026-09-17", lot: "2.5.a",
		coupe: "facts -> source : meme coupe que l arete 2, par le `git mv` de `filmsource`",
	},
	{
		de: "internal/games/halo_infinite/film/replay", vers: "internal/analysis",
		pose: "2026-09-17", lot: "2.5.c",
		coupe: "4 usages (`ParseHighlightEvents` x3, `WeaponIDToName`, `HighlightEvent`, " +
			"`EventTypeDeath`) : ils suivent la descente de l arete 3",
	},
	{
		de: "internal/games/halo_infinite/film/replay", vers: "internal/analysis/filmsource",
		pose: "2026-09-17", lot: "2.5.a",
		coupe: "`replay` NE DECODE RIEN (ADR 0034 D-1) : les quatre sites qui chargent le film " +
			"eux-memes (`deaths_source.go`, `inventory_decode.go`, `origin.go`, " +
			"`player_index.go`) remontent en `facts`, ou `replay` recoit un film deja charge",
	},
	{
		de: "internal/games/halo_infinite/film/replay", vers: "internal/analysis/objectiveevents",
		pose: "2026-09-17", lot: "2.5.d",
		coupe: "`objectiveevents` descend en `facts` ; l arete devient replay -> facts, au bon " +
			"lieu comme au bon sens",
	},
	{
		de: "internal/games/halo_infinite/film/replay", vers: "internal/analysis/weaponv3",
		pose: "2026-09-17", lot: "2.5.c",
		coupe: "3 symboles cote `weaponv3` (`CommonWeaponSuffix`, `WeaponFusionMap`, " +
			"`WeaponIDToName`) : le resolveur descend en `grammar`, le catalogue d armes remonte " +
			"en `games/weapons`",
	},
	{
		de: "internal/analysis/objectiveevents", vers: "internal/analysis/filmsource",
		pose: "2026-09-17", lot: "2.5.a puis 2.5.d",
		coupe: "facts -> source : le sens est bon, les deux paquets descendent sous " +
			"`film/internal/` ; l entree tombe au second des deux deplacements",
	},
	{
		de: "internal/analysis/weaponv3", vers: "internal/analysis/filmsource",
		pose: "2026-09-18", lot: "2.5.a puis 2.5.c",
		coupe: "NEE DU LOT 2.4.2, ET VOULUE PAR V15 (1) : la facade `film.Source` nait dans " +
			"`internal/analysis/filmsource` precisement parce que `weaponv3` et " +
			"`objectiveevents` ne peuvent pas importer un paquet de titre sans rougir D9. " +
			"`weaponv3` y a perdu son lecteur de bits, sa copie divergente de la lecture par " +
			"mot et son marcheur de paquets. Le sens est bon (grammar -> source) ; c est le " +
			"LIEU qui ne l est pas encore, et 2.5.a le corrige par `git mv` pur",
	},
	{
		de: "internal/analysis/weaponv3", vers: "internal/analysis",
		pose: "2026-09-17", lot: "2.5.c",
		coupe: "le catalogue d armes (3 symboles) remonte en `games/weapons` ; apres quoi " +
			"`weaponv3` se dissout et l entree n a plus d objet",
	},
}

// paquetHorsLieuTolere : un paquet classe dans une couche mais qui ne vit pas encore sous
// `film/`. C est la cause RACINE de la plupart des aretes ci-dessus ; la table le dit separement
// parce que c est le deplacement du PAQUET qui les casse toutes d un coup.
type paquetHorsLieuTolere struct {
	paquet string
	pose   string
	lot    string
	coupe  string
}

// paquetsHorsLieuToleres — 3 paquets mesures le 2026-09-17.
var paquetsHorsLieuToleres = []paquetHorsLieuTolere{
	{
		paquet: "internal/analysis/filmsource", pose: "2026-09-17", lot: "2.5.a",
		coupe: "`git mv` vers `film/internal/source` (decision V5 du plan), plus la remontee des " +
			"14 fichiers de `filmdec` qui lisent `chunk_00` (note de preparation §2.3)",
	},
	{
		paquet: "internal/analysis/objectiveevents", pose: "2026-09-17", lot: "2.5.d",
		coupe: "`git mv` vers `film/internal/facts` ; les trois helpers de lecture de bits de " +
			"`film.go` appartiennent a `source` et disparaissent avant, au lot 2.4",
	},
	{
		paquet: "internal/analysis/weaponv3", pose: "2026-09-17", lot: "2.5.c",
		coupe: "dissolution, pas deplacement : `ResolveXuidToPI` descend en `grammar`, le " +
			"catalogue d armes remonte en `games/weapons`",
	},
}

// couchesVidesTolerees : une couche declaree que AUCUN paquet ne porte encore, avec le lot qui
// la peuplera. Une couche vide ne garde rien — elle ne se tolere que datee.
var couchesVidesTolerees = map[string]string{
	"profile": "2026-09-17, lot 2.5.b — la part profil vit dans `filmdec` (profile.go, " +
		"profile_table.go, build_profile.go, map_bounds.go, i0_layout.go, mpp_widths.go, " +
		"slot_band_dense.go, 1 695 L). Au niveau PAQUET, `filmdec` vaut donc `grammar` tant " +
		"que 2.5.b n a pas extrait ces sept fichiers.",
}

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
	tolerees := indexDesAretesTolerees()
	var violations []string
	for _, de := range clesTrieesFilm(graphe) {
		for _, vers := range graphe[de] {
			motifs := motifsDeViolationDeCouche(de, vers)
			if len(motifs) == 0 {
				continue
			}
			if _, ok := tolerees[cleArete(de, vers)]; ok {
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
		"descendre le symbole dans la couche qui le produit. Ajouter une entree a "+
		"`aretesTolerees` N EST PAS une reponse : cette table est datee au 2026-09-17 et "+
		"recense les coupes que le lot 2.5 doit faire, pas la dette a venir.",
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
	horsLieu := indexDesPaquetsHorsLieu()
	for _, rel := range clesTrieesFilm(couchesDuDecodeur) {
		c := couchesDuDecodeur[rel]
		if c.rang < 0 || souscheminDeFilm(rel, racineDecodeurFilm) {
			continue
		}
		if _, ok := horsLieu[rel]; ok {
			continue
		}
		t.Errorf("%s porte la couche %q mais ne vit pas sous %s : une couche du decodeur vit "+
			"DANS le decodeur (ADR 0034 D-1). La deplacer, ou dater son sursis dans "+
			"`paquetsHorsLieuToleres` avec le lot qui la deplacera.",
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
		if _, tolere := couchesVidesTolerees[c.nom]; tolere {
			continue
		}
		t.Errorf("la couche %q ne porte aucun paquet : un rang sans paquet ne garde rien. "+
			"Classer le paquet qui la porte, ou dater son absence dans `couchesVidesTolerees`.",
			c.nom)
	}
}

// TestAllowlistsDesCouchesNeSontPasPerimees : une entree qui ne decrit plus une violation reelle
// se RETIRE, dans le commit meme qui la resout. Une allowlist perimee finit par autoriser autre
// chose que ce qu elle nommait (meme regle que les autres ratchets de ce paquet).
func TestAllowlistsDesCouchesNeSontPasPerimees(t *testing.T) {
	graphe, _ := balayerPaquetsDuDecodeur(t)
	vivantes := map[string]bool{}
	for de, imports := range graphe {
		for _, vers := range imports {
			if len(motifsDeViolationDeCouche(de, vers)) > 0 {
				vivantes[cleArete(de, vers)] = true
			}
		}
	}
	for _, a := range aretesTolerees {
		if vivantes[cleArete(a.de, a.vers)] {
			continue
		}
		t.Errorf("`aretesTolerees` cite %s (pose %s, lot %s), qui n est plus une violation : "+
			"entree perimee, la retirer.", cleArete(a.de, a.vers), a.pose, a.lot)
	}
	verifierAllowlistDeLieu(t)
	verifierAllowlistDesCouchesVides(t)
}

// verifierAllowlistDeLieu : un paquet rentre a sa place sort de la table.
func verifierAllowlistDeLieu(t *testing.T) {
	t.Helper()
	for _, p := range paquetsHorsLieuToleres {
		c, classe := couchesDuDecodeur[p.paquet]
		if !classe {
			t.Errorf("`paquetsHorsLieuToleres` cite %s, qui n est plus dans aucune couche : "+
				"entree perimee, la retirer.", p.paquet)
			continue
		}
		if souscheminDeFilm(p.paquet, racineDecodeurFilm) || c.rang < 0 {
			t.Errorf("`paquetsHorsLieuToleres` cite %s, qui est rentre a sa place (lot %s) : "+
				"entree perimee, la retirer.", p.paquet, p.lot)
		}
	}
}

// verifierAllowlistDesCouchesVides : une couche peuplee sort de la table des couches vides.
func verifierAllowlistDesCouchesVides(t *testing.T) {
	t.Helper()
	for _, nom := range clesTrieesFilm(couchesVidesTolerees) {
		for _, c := range couchesDuDecodeur {
			if c.nom != nom {
				continue
			}
			t.Errorf("`couchesVidesTolerees` cite la couche %q, qui porte desormais au moins un "+
				"paquet : entree perimee, la retirer.", nom)
			break
		}
	}
}
