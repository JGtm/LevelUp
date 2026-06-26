# Halo 5 — Enrichir l'engagement (et autres graphes) avec les events de match

## Comment l'engagement marche aujourd'hui (rappel pour cadrer)
L'engagement N'EST PAS une somme pondérée de stats. C'est un **percentile de résidu de
rythme** (`internal/analysis/temporal/engagement_*.go`) :
- Courbe de rythme sur fenêtre glissante 90s (échantillon 10s) : `pace_joueur(t)` = events
  du joueur/min, comparé à `pace_attendu(t) = coef_team_share × pace_team(t)`.
- `résidu = moyenne(pace_joueur − pace_attendu)` → converti en **percentile vs l'historique
  du joueur** (200 derniers matchs, même catégorie de mode). 50 = « comme d'habitude ».
- Les « events » = lignes de `shared.highlight_events`. **Pour Halo Infinite, seulement
  kill/death/medal** y figurent.
- Consommateurs : axe « Activité » du **profil de combat** (radar), alertes du **Coach**
  (`combat_actif`/`discret`/`fragile` sur le résidu ±5).

## Le vrai blocage H5 (au-delà de « recalibrer »)
Pour H5, `highlight_events` ne contient **QUE les médailles** : les kills/deaths partent
dans `killer_victim_pairs` + `weapon_kills`, et impulse/spawn/pickup/drop sont **droppés à
l'ingest** (`internal/games/halo_5/ingest/collect.go`). Donc l'algo actuel tournerait sur
un rythme « médailles-only » → échelle et sens faussés.

**Prérequis #1 (débloque tout le reste)** : émettre aussi des lignes kill/death dans
`highlight_events` depuis la timeline H5 (la donnée est déjà fetchée), + une baseline de
coefficients par titre (calque du `effective_hp_to_kill` par titre déjà en place).

## Event par event — ce que ça apporte à l'engagement
| Event H5 (déjà fetché) | Champ clé | Apport engagement |
|---|---|---|
| **Death** (kill) | Killer/Victim, Time | **Prérequis** : rythme kill/death (cœur de la courbe). Aujourd'hui absent de highlight_events H5. |
| **Death.Assistants[]** | gamertags assistants | Rythme d'**assist par kill** (support actif). Un joueur qui assiste beaucoup est engagé même sans frags. Aujourd'hui `assist` est une branche morte (jamais peuplée en HINF) — H5 a la vraie donnée. |
| **Impulse objectif** (Flag Pickup/Capture/Pulls, Point Victories, PowerWeaponGrabbed…) | ImpulseId (nommé via /impulses) | Rend l'engagement **objective-aware** : porteur de drapeau à faibles frags = très engagé. Branche le hook mort `EventsObjectifEstimes` (`ObjectivePointsPerCapture=25`) sur de **vrais** events au lieu d'une estimation. Décisif hors-Slayer. |
| **Death.DeathDisposition** | suicide/trahison/ennemi | **Dé-pondérer** : un suicide n'est pas de l'« activité d'engagement » ; une trahison est négative. Alimente plutôt un signal **discipline** distinct. |
| **Medal** (déjà dans highlight_events) | MedalId + type/tier | Aujourd'hui seul le COMPTE compte. Pondérer par tier (multikill/spree) → signal **intensité/flair**. |
| **WeaponPickup / PowerWeaponGrabbed** | WeaponStockId / impulse | Signal **contrôle de map / agressivité** (qui prend les armes lourdes, quand). |

> Note GameVariant : le CMS `GameVariantDefinition` donne les **règles du mode**
> (`ScoreToWin`, `Rounds`, `MatchLength`, type de base Slayer/CTF/Strongholds) → utile pour
> **normaliser** la contribution objectif (ex. caps / ScoreToWin). MAIS les UUID de
> `FlexibleStats.ImpulseStatCounts` du carnage vivent dans le **blob binaire** du variant,
> **non résolus** par /impulses → on les identifie par inférence (count == kills) ou on
> s'appuie sur les **events Impulse de la timeline** (eux, nommés via /impulses). Pour
> l'engagement, **utiliser la timeline d'events** (nommée), pas les UUID FlexibleStats.

