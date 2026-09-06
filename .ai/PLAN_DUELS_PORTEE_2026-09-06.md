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

- [x] 1.1 **Lecture de la base en `OpenReadForQuery` UNIQUEMENT** — le serveur peut tenir le
      fichier en RW ; jamais `OpenReadOnly` forcé, jamais RW (modèle mono-process, ADR 0013/0016).
      → `duelsBOuvrirBase` ; chemin par `title.NewPathResolver(DUELS_DATA_ROOT).SharedDBPath`.
- [x] 1.2 Composer sur 4 matchs : `match_kill_events_latest` (tueur, victime, `time_ms`) ×
      `ResolveSlotXUID` × `ScanBipedPositions` avec `CaptureDirs` (bouclier).
      → 409 kills du feed exploitables sur les 4 films ; horloge posée par `ScanClockOrigin`,
      comme `buildPositionRows` (positions.go) — aucun calage réinventé.
- [x] 1.3 Mesurer **A** : part des kills du feed dont le slot du TUEUR est résolu.
      → **370/409 = 90,5 %** (par `BuildKillPositions` elle-même) ; 409/409 = 100 % des tueurs
      ont au moins un slot au pont.
- [x] 1.4 Mesurer **O** (oracle) : part des kills où le bouclier de la VICTIME chute dans
      [T-2 s, T] — le plafond de capture du canal. → **237/409 = 57,9 %**.
- [x] 1.5 Mesurer **B** : part des kills où le bouclier du TUEUR chute dans [T-2 s, T].
      → **135/409 = 33,0 %**, soit **B/O = 0,57**.
- [x] 1.6 Mesurer le **témoin** : B avec la fenêtre déplacée de 37 s vers le passé.
      → **22/376 = 5,9 %** contre B = 123/376 sur la même population : **B/témoin = 5,59**.
      Dénominateur borné au domaine observable (un kill des 39 premières secondes verrait sa
      fenêtre reculée tomber avant la première lecture de bouclier — durcissement par rapport
      à la sonde n°1, justifié dans la note).
- [x] 1.7 Mesurer la **discrimination** : parmi les adversaires vivants ayant chuté dans la
      fenêtre, la victime est-elle la seule ? → **71/135 = 52,6 %**, contre un seuil de 60 %.
      **C'est le gate qui échoue.** L'équipe vient de `match_participants.team_id`.
- [x] 1.8 Note `.ai/V7.5/film_re/SONDE_DUELS_BOUCLIER_2026-09-06.md`, quatre nombres + verdict.

**Gate chiffré, écrit maintenant** — le lot 7 s'exécute si et seulement si :
`A >= 80 %` **ET** `B / O` compris entre 0,35 et 0,90 (la réciprocité normalisée : hors de cet
intervalle le signal est absent ou constant, donc muet) **ET** `B / témoin >= 3` **ET** la
victime est l'unique candidate dans **>= 60 %** des fenêtres.

```bash
cd apps/go-api && CGO_ENABLED=1 \
  DUELS_DATA_ROOT=<racine des donnees> DUELS_MATCH=000d5950 DUELS_MAP=Cliffhanger \
  go test ./internal/sync/killcollector -run TestSondeDuelsBouclier -v -timeout 900s -count=1
```

⚠ `CGO_ENABLED=0` (valeur écrite dans le plan avant exécution) était FAUX : cette sonde ouvre
DuckDB, qui exige CGO. Un match par process (verrou `filmdec.LockProcessDecode`).

**Clos quand** : les nombres et le verdict (GO / NO-GO lot 7) sont dans la note, l'entrée
`thought_log.md` est posée. Si NO-GO : report au `.ai/V7.5/REGISTRE_REPORTS.md` avec sa
condition de reprise, et le lot 7 se statue `[!]`.

**RÉSULTAT (2026-09-06) — NO-GO.** Trois gates sur quatre passent, et nettement : A = 90,5 %,
B/O = 0,57 (centre de la bande), B/témoin = 5,59. **D = 52,6 % contre 60 % : ÉCHOUE.** Dans
47,4 % des fenêtres où le bouclier du tueur chute, un AUTRE adversaire chute aussi : le signal
dit « le tueur a pris des coups », jamais « de sa victime ». Et D varie de 40,0 % (Catalyst) à
64,5 % (Bazaar) — un biais qui rend deux matchs incomparables. Les deux canaux du film sont
exactement complémentaires dans ce qu'ils manquent : la sonde n°1 avait le lien sans le rappel,
la n°2 a le rappel sans le lien. Détail, réserves et condition de reprise :
`.ai/V7.5/film_re/SONDE_DUELS_BOUCLIER_2026-09-06.md`.

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

### Revue adversariale ronde 1 (2026-09-06, branche `feat/duels-lot2-fix`)

Deux relecteurs indépendants ; sept constats retenus par le pilote, tous corrigés. Le lot 2
n'était PAS livrable en l'état : F1 publiait des points de réapparition comme des entames.

- [x] **F1 (P1) — la position d'entame pouvait être un POINT DE RÉAPPARITION.** Le décalage
      seul ne suffit pas : `BuildKillPositions` ne connaît aucune frontière de vie et rend
      l'échantillon le plus proche à 120 ms près. Reproduit par les deux relecteurs (mort à
      1 550 ms, apparitions à t = 0 : entame « résolue » sur les deux spawns, 1 131 m d'écart
      pour une mort à 5 m). CORRIGÉ par `replay.BuildKillOpenings(pos, slotXUID, kills,
      offsetUS) ([]KillPosition, KillPosReport)` — décale par `ShiftKillRefs`, place par LA
      fonction de placement (aucune seconde), puis ne garde un côté que si le slot qui a fourni
      la position porte l'instant du KILL dans la MÊME vie (`buildLifeSpans`, trou `lifeGapUS`).
      L'instant rendu est celui du KILL (clé de jointure `time_ms`). Compteur dédié
      `KillPosReport.OpeningOutOfLife` (compte des CÔTÉS). Le placement est factorisé dans
      `placeKillPositions`, qui rend en plus le SLOT retenu (`positionOf` rend désormais la
      position ET son slot) : `BuildKillPositions` en reste la façade publique, inchangée.
      En-tête de `killpos_opening.go` réécrit — il affirmait l'inverse de la vérité.
