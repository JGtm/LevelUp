package archlint

// filmdec_package_vars_test.go — L'ETAT GLOBAL DE `grammar` NE CROIT PLUS (item 1.10, lot 1 de
// PLAN_CUISSON_PERF).
//
// # POURQUOI CE RATCHET EXISTE
//
// `grammar` PORTAIT son etat de reglage dans des VARIABLES DE PAQUET (largeurs d'axe, crochets
// de deserialisation, compteurs d'observation). C'est ce qui obligeait tout le decodage a passer
// sous un verrou de paquet : deux films decodes en parallele dans le meme processus se seraient
// vole leurs largeurs. Ce ratchet a d'abord GELE le compte (2026-09-03, decision D10 de
// PLAN_CUISSON_PERF : ne pas aggraver tant que la de-globalisation n'etait pas au programme),
// puis il l'a fait DESCENDRE, lot par lot, jusqu'a ZERO VARIABLE ECRITE au lot 2.3.
//
// LE COMPTE QUI FAIT FOI EST DESORMAIS CELUI DES VARIABLES ECRITES, et il vaut ZERO — c'est le
// critere que `profil_herite.go` et `observateur.go` avaient ecrit, et ce que
// `TestAucunVarDePaquetEcriteDansFilmdec` mesure plus bas. Le compte TOTAL reste gele en second
// garde-fou : une table de grammaire de plus doit rester un geste conscient.
//
// # CE QUI EST COMPTE, EXACTEMENT
//
// Les NOMS declares par un `var` de NIVEAU PAQUET dans les fichiers non-test de
// `internal/games/halo_infinite/film/grammar` — un bloc `var ( a = 1; b = 2 )` compte donc pour DEUX, parce que
// c'est deux morceaux d'etat, pas une ligne de syntaxe. L'identifiant blanc (`var _ = ...`,
// assertion de compilation) n'est PAS compte : il ne porte aucun etat. Le comptage se fait par
// `go/ast` et non par grep — un `var` dans un commentaire ou dans un corps de fonction ne doit
// pas entrer dans la mesure.
//
// MESURE DU 2026-09-03, a l'etat final du lot 1 : 113 noms, repartis sur 98 declarations `var`
// (109 specs). L'audit source parlait de « >= 80 vars » : il comptait autrement (declarations,
// et sur un perimetre anterieur) — la seule grandeur qui fait foi ici est celle que ce test
// mesure lui-meme.
//
// RE-MESURE DU 2026-09-03, APRES LA RECONCILIATION AVEC `feat/v75` : 116. Les TROIS de plus
// viennent de l'AMONT, pas du chantier — deux fichiers neufs du chantier « precision par arme »
// (remise le 2026-09-01, acquis backend conserves) : `weapon_hits.go`
// (`WeaponHitDistanceEdges`, les bornes de l'histogramme de distance) et `weapon_hits_decode.go`
// (`lot1RefDomWidths`, `lot1chBases`, deux tables de grammaire mesurees). Le ratchet monte a 116
// parce qu'il gele CE QUE LE CHANTIER TROUVE, et qu'il a trouve une base qui a bouge : il
// continue d'interdire au chantier d'en ajouter. Ces trois-la sont des TABLES CONSTANTES
// deguisees en `var` (Go n'a pas de `const` composite) — leur retrait naturel est un `const`
// scalaire ou une fonction, pas une de-globalisation.
//
// # COMMENT LE FAIRE BOUGER
//
//   - VERS LE HAUT : interdit tant que ce chantier dure. Un nouveau reglage de decodage se passe
//     en parametre (`ScanFilmOptions`, `FilmContext` au lot 2), pas en variable de paquet.
//   - VERS LE BAS : bienvenu. Le test NE FAIT PAS ECHOUER une baisse — il l'annonce avec le
//     nouveau compte a inscrire dans `filmdecVarsGeles`, pour que le resserrage soit un geste
//     CONSCIENT et date, jamais un effet de bord invisible.
//
// RE-MESURE DU 2026-09-05, A L'ARRIVEE DU CHANTIER VEHICULES : 118. Les DEUX de plus viennent
// de la branche `feat/v75-vehicules-sons`, et chacune est justifiee :
//
//   - `unit_ref_probe.go` : `unitRefHook`, la SONDE des references d unite (`nil` en production,
//     comme `equipmentCreationHook`, `mppHook`, `recordMaskHook` et les autres sondes deja
//     comptees). C est le patron etabli du paquet pour observer une traversee sans la modifier ;
//     de-globaliser les sondes est un chantier a part, et il les concerne TOUTES.
//   - `default_state_ti40.go` : `vehicleMediaFrameBits`, la largeur MESUREE de la feuille
//     config-dependante du default-state de `ti=40` (le quaternion du vehicule). C est une
//     TABLE DE GRAMMAIRE deguisee en `var`, du meme genre que `lot1RefDomWidths` : elle ne porte
//     aucun etat de balayage.
//
// Le ratchet ne monte QUE de ces deux-la : l integration n a ajoute aucune variable de son fait
// (la seule erreur sentinelle qu elle a failli poser a ete rendue locale a son site).
//
// RESSERRAGE DU 2026-09-05 (lot E, item E.2 du PLAN_V2_REJEU_FILM) : 118 -> 113. CINQ
// variables de paquet ont ete SUPPRIMEES, et aucune n etait un reglage vivant :
//
//   - `dynPrecHook` (components_movement.go) et `repTraceHook` (default_state.go) : deux
//     crochets de capture PROUVABLEMENT toujours nil — aucun site du depot ne les installait
//     non-nil, tests compris — que huit blocs de sauvegarde/restauration promenaient.
//   - `useLegacyAngularVel` et `useBipedDefaultStateDeser` (traverse.go) : deux bascules A/B
//     sans date ni critere (regle 11), dont le setter n avait aucun appelant : la branche
//     opposee au defaut etait donc inatteignable dans les deux cas.
//   - `defaultStateBitsByTI` (traverse.go) : table de surcharge peuplee par le seul
//     `SetDefaultStateBitsForTI`, sans appelant — vide a jamais, deux branches mortes.
//
// Les 22 reglages `Set*` sans appelant ont disparu dans le meme lot ; les 17 variables qu ils
// ecrivaient RESTENT, avec leur valeur de production, parce qu elles sont lues par le decodage
// et que leur retrait serait une de-globalisation (D10), pas un retrait de code mort.
//
// SECOND RESSERRAGE DU 2026-09-05 (meme lot, item E.3) : 113 -> 111. Les DEUX copies de la table
// des largeurs de reference par domaine — `lot1RefDomWidths` (weapon_hits_decode.go) et
// `zoomRefWidth` (zoom_events.go) — sont remplacees par la fonction `refDomWidth`
// (event_list.go), qui ne porte AUCUN etat : une table de grammaire deguisee en `var` redevient
// ce qu elle est, du code. Garde-rail : `filmdec/event_preamble_guard_test.go`.
//
// TROISIEME RESSERRAGE DU 2026-09-06 (lot E, item E.8) : 111 -> 96. QUINZE noms de moins, et
// aucun n avait d ecrivain : ils gardaient tous leur valeur initiale depuis que les 22 reglages
// publics morts sont partis (E.2). Trois traitements, selon ce que la variable PORTAIT :
//
//   - DIX sont devenues des CONSTANTES NOMMEES, avec leur provenance ecrite au-dessus
//     (`absDequantMode`, `bipedActionLoop2Count`, `bipedDefaultStateDecodeMovement`,
//     `bipedDefaultStateTailBits`, `bipedMediaFramePresent`, `deadStatePreSkip`,
//     `deadStateVelocityPresent`, `inferRepair`, `inferRequireBoundSuccessor`,
//     `vehicleMediaFrameBits`). Plusieurs portent une LARGEUR MESUREE d un chemin non nominal —
//     un savoir de retro-ingenierie ne se jette pas, il se fige et se date.
//   - QUATRE ont ete SUPPRIMEES : l instrumentation i63 (`biDebug`, `biCurSeq`, `BiBadSeqs`,
//     `BiOkSeqs`), annotee « a retirer apres » par son auteur, sans activateur depuis E.2 et sans
//     aucun lecteur de ses deux tranches exportees. Elle ne portait aucune valeur mesuree.
//   - UNE a ete SUPPRIMEE en gardant son modele : `absPerIndexAxisW`, table nil dont le seul
//     installateur avait disparu. Le desassemblage qu elle documentait (les deux tables du moteur,
//     l immediat LEVEL) est deplace sur `absAxisWFor`, la ou un futur portage viendra le lire.
//
// DEUX VARIABLES SANS ECRIVAIN RESTENT, et c est deliberé : `accumWorld` (avec `accumSlot`) et
// `inferResyncTargets`. Elles ne sont ni des largeurs ni des valeurs — ce sont les INTERRUPTEURS
// de deux mecanismes entiers (l accumulation de position par World, la recuperation par resync
// valide). Les retirer supprimerait ces mecanismes et leur cloture — `setAccumSlot` et ses sites
// d appel dans les decodeurs, `validatedResync`, `scanForTargetDelta` et l unique installateur de
// production de `posCaptureHook`. C est une suppression de FONCTIONNALITE, pas un pliage de
// constante : elle se decide, elle ne se glisse pas dans un lot a comportement identique.
//
// CE JOUR EST ARRIVE AU LOT 2.3 : le verrou de paquet n'a plus de raison d'etre, et il a ete
// retire. Le ratchet, lui, RESTE — il interdit desormais la resurrection, avec son ratchet
// frere `decode_lock_interdit_test.go`.

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// filmdecVarsGeles : le compte GELE des variables de paquet de `grammar` (cf. l en-tete pour la
// convention de comptage et la date de mesure).
//
// RESSERRE A 94 LE 2026-09-16 (revue de jalon M1, constat C4) : `keyframeBodyVariants` et la
// table de variantes qui l accompagne quittent la production avec `walkKeyframeBody` — zero
// appelant de production depuis le lot 1.4 — pour un fichier `_test.go` du paquet. Le ratchet ne
// DESCEND que, et son en-tete le demande explicitement des qu une baisse est mesuree.
//
// RESSERRE A 90 LE 2026-09-17 (lot 2.2.a du PLAN_DECODEUR_FILM, famille « positions ») : les CINQ
// valeurs du chemin de position — `TraversalPrecision`, `absoluteAxisW`, `PositionFullPrecision`,
// `PositionDeltaHasHandleTail`, `PositionCalibratedSkip` — ne sont plus des variables de paquet :
// elles voyagent avec le LECTEUR DE BITS (`BitReader.mv`), semees par le profil
// (`filmdec.mouvementDuProfil`) et posees EN TETE de balayage par les portes a `FrameConfig`.
// UNE variable NEUVE les remplace, et elle est nommee : `mouvementHerite`
// (`filmdec/mouvement_herite.go`), le profil qu une passe laisse a la suivante dans le meme
// processus — kill-switch date, retrait cible lot 2.3, critere « `replay.BuildFromFilm` recoit
// le profil de son appelant ». Bilan net : -5 +1 = 90.
//
// RESSERRE A 86 LE 2026-09-17 (lot 2.2.b, famille « objets du monde ») : `WorldObjectPrecision`,
// `WorldPositionRange`, `DeltaQuantum` et `DeltaAxisWidth` rejoignent le PROFIL que le lecteur
// porte. Trois etaient sans ecrivain depuis le lot E ; la quatrieme —
// `WorldObjectPrecision` — est posee par carte, et son installateur
// (`replay/world_object_precision.go`) ecrit desormais le profil HERITE au lieu d une globale
// propre : aucune variable neuve, le canal existait deja. Bilan net : -4 = 86.
//
// RESSERRE A 79 LE 2026-09-17 (lot 2.2.e, famille « equipement et mobilite ») : SEPT de moins.
// Les deux largeurs du bloc `object-multiplayer-properties` (`mppLeadBits`, `mppIndexBits`), le
// `param_4` force par un harnais et son drapeau (`recordStateParam`,
// `recordStateParamOverride`) et les bits supplementaires d une action de mobilite
// (`MobilityActionExtraBits`) rejoignent le PROFIL que le lecteur porte ; les deux listes de
// candidats de la calibration MPP (`mppLeadCandidates`, `mppIndexCandidates`) redeviennent des
// FONCTIONS — c etaient des tables de grammaire deguisees en `var`, que rien n ecrivait (meme
// geste qu au lot E.3 du 2026-09-05). Aucune variable neuve : l heritage du lot 2.2.a les
// accueille toutes, et c est desormais UNE structure (`filmdec.profilHerite`) au lieu d une
// valeur. Bilan net : -7 = 79.
//
// RESSERRE A 43 LE 2026-09-17 (lot 2.2.f, famille « crochets d observation ») : TRENTE-SIX de
// moins, la plus grosse baisse de la serie. Les VINGT-NEUF crochets de deserialiseur et les
// HUIT compteurs de l inference de chaine — dont `compWidthObs`, « la table sans verrou » que
// l en-tete du verrou de decodage nommait comme l une des deux raisons de son existence —
// deviennent les CHAMPS d un seul objet, `filmdec.Observation`. Un observateur ne change aucune
// consommation de bits : c est la propriete qui le distingue du profil, et elle est ecrite en
// tete de `observateur.go`. Bilan net : -36 = 43.
// RESSERRE A 42 LE 2026-09-17 (lot 2.3, famille « le profil de balayage remplace l heritage ») :
// `herite` — la DERNIERE variable que les lots 2.2.a/b/e avaient regroupee, « ce qu une passe
// laisse a la suivante dans le processus » — disparait. Ce qu elle portait (descripteur de
// traversee, largeur d axe absolue, largeurs d axe des objets du monde, decoupage MPP,
// `param_4` force) devient [grammar.ProfilDeBalayage], une VALEUR : le lecteur de bits en tient
// une copie, le contexte du film celle du decodage courant, et `FrameConfig.Profil` la passe aux
// portes de balayage. La calibration de `killsource` la REND desormais
// (`Result.ProfilCalibre`), et `replaybuild` la passe explicitement a `replay.BuildFromFilm` —
// l heritage par l etat du processus, nomme par la decouverte D1 du lot 2.2.a, n existe plus.
// Bilan net : -1 = 42.
//
// RESSERRE A 30 LE 2026-09-17 (lot 2.3, famille « les bascules de grammaire ») : DOUZE de
// moins, avec leurs douze reglages publics. Les A/B de retro-ingenierie — controle de
// corruption, queue de record NEW, deserialiseur d etat par archetype, `simulation-state`
// complet, portee baseline, grammaire d ecrivain d i0, corps d action de mobilite, corps
// d ancrage de capacite, inference de chaine, generation stricte, et les DEUX tables de
// largeurs (calibrees, bouchon) — deviennent [grammar.GrammaireBalayage], un champ du profil
// que le lecteur de bits porte. Un instrument qui en pose une la pose pour SON balayage.
// Bilan net : -12 = 30.
//
// RESSERRE A 23 LE 2026-09-17 (lot 2.3, famille « la capture de position ») : SEPT de moins.
// SIX decrivaient UN record en cours de decodage (`posCaptureStartBit`, `posCaptureSlot`,
// `accumWorld`, `accumSlot`, `absViaFallback`, et la portee de `setAccumSlot`) : elles
// deviennent [grammar.captureDePosition], un champ du LECTEUR de bits — deux balayages
// simultanes n ont rien a partager la-dedans. La septieme, `absIdxHist`, est un COMPTEUR
// D OBSERVATION : elle rejoint [grammar.Observation]. `lastRepVersion` et son accesseur
// exporte `LastRepVersion()` sont SUPPRIMES — aucun appelant dans le depot, tests compris
// (regle 7). Bilan net : -7 = 23, dont UNE SEULE encore ecrite : `observateur`.
//
// RESSERRE A 22 LE 2026-09-17 (lot 2.3, famille « l observateur ») — ET LE COMPTE QUI FAIT FOI
// EST L AUTRE : **ZERO variable de paquet ECRITE**. `observateur` etait la derniere ; les
// vingt-huit reglages publics (`SetXxxHook`) qui l ecrivaient ont disparu avec elle. Chaque
// balayage construit desormais SON observateur ([grammar.NouvelleObservation]) et le pose sur
// ses lecteurs avec son profil ([grammar.ContexteDeLecture]).
//
// RESSERRE A 21 LE 2026-09-17 (revue adversariale du lot 2.3, constat P1-1). Le lot avait
// laisse 22 apres la famille « observateur » et n a pas re-mesure apres le retrait du verrou :
// le compte REEL est 21, et l ecart etait une PLACE LIBRE — une table de grammaire neuve serait
// entree sans rougir, ce qui est exactement ce que ce ratchet existe pour empecher. L en-tete le
// prescrivait (« un nouveau compte a inscrire dans `filmdecVarsGeles`, pour que le resserrage
// soit un geste ») ; c est fait ici.
//
// LES VINGT ET UNE QUI RESTENT NE SONT ECRITES PAR PERSONNE : quatre erreurs sentinelles (Go n a
// pas de `const` d erreur), seize tables de grammaire deguisees en `var` (Go n a pas de `const`
// composite) et le dedoublonneur d avertissement de registre (`registryWarned`, un `sync.Map`
// qu aucun decodage ne lit). C est ce que le ratchet mesure desormais, et c est le critere que
// `profil_herite.go` et `observateur.go` avaient ecrit : « `filmdecVarsGeles` tombe a 0 variable
// mutable ».
const filmdecVarsGeles = 21

