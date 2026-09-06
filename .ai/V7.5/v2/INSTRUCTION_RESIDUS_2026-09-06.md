# Instruction des quinze faits residuels du balayage du parc — 2026-09-06

Branche `feat/v2-residus`, worktree `LevelUp-wt-v2-residus`, base `b510d8340` (= `feat/v75`,
schema 41 ; le dernier commit du principal ne porte que des fonds de carte PNG, aucun code).

Source : `.ai/V7.5/v2/BALAYAGE_PARC_2026-09-06.md`, section « Liste EXHAUSTIVE des pertes
restantes ». Quinze faits, tous anterieurs au chantier v2, qu'aucune chronique de schema
n'expliquait.

---

## Verdict en une page

| # | Fait | Verdict | Cause |
|---|---|---|---|
| **1** | `d9781168` (s23) `skullCarries/n` **36 -> 30** | **REGRESSION**, corrigee | Le gate de presence lisait « aucune vie NOMMEE » comme « porteur ABSENT » |
| 2 | `2cf24f30` (s31) episodes de surbouclier 7 -> 6 | **ancien artefact FAUX** | Episode de duree NULLE ancre sur un point aberrant (z = -304) que l'assainissement a supprime |
| 3 | `24dbb67d` (s20/s21) vies nommees 90 -> 89 | **decouverte deja enregistree** | Corps immobile dans un trou de replication : `lifeGapUS` coupe, `minPoints` jette, la mort nomme une vie non publiee |
| 4 | `4f77afc1` (s34) vies nommees 48 -> 47 | **ancien artefact FAUX** | Meme mecanique, mais sur un point VRAIMENT aberrant (z +107 -> -387, 566 s plus tard) |
| 5 | `084a804d` (s20) vies d'un xuid 6 -> 5 | **ancien artefact FAUX** | Idem (z +91 -> -370) |
| 6-9 | `1b2d9e08` (s38), `1cd3848a`, `3923bede`, `e85d7bad` (s32) `pickups.originGround` −1 a −2 | **reclassement dans un GAIN** | Le catalogue de socles est passe `established` : des ramassages classes « au sol » sont reconnus « au socle » |
| 10 | `21ece4d8` (s23) `abilities/n` 23 -> 22 | **ancien artefact FAUX** | Capacite datee HORS de toute piste publiee de son slot — rien a quoi la rattacher |
| 11 | `000d5950` (s2) `abilityLabels/n` 4 -> 2 | **ancien artefact FAUX** | Dictionnaire d'un espace de cles mort (3-6 contre 20-21), et qui ne referencait AUCUNE capacite |
| 12 | `11de8353` (s20) `coverage.score.points` 506 -> 504 | **normalisation** | Une ancre `{t:0,v:0}` redondante, que l'equipe adverse n'avait pas : la courbe est desormais coherente entre les deux camps |
| 13 | `bf15f7ab` (s34) `weaponLabels/n` 17 -> 16 | **ancien artefact FAUX** | Un socle etiquete « Carabine Vestige » : ce code n'apparait NULLE PART ailleurs dans le film, la Carabine a impulsion y est attestee 3 fois |
| 14 | `43716616` (s21) frags/morts/assistances/score d'UN joueur | **re-attribution dans un gain** — verifie | 1 joueur publie / 1 exact contre la feuille -> **8 publies / 4 exacts** |
| 15 | `51ebbc0f` (s21) frags/assistances/score d'UN joueur | **re-attribution dans un gain** — verifie | 3 publies / 0 exact -> **8 publies / 0 exact** ; film a 2 manches, le pont par TRIPLET n'y a aucun sens |
| — | `bcb6d393` (s20) 3 actions `kills` sur 2 joueurs | **re-attribution dans un gain** — verifie | 67 -> **76** actions, exactitude inchangee (4/7), les 5 joueurs concernes sont tous a la feuille |

**Un seul fait sur quinze est une regression.** Elle est corrigee, prouvee par mutation, et
mesuree contre une chaine de controle independante du film.

