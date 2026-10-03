# Instruction du fait « un joueur porte son propre drapeau » — 2026-09-06

Branche `feat/v2-drapeaux`, worktree `LevelUp-wt-v2-drapeaux`, base `7e5c454bc` (= `feat/v2-durees`,
SchemaVersion 45, revue DUREES-R1 close). Fait ouvert par la **decouverte n° 1** de
`.ai/V7.5/v2/INSTRUCTION_DUREES_2026-09-06.md`, laissee non traitee par ce lot-la.

---

## Verdict en une page

**DEFAUT DU CALQUE, corrige a la source.** L'etiquette d'equipe du drapeau est JUSTE ; c'est
l'attribution du portage qui etait fausse, pour deux raisons cumulees dans `assignFlags` :

| | Cause | Nature |
|---|---|---|
| **(1)** | L'etat « ou git chaque drapeau » se tenait a jour **en parcourant les prises**, si bien que la position de LACHER d'un portage y etait inscrite des son attribution — donc avant d'avoir eu lieu. Deux portages qui se recouvrent suffisent. | erreur d'**ORDRE** |
| **(2)** | Le repli sur le socle le plus proche, juste pour un VOL (il se fait a un socle), est **faux pour une PRISE** : elle se fait la ou l'objet est tombe, souvent pres du socle ADVERSE — c'est-a-dire du socle du camp qui marque, donc du porteur. | **REGLE manquante** |

Sur `bcb6d393` : **15 portages sur le drapeau de l'equipe 1 et 1 sur celui de l'equipe 0**
deviennent **16 sur 16 pour l'equipe 1, zero pour l'equipe 0**, et les **trois** captures se
publient sur le drapeau adverse (une l'etait sur le drapeau du camp qui marquait). Schema **45 ->
46**.

Deux hypotheses de depart sont **REFUTEES** : le 16e portage n'est ni un retour de drapeau ni un
artefact de joueur entre/sorti en cours de partie (§4), et l'etiquette d'equipe du catalogue de
carte n'est pas decalee par rapport a la feuille de match (§2 — preuve par la geometrie des trois
livraisons).

La premisse « une seule des trois captures est liee a un portage » est, elle, **a corriger** : les
trois portages captures existaient deja (`captured = true` sur les trois), et les trois `home`
etaient publies. Ce qui manquait, c'est que l'un des trois l'etait **sur le mauvais drapeau** (§3).

---

## Methode

Cuisson de production (`cmd/replay-build`), **un film par processus**, sous verrou d'exclusion
`filmproc.AcquireSolo` + verrou inter-agents (`mkdir`/`rmdir` sur le scratchpad), plafond memoire
3 Gio, priorite basse. Racine de travail dediee : `data/cache/{film_chunks,film_manifests,mvar}` et
`data/titles` en JONCTIONS vers le parc (lecture), `config/` en jonction vers le checkout teste,
`data/cache/replays` en VRAI dossier — **le parc n'a recu aucune ecriture** (verifie : `git status`
de `data/` vide des deux cotes).

Faits de match : `scratchpad/balayage/facts/*.facts.json`, exportes en lecture seule par
`levelup replay-facts-export` lors du balayage du parc.

Comparaison axe par axe : `cmd/replay-diff`. Diagnostic interne : **une sonde temporaire** posee
dans `assignFlags` (un `slog.Warn` par portage), cuisson, lecture du journal, puis
`git checkout --` du fichier — verifie identique a l'octet pres avant toute modification reelle.

---

## 1. A quel objet le film rattache un portage, et d'ou vient l'etiquette

**Le film ne rattache un portage a AUCUN objet.** Un portage est un compteur de statistique du
statborg (`flag_grabs` / `flag_steals`) sur un SLOT, plus la trajectoire de son porteur. L'objet
drapeau ne se lit que **libre** (`flag_objects.go`, vies libres portant `ID` = mot MPP et
`Key` = (slot, generation)) — et meme alors il ne porte ni equipe ni camp. `Track.Team` n'est pas
dans le film non plus. Il n'existe donc, dans le film, **aucune chaine qui nomme le drapeau d'un
portage**.

L'etiquette vient du **catalogue versionne de carte**
(`data/titles/halo_infinite/reference/map_objectives.json`), lu par `replaybuild.flagSpawns` : role
`flag_spawn`, champ `team_index`. Pour Cliffhanger (`mapId 5324364b-39a8-4f93-96a6-b80a1f18ce8a`) :

| socle | `team_index` | position |
|---|---|---|
| `flag_spawn` | **1** | (-7,30 · -1,79) |
| `flag_spawn` | **-1** (neutre, centre) | (18,44 · -20,81) |
| `flag_spawn` | **0** | (41,03 · 9,72) |

