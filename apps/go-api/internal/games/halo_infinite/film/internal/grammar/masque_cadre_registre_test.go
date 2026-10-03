package grammar

// masque_cadre_registre_test.go — LE RATCHET DU CADRE DU MASQUE DE PRESENCE (lot 5.17.2).
//
// # CE QU IL FIXE, ET POURQUOI IL EXISTE
//
// Le lot 5.16 avait releve chez l ecrivain que `FUN_14076cb60` teste le bit
// `masque >> ((i - decales) & 0xff)` et non le bit `i` brut, et en avait fait le premier suspect
// du residu de `bfecd02b` (D1 du 5.16). Le lot 5.17 a lu les DEUX bouts de la chaine, et le
// decalage n a PAS d image hors ligne. Les adresses :
//
//	FUN_14076cb60   iterateur de DELTA. Il parcourt le descripteur d archetype du PROCESSUS
//	                (`*(int *)(desc + 0x4320)` composants, deserialiseurs en `desc + i*8`), et il
//	                ECARTE, sans consommer un bit, tout composant que le FILM ne declare pas :
//	                  si FUN_1404f2b4c() et !FUN_1428e1dac(&filmSingleton, ti, nom) :
//	                     decales++ ; continue
//	                `FUN_1404f2b4c` est vrai UNIQUEMENT en rejeu de film (`TLS+0x238` porte un nom
//	                de contexte non vide ET l etat vaut 2). Une SECONDE sortie incremente
//	                `decales` : `niveau = FUN_1428e1b50(&filmSingleton, ti, nom)` puis
//	                `deser->vtable[0x10](niveau)` — le niveau DU FILM refuse par le
//	                deserialiseur DU PROCESSUS.
//	FUN_1428e1dac   le filtre. Il cherche `nom` dans le bloc `ti` du registre de `chunk_00` :
//	                `base + 8 + ti*0x4100`, pas de `0x104`, 64 entrees.
//	FUN_1428e1b50   le niveau. MEME cadrage, et il rend le `u32` en `entree + 0x100` (defaut 1).
//	FUN_142e2c690   l iterateur d ETAT COMPLET, et c est lui qui tranche : il parcourt le bloc du
//	                registre DU FILM entree par entree (`param_4 += 0x104`, 64 tours, arret au
//	                premier nom vide), resout le deserialiseur PAR NOM, et lui passe le niveau lu
//	                en `entree + 0x100`. Aucun decalage, aucun masque : l index du jeu sur ce
//	                chemin EST l index du registre du film.
//
// Autrement dit `i - decales` est la conversion « descripteur du build qui REJOUE » ->
// « registre du film », et le decodeur hors ligne est deja du cote du registre du film : les
// composants qu il itere ([Archetype.Components]) sont les entrees NOMMEES du bloc, lues par
// `registry.go` avec les MEMES constantes de cadrage que le filtre. Porter `i - decales` hors
// ligne reviendrait a appliquer deux fois la meme conversion.
//
// Ce fichier fige les deux moities de cette egalite pour qu elle ne derive pas en silence :
// le CADRAGE (les quatre constantes, contre les litteraux de l ecrivain) et le CADRE D INDEX
// (le bit `k` du masque designe l entree `k` du bloc).

import "testing"

// TestCadreDuRegistreEstCeluiDuFiltre fige les quatre constantes de cadrage du registre contre
// les litteraux que `FUN_1428e1dac` et `FUN_1428e1b50` portent. Elles sont la raison pour
// laquelle l index du masque du depot est deja celui des composants RETENUS.
func TestCadreDuRegistreEstCeluiDuFiltre(t *testing.T) {
	cas := []struct {
		nom      string
		vu, veut int
		adresse  string
	}{
		{"base des entrees", registryEntryBase, 8,
			"FUN_1428e1dac @1428e1e0f : lVar6 + 8 + ti*0x4100"},
		{"taille d un bloc", archetypeBlockSize, 0x4100,
			"FUN_1428e1dac : (longlong)param_2 * 0x4100 ; borne pcVar4 + 0x4100"},
		{"pas d une entree", registrySlotSize, 0x104,
			"FUN_1428e1dac @1428e1e4c : pcVar4 = pcVar4 + 0x104"},
		{"entrees par bloc", archetypeBlockSlots, 0x4100 / 0x104,
			"FUN_1428e1b50 : borne uVar4 > 0x3f ; FUN_142e2c690 : borne uVar14 > 0x3f"},
		{"offset du niveau", registryEntryLevelOffset, 0x100,
			"FUN_1428e1b50 @1428e1bdf : *(u32 *)(lVar6 + 0x100) ; FUN_142e2c690 : param_4 + 0x100"},
	}
	for _, c := range cas {
		if c.vu != c.veut {
			t.Errorf("%s = %#x, l ecrivain dit %#x (%s)", c.nom, c.vu, c.veut, c.adresse)
		}
	}
	// 0x4100 / 0x104 = 64 exactement : le bloc est PLEIN, sans octet de rab. C est ce qui fait
	// que la borne `0x3f` de l ecrivain et notre `archetypeBlockSlots` sont le meme nombre.
	if archetypeBlockSlots*registrySlotSize != archetypeBlockSize {
		t.Errorf("%d entrees de %#x ne remplissent pas un bloc de %#x",
			archetypeBlockSlots, registrySlotSize, archetypeBlockSize)
	}
}

