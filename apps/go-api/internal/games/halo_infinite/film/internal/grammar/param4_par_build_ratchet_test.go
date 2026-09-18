package grammar

// param4_par_build_ratchet_test.go — LE RATCHET DE `param_4`, PAR BUILD (lot 5.1.7, 2026-09-18).
//
// # CE QU IL REMPLACE, ET POURQUOI L ANCIEN NE GARDAIT RIEN
//
// `TestParamByComponentEgaleLeNiveauDuRegistre` confrontait la table au `level` d UN SEUL film.
// Il etait VERT et il passait a cote de l essentiel : `param_4` N EST PAS CONSTANT d un build a
// l autre. Mesure du 2026-09-18 sur dix-neuf films — `ti=40 i2` vaut 1 jusqu a HI_1_9_0 et 2
// depuis HI_1_10_0 ; `biped-malleable-property` vaut 1 jusqu a HI_1_10_0 et 2 depuis HI_1_11_0 ;
// `unit-malleable-property` vaut 3 au format 20 et 4 ensuite. Un ratchet qui n interroge qu un
// seul registre ne peut pas voir cela.
//
// # LES DEUX SOURCES DE L ECRIVAIN, ET CE QUI LES REUNIT
//
// Desassemblage du 2026-09-18, arg5 en `[rsp+0x20]` :
//
//	FUN_142e2c690 (etat complet)  arg5 = `entree + 0x100` — le NIVEAU du registre DU FILM
//	FUN_14076cb60 (delta, et le record NEW par FUN_141f86704) arg5 = `vtable[0]()` du
//	                              descripteur — une CONSTANTE de l EXECUTABLE
//
// Les deux se rejoignent par l HYPOTHESE DU MIROIR, mesuree et tenue : le niveau du registre EST
// le `vtable[0]` du build ENREGISTREUR. Le film porte la constante de son propre build, et c est
// LUI la source sur les trois chemins — la table de l executable courant n en est que le miroir
// POUR LE BUILD COURANT, d ou son statut de ratchet et non de source.
//
// CE TEST GARDE DEUX CHOSES :
//
//  1. la table du depot vaut la constante de l EXECUTABLE COURANT (treize valeurs LUES, avec
//     l adresse de la fonction de six octets qui les rend) ;
//  2. le registre de CHAQUE BUILD temoin vaut cette meme constante, SAUF aux ecarts connus et
//     dates. Toute AUTRE divergence est rouge.
//
// Il ne decode aucun paquet : il lit le chunk 0 de sept mini-bobines VERSIONNEES, une par cle de
// profil. Les deux cles de format 20 n ont pas de mini-bobine au depot ; leur ecart mesure est
// ecrit en toutes lettres plus bas, pour le jour ou l une entre.

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// p4Constante porte la constante que `vtable[0]` du descripteur rend, et l adresse de la
// fonction qui la rend — LUE dans `HaloInfinite.exe`, jamais deduite d une statistique.
type p4Constante struct {
	valeur  uint32
	vtable0 string
}

// p4ConstantesExe : les treize composants dont un deserialiseur du depot branche sur `param_4`.
// `mov eax,K ; ret`, sauf `object-frame-configuration` qui fait `xor eax,eax ; ret`.
var p4ConstantesExe = map[string]p4Constante{
	compForwardUpDynPrec:                   {2, "0x141179610"},
	compObjectParentState:                  {3, "0x14117e0e0"},
	"unit-actor-control-component":         {2, "0x141179610"},
	"unit-actor-state-component":           {4, "0x140c85020"},
	"unit-malleable-property-component":    {4, "0x140c85020"},
	"biped-malleable-property-component":   {2, "0x141179610"},
	"biped-slide-component":                {1, "0x14117b4a0"},
	"object-maximum-vitalities-component":  {3, "0x14117e0e0"},
	"object-low-frequency-component":       {2, "0x141179610"},
	"object-frame-configuration-component": {0, "0x1405f0ac0"},
	"unit-control-component":               {2, "0x141179610"},
	compNavpointDistanceFilters:            {3, "0x14117e0e0"},
	compNavpointOffscreenFilters:           {2, "0x141179610"},
}

// p4BobinesParCle : une mini-bobine VERSIONNEE par cle de profil. Meme liste que le ratchet
// 0.A.3, et pour la meme raison : ce sont les seuls films que le depot porte.
var p4BobinesParCle = []struct{ court, build string }{
	{"a521164d", "HI_1_4_1"}, {"60ae07c4", "HI_1_8_0"}, {"11de8353", "HI_1_9_0"},
	{"111fa685", "HI_1_10_0"}, {"e5adf7b2", "HI_1_11_0"}, {"bcb6d393", "HI_1_12_0"},
	{"fb1a1a72", "HI_1_13_0"},
}

