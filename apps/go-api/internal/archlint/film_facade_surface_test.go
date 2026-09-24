package archlint

// film_facade_surface_test.go — LE COMPTEUR DE SURFACE DE LA FRONTIERE DU DECODEUR.
//
// # POURQUOI IL EXISTE (lot 4.1.1-a, 2026-09-17)
//
// La facade `film/decfilm` pese 166 declarations exportees, et RIEN NE LE COMPTAIT : le chiffre
// ne vivait que dans trois commentaires (`decfilm.go:22`, ADR 0034 `:340` et `:511`), tous
// ecrits a la main et tous deja perimes (163 avant la fusion du lot 3.4.1). La seule borne
// effective etait le plafond de 500 lignes par fichier (`film_file_size_test.go`), soit ~20
// symboles de marge au rythme mesure (446 lignes / 166 symboles = 2,69 L par symbole).
//
// M4 AJOUTE a cette frontiere sans rien lui retirer, et c est la raison d etre de ce compteur :
// 4.1 y ajoute une porte d ecriture et une porte de lecture des faits, 4.4.1 les quatre
// revisions de couche la ou la facade n en exporte qu une. Sans ce test, la surface grossit sans
// qu aucune mesure ne le dise (note de preparation de M4, §1.2 et §1.4).
//
// CE N EST PAS UN LOT DE REDUCTION. V16 a reporte 4.3 et la note §1.3 a ecarte le lot « 4.0
// facade reduite » : ce test MESURE, il ne prescrit pas. Un plafond qui descend est le sens de la
// marche ; il se descend DANS LE COMMIT qui retire le symbole.
//
// # ET LA REDUCTION N ARRIVERA PAS : DECISION V25 (2026-09-18, utilisateur, cloture de M4)
//
// « Laisser les deux facades telles quelles, avec un ratchet. » La reduction de la facade (166) et
// de la surface compagnon (257) est NON RETENUE — ce n est plus un report, c est une decision, et
// elle est consignee au §1.4 du plan `.ai/PLAN_DECODEUR_FILM_2026-09-13.md` ainsi qu a l ADR 0034
// (section « State reached at M4 », D-1). CONSEQUENCE POUR CE FICHIER : il n est plus la mesure
// d entree d un lot a venir, il est LA SEULE CHOSE qui tient la ligne — d ou la phrase
// [exigenceDeJustificationDatee] dans ses trois messages d erreur.
//
// Mesure re-verifiee a la cloture de M4 (base `896a9ce04`) : facade **166**, INCHANGEE depuis la
// cloture de M3 — la projection de la note de preparation (« M4 augmente la facade ») est refutee
// par la mesure, les portes neuves de 4.1 et les deux accesseurs de revisions de 4.4.1 servant
// `replaybuild`, qui importe la couche de publication directement. Le compagnon, lui, est passe de
// 245 a **258**, chaque marche datee ci-dessous.
//
// # LA METHODE DE COMPTAGE, ECRITE ET REPRODUCTIBLE EN UNE COMMANDE
//
// Un plafond dont la valeur ne se reproduit pas n est pas un ratchet. Les deux mesures sont donc
// reproductibles a la main, et les commandes sont LE controle croise du compte par AST :
//
//	# (a) surface de la facade
//	grep -cE '^(const|type|var|func) [A-Z]' \
//	  apps/go-api/internal/games/halo_infinite/film/decfilm/decfilm.go
//	#   -> 166 sur a5d15e634
//
//	# (b) plafond compagnon : IDENTIFIANTS DISTINCTS cites hors de film/, tests compris
//	find apps/go-api -name '*.go' -not -path '*/film/*' -print0 \
//	  | xargs -0 grep -hoE '\breplay\.[A-Z][A-Za-z0-9_]*' | sort -u | wc -l
//	#   -> 245 sur a5d15e634
//
// Definitions, sans lesquelles les valeurs ne sont pas reproductibles :
//
//   - PERIMETRE (a) : le SEUL fichier `film/decfilm/decfilm.go`, declarations de premier niveau
//     (colonne 0) dont le nom est exporte. Le test les compte par `go/parser` ; le grep ci-dessus
//     est le controle croise, et les deux rendent le meme nombre.
//   - PERIMETRE (b) : tout fichier `.go` du module DONT LE CHEMIN NE CONTIENT PAS `/film/`,
//     TESTS COMPRIS, y compris les occurrences en COMMENTAIRE — c est ce que le grep voit, et la
//     mesure doit etre celle de la commande ecrite, sinon elle ne se reproduit pas.
//   - IDENTIFIANTS DISTINCTS, JAMAIS OCCURRENCES. Les occurrences valent 1 277 sur la meme base
//     et ne mesurent que la verbosite du code appelant.
//   - CONTROLE CROISE DE (b) : restreindre l ensemble aux seuls fichiers qui IMPORTENT
//     `halo_infinite/film/replay` rend 244 sur cette base, et l unique ecart est nomme :
//     `replay.TestScanFilmPlayerTableCableLeCompteurDeBuildInconnu`, un NOM DE TEST cite dans le
//     commentaire d en-tete de `internal/sync/killcollector/cle_inconnue_test.go:13`. Le lecteur
//     qui refait la mesure doit retrouver le meme ecart ; s il en trouve un autre, c est le
//     perimetre qui a bouge.
//   - DATE ET BASE : les valeurs gelees portent leur date et le sha de leur base. Elles se
//     RE-MESURENT a l entree d un lot, jamais ne se recopient d un document.
//
// # LE RATCHET ROUGIT DANS LES DEUX SENS, ET C EST VOLONTAIRE
//
// L egalite, pas l inegalite. Un symbole de plus rougit (c est l objet du compteur) ; un symbole
// de MOINS rougit aussi, parce qu un plafond laisse au-dessus de la mesure reconstitue une marge
// en silence — et une marge silencieuse est exactement ce que les trois commentaires perimes de
// `decfilm.go` ont produit. Les deux sens ont ete prouves par mutation a la main avant le commit
// de pose.
//
// # LE PLANCHER ANTI-MUET
//
// Un compteur qui ne compte plus rien passe au vert. Les deux mesures font donc `t.Fatalf` quand
// le fichier de la facade manque ou quand le compte tombe a zero.

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"testing"
)

