# Halo 5 — Précision par arme par joueur : VALIDÉ reconstructible

## Question
A-t-on l'arme par kill et la précision par arme par joueur ?

## Réponse
- **Arme par kill** : OUI, natif (`Death.KillerWeaponStockId`), **déjà ingéré** dans
  `weapon_kills` (~268k lignes shared H5). Le sujet « abandonné faute de données » =
  c'était **Halo Infinite** (décodage film). H5 le donne nativement.
- **Précision par arme par joueur** : le carnage `WeaponStats[]` (arsenal complet) est
  servi **VIDE** → PAS directement disponible. MAIS **reconstructible à 100 %** depuis les
  events `WeaponDrop` (`ShotsFired`/`ShotsLanded` par instance d'arme).

## Validation (sonde, match CTF 8 joueurs, même match events+carnage)
Somme des `WeaponDrop.ShotsFired/ShotsLanded` par joueur **vs** `carnage.TotalShotsFired/
TotalShotsLanded` :

| joueur | events (F/L) | carnage (F/L) | |
|---|---|---|---|
| KNeow1 | 725/182 | 725/182 | EXACT |
| MsTada87 | 79/34 | 79/34 | EXACT |
| Chocoboflor | 320/71 | 320/71 | EXACT |
| JGtm | 368/121 | 368/121 | EXACT |
| Pancakeflips | 157/62 | 157/62 | EXACT |
| Madman684844 | 307/139 | 307/139 | EXACT |
| Treitor121 | 209/71 | 209/71 | EXACT |
| Madina97294 | 179/86 | 179/86 | EXACT |

**8/8 EXACT** → aucune fuite (les armes tenues à la mort/fin de partie émettent bien un
WeaponDrop final). La reconstruction est complète, pas approximative.

Détail par arme (KNeow1) : arme `2278207101` = 232 tirs / 76 touchés = **33 %** — identique
au `WeaponWithMostKills` du carnage (cohérence croisée). arme `2140505068` = 448/85 = 19 %.

## Ce qu'on obtient (par joueur, par arme, par match)
- **tirs / touchés / précision** (WeaponDrop, validé)
- **kills** (Death.KillerWeaponStockId, déjà ingéré)
- **temps actif** (`WeaponDrop.TimeWeaponActiveAsPrimary`)
- nom résolu via `weapon_labels` H5 (déjà seedé).
→ proficience complète par arme (précision + létalité + usage).

## État & action
Aujourd'hui **non ingéré** : le DTO `WeaponDrop` (`events_dto.go`) ne modélise que
`WeaponStockId`+`Player` (shots jetés). Pour capturer : étendre le DTO WeaponDrop
(ShotsFired/ShotsLanded/TimeWeaponActiveAsPrimary) + agréger par (xuid, weapon_stock_id)
au match, persister (table append-only type `weapon_accuracy` ou colonnes sur un agrégat
arme existant). Combiner avec `weapon_kills` pour la proficience.
