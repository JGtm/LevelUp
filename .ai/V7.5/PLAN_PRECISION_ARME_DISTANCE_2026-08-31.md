# PLAN — Précision et distance des touches PAR ARME, depuis le film Theater

> Date : 2026-08-31. Branche cible : `feat/v75` (lots = commits ; mode branche unique v7.5).
> Worktree d'exécution : dédié, hors du principal partagé.
> Contrat d'exécution : **skill `plan-execution`** (ordre strict, aucune étape différée,
> chaque item statué `[x]`/`[~]`/`[!]`, zéro fix hors périmètre — voir §10).
> Ce fichier est un PLAN. Il ne contient aucun code de production ; il décrit ce qui reste à
> écrire, où, et comment le vérifier.

---

## 0. Le plan en une page

Le moteur calcule nativement une précision PAR ARME (Ghidra : compteurs `ShotsFired` /
`ShotsLanded` côte à côte, `automatic_weapon_shot_count`), mais ne la réplique PAS dans le film
(`VERDICT_PRECISION_PROJECTILES.md` §4bis). L'API du match ne donne que l'Accuracy GLOBALE
(`match_participants.shots_fired` / `shots_hit`, toutes armes confondues). **La ventilation par
arme et par distance n'existe donc nulle part — sauf à la reconstruire du film.**

Le **dénominateur existe déjà en prod** : `shared.match_weapon_shots` (grain match × joueur ×
arme, colonne `shots_decoded`, porte ±10 %, capability `film.weapon_shots`), peuplé par la passe
`killcollector` (même passe que les morts). Ce plan ajoute le **NUMÉRATEUR (touches)** et la
**DISTANCE**, par la méthode d'attribution PAR LE TIR validée le 2026-08-31
(`NOTE_ATTRIBUTION_ARME_TIR_2026-08-31.md`) : apparier chaque tir `action_weapon_fire` (0xD2) au
`damage_aftermath` (0xC0) du même attaquant dans une fenêtre temporelle, clé `WeaponID`.