// cheminDeLaFacade — le fichier mesure par (a), relatif a `apps/go-api`.
const cheminDeLaFacade = "internal/games/halo_infinite/film/decfilm/decfilm.go"

// exigenceDeJustificationDatee — LA PHRASE QUE LES TROIS MESSAGES D ERREUR DOIVENT DIRE.
//
// POURQUOI ELLE EST ECRITE UNE SEULE FOIS (regle des 2 copies du depot) : les trois plafonds de
// ce fichier — surface de la facade, plafond compagnon, ventilation par famille — posent la MEME
// exigence, et une phrase recopiee trois fois aurait derive au premier lot qui en reformule une.
//
// POURQUOI ELLE EXISTE (cloture M4, 2026-09-18, decision V25) : la reduction des deux facades est
// NON RETENUE, et ce ratchet est DESORMAIS LA SEULE CHOSE qui tient la ligne. Un message qui dit
// seulement « le plafond a bouge » laisse croire qu on le remonte en changeant le chiffre ; la
// regle du depot est qu une hausse se JUSTIFIE, avec sa date, dans ce fichier — c est ce que
// l historique des montees du plafond compagnon fait deja, ligne par ligne.
const exigenceDeJustificationDatee = "UNE HAUSSE EXIGE UNE JUSTIFICATION DATEE DANS CE FICHIER " +
	"(une ligne : la date, le sha de la base re-mesuree, et les symboles qui la font monter) — " +
	"jamais un chiffre change seul."

// plafondSurfaceFacade — declarations exportees de premier niveau de la facade.
const plafondSurfaceFacade = 166 // 2026-09-17 — base a5d15e634