Le socle neutre est ecarte par `flag_neutral.go` (naissances aux socles d'equipe = 8, au neutre =
0 : la partie n'est pas a drapeau neutre). Restent deux drapeaux, dans l'ordre du catalogue :
**index 0 = equipe 1**, **index 1 = equipe 0**. Le rattachement portage -> index est ensuite
**purement geometrique** (`assignFlags`).

## 2. L'etiquette d'equipe est JUSTE — preuve independante par la livraison

La feuille de match `facts/bcb6d393.facts.json` donne `teamScores = [3, 0]` et place les
**quatre** porteurs dans l'equipe **0** (`2533274823110022`, `2533274858283686`,
`2535429985869093`, `2535469190789936`).

En CTF on livre le drapeau adverse **a sa propre base**. Les trois portages captures se terminent
en :

| capture | instant | porteur | equipe | position de fin | distance au socle `team_index` 0 (41,03 · 9,72) |
|---|---|---|---|---|---|
| 1 | 188 073 ms (frame 1751) | `2533274858283686` | 0 | (40,59 · 9,95) | **0,50 m** |
| 2 | 259 946 ms (frame 2470) | `2533274823110022` | 0 | (41,46 · 9,27) | **0,62 m** |
| 3 | 351 838 ms (frame 3389) | `2535429985869093` | 0 | (40,37 · 9,85) | **0,67 m** |

Les trois livraisons de l'equipe 0 se font au socle que le catalogue etiquette `team_index = 0`.
Le catalogue et la feuille de match **coincident donc**, et le `flag_delivery` du meme catalogue le
confirme (equipe 1 en (-7,30 · -1,79), equipe 0 en (41,02 · 9,73) — chaque zone de livraison est
au socle de son camp).

**Consequence** : le drapeau porte 15 fois est bien celui de l'equipe 1, et le 16e portage, pose
sur le drapeau de l'equipe 0 par un joueur de l'equipe 0, est **impossible**. L'etiquette n'est pas
en cause : l'attribution l'est.

## 3. Les trois captures, et pourquoi une seule paraissait liee

Le calque des ACTIONS (`objectives`, cuisson au schema 45) publie **trois** `flag_captures`, tous
a des joueurs de l'equipe 0 — le score `3-0` est donc entierement explique :

```
frame 1751 (188 073 ms)  flag_captures  2533274858283686   + 2 flag_capture_assists
frame 2470 (259 946 ms)  flag_captures  2533274823110022
frame 3389 (351 838 ms)  flag_captures  2535429985869093
```

La sonde montre que **les trois portages correspondants portaient deja `captured = true`**
(indices 4, 6 et 15 ci-dessous) et que **les trois `home` etaient publies**. La premisse « une
seule capture est liee » vient d'une lecture du document : seul le portage du drapeau « equipe 0 »
presentait la sequence lisible `carried` -> `home`, les deux autres etant separes de leur `home`
par un `dropped` d'une frame (cf. decouverte n° 2). Le releve exact, AVANT correctif :

| capture | ou le `home` etait publie |
|---|---|
| frame 1751 | **drapeau 1 (equipe 0)** — le camp qui marque, donc FAUX |
| frame 2470 | drapeau 0 (equipe 1) — juste |
| frame 3389 | drapeau 0 (equipe 1) — juste |

Le drapeau de l'equipe 1, prive de sa capture a 1751, rentrait chez lui par une **rentree
d'objet** (`homeByObject`) : une inference tenait la place d'un fait date. Apres correctif,
`homeByObject` passe de **2 a 1** — la capture reprend sa place et la rentree devient inutile.

## 4. Le 16e portage : un VRAI portage du drapeau adverse, mal etiquete

Sonde dans `assignFlags` (extrait, cuisson au HEAD 45 ; `drop0` = position du drapeau 0 « au sol »
telle que la boucle la croyait, `viaDrop` = index rendu par la regle du drapeau au sol) :

```
i=2  2533274858283686  t0=143777  t1=180531  steal=false capt=false  x0=(-0.05,-5.19) x1=(33.61,2.48)  viaDrop=0  fi=0  drop0=&[1.97 -7.35]
i=3  2535429985869093  t0=171941  t1=325913  steal=false capt=false  x0=(29.86,-3.75) x1=(-6.53,-2.19) viaDrop=0  fi=0  drop0=&[33.61 2.48]
i=4  2533274858283686  t0=180531  t1=188073  steal=false capt=TRUE   x0=(33.61,2.48)  x1=(40.59,9.95) viaDrop=-1 fi=1  drop0=&[-6.53 -2.19]
```

Trois faits se lisent sur ces trois lignes :

1. **La prise n'est pas un vol** : `steal = false` — c'est un `flag_grabs`, donc le ramassage d'un
   drapeau DEJA au sol. Le film ne publie aucun `flag_steals` a cet instant (4 vols au total :
   frames 471, 1084, 2042, 3130).
2. **Elle ramasse le drapeau que le portage precedent vient de poser** : `x0 = (33,61 · 2,48)` est
   **exactement** `x1` du portage `i=2`, au meme instant `180 531 ms` — **distance 0 m**, quand le
   rayon de ramassage est de 8 m.
