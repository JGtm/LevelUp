# Instruction des deux pertes de DURÉE du corpus témoin — 2026-09-06

Branche `feat/v2-durees`, worktree `LevelUp-wt-v2-durees`, base `88e253e06` (= `feat/v2-corpus`,
d'où viennent l'axe « somme des durées » et le gate). Films comparés à `a059caefc` (`feat/v75`,
schéma 43) puis au HEAD corrigé (schéma 45). Grille de rejeu : 100 ms par frame sur les deux
films — une frame = 0,1 s partout ci-dessous.

Source : `.ai/V7.5/v2/CORPUS_TEMOIN_2026-09-06.md` §3.3 (« faits nouveaux ») et les deux entrées
correspondantes de `.ai/V7.5/REGISTRE_REPORTS.md`. Les deux faits ont été détectés PAR L'AXE DES
DURÉES et par lui seul : aucun comptage d'éléments ne pouvait les voir.

---

## Verdict en une page

| Fait | Perte annoncée | Verdict | Cause |
|---|---|---|---|
| **(A)** `bcb6d393` — durée de port de drapeau | −76 et −10 frames sur 2 joueurs (−8,6 s) | **RÉGRESSION**, corrigée | Une vie ANONYME lue comme une ABSENCE : `tracksByXUID` n'indexait que les pistes nommées, 9 prises sur 16 sortaient `NoTrack` |
| **(B)** `084a804d` — durée d'épisodes d'équipement | −68 frames (−6,8 s), le NOMBRE d'épisodes inchangé | **RÉGRESSION**, corrigée | Un épisode à cheval sur deux vies d'un même slot était borné à UNE d'elles — son instant d'ACTIVATION passait à la trappe |

**Les deux faits ont la même racine** : le découpage « une track = une vie » du **schéma 36**
(`48cf4905d`, 2026-09-02), dont trois consommateurs avaient déjà été rattrapés le 2026-09-06
(`79bf2e6d2`, schéma 41) et un quatrième au schéma 43 (portages de crâne). Ceux-ci sont les
cinquième et sixième — et les premiers que seule une mesure de DURÉE pouvait révéler, parce que
le compte d'éléments y est soit compensé (A : 16 prises annoncées, 7 publiées, mais le gate
n'aurait vu que `carries`), soit strictement inchangé (B : 21 épisodes des deux côtés).

Hypothèse de départ du fait (A) — « une ré-attribution entre joueurs, total d'équipe inchangé » —
**REFUTÉE** : le total d'équipe baissait lui aussi (871 → 862 sur un drapeau, 77 → 0 sur l'autre).
Hypothèses de départ du fait (B) — « assainissement de bornes sur points aberrants » ou « gate
ajouté récemment » — **REFUTÉES toutes les deux** (§2.3).

Schéma bumpé **43 → 45**. **44 est SAUTÉ et RÉSERVÉ** au lot des manches, en cours sur une autre
branche : deux chantiers parallèles ne peuvent pas revendiquer le même numéro (même règle qu'au
v42). À l'heure du commit, `feat/v75` ne porte pas encore le 44 — le numéro est néanmoins laissé
libre, conformément à la consigne.

---

## Méthode

Cuisson par le gate lui-même (`cmd/replay-corpus-gate`), qui est le chemin de production
(`replaybuild.NewBuilder` + `BuildMatch`) sous les mêmes protections que `cmd/replay-build` :
verrou d'exclusion `filmproc.AcquireSolo` sur le cache du PARC, plafond mémoire 3 Gio, priorité
basse, un film à la fois. Verrou inter-agents (`mkdir`/`rmdir` sur le scratchpad) posé autour de
chaque exécution.

Le parc n'a reçu AUCUNE écriture : la racine de travail du gate est une COPIE physique (chunks +
manifeste depuis le parc, catalogues versionnés depuis le checkout testé), pas une jonction —
l'artefact frais s'écrit sous un chemin qui n'existe pas dans le parc. Faits du match exportés en
lecture seule par `levelup replay-facts-export` (`OpenReadForQuery`).

Trois exécutions : (1) manifeste restreint aux deux témoins au HEAD `a059caefc` pour reproduire
le fait, (2) même manifeste après correctif, (3) **manifeste complet des 7 témoins** au HEAD
corrigé, comme contrôle de non-régression sur les cinq autres familles (§4).