- [x] **F2 (P1) — le test « instant négatif » passait pour la mauvaise raison** (échantillons
      hors tolérance ; clamper l'instant à 0 laissait la suite verte). RÉÉCRIT avec un
      `offsetUS` non nul : l'instant de match négatif devient un instant de film POSITIF, DANS
      la tolérance — un témoin vérifie que le placement nu tombe dans le piège. Ce qui refuse
      est désormais la frontière de vie, plus un débordement `uint64`.
- [x] **F3 (P1) — le tri des distances n'était couvert par aucun test** (`sort.Float64s`
      remplacé par un no-op : suite verte). `TestWeaponRangeAggregateDistancesNonTriees` :
      distances 40, 2, 9, 15, 4, 30, 7, 12 ; p10 = 3,4 · médiane = 10,5 · p90 = 33,0, calculés
      à la main. Sans tri : 13,4 · 9,5 · 8,5 — mutation prouvée rouge.
- [x] **F4 (P2) — garde-rail sans contrôle positif** (`if false && fautifPortee(...)` laissait
      le test vert). Le détecteur prend sa RACINE en paramètre ;
      `TestSeuilsPorteeDetecteUneCopie` plante deux copies fautives et un fichier licite (DDL +
      commentaire) dans un `t.TempDir()` et vérifie les trois verdicts. En-tête complété par ce
      que les regex NE captent PAS (SQL multiligne, alias `dz`, `HAVING count(*) >= 8`) —
      consigné, non corrigé : le lot 3 n'écrit aucun seuil en SQL.
- [x] **F5 (P2) — doc inversée** : `WeaponRangeMinMeasured` promettait « N armes sous le
      seuil » alors que `BelowThreshold` compte des COUPLES (arme, côté). Doc corrigée, la
      formulation publiée renvoie à `BelowThresholdBySide`.
- [x] **F6 (P2) — code mort** : le clamp `if minMeasured < 1` était inatteignable (un groupe
      porte toujours >= 1 frag). Supprimé ; le pourquoi est écrit dans la doc de
      `WeaponRangeAggregate` pour qu'il ne revienne pas. Test conservé (il asserte encore vrai),
      commentaire corrigé.
- [x] **F7 (P2) — code mort** : la garde `if lo >= n-1` de `percentileLinear` était
      inatteignable (`p >= 100` a déjà rendu la main). Supprimée ; les gardes `p <= 0` et
      `p >= 100` restent (leur suppression conjointe panique).

Gates de la ronde (`CGO_ENABLED=0`, `GOCACHE` isolé `.gocache-fix2`) : `go test -count=1
./internal/analysis/ ./internal/analysis/replay/ -timeout 900s` (ok 37,4 s / 23,9 s) ·
`go vet` (0) · `gofmt -l internal/analysis` (vide) · `go build ./internal/sync/killcollector/`
avec CGO (le consommateur de `KillPosReport` compile). Mutations prouvées rouges puis
restaurées : filtre de vie neutralisé -> 3 tests d'entame rouges ; `sort.Float64s` retiré ->
`TestWeaponRangeAggregateDistancesNonTriees` rouge ; détecteur du garde-rail neutralisé ->
`TestSeuilsPorteeDetecteUneCopie` rouge.

---

## Lot 3 — Port + repo DuckDB, avec l'extraction de la jointure

**Exécuté le 2026-09-06** sur la branche `feat/duels-lot3` (worktree `LevelUp-wt-duels-lot3`).
Tous les items sont statués. Trois consignes du pilote, issues d'une revue adversariale du
lot 2, ont été appliquées en cours de lot : la clé de jointure des entames est l'instant DU
KILL (3.10), le filtre « même vie » ne se réimplémente pas ici (3.10 bis, réserve écrite), et
aucun seuil de publication ne descend dans le SQL (3.3/3.4).

- [x] 3.1 **Extraire l'helper canonique** `measuredKillsQuery(where string)` dans
      `platform/duckdb/kill_measured.go` : la jointure `match_kill_events_latest × kill_positions_latest`,
      la garde `publishable`, la garde d'unanimité `HAVING count(DISTINCT e.source_tag) = 1`, le
      `Scan` en `killDistanceMeasured` étendu de `deltaZ` et `victimXUID`. `KillDistanceRepo.LoadMatch`
      MIGRE dessus dans le même commit (règle 6 : à la 3e copie on centralise ET on migre).
      FAIT — `kill_measured.go` porte `measuredKillsQuery(table, where)`, le type `killMeasured`
      (étendu de `matchID`, `timeMS`, `victimXUID`, `deltaZ`), `queryMeasuredKills` et `hypot3D`
      (déplacée depuis le POC : elle sert maintenant les deux grandeurs dérivées).
      `KillDistanceRepo` ne garde que sa clause de portée (`killDistanceWhere`) ; ses 9 tests
      d'origine passent INCHANGÉS. La signature prend la table en premier paramètre — 3.12 est
      donc couvert par construction, pas par un second passage.
- [x] 3.2 **Garde-rail** `kill_measured_guard_test.go` : grep interdisant le littéral
      `JOIN kill_positions_latest` hors de `kill_measured.go` (Q21b de `queries_match.go` reste
      sur sa propre jointure kill-feed sans positions — hors motif, documenté dans le test).
      FAIT — deux tests : l'interdit (les .go non-test de `platform/duckdb` + `halo5/`) et la
      vitalité du propriétaire (il doit toujours nommer les DEUX tables ET porter ses deux
      gardes, sinon le garde-rail ne garde plus rien). VU ROUGE avant commit (témoin : littéral
      réintroduit dans `kill_distance_repo.go`, test rejoué, littéral retiré). Deux exceptions
      nominatives et datées, écrites dans le test : le propriétaire, et
      `halo5/halo5_match_events_source.go` (LEFT JOIN d'énumération d'events, positions natives,
      aucune mesure de distance). Q21b est hors motif parce qu'elle ne joint AUCUNE table de
      positions — le motif vise la jointure, et c'est écrit dans le test.
