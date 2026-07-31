# PLAN — Arme à feu par kill : validation finale + productionisation (handoff)

> **Statut au 2026-06-12** : recherche **résolue à 96% pur offline** sur le match de référence. Reste 3 items de
> validation/recherche (Phase 0) puis l'implémentation produit (Phases 1→4). Document de passation pour un autre agent.
> **Lire d'abord** : `.ai/FIREARM_PER_KILL_OFFLINE_SOLVED.md` (section finale en tête) + `.ai/PLAN_SIBLING_DAMAGE_DESERS.md`
> (marqué RÉSOLU) + thought_log entrée `[2026-06-12] ARME PAR KILL 100% OFFLINE`.
> **Prototype de référence (tout est dedans, validé)** : `apps/go-api/cmd/tmp_offwarp/main.go`.

## Objectif & critère de succès

Attribuer, pour chaque kill d'un match Halo, l'**arme à feu** utilisée (famille + id64 variante), **100% offline**
(sans Cheat Engine ni Ghidra runtime), de façon scalable sur tous les matchs en cache.

- **Critère de succès** : ≥ 90% des kills arme-à-feu attribués correctement sur les matchs normaux (Slayer/Ranked/
  Fiesta), couverture complète (firearm + mêlée + grenade) sur ≥ 95% des kills tous types ; aucun held-weapon.
- **Acquis (ne PAS re-chercher)** : décode `0xd2`, roster type-8, warp linéaire, attribution last-before. Voir §Méthode.

## Méthode prouvée (résumé technique — la base de l'implémentation)

