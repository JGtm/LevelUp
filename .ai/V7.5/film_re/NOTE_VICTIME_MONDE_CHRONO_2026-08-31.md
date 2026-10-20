# damage_aftermath : BLESSE vs RESPONSABLE, et le monde chronologique

Date : 2026-08-31. Branche `wt/trame-recherche`. Deux questions tranchees sur pieces
(2-3 films temoins, seuils ecrits AVANT la mesure). Le record vise est `damage_aftermath`
(paquet `0xC0`, type 0 — le vrai enregistrement de degat, cf.
`NOTE_MODELE_EVENEMENTS_2026-08-30.md`). Il porte, en tete, DEUX references du domaine 1
(descripteur `0x1451f98d0`) qui resolvent toutes deux en slots de bipede (joueurs) :
appelons-les **ref0** (la premiere, toujours presente) et **ref1** (la seconde, parfois
absente).

Instruments (garde `LOT1_TRAME_FILM`, un film par process, lecture seule) :
`internal/analysis/filmdec/lot1_degats_blesse_research_test.go` et
`internal/analysis/filmdec/lot1_monde_chrono_research_test.go`. Films : `000d5950`,
`01e1f945`, `00502e52`. Fenetre de mesure : 12 premiers chunks de replication.

---

## A. ref0 est le BLESSE (touche) ; ref1 est le RESPONSABLE (attaquant)

**Juge (physique du degat)** : autour de l'instant d'un degat, la VITALITE (sante i4 +
bouclier i5, decodees offline par `ScanFilmBipedPositions`/CaptureDirs) du BLESSE **baisse** ;
celle de l'ATTAQUANT ne bouge pas — frapper ne coute rien. Pour chaque evenement horodate T,
on lit la vitalite du slot resolu juste avant et juste apres T.

**Seuils ecrits avant la mesure** : fenetre W = 500 ms (avant = derniere mesure `ts < T` ;
apres = premiere `ts >= T`, chacune dans `[T-W, T+W]`) ; baisse = delta <= -0,05 (5 % d'une
barre) ; le bouclier et la sante sont sommes (le bouclier absorbe le premier degat, il porte
donc l'essentiel du signal : ~11 000-18 000 mesures de bouclier par film contre ~300-440 de
sante).

**Calibration de la base** : la reference du domaine 1 se resout `slot = base + index`. La
bande bipede etant CONTIGUE, le critere de `victime_slot` (« base+index tombe sur un bipede
LIE ») ne departage pas les bases voisines (510 vs 512 sont toutes deux « en bande »). On
retient donc la base qui resout le PLUS d'evenements a des slots PORTEURS DE VITALITE
(echantillon max) — critere qui, lui, discrimine (les bipedes vivants forment un ensemble
creux). Resultat : **base 512 sur les trois films** (couverture 130 / 25 / 191 contre <=56
aux bases voisines).

**Mesures a la base calibree 512** :

| Film | evts | ref0 baisse (n) | ref1 baisse (n) | tete-a-tete ref0/ref1/ambigu | conclusion |
|---|---|---|---|---|---|
| `000d5950` | 190 | **94,8 %** (115) | 13,3 % (15) | 11 / 0 / 2 | ref0 = blesse |
| `01e1f945` | 44 | **70,6 %** (17) | 37,5 % (8) | 2 / 0 / 3 | ref0 = blesse |
| `00502e52` | 150 | **92,4 %** (131) | 1,7 % (60) | 50 / 0 / 4 | ref0 = blesse |

Sur les trois films, la vitalite du slot de **ref0** chute massivement autour du degat ;
celle de **ref1** reste proche du bruit de fond (1,7-37,5 %). En tete-a-tete (les deux refs
resolues et mesurables au meme evenement), **ref1 ne l'emporte JAMAIS** : c'est toujours ref0
qui baisse le plus (63 gagnants ref0, 0 ref1, 9 ambigus toutes mesures confondues). Les
ambigus sont des cas ou les deux deltas sont petits (bouclier en cours de regen des deux
cotes) — le taux de baisse ABSOLU est le signal net.

