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
(`document_pickups.go`) et les frags sous équipement actif. Une piste anonyme dont le pont ne
nomme pas le slot **reste écartée** : on n'invente aucun porteur.

> **CORRIGÉ LE 2026-09-06 (revue DUREES-R1, constat C1).** Cette section a d'abord ajouté, à tort,
> que « sa règle de collision refuse déjà un slot que deux joueurs se partagent ». **C'est FAUX sur
> pièces** : `ownersFromLives` (`lives.go:229-232`) compte la collision puis `continue`, et le
> PREMIER nommé reste publié dans `SlotXUID` — choisi par l'ordre des vies, pas par la proximité
> temporelle (`owners.go` le dit : « première nommée, collisions comptées »). De surcroît
> `SlotCollisions` ne voit que les conflits entre vies NOMMÉES : une vie anonyme y est invisible.
> La garde manquante est posée en §R1.C1 ci-dessous.

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

---

## Corrections R1 — les trois constats de la revue adversariale

Revue `DUREES-R1` (contexte frais, HEAD `7e5c454bc`, 22/22 conditions tenues) : **les deux
correctifs sont exacts, additifs et prouvés**, aucun ne remet en cause la livraison. Trois
constats à traiter, tous traités ci-dessous, périmètre strict.

### R1.C1 (MOYENNE) — la garde invoquée pour autoriser le repli n'existait pas

**Le constat, et il est juste.** Le commentaire de `flag_carrier_tracks.go` — et §A.3 de ce
journal, mot pour mot — justifiaient le repli par « la règle de collision du pont refuse déjà un
slot que deux joueurs se partagent ». `lives.go:229-232` dit le contraire : la collision est
COMPTÉE puis `continue`, et `out[l.slot]` / `byXUID[l.slot]` **conservent le PREMIER nommé**. Le
slot reste donc publié dans `SlotXUID` — nommé par l'ordre des vies, jamais par la proximité
temporelle (`owners.go` l'écrit : « première nommée, collisions comptées »). Faiblesse
redoublée : `SlotCollisions` ne compte que les conflits entre vies **NOMMÉES**, or une vie
ANONYME — la population exacte que le repli croit — y est **invisible**.

**Déclenchement, mesuré au parc** : `084a804d` slot 734 — `[5872..6981]` nommée A,
`[7123..7158]` **ANONYME**, `[7457..7591]` nommée B ; 9 artefacts sur 106 portent au moins un slot
en collision. Non matérialisé sur ce corpus (aucune prise ne tombe dans ces 36 frames), mais
c'est un résultat **FAUX** là où l'ancien code rendait un résultat **ABSENT**.

**Le correctif.** La garde est posée là où la matière existe — sur les vies PUBLIÉES, seules à
dire qui a occupé le slot et quand. Le repli est REFUSÉ dès que les vies nommées du slot ne
s'accordent pas avec le pont :

| cas | verdict |
|---|---|
| deux xuid nommés distincts sur le slot | **refus** — la vie anonyme peut être de l'un ou de l'autre |
| le pont nomme un joueur que les vies nommées du slot ne portent pas | **refus** — pont et document se contredisent, on ne choisit pas |
| aucune vie nommée sur le slot | accepté — rien ne contredit le pont |

Le refus est **compté** (`coverage.flagCarries.ambiguousSlot`, un compteur de SLOTS, hors de
`Balanced()` puisqu'il ne partitionne pas les prises) et **journalisé** (`slog.Warn` structuré,
liste des slots). Publié parce que sans lui, un portage manquant faute d'identité disponible
serait indistinguable d'un portage qui n'a jamais eu lieu.

**Ce que la garde ne peut PAS attraper, et c'est écrit dans le code pour que personne ne le
croie** : un slot occupé par A (vie nommée) puis par B dont AUCUNE vie n'est nommée sort avec un
seul xuid nommé, et la vie de B est rangée sous A. Le document publié ne porte rien qui distingue
ce cas d'une vie de A coupée par un trou de réplication ; le trancher demanderait de DATER le
pont, ce qu'`OwnerReport` ne fait pas.

**Tests, prouvés par mutation** — `TestFlagCarriesSlotPartageRefuseLeRepli` (slot A / anonyme / B,
pont nommant A : 0 portage, `noTrack = 1`, `ambiguousSlot = 1`). **Rouge sans la garde** :
`Carries:1 NoTrack:0 AmbiguousSlot:0`. Contre-épreuve
`TestFlagCarriesSlotNonPartageAccepteEtNeCompteRien` (A / anonyme seuls : 1 portage, 0 refus).

### R1.C2 (FAIBLE) — l'union ne franchit plus une mort

**Le constat.** `spanFor` ne recevait aucune information de mort : l'union unissait toutes les
vies recouvertes, qu'une mort les sépare ou non. Sonde de la revue : trois vies NOMMÉES
`[0..50]`, `[60..300]`, `[400..500]`, camo actif à 20 et inactif à 450 → épisode `[20..450]`,
**enjambant deux morts**. `equipmentFx.ts` aurait peint l'effet sur des vies où rien ne l'a lu.
L'invariant ne tenait que par la mesure (un seul épisode franchit une frontière de vie sur les
trois films recuits, et c'est la frontière ANONYME visée), pas par construction.

**Le correctif.** `trackFrameWindows` rend désormais des `lifeWindow` **triées** portant
`named` — la vie porte-t-elle une identité ? L'identité d'une vie vient de la mort qui la
TERMINE (`nameLivesByDeaths`) ; une vie ANONYME est au contraire une vie coupée par un trou de
réplication. `spanFor` part de l'ancre (la vie qui contient l'ouverture) et n'étend l'union
qu'à travers les frontières qu'**aucune identité ne date**.

