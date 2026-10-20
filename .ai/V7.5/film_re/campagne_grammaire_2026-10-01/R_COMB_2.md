# R-COMB-2 : tous les leviers mesurés de la campagne, ensemble, sur la surcouche unique post-J12 (2026-10-02)

> Recherche R-COMB-2 du PLAN §6.0 (« Mesures ouvertes ») ; traite les points A4, B5, B6, B7, B8, B9, C10,
> C11, E21 et E22 de `CRITIQUE_COMPLETUDE_R.md`. Worktree `LevelUp-wt-campagne-grammaire`, branche
> `feat/campagne-grammaire`, tête `df228c24c`. Rien n'est commité. Aucun fichier de production modifié :
> trois sondes neuves `r_comb2_*_research_test.go`, deux fichiers AJOUTÉS à la surcouche unique (§1.2), des
> TSV. `grammar.Rev` reste `grammar-2026-09-27.3`. Aucune base, aucune cuisson, aucun backfill. Films lus en
> lecture seule, un à la fois par processus, sentinelle `filmproc` à 4 Gio (pics de 110 à 425 Mio).
>
> Conventions :
> - **mesuré** : compté par une sonde ou un binaire sur les films ;
> - **établi** : lu dans le code (ou Ghidra) et confirmé par la mesure ;
> - **estimé** / **supposé** : dit comme tel ;
> - **sain** : paquet fermé que le juge des trois invariants de l'écrivain (`cmContredit`, décision D2) ne
>   contredit pas ; **brut** : tous les paquets fermés, factices compris. « Sain » ne veut pas dire « juste ».
> - Indicateur D1 : records utiles fermés dans des paquets SAINS, sur le dénominateur FIXE consolidé (§3).

## 0. Réponse

### 0.1 Ce qui a été mesuré

Douze leviers, chacun la même copie de recherche que sa mesure d'origine (contrôles §2) :
L8, L2, LM, L3a, L6a, L6b, L4a (composants), LS, L1a (forme causale), LP, L7 (marche), L9 (oracle). Deux
combinaisons complètes :
- **F12** = les douze leviers ;
- **C11** = F12 sans L7. **L7 est rejeté par la mesure** (§5.3) : seul, il fait baisser 9 films sur 20 ;
  dans F12, sa marginale vaut −58 719 paquets sains et `f75e7053` baisse de −2 403 sains.

Pour chaque combinaison : la combinaison privée d'UN levier à la fois (contribution marginale saine), le
gate 2 par film, et le gate 3 killsource pour chaque levier mesurable dans le binaire (§6).

### 0.2 L'indicateur par build (C11, mesuré, `r_comb2_par_build_C11.tsv`)

Dénominateur fixe consolidé recalculé (§3). « Sans oracle » = C11 privée de l'oracle L9.

| Build | Fixe | Référence | **C11** | C11 sans L9 (oracle) | C11 brut | C11 variable | F12 (avec L7) |
|---|---|---|---|---|---|---|---|
| HI_1_13_0 (10 films) | 3 073 267 | 65,4 % | **93,8 %** | 92,3 % | 93,8 % | 94,0 % | 81,3 % |
| HI_1_12_0 | 148 160 | 23,7 % | **89,7 %** | 89,7 % | 89,7 % | 89,7 % | 68,3 % |
| HI_1_11_0 | 383 476 | 20,7 % | **90,6 %** | 89,9 % | 90,8 % | 90,6 % | 81,4 % |
| HI_1_10_0 (3 films) | 2 423 551 | 12,2 % | **81,5 %** | 76,8 % | 82,3 % | 81,5 % | 66,0 % |
| HI_1_9_0 | 324 613 | 20,1 % | **89,4 %** | 89,1 % | 89,4 % | 89,4 % | 76,6 % |
| HI_1_8_0 (Live Fire, contexte de production) | 359 291 | 23,1 % | **95,2 %** | 94,6 % | 95,2 % | 95,2 % | 84,1 % |
| HI_1_4_1, version-33, version-31 | | ≤ 0,7 % | ≤ 0,7 % | ≤ 0,7 % | | | ≤ 0,7 % |
| **corpus (20 films)** | 7 758 290 | 33,2 % | **77,0 %** | 74,9 % | 77,3 % | 77,3 % | 65,3 % |
| hors corpus `81c02726` | 161 335 | 76,6 % | 98,5 % | 98,5 % | 98,5 % | 98,5 % | 98,5 % |

En paquets (corpus) : référence 284 619 fermés dont 276 236 sains ; C11 497 347 fermés dont **491 567 sains
(+215 331)** ; 216 039 paquets gagnés, 3 311 perdus bruts, dont 1 337 sains en référence (444 deviennent
contredits, 893 deviennent non fermés).

**Lecture (mesuré, avec trois réserves).** Un seul build atteint 95 % : HI_1_8_0 (un film, `60ae07c4`, en
contexte de production). HI_1_13_0 est à 93,8 %. Les réserves, qui jouent toutes dans le même sens (vers le
haut) :
1. **L9 est un oracle** : 1,5 point sur HI_1_13_0, 2,1 points sur le corpus.
2. **L1a choisit son début de liste avec le juge** : `cmTeteInv` ne garde un candidat que si le paquet décodé
   depuis lui « ne contredit aucun invariant » (`campagne_bis1_research_test.go:159-178`). Ses paquets sont
   donc sains en partie par construction. Un L1a de production exige les invariants en code (L0). Établi par
   lecture ; l'effet sur le chiffre n'est pas mesuré.
3. **Le juge n'a que trois invariants** : l'invariant L0.6 (« sortie de vue B par rejet ⇒ non fermé ») n'y est
   pas ; 1 008 faux « sains » sont connus en référence (R-L1 (b)).

Et une réserve de dénominateur : le fixe est le maximum des marches mesurées, et C11 ou une configuration voisine en est la plus longue
sur 17 films sur 21 (§3). Le pourcentage est donc proche de la part fermée de ce que C11 lit, ce qui le fait
monter mécaniquement.

