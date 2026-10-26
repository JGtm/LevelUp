# RECALAGE — précision par arme du film Infinite vs accuracy API (Lot 4a)

> Date : 2026-09-01. Branche `feat/precision-arme` (worktree dédié). Mesure demandée par le
> Lot 4a (« c'est fiable ? ») AVANT tout câblage de la vue (a). Instrument : le code de
> production `filmdec.ScanFilmWeaponShots / ScanFilmWeaponDamages / PairWeaponHits` via
> `TestLot1AttribArmeTir` (fenêtre de production W=1 s, une-touche-par-tir, clé WeaponID),
> 12 premiers chunks par film. Baseline API : `AVG(match_participants.accuracy)` (0..100)
> du même match, lu en RO sur le shared de prod (B-swap).

## Contexte de l'état des données

- `weapon_accuracy` pour `halo_infinite` en prod = **0 ligne** (`SELECT COUNT(*)` sur le
  shared). La passe film (`killcollector/hits.go`) n'a jamais tourné/backfillé : la table
  est vide, la vue (a) serait de toute façon **vide** en l'état.
- La feature vit sur une **branche** (`feat/precision-arme`), pas en prod : aucun risque live
  immédiat ; la décision d'allumage se prend AVANT merge.

## Mesure (8 films, arène/BTB)

| film     | tirs | film GLOBAL % | API moy % | ratio film/API | armes-clés (W=1 s) |
|----------|-----:|--------------:|----------:|---------------:|--------------------|
| 000d5950 |  245 | 16.7 | 28.0 | 0.60 | Disruptor 29%, BR75 33%, Needler 40%, Sniper 19% (plausibles) |
| 0014603f |  365 |  4.9 | 31.5 | 0.16 | — |
| 00162144 |  902 |  8.9 | 36.0 | 0.25 | — |
| 00502e52 |  276 | 17.4 | 28.7 | 0.61 | — |
| 00761d27 |  954 |  6.2 | 40.6 | 0.15 | **MA40 AR 0.9%** (346 tirs/3), Sidekick 9.9%, Needler 17% |
| 0076ebdc |  509 | 13.4 | 27.7 | 0.48 | — |
| 008e1bba |  573 |  2.4 | 42.1 | 0.06 | **MA40 AR 3.3%** (359 tirs/12), Sidekick 1.3%, Vestige 0% |
| 010d1f7b |  360 | 21.4 | 28.4 | 0.75 | — |

## Verdict : ABERRANT au grain qui alimente la vue — NE PAS câbler l'UI

1. **Aucune relation stable film↔API.** Le ratio film/API varie de **0.06 à 0.75** (≈ 12×),
   et il est *anti-corrélé* : le film le plus haut en API (008e1bba, 42%) est le plus bas en
   film (2.4%). Un facteur de recalage unique n'existe pas → le « recalage sur l'accuracy
   globale API » prévu au plan §6 n'a pas de point d'ancrage fiable.

2. **Les armes AUTOMATIQUES lisent des taux invraisemblables.** Le MA40 AR (hitscan, arme de
   base) sort à **0.9 % (00761d27) et 3.3 % (008e1bba)** alors que l'accuracy API du match est
   à 40–42 %. Le Sidekick tombe à 1.3–9.9 %. La fenêtre 2 s remonte ces mêmes armes à 5–28 % :
   la méthode une-touche-par-tir keyée WeaponID à W=1 s **sature/perd le volume** sur les armes
   à cadence élevée (le film n'émet que ~190 `damage_aftermath` pour 245+ tirs ; chaque dégât
   ne s'apparie qu'à UN tir). Publier « Précision par arme » avec un fusil d'assaut à 3 %
   détruirait la confiance — c'est précisément la question « c'est fiable ? » : **non, pas au
   grain par-arme, pas encore.**

3. **Faux 0 % des armes à projectile** (défaut distinct, cumulatif). Le mapper
   (`weapon_accuracy_film.go`) et le persister (`EvaluateHitsGate`, Nmin=8 seul) n'appliquent
   PAS la « porte par-arme (capturée)» du plan §6. Résultat : Ravager, M41 SPNKr, Mangler,
   Stalker, Pulse Carbine… — dégât projectile INVISIBLE au type 0 — passent Nmin et seraient
   écrits à **0 % touche** (ex. 000d5950 : Ravager 16 tirs/0, SPNKr 16/0, Mangler 15/0). Côté
   lecture, `buildWeaponAccuracy` ne filtre que par `WeaponClassHasAccuracy` (classe) : ces
   armes sont des « guns » → NON exclues → afficheraient un faux 0 %, interdit par le plan §6
   (« une arme non capturée n'est JAMAIS affichée à 0 % »). Le grain « balle » sain (000d5950 :
   ~25 % vs API 28 %) est noyé par ce bruit dans le GLOBAL.

## Ce qui reste solide

- **Idempotence : garantie à l'écriture.** `WeaponHitDistancePersister.insertAccuracy` fait un
  SELECT-then-INSERT (skip si le match a déjà des lignes) — un ré-décodage ne DOUBLE pas
  `weapon_accuracy` (table sans `_latest`) ; la distance, append-only, régénère par
  `decode_pass` (vue `_latest`). Prouvé par `TestWeaponHitDistanceIdempotenceAccuracy`
  (intégration). La lecture brute `FROM weapon_accuracy` est donc sûre (une seule génération).
- **DI title-agnostic : déjà en place (Lot 3).** `SynthesisCtx` /
  `Timeseries` / `SessionPage` / `TeammatesCtx` câblent
  `WithWeaponAccuracyRepo(NewWeaponAccuracyRepo(pdb))` **inconditionnellement** (jamais
  `slug==`) → le service Synthèse d'Infinite reçoit bien le repo.
- **Gate web déjà correct.** `useCapability('weapon_accuracy')` lit la capability PRODUIT du
  bootstrap (`availableTitles[].capabilities`, miroir de `title.registry.go`), que le Lot 3 a
  posée sur Infinite. Le chart `SynthesisWeaponAccuracyChart` s'allumerait donc automatiquement.

## Recommandation (décision pilote requise — hors périmètre vue seule)

La vue (a) est **techniquement prête à s'afficher** (DI + capability + chart), mais la DONNÉE
n'est pas fiable au grain par-arme. Deux verrous, tous deux dans la zone WRITE/filmdec
(interdite au Lot 4a) :

- **V1 — méthode de pairing (filmdec / Phase 2)** : W=1 s + une-touche-par-tir écrase les armes
  automatiques. Piste : compter le VOLUME de dégâts appariés (pas 1/ tir), ou élargir W par
  classe, ou recaler par arme sur l'API — à trancher hors vue.
- **V2 — porte « capturée » (persister / plan §6)** : n'écrire/ne publier que les armes dont la
  classe est réellement mesurable par le film (dégât type 0), les projectiles → `gate_reason`
  au lieu de 0 %.

**En l'état : capability `weapon_accuracy` d'Infinite à considérer comme PRÉMATURÉE.** Tant que
V1/V2 ne sont pas levés, la vue (a) ne doit pas être présentée (soit garder la capability OFF,
soit n'afficher que le grain « balle » recalé). Escaladé au pilote ; non tranché ici (unwind
d'un lot précédent = décision du pilote).
