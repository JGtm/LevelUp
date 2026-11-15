# Constats A2.A (biped-spartan-ability, ti=35 composant 57, tag 3)

Instruments : memes que A2.T (`C:\t\marche\A2T\mesure.sh`, `compare.sh`, `pq_all.sh`). Avant = arbre apres A2.T (commit 40dc36e9bd ; `A2A/avant` = copie de `A2T/apres`, meme comportement), apres = exe `A2A/div_apres.test.exe`.

## Lecture du jeu relue (Ghidra, lecture seule, FUN_142f262d4 decompile)
`FUN_140f03dfc` ; a = R(1) -> dst[0] ; si a : `FUN_14297ea84` (R(6), range en dst[2]) ; si `dst[2] & 1` : c = R(1) ; c == 0 : `FUN_142f04664(c=0)` ; c != 0 : `FUN_1406d3140` puis, si `dst[2] & 0x10`, `FUN_142f04664(c=1)` ; b = R(1) -> dst[1] ; si b : `FUN_14076e494(.., 0x10, ..)`. Conforme a la note A2 point 2.

## Mesure
- Fermeture (38 films) : aucun film en baisse. HI_1_13_0 821 713 -> 822 612 (+899, 98,79 -> 98,90 %), HI_1_12_0 220 481 -> 220 599 (+118), HI_1_11_0 +46, HI_1_10_0 +159, HI_1_9_0 +30, HI_1_8_0 +126. Plus gros gains : 8b1f15cb +681, 6ceaf3b8 +107, 108a4e4c +82, 3274c610 +58.
- Divergence (10 temoins) : premiers desaccords « ti=35 composant 57 (update) » 599 -> 3 (rejet 543 -> 3, vue C hors cadre 48 -> 0, kind 8 -> 0) ; decalages de frontiere rattrapes 93 -> 1. Fermes nous 404 546 -> 405 267 ; temoin seul 9 571 -> 9 055 ; nous seuls 268 -> 473 (+205, tous 8b1f15cb) ; fermes des deux en desaccord de position 262 -> 265 (+3 : deux « type de l'entite » NEW lu ti=42 / 36 slot 3468 sur des paquets maintenant fermes, un composant 59 a 5 bits d'ecart au paquet 8b1f15cb 31/674 ou les deux lectures ferment).
- 205 paquets 8b1f15cb fermes chez nous, ouverts chez le temoin, premier desaccord sur ce composant (ex. 10/1112 : composant 59 nous 2265, temoin 2198) : fermeture au bit pres chez nous.

## Gate (replay-corpus-gate, base origin/release/v8.1.2, 18 temoins, `A2A/gate_run.log`, `A2A/gate.json`) contre le gate de fin A2.T (`A2T/gate_run.log`)
gains/pertes/changements, fin A2.T -> A2.A : 084a804d 21/10/6 -> 92/24/15 ; 0797ce72 112/30/15 -> 114/36/18 ; 111fa685 33/16/3 -> 86/29/14 ; e5adf7b2 164/36/22 -> 177/42/22 ; 11de8353 5/6/1 -> 45/1/5 ; 50247b26 3/3/1 -> 5/4/2 ; 4f77afc1 0/0/0 (ok) -> 32/7/9 (PERTE, NOUVEAU) ; bf15f7ab 4/0/0 -> 12/0/0 (ok) ; a349fea8, a521164d : memes comptes (holesNotClosing +-2) ; bcb6d393, 60ae07c4 : identiques ligne a ligne. Banc de verite (O-K, O-S, O-T, O-V, V-1 a V-8, P-4) : IDENTIQUE sur les 18 temoins.
Methode : faits A2.T-final (`A2T/gate_work`) contre faits A2.A (`A2A/gate_work`) par `faitsdiff` (canal par canal), detail de fermeture par paquet A2.T-final / A2.A (`gt/run.sh`), attribution de chaque element qui change au paquet qui le porte (`gt/attrib.js`), temoin libfilmshell quand il lit le film (0797ce72, 4f77afc1).
Paquets : ouvert->ferme 0797ce72 35, 084a804d 174, 111fa685 127, 11de8353 30, 4f77afc1 27, e5adf7b2 281, 50247b26 0 ; ferme->ouvert : 0 partout.
Temoin sur les paquets que nous fermons maintenant : 0797ce72 34 sur 35 fermes aussi par le temoin ; 4f77afc1 24 sur 27.
4f77afc1 (nouveau) : AbilityImpulses 1 -> 0 = slot 594 au paquet 20/490 (TS 7676960608) : avant, 8 records puis rejet a i59 ; apres, 31 records, terminateur, vue C ferme au bit (6669/6672) ; le temoin s'arrete sur BadHeader au bit 2319 dans le meme paquet, i57 du slot 594 5 bits plus long que le notre (138 contre 133). 4 creations de pads (armes 2, powerups 2) lues dans des paquets ouverts devenus fermes (32/1072, 33/722, 41/14, 47/400) ; jumpDerived 1804 -> 1803 : l'episode du slot 577 recouvre le paquet 14/1042 devenu ferme.