// p4EcartsAttendus : ce que le registre d un build donne QUAND il differe de l executable
// courant. Mesure du 2026-09-18, dix-neuf films. Cle `<court>|<ti>|<composant>`, valeur = le
// niveau que CE registre porte.
//
// LES DEUX CLES DE FORMAT 20 N ONT PAS DE MINI-BOBINE (`a349fea8`, `50247b26`) : leur ecart est
// MESURE et ecrit ici pour memoire, pas garde — `a349fea8` rend `ti=40 i2 = 1` et
// `biped-malleable-property = 1` ; `50247b26` ajoute `unit-malleable-property = 3` (sur `ti=35`
// ET `ti=40`) et porte `i52`/`i61` la ou les autres ont `i53`/`i62`.
var p4EcartsAttendus = map[string]uint32{
	// `ti=40 i2` : 1 jusqu a HI_1_9_0, 2 depuis HI_1_10_0.
	"a521164d|40|" + compForwardUpDynPrec: 1,
	"60ae07c4|40|" + compForwardUpDynPrec: 1,
	"11de8353|40|" + compForwardUpDynPrec: 1,
	// `biped-malleable-property` : 1 jusqu a HI_1_10_0, 2 depuis HI_1_11_0.
	"a521164d|35|biped-malleable-property-component": 1,
	"60ae07c4|35|biped-malleable-property-component": 1,
	"11de8353|35|biped-malleable-property-component": 1,
	"111fa685|35|biped-malleable-property-component": 1,
}

// TestParam4TableEgaleLExecutable : la table du depot est le miroir du build COURANT.
func TestParam4TableEgaleLExecutable(t *testing.T) {
	for nom, c := range p4ConstantesExe {
		v, ok := paramByComponent[nom]
		if !ok {
			if c.valeur == 1 {
				continue // pas d entree : le defaut (1) VAUT la constante lue.
			}
			t.Errorf("%s : absent de paramByComponent, mais l executable rend %d (%s)",
				nom, c.valeur, c.vtable0)
			continue
		}
		if v != c.valeur {
			t.Errorf("%s : la table dit %d, l executable rend %d (vtable[0] = %s).\n"+
				"La table est le MIROIR du build courant : elle ne peut pas le contredire.",
				nom, v, c.valeur, c.vtable0)
		}
	}
}

// TestParam4RegistreParBuild : le registre de chaque build temoin vaut la constante de
// l executable, SAUF aux ecarts connus et dates. Toute autre divergence est rouge.
func TestParam4RegistreParBuild(t *testing.T) {
	restants := map[string]bool{}
	for k := range p4EcartsAttendus {
		restants[k] = true
	}
	for _, b := range p4BobinesParCle {
		p4UneBobine(t, b.court, b.build, restants)
	}
	if len(restants) == 0 {
		return
	}
	var manquants []string
	for k := range restants {
		manquants = append(manquants, k)
	}
	sort.Strings(manquants)
	t.Errorf("ECART ATTENDU QUI N EXISTE PLUS : %s.\n"+
		"Le registre d un build temoin a change, ou la bobine a ete regeneree : re-mesurer avant "+
		"de retirer la ligne.", strings.Join(manquants, " - "))
}

// p4UneBobine confronte le registre d UNE bobine aux treize constantes.
func p4UneBobine(t *testing.T, court, build string, restants map[string]bool) {
	t.Helper()
	dir := filepath.Join("..", "..", "replay", "testdata", "minifilm_"+court)
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		t.Fatalf("%s (%s) : %v", court, build, err)
	}
	reg, err := NewFilmContext(film).Registry()
	if err != nil {
		t.Fatalf("registre de %s (%s) : %v", court, build, err)
	}
	for _, a := range reg.Archetypes {
		for i, nom := range a.Components {
			c, ok := p4ConstantesExe[nom]
			if !ok {
				continue
			}
			p4UnComposant(t, p4Cas{court, build, a.Index, i, nom, a.Level(i), c}, restants)
		}
	}
}

// p4Cas porte ce qu un composant confronte a l executable (regle des 5 parametres).
type p4Cas struct {
	court, build string
	ti, i        int
	nom          string
	lu           uint32
	c            p4Constante
}

// p4UnComposant statue UN composant : ecart attendu, ou divergence rouge.
func p4UnComposant(t *testing.T, k p4Cas, restants map[string]bool) {
	t.Helper()
	cle := fmt.Sprintf("%s|%d|%s", k.court, k.ti, k.nom)
	attendu, connu := p4EcartsAttendus[cle]
	if connu {
		delete(restants, cle)
		if k.lu != attendu {
			t.Errorf("%s (%s) ti=%d i%d %s : registre=%d, ecart ATTENDU=%d (exe %d).",
				k.court, k.build, k.ti, k.i, k.nom, k.lu, attendu, k.c.valeur)
		}
		return
	}
	if k.lu != k.c.valeur {
		t.Errorf("%s (%s) ti=%d i%d %s : registre=%d, executable=%d (%s) — DIVERGENCE NON "+
			"ATTENDUE.\nSoit ce build porte un `param_4` que la mesure du 2026-09-18 n a pas vu, "+
			"soit une largeur a bouge. Re-mesurer sur le film ENTIER avant d ajouter une ligne a "+
			"`p4EcartsAttendus` : une ligne ajoutee sans mesure fait taire ce ratchet.",
			k.court, k.build, k.ti, k.i, k.nom, k.lu, k.c.valeur, k.c.vtable0)
	}
}