- [x] 3.3 `port.WeaponRangeRepository` :
      `LoadWeaponRange(ctx, slug string, f WeaponRangeFilters) ([]analysis.MeasuredKill, error)`,
      `WeaponRangeFilters{MatchIDs, Gamertag, XUIDs}` + `Validate()` calqué sur
      `WeaponAccuracyFilters` (D10). Le repo rend les kills MESURÉS ; l'agrégat est en analysis.
      FAIT — `port/weapon_range.go`. DEUX méthodes : `LoadWeaponRange` (coup fatal) et
      `LoadWeaponOpening` (entame), même filtre, même contrat de capability. `Validate()` refuse
      les deux formes de scan complet (aucun match / aucun joueur). AUCUN seuil de publication
      dans ce port ni dans son repo : le seuil D9 vit dans `WeaponRangeAggregate`, qui compte ce
      qu'il écarte — un `HAVING count(*) >= 8` en SQL rendrait ce décompte impossible.
- [x] 3.4 `platform/duckdb/weapon_range_repo.go` : deux requêtes via l'helper, `WHERE e.match_id IN (?)
      AND e.feed_killer_xuid = ?` (côté tueur) et `... AND e.victim_xuid = ?` (côté victime) ;
      classification `source_tag -> weapon_key` par le `port.KillSourceClassifier` injecté ; distance
      par `hypot3D` (déjà l'unique formule du paquet) ; `deltaZ = killer_z - victim_z`.
      FAIT — les deux côtés sous UN SEUL emprunt du lecteur partagé (le lease est la ressource la
      plus disputée du process). `analysis.MeasuredKill` a été ÉTENDU (jamais réécrit) de la clé
      du frag — `MatchID`, `KillerXUID`, `TimeMS` — sans quoi le lot 4 ne pourrait pas apparier
      l'entame à son coup fatal par frag (D5 / item 4.2) ; l'agrégat ne groupe toujours que par
      (arme, côté), les nouveaux champs ne le touchent pas. Le dénivelé est écrit BRUT des deux
      côtés, jamais inversé ici : un test dédié l'épingle (une seconde inversion annulerait celle
      de `WeaponRangeAggregate`, et le produit dirait l'exact contraire de la vérité).
      `appendXUIDFilter` (weapon_kills_repo.go) a été généralisé de l'alias de table à la COLONNE
      complète, ses 4 appelants migrés : sans quoi ce lecteur, qui filtre sur DEUX colonnes de
      xuid de la même table, aurait posé une QUATRIÈME copie du sous-select `xuid_aliases`.
- [x] 3.5 Table `kill_positions_latest` absente ou vide -> `games.ErrCapabilityNotSupported`
      (même contrat que `WeaponAccuracyRepository`).
      FAIT, avec une NUANCE ASSUMÉE que le plan confondait : table ABSENTE ->
      `ErrCapabilityNotSupported` ; table PRÉSENTE MAIS VIDE -> zéro ligne, AUCUNE erreur. Les
      deux états sont distincts et deux tests les séparent. Les confondre ferait dire « ce titre
      ne sait pas faire » à un scope simplement pas encore décodé — et sur `kill_openings`, dont
      le backfill n'a pas tourné, ce serait le cas NOMINAL.
- [x] 3.6 `slog.DebugContext` : lignes lues, armes retenues, armes sous seuil ;
      `slog.ErrorContext(ctx, "...", "err", err)` sur toute erreur, ligne illisible comprise
      (jamais avalée).
      FAIT — un Debug par côté (`lignes_lues`, `retenues`, `hors_registre` : un trou de registre
      se voit sans relire le code), Error sur lecteur indisponible, requête échouée, ligne
      illisible et itération interrompue. « Armes sous seuil » n'est PAS journalisé ici et ne peut
      pas l'être : le seuil est en analysis (D9), le repo ne le connaît pas.
- [x] 3.7 Test DuckDB `:memory:` : schéma créé PAR LES MIGRATIONS RÉELLES
      (leçon `reference_test_ddl_copies_derivent`) ; cas : nominal deux côtés, unanimité violée,
      position manquante d'un côté, table absente.
      FAIT — `weapon_range_repo_test.go` (12 tests) : nominal deux côtés avec dénivelé BRUT
      vérifié dans les deux sens, résolution par gamertag via `xuid_aliases`, unanimité violée,
      position absente ET partielle, passe non publiable, hors scope, entame nominale (avec
      l'instant du kill), entame sur table vide, entame sur vue absente, filtres trop larges
      (les deux formes, sur les deux méthodes), classificateur nil, source hors registre. La
      fixture partagée `insertKill` a été refactorée en `insertKillEvent(killEventFixture)` pour
      nommer la VICTIME sans recopier l'INSERT — un seul INSERT dans la fixture, toujours.

- [x] 3.8 **Table `kill_openings`** (D5, si le gate 2.5 est GO) : migration append-only
      (`id` PK séquence, `written_at`, `match_id`, `killer_xuid`, `time_ms`, six coordonnées à
      T-lead) + vue `kill_openings_latest`, dans `steps_appendonly_*` et `migration/order.go`,
      recette ADR 0026 (`append_only_rebuild.go`). Isolation par titre vérifiée par le test
      `synthetic_title_b/migration_isolation_test.go`.
      FAIT — `games/halo_infinite/migrations/steps_shared_kill_openings.go`, à côté de sa sœur
      `kill_positions` plutôt que dans `steps_appendonly_misc.go` : ce fichier-là regroupe des
      CONVERSIONS (`ApplyAppendOnlyRebuild`), et `kill_openings` est NET-NEUVE — elle se crée
      DIRECTEMENT append-only, patron `match_bomb_stats`. Nom au canonicalOrder juste après
      `shared_append_only_kill_positions_v1`. Tests verts : `TestCanonicalCoversGlobalAndTitle`,
      `TestSyntheticTitleB_MigrationIsolation`, tout `games/halo_infinite/migrations`.
      Deux tables et pas six colonnes de plus sur `kill_positions` : les deux couvertures ne
      peuvent PAS être les mêmes, et les fusionner obligerait à réécrire une ligne append-only.
- [x] 3.9 Persister INSERT-only `KillOpeningPersister` via `BatchBuilder.AddKillOpenings` —
      calqué sur `kill_position_persister.go` ; allowlist `no_art_patterns_test.go` inchangée.
      FAIT — `persist/kill_opening_persister.go` + `KillOpeningInsert` (rows.go) +
      `Shared.KillOpenings` (batch.go) + `AddKillOpenings` (builder.go) + `persistKillOpenings`
      (shared_persister.go), plus 5 tests d'intégration. Type DISTINCT de `KillPositionInsert`
      malgré une forme identique : un type partagé laisserait écrire, sans que rien ne rougisse,
      une passe d'entames dans `kill_positions` — c'est-à-dire des positions fausses de 1,5 s
      présentées comme celles du coup fatal. `no_art_patterns_test.go` INCHANGÉ (vérifié : suite
      `internal/sync` verte). `append_only_state_guard_test.go` s'est vu AJOUTER `kill_openings`
      (recette ADR 0026 étape 5) — un durcissement, jamais un élargissement d'allowlist.
- [x] 3.10 Producteur : dans `killcollector/positions.go`, après `BuildKillPositions`, un second
      appel avec `ShiftKillRefs` ; compteurs ADR 0009 (`killsource_openings_lignes_ecrites`,
      `..._morts_sans_position`) ; échec = journalisé + compté, JAMAIS fatal à la passe de morts.
      FAIT — `buildPositionRows` rend désormais une `passePositions` (les deux jeux de lignes
      d'UNE seule lecture du film) ; `persistOpenings` écrit sous son propre lease court,
      best-effort au carré (son échec ne fait retomber ni la passe de morts ni celle des
      positions). Quatre compteurs :
      `killsource_openings_{matchs_couverts,lignes_ecrites,morts_sans_position,erreurs_ecriture}`
      — zéro ligne ne compte PAS un match couvert, sinon le compteur serait muet sur la seule
      question qu'il sert à poser. `toKillOpeningRows` est une PROJECTION DÉDIÉE (consigne du
      pilote) : elle réassocie chaque entame à l'instant DU KILL, seule clé par laquelle
      `kp.time_ms = e.time_ms` peut rendre quelque chose. La réassociation est ARITHMÉTIQUE et
      non par index, à dessein — `BuildKillPositions` écarte les morts non localisables, donc sa
      sortie n'est pas alignée sur l'entrée et un appariement par rang attribuerait à une entame
      l'instant d'une autre mort. Test dédié :
      `TestToKillOpeningRows_LInstantRedevientCeluiDuKill`.
- [x] 3.10 bis **Filtre « même vie » de la position d'entame — FAIT le 2026-09-06** (bascule au
      merge de `feat/duels`, correctif A de la revue adversariale ci-dessous). Le producteur
      appelle `replay.BuildKillOpenings(positions, slotXUID, kills, int64(originUS))` et
      `toKillOpeningRows` NE réadditionne plus `replay.OpeningLeadMS` — les deux gestes ont
      basculé ensemble, comme la réserve l'exigeait. Nouveau compteur ADR 0009
      `killsource_openings_cotes_hors_vie` (`rep.OpeningOutOfLife`), journalisé dans les deux
      traces de la passe. Entrée du `.ai/V7.5/REGISTRE_REPORTS.md` CLOSE. L'énoncé du report
      d'origine est conservé ci-dessous, pour que la condition de reprise reste lisible.
      <br>_Énoncé du report (2026-09-06, avant merge)_ :
      Consigne du pilote (2026-09-06) : si un joueur a réapparu entre T-1,5 s et T, l'instant
      décalé peut tomber sur son premier échantillon de vie et l'« entame » publiée serait un
      point de réapparition. La correction est `replay.BuildKillOpenings(pos, slotXUID, kills,
      offsetUS)` — décalage ET filtre de vie, `KillRef.TimeMS` déjà ramené à l'instant du kill —
      attendue sur `feat/duels`. VÉRIFIÉ SUR PIÈCES au moment d'écrire : `git grep "func
      BuildKillOpenings" feat/duels` ne rend RIEN. Le filtre n'est PAS réimplémenté ici (consigne
      explicite, et règle « deux décodeurs du même fait divergeraient »). La réserve est écrite au
      point d'appel (`buildPositionRows`) avec sa condition de bascule EN DEUX GESTES : l'appel
      devient `replay.BuildKillOpenings(...)` ET `toKillOpeningRows` cesse de rajouter
      `OpeningLeadMS` — l'un sans l'autre décalerait toutes les lignes de 1,5 s. Inscrit au
      `.ai/V7.5/REGISTRE_REPORTS.md`.