// plafondSurfaceReplay — identifiants `replay.<Symbole>` DISTINCTS cites hors de `film/`.
//
// 1,5 fois la facade : c est la mesure qui dit ou est le vrai poids de la frontiere (note de
// preparation de M4, §1.3 point 4). L ADR 0034 `:335` en annoncait 239 a la cloture de M2.
//
// HISTORIQUE DES MONTEES, une ligne par commit qui la fait bouger — c est ce qui rend le ratchet
// lisible plutot qu une valeur qui change sans raison ecrite :
//
//	245  4.1.1-a (2026-09-17, base a5d15e634)  pose du compteur
//	246  4.1.1-c (2026-09-17)                  `replay.FilmFacts` nomme dans le godoc de
//	                                           `domain/title/registry_film_facts.go` (la
//	                                           distinction avec `.facts.json`)
//	253  4.1.2   (2026-09-17)                  la bascule : `replaybuild` cite les sept symboles
//	                                           du fichier de faits (`FilmFactsFile`,
//	                                           `FilmStatborg`, `DecodeFilmFactsEntete`,
//	                                           `DecodeFilmFactsFile`, `EncodeFilmFactsFile`,
//	                                           `BuildFromFacts`, `BuildFromFilmAvecFaits`)
//	255  fusion  (2026-09-18, integration 4.2)  la montee de schema 62 : `replaybuild/kills.go`
//	                                           cite `DeathsPathsCoverage` et `DeathsPathTally`
//	                                           pour publier `coverage.deathsPaths`. RE-MESURE A
//	                                           L ENTREE de la fusion (`comm` sur les deux
//	                                           inventaires : 2 ajouts, 0 disparition) — un
//	                                           plafond gele sur la base d un lot rougit a la
//	                                           fusion sur tout ce qui a grossi ailleurs.
//	257  4.4.1   (2026-09-18)                  le verdict par couche : `replaybuild` cite
//	                                           `RevisionsCourantesDesCouches` et `FamilleDeRevision`,
//	                                           les deux seuls symboles par lesquels un paquet hors du
//	                                           decodeur peut connaitre les revisions courantes (les
//	                                           quatre couches vivent sous `film/internal/`, le
//	                                           compilateur les refuse). RE-MESURE A L ENTREE du lot :
//	                                           255 sur la base fusionnee, 257 apres.
//	258  5.1.7   (2026-09-19)                  la ventilation des vies de vehicule : l instrument
//	                                           `replaybuild/ventilation_vies_research_test.go` cite
//	                                           `replay.VehicleScan` — le type que l observateur rend
//	                                           a l etape `vehicles`, et la SEULE facon de mesurer la
//	                                           cause par vie sans redeviner une largeur. Un seul
//	                                           symbole neuf. RE-MESURE A L ENTREE du lot : 257 sur
//	                                           `50136328c`, 258 apres.
//	259  5.1.5   (2026-09-19)                  la montee de schema 63 : `replay.VehicleCycle`, le
//	                                           type publie du cycle de reapparition par
//	                                           emplacement, cite par le convertisseur jumeau
//	                                           `service/replayview/convert_vehicles.go`. Un seul
//	                                           symbole neuf — `returnProgress` n en ajoute AUCUN,
//	                                           il reutilise `replay.GaugePoint`, deja cite par le
//	                                           meme convertisseur pour la jauge des zones.
//	                                           RE-MESURE A L ENTREE du lot : 258 sur `8b8d93c87`,
//	                                           259 apres.
//	260  5.2-A   (2026-09-20)                  la montee de schema 64 : `replay.ZoneGaugeRamp`, le
//	                                           span de rampe de jauge qui porte `capturingTeam`,
//	                                           cite par le convertisseur jumeau
//	                                           `service/replayview/convert_objectives.go`. Un seul
//	                                           symbole neuf : le champ `GaugeRamps` de
//	                                           `ZoneState` n en ajoute AUCUN, `replay.ZoneState`
//	                                           etant deja cite par le meme convertisseur.
//	                                           RE-MESURE A L ENTREE du lot : 259 sur `65e5c0731`,
//	                                           260 apres.
//	263  ajsup-E (2026-09-21)                  le correctif des ARMES DE BASE (constat utilisateur
//	                                           sur `b1ad85eb` : l Empaleur classe « arme de base »
//	                                           alors que le loadout de depart est egal pour tous).
//	                                           La cause est que le canal `loadouts` est publie sur
//	                                           une grille d images-cles GLOBALE, jamais au spawn :
//	                                           le convertisseur doit donc ECARTER une emission qui
//	                                           suit une prise d arme de la meme vie, et il nomme
//	                                           pour cela les natures de prise du film —
//	                                           `replay.PickupWeapon`, `replay.WeaponTaken`,
//	                                           `replay.WeaponSwapped`. TROIS symboles neufs, tous
//	                                           dans le seul `sync/replayartifacts/padtiers_prises.go` ;
//	                                           les canaux eux-memes (`Pickup`, `WeaponChange`,
//	                                           `GroundWeapon`, `Track`) n en ajoutent AUCUN, ils
//	                                           sont lus par champ sur `ReplayDocument`, deja cite.
//	                                           Les comparer a des litteraux aurait evite la montee
//	                                           en recreant le vocabulaire du film hors du film :
//	                                           c est exactement ce que la frontiere interdit.
//	                                           RE-MESURE A L ENTREE du lot : 260 sur `1840d0cbc`,
//	                                           263 apres.
//	262  5.3.6   (2026-09-21)                  la montee de schema 65 : `replay.Stance` et
//	                                           `replay.StanceCoverage`, l intervalle d etat de
//	                                           mouvement et sa couverture, cites par les deux
//	                                           convertisseurs jumeaux
//	                                           `service/replayview/convert_inventory.go` et
//	                                           `convert_coverage.go`. DEUX symboles neufs — le
//	                                           champ `Stances` du document et celui de `Coverage`
//	                                           n en ajoutent aucun, `replay.ReplayDocument` et
//	                                           `replay.Coverage` etant deja cites.
//	                                           RE-MESURE A L ENTREE du lot : 260 sur `0fc6a3349`,
//	                                           262 apres.
//	265  fusion serie 5 -> feat/v75 (2026-09-22)  les deux rangs ci-dessus sont INDEPENDANTS (ajsup-E : +3 natures
//	                                           de prise d arme ; 5.3.6 : +2 pour replay.Stance et
//	                                           replay.StanceCoverage) et se cumulent : 260 + 3 + 2 = 265,
//	                                           re-mesure a la fusion.
//	267  rr(m3)  (2026-09-24)                  la montee de schema 69 de la campagne « retours
//	                                           rejeu » : `replay.KeyframeCoverage` (sante de la
//	                                           marche d image-cle, lot M3.1) et
//	                                           `replay.BirthLoadoutCoverage` (dotations de
//	                                           naissance, lot M3.2), cites par le convertisseur
//	                                           jumeau `service/replayview/convert_coverage_armes.go`.
//	                                           DEUX symboles neufs — les champs `Coverage.Keyframes`,
//	                                           `Coverage.BirthLoadouts`, `Loadout.Src`/`K` et
//	                                           `WeaponChange.K` n en ajoutent aucun.
//	                                           RE-MESURE A L ENTREE du lot : 265 sur `fe7079f41`,
//	                                           267 apres.
const plafondSurfaceReplay = 267 // 2026-09-24 — rr(m3) sur fe7079f41 : 265 + 2 (KeyframeCoverage, BirthLoadoutCoverage)