// TestMasqueIndexeSurLEntreeDuRegistre fige le CADRE D INDEX : le bit `k` du masque de presence
// designe l entree `k` du bloc du registre du film, sans decalage.
//
// Le test pilote la boucle de PRODUCTION (`traverseComponentLoopFrom`) avec un masque a un seul
// bit et verifie que le composant consomme est bien celui de l index de ce bit. Un port de
// `i - decales` ferait echouer tous les cas sauf `k = 0`.
func TestMasqueIndexeSurLEntreeDuRegistre(t *testing.T) {
	noms := []string{
		compObjectPosition,
		compObjectShieldVitality,
		compObjectDissolver,
		"managed-object-networked-splash-message-dynamic-component",
	}
	reg := parseRegistry(buildRegistry(noms))
	arch, ok := reg.Archetype(0)
	if !ok || len(arch.Components) != len(noms) {
		t.Fatalf("bloc 0 = %v", arch.Components)
	}
	for k := range noms {
		br := LecteurSur(make([]byte, 64))
		tr := EntityTrace{Mask: uint64(1) << uint(k)}
		traverseComponentLoopFrom(br, arch, &tr, 0)
		if len(tr.Comps) != 1 {
			t.Fatalf("bit %d : %d composants consommes, attendu 1 (%+v)", k, len(tr.Comps), tr.Comps)
		}
		if got := tr.Comps[0]; got.Index != k || got.Name != noms[k] {
			t.Errorf("bit %d -> index %d nom %q, attendu index %d nom %q "+
				"(le cadre du masque a DERIVE : cf. FUN_142e2c690)",
				k, got.Index, got.Name, k, noms[k])
		}
	}
}

// TestNiveauVientDeLEntreeDuRegistre fige la seconde moitie de `FUN_1428e1b50` : le niveau de
// precision qu un deserialiseur recoit est celui de l ENTREE DU REGISTRE DU FILM
// (`entree + 0x100`), pas celui du descripteur du processus. C est ce que
// `FUN_142e2c690(param_4 + 0x100)` passe a `vtable[0x28]`, et ce que [Archetype.Level] rend.
func TestNiveauVientDeLEntreeDuRegistre(t *testing.T) {
	noms := []string{compObjectPosition, compObjectDissolver}
	data := buildRegistry(noms)
	// Le niveau de l entree 1 du bloc 0, ecrit la ou l ecrivain le lit.
	off := registryEntryBase + registrySlotSize + registryEntryLevelOffset
	data[off], data[off+1], data[off+2], data[off+3] = 7, 0, 0, 0
	arch, ok := parseRegistry(data).Archetype(0)
	if !ok {
		t.Fatal("bloc 0 absent")
	}
	if got := arch.Level(1); got != 7 {
		t.Errorf("niveau de l entree 1 = %d, attendu 7 (lu en entree + %#x)",
			got, registryEntryLevelOffset)
	}
	// Hors bornes, `FUN_1428e1b50` rend 1 (son `uVar5` initial) parce que le nom cherche n est
	// dans AUCUNE entree. Hors ligne ce cas n existe pas : on itere les entrees elles-memes, donc
	// [Archetype.Level] rend 0 pour un index hors liste, et aucun appelant ne l atteint.
	if got := arch.Level(len(noms)); got != 0 {
		t.Errorf("niveau hors liste = %d, attendu 0", got)
	}
}