- [~] 3.11 `backfill-killsource` : la capture d'entame suit `WithPositionCapture` (même
      drapeau, même catalogue de bornes) — aucun nouveau flag.
      COUVERT PAR 3.10, RIEN À ÉCRIRE — vérifié sur pièces :
      `cmd/levelup/cmd_backfill_killsource.go:259-263` appelle `WithPositionCapture(mapNames,
      mapBounds)`, qui arme `c.mapNames`/`c.mapBounds` ; `collectPositions` refuse la passe
      entière quand ils sont nils, et la production d'entames vit À L'INTÉRIEUR de cette passe.
      Le drapeau existant les gouverne donc toutes les deux, et aucun flag n'a été ajouté
      (règle 11 : pas de feature OFF « pour plus tard »).
- [~] 3.12 L'helper 3.1 accepte la table source en paramètre (`kill_positions_latest` |
      `kill_openings_latest`) — une seule jointure pour les deux lectures.
      COUVERT PAR 3.1 : `measuredKillsQuery(table measuredPositionsTable, where string)`, avec
      les deux constantes `positionsAtKill` / `positionsAtOpening`. Un type nommé plutôt qu'une
      `string` : une table arbitraire n'a rien à faire dans cette jointure.

**Gate** (CORRIGÉ le 2026-09-06 : la forme écrite à l'origine, sans `-tags=integration`, ne
lançait AUCUN test — cf. « Découvertes ») :
`cd apps/go-api && go test -tags=integration -p 1 -count=1 ./internal/platform/duckdb/ -run 'WeaponRange|WeaponOpening|KillMeasured|KillDistance|Jointure|Proprietaire' -v`
— `KillDistance` inclus : la migration de 3.1 ne doit rien changer au POC. Puis
`go test -tags=integration -p 1 ./internal/persist/... ./internal/sync/killcollector/... ./internal/migration/...`
(3.8-3.10 touchent persist/sync/migration : run nu = FAUX VERT).

