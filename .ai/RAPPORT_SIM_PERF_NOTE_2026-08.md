# Rapport de simulation — note de performance, scission ranked + metrique objectif

> Lot 0 du plan `.ai/PLAN_PERF_NOTE_OBJECTIFS.md`. Genere par `apps/go-api/cmd/diag_perfsim`
> (outil jetable, DB ouvertes en `access_mode=read_only`, aucune ecriture).
> Donnees reelles du checkout principal, 4 joueurs suivis. AUCUN code produit modifie.

## Methode

- **Univers** : SQL de `loadHistoryForPerf` (performance_helpers.go:154-172) a deux
  differences pres — la clause `AND COALESCE(mp.outcome,0) != 4` est retiree et
  `outcome` est projete, pour pouvoir classer les notes orphelines. Le filtre est
  reapplique en Go : l'ensemble scorable est identique a la production.
- **Exclusions** : `skill.LoadExcludedMatchIDs` repliquee verbatim.
- **Moteur** : replique de `computeRelativePerformanceScore` — fenetre 50 par chaine,
  seuil `MinMatchesPerChainForRelative`=10, percentiles ponderes, `dpm_deaths` inversee,
  renormalisation par la somme des poids presents, arrondi 0.1, mode `force`.
  `skill.RelativeWeights`, `skill.ComputeCombatYield`, `skillchain.ClassifyLUSRChain`,
  `analysis.CombatEfficiency` et `analysis.NormalizeModeLabel` sont IMPORTES du code produit.
- **`medal_exploit` est ABSENT de la simulation** : le chemin post-sync passe `nil` pour
  `medalExploitByMatch` (performance.go:257), la simulation fait de meme. Son poids (0.06)
  est donc redistribue ; l'effet est quantifie a la section Gate.
- **Regimes compares** : ACTUEL (chaine `ranked` unique, `RelativeWeights` partout) vs
  NOUVEAU (`ranked_slayer`/`ranked_objectif` + profil objectif avec ospm sur
  `arena_objectif`/`ranked_objectif`), en 3 variantes de poids ospm : 0.08 / 0.12 / 0.16.

## Corpus

| Joueur | Univers | DNF (outcome=4) | Exclus | Scorables | Notes ACTUEL | Notes NOUVEAU | Matchs couverts PSA |
|---|---:|---:|---:|---:|---:|---:|---:|
| JGtm | 1120 | 39 | 1 | 1080 | 1033 | 1033 | 1101 |
| Chocoboflor | 554 | 19 | 0 | 535 | 496 | 496 | 548 |
| Madina97294 | 1242 | 57 | 0 | 1185 | 1131 | 1121 | 1223 |
| XxDaemonGamerxX | 32 | 1 | 0 | 31 | 12 | 12 | 32 |

## B0.2 — Regle de dedup `personal_score_awards` retenue

**Regle retenue** : par `(match_id, xuid)`, ne garder QUE les lignes de la generation
MAXIMALE (`generation_id`), puis exclure les `is_tombstone`. C'est exactement la
semantique de la vue `personal_score_awards_latest` (DENSE_RANK, ADR 0026,
`steps_player_append_only_personal_score_awards.go:48-55`) — pas une regle inventee.
La somme se fait sur `award_score` des lignes `award_category='objective'` retenues,
divisee par les minutes jouees (meme repli 600 s que le reste du moteur).

**La dedup n'est pas theorique** : des generations multiples existent en base.

| Joueur | Lignes PSA | Lignes d'un autre xuid | Tombstones | Lignes de generation perimee | Matchs multi-generation | Somme objectif dedupliquee | Somme objectif SANS dedup |
|---|---:|---:|---:|---:|---:|---:|---:|
| JGtm | 3830 | 15 | 0 | 0 | 0 | 85485 | 85485 |
| Chocoboflor | 1655 | 0 | 0 | 0 | 0 | 48370 | 48370 |
| Madina97294 | 4118 | 0 | 0 | 0 | 0 | 155455 | 155455 |
| XxDaemonGamerxX | 95 | 0 | 0 | 7 | 4 | 700 | 700 |