## Idées de graphes (au-delà de l'engagement)
Tous alimentés par des données déjà fetchées (events + carnage) :
1. **Courbe d'engagement annotée objectif** — la courbe de rythme + marqueurs Flag Capture/
   Pull/KOTH sur la timeline (où le joueur « pousse »).
2. **Matrice tête-à-tête (nemesis/victime)** — `KilledOpponentDetails`/`KilledByOpponentDetails`
   (carnage, déjà peuplé) → qui te tue / que tu domines, par match ou session.
3. **Réseau d'assists (synergie d'escouade)** — `Death.Assistants[]` → graphe « qui finit
   les kills de qui » → page Escouade.
4. **Proficience par arme** — `WeaponDrop.ShotsFired/ShotsLanded` (events) → précision par
   arme (le carnage `WeaponStats[]` est vide → events seuls).
5. **Donut discipline** — `DeathDisposition` : ennemi / suicide / trahison → signal propreté.
6. **Timeline de contribution objectif** — Flag Pickup/Capture/Pulls par joueur → qui porte
   réellement l'objectif (vs le fragger).
7. **Heatmap spatiale** — `KillerWorldLocation`/`VictimWorldLocation` (events) → positions
   kills/morts (canonical modélise déjà les positions).
8. **Timeline multikills/sprees** — médailles (timeline) → pics de domination.

## Où ça se branche
- Étendre l'**ingest** (`halo_5/ingest/collect.go`) pour émettre kill/death (+ assist /
  objectif) dans `highlight_events`, et la struct canonique `HighlightEvent` (`canonical/
  events.go` modélise déjà kill/assist/medal/impulse/KillKind/Headshot/positions).
- L'algo engagement ne change pas de signature au début : il consomme `[]HighlightEvent`.
- `DeathDisposition` est greenfield (aucun ingest aujourd'hui) : suicide inférable si
  `Killer == nil` ; trahison = killer même équipe que victime (à câbler).
- Baseline coefficients **par titre** (mirror du damage-model par titre).

## DESIGN VERROUILLÉ (2026-06-26, décisions user)
### Types comptés dans la courbe (les DEUX titres)
| Type | Inclus | Poids (départ, à calibrer) | Note |
|---|---|---|---|
| `kill` (Death) | oui | 1.0 | combat de base |
| `medal` | oui | 1.0 (additif) | **INTENSITÉ** : double kill = 2 kill + 1 medal = 3 > 2 kills isolés. **NE PAS dé-dupliquer** (décision user : un multikill ≠ deux kills) |
| `objectif` (mode / impulse) | oui | 1.5 | prime « meneur d'objectif » |
| `assist` | oui (H5 maintenant, **Infinite à terme**) | 0.5 | support ; type 1re classe les 2 titres (branche `EventAssist` = code en avance) |
| `death` | oui | 0.4 | présence subie → poids moindre |
### Exclus (ne pas écrire dans highlight_events / ne pas compter)
- **H5** : WeaponPickup/Drop, PlayerSpawn, Round*, **impulses kill-bruts** (`Kills`/`Enemy Player Kill`/`Headshot`/`Perfect Kill` = doublon des Death, contrairement aux médailles qui portent le palier), `PlayerScoreImpulse` (tick sans montant), bonus PvE/Warzone, `Revived`, `Suicides` (→ discipline, pas engagement).
- **Infinite** : déjà propre (kill/death/medal/mode seulement) → rien à exclure. Le recadrage met H5 au niveau d'Infinite, pas l'inverse.
### Pondération — changement MODÈLE (2 titres)
Adoptée pour les deux. Touche `ComputeEngagementScore` + recompute coefficients + **re-backfill des 2 titres** + validation (décale les percentiles historiques d'une métrique en prod). → étape DÉDIÉE, **après** l'enrichissement ingest H5 (lui additif/sans risque). Poids ci-dessus = départ à calibrer (sanity vs win/CSR).
### Parité
H5 = mirror d'Infinite (kill+death+medal+objectif+assist). Infinite déjà objective-aware via events `mode` ; `EventsObjectifEstimes` (estimation depuis personal_score) = **code mort**, inutile (garder ou supprimer).
