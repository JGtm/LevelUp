# Instruction « CTF drapeaux » — instruction, cause, correctif (2026-09-06)

Branche `feat/v2-ctf-drapeaux`, worktree `LevelUp-wt-v2-ctf`, base `a21fd77f4` (= `feat/v75`).

---

## ERRATUM du 2026-09-06 (revue adversariale CTF-R1) — À LIRE AVANT LE RESTE

Le corps de ce journal (§0 à §8) est conservé **tel qu'écrit** ; trois de ses affirmations sont
**infirmées** par la revue, sur pièces. Ne pas se fier aux passages listés ci-dessous sans lire
la correction.

**E1. « La seule cuisson réelle de la CI décodait sans registre » est FAUX** (touche §0, §4.2,
§5, §7 et le message du commit `086a15f62`). Le téléchargeur de l'ouvrier **pèle déjà une couche
zlib** (`cmd/replay-worker/job.go`, `downloadChunk`) et `filmsource.Load` pèle la seconde : les
deux couches du fixture étaient absorbées par **deux étages différents**, et le registre arrivait
INTACT. Mesure de la revue : fixture d'origine (deux couches) remis au HEAD, l'épreuve E2E est
**verte**, assertions de valeur comprises, artefact de **283 260 octets — exactement la taille
obtenue avec le fixture corrigé**. Les « onze calques tombés », les « quatre jours de CI verte à
tort » et la phrase « l'artefact passe de 262 535 à 283 260 octets » viennent de mon outil
jetable, qui lisait `testdata/.../chunk_00.bin` **en direct** — un chemin qu'aucun test ni aucun
code de production n'emprunte. **Aucun sinistre n'a eu lieu.**

**Ce qui reste vrai et pourquoi le correctif reste bon** : les deux couches sont avérées, le
fixture corrigé est octet pour octet celui du cache, `c17f4941f` a bien retiré le repli zlib, la
détection RFC 1950 est sans faux positif (0 sur 1 378 registres du cache), et un tampon compressé
rendait bien un **registre vide en silence**. Un fixture doit dire la vérité sur ce que le CDN
sert, et un refus explicite vaut mieux qu'un registre vide muet — le défaut était **latent**, il
est désormais fermé.

**E2. `objectLives` n'est pas « la signature du registre ECS »** (touche §5 et le commentaire de
`assertCalquesDObjectif`) : elle vaut 4 même avec le fixture mal généré, puisque la chaîne E2E
pelait les deux couches. C'est une épingle de caractérisation. Le seul garde-rail qui attrape
réellement le double pelage est `film_fixture_integrite_cgo_test.go`. Justification corrigée dans
le code.

**E3. « Le porteur de drapeau n'est structurellement pas pontable sur ce film » est CONTREDIT par
le parc** (touche §4.4, §8.1, et les épingles 8/4/12 du test E2E). L'artefact du parc au schéma 20
**nomme** les actions de drapeau de SweatyYeti75. Le calque `objectives` a donc DÉRIVÉ entre le
schéma 20 et aujourd'hui, et cette dérive est instruite au **§9 ci-dessous**. Les épingles 8/4/12
et la liste des joueurs pontés ont été **retirées** du test E2E : on ne sanctuarise pas un état
non compris. L'oracle des captures, qui comparait `TeamScores[0]` seul alors que
`coverage.flagCarries.captures` compte les DEUX camps, compare désormais la **somme** (constat 4
de la revue).

**E4. Deux affirmations de commentaire démenties par la mesure**, corrigées dans le code : le
premier octet d'un registre inflaté n'est `0x29` que sur 1 117 des 1 378 films du cache (204 en
`0x28`, qui passent la condition CM=8 — seule la somme de contrôle les écarte) ; et ce fixture
n'est pas « le seul film que la CI décode » (`TestGoldenMiniBobine` et `TestEquivalenceMiniFilm`
décodent des films réels versionnés, sans condition).

Les découvertes du §8 sont désormais portées au registre du chantier
(`.ai/V7.5/REGISTRE_REPORTS.md`), ce qui n'avait pas été fait.

---

## 0. Verdict en une ligne

