# NOTE — Côté VICTIME : le bipède réplique-t-il un « dernier dégât reçu / dernier attaquant » non-fatal ?

Date : 2026-09-01. Branche : `wt/trame-victim`. Périmètre : `internal/analysis/filmdec` (lecture
seule du film + Ghidra lecture seule). Instrument : `victime_degat_recu_research_test.go`
(garde `LOT1_TRAME_FILM`, borné à 12 chunks).

## Question

En jeu, on VOIT qu'un joueur est juste blessé. Le film capture l'état répliqué du bipède
(santé i4, bouclier i5). Existe-t-il, CÔTÉ VICTIME, un champ répliqué « dégât reçu / dernier
attaquant » portant l'INSTIGATEUR d'un coup (y compris explosif), mis à jour à CHAQUE coup et
non seulement à la mort ? Un tel champ attribuerait proprement les touches explosives non-fatales
à leur tireur, tous modes, sans passer par le projectile.

## Réponse : NON. Le bipède ne réplique aucun champ « dernier attaquant » non-fatal.

Le struct existe DANS LE MOTEUR mais c'est une **valeur runtime**, pas un **composant répliqué**.
Il n'est sérialisé au film que comme l'ÉVÉNEMENT `damage_aftermath`, dont le « responsable » pour
un explosif est le PROJECTILE, jamais le tireur. La distinction runtime vs répliqué est le cœur
du résultat.

## Preuve 1 — Ghidra : le struct est un accesseur RUNTIME (reflection/script)

Le moteur porte bien la notion recherchée. Famille de getters, noms lisibles bruts en `.rdata`
(chaînes, PAS des fonctions), référencés comme DATA depuis la table de binding `FUN_140e61df8`
(reflection/script — HUD, télémétrie, médailles) :

| Adresse | Nom du getter | Sémantique |
|---|---|---|
| `0x143c58f78` | `Unit_GetReceivedDamage_WeakDamageOwnerObject` | **INSTIGATEUR** (objet) |
| `0x143c58e10` | `Unit_GetReceivedDamage_WeakDamageOwnerPlayer` | **INSTIGATEUR** (joueur) |
| `0x143c58e40` | `Unit_GetReceivedDamage_WeakDamageSourceObject` | **SOURCE** (arme / projectile) |
| `0x143c58ee8` | `Unit_GetReceivedDamage_TimeStamp` | horodatage du dernier dégât |
| `0x143c58f50` | `Unit_GetReceivedDamage_Normalized` | magnitude normalisée |
| `0x143c58fa8` | `Unit_GetReceivedDamage_Body` | part santé |
| `0x143c58fc8` | `Unit_GetReceivedDamage_Shield` | part bouclier |

C'est EXACTEMENT le struct rêvé : un « dernier dégât » qui **distingue l'OWNER (tireur) de la
SOURCE (projectile)** — donc capable, en théorie, d'attribuer un explosif non-fatal à son tireur.
Le `TimeStamp` trahit un **cache transitoire de RAM** (dernier dégât, écrasé à chaque coup),
interrogé par script — pas un flux répliqué image par image. Ce sont des getters ; leur existence
ne prouve AUCUNE réplication.

## Preuve 2 — Schéma du film : aucun composant du bipède n'est ce struct

L'archétype bipède (ti=35) réplique EXACTEMENT **64 composants** (i0..i63), énumérés depuis le
registre du film lui-même (`ParseRegistryChunk`, chunk_00). `TestVictimeSchemaAucunChampAttaquant` :

- **AUCUN** des 64 noms ne contient `received-damage` / `damage-owner` / `weak-damage` /
  `last-attacker` / `recent-damage` / `aftermath` / `instigator` / `attacker`.
- Les seuls composants liés au dégât sont : **i4** santé (valeur), **i5** bouclier (valeur),
  **i6** régions, **i7** sections, **i11** dead-state.
- i6/i7 ne portent AUCUNE référence d'entité (grammaire relue : i7 = count + `{bit; R7+R16}`
  par section ; i6 = count + `R3`/`R10` par région — des NIVEAUX de dégât, pas un instigateur).
