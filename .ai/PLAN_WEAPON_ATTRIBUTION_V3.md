# PLAN — Weapon-of-kill attribution v3 (SHADOW, indépendant de la prod)

> Objectif : remonter la confidence HIGH de l'attribution d'arme-de-kill (~71.5% → viser ~90%) en fiabilisant la
> DONNÉE SOURCE, **sans toucher au pipeline v2-legacy**. La v3 est développée et évaluée en parallèle ; elle n'est
> promue (remplacement v2) que **si les résultats sont concluants**, par un switch explicite.
> Réfs : `.ai/REFERENCE_WEAPON_IDS.md`, `.ai/RESEARCH_THEATER_RE.md` (§C grenade, §K-bis melee, §L paquets µs).
> Branche : `feat/weapon-attribution-v3`. Câblage prod = GATE utilisateur.

## 0. Critère de succès / promotion (la "done definition")

La v3 est **concluante** (donc promouvable) si, sur un panel de validation (≥ 20-50 matchs, multi-modes), elle obtient
TOUTES les conditions :
- **HIGH ≥ 88%** des kills (vs 71.5% v2), ou à défaut un plancher mesuré ≥ 82% avec un chemin clair vers 88%.
- **NULL/no-weapon ≤ 5%** (vs 14.2% v2) — melee/grenade sortis du NULL.
- **Aucune régression** : sur les kills où v2=HIGH, v3 doit donner la MÊME arme (high-32) à ≥ 98%.
- **Cohérence agrégats** : par joueur, melee/grenade attribués v3 ≈ `match_participants.melee_kills/grenade_kills` (±1) ;
  somme des kills par arme cohérente avec les agrégats API quand disponibles.
- **Armes distinctes** : ~37 (vs 174), 0 id "bruit" en HIGH.

Tant que ces seuils ne sont pas atteints, la v2-legacy reste la source active. La décision de flip est **manuelle**.

## 1. Contrainte d'architecture — INDÉPENDANCE TOTALE

### Code v2-legacy GELÉ (aucune modification)
`internal/analysis/{weapon_scanner,weapon_parser,weapon_correlation,weapon_reconciliation,kill_attribution,weapon_data}.go`,
`internal/sync/backfill_weapons.go`, `internal/platform/duckdb/weapon_kills_repo.go`, vue `v_weapon_kills`, table
`weapon_kills`. Le pipeline live (PostSync `processWeaponKillsInline`, backfill `--weapons`) continue d'écrire `weapon_kills`.

### Code v3 NOUVEAU et ISOLÉ
- **Package `internal/analysis/weaponv3/`** (algos purs, 0 accès DB, 0 Streamlit) :
  - `melee_scanner.go` — décodeur §K-bis (ancre 0x34/0x35, type@+76 ∈{0x42,0x47,0x60}, weapon-id offsets
    {0x42:[88],0x47:[86],0x60:[101,103]}, pi=octet@+20 bits0-4) → `MeleeHit{PI, TimeMS, WeaponID}`.
  - `grenade_scanner.go` — décodeur §C (marqueur 0x4c0c00, weapon@+24, allowlist {Frag 0xB0171062, Plasma 0xC0E34C44,
    Shock 0x3B2567D4, Spike 0x9212E428}) sur TOUS les chunks type-2 → `GrenadeHit{PI, TimeMS, WeaponID}`.
  - `fire_scanner.go` — fire events DURCIS : weapon-id (high-32 validé), `PI`, `Slot`, **bit hit** (3e + bit de
    confirmation §acurtis), **bit burst-final**, `ShotCounter` (0-127). Réutilise le marqueur 11-bit existant mais
    expose hit/burst (que la v2 ignore).
  - `timing.go` — timestamps **µs-précis** via l'en-tête de paquet 16 octets (§L : `[Type u16][b2][b3][Size u32][µs u64]`),
    chaque event daté par le paquet FRAME qui le contient (remplace le bucketing frame-marker grossier).
  - `canon.go` — identité = **high-32** ; valide high-32 ∈ set d'armes connues même pour le suffixe commun `0x42c9679f` ;
    fold des variantes par high-32 ; rejette le bruit (→ unresolved honnête).
  - `correlate.go` — orchestration d'attribution v3 (cf. §2), produit `[]AttributionV3`.
  - `attribution.go` — type `AttributionV3` (superset de `KillAttribution` : + `SourceSignal`, `KillingShotHit`,
    `BurstFinal`, `ShotCounter`, `HighWeaponID`).
