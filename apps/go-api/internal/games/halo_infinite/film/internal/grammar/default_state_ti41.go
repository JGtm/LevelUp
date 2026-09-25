package grammar

// default_state_ti41.go — L ETAT PAR DEFAUT DU PROJECTILE (`ti=41`), LU CHEZ L ECRIVAIN (lot M4b de
// la campagne « retours rejeu », 2026-09-25).
//
// # POURQUOI IL EST PORTE ICI
//
// Un record NEW de projectile (grenade, roquette, trait du Ghost ou de la Banshee) passait par le
// repli « 0 bit » de `default_state_arch.go` (population (b) : `FUN_1408EFB58` non resolue) : sa
// traversee partait trop tot, le masque et la boucle de composants lisaient l etat par defaut,
// et la liste du paquet se perdait sur le record suivant. Un tir de vehicule a projectiles cree
// un projectile par coup : pendant qu un Ghost tire, chaque paquet ou nait un trait perdait sa vue
// B, donc sa vue C — le canal du tir continu. Mesure sur `8a485699` (Ghost de l index 4) : les
// paquets 902-926 ouvrent sur `NEW <projectile>` puis un faux `NEW 4096/35`.
//
// # LA GRAMMAIRE (Ghidra, HaloInfinite.exe, base 0x140000000)
//
// `vtable[0x60]` de l archetype = `FUN_1408efb58(desc, taille, dst, lecteur, param_5)`. Le lecteur
// de record NEW (`FUN_1408f1aa4`) l appelle avec `param_5 = 1` (`MOV byte [RSP+0x20],0x1`
// @1408f1bfb) ; le lecteur d etat complet d image-cle (`FUN_142e2bfd0`) avec `param_5 = 0`
// (@142e2c476, sous `DAT_144e61ea0 = 1`). `param_5` choisit la forme des deux lecteurs de
// reference ci-dessous : les deux formes sont portees, et seule celle du record NEW est branchee
// (la marche d image-cle garde son cadre, cf. [consumeKeyframeDefaultState]).
//
//	V                          R(1) ; si 1 : R(8) la version (1 sinon)          @1408efb84
//	FUN_14080cfe8              le bloc MPP ([consumeMultiplayerPropertiesBlock])  @1408efbd4
//	R(1) ; si 1 : R(5)         -> dst+0x70                                        @1408efbe2
//	FUN_1404785a0, FUN_1408ee96c   0 bit (predicats sur dst+0x14)
//	R(1) ; si 1 :              drapeau 2
//	   FUN_1408f0ac4(dst+0x64, lecteur, 1)   categorie 1 (`MOV R8D,EBP`, EBP = 1) @1408efcb1
//	   FUN_1406d00ec           R(1) ; si 0 : R(2)                                 @1408efcb9
//	FUN_1408eff64(dst+0x74, lecteur, param_5)   [consume1408eff64]                @1408efd15
//	R(1)                       drapeau 4                                          @1408efd1d
//	R(1) ; si 1 : 2 x FUN_1406d84b4 largeur 5 (`MOV [RSP+0x20],R13D`, R13D = 5) @1408efd5d/7c
//	R(5)                       -> dst+0x90 (R13D)                                 @1408efdb0
//	R(1) ; si 1 :              drapeau 8
//	   FUN_14076e494(lecteur, dst+0x94, 0x10, 0, param_5, 0)   la position de niveau 16
//	   FUN_14076dc04           R(19) (`MOV R9D,0x13`)                             @1408efe26
//	   FUN_1406d84b4           largeur 0xc                                        @1408efe4b
//	R(1) ; si 1 : FUN_1408f0ac4(dst+0xb0, lecteur, 0)   categorie 0               @1408efe82
//	R(1) ; si 1 : c = R(1) ; FUN_142f04664(dst+0xb8, lecteur, c, param_5)        @1408efebc
//	si version > 2 : R(1)      drapeau 0x40
//	R(1) ; si 1 : FUN_141fcf730(lecteur)                                          @1408eff03

