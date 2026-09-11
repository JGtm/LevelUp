# Mesure M — l'en-tête d'un record d'image-clé est-il propre au type d'entité ? (2026-09-11)

> Mesure, pas fonctionnalité : **aucun code de production n'a changé**. Trois fichiers de banc
> ont été ajoutés, tous sautés sans films (donc verts en CI). Branche `wt/mesure-ti9`.

## Verdict

**Thèse CONFIRMÉE, et reproduite sur notre arbre avec nos films.** L'en-tête d'un record
d'image-clé n'est pas constant : il est propre au type d'entité. `ti=9` demande **47 bits**,
`ti=35` (biped) en demande **64**, et les deux sont mutuellement exclusifs — à 47 bits le biped
décode **zéro** composant, à 64 bits `ti=9` en décode **zéro**. Sur les trois films mesurés,
47 bits rend **huit** entités `ti=9` portant un `managed-player-team-designator-component`
strictement **stable**, réparties **4-4**, sur toute la durée du film. C'est le **seul** préfixe
sur les 301 essayés (0..300) à tenir ce critère sur les trois films.

Conséquence directe : **le film porte bien l'équipe**, contrairement à ce qu'affirment
`document.go:346` et `document.go:558-561`. Elle n'est pas encore **exploitable** : le lien
entité → joueur reste non résolu, et deux pistes bon marché ont été mesurées et **écartées**
(volets 3 et 3 bis ci-dessous).

## Méthode

Grammaire rejouée, celle du fork (commit `2a8b21a51`) :

```
[en-tête : H bits][état par défaut de l'archétype][1 bit porte has-components][masque + composants]
```

`H` est la variable. L'état par défaut, la porte, le masque et la boucle de composants sont
ceux de **production** (`consumeKeyframeDefaultState`, `decodeDeltaWithArch`,
`traverseComponentLoop`) : rien n'est recopié, seul l'offset de départ change. Pour `ti=9`,
`consumeDefaultStateTI9` consomme 14 bits (V(1) + 6 + 6 + 1), d'où le préfixe total de 61 bits
mesuré par le fork = 47 + 14.

Le désignateur d'équipe est **relu** à `CompResult.StartBit` sur 4 bits. Le décodeur ne capture
pas ce composant ; la mesure ne le lui fait pas capturer, elle relit les mêmes bits. C'est ce qui
permet de ne toucher à aucun fichier de production.

**Anti-piège « 0 désync ».** Le fork a documenté que « 0 désync » ne prouve rien : un masque lu
à zéro fait sortir la boucle immédiatement, sans consommer un composant. Tous les tableaux
ci-dessous publient le nombre de composants décodés **à côté** du compte de désyncs, et le
balayage écarte d'office tout préfixe à 0 composant. Le tableau A montre que ce piège se serait
déclenché ici : à 64 et 108 bits, `ti=9` donne **0 désync** et **0 composant** — une lecture
« propre » qui ne lit rien.

## Films mesurés

Trois matchs d'arène 4v4, tous les huit joueurs présents du début à la fin, aucun bot
(`match_participants`, lecture en `read_only` via `cmd/diag_q`).

| Film | Match | Carte / mode | Chunks | Vérité (table des scores) |
|---|---|---|---|---|
| `4a93f0e2` | `4a93f0e2-5bc1-…` | Dynasty — Arena:CTF | 53 | 4 joueurs équipe 0, 4 équipe 1 |
| `8b512df2` | `8b512df2-2b5a-…` | Forest — Arena:CTF | 52 | 4 / 4 |
| `ce083875` | `ce083875-2d2d-…` | Origin — Assault:Neutral Bomb | 51 | 4 / 4 |

## A. Marche des records, par largeur d'en-tête et par type

| en-tête | ti | records | traversés | désync | **vides (0 comp.)** | **composants** | désignateurs |
|---:|---:|---:|---:|---:|---:|---:|---:|
| **47** | **9** | 1151 | 1151 | **0** | **0** | **1151** | **1151** |
| 64 | 9 | 1151 | 1151 | 0 | 1151 | 0 | 0 |
| 108 | 9 | 1151 | 1151 | 0 | 1151 | 0 | 0 |
| **47** | **35** | 1101 | 1101 | 0 | **1101** | **0** | 0 |
| 64 | 35 | 1101 | 1101 | 280 | 390 | 13 449 | 0 |
| 108 | 35 | 1101 | 1101 | 11 | 823 | 5 525 | 0 |