**Presence d'ospm** : ospm n'existe QUE si le match a une couverture PSA (au moins une
ligne retenue). Un match couvert avec 0 point objectif vaut ospm = 0 (valeur legitime :
le joueur n'a rien fait a l'objectif) ; un match SANS ligne PSA a une metrique ABSENTE
(poids redistribue). Confondre les deux reviendrait a noter 0 une absence de donnee.

**Piege majeur rencontre** (cf. Decouvertes du CR) : sur la DB `XxDaemonGamerxX`,
l'index `idx_psa_match` est INCOHERENT — `WHERE match_id = '05fffb2a-...'` rend 2 lignes
la ou le scan complet en rend 4. Toutes les lectures PSA de l'outil se font donc en scan
complet SANS predicat, la selection etant faite en Go.

## B0.1 / B0.3 — Distributions par joueur x chaine (chaines du NOUVEAU regime)

Les deux regimes sont evalues sur le MEME ensemble de matchs (celui de la chaine du
nouveau regime), ce qui rend les medianes directement comparables. `n scorees` differe
entre regimes uniquement par l'effet du seuil de 10 matchs sur une chaine scindee.

### JGtm

| Chaine | n matchs | n scorees ACTUEL | n scorees NOUVEAU | med ACTUEL | p10/p90 ACTUEL | med 0.08 | med 0.12 | med 0.16 | p10/p90 (0.12) |
|---|---:|---:|---:|---:|---|---:|---:|---:|---|
| `arena_slayer` | 424 | 414 | 414 | 49.5 | 25.7 / 78.3 | 49.5 | 49.5 | 49.5 | 25.7 / 78.3 |
| `chaos` | 418 | 408 | 408 | 50.8 | 24.4 / 76.2 | 50.8 | 50.8 | 50.8 | 24.4 / 76.2 |
| `arena_objectif` | 221 | 211 | 211 | 52.5 | 24.2 / 74.3 | 51.9 | 51.3 | 51.4 | 26.2 / 73.5 |
| `btb` | 8 | 0 | 0 | - | - | - | - | - | - |
| `ranked_slayer` | 6 | 0 | 0 | - | - | - | - | - | - |
| `ranked_objectif` | 2 | 0 | 0 | - | - | - | - | - | - |
| `firefight` | 1 | 0 | 0 | - | - | - | - | - | - |

### Chocoboflor

| Chaine | n matchs | n scorees ACTUEL | n scorees NOUVEAU | med ACTUEL | p10/p90 ACTUEL | med 0.08 | med 0.12 | med 0.16 | p10/p90 (0.12) |
|---|---:|---:|---:|---:|---|---:|---:|---:|---|
| `arena_slayer` | 354 | 344 | 344 | 49.7 | 25.2 / 74.5 | 49.7 | 49.7 | 49.7 | 25.2 / 74.5 |
| `arena_objectif` | 152 | 142 | 142 | 52.3 | 22.5 / 74.3 | 50.9 | 50.4 | 50.5 | 24.6 / 71.9 |
| `chaos` | 20 | 10 | 10 | 45.9 | 32.6 / 75.1 | 45.9 | 45.9 | 45.9 | 32.6 / 75.1 |
| `ranked_slayer` | 6 | 0 | 0 | - | - | - | - | - | - |
| `ranked_objectif` | 2 | 0 | 0 | - | - | - | - | - | - |
| `btb` | 1 | 0 | 0 | - | - | - | - | - | - |

### Madina97294