// TestFilmdecPackageVarsNeCroitPas — LE RATCHET.
func TestFilmdecPackageVarsNeCroitPas(t *testing.T) {
	pkgDir := filepath.Join(apiRootDepuisIci(t), filepath.FromSlash("internal/games/halo_infinite/film/grammar"))
	compte, parFichier := compterVarsDePaquet(t, pkgDir)
	switch {
	case compte > filmdecVarsGeles:
		t.Fatalf("l'etat global de `grammar` a CRU : %d variables de paquet, gelees a %d "+
			"(D10 de PLAN_CUISSON_PERF, mesure du 2026-09-03).\n%s\n"+
			"Un nouveau reglage de decodage se passe en PARAMETRE (`filmdec.ContexteDeLecture`, "+
			"`ScanFilmOptions`, `FilmContext`), pas en variable de paquet : deux films se decodent "+
			"en parallele depuis le lot 2.3, et une variable de paquet ecrite les remettrait a la "+
			"queue leu leu.",
			compte, filmdecVarsGeles, detailParFichier(parFichier))
	case compte < filmdecVarsGeles:
		t.Logf("l'etat global de `grammar` a BAISSE : %d variables de paquet au lieu de %d — "+
			"resserrer le ratchet en mettant `filmdecVarsGeles` a %d (avec la date de la mesure).",
			compte, filmdecVarsGeles, compte)
	}
}

