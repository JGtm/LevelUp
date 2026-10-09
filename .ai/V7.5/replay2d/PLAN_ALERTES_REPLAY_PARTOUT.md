# Plan — Le rejeu fonctionnel PARTOUT : killsource aux bonnes largeurs, bornes pour toutes les cartes

> Ecrit le 2026-08-16. Decision utilisateur : « les alertes c'est une top priorite pour le
> replay fonctionnel partout ». Deux alertes de justesse restent ouvertes apres le correctif
> `4e2084d8e` (BuildFromFilm) : le decodeur killsource, et les cartes absentes du catalogue
> de bornes. Branche `feat/v75` (worktree principal — ce lot exige `data/`), contrat
> `plan-execution`.

## Phase 0 — QUANTIFIER « partout »

- [x] 0.1 Lister les cartes du registre des matchs (via `match_registry.parquet` du
      snapshot, DuckDB EN MEMOIRE — jamais une base ouverte) absentes de
      `map_quant_bounds.json`, avec le NOMBRE DE MATCHS par carte manquante. Threshold est
      un cas connu (`06dfe6d9` non constructible).
- [x] 0.2 Publier : combien de matchs du registre sont aujourd'hui NON constructibles en
      rejeu faute de bornes, en valeur et en pourcentage.

**Gate 0 : PASSE.** Snapshot 081 (`data/titles/halo_infinite/warehouse/snapshots/
v00000000000000000081/shared/match_registry.parquet`), lu par `read_parquet` en base
memoire, catalogue lu par `read_text` + `json_keys` — aucune base ouverte. Normalisation
rejouee en SQL a l'identique de `filmdec.NormalizeMapName` (minuscules, espaces resserres,
suffixes ` - ranked` puis ` heavies`).

**1 822 matchs au registre · 92 cles de carte distinctes · 56 cartes au catalogue.**

| | matchs | part |
|---|---:|---:|
| couverts par le catalogue | 1 607 | **88,20 %** |
| **NON constructibles faute de bornes** | **215** | **11,80 %** |

36 cles distinctes hors catalogue. Aucune carte du catalogue n'est inutilisee (56/56 jouees).

| carte hors catalogue | matchs | films en cache | module (depot `.mvar`) | statut |
|---|---:|---:|---|---|
| Live Fire | 61 | 33 | `sgh_interlock` (prouve) | **BLOQUEE** : aucun tag sbsp dans le module ds (registre l.101) |
| *(map_name NULL)* | 34 | — | — | non resoluble par nom |
| Detachment | 25 | 15 | **inconnu** | module absent de l'installation ET du depot `.mvar` |
| Argyle | 21 | 9 | **inconnu** | idem |
| Ecotone | 8 | 8 | `fo11_blank` | traitable |
| Thunderhead | 8 | 6 | `fo08_wetland` | traitable |
| Insolence | 7 | 7 | `fo09_academy` | traitable |
| Solution | 7 | 6 | `fo05_desert` | traitable |
| Flood Gulch | 6 | 6 | `fo05_desert` | traitable |
| Threshold | 5 | 5 | `fo11_blank` | traitable |
| Pharaoh | 3 | 3 | `fo11_blank` | traitable |
| Credence · Disciple | 2 · 2 | 2 · 2 | `fo11_blank` | traitable |
| Merchant's Square · Urban Raid | 2 · 2 | 2 · 2 | `fo09_academy` | traitable |
| Vallaheim Firefight | 2 | 0 | `fo05_desert` | traitable (sans film de controle) |
| Nadair · Warehouse | 1 · 1 | 1 · 1 | `fo11_blank` | traitable |
| Outlook · Lattice - Ranked | 1 · 1 | 1 · 1 | `fo13_frost` | traitable |
| Dawnbreaker | 1 | 1 | `fo05_desert` | traitable |
| Ronin · Rat's Nest · Scarlett's Landing | 1 · 1 · 1 | 1 · 1 · 1 | `fo08_wetland` | traitable |
| Cole Protocol | 1 | 0 | `fo09_academy` | traitable (sans film de controle) |
| Oasis Sentry Defense · Oasis Firefight | 1 · 1 | 1 · 0 | `btb_exiled` | suffixe de variante (registre l.102) |
| Highpower Sentry Defense | 1 | 1 | `btb_highpower` | suffixe de variante (registre l.102) |
| 6 cartes nommees par UUID brut | 6 | 1 | 1 seule au depot (`944396dd` -> `fo13_frost`) | nom d'affichage jamais resolu |
| TFF \| Night Of The Undead | 1 | 0 | — | absente du depot |

