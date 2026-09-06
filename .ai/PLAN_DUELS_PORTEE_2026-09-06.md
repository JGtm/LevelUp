# PLAN — Duels, portée et dénivelé des engagements

> Date : 2026-09-06 (révisé le même jour après revue — six corrections d'architecture, côté
> victime et proxy d'ouverture promus dans les lots, dénivelé signé ajouté sur décision
> utilisateur). Branche `feat/duels`, worktree dédié `LevelUp-wt-duels`.
> Contrat d'exécution : skill `plan-execution` (ordre strict, aucun report, tout item statué).
> Mesure préalable : `.ai/V7.5/film_re/SONDE_DUELS_2026-09-06.md` — À LIRE AVANT TOUT.

## Objectif et critère de succès

Répondre à la demande utilisateur du 2026-09-06 : « combien de duels, combien gagnés, combien
devenus des trios », « croiser l'arme et la distance », et depuis la revue : « où je frague et
où je meurs », « d'en haut ou d'en bas ».

**Critère de succès** : la page Synthèse porte, sur la période filtrée, la PORTÉE PAR ARME de
mes frags ET de mes morts, le DÉNIVELÉ signé des deux, et un proxy de DISTANCE D'OUVERTURE
validé sur pièces — servis par capability, dégradant proprement sur un titre sans positions.
Et la question des duels est TRANCHÉE par une mesure, pas repoussée : soit le lot 7 s'exécute,
soit son gate a échoué et le report est inscrit au registre avec sa condition de reprise.

## Ce que la sonde a tranché (lot 0 — FAIT)

Quatre films d'arène, quatre cartes, décodeurs de production uniquement.

- Le critère « réciprocité du dégât » est le bon : le témoin décalé rend **0 faux positif sur
  les quatre films**. Ce qu'on détecte est vrai.
- Mais le film n'émet que **91 à 428 enregistrements de dégât pour 90 à 117 morts**. La
  réciprocité mesure **0 à 12,8 %** contre un seuil de 40 % écrit avant la mesure. Le compte
  des duels serait un sous-comptage de 4 à 8x, variable d'un facteur 5 entre matchs.
- Le sous-type 0xC0 non décodé n'ouvre aucun réservoir. **Le flux de dégâts complet n'est pas
  dans le film.**
- **La vitesse du jeu est le mécanisme.** Temps-pour-tuer 1,2-1,5 s au BR ; le bouclier n'est
  lu qu'environ toutes les 600 ms par joueur : un échange donne deux lectures. C'est pourquoi
  l'oracle « le mourant a perdu du bouclier dans les 3 s » plafonne à 44-76 % au lieu de 100 %.
  Et la réciprocité mesurée est IDENTIQUE à 2, 3 et 5 s : la riposte vit dans les deux premières
  secondes ou n'existe pas.
- Acquis durables : base d'atterrissage **512**, détachée d'un facteur 2,1 à 2,6 sous le
  critère de vivacité ; distance de l'engagement résolue à **95,7-100 %** ; portée médiane des
  éliminations **stable à 5,8-6,9 m sur quatre cartes** ; dénivelé |dz| **médian 0,4-0,9 m,
  16-40 % des engagements à plus d'un mètre**.

## Décisions tranchées AVANT exécution

**D1 — le comptage de duels n'est PAS livré sur le flux de dégâts.** Mesuré non atteignable ;
aucun réglage de fenêtre n'y change rien, le problème est le rappel, pas le seuil.

**D2 — une seule mesure peut rouvrir les duels : le bouclier DU TUEUR (lot 1).** Le tueur est
connu par le kill-feed (`match_kill_events_latest`, 97,6 %), pas par le film. Son résultat
sera une BORNE BASSE plafonnée par le taux de capture du canal : le gate se lit NORMALISÉ par
l'oracle victime (le taux de capture d'une chute dont on sait qu'elle a eu lieu). Fenêtre
**2 s**, mesurée, pas 5.