// plafondsParFamilleFacade — la surface de la facade VENTILEE PAR PAQUET D ORIGINE.
//
// LA FAMILLE EST LE PAQUET DE DESTINATION DU RENVOI, pas la section du godoc de `decfilm.go` :
// les deux divergent, et c est mesure (note §5, decouverte 9 — 10 alias de `film/types` sont
// ranges sous les sections `source` / `killsource` / `objectives`). Le total est le meme ; la
// ventilation par paquet est celle qui se reproduit par AST.
//
// A quoi elle sert : un total qui ne bouge pas peut cacher un symbole retire d un cote et ajoute
// de l autre. La ventilation nomme alors la couche qui a grossi.
var plafondsParFamilleFacade = map[string]int{
	"grammar":    46, // 2026-09-17 — base a5d15e634
	"objectives": 37,
	"killsource": 35,
	"fallback":   11,
	"types":      10,
	"profile":    8,
	"source":     7,
	"weaponscan": 5,
	"weaponv3":   4,
	"positions":  2,
	"facts":      1,
}

// TestSurfaceDeLaFacadeDuDecodeur : (a) — le compte par AST, et sa ventilation par famille.
func TestSurfaceDeLaFacadeDuDecodeur(t *testing.T) {
	parFamille, total := surfaceDeLaFacade(t)
	if total == 0 {
		t.Fatalf("zero declaration exportee lue dans %s : le compteur ne compte plus rien "+
			"(fichier vide, parseur en echec, ou la facade a DEMENAGE — deplacer "+
			"`cheminDeLaFacade` avec elle)", cheminDeLaFacade)
	}
	if total != plafondSurfaceFacade {
		t.Errorf("surface de la facade : %d declarations exportees, plafond gele a %d.\n"+
			"Ce test rougit DANS LES DEUX SENS : au-dessus, la frontiere a grossi (c est ce que "+
			"le compteur mesure) ; en dessous, baisser la constante DANS CE COMMIT — un plafond "+
			"laisse au-dessus de la mesure reconstitue une marge en silence.\n%s\n"+
			"Controle croise : grep -cE '^(const|type|var|func) [A-Z]' apps/go-api/%s",
			total, plafondSurfaceFacade, exigenceDeJustificationDatee, cheminDeLaFacade)
	}
	for famille, n := range parFamille {
		plafond, connue := plafondsParFamilleFacade[famille]
		if !connue {
			t.Errorf("famille %q (%d symbole(s)) absente de `plafondsParFamilleFacade` : une "+
				"couche neuve traverse la facade sans etre comptee — l inscrire avec sa valeur.",
				famille, n)
			continue
		}
		if n != plafond {
			t.Errorf("famille %q : %d symbole(s), plafond gele a %d.\n%s",
				famille, n, plafond, exigenceDeJustificationDatee)
		}
	}
	for famille, plafond := range plafondsParFamilleFacade {
		if _, vivante := parFamille[famille]; !vivante {
			t.Errorf("famille %q est au plafond (%d) mais ne traverse plus la facade : entree "+
				"perimee, la retirer dans le commit qui l a videe.", famille, plafond)
		}
	}
}