Schema bumpe **41 -> 43**. **42 est RESERVE** au complement des ports de drapeau, en cours sur
une autre branche : deux chantiers paralleles ne peuvent pas revendiquer le meme numero.

---

## Methode

Les treize faits qui ne demandaient pas de re-cuisson ont ete instruits sur les artefacts deja
produits par le balayage (`reference/`, `apres3/`) et sur les FAITS DE MATCH exportes de l'API
(`facts/`), tous en LECTURE SEULE dans le scratchpad du balayage. Le checkout principal n'a recu
aucune ecriture. La cuisson de controle du fait n° 1 suit la technique du balayage : un film par
processus borne (`cmd/replay-build`, verrou `filmproc.AcquireSolo`, plafond 3 Gio, priorite
basse), racine de travail dediee ou `film_chunks`, `film_manifests`, `mvar`, `titles` et `config`
sont des jonctions en LECTURE et `data/cache/replays` un vrai dossier.

---

## Fait n° 1 — les portages de crane de `d9781168` (REGRESSION)

### 1.1 Ce que le parc dit, et ce que le code d'aujourd'hui rendait

| | artefact du parc (s23) | schema 41 |
|---|---|---|
| `skullCarries/n` | **36** | **30** |
| `coverage.skullCarries` | `trains=37 carries=36 noBridge=1` | `trains=37 carries=30 noBridge=1` **`carrierAbsent=6`** |
| par xuid | — | `2535435655459376` 8 -> 4, `2535446563676950` 4 -> 3, `2535463284150067` 2 -> 1 |

`trains`, `grabs` et `noBridge` sont IDENTIQUES : la lecture du film n'a rien perdu. Le compteur
`carrierAbsent=6` dit que c'est un GATE qui jette. Quatre portages de plus sont ROGNES sans que
le compte le montre (`492 -> 974`, `3083 -> 3138`, `4480 -> 4583`, `5796 -> 6068`).

### 1.2 La verite terrain — une chaine INDEPENDANTE du film

En Oddball, le score EST le temps de portage : une seconde de possession, un point. La feuille de
match (API, `teamScores`) donne donc la duree de portage attendue, sans jamais toucher au film.

| etat | equipe 0 | equipe 1 | total |
|---|---|---|---|
| **feuille de match (verite)** | **191 s** | **196 s** | **387 s** |
| artefact du parc (s23, 36 portages) | 172,5 s | 158,8 s | 331,3 s |
| **schema 41 (30 portages)** | **60,1 s** | 147,4 s | **207,5 s** |

Le gate fait tomber l'equipe 0 de 90 % a **31 %** de son temps reel. Il n'ecarte pas des
fantomes : il ELOIGNE l'artefact de la verite. Sur les deux autres matchs touches, meme sens —
`51ebbc0f` publie 65,6 s pour 255 s reelles (7 portages sur 14 ecartes), `24dbb67d` 248 s pour
321 s (3 sur 20).

### 1.3 La cause, sur pieces

Commit `af89b091b` (2026-08-30, « ecarter les portages attribues a un porteur ABSENT de la
carte »), `internal/analysis/replay/skull_carries.go`. `skullCarrierPresence` n'indexait que les
vies **NOMMEES** :

```go
for _, t := range tracks {
    if t.XUID == "" { continue }   // <- la vie anonyme est JETEE
    out[t.XUID] = append(out[t.XUID], presenceSpan{t.StartFrame, t.EndFrame})
}
```

et `buildSkullCarries` lisait « aucune vie nommee de X ne recouvre l'intervalle » comme « X est
absent ». Or le pont d'identite laisse des vies ANONYMES : sur `d9781168`, **18 slots sur 142**
n'ont aucune vie nommee (`livesNamed=140`, `livesTotal=176`). Verification faite portage par
portage : **les 6 portages ecartes sont TOUS recouverts par au moins une vie anonyme**, et les 4
prefixes rognes aussi. Une vie sans nom est une PRESENCE SANS IDENTITE, pas une absence.