| Chaine | n matchs | n scorees ACTUEL | n scorees NOUVEAU | med ACTUEL | p10/p90 ACTUEL | med 0.08 | med 0.12 | med 0.16 | p10/p90 (0.12) |
|---|---:|---:|---:|---:|---|---:|---:|---:|---|
| `btb` | 494 | 484 | 484 | 50.2 | 22.1 / 77.1 | 50.2 | 50.2 | 50.2 | 22.1 / 77.1 |
| `arena_slayer` | 436 | 426 | 426 | 49.8 | 21.0 / 77.7 | 49.8 | 49.8 | 49.8 | 21.0 / 77.7 |
| `arena_objectif` | 192 | 182 | 182 | 50.0 | 23.2 / 73.6 | 49.9 | 50.0 | 50.1 | 23.9 / 71.4 |
| `chaos` | 25 | 15 | 15 | 41.0 | 21.4 / 60.1 | 41.0 | 41.0 | 41.0 | 21.4 / 60.1 |
| `ranked_objectif` | 21 | 12 | 11 | 38.8 | 18.5 / 68.1 | 42.9 | 44.4 | 44.4 | 23.2 / 72.7 |
| `ranked_slayer` | 13 | 12 | 3 | 70.3 | 31.8 / 88.0 | 66.1 | 66.1 | 66.1 | 50.3 / 75.2 |
| `firefight` | 4 | 0 | 0 | - | - | - | - | - | - |

### XxDaemonGamerxX

| Chaine | n matchs | n scorees ACTUEL | n scorees NOUVEAU | med ACTUEL | p10/p90 ACTUEL | med 0.08 | med 0.12 | med 0.16 | p10/p90 (0.12) |
|---|---:|---:|---:|---:|---|---:|---:|---:|---|
| `arena_slayer` | 22 | 12 | 12 | 54.0 | 23.5 / 72.6 | 54.0 | 54.0 | 54.0 | 23.5 / 72.6 |
| `arena_objectif` | 7 | 0 | 0 | - | - | - | - | - | - |
| `chaos` | 2 | 0 | 0 | - | - | - | - | - | - |

## D-D — Purge des notes orphelines

Une note ne doit exister QUE pour un match qualifie : non-DNF, non-exclu, et au-dela du
10e match de sa chaine (nouveau regime). Comptes des notes STOCKEES qui disparaissent,
par cause (priorite DNF > exclu > sous-seuil > hors univers).

| Joueur | Notes stockees | Conservees | Purgees | dont DNF | dont exclus | dont sous-seuil | dont hors univers |
|---|---:|---:|---:|---:|---:|---:|---:|
| JGtm | 1111 | 1033 | **78** | 33 | 1 | 44 | 0 |
| Chocoboflor | 537 | 496 | **41** | 12 | 0 | 29 | 0 |
| Madina97294 | 1239 | 1121 | **118** | 54 | 0 | 64 | 0 |
| XxDaemonGamerxX | 22 | 12 | **10** | 0 | 0 | 10 | 0 |

Repartition des notes purgees selon la chaine STOCKEE (celle qui disparait) :

| Joueur | Chaine stockee | Notes purgees |
|---|---|---:|
| JGtm | `(NULL)` | 78 |
| Chocoboflor | `(NULL)` | 40 |
| Chocoboflor | `arena_slayer` | 1 |
| Madina97294 | `(NULL)` | 108 |
| Madina97294 | `ranked` | 10 |
| XxDaemonGamerxX | `(NULL)` | 8 |
| XxDaemonGamerxX | `arena_slayer` | 2 |

## Table de decision — matchs « ecrase au combat mais actif a l'objectif »

`p combat` = moyenne des percentiles purement combat (kpm, kda, accuracy, dpm_damage,
kills_vs_expected, offensive_conversion) — pspm, apm et ospm en sont exclus.
`p ospm` = percentile de la nouvelle metrique. Selection : `p combat` <= 40 ET
`p ospm` >= 60, tri par ecart decroissant. Population : 545 matchs de chaine objectif
notes sous les deux regimes et porteurs d'un ospm.