// compterVarsDePaquet rend le nombre de NOMS declares par un `var` de niveau paquet, et leur
// repartition par fichier.
func compterVarsDePaquet(t *testing.T, pkgDir string) (int, map[string]int) {
	t.Helper()
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		t.Fatalf("paquet %s introuvable (%v) : s'il a DEMENAGE, deplacer ce ratchet avec lui", pkgDir, err)
	}
	fset := token.NewFileSet()
	total := 0
	parFichier := map[string]int{}
	for _, e := range entries {
		nom := e.Name()
		if e.IsDir() || !strings.HasSuffix(nom, ".go") || strings.HasSuffix(nom, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(pkgDir, nom), nil, 0)
		if err != nil {
			t.Fatalf("analyse de %s : %v", nom, err)
		}
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue // `const`, `type`, `import`, et les fonctions : hors mesure
			}
			for _, sp := range gd.Specs {
				vs, ok := sp.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, id := range vs.Names {
					if id.Name == "_" {
						continue // assertion de compilation : aucun etat
					}
					total++
					parFichier[nom]++
				}
			}
		}
	}
	if total == 0 {
		t.Fatalf("aucune variable de paquet trouvee dans %s : le ratchet ne mesure plus rien "+
			"(paquet vide, ou fichiers non parses)", pkgDir)
	}
	return total, parFichier
}

