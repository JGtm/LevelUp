# PLAN — Précision et distance des touches PAR ARME (Halo Infinite, depuis le film)

> Date : 2026-08-31 (réécrit — mode « réutilise l'existant » après cartographie de la feature
> weapon_accuracy). Branche cible : `feat/v75` (lots = commits). Worktree d'exécution dédié.
> Contrat d'exécution : skill `plan-execution`. Ce fichier est un PLAN, pas du code.

---

## 0. Le plan en une page

La feature « précision par arme » **existe déjà de bout en bout** — table, repo, taxon
arme→classe/rôle, DTO, **et les composants React** — mais **native Halo 5 seulement**. Pour
Halo Infinite elle est **éteinte** (capability produit `weapon_accuracy` absente du `title.toml`),
faute d'un **numérateur** de touches fiable côté film.

**Ce chantier produit ce numérateur pour Infinite depuis le film**, par la méthode d'attribution
PAR LE TIR validée le 2026-08-31 (`NOTE_ATTRIBUTION_ARME_TIR_2026-08-31.md`) : apparier chaque tir
`action_weapon_fire` (0xD2) au `damage_aftermath` (0xC0) du même attaquant dans une fenêtre W,
clé `WeaponID`. Le **dénominateur** (tirs par arme) est déjà en prod (`match_weapon_shots`,
`film.weapon_shots`). On alimente la **table `weapon_accuracy` existante** pour Infinite, on
**allume la capability**, et les vues existantes s'affichent. On ajoute la **distance** (neuve).