---

## Fait (A) — `bcb6d393`, la durée de port de drapeau

### A.1 Ce qui était mesuré

| | artefact du parc (s20) | HEAD 43 | HEAD 45 (corrigé) |
|---|---|---|---|
| `coverage.flagCarries.carries` / `.closed` | **16 / 16** | 7 / 7 | **16 / 16** |
| `coverage.flagCarries.noTrack` | 0 | **9** | 0 |
| `markerObserved` / `markerConfirmed` | 6 / 6 | 5 / 5 | 6 / 6 |
| `dropsRepositioned` | 4 | 1 | 4 |
| spans publiés (tous états) | 34 | 17 | 34 |

**Durée de portage par joueur** (frames ; ×0,1 s) :

| xuid | parc s20 | HEAD 43 | écart | HEAD 45 |
|---|---|---|---|---|
| `2533274823110022` | 441 (3 portages) | 441 (3) | — | **441 (3)** |
| `2533274858283686` | 358 (2) | **282 (2)** | **−76** | **358 (2)** |
| `2535429985869093` | 96 (10) | **86 (1)** | **−10** | **96 (10)** |
| `2535469190789936` | 53 (1) | 53 (1) | — | **53 (1)** |
| **total** | **948** | **862** | **−86** | **948** |

**Par drapeau** :

| drapeau | parc s20 | HEAD 43 | HEAD 45 |
|---|---|---|---|
| équipe 1 | 871 (15 portages) | 862 (7) | **871 (15)** |
| équipe 0 | 77 (1 portage) | **0 (0)** | **77 (1)** |

### A.2 La cause, sur pièces

Le porteur `2535429985869093` occupe le **slot de bipède 536**. Au parc, ce slot porte UNE piste
`[1370..3464]`, 1 943 points, nommée. Au HEAD, il en porte **deux** :

| | intervalle | points | identité |
|---|---|---|---|
| vie 1 | `[1370..2736]` | 1 284 | **`2535429985869093`** (nommée par la mort qui la termine) |
| vie 2 | `[2795..3464]` | 659 | **ANONYME** |

1 284 + 659 = **1 943** : la lecture du film n'a rien perdu, seule la SEGMENTATION a changé
(trou de réplication de 58 frames = 5,8 s, au-dessus de `lifeGapUS` = 5 s).

Ses **neuf** dernières prises — frames 3129, 3152, 3169, 3183, 3238, 3264, 3337, 3356, 3370 —
tombent toutes dans la vie 2. `attachFlagCarryPositions` cherche la position du porteur via
`tracksByXUID`, qui n'indexait que les pistes dont `XUID != ""` : aucun point nommé à ces
instants, donc `NoTrack++` et le portage disparaît. Ce sont exactement les 9 de
`carries 16 -> 7`, et exactement les 9 frames manquantes du joueur (−9 sur les −10).

**C'est le même défaut que le gate des portages de crâne, corrigé au schéma 43** : « aucune vie
NOMMÉE ne couvre l'intervalle » y était lu « le porteur est ABSENT », alors qu'une vie sans nom
est une **présence sans identité publiée**.