Gates réellement exécutés le 2026-09-06 (`CGO_ENABLED=1` — DuckDB exige CGO ; `GOCACHE` et
`GOLANGCI_LINT_CACHE` isolés dans le worktree), tous verts :

- `go test ./internal/platform/duckdb/ -run 'WeaponRange|KillMeasured|KillDistance' -v` -> `ok
  [no tests to run]`. **Le gate écrit dans le plan ne lance AUCUN des tests qu'il vise** : ces
  tests-là sont `//go:build integration` (le POC KillDistance l'était déjà). La forme utile est
  `go test -tags=integration -p 1 ...` : 23 tests, tous PASS (9 KillDistance inchangés,
  12 WeaponRange/WeaponOpening, 2 garde-rails).
- `go test ./internal/platform/duckdb/ ./internal/port/... ./internal/persist/...
  ./internal/sync/killcollector/... ./internal/migration/... ./internal/games/...` -> 26 paquets `ok`.
- `go test -tags=integration -p 1 ./internal/persist/... ./internal/sync/... ./internal/migration/...
  ./internal/games/...` -> code de sortie **0** (vérifié sur le code de sortie, pas sur un filtre
  de sortie : un `grep FAIL` nu attrape des logs applicatifs).
- `go vet` sur les cinq paquets + `go vet -tags=integration ./internal/platform/duckdb/` -> silencieux.
- `gofmt -l internal` -> sortie vide.
- `golangci-lint run --new-from-merge-base=origin/main` -> **0 issues** (baseline non accrue).


### Revue adversariale ronde 1 (2026-09-06, branche `feat/duels-lot3`)

Trois relecteurs indépendants ont relu le lot 3. Le pilote a retenu les constats ci-dessous ;
tous sont traités, aucun autre geste n'a été posé (les découvertes hors périmètre sont au
registre du bas de ce fichier). La base a d'abord été fusionnée depuis `feat/duels`
(`replay.BuildKillOpenings` y est arrivée avec les correctifs du lot 2).

- [x] **A — Bascule sur `replay.BuildKillOpenings` (item 3.10 bis fermé).** Les DEUX gestes
      ensemble : `composerPassePositions` appelle `BuildKillOpenings(positions, slotXUID, kills,
      originUS)` et `toKillOpeningRows` ne réadditionne plus `OpeningLeadMS`. Un côté dont
      l'entame et le coup fatal ne partagent pas la même vie est écarté (plus jamais un point de
      réapparition présenté comme une entame) et COMPTÉ : nouveau compteur ADR 0009
      `killsource_openings_cotes_hors_vie`, journalisé dans les deux traces de la passe. Aucune
      donnée d'entame n'ayant été cuite (backfill jamais lancé), rien n'est à recuire.
      `sync/killcollector/positions.go` + `positions_openings.go` (la passe d'entames a son
      fichier : positions.go dépassait 500 lignes en portant les deux).
- [x] **B1 — le test qui pince l'accord décalage ↔ instant persisté.** La couture a été
      extraite : `composerPassePositions(positions, slotXUID, kills, originUS, matchID)` est PURE
      et ne demande aucun film. `TestComposerPassePositions_LEntameEstPriseAvantLeKillEtPorte
      SonInstant` vérifie ENSEMBLE que la ligne porte le `time_ms` DU KILL et que ses
      coordonnées sont celles de l'échantillon 1,5 s AVANT — distinctes de celles du kill et de
      celles 1,5 s après. Un second test épingle la remontée du filtre de vie
      (`OpeningOutOfLife`). MUTATION : signe du décalage inversé dans `BuildKillOpenings` ->
      `killer_x = 6.5, attendu 3.5` + « le SIGNE du décalage est inversé ». Rouge, restauré vert.
- [x] **B2 — la passe d'entames est exercée.** `ecrireLesDeuxPasses` orchestre les deux
      écritures ; `TestEcrireLesDeuxPasses_EcritLesDeuxTables` (base montée par les migrations
      RÉELLES) vérifie que les deux vues `_latest` portent leur ligne, `time_ms` et coordonnées
      comprises. MUTATION : appel à `persistOpenings` retiré -> `kill_openings_latest = 0
      ligne(s), attendu 1`. Rouge, restauré vert.
- [x] **C1 — double frag : la garde ne se contournait plus par le filtre de côté.** Le `WHERE`
      s'applique AVANT le `GROUP BY` : une lecture côté victime ne voyait qu'une ligne et jugeait
      l'unanimité d'un singleton. Le groupe complet `(match_id, feed_killer_xuid, time_ms)` est
      désormais calculé par la sous-requête `fragSolo`, sur la vue ENTIÈRE, avec `count(*) = 1`
      EN PLUS de l'unanimité. Le commentaire qui affirmait le contraire est réécrit.
      `TestWeaponRange_DoubleFragMemeArme_ExcluDesDeuxCotes` couvre les deux côtés. Durcissement,
      pas correction de chiffres : 0 groupe multi-victimes sur 138 293 événements du corpus.
- [x] **C2 — `victimXUID` supprimé** de `killMeasured` (projection, scan, champ) : projeté,
      scanné, jamais lu. La garde C1 n'en a pas besoin — elle compte des lignes, elle ne les
      nomme pas.
- [x] **C3 — le chemin builder de `kill_openings` est SUPPRIMÉ.** `BatchBuilder.AddKillOpenings`
      (0 appelant), `SharedBatch.KillOpenings` et l'appel inconditionnellement no-op de
      `SharedPersister.Persist` : c'était le « au cas où » de l'anti-pattern n°1. L'INSERT vit
      désormais dans `kill_opening_persister.go`, à côté de son unique appelant, et
      `KillOpeningPersister` est le SEUL chemin d'écriture.
- [x] **C4 — `shared_persister.go` revient à 650 lignes** (679 après le lot 3, 650 avant lui) :
      la suppression C3 suffit, il ne reste aucun code d'entame dans ce fichier — d'où pas de
      `shared_persister_kill_openings.go`, qui aurait été un fichier sans appelant local.
