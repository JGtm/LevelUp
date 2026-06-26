# Halo 5 — Match Events + Impulses : ce qu'on a / ce qu'on n'a pas

Source des samples : sonde live `cmd/probe-h5` (owner=JGtm, SpartanToken v4 du pool
Infinite) + API Metadata officielle (`www.haloapi.com`, clé Ocp-Apim). Dumps bruts :
`h5_events.json` (983 events), `h5_carnage.json` (social), `h5_carnage_ranked.json`
(Arena classé), `h5_impulses_meta.json` (catalogue 66 entrées).

> Rappel architecture : la donnée H5 vient des endpoints internes
> `spartanstats.svc.halowaypoint.com` (SpartanToken), PAS du portail
> `developer.haloapi.com` (officiel mais dégradé). Les 2 URLs que tu as données
> (Match-Events / Impulses) sont la *doc* de ces surfaces ; la vraie réponse est dans
> le payload live ci-dessous.

---

## 1. Endpoint MATCH EVENTS — `GET /h5/matches/{id}/events`

Timeline native, tableau `GameEvents[]` hétérogène discriminé par `EventName`.
983 events sur le match sondé :

| EventName     | n   |
|---------------|-----|
| WeaponDrop    | 305 |
| WeaponPickup  | 269 |
| Medal         | 126 |
| PlayerSpawn   | 113 |
| Death         | 107 |
| Impulse       | 61  |
| RoundStart    | 1   |
| RoundEnd      | 1   |

Identité = `Gamertag` brut (`Xuid` toujours null en H5). Temps = `TimeSinceStart`
ISO8601 (`PT33.2154416S`).

### Death event — champs RÉELS (complet)
```json
{
  "EventName": "Death",
  "TimeSinceStart": "PT33.2154416S",
  "Killer":  { "Gamertag": "Madman684844", "Xuid": null },
  "Victim":  { "Gamertag": "Madina97294",  "Xuid": null },
  "Assistants": [],                         // <-- gamertags qui ont ASSISTÉ ce kill
  "KillerWeaponStockId": 2650887244,        // arme du kill (déjà ingérée)
  "KillerWeaponAttachmentIds": [2758383128],// variante/skin de l'arme (DROPPÉ)
  "VictimStockId": 0,                       // arme tenue par la victime (DROPPÉ)
  "VictimAttachmentIds": [],
  "KillerAgent": 1,                         // type d'agent tueur : 1=joueur (DROPPÉ)
  "VictimAgent": 1,                         // 0/autre = IA/PNJ (Warzone/PvE)
  "DeathDisposition": 1,                    // 1=kill ennemi ; autres=suicide/trahison (DROPPÉ)
  "IsHeadshot": false, "IsMelee": false, "IsGroundPound": false,
  "IsShoulderBash": false, "IsAssassination": false, "IsWeapon": true,
  "KillerWorldLocation": { "x": 24.9, "y": -42.0, "z": -7.6 },
  "VictimWorldLocation": { "x": 24.7, "y": -41.7, "z": -7.6 }
}
```

### WeaponDrop event — porte la PRÉCISION par arme
```json
{
  "EventName": "WeaponDrop",
  "Player": { "Gamertag": "Madina97294" },
  "WeaponStockId": 2650887244,
  "WeaponAttachmentIds": [2758383128],
  "ShotsFired": 0,                          // <-- tirs/touchés PAR INSTANCE d'arme
  "ShotsLanded": 0,
  "TimeWeaponActiveAsPrimary": "P0DT0H0M8.0500S",
  "TimeSinceStart": "PT33.2224438S"
}
```