**La régression annoncée n'existe pas en production.** Sur le film COMPLET du cache, la cuisson
rend EXACTEMENT la même chose avant et après le merge `736ccf3c3`. Ce qui a réellement cassé,
c'est **le fixture E2E** `film_e2e/c0a82e88` : ses morceaux 00 et 07 portent **deux couches
zlib** au lieu d'une, et depuis `c17f4941f` (2026-09-02) `ParseRegistryChunk` ne décompresse
plus — le registre ECS du fixture est donc **vide**, et onze calques du rejeu tombaient
silencieusement dans la seule cuisson réelle de la CI.

Et le « 92, famille flag » n'a jamais été le contenu du document : c'est le compteur de journal
`nommees`, qui vaut toujours 92 aujourd'hui.

## 1. Re-mesure au HEAD (étape 1)

Instrument : cuisson par le chemin de production (`replaybuild.Builder`, celui qu'appelle
`cmd/replay-worker`), outil jetable supprimé après usage.

| grandeur | fixture (HEAD, avant correctif) |
|---|---|
| variante lue | `Husky Raid:CTF` → `objectiveevents.ObjectiveTypeOf` = **`flag`** (correct) |
| `schemaVersion` | 39 |
| `flagCarries` | **0** |
| `objectiveObjects` | **0** |
| actions d'objectif | **12** — 8 `kills`, 4 `assists` |
| journal | `nommees=92 identifiees=12` |
| `coverage.flagCarries` | `bursts=3 captures=3 steals=3 openings=3 carries=0 noBridge=3 spawns=2 objectLives=0` |
| alertes | `empreinte du registre ECS INCONNUE … blocs=0 slots_non_vides=0`, puis « archétype biped 35 / arme au sol 42 / équipement 37 / véhicule 40 absent du registre » |

La reconnaissance de mode est donc HORS DE CAUSE : `ctf` est bien détecté (`extract.go`,
`classifyObjectiveMode`), et les 92 événements nommés sortent bien de la table `flag`.

## 2. Témoins (étape 2)

106 artefacts du parc local (`data/cache/replays/halo_infinite/`) relevés par schéma.

- **Aucun artefact CTF au schéma ≥ 38 dans le parc** : les deux artefacts au schéma 38 ne sont
  pas des CTF. Le parc ne peut donc PAS répondre « la perte est-elle générale ? » — il fallait
  re-cuire.
- **`c0a82e88` EST au parc, au schéma 20** :
  `flagCarries=0 objectives=17 | bursts=3 captures=3 openings=3 carries=0 noBridge=3 spawns=2`.
  `flagCarries=0` avec `noBridge=3` est donc **déjà vrai au schéma 20** — ce n'est pas une
  régression du 39.
- 27 artefacts CTF au parc (18 au schéma 20, 3 au 32, 6 au 34) ont `flagCarries` de 0 à 7 : le
  calque fonctionne ailleurs, il est muet sur CE film.

## 3. Bissection (étape 3)

Instrument : cuisson du même film, compteurs `flagCarries` / `objectives` / `viesLibres`.

| commit | schéma | film COMPLET (cache) | fixture (testdata) |
|---|---|---|---|
| `6d8c5e921` (2026-08-25, écrit la ligne « 92, famille flag ») | 18 | — | `flagCarries=0` `objectives=0` `objectLives=4` ; journal `nommees=92 identifiees=17` |
| `c17f4941f^` | 34 | — | `objectLives=4`, registre lu, aucun avertissement |
| **`c17f4941f`** | 34 | — | **registre `blocs=30 slots=0`**, tous les archétypes tombent |
| `736ccf3c3^1` (feat/v75 avant le merge) | 38 | `flagCarries=0` `objectives=12` `objectLives=4` | `objectives=0` (pied illisible), `objectLives=4` |
| `a21fd77f4` (HEAD) | 39 | `flagCarries=0` `objectives=12` `objectLives=4` | `objectives=12`, `objectLives=0`, registre vide |

**Sur le film complet, schéma 38 et schéma 39 sont strictement identiques** (`RESULT` et
`COVERAGE` comparés ligne à ligne : aucune différence).

**Commit fautif** : `c17f4941f` — *cuisson-perf(L1a) : la source unique du film*.
**Ligne** : `apps/go-api/internal/analysis/filmdec/registry.go`, `ParseRegistryChunk` — le bloc
`if len(raw) >= 2 && raw[0] == 0x78 { zlib.NewReader… }` est retiré, et le commentaire acte
« Feeding it a still-compressed buffer now yields an EMPTY registry rather than an error ».