- **Table séparée `weapon_kills_v3`** (même schéma que `weapon_kills` + colonnes v3 ; migration additive, ne touche pas
  `weapon_kills`). Repo lecture/écriture dédié `internal/platform/duckdb/weapon_kills_v3_repo.go`.
- **CLI diagnostic SEUL point d'entrée** : `cmd/diag_weapons_v3/` — exécute la v3 sur N matchs (flag `--match`/`--all`/`--sample`),
  écrit `weapon_kills_v3`, et imprime le **rapport de comparaison v3 vs v2** (§4). **AUCUN câblage dans le sync/backfill live.**

### Switch de promotion (par défaut OFF)
- Feature flag `LEVELUP_WEAPON_ATTRIB` ∈ {`v2` (défaut), `v3`}. Lu UNIQUEMENT au point de lecture (la vue / le repo de
  lecture des stats d'arme). En `v3` : `v_weapon_kills` (ou un `v_weapon_kills_active`) pointe sur `weapon_kills_v3`.
- Le flip n'est fait QUE par l'utilisateur après validation (§0). Réversible instantanément (flag → v2).
- Optionnel à la promotion : re-backfill `weapon_kills` depuis la v3 (sweep existant) pour retirer la double-table.

## 2. Algorithme v3 (ordre de résolution par kill)

Pour chaque kill (highlight event : killer xuid + time_ms ; `kill.XUID` = TUEUR) :
1. **Melee** : si un `MeleeHit` du tueur (pi) existe dans [t-W_m, t] → arme RÉELLE (épée/marteau/crosse), `SourceSignal=melee`,
   confidence **HIGH**. (W_m petit, ~1s.)
2. **Grenade** : sinon si un `GrenadeHit` du tueur dans [t-W_g, t] → arme grenade réelle, `SourceSignal=grenade`, **HIGH**.
3. **Fire (tir fatal)** : sinon, parmi les fire events du tueur (pi) dans [t-W_f, t] :
   - ne garder que les **tirs TOUCHÉS** (bit hit) ; préférer le **dernier tir burst-final touché** avant t.
   - delta µs-précis → confidence : delta < SwapMS = **HIGH** ; ≤ TravelMax = MEDIUM ; sinon LOW. `SourceSignal=fire`.
   - claim-and-remove par pi (anti-vol entre coéquipiers).
4. **FormulaA durci** : sinon snapshot per-chunk MAIS high-32 validé ∈ armes connues (sinon rejeté) → MEDIUM/LOW, `SourceSignal=formula_a`.
5. **Sinon** : `none`/NULL honnête.
- **player_index** : valider le mapping pi↔xuid contre le film (ancre RE : fire-pi corrélé aux morts ; ≠ ordre DB présumé)
  AVANT corrélation — sinon mis-attribution silencieuse. Si non validable, fallback explicite + flag low.
- **Pas de réconciliation API** (droppée). `swap_detected` = diagnostic. Loadout-victime (burst de mort) = filtre NÉGATIF optionnel.

## 3. Phases (ordonnées, chacune testable + réversible, 0 impact prod)

| Phase | Contenu | Vérifié ? | Done |
|------|---------|-----------|------|
| **P0** | Échafaudage : package `weaponv3/`, table `weapon_kills_v3` (migration additive), CLI `diag_weapons_v3` (no-op qui copie v2), rapport de comparaison §4 vide | — | CLI tourne, écrit la table, compare (égalité triviale) |
| **P1** | `melee_scanner.go` + `grenade_scanner.go` + branchement dans `correlate.go` (étapes 1-2) | chemin mort v2 confirmé ; décodeur melee validé 1 match | sur 000d5950 : 4 melee API + grenades → vraies armes HIGH (vs NULL en v2) |
| **P2** | `timing.go` µs (§L) + tir-fatal hit/burst dans `fire_scanner.go`/`correlate.go` (étape 3) | parser µs vérifié ; **taux à mesurer** | sur 10-20 matchs : mesurer la part des fire-events soft<2s reclassés HIGH |
| **P3** | `canon.go` high-32 + rejet bruit FormulaA (étape 4) | vérifié (137 bruit, 1 variante) | armes distinctes 174→~37 ; 1 207 junk → NULL ; Sentinel Beam variant foldé |
| **P4** | Rapport de comparaison complet + panel de validation (§4) | — | métriques §0 calculées sur le panel |
| **P5 (GATE)** | Switch `LEVELUP_WEAPON_ATTRIB=v3` + (option) re-backfill `weapon_kills` | — | **décision utilisateur** sur preuve §0 |

P1 et P3 sont les gains "sûrs" (vérifiés). P2 est le lever vers 90% mais **non garanti** — sa passe de mesure (P2/P4)
décide si 90% est atteignable ou si on plafonne à ~82%.

## 4. Méthodologie de comparaison (le rapport `diag_weapons_v3`)

Sur chaque match du panel, calcule et imprime (v3 vs v2, + ground truth) :
- Distribution confidence (high/medium/low/none) — v3 vs v2.
- % NULL/no-weapon, # armes distinctes effectives.
- **Agreement** : sur les kills v2=HIGH, % où v3 donne le même high-32 (régression si < 98%).
- **vs agrégats** : melee/grenade per-joueur v3 vs `match_participants.{melee,grenade}_kills` ; somme par arme vs API si dispo.
- **P2 lever** : # fire-events soft (v2 low/medium <2s) reclassés HIGH par le timing µs.
- Liste des kills passant de NULL→résolu (melee/grenade) et de soft→HIGH.
Panel : inclure 000d5950 (Slayer Super Fiesta, ground-truth K/D), des CTF (`53ce4390`, `64e8adfa`), des Strongholds,
des modes variés, + des matchs à forte densité de kills (test claim-and-remove/pi).

## 5. Grille plan-review

- **Couches** : algos `internal/analysis/weaponv3/` (purs) ✓ ; type résultat `AttributionV3` (domaine) ✓ ; pas de DuckDB
  dans l'algo (le CLI/repo fait l'IO) ✓ ; handler — N/A (CLI diagnostic, pas d'endpoint v3 tant que non promu).
- **Multi-titres** : chemins film via `PathResolver` (jamais `filepath.Join` direct) ; gate sur capability "film
  disponible" (pas sur `slug=='halo_infinite'`) ; dégradation `ErrCapabilityNotSupported` si pas de film.
- **Tests** : `weaponv3/*_test.go` unitaires (décodeurs melee/grenade/fire sur fixtures film + vecteurs acurtis ;
  canon high-32 ; timing µs sur un chunk connu) ; `weapon_kills_v3_repo` test DuckDB `:memory:` ; le CLI a un test de
  comparaison sur un petit match fixture.
- **Logging** : `slog.InfoContext` (résumé par match : counts par signal/confidence), `slog.ErrorContext(ctx,...,"err",err)` ;
  pas de `fmt.Println` (hors sortie CLI volontaire du rapport).
- **Livraison** : `thought_log.md` à chaque phase ; aucune dépendance externe bloquante ; v2 intacte à tout instant.

## 6. Risques / inconnues

- **Lever µs (P2) NON garanti** : le taux de reclassement soft→HIGH est l'inconnue dominante ; mesuré en P2/P4 avant
  toute promesse de 90%. Si faible → on plafonne ~82% (toujours mieux que 71.5%, sans régression).
- **player_index** : si le mapping pi↔xuid film n'est pas validable de façon fiable, l'attribution fire reste exposée
  à des mis-attributions ; à mesurer (agreement v2=HIGH) — un agreement bas révélerait le problème pi.
- **Melee/grenade completeness** : les marqueurs sont précis (~0 faux positif) mais épars ; valider la couverture vs
  les agrégats per-joueur (un manque = sous-attribution, pas une fausse attribution).
- **Coût re-backfill** (P5, à la promotion) : 1 141 matchs, sweep existant (DELETE-then-INSERT, sérialisé) — non bloquant
  car hors chemin live et déclenché manuellement.

## 7. P2 MESURÉ (2026-06-02) — verdict + priorité révisée

Workflow P2 (shadow, validateur `cmd/tmp_p2valid`/`tmp_p2sample` qui rejoue le VRAI pipeline internal/analysis sur 27
matchs cachés, 2× : bucket vs µs ; aucune écriture prod) :

- **Le timing µs MARCHE, gain réel et large.** Le bucket frame-marker actuel **dérive DANS chaque chunk** (slop médian
  183ms, p95 703ms, max 1728ms — pas frame-level ; nMarkers ≠ vrai nb de FRAME). Décodeur µs validé : marcher les paquets
  16o `[Type u16][b2][b3][Size u32][µs u64]`, dater chaque fire par le FRAME le contenant, **ancrer sur `start_ms` du
  manifest** (PAS (idx-1)*20000 → sinon deltas négatifs). 100% des fires sont dans un FRAME (605/605).
- **Lift chemin fire_event (NOPI propre, 1532 kills/5 matchs)** : HIGH **78.4% → 92.6%** (+14.2pp), **77% des soft
  récupérés** (delta s'effondre de ~1300ms de dérive à ~100ms). → **~90-93% HIGH sur le chemin fire_event** ; sur le mix
  global : 71.5% → **~85-88%**.
- **⚠️ DEUX correctifs OBLIGATOIRES, révélés par P2 :**
  1. **`player_index` est CASSÉ (index-spaces différents) — c'est désormais le levier #1, avant le µs.** Le pi du fire
     event = un **slot film 4-bit (0-15)** ; `getXuidToPI` assigne **0..N-1**. Espaces DIFFÉRENTS → les matchs >16 joueurs
     dumpent les tueurs pi>15 entièrement en formula_a, et les petits matchs mis-claim des fires à pi coïncident. Sur le
     chemin **realPI (= prod actuel + un re-backfill v3)** : HIGH chute à **41.2%**, µs ne récupère que 21.3%, régressions
     **24%**. → **Sans corriger le mapping pi↔xuid (via le slot film réel, pas l'ordre DB), le µs ne sert quasi à rien.**
  2. **Garde de stabilité des claims** : re-timer change quel fire chaque kill réclame (claim-and-remove) → **4.1% des
     v2-HIGH régressent** (net +207 HIGH quand même). Épingler les claims v2-HIGH ou re-ranker par temps µs de façon
     monotone pour supprimer cette churn.
- **Résidu (NOPI, kills qui restent soft)** : 54% claim-starvation (le fire du kill dédup/déjà réclamé), 34% armes à
  temps de vol (Needler/SPNKr/Cindershot — dégât arrive bien après le tir), 12% sniper/long-swap. **NON corrigeables par
  le timing** — il faut un meilleur recall des fires + un claiming par-pi.

**VERDICT v3** : **~88-92% HIGH est ATTEIGNABLE**, mais l'ordre est : **(1) fixer `player_index`** (le slot film 4-bit,
prouvé cassé et bridant), **(2) timing µs** (+14pp), **(3) garde de claims** (anti-régression), **(4) grenade/melee +
canon** (le slice no-weapon). Le 90% n'est PAS un gain « gratuit du µs » : il dépend d'abord du fix pi. Les armes à temps
de vol + claim-starvation plafonnent en-dessous de 100%.

## 8. PI-FIX RÉSOLU — méthode acurtis, VÉRIFIÉE (2026-06-02)

Le `player_index` correct = **les 5 bits immédiatement AVANT la représentation 64-bit LITTLE-ENDIAN du xuid, recherchée
au niveau BIT (pas byte-aligned) dans le chunk gameplay**. (acurtis `get_player_index` : `bits.find(uintle(xuid,64))`
puis `bits[pos-5:pos].uint`.) **Vérifié sur 000d5950 chunk_01** : les 8 xuids → pi 0-7 distincts, sans conflit, et
…0022→2 = exactement l'ancre fire-pi=2 (death-gap, p=0.006). Map 000d5950 : pi0=…4760703, pi1=…7245250, pi2=…0022,
pi3=…0284321, pi4=…5845110, pi5=…4178793711, pi6=…2097883, pi7=…0416.
- **PAS de formule structurelle** (pi ≠ team*4+b36 ; pi=2=team0/b36=1, pi=7=team0/b36=0). Index assigné par match →
  **résoudre par-match** via la recherche bit-level du xuid, JAMAIS l'ordre DB.
- **IMPLÉMENTATION v3** : remplacer `getXuidToPI` (ordre DB, cassé) par un résolveur qui, pour chaque xuid du roster,
  bit-search le chunk pour `uintle(xuid,64)` et lit les 5 bits précédents → `xuid→pi` ; inverser pour `pi→xuid`. Le pi du
  fire event (byte5>>4, à élargir à 5 bits si >16 joueurs) matche alors le killer correctement → débloque le chemin realPI
  (qui était à 41% HIGH) et permet au timing µs (+14pp) de porter ses fruits. C'est le levier #1, désormais RÉSOLU.
- Outil de vérif : tmp_film_explore/piverify/ (bit-level xuid search + 5-bit-before read). Bots (`bid…`) → pi=-1 (TODO).

## 9. FIRE-EVENT RECALL — marqueur trop strict (acurtis→JGtm, 2026-06-02)

acurtis : « byte 0 validation may be too strict — only the final 3 bits are constant for the event start. » → le
marqueur de début de fire-event (prod `ScanFireEventsB5` = marqueur 11-bit `0b10100100110`, weapon_scanner.go:217-298,
+ la validation du 1er octet) est **trop strict** : seuls les **3 derniers bits du byte 0** sont invariants ; exiger le
pattern complet **rate des fire-events valides** (bits de tête variables) → recall sous-optimal.
- **LIEN P2** : c'est précisément le résidu dominant mesuré — **54% des kills qui restent soft = « claim-starvation »**
  (le fire propre du kill a été déduplé/raté → le kill réclame un fire plus ancien). Améliorer le RECALL (relâcher le
  marqueur aux 3 bits constants) **récupère une partie de ces soft** → pousse le plafond au-dessus des ~92% du chemin µs.
- **IMPLÉMENTATION v3** (à ajouter au fire_scanner) : détecter l'event-start sur les **3 bits constants** seulement, puis
  valider l'event a posteriori (weapon-id ∈ set connu via le suffixe `42c9679f` + high-32 connu, pi ∈ 0-31) pour écarter
  les faux positifs que la stricte-validation évitait. Mesurer recall (fires trouvés) + FP avant/après sur le panel.
- Mises à jour fire-event v3 (récap) : (a) relax marqueur 3-bit (recall), (b) bits hit/burst-final (tir-fatal),
  (c) pi via le 5-bit-avant-xuid (§8), (d) timing µs (§7), (e) garde de claims. Ensemble → vise la borne haute ~90%+.

## 10. LIVRABLE SIBLING — table `match_objective_events` (net-new, mode-agnostique)

Persister les timelines d'events objectif décodées (CTF captures, Strongholds zones, KOTH collines, Oddball crâne) dans
UNE table générique, extensible sans migration (on ne connaît pas tous les events). **Net-new** : aucun consommateur
actuel (les events objectif ne sont pas stockés ; `highlight_events` = kill/death/medal/mode seulement) → additif, zéro
conflit v2, peut vivre dès maintenant en parallèle.

**N:1 — un event peut impliquer PLUSIEURS joueurs** (ex. Strongholds : plusieurs coéquipiers capturent une zone
ensemble). Donc le `xuid` n'est PAS dans l'event (team-level) mais dans une **table enfant `participants`** (0..N, avec rôle).

```sql
-- 1) Event = quoi/quand/quelle équipe (team-level)
match_objective_events (
  match_id VARCHAR, seq INTEGER, time_ms INTEGER,
  objective_type VARCHAR,   -- parent: flag|zone|hill|skull|bomb (extensible)
  event_type     VARCHAR,   -- action: capture|grab|return|pickup|drop|score_tick|contest|… (extensible)
  team_id INTEGER,          -- équipe qui marque/agit (la vérité ; nullable si neutre)
  objective_id INTEGER,     -- zone#/flag-color/hill# — SOUVENT NULL (zone Strongholds non récupérable)
  value INTEGER,            -- +1 capture / delta points / secondes possession — nullable
  source VARCHAR,           -- burst|th10|score_counter|score_rate (provenance)
  confidence VARCHAR,       -- exact (ms) | approx (~20s)  ← CRITIQUE : la précision varie par mode
  details JSON,             -- extras non modélisés (agnostique)
  written_at TIMESTAMP, PRIMARY KEY (match_id, seq)
)
-- 2) Joueurs impliqués = 0..N par event, avec rôle (gère le multi-capturer)
match_objective_event_players (
  match_id VARCHAR, seq INTEGER,   -- FK -> match_objective_events
  xuid VARCHAR,
  role VARCHAR,                    -- scorer|capturer|assist|carrier|contributor
  PRIMARY KEY (match_id, seq, xuid)
)
```
(Alternative DuckDB-native : une colonne LIST `participants STRUCT(xuid,role)[]` dans l'event, requêtable via UNNEST —
plus simple, sans jointure, mais moins conventionnelle ; on retient la table enfant pour rester relationnel + role-aware.)
- **Type à 2 niveaux** (objective_type parent + event_type action) ; le mode se dérive de `match_registry.game_variant_name`.
- **`source`+`confidence` obligatoires** : CTF = ms-exact (burst tiers==6) ; Strongholds/KOTH/Oddball = ~5-20s approx
  (heartbeat / inflexion de pente). Sans ça un consommateur sur-ferait confiance à un event ±20s.
- **Append-only** (PK technique seq + written_at, INSERT pur, DELETE-then-INSERT par match au re-backfill) — règle anti-ART.
- **Couches** : extraction algo `internal/analysis/objectiveevents/` ; type canonical ; repo
  `platform/duckdb/objective_events_repo.go` ; ingestion via le CLI diagnostic v3 (shadow), PAS le pipeline live.
- **Multi-titres** : schéma générique + capability-gated (film + modes objectif), pas sur le slug.
- Source des données = la timeline unifiée produite par le workflow `wc02iibum` (CTF team + Strongholds/KOTH/Oddball events).

## 11. BACKFILL des matchs DÉJÀ en base

- **Périmètre** : les ~942 matchs avec films cachés (+ footers backfillés). Les matchs sans film (66 expirés + ceux jamais
  fetchés) → pas de données film = **dégradation gracieuse** (lignes absentes, jamais d'erreur). Couverture rapportée.
- **Job** : un sweep batch (le CLI diagnostic v3 en mode `--all`, ou `cmd/diag_weapons_v3 --backfill`) qui, par match caché :
  (1) résout le pi par-match (acurtis 5-bit-avant-xuid), (2) attribue les armes → `weapon_kills_v3`, (3) extrait les events
  objectif → `match_objective_events`(+`_players`). **Idempotent** (DELETE-then-INSERT par match), **resumable**, sérialisé
  par lease + MaxOpenConns(1), **déclenché manuellement, hors chemin live**.
- **Shadow strict** : écrit UNIQUEMENT dans les tables v3/net-new ; ne touche JAMAIS `weapon_kills`/v2 → backfillable sur
  TOUS les matchs sans risque prod, à tout moment, même avant la promotion.
- **Promotion découplée** : `match_objective_events` (net-new, 0 conflit v2) peut être considéré pour usage prod dès
  validation ; `weapon_kills_v3` reste shadow jusqu'au switch `LEVELUP_WEAPON_ATTRIB=v3` (cf. §1, gate sur preuve §0).
- **Re-runs** : à chaque amélioration de décodeur (recall, hit/burst, zone-id…), re-backfill le sous-ensemble concerné
  (ou tout) ; les tables étant append-only-par-match-DELETE+INSERT, c'est rejouable proprement.

## 12. FRONTEND (préparation — viz à définir)

Distinction de promotion qui pilote le frontend : **(A) events objectif = surfaçables indépendamment** (net-new) ;
**(B) arme-de-kill v3 = derrière le switch** (shadow jusqu'à preuve). Le front peut donc livrer l'objectif d'abord.

- **API** (handlers fins → service orchestre → types canoniques via adapter ; aucun SQL inline en service) :
  - `GET /matches/{id}/objective-events` → timeline events+players → alimente la **match view** (timeline + « qui tient
    quelle base à T » par replay) et les agrégats. (A — dispo dès validation.)
  - `GET /matches/{id}/weapon-kills` → kill-feed avec arme + confidence/source. (B — gated promotion.)
  - `GET /players/{slug}/weapon-stats` + `/objective-stats` → agrégats per-joueur (Solo). Variantes escouade (Escouade).
  - Types canoniques nouveaux : `canonical.ObjectiveEvent`, `canonical.WeaponKill` (+ FieldKeys + TOML mappings).
- **Pages × data candidate** (à designer, pas figé) :
  - **Synthesis** : agrégats globaux — distribution arme-de-kill, totaux/ratios d'events objectif (captures, holds).
  - **Solo** : breakdown arme-de-kill du joueur (camembert/temps), participation objectif (captures/carries/holds), heatmap positions.
  - **Escouade** : stats arme/objectif par coéquipier + synergie (qui capture quoi, qui couvre).
  - **Match view** (le plus riche) : timeline d'events sur axe-temps, **ruban de contrôle de zone** (owner A/B/C à T),
    kill-feed avec arme, heatmap positions/morts.
- **Conventions** (grille plan-review §6/§7) : query keys dans `apps/web/src/lib/query/keys.ts` ; strings i18n **FR+EN** ;
  couleurs via `tokenCssVar`/`resolveToken` (zéro hex) ; routes file-based ; labels via `useFieldLabel`/`useOutcomeLabel` ;
  **capability-gated** (film disponible) → dégradation propre si pas de données film.
- **Honnêteté UI** : exposer `confidence`/`source` (ex. event objectif ±20s vs ms ; arme-de-kill low-confidence) pour ne
  PAS survendre la précision — un badge/tooltip plutôt qu'une fausse exactitude.

## 13. DOMINANCE FLAGS objectif — ÉTENDRE l'existant (NE PAS réinventer)

Le système EXISTE : `internal/analysis/comeback.go` (port de `comeback_analysis.py`) calcule `dominance_flag (0-5)` depuis
une **courbe de score** (`BuildScoreSnapshots`) : 1=DOMINATION, 2=HUMILIATION, **3=REMONTADA**, **4=DÉBÂCLE(collapse)**,
**5=CONTRE-REMONTADA** (+ seuils par sensibilité relaxed/standard/strict). Persistance via `internal/sync/comeback.go` +
`comeback_postsync_persist.go` ; affichage via `service/match_view_dominance` + `narrative/dominance.go`.
- **Aujourd'hui** : la courbe est reconstruite depuis les **kill-events** (highlight_events) → ne marche que pour les
  modes frags. La logique de flag elle-même est **agnostique du mode** (elle opère sur des `ScoreSnapshot{TimeMS, Team0, Team1}`).
- **EXTENSION (le travail)** : pour les modes OBJECTIF, alimenter `BuildScoreSnapshots`/l'analyse avec le **score-over-time
  objectif décodé** (CTF byte883/captures ; Strongholds byte842 varint×4.099 ; KOTH meters ; Oddball intégration de
  possession). Les seuils (leadPct/comebackPct) s'appliquent à l'échelle objectif (~200). Les flags
  remontada/débâcle/contre-remontada deviennent alors disponibles pour CTF/Strongholds/KOTH/Oddball.
- **Source de la courbe** : la courbe objectif vient des tables `match_objective_events` (events) + le score décodé
  per-chunk (TYPE_2). Un `ScoreSnapshot` par chunk (~20s) suffit pour détecter les retournements (la marge croise 0,
  retard max, etc.). CAVEAT résolution : un changement de tête bref entre deux keyframes (~20s) peut être manqué → flag
  conservateur (préférer rater un micro-flip que d'en inventer un).
- **Net-new vs prod** : l'extension touche le calcul de `dominance_flag` pour les matchs objectif. Comme c'est un champ
  déjà existant alimenté en post-sync, l'ajouter pour l'objectif = soit (a) en shadow (recalcul diagnostic, comparé), soit
  (b) directement si on juge la courbe objectif fiable (le score est validé ±1 sur Strongholds/KOTH, exact sur CTF). À
  décider à la promotion. Vérifier d'abord `comeback.go` + `comeback_postsync_persist.go` pour brancher la courbe objectif
  au bon endroit (NE PAS dupliquer la logique de flag).