L'ironie est dans le commentaire d'origine, qui enonçait deja le bon principe — « une presence
inconnue ne doit pas se faire passer pour une absence » — juste au-dessus du `continue` qui
faisait exactement cela.

**Le message du commit designe lui-meme ce film** (« sur d9781168, 1er portage SHROOM 326-426
mais 1re vie a 868 »), et son diagnostic est le meme fait lu a l'envers : la 1re vie NOMMEE est a
868, mais la piste anonyme du slot 518 couvre [227..534].

### 1.4 Et le symptome d'origine ? Le gate l'a aggrave

Le commit voulait reparer une « icone absente ». Or `skullCarries` sert QUATRE consommateurs web,
et un seul a besoin de la position du porteur :

- `useReplaySkullCarrier` — le calque du porteur. Son en-tete le dit : « **sans position, le crane
  ne se dessine pas** ». Il degradait deja proprement ; le symptome etait HONNETE.
- `skullPresence.ts` — la regle de presence. Sa **regle 1** est « une CARRY couvre F -> `carried` »,
  ce qui fait TAIRE le calque du crane libre. Ecarter le portage fait retomber la regle sur le
  « trou de repos », qui pose le crane a sa derniere position connue **pendant qu'un joueur court
  avec** — le fantome que l'invariant d'`objectiveObjectsLayer` interdit explicitement.
