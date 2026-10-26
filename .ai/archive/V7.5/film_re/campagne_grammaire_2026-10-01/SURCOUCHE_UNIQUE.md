# SURCOUCHE_UNIQUE — une seule surcouche de mesure post-J12 (2026-10-02)

Ce document traite les points D16 de `CRITIQUE_COMPLETUDE_R.md` et D-100 du plan. Il applique la décision technique
D17 du superviseur : une seule surcouche de mesure, bâtie sur la tête post-J12 (`df228c24c`), qui réunit les crochets
de toutes les familles.

Conventions :
- **mesuré** : sortie d'une commande lancée ici ;
- **établi** : lu dans le code ou un fichier, puis confirmé ;
- **supposé** : non vérifié.

Cadre du travail :
- aucun fichier de production n'a été touché et aucun commit n'a été fait ;
- aucune sonde n'a été modifiée ;
- `grammar.Rev` vaut toujours `grammar-2026-09-27.3` ;
- la surcouche est un outil de MESURE seulement (D10) : aucune preuve de gate ne repose sur elle.

## 0. En bref

- La nouvelle surcouche est `surcouche_unique_postj12/` : 7 fichiers et `overlay.json`, avec les chemins de
  `LevelUp-wt-campagne-grammaire`.
- Avec elle, `go vet -tags=research,campagne_overlay` rend **rc=0** (mesuré). Il couvre `grammar`,
  `facts/killsource`, `cmd/killsource` et `research/cmd_fermeture`, soit **toutes** les sondes taggées
  `campagne_overlay` sous un même tag. Avant, cette commande était rouge avec les surcouches bis2, fusion, R-COMB,
  R-COMP et R-LOC, et verte avec R-VEH seule (§3).
- **Aucun tag propre n'a été nécessaire.** Aucune sonde n'a été retaguée et aucun symbole n'entre en collision.
- **Inertie, mesurée.** Sans bascule, `cmd_fermeture -mode v2` construit sous la surcouche unique rend les
  10 TSV de la carte v2 identiques à ceux de la production post-J12 sur les 20 films. Ils sont aussi identiques à
  la référence de phase 1, `carte_fermeture_v2_2026-10-01/`. `fermeture_films.tsv` est comparé hors `pic_octets`
  et `duree_ms`, les deux colonnes propres à la machine.
- **Chaque famille rejoue sa référence et une variante clé à l'identique de sa note** (§4). Le seul écart est
  dans BIS_2 : la colonne texte `cause_ti43_principale` de `mb2_ti43.tsv` diffère sur 4 lignes de la variante
  `reference`. Ce sont des ex aequo, départagés par l'ordre d'une table de hachage : toutes les colonnes chiffrées
  sont identiques, et `mb2_ti43_causes.tsv` aussi.
- **Changement de comportement à connaître (famille LOC).** `r_loc_overlay/` n'était PAS inerte : il remplaçait
  sans condition le localisateur de `marchLocateStrict` et de `killsource.locateStrict` par la règle R-LS. Dans la
  surcouche unique, cette règle passe derrière une bascule, la variable d'environnement `CAMPAGNE_RLOC_LS=1`. Si la
  variable est absente, on obtient la production.
- **D-102.** L'édition de `r_veh_ti40_research_test.go` faite par `python3` est jugée juste :
  - `vet` et `gofmt` sont propres ;
  - le fichier est identique à celui du worktree `LevelUp-wt-cg-veh` ;
  - la sonde rejoue `r_veh_ti40_variantes.tsv` (22 variantes, 21 films) **identique à l'octet** sous la
    surcouche unique.

  La version d'avant l'édition n'a été conservée nulle part. Le diff de l'édition elle-même ne peut donc pas
  être produit (§5).

## 1. Méthode

### 1.1 Recensement

Il y a sept fichiers de production remplacés par au moins une surcouche (établi par lecture des six
`overlay.json`) :

| Fichier (sous `film/internal/`) | bis2 | fusion | comb | comp | loc | veh |
|---|---|---|---|---|---|---|
| `grammar/capture.go` | x | x | x | x | | x |
| `grammar/lecteur_position.go` | x | x | x | x | | x |
| `grammar/lecteur_position_exceptions.go` | x | x | x | x | | x |
| `grammar/traverse.go` | | | | | | x |
| `grammar/default_state_ti40.go` | | | | | | x |
| `grammar/object_deaths_march.go` | | | | | x | |
| `facts/killsource/walk.go` | | | | | x | |