// consumeDefaultStateTI41 lit `FUN_1408efb58` ; `param5` est le cinquieme argument du jeu.
func consumeDefaultStateTI41(br *Lecteur, param5 bool) {
	version := uint64(1)
	if br.ReadBit() {
		version = br.ReadBits(8)
	}
	consumeMultiplayerPropertiesBlock(br)
	consumeGateR(br, 5) // -> dst+0x70
	if br.ReadBit() {   // drapeau 2
		consume1408f0ac4(br, 1)
		consumeID2(br) // FUN_1406d00ec
	}
	consume1408eff64(br, param5)
	br.ReadBit()      // drapeau 4
	if br.ReadBit() { // dst+0x88, dst+0x8c
		br.ReadBits(largeurEchelleProjectile)
		br.ReadBits(largeurEchelleProjectile)
	}
	br.ReadBits(largeurEchelleProjectile) // dst+0x90
	if br.ReadBit() {                     // drapeau 8
		consumeSimStateHandleTail(br) // FUN_14076e494(..., 0x10, 0, param_5, 0)
		br.ReadBits(largeurVecteurDirection)
		br.ReadBits(largeurVitesseProjectile)
	}
	if br.ReadBit() { // drapeau 0x10
		consume1408f0ac4(br, 0)
	}
	if br.ReadBit() { // drapeau 0x20
		consume142f04664(br, br.ReadBit())
	}
	if version > versionDrapeau40Projectile {
		br.ReadBit() // drapeau 0x40
	}
	if br.ReadBit() {
		consume141fcf730(br)
	}
}

// Largeurs de `FUN_1408efb58`, lues au desassemblage (cf. l en-tete).
const (
	// largeurEchelleProjectile : R13D = 5, pousse avant les deux `FUN_1406d84b4` et lu inline.
	largeurEchelleProjectile = 5
	// largeurVitesseProjectile : `MOV [RSP+0x20],0xc` @1408efe43.
	largeurVitesseProjectile = 12
	// versionDrapeau40Projectile : le drapeau 0x40 n est lu que si la version depasse 2.
	versionDrapeau40Projectile = 2
)

// consume1408eff64 lit `FUN_1408eff64(dst, lecteur, p)` : R(1) ; si 1 : `FUN_1407f0278` = R(2)
// genre ; genre 1 : [p ? entier a largeur variable en categorie 1 (`LEA R8D,[RSI-0x3f]`, RSI =
// 0x40, @14234fe1c) : R(32)] puis R(1)[R(6)] ; genre 2 : [p ? entier en categorie 2 : R(32)] ;
// genres 0 et 3 : rien. Avec `p` vrai, c est [consume140c9e990] derriere sa porte.
func consume1408eff64(br *Lecteur, p bool) {
	if !br.ReadBit() {
		return
	}
	if p {
		consume140c9e990(br)
		return
	}
	switch br.ReadBits(2) {
	case genreCibleCategorie1:
		br.ReadBits(32)
		consumeGateR(br, 6)
	case genreCibleCategorie2:
		br.ReadBits(32)
	}
}

// consume142f04664 lit `FUN_142f04664(dst, lecteur, c, p)` : c nul -> `FUN_14076e494(lecteur,
// dst, 0x10, 0, p, 0)` (la position de niveau 16) ; sinon R(2), `FUN_140c1e924` = trois R(13)
// (`MOV R9D,0xd` @142f04745, boucle de `FUN_140c1e9d4`), puis R(1)[R(16)].
func consume142f04664(br *Lecteur, c bool) {
	if !c {
		consumeSimStateHandleTail(br)
		return
	}
	br.ReadBits(2)
	for i := 0; i < 3; i++ {
		br.ReadBits(largeurAxe142f04664)
	}
	consumeGateR(br, 16)
}

// largeurAxe142f04664 : la largeur des trois axes de `FUN_140c1e9d4` (`MOV R9D,0xd`).
const largeurAxe142f04664 = 13

// consume141fcf730 lit `FUN_141fcf730` : `FUN_1407f2058` (R(1) ; si 0 : R(5)), `FUN_141fcf670`
// (R(7) puis R(1)), puis R(4).
func consume141fcf730(br *Lecteur) {
	consumeGate0R(br, 5)
	br.ReadBits(7)
	br.ReadBit()
	br.ReadBits(4)
}
