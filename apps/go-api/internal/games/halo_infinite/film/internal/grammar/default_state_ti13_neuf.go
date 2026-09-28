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