Le retrait de l'inflate est JUSTE (la décompression se fait une fois par film, dans
`filmsource`). Ce qui ne l'est pas, c'est le silence qui l'accompagne.

## 4. Cause, sur pièces (étape 4)

### 4.1 Le fixture ne reproduit pas le CDN

Comptage des couches zlib, fixture contre cache, sur les 8 morceaux (contenu final identique
partout, sha256 égaux) :

| morceau | fixture | cache | contenu final |
|---|---|---|---|
| `chunk_00` (registre ECS) | **2 couches** (505 339 o) | 1 couche (511 264 o) | 1 973 120 o, sha `7d49876…` |
| `chunk_01` … `chunk_06` | 1 couche | 0 couche (stocké déjà décompressé) | identiques |
| `chunk_07` (pied) | **2 couches** (36 552 o) | 1 couche (36 949 o) | 189 044 o, sha `ce8b812…` |

Le fixture a été généré le 2026-08-25 en compressant chaque fichier du cache local. Ce cache est
**hétérogène** (morceaux de jeu du 14 mars stockés décompressés ; morceaux 00 et 07 re-téléchargés
le 2 juin par le téléchargeur actuel, donc stockés compressés) : compresser tout le monde a mis
**deux couches** sur les deux qui en avaient déjà une.

Jusqu'au 2026-09-02 c'était invisible : `ParseRegistryChunk` pelait la couche en trop.

### 4.2 Ce que le registre vide a emporté

Sur le fixture, au HEAD, avant correctif : biped 35, arme au sol 42, équipement 37, véhicule 40 et
l'objet d'objectif sont tous « absents du registre ». Donc **plus de** changements d'arme, poses et
charges d'équipement, camouflage, grappin, impulsions de capacité, socles d'arme, socles de
power-up, armes au sol individuelles, véhicules, vies libres de l'objet drapeau
(`objectLives : 4 → 0`), origine des ramassages. La CI restait verte parce que l'épreuve E2E
n'assertait que la FORME du document.

### 4.3 Le « 92, famille flag » n'a jamais décrit le document

Mesure au commit qui a écrit la ligne (`6d8c5e921`, 2026-08-25, schéma 18) : journal
`nommees=92 … famille=flag identifiees=17`, et **`doc.objectives` = 0** (le calque était vide en
production à cette date). Le 92 est le compteur AMONT de `objectiveevents.NamedEvents`, avant le
pont d'identité — il vaut toujours 92 au HEAD. La ligne confondait le compteur amont et le contenu
publié ; c'est cette confusion qui a fait prendre pour une régression un chiffre qui n'a jamais
été celui du document.

### 4.4 `flagCarries = 0` sur ce match n'est pas un défaut à corriger

Mesuré identique à TOUS les schémas essayés (18, 20 au parc sur le film complet, 34, 38, 39) :
3 prises, **3 `noBridge`**, 0 portage. Le slot statborg du porteur n'est pas résolu en xuid, et le
pont se tait plutôt que d'attribuer le drapeau au mauvais joueur. Le critère d'acceptation
« `flagCarries > 0` » de l'instruction **n'est donc pas atteignable** par un correctif de la
reconnaissance de mode, de la grammaire ou du registre : il demande une amélioration du pont
d'identité par manche, qui est un autre chantier.

## 5. Correctif

1. **`apps/go-api/internal/api/wire/testdata/film_e2e/c0a82e88/chunks/chunk_00.bin` et
   `chunk_07.bin`** — une couche zlib pelée. Les fichiers obtenus sont **octet à octet ceux du
   cache** (sha256 vérifiés), c'est-à-dire ce que le CDN sert.
2. **`apps/go-api/internal/analysis/filmdec/registry.go`** — `ParseRegistryChunk` REFUSE un tampon
   encore compressé (`ErrRegistryStillCompressed`) au lieu de rendre un registre vide et une erreur
   nulle. La détection lit les DEUX octets de l'en-tête RFC 1950 (méthode, dictionnaire, somme de
   contrôle) : aucun faux positif, et **aucune décompression** — le garde-rail `archlint` qui
   interdit `zlib.NewReader` dans `filmdec` reste satisfait. La sentinelle est un `const` typé et
   non un `var` : le ratchet `TestFilmdecPackageVarsNeCroitPas` vise l'état global MUTABLE, une
   constante n'en est pas.