**D3 — livrable inconditionnel : la portée par arme en agrégat multi-matchs, DES DEUX CÔTÉS.**
Côté tueur (« où je frague ») ET côté victime (« où je meurs, et à quelle arme » — l'arme de
ma mort est celle du tueur). Même SQL, même chart. C'est l'item que le POC G.3 (2026-08-30)
avait fermé par cadrage (« pas d'agrégat multi-matchs »). Le grain match existe et tourne :
`domain.MatchKillDistanceWeapon`, `duckdb.KillDistanceRepo`, `MatchKillDistanceSection`.

**D4 — le dénivelé signé est livré (décision utilisateur 2026-09-06).** `killer_z - victim_z`
par frag mesuré, en trois classes : au-dessus (> +1 m), à niveau, en dessous (< -1 m). Le
seuil d'un mètre est celui de la sonde (16-40 % des engagements au-delà) — une marche, un
rebord, pas un décalage de capsule. Rendu : une barre empilée à trois segments par arme, sur
les MÊMES lignes que la portée, pour que les deux se lisent ensemble. Si la lecture n'apprend
rien, l'utilisateur l'ignorera : c'est le coût raisonnable d'une mesure plutôt que d'une
supposition.

**D5 — le proxy de distance d'OUVERTURE est livré, et il se valide lui-même.** La vraie
ouverture exigerait le premier dégât de l'échange, invisible. Proxy : **la distance 1,5 s
avant le coup fatal** (un temps-pour-tuer, pas un événement), lue sur les trajectoires du
rejeu. Validation OBLIGATOIRE avant exposition (lot 2) : sur les morts où le premier dégât EST
capturé (10-50 % selon le film), comparer la distance à cet instant et la distance à T-1,5 s ;
gate : écart médian <= 2 m. Sans ce gate, le proxy n'est pas publié.
**Où il se calcule, et pourquoi pas dans l'artefact de rejeu** : vérifié sur pièces, les
artefacts `data/cache/replays` sont PURGÉS par cron (`replay_purge_cron.go`, rétention en
mois ; 107 artefacts locaux pour 1 380 films). Ce n'est pas un substrat durable. Le proxy se
calcule donc LÀ OÙ `kill_positions` se calcule déjà : la passe positions de `killcollector`
(`positions.go`), qui tient les positions décodées en mémoire — `BuildKillPositions` est pure
et prend l'instant en paramètre : la MÊME fonction, appelée avec les `KillRef` décalés de
-1 500 ms, rend la position d'entame. Persistée dans une table append-only `kill_openings`
(+ vue `_latest`, recette ADR 0026), INSERT-only via le persister. Les matchs à venir la
reçoivent au sync ; l'historique exige un **backfill** (`backfill-killsource` avec capture de
positions, films du cache local) — opération lourde, **DÉCISION UTILISATEUR, jamais lancée
d'office** (règle `never_batch_replay_artifacts_ask_first`). Tant que le backfill n'a pas
tourné, la section publie la portée sans le proxy et le dit (« distance d'entame : N frags
mesurés »), jamais un zéro.
**VALIDÉ le 2026-09-06 (item 2.5)** : écart médian cumulé 1,24 m sur 87 cas des quatre
films, pour un gate à 2 m écrit avant la mesure. D5 reste GO — voir la section « Validation
du proxy d'entame » de `.ai/V7.5/film_re/SONDE_DUELS_2026-09-06.md`.

**D6 — la grammaire graphique est celle DÉJÀ VALIDÉE par l'utilisateur le 2026-09-02** : un
bâton par arme, un losange sur la valeur centrale (`_killDistanceChart.ts`). Transposition à
l'agrégat : **p10 -> p90 pour le bâton, médiane pour le losange** — sur des centaines de frags,
min et max décrivent deux accidents, pas une portée. Tri par médiane croissante : le graphe se
lit comme un continuum du contact à la longue portée. **PAS de grille arme x tranche.**
UN SEUL graphe, deux bâtons par ligne (frags en haut, morts en bas — fusion demandée par
l'utilisateur sur la maquette du 2026-09-06, cf. lot 5) ; au-dessus, les deux médianes en
chiffres — le diagnostic en deux nombres.

**D7 — aucune re-cuisson d'artefact.** `shared.kill_positions` est peuplée ; l'agrégat est du
SQL sur les vues `_latest`. La SEULE opération lourde du plan est le backfill de `kill_openings`
(D5), et elle est explicitement soumise à l'utilisateur — la portée et le dénivelé se livrent
sans elle.

**D8 — la portée mesurée est un USAGE, jamais une portée théorique.** Doctrine déjà posée
(`REFERENCE_VARIANTES_ARMES_REDDIT.md`). Aucun libellé « portée de l'arme ».

**D9 — le seuil de publication `WeaponRangeMinMeasured = 8`** est DÉFINI UNE FOIS dans
`internal/analysis/` (vérifié sur pièces : la constante du chantier précision vit sur
`feat/precision-arme`, pas sur `feat/v75` — rien à importer). Les armes sous le seuil ne sont
pas cachées en silence : leur NOMBRE est publié (« N armes sous le seuil »).

**D11 — frontière de stockage (échange utilisateur 2026-09-06).** Les STATS ne lisent JAMAIS
un artefact de rejeu : l'artefact est un cache (contrat de purge, `replay_retention_months`,
0 = illimité — un réglage, pas un cron à débrancher), un contrat de RENDU (schéma 39, il bouge
à chaque lot) et un coût (2,0 Mo par match mesuré : une Synthèse sur 500 matchs en lirait 1 Go
pour deux médianes). La base porte des MESURES PAR KILL — quelques nombres, ~100 octets la
ligne : `kill_positions` et `kill_openings` pèsent ensemble quelques dizaines de Mo pour la
totalité du corpus, dans un `shared_matches_v2` de 289 Mo. Ce qui ferait grossir la base « en
un rien de temps », ce sont les TRAJECTOIRES (48 000 points par match) — elles restent dans
l'artefact, et l'artefact reste régénérable depuis le cache de films, qui est la vraie source.