Pour les trois fichiers partagés, les copies de bis2, comb, comp et veh sont **identiques à l'octet** (`md5sum`) :
- `capture.go` : `1f6a1c53…` ;
- `lecteur_position.go` : `4732cee0…` ;
- `lecteur_position_exceptions.go` : `a69d7677…`.

Ces familles n'ajoutent donc aucun crochet propre à ces fichiers. Leurs leviers sont des variables posées par les
sondes : `bis2Intercepteur`, `bis2C3`, `bis2SitesJeu`, etc.

Les copies de `capture.go` sont toutes identiques, fusion comprise, car J12 ne touche pas ce fichier.

### 1.2 Fusion à trois voies

La fusion utilise `git merge-file -p <ours> <base> <theirs>`, avec :
- `ours` = `git show HEAD:<fichier>` (post-J12, `df228c24c`) ;
- `base` = `git show fe18bf67c:<fichier>` ;
- `theirs` = la copie de la famille.

| Fichier | theirs | rc | Résultat |
|---|---|---|---|
| `capture.go`, `lecteur_position.go`, `lecteur_position_exceptions.go` | bis2 (= comb = comp = veh) | 0 | **identique à l'octet** à `r_fusion_overlay_postj12/` : deux fusions indépendantes donnent le même fichier |
| `traverse.go`, `default_state_ti40.go` | veh | 0 | sans conflit ; les 3 boucles `for range` et la boucle `for range n` de J12 sont conservées |
| `object_deaths_march.go`, `walk.go` | loc | 0 | sans conflit, mais **non inerte** : voir 1.3 |

### 1.3 La famille LOC : de la copie de remplacement à la bascule

La copie de `r_loc_overlay/` réécrivait le corps de `marchLocateStrict` (grammar) et de `locateStrict` (killsource).
Avec elle, la marche suivait la règle R-LS sans condition. C'était voulu : `cmd_fermeture` et `cmd/killsource`
sont des binaires, sans crochet de test. Cette copie ne satisfaisait donc pas l'exigence « inerte par défaut ».

Dans la surcouche unique, chacun des deux fichiers part de la version **post-J12 de production**, à laquelle
s'ajoutent :
- `var rlocLS = os.Getenv("CAMPAGNE_RLOC_LS") == "1"`, lu au chargement du paquet ;
- en tête de la fonction de production, `if rlocLS { return rloc…LS(…) }` ;
- la règle R-LS **recopiée à l'identique** depuis `r_loc_overlay/` dans `rlocMarchLocateStrictLS` /
  `rlocLocateStrictLS`, ainsi que `rlocArchetypesHauteFrequence`.

Le corps de production n'est pas modifié (le diff ne retire aucune ligne). Si la variable est absente, le chemin
exécuté est celui de la production. Cette inertie est mesurée par la carte v2 (§4.0).

### 1.4 Fichiers livrés

Dossier `surcouche_unique_postj12/` :
- `capture.go`, `lecteur_position.go`, `lecteur_position_exceptions.go`, `traverse.go`, `default_state_ti40.go`,
  `object_deaths_march.go`, `walk.go` ;
- `overlay.json` : 7 remplacements, chemins absolus de ce worktree ;
- `delta/<fichier>.diff` : le DELTA lisible de chaque fichier contre la version post-J12 (`diff -u`).

## 2. Deltas (ajouts de mesure seulement)