3. **Le sol mentait** : `drop0 = (-6,53 · -2,19)`, c'est-a-dire la position de lacher du portage
   `i=3`, **qui ne se ferme qu'a 325 913 ms — 145 secondes plus tard**. `i=3` s'ouvre avant `i=4`,
   donc l'ancienne boucle, qui parcourait les PRISES, avait deja inscrit son lacher futur. La
   regle du drapeau au sol echoue (`viaDrop = -1`), et le repli geometrique tranche :

   | | distance depuis (33,61 · 2,48) |
   |---|---|
   | socle equipe 1 (-7,30 · -1,79) | **41,13 m** |
   | socle equipe 0 (41,03 · 9,72) | **10,37 m** |

   -> `fi = 1`, le drapeau de l'equipe **0**, celle du porteur.

**Ce n'est donc pas un artefact.** Les trois hypotheses concurrentes tombent :

- **retour de drapeau** : un `flag_returns` n'ouvre aucun portage (le calque ne lit que
  `flag_grabs` / `flag_steals`) ; les deux `flag_returns` du film sont a 1556 et 2174 ;
- **joueur entre en cours de partie** : les trois `joinedInProgress` (`2533274811363842` a
  282 262 ms, `2535418713587213`, `2535468064146356`) et les trois `leftInProgress`
  (`2533274876732804`, `2535460750735339`, `2791963697577117`) sont **tous de l'equipe 1**, et
  **aucun** ne porte le drapeau — les 16 portages sont a quatre joueurs de l'equipe 0 ;
- **slot re-attribue** : la couverture publie `noBridge = 0` et `noTrack = 0`, le film est
  MONO-MANCHE, et le portage se ferme sur la capture du MEME slot 8 s plus tard.

Le 16e portage est le dernier relais d'une course a trois : `i=2` porte le drapeau de l'equipe 1
jusqu'a 10 m de la base adverse, meurt ; `i=4` le ramasse a 0 m de la ou il est tombe et le livre.

## 5. Le correctif

Nouveau fichier `internal/analysis/replay/flag_assign.go` — deplacement PUR de `assignFlags`,
`nearestDroppedFlag` et `nearestSpawn` (`flag_carries.go` passe de 451 a 396 lignes), plus deux
changements :

1. **Le parcours se fait par EVENEMENTS DATES** (`flagGroundTimeline`) : une prise enleve le
   drapeau du sol, une fin l'y repose — ou le renvoie a sa base quand elle est une capture. A
   instant egal, **la fin passe avant la prise**, le meme ordre qu'`assembleFlagLives`
   ([flagLifeClose] avant [flagLifeOpen]) et pour la meme raison : un drapeau libere dans la meme
   milliseconde se reprend tout de suite — c'est le cas NOMINAL ici (180 531 ms).
2. **Une troisieme regle** : une PRISE que le sol ne rattache a rien va au drapeau **DEJA EN JEU**
   quand il est le SEUL. `enJeu[f]` = le drapeau a quitte son socle et n'y est pas rentre ; une
   capture l'y ramene. A deux drapeaux dehors, la regle **se tait** et le socle le plus proche
   reprend la main — on RETRECIT le repli, on ne le supprime pas.

L'ordre des trois regles est fige : **vol -> socle** ; **prise -> drapeau au sol a moins de 8 m** ;
**prise -> seul drapeau en jeu** ; sinon socle.

Limite ecrite dans le code : les retours credites (`flag_returns`) et les rentrees d'objet ne sont
pas connus a cet endroit (ils vivent dans `assembleFlagLives`, en aval). Ils ne peuvent donc que
laisser un drapeau « en jeu » **de trop**, ce qui fait TAIRE la troisieme regle au lieu de la faire
mentir.

## 6. Tests, prouves par mutation

`internal/analysis/replay/flag_assign_test.go` — quatre tests sur la geometrie de `bcb6d393`
reduite au banc (socle du camp qui subit en (0,0), socle du camp qui marque en (100,100)) :

| test | ce qu'il fige |
|---|---|
| `TestFlagAssignLeSolSuitLeTempsEtNonLOrdreDesPrises` | la cause (1). Les DEUX drapeaux sont dehors, donc la regle du « seul drapeau en jeu » est muette : seul l'ordre du temps peut repondre |
| `TestFlagAssignPriseVaAuSeulDrapeauEnJeu` | la cause (2). Le sol ne rattache rien, la prise est a 17 m du socle du porteur et a 124 m de l'autre |
| `TestFlagAssignADeuxDrapeauxEnJeuLaRegleSeTait` | **la contre-epreuve** : a deux drapeaux dehors, le socle reprend la main |
| `TestFlagAssignLeVolResteAuSocle` | l'ORDRE des regles : un vol prend le drapeau de son socle meme si l'autre git a ses pieds |

**Quatre mutations jouees, chacune ROUGE puis VERTE** :