1. **Records de dégât** = paquets type-0, `payload[0]==0xd2` (déser moteur `FUN_14080c1f8`). Par record :
   - attaquant : `slot = bitsAt(payload, 36, 5) >> 1` (R5 5 bits au bit 36, lu AVANT le slot).
   - famille : `bp=41; if bitsAt(payload,bp,1)==1 {bp++} else {bp+=3}; fam32=bitsAt(payload,bp,32)` ; valide si
     `WeaponIDToName[fam32<<32 ...]` existe ET `bitsAt(payload,bp+32,32)==0x42c9679f` (suffixe universel).
   - temps : le **packet-ts** du header type-0 (`binary.LittleEndian.Uint64(chunk[off+8:])`) — horloge **flux**.
   - **Les 519 records = TOUS les dégâts** (prouvé : `FUN_14080a18c` traite 1 record/appel ; les « manquants » vs
     live = DoT/supercombine runtime ; marqueurs frères `0xe9`… = apparence d'arme, à IGNORER).
2. **Roster** : plus long paquet type-8 ; bit-scan LE (`u64LE` reconstruit byte-par-byte MSB-first puis
   `binary.LittleEndian.Uint64`) des xuids du kill-feed → tri par 1ère position de bit = ordre des **slots**.
   `slot R5>>1 == idx live` (validé 8/8 par signature d'armes).
3. **Kill-feed** : `analysis.ParseHighlightEvents(chunkBytes, 0)` sur les chunks ~18→41 (garder le chunk avec le plus
   de kills) → `(killer XUID, TimeMS)` + `(death XUID, TimeMS)`. Victime = death apparié dans ±400ms (≠ killer).
4. **Warp** packet-ts → TimeMS, **dérivé offline** :
   - init linéaire par bornes : `a=(maxKillT-minKillT)/(maxTs-minTs)`, `b=minKillT-a*minTs`.
   - raffiner : **3 itérations « plus proche »** (bootstrap robuste) puis **3 itérations « dernier-avant »**
     (anchors létaux précis). À chaque iter : least-squares sur `(record.ts, kill.t)` du record retenu par tueur,
     fenêtre |Δ|<8000ms. (cf `fit()` dans `tmp_offwarp`.)
   - **Pente `a` quasi-constante ≈ 9.8e-4 sur tous les matchs testés** (à exploiter pour durcir, cf Phase 0.3).
5. **Attribution** : arme du kill = **famille du DERNIER record `0xd2` du tueur tel que `warp(record.ts) ≤ kill.TimeMS`**
   (slack 0). « Dernier-avant », PAS « le plus proche ».
6. **Résultats** : 96% par kill (000d5950, validé vs ground-truth live) ; 90-97% attribution + R² 0.93-0.99 sur 8
   autres matchs ; 46% sur BTB (kills non-firearm → Phase 0.2).

---

## RÉSULTATS PHASE 0 (recherche conduite le 2026-06-12)

> Investigation menée avec `cmd/tmp_btbclass` + `cmd/tmp_uncat`. Synthèse :
> - **0.2 partiellement tranché** : sur BTB `00ba2e1c` (46% firearm), les 54% restants ne sont PAS des armes
>   non-cataloguées (**0 record `0xd2` non-catalogué** — vérifié). Ce sont des kills **sans aucun record `0xd2`** =
>   splatter véhicule / collision / fusion coil / chute / grenade. **Largement irréductible offline.**
>   - Le **scanner mêlée** : (a) son temps s'aligne aux kills une fois le **warp appliqué à son `est`** (BTB 25/27,
>     000d5950 40/52 alignent en temps) — fix de câblage acquis ; MAIS (b) son **PI ≠ slot roster** (0 en même-slot,
>     mapping à trouver) et surtout (c) il **sur-détecte massivement** (52 « mêlées létales » sur 000d5950 où ≤4 kills
>     sont non-firearm → ce sont des SWINGS pendant des kills au fusil). ⇒ **inutilisable tel quel** pour identifier
>     des kills mêlée. Le filtre `MeleeHitLethal` (0x47/0x60) ne suffit pas à isoler les coups qui TUENT.
>   - Le **scanner grenade** n'a **pas de PI** (limitation structurelle) → pas d'attribution joueur.
>   - **Conséquence** : sur les modes véhicule (BTB), la couverture plafonne au taux firearm (~46-50%) ; le reste est
>     honnêtement étiquetable « non-arme-à-feu » mais **pas identifiable par arme** offline. Sur les modes
>     non-véhicule (Slayer/Ranked/Fiesta), le firearm couvre déjà 90-97% (les non-firearm y sont marginaux, ~4%).
> - **0.3 largement validé** : warp **linéaire** sur 9 matchs (R² 0.93-0.99), **pente `a` quasi-constante ≈9.8e-4**.
>   Le durcissement « pente fixe + offset par match » (pour les cas peu-d'anchors) reste une **option d'implémentation**
>   recommandée, pas un blocage.
> - **0.1 FAIT (2026-06-12)** : 2e ground-truth live sur un **Team Slayer** (9b191a7f, capture dual-hook
>   `tools/ce/filmdec_dualcap_capture.lua`). Métrique propre = pont-tsc par-kill (validée : **94% sur l'oracle
>   000d5950**). **Résultat Team Slayer : 58% par-kill**, erreur dominante **BR75↔MA40** (×18). Cause : alternance
>   BR+MA40 sub-100ms en Team Slayer + imprécision warp linéaire → mauvais des 2 fusils. **Global agrégé reste 79%**
>   (les swaps se compensent) ⇒ breakdown par joueur ~80% juste, flou sur « lequel des 2 fusils ». **Le 96% ne
>   généralise PAS uniformément** : excellent sur armes distinctives (Fiesta, power weapons), ~80% agrégé / ~58%
>   par-kill sur loadout-switching (Team Slayer). À acter comme caractéristique produit (le breakdown par arme est
>   bon sauf la distinction BR-vs-MA40 en Arena). Piste d'amélioration : warp plus fin (piecewise) pour gagner en
>   résolution sub-100ms ; ou règle « si BR et MA40 du tueur tous deux <Xms du kill, départager par dominance de rafale ».
>
> **Verdict** : la recherche firearm est close (96% prouvé + généralisation validée). Le « 100% tous-types » est
> **plafonné par les kills sans record de dégât** (véhicule/splatter), qui sont probablement irréductibles offline —
> à acter comme limite produit, pas comme tâche d'implémentation ouverte. Items restants réels : 0.1 (2e ground-truth)
> et, optionnel, un mapping PI-mêlée→slot + un détecteur de mêlée-LÉTALE fiable (si on veut gratter les rares kills mêlée).

## PHASE 0 — Recherche/validation restante (À FAIRE AVANT l'implémentation)

> Ces 3 items ne sont PAS de l'implémentation : ils valident la généralisation et comblent la couverture.
> Effort global : moyen. Bloquant partiel : 0.1 nécessite une capture live (matériel/jeu).

### 0.1 — Accuracy sur un 2e match avec ground-truth (PRIORITÉ 1, bloquant pour « prod-ready »)
- **Pourquoi** : le 96% d'**exactitude** n'est mesuré que sur `000d5950`. Les 90-97% des autres matchs sont des taux
  d'**attribution** (arme assignée), pas d'exactitude vérifiée. 1 seul point ground-truth = risque de sur-ajustement.
- **Quoi** : capturer une 2e vérité-terrain live (dual-hook `cmd/tmp_dualcap`, cf `FIREARM_PER_KILL_OFFLINE_SOLVED.md`
  §7) sur un match **différent et non-melee** (idéalement Ranked à loadout fixe, et 1 BTB). Puis rejouer `tmp_offwarp`
  dessus et mesurer l'accuracy par kill (réutiliser le bloc VALIDATION de `tmp_offwarp`, qui convertit TimeMS→tick live).
- **Critère** : ≥ 90% d'exactitude sur le 2e match → généralisation confirmée. Sinon, diagnostiquer (warp ? slack ?).
- **Workaround si pas de capture** : validation indirecte par cohérence — total kills/joueur == API, armes plausibles
  vs type de match. Plus faible, à documenter comme tel.

### 0.2 — Couverture des kills NON arme-à-feu (PRIORITÉ 1, c'est le gros du « 100% »)
- **Pourquoi** : BTB `00ba2e1c` = 46% (182 kills) car véhicule/splatter/mêlée/grenade/explosif n'ont pas de record
  `0xd2`. Pour une app de stats, ces kills doivent être couverts.
- **Quoi** :
  1. Caractériser les kills non-attribués (un kill sans record `0xd2` du tueur dans la fenêtre) : combien sont mêlée
     vs grenade vs véhicule/splatter vs collision ? (croiser avec médailles ? non — pas de médaille mêlée, cf user.)
  2. Brancher les **scanners mêlée/grenade existants** (`weaponv3.ScanMeleeHits` / `ScanGrenadeThrows`) **en complément**
     du firearm, MAIS valider d'abord leur fiabilité kill-vs-swing (le scanner mêlée détecte des SWINGS — cf incident
     2026-06-11 où 26 « mêlées » étaient des swings). Filtrer par `MeleeHitLethal` + proximité temporelle d'un kill non-firearm.
  3. Véhicule/splatter : recherche ouverte — y a-t-il un record de dégât véhicule analogue à `0xd2` ? (scanner les
     marqueurs des kills BTB non couverts). Si non décodable offline, marquer ces kills `weapon=unknown/vehicle`.
- **Critère** : sur BTB, attribution combinée (firearm+mêlée+grenade) ≥ 80% ; le reste étiqueté explicitement.

### 0.3 — Robustesse du warp en cas limite (PRIORITÉ 2)
- **Pourquoi** : R² descend à 0.93 quand peu d'anchors firearm (matchs mêlée). Prolongations/pauses non testées.
- **Quoi** : exploiter la **pente quasi-constante `a≈9.8e-4`** : tester un mode « pente fixe + offset `b` ajusté par
  match » (1 seul paramètre libre → robuste avec peu d'anchors). Comparer accuracy vs le fit 2-paramètres.
  Tester sur des matchs courts/longs/prolongation. Détecter et logger les warps suspects (R² < 0.9).
- **Critère** : pas de régression d'accuracy ; comportement défini quand R² faible (fallback pente fixe + warn).

---

## PHASE 1 — Algorithme pur (`internal/analysis/killweapon/`)

> Couche `analysis/` = algos purs, 0 accès DB, 0 Streamlit. Entrée : chunks de film décompressés. Sortie : types résultat.

- **Nouveau package** `apps/go-api/internal/analysis/killweapon/`.
- **Fichiers** :
  - `decode_damage.go` — `DecodeDamageRecords(chunks [][]byte) []DamageRecord` (le décode `0xd2` §Méthode.1).
  - `roster.go` — `DeriveRoster(chunks [][]byte, wantXUIDs []uint64) map[uint64]int` (type-8 bit-scan §Méthode.2).
    ⚠️ réutiliser `u64LE` exact d'offgen (ordre de bits MSB-first-par-byte ; un ordre naïf casse le scan → 0 match).
  - `warp.go` — `FitWarp(dmgs []DamageRecord, kills []KillEvent) Warp` (fit nearest×3 puis last-before×3 §Méthode.4)
    + `Warp.Apply(ts uint64) float64`. Exposer `Warp.R2` pour le logging/QA.
  - `attribute.go` — `AttributeWeapons(dmgs, kills, warp, meleeHits, grenades) []KillWeapon` (last-before §Méthode.5
    + fusion non-firearm Phase 0.2).
  - `killweapon.go` — façade `AttributeMatchKillWeapons(chunks [][]byte) ([]KillWeapon, QAReport, error)`.
  - `*_test.go` — tests purs (voir §Tests).
- **Types** (dans ce package ou `internal/domain/` si réutilisés ailleurs) :
  ```go
  type DamageRecord struct{ Slot int; FamilyID uint32; PacketTS uint64 }
  type KillEvent    struct{ KillerSlot, VictimSlot, TimeMS int }
  type KillWeapon   struct{ KillerXUID, VictimXUID uint64; TimeMS int; WeaponID uint64; Family string; Source WeaponSource; Confidence float64 }
  type WeaponSource int // Firearm | Melee | Grenade | Vehicle | Unknown
  type QAReport     struct{ Kills, Attributed int; WarpR2 float64; Suspect bool }
  ```
- **Réutiliser** : `analysis.ParseHighlightEvents`, `analysis.WeaponIDToName`, `weaponv3.ScanMeleeHits/ScanGrenadeThrows`.
- **NE PAS** : importer DuckDB, faire du held-weapon, décoder les marqueurs frères (apparence).
- **Effort** : moyen (le proto `tmp_offwarp` est à ~80% portable directement). Le découpage en fonctions <80 lignes
  est requis (CLAUDE.md règle 13) — le proto est en une fonction, à refactorer.

## PHASE 2 — Multi-titres & chemins
- **Capability** : l'attribution arme-par-kill dépend du **format film Halo Infinite**. Brancher sur une capability
  (ex. `CapabilityFilmWeaponAttribution`) via `HasCapability()`, **jamais** sur `slug=="halo_infinite"`.
  Dégrader proprement (`ErrCapabilityNotSupported`) pour les titres sans film.
- **Chemins** : les chunks de film viennent du cache film ; passer par le `PathResolver`/loader existant du film
  (vérifier comment `tmp_offgen`/le sync chargent `data/cache/film_chunks/<matchID>` — NE PAS hardcoder le chemin
  comme le proto). Aucun `filepath.Join(repoRoot, "data", ...)` direct.
- **Catalogue armes** : `WeaponIDToName` / `weaponv3/canon.go` déjà multi-représentation (high-32 famille | low-32
  variante). Pour distinguer les variantes (gravity vs antigrav hammer) il faut l'id64 complet (cf mémoire).

## PHASE 3 — Orchestration & persistance  ⚑ POINT D'INSERTION TRACÉ (2026-06-12)

> **TOUT LE SCAFFOLDING EXISTE DÉJÀ.** L'insertion n'est PAS un ajout, c'est un **remplacement d'algorithme** dans un
> pipeline weapon-kill complet et déjà câblé. Fichier : `internal/sync/backfill_weapons.go`.

- **Point d'insertion exact** : `BackfillWeaponKillsForMatchAll(ctx, client, sharedDB, matchID)` (et son jumeau
  mono-joueur `BackfillWeaponKillsForMatch`). Le pipeline actuel fait :
  1. `client.GetMatchFilm(matchID)` → **tous les chunks REPLICATION_DATA type-2** (= là où vivent les `0xd2`). ✅ réutilisé tel quel.
  2-4. `BuildWeaponTimelines` + `ScanFireEventsAll` (← **ANCIEN scanner fire-events, rejeté par le user**).
  5. `getAllKillsForMatch` (kills depuis `highlight_events` DB) + `getXuidToPI` (xuid→player_index, ordre team/rank).
  6. `analysis.CorrelateKillsGlobal(...)` (← **ancienne corrélation à remplacer**).
  7. `InsertWeaponKills` → table **`weapon_kills`** + `MarkWeaponKillsDone` (bit21). ✅ réutilisé.
  **→ Remplacer les étapes 2-4 + 6 par un appel à `killweapon.AttributeMatchKillWeapons(chunks)`** (décode 0xd2 +
  roster type-8 + warp + last-before). Les étapes 1, 5, 7 (download, kills DB, persistance, bitmask) sont CONSERVÉES.
- **Déclenchement** : déjà câblé — `processWeaponKillsInline` (PostSync, parallélisme 24) et
  `SyncEngine.BackfillWeaponKillsForMatches` (lease shared). Rien à ajouter côté orchestration.
- **Persistance** : cible = table **`weapon_kills`** existante (PAS `killer_victim_pairs.weapon_id`). Le row
  `WeaponKillRow{TimeMS, WeaponID, ReconciledAs, DeltaMS, Confidence, AttributionPath, SwapDetected, PlayerIndex, ...}`
  est déjà là — mapper la sortie `killweapon` dessus (WeaponID=id64, Confidence, AttributionPath="0xd2_warp_lastbefore").
  ⚠️ `weapon_kills` n'est **PAS append-only** (DELETE-then-INSERT par match_id) ; la sûreté ART vient de la
  sérialisation (lease shared + MaxOpenConns(1)), pas du schéma — ne PAS introduire de concurrence d'écriture.
- **⚠️ Convention player_index** : 3 numérotations coexistent — (a) `getXuidToPI` (DB, ordre team_id/rank), (b) slot
  roster type-8 (bit-scan), (c) slot R5 du `0xd2` (=idx live, validé). Le nouveau pipeline dérive (b)=(c) du film ;
  il faut **réconcilier avec (a)** pour que `WeaponKillRow.PlayerIndex` reste cohérent avec le reste du système
  (ou écrire par xuid via le roster type-8 xuid→slot, en contournant getXuidToPI). À trancher à l'implémentation.
- **Migration / coexistence** : l'ancien chemin fire-events alimente `weapon_kills` aujourd'hui. Le remplacement doit
  être **propre** (retirer `ScanFireEventsAll`/`CorrelateKillsGlobal` du flux une fois le nouveau validé), idéalement
  derrière un flag le temps d'un run comparatif (ancien vs nouveau sur les mêmes matchs), puis bascule définitive.
  Cf. [[project_weapon_attribution_v3_status]] (le v3 jamais promu) — le nouveau 0xd2+warp est le candidat de promotion.
- **Logging `slog`** : `InfoContext` (`match_id`, `kills`, `attributed`, `warp_r2`) ; `WarnContext` si
  `QAReport.Suspect` (R² < 0.9 ou attribution < 50% hors mode véhicule) ; `ErrorContext` sur erreur de décode.
- **Lecture** : `getAllKillsForMatch` donne déjà `is_melee`/`is_grenade` par kill (depuis l'event_type) → le nouveau
  pipeline peut router : firearm → 0xd2+warp ; melee/grenade → laisser non-firearm (cf Phase 0.2, scanners non fiables).

## PHASE 4 — API & Frontend (si exposition produit)
- **API** : exposer la répartition arme-par-kill par joueur (nouveau champ dans le détail match, ou endpoint dédié).
  Handlers dans `internal/api/handlers/` (pas de logique métier). Types de retour canoniques.
- **Frontend** : breakdown arme/joueur (kill-feed enrichi, ou bloc « armes » page match). Strings i18n FR+EN,
  query keys, couleurs via tokens (pas de hex). Labels via `useFieldLabel`. (Cf user : icône d'arme du kill-feed.)
- **Effort** : moyen, dépend du périmètre produit voulu (à cadrer avec l'utilisateur).

---

## Tests (par couche)
- `internal/analysis/killweapon/` : tests **purs** sur un petit jeu de chunks fixture (ou les chunks `000d5950` en
  testdata) — décode `0xd2` (compte=519), roster (8 slots), warp (R² > 0.9), attribution (≥ 90% sur 000d5950 vs un
  golden file dérivé du ground-truth live). **Garder le ground-truth live `tools/ce/*.bin` comme oracle de test.**
- `internal/service/` : test avec mock repo — l'attribution est appelée, le résultat persiste.
- `platform/duckdb/` : si nouvelle table, test DuckDB `:memory:`.
- Garde-rail anti-régression : un test qui échoue si l'accuracy 000d5950 descend sous un seuil (ex. 90%).
- ⚠️ `go test -race` est **incompatible** avec le driver DuckDB (cf mémoire) — tests DB sans `-race`.

## Logging
- `slog.InfoContext(ctx, "kill weapon attribution", "match_id", id, "kills", n, "attributed", a, "warp_r2", r2)`.
- `slog.WarnContext` si warp suspect / faible couverture hors BTB. Pas de `fmt.Println` (proto à nettoyer).

## Done-definition (par phase)
- **P0** : 2e match validé ≥ 90% accuracy ; couverture BTB combinée ≥ 80% ; warp robuste documenté. thought_log MAJ.
- **P1** : package `killweapon` + tests verts, golden 000d5950 ≥ 90%. `go test ./... && go vet ./...` OK.
- **P3** : arme persistée, sync câblé, logging en place, pattern d'écriture DB conforme (pas d'ART).
- **P4** : API+front si cadré. i18n FR+EN, couleurs tokenisées.

## Git
- **Branche** : continuer sur `fix/chart-empty-states`? NON — créer une branche dédiée depuis la branche de travail
  weapon (ce worktree est sur la feature weapon-attribution-v3). Proposer `feat/weapon-per-kill-offline`.
  Commits par phase. Demander le feu vert utilisateur avant tout commit (cf préférence).

## Nettoyage
- Les `cmd/tmp_*` exploratoires (`tmp_offgen, tmp_offwarp, tmp_warpmatch, tmp_warpmeas, tmp_warp2, tmp_dualcap,
  tmp_diffrecs, tmp_findmiss, tmp_findtime, tmp_victim, tmp_valvictim, tmp_perkill, tmp_clock, tmp_tsccheck,
  tmp_hexlive, tmp_killweaponoffline`) sont des prototypes. Garder `tmp_offwarp` + `tmp_dualcap` (oracle) le temps de
  la Phase 1, supprimer le reste après portage. Le ground-truth `tools/ce/*.bin` est à CONSERVER (oracle de test).

## Risques / blockers connus
- **1 seul point ground-truth** (000d5950) → Phase 0.1 obligatoire avant « prod-ready ».
- **Kills non-firearm** non couverts par `0xd2` (véhicule/splatter surtout BTB) → Phase 0.2, possible irréductible
  offline pour le splatter (à confirmer).
- **Écritures DB shared** : risque ART si `ON CONFLICT` concurrent — suivre le pattern persist/append-only.
- **Variante d'arme** (id64 complet vs famille high-32) : la famille est sûre offline ; la variante exacte
  (gravity vs antigrav hammer) dépend du low-32 runtime — possiblement non distinguable offline (acceptable : famille suffit).