| Fichier | + | − | Contenu |
|---|---|---|---|
| `capture.go` | 18 | 0 | `bis2Intercepteur` (crochet par nom de composant, consulté avant le dispatch), `bis2ComposantCourant`, `bis2TICourant` |
| `lecteur_position.go` | 46 | 0 | `bis2NoterIndex` (relevé des index de plage lus), `bis2LargeursDeLIndex` (largeurs de la ligne de l'index lu, si `bis2C3.parIndex`) ; `bis2C3 == nil` donne la production |
| `lecteur_position_exceptions.go` | 145 | 3 | Une bascule par site (13 sites, `bis2SitesJeu`) vers la lecture du jeu, plus `bis2QueuesSansCondition` et `bis2WaypointQueue`. Les 3 lignes « retirées » sont des réécritures sans effet : la signature `consumePrecHautDuBipede(_ *Lecteur)` devient `(br *Lecteur)` avec un corps vide hors bascule ; dans `consumeTacmapWaypointState`, `lireCorpsDeTraverseeAncien(br)` passe dans la branche `else` de la bascule ; un commentaire est réaligné par gofmt. |
| `traverse.go` | 14 | 0 | `rvehPorteeNeuf` : sous bascule, tout record NEW est lu avec `PorteeBaseline` et `GrammaireEcrivainI0`, puis le contexte est restauré par `defer` |
| `default_state_ti40.go` | 11 | 0 | `rvehPorteeEtat` : sous bascule, la feuille 4 de `consumeVehicleMediaFrame` est lue en `R(96)` |
| `object_deaths_march.go` | 57 | 0 | import `os`, `rlocLS`, `rlocMarchLocateStrictLS`, `rlocArchetypesHauteFrequence` (§1.3) |
| `facts/killsource/walk.go` | 50 | 0 | import `os`, `rlocLS`, `rlocLocateStrictLS`, `rlocArchetypesHauteFrequence` (§1.3) |

Bascules et valeur par défaut :

| Bascule | Défaut | Posée par |
|---|---|---|
| `bis2Intercepteur` | `nil` | sondes |
| `bis2C3` | `nil` | sondes |
| `bis2SitesJeu[*]` | `false` | sondes |
| `bis2QueuesSansCondition` | `false` | sondes |
| `bis2WaypointQueue` | `false` | sondes |
| `rvehPorteeNeuf` | `false` | sondes |
| `rvehPorteeEtat` | `false` | sondes |
| `rlocLS` | `false` | `CAMPAGNE_RLOC_LS=1` |

## 3. Compilation (mesuré, depuis `apps/go-api`)

Environnement :
- `GOCACHE` = `scratchpad/gocache-surcouche` ;
- `GOTMPDIR` dans le scratchpad ;
- `PATH` avec `C:/msys64/ucrt64/bin` ;
- une commande `go` à la fois.

| Commande | rc |
|---|---|
| `go vet -tags=research,campagne_overlay -overlay=<surcouche_unique>/overlay.json ./internal/games/halo_infinite/film/internal/grammar/ ./internal/games/halo_infinite/film/internal/facts/killsource/ ./cmd/killsource/ ./internal/games/halo_infinite/film/research/cmd_fermeture/` | **0** |
| même `go vet` sur `grammar`, sans `-overlay` | 1 (`bis2*` indéfinis : 10 erreurs) |
| idem, `-overlay=mesures_bis2_overlay/overlay.json` | 1 (`rvehPorteeNeuf`, `rvehPorteeEtat` indéfinis) |
| idem, `r_fusion_overlay_postj12/overlay_campagne.json` | 1 (même cause) |
| idem, `r_comb_overlay/overlay_campagne.json` | 1 (même cause) |
| idem, `r_comp_overlay/overlay_campagne.json` | 1 (même cause) |
| idem, `r_loc_overlay/overlay_campagne.json` | 1 (`bis2*` indéfinis) |
| idem, `r_veh_overlay/overlay_campagne.json` | 0, mais avec les versions d'avant J12 de 5 fichiers |

Ce tableau confirme D-100 et corrige le point 12 de la critique : avant ce travail, cinq surcouches sur six étaient
rouges sous le tag commun, et non trois ou quatre. La surcouche LOC l'était aussi.

`go build` réussit sous la surcouche unique pour `cmd_fermeture` (`-tags=research`) et pour `cmd/killsource`
(sans tag), ainsi que `go test -c -tags=research,campagne_overlay` pour `grammar`. Ces commandes n'ont pas été
rejouées sans surcouche, sauf pour les deux binaires de production servant à la comparaison.

## 4. Preuves sur films (mesuré)

Les films sont lus en lecture seule depuis `LevelUp/data/cache/film_chunks`, un à la fois, sous la sentinelle
`filmproc` à 4 Gio. Les sorties vont dans le scratchpad. Les sondes sont exécutées depuis le binaire
`grammar_su.test.exe` (`go test -c` sous la surcouche unique), avec le répertoire du paquet comme répertoire courant.

Corpus :
- 20 films = 19 témoins de `config/replay_corpus.toml` + `1c4c63c2` ;
- 21 films = les mêmes + `81c02726`, quand la note de la famille l'inclut.

### 4.0 Inertie : la carte v2 officielle (`cmd_fermeture -mode v2`, 20 films)

On compare trois binaires construits sur la tête :
- `cf_prod`, sans surcouche ;
- `cf_su`, sous la surcouche unique ;
- `cf_su` lancé avec `CAMPAGNE_RLOC_LS=1`.

| TSV | référence phase 1 | prod post-J12 | surcouche unique |
|---|---|---|---|
| `fermeture_archetypes` | 2dd7d464 | 2dd7d464 | 2dd7d464 |
| `fermeture_bloquants` | 0f4ef0b4 | 0f4ef0b4 | 0f4ef0b4 |
| `fermeture_borne` | 8797df8c | 8797df8c | 8797df8c |
| `fermeture_chunk3` | f024ae05 | f024ae05 | f024ae05 |
| `fermeture_entrees` | dc1404b8 | dc1404b8 | dc1404b8 |
| `fermeture_films` (hors pic et durée) | 88587ee6 | 88587ee6 | 88587ee6 |
| `fermeture_hors_cadre` | f97aa4b0 | f97aa4b0 | f97aa4b0 |
| `fermeture_hors_cadre_dernier` | 257b024b | 257b024b | 257b024b |
| `fermeture_rejets` | 64ed1c20 | 64ed1c20 | 64ed1c20 |
| `fermeture_sorties_vueB` | 52e25170 | 52e25170 | 52e25170 |

Les valeurs sont des préfixes md5. Totaux sur 20 films :
- 284 704 paquets fermés ;
- 48 720 listes non localisées ;
- 2 588 167 records utiles fermés sur 5 961 028 lus ;
- 264 757 paquets hors cadre.

La production après J12 rend la carte v2 de phase 1 à l'octet près, ce qui confirme D-F3 pour la référence. La
surcouche unique, sans bascule, rend la même carte.

### 4.1 LOC : variante clé `ls` (bascule `CAMPAGNE_RLOC_LS=1`)

`fermeture_films.tsv` est comparé hors pic et durée à `r_loc_tsv/rloc_carte_v2_surcouche_ls_films.tsv` : **identique
sur 20/20 films**. Totaux :
- 296 755 paquets fermés ;
- 26 421 listes non localisées ;
- 2 768 993 records utiles fermés sur 6 284 483 lus.

Ce sont les chiffres de `R_LOC.md` §0 point 2 : 48 720 → 26 421.

La référence de la famille est la carte de §4.0. La sonde `TestRLocLocalisateur` est taggée `research` seul et ne
dépend d'aucune surcouche : elle n'a pas été rejouée.

Pour killsource, voir §4.6.

### 4.2 BIS_2 : `TestCampagneBis2Dispositifs` (référence et grammaire `ti=43` de T7, 21 films, 3 min 42 s)

- `mb2_ti43_causes.tsv` : identique à l'octet.
- `mb2_ti43_fenetre.tsv` (`CAMPAGNE_FENETRE=81c02726:725-775`) : identique à `mb2_ti43_fenetre_81c02726_725-775s.tsv`.
- `mb2_ti43.tsv` : toutes les colonnes chiffrées sont identiques sur les 42 lignes. La colonne texte
  `cause_ti43_principale` diffère sur 4 lignes `reference` :
  - `111fa685` : i19 au lieu de i20 ;
  - `60ae07c4` : i22 au lieu de i20 ;
  - `bf15f7ab` : i19 au lieu de i21 ;
  - `d9781168` : i39 au lieu de i22.

  Dans chacun de ces cas, `mb2_ti43_causes.tsv` (identique) montre une égalité. Par exemple pour `111fa685`,
  i19 = 1 et i20 = 1 ; pour `bf15f7ab`, i19 = i20 = i21 = 2. La « principale » est donc tirée parmi des ex aequo par
  l'ordre d'itération d'une map Go. C'est un défaut de déterminisme de la sonde, pas un écart de mesure (établi).

Sur 20 films :

| Variante | Fermés | Utiles fermés | Utiles lus | Hors cadre | Gagnés | Perdus |
|---|---|---|---|---|---|---|
| `reference` | 284 704 | 2 588 167 | 5 961 028 | 264 757 | 0 | 0 |
| `grammaire-ti43-T7` (L2) | 302 797 | 2 731 481 | 5 988 139 | 259 883 | +18 105 | −12 |

Ce sont les chiffres de `MESURES_BIS_2.md` (« +18 105 / −12 »).

### 4.3 R-COMP : `TestRCompL3ImagesCles` et `TestRCompL3Delta` (20 films, 17 s et 868 s)

Les quatre TSV `r_comp_l3_images_cles.tsv`, `r_comp_l3_decalages.tsv`, `r_comp_l3_delta.tsv` et
`r_comp_l3_delta_tables.tsv` sont **identiques à l'octet** à ceux de `r_comp_tsv/`.

| Variante | Fermés | Sains | Utiles fermés | Utiles lus | Hors cadre | Gagnés | Perdus |
|---|---|---|---|---|---|---|---|
| `reference` | 284 704 | 276 316 | 2 588 167 | 5 961 028 | 264 757 | 0 | 0 |
| `moteur` (L3a) | 288 618 | 279 891 | 2 642 377 | 6 019 047 | 265 274 | 3 924 | 10 |

### 4.4 R-VEH : `TestRVehDeltaTi40` (21 films, 914 s) et `TestRVehTi40Variantes` (21 films, 15 s)

- `r_veh_delta.tsv` : **identique à l'octet** (9 variantes, 21 films).
- `r_veh_ti40_variantes.tsv` : **identique à l'octet**. `CAMPAGNE_PHYSIQUE` est reconstruit par awk depuis
  `mesures_bis2_tsv/mb2_vehi_type_physique.tsv` (châssis:type, 42 châssis).

| Variante (20 films) | Fermés | Utiles fermés | Hors cadre | Gagnés | Perdus | Gagnés contredits |
|---|---|---|---|---|---|---|
| `reference` | 284 704 | 2 588 167 | 264 757 | 0 | 0 | 0 |
| `ti40-composants` (L4a) | 286 137 | 2 615 788 | 264 988 | 1 433 | 0 | 35 |

Avec `81c02726`, on retrouve les +1 436 gagnés de `R_VEH.md` §3.1, qui compte 21 films.

### 4.5 R-COMB : `TestRComb` (24 configurations, 21 films, 2 360 s)

`r_comb_configs.tsv` est **identique à l'octet** à `r_comb_tsv/r_comb_configs.tsv`, soit 506 lignes :
24 configurations × 21 films, plus 2 lignes de contrôle en contexte des instruments.

Dans cette sonde, les deux films Live Fire (`0797ce72`, `60ae07c4`) sont joués en contexte de production.
Sa `reference` vaut donc 284 619 fermés sur 20 films. Ses deux lignes `controle:reference-instruments`
redonnent les valeurs des instruments : 284 619 − (19 124 + 13 965) + (19 205 + 13 969) = 284 704, le compte de la
carte v2.

| Configuration (20 films) | Fermés | Sains | Utiles fermés | Utiles sains | Utiles lus | Hors cadre |
|---|---|---|---|---|---|---|
| `reference` | 284 619 | 276 236 | 2 587 444 | 2 572 406 | 5 960 464 | 264 803 |
| `full` (L1+L8+L2+L9+L6a+L6b) | 366 219 | 357 291 | 3 348 115 | 3 329 546 | 6 371 533 | 180 674 |

### 4.6 LOC dans killsource : 3 films de l'enquête (`8f7f5806` Live Fire, `6b0e6f0f` Refuge, `9c0ec856` Recharge)

Trois binaires `cmd/killsource` construits sur la tête sont comparés :
- `ks_prod` : sans surcouche ;
- `ks_su` : sous la surcouche unique ;
- `ks_su` lancé avec `CAMPAGNE_RLOC_LS=1`.

La commande est `json <id> -carte <carte> -cache <LevelUp>/data/cache -catalogue <map_quant_bounds.json>`.

**Inertie : `ks_prod` et `ks_su` rendent un JSON identique à l'octet sur les 3 films.**

Voies de lecture (`"voie"` : `sequentielle` = marche, `balayage` = scan), production puis LS :

| Film | Production (marche / scan) | LS (marche / scan) |
|---|---|---|
| `8f7f5806` | 58 / 168 | 216 / 10 |
| `6b0e6f0f` | 45 / 210 | 226 / 29 |
| `9c0ec856` | 72 / 28 | 97 / 3 |

Ces comptes sont identiques aux colonnes `marche_*` / `scan_*` de `r_loc_tsv/rloc_killsource_comparaison.tsv`.

Entre production et LS, aucun champ des morts ne change hors `voie`. Les seules autres différences portent sur
`gate_par_voie` et la concordance des deux voies.

**Constaté, non instruit.** Sur `6b0e6f0f`, une alerte de santé passe de « 10 dead-state(s) à tag `jpt!` valide
portent un indice hors du roster retenu » à 42 sous LS. `R_LOC.md` §3.2 ne la mentionne pas. C'est un diagnostic,
pas une valeur publiée.

Les 25 autres films de R_LOC (28 films au total) n'ont pas été rejoués.

### 4.7 Récapitulatif

| Famille | Référence sous surcouche unique | Variante clé | Verdict |
|---|---|---|---|
| (production) | carte v2, 10 TSV, 20 films | — | identique |
| LOC | carte v2 (= production) | `ls` (carte v2 + killsource sur 3 films) | identique |
| BIS_2 | `reference` de `mb2_ti43` | `grammaire-ti43-T7` (L2) | identique, sauf une colonne texte à ex aequo (expliqué au §4.2) |
| R-COMP | `reference` de `r_comp_l3_delta` | `moteur` (L3a) et les 6 autres | identique à l'octet |
| R-VEH | `reference` de `r_veh_delta` | `ti40-composants` (L4a) et les 7 autres ; `r_veh_ti40_variantes` | identique à l'octet |
| R-COMB | `reference` et contrôle des instruments | `full` et les 22 autres | identique à l'octet |
| fusion | — | — | ses trois fichiers sont ceux de la surcouche unique (§1.2) |

**Conclusion (mesuré).** Les chiffres de BIS_2, R-COMB, R-COMP, R-VEH et R-LS avaient été mesurés sur
`fe18bf67c`, avec des surcouches d'avant J12. Rejoués ici sur la tête post-J12 sous la surcouche unique, ils sont
identiques, pour tout ce qui a été rejoué. L'écart que D-100 disait « nul : supposé » est donc mesuré nul pour ces
sondes.

## 5. D-102 : la sonde éditée par `python3` (R_VEH §8)

**Fichier.** `apps/go-api/internal/games/halo_infinite/film/internal/grammar/r_veh_ti40_research_test.go` :
462 lignes, `//go:build research && campagne_overlay` en ligne 1. L'édition a ajouté la structure de variante
`rvVariante` (l. 403-420) et ses usages.

**Relecture (établi).** Les 8 champs sont lus là où le commentaire le dit :
- `crochet` et `porte` dans `rvRecords` ;
- `portee == 2` et `i0` dans `rvContexte` ;
- `decale` dans `rvMarcher`, sous `rvPortee` ;
- `etatSansListe` dans `rvPrefixe` ;
- `mpp` dans `TestRVehTi40Variantes`, via `PoserMPP` ;
- `nom` dans la sortie.

Les 22 variantes de `rvVariantes()` sont cohérentes avec leurs noms. Les témoins décalés ont tous
`portee: 2` : `rvPortee` vaut donc vrai, et le décalage s'applique bien.

**Octets.** Pas de CR, fin de ligne finale présente. Les seuls caractères hors ASCII sont de l'UTF-8 dans des
commentaires (tirets, guillemets).

**`gofmt -l` et `go vet`.** `gofmt -l` est vide. `go vet` est vert avec `-tags=research` (le fichier est alors
exclu par son tag) et avec `-tags=research,campagne_overlay` sous la surcouche unique.

**Diff.** La version d'avant l'édition n'existe nulle part : le fichier est neuf, sans historique git, et le
worktree temporaire n'en garde pas de copie. On a vérifié deux choses à la place :
- la copie de `LevelUp-wt-cg-veh` est **identique** à celle de la campagne ;
- parmi les 24 sondes `r_*` des cinq worktrees, seules deux diffèrent de la campagne, de deux lignes chacune :
  `r_comb_research_test.go` et `r_veh_chassis_research_test.go`. Dans les deux, `profile.QuantRangeCEBiped`
  devient `profile.QuantRangeCEBiped()` : c'est l'accesseur de J12, corrigé à l'intégration.

**Mesure.** Sous la surcouche unique, `TestRVehTi40Variantes` rend `r_veh_ti40_variantes.tsv` **identique à
l'octet** à celui de la note. `TestRVehDeltaTi40` aussi (§4.4).

**Verdict : l'édition est juste.** Elle compile, elle est formatée, elle se lit comme son commentaire la décrit,
et elle rejoue ses chiffres. L'écart à la règle « pas de Python » reste consigné ; il n'appelle aucune correction
de la sonde.

**Découverte, non traitée (règle 5).** L'en-tête de `r_veh_ti40_variantes.tsv` a 13 colonnes, alors que chaque
ligne en a 15 : `rvModal(c.n2)` ajoute deux champs sans nom (valeur modale de `n2` et son effectif). Voir
`r_veh_ti40_variantes_research_test.go`, l. 79-80 pour l'en-tête et l. 130-131 pour la ligne. Le défaut est dans
l'en-tête, pas dans `rvVariante`.

## 6. Gate (mesuré, depuis `apps/go-api` de ce worktree, 2026-10-02)

| Commande | Résultat |
|---|---|
| `gofmt -l <surcouche_unique_postj12>/ internal/ cmd/` | vide |
| `go vet -tags=research ./...` (comme la CI) | rc=0 (24 s) |
| `go vet -tags=research,campagne_overlay -overlay=<surcouche_unique>/overlay.json` sur `grammar`, `facts/killsource`, `cmd/killsource`, `research/cmd_fermeture` | rc=0 |
| `go vet -tags=research -overlay=<surcouche_unique>/overlay.json` sur `grammar`, `facts/killsource`, `cmd/killsource` | rc=0 |
| `go test -count=1 ./internal/archlint/` | `ok` (32,9 s) |
| `git status --short` | 3 entrées non suivies : `CRITIQUE_COMPLETUDE_R.md` (déposée par le superviseur), ce document, `surcouche_unique_postj12/` ; un seul fichier suivi modifié, `.ai/thought_log.md` (entrée ajoutée en fin de fichier). Aucun fichier de production ni aucune sonde modifiés. |

## 7. Commande à utiliser désormais

Depuis `apps/go-api`, pour toute sonde `research && campagne_overlay` :

```
go test -tags=research,campagne_overlay \
  -overlay=C:/Users/Guillaume/Downloads/Scripts/LevelUp-wt-campagne-grammaire/.ai/V7.5/film_re/campagne_grammaire_2026-10-01/surcouche_unique_postj12/overlay.json \
  -count=1 -timeout 240m -run '<Test>' ./internal/games/halo_infinite/film/internal/grammar/
```

Pour les instruments binaires sous la règle R-LS :

```
go build -tags=research -overlay=<…/overlay.json> -o <scratchpad>/cf_su.exe ./internal/games/halo_infinite/film/research/cmd_fermeture
go build -overlay=<…/overlay.json> -o <scratchpad>/ks_su.exe ./cmd/killsource
CAMPAGNE_RLOC_LS=1 <scratchpad>/cf_su.exe -racine … -films … -sortie … -mode v2
```

Sans `CAMPAGNE_RLOC_LS`, ces binaires lisent comme la production (§4.0, §4.6).

## 8. Ce qui reste

- **Anciennes surcouches.** La surcouche unique remplace `mesures_bis2_overlay/`, `r_fusion_overlay_postj12/`,
  `r_comb_overlay/`, `r_comp_overlay/`, `r_loc_overlay/` et `r_veh_overlay/`, qui n'ont pas été supprimés. Restent
  au superviseur :
  - leur suppression ;
  - la mise à jour des en-têtes de sondes qui les citent (la commande « `-overlay=<json>` » des commentaires) ;
  - l'item `[~]` du §6.0 du plan, selon lequel « `mesures_bis2_overlay/` reste pour rejouer la phase 1 » : la
    surcouche unique rejoue déjà BIS_2 à l'identique (§4.2).
- **Sondes non rejouées.** Leurs chiffres sous la surcouche unique sont **supposés** identiques : leurs crochets
  sont les mêmes octets que ceux rejoués ici, mais ce n'est pas mesuré. Il s'agit de :
  - `TestRCombDerivation`, `TestRCompP3`, `TestRCompP3LiveFire` ;
  - `TestCampagneBis2Positions`, `TestCampagneBis2PositionsProduction`, `TestCampagneBis2ImagesClesTi40` ;
  - `TestCampagneBis3*`, `TestCampagneBis4JugePositions`, `TestRVehIndex`, `TestRVehImageCle` ;
  - killsource sur 25 des 28 films de R_LOC.
- **Déterminisme.** La colonne `cause_ti43_principale` de `TestCampagneBis2Dispositifs` est choisie parmi des
  ex aequo selon l'ordre d'une map (§4.2). Il faudra la corriger dans la sonde (tri à comparateur total) si elle
  sert un jour à un gate.
- **Alerte killsource sous LS** (`6b0e6f0f`, 10 → 42 dead-states hors roster) : non instruite (§4.6). Elle est à
  verser au dossier du lot LS (gate 3).
- **Hors CI.** Comme toutes les sondes `campagne_overlay`, rien de ceci n'est compilé par la CI (D10). Le gate de
  surcouche reste manuel : c'est la commande du §6.

## 9. Ajout R-COMB-2 (2026-10-02) : deux fichiers de plus, inertie remesurée

R-COMB-2 (`R_COMB_2.md`) a étendu la surcouche unique (décision D17 : une seule surcouche) de deux fichiers ;
`overlay.json` passe de 7 à 9 entrées. L'ancien `overlay.json` est conservé dans le scratchpad de la session
(`rcomb2/overlay_unique_avant_rcomb2.json`), il n'est pas versionné.

| Fichier | Nature | Contenu |
|---|---|---|
| `frame_infer.go` | copie de la production post-J12 (`git show HEAD:`), **une ligne changée** | `case contreditUneEntiteVivante(w, rec) && !rc2L7:` — la bascule L7 (« NEW sur slot occupé » lié au lieu d'être refusé) |
| `rcomb2_leviers.go` | fichier AJOUTÉ (absent du dépôt) | `rc2L7` (défaut : `CAMPAGNE_RCOMB2_L7=1`, sinon faux) ; un `init()` qui, si `CAMPAGNE_RCOMB2_KS` est posée, installe les leviers de composant L8, L2, L3a, L4a, L6b, WO, L6a dans les binaires (cmd/killsource) ; recopies textuelles des lecteurs des sondes (`b2vLireTi40`, `b2vLireTi43`, `b3LireBasseFrequence` : identité vérifiée par `diff` après normalisation des noms et commentaires) |

Bascules ajoutées, défaut inerte : `rc2L7 = false` ; `CAMPAGNE_RCOMB2_KS` absente.

Inertie (mesurée le 2026-10-02, surcouche à 9 fichiers, aucune variable posée) :
- `go vet -tags=research,campagne_overlay` sur `grammar`, `facts/killsource`, `cmd/killsource`, `research/cmd_fermeture` :
  rc=0 ;
- `cmd_fermeture -mode v2` (20 films) : les 10 TSV identiques à `carte_fermeture_v2_2026-10-01/`
  (`fermeture_films.tsv` hors `pic_octets` et `duree_ms`) ;
- `cmd/killsource json -carte` sur les 19 témoins de carte connue : JSON identique à l'octet à celui du binaire
  de production, 19/19 ;
- la référence de `TestRComb2` redonne les lignes `reference` de `r_comb_configs.tsv` (voir `R_COMB_2.md` §2).