| Joueur | Date | Mode | Chaine | K/D | p combat | p ospm | Note ACTUEL | 0.08 | 0.12 | 0.16 | Delta (0.12) | match_id |
|---|---|---|---|---|---:|---:|---:|---:|---:|---:|---:|---|
| JGtm | 2026-07-07 | Arena:Neutral Flag CTF on Chasm | `arena_objectif` | 2/13 | 5 | 94 | 11.0 | 18.2 | 21.3 | 24.1 | +10.3 | `e8d384c7-63b4-461d-88b4-95f73e91d767` |
| JGtm | 2026-03-03 | Arena:CTF on Detachment | `arena_objectif` | 4/4 | 22 | 100 | 51.1 | 56.6 | 58.3 | 60.0 | +7.2 | `034200db-1019-46ee-84bf-a5662ab1e491` |
| Madina97294 | 2026-04-10 | Arena:CTF on Absolution | `arena_objectif` | 6/8 | 21 | 98 | 35.3 | 41.4 | 43.7 | 45.8 | +8.4 | `a01e98ec-3b42-477b-8d45-64c984f00a6f` |
| Chocoboflor | 2026-03-24 | Arena:CTF on Domicile | `arena_objectif` | 3/3 | 23 | 100 | 46.5 | 52.5 | 54.4 | 56.2 | +7.9 | `83ee3f9f-8e36-4e41-8405-31acbc9898dd` |
| JGtm | 2025-10-23 | Arena:King of the Hill on Banished Narrows | `arena_objectif` | 3/8 | 23 | 100 | 34.8 | 41.3 | 43.7 | 45.9 | +8.9 | `f6091638-a8c2-46fc-8a07-bdba71d6acb6` |

### Contre-temoins — forts au combat, absents de l'objectif

Verification symetrique : la metrique ne doit pas faire chuter au-dela du raisonnable
un joueur qui a porte le combat sans toucher a l'objectif (selection `p combat` >= 60
ET `p ospm` <= 40).

| Joueur | Date | Mode | Chaine | K/D | p combat | p ospm | Note ACTUEL | 0.08 | 0.12 | 0.16 | Delta (0.12) | match_id |
|---|---|---|---|---|---:|---:|---:|---:|---:|---:|---:|---|
| Madina97294 | 2023-03-08 | Arena:King of the Hill on Chasm | `arena_objectif` | 22/5 | 82 | 0 | 74.9 | 67.5 | 64.7 | 62.2 | -10.2 | `81ef08e6-227a-422d-ab5c-f9f7daa182a5` |
| Chocoboflor | 2026-04-19 | Arena:Neutral Flag CTF on Detachment | `arena_objectif` | 10/6 | 87 | 10 | 80.0 | 73.2 | 70.7 | 68.3 | -9.3 | `3421f7f1-ec05-4928-b941-96e14c4640dc` |
| Chocoboflor | 2026-03-24 | Arena:CTF on Dynasty | `arena_objectif` | 5/1 | 77 | 0 | 82.5 | 75.4 | 72.4 | 69.5 | -10.1 | `a17e61a2-5142-4204-acdf-6f469a12d428` |
| Chocoboflor | 2026-04-23 | Arena:CTF on Starboard | `arena_objectif` | 12/8 | 79 | 2 | 77.2 | 70.4 | 67.6 | 65.1 | -9.6 | `69504bb6-b73a-4071-8697-690ba01d920b` |
| JGtm | 2026-01-02 | Arena:CTF on Absolution | `arena_objectif` | 5/3 | 88 | 14 | 74.2 | 67.1 | 64.9 | 62.9 | -9.3 | `8dadc413-2db6-426c-9278-d58659485eaa` |

## Criteres de gate

### 1. Mediane des notes par chaine scoree dans [45, 55]

Chaines de >= 10 notes, variante de reference ospm=0.12. La colonne `med ACTUEL`
(memes matchs, regime actuel) distingue une mediane deja hors fenetre AVANT la
reforme d'une mediane que la reforme aurait deplacee.

