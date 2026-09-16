package archlint

// filmdec_package_vars_test.go — L'ETAT GLOBAL DE `filmdec` NE CROIT PLUS (item 1.10, lot 1 de
// PLAN_CUISSON_PERF).
//
// # POURQUOI CE RATCHET EXISTE
//
// `filmdec` porte son etat de reglage dans des VARIABLES DE PAQUET (largeurs d'axe, crochets de
// deserialisation, compteurs d'observation). C'est ce qui oblige tout le decodage a passer sous
// `LockProcessDecode()` : deux films decodes en parallele dans le meme processus se voleraient
// leurs largeurs. La decision D10 du plan est de NE PAS de-globaliser pendant ce chantier — le
// perimetre est deja lourd, et une de-globalisation touche 28 crochets. Mais ne pas corriger
// n'autorise pas a AGGRAVER : ce test gele le compte au niveau mesure, pour que le chantier de
// performance ne laisse pas derriere lui dix globaux de plus que ce qu'il a trouve.
//
// # CE QUI EST COMPTE, EXACTEMENT
//
// Les NOMS declares par un `var` de NIVEAU PAQUET dans les fichiers non-test de
// `internal/games/halo_infinite/film/filmdec` — un bloc `var ( a = 1; b = 2 )` compte donc pour DEUX, parce que
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
// RETRAIT CIBLE : le jour ou `filmdec` est de-globalise (hors de ce plan, cf. §7). Critere
// mesurable de ce jour-la : `LockProcessDecode` n'a plus de raison d'etre.

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

// filmdecVarsGeles : le compte GELE des variables de paquet de `filmdec` (cf. l en-tete pour la
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
// l en-tete de `decode_gate.go` nomme comme l une des deux raisons du verrou de processus —
// deviennent les CHAMPS d un seul objet, `filmdec.Observation`. Un observateur ne change aucune
// consommation de bits : c est la propriete qui le distingue du profil, et elle est ecrite en
// tete de `observateur.go`. Bilan net : -36 = 43.
// RESSERRE A 42 LE 2026-09-17 (lot 2.3, famille « le profil de balayage remplace l heritage ») :
// `herite` — la DERNIERE variable que les lots 2.2.a/b/e avaient regroupee, « ce qu une passe
// laisse a la suivante dans le processus » — disparait. Ce qu elle portait (descripteur de
// traversee, largeur d axe absolue, largeurs d axe des objets du monde, decoupage MPP,
// `param_4` force) devient [filmdec.ProfilDeBalayage], une VALEUR : le lecteur de bits en tient
// une copie, le contexte du film celle du decodage courant, et `FrameConfig.Profil` la passe aux
// portes de balayage. La calibration de `killsource` la REND desormais
// (`Result.ProfilCalibre`), et `replaybuild` la passe explicitement a `replay.BuildFromFilm` —
// l heritage par l etat du processus, nomme par la decouverte D1 du lot 2.2.a, n existe plus.
// Bilan net : -1 = 42.
const filmdecVarsGeles = 42

// TestFilmdecPackageVarsNeCroitPas — LE RATCHET.
func TestFilmdecPackageVarsNeCroitPas(t *testing.T) {
	pkgDir := filepath.Join(apiRootDepuisIci(t), filepath.FromSlash("internal/games/halo_infinite/film/filmdec"))
	compte, parFichier := compterVarsDePaquet(t, pkgDir)
	switch {
	case compte > filmdecVarsGeles:
		t.Fatalf("l'etat global de `filmdec` a CRU : %d variables de paquet, gelees a %d "+
			"(D10 de PLAN_CUISSON_PERF, mesure du 2026-09-03).\n%s\n"+
			"Un nouveau reglage de decodage se passe en PARAMETRE (ScanFilmOptions, FilmContext), "+
			"pas en variable de paquet : chaque global de plus est un verrou process de plus a "+
			"tenir, et c'est deja `LockProcessDecode` qui serialise toute la cuisson.",
			compte, filmdecVarsGeles, detailParFichier(parFichier))
	case compte < filmdecVarsGeles:
		t.Logf("l'etat global de `filmdec` a BAISSE : %d variables de paquet au lieu de %d — "+
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