- [x] **C5 — le garde-rail mord.** (a) le motif ne cherche plus `JOIN <table>` mais les NOMS
      eux-mêmes, `kill_positions` / `kill_openings` et leurs vues, dans le CODE (les lignes de
      commentaire sont retirées avant la recherche) : une jointure écrite `FROM … JOIN …`, une
      composée par `Sprintf` et surtout une posée sur les TABLES BRUTES sont désormais captées.
      `KillDistanceRepo` nommait une table dans un message de journal : il passe par la constante
      `positionsAtKill`. (b) le pin des gardes porte sur `measuredKillsSQLTemplate` — la VALEUR
      de la constante — et non sur les octets du fichier, que le commentaire d'en-tête satisfaisait.
      MUTATIONS : garde d'unanimité retirée de la REQUÊTE (commentaire intact) -> rouge ; copie
      `FROM kill_positions kp JOIN …` posée dans `kill_distance_repo.go` -> rouge. Restaurés verts.
- [x] **C6 — l'échec d'écriture d'une entame ne peut plus se taire.**
      `TestPersistOpenings_EchecDEcriture_CompteEtNePublieRien` : compteur d'erreurs +1, compteur
      de matchs COUVERTS +0, et AUCUNE trace « entames decodees » (journal capturé). MUTATION :
      `_ = persistKillOpenings(...)` -> les trois assertions rougissent, dont
      `matchs_couverts a bougé de 1, attendu 0`. Restauré vert.
- [x] **C7 — l'échec des positions n'annule plus l'entame.** `ecrireLesDeuxPasses` journalise et
      compte l'échec des positions puis appelle la passe d'entames dans TOUS les cas : le code
      fait enfin ce que la doc de `writeOpenings` promettait. Test dédié (table `kill_positions`
      supprimée, l'entame s'écrit quand même).
- [x] **C8 — la garde `rowsWritten == 0` est testée** :
      `TestPublishOpeningsPass_ZeroLigneNeCompteAucunMatch`.
- [x] **D — `decode_pass` sur `kill_openings` (COMMIT SÉPARÉ, le dernier de la branche).** La vue
      arbitrait par CLÉ (`written_at` puis `id` par `(match_id, killer_xuid, time_ms)`) : un
      re-décodage qui ne résout PLUS une entame — ce que le filtre de vie de A fait
      régulièrement — laissait la ligne de la passe précédente servie à jamais. La table gagne
      `decode_pass VARCHAR NOT NULL` (migration modifiée EN PLACE : la table n'a jamais été créée
      nulle part, ni prod ni backfill — écrit et daté dans le commentaire de migration), le
      persister le tire par `newDecodePassID()`, et la vue devient « DERNIÈRE PASSE ENTIÈRE PAR
      MATCH » sur le modèle exact de `match_kill_events_latest`. `ReDecodeSupersede` prouve
      désormais qu'une passe B SANS la ligne de A la fait DISPARAÎTRE de la vue.
      **Ce point est isolé dans son propre commit pour pouvoir être retiré d'un `git reset` si
      l'utilisateur le refuse.**
- [x] **E — gates du plan corrigés** (lots 3, 4, 5 et 6) : toute commande visant
      `./internal/platform/duckdb/` porte `-tags=integration -p 1` — un run nu rend « no tests to
      run » et c'est un faux vert. Le gate du lot 3 lui-même, dont le lot avait constaté qu'il ne
      lançait aucun test, est corrigé plutôt que laissé en l'état.

## Lot 4 — Service, capability, contrat API

**Résidus de la ronde 2 du lot 3 (4 P2, statués par le pilote le 2026-09-06) — à traiter EN TÊTE du lot 4, avant 4.1, avec preuve dans le CR :**

- [x] 4.0a `platform/duckdb/kill_measured.go:137-146` — la sous-requête `fragSolo` n est bornée par aucun scope : sur la forme `e.match_id IN (...) AND <côté> IN (...)`, DuckDB ne pousse pas le filtre (SEQ_SCAN complet, mesuré ×15,6 : 192 ms contre 12 ms sur 140 000 événements). Borner `fragSolo` par le même `match_id IN (...)` que la requête principale (paramètres dupliqués), prouver par `EXPLAIN` que le SEQ_SCAN de la branche porte `Filters: match_id`, et vérifier que `KillDistanceRepo.LoadMatch` (forme `= ?`) reste identique.
      **FAIT (2026-09-06)** — `measuredKillsQuery(table, fragSoloScope, where)` : la sous-requête porte désormais SON scope (`s.match_id IN (...)` / `s.match_id = ?`), paramètres dupliqués, scope AVANT le `WHERE` dans l'ordre de liaison (documenté aux trois points d'écriture : l'helper et ses deux appelants). Preuve par `EXPLAIN (FORMAT JSON)` plutôt que par le format par défaut, dont les boîtes ASCII coupent le texte des filtres à 27 caractères : `kill_measured_scope_test.go` décode le plan et vérifie que TOUT `SEQ_SCAN` de `match_kill_events` porte un `Filters` sur `match_id`. Forme réelle du plan, vérifiée sur pièces : DuckDB matérialise `match_kill_events_latest` en UNE `CTE` relue par les deux branches — il n'y a qu'un balayage, et il ne peut porter le filtre que si les DEUX branches le demandent. MUTATION PROUVÉE ROUGE : `fragScope` ramené à `TRUE` -> « balayage 2/2 : Filters = "" » (et le plan repasse à DEUX balayages non filtrés), pendant que les douze tests de résultat WeaponRange restent verts. `KillDistanceRepo` : ses neuf tests d'origine passent inchangés.