```
M1  parcours par les PRISES (lacher inscrit d avance)
    --- FAIL: TestFlagAssignLeSolSuitLeTempsEtNonLOrdreDesPrises
        drapeau d'equipe 1 porte par [1 2], attendu [1 2 3]
M2  regle du « seul drapeau en jeu » retiree
    --- FAIL: TestFlagAssignPriseVaAuSeulDrapeauEnJeu
        drapeau d'equipe 1 porte par [1], attendu [1 2]
M3  seulEnJeu devient un fourre-tout (rend le PREMIER en jeu)
    --- FAIL: TestFlagAssignADeuxDrapeauxEnJeuLaRegleSeTait
        drapeau d'equipe 1 porte par [1 3], attendu [1]
M4  le vol ne court-circuite plus le sol
    --- FAIL: TestFlagAssignLeVolResteAuSocle    (+ 2 autres)
        drapeau d'equipe 1 porte par [1 2], attendu [1]
```

## 7. Mesure — cinq films re-cuits des deux cotes

Invariants du controle : **(A)** aucun portage sur le drapeau de sa propre equipe ;
**(B)** nombre de captures = score.

| film | variante | score | fautes (A) avant | fautes (A) apres | captures / score |
|---|---|---|---|---|---|
| **`bcb6d393`** | CTF:Arena | 3-0 | **1** | **0** | 3 = 3 — conforme |
| **`e94163af`** | CTF:Arena Neutral Flag | 1-5 | 0 | 0 | 6 = 1+5 — conforme (drapeau neutre : (A) sans objet) |
| **`c0a82e88`** | Husky Raid:CTF | 3-0 | 0 | 0 | 1 publiee (2 prises `noBridge`) |
| `cde26226` | CTF:Arena | 2-3 | 0 | 0 | 4 — conforme |
| `64e8adfa` | CTF:Arena, **2 manches** | 2-3 | **13** | **7** | 3 — conforme |

Par drapeau sur `bcb6d393` :

| | avant (schema 45) | apres (schema 46) |
|---|---|---|
| drapeau 0 — equipe **1** | 15 portages, 33 spans | **16 portages**, 35 spans |
| drapeau 1 — equipe **0** | **1 portage**, 3 spans | **0 portage**, 1 span (`home` tout le match) |
| `coverage.flagCarries.homeByObject` | 2 | **1** (la capture remplace une rentree inferee) |

**`cmd/replay-diff`, HEAD 45 contre HEAD 46** — l'effet est confine a `flagCarries` :

```
bcb6d393   12 ecarts / 679 mesures   ports 10 · couverture 1 (homeByObject) · entete 1 (schema)
e94163af    1 ecart  / 645 mesures   entete 1 (schema)  -- identique par ailleurs
c0a82e88    1 ecart  / 569 mesures   entete 1 (schema)  -- identique par ailleurs
cde26226    1 ecart  / 660 mesures   entete 1 (schema)  -- identique par ailleurs
64e8adfa   13 ecarts / 699 mesures   ports 8 gains / 3 pertes · homeByObject 10 -> 12 · schema
```

`changements = 0` partout : **aucune mesure ne change de valeur**, seules des mesures apparaissent
ou disparaissent.

## 8. Gates joues

```
cd apps/go-api
go test -count=1 ./internal/analysis/... ./internal/replaybuild/... ./internal/replaydiff/... \
        ./internal/archlint/... ./contracttest/...            # ok, 23 paquets
go test -count=1 -tags=integration -p 1 ./internal/api/wire/...  # ok 18,6 s (code de cuisson touche)
go build ./...                                                    # ok (CGO_ENABLED=1)
golangci-lint run --new-from-merge-base=origin/main ./...         # 0 issues
```

**Golden d'assemblage regenere : unique ecart = la ligne de version** (`schema 45` -> `schema 46`,
1 ligne sur 606 des deux cotes) — le film de reference `000d5950` n'est pas un CTF et n'est touche
par rien. Le contrat OpenAPI declare `schemaVersion` sans `enum`/`const`/`default` : un bump ne le
deplace pas ; aucun champ n'est ajoute au document.

## 8 bis. Un rouge de CI REPARE, apporte par le merge des corrections DUREES-R1

Le job « Go Coverage + Baseline non-regression » (`./...` complet, seul endroit ou la CI
execute `internal/service/replayview`) est sorti ROUGE apres le merge de `feat/v2-durees` :
**25 tests en echec, tous dans ce paquet, une seule cause**.

```
--- FAIL: TestChaqueChampStockeAUneDecision
    FlagCarriesCoverage.AmbiguousSlot est publie par l artefact et ABSENT du document servi,
    sans entree dans champsNonServis — soit le contrat le porte, soit la decision s ecrit
```

Le correctif C1 de DUREES-R1 ajoute le compteur `AmbiguousSlot` a
`replay.FlagCarriesCoverage` (les slots dont la vie anonyme est refusee au repli) sans le
miroiter dans le document SERVI. Le garde-rail de parite fait exactement son travail : un
compteur de couverture qui n atteint pas le client sortirait du contrat par omission.