**La lecture est CONSERVATRICE, et c'est assumé** : les fermetures nomment aussi des vies
(`nameClosedLives`) sans qu'une mort les termine, si bien qu'une couture légitime peut être
refusée — jamais l'inverse. Un épisode trop court est une mesure incomplète ; un épisode qui
enjambe une mort est une mesure FAUSSE.

**Tests, prouvés par mutation** — `TestEpisodeNEnjambePasUneMort` (trois vies nommées, activation
dans la PREMIÈRE : attendu `[20..50]`) et `TestEpisodeFranchitUnTrouAnonymeMaisPasLaMortSuivante`
(vie ANONYME puis vie NOMMÉE : la couture traverse le trou et s'arrête à la mort — `[45..300]`).
**Rouges sans la borne** : `[20..450]` et `[45..450]`, exactement la valeur de la sonde de la
revue. La mutation prescrite au premier correctif n'exerçait pas ce risque (activation dans la
SECONDE vie, où le clamp ne peut que rétrécir).

### R1.C3 (FAIBLE) — le contrôle indépendant réaligné sur la production

**Le constat.** `drapeau_objet_controle_test.go` passait `nil` en guise de pont : depuis le
correctif, la production était strictement plus large que son contrôle. Pire, un porteur dont
toutes les vies sont anonymes produisait `objDrapeauRef{porteur: nil, x: 0, y: 0}`, que
`objDrapeauPres` lit comme un **SOCLE fantôme à l'origine du monde** — toute création à moins de
1,5 m de (0,0) aurait été comptée « née à un socle ». Le contrôle se serait dégradé en silence,
dans le sens qui l'assouplit. Test opt-in, donc dormant.

**Le correctif, ses deux moitiés.** (a) Le pont est RECONSTRUIT depuis les seules vies publiées
(`objDrapeauPontDuDocument` : un slot dont les vies nommées désignent un seul joueur), ce qui
applique au contrôle exactement la garde de la production ; (b) un portage dont le porteur n'a
malgré tout aucune piste est **sauté**, jamais transformé en référence vide — la position par
défaut (0,0) n'est celle d'aucun socle.

### Sortie des témoins, et pourquoi le schéma reste 45

Re-cuisson des deux témoins après R1, comparée à la re-cuisson d'avant R1 (mêmes racines, même
parc, même exécuteur) :

| témoin | gains | pertes | lignes de perte |
|---|---|---|---|
| `bcb6d393` | 211 → **211** | 9 → **9** | **identiques, ligne pour ligne** |
| `084a804d` | 482 → **483** | 19 → **19** | **identiques, ligne pour ligne** |

**Aucune valeur existante ne bouge, aucune perte n'apparaît.** Le seul écart est la mesure NEUVE
demandée par C1 : `coverage.flagCarries.ambiguousSlot` — **1 sur `084a804d`** (le slot 734
exactement, celui que la revue avait mesuré : la garde se déclenche là où elle devait, et le
calque n'en perd rien) et **0 sur `bcb6d393`**.

Substance vérifiée à l'identique : `084a804d` 21 épisodes / 3 697 frames, slot 620 `[3105..3672]` ;
`bcb6d393` `carries` 16, `noTrack` 0, durées par joueur 441 / 358 / 96 / 53 (total 948).

**`SchemaVersion` reste donc 45.** Le contenu des calques est strictement inchangé, et le ratchet
l'écrit lui-même : « un champ optionnel de plus n'en est pas une [raison] ». Golden d'assemblage
inchangé.

### Gates rejoués après R1

```
go test -count=1 ./internal/analysis/replay/... ./internal/replaybuild/... \
        ./internal/replaydiff/... ./internal/archlint/... ./contracttest/...   # ok
go test -count=1 -tags=integration -p 1 ./internal/api/wire/...                # ok (49,9 s)
go build ./...                                                                 # ok
golangci-lint run --new-from-merge-base=origin/main ./...                      # 0 issues
```

Seuils : `equipment_episodes.go` 442 L, `flag_carries.go` 455 L, `flag_carrier_tracks.go` 148 L,
`flag_objects.go` 426 L, `document_objectives_live.go` 320 L — tous sous 500.

### Découverte R1, notée et NON traitée

`OwnerReport.SlotXUID` publie un slot en collision sous le nom de son PREMIER occupant nommé,
sans que rien à la lecture ne dise que ce slot est disputé (`SlotCollisions` est un compteur
global, pas un marqueur par slot). La garde ci-dessus le contourne pour le seul calque drapeau,
en relisant les vies publiées ; **les autres consommateurs du pont** (marques de portage,
ramassages, frags sous équipement actif) n'ont pas été inventoriés. Un pont qui MARQUERAIT ses
slots disputés — ou qui les daterait — fermerait la question à la source pour tous.