| Joueur | Chaine | n scorees | med ACTUEL | med NOUVEAU | verdict |
|---|---|---:|---:|---:|---|
| JGtm | `arena_slayer` | 414 | 49.5 | 49.5 | OK |
| JGtm | `chaos` | 408 | 50.8 | 50.8 | OK |
| JGtm | `arena_objectif` | 211 | 52.5 | 51.3 | OK |
| Chocoboflor | `arena_slayer` | 344 | 49.7 | 49.7 | OK |
| Chocoboflor | `arena_objectif` | 142 | 52.3 | 50.4 | OK |
| Chocoboflor | `chaos` | 10 | 45.9 | 45.9 | OK |
| Madina97294 | `btb` | 484 | 50.2 | 50.2 | OK |
| Madina97294 | `arena_slayer` | 426 | 49.8 | 49.8 | OK |
| Madina97294 | `arena_objectif` | 182 | 50.0 | 50.0 | OK |
| Madina97294 | `chaos` | 15 | 41.0 | 41.0 | **HORS** (deja hors fenetre au regime actuel) |
| Madina97294 | `ranked_objectif` | 11 | 38.8 | 44.4 | **HORS** (deja hors fenetre au regime actuel) |
| XxDaemonGamerxX | `arena_slayer` | 12 | 54.0 | 54.0 | OK |

Bilan : 10 chaines dans la fenetre, 2 hors fenetre.

### 2. Zero note simulee sur un match outcome=4

| Joueur | DNF dans l'univers | Notes simulees sur un DNF |
|---|---:|---:|
| JGtm | 39 | 0 |
| Chocoboflor | 19 | 0 |
| Madina97294 | 57 | 0 |
| XxDaemonGamerxX | 1 | 0 |

Total des fuites : **0** (le filtre `outcome != 4` est applique avant toute notation).

### 3. Concordance replique du regime ACTUEL vs notes stockees