**Repare ici** (directive « tout rouge se repare, meme prealable ») et par le cote QUI SERT,
pas par une exemption : les vingt-huit autres compteurs de ce bloc sont servis, celui-ci n a
aucune raison de ne pas l etre. Trois pas, sans decision de produit :
`replaydoc.FlagCarriesCoverage` gagne le champ a la MEME place, `toFlagCarriesCoverage` le
recopie, et `openapi.yaml` puis `generated.ts` sont REGENERES (jamais edites a la main) —
diff : 4 lignes au contrat, 2 aux types web. Paquet vert, contrat vert.

## 8 ter. Complement : spans masques — une lecture vraie n'est ni perdue ni masquee

La premiere version de ce lot mettait le 16e portage sur le BON drapeau et, ce faisant, le
faisait tomber sous un defaut PREEXISTANT (decouverte n° 2 d'alors) : la duree PUBLIEE de
`2533274858283686` passait de 358 a 282 frames. Le lot des durees venait de retablir ces 358
contre le parc ; l'axe des durees d'un gate en mode base serait donc sorti EN ECHEC. Corrige ici.

### Le defaut, a la ligne

`applyFlagLifeEvent` date l'ouverture d'un portage a `frameOfMatchMS(t0)` et sa fin a
`frameOfMatchMS(t1) + 1`. Quand un portage est ferme par LA PRISE SUIVANTE DU MEME SLOT
(`boundFlagCarries`, quatrieme fait de fermeture), `t1` vaut exactement le `t0` du suivant : la
fin tombe donc UNE FRAME APRES l'ouverture de la reprise, et `spansOfTransitions`, qui trie par
FRAME, la laisse ecraser l'etat `carried`. Le portage repris se reduisait a UNE frame, suivi
d'un `dropped` qui couvrait tout le trajet — **le drapeau se dessinait au sol pendant qu'un
joueur courait avec**.

### L'invariant pose, plus general que le symptome

Une fin ne publie pas `dropped` quand, a cet instant, **un autre portage du meme drapeau est
encore ouvert** (`flagTenuParUnAutre`) : le drapeau passe d'une main a l'autre, il ne touche pas
le sol. Deux situations le declenchent :

- la **reprise** du meme porteur — aucun lacher DATE : le modele borne lui-meme le sejour au sol
  a zero (11 fois sur 16 portages sur `bcb6d393`) ;
- un **recouvrement** de deux portages du meme drapeau — une incoherence que la couverture
  publie deja (`overlaps`, `closedOverlaps`) ; publier le lacher de l'un ECRASAIT le portage de
  l'autre (c'est ce qui coutait 4 frames a `2533274823110022` sur `64e8adfa`).

**UNE CAPTURE N'EST JAMAIS RETENUE** : elle ne pose pas le drapeau au sol, elle le renvoie a sa
base, et c'est un fait DATE qui tranche sur tout recouvrement. La garde est ecrite et testee.

Le biais est celui que l'en-tete de `flag_carries.go` assume depuis l'origine : se tromper en
dessinant le drapeau dans une main qui ne le tient plus, jamais en le posant au sol alors qu'un
joueur court avec.

### Les deux compteurs, servis jusqu'au contrat

`coverage.flagCarries` publie desormais ce que les regles ont DECIDE, et les deux traversent
`replaydoc` -> `replayview` -> `openapi.yaml` -> `generated.ts` (cliquet de parite
`TestChaqueChampStockeAUneDecision`) :

- **`assignedByPlay`** — prises attribuees au SEUL drapeau en jeu (troisieme regle
  d'attribution, une attribution par elimination) ;
- **`dropsWithheld`** — fins dont l'etat `dropped` n'a pas ete publie parce qu'un autre portage
  tenait encore le drapeau.

### Mesure, base `feat/v2-durees` HEAD (`0930cc692`, recompilee) contre ce HEAD

| film | duree totale portee | `dropsWithheld` | joueurs dont la duree BAISSE | spans masques |
|---|---|---|---|---|
| `bcb6d393` | 948 -> **1 423** | 11 | **aucun** | 10 -> **0** |
| `e94163af` | 664 -> **1 159** | 18 | **aucun** | 17 -> **0** |
| `c0a82e88` | 67 -> 67 | 0 | **aucun** | 0 -> 0 |
| `cde26226` | 4 119 -> **6 422** | 63 | **aucun** | 63 -> **0** |
| `64e8adfa` | 2 441 -> **4 388** | 49 | **aucun** | 41 -> **0** |

Durees par joueur sur `bcb6d393` (frames), base contre HEAD :

| xuid | base | HEAD |
|---|---|---|
| `2533274823110022` | 441 | **666** |
| `2533274858283686` | **358** | **358** (tenu) |
| `2535429985869093` | 96 | **346** |
| `2535469190789936` | 53 | 53 |
| **total** | **948** | **1 423** |

Les trois temoins mandates gardent 0 portage sur son propre drapeau et leurs captures sur le
drapeau adverse. `64e8adfa` : 13 -> 7 fautes, aucune perte de duree.

### Tests, prouves par mutation

`internal/analysis/replay/flag_reprise_test.go` :

- `TestFlagRepriseNePubliePasDeLacher` — deux portages du meme porteur se fondent en UN span
  continu `[10..50]` (41 frames), `dropsWithheld = 1`, aucun `dropped` ;
- `TestFlagRepriseSurUnAutreDrapeauLacheBien` — **la contre-epreuve** : si la reprise porte sur
  l'AUTRE drapeau, le premier est bel et bien lache et son `dropped` se publie.

```
M-A  la fin de la reprise est emise quand meme
     --- FAIL: TestFlagRepriseNePubliePasDeLacher
         etats [home carried dropped home], attendu [home carried home]
M-B  la garde « meme drapeau » retiree
     --- FAIL: TestFlagRepriseSurUnAutreDrapeauLacheBien
         dropsWithheld = 1, attendu 0 ; etats [home carried], attendu [home carried dropped]
M-C  la garde « une capture n'est jamais retenue » retiree
     --- FAIL: TestFlagAssignLeSolSuitLeTempsEtNonLOrdreDesPrises
         aucun retour a la base a la frame 71 sur le drapeau adverse
```

Les deux tests d'attribution asserent en outre `assignedByPlay` (1 quand la troisieme regle
tranche, 0 quand elle se tait).

