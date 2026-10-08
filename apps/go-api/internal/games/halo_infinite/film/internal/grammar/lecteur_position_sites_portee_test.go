package grammar

// lecteur_position_sites_portee_test.go — LES EXCEPTIONS DATEES DU PORTAGE, SOUS LA PORTEE DE
// L ETAT COMPLET (plan LK, LK.5.4).
//
// Sous la portee `DAT_144e61ea0` ([Lecteur.portee]), la garde de pleine precision du jeu
// (`FUN_14076f91c`) est vraie : `FUN_14076e494` lit le vecteur BRUT (`FUN_1411b259c`, R(96)) a la
// place de la position quantifiee. Un site en exception datee (`lecteur_position_exceptions.go`) dont
// la lecture sous la garde est relue chez le jeu la porte en tete de sa fonction : chaque cas ci-dessous
// fabrique le flux qu ECRIT le jeu sous la portee et exige que le site le lise au bit pres, SANS noter
// d exception. Hors de la portee, les cas de `lecteur_position_sites_test.go` et des fichiers
// `lecteur_position_exceptions_*_test.go` tiennent l ancien lecteur.
//
// Mutation jouee pour chaque site (retirer le bloc de tete) : son cas rougit.

import (
	"go/ast"
	"testing"
)

// brut96 est le vecteur brut de `FUN_1411b259c` : trois flottants de 32 bits, finis (motif alterne).
func brut96() []champDeFlux { return seul(fixe(32), fixe(32), fixe(32)) }

// casDesExceptionsSousLaPortee rend un flux par site, tel que l ecrit le jeu sous la portee.
func casDesExceptionsSousLaPortee() []casDeSite {
	return []casDeSite{
		// ti=5 i12 player-desired-respawn-location (FUN_142f03ec8) : porte, R(96), FUN_14076dc04 R(19).
		{nom: "player-desired-respawn-location", indexW: 1,
			flux: concat(seul(bit(true)), brut96(), seul(fixe(19))),
			lire: parNom(compPlayerDesiredRespawnLoc, 5, 0)},
		// ti=14 i0 crew-order (FUN_142ed9120) : FUN_142b1cf3c R(3), porte, R(96).
		{nom: "crew-order", indexW: 1, flux: concat(seul(fixe(3), bit(true)), brut96()),
			lire: parNom("crew-order-component", 14, 0)},
		// ti=30 i0 tacmap-poiicon (FUN_142ed8418) : le bloc, R(96) (thunk FUN_1424e0e38), la queue.
		{nom: "tacmap-poiicon", indexW: 1,
			flux: concat(seul(fixe(32), fixe(32), bit(true), fixe(3), fixe(32), fixe(32), fixe(9), fixe(9)),
				brut96(), seul(fixe(32), bit(false), fixe(8), fixe(8), fixe(8), fixe(8))),
			lire: parNom("tacmap-poiicon", 30, 0)},
		// ti=32 i0 tacmap-areaofinterest (FUN_142ed7764) : R(32), R(3), R(96), R(12).
		{nom: "tacmap-areaofinterest", indexW: 1, flux: concat(seul(fixe(32), fixe(3)), brut96(), seul(fixe(12))),
			lire: parNom("tacmap-areaofinterest", 32, 0)},
		// ti=33 i0 tacmap-displayasset (FUN_142ed7d38) : R(32), R(32), R(2), R(96), R(96), R(96), R(1).
		{nom: "tacmap-displayasset", indexW: 1,
			flux: concat(seul(fixe(32), fixe(32), fixe(2)), brut96(),
				seul(fixe(64), fixe(32), fixe(64), fixe(32), bit(true))),
			lire: parNom("tacmap-displayasset", 33, 0)},
		// ti=34 i11 tacmap-cooptetherarea (FUN_142ed4198) : R(96), R(12), R(12).
		{nom: "tacmap-cooptetherarea", indexW: 1, flux: concat(brut96(), seul(fixe(12), fixe(12))),
			lire: parNom("tacmap-cooptetherarea", 34, 0)},
		// ti=34 i7 tacmap-waypointstate (FUN_140f04d88) : R(1), R(32), R(96), puis R(1) au niveau 2 ;
		// au niveau 1, pas de R(1).
		{nom: "tacmap-waypointstate niveau registre 2", indexW: 1,
			flux: concat(seul(bit(true), fixe(32)), brut96(), seul(bit(true))),
			lire: parNom("tacmap-waypointstate", 34, 2)},
		{nom: "tacmap-waypointstate niveau registre 1", indexW: 1,
			flux: concat(seul(bit(true), fixe(32)), brut96()), lire: parNom("tacmap-waypointstate", 34, 1)},
		// ti=21 i2..i11 flock-destination (FUN_140fb8af0) : R(1), R(96), puis R(2) au-dela du niveau 1.
		{nom: "flock-destination niveau registre 2", indexW: 1,
			flux: concat(seul(bit(true)), brut96(), seul(fixe(2))),
			lire: parNom("flock-destination-component", 21, 2)},
		{nom: "flock-destination niveau registre 1", indexW: 1,
			flux: concat(seul(bit(true)), brut96()), lire: parNom("flock-destination-component", 21, 1)},
	}
}