## Phase 1 — MESURER l'effet des largeurs sur killsource (prealable du registre)

- [x] 1.1 Sur un film a decoupage eloigne du defaut (`07aa428d` Illusion `[18 18 17]`, ou
      `00502e52` Bazaar `[17 17 16]`) : rejouer `killsource.Decode` AVANT/APRES installation
      des largeurs de catalogue, et publier les taux d'attribution (sources resolues,
      dead-states lus, divergences) — memes denominateurs que le paquet publie deja.
- [x] 1.2 Temoin : `000d5950` (Cliffhanger) — AUCUN changement attendu.
- [x] 1.3 Si l'ecart est NUL (ancrage deja robuste) : le consigner, fermer la ligne de
      registre, et NE PAS modifier killsource (simplification seule envisageable plus
      tard). Si l'ecart est reel : phase 2.

**Gate 1 : PASSE — l'ecart est NUL sur les trois films. killsource n'est PAS modifie.**

Instrument versionne et garde par variable d'environnement :
`internal/games/halo_infinite/film/killsource/world_precision_test.go`
(`KSPREC_FILM` / `KSPREC_BOUNDS` / `KSPREC_MAP`). Un film par process, `CGO_ENABLED=0`.

### 1.1 / 1.2 — la sortie publiee, denominateurs du paquet

| film | carte | largeurs | morts publiees / couples REELS | marche appariee / population | scan | divergences | lignes changees |
|---|---|---|---:|---:|---:|---:|---:|
| `00502e52` | Bazaar | defaut `[13 13 14]` | 95 / 95 (100,00 %) | 89 / 90 (98,89 %) | 6 / 8 | 0 | — |
| | | catalogue `[17 17 16]` | 95 / 95 (100,00 %) | 89 / 90 (98,89 %) | 6 / 8 | 0 | **0 / 95** |
| | | sonde `[6 6 6]` | 95 / 95 (100,00 %) | 89 / 90 (98,89 %) | 6 / 8 | 0 | **0 / 95** |
| `07aa428d` | Illusion | defaut `[13 13 14]` | 91 / 91 (100,00 %) | 83 / 84 (98,81 %) | 7 / 7 | 1 | — |
| | | catalogue `[18 18 17]` | 91 / 91 (100,00 %) | 83 / 84 (98,81 %) | 7 / 7 | 1 | **0 / 91** |
| | | sonde `[6 6 6]` | 91 / 91 (100,00 %) | 83 / 84 (98,81 %) | 7 / 7 | 1 | **0 / 91** |
| `000d5950` | Cliffhanger (TEMOIN) | defaut = catalogue `[13 13 14]` | 93 / 93 (100,00 %) | 84 / 86 (97,67 %) | 8 / 10 | 1 | **0 / 93** |
| | | sonde `[6 6 6]` | 93 / 93 (100,00 %) | 84 / 86 (97,67 %) | 8 / 10 | 1 | **0 / 93** |

Sante identique partout (candidats 101 / 95..96 / 100, alertes vides, marge de bijection
27 / 31 / 36, `LineByLinePublishable` vrai), a une unite pres sur Illusion (candidats 96 au
defaut contre 95 au catalogue et a la sonde : un « inexplique soi » de moins, aucune ligne
publiee changee).

### La SONDE D'ABSURDE, et pourquoi un negatif sans elle ne vaudrait rien