Découpage : **Phase 1** = classe « balle » (~2/3 de l'arsenal, dégât dans `damage_aftermath`) —
livrable. **Phase 2** = classe « lourde » (armes à projectile : SPNKr, Hydra, Skewer, Ravager,
Shock, Mangler, Stalker, Bulldog) — leur dégât n'est ni dans le type 0 ni dans le type 1
(`damage_section_response` RÉFUTÉ le 31/08, ne porte pas d'attaquant) ; **voie = les événements
projectile** (`projectile_detonate` 0xC2 t5, `projectile_impact_effect` 0xC3), investigation
séparée.

**Ce qui N'est PAS refait** (fermé par mesure) : pas de `HitLikely` (scanner refusé, faux ×2) ;
pas de déconvolution ni de coefficient par arme ; pas d'attribution par trajectoire de projectile.

---

## 1. Ce qui EXISTE — à réutiliser tel quel (cartographie 31/08)

| Brique | Emplacement | Réutilisation |
|---|---|---|
| Table `weapon_accuracy` (match_id, xuid, weapon_id, shots_fired, shots_landed, drops) | `games/halo_infinite/migrations/steps_shared_core.go:607` (shared, tous titres) | TABLE EXISTE, vide pour Infinite |
| Persister | `persist/shared_persister.go:342` `persistWeaponAccuracy` + `WeaponAccuracyInsert` (rows.go:93) + `AddWeaponAccuracy` (builder) | title-agnostic — réutilisable |
| Repo/port | `port/weapon_accuracy.go` (`WeaponAccuracyRow`, `LoadWeaponAccuracyAggregated`) + `platform/duckdb/weapon_accuracy_repo.go` | par (xuid, weapon_id) ; filtre 1 joueur OU roster |
| Taxon arme→classe/rôle | `games/weapons/registry.go` (classes: sidearm/shoulder/heavy/… ; rôles: automatic/precision/sniper/shotgun/…) ; `resolveWeaponMeta` ; `domain.WeaponClassHasAccuracy` | réutilisable |
| DTO joueur | `domain.SynthesisWeaponAccuracyEntry` (synthesis.go:174) + `buildWeaponAccuracy` | vue (a) |
| DTO roster | `domain.SquadWeaponAccuracyBar{Role, AccuracyByPlayer…}` (teammates.go:133) | vue (c) |
| React — par arme, joueur actif | `web/src/features/synthesis/SynthesisWeaponAccuracyChart.tsx` (page Synthesis, gaté `useCapability('weapon_accuracy')`) | **vue (a) 100 % réutilisable** |
| React — roster par rôle | `web/src/features/squad/SquadWeaponAccuracyBarsChart.tsx` | **vue (c)** (grouper par classe) |
| Distance des kills (pattern) | `platform/duckdb/kill_distance_repo.go` (hypot3D, `kill_positions_latest`) | pattern distance pour (b) |
| Capabilities | data-level `match.weapon.accuracy` (adapter) ; **produit `weapon_accuracy`** (`title.toml`, front `useCapability`) | à activer pour Infinite |

## 2. Ce qui MANQUE — à construire

1. **Numérateur film (`shots_landed`) pour Infinite** : un mapper qui, dans la passe film,
   apparie tir↔dégât (méthode validée) et produit des `WeaponAccuracyInsert` par (joueur, arme).
2. **`shots_fired` cohérent** : le compte de tirs par arme (même passe ; aligné avec
   `match_weapon_shots.shots_decoded`, par balle — mesure §détente-vs-balle tranchée : le film
   émet un tir PAR BALLE, directement comparable à l'API, aucune conversion rafale).
3. **Activer la capability** `weapon_accuracy` pour Infinite → allume les vues (a) et (c).
4. **Vue (c) par CLASSE** : le chart roster existant groupe par RÔLE ; ajouter le groupement par
   classe (petit ajout dans `aggregateSquadWeaponAccuracy`, libellé `frags.class.<class>`).
5. **Vue (b) précision × distance** (neuve, aucun titre ne l'a) : distance tireur→victime par
   touche (déjà calculée par l'instrument d'attribution) → histogramme par arme ; nouveau DTO +
   composant React.

## 3. Les 3 vues — existe vs à faire

| Vue | Donnée | UI |
|---|---|---|
| (a) précision moy. par arme, joueur actif | numérateur film à produire (dénom. existe) | **réutilise** `SynthesisWeaponAccuracyChart` |
| (b) précision par arme selon la distance | **neuve** : touches bucketées par distance (l'attribution la calcule) | **neuve** : DTO + composant |
| (c) précision par classe, roster | même numérateur ; groupement par classe | réutilise le chart roster (grouper par classe) |

---

## 4. Définition (méthode validée 31/08)

- **Tir** = `action_weapon_fire` 0xD2 long (WeaponID + attaquant lisibles), unité PAR BALLE.
- **Touche** = un tir apparié à ≥ 1 `damage_aftermath` (0xC0 t0) du même attaquant (ref0 tir ==
  ref1 dégât, domaine-1), avec `|ts_dégât − ts_tir| ≤ W`. Un tir → une touche (pas le volume).
- **W = 1 s** (le dégât est horodaté à l'IMPACT). Sweep 250/500/1000/2000 ms conservé comme test
  de non-régression (ratio au témoin +3 s ≥ 3 à 1 s). Verdict de fiabilité = RATIO au témoin.
- **Distance** = `dist(pos_tireur, pos_victime)` au ts du dégât (base slot 512 → position),
  calculée seulement si les deux positions se résolvent.
- **Classe « balle »** = mesurée, pas listée : la porte par-arme (§6) tranche « capturée / non »
  (les armes lourdes = 0 touche type 0 → « non capturée », jamais 0 % faux).

---

## 5. Schéma — RÉUTILISER `weapon_accuracy` (pas de nouvelle table)

- On **écrit dans `weapon_accuracy`** pour Infinite (grain match × xuid × weapon_id) : `shots_fired`
  (film, par balle), `shots_landed` (touches appariées), `drops` = 0 (non pertinent Infinite).
- Idempotence via l'ancre `match_registry` (comme aujourd'hui H5) — INSERT pur, pas d'UPDATE
  (anti-ART ADR 0019/0026/0030 : rien à allowlister).
- **Distance** : la table `weapon_accuracy` n'a pas de colonne distance et on ne l'y ajoute PAS
  (éviter le rebuild ADR 0026 d'une table peuplée H5). La distance va dans une **table sœur neuve**
  `match_weapon_hit_distance` (match_id, xuid, weapon_id, dist_bucket_json, dist_n, decode_pass,
  decoder_rev, written_at) + vue `_latest`. Lecture (b) = jointure weapon_accuracy × distance.
- **`decoder_rev`** propre au numérateur film (`WeaponHitsDecoderRev`), distinct du décodeur de
  tirs.
- Piège UBIGINT weapon_id : chaîne décimale à l'INSERT (`ubigintArg`), comme l'existant.

---

## 6. Qualité du taux (notre contrôle, normal — pas un gate hérité)

Toute feature d'accuracy doit montrer des taux **sensés**. Deux contrôles internes, dans les gates
des lots, pas un blocage préalable :
- **Recalage sur l'API** : le film ne voit qu'un échantillon des dégâts → les taux ABSOLUS sont
  partiels. La table stocke les BRUTS ; un lecteur `analysis` recale la forme sur l'accuracy
  globale de l'API (`match_participants.accuracy`) — la valeur affichée est la forme film ancrée
  API. (Calcul de lecture ; l'écriture stocke les bruts.)
- **Porte par-arme** (`publishable` + `gate_reason`, une seule copie dans le persister) : effectif
  `shots_paired ≥ Nmin` (fixé au Lot par mesure) ; arme « capturée » (`hits > 0` ou signature de
  classe balle) ; sinon `gate_reason` explicite. Une arme « non capturée » n'est JAMAIS affichée
  à 0 %.

Note : une vieille note `capabilities.toml` évoquait un taux faux (inversion MA40/Sidekick) — elle
portait sur **HitLikely** (autre méthode, refusée). Elle ne conditionne PAS ce chantier ; notre
contrôle est le recalage API + la porte ci-dessus, sur NOTRE donnée.

---

## 7. Architecture Go (couches)

| Rôle | Emplacement | Statut |
|---|---|---|
| Décode tir 0xD2 + dégât 0xC0 t0 + pairing + distance | `internal/analysis/filmdec` (sortir du `_test`) | **Lot 2** |
| Mapper Infinite → `WeaponAccuracyInsert` + distance | `games/halo_infinite/ingest/weapon_accuracy_film.go` (neuf, pendant Infinite du mapper H5) | **Lot 3** |
| Orchestration passe (greffe `killcollector`) | `sync/killcollector/hits.go` (neuf) | **Lot 3** |
| Persist `weapon_accuracy` + distance | `persist/shared_persister.go` (réutilisé) + `weapon_hit_distance_persister.go` (neuf) | **Lot 3** |
| DDL distance + `_latest` | `migration/steps_shared_weapon_hit_distance.go` (neuf) | **Lot 1** |
| Vue (c) par classe | `service/teammates` (groupement classe) | **Lot 4** |
| Vue (b) distance | DTO domain + `analysis` lecteur + React | **Lot 5** |

- Logging `slog.*Context` (lignes écrites, joueurs refusés, indices non résolus).
- `PathResolver.SharedDBPath(slug)` — jamais `filepath.Join`.

---

## 8. Multi-titre (capability, jamais slug)

- Activer **produit** `weapon_accuracy` dans `config/titles/halo_infinite/title.toml` (le front
  `useCapability` allume alors les charts a/c).
- Data-level `match.weapon.accuracy` : passer à `supported` pour Infinite une fois la donnée
  écrite (adapter). Branchement `HasCapability`, jamais `slug ==` (ratchet
  `no_slug_comparison_test.go`).
- H5 inchangé (source native `weapon_drop`). La distance (b) est Infinite-only (capability
  `weapon.hit_distance`, absente H5) → dégradation propre.

---

## 9. Réserves

1. **Classe lourde** : 0 touche en type 0/1 → Phase 2 (événements projectile). Porte = « non
   capturée », jamais 0 % faux.
2. **Fenêtre d'impact** W = 1 s (compromis, ratio témoin documenté).
3. **Une-ref** : ~33 % des dégâts n'ont qu'une réf → pas de distance (touche comptée quand même).
4. **Échantillon film partiel** → recalage API obligatoire pour l'affichage.
5. **Cartes à signature ambiguë** : distance désactivée si bornes non résolues (touche comptée).

---

## 10. Lots (gates)

> Contrat `plan-execution` : ordre strict, une étape à la fois, gate vérifié, items statués.
> Cache Go privé au worktree. Tests film : `-count=1`, `LOT1_TRAME_FILM` par film.

### Lot 0 — Réconciliation doc (bloquant)
- [ ] Corriger le doc-header de `filmdec/fire_events.go` (affirme à tort 0xD2 == record de dégât ;
      0xD2 = tir, 0xC0 = dégât). Commentaire seul, zéro comportement.
- **Gate** : `go test ./internal/analysis/filmdec/ -count=1` vert ; `git diff` = commentaire seul.

### Lot 1 — DDL distance + seuils de porte  — CLOS 2026-09-01
- [x] `steps_shared_weapon_hit_distance.go` (table sœur `match_weapon_hit_distance` + vue
      `_latest`, append-only INSERT-only ADR 0026) + `order.go` (position AVANT
      `shared_weapon_kills_v3` — dictée par l'init alphabétique du nom de fichier ; no-op
      test vert). Test `steps_shared_weapon_hit_distance_test.go` (3 cas, `-run WeaponHitDistance`).
- [x] `Nmin` fixé à **8** (`migration.WeaponHitsMinShots`) par mesure sur les 3 films
      (instrument `filmdec/lot1_nmin_effectif_research_test.go`, W=250 ms, 12 chunks témoin) :
      le sous-ensemble « ≥ 8 tirs/clé (joueur,arme) » est stable (11 clés sur les 3 films quand
      le total va de 14 à 34) → isole les armes réellement utilisées. Constante + tableau de
      mesure documentés dans le doc-header de la migration. `decoder_rev = "whd-v1"`
      (`WeaponHitDistanceDecoderRev`). Règle de porte écrite (dist_n ≥ Nmin, tranchée par le
      lecteur au Lot 5).
- **Gate** : `go test ./internal/migration/ -run WeaponHitDistance -count=1` **vert** ;
      `CanonicalOrder`/`SortByCanonical` verts ; `gofmt -l` vide ; `go vet` propre.
- Prochaine étape : **Lot 2** (décodeur dégât + pairing + distance, sortis du `_test`).

### Lot 2 — Décodeur dégât + pairing + distance, sortis du `_test`  — CLOS 2026-09-01
- [x] API prod exposée dans `filmdec` (`weapon_hits.go` + `weapon_hits_decode.go`) : `WeaponShot`,
      `WeaponDamage`, `WeaponHitStats{FilmIndex, WeaponID, ShotsPaired, Hits, DistBuckets}` ;
      `PairWeaponHits(shots, damages, window, dist)` (pur) ; `WeaponHitBucket`/`WeaponHitDistanceEdges`
      /`WeaponHitPairWindowUS`(=1 s) ; scanners `ScanFilmWeaponShots`, `ScanFilmWeaponDamages`
      (rend aussi la base-512). Décodeurs `lot1RefDom/lot1RefDom1/lot1DecodeDamageAftermath/
      lot1Dequant` + résolution slot (`lot1chBases/lot1chIsBiped/lot1ArgmaxBase`) DÉPLACÉS des
      `_test` vers `weapon_hits_decode.go` (une seule copie). `sondeScanDamage`, `attribCollectShots`,
      `attribM2`/`attribM3`, `sondeBucket`/`sondeDistEdges` DÉLÈGUENT désormais au code prod.
- [x] Test unitaire pur (`weapon_hits_test.go`, sans DuckDB/film) : tir apparié / non apparié /
      hors fenêtre / un tir → une seule touche pour N dégâts / non-appariable écarté / bucket
      distance / distance non résolue / clés par (index, arme) / `WeaponHitBucket`.
- **Gate** : `go test ./internal/analysis/filmdec/ -count=1` **vert** (8,85 s) ; `gofmt -l` vide ;
      `go vet` propre. `TestLot1AttribArmeTir` (film 000d5950, `LOT1_TRAME_FILM`) REPRODUIT via le
      code productionisé : 245 tirs / 190 dégâts, base 512 ; **sweep W=1000 ms = 100/245 (40,8 %)
      contre témoin +3 s 5,3 % → ratio 7,7× ≥ 3** (M1 avant 16,7× / arrière 8,0× → TENU).
- Prochaine étape : **Lot 3** (mapper Infinite → `WeaponAccuracyInsert` + distance ; passe
      killcollector ; persist ; capability).

### Lot 3 — Mapper + passe + persist (weapon_accuracy Infinite + distance)  — CLOS 2026-09-01
- [x] **Sous-étape 1 (résolveur distance productionisé)** : `filmdec/weapon_hit_distance_resolver.go`
      — `BuildBipedTracks`/`ResolveHitDistanceBase`/`NewWeaponHitDistanceFunc`/`FilmWeaponHitDistance`
      (WeaponHitDistanceFunc depuis positions bipèdes base 512, ScanFilmBipedPositions) +
      `DetectFilmWorldRange` (auto-détection carte par signature de largeurs d'axe). Copie #2 du
      résolveur sonde de recherche (≤ 2, CLAUDE.md rule 6). Tests filmdec unitaires verts.
- [x] `games/halo_infinite/ingest/weapon_accuracy_film.go` : `MapWeaponAccuracyFilm` (miroir H5) →
      `[]WeaponAccuracyInsert` (shots_fired=ShotsPaired, shots_landed=Hits, drops=0) + rows distance
      (dist_bucket_json, dist_n). Pont FilmIndex→xuid injecté. 4 tests purs verts (`-run WeaponAccuracy`).
- [x] `killcollector/hits.go` : `collectHits` greffé sur la passe (`collect()`), résout FilmIndex→xuid
      par `resolvePlayerIndices` EXISTANT, métriques `slog` (accuracy/distance/indices non résolus).
      DIR-BASE (les scanners filmdec du Lot 2 lisent chunk_NN.bin sur disque — le rejeu en mémoire
      dupliquerait le décodeur) ; branché via `ConfigureFilmAccuracy` (filmDir + catalogue bornes),
      best-effort si non configuré. Test capability calqué sur shots (porte via résolveur invoqué).
- [x] `weapon_hit_distance_persister.go` (INSERT-only, porte `EvaluateHitsGate` = copie UNIQUE Nmin=8).
      DÉVIATION du plan consignée : `weapon_accuracy` NE réutilise PAS `persistWeaponAccuracy` (chemin
      BatchBuilder = **no-op** sur un match déjà inséré, cas d'une passe film tardive) — écriture
      directe sous le même lease, forme SQL identique + **garde SELECT-then-INSERT** anti-doublon
      (weapon_accuracy n'a ni decode_pass ni `_latest`, un backfill re-run doublerait). Distance =
      append-only (decode_pass + `_latest`). Anti-ART : rien à allowlister. 6 tests intégration verts.
- [x] `capabilities.toml` (data-level `match.weapon.accuracy` → `supported`, commentaire daté 09-01) +
      `adapter_data.go` (CapabilityMap → `CapSupported`, gate `collectHits`) + `registry.go` (produit
      `weapon_accuracy` ajouté au descripteur Infinite **built-in** — Infinite n'a pas de `title.toml`,
      son manifeste est en dur, cf. config_loader.go). Ratchet `no_slug_comparison` vert (jamais slug==).
- **Gate** : `go test ./internal/persist/ ./internal/sync/killcollector/ ./internal/games/halo_infinite/ -count=1`
      **vert** ; `go test -tags=integration -p 1 ./internal/persist/` **vert** (23 s, anti-ART) ;
      slug ratchet (`internal/archlint TestNoNewSlugComparison`) + ART (`internal/sync
      TestNoARTPatterns*`) **verts** ; mapper `-run WeaponAccuracy` + persister `-run WeaponHitDistance`
      **verts** ; gofmt/vet propres ; fichiers ≤ 219 L, fonctions ≤ 60 L.
- **Blocage hors périmètre (non-fatal)** : `go test ./internal/...` échoue à COMPILER pour
      `internal/service`, `internal/api*`, `internal/scheduler`, `internal/worldenrich`,
      `internal/service/teammates`, `internal/games/halo_5/livesync` — casse PRÉ-EXISTANTE
      (`internal/service/killsourceload` importé mais absent du disque + `port.KillSourceClassRepository`
      indéfini), hors des fichiers touchés (cf. §11). Gate exécuté sur les paquets compilables.
- Prochaine étape : **Lot 4** (vues a/c allumées + groupement par classe).

### Lot 4 — Vues (a) et (c) allumées + (c) par classe
- [ ] Vérifier que `SynthesisWeaponAccuracyChart` (a) et le chart roster (c) s'affichent pour
      Infinite (capability on, données présentes). Ajouter le groupement par CLASSE au chart roster
      (libellés i18n `frags.class.<class>` FR+EN).
- [ ] Recalage API dans le lecteur (`analysis`), title-agnostic, écart à l'API exposé à côté.
- **Gate** : `make check-types` + `make test-web` + `make go-api-test` verts ; capture visuelle
      (gate visuel user) des deux charts sur un joueur Infinite avec film.

#### Lot 4a (vue a — précision moyenne par arme, joueur actif) — BLOQUÉ / ESCALADÉ 2026-09-01
> Périmètre : vue (a) SEULE. Zone WRITE/filmdec interdite (mapper, persister, killcollector, filmdec).
> Note complète : `.ai/V7.5/film_re/RECALAGE_WEAPON_ACCURACY_FILM_2026-09-01.md`.
- [x] **Idempotence** — garantie à l'ÉCRITURE (SELECT-then-INSERT dans
      `WeaponHitDistancePersister.insertAccuracy`) ; un ré-décodage ne double pas `weapon_accuracy`.
      Prouvé par `TestWeaponHitDistanceIdempotenceAccuracy` (intégration, déjà vert). La lecture
      brute `FROM weapon_accuracy` est donc sûre (génération unique) → aucun changement de lecture
      requis, PAS de nouvelle vue `_latest` à ajouter.
- [x] **DI** — `WithWeaponAccuracyRepo(NewWeaponAccuracyRepo(pdb))` câblé INCONDITIONNELLEMENT
      (title-agnostic, jamais `slug==`) dans `SynthesisCtx` (registry_pages_home.go:307),
      `Timeseries`, `SessionPage`, `TeammatesCtx` → le Synthèse d'Infinite reçoit bien le repo.
      Aucun câblage à ajouter.
- [x] **Gate web** — `useCapability('weapon_accuracy')` lit la capability PRODUIT du bootstrap
      (miroir `title.registry.go`, posée sur Infinite au Lot 3). Le chart s'allumerait automatiquement.
- [x] **RECALAGE mesuré** (8 films, W=1 s, code prod, vs `AVG(match_participants.accuracy)`) :
      ratio film/API 0.06–0.75 (≈12×, anti-corrélé) ; armes automatiques invraisemblables
      (MA40 AR 0.9–3.3 % vs API 40–42 %) ; faux 0 % des armes projectile (Ravager/SPNKr/Mangler…
      passent Nmin, non filtrées par `WeaponClassHasAccuracy`). Grain « balle » seul plausible
      (000d5950 ~25 % vs API 28 %) mais noyé.
- [!] **UI NON câblée** — recalage ABERRANT au grain qui alimente la vue (question user « c'est
      fiable ? » = NON en l'état). Deux verrous en zone WRITE/filmdec (interdite Lot 4a) : (V1)
      méthode de pairing qui écrase les armes automatiques ; (V2) porte « capturée » plan §6 absente
      (persister ne gate que Nmin). Prod `weapon_accuracy` Infinite = 0 ligne (vue vide de toute
      façon). Décision d'allumage/gating de la capability (posée prématurément au Lot 3) =
      **ressort du pilote**, pas d'un unwind unilatéral. Escaladé + report au REGISTRE_REPORTS.

### Lot 5 — Vue (b) précision × distance
- [ ] DTO domain + lecteur `analysis` (jointure weapon_accuracy × distance) + endpoint.
- [ ] Composant React (histogramme précision par arme selon la distance), i18n FR+EN, tokens
      couleur (skill `color-tokens`), query key dans `lib/query/keys.ts`.
- **Gate** : `make check-types` + `make test-web` verts ; gate visuel user.

### Lot 6 — Livraison
- [ ] `make gate-push` ; suite complète verte ; entrée `thought_log.md` ; MAJ `MEMORY.md` /
      `.ai/V7.5/` ; report Phase 2 (classe lourde / projectiles) au `REGISTRE_REPORTS.md`.
- **Gate** : `skill delivery-checklist` ; CI verte au niveau JOB.

---

## 11. Découvertes hors périmètre (consigner, NE PAS traiter)
- **[Lot 3] Casse pré-existante `killsourceload`** : `internal/service/match_view_data_loaders.go:21`
  importe `internal/service/killsourceload`, paquet ABSENT du disque (dossier inexistant), et
  `internal/service/teammates` référence `port.KillSourceClassRepository` indéfini → `internal/service`
  et tout son aval (`internal/api*`, `scheduler`, `worldenrich`, `teammates`, `halo_5/livesync`) ne
  compilent pas. Documenté dans MEMORY.md (« importeur commité + paquet non suivi = branche non
  compilable »). HORS périmètre Lot 3 (interdiction de toucher `internal/service`/`cmd/*`). À résoudre
  avant tout gate `./internal/...` global (Lot 4 touche le service teammates : il faudra le lever).
- **[Lot 3] Réserve pont FilmIndex→xuid — LEVÉE ET CORRIGÉE (mesuré 2026-09-01)** : la réserve était
  que `filmdec.WeaponHitStats.FilmIndex` (`decodeFireEvent`, bits 36-40 >>1 = 4 bits) et l'indice
  qu'indexe `resolvePlayerIndices` (5 bits, aligné sur `analysis.PlayerIndex5` = event_start+31)
  soient des champs différents. VERDICT MESURÉ : le 4 bits n'était que la MOITIÉ BASSE du 5 bits
  (`ShooterIndex5 & 0x0F == FilmIndex`, invariant algébrique). L'ancre paquet de filmdec place le champ
  au bit 35 du payload, qui coïncide au bit près avec `event_start+31` d'`analysis` → `ShooterIndex5`
  (bits 35-39) == `PlayerIndex5`. Preuve : `TestWeaponIndexNumDenomEquivalence` (package analysis) —
  **mismatch 0** sur 4342 records BTB corrélés + tous les records arène ; table de correspondance
  publiée. Sous 17 joueurs (arène) les deux lectures coïncidaient déjà (correction = no-op, tables
  d'identité identiques) ; au-delà (BTB 4f77afc1, lobby **24**), le 4 bits saturait à 15 (distinct=16)
  et fusionnait 8 paires (idx5 16-23 → idx4 0-7) → précision fausse. CORRECTION : `fire_events.go`
  expose `FireEvent.ShooterIndex5` (bits 35-39, R(5) sans >>1) ; `ScanFilmWeaponShots` key le
  numérateur dessus. `FilmIndex` (4 bits) reste pour le regroupement visée de replay (golden verts).
  Vérité terrain : `TestWeaponIndexGroundTruth` (filmdec) — 4 bits structurellement insuffisant pour le
  lobby BTB de 24 (preuve par dénombrement), 5 bits le couvre ; no-op en arène confirmé.
- **[Lot 3] `weapon_accuracy` sans idempotence native** : la table partagée (peuplée nativement côté
  H5) n'a ni `decode_pass` ni vue `_latest` ; pour Infinite elle n'est écrite QUE par la passe film,
  qui est re-jouable (backfill) → risque de doublon. Résolu Lot 3 par garde SELECT-then-INSERT
  (skip si le match a déjà des lignes). Si un jour H5 et Infinite partageaient la même DB (aujourd'hui
  DB par titre), cette garde deviendrait insuffisante — à revoir alors.

## 12. Reprise de session
- Avancement : cases `[ ]` de §10. Reprendre au premier lot non clos (gate vert + items statués).
- Contexte figé : `NOTE_ATTRIBUTION_ARME_TIR_2026-08-31.md` (méthode), cette cartographie (§1),
  ce plan (décisions tranchées).
