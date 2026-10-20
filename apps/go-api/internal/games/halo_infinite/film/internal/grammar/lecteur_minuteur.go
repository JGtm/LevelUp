package grammar

// lecteur_minuteur.go — LE LECTEUR DE MINUTEUR DU JEU, EN UN SEUL EXEMPLAIRE.
//
// Lu dans HaloInfinite.exe (HI_1_13_0, base 0x140000000), lecteur et ecrivain :
//
//	FUN_140d580d0(dest, flux, n, max)   FUN_1406d84b4 x 2 (reel quantifie sur n bits, [0, max],
//	                                    extremites exactes) -> dest+0, dest+4 ; puis FUN_1407f0354
//	                                    (`+0x2c += 5`) -> dest+0xb. Largeur 2n + 5.
//	FUN_142b6f76c                       son ecrivain : FUN_1406d22c0 x 2 sur n bits, puis l octet
//	                                    dest+0xb (FUN_142af2af0).
//	FUN_142ba78dc(dest, flux, n, max)   FUN_140d580d0 puis un troisieme FUN_1406d84b4 sur n bits
//	                                    -> dest+0xc. Largeur 3n + 5. Ecrivain : FUN_142ba7c74
//	                                    (FUN_142b6f76c puis FUN_1406d22c0 sur n bits).
//
// Les appelants du jeu passent n et max : `ti=5 i2` (FUN_140d580a8, n = 5), `ti=0 i5`
// (FUN_1407ee790), `i6` (FUN_14116d3a4), `i7` (FUN_141165d24), `i12` (FUN_1407ee764) et les fentes
// du bassin `i15` (FUN_1407ee87c), tous a n = 16 ; les deux entrees de `ti=43 i37`
// (FUN_142f02c94 -> FUN_142ba78dc), a n = 10. Le max ne change aucun bit lu : la valeur reelle
// reste a l appelant (`DequantEndpoint`) : [lireMinuteur140d580d0] rend les quanta bruts.
//
// GARDE-RAIL (regle des deux copies) : `lecteur_minuteur_guard_test.go` interdit, hors de ce
// fichier et dans la production du paquet, la sequence de lectures de FUN_140d580d0 ecrite en
// ligne (cherchee dans l arbre syntaxique, quelle que soit la mise en page), les sauts litteraux
// `Skip(37)` / `Skip(53)` et tout `Skip` dont l argument nomme une largeur de minuteur
// (`largeurQueueMinuteur`, `roundTimerBits`, `largeurMinuteurSoftKill`,
// `largeurMinuteurDistributeur`). Il ne verifie pas qui
// cite FUN_140d580d0 dans un commentaire.

// largeurQueueMinuteur : la queue de FUN_1407f0354 (`+0x2c += 5`).
const largeurQueueMinuteur uint = 5

// Minuteur porte les quanta bruts lus par FUN_140d580d0.
type Minuteur struct {
	A, B  uint64 // les deux reels quantifies sur n bits, dans l ordre du flux
	Queue uint64 // l octet de FUN_1407f0354, R(5)
}

// lireMinuteur140d580d0 lit FUN_140d580d0 sur n bits : R(n) + R(n) + R(5).
func lireMinuteur140d580d0(br *Lecteur, n uint) Minuteur {
	a := br.ReadBits(n)
	b := br.ReadBits(n)
	return Minuteur{A: a, B: b, Queue: br.ReadBits(largeurQueueMinuteur)}
}

// lireMinuteur142ba78dc lit FUN_142ba78dc sur n bits : FUN_140d580d0 puis R(n). Aucun appelant
// n en lit les quanta (ni la fente `i15` ni `ti=43 i37` ne publient rien) : il consomme ses
// 3n + 5 bits sans rien rendre.
func lireMinuteur142ba78dc(br *Lecteur, n uint) {
	lireMinuteur140d580d0(br, n)
	br.ReadBits(n) // le troisieme reel, dest+0xc
}