Un instrument mal branche rend TOUJOURS zero. Une troisieme passe a `[6 6 6]` — hors de
toute entree de catalogue, 26 bits de moins par record d'objet du monde — separe les deux
lectures. Elle ne deplace rien non plus.

### Ce que le negatif NE dit PAS : la marche, elle, change

Second instrument, meme fichier : `TestKillSourceWalkArchetypes` rejoue EXACTEMENT le
parcours de `runWalk` et compte au lieu de filtrer. `object-position-component` — le seul
deser qui lise `filmdec.WorldObjectPrecision` — **est bel et bien atteint**, et les largeurs
changent massivement ce que la marche lit :

| film | largeurs | records lus | dont position d'objet du monde | dead-states lus | dead-states lus APRES un tel composant |
|---|---|---:|---:|---:|---:|
| `00502e52` Bazaar | defaut `[13 13 14]` | 262 010 | 7 360 (2,81 %) | 228 | 85 |
| | **catalogue `[17 17 16]`** | 269 192 | 14 840 (5,51 %) | **149** | **6** |
| | sonde `[6 6 6]` | 261 020 | 7 299 (2,80 %) | 198 | 54 |
| `07aa428d` Illusion | defaut `[13 13 14]` | 309 072 | 6 845 (2,21 %) | 268 | 65 |
| | **catalogue `[18 18 17]`** | 317 921 | 12 844 (4,04 %) | **217** | **14** |
| | sonde `[6 6 6]` | 308 793 | 6 827 (2,21 %) | 241 | 38 |
| `000d5950` Cliffhanger | defaut = catalogue `[13 13 14]` | 238 962 | 6 017 (2,52 %) | 144 | 8 |
| | sonde `[6 6 6]` | 236 289 | 3 239 (1,37 %) | 188 | 52 |

Aux largeurs de la CARTE, la marche lit PLUS de records et MOINS de dead-states, et surtout
les dead-states lus derriere un objet du monde s'effondrent (85 -> 6 sur Bazaar, 65 -> 14 sur
Illusion). Ce sont des dead-states FABRIQUES par le desalignement, que le filtre de
credibilite rejetait deja. Le temoin Cliffhanger le confirme en negatif : ses colonnes
defaut et catalogue sont identiques a l'unite pres, et seule la sonde d'absurde les deplace
(8 -> 52).