**D10 — les filtres sont ceux de la Synthèse.** Le scope arrive par `MatchIDs` (déjà filtré
par période côté service, cf. `loadWeaponAccuracy`) + `Gamertag`/`XUIDs`, avec `Validate()`
qui refuse un scan complet. AUCUN filtre temporel en SQL dans ce repo : la période est déjà
résolue en amont.

---

## Lot 1 — La mesure qui décide des duels

Sonde n°2, côté base. Instrument : `internal/sync/killcollector/duels_bouclier_research_test.go`
(seul paquet qui atteint déjà film + base + pont d'identité).

- [ ] 1.1 **Lecture de la base en `OpenReadForQuery` UNIQUEMENT** — le serveur peut tenir le
      fichier en RW ; jamais `OpenReadOnly` forcé, jamais RW (modèle mono-process, ADR 0013/0016).
- [ ] 1.2 Composer sur 4 matchs : `match_kill_events_latest` (tueur, victime, `time_ms`) ×
      `ResolveSlotXUID` × `ScanBipedPositions` avec `CaptureDirs` (bouclier).
- [ ] 1.3 Mesurer **A** : part des kills du feed dont le slot du TUEUR est résolu.
- [ ] 1.4 Mesurer **O** (oracle) : part des kills où le bouclier de la VICTIME chute dans
      [T-2 s, T] — le plafond de capture du canal.
- [ ] 1.5 Mesurer **B** : part des kills où le bouclier du TUEUR chute dans [T-2 s, T].
- [ ] 1.6 Mesurer le **témoin** : B avec la fenêtre déplacée de 37 s vers le passé.
- [ ] 1.7 Mesurer la **discrimination** : parmi les adversaires vivants ayant chuté dans la
      fenêtre, la victime est-elle la seule ?
- [ ] 1.8 Note `.ai/V7.5/film_re/SONDE_DUELS_BOUCLIER_2026-09-XX.md`, quatre nombres + verdict.

**Gate chiffré, écrit maintenant** — le lot 7 s'exécute si et seulement si :
`A >= 80 %` **ET** `B / O` compris entre 0,35 et 0,90 (la réciprocité normalisée : hors de cet
intervalle le signal est absent ou constant, donc muet) **ET** `B / témoin >= 3` **ET** la
victime est l'unique candidate dans **>= 60 %** des fenêtres.

```bash
cd apps/go-api && CGO_ENABLED=0 go test ./internal/sync/killcollector \
  -run TestSondeDuelsBouclier -v -timeout 900s
```

**Clos quand** : les nombres et le verdict (GO / NO-GO lot 7) sont dans la note, l'entrée
`thought_log.md` est posée. Si NO-GO : report au `.ai/V7.5/REGISTRE_REPORTS.md` avec sa
condition de reprise, et le lot 7 se statue `[!]`.

---

## Lot 2 — Analysis : agrégats purs, et la validation du proxy

Couche `internal/analysis/` — fonctions PURES, aucune I/O, aucun SQL.

**Exécuté le 2026-09-06** sur la branche `feat/duels-lot2` (worktree `LevelUp-wt-duels-lot2`).
Tous les items sont statués ; le gate 2.5 est TENU sur les quatre films et sur le cumul, donc
2.4 est conservé et D5 reste GO. Journal détaillé : `.ai/thought_log.md` (entrée du
2026-09-06) et `.ai/V7.5/film_re/SONDE_DUELS_2026-09-06.md`, section « Validation du proxy
d'entame ».

- [x] 2.1 `analysis/weapon_range.go` : type `MeasuredKill{WeaponKey string; DistanceM, DeltaZ float64; Side}`
      (`Side` = tueur | victime), et `WeaponRangeAggregate(kills []MeasuredKill, minMeasured int) []WeaponRange`
      où `WeaponRange` porte `WeaponKey, Side, Measured, P10, Median, P90, Above, Level, Below`.
      FAIT — la signature rend DEUX valeurs : `([]WeaponRange, WeaponRangeSummary)`. Le résumé
      est un second RETOUR et non un champ de sortie, précisément pour qu'un appelant qui
      l'ignore ne puisse pas prendre l'absence d'une arme pour un zéro (D9). Il porte
      `BelowThreshold`, `BelowThresholdBySide` (le rendu 5.3 dit « frags : N · morts : M ») et
      `MeasuredBelowThreshold` (les frags réels laissés de côté). Tri de sortie : médiane
      croissante, puis clé d'arme, puis côté.
- [x] 2.2 `const WeaponRangeMinMeasured = 8` et `const WeaponRangeLevelBandM = 1.0` (D4, D9),
      définies ICI et nulle part ailleurs — garde-rail grep sur le littéral `1.0` accolé à
      `killer_z`/`victim_z` ailleurs que dans ce fichier.
      FAIT — `weapon_range_guard_test.go` marche TOUT `internal/` (pas seulement le paquet : la
      copie qui divergerait viendrait du repo DuckDB) et échoue aussi si les deux déclarations
      disparaissent du propriétaire. Hors périmètre du grep, écrit dans le test : le TypeScript
      de `apps/web` (hors module Go) et les `_test.go`, dont les fixtures portent légitimement
      ±1,0 m exactement pour éprouver les bornes.
- [x] 2.3 Percentiles par interpolation linéaire, une seule implémentation, testée aux bornes
      (n=1, n=2, valeurs égales).
      FAIT — `percentileLinear` (convention « type 7 », celle de `quantile_cont`), non exportée.
      Vérifié avant d'écrire : le paquet `internal/analysis` n'avait AUCUN percentile interpolé
      (`PercentileRank` est un rang, `MedianFloat` une médiane seule) ; les deux implémentations
      voisines vivent dans des SOUS-paquets et sont non exportées (`temporal.quantileSorted`,
      interpolée ; `patterns.percentile`, au rang le plus proche). Un test épingle la
      coïncidence `percentileLinear(s, 50) == MedianFloat(s)`.
- [x] 2.4 `replay.OpeningLeadMS = 1_500` (nommé, commenté : un temps-pour-tuer) et
      `replay.ShiftKillRefs(kills []KillRef, leadMS int64) []KillRef` — pure ; la position
      d'entame est `BuildKillPositions(pos, slotXUID, ShiftKillRefs(kills, -OpeningLeadMS), off)`,
      AUCUNE seconde fonction de placement (règle « deux décodeurs du même fait divergeraient »).
      FAIT — `replay/killpos_opening.go`, CONSERVÉ parce que le gate 2.5 est tenu. Un couple
      dont l'instant décalé passerait avant l'origine du film n'est PAS écarté par la fonction :
      `BuildKillPositions` ne lui trouve alors aucune position, ce qui est le résultat correct
      (jamais une position de réapparition présentée comme une entame). Le comportement est
      épinglé par un test dédié, pour qu'un changement de la porte de `positionOf` le dise.
- [x] 2.5 **Validation du proxy (D5)** : instrument `replay/duels_ouverture_research_test.go`
      qui, sur les 4 films de la sonde, compare distance au premier dégât vs distance à T-1,5 s.
      Gate : écart médian <= 2 m. Résultat consigné dans la note de la sonde. Si le gate échoue,
      2.4 est supprimé (pas désactivé — règle 0 code mort) et D5 passe `[!]`.
      FAIT — **GATE TENU**. Cumul des quatre films : n = 87 écarts, médiane **1,24 m**, p90
      3,28 m, 69,0 % des cas à moins de 2 m. Par film : Cliffhanger 1,29 m (n=28), Catalyst
      0,40 m (n=8), Bazaar 1,22 m (n=40), Vagabond 1,61 m (n=11). Le gate est tenu PAR LE CODE
      (`t.Errorf`), pas par une lecture de log. Sensibilité mesurée à 1,0 s et 2,0 s : l'écart
      croît avec l'avance, ce qui s'explique par le délai médian premier dégât -> fin de vie
      (551 à 1 335 ms) — la population de validation est biaisée vers les échanges courts, elle
      ne dit donc PAS de descendre la constante. `OpeningLeadMS` reste à 1,5 s, la valeur de D5.
- [x] 2.6 Tests unitaires purs : agrégat nominal, seuil non atteint, une arme, doublons, les
      trois classes de dénivelé aux bornes (+1,0 exactement = à niveau).
      FAIT — `weapon_range_test.go` : nominal deux côtés, seuil non atteint (avec la ventilation
      du résumé), une arme au seuil exact, doublons, bornes de dénivelé sur un jeu VOLONTAIREMENT
      ASYMÉTRIQUE, convention de signe côté victime isolée, tri déterministe à médianes égales,
      non-mutation de l'entrée, entrée vide et seuil absurde.

**Convention de signe du dénivelé, tranchée ici et à respecter par le lot 3** :
`MeasuredKill.DeltaZ` porte la grandeur PHYSIQUE `killer_z - victim_z`, sans point de vue — le
repo DuckDB l'écrit telle quelle pour les DEUX lectures. C'est `WeaponRangeAggregate` qui la
ramène au point de vue du côté demandé (côté victime : l'opposé), pour que « où je meurs, d'en
haut ou d'en bas » réponde MA position et non celle du tueur. Le repo ne doit donc PAS inverser
le signe de son côté : le faire deux fois annulerait l'inversion.

**Gate** : `cd apps/go-api && go test ./internal/analysis/ -run 'WeaponRange|OpeningDistance' -v`

Gates réellement exécutés le 2026-09-06 (`CGO_ENABLED=0`, `GOCACHE` isolé), tous verts :
`go test ./internal/analysis/ -run 'WeaponRange|Percentile|SeuilsPortee' -v` ·
`go test ./internal/analysis/ -timeout 900s` · `go test ./internal/analysis/replay/ -timeout 900s` ·
`go vet ./internal/analysis/ ./internal/analysis/replay/` · `gofmt -l` (sortie vide) ·
`TestSondeDuelsOuverture` sur les 4 films, un par process.

---

## Lot 3 — Port + repo DuckDB, avec l'extraction de la jointure

- [ ] 3.1 **Extraire l'helper canonique** `measuredKillsQuery(where string)` dans
      `platform/duckdb/kill_measured.go` : la jointure `match_kill_events_latest × kill_positions_latest`,
      la garde `publishable`, la garde d'unanimité `HAVING count(DISTINCT e.source_tag) = 1`, le
      `Scan` en `killDistanceMeasured` étendu de `deltaZ` et `victimXUID`. `KillDistanceRepo.LoadMatch`
      MIGRE dessus dans le même commit (règle 6 : à la 3e copie on centralise ET on migre).
- [ ] 3.2 **Garde-rail** `kill_measured_guard_test.go` : grep interdisant le littéral
      `JOIN kill_positions_latest` hors de `kill_measured.go` (Q21b de `queries_match.go` reste
      sur sa propre jointure kill-feed sans positions — hors motif, documenté dans le test).
- [ ] 3.3 `port.WeaponRangeRepository` :
      `LoadWeaponRange(ctx, slug string, f WeaponRangeFilters) ([]analysis.MeasuredKill, error)`,
      `WeaponRangeFilters{MatchIDs, Gamertag, XUIDs}` + `Validate()` calqué sur
      `WeaponAccuracyFilters` (D10). Le repo rend les kills MESURÉS ; l'agrégat est en analysis.
- [ ] 3.4 `platform/duckdb/weapon_range_repo.go` : deux requêtes via l'helper, `WHERE e.match_id IN (?)
      AND e.feed_killer_xuid = ?` (côté tueur) et `... AND e.victim_xuid = ?` (côté victime) ;
      classification `source_tag -> weapon_key` par le `port.KillSourceClassifier` injecté ; distance
      par `hypot3D` (déjà l'unique formule du paquet) ; `deltaZ = killer_z - victim_z`.
- [ ] 3.5 Table `kill_positions_latest` absente ou vide -> `games.ErrCapabilityNotSupported`
      (même contrat que `WeaponAccuracyRepository`).
- [ ] 3.6 `slog.DebugContext` : lignes lues, armes retenues, armes sous seuil ;
      `slog.ErrorContext(ctx, "...", "err", err)` sur toute erreur, ligne illisible comprise
      (jamais avalée).
- [ ] 3.7 Test DuckDB `:memory:` : schéma créé PAR LES MIGRATIONS RÉELLES
      (leçon `reference_test_ddl_copies_derivent`) ; cas : nominal deux côtés, unanimité violée,
      position manquante d'un côté, table absente.

- [ ] 3.8 **Table `kill_openings`** (D5, si le gate 2.5 est GO) : migration append-only
      (`id` PK séquence, `written_at`, `match_id`, `killer_xuid`, `time_ms`, six coordonnées à
      T-lead) + vue `kill_openings_latest`, dans `steps_appendonly_*` et `migration/order.go`,
      recette ADR 0026 (`append_only_rebuild.go`). Isolation par titre vérifiée par le test
      `synthetic_title_b/migration_isolation_test.go`.
- [ ] 3.9 Persister INSERT-only `KillOpeningPersister` via `BatchBuilder.AddKillOpenings` —
      calqué sur `kill_position_persister.go` ; allowlist `no_art_patterns_test.go` inchangée.
- [ ] 3.10 Producteur : dans `killcollector/positions.go`, après `BuildKillPositions`, un second
      appel avec `ShiftKillRefs` ; compteurs ADR 0009 (`killsource_openings_lignes_ecrites`,
      `..._morts_sans_position`) ; échec = journalisé + compté, JAMAIS fatal à la passe de morts.
- [ ] 3.11 `backfill-killsource` : la capture d'entame suit `WithPositionCapture` (même
      drapeau, même catalogue de bornes) — aucun nouveau flag.
- [ ] 3.12 L'helper 3.1 accepte la table source en paramètre (`kill_positions_latest` |
      `kill_openings_latest`) — une seule jointure pour les deux lectures.

**Gate** : `cd apps/go-api && go test ./internal/platform/duckdb/ -run 'WeaponRange|KillMeasured|KillDistance' -v`
— `KillDistance` inclus : la migration de 3.1 ne doit rien changer au POC. Puis
`go test -tags=integration -p 1 ./internal/persist/... ./internal/sync/killcollector/... ./internal/migration/...`
(3.8-3.10 touchent persist/sync/migration : run nu = FAUX VERT).

---

## Lot 4 — Service, capability, contrat API

- [ ] 4.1 `SynthesisService.WithWeaponRangeRepo(repo)` ; câblage INCONDITIONNEL dans
      `SynthesisCtx` (jamais `slug ==`), sur le modèle de `WithWeaponAccuracyRepo`.
- [ ] 4.2 `loadWeaponRange` calqué sur `loadWeaponAccuracy` (synthesis_service.go) : scope par
      `MatchIDs`, `ErrCapabilityNotSupported` -> Debug, autre erreur -> Warn, best-effort nil.
      Appelle `analysis.WeaponRangeAggregate` sur les deux lectures du repo (kill, et si D5 est
      GO, entame) ; le delta entame -> kill se calcule en analysis, par frag apparié
      (`match_id, killer_xuid, time_ms`), jamais entre deux médianes.
- [ ] 4.3 `domain.Synthesis.WeaponRange *SynthesisWeaponRange` : `Kills []WeaponRangeEntry`,
      `Deaths []WeaponRangeEntry`, `BelowThreshold int` (D9), `MedianKillsM`, `MedianDeathsM`,
      et si D5 GO `MedianOpeningM` + `MedianDeltaM`.
- [ ] 4.4 Capability DONNÉE : réutiliser `games.CapFilmKillPositions` (`film.kill_positions`,
      déjà `supported` dans `capabilities.toml`). Capability PRODUIT `CapWeaponRange = "weapon_range"`
      dans `title.registry.go` (Infinite oui ; Halo 5 non) + clé miroir dans
      `config/titles/halo_infinite/mappings/capabilities.toml`.
- [ ] 4.5 Contrat `openapi.yaml` + `make generate-types`.
- [ ] 4.6 Tests service (mock `port.WeaponRangeRepository`) : nominal, capability absente, repo
      nil, scope vide. Tests `httptest` sur la page Synthèse : la section absente ne casse rien.

**Gate** : `make go-api-test && cd apps/go-api && go test ./internal/api/... ./internal/service/...`

---

## Lot 5 — Web : les graphes

Skills à invoquer AVANT d'écrire : `foundations-usage`, `dataviz`, `color-tokens`,
`frontend-patterns`.

**Maquette validée par l'utilisateur le 2026-09-06** (artefact « Portée des engagements »,
source versionnée `.ai/V7.5/MAQUETTE_PORTEE_ENGAGEMENTS_2026-09-06.html`, à ouvrir dans un navigateur) : UNE carte, une ligne par arme,
DEUX bâtons par ligne (frags en haut, morts en bas), même axe — l'utilisateur a demandé la
fusion des deux graphes jumeaux. C'est la référence de rendu du lot.

- [ ] 5.1 `features/synthesis/_weaponRangeChart.ts` — logique PURE, testée hors composant.
      Série ECharts `custom` (`renderItem`) : par ligne, deux rectangles arrondis p10->p90
      (hauteur 7 px, écart 4 px, décalés de part et d'autre du centre de bande) et un losange
      sur chaque médiane. POURQUOI `custom` ET PAS bar+scatter comme `_killDistanceChart.ts` :
      sur un axe de catégories un `scatter` ne se décale pas d'un demi-bâton, le losange
      tomberait entre les deux bâtons. Bande de 34 px par ligne (`48 + 34 × n`). Tri par
      médiane de frags croissante ; une arme mesurée d'un seul côté n'a qu'un bâton, l'infobulle
      dit « aucune mesure » pour l'autre. Libellé de ligne « Arme ×frags/morts ».
- [ ] 5.2 `_weaponElevationChart.ts` — deux barres empilées 100 % par ligne (piles `frags` et
      `morts`, `barGap` 55 %), trois segments d'en haut / à niveau / d'en bas ; MÊMES catégories
      et MÊME ordre que la portée ; pourcentage inscrit dans le segment à partir de 18 %.
      Couleurs : une seule teinte du clair (`chart-series-1`, d'en bas) au foncé
      (`chart-series-3`, d'en haut), `muted-foreground` à niveau — la couleur ne juge pas.
- [ ] 5.3 `SynthesisWeaponRangeSection.tsx` (`SectionCard`) : bandeau « Portée par arme — mes
      frags et mes morts » + compte ; sous-titre + légende HTML (deux entrées, position ET
      libellé) ; graphe de portée ; sous-titre + légende ; graphe de dénivelé ; ligne « Sous le
      seuil de 8 mesures — frags : … · morts : … » ; note de couverture ; `<details>` « Voir en
      tableau » (les deux côtés). Au-dessus, quatre `AccentCard` : médiane frags, médiane morts,
      distance d'entame, entame -> frag (avec la part de frags où la distance se ferme). Aucune
      logique métier dans le composant.
- [ ] 5.4 Strings FR **et** EN dans `i18n.ts` (parité typée `Record<Locale, T>`). FR sans
      anglicismes : « portée », « portée basse / haute », « frags mesurés », « d'en haut »,
      « à niveau », « d'en bas », « distance d'entame ».
- [ ] 5.5 Zéro hex, zéro classe Tailwind couleur ; tokens sémantiques uniquement. Query key
      dans `lib/query/keys.ts`. Montage dans `SynthesisPage.tsx` à côté de
      `SynthesisWeaponAccuracyChart`, gate `useCapability('weapon_range')`.
- [ ] 5.6 Tests : logique de projection (`_weaponRangeChart.test.ts`, `_weaponElevationChart.test.ts`),
      rendu de la section avec et sans données, état « sous seuil ».

**Gate** : `Remove-Item -Recurse -Force apps/web/node_modules/.tmp ; make check-types && make test-web`

---

## Lot 6 — Livraison

- [ ] 6.1 Skill `delivery-checklist`.
- [ ] 6.2 `cd apps/go-api && go test ./... && go vet ./...` puis
      `go test -tags=integration -p 1 ./...` (le lot 3 touche `platform/duckdb` ; `-p 1` non
      négociable ; code de sortie vérifié, pas la sortie filtrée).
- [ ] 6.3 `make go-api-lint` — baseline non accrue.
- [ ] 6.4 Gate visuel : capture de la section Synthèse soumise à l'utilisateur, témoins nommés.
- [ ] 6.5 Entrée `thought_log.md` ; `.ai/project_map.md` si la carto bouge.
- [ ] 6.6 Commit + push + **CI surveillée jusqu'au niveau JOB** (`gh run list --branch feat/duels`) ;
      tout rouge se répare, même préexistant.

---

## Lot 7 — Duels — CONDITIONNÉ au gate du lot 1

À n'ouvrir QUE si le lot 1 rend GO. Périmètre fermé d'avance :

- [ ] 7.1 Table append-only `match_engagements` + vue `_latest` (recette ADR 0026).
- [ ] 7.2 Producteur dans `killcollector` (INSERT-only via `BatchBuilder.Submit` — ADR 0019/0030).
- [ ] 7.3 Classification : duel / élimination / non conclu ; effectif à la résolution
      (tête-à-tête, 2v1, 1v2) — l'équipe vient du roster, disponible côté base.
- [ ] 7.4 Lecture, service, capability `duels`, chart.
- [ ] 7.5 Gate de vérité terrain : l'utilisateur visionne 15 à 20 engagements dans Theater et
      tranche. Aucune publication avant ce gate.

Si NO-GO : statuer `[!]` avec renvoi à la note du lot 1, et report au registre avec sa
condition de reprise (« un flux de dégâts dense, ou un compteur d'état ECS répliqué »).

---

## Règles d'exécution

1. **Ordre strict.** Lot N+1 interdit tant que le gate de N n'est pas passé. Le lot 1 est
   premier parce que lui seul ouvre ou ferme le lot 7 ; les lots 2 à 6 ne dépendent pas de lui.
2. **Aucun item sans statut à la clôture** : `[x]` fait, `[~]` couvert ailleurs (référence),
   `[!]` non traité (justification écrite).
3. **Vérifier sur pièces avant de coder ET avant de cocher.**
4. **Zéro fix opportuniste hors périmètre.** Toute découverte va dans « Découvertes ».
5. **Aucun report d'une étape exécutable maintenant.**
6. **Reprise de session** : ce fichier (les cases font foi), `git log --oneline -10` sur
   `feat/duels`, entrées récentes de `thought_log.md`.

## Découvertes (à ne PAS traiter)

- `hypot3D` (`platform/duckdb`) et `dist3` (`analysis/replay`) sont la même formule dans deux
  paquets. Deux copies, dans la limite ; à surveiller si un troisième paquet en a besoin.
- (lot 2, 2026-09-06) **Le mot « percentile » recouvre DEUX conventions dans le dépôt.**
  `analysis/temporal.quantileSorted` et le nouveau `analysis.percentileLinear` interpolent ;
  `analysis/patterns.percentile`, `analysis/replay.gwPadsQuantile` et `media.percentile`
  prennent le rang le plus proche. Cinq implémentations, toutes non exportées, dans cinq
  paquets — aucune n'était réutilisable depuis `internal/analysis`. Les deux familles donnent
  des p10/p90 différents sur les petits effectifs. NON TRAITÉ (hors périmètre) : la
  centralisation supposerait de trancher la convention pour des mesures déjà publiées.
- (lot 2, 2026-09-06) `TestSeuilsPorteeDefinisUneSeuleFois` coûte ~19 s à froid : il marche
  tout `internal/` et lit chaque `.go`. C'est le prix d'un garde-rail qui couvre les couches
  aval. À surveiller si d'autres garde-rails adoptent la même marche — à la troisième, il
  faudra un index partagé plutôt que trois marches complètes.
- (lot 2, 2026-09-06) **Le délai médian entre le premier dégât CAPTURÉ et la fin de vie vaut
  551 à 1 335 ms** sur les quatre films. C'est un second angle sur le constat de la sonde n°1
  (« la riposte vit dans les deux premières secondes ou n'existe pas ») et c'est aussi ce qui
  biaise la population de validation du proxy d'entame vers les échanges courts. Utile au lot 7
  s'il s'ouvre ; rien à traiter ici.