// TestSurfaceDeReplayCiteeHorsDuDecodeur : (b) — le plafond compagnon.
func TestSurfaceDeReplayCiteeHorsDuDecodeur(t *testing.T) {
	ids := identifiantsReplayHorsDuDecodeur(t)
	if len(ids) == 0 {
		t.Fatal("zero identifiant `replay.<Symbole>` lu hors de `film/` : le compteur ne compte plus " +
			"rien (perimetre de balayage casse). `film/replay` est cite par une trentaine de " +
			"paquets — un zero est une panne, pas une victoire.")
	}
	if len(ids) != plafondSurfaceReplay {
		sort.Strings(ids)
		apercu := ids
		if len(apercu) > 12 {
			apercu = apercu[:12]
		}
		t.Errorf("surface de `film/replay` citee hors du decodeur : %d identifiants distincts, "+
			"plafond gele a %d.\nCe test rougit DANS LES DEUX SENS (cf. en-tete).\n%s\n"+
			"Controle croise : find apps/go-api -name '*.go' -not -path '*/film/*' -print0 "+
			"| xargs -0 grep -hoE '\\breplay\\.[A-Z][A-Za-z0-9_]*' | sort -u | wc -l\n"+
			"Douze premiers lus : %s",
			len(ids), plafondSurfaceReplay, exigenceDeJustificationDatee, strings.Join(apercu, " "))
	}
}