**Conclusion du gate 1, ecrite franchement : l'ecart NE MORD PAS.** Le desalignement existe
et il est mesure, mais il est absorbe en amont de la publication par le filtre de
credibilite (plage de slots de bipede + indices dans le roster + categorie dans l'enum),
l'appariement par couple EXACT au kill-feed, et le rattrapage du scan. La sortie de
`killsource.Decode` est donc **insensible** aux largeurs d'objet du monde sur les trois
films, y compris a `[6 6 6]`.

## Phase 2 — CORRIGER killsource (seulement si 1 le justifie)

**Phase NON OUVERTE — le gate 1 l'interdit explicitement (item 1.3).** Les quatre items sont
statues `[!]` avec cette justification unique : cabler l'entree de catalogue jusqu'a
`killsource.Decode` ajouterait un parametre, trois appelants et un garde-rail pour un effet
MESURE NUL sur la sortie. Ce serait de la complexite sans contrepartie.

- [!] 2.1 `killsource.Options` recoit l'entree de catalogue — non traite (gate 1).
- [!] 2.2 Appelants `replaybuild` / `killcollector` / `cmd/killsource` — non traites (gate 1).
- [!] 2.3 Balayage de calibration transforme en controle logge — non traite (gate 1).
- [!] 2.4 Garde-rail du branchement — non traite (gate 1) : il n'y a pas de branchement a garder.

Ce que la mesure laisse ouvert, et qui va au registre plutot que d'etre traite ici : la
marche fabrique des dead-states sous les mauvaises largeurs (85 sur Bazaar, 65 sur Illusion,
52 sur Cliffhanger a `[6 6 6]`). Ils sont TOUS rejetes aujourd'hui. Le jour ou le filtre de
credibilite serait relache — plage de slots elargie, roster incomplet, categorie non
contrainte — cette marge disparaitrait. C'est une condition de reprise, pas une dette.

## Phase 3 — ETENDRE le catalogue de bornes

- [x] 3.1 Pour chaque carte manquante INSTALLEE localement : produire l'entree via
      `cmd/mapquant-build` (offline-pur, fichiers du jeu). Threshold en premier.
- [x] 3.2 Controle par carte ajoutee : `DetectI0Layout` d'un film de cette carte EGALE les
      largeurs deduites (le controle 7/7 du lot precedent, etendu).
- [x] 3.3 Les cartes NON installees restent hors catalogue : les compter, les nommer au
      registre (condition de reprise : installation du contenu).
- [!] 3.4 Re-cuire les artefacts temoins des cartes debloquees (`06dfe6d9` en premier) en
      commande bloquante. **`06dfe6d9` N'EST PAS DEBLOQUEE** — Threshold est sur le canevas
      `fo11_blank`, dont le controle 3.2 REFUSE les bornes. Temoin de remplacement :
      `11de8353` (Thunderhead, `fo08_wetland`, controle ACCORD).

**Gate 3 : PASSE, mais BIEN PLUS SEVERE que prevu — 8 cartes ajoutees sur 22 prouvees, et le
controle a decouvert un defaut PREEXISTANT de 276 matchs.**

### 3.1 — la preuve du module : 22 cartes, unicite 1/1 chacune

Methode doctrinale de `cmd/mapquant-build` : `level_id` lu dans `<carte>_map.mvar` (variante
par defaut), corrobore par le fichier-lien `<carte>_<module>.mvar`, unicite exigee sur les 64
modules de `any/levels` + `ds/levels`. Rejouee en continu par
`internal/himap/sonde_levelid_gamefiles_test.go` (`TestPreuveLevelIDCartes`), qui passe de 44 a
**66 sous-tests, tous verts**, sur 32 level_id distincts.

| canevas | level_id | cartes prouvees |
|---|---|---|
| `fo11_blank` | 426470249 (0x196B6B69) | Ecotone, Threshold, Pharaoh, Credence, Disciple, Nadair, Warehouse |
| `fo09_academy` | 1437677928 (0x55B13968) | Insolence, Merchant's Square, Urban Raid, Cole Protocol |
| `fo05_desert` | 1804860316 (0x6B93FB9C) | Solution, Flood Gulch, Dawnbreaker, Vallaheim Firefight |
| `fo08_wetland` | 88891201 (0x054C5F41) | Thunderhead, Ronin, Rat's Nest, Scarlett's Landing |
| `fo13_frost` | -992358985 (0xC4D9CDB7) | Outlook, Lattice - Ranked, 944396dd-5661-4a16-b1d8-a6053f762c55 |

### 3.2 — le controle `DetectI0Layout` : il REFUSE 14 cartes sur 22

19 films (un par carte disposant d'un film en cache), instrument existant
`filmdec.TestWorldObjectPrecisionLayout`. Le verdict est SYSTEMATIQUE PAR CANEVAS, jamais par
carte — ce qui est attendu, puisque les bornes sont celles du MODULE :

| canevas | largeurs deduites des bornes | decoupage lu dans les films | verdict |
|---|---|---|---|
| `fo08_wetland` | `[15 15 17]` | `[15 15 17]` | **ACCORD 4/4** |
| `fo09_academy` | `[15 15 17]` | `[15 15 17]` | **ACCORD 3/3** |
| `fo05_desert` | `[18 18 18]` | `[15 15 17]` | **DESACCORD 3/3** |
| `fo11_blank` | `[18 18 18]` | `[15 15 17]` | **DESACCORD 7/7** |
| `fo13_frost` | `[18 18 18]` | `[15 15 17]` | **DESACCORD 2/2** |

**Les 8 qui entrent** (toutes ACCORD ; Cole Protocol n'a aucun film et entre sur le controle
de son CANEVAS, 3/3 — les bornes sont celles du module, pas de la carte) : Thunderhead, Ronin,
Rat's Nest, Scarlett's Landing (`fo08_wetland`) · Insolence, Merchant's Square, Urban Raid,
Cole Protocol (`fo09_academy`).

**Les 14 qui n'entrent pas** : les 7 de `fo11_blank` (dont **Threshold**), les 4 de
`fo05_desert`, les 3 de `fo13_frost`. Leur module est PROUVE, leurs bornes sont FAUSSES —
exactement le cas de figure de Live Fire, deja au registre. Elles restent dans
`preuvesLevelID` (la preuve du module vaut) et hors de `mapModule` (le catalogue n'accepte pas
de bornes refutees).

### LA DECOUVERTE QUI DEBORDE LE LOT : 18 entrees DEJA CATALOGUEES ont ces memes bornes

Le desaccord n'est pas propre aux 14 nouvelles : il porte sur le MODULE. Or 18 cartes deja au
catalogue vivent sur ces trois canevas, et pesent **276 matchs — plus que les 215 sans bornes
du tout** :

| canevas | cartes deja cataloguees | matchs |
|---|---|---:|
| `fo11_blank` | Corpo, Critical Dewpoint, Curfew, Elevation, Empyrean, Goliath, Opulence, Salvation, Shogun, Solitude, Takamanohara | 155 |
| `fo05_desert` | Banished Narrows, Cliffside, Domicile, Fortitude, Kaiketsu, Shiro, Sylvanus | 98 |
| `fo13_frost` | Snowbound | 23 |

Le tag sbsp le plus gros de ces modules couvre ~3 870 unites la ou le film quantifie ~550 :
ce n'est pas le BSP contre lequel le jeu quantifie. **Ceci EXPLIQUE le report « ecart vertical
de ~1 270 m » du 2026-08-14** (registre l.120) : ses cartes touchees sont Solitude, Cliffside,
Snowbound (x2), Opulence, Shogun, Curfew, Domicile, Kaiketsu — **toutes sur ces trois
canevas** — et ses cartes saines sont Forest, Fortress, Houseki, Prism, Vagabond, Streets,
Recharge, dont les trois Forge sont sur `fo08_wetland` / `fo09_academy`, les deux canevas
ACCORD. La correlation est complete, et le diagnostic descend d'un cran : ce sont bien les
BORNES, et leur signature est la LARGEUR D'AXE.

**NON CORRIGE ICI** (regle « zero fix hors perimetre ») : porte au registre avec sa mesure et
sa condition de reprise. Retirer ces 18 entrees ferait perdre 276 rejeux aujourd'hui servis
(faux, mais servis) — c'est une decision produit, pas un correctif de lot.

### 3.3 — la couverture, avant et apres

| | avant | apres |
|---|---:|---:|
| cartes au catalogue | 56 | **64** |
| matchs couverts | 1 607 (88,20 %) | **1 630 (89,46 %)** |
| matchs sans bornes | 215 (11,80 %) | **192 (10,54 %)** |
| cles de carte hors catalogue | 36 | 28 |

Gain : **+23 matchs**. Diff du catalogue : **8 ajouts, 0 suppression, 0 entree preexistante
modifiee** (verifie par comparaison JSON entree par entree).

Ce qui reste hors catalogue, par motif :

| motif | cles | matchs |
|---|---:|---:|
| bornes REFUTEES par le controle 3.2 (`fo11_blank`, `fo05_desert`, `fo13_frost`) | 14 | 55 |
| Live Fire — module prouve, aucun tag sbsp dans le module ds (registre l.101) | 1 | 61 |
| `map_name` NULL au registre — rien a resoudre | 1 | 34 |
| Detachment, Argyle — module INCONNU (ni installe, ni au depot `.mvar`) | 2 | 46 |
| noms d'asset jamais resolus (5 UUID + « TFF \| Night Of The Undead ») | 6 | 6 |
| suffixes de variante (Oasis Sentry Defense / Firefight, Highpower Sentry Defense) | 3 | 3 |
| Insolence Heavies — normalise en `insolence`, desormais COUVERTE | — | — |

### 3.4 — l'artefact temoin

**`06dfe6d9` n'est pas debloquee** et le refus est VERIFIE, pas suppose :
`replay-build --map Threshold 06dfe6d9-...` sort en erreur
`replaybuild: carte hors catalogue de bornes (candidats: [Threshold])`. C'est le comportement
voulu — pas de coordonnee devinee.

Temoin de remplacement : `11de8353` (Thunderhead, `fo08_wetland`, controle ACCORD), cuit en
commande bloquante. Resultat au journal du thought_log.

## Gates

Go complets + `golangci-lint --new-from-merge-base` 0 issue ; chiffres avec denominateurs a
chaque phase ; thought_log ; registre (lignes killsource et Threshold mises a jour) ;
commits sur `feat/v75`, pas de push. Le re-build de MASSE des ~949 artefacts reste un lot
OPS a fenetre utilisateur (serveur arrete) — hors perimetre ici, deja au registre.

## Regles dures

AUCUNE base DuckDB en ecriture (serveur utilisateur) ; un decodage filmdec par process ;
JAMAIS de pause d'attente passive ; zero fix hors perimetre.

## Decouvertes (portees au registre, NON traitees)

1. **`Detachment` (25 matchs) et `Argyle` (21) n'ont pas de module connu** — ce ne sont pas
   des cartes Forge, et pourtant ni l'installation locale (31 dossiers dans
   `deploy/{pc,any,ds}/levels/multi`) ni le depot `.mvar` ne portent leur module. Ensemble
   elles pesent 46 matchs, soit plus que toutes les cartes Forge manquantes reunies.
2. **La marche de killsource fabrique des dead-states sous de mauvaises largeurs** (85 sur
   Bazaar, 65 sur Illusion, 52 sur Cliffhanger a `[6 6 6]`), tous rejetes aujourd'hui par le
   filtre de credibilite. Marge invisible tant que le filtre reste ce qu'il est.
3. **34 matchs du registre n'ont AUCUN `map_name`** — hors du perimetre des bornes, mais
   c'est le deuxieme poste de non-constructibilite apres Live Fire.
4. **Les bornes de `fo05_desert`, `fo11_blank` et `fo13_frost` sont FAUSSES au catalogue** —
   18 entrees, 276 matchs, servis aujourd'hui avec une echelle et une origine erronees. Plus
   gros que le probleme que ce plan devait resoudre. Diagnostic et correlation avec le report
   « ecart vertical ~1 270 m » au registre ; NON traite ici.

## Journal d'execution

- **2026-08-16, phases 0 a 2** — gate 0 passe (215 matchs / 11,80 % non constructibles) ;
  gate 1 passe avec un NEGATIF chiffre (ecart nul sur 3 films, sonde d'absurde comprise) ;
  phase 2 NON OUVERTE par decision du gate 1, ses 4 items statues `[!]`. Aucune ligne de
  production modifiee : le lot n'ajoute que l'instrument de mesure. Commit `efec87d71`.
- **2026-08-16, phase 3** — gate 3 passe, mais le controle 3.2 a REFUSE 14 des 22 cartes
  prouvees et decouvert un defaut PREEXISTANT de 276 matchs (bornes des canevas
  `fo05_desert` / `fo11_blank` / `fo13_frost`). Catalogue 56 -> 64 entrees, couverture
  88,20 % -> 89,46 %, diff en AJOUT PUR. Item 3.4 statue `[!]` : `06dfe6d9` n'est pas
  debloquee (Threshold est sur `fo11_blank`), temoin de remplacement `11de8353`
  (Thunderhead). Trois lignes ajoutees au registre des reports.