- `objectiveMark.ts` — la marque d'objectif du joueur.
- `skullSound.ts` — les SONS de prise et de chute du crane, dates sur `skullCarries`. Six portages
  ecartes = douze sons perdus, quatre `t0` rognes = quatre prises jouees en retard (jusqu'a 48 s).

Le gate echangeait donc « crane invisible » (honnete) contre « crane pose au mauvais endroit »
(faux), au prix de la donnee.

### 1.5 Le correctif

`carrierPresence` remplace la map nue : elle porte les vies NOMMEES par xuid **et les vies
ANONYMES a part**, ce qui rend l'ignorance visible au gate. `carrierPresence.gate` s'abstient
dans trois cas, tous ramenes au meme principe — **on ne rejette pas l'inconnu** :

1. le porteur n'a aucune vie nommee (deja le cas avant) ;
2. **une vie ANONYME recouvre l'intervalle** : ni rejet ni rognage ;
3. sinon, rognage a la vie nommee qui recouvre le plus — inchange.

Le rejet ne subsiste que quand les pistes publiees rendent compte de tout l'intervalle et que le
porteur n'y est pas : le seul cas ou « absent » est une mesure et non une ignorance.

**L'IGNORANCE PASSE AVANT LE ROGNAGE**, et c'est la moitie la plus couteuse : le rejet valait
32,6 s de portage, le rognage **91,2 s**. Une premiere version du correctif testait le rognage
d'abord et ne recuperait qu'un quart du manque ; le test l'a attrapee.

Le gate de la BOMBE (`bomb_carries.go`) portait la meme dizaine de lignes copiees : elle est
supprimee au profit du meme `gate`. Une seule copie, plus de derive possible. Aucun artefact du
parc ne porte `bombCarries.carrierAbsent > 0` — le defaut y etait latent, il est ferme.

### 1.6 Tests, prouves par mutation

- `TestSkullCarriesVieAnonymeNEstPasUneAbsence` — une vie anonyme couvre les deux intervalles :
  aucun portage ecarte, aucun rogne. **Rouge sans le correctif** (« portages = 3, attendu 4 »).
- `TestSkullCarriesFantomeResteEcarte` — la CONTRE-EPREUVE : vie anonyme loin des portages, le
  fantome reste ecarte et le rognage s'applique. Le correctif RETRECIT le gate, il ne le supprime
  pas.
- `TestSkullCarrierPresence` — l'index retient desormais les anonymes au lieu de les jeter.

---

## Fait n° 2 — l'episode de surbouclier de `2cf24f30` (ancien artefact FAUX)

L'episode perdu est `{"slot":560,"fam":"overshield","t0":2929,"t1":2929}` — **duree NULLE**. La
piste du slot 560 finissait, au parc, sur un point isole a **t=2929** : `(-14,88 · -209,16 ·
-304,88)`, alors que son point precedent, 12,5 s plus tot, est a `(11,29 · -109,03 · +57,06)`.
Une chute de 362 unites en Z sur une carte dont le reste tient entre +53 et +61.

Les bornes de scene le prouvent : `minZ` passe de **-555,04** a **+52,99**, `minX` de -109,55 a
-6,40. L'assainissement des trajectoires a supprime le point ; l'episode qui s'y ancrait est parti
avec lui.

C'est **exactement** la reserve deja documentee pour `13d92593` dans la chronique de
`document.go` (« un episode ancre sur un point de trajectoire ABERRANT reste ecarte, et c'est
voulu [...] il ne doit pas revenir »). Rien a corriger — et le seul residu reel de la candidate 3
du balayage se referme ainsi sur la meme raison que la reserve connue.

---

## Faits n° 3 a 5 — les vies nommees perdues (`24dbb67d`, `4f77afc1`, `084a804d`)

Les trois perdent **exactement une vie nommee d'un seul joueur**, et perdent en meme temps 2 a 9
points de trajectoire. La mecanique est une seule :

1. la vie publiee du parc finissait sur un point ISOLE, tres eloigne dans le temps ;
2. `decimateTracks` coupe a `lifeGapUS` (5 s) : ce point devient une vie a lui tout seul ;
3. `minPoints` jette une vie d'un point — le point disparait, la vie publiee RACCOURCIT ;
4. `nameLivesByDeaths` apparie une mort a la FIN de vie la plus proche : la mort tombe desormais
   sur la vie d'un point, non publiee. La vie publiee reste anonyme.

Ce qui les separe, c'est la NATURE du point isole :

| match | slot | dernier point du parc | verdict |
|---|---|---|---|
| `4f77afc1` | 641 | t=10577, `(79,95 · -186,66 · **-387,05**)` contre `(20,75 · -11,50 · +107,61)` 566 s plus tot | point **aberrant** : sa suppression est un gain |
| `084a804d` | 539 | t=2739, `(195,61 · -200,96 · **-370,22**)` contre `(10,46 · -41,43 · +91,33)` | point **aberrant** : idem |
| `24dbb67d` | 586 | t=4097, `(17,62 · -15,98 · 4,40)` contre `(17,64 · -15,99 · 4,40)` 31,9 s plus tot | **corps IMMOBILE** — 2 cm d'ecart |

Les deux premiers : **ancien artefact faux**, la geometrie qu'il publiait etait mensongere.

Le troisieme est la **decouverte n° 1 deja enregistree** par l'instruction des regressions 2-4
(« une vie coupee par un trou de replication a corps immobile [...] `lifeGapUS` coupe une vie que
rien ne separe [...] decision produit, hors mandat »), constatee la sur `145908d1` slot 562. Meme
signature au centimetre pres. Rien de nouveau, rien a corriger ici : la condition de reprise est
deja au registre des reports.

---

## Faits n° 6 a 9 — `pickups.originGround` (`1b2d9e08`, `1cd3848a`, `3923bede`, `e85d7bad`)

Les quatre sont le MEME evenement, et ce n'est pas une perte :

| match | `spawnPointsState` | `mapCatalogPoints` | `originUnknown` | `originSpawner` | `originGround` |
|---|---|---|---|---|---|
| `1b2d9e08` | `map_absent` -> **`established`** | 0 -> 17 | 91 -> 28 | 0 -> **65** | 16 -> 14 |
| `1cd3848a` | `not_established` -> **`established`** | 0 -> 49 | 41 -> 33 | 0 -> **9** | 15 -> 14 |
| `3923bede` | `not_established` -> **`established`** | 0 -> 34 | 117 -> 38 | 0 -> **81** | 9 -> 7 |
| `e85d7bad` | `not_established` -> **`established`** | 0 -> 34 | 72 -> 22 | 0 -> **51** | 7 -> 6 |

Le catalogue de socles de la carte est desormais ETABLI. Des ramassages qui n'avaient aucune
origine (`unknown`, de 41 a 117 par match) sont reconnus « pris au SOCLE », et un ou deux qui
etaient etiquetes « ramasse au sol » sont reconnus pour ce qu'ils sont : un objet pris sur son
socle. **Reclassement a l'interieur d'un gain massif** (jusqu'a +81 origines resolues). Rien a
corriger.

---

## Faits n° 10 et 11 — les capacites (`21ece4d8`, `000d5950`)

**`21ece4d8` (23 -> 22).** L'unique capacite perdue est `{"t":2407,"slot":576,"r":3,"src":"i48"}`.
La piste du slot 576 est **identique** dans les deux artefacts : `[2857..3032]`, 175 points. La
capacite est datee a **t=2407**, soit 450 frames AVANT que le slot ait la moindre piste publiee —
et aucune autre piste ne porte ce slot. Le client n'aurait rien a quoi la rattacher. L'autre
capacite du meme slot (t=3001, dans la fenetre) est conservee. Ancien artefact faux.

**`000d5950` (4 -> 2).** L'artefact de reference est au **schema 2** (2026-08-03). Son
`abilityLabels` porte les cles `3,4,5,6` (mur portatif, grappin, propulseur, capteur de menace) —
et `abilities` y est **absent** : quatre libelles pour une table que rien ne referencait. Le
document d'aujourd'hui publie **214 capacites**, 69 charges et 37 impulsions, dans deux familles
seulement (`grapple` 31, `thruster` 38) — donc **deux** libelles, cles `20` et `21`, avec leur
image. Le dictionnaire ne liste plus que ce qui existe. Espace de cles mort d'un cote, table
referencee de l'autre : ancien artefact faux.

---

## Fait n° 12 — `11de8353`, `coverage.score.points` 506 -> 504

Diff integral des deux courbes de score : **une seule difference**. L'equipe 1 portait un point
de tete `{"t":0,"v":0}`, present a la fois dans `rounds[0].points` et dans `total` — d'ou 2 points
de moins. L'equipe 0 n'en avait pas. Aucune valeur de score ne change, aucun instant ne bouge :
l'ancre nulle redondante disparait et les deux camps deviennent coherents. Normalisation, pas
perte.

---

## Fait n° 13 — `bf15f7ab`, `weaponLabels/n` 17 -> 16

Le libelle disparu est `0x3E070217` « Carabine Vestige ». Il n'etait reference que par UN endroit,
`weaponPads[7].weapon` — un socle situe a l'ORIGINE du monde `(-0,00089 · -0,00003 · 0,0018)`. Les
douze socles sont la des deux cotes ; c'est l'ARME de celui-la qui change, en `0x30484EA6`
« Carabine a impulsion ».

Qui a raison ? Le film lui-meme :

- `0x3E070217` n'apparait **nulle part ailleurs** dans l'artefact du parc, et **zero fois** dans
  celui d'aujourd'hui ;
- `0x30484EA6` est atteste : **2 armes au sol** et **1 ramassage** `hinf_pulse_carbine`.

L'ancienne identification n'etait corroboree par rien, la nouvelle l'est par deux canaux. Le
dictionnaire retrecit parce qu'un code errone a disparu. Ancien artefact faux.

---

## Faits n° 14, 15 et `bcb6d393` — les re-attributions dans un gain

Verifies en dernier et rapidement, comme prescrit. Les trois sont confirmes, sur pieces.

**Ce que mesure l'axe `joueurs` du comparateur** : la DERNIERE valeur des courbes de score
par joueur (`scoreTimeline.players[].{score,kills,deaths,assists}`) — donc ce que le FILM
reconstruit, pas ce que l'API dit. La feuille de match est donc un oracle independant.

| match | etat | joueurs a courbe publiee | **exacts contre la feuille** | points de courbe |
|---|---|---|---|---|
| `43716616` | parc s21 | 1 | 1 | 516 |
| | schema 41 | **8** | **4** | **988** |
| `51ebbc0f` | parc s21 | 3 | 0 | 750 |
| | schema 41 | **8** | 0 | **1 112** |
| `bcb6d393` | parc s20 | 7 | 4 | — |
| | schema 41 | 7 | 4 | — |

Les deux premiers sont des films a **DEUX MANCHES** (`coverage.score.rounds = 2`), et leur
identite d'equipes passe de `unresolved` a `a`. Le joueur unique du parc etait nomme par le pont
par TRIPLET, qui apparie des **TOTAUX DE MATCH** — ce que le chantier CTF a etabli comme n'ayant
« aucun sens sur un film multi-manche ». Que son chiffre ait colle a la feuille n'est pas une
mesure du film : c'est la tautologie du pont qui l'a choisi pour cette raison meme. Le pont par
manche mesure, lui, ce que le film montre slot par slot — et publie huit joueurs au lieu d'un.

Pour `bcb6d393` : 67 -> **76** actions d'objectif (+4 `kills`, +2 `flag_steals`, +1 `flag_captures`,
+1 `flag_grabs`, +1 `assists`). Les 3 actions `kills` qui changent de main vont de deux joueurs a
trois autres, et **les cinq sont a la feuille de match** — aucune attribution fantome d'un cote ni
de l'autre. Exactitude identique, volume superieur.

Aucun des trois n'est une regression. Rien a corriger.

---

## Schema

`SchemaVersion` **41 -> 43** ; chronique ecrite dans `document.go`, raison dans le ratchet
`structure_test.go`. **42 est saute et reserve** au complement des ports de drapeau, en cours sur
une autre branche.

Golden d'assemblage regenere : son **unique** ecart est la ligne de version (`schema 41` ->
`schema 43`, 1 ligne sur 605, verifie par diff AVANT regeneration) — le correctif ne change rien
sur le film de reference `000d5950`, qui n'est pas un Oddball.