### Ce que le complement ne repare pas

Sur `64e8adfa` (2 manches, `closedOverlaps = 10`), la capture de 529 075 ms passe du drapeau
adverse au drapeau du camp de son auteur. Ce n'est PAS le fait de cet invariant — c'est le
portage de 5 169 qui est mal attribue, l'une des 7 fautes residuelles deja au registre : la prise
se fait a 2,3 m du socle de son PROPRE camp, les deux drapeaux sont dehors, et la regle du seul
drapeau en jeu se tait. La regle qui la reparerait (« une prise au socle S ne porte pas sur le
drapeau de S ») reste REFUTEE par la faute de 6 569 sur le meme film.

## 8 quater. Corrections R1 — jamais son propre drapeau, et les retours remettent les deux etats

La revue adversariale **DRAPEAUX-R1** valide la cause premiere et confirme que rien n'est perdu ni
invente (trois spans rallonges controles par les positions ET par le calque des actions), mais
elle refuse la livraison sur **un P0** : le lot servait un resultat FAUX NEUF. Quatre constats,
les quatre traites ici.

### C1 (P0) — une capture datee passait sur le drapeau de sa propre equipe

Sur `64e8adfa`, a 527 555 ms, un joueur de l'equipe 1 ramasse a **2,4 m de son propre socle** ; le
drapeau adverse git a **11,2 m**, au-dela de `flagPickupRadiusM` = 8, et les deux drapeaux passent
pour « en jeu » — la regle 2 se tait, la regle 3 aussi. Restait le repli `nearestSpawn`, que
l'en-tete du meme fichier declare pourtant FAUX pour une prise : il designe le socle le plus
proche, celui du PRENEUR. Le portage — et la **capture** de 529 075 ms qu'il porte — basculaient
sur le drapeau du camp de leur auteur. Le document se contredisait : il publiait ce drapeau
`home` a la frame 5168 et `carried` par un joueur de son propre camp a la 5169.

**L'INVARIANT DUR, pose avant toute geometrie.** En CTF on RENVOIE son drapeau, on ne le porte
pas : c'est la regle du mode, tranchee par l'utilisateur, et elle prime sur toute inference. Tout
candidat qui aboutit au drapeau de l'equipe du porteur est **REFUSE** (`ownFlagRefused`) ; s'il ne
reste qu'un candidat — l'autre drapeau — il est pris ; s'il n'en reste aucun, le portage sort
**NON ATTRIBUE** (`flagIndex` = -1, `unresolved`) et n'est publie sur aucun drapeau. **On n'invente
jamais un drapeau.**

**L'equipe du porteur ne vient pas du film** : elle arrive par `FlagInput.TeamOf`, une table
xuid -> equipe **DEJA RESOLUE** par l'appelant — la meme forme et la meme frontiere que
`FlagInput.Identity` : `analysis/replay` recoit une table, jamais des lignes de match. Une equipe
inconnue (-1 en base) n'y entre pas : on ne refuse que sur une equipe LUE. Table vide (CLI hors
ligne, ouvrier sans faits) : l'invariant **se tait**, et l'artefact est celui d'avant a l'octet
pres — c'est `TestFlagInvariantSansEquipeConnueSeTait`.

**LE CABLAGE S'EST TU UNE FOIS, ET LA MESURE L'A DIT.** La premiere cuisson R1 rendait
`ownFlagRefused = 0` : la table remontait bien de `replaybuild` jusqu'a `FlagInput`, mais
`attachFlagCarries` ne la recopiait pas dans `FlagCarryScan`. Un chainon muet ne casse aucun test
unitaire — d'ou le garde-rail ajoute chez l'appelant
(`TestFlagInputDescendLesEquipesJusquAuScan`, plus `TestFlagInputPorteLesEquipesDesLignesDeMatch`
pour la table elle-meme).