- [x] 4.0b `games/halo_infinite/migrations/steps_shared_kill_openings.go:41-44` contredit `persist/kill_opening_persister.go:62-66` : une passe B qui ne résout AUCUNE entame sur tout le match n écrit aucune ligne, donc aucun `decode_pass` neuf, et la vue continue de servir la passe A entière. Comportement ASSUMÉ (consigné) ; corriger l en-tête de migration pour qu il dise ce que la vue fait réellement (rétractation seulement si la passe B écrit au moins une ligne) et ajouter le cas « passe B vide » au test `ReDecodeSupersede` en l ASSERTANT tel quel.
      **FAIT (2026-09-06)** — en-tête de migration complété d'une section « la rétractation exige que la passe suivante écrive au moins une ligne » : la vue rend la dernière passe QUI EXISTE, une passe vide n'écrit aucune génération, la passe précédente reste servie entière. Le pourquoi est écrit (une sentinelle « passe vide » serait une donnée inventée, et il faudrait alors distinguer en base un match sans entame lisible d'un match jamais décodé). Test `TestKillOpeningPersistPass_PasseVideNeRetractePas` : passe A à une ligne, passe B nulle -> la vue sert toujours la ligne de A, ASSERTÉ TEL QUEL.
- [x] 4.0c `kill_measured.go:142` — `HAVING count(*) = 1` change le comportement du POC `KillDistanceRepo` sur un double frag à la MÊME arme (avant : 1 mesure sur une position arbitraire ; maintenant : exclu). Impact mesuré nul (0 groupe sur 138 293 événements). STATUÉ : changement accepté ; l écrire dans le commentaire de `kill_distance_repo.go` (ex-phrase « héritée du POC » retirée sans énoncé).
      **FAIT (2026-09-06)** — l'énoncé est écrit sur `killDistanceWhere` (`kill_distance_repo.go`) : garde d'unicité du frag, ce qu'elle change pour le POC (avant : une mesure sur une position arbitraire ; maintenant : exclu), l'impact mesuré nul sur le corpus et la référence au résidu. Vérifié sur pièces : la phrase « héritée du POC » n'existait plus nulle part (retirée au lot 3), il n'y avait donc rien à remplacer, seulement à énoncer.
- [x] 4.0d `sync/killcollector/positions_openings.go:59` — sur échec d écriture, le `return` précède `publishOpeningsPass` : `cotes_hors_vie` et `morts_sans_position` ne bougent pas alors que la doc (:117-121) et le test (:159-160) promettent « il compte MÊME quand rien n est écrit ». Publier les compteurs de LECTURE avant le retour d échec, garder `matchs_couverts`/`lignes_ecrites` conditionnés au succès ; test.
      **FAIT (2026-09-06)** — les deux pertes de LECTURE sont extraites dans `publishOpeningsReadCounters(rep)`, appelée par les DEUX chemins de sortie de `persistOpenings` (succès et échec d'écriture), chacun exactement une fois ; `matchs_couverts` / `lignes_ecrites` restent conditionnés au succès. La trace d'erreur porte désormais les trois nombres de la lecture. Test `TestPersistOpenings_EchecDEcriture_CompteQuandMemeLaLecture` ; MUTATION PROUVÉE ROUGE (appel retiré -> les deux compteurs bougent de 0, attendu 3 et 5). La doc de `publishOpeningsPass` est corrigée : elle décrivait un comportement que le code n'avait pas.


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
puis, parce que le lot branche le repo DuckDB du lot 3 :
`go test -tags=integration -p 1 -count=1 ./internal/platform/duckdb/`.
**`-tags=integration -p 1` N'EST PAS OPTIONNEL sur `platform/duckdb`** : toute la famille de
tests de ce paquet est derrière ce tag, un run nu rend « no tests to run » et c'est un FAUX
VERT (constaté au lot 3, cf. « Découvertes »). Vérifier le CODE DE RETOUR, jamais un filtre
sur « FAIL » (il attrape des logs applicatifs).

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
— lot WEB : aucun `go test` ici. Si le lot devait toucher au Go (il ne le doit pas), toute
commande visant `./internal/platform/duckdb/` porterait `-tags=integration -p 1`.

---

## Lot 6 — Livraison

- [ ] 6.1 Skill `delivery-checklist`.
- [ ] 6.2 `cd apps/go-api && go test -count=1 ./... && go vet ./...` puis
      `go test -tags=integration -p 1 -count=1 ./...`. **LE RUN NU NE VAUT PAS GATE POUR
      `platform/duckdb`** : toute la famille de tests de ce paquet est derrière
      `//go:build integration`, un run sans le tag rend « no tests to run ». C'est la SECONDE
      commande qui fait foi pour ce paquet ; `-p 1` non négociable (DuckDB mono-writer) ; code
      de sortie vérifié (`$?`), jamais un filtre sur « FAIL » — il attrape des logs applicatifs.
- [ ] 6.3 `make go-api-lint` — baseline non accrue.
- [ ] 6.4 Gate visuel : capture de la section Synthèse soumise à l'utilisateur, témoins nommés.
- [ ] 6.5 Entrée `thought_log.md` ; `.ai/project_map.md` si la carto bouge.
- [ ] 6.6 Commit + push + **CI surveillée jusqu'au niveau JOB** (`gh run list --branch feat/duels`) ;
      tout rouge se répare, même préexistant.

---

## Lot 7 — Duels — **FERMÉ, NO-GO du gate du lot 1 (2026-09-06)**

À n'ouvrir QUE si le lot 1 rend GO. **Le lot 1 rend NO-GO** (D = 52,6 % contre 60 %) : les cinq
items ci-dessous sont statués `[!]` — non traités, aucun code écrit.

- [!] 7.1 Table append-only `match_engagements` + vue `_latest` (recette ADR 0026).
      NON TRAITÉ : gate du lot 1 échoué, cf. `.ai/V7.5/film_re/SONDE_DUELS_BOUCLIER_2026-09-06.md`.
- [!] 7.2 Producteur dans `killcollector` (INSERT-only via `BatchBuilder.Submit` — ADR 0019/0030).
      NON TRAITÉ : même cause. Rien à produire tant qu'une chute de bouclier ne désigne personne.
- [!] 7.3 Classification : duel / élimination / non conclu ; effectif à la résolution
      (tête-à-tête, 2v1, 1v2) — l'équipe vient du roster, disponible côté base.
      NON TRAITÉ : c'est exactement l'affirmation que la mesure interdit de publier — dans
      47,4 % des fenêtres, plusieurs adversaires sont candidats.
- [!] 7.4 Lecture, service, capability `duels`, chart. NON TRAITÉ : rien à servir.
- [!] 7.5 Gate de vérité terrain : l'utilisateur visionne 15 à 20 engagements dans Theater et
      tranche. NON TRAITÉ : on ne soumet pas à l'utilisateur une classification dont on a
      mesuré qu'elle se trompe d'adversaire une fois sur deux.

Report inscrit au `.ai/V7.5/REGISTRE_REPORTS.md` avec sa condition de reprise (« un canal qui
porte l'AUTEUR du dégât à la densité du bouclier : flux de dégâts dense, compteur d'état ECS
répliqué, ou source hors film »).

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
- (lot 3 — revue ronde 1, 2026-09-06) **`kill_positions` porte EXACTEMENT le défaut que D vient
  de corriger sur `kill_openings`** : sa vue `kill_positions_latest` arbitre par CLÉ
  (`written_at`, `id` par `(match_id, killer_xuid, time_ms)`), pas par PASSE. Un re-décodage qui
  ne retrouverait plus une position — film re-téléchargé plus court, pont d'identité qui perd un
  slot — laisserait la ligne de la passe précédente servie à jamais, mélangée aux nouvelles. Le
  cas est moins probable que pour l'entame (les positions du coup fatal ne dépendent d'aucun
  filtre de vie), et surtout la table est PEUPLÉE en production : lui ajouter `decode_pass`
  exigerait une reconstruction, donc une décision utilisateur. NON TRAITÉ — hors périmètre de
  cette revue, qui ne portait que sur le diff du lot 3. À reprendre avec le backfill de
  `kill_openings`, si l'utilisateur l'ouvre : les deux tables se recuiraient de la même passe.
- (lot 3, 2026-09-06) **`kill_positions` n'a JAMAIS été inscrite à `appendOnlyStateTables`**
  (`internal/sync/append_only_state_guard_test.go`) alors qu'elle est append-only depuis G.2
  (2026-08-30) : l'étape 5 de la recette ADR 0026 a été sautée à sa conversion, exactement
  comme elle l'avait été pour `match_usage_players`/`match_usage_films` (dont l'inscription
  porte la mention « l'inscription manquait à l'arrivée de la branche »). `kill_openings` y a
  été inscrite par ce lot ; sa sœur NON — hors périmètre. Rien ne casse aujourd'hui (aucun
  writer ne la mute), mais le garde-rail ne la protège d'aucun DELETE futur.