**Corollaire coherent** : ref0 est presente sur 100 % des degats (190/190, 150/150) — tout
degat a une victime ; ref1 manque parfois (128/190) — c'est le comportement d'un ATTAQUANT
(absent des degats d'environnement / de chute, ou source non-bipede).

**CONCLUSION A** : dans `damage_aftermath`, **la premiere reference d'en-tete (ref0) est le
BLESSE ; la seconde (ref1) est le RESPONSABLE**. Chiffre porteur : part des evenements ou le
slot de ref0 perd de la vitalite = 94,8 % / 70,6 % / 92,4 % (contre 13,3 / 37,5 / 1,7 % pour
ref1).

**Reserve honnete (piste iii non corroboree)** : le temoin « soin » (magnitude negative,
`Kscale=-1` dans `lot1DecodeDamageAftermath`) devait faire MONTER la vitalite du slot resolu.
Mesure : 0 % de hausse sur 18 / 5 / 27 evenements negatifs — au contraire ces evenements
ressemblent a des degats ordinaires sur ref0. Le drapeau de magnitude negative NE coincide
donc PAS avec un soin auto-cible sur ces films : a ne pas utiliser comme marqueur de soin
sans RE supplementaire.

---

## B. Le monde chronologique n'ameliore PAS la resolution (plafond structurel, pas temporel)

**Hypothese testee** : la resolution ref -> slot -> bipede utilise aujourd'hui l'etat du
monde en FIN de chunk (taux 82-89 %). Un bipede mort en cours de chunk (DEL avant la fin)
serait alors deja delie ; reconstruire le monde CHRONOLOGIQUE (seuls les tick-frames
ANTERIEURS a l'evenement) devrait faire remonter le taux.

**Methode** : passe unique en ordre de paquets ; le monde avance tick-frame par tick-frame ;
a chaque `0xC0` on resout contre le monde A CET INSTANT (APRES). Une fois TOUS les tick-frames
appliques, le meme monde vaut l'etat fin-de-chunk et on re-resout les memes evenements
(AVANT) — meme base, seul l'instant du monde change. Le AVANT reproduit `victime_slot` au
chiffre pres (validation : `000d5950` ref0 = 82,1 % identique).

**Mesures (taux = base+index lie a un bipede)** :

| Film | ref | base | AVANT (fin de chunk) | APRES (chronologique) | gain |
|---|---|---|---|---|---|
| `000d5950` | ref0 | 512 | 82,1 % (156/190) | 82,6 % (157/190) | +0,5 |
| `000d5950` | ref1 | 512 | 59,4 % (76/128) | 60,9 % (78/128) | +1,6 |
| `01e1f945` | ref0 | 510 | 52,3 % (23/44) | 52,3 % (23/44) | +0,0 |
| `01e1f945` | ref0 | 512 | 38,6 % (17/44) | 38,6 % (17/44) | +0,0 |
| `01e1f945` | ref1 | 512 | 75,7 % (28/37) | 75,7 % (28/37) | +0,0 |
| `00502e52` | ref0 | 510 | 75,3 % (113/150) | 75,3 % (113/150) | +0,0 |
| `00502e52` | ref0 | 512 | 70,7 % (106/150) | 71,3 % (107/150) | +0,7 |
| `00502e52` | ref1 | 512 | 89,3 % (134/150) | 91,3 % (137/150) | +2,0 |

**CONCLUSION B** : le gain chronologique est NEGLIGEABLE (0 a +2 pts). Les bipedes sont lies
des l'image-cle de debut de chunk et PERSISTENT ; l'etat fin-de-chunk contient donc deja,
a l'instant de l'evenement, les memes liaisons bipedes. Le plafond de 82-89 % n'est PAS un
probleme de fraicheur temporelle (morts en cours de chunk) — c'est l'ambiguite de la
reference domaine-1 elle-meme (largeur 9 vs 13 bits selon la sonde, generation), que le rejeu
chronologique ne touche pas. Reconstruire le monde jusqu'a l'evenement ne vaut donc pas son
cout ici.

**Fait de bord important** : sur `01e1f945` et `00502e52`, la base a « meilleur AVANT » est
510, PAS 512 — le taux « base+index = bipede lie » est plus HAUT a 510 (bande contigue : 510
+idx tombe aussi sur des bipedes... les MAUVAIS). L'instrument A montre pourtant que c'est a
512 que les vitalites baissent. Autrement dit : **le taux de resolution « bipede lie » est un
proxy FAIBLE de correction** — il recompense des bases voisines qui atterrissent sur des
bipedes voisins. Le vrai juge de correction est la baisse de vitalite (A), pas le comptage de
liaisons. C'est la lecon transverse des deux instruments.

---

## Fichiers

- `apps/go-api/internal/analysis/filmdec/lot1_degats_blesse_research_test.go` (question A)
- `apps/go-api/internal/analysis/filmdec/lot1_monde_chrono_research_test.go` (question B)

Lancer : `LOT1_TRAME_FILM=<dir_film> go test ./internal/analysis/filmdec/ -run
'TestLot1DegatsBlesse|TestLot1MondeChrono' -count=1 -v` (GOCACHE prive conseille).