// detailParFichier liste les fichiers porteurs, pour que le message d'echec designe OU l'etat a
// grossi au lieu de rendre un simple total.
func detailParFichier(parFichier map[string]int) string {
	noms := make([]string, 0, len(parFichier))
	for nom := range parFichier {
		noms = append(noms, nom)
	}
	sort.Strings(noms)
	var b strings.Builder
	b.WriteString("  variables de paquet par fichier :\n")
	for _, nom := range noms {
		fmt.Fprintf(&b, "    %s : %d\n", nom, parFichier[nom])
	}
	return b.String()
}

// TestAucunVarDePaquetEcriteDansFilmdec — LE RATCHET QUI COMPTE, depuis le lot 2.3.
//
// # CE QU IL MESURE, EXACTEMENT
//
// Une variable de paquet est dite ECRITE si un fichier du paquet (TESTS COMPRIS) lui affecte une
// valeur ailleurs qu a sa declaration : affectation simple ou multiple, increment, ecriture par
// index, ou prise d adresse. Le compte doit valoir ZERO.
//
// POURQUOI « ECRITE » ET NON « DECLAREE ». Ce qui obligeait le decodage a se serialiser n etait
// pas l existence d une variable de paquet, c etait son ECRITURE pendant un balayage. Une table
// de grammaire que personne n ecrit est une constante que Go ne sait pas exprimer ; une erreur
// sentinelle aussi. Les distinguer est ce qui rend le critere vrai plutot que decoratif.
//
// LES TESTS SONT DANS LE PERIMETRE, et c est delibere : un harnais qui ecrirait une variable de
// PRODUCTION la rendrait partagee de fait. Les harnais de ce paquet portent leur propre etat
// dans des fichiers `_test.go` (`harnais_profil_test.go`, `harnais_observation_test.go`), que le
// comptage des DECLARATIONS ignore — mais dont les ecritures ne visent que ces memes fichiers.
//
// ANGLE MORT ASSUME : une table passee en ARGUMENT a une fonction qui la mute ne serait pas vue.
// Aucune des vingt-deux restantes n est dans ce cas (elles sont lues par indexation ou par
// `range`), et le compte TOTAL gele plus haut interdit d en ajouter sans le dire.
func TestAucunVarDePaquetEcriteDansFilmdec(t *testing.T) {
	pkgDir := filepath.Join(apiRootDepuisIci(t), filepath.FromSlash("internal/games/halo_infinite/film/grammar"))
	ecrites := varsDePaquetEcrites(t, pkgDir)
	if len(ecrites) == 0 {
		return
	}
	sort.Strings(ecrites)
	t.Fatalf("UNE VARIABLE DE PAQUET DE `grammar` EST ECRITE (%d) :\n  %s\n\n"+
		"Le decodeur n'en a plus AUCUNE depuis le lot 2.3 : c'est ce qui permet a deux films de\n"+
		"se decoder en parallele, et ce qui a permis de retirer le verrou de paquet.\n"+
		"Ce qu'il faut a la place : porter la valeur dans `filmdec.ProfilDeBalayage` (si elle\n"+
		"DECIDE d'une largeur) ou dans `filmdec.Observation` (si elle ne fait que RECEVOIR), et\n"+
		"la passer par `filmdec.ContexteDeLecture`.",
		len(ecrites), strings.Join(ecrites, "\n  "))
}

