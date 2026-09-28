package grammar

// default_state_ti13_neuf.go — L ETAT PAR DEFAUT DE ti=13 DANS UN RECORD NEW DE TRAME DELTA : UNE
// EXCEPTION DATEE (lot J6-bis du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, 2026-09-28).
//
// # CE QUE LE JEU LIT
//
// `FUN_140ce55e8` (vtable 0x1437002a0, emplacement 0x60), porte par [consumeDefaultStateTI13] :
// la version, « propertyName » R(32), g = R(1), puis 1 variant en mode A (g = 0) ou 32 en mode B
// (g = 1), chacun `FUN_140ce59bc` = l etiquette R(4) PUIS la charge de `FUN_140ce5aa4` (GA2-4, lot
// J6.3). Le lecteur de record NEW (`FUN_1408f1aa4`) appelle le meme emplacement que l image-cle.
// Dans les images-cles, cette lecture ferme ti=13 de 0 % a ~100 % sur les sept bobines : elle y
// reste.
//
// # CE QUE LE FLUX DIT DES RECORDS NEW
//
// Dans les trames delta, la lecture du jeu fait BAISSER la fermeture des listes d evenements
// (marche des trames, entrees figees du 2026-09-28) : `fb1a1a72` chunk 24 paquet 426 (7 entrees de
// controle) et chunk 34 paquet 568 (1), `60ae07c4` chunk 10 paquet 952 (0) et chunk 33 paquet 800
// (1), toutes fermees par la lecture d AVANT (l etiquette R(4) seule, puis porte, masque et
// composants) et non localisees par celle du jeu. Elle en fait monter d autres sur `60ae07c4`
// (chunks 17, 23, 27, 28, 33 et 40). Aucun modele essaye ne ferme les deux familles : porte et
// masque omis apres l etat par defaut (ce que dit `vtable[0x30]` de ti=13, un masque vide, sauf
// mode d execution `FUN_1404f2b4c`), porte seule, ou porte et masque. La grammaire du record NEW de
// ti=13 n est donc pas etablie ; comme les sites de `FUN_14076e524` du meme cas
// (`lecteur_position_exceptions.go`), il garde sa lecture d AVANT le lot J6.3, et le CRITERE DE
// RETRAIT est le meme : la lecture du jeu fait monter la fermeture sans aucune baisse sur les
// bobines, ou la grammaire dependante du build est etablie.
//
// # LA PISTE « PORTE SOUS MASQUE » EST REFUTEE (lot J6-ter, 2026-09-28)
//
// Releve Ghidra (lecture seule) : apres `vtable[0x60]` et `vtable[0x88]`, `FUN_1408f1aa4` appelle
// `vtable[0x30](this, &autorises)` (@1408f1c4b, un masque de 256 bits), puis `FUN_1404f2b4c`
// (@1408f1c60 : vrai quand le mode de partie vaut 2, le rejeu). La porte `FUN_1406cf008` n est lue
// d office que si ce mode ET `DAT_144c232e1` sont vrais (@1422780e7) ; sinon elle n est lue que si
// `autorises` est non vide, et une porte a 0 saute `FUN_14076cb60`, donc le MASQUE aussi (le masque
// est la premiere lecture de `FUN_14076cb60`, `FUN_1406d7610`). Le lecteur d image-cle
// `FUN_141f86704` porte la meme regle. `vtable[0x30]` vaut `0x1408f2130` (masque VIDE) pour
// ti = 7, 8, 13, 14, 20, 23, 28, 29, 31, 46, 47 et 49 ; tous les autres archetypes rendent un
// masque non vide. Le Go lit toujours porte puis masque.
//
// Mesure (carte de fermeture, `cmd_fermeture`, huit films des fixtures, paquets fermes / listes
// d evenements fermees), la regle du jeu appliquee a TOUS les archetypes avec `DAT_144c232e1` nul
// (NOTE 5.18 §4) et la lecture du jeu pour ti=13 : 000d5950 9028 -> 9033, fb1a1a72 22320 -> 22321,
// 111fa685 4153 -> 4170, a521164d 701 -> 704, bcb6d393 5835 -> 6031, mais 11de8353 5741 -> 5722
// (listes 1337 -> 1322), e5adf7b2 4288 -> 4239 (1329 -> 1301), 60ae07c4 13971 -> 13894 (916 -> 890).
// Trois des quatre paquets de ce fichier restent non localises (seul fb1a1a72 24:426 ferme) et
// AUCUNE des six fermetures abandonnees de 60ae07c4 (23:1622, 27:2316, 28:652, 28:654, 33:2238,
// 40:990) ne revient ; la lecture du jeu SANS la regle les ferme toutes : le flux porte donc une
// porte et un masque apres l etat par defaut de ti=13 dans ces paquets. Les deux moities de la
// regle baissent chacune quelque part : masque vide seul, 111fa685 -10, e5adf7b2 -16 ; porte a 0
// sans masque seule, 11de8353 -16, e5adf7b2 -40, 60ae07c4 -71. Par archetype (masque vide, un a
// la fois), aucun gain sans baisse hors ti=23 (+1 sur 11de8353 et e5adf7b2) et ti=31 (+1 sur
// 000d5950 et 60ae07c4) ; ti=47 porte le +194 de bcb6d393 et -7 sur e5adf7b2. Le flux contredit
// donc `DAT_144c232e1 == 0` en rejeu, ou ce lecteur n est pas celui de ces records : non tranche.
// Les trois exceptions de site du lot J6-bis ne tombent pas non plus sous la regle (11de8353 9:1146,
// 19:494, 21:1032 et e5adf7b2 25:344 restent non localises).

// consumeDefaultStateTI13RecordNeuf lit l etat par defaut de ti=13 d un record NEW avec la lecture
// d AVANT le lot J6.3 : la version, « propertyName », g, puis 1 ou 32 etiquettes R(4) sans charge.
// EXCEPTION DATEE (2026-09-28) : cf. l en-tete du fichier.
func consumeDefaultStateTI13RecordNeuf(br *Lecteur) {
	consumeVersionPrefix(br)
	br.ReadBits(32) // FUN_14080dec4 "propertyName"
	n := 1
	if br.ReadBit() {
		n = managedPropertyPlayerCount
	}
	for i := 0; i < n; i++ {
		br.ReadBits(managedPropertyTagBits) // l etiquette seule
	}
}

// archetypeProprieteGeree est l archetype 13, « managed-object-property-name ».
const archetypeProprieteGeree = 13