### Tests de non-régression ajoutés

- `internal/analysis/filmdec/registry_compressed_test.go` — le refus, les quatre en-têtes zlib
  rencontrés sur les films, le piège « premier octet 0x78 mais somme fausse », et le chemin nominal.
- `internal/api/wire/film_fixture_integrite_cgo_test.go` (tag `cgo`, autonome, ~10 ms) —
  **chaque morceau du fixture porte exactement UNE couche zlib**, et le morceau 00 décompressé rend
  un registre dont l'empreinte est `KnownRegistryFingerprint` et qui porte les archétypes 35, 37,
  40, 42. *Vérifié qu'il attrape le défaut* : remis le morceau d'origine, les deux tests échouent
  avec le bon message.
- `internal/api/wire/build_queue_worker_binary_integration_test.go` — `assertCalquesDObjectif`
  fige, sur la cuisson E2E réelle : captures reconstruites = `bursts` = **3** = le score d'équipe de
  la feuille de match (oracle indépendant) ; `objectLives` = **4** (la signature du registre ECS) ;
  12 actions publiées, 8 `kills` + 4 `assists` ; et `openings == noBridge` — le silence de
  `flagCarries` est ainsi EXPLIQUÉ, et une future amélioration du pont fera tomber ce test.
- L'en-tête du même fichier est corrigé : la ligne « 92 actions d'objectif nommées (famille flag) »
  disait faux sur le document et dit désormais ce que 92 est.

## 6. Gates (tous verts, 2026-09-06)

```
go build -p 2 ./...                                                          exit 0
go test ./internal/analysis/objectiveevents/... ./internal/analysis/replay/...
        ./internal/replaybuild/... ./internal/analysis/filmdec/...            ok (×5)
go test -tags=integration -p 1 ./internal/api/wire/...                        ok 21,1 s
go test ./internal/games/halo_infinite/film/... ./internal/archlint/...
        ./internal/analysis/filmsource/...                                    ok (×7)
golangci-lint run --new-from-merge-base=origin/main ./...                     0 issues
```

`internal/sync/replaybuild` n'existe pas — le paquet est `internal/replaybuild` (déplacé avant ce
chantier) ; c'est lui qui a été passé. Les goldens inconditionnels de `analysis/replay` et de
`killsource` (dont `TestGoldenMiniBobine`) sont dans les suites ci-dessus et restent verts.

Note d'environnement : `go build ./...` a d'abord échoué en « No space left on device » (disque à
99 %). 62 dossiers `go-link-*` / `go-build*` orphelins purgés du Temp (+4 Gio), puis build en
`-p 2`. Rien à voir avec le correctif.

## 7. Périmètre d'impact sur le parc, et bump de schéma

- **Artefacts du parc concernés : AUCUN.** Le correctif de code ne change la sortie d'aucune
  cuisson valide — il ne se déclenche que sur une entrée déjà cassée (tampon compressé), qui
  produisait jusqu'ici un artefact appauvri sans le dire. Le correctif de données ne touche qu'un
  fixture de test.
- Preuve : sur le film complet `c0a82e88`, la cuisson au HEAD (schéma 39) est identique à celle
  d'avant le merge (schéma 38), `RESULT` et `COVERAGE` compris.
- **Bump de `SchemaVersion` : NON**, et c'est un « non » mesuré, pas un « non » par défaut. Un bump
  périme tout le parc pour forcer une re-cuisson ; il ne se justifie que si la sortie de production
  change. Ici elle ne change pas : aucun artefact du parc n'a été cuit avec un registre ECS vide
  (aucun film du cache n'est doublement compressé — les trois autres mini-bobines versionnées ont
  été vérifiées, et leurs goldens passent avec le refus actif). Le seul artefact qui l'était est
  celui que la CI recuisait à chaque exécution, et il est recuit correct dès ce commit.

## 8. Découvertes (notées, NON traitées — hors périmètre)