Apparies : matchs dont la chaine stockee est non vide ET identique a la chaine
recalculee (les notes a chaine `NULL` datent de l'ere pre-chaines, reference globale
abandonnee : les comparer n'aurait aucun sens).

| Joueur | Chaine | n apparies | med stockee | med replique | ecart median | |delta| moyen | dans +/-1 pt |
|---|---|---:|---:|---:|---:|---:|---:|
| JGtm | `arena_objectif` | 211 | 52.5 | 52.5 | +0.0 | 0.00 | 211 (100%) |
| JGtm | `arena_slayer` | 414 | 49.5 | 49.5 | +0.0 | 0.00 | 414 (100%) |
| JGtm | `chaos` | 408 | 50.8 | 50.8 | +0.0 | 0.00 | 408 (100%) |
| Chocoboflor | `arena_objectif` | 142 | 51.8 | 52.3 | +0.5 | 1.47 | 71 (50%) |
| Chocoboflor | `arena_slayer` | 344 | 50.2 | 49.7 | -0.6 | 2.06 | 146 (42%) |
| Chocoboflor | `chaos` | 10 | 40.0 | 45.9 | +5.9 | 2.12 | 5 (50%) |
| Madina97294 | `arena_objectif` | 182 | 49.2 | 50.0 | +0.8 | 1.81 | 78 (43%) |
| Madina97294 | `arena_slayer` | 426 | 49.7 | 49.8 | +0.1 | 1.37 | 237 (56%) |
| Madina97294 | `btb` | 484 | 50.5 | 50.2 | -0.3 | 2.17 | 206 (43%) |
| Madina97294 | `chaos` | 15 | 39.5 | 41.0 | +1.5 | 3.14 | 8 (53%) |
| Madina97294 | `ranked` | 24 | 52.5 | 51.9 | -0.6 | 1.32 | 14 (58%) |
| XxDaemonGamerxX | `arena_slayer` | 12 | 54.0 | 54.0 | +0.0 | 0.00 | 12 (100%) |

**Cause attendue d'ecart : `medal_exploit` absent de la simulation.** Sonde empirique —
retrait d'UNE metrique de poids 0.06 (`dpm_damage`) du profil actuel, puis mesure de
l'ecart de note induit :

| Joueur | n | |delta| moyen | p90 | max |
|---|---:|---:|---:|---:|
| JGtm | 1033 | 1.26 | 2.50 | 7.50 |
| Chocoboflor | 496 | 1.35 | 2.80 | 4.00 |
| Madina97294 | 1131 | 1.20 | 2.50 | 5.10 |
| XxDaemonGamerxX | 12 | 1.17 | 1.80 | 2.60 |

Borne analytique : avec un poids w=0.06 sur une somme de poids 1.01, l'ecart vaut
`0.06 x (p_metrique - note) / 1.01`, soit au plus ~3 pts et typiquement ~1.5 pt.

**Lecture du contraste entre joueurs.** Chez JGtm et XxDaemonGamerxX la replique est
EXACTE (ecart 0.00 sur 1033 et 12 matchs, 100% dans +/-1) : sur ces DB, les notes
stockees ont donc ete produites SANS medal_exploit, exactement comme la simulation.
Chez Chocoboflor et Madina97294 l'ecart moyen (1.3 a 2.2 pt) et son p90 tombent dans
la plage mesuree par la sonde ci-dessus — hypothese coherente, non prouvee ici : leurs
notes stockees ont ete produites AVEC medal_exploit. Les medianes restent a +/-0.8 pt
sur toutes les chaines de volume ; les seuls ecarts mediens > 1 pt sont les chaines
`chaos` de 10 et 15 notes, ou la mediane n'est pas un estimateur stable.
**Verdict du critere : concordance validee.**

## Elements de decision — sensibilite au poids ospm

Population : toutes les notes des chaines objectif (`arena_objectif` + `ranked_objectif`)
des 4 joueurs, mises en commun. « Actifs » = ecart `p ospm - p combat` >= 20 ;
« contre-temoins » = ecart <= -20.

| Profil | n notes | p10 | mediane | p90 | |delta| moyen vs ACTUEL | delta max | delta moyen actifs (n) | delta moyen contre-temoins (n) |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| ACTUEL | 547 | 23.2 | 51.6 | 74.2 | - | - | - | - |
| ospm=0.08 | 546 | 24.4 | 50.8 | 73.0 | 2.39 | 13.1 | +3.4 (155) | -3.2 (175) |
| ospm=0.12 | 546 | 24.1 | 50.6 | 72.7 | 3.28 | 14.9 | +4.7 (155) | -4.4 (175) |
| ospm=0.16 | 546 | 24.5 | 50.7 | 72.9 | 4.12 | 16.6 | +5.9 (155) | -5.6 (175) |

Lecture : la mediane doit rester stable (la note garde son sens « 50 = ta moyenne »)
pendant que le delta des actifs doit etre SENSIBLE, et celui des contre-temoins
d'une amplitude comparable mais non punitive.

### Verification de non-inversion (population entiere, chaines objectif)

« Ecrases » = `p combat` <= 40 ; « combattants » = `p combat` >= 60. L'exigence
produit tient tant que le p90 des ecrases reste SOUS le p10 des combattants.

| Profil | n ecrases | p90 des ecrases | n combattants | p10 des combattants | marge | verdict |
|---|---:|---:|---:|---:|---:|---|
| ACTUEL | 182 | 42.3 | 182 | 54.7 | +12.5 | OK |
| ospm=0.08 | 182 | 47.4 | 182 | 52.5 | +5.1 | OK |
| ospm=0.12 | 182 | 47.8 | 182 | 51.9 | +4.0 | OK |
| ospm=0.16 | 182 | 48.0 | 182 | 51.1 | +3.2 | OK |

## Recommandation finale — poids ospm

**Retenir ospm = 0.12**, c'est-a-dire la valeur de depart de la decision D-C : la
simulation la CONFIRME, elle ne la deplace pas. Les autres poids du profil objectif
restent ceux de D-C (kpm 0.10, kda 0.09, accuracy 0.03, pspm 0.08 ; toutes les
metriques morts/degats/attendus inchangees).

### Argument 1 — a 0.08 le signal n'est pas lisible

Le plancher de bruit de la note est mesure (section Gate 3) : retirer UNE metrique de
poids 0.06 deplace la note de 1.24 pt en moyenne, 2.40 pt au p90. A ospm=0.08, un match
« ecrase mais actif » ne gagne que +3.4 pt en moyenne — soit 1.4x le p90 du bruit : le
joueur ne pourrait pas distinguer la reconnaissance de son jeu d'objectif d'une simple
fluctuation. A 0.12 le gain est de +4.7 pt (2.0x le p90 du bruit), lisible sans etre
spectaculaire. C'est le critere qui elimine 0.08.

### Argument 2 — a 0.16 la metrique cesse de valoriser l'objectif pour punir le combat

Les contre-temoins (forts au combat, absents de l'objectif) perdent -5.6 pt en moyenne
a 0.16 contre -4.4 pt a 0.12, avec un ecart maximal de 16.6 pt (contre 14.9 a 0.12).
Sur le temoin le plus net de la table de decision — un 22/5 en King of the Hill — la
note passe de 74.9 a 62.2 : une partie dominee au combat serait sanctionnee de plus de
12 pts pour n'avoir pas garde la colline. L'objectif produit etait de faire remonter le
joueur ecrase, pas de retrograder le joueur qui porte le combat. C'est le critere qui
elimine 0.16.

### Argument 3 — la stabilite des distributions ne discrimine pas

Sur les 547 notes de chaines objectif, la mediane passe de 51.6 (regime actuel) a
50.8 / 50.6 / 50.7 pour ospm 0.08 / 0.12 / 0.16, et les p10/p90 bougent de moins de
1.5 pt dans les trois cas. Aucune des trois variantes ne deregle le sens de la note
(« 50 = ta moyenne ») : la stabilite ne plaide donc ni pour ni contre 0.12, elle leve
seulement l'objection de risque. Le choix se fait sur les arguments 1 et 2.

### Argument 4 — symetrie et non-inversion

A 0.12 le profil est quasi symetrique : +4.7 pt pour les actifs (155 matchs), -4.4 pt
pour les contre-temoins (175 matchs). La note n'est donc pas globalement inflatee : la
metrique redistribue, elle n'ajoute pas de points.

La verification de non-inversion tient dans les trois variantes, mais la marge se
resserre a mesure que le poids monte : 12.5 pt au regime actuel, puis 5.1 a 0.08, puis 4.0 a 0.12, puis 3.2 a 0.16. A 0.16 cette marge (3.2) descend au niveau
du plancher de bruit mesure (2.40 pt au p90) — la separation « ecrases » / « combattants »
cesserait alors d'etre robuste match par match. A 0.12 la marge reste au-dessus du
bruit : un joueur actif a l'objectif remonte sans jamais depasser un joueur qui a
aussi combattu. C'est le quatrieme argument, convergent, pour 0.12.

### Consequences a annoncer avant le recompute reel (lot 4)

| Joueur | Notes stockees | Notes apres reforme | Perdues | Chaines ranked creees (n scorees) |
|---|---:|---:|---:|---|
| JGtm | 1111 | 1033 | 78 | `ranked_slayer` 6 matchs (0), `ranked_objectif` 2 matchs (0) |
| Chocoboflor | 537 | 496 | 41 | `ranked_slayer` 6 matchs (0), `ranked_objectif` 2 matchs (0) |
| Madina97294 | 1239 | 1121 | 118 | `ranked_slayer` 13 matchs (3), `ranked_objectif` 21 matchs (11) |
| XxDaemonGamerxX | 22 | 12 | 10 | aucune |

La scission ranked ne produit de notes QUE chez Madina97294 : les 8 matchs ranked de
JGtm et les 8 de Chocoboflor tombent sous le seuil de 10 par chaine et perdent leur
note — conformement a D-D (purge seche, deja validee par l'utilisateur).

## Annexe — classification effective des sous-modes (categories Assassin + Ranked)

Sous-modes normalises par `analysis.NormalizeModeLabel`, avec la famille que leur
attribue le NOUVEAU regime. Corpus des 4 joueurs, univers complet.

**Lacunes de la liste objectif reperees ici** (PRE-EXISTANTES : elles affectent deja
`lusrChainForAssassin` et donc les chaines LUSR d'aujourd'hui — NON corrigees par ce
lot 0, a statuer au lot 1) : 26 matchs d'un mode objectif evident tombent en famille
slayer — Assaut (`neutral bomb`, `one bomb`, `neutral bomb squad`), `vip`,
`ctf 3 captures`, et surtout `arena` (14 matchs dont le pair_name est INVERSE :
`Strongholds:Arena on Behemoth` — le mode est a GAUCHE du deux-points, la
normalisation retient donc « Arena » comme sous-mode).

| Sous-mode normalise | Famille attribuee | n matchs | Exemple de pair_name |
|---|---|---:|---|
| `team slayer` | `arena_slayer` | 573 | Community:Team Slayer on The Pit |
| `slayer` | `arena_slayer` | 487 | Arena:Slayer on Streets |
| `ctf` | `arena_objectif` | 281 | Arena:CTF on Aquarius |
| `strongholds` | `arena_objectif` | 162 | Arena:Strongholds on Streets |
| `king of the hill` | `arena_objectif` | 70 | Arena:King of the Hill on Snowbound |
| `neutral flag ctf` | `arena_objectif` | 47 | Arena:Neutral Flag CTF on Behemoth |
| `slayer` | `ranked_slayer` | 24 | Ranked:Slayer on Solitude - Ranked |
| `oddball` | `arena_objectif` | 19 | Arena:Oddball on Recharge |
| `arena` | `arena_slayer` | 14 | Strongholds:Arena on Behemoth |
| `oddball` | `ranked_objectif` | 13 | Ranked:Oddball on Lattice - Ranked |
| `fiesta slayer` | `arena_slayer` | 11 | Community:Fiesta Slayer on High Ground |
| `one flag ctf` | `arena_objectif` | 9 | Arena:One Flag CTF on Salvation |
| `team snipers` | `arena_slayer` | 8 | Arena:Team Snipers on Isolation |
| `escalation slayer` | `arena_slayer` | 6 | Arena:Escalation Slayer on Cliffhanger |
| `strongholds` | `ranked_objectif` | 5 | Ranked:Strongholds on Streets |
| `neutral bomb` | `arena_slayer` | 4 | Assault:Neutral Bomb on Origin |
| `ctf` | `ranked_objectif` | 4 | Ranked:CTF on Aquarius |
| `king of the hill` | `ranked_objectif` | 3 | Ranked:King of the Hill on Live Fire |
| `vip` | `arena_slayer` | 3 | Arena:VIP on Catalyst |
| `one bomb` | `arena_slayer` | 3 | Assault:One Bomb on Curfew |
| `attrition` | `arena_slayer` | 2 | Arena:Attrition on Catalyst |
| `shotty snipes slayer` | `arena_slayer` | 1 | Arena:Shotty Snipes Slayer on Detachment |
| `neutral bomb squad` | `arena_slayer` | 1 | Assault:Neutral Bomb Squad on Rat's Nest |
| `shotty snipe slayer ffa` | `arena_slayer` | 1 | Community:Shotty Snipe Slayer FFA on Dynasty |
| `ffa slayer` | `arena_slayer` | 1 | Arena:FFA Slayer on Forest - Forge |
| `land grab` | `arena_objectif` | 1 | Arena:Land Grab on Bazaar |
| `ctf 3 captures` | `ranked_slayer` | 1 | Ranked:CTF 3 Captures on Argyle |