Le contrat OpenAPI declare `schemaVersion` comme un `integer` sans `enum`, `const`, `default` ni
`example` : un bump ne le deplace pas.

---

## Impact sur le parc

Trois matchs du parc portent `coverage.skullCarries.carrierAbsent > 0` — `d9781168` (6 portages
sur 36), `51ebbc0f` (7 sur 14), `24dbb67d` (3 sur 20) — et tout artefact Oddball cuit depuis le
**2026-08-30** est concerne des lors que son film porte une vie anonyme sur un intervalle de
portage. Le rognage, lui, ne laisse aucun compteur : il touche potentiellement tous les Oddball,
et il coute trois fois plus que le rejet. La propagation passe par la re-cuisson de release
(`backfill-replay`), que le bump 41 -> 43 rend obligatoire.

Les quatorze autres faits ne demandent aucune re-cuisson : le parc est deja plus juste que ses
anciens artefacts sur chacun d'eux.

---

## Decouvertes, notees et NON traitees

1. **`51ebbc0f` publie `assists = 63` pour `2535439712156981`**, quand la feuille en donne **5**.
   Valeur APPARUE (le joueur n'avait aucune courbe au parc), donc invisible au balayage, qui ne
   compare que ce qui existait des deux cotes. Le film a deux manches et ce meme match ne rend
   AUCUNE courbe exacte sur huit joueurs : la reconstruction par manche y est douteuse de bout en
   bout. A instruire pour lui-meme.
2. **Aucune des 8 courbes de `51ebbc0f` ne colle a la feuille**, contre 4 sur 8 pour `43716616`
   (meme nombre de manches). L'ecart entre les deux films n'est pas explique.
3. **Le comparateur ne voit pas les rognages.** Un intervalle raccourci ne change ni `n` ni le
   nombre de champs presents : `d9781168` perdait 91,2 s de portage sans qu'aucune ligne de
   `diffs3.tsv` ne le dise, contre 32,6 s pour les rejets, qui eux se voyaient. Un axe « somme des
   durees » par calque d'intervalles fermerait cet angle mort du balayage.