Découpage : **Phase 1** = classe « balle » (~2/3 de l'arsenal, dégât dans `damage_aftermath`
type 0), livrable. **Phase 2** = classe « lourde » (SPNKr/Hydra/Skewer/Ravager/Shock/Mangler/
Stalker/Bulldog, dégât hors type 0) — **voie NON trouvée** : le type 1 (`damage_section_response`)
a été **RÉFUTÉ le 31/08** (grammaire percée mais ne porte pas d'attaquant), le dégât lourd est
probablement dans les composants ECS projectile/détonation (investigation séparée, non planifiée).

**Ce que ce plan NE fait PAS** (fermé par mesure, ne pas rouvrir sans nouveau fait) :
- pas de `HitLikely` (scanner) : MORT, annonce 75-79 % pour une précision réelle de 0,446 ;
- pas de déconvolution ni de coefficient par arme : rate son contrôle positif (VERDICT §4-5) ;
- pas d'attribution par trajectoire de projectile : bloquée par `object-position-component`
  ti=41 non bit-exact — travail de décodeur, hors périmètre (VERDICT §4ter, §9).

---

## 1. Mesure préalable — DÉTENTE vs BALLE (faite, tranchée)

**Question** (hypothèse utilisateur) : le film émet-il un `action_weapon_fire` par DÉTENTE
(une rafale BR75 = 1 event) ou par BALLE (une rafale = 3 events) ? Le dénominateur `shots_decoded`
n'est comparable à `shots_fired` de l'API que si les deux comptent la même chose.

**Instrument** : `apps/go-api/internal/analysis/filmdec/lot1_cadence_detente_research_test.go`
(garde `LOT1_TRAME_FILM`, borné 12 chunks, `-count=1`). Groupe les tirs 0xD2 longs par
(tireur `FilmIndex`, arme `WeaponID`) et mesure l'espacement inter-tir consécutif. Contrôle
positif intégré : le MA40 (automatique pur) DOIT sortir unimodal à sa cadence connue.

**Résultat (3 films, 12 chunks)** :

| arme | film | n écarts | médiane | forme |
|---|---|---|---|---|
| MA40 AR (auto, **contrôle**) | 01e1f945 | 557 | 83 ms | **unimodal** ~83 ms (353/557 en 60-90 ms) = 720 RPM nominal |
| BR75 (rafale de 3) | 000d5950 | 26 | 66 ms | **bimodal** : mode intra ~66 ms + mode inter 200-500 ms (9 écarts) |
| BR75 (rafale de 3) | 00502e52 | 18 | 67 ms | **bimodal** : 13 écarts en 60-90 ms + 5 en 200-1000 ms |
| Needler | 00502e52 | 59 | 83 ms | unimodal ~83 ms (temps de cycle) |
| Sidekick | 01e1f945 | 209 | 184 ms | unimodal (semi-auto = 1 détente = 1 balle) |
| VK78 Commando | 00502e52 | 23 | 150 ms | unimodal ~150 ms |

**VERDICT : le film émet un `action_weapon_fire` PAR BALLE (par round), pas par détente.**
Preuve : (1) le MA40 sort unimodal à sa cadence auto par round (contrôle positif = l'instrument
lit vraiment la cadence) ; (2) le BR75 est **bimodal** — un mode intra-rafale ~67 ms (les 3
rounds d'une même rafale) ET un mode inter-rafale 200-500 ms ; une émission par détente ne
montrerait QUE le mode inter-rafale, sans le mode ~67 ms. Les cadences BR75 67 ms, MA40 83 ms,
Sidekick 184 ms, Commando 150 ms reproduisent au ms près les cadences nominales connues
(VERDICT §3.2).

**Conséquence pour le dénominateur** : `shots_decoded` (film) est déjà par balle, **directement
comparable à `shots_fired` de l'API (par balle) — aucune conversion rafale→balle n'est requise.**
Le seul écart film↔API est un ÉCHANTILLONNAGE PARTIEL (le film ne capture pas tous les tirs :
déficit ~15 % même en hitscan pur, VERDICT §4bis.4), pas un décalage d'unité. C'est ce qui
justifie la porte ±10 % existante et le recalage sur l'API (§6), pas une correction de rafale.

---

## 2. À RÉCONCILIER AVANT DE CODER (blocage documentaire, pas technique)

Le doc-header de `apps/go-api/internal/analysis/filmdec/fire_events.go` (lignes 8-16) affirme :
« le fire event et le record de dégât 0xd2 sont LE MÊME RECORD, et il n'existe QUE lorsqu'un
dégât est appliqué (« touché » est une propriété de tous les records lus ici) ».

**Ceci est incompatible avec la feature ET avec la mesure.** Si tout 0xD2 était un dégât appliqué,
la précision par arme vaudrait 100 % partout. Or : (a) M1 de la NOTE apparie 0xD2 (tir) à un
`damage_aftermath` 0xC0 (dégât) SÉPARÉ, avec un taux de 17-51 % seulement ; (b) M2 mesure des
précisions 19-100 % variables par arme ; (c) la mesure §1 montre des tirs BR75 sans dégât apparié.
Donc **0xD2 = tir (fire), 0xC0 = dégât (damage_aftermath) sont deux records distincts** ; le
commentaire de `fire_events.go` reflète une lecture Ghidra ancienne, contredite depuis.

**Action Lot 0 (obligatoire, avant tout code de la feature)** : corriger le doc-header de
`fire_events.go` pour distinguer 0xD2 (tir) de 0xC0 (dégât) et retirer l'affirmation « touché est
une propriété de tous les records ». Anti-pattern visé : « doc inversée » (CLAUDE.md §9). Zéro
changement de comportement — commentaire seulement. Vérif : `go test ./internal/analysis/filmdec/`
reste vert (aucun code touché).

---

## 3. Périmètre et NON-périmètre

### 3.1 Phase 1 (ce plan, livrable)
- **Numérateur (touches) par arme** : un tir « touche » = il s'apparie à ≥ 1 `damage_aftermath`
  du même attaquant dans la fenêtre W (§4). Clé `WeaponID`. Grain match × joueur × arme.
- **Distance par arme** : distance tireur↔victime au ts du dégât apparié, en histogramme
  (buckets), par arme. Grain identique.
- **Classe « balle » uniquement** : armes dont le dégât passe par `damage_aftermath` (type 0) —
  ~2/3 de l'arsenal (Needler, Disruptor, Pulse Carbine, BR75, VK78, Heatwave, MA40, S7 Sniper,
  Sidekick, Commando…). La liste n'est PAS tenue à la main : c'est la porte par-arme (§5.3) qui
  tranche « capturée / non capturée » sur mesure.
- Greffé sur la passe film `killcollector` existante (§7.1), écrit dans une table sœur
  append-only de `match_weapon_shots` (§5).

### 3.2 NON-périmètre Phase 1 (explicite)
- `[!]` **Classe lourde/spéciale** (SPNKr, Hydra, Skewer, Ravager, Shock Rifle, Mangler, Stalker,
  Bulldog) : dégât absent de `damage_aftermath` type 0 (0 % même à W = 2 s, NOTE M2). Renvoyé
  Phase 2. La porte par-arme les marque « non capturée » — jamais 0 % faux.
- `[!]` **UI / surface web** : aucune page, aucun chart. La feature Phase 1 s'arrête aux agrégats
  persistés + un lecteur d'agrégat côté Go. La visualisation est un chantier séparé (décision
  produit ultérieure), conditionnée à un volume de corpus suffisant.