### Autres events
- `Medal` : `{ MedalId, Player, TimeSinceStart }`
- `Impulse` : `{ ImpulseId, Player, TimeSinceStart }` (PAS d'amount — voir §2)
- `WeaponPickup` : `{ WeaponStockId, WeaponAttachmentIds, Player }`
- `PlayerSpawn` : `{ Player }`
- `RoundStart` / `RoundEnd` : `{ RoundIndex }`

### Match Events : ce qu'on a / ce qu'on n'a pas
- **DÉJÀ ingéré** : Death→kills (tueur/victime/arme), Medal, position monde, WeaponPickup/Drop (stock id), Impulse id.
- **Présent mais DROPPÉ (récupérable, gratuit, déjà fetché)** :
  - `Assistants[]` par kill → **attribution d'assist au kill** (38 kills assistés nommés dans le sample). Le carnage ne donne que `TotalAssists` scalaire.
  - `WeaponDrop.ShotsFired/ShotsLanded/TimeWeaponActiveAsPrimary` → **précision PAR ARME reconstructible** (119 drops avec tirs>0) — alors que le carnage `WeaponStats[]` est servi VIDE.
  - `KillerWeaponAttachmentIds` → variante/skin d'arme.
  - `KillerAgent`/`VictimAgent` → discriminant **IA vs joueur** (utile Warzone/PvE).
  - `DeathDisposition` → **suicides / trahisons / morts environnement** (tout =1 ici car PvP pur ; varie en Warzone).
  - `VictimStockId` → arme tenue par la victime au moment du kill.

---

## 2. Endpoint IMPULSES — `GET /metadata/h5/metadata/impulses` (catalogue)

**66 entrées** `{ internalName, id (numérique = ImpulseId des events), contentId (UUID = ids du carnage FlexibleStats) }`.
C'est le RÉFÉRENTIEL qui décode les `ImpulseId` de la timeline ET les UUID du carnage.

Exemples notables :
```
233925220   Kills              391249916   PlayerScoreImpulse
2154529634  Assist             3174430457  Perfect Kill
1758813028  Headshot           2982539900  PowerWeaponGrabbed
1408036107  Suicides           978685900   Revived
2944278681  FlagCapturedImpulse 1095759319 Enemy Player Kill
3711392074  Warzone Boss Takedown
2988432574  PvE Rounds Complete  2777384751 PvE Rounds Survived
... + ~40 bonus PvE/Warzone par round (PvEKillBonus_R1..5, etc.)
```

- Un **Impulse event** = `{ ImpulseId, Player, Time }` : marque QUI / QUAND / QUOI,
  **sans montant**. Donc PlayerScoreImpulse dit *quand* un joueur a marqué, pas combien.
- Le **carnage** porte aussi les impulses par joueur dans `FlexibleStats.ImpulseStatCounts`
  (UUID→count, ex. count=17) et `ImpulseTimelapses` (UUID→1er/dernier timing).
- `Impulses[]` top-level du carnage = **VIDE** (8/8) — la donnée vit dans `FlexibleStats`.

### Impulses : ce qu'on en tire
Objectifs/jalons par mode SANS film : flag caps, power weapon grabs, revives, perfect
kills, prises de base Warzone, boss takedowns, progression PvE par round. Tout en
timeline (events) + agrégé (carnage FlexibleStats). **Aucun n'est exploité aujourd'hui.**

---

## 3. Tes 3 axes — verdict

### A) Team rank / MMR  → **PAS de MMR brut. CSR = seul signal (déjà capté).**
- `PreMatchRatings` / `PostMatchRatings` : champs présents au schéma mais **null sur
  25/25 matchs (dont 11 classés)** → vestige non peuplé par 343 en 2026.
- Aucun champ `Mmr / MMR / Skill / Rating / TeamMmr` nulle part dans le carnage.
- **CSR par joueur** (Arena classé) : `CurrentCsr`/`PreviousCsr` =
  `{ Tier, DesignationId, Csr, PercentToNextTier, Rank }`. Peuplé pour les actifs
  (5/8 null = placement/inactifs). **Déjà extrait par le projet** (`match_skill_rank`).
  - Sample réel : `{"Tier":1,"DesignationId":6,"Csr":1739,"PercentToNextTier":0,"Rank":236}`
    = **Champion #236** → débloque le TODO `csr_mapper.go` (DesignationId 6 jamais vu jusqu'ici).
- **« Team rank » au sens placement** = `TeamStats[].Rank` (1=victoire) + `Score` +
  `RoundStats[]` (scores par round) — placement/score déjà extraits ; **scores par
  round non modélisés**.

### B) Armes de kill  → **DÉJÀ ingéré + précision par arme récupérable.**
- `KillerWeaponStockId` (Death events) déjà en base (~268k rows), noms via `weapon_labels`.
- **Gratuit en plus** : précision par arme via `WeaponDrop.ShotsFired/ShotsLanded` ;
  variante via `KillerWeaponAttachmentIds` ; arme top + précision via carnage
  `WeaponWithMostKills`.
- `WeaponStats[]` (arsenal complet par joueur dans le carnage) = **VIDE** côté 343 (infixable par là — mais reconstructible via les WeaponDrop).

### C) Player score  → **DÉJÀ ingéré (scalaire). Pas de breakdown objectif.**
- `PlayerScore` par joueur (carnage) déjà extrait (`PersonalScore`). Peuplé en classé.
- **Timeline de score** : `PlayerScoreImpulse` marque quand, **sans montant** → pas de
  breakdown « +X pour capture flag ». La décomposition objectif reste indisponible.
- Détail XP/crédits par match présent (`XpInfo`, `CreditsEarned`, `RewardSets`) — non modélisé.

---

## 4. Carnage — autres champs PEUPLÉS non modélisés (bonus)
`TotalSpartanKills`, `DestroyedEnemyVehicles`, `TotalPowerWeaponGrabs`,
`TotalPowerWeaponPossessionTime`, `TotalPossessionTime`, dégâts par type
(`TotalGrenadeDamage`/`TotalMeleeDamage`/`TotalPowerWeaponDamage`/`TotalShoulderBashDamage`/`TotalGroundPoundDamage`),
`MedalAwards[]` (médailles par match), `KilledOpponentDetails`/`KilledByOpponentDetails`
(**matrice tueur↔victime tête-à-tête agrégée**), `BoostInfo`, `ProgressiveCommendationDeltas`
(déjà utilisé pour les commendations).

## 5. Vrais murs infixables (343 ne fournit pas)
- `damage_taken` (0/13241) → résistance défensive / assists model impossibles.
- `WeaponStats[]` carnage vide → arsenal complet par arme via carnage (contourné par events).
- Breakdown de score objectif (PlayerScoreImpulse sans montant).