// consultationsDirectesDeLaGarde : les fonctions de `lecteur_position_exceptions.go` qui consultent
// encore la garde sans passer par [Lecteur.sousLaGardeSinonException], et pourquoi (date).
var consultationsDirectesDeLaGarde = map[string]string{
	"consumeFlockPosition": "2026-10-08 (plan LK) : conforme sous la garde depuis le lot J6.3, hors de " +
		"la liste de LK.5 ; il note l exception meme sous la garde (decouverte D-20 du plan LK)",
}

// TestLesExceptionsDecidentLaGardeParUnSeulGeste — regle 6 : un site en exception datee decide entre la
// lecture du jeu sous la garde et son ancien lecteur par [Lecteur.sousLaGardeSinonException] ; hors de
// la liste datee, aucune fonction du fichier n appelle `fullPrecisionGate`. Mutation jouee (un site qui
// consulte la garde a la main) : rouge.
func TestLesExceptionsDecidentLaGardeParUnSeulGeste(t *testing.T) {
	asts, fset := parserFichiers(t, []string{fichierDesExceptions})
	vus := map[string]bool{}
	for _, d := range asts[fichierDesExceptions].Decls {
		fd, ok := d.(*ast.FuncDecl)
		if !ok || fd.Body == nil {
			continue
		}
		ast.Inspect(fd.Body, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			if id, ok := call.Fun.(*ast.Ident); ok && id.Name == "fullPrecisionGate" {
				vus[fd.Name.Name] = true
				if _, permis := consultationsDirectesDeLaGarde[fd.Name.Name]; !permis {
					t.Errorf("%s : %s consulte la garde a la main — passer par sousLaGardeSinonException",
						fset.Position(call.Pos()), fd.Name.Name)
				}
			}
			return true
		})
	}
	for nom := range consultationsDirectesDeLaGarde {
		if !vus[nom] {
			t.Errorf("%s ne consulte plus la garde : le retirer de consultationsDirectesDeLaGarde", nom)
		}
	}
}

// TestChaqueExceptionLitLeFluxDuJeuSousLaPortee — LK.5.4 : sous la portee, chaque exception relue lit
// comme le jeu, et n est pas notee comme exception.
func TestChaqueExceptionLitLeFluxDuJeuSousLaPortee(t *testing.T) {
	for _, c := range casDesExceptionsSousLaPortee() {
		t.Run(c.nom, func(t *testing.T) {
			buf, total := ecrireFlux(c.flux)
			br := sousLaPortee(lecteurDeSite(buf, c.indexW))
			c.lire(br)
			if got := br.BitPos(); got != total || br.arret != ArretAucun {
				t.Fatalf("%s sous la portee : %d bits lus, arret %v ; l ecrivain du jeu en pose %d", c.nom, got,
					br.arret, total)
			}
			if br.exceptionDatee {
				t.Errorf("%s sous la portee : le site lit comme le jeu, il ne doit pas noter d exception", c.nom)
			}
		})
	}
}