// racineAPIDuCompteur rend `.../apps/go-api`.
func racineAPIDuCompteur(t *testing.T) string {
	t.Helper()
	_, ici, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller a echoue")
	}
	return filepath.Dir(filepath.Dir(filepath.Dir(ici)))
}

// surfaceDeLaFacade compte PAR AST les declarations exportees de premier niveau de la facade, et
// les ventile par paquet d origine.
//
// LA FAMILLE EST LUE DANS LE RENVOI : le qualifieur de paquet de la valeur renvoyee (alias, const,
// var) ou du corps du renvoi (func). Un renvoi qui ne nommerait aucun paquet de couche serait de
// la LOGIQUE posee dans la facade — ce que son en-tete interdit — et sort en famille `(aucune)`,
// donc en famille inconnue, donc en rouge.
func surfaceDeLaFacade(t *testing.T) (map[string]int, int) {
	t.Helper()
	chemin := filepath.Join(racineAPIDuCompteur(t), filepath.FromSlash(cheminDeLaFacade))
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, chemin, nil, 0)
	if err != nil {
		t.Fatalf("analyse de %s : %v — si la facade a DEMENAGE, deplacer `cheminDeLaFacade` "+
			"avec elle", cheminDeLaFacade, err)
	}
	paquets := paquetsImportes(f)
	parFamille := map[string]int{}
	total := 0
	for _, decl := range f.Decls {
		for _, nom := range nomsExportesDeLaDeclaration(decl) {
			total++
			parFamille[familleDuRenvoi(decl, nom, paquets)]++
		}
	}
	return parFamille, total
}

// paquetsImportes rend les noms locaux des paquets importes : l ensemble des qualifieurs qui
// peuvent NOMMER une famille.
func paquetsImportes(f *ast.File) map[string]bool {
	out := map[string]bool{}
	for _, imp := range f.Imports {
		chemin := strings.Trim(imp.Path.Value, `"`)
		nom := chemin[strings.LastIndex(chemin, "/")+1:]
		if imp.Name != nil {
			nom = imp.Name.Name
		}
		out[nom] = true
	}
	return out
}

// nomsExportesDeLaDeclaration rend les noms EXPORTES declares au premier niveau par une
// declaration. Une declaration groupee (`const (...)`) compte chacun de ses noms.
func nomsExportesDeLaDeclaration(decl ast.Decl) []string {
	var out []string
	switch d := decl.(type) {
	case *ast.FuncDecl:
		if d.Recv == nil && d.Name.IsExported() {
			out = append(out, d.Name.Name)
		}
	case *ast.GenDecl:
		for _, spec := range d.Specs {
			switch s := spec.(type) {
			case *ast.TypeSpec:
				if s.Name.IsExported() {
					out = append(out, s.Name.Name)
				}
			case *ast.ValueSpec:
				for _, n := range s.Names {
					if n.IsExported() {
						out = append(out, n.Name)
					}
				}
			}
		}
	}
	return out
}

// familleDuRenvoi rend le paquet d origine d un symbole re-exporte : le PREMIER qualifieur de
// paquet importe rencontre dans la declaration, `nom` servant a isoler le bon `ValueSpec` d une
// declaration groupee.
func familleDuRenvoi(decl ast.Decl, nom string, paquets map[string]bool) string {
	noeud := noeudPorteurDuNom(decl, nom)
	if noeud == nil {
		return "(aucune)"
	}
	famille := "(aucune)"
	ast.Inspect(noeud, func(n ast.Node) bool {
		if famille != "(aucune)" {
			return false
		}
		sel, ok := n.(*ast.SelectorExpr)
		if !ok {
			return true
		}
		if id, ok := sel.X.(*ast.Ident); ok && paquets[id.Name] {
			famille = id.Name
			return false
		}
		return true
	})
	return famille
}