Lecture : la **contre-épreuve** de la thèse « par type » tient. À 47 bits le biped s'effondre
exactement comme le fork l'annonce (0 composant contre 13 449 à 64). Et à 64 ou 108 bits, `ti=9`
ne décode rien — ce sont les lignes que le critère naïf « 0 désync » aurait déclarées bonnes.

Note : 1151 composants pour 1151 records `ti=9`, soit **exactement un composant par record** —
sur les 10 que porte l'archétype, seul i0 (le désignateur) est présent au masque.

## B. Désignateur d'équipe (en-tête 47, ti=9), par entité

| Film | Slots des huit entités | Lectures / entité | Stable | Valeurs |
|---|---|---:|---|---|
| `4a93f0e2` | 1297, 1299, 1301, 1303, **1305, 1307, 1309, 1311** | 49 à 50 | 8/8 | `0` pour les 4 premiers, `1` pour les 4 derniers |
| `8b512df2` | 1288, 1290, 1292, 1294, **1296, 1298, 1300, 1302** | 49 | 8/8 | idem |
| `ce083875` | 1297, 1299, 1301, 1303, **1305, 1307, 1309, 1311** | 45 | 8/8 | idem |

Aucune entité ne change jamais de valeur : 143 relevés par film environ (≈ 400 au total), zéro
divergence. Les entités sont présentes du premier chunk de réplication au dernier (chunks 2→51,
2→50, 5→49).

Verdict par film : **TENU — 8 entités stables, 4-4** dans les trois cas, ce qui **coïncide avec
la table des scores** (4 joueurs par équipe, dans les trois matchs).

## C. Balayage des préfixes 0..300 (contre-épreuve d'unicité)

| | |
|---|---|
| Préfixes essayés | 301 |
| Préfixes qui décodent **au moins un** composant | 160 |
| Préfixes qui tiennent le critère 4-4 sur **les trois** films | **1** (= 47) |

Les 141 préfixes restants décodent zéro composant : le garde anti-piège les écarte avant tout
jugement.

## D. Volet 3 — le lien entité → joueur n'est pas dans l'état par défaut de ti=9

`consumeDefaultStateTI9` porte `V ; R(6) ; R(6) ; R(1)`. Les deux champs de 6 bits sont assez
larges pour un index de joueur. Relus bit à bit au même offset que la marche :

| Champ | Valeur observée, sur les 3 films et les 8 entités |
|---|---|
| bit de version V | 0 partout |
| champ 1 `R(6)` | **0** partout (≈ 1151 relevés) |
| champ 2 `R(6)` | **0** partout |
| drapeau `R(1)` | **1** partout |

Aucun index. Piste morte.

**Découverte de voisinage.** Les huit entités `ti=9` occupent des slots **consécutifs de deux en
deux**, intercalées avec autant d'entités `ti=47`
(`managed-object-networked-splash-message-*`, `personal-ai-data-component` — une par joueur), le
bloc étant précédé de quatre `ti=38` (objets positionnés) et suivi de trois `ti=14`
(`crew-order-*`). Le bloc `ti=9`/`ti=47` est donc bien un **bloc joueur**.

## E. Volet 3 bis — l'ordre des slots n'est PAS l'ordre des index de joueur

Le film porte déjà, ailleurs, un lien xuid → index de joueur : `weaponv3.ResolveXuidToPI`
(5 bits devant le xuid petit-boutiste, mesuré juste sur 116 films sur 116, branché par
`replay.ScanPlayerIndices`). Si le rang de slot `ti=9` était cet index, chaque équipe
occuperait un **bloc contigu** d'index — condition nécessaire. Confronté à la vérité de la base :

| Film | Lectures concordantes | Équipe 0 → index | Équipe 1 → index | Bloc contigu |
|---|---:|---|---|---|
| `4a93f0e2` | 52 (7 désaccords) | 1, 4, 5, 7 | 0, 2, 3, 6 | **non** |
| `8b512df2` | 51 (7 désaccords) | 0, 1, 3, 5 | 2, 4, 6, 7 | **non** |
| `ce083875` | 50 (7 désaccords) | 0, 3, 6, 7 | 1, 2, 4, 5 | **non** |

Les équipes sont **entrelacées** dans l'espace des index de joueur, dans les trois films. La
piste « rang de slot = index de joueur » est **réfutée**. Le lien reste non résolu, exactement
comme chez le fork.

*Limite honnête* : ce banc accepte un chunk dès que les huit xuids y sont résolus, règle plus
permissive que `injectiveOrEmpty` de `replay` ; les 7 désaccords par film en sont la trace. La
conclusion ne s'appuie pas dessus — un entrelacement dans les trois films ne devient pas
contigu en durcissant l'acceptation.

## Ce que ça implique