1. **`c0a82e88` : 3 prises de drapeau sur 3 sans pont.** Le pont d'identité par manche
   (`objectiveevents.ResolveRoundIdentity`) ne résout pas les slots porteurs de ce film — d'où
   `flagCarries` vide et `identiteEquipes=unresolved` (1 équipe, 5 joueurs sur 8). C'est la seule
   piste réelle pour obtenir `flagCarries > 0` sur ce match.
2. **`objectiveObjects` reste 0** même registre lu : `declares=1 vies=0` — le canal déclare un objet
   d'objectif publiable mais n'en lit aucune vie sur ce film.
3. **La cuisson-perf a prouvé « identité 9/9 »** sur ses films témoins ; aucun d'eux n'a détecté ce
   cas, parce que le défaut vit dans un FIXTURE, pas dans le corpus.
4. **Le commit `4cf807d64`** (`feat/v2-tests-ci`, lot F.1 du 2026-09-05) a **figé comme oracle les
   valeurs dégradées** du fixture cassé (`originMs`, `t0FilmMs`, roster, courbe d'équipe, 22 vies).
   Ces valeurs-là ne dépendent pas du registre et restent probablement bonnes, mais **cette branche
   doit être re-mesurée sur le fixture corrigé avant merge** — et son « écart consigné en découverte
   du lot F » (« 92 que le décodeur ne rend plus ») est à requalifier avec le §4.3 ci-dessus.
5. **Le cache film local est hétérogène** : les morceaux du 14 mars sont stockés décompressés, ceux
   de juin compressés. `filmsource.inflate` absorbe les deux, mais toute génération de fixture par
   recompression uniforme re-fabriquera le même défaut. Un générateur de fixture devrait normaliser
   à « une couche », comme le fait désormais le garde-rail.

---

## 9. La vraie instruction : où sont passées les actions `flag_captures` / `flag_steals` ? (2026-09-06)

Ouverte par le constat 3 de la revue CTF-R1. Instruite ici de bout en bout.

### 9.1 Les deux états, joueur par joueur

Artefact du parc `data/cache/replays/halo_infinite/c0a82e88.json` (schéma 20, cuit le 2026-08-26,
lecture seule) contre la cuisson du **film complet du cache** au HEAD (`cmd/replay-build`,
`--map Corpo`, faits du fixture, `LEVELUP_REPO_ROOT` sur le worktree pour ne rien écraser).

| xuid | nom | courbe de score, parc | actions, parc (schéma 20) | courbe, HEAD | actions, HEAD (avant correctif) |
|---|---|---|---|---|---|
| 2533274823110022 | JGtm | non | — | non | `kills=2 assists=1` |
| 2533275001554469 | GEK XD | oui | `kills=1 assists=2` | oui | `kills=1 assists=2` |
| 2535429692041611 | LostYeti71 | oui | `kills=1` | oui | `kills=1` |
| 2535432531943478 | TheBackwoodBoss | oui | `kills=2` | oui | `kills=2` |
| **2535463878425995** | **SweatyYeti75** | oui | **`kills=7 assists=1 flag_captures=1 flag_steals=1`** | oui | **—** |
| 2535465632069522 | DiegoGamer8K | oui | `assists=1` | oui | **—** |
| 2535465779546251 | EwN1W | non | — | non | `kills=2 assists=1` |

Totaux : **17 actions au parc** (11 `kills`, 4 `assists`, 1 `flag_captures`, 1 `flag_steals`)
contre **12 au HEAD** (8 `kills`, 4 `assists`, **aucune famille `flag`**). La couverture du
drapeau, elle, est **identique** aux deux états (bursts/captures/steals/openings/noBridge/spawns/
objectLives = 3/3/3/3/3/2/4) : c'est bien le calque `objectives` qui a bougé, pas `flagCarries`.

Les deux actions de drapeau du match sont **toutes deux à SweatyYeti75** : `flag_steals` à
t=99 002 ms, `flag_captures` à t=105 526 ms.

### 9.2 Bissection — commit fautif et ligne

`git log` sur `objectiveevents/slotidentity*.go` et `replaybuild/matchfacts.go` entre les deux
états désigne **`d173b1a8c`** (2026-08-28, « obj-parmanche(1) : calque Objectives identifié PAR
MANCHE (fil des morts) »). Cuisson du même film complet de part et d'autre, **même schéma 23** :

| commit | actions | familles | joueurs porteurs d'actions |
|---|---|---|---|
| `d173b1a8c^` (`ed5c378b8`) | **17** | `kills=11 assists=4 flag_captures=1 flag_steals=1` | GEK XD, LostYeti71, TheBackwoodBoss, **SweatyYeti75**, DiegoGamer8K |
| **`d173b1a8c`** | **12** | `kills=8 assists=4` | JGtm, GEK XD, LostYeti71, TheBackwoodBoss, EwN1W |

**Ligne** : `apps/go-api/internal/replaybuild/matchfacts.go`, `identifiedEvents` — l'appel
`objectiveevents.SlotIdentityFrom(recs, lines)` (pont par **TRIPLET**, exige les lignes de match)
remplacé par `objectiveevents.ResolveRoundIdentity(recs, deaths)` (pont par **MORTS**).

### 9.3 La cause, sur pièces

Les deux ponts, mesurés **sur le même film**, slot par slot :

| slot | pont TRIPLET | pont MORTS | verdict |
|---|---|---|---|
| 10 | TheBackwoodBoss | TheBackwoodBoss | accord |
| 14 | — | JGtm | le triplet se tait |
| 16 | GEK XD | GEK XD | accord |
| 18 | LostYeti71 | LostYeti71 | accord |
| 20 | DiegoGamer8K | — | **les morts se taisent** |
| 22 | **SweatyYeti75** | — | **les morts se taisent** |
| 24 | — | EwN1W | le triplet se tait |

**ZÉRO CONTRADICTION.** Chacun nomme 5 slots, 3 en commun et à l'identique ; chacun nomme 2 slots
que l'autre laisse tomber. Ce ne sont pas deux réponses concurrentes, ce sont **deux couvertures
complémentaires** — et `d173b1a8c` a remplacé l'une par l'autre au lieu de les additionner.

**Pourquoi le pont par morts se tait sur les slots 20 et 22** : `deathInstantMin = 3`
(`objectiveevents/slotidentity_deaths.go`). Un slot n'est nommé que s'il aligne au moins **trois**
instants de mort coïncidents. SweatyYeti75 meurt **2** fois, DiegoGamer8K **1** fois. Ils sont
hors de portée **par construction** — et ce sont, par définition, les joueurs qui meurent le
moins, c'est-à-dire les meilleurs, c'est-à-dire ceux qui portent le drapeau.

**Contrôle croisé, deux chaînes disjointes** (les totaux du match d'un côté, les instants du film
de l'autre) :

```
slot 20 : 1 progression de mort [59881]        -> 2535465632069522 (DiegoGamer8K)  1 sur 1, et LUI SEUL
slot 22 : 2 progressions de mort [46451 70141] -> 2535463878425995 (SweatyYeti75)  2 sur 2, et LUI SEUL
```

Les instants de mort de ces deux slots coïncident **exclusivement** avec le xuid que le triplet
désigne. Le pont par morts ne se tait pas parce qu'il doute : il se tait parce qu'il **n'a pas
assez d'ancres** pour que sa marge s'applique.

### 9.4 Régression ou décision documentée ? — RÉGRESSION

Le message de `d173b1a8c` annonce : « Neutralité mono-manche prouvée par construction (une manche
= pont plat par morts) ». Cette phrase est **vraie contre le pont PLAT PAR MORTS**
(`SlotIdentityByDeaths`), qui alimentait déjà la couronne VIP, le drapeau et le crâne — pour ces
calques-là, la neutralité tient. Mais **le calque `objectives` n'était pas sur ce pont-là** : il
était sur le TRIPLET. La neutralité a donc été prouvée **contre le mauvais témoin**, et sur un
film à UNE MANCHE le calque a perdu deux joueurs sur cinq. Aucun ADR, aucun plan, aucune entrée du
registre des reports ne mentionne ce renoncement : ce n'est pas un arbitrage, c'est un angle mort.

### 9.5 Le correctif

`objectiveevents.RoundIdentity.CompletedByLines(recs, lines)` — complète l'identité par manche
avec le pont par triplet, sur les **seuls slots que le pont par morts n'a pas nommés**, sous
**trois gardes non négociables** :

1. **mono-manche seulement** — le triplet apparie des TOTAUX de match ; en multi-manche le slot est
   réattribué et le compteur repart de zéro, ce qui est exactement le défaut que `d173b1a8c` a
   corrigé et qu'il n'est pas question de réintroduire ;
2. **compléter, jamais contredire** — un slot déjà nommé par les morts garde son nom ;
3. **aucun xuid deux fois** — même règle que la seconde passe de `SlotIdentityFrom`.

`lines` vide rend l'identité **inchangée** : le calque reste publiable hors ligne, sans base — la
propriété que `d173b1a8c` avait gagnée est conservée.

Câblé dans `replaybuild.identifiedEvents`, qui recevait déjà les faits pour la courbe de score.

**Résultat mesuré sur le film complet** : **23 actions**, **7 joueurs pontés sur 7 présents au
film**, et chacun publie **EXACTEMENT sa ligne de la feuille de match** (JGtm 2/1, GEK XD 1/2,
LostYeti71 1/0, TheBackwoodBoss 2/0, SweatyYeti75 7/1, DiegoGamer8K 0/1, EwN1W 2/1 — total
15 `kills`, 6 `assists`, conforme à l'API). Les deux actions de drapeau sont revenues, sur
SweatyYeti75, aux mêmes instants qu'au parc. Le 8e joueur de la feuille (5 frags, 0 mort) n'a
aucune trajectoire dans le film : il n'a pas de slot à nommer.

### 9.6 Tests de non-régression

- `objectiveevents/slotidentity_completion_test.go` — quatre tests : le rattrapage du joueur qui
  meurt deux fois, la neutralité sans lignes, **le refus du multi-manche**, et le refus d'un xuid
  déjà pris.
- `wire/build_queue_worker_binary_integration_test.go` — sur la cuisson E2E réelle : 7 joueurs
  pontés, chacun **à égalité** avec la feuille de match, et `flag_captures = 1` / `flag_steals = 1`
  au porteur `2535463878425995`. **Vérifié par MUTATION** : correctif débranché, le test rougit
  avec « joueurs pontés = 5, attendu 7 » et les deux actions de drapeau à 0.

### 9.7 Périmètre sur le parc, et bump de schéma

- **Artefacts concernés** : tous ceux cuits depuis le **2026-08-28** (`d173b1a8c`) sur un film
  **mono-manche** où au moins un joueur meurt moins de trois fois — ses actions d'objectif y sont
  absentes. Ce n'est pas propre au CTF ni à Husky Raid : le pont est title-agnostic et sert toutes
  les familles (`kills`, `assists`, `zone_*`, `flag_*`, `bomb_detonations`). Relevé du parc local :
  16 artefacts à calque `objectives` ont été cuits après la bascule ; sept d'entre eux nomment
  8 joueurs sur 8 (les matchs où tout le monde meurt au moins 3 fois), les autres en perdent.
  Les films **multi-manche** ne sont pas concernés : la complétion s'y abstient, à dessein.
- **Bump de `SchemaVersion` : NON.** La convention du dépôt, posée par `d173b1a8c` lui-même
  (« la forme du document est INCHANGÉE (contrat stable) → aucun bump de schéma »), réserve le
  bump aux changements de FORME. Ici la forme ne bouge pas, seul le CONTENU s'enrichit. La
  propagation passe donc par la re-cuisson de release (`backfill-replay`), déjà inscrite au
  registre des reports pour les autres correctifs de contenu — entrée ajoutée pour celui-ci.

### 9.8 Découverte, notée et NON traitée

`flagCarries` reste à 0 sur ce film, et le correctif n'y change rien : le calque du drapeau
construit sa propre identité **dans `analysis/replay`** (`build_objectives_live.go`), paquet
title-agnostic qui ne reçoit **aucun fait de match** — c'est une frontière délibérée
(« publiable hors ligne »). Mesure : les 3 prises `noBridge` se répartissent en **2 sur un slot 12
agrégé** (jamais nommable : ses compteurs ne correspondent à aucun joueur) et **1 sur le slot 22**
(SweatyYeti75). Faire descendre les lignes jusqu'à ce calque ferait donc passer `flagCarries` de
0 à 1 portage sur ce film — gain réel mais modeste, au prix du franchissement d'une frontière
d'architecture. **Décision produit, hors mandat** : portée au registre des reports.