// noeudPorteurDuNom isole la portion de declaration qui porte LA VALEUR RENVOYEE pour `nom` —
// jamais la signature.
//
// LA DISTINCTION EST LA MESURE ELLE-MEME. `func ScanFilmDeaths(f *source.Film) []types.Death`
// NOMME `source` et `types` dans sa signature et renvoie `grammar.ScanFilmDeaths(f)` : sa famille
// d origine est `grammar`, et prendre le premier qualifieur de la declaration rangerait 15
// symboles sous `source` (mesure du 2026-09-17, avant correction). Pour une fonction, seul le
// CORPS compte ; pour un alias, le type renvoye ; pour un `const`/`var`, la valeur (ou le type, a
// defaut de valeur).
func noeudPorteurDuNom(decl ast.Decl, nom string) ast.Node {
	switch d := decl.(type) {
	case *ast.FuncDecl:
		if d.Body == nil {
			return nil
		}
		return d.Body
	case *ast.GenDecl:
		for _, spec := range d.Specs {
			switch s := spec.(type) {
			case *ast.TypeSpec:
				if s.Name.Name == nom {
					return s.Type
				}
			case *ast.ValueSpec:
				for _, n := range s.Names {
					if n.Name != nom {
						continue
					}
					if len(s.Values) > 0 {
						return valeurDuNom(s, nom)
					}
					return s.Type
				}
			}
		}
	}
	return nil
}

// valeurDuNom rend l expression affectee a `nom` dans un `ValueSpec`. Une declaration groupee de
// la facade affecte un nom par valeur ; le cas « moins de valeurs que de noms » (heritage d un
// `const` implicite) retombe sur la premiere valeur, qui est celle qui nomme le paquet.
func valeurDuNom(s *ast.ValueSpec, nom string) ast.Expr {
	for i, n := range s.Names {
		if n.Name == nom && i < len(s.Values) {
			return s.Values[i]
		}
	}
	return s.Values[0]
}

// motifIdentifiantReplay — le MEME motif que le grep de l en-tete, a la lettre.
var motifIdentifiantReplay = regexp.MustCompile(`\breplay\.[A-Z][A-Za-z0-9_]*`)

// identifiantsReplayHorsDuDecodeur rend les identifiants `replay.<Symbole>` DISTINCTS cites par un
// fichier `.go` du module dont le chemin ne contient pas `/film/`.
//
// SUR LES OCTETS BRUTS, ET PAS PAR AST : la mesure doit etre celle de la commande ecrite en
// en-tete, commentaires compris. Un compte par AST serait plus « propre » et ne se reproduirait
// pas a la main — ce qui est precisement le defaut que ce fichier ferme.
func identifiantsReplayHorsDuDecodeur(t *testing.T) []string {
	t.Helper()
	racine := racineAPIDuCompteur(t)
	vus := map[string]bool{}
	err := filepath.WalkDir(racine, func(chemin string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			nom := d.Name()
			if chemin != racine && (strings.HasPrefix(nom, ".") || nom == "node_modules") {
				return fs.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(chemin, ".go") {
			return nil
		}
		rel, rerr := filepath.Rel(racine, chemin)
		if rerr != nil {
			return rerr
		}
		if strings.Contains(filepath.ToSlash(rel), "/film/") {
			return nil
		}
		blob, rerr := os.ReadFile(chemin) //nolint:gosec // chemin derive du module
		if rerr != nil {
			return rerr
		}
		for _, m := range motifIdentifiantReplay.FindAll(blob, -1) {
			vus[string(m)] = true
		}
		return nil
	})
	if err != nil {
		t.Fatalf("balayage de %s : %v", racine, err)
	}
	out := make([]string, 0, len(vus))
	for id := range vus {
		out = append(out, id)
	}
	return out
}