### C4 — un retour remet `sol` **ET** `enJeu`, jamais l'un sans l'autre

La note du fichier avouait une limite en n'en disant que la moitie : « les retours credites et les
rentrees d'objet [...] ne peuvent que laisser un drapeau en jeu de trop, ce qui fait **TAIRE** la
troisieme regle ». Vrai pour `enJeu` ; **faux pour `sol`**, qui alimente la regle 2 —
**prioritaire** — et pouvait donc la faire **MENTIR**, en rattachant une prise a une position de
lacher devenue caduque. `assignFlags` consomme desormais les deux chaines : `flag_returns` (qui ne
nomme pas son drapeau : applique au SEUL qui git au sol, meme abstention qu'en aval) et les
rentrees d'objet (qui nomment le leur par leur socle). Les evenements sont ordonnes comme dans
`assembleFlagLives` : fin, puis retour, puis rentree, puis prise.

### C2 — le champ mort `flagCarryRaw.reprise`

Ecrit trois fois, **jamais lu** : la garde livree est `flagTenuParUnAutre`, qui ne consulte que
`t0` / `t1` / `flagIndex`. Son commentaire affirmait pourtant que `flag_carries_lives.go` s'en
sert — une doc inversee doublee d'un commentaire d'entretien dans `closeByCarrierKills`. Champ et
commentaires **supprimes** (CLAUDE.md regle 7).

### C3 — les recouvrements se comptent PAR DRAPEAU

`countFlagOverlaps` exigeait **plus de deux** portages ouverts, **tous drapeaux confondus** :
`flagIndex` n'y entrait pas, si bien que deux porteurs du MEME drapeau — le recouvrement qui fait
vraiment se contredire le calque, et le cas nominal d'un CTF a deux drapeaux — n'y entraient
jamais. Mesure du relecteur sur un banc a deux porteurs et un drapeau : `overlaps = 0`,
`closedOverlaps = 0`. Le seuil est desormais « plus d'UN portage sur LE MEME drapeau », les
portages non attribues exclus. La note de `flag_carries_lives.go` qui s'appuyait sur ce compteur
est corrigee.

### Mesure — `64e8adfa`, la ou le P0 se voyait

| | base du lot `0930cc692` | **parent immediat `33aad602c`** | **HEAD R1** |
|---|---|---|---|
| captures publiees sur le drapeau de LEUR AUTEUR | 1 | **2** (P0) | **0** |
| portages sur son propre drapeau | 13 | 7 | **0** |
| spans masques (1 frame + `dropped`) | 41 | 0 | 0 |
| duree totale portee | 2 441 | 4 388 | **4 438** |

**LA BASE DE COMPARAISON DE CETTE RONDE EST LE PARENT IMMEDIAT** `33aad602c`, pas la base du lot :
c'est contre lui que se juge ce que les corrections R1 ont change, et c'est lui qui portait le P0.

Les **trois** captures de l'oracle sont desormais sur le drapeau adverse — y compris celle de
472 578 ms, **deja fausse a la base** (le P1 preexistant que la revue mettait hors perimetre) :
l'invariant la repare aussi. Couverture : `ownFlagRefused = 4`, `unresolved = 0`,
`assignedByPlay = 4`, `overlaps` / `closedOverlaps` = 12 / 12 (comptes par drapeau).

### Les trois temoins mandates ne bougent pas

`replay-diff` du HEAD revu contre le HEAD R1 — **aucun ecart de donnee**, seuls les compteurs que
C3 et C4 rendent justes :

```
bcb6d393   2 ecarts / 684 mesures   overlaps 0 -> 4 · closedOverlaps 0 -> 4
e94163af   3 ecarts / 650 mesures   overlaps 0 -> 1 · closedOverlaps 0 -> 1 · assignedByPlay 1 -> 0
c0a82e88   aucune difference (574 mesures identiques)
```

Sur `e94163af` (drapeau NEUTRE, un seul socle) `assignedByPlay` tombe a 0 parce que les retours
remettent `enJeu` : la regle 3 se tait la ou elle repondait, et le repli rend **le meme** drapeau —
il n'y en a qu'un. Les spans sont identiques au bit pres. Les trois temoins gardent 0 portage sur
son propre drapeau, leurs captures sur le drapeau adverse, 0 span masque et **aucune perte de
duree par joueur** contre la base.

### Tests, prouves par mutation

`internal/analysis/replay/flag_invariant_test.go` — cinq tests : le refus et son unique repli, le
silence sans equipe lue, **l'abstention qui n'invente rien** (un seul socle : `unresolved = 1`,
aucun span publie), le retour qui remet les deux etats, et les recouvrements par drapeau.

```
M-E  l invariant « jamais son propre drapeau » retire
     --- FAIL: TestFlagInvariantJamaisSonPropreDrapeau    drapeau d equipe 1 porte par [1], attendu [1 3]
     --- FAIL: TestFlagInvariantSansCandidatNInventeRien  un portage non attribue a ete publie
M-F  le refus INVENTE un drapeau (index 0) au lieu de s abstenir
     --- FAIL: TestFlagInvariantSansCandidatNInventeRien  attendu 1 refus et 1 portage non attribue
M-G  le retour ne remet que enJeu, pas sol
     --- FAIL: TestFlagRetourRemetLeSolEtEnJeu            drapeau d equipe 1 porte par [1 2], attendu [1]
M-H  les recouvrements redeviennent tous drapeaux confondus
     --- FAIL: TestFlagOverlapsComptesParDrapeau          overlaps 0, closedOverlaps 0
```

**M-G a d'abord SURVECU**, et c'est instructif : sur une carte a deux drapeaux, l'invariant dur ne
laisse qu'un candidat et masque l'effet de `sol`. Le test a ete refait **sans equipe connue**, la
seule facon d'isoler le constat C4 — l'invariant s'y tait, et c'est bien l'etat du sol qui tranche.

### Corrections R2 — le maillon que personne ne gardait

La ronde **DRAPEAUX-R2** declare C1 a C4 exacts (quatre mutations rouges, **0 portage sur son
propre drapeau sur 88 spans controles exhaustivement**, 3 captures = oracle) et ne laisse qu'un
constat, **W1 (P1, test seul)** : supprimer la ligne `TeamOf: in.TeamOf` d'`attachFlagCarries`
(`build_objectives_live.go:162`) laissait les **166 paquets verts**. La table des equipes traverse
DEUX maillons — `replaybuild` -> `FlagInput`, puis `FlagInput` -> `FlagCarryScan` — et les deux
garde-rails du lot ne couvraient que le premier. C'est exactement le chainon qui s'etait tu au
premier essai R1.

`TestAttachFlagCarriesDescendLesEquipesJusquAuScan` exerce `attachFlagCarries` sur un film
synthetique (records du statborg, bursts, socles, pont pose, `TeamOf` peuple) et verifie non pas
un champ — il est interne — mais son **EFFET** : l'invariant a refuse. Mutation jouee :

```
W1  la ligne « TeamOf: in.TeamOf » retiree d attachFlagCarries
    --- FAIL: TestAttachFlagCarriesDescendLesEquipesJusquAuScan
        ownFlagRefused = 0, attendu 1 : la table des equipes n a pas atteint le calque
        drapeau d equipe 0 porte par [2 3], attendu [2]
```

### Observation O1 de la ronde R2 — trois durees baissent contre le PARENT, et c'est une defusion

Contre `33aad602c`, trois xuid de `64e8adfa` perdent de la duree publiee : **612 -> 596**,
**810 -> 464**, **552 -> 142**. Ce n'est **pas une perte de donnee** : au parent, ces joueurs
portaient — a tort — le drapeau de leur PROPRE equipe, et deux spans du meme joueur sur deux
drapeaux differents se cumulaient dans la somme par xuid. L'invariant les ramene sur un seul
drapeau ; l'artefact de rendu que la faute d'attribution creait **se defait**, et la duree publiee
redescend a ce que le film dit. Le total du film MONTE (4 388 -> 4 438) et aucune duree ne baisse
contre la base du lot `0930cc692` — c'est bien le portage fautif, et lui seul, qui disparait.

## 9. Decouvertes, notees et NON traitees

Les decouvertes n° 2 (spans masques) et n° 3 (compteur de couverture) de la premiere redaction
sont **TRAITEES** au §8 ter ; la n° 1 (les fautes residuelles de `64e8adfa`) l'est au §8 quater —
l'invariant dur les ferme toutes, y compris la capture qui etait deja fausse a la base. Il reste
une observation, qui n'appartient pas a ce lot.

1. **Un porteur nomme par le PONT n'est pas localisable dans le document** (observation O1 de la
   revue DRAPEAUX-R1). Sur `bcb6d393`, les 9 portages de `2535429985869093` entre les frames 3129
   et 3388 sont servis `carried` par ce xuid, alors qu'AUCUNE piste publiee ne porte ce xuid apres
   la frame 2736 : la seconde vie du slot 536 est ANONYME dans `tracks`. Le serveur la nomme par
   le pont canonique (`attachFlagCarryPositions`, `noTrack = 0`) mais ne le dit pas au document ;
   le client joint par XUID (`flagCarriesLayer.ts`, `flagPointAt`) et retombe sur la position
   figee du span — le drapeau reste immobile au point de prise pendant que le porteur court.
   **PREEXISTANT et NON AGGRAVE EN POSITION** : a la base, les memes frames etaient `dropped` aux
   memes coordonnees ; seul l'ETAT publie change. *Reprise* : c'est le sujet du lot des vies
   anonymes (nommer la vie dans `tracks`, ou faire porter au span la cle de vie qui le localise),
   pas celui de l'attribution des drapeaux.