- **i11 dead-state est le SEUL composant du bipède portant une référence d'attaquant** — et
  c'est l'événement de LA MORT (drapeau `mort` + réfs tueur/victime, source du dégât fatal).

Contrôle transverse : un balayage de TOUS les archétypes de la table ECS ne trouve AUCUN composant
`received-damage` / `damage-owner` / `aftermath` nulle part. Le struct runtime n'est répliqué
comme composant sur AUCUNE entité.

## Preuve 3 — Corpus : i11 (l'unique porteur d'attaquant) est un événement de MORT, pas un per-coup

`TestVictimeDeadStateEstMortPasCoup` balaie les records bipèdes per-tick (position + vitalité,
`ScanFilmBipedPositions`/CaptureDirs, `MaskBits`) et compte la présence de chaque composant-dégât
dans leur masque, contre le nombre de touches (événements `damage_aftermath` 0xC0 type 0 = oracle
du « coup »). 12 chunks par film.

| Film (mode) | records per-tick | i5 bouclier | i4 santé | i11 dead-state | touches (0xC0 t0) | verdict |
|---|---|---|---|---|---|---|
| 000d5950 (Fiesta) | 77 858 | 14,90 % | 0,48 % | **0** (0,00 %) | 190 | NÉGATIF CONFIRMÉ |
| 01e1f945 | 70 508 | 25,51 % | 0,62 % | **0** (0,00 %) | 44 | NÉGATIF CONFIRMÉ |
| 00502e52 | 73 228 | 15,31 % | 0,41 % | **1** (0,00 %) | 150 | NÉGATIF CONFIRMÉ |

Lecture : la VITALITÉ **est** répliquée per-tick (bouclier i5 sur 15–25 % des records), donc le
canal « état du bipède au fil de l'eau » existe et fonctionne — mais le dead-state i11, unique
porteur d'un instigateur, est **quasi absent** des records per-tick (0, 0, 1) alors que les touches
sont nombreuses. Le seul i11 croisé (00502e52) est une mort captée dans un record encore porteur de
position. Vérité terrain double :

- **(a) coup direct** : la présence d'un attaquant per-coup, si elle existait, se lirait comme un
  composant présent à la fréquence des touches. i11 n'y est jamais → aucun champ per-coup.
- **(b) mort** : i11 reste l'événement de mort, déjà exploité par `killsource` (tueur +0x08, 97,6 %).

## Conséquence pour l'attribution non-fatale explosive

Le pendant NON-FATAL du dead-state **n'existe pas côté victime dans le record répliqué**. Le film
ne porte l'attaquant non-fatal que par l'ÉVÉNEMENT `damage_aftermath` — dont le « responsable »
(ref1 dom1), pour un explosif, résout vers le PROJECTILE et non le tireur (mesuré :
`explo_touches_research_test.go`). L'OWNER (tireur) distinct de la SOURCE (projectile) du struct
runtime **n'est pas sérialisé** : la seule ref d'en-tête de l'événement est l'entité qui inflige
le dégât (le projectile), pas son propriétaire.

Cette note **confirme et complète** deux résultats antérieurs, convergents :
- `projectile_owner` (commit `24d192ba5`) : le lien projectile→tireur n'existe QU'À LA MORT ; le
  projectile vivant ne réplique pas son tireur.
- Ici : la VICTIME non plus. Ni le projectile vivant, ni la victime, ne portent l'instigateur
  hors de l'événement de mort.

## Ce qui reste vrai / hors périmètre

- La MORT explosive reste attribuée (i11 dead-state, killsource, 97,6 %).
- L'attribution non-fatale explosive au tireur reste possible SEULEMENT par corrélation externe
  (fenêtre de vol + tir lourd, `explo_touches` M3/M4), pas par un champ propre du bipède.
- Non refait ici : re-mesure de `explo_touches` (état de l'art déjà établi).

## Garde-fou respecté

Le struct `Unit_GetReceivedDamage_WeakDamageOwner*` existe (l'intuition utilisateur est fondée
côté MOTEUR) mais c'est une valeur RUNTIME, pas un composant répliqué. Rien n'est survendu : le
négatif est prouvé sur pièces (schéma déterministe + 3 films tous modes) et distingue explicitement
« composant répliqué » de « valeur runtime ».
