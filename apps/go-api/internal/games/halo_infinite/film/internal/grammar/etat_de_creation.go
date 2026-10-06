package grammar

// etat_de_creation.go — UN RECORD NEW DONT LE JEU NE SAIT PAS LIRE L ETAT DE CREATION N EST PAS LU.
//
// Ghidra, HaloInfinite.exe (base 0x140000000), relu le 2026-10-05 :
//
//	FUN_1408f1aa4   le lecteur de record NEW, appele par `FUN_1406cbaa0` @1406cc252 (la branche
//	                que le jeu emprunte en rejouant un film, cf. [World.VuePossede]). Il appelle
//	                le lecteur d etat de creation de
//	                l archetype (`CALL [RAX+0x60]` @1408f1c0f, cinquieme argument 1) ;
//	                `TEST AL,AL ; JZ 0x1408f210e` @1408f1c12 : sur un echec, il ne lit NI
//	                `vtable+0x88`, NI la boucle de composants (`FUN_14076cb60`), et ne cree pas
//	                l entite (`FUN_1408f2150`). Le corps du record n est pas lu.
//	FUN_14080cfe8   le bloc `object-multiplayer-properties` ([consumeMultiplayerPropertiesBlock]).
//	                Le compte R(3) : `CMP ECX,0x4 ; JA 0x14080d319` @14080d238, ou
//	                `XOR SIL,SIL` pose l echec ; le lecteur lit quand meme la suite
//	                (`FUN_14080d4d0`, la queue G3), puis `TEST SIL,SIL ; JZ` @14080d327 rend 0.
//
// Les lecteurs d etat qui lisent ce bloc, et ce qu ils font de son echec :
//
//	FUN_140f44c38  bipede (ti=35)           rend 0                   [consumeBipedDefaultState]
//	FUN_1407f2224  ti=36, et ti=37 par      rend 0 ; FUN_1407f105c   [consumeDefaultStateTI36]
//	               FUN_1407f105c            rend 0 a son tour        [consumeDefaultStateTI37]
//	FUN_140fe7630  ti=43                    rend 0                   (meme port que ti=36)
//	FUN_1407f0c68  ti=42, par FUN_1407f2224 rend 0, apres avoir lu   [consumeDefaultStateTI42]
//	               (arme au sol)            ses autres feuilles
//	FUN_1408f0b48  ti=38 et ti=39           rend 0                   [consumeDefaultStateTI38]
//	FUN_1410a5a74  vehicule (ti=40)         rend 0                   [consumeDefaultStateTI40]
//	FUN_1408efb58  projectile (ti=41)       rend 0 sauf drapeau 2    [consumeDefaultStateTI41]
//	                                        et index absent
//
// `FUN_1407f0c68` (ti=42), decompile le 2026-10-06 : `cVar1 = FUN_1407f2224(...)`, puis toutes
// ses feuilles, puis `if (*(lecteur+0x18)*8 < *(lecteur+0x2c) || cVar1 == 0) -> 0`. Le port y
// applique la regle par [consumeDefaultStateTI36], qu il appelle pour `FUN_1407f2224`.
//
// D ou la regle, LUE DANS LE JEU et non mesuree : un record NEW dont le bloc MPP annonce un compte
// superieur a quatre (sous la reserve de `ti=41`) n est pas lisible par le jeu — son corps ne
// serait pas lu et l entite ne serait pas creee. Le film est le flux que le jeu relit : une lecture
// qui y trouve un tel record ne lit pas un record ecrit la. [TraverseEntity] arrete donc le record
// a la fin de son etat ([EntityTrace.EtatIllisible]), et le juge de l ecrivain la contredit
// ([InvariantEtatDeCreationIllisible]).

// mppCompteMax est le plus grand compte que `FUN_14080cfe8` accepte : `CMP ECX,0x4 ; JA`
// @14080d238. Au-dela, le lecteur echoue.
const mppCompteMax = 4

// lireLeBlocMPPDeLEtat lit le bloc MPP d un etat de creation dont le lecteur rend 0 quand le bloc
// echoue, et fait echouer l etat. `ti=41` ne passe pas par ici : son verdict a une exception
// ([consumeDefaultStateTI41]).
func lireLeBlocMPPDeLEtat(br *Lecteur) {
	if !consumeMultiplayerPropertiesBlock(br) {
		br.echouerLEtatDeCreation()
	}
}