**Phrase « la phase 2 seule n'atteindra pas 95 % »** :
- **HI_1_13_0** : sous C11, 93,8 % (92,3 % sans l'oracle L9). La phrase tient de justesse en mesuré, mais pas
  avec la marge de R-COMB : l'estimation « ≤ 90 % » de `R_COMB.md` §7 est **réfutée** (§8).
- **HI_1_8_0** : la phrase est **fausse en mesuré** (95,2 %, un film), sous les trois réserves.
- **HI_1_12_0, HI_1_11_0, HI_1_9_0** : 89 à 91 %.
- **HI_1_10_0** : 81,5 %.
- **Builds anciens** : ≤ 0,7 %.

### 0.3 Contributions marginales SAINES (C11, corpus, `r_comb2_marginaux_C11.tsv`)

Marginal = C11 − (C11 privée du levier). Unités : paquets sains / records utiles sains / points de
l'indicateur sur le fixe du corpus.

| Levier | Seul | Marginal dans C11 | Points | Gate 2 seul | Gate 2 marginal | Gate 3 killsource |
|---|---|---|---|---|---|---|
| **LM** | +78 445 / +1 734 132 | **+109 844 / +2 379 937** | 30,68 | tenu | tenu | non mesurable dans le binaire ; nul par construction pour un lot limité au profil (§6.3) |
| **L8** | +30 596 / +234 374 | **+37 199 / +296 293** | 3,82 | **1 film** (`1c4c63c2` : −1 paquet, requalifié contredit, 0 utile) | **1 film** (`1c4c63c2` −2 / −48) | nul (morts) |
| **LS** | +18 087 / +169 449 | **+34 222 / +443 890** | 5,72 | tenu | tenu | **229 morts du scan à la marche, aucune valeur changée** |
| **L1a** (causal) | +13 374 / +141 226 | **+29 594 / +432 077** | 5,57 | tenu | tenu | non applicable (killsource n'appelle pas `debutDeLaListe`) |
| **L2** | +17 598 / +142 399 | +19 657 / +170 094 | 2,19 | **2 films** (`084a804d` −1 / −105 ; `1c4c63c2` −28 / −2) | **1 film** (`1c4c63c2` −913 / −16 078) | nul (morts) |
| **L3a** | +3 575 / +53 521 | +12 000 / +215 560 | 2,78 | tenu (48 sains requalifiés) | tenu | nul (morts) |
| **L6a** (Live Fire) | +5 070 / +38 063 | +8 261 / +65 783 | 0,85 | tenu | tenu | nul (morts) |
| **L9** (oracle) | +1 774 / +45 226 | +6 429 / +164 666 | 2,12 | tenu | tenu | non mesurable (oracle, §6.3) |
| **L4a** | +1 403 / +27 364 | +4 440 / +115 847 | 1,49 | tenu | **1 film** (`d9781168` 0 / −11 utiles) | nul (morts) |
| **L6b** | +460 / +5 144 | +1 751 / +25 948 | 0,33 | **3 films** (`60ae07c4` 0 / −7, `51ebbc0f` 0 / −4, `1c4c63c2` −1 / 0) | **1 film** (`1c4c63c2` 0 / −23) | 8 morts du scan à la marche, aucune valeur |
| **LP** | +694 / +7 800 | +1 281 / +23 474 | 0,30 | tenu | tenu | non applicable (killsource ne lit pas le bloc de type 1) |
| L7 | −2 688 / −45 960 | rejeté (F12 : −58 719 / −908 824) | — | **9 films** | — | nul (JSON identique sur 19/19) |

La combinaison C11 entière : **aucun film en baisse** (paquets sains et utiles sains, 21/21,
`r_comb2_par_film_C11.tsv`). Les leviers ne s'additionnent pas, et cette fois en positif : somme des seuls
+171 076 sains, combinaison +215 331. LM, L1a, LS, L3a et L9 gagnent plus en combinaison que seuls.

### 0.4 Ordre de vague proposé, par contribution SAINE marginale (§9)

- Vague 1 (composants) : **LM** (sous D6), **L8**, **L2** (gate à réparer sur `1c4c63c2`), **L3a**, **L6a**
  (Live Fire), **L9**, **L4a**, **L6b** (à réparer ou à réduire).
- Vague 2 (marche) : **LU**, puis **LS**, **L1a** (causale, invariants en code), **LP**.
- Rejetés : **L7**, et **world-object-i0** comme extension globale de L6a.

---

## 1. Protocole

### 1.1 Les leviers

| Levier | Copie de recherche | Mesure d'origine (reproduite §2) |
|---|---|---|
| L8 | `b3CrochetTi3(true, true)` : `ti=3 i0` porté, `ti=3 i1` sur 26 bits | BIS_3 §6, R-COMB |
| L2 | `b2vLireTi43` | BIS_2 §5.4, R-COMB |
| LM | `cfg.Profil.MPP = 8/3`, flux delta, **films de format 24 ou 25 seulement** (`FilmFormatVersion`) | R_VEH §3.3 (`mpp8/3`) |
| L3a | `rl3Crochet` variante `moteur` (i11, i13 à i17) | R_COMP §2.3 (`moteur`) |
| L6a | `bis2C3.parIndex` + bornes des plages 0, 2, 3 de Live Fire | BIS_2 §3.3, R-COMB |
| L6b | sites `flock-position` et `tacmap-displayasset` au jeu | BIS_2 §3.1, R-COMB |
| L4a | `b2vLireTi40` + porte `+0x818` posée | R_VEH §3.1 (`ti40-composants`) |
| LS | `rlocLocaliser("ls2")` : 123 strict → fermeture par NEW de tête → signature high-frequency (ordre de la cuisson) | R_LOC §3.1 (`ls2`) |
| L1a | `cmTeteInv(…, bloc=true, paquet=false)` sous condition **causale EN LIGNE** : score d'allocateur cumulé des chunks ANTÉRIEURS de la marche elle-même (≥ 30 NEW, ≥ 50 %) | R_NAIS §3 |
| L1a-ref (contrôle) | même localisateur, condition calculée sur la marche de RÉFÉRENCE (forme `|cumul` de R_NAIS) | R_NAIS §3 (`tete-bloc+inv|cumul`) |
| LP | désaveu à l'ouverture du chunk de la liaison d'image-clé d'un slot que le bloc de type 1 du même chunk dit non vivant | R_COMP §4.4 |
| L7 | NEW sur slot occupé LIÉ au lieu d'être refusé (bascule `rc2L7` dans `frame_infer.go`, §1.2) : la règle exacte, et non une approximation | neuf (borne « ≤ 137 » de MESURES_CIBLEES T1-4) |
| L9 | oracle P1-pont, DÉRIVÉ de la marche qui porte tous les autres leviers de la configuration (`b3Diag.imageCle`) | BIS_3 §2, R-COMB |

Variantes de vérification : WO (`world-object-i0` au jeu), L6b-flock (`flock-position` seul), L8+LS, LM+L6a,
L6a+WO, C11+WO.

**Composition des localisateurs LS × L1a** (un choix, écrit dans `r_comb2_research_test.go`, `tete`) : quand
L1a est actif sur le chunk, sa décision passe d'abord (123 strict, chaîne ou fermeture élargie). S'il ne trouve
rien, on retombe sur la signature high-frequency de LS, avec la chaîne de tête. Sinon, chacun seul est sa
copie d'origine.

**Contexte** : 19 films en contexte des instruments (`cmOuvrir`, celui de la carte v2), et les deux films de
Live Fire (`0797ce72`, `60ae07c4`) en contexte de production (`b2pOuvrirProduction`), requis par L6a. La
référence des instruments y est rejouée comme contrôle.

### 1.2 Surcouche : la surcouche unique, étendue de deux fichiers

Une seule surcouche (D17) : `surcouche_unique_postj12/overlay.json`, qui passe de 7 à 9 entrées
(`SURCOUCHE_UNIQUE.md` §9) :
- `frame_infer.go` : copie de production, une ligne changée, `case contreditUneEntiteVivante(w, rec) && !rc2L7:` ;
- `rcomb2_leviers.go`, fichier ajouté :
  - `rc2L7` ;
  - un `init()` qui installe les leviers de composant dans `cmd/killsource` si `CAMPAGNE_RCOMB2_KS` est posée ;
  - des recopies textuelles des lecteurs des sondes (identité vérifiée par `diff` : `b2vLireTi40` 58 lignes,
    `b2vLireTi43` 87, `b3LireBasseFrequence` 20).

Inertie remesurée, sans variable posée :
- carte v2 (`cmd_fermeture -mode v2`, 20 films) : 10 TSV identiques à `carte_fermeture_v2_2026-10-01/` ;
- killsource : JSON identique à l'octet au binaire de production, 19/19 films ;
- `TestRComb2` : sa référence redonne les lignes de R-COMB (§2).

### 1.3 Configurations et ordre de dérivation

`TestRComb2` joue, par film, les configurations suivantes :
- la référence (avec le diagnostic qui dérive l'oracle L9) ;
- la combinaison (base sans L9 + diagnostic, puis la même avec l'oracle L9 dérivé de cette base) ;
- chaque levier seul ;
- les variantes ;
- pour chaque levier X ≠ L9 : `full-X-L9` (base, avec diagnostic) puis `full-X`.

Une configuration dont les leviers effectifs coïncident avec une configuration déjà jouée est recopiée et
marquée `non (identique par construction à …)` : LM hors des formats 24-25, L6a hors Live Fire.

Deux passes, sur la même sonde :
- **passe 1** : F12 (`CAMPAGNE_RCOMB2_RETIRES` vide), 45 ou 46 configurations par film ;
- **passe 2** : C11 (`CAMPAGNE_RCOMB2_RETIRES=L7`), 27 ou 28 configurations.

Contrôle de déterminisme : `full`, `full-L9` et `reference` de la passe 2 égalent `full-L7`, `full-L7-L9` et
`reference` de la passe 1 sur les 21 films (63 lignes, 0 écart).

### 1.4 Sondes et commandes

| Fichier (`apps/go-api/internal/games/halo_infinite/film/internal/grammar/`) | Lignes | Tags | Rôle |
|---|---|---|---|
| `r_comb2_research_test.go` | 348 | `research && campagne_overlay` | leviers, écouteur (juge, score L1a, désaveu LP), localisateurs LS × L1a |
| `r_comb2_configs_research_test.go` | 326 | idem | `TestRComb2` : configurations, comparaisons paquet par paquet, TSV |
| `r_comb2_signatures_research_test.go` | 98 | idem | `TestRComb2Signatures` : archétype du record de chaque signature (point 3) |

Commandes, depuis `apps/go-api`, avec `GOCACHE` et `GOTMPDIR` dans le scratchpad, `PATH` avec
`C:/msys64/ucrt64/bin` et `-count=1`. Une seule commande `go` à la fois ; le binaire de test compilé une fois
(`go test -c`) tourne en trois processus sur trois lots de films :

```
go test -c -tags=research,campagne_overlay -overlay=<surcouche_unique_postj12/overlay.json> -o grammar_rc2.test.exe ./internal/games/halo_infinite/film/internal/grammar/
# depuis le dossier du paquet :
CAMPAGNE_RACINE=<LevelUp>/data/cache/film_chunks CAMPAGNE_FILMS=<lot> CAMPAGNE_SORTIE=<scratchpad> \
CAMPAGNE_CATALOGUE=<LevelUp>/data/titles/halo_infinite/reference/map_quant_bounds.json \
CAMPAGNE_CARTES="0797ce72=Live Fire;60ae07c4=Live Fire - Ranked" CAMPAGNE_BORNES_sgh_interlock="<plages 0,2,3>" \
[CAMPAGNE_RCOMB2_RETIRES=L7] grammar_rc2.test.exe -test.run '^TestRComb2$' -test.timeout 600m
# killsource : go build -overlay=<…> ./cmd/killsource, puis par variante :
[CAMPAGNE_RCOMB2_KS=L8,L2,…] [CAMPAGNE_RLOC_LS=1] [CAMPAGNE_RCOMB2_L7=1] [CAMPAGNE_RCOMB2_KS_BORNES=<plages>] \
  ks_su.exe json <film> -carte <carte> -cache <LevelUp>/data/cache -catalogue <map_quant_bounds.json>
```

`TestRComb2` refuse de tourner si une variable de binaire (`CAMPAGNE_RLOC_LS`, `CAMPAGNE_RCOMB2_KS`,
`CAMPAGNE_RCOMB2_L7`) est posée : sa référence ne serait plus la production.

Durées : passe 1, 43 min de mur (`1c4c63c2` 29 min 15 s) ; passe 2, 20 min ; signatures, 7 min ; killsource,
26 variantes de leviers + production + surcouche inerte, × 19 films, en 1 h 20 (16 h 13 → 17 h 33).

Les scripts d'agrégation sont dans `r_comb2_tsv/` : `agreger.awk`, `marginaux.awk`, `denominateur.sh`,
`ks_jouer.sh`, `ks_comparer.sh`. Ils sont en awk et bash, avec jq pour le JSON.

## 2. Contrôles (mesurés)

- **Référence = R-COMB** : les 21 lignes `reference` et les 2 lignes `controle:reference-instruments` égalent
  `r_comb_configs.tsv`. Il y a 0 écart sur fermés, sains, utiles fermés, utiles sains, utiles lus et hors
  cadre. Donc 284 619 fermés en contexte mixte, et 284 704 en contexte des instruments (carte v2).
- **Chaque levier seul redonne sa mesure d'origine**, film par film, en paquets fermés et sains :

| Levier | Source | Films identiques | Écarts |
|---|---|---|---|
| L8, L2, L9, L6a, L6b | `r_comb_configs.tsv` | 21/21 | — |
| L3a | `r_comp_l3_delta.tsv` (`moteur`) | 18/20 | les deux Live Fire (contexte de production ici, instruments là) |
| L4a | `r_veh_delta.tsv` (`ti40-composants`) | 19/21 | idem |
| LM | `r_veh_delta.tsv` (`mpp8/3`) | 5/5 films des formats 24-25 en contexte des instruments | `60ae07c4` 32 636 contre 32 635 (contexte) ; sur les autres formats, R_VEH posait 8/3 partout (témoin), la copie ici ne le pose pas |
| LS | `rloc_variantes_passe2_ls2.tsv` (`ls2`) | 18/20 | les deux Live Fire |
| L1a-ref | `r_nais_variantes.tsv` (`tete-bloc+inv|cumul`) | 18/20 | les deux Live Fire |
| LP | `r_comp_p3.tsv` (`désaveu image-clé hors bloc`) | 18/20 | les deux Live Fire |
| WO, L6b-flock | `mb2_positions.tsv` | 19/20, 18/20 | Live Fire |

- **L1a causal en ligne = L1a-ref** sur le corpus quand L1a est seul : +13 374 / +141 226 dans les deux cas.
  R_NAIS donnait +13 381 / +141 294, l'écart venant du contexte de production sur Live Fire. En combinaison,
  les deux formes divergent (§5.2).

## 3. Dénominateur fixe consolidé, recalculé selon sa règle (C10, `r_comb2_denominateurs*.tsv`)

**Règle appliquée.** Par film, le maximum des records utiles lus sur toutes les marches de la campagne, chaque
couple (source, variante) étant classé. 14 sources brutes :
- bis 1, bis 2 (`ti=43`, positions instruments et production), bis 3 (populations, `ti=3`) ;
- R-COMB (configurations et dérivation) ;
- R-LOC (deux passes), R-NAIS, R-VEH, R-COMP (L3 et P3) ;
- R-COMB-2 (deux passes).

`r_comp_p3_livefire.tsv` n'a pas de colonne d'utiles lus : il n'entre pas.

Exclus, avec le nombre de lignes de chaque classe dans `r_comb2_denominateurs_classes.tsv` :
- **témoins négatifs** : `largeur-*`, `ti4-i0-26`, `moteur, i15 ±1`, et `mpp8/3*` hors des formats 24-25 ;
- **contrôles** : `controle:reference-instruments` ;
- **variantes rejetées par la mesure** :
  - `libre` et `ls+libre` (D-77) ;
  - `portee-neuf*` et `feuille4-brute*` (R_VEH §3.2) ;
  - `*-hors-evenements` (R_NAIS §3.2) ;
  - `désaveu NEW à masque impossible` et `désaveu des deux` (R_COMP §4.4) ;
  - les sites de position rejetés et `jeu:tous` (BIS_2, BIS_4) ;
  - les hypothèses de format `i4` (L3b non discriminé) ;
- **oracles de lots retirés ou sans lot** :
  - `ls+vueA` (L1b sorti, D15 ; 45 % de gains factices) ;
  - `(iii')`, `(iii)` et `a-(iii')*` (L1c retiré) ;
  - toutes les variantes `P6` (pas de lot P6, D3).

Une forme « large » (C10 au pied de la lettre : seuls `libre`, `ls+libre` et `ls+vueA` sont retirés, avec les
contrôles et les témoins) est publiée à côté. **Elle donne le même maximum sur les 21 films** : la règle n'a
pas d'effet sur le chiffre.

| Groupe | Fixe §6.5.3 (consolidé du plan) | **Fixe R-COMB-2** | Écart |
|---|---|---|---|
| HI_1_13_0 | 2 959 104 | **3 073 267** | +3,9 % |
| corpus (20 films) | 7 176 150 | **7 758 290** | +8,1 % |

Source du maximum : une configuration de R-COMB-2 (C11, C11+WO, `full-L2`, `full-L4a`, ou F12 pour `81c02726`) pour 17 films sur 21.
Pour `50247b26`, `a521164d` et `f75e7053`, c'est `r-comb : full` ; pour `a349fea8`,
`r-comb-deriv : L1(communes)+L9`. Le fixe monte surtout par LM sur les formats 24-25 : `1c4c63c2` passe de
981 811 à 1 314 310.

**C10, ce qui change** : le maximum de `d9781168`, `c75f33b8` et `bf15f7ab` ne vient plus de `ls+vueA`, ni
celui de `1c4c63c2` et `50247b26` de `libre` ou `ls+libre` (classés exclus). Il vient de C11, sauf `50247b26`
(`r-comb : full`). R-COMP est désormais versé.

**C11 / D-71 (RC-4)** : sur ce fixe, sous les six leviers de R-COMB (ordre D-RI), seuls `f75e7053` (98,3 %) et
`81c02726` (97,5 %, hors corpus) dépassent 95 % ; `c75f33b8` tombe à 87,7 %. La correction de la critique
tient. Sous C11, neuf films dépassent 95 % :
- `bfecd02b` 99,8 %, `bf15f7ab` 99,3 %, `81c02726` 98,5 %, `c75f33b8` 98,3 % ;
- `51ebbc0f` 97,3 %, `396cfc92` 96,5 %, `d9781168` 96,2 %, `fb1a1a72` 95,9 % ;
- `60ae07c4` 95,2 %.

## 4. Par film : la combinaison C11 (gate 2, `r_comb2_par_film_C11.tsv`)

| Film | Build | Sains réf. → C11 | Utiles sains réf. → C11 | Fixe : réf. → C11 | Perdus sains bruts (dont → contredits) |
|---|---|---|---|---|---|
| `bfecd02b` | HI_1_13_0 | 26 380 → 30 637 | 226 529 → 272 423 | 83,0 → 99,8 % | 0 |
| `bf15f7ab` | HI_1_13_0 | 28 465 → 30 229 | 214 974 → 231 107 | 92,3 → 99,3 % | 0 |
| `c75f33b8` | HI_1_13_0 | 22 855 → 26 792 | 144 430 → 176 109 | 80,6 → 98,3 % | 0 |
| `51ebbc0f` | HI_1_13_0 | 9 759 → 28 900 | 58 585 → 217 849 | 26,2 → 97,3 % | 0 |
| `396cfc92` | HI_1_13_0 | 22 819 → 29 903 | 166 984 → 231 140 | 69,7 → 96,5 % | 0 |
| `d9781168` | HI_1_13_0 | 26 214 → 40 589 | 180 053 → 309 622 | 55,9 → 96,2 % | 0 |
| `fb1a1a72` | HI_1_13_0 | 22 279 → 45 492 | 161 566 → 345 033 | 44,9 → 95,9 % | 0 |
| `0797ce72` | HI_1_13_0 | 19 088 → 24 470 | 159 329 → 207 817 | 70,8 → 92,3 % | 0 |
| `f75e7053` | HI_1_13_0 | 23 416 → 25 677 | 160 042 → 177 529 | 83,2 → 92,3 % | 0 |
| `4f77afc1` | HI_1_13_0 | 21 976 → 27 941 | 537 801 → 714 792 | 65,1 → 86,6 % | 37 (0) |
| `81c02726` (hors corpus) | HI_1_13_0 | 15 184 → 18 925 | 123 525 → 158 889 | 76,6 → 98,5 % | 0 |
| `bcb6d393` | HI_1_12_0 | 5 830 → 19 118 | 35 143 → 132 854 | 23,7 → 89,7 % | 0 |
| `e5adf7b2` | HI_1_11_0 | 4 121 → 14 007 | 79 402 → 347 573 | 20,7 → 90,6 % | 29 (1) |
| `084a804d` | HI_1_10_0 | 4 774 → 25 519 | 88 029 → 689 473 | 11,3 → 88,3 % | 38 (1) |
| `111fa685` | HI_1_10_0 | 4 017 → 13 285 | 42 600 → 279 576 | 13,0 → 85,1 % | 72 (0) |
| `1c4c63c2` | HI_1_10_0 | 13 540 → 47 337 | 164 875 → 1 005 401 | 12,5 → 76,5 % | 1 141 (442) |
| `11de8353` | HI_1_9_0 | 5 613 → 14 718 | 65 229 → 290 311 | 20,1 → 89,4 % | 10 (0) |
| `60ae07c4` | HI_1_8_0 | 13 828 → 45 585 | 82 888 → 342 032 | 23,1 → 95,2 % | 10 (0) |
| `a521164d` | HI_1_4_1 | 692 → 765 | 78 → 78 | 0,0 % | 0 |
| `a349fea8` | version-33 | 427 → 454 | 3 595 → 3 660 | 0,7 % | 0 |
| `50247b26` | version-31 | 143 → 149 | 274 → 298 | 0,1 % | 0 |

**Gate 2 de la combinaison C11 : tenu sur 21/21 films**, en paquets sains et en records utiles sains. En brut,
1 337 paquets sains sont perdus sur le corpus, dont 1 141 sur `1c4c63c2` ; chacun est compensé dans son film.

**F12 (avec L7) ne tient pas le gate** : `f75e7053` passe de 23 416 à 21 013 sains (−2 403, −18 498 utiles).

## 5. Contributions marginales et gate 2 levier par levier

### 5.1 Par build (C11, paquets sains / utiles sains marginaux, `r_comb2_marginaux_C11.tsv`)

| Levier | HI_1_13_0 | HI_1_12_0 | HI_1_11_0 | HI_1_10_0 | HI_1_9_0 | HI_1_8_0 | anciens (3) | `81c02726` |
|---|---|---|---|---|---|---|---|---|
| LM | 0 | 0 | +9 513 / +262 865 | +62 514 / +1 651 802 | +9 069 / +224 266 | +28 748 / +241 004 | 0 | 0 |
| L8 | +37 201 / +296 341 | 0 | 0 | −2 / −48 | 0 | 0 | 0 | 0 |
| LS | +18 512 / +166 567 | 0 | 0 | +8 557 / +215 552 | 0 | +7 153 / +61 771 | 0 | 0 |
| L1a | +19 874 / +220 453 | +1 736 / +15 693 | +644 / +17 254 | +6 303 / +159 758 | +659 / +16 522 | +356 / +2 395 | +22 / +2 | +9 / +92 |
| L2 | +7 893 / +93 604 | +12 670 / +92 486 | +1 / +23 | **−913 / −16 078** | +2 / +25 | +2 / +10 | +2 / +24 | +3 615 / +34 206 |
| L3a | +6 855 / +110 746 | +169 / +1 253 | +1 / +1 | +2 833 / +83 438 | +103 / +2 322 | +2 035 / +17 741 | +4 / +59 | +149 / +1 402 |
| L6a | +4 444 / +38 691 | — | — | — | — | +3 817 / +27 092 | — | — |
| L9 | +1 601 / +45 605 | 0 | +87 / +2 659 | +4 451 / +112 993 | +44 / +1 111 | +246 / +2 298 | 0 | 0 |
| L4a | +2 431 / +59 984 | 0 | +216 / +6 304 | +1 560 / +43 392 | +232 / +6 165 | +1 / +2 | 0 | +5 / +45 |
| L6b | +176 / +37 | +310 / +2 738 | +951 / +23 147 | 0 / −23 | 0 | +218 / +43 | +96 / +6 | 0 |
| LP | +644 / +5 986 | 0 | +257 / +8 014 | +4 / +35 | +376 / +9 439 | 0 | 0 | 0 |

Points de l'indicateur sur HI_1_13_0 (marginal, fixe 3 073 267) :
- L8 9,64 ; L1a 7,17 ; LS 5,42 ;
- L3a 3,60 ; L2 3,05 ; L4a 1,95 ; L9 1,48 ; L6a 1,26 ;
- LP 0,19 ; L6b 0,00.

### 5.2 Interactions (mesuré)

- **Elles sont positives pour presque tous les leviers.** Corpus, interaction = marginal − seul :
  - LM +31 399, L1a +16 220, LS +16 135, L3a +8 425, L8 +6 603 ;
  - L9 +4 655, L6a +3 191, L4a +3 037, L2 +2 059, L6b +1 291, LP +587.

  L'inverse de R-COMB, où L1 (oracle) et L2 se recouvraient : ici L1a est un localisateur réel, et L2 n'a plus
  de recouvrement négatif.
- **La condition causale de L1a s'allume sous LM.** Seul, L1a est actif sur 0 chunk de `084a804d`,
  `111fa685`, `1c4c63c2`, `60ae07c4` et `e5adf7b2`. Dans C11, il l'est sur 52, 26, 19, 42 et 21 chunks : une
  fois le MPP lu à 8/3, la marche lit proprement assez de NEW pour que le score d'allocateur dépasse 50 %.
  Dans C11 privée de LM, il retombe à 0. Conséquence : le verdict de R_NAIS « la condition désactive L1a sur
  HI_1_8_0 à HI_1_11_0 » ne vaut plus en combinaison. La forme EN LIGNE (score de la marche elle-même) est
  celle qui suit le monde ; la forme `|cumul` de R_NAIS (score de la référence) ne s'allumerait pas.
- **L8 + LS** : +49 740 sains, pour une somme des seuls de +48 683.
- **L7 détruit la combinaison.**
  - Seul : −2 688 sains et 9 films en baisse (`51ebbc0f` −1 032, `4f77afc1` −704, `1c4c63c2` −556,
    `60ae07c4` −276, `0797ce72` −143, `fb1a1a72` −102 ; `111fa685`, `e5adf7b2` et `084a804d` de −8 à −15).
  - Dans F12, marginale −58 719 / −908 824.
  - L1a perd sa marginale sous L7 (F12 : −23 695) : lier un NEW sur un slot occupé casse les chaînes que L1a
    valide.
  - Mesuré : le refus actuel (`contreditUneEntiteVivante`) protège. L7 sort du plan ; la piste de la chronique
    `11de8353` slot 688 (R_COMP §4.4, point 3) reste ouverte, mais pas par cette règle.

### 5.3 Gate 2 par film, levier par levier (`r_comb2_gate2_par_film_C11.tsv`)

« Seul » = levier seul contre la référence ; « marginal » = C11 contre C11 privée du levier. Toute ligne non
listée est tenue.

| Levier | Film | Seul : sains / utiles sains | Marginal : sains / utiles sains | Nature (mesuré) |
|---|---|---|---|---|
| L8 | `1c4c63c2` | −1 / 0 | −2 / −48 | seul : 1 paquet sain devient contredit (requalification) |
| L2 | `084a804d` | −1 / −105 | +4 / +18 | seul : 4 sains perdus, tous devenus contredits |
| L2 | `1c4c63c2` | −28 / −2 | **−916 / −16 120** (C11 47 337 contre 48 253 sans L2) | seul : 47 sains requalifiés et 358 gains, dont 339 contredits ; marginal : 1 141 sains bruts perdus par C11 contre 689 sans L2 |
| L4a | `d9781168` | 0 / 0 | 0 / −11 | records utiles d'un même paquet |
| L6b | `60ae07c4` (production) | 0 / −7 | +218 / +43 | seul : 8 gagnés et 8 perdus sains ; `flock-position` seul donne la même chose (0 / −7) |
| L6b | `51ebbc0f` | 0 / −4 | 0 / 0 | |
| L6b | `1c4c63c2` | −1 / 0 | 0 / −23 | |
| L7 | 9 films | jusqu'à −1 032 / −22 444 | rejeté | §5.2 |

**Verdicts gate 2** :
- tenus seuls et en marginal : LM, LS, L1a, L3a, L6a, L9, LP ;
- en défaut : L8 (un paquet requalifié), L2 (`1c4c63c2`, marginal −916), L4a (−11 utiles), L6b (trois films
  seul, un en marginal), L7 (rejeté).

La règle « seule exception : une baisse expliquée par une fermeture factice retirée » ne couvre AUCUN de ces
cas : ce sont des paquets sains qui deviennent contredits ou non fermés, pas des fermetures factices retirées.

### 5.4 Requalifications et pertes jugées (B9, LM)

- **L3a, « 0 sain perdu » (B9)** : faux à la lettre. Mesuré seul :
  - 48 paquets sains en référence deviennent contredits (`1c4c63c2` 41, `4f77afc1` 5, `11de8353` 1,
    `396cfc92` 1) ;
  - 0 deviennent non fermés ;
  - 3 623 gains sains ;
  - net +3 575, 0 film en baisse nette.

  La bonne formule : « 0 film en baisse nette ; 48 sains requalifiés contredits, aucun perdu non fermé ».
- **LM, pertes jugées** (prérequis du §6.5.4 du plan) : 2 101 paquets perdus bruts (corpus, 6 films). Seuls
  750 étaient sains en référence :
  - 443 deviennent contredits ;
  - 307 deviennent non fermés ;
  - `1c4c63c2` porte 648 de ces 750 (425 → contredits) ;
  - en face, 79 195 gains sains ;
  - aucun film en baisse nette (`1c4c63c2` : 13 540 → 32 910 sains, +19 370).

  La largeur 8/3 reste mesurée, pas lue dans un exécutable de ces builds (D6 ouverte).

## 6. Gate 3 killsource, levier par levier (B7, B8, `r_comb2_ks_*.tsv`)

### 6.1 Protocole

`cmd/killsource json <film> -carte <carte>` sous la surcouche unique, 19 films : les témoins de
`config/replay_corpus.toml`, qui portent leur carte. Chaque variante est comparée à la production, mort par
mort (clé instant + victime), sur la voie (marche ou scan), le tag, le statut, la nature de la source, le
crédit, l'assistant, la part du tueur et la divergence. Les champs hors morts sont comparés chemin par chemin.

**Films manquants (B8)** :
- `1c4c63c2` : sa carte n'est donnée par aucune source lisible. Ni `config/replay_corpus.toml` (il n'en fait
  pas partie), ni son manifeste (`film_manifests/1c4c63c2.json` ne porte que `chunks`), ni la section
  d'identification du film (`ReadFilmIdentity` ne lit pas de carte). La commande refuse sans carte (règle du
  2026-09-27), et la base de matchs est hors de ce périmètre.
  - Indice non utilisé, supposé : son registre lie les slots 123, 126 à 129 à `ti=4` (`rloc_registre.tsv`),
    comme `6b0e6f0f` (Refuge). Décoder sous la carte d'un autre film est exclu par la règle.
- `81c02726` : même cause. Il est hors corpus.

**Positions** : la sortie JSON de killsource ne publie AUCUNE position. Les chemins des morts : `temps_ms`,
`victime`, `credit_du_jeu`, `assistant`, `parts_de_degats`, `source_du_degat_fatal`, `lecture`,
`divergence`. Le delta « positions » ne se mesure donc pas à ce gate (établi par la liste des chemins).
L'arme = `source_du_degat_fatal.tag` / `statut` / `nature`.

### 6.2 Résultat (mesuré, 19 films, 2 747 morts)

| Variante | Morts | Scan → marche | Marche → scan | Tag, statut, nature, crédit, assistant, part, divergence | JSON différent (films) | Champs hors morts qui changent |
|---|---|---|---|---|---|---|
| L8 | 2 747 = | 0 | 0 | 0 | 3 | `calibration` (score de l'oracle axisW seulement) |
| L2 | = | 0 | 0 | 0 | 1 | `sante.taux_inexpliques`, un compteur expvar (`bfecd02b`) |
| L3a | = | 0 | 0 | 0 | 12 | `calibration` ×11 (ORACLE axisW 11 → 26 sur `51ebbc0f`) ; `111fa685` : population et taux de la voie séquentielle |
| L4a | = | 0 | 0 | 0 | 1 | `calibration` |
| L6a (Live Fire) | = | 0 | 0 | 0 | 1 | `calibration` (`60ae07c4`, ORACLE axisW 8 → 6) |
| L6b | = | **8** (`a521164d` 5, `a349fea8` 2, `e5adf7b2` 1) | 0 | 0 | 5 | concordance et gate par voie sur ces trois films |
| WO | = | 0 | 0 | 0 | 4 | `calibration` ; `111fa685` voie séquentielle |
| LS | = | **229** (`d9781168` 103, `60ae07c4` 78, `c75f33b8` 48) | 0 | 0 | 3 | concordance, gate par voie |
| L8+LS | = | 229 | 0 | 0 | 5 | idem + `calibration` |
| L7 | = | 0 | 0 | 0 | **0** | — |
| L8+L2+L3a+L4a+L6b+L6a+LS (= C11 côté binaire) | = | **237** | 0 | 0 | 16 | concordance, gate par voie, `calibration` |

Les variantes « combinaison moins un levier » (sept, plus les huit de F12) sont dans `r_comb2_ks_resume.tsv`.
Chacune vaut exactement la somme des voies de ses leviers : 237 = 229 (LS) + 8 (L6b), et aucune interaction
n'apparaît.

Dans les sept variantes, `calibration` ne change que dans son diagnostic d'ORACLE. Les parties `LU`
(largeurs lues) et `DECIDE` sont identiques sur toutes les variantes et tous les films (comparaison des
chaînes, scores retirés).

**Delta déclaré par lot (gate 3)** :
- **L8, L2, L3a, L4a, L6a** : aucune mort, aucune valeur, aucune voie. Seuls changent les diagnostics de
  calibration (oracle) et, pour L3a et L2, des compteurs de santé sur un film. `killsource.Rev` n'a pas à
  monter pour la sortie publiée (supposé : la règle de montée sur un diagnostic n'est pas instruite ici).
- **L6b** : voie de 8 morts (scan → marche), aucune valeur. Delta à déclarer ligne à ligne.
- **LS** : voie de 229 morts sur ces 19 films, aucune valeur. C'est exactement le chiffre de R_LOC §3.2 pour
  les 19 témoins (229). `killsource.Rev` et le backfill sont DUS (D-74, voie publiée).
- **L7** : JSON identique 19/19. Le rejet vient de la carte, pas de killsource.
- **Non mesurable dans le binaire, et pourquoi** :
  - **LM** : killsource ne lit pas le profil par format. Il pose `profile.MPPParDefaut()`
    (`grammar/profil_balayage.go:58`, `facts/killsource/calibrate.go`), donc un lot LM limité à
    `profile.MPPPourFormat` laisse killsource inchangé (établi par lecture). Étendre LM à killsource serait un
    autre lot.
  - **L1a** : modifie `debutDeLaListe`, que killsource n'appelle pas (R_LOC §2.1 : il a son propre
    localisateur, `locateStrict` + repli) ; nul par construction.
  - **LP** : lit le bloc de type 1 à l'ouverture du chunk ; killsource ne lit pas ce bloc (son monde : première
    déclaration + images-clés, `world.go:57-140`) ; nul par construction pour un lot limité à la marche de la
    cuisson.
  - **L9** : oracle. Un lot L9 change `MarcheDImageCle`, que killsource utilise (`world.go:72-73`) : son gate 3
    se mesure sur le code du lot, pas sur un oracle. **Item ouvert.**

## 7. Point 3 (B7) : la signature de LS sous L8

**Question.** R-LS identifie l'archétype `high-frequency` par NOM (« seul composant nommé `high-frequency` »,
`rlocArchetypesHF`). R-HOM dit « routage par archétype ou par table, jamais par nom », et L8 route
`ti=3 i1 high-frequency` vers sa vraie table (26 bits, `FUN_142ed4880`, table `0x143d07af0`). La signature
reste-t-elle juste ?

**Mesure** (`TestRComb2Signatures`, 21 films, `r_comb2_signatures_ti.tsv`). On relit, au moment de la
localisation (monde d'avant le paquet), le record de chaque signature trouvée :

| Marche | Signature stricte du slot 123 | Signature high-frequency de LS |
|---|---|---|
| référence | 106 558 : toutes `ti=4`, 35 bits, 1 composant | — |
| L8 | 106 558 : toutes `ti=4`, 35 bits, 1 composant | — |
| LS | 106 558 : idem | 22 005 : toutes `ti=4`, 35 bits, 1 composant |
| L8 + LS | 106 558 : idem | 21 860 : toutes `ti=4`, 35 bits, 1 composant |

Début de liste paquet par paquet, LS contre L8+LS (`r_comb2_ls_l8.tsv`) :
- 304 paquets à événements diffèrent sur 169 079 ;
- dont 158 sur `51ebbc0f` et 128 sur `fb1a1a72`, les films où L8 agit ;
- 69 paquets ne sont localisés que sous L8+LS, 3 que sous LS.

**Réponse (mesuré).** La signature reste juste sous L8 :
- aucune signature, ni du slot 123 ni high-frequency, ne tombe sur un record `ti=3` ;
- le prédicat par nom « composant UNIQUE nommé `high-frequency` » exclut `ti=3` (deux composants) dans les 30
  registres lus par R-LOC ;
- les 145 signatures high-frequency en moins sous L8+LS viennent du monde (L8 lit autrement les records
  d'avant), pas de l'identification ;
- L8 et LS s'additionnent : +49 740 sains contre +48 683 pour la somme des seuls ;
- en killsource, L8+LS = LS en voies (229).

**Règle proposée** (pour LS et LU) : identifier l'archétype par la **table de son composant**, jamais par le
nom. Concrètement : « archétype à composant unique dont le composant est lu par le lecteur de `ti=4 i0`
(`R(8)`, table `0x143d06a60`, lecteur `FUN_14076d034`) ». Dans le Go, c'est la même clé de routage que L8
introduit (archétype + indice → lecteur, R-HOM), donc une seule source de vérité.

Sur les 21 films, la règle par table et la règle par nom désignent le même archétype (`ti=4`) : le résultat ne
change pas. La règle par table supprime la dépendance au nom, qui deviendrait fausse si un build donnait un
composant unique homonyme à un autre archétype (supposé : non observé).

## 8. E21 : requalification des « bornes »

Une « borne » (PLAN §6.0) est le gain maximal d'UN mécanisme mesuré par un oracle ou une copie de recherche,
**dans le monde de la référence**. La mesure ensemble montre que ce n'est une borne ni du lot réel en
combinaison, ni d'un autre mécanisme.

| Valeur du plan | Nature | Ce qu'en dit la mesure | Requalification |
|---|---|---|---|
| L1 « borne oracle (i)+(ii) +28 168 nets » | oracle de liaison, monde de référence | les lots réels qui couvrent ces régions (L1a causal et LS) font +29 594 et +34 222 sains marginaux dans C11 ; LS seul +18 087 > l'oracle (ii) de BIS_1 +11 540 (D-79) | **pas une borne** des lots de naissances : borne d'un oracle de liaison dans un monde donné. Dépassée par des mécanismes de LOCALISATION qu'elle ne modélise pas. À citer comme « gain de l'oracle (i)+(ii) en référence » |
| L9 « oracle +1 784 / −1 » | oracle | seul +1 774 sains ; marginal dans C11 +6 429 | borne du mécanisme EN RÉFÉRENCE seulement ; en combinaison, L9 libère des paquets que d'autres leviers ferment |
| L3 « ≤ 4 331 » (estimé) | estimation | L3a seul +3 575 ; marginal +12 000 | estimation périmée (déjà remplacée par R-L3) ; ce n'était pas une borne |
| L4 « ≤ 1 666 » (estimé) | estimation | L4a seul +1 403 ; marginal +4 440 | idem |
| L7 « ≤ 137 paquets » (mesuré) | compte de paquets touchés | seul −2 688 sains ; F12 −58 719 | ce n'était pas une borne de gain : c'était un compte de paquets qui suivent un refus. Le lot est nuisible |
| R-COMB « HI_1_13_0 ≤ 90 % » (estimé) | estimation par densité | 93,8 % mesuré (92,3 % sans L9) | **réfutée** |
| L1b « borne +8 826 sains » (oracle de la vue A) | oracle à 45 % de gains factices | non rejoué ici (L1b sorti, D15) | reste une estimation, ni borne ni gain |
| LS « +18 119 » | A/B d'une copie de recherche | seul +18 087 sains ; marginal +34 222 | mesure d'un mécanisme réel en référence ; minorant de sa marginale ici |

**Règle proposée pour le plan** : réserver « borne » à un majorant démontré d'un mécanisme dans un monde fixé.
Publier tout chiffre d'oracle avec son monde de dérivation. Ne jamais additionner ni comparer des « bornes »
entre leviers ; seule une combinaison mesurée (R-COMB-2) dit ce que donne une vague.

## 9. Point 4 : vérifications ciblées

- **L6b au gate par film (B5)** : **confirmé**, et en contexte de production.
  - `60ae07c4` (Live Fire, production) : L6b seul 0 paquet / −7 utiles sains (BIS_4 donnait −6 en
    instruments). `flock-position` seul donne la même chose.
  - S'y ajoutent `51ebbc0f` (0 / −4) et `1c4c63c2` (−1 / 0).
  - En marginal dans C11, `60ae07c4` passe à +218 / +43, mais `1c4c63c2` est à 0 / −23.
  - **L6b n'est pas prêt** au sens du gate 2.
- **L6a étendu à `world-object-i0` (B6)** : **infirmé comme extension globale**.
  - WO seul : 6 films en baisse (`60ae07c4` −30 / −40, `4f77afc1` −10, `e5adf7b2` −6 / −221, `1c4c63c2` +3 /
    −25, `51ebbc0f` −1, `d9781168` −1).
  - Sous C11, WO ajoute +989 / +11 484 sur `0797ce72` et +65 sur `60ae07c4`, mais retire −494 / −15 815 sur
    `1c4c63c2`.
  - Limité à Live Fire : L6a+WO perd 28 sains contre L6a seul sur `60ae07c4` (R-P3 : −28), et gagne dans C11
    sur les deux films Live Fire.
  - **Non prêt** ; à instruire comme lot propre (lecture du jeu à la carte, et pourquoi `1c4c63c2` régresse).
- **L3a « 0 sain perdu » (B9)** : voir §5.4 (48 requalifiés).
- **LM sous le contexte de PRODUCTION sur `60ae07c4` (E22)**, mesuré :
  - LM seul +18 678 sains (13 828 → 32 506), 19 sains perdus bruts, dont 2 contredits ;
  - L6a seul +1 790 ;
  - **LM × L6a +22 194** (36 022 sains), soit une interaction de +1 726 ; aucune baisse ;
  - dans C11, LM sur ce film : +28 748 sains marginaux.
- **LM et D6** : la combinaison LM × L6a est désormais mesurée. La largeur 8/3 reste une mesure (fermeture,
  oracle `n2`, châssis), pas une lecture d'exécutable de ces builds : D6 reste à trancher par l'utilisateur
  avant le lot.

## 10. Ordre de vague proposé (contribution SAINE marginale, corpus, C11)

Le cadre reste celui des décisions FERMES (D-RI : composants en vague 1, marche en vague 2). Cet ordre révise
la recommandation du §6.5.4 du plan ; il ne change aucune décision du §3.

**Vague 1, composants**, par paquets sains marginaux :

| Rang | Lot | Sains marginaux | Utiles sains marginaux | Condition avant le lot |
|---|---|---|---|---|
| 1 | LM | +109 844 | +2 379 937 | décision D6 (largeur mesurée, pas lue) ; formats 24-25 seulement ; pertes jugées (§5.4) |
| 2 | L8 | +37 199 | +296 293 | `1c4c63c2` : 1 paquet requalifié seul, −2 / −48 en marginal ; à instruire ou à admettre explicitement |
| 3 | L2 | +19 657 | +170 094 | **gate 2 en défaut** : `1c4c63c2` −916 sains en marginal, `084a804d` et `1c4c63c2` seul. HI_1_10_0 : 382 / 401 gains factices (BIS_2) ; à réparer avant le lot |
| 4 | L3a | +12 000 | +215 560 | — |
| 5 | L6a (Live Fire) | +8 261 | +65 783 | sans `world-object-i0` |
| 6 | L9 | +6 429 | +164 666 | oracle : le lot réel est à mesurer, et son gate 3 aussi (`MarcheDImageCle` est lue par killsource) |
| 7 | L4a | +4 440 | +115 847 | `d9781168` −11 utiles en marginal |
| 8 | L6b | +1 751 | +25 948 | **gate 2 en défaut** seul (3 films) et en marginal (`1c4c63c2`) |

**Vague 2, marche** :

| Rang | Lot | Sains marginaux | Utiles sains marginaux | Condition |
|---|---|---|---|---|
| 0 | LU | 0 (structure) | — | paramètre d'ordre (D-73) ; prédicat par table, pas par nom (§7) |
| 1 | LS | +34 222 | +443 890 | delta killsource : 229 morts (19 témoins), `killsource.Rev` et backfill dus |
| 2 | L1a (causale en ligne) | +29 594 | +432 077 | invariants du juge en code (L0), sinon le chiffre est biaisé (§0.2) ; seuil choisi sur le corpus |
| 3 | LP | +1 281 | +23 474 | bloc de type 1 en production (gate 4) |

Rejetés : **L7** (§5.2) ; **world-object-i0** comme extension de L6a (§9).

Par utiles sains, l'ordre de la vague 1 serait LM, L8, L3a, L2, L9, L4a, L6a, L6b. Les deux ordres placent LM
et L8 en tête. Sur HI_1_13_0 seul, le rang est L8, L1a, LS, L2, L3a, L6a, L4a, L9, LP, L6b (LM y vaut 0).

## 11. Limites

- **Oracles et juge** : L9 est un oracle. L1a filtre ses candidats par le juge. Le juge n'a que trois
  invariants (L0.6 absent). Toutes ces limites poussent l'indicateur vers le haut (§0.2).
- **Dénominateur** : C11 ou une configuration voisine est la marche la plus longue sur 17 films ; l'indicateur se rapproche donc d'un taux
  de fermeture de C11.
- **LM** : flux delta seulement, comme R_VEH §3.3 ; l'effet de 8/3 sur la marche d'image-clé n'est pas dans la
  carte (R_VEH §4).
- **LS × L1a** : la composition est un choix de cette recherche (§1.1), pas une règle de l'écrivain.
- **Contexte** : Live Fire en production, les 19 autres films en contexte des instruments (comme R-COMB). Le
  dénominateur mélange les deux.
- **Killsource** : 19 films. `1c4c63c2`, le film qui porte le plus de pertes en carte, n'est pas joué (§6.1).
- **L7** est la règle exacte en production (bascule dans `frame_infer.go`) ; il est mesuré seul et dans F12,
  pas dans d'autres combinaisons.

## 12. Découvertes (consignées, non traitées)

- **RC2-1** — L7 (lier le NEW sur slot occupé) est nuisible partout : −2 688 sains seul, −58 719 en
  combinaison, et il annule L1a. Le commentaire de `frame_infer.go` (« le jeu ne crée pas une entité sur une
  entrée occupée ») est conforté par la mesure ; D-13 (« le jeu crée sur une entrée occupée »,
  `FUN_1408f18d0`) est contredit dans sa conséquence de lot.
- **RC2-2** — L2 fait perdre 916 paquets sains à `1c4c63c2` dans la combinaison, alors que seul il n'en perd
  que 28 nets. C'est le film à 382 / 401 gains factices sous `ti=43` (BIS_2) : la grammaire T7 de `ti=43` lit
  probablement faux sur HI_1_10_0.
- **RC2-3** — La condition causale de L1a dépend des autres leviers. Sous LM, elle s'allume sur les formats
  24-25, où R_NAIS la disait éteinte. Un seuil choisi sur la référence n'est pas un seuil de la vague.
- **RC2-4** — Le dénominateur fixe monte de 8,1 % sur le corpus avec C11 (`1c4c63c2` +34 %). Toute vague
  devra le recalculer (D-42).
- **RC2-5** — Killsource ne lit pas le découpage MPP par format (`MPPParDefaut`). Si LM est admis, la cuisson
  et killsource liront deux découpages différents pour les formats 24-25 : un écart d'équivalence à décider
  (gate 3, équivalence lecteur).
- **RC2-6** — Les sondes de position (WO) et L6b changent la population des voies killsource sur des films
  sans objectif porté (`a521164d`, `a349fea8`, `e5adf7b2`) : non instruit.
- **RC2-7** — La recherche de l'archétype de la signature par `HeaderBit` dans `r_comb2_research_test.go`
  (colonne « 123 strict : record non retrouvé » de `r_comb2_passes_localisateur.tsv`) ne retrouve jamais le
  record. La convention de `HeaderBit` n'est pas celle de `marchLocateStrict`. La mesure a été refaite par
  `TestRComb2Signatures` (§7). Ces lignes du TSV sont sans valeur.

## 13. Gate et écarts

| Commande (depuis `apps/go-api`, 2026-10-02) | Résultat |
|---|---|
| `gofmt -l` sur `grammar/` et `surcouche_unique_postj12/` | vide |
| `go vet -tags=research ./...` | rc=0 |
| `go vet -tags=research,campagne_overlay -overlay=<surcouche unique, 9 entrées>` sur `grammar`, `facts/killsource`, `cmd/killsource`, `research/cmd_fermeture` | rc=0 |
| `go test -count=1 ./internal/archlint/` | ok (35,9 s) |
| `git status --short` | fichiers neufs seulement (3 sondes, `r_comb2_tsv/`, les deux ajouts de la surcouche, cette note), `SURCOUCHE_UNIQUE.md` (§9 ajouté) ; `.ai/thought_log.md` (entrée ajoutée) ; aucun fichier de production |

Sondes : 348, 326 et 98 lignes, `//go:build research && campagne_overlay` en ligne 1, sans suffixe de garde.

**Écart à la règle « pas de Python »** : une commande `python3 --version` a été lancée une fois, par erreur,
dans une commande de vérification (recherche d'un nom de fonction). Aucun script n'a été exécuté et aucun
fichier n'a été lu ni écrit par Python. Elle n'a aucun effet sur les mesures.

## 14. Fichiers

- Note : `.ai/V7.5/film_re/campagne_grammaire_2026-10-01/R_COMB_2.md`.
- Sondes : `apps/go-api/internal/games/halo_infinite/film/internal/grammar/r_comb2_research_test.go`,
  `r_comb2_configs_research_test.go`, `r_comb2_signatures_research_test.go`.
- Surcouche : `surcouche_unique_postj12/frame_infer.go`, `rcomb2_leviers.go`, `overlay.json` (9 entrées).
- TSV : `r_comb2_tsv/`, en trois familles.
  - Bruts :
    - `r_comb2_configs.tsv` (passe 1, F12), `r_comb2_configs_passe2_sans_L7.tsv` (passe 2, C11) ;
    - `r_comb2_passes_localisateur.tsv`, `r_comb2_signatures_ti.tsv`, `r_comb2_ls_l8.tsv` ;
    - `r_comb2_ks_resume.tsv`, `r_comb2_ks_morts_changees.tsv`, `r_comb2_ks_champs_hors_morts.tsv`,
      `r_comb2_ks_cartes.tsv`.
  - Dénominateurs : `r_comb2_denominateurs.tsv`, `r_comb2_denominateurs_classes.tsv`.
  - Agrégats, en `_C11` et en `_F12` : `r_comb2_par_build_*`, `r_comb2_par_film_*`,
    `r_comb2_gate2_par_film_*`, `r_comb2_marginaux_*`.
  - Scripts d'agrégation : voir §1.4.