- (lot 3, 2026-09-06) **Le sous-select `SELECT xuid FROM xuid_aliases WHERE gamertag = ?` existe
  en TROIS copies** dans `platform/duckdb` : `appendXUIDFilter` (weapon_kills_repo.go, désormais
  généralisé à la colonne et partagé par 4 appelants dont le nouveau lecteur de portée),
  `killsource_weapon_kills_repo.go:261` et `objective_index_repo.go:106`. Les deux dernières
  sont préexistantes et hors périmètre ; à la quatrième copie la règle n°6 imposera de les
  migrer sur le helper. Ce lot a évité d'en créer une en généralisant le paramètre du helper
  plutôt qu'en recopiant.
- (lot 3, 2026-09-06) **Le gate écrit pour ce lot ne lançait aucun test.** `go test
  ./internal/platform/duckdb/ -run 'WeaponRange|KillMeasured|KillDistance'` rend « no tests to
  run » : toute cette famille de tests est `//go:build integration`. Le piège est générique — un
  gate sans `-tags=integration` sur un paquet dont les tests DB sont tagués est un FAUX VERT, et
  le plan le disait déjà pour persist/sync/migration sans le voir pour `platform/duckdb`. À
  corriger dans les gates des lots suivants.
- (lot 2 — revue ronde 1, 2026-09-06) **Ce que les regex de `weapon_range_guard_test.go` ne
  captent PAS**, consigné et NON corrigé : une comparaison SQL MULTILIGNE (les 80 caractères de
  contexte ne franchissent pas le retour à la ligne), un ALIAS qui masque la colonne
  (`... AS dz ... WHERE dz > 1.0` — élargir `delta_?z` à `dz` ferait tomber la moitié du dépôt),
  et le seuil de publication écrit sans son nom (`HAVING count(*) >= 8`). Accepté parce que le
  lot 3 a pour consigne de n'écrire AUCUN seuil ni comparaison de dénivelé en SQL : le repo rend
  les frags MESURÉS, les seuils restent en `analysis`. Si cette consigne bougeait, le garde-rail
  de NOMMAGE du lot 3.2 (`kill_measured.go`) couvrirait mieux le motif qu'une regex de littéral.
- (lot 2 — revue ronde 1, 2026-09-06) La formulation de **D9** ci-dessus (« leur NOMBRE est
  publié — N armes sous le seuil ») a le même défaut que la doc corrigée en F5 : le compteur
  porte des COUPLES (arme, côté), pas des armes. La doc de `WeaponRangeMinMeasured` fait foi ;
  le rendu du lot 5.3 dit déjà « frags : N · morts : M ». Non corrigé ici : D9 est un relevé de
  décision daté, on ne réécrit pas une décision passée.
- (lot 2, 2026-09-06) **Le délai médian entre le premier dégât CAPTURÉ et la fin de vie vaut
  551 à 1 335 ms** sur les quatre films. C'est un second angle sur le constat de la sonde n°1
  (« la riposte vit dans les deux premières secondes ou n'existe pas ») et c'est aussi ce qui
  biaise la population de validation du proxy d'entame vers les échanges courts. Utile au lot 7
  s'il s'ouvre ; rien à traiter ici.
- **Revue lot 2, ronde 2 (2026-09-06) — P2 consigné, non corrigé (borne de boucle : pas de
  ronde 3)** : `replay/killpos_opening.go:170`, la marge AVAL de 120 ms de `coversInstant`
  (après la fin de vie) n'est couverte par aucun test — la supprimer laisse la suite verte,
  alors que sans elle aucune entame de victime ne passerait. La direction « marge trop grande »
  est couverte (`lifeGapUS` fait rougir `RefuseUnPointDApparition`), la direction « marge
  absente » ne l'est pas. Ronde 2 : 0 P0, 0 P1 (contre 3 P1 en ronde 1), 12 conditions tenues,
  7 mutations dont 6 rouges. Équivalence de `BuildKillPositions` prouvée par test différentiel
  sur 20 000 tirages.