- `[!]` **Halo 5** : autre format de film, pas de décodeur (comme `film.kill_source`). Capability
  absente → dégradation propre (§8). Aucun code `slug ==`.
- `[!]` **Recalage écrit** : la table stocke les BRUTS (tirs appariables, touches, distances) +
  le verdict de porte ; la normalisation sur l'API (§6) est un calcul de LECTURE, hors périmètre
  d'écriture de ce plan.

### 3.3 Phase 2 (classe lourde) — voie du type 1 RÉFUTÉE (31/08)
Hypothèse initiale : `damage_section_response` (0xC0 type 1) porterait le dégât de la classe
lourde. **RÉFUTÉE** — `NOTE_DAMAGE_SECTION_RESPONSE_2026-08-31.md` : grammaire percée (oracle de
trame validé, 3,0/paquet), mais le type 1 ne porte QUE la victime (ref0 dom1), **AUCUN attaquant**
(ref1 dom8 / ref2 dom7 présentes 0 %), aucun tag source, aucune magnitude — une « réponse de
section » (section touchée + direction d'impact), pas un dégât autoritaire. Le lien tir-lourd↔type1
est du bruit (clé ref0==attaquant 3-10 %, ratio ~1× au témoin) → RATE ×3 films.
**La classe lourde reste NON couverte** : son dégât n'est ni dans le type 0 ni dans le type 1.
Piste restante (plus lourde, hors périmètre de ce plan) : les composants ECS de
projectile/détonation (`projectile_detonate` 0xC2 type 5 ; ou la position projectile ti=41, non
bit-exacte). À porter au `REGISTRE_REPORTS.md` comme piste ouverte, sans critère de reprise ferme.
En attendant, la porte par-arme (§5.4) marque ces armes « non capturée » — jamais 0 % faux.

---

## 4. Définition exacte de la « touche » et du « tir »

- **Tir** = un record `action_weapon_fire` 0xD2 LONG (variante 0, porte le `WeaponID` 64 bits et
  l'index tireur), tel que décodé par `fire_events.decodeFireEvent`. Unité = **PAR BALLE**
  (§1, tranché). Un tir est « appariable » s'il porte à la fois un attaquant (ref0 dom1) et un
  `WeaponID` lisibles.
- **Touche** = un tir qui s'apparie à ≥ 1 `damage_aftermath` (0xC0 type 0) dont le RESPONSABLE
  (ref1 dom1) a le même index brut d'attaquant que le tir, avec `|ts_dégât − ts_tir| ≤ W`.
  Un tir apparié à plusieurs dégâts compte pour **une** touche (précision, pas volume de dégât).
- **Fenêtre W = 1 s** (1 000 000 µs). **TRANCHÉ** : le dégât est horodaté à l'IMPACT, différé du
  tir (vol du projectile, tir soutenu, groupage du record). À 250 ms le taux sous-compte
  massivement ; à ~1 s il double-triple avec un ratio au témoin encore fort (~4-8×) ; à 2 s le
  témoin (fond de tir dense) rejoint (NOTE M1bis). W = 1 s est le point de fonctionnement.
  Un SWEEP 250/500/1000/2000 ms est conservé comme TEST de non-régression (le ratio au témoin à
  1 s doit rester ≥ 3), pas comme réglage libre.
- **Témoin** = même mesure décalée de +3 s (OFF). Le verdict de fiabilité porte sur le RATIO au
  témoin, jamais sur le taux absolu (plafonné par la densité des dégâts).
- **Distance** = `dist(pos_tireur, pos_victime)` au ts du dégât apparié ; tireur = ref1 (base
  512 → slot → position), victime = ref0 du dégât (base 512 → slot → position). Tolérance de
  lecture de position = 120 µs (`sondePosTolUS`). Une distance n'est calculée que si LES DEUX
  positions se résolvent (sinon le tir compte pour la précision mais pas pour la distance).

---

## 5. Schéma DuckDB des AGRÉGATS

### 5.1 DB cible et grain — TRANCHÉ
- **DB** : `data/titles/{slug}/warehouse/shared_matches_v2.duckdb` (via `PathResolver.SharedDBPath`),
  **à côté de `match_weapon_shots`**. Motif (db-schema) : la donnée est produite par la MÊME passe
  film que `match_weapon_shots` / `match_kill_events` (qui vivent dans shared), et elle couvre
  TOUS les joueurs d'un match, pas seulement un joueur (donc PAS la `stats.duckdb` par joueur, qui
  est réservée aux enrichissements individuels).
- **Grain** : match × `player_index` (réplication brut 5 bits) × `weapon_id` (filmshell 64 bits),
  exactement comme `match_weapon_shots`.

### 5.2 Table sœur vs extension de `match_weapon_shots` — TRANCHÉ : table sœur
Nouvelle table **`match_weapon_hits`** (append-only), NE PAS étendre `match_weapon_shots`. Motifs :
1. **Porte distincte** : `match_weapon_shots` porte une porte ±10 % sur le TOTAL de tirs du joueur
   (`shots_fired`). Les touches ont une validité DIFFÉRENTE (couverture par classe d'arme, W). Une
   colonne `touches` dans `match_weapon_shots` hériterait d'un `gate_reason` qui ne parle pas
   d'elle.
2. **`decoder_rev` distinct** : le décodeur de touches (pairing tir↔dégât) évolue indépendamment
   du décodeur de tirs — les confondre force un redécodage complet à chaque changement de l'un
   (motif déjà écrit pour `WeaponShotsDecoderRev` vs `KillSourceDecoderRev`).
3. **ADR 0026** : ajouter des colonnes à `match_weapon_shots` (append-only, peuplée en prod)
   impose la recette de rebuild `append_only_rebuild.go` sur une table vivante. Une table neuve
   n'a pas ce coût.

### 5.3 DDL (recette : calquer `migration/steps_shared_weapon_shots.go` + `append_only_rebuild.go`)
Colonnes de `match_weapon_hits` :

| colonne | type | rôle |
|---|---|---|
| `match_id` | VARCHAR | clé match |
| `decode_pass` | VARCHAR | id de passe (généra `newDecodePassID`) — retenu par `_latest` |
| `decoder_rev` | VARCHAR | version du décodeur de touches (constante neuve `WeaponHitsDecoderRev`) |
| `written_at` | TIMESTAMP | horodatage d'écriture (append-only) |
| `player_index` | SMALLINT | indice de réplication brut (0..31) |
| `player_xuid` | UBIGINT/NULL | xuid résolu, NULL = bot/non rattaché |
| `weapon_id` | UBIGINT | filmshell 64 bits (chaîne décimale à l'INSERT, cf. piège `ubigintArg`) |
| `shots_paired` | INTEGER | **dénominateur film** : tirs de cette arme APPARIABLES (attaquant+WeaponID lisibles) |
| `hits` | INTEGER | **numérateur** : tirs appariés à ≥ 1 dégât dans W |
| `window_us` | INTEGER | W utilisé (1 000 000) — grave le point de fonctionnement dans la donnée |
| `dist_bucket_json` | VARCHAR | histogramme de distance (bornes `sondeDistEdges`), JSON compact |
| `dist_n` | INTEGER | nb de touches à distance résolue (≤ hits ; base de l'histogramme) |
| `publishable` | BOOLEAN | verdict de la porte par-arme (§5.4) |
| `gate_reason` | VARCHAR | motif écrit (jamais recalculé par le lecteur) |

- **Vue `match_weapon_hits_latest`** (obligatoire, ADR 0026 : lecteurs → `_latest` uniquement).
- **`shots_paired` vs `shots_decoded`** : `shots_paired` (dénominateur des touches) ⊆
  `shots_decoded` (dénominateur des tirs), car une touche exige un tir APPARIABLE (attaquant ET
  WeaponID lisibles), sous-ensemble des tirs décodés. On stocke `shots_paired` dans CETTE table
  pour que la précision film = `hits / shots_paired` soit cohérente sans jointure ; la comparaison
  à `shots_decoded` reste possible par jointure sur (match, player_index, weapon_id).
- Pas de colonne « précision » ni « distance médiane » : ce sont des dérivés de lecture (comme
  `match_weapon_shots` refuse la colonne « total »).

### 5.4 Porte par-arme (le verdict, écrit une fois — motif `EvaluateShotsGate`)
Une ligne `match_weapon_hits` est `publishable` si :
1. l'arme est de **classe « balle »** — mesurée, pas listée : `hits > 0` OU (`shots_paired ≥ Nmin`
   ET l'arme apparaît dans l'ensemble « capturée » du film). Concrètement : `publishable = false`
   avec `gate_reason = "arme-non-capturee-classe-lourde"` si `shots_paired ≥ Nmin` et `hits == 0`
   (signature de la classe lourde : émet des tirs, aucun dégât type 0) ;
2. `shots_paired ≥ Nmin` (seuil d'effectif à FIXER dans le Lot 1 par mesure, ordre de grandeur
   ≥ 5-8, sinon `gate_reason = "effectif-insuffisant"`) ;
3. la passe de tirs du même joueur est elle-même dans la tolérance (jointure au verdict
   `match_weapon_shots.publishable`) — sinon `gate_reason = "porte-tirs-refusee"`. Une précision
   par arme n'a de sens que si le dénominateur tirs du joueur est fiable.

Le seuil `Nmin` et la règle exacte du (1) sont à ARRÊTER dans le Lot 1 par la mesure sur les 3
films (gate du lot), pas à l'exécution des lots aval.

---

## 6. Où vit le calcul, et le recalage sur l'API

### 6.1 Écriture (Phase 1)
- **Passe** : la MÊME passe film que `killcollector` (morts + tirs). Motif (shots.go) : le décodage
  d'un film est LA passe chère ; un producteur séparé re-lirait/re-décompresserait les mêmes
  chunks, et un film décodé deux fois a deux occasions de diverger. Le pairing tir↔dégât se greffe
  sur les chunks déjà tenus.
- **Décodeur de dégât — PRODUCTIONISATION REQUISE (dépendance interne)** : le décodage
  `damage_aftermath` + le pairing n'existent aujourd'hui QUE comme code de recherche `_test` dans
  `internal/analysis/filmdec` (`sondeScanDamage`, `lot1_attrib_arme_tir_research_test.go`,
  `lot1_degats_blesse_research_test.go`). La passe de tirs de prod, elle, utilise
  `analysis.ScanFireEventsB5` (package `analysis`, pas `filmdec`). Lot 2 = sortir le décodeur de
  dégât + le pairing du `_test` vers du code non-test (exporté), et l'appeler depuis
  `killcollector`. **Décision d'emplacement TRANCHÉE** : le décodeur de dégât et le pairing vivent
  dans `internal/analysis/filmdec` (cohérent avec l'existant du décodeur film, et avec l'item
  d'audit F12 qui veut migrer le pipeline film vers `games/halo_infinite/film/` — ne PAS anticiper
  cette migration ici, rester où vit le décodeur). `killcollector` (couche sync) orchestre :
  décode tirs + dégâts + positions → construit le `WeaponHitsBatch` → persiste.
- **Persister neuf** : `WeaponHitsPersister` (calqué sur `WeaponShotsPersister`), INSERT-only,
  porte appliquée DANS le persister (`EvaluateHitsGate`, unique copie), `PersistPass` + chemin
  `BatchBuilder`. Anti-ART (ADR 0019/0026/0030) : aucun DELETE/UPDATE/ON CONFLICT → rien à
  allowlister dans `no_art_patterns_test.go`.

### 6.2 Recalage sur l'API (calcul de LECTURE, hors écriture)
Le film ne voit qu'un ÉCHANTILLON des dégâts (§1). Donc :
- Les taux ABSOLUS de la table sont des captures partielles, à RECALER sur le total API
  (`match_participants.shots_fired` / `shots_hit`, l'ANCRE).
- **Ce que le film apporte = la FORME** : (a) la ventilation RELATIVE par arme À L'INTÉRIEUR de la
  classe « balle » ; (b) la distribution par DISTANCE. La forme n'est fiable qu'entre armes de la
  même classe, jamais entre classes (la lourde est absente).
- Le recalage est un futur lecteur `analysis` (title-agnostic) qui lit `match_weapon_hits_latest`
  + `match_participants` et redistribue. **HORS PÉRIMÈTRE d'écriture** ; le plan garantit seulement
  que la table porte les bruts nécessaires (tirs appariables, touches, distances) pour le
  permettre.

---

## 7. Architecture Go (couches)

| Rôle | Emplacement | Statut |
|---|---|---|
| Décodage tirs 0xD2 | `analysis.ScanFireEventsB5` (prod) / `filmdec.decodeFireEvent` (recherche) | existe |
| Décodage dégâts 0xC0 type 0 + pairing tir↔dégât + distance | `internal/analysis/filmdec` (à sortir du `_test`) | **Lot 2** |
| Orchestration passe (décode → batch) | `internal/sync/killcollector/hits.go` (neuf, calqué `shots.go`) | **Lot 3** |
| Type de résultat (batch) | `internal/persist/rows.go` : `WeaponHitsBatch` / `WeaponHitsPlayer` / `WeaponHitCount` | **Lot 3** |
| Écriture INSERT-only + porte | `internal/persist/weapon_hits_persister.go` (neuf) | **Lot 3** |
| DDL + vue `_latest` | `internal/migration/steps_shared_weapon_hits.go` (neuf) + `order.go` | **Lot 1** |
| Lecture/recalage | `internal/analysis` (futur) | hors périmètre |

- Aucun accès DuckDB dans un handler/service : tout par persister (écriture) / adapter (lecture).
- `PathResolver.SharedDBPath(slug)` pour le chemin, jamais `filepath.Join`.
- Logging `slog.*Context(ctx, ...)` : métriques de passe (lignes écrites, joueurs refusés, indices
  non résolus) sur le modèle des `shotsMetric*` d'observabilité (ADR 0009, entiers snake_case).

---

## 8. Multi-titre (capability, jamais slug)

- **Nouvelle capability produit** : `film.weapon_hits` = "supported" pour `halo_infinite`,
  ABSENTE pour `halo_5` (autre format de film — même raison que `film.kill_source`). Déclarée dans
  `config/titles/halo_infinite/mappings/capabilities.toml`.
- La passe et l'adapter branchent sur `HasCapability`/`CapabilityMap.Has("film.weapon_hits")`,
  jamais sur `slug == "halo_infinite"` (ratchet `no_slug_comparison_test.go`).
- Halo 5 : capability absente → la passe ne s'exécute pas, le lecteur renvoie
  `ErrCapabilityNotSupported` → dégradation propre (réponse partielle), jamais de panic ni de
  données d'un autre titre.
- Un test de capability calqué sur `shots_capability_test.go` garde ce branchement.

---

## 9. Réserves et risques (à porter dans toute lecture/UI future)

1. **Biais de classe** : la classe lourde est à ~0 % même à W = 2 s. Une précision « par arme »
   qui les afficherait à 0 serait FAUSSE. La porte par-arme (§5.4) les marque « non capturée » —
   ne JAMAIS afficher 0 % pour une arme non capturée.
2. **Fenêtre d'impact** : W = 1 s est un compromis. À documenter avec le ratio au témoin ; le
   sweep est un test de non-régression, pas un réglage.
3. **Une-ref** : ~33 % des dégâts (000d5950) n'ont qu'une ref d'en-tête → pas de distance, et si
   c'est la ref1 (responsable) qui manque, léger sous-comptage des touches. À porter comme réserve
   de couverture, pas à « réparer ».
4. **Échantillon film-seul petit** : 12 chunks donnent des effectifs faibles ; la mesure de
   viabilité vit sur un corpus, mais l'écriture est par match. Le recalage API (§6.2) est ce qui
   rend les taux exploitables.
5. **Cartes à signature ambiguë** : distances désactivées quand les bornes de carte ne se
   résolvent pas (ex. catalyst/deadlock, signature `[15 15 15]`). La touche compte quand même ;
   seule la distance manque.
6. **Modes non livrables** : Fiesta et grand format sortent massivement de la porte tirs
   (6,9 % / 38,8 % dans la tolérance) — la porte par-arme (§5.4 critère 3) les refusera par
   héritage. C'est correct : ne pas chercher à les récupérer.

---

## 10. Découpage en lots exécutables (gates)

> Contrat `plan-execution` : ordre strict, une étape à la fois, gate vérifié avant l'étape
> suivante. Aucune étape différée. Chaque item statué à la clôture (`[x]` fait / `[~]` couvert
> ailleurs / `[!]` non traité justifié). Zéro fix hors périmètre → §11.
> Cache Go privé au worktree avant tout `go`. Tests film : `-count=1`, `LOT1_TRAME_FILM` par film.

### Lot 0 — Réconciliation doc (bloquant, §2)
- [ ] Corriger le doc-header de `fire_events.go` : 0xD2 = tir, 0xC0 = dégât, retirer « touché
      propriété de tous les records ».
- **Gate** : `cd apps/go-api && go test ./internal/analysis/filmdec/ -run TestLot1 -count=1` vert ;
      `git diff --stat` ne montre que le commentaire.

### Lot 1 — Migration : table `match_weapon_hits` + vue `_latest` + seuils de porte
- [ ] `steps_shared_weapon_hits.go` (DDL §5.3) + enregistrement dans `order.go`.
- [ ] Vue `match_weapon_hits_latest` (retient la dernière `decode_pass`).
- [ ] ARRÊTER par mesure (sur 000d5950/01e1f945/00502e52) : `Nmin` et la règle exacte du critère
      (1) de la porte (§5.4). Documenter la valeur retenue dans le doc-header de la table.
- **Gate** : `go test ./internal/migration/ -run WeaponHits -count=1` vert (migration + `_latest`
      testées `:memory:`) ; `make go-api-lint` sur les fichiers touchés.

### Lot 2 — Décodeur de dégât + pairing, sortis du `_test` (productionisation)
- [ ] Exposer dans `internal/analysis/filmdec` (code non-test) : scan `damage_aftermath` (0xC0
      type 0) → événements (ts, attaquant ref1, victime ref0, magnitude) ; fonction pure de
      pairing tir↔dégât (W = 1 s) → touches par (index, WeaponID) ; distance tireur↔victime →
      buckets. Réutiliser `sondeScanDamage`/`lot1RefDom1`/`sondeBipedTracks`/`sondeDist` — ne pas
      dupliquer (≤ 2 copies, CLAUDE.md §6).
- [ ] Test unitaire pur du pairing (fixture minimale, sans DuckDB) : un tir apparié / non apparié
      / apparié hors W ; une touche unique pour N dégâts.
- **Gate** : `go test ./internal/analysis/filmdec/ -count=1` vert ; l'instrument de recherche
      `TestLot1AttribArmeTir` reproduit ses taux (M1 ratio témoin ≥ 3 à W = 1 s) via le décodeur
      productionisé (mêmes chiffres qu'avant sortie du `_test`).

### Lot 3 — Persister + orchestration passe + capability
- [ ] `WeaponHitsBatch`/`WeaponHitsPlayer`/`WeaponHitCount` dans `persist/rows.go`.
- [ ] `weapon_hits_persister.go` : `EvaluateHitsGate` (unique copie), `PersistPass`, chemin
      `BatchBuilder`, INSERT-only, UBIGINT en chaîne décimale (piège `ubigintArg`), validation
      (indice 0..31, sentinelles 0/1/2 refusées, doublon (index, arme) refusé).
- [ ] `killcollector/hits.go` : greffe sur la passe existante (mêmes chunks), construit le batch
      via le pairing du Lot 2, résout `player_index → xuid` par le résolveur existant
      (`resolvePlayerIndices`, ne pas redécoder), métriques `slog`.
- [ ] `capabilities.toml` : `film.weapon_hits = "supported"` (halo_infinite) ; branchement
      `HasCapability`, test capability calqué `shots_capability_test.go`.
- **Gate** : `go test ./internal/persist/ ./internal/sync/killcollector/ -count=1` vert ;
      `go test -tags=integration ./internal/persist/ -p 1` vert (anti-ART OBLIGATOIRE avant
      livraison persist) ; `no_slug_comparison_test.go` + `no_art_patterns_test.go` verts.

### Lot 4 — Livraison
- [ ] `make go-api-lint` + `make go-api-test` verts ; suite complète `go test ./...` verte.
- [ ] Entrée `thought_log.md` (date, titre, statut, décision, résultats, prochaine étape).
- [ ] MAJ `MEMORY.md` / index `.ai/V7.5/` (feature precision/distance par arme livrée Phase 1) ;
      report Phase 2 (classe lourde / type 1) au `REGISTRE_REPORTS.md` avec critère de reprise.
- **Gate** : `skill delivery-checklist` passé ; CI verte au niveau JOB (mode branche unique v7.5).

---

## 11. Découvertes hors périmètre (à consigner, NE PAS traiter)
- (Réservé à l'exécution.) Toute anomalie rencontrée hors des lots ci-dessus s'écrit ici avec sa
  localisation ; elle ne se corrige pas dans ce chantier (CLAUDE.md règle 7 / diagnostic n°8).

## 12. Protocole de reprise de session
- Avancement : cases `[ ]` de §10 (statuer `[x]`/`[~]`/`[!]` à la clôture de chaque lot).
- Reprendre au premier lot non clos ; un lot n'est clos que son gate vert ET ses items statués.
- Contexte figé : `NOTE_ATTRIBUTION_ARME_TIR_2026-08-31.md` (méthode), `VERDICT_PRECISION_
  PROJECTILES.md` (ce qu'il ne faut pas refaire), ce plan (décisions tranchées).