### Pour `document.go`

Deux commentaires sont **faux** et doivent être corrigés quand le lien sera résolu, pas avant :

- `document.go:345-346` : « l'équipe d'un joueur n'est PAS dans le film (cf. `Track.Team`) ».
- `document.go:558-561` : « Team vaut -1 : L'ÉQUIPE N'EST PAS DANS LE FILM ».

La formulation exacte à viser : *l'équipe est dans le film (huit désignateurs `ti=9`, stables,
4-4) mais elle n'est pas rattachable à un joueur — l'identité continue de venir de la base.*
Rien à changer dans le **comportement** : la base joint déjà l'équipe, et le ferait mieux.

### Pour nos bancs — le piège « 0 composant »

Le tableau A est la démonstration que le piège est **réel chez nous** : `ti=9` à 64 et 108 bits
donne 0 désync et 0 composant. Tout banc de `filmdec` qui juge une hypothèse de grammaire sur le
seul compte de désyncs peut donc conclure à l'envers. **Règle à retenir** : un banc de
calibration publie le nombre de composants décodés à côté du compte de désyncs, et ne compte pas
un succès sans composant. À vérifier au passage dans les bancs existants
(`keyframe_fullstate_loop_test.go`, `default_state_*`, `prefix`-like) — **non traité ici**, hors
périmètre de la mesure.

### Pour `keyframe_fullstate_loop.go`

Notre modèle traite déjà la largeur d'en-tête comme une variable (`HeaderBits` 64 vs 108) mais
**une seule largeur pour tous les types**. La mesure montre qu'il faut une **table par type**
(`ti -> bits`), avec 9 → 47 et 35 → 64 comme deux premières entrées mesurées, et 64 comme
**héritage** explicitement non vérifié pour les 48 autres types — pas comme un défaut légitime.

## Coût estimé d'une exploitation

| Étape | Coût | Note |
|---|---|---|
| Table `keyframeHeaderByTI` + routage dans le lecteur de records | petit (≈ 40 L + tests) | Aucune valeur de production ne change tant que seuls 9 et 35 y figurent (35 = le défaut actuel) |
| Capture du désignateur (`captureNames` + `decodeManagedPlayerTeamDesignator`) | petit (≈ 20 L) | Sous le garde-fou de parité de bits existant |
| **Lien entité → joueur** | **inconnu, bloquant** | Les deux pistes gratuites sont mortes (volets 3 et 3 bis). Il faut une retro-ingénierie du bloc joueur `ti=9`/`ti=47`, ou le pont `statborg-entry-index-and-type-component` |
| Gain utilisateur une fois le lien résolu | **nul à court terme** | La base porte déjà l'équipe, avec le gamertag. Le gain réel serait sur les films **sans** match en base, cas qui n'existe pas dans notre produit |

**Recommandation** : ne pas exploiter. Garder la mesure comme acquis (l'en-tête est par type,
c'est la raison des lectures vides sur respawn timers / vies / statborg), et rouvrir le sujet
seulement si un autre chantier a besoin d'un type d'entité dont l'en-tête reste à calibrer — le
balayage de préfixes est alors l'outil, et le critère « au moins un composant décodé » sa
garde.

## Bancs ajoutés

| Fichier | Ce qu'il mesure | Garde |
|---|---|---|
| `filmdec/mesure_entete_ti9_test.go` | Tableaux A (marche par en-tête × type), B, C ; balayage D | `MESURE_TI9_ROOT` + `MESURE_TI9_IDS` ; balayage : `MESURE_TI9_BALAYAGE` |
| `filmdec/mesure_entete_ti9_identite_test.go` | Champs de l'état par défaut ti=9 (E) et voisinage de slots (F) | idem |
| `filmdec/mesure_entete_ti9_lien_test.go` | Index de joueur du film confrontés aux équipes de la base (G) | + `MESURE_TI9_ROSTER` |

Sans films, les trois se sautent proprement. Aucun ne lit l'installation du jeu — pas de tag
`gamefiles`.

Invocation complète :

```bash
cd apps/go-api
MESURE_TI9_ROOT=<depot>/data/cache/film_chunks \
MESURE_TI9_IDS=4a93f0e2,8b512df2,ce083875 \
MESURE_TI9_BALAYAGE=1 \
MESURE_TI9_ROSTER='4a93f0e2:<xuid>/<team>,...;8b512df2:...;ce083875:...' \
  go test ./internal/analysis/filmdec/ -run Mesure -v -timeout 60m
```

Gate passé : `go test ./internal/analysis/filmdec/` et `go vet ./internal/analysis/filmdec/`.