**Cascade mesurée sur un dixième portage.** L'élagage de ces 9 portages bruts déplaçait aussi
l'attribution de drapeau du portage `[1675..1751]` de `2533274858283686` : du drapeau de
l'équipe 0 (parc, et HEAD corrigé) vers celui de l'équipe 1, où la fermeture du portage
précédent le tronquait à UNE frame et posait un `dropped` de 1676 à 1751 devant une rentrée
`home` à 1752 — le document se contredisait alors lui-même. Le mécanisme exact à l'intérieur
d'`assignFlags` n'a PAS été isolé (il n'est pas nécessaire au verdict) ; le fait est mesuré des
deux côtés, et il disparaît avec le correctif. C'est lui qui explique les −76 frames restantes.

### A.3 Le correctif

`tracksByXUID` accepte désormais une piste ANONYME quand le **pont canonique** `slotXUID`
(`ResolveSlotXUID` / `OwnerReport`) nomme son slot. Ce n'est pas une déduction locale : c'est le
MÊME pont qui nomme déjà les marques de portage (`flag_carries_marker.go`), les ramassages
(`document_pickups.go`) et les frags sous équipement actif, et sa règle de collision refuse déjà
un slot que deux joueurs se partagent. Une piste anonyme dont le pont ne nomme pas le slot **reste
écartée** : on n'invente aucun porteur.

Les deux appelants de l'index en profitent — `attachFlagCarryPositions` (position de prise et de
lâcher) et `closeByFreeLives` (lâcher volontaire daté par la vie libre de l'objet). Les deux
helpers sortent dans `flag_carrier_tracks.go` : `flag_carries.go` franchissait les 500 lignes,
c'est un déplacement PUR.

### A.4 Tests, prouvés par mutation

- `TestFlagCarriesVieAnonymeNEstPasUneAbsence` — une prise que seule la vie ANONYME du porteur
  recouvre, avec le pont qui nomme son slot : 1 portage publié, `NoTrack = 0`, position lue sur
  la vie anonyme. **Rouge sans le correctif** (`Carries:0 NoTrack:1`).
- `TestFlagCarriesVieAnonymeSansPontResteEcartee` — la **CONTRE-ÉPREUVE**, trois sous-cas : pont
  muet, pont sur un autre slot, pont sur un autre nom. La prise reste `NoTrack` dans les trois.
  Le correctif RÉTRÉCIT le rejet, il ne le supprime pas.

---

## Fait (B) — `084a804d`, la durée des épisodes d'équipement

### B.1 Ce qui était mesuré

| | parc s20 | HEAD 43 | HEAD 45 |
|---|---|---|---|
| `equipmentEpisodes/n` | 21 | **21** | 21 |
| `equipmentEpisodes/duree-totale` | 3 697 | **3 629** | **3 697** |
| slot 620, épisode `camo` | `[3105..3672]` — **568 frames (56,8 s)** | `[3173..3672]` — **500 (50,0 s)** | `[3105..3672]` — **568** |

Le compte ne bouge pas, la durée baisse de **68 frames (6,8 s)** : le rognage que l'axe des durées
existe pour attraper. Aucun autre épisode du film ne bouge (diff élément par élément des 21
épisodes : ensemble vide dans les deux sens après correctif).

### B.2 La cause, sur pièces

Le slot 620 porte au parc UNE piste `[2842..3732]`, 835 points. Au HEAD il en porte **deux** :

| | intervalle | points | identité |
|---|---|---|---|
| vie 1 | `[2842..3120]` | 275 | **ANONYME** |
| vie 2 | `[3173..3732]` | 560 | `2533274806581989` |

275 + 560 = **835** — **le même nuage de points, au point près**. Seule la segmentation change,
sur un trou de réplication de 52 frames (5,2 s, au-dessus de `lifeGapUS`).

Les deux bornes de l'épisode sont des **transitions LUES** sur l'interrupteur i28 : 4095 à la
frame 3105, retour à 0 à la frame 3672 (`endRead: true`). `close` appelait `windowFor`, qui rend
la vie de **recouvrement MAXIMAL** — 16 frames pour la vie 1 contre 500 pour la vie 2 — puis
clampe : `t0` remonté de 3105 à 3173. Les 68 frames perdues comprennent **16 frames à
l'INTÉRIEUR d'une vie publiée** (3105→3120) et 52 dans le trou.

**Ce n'est pas une mort, et trois lectures indépendantes le disent :**

1. **La vie 1 est ANONYME.** `nameLivesByDeaths` nomme une vie par la mort qui la TERMINE ; rien
   n'a nommé `[2842..3120]`. Si le joueur était mort à 3120, cette vie porterait son nom.
2. **Le corps n'a pas bougé** : `(-26,92 · -5,48 · 94,77)` à la frame 3120,
   `(-27,45 · -5,60 · 94,71)` à la frame 3173 — **0,55 unité en 5,3 s**. Une réapparition
   téléporte le joueur à un point de spawn.
3. **Le canal du camouflage est CONTINU** : aucune lecture i28 à 0 entre 3105 et 3672 (sinon la
   machine à états aurait fermé l'épisode là). Une mort remet le camouflage à zéro.

**Le camouflage est même la CAUSE du trou** : un porteur invisible et immobile cesse d'être
répliqué. Borner la mesure à la vie que ce silence a découpée, c'est laisser l'effet effacer sa
propre trace.

### B.3 Les deux hypothèses de départ, réfutées

- **« Assainissement de bornes sur points aberrants »** (comme les faits `2cf24f30` et
  `4f77afc1`) : NON. Le slot 620 porte **exactement les mêmes 835 points** des deux côtés — aucun
  n'a été supprimé. Les 9 points que le film perd au total (111 956 → 111 947) et les bornes de
  scène assainies (`maxX` 202,97 → 43,03, `maxZ` 205,81 → 115,27) concernent le **slot 539**, et
  c'est le **fait n° 5 de `INSTRUCTION_RESIDUS_2026-09-06.md`**, déjà instruit et clos (« ancien
  artefact FAUX »). La perte de vie nommée `tracks/vies-par-xuid/2533274806581989` 6 → 5 que le
  gate signale sur le même match est CE fait-là (slot 539, piste `[559..2739]` réduite à 50
  points par un point à `z = -370`), pas celui-ci : sur le slot 620, le même joueur **garde** sa
  vie nommée.
- **« Un gate ajouté récemment »** (`git log -S` sur les fichiers d'équipement depuis le
  2026-08-25) : NON. Trois commits seulement touchent `equipment_episodes.go` /
  `document_equipment_changes.go` / `equipment_episode_kills.go` depuis cette date —
  `fa09f4ee5` (03/09, lecture d'usage), `79bf2e6d2` et `13c0336b6` (06/09). Aucun n'ajoute de
  porte. `79bf2e6d2` est au contraire le correctif PARTIEL du même défaut : il a rendu au film
  ses 2 épisodes de camouflage perdus (19 → 21 sur ce match), **sans traiter la durée** d'un
  épisode à cheval.

### B.4 Le correctif

`close` borne désormais à l'**UNION** des vies du slot que l'intervalle mesuré recouvre
(`spanFor`), au lieu de la seule vie de recouvrement maximal. La règle de rejet est conservée
telle quelle : un épisode qu'AUCUNE vie publiée ne recouvre reste écarté (il n'a aucune fiche où
s'afficher), et l'union ne dépasse jamais l'intervalle mesuré. `windowFor` reste en service pour
ses deux autres appelants — `finish` (la vie qui contient l'OUVERTURE, dont la fin date la mort)
et `equipmentCoverage` (la clé de vie du dénominateur) — dont le comportement ne change pas.

Ce que le rognage coûtait au produit, au-delà du chiffre : `replaySound.ts` sonne l'activation à
**chaque `t0`** d'épisode — le camouflage de ce joueur s'annonçait **6,8 s en retard** —, et
`equipmentFx.ts` n'appliquait l'effet de fiche qu'à partir de 3173, alors que la vie 1, publiée,
couvre 3105→3120.

### B.5 Test, prouvé par mutation

`TestEpisodeAChevalSurDeuxViesGardeSesBornesMesurees` — deux vies séparées d'un trou plus une
troisième vie NON recouverte (garde contre une union naïve de toutes les fenêtres du slot) ;
l'épisode doit sortir `[45..250]`. **Rouge sans le correctif** : il rend `[60..250]`, exactement
le symptôme mesuré sur `084a804d`.

---

## Contrôle : les témoins re-cuits

Corpus complet re-cuit au HEAD corrigé (schéma 45), comparé au même parc que l'exécution de
référence de `CORPUS_TEMOIN_2026-09-06.md` §2 :

| témoin | famille | gains (réf → corrigé) | pertes (réf → corrigé) |
|---|---|---|---|
| `bcb6d393` | ctf_mono_manche | 205 → **211** | 27 → **9** |
| `fb1a1a72` | ctf_multi_manche | 27 → 27 | 2 → 2 |
| `d9781168` | oddball | 176 → 176 | 6 → 6 |
| `c75f33b8` | assaut_bombe | 168 → 168 | 8 → 8 |
| `bf15f7ab` | slayer | 41 → 41 | 2 → 2 |
| `51ebbc0f` | deux_manches | 184 → 184 | 9 → 9 |
| `084a804d` | vehicules | 482 → 482 | 21 → **19** |
| **total** | | | **75 → 55** |

**Les cinq familles non concernées sont identiques au chiffre près**, lignes de perte comprises —
« identique hors numéro de schéma ». Sur les deux témoins corrigés, `flagCarries` et
`equipmentEpisodes` redeviennent **égaux à l'artefact du parc, élément par élément** (per-xuid,
per-équipe, per-slot, et diff d'ensemble vide sur les 21 épisodes).

Les 20 lignes fermées dépassent les 7 pertes brutes que `CORPUS_TEMOIN` §3.3 attribuait à ces
deux faits : la régression des drapeaux traînait avec elle `spans.*/presents`,
`spans/par-state/*`, `markerObserved`, `markerConfirmed` et `dropsRepositioned`, comptés ailleurs
dans le décompte de la première exécution.

Ce qui reste en perte sur les deux témoins est déjà expliqué par la chronique : reclassement de
la famille `other` des poses d'équipement, ré-attribution de 3 actions `kills` dans un gain
(`BALAYAGE_PARC` §6.3, entrée `bcb6d393`), bornes de scène assainies, `coverage.shots.noSlot` en
baisse (amélioration), fait n° 5 des résidus (slot 539), et `flagCarries.spans/n 28 -> 1` — le
**bug de mesure préexistant** de `mesurerTableau`, déjà consigné au registre et **non traité ici**
(règle du zéro fix hors périmètre).

---

## Gates joués

```
cd apps/go-api
go test -count=1 ./internal/analysis/replay/... ./internal/replaybuild/... \
        ./internal/replaydiff/... ./internal/archlint/... ./contracttest/...   # ok
go test -count=1 -tags=integration -p 1 ./internal/api/wire/...                # ok (code de cuisson touché)
go build ./...                                                                 # ok (CGO_ENABLED=1)
golangci-lint run --new-from-merge-base=origin/main ./...                      # 0 issues
```

Golden d'assemblage régénéré : **unique écart = la ligne de version** (`schema 43` → `schema 45`,
1 ligne sur 606, vérifié par `git diff` après régénération) — le film de référence `000d5950`
n'est touché par aucun des deux correctifs. Le contrat OpenAPI déclare `schemaVersion` sans
`enum`/`const`/`default` : un bump ne le déplace pas ; aucune constante de schéma côté web.

---

## Découvertes, notées et NON traitées

1. **L'attribution du drapeau à une ÉQUIPE ne se recoupe pas avec la feuille de match.** Sur
   `bcb6d393` (parc comme HEAD corrigé — ce n'est donc PAS une régression), les quatre porteurs
   sont tous `teamId = 0` d'après les faits du match, et le score d'équipe est 3–0 ; or 15
   portages sur 16 sont posés sur le drapeau étiqueté « équipe 1 » et le seizième sur celui
   étiqueté « équipe 0 », qui est en outre le seul à connaître une capture. Un joueur ne porte
   jamais son propre drapeau : l'une des deux affectations est fausse quelle que soit la
   convention. `assignFlags` attribue par GÉOMÉTRIE (socle le plus proche / drapeau lâché le plus
   proche) et l'étiquette d'équipe vient du catalogue de carte, dont la numérotation n'est PAS
   prouvée coïncider avec celle de la feuille de match. Hors périmètre de ces deux faits de durée.
2. **Le mécanisme interne d'`assignFlags` qui déplaçait le portage `[1675..1751]` d'un drapeau à
   l'autre** (§A.2) n'a pas été isolé : mesuré des deux côtés, il disparaît avec le correctif, et
   l'isoler aurait demandé d'instrumenter le calque hors périmètre.
3. **`flagCarries.spans/n 28 -> 1` sur `084a804d`** reste au rapport : c'est le bug de mesure
   PRÉEXISTANT de `mesurerTableau` (`e.num` au lieu de `e.incr`), déjà au registre depuis la
   construction du gate. Non traité ici.
4. **Les autres consommateurs de « une track = une vie » n'ont pas été balayés.** Six ont été
   rattrapés à ce jour (pistes, grappin, épisodes-compte au schéma 41 ; portages de crâne et de
   bombe au 43 ; épisodes-durée et drapeaux ici). Aucun inventaire systématique des lecteurs qui
   supposent encore « un slot = une piste nommée » n'existe — un balayage `grep` des index par
   `XUID != ""` serait le point de départ.