// varsDePaquetEcrites rend les noms des variables de paquet auxquelles un fichier du paquet
// affecte une valeur hors de leur declaration.
func varsDePaquetEcrites(t *testing.T, pkgDir string) []string {
	t.Helper()
	entries, err := os.ReadDir(pkgDir)
	if err != nil {
		t.Fatalf("paquet %s introuvable : %v", pkgDir, err)
	}
	fset := token.NewFileSet()
	fichiers := map[string]*ast.File{}
	declarees := map[string]bool{}
	for _, e := range entries {
		nom := e.Name()
		if e.IsDir() || !strings.HasSuffix(nom, ".go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(pkgDir, nom), nil, 0)
		if err != nil {
			t.Fatalf("analyse de %s : %v", nom, err)
		}
		fichiers[nom] = f
		if strings.HasSuffix(nom, "_test.go") {
			continue // les harnais declarent leur propre etat, hors mesure
		}
		for _, d := range f.Decls {
			gd, ok := d.(*ast.GenDecl)
			if !ok || gd.Tok != token.VAR {
				continue
			}
			for _, sp := range gd.Specs {
				vs, ok := sp.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, id := range vs.Names {
					if id.Name != "_" {
						declarees[id.Name] = true
					}
				}
			}
		}
	}
	return ecrituresVers(fset, fichiers, declarees)
}

// ecrituresVers balaie les fichiers et rend, pour chaque ecriture vers une variable declaree, une
// ligne « nom (fichier:ligne, nature) ».
func ecrituresVers(fset *token.FileSet, fichiers map[string]*ast.File, declarees map[string]bool) []string {
	vues := map[string]bool{}
	var out []string
	for nom, f := range fichiers {
		noter := func(x ast.Expr, nature string, pos token.Pos) {
			id := racineDIdentifiant(x)
			if id == nil || !declarees[id.Name] {
				return
			}
			cle := fmt.Sprintf("%s (%s:%d, %s)", id.Name, nom, fset.Position(pos).Line, nature)
			if !vues[cle] {
				vues[cle] = true
				out = append(out, cle)
			}
		}
		ast.Inspect(f, func(n ast.Node) bool {
			switch s := n.(type) {
			case *ast.GenDecl:
				if s.Tok == token.VAR {
					return false // la declaration elle-meme n'est pas une ecriture
				}
			case *ast.AssignStmt:
				if s.Tok == token.DEFINE {
					return true
				}
				for _, lhs := range s.Lhs {
					noter(lhs, "affectation", s.Pos())
				}
			case *ast.IncDecStmt:
				noter(s.X, "increment", s.Pos())
			case *ast.UnaryExpr:
				if s.Op == token.AND {
					noter(s.X, "prise d'adresse", s.Pos())
				}
			}
			return true
		})
	}
	return out
}

// racineDIdentifiant rend l'identifiant a la racine d'une expression d'affectation, ou nil.
func racineDIdentifiant(x ast.Expr) *ast.Ident {
	for {
		switch e := x.(type) {
		case *ast.Ident:
			return e
		case *ast.IndexExpr:
			x = e.X
		case *ast.SelectorExpr:
			x = e.X
		case *ast.StarExpr:
			x = e.X
		case *ast.ParenExpr:
			x = e.X
		case *ast.SliceExpr:
			x = e.X
		default:
			return nil
		}
	}
}
