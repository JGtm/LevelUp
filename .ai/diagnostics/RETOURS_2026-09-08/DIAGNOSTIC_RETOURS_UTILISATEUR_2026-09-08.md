# Diagnostic des retours utilisateur — 2026-09-08

Registre d'audit sur pièces. **Aucun code n'a été modifié** (doctrine `adversarial-audit` : l'audit
ne corrige pas). Chaque constat porte sa preuve reproductible.

## Conditions de mesure

| Élément | Valeur |
|---|---|
| Branche | `feat/v75` @ `7254b3853` |
| API | binaire reconstruit depuis HEAD (`go build ./cmd/server`), `LEVELUP_AUTH_MODE=none`, port 8000 |
| Web | Vite dev, port 5173 |
| Joueur de référence | `JGtm` (xuid `2533274823110022`) |
| Données | parc local : 1 967 matchs, 129 artefacts de rejeu (tous schéma 48) |
| Captures | [`captures/`](captures/) |

**Piège écarté d'entrée** : le binaire `tmp_server.exe` présent à la racine date du 1er septembre.
Il ne déclare **ni la capability `replay` ni `weapon_range`** et rendait la page de rejeu
« Indisponible pour ce titre ». Tout ce qui suit a été mesuré sur un binaire reconstruit depuis
HEAD, qui les déclare bien (vérifié : `GET /api/v1/bootstrap` → `capabilities` contient `replay`
et `weapon_range`).

---

## LA CAUSE RACINE COMMUNE : `publishable = false`

**Un seul mécanisme explique les points 2, 12, 14 et une partie du 19 et du 21.**

`match_kill_events.publishable` est un attribut de la PASSE DE DÉCODAGE, pas de la ligne :
`killsource.Result.LineByLinePublishable()` (`apps/go-api/internal/games/halo_infinite/film/killsource/kill.go:331`)
rend `false` dès que la marge de bijection est nulle (deux joueurs interchangeables) OU que la
santé du décodage est en alerte. Il vaut alors : « les lignes sont justes EN AGRÉGAT et fausses
INDIVIDUELLEMENT ».

**Mesure sur le parc :**

```bash
./tmp_diag_q.exe data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb "
WITH m AS (SELECT match_id, max(CASE WHEN publishable THEN 1 ELSE 0 END) AS pub
           FROM match_kill_events_latest GROUP BY 1)
SELECT pub, count(*) AS matchs FROM m GROUP BY 1 ORDER BY 1"
```

| `publishable` | matchs |
|---|---|
| 0 (aucune ligne publiable) | **498** |
| 1 | 886 |

**498 sur 1 384 matchs porteurs de lignes de kill, soit 36 %** — et **5 des 8 derniers matchs
synchronisés (2026-09-07)** :

| Carte | Date | lignes | publiables |
|---|---|---|---|
| Domicile (Slayer) | 07/09 20:24 | 77 | **0** |
| Absolution (CTF) | 07/09 20:14 | 81 | 81 |
| Domicile (CTF) | 07/09 20:10 | 27 | 27 |
| **Origin (CTF)** | 07/09 20:00 | 81 | **0** |
| Fortress (Zones) | 07/09 19:52 | 66 | **0** |
| Illusion (Zones) | 07/09 19:42 | 67 | **0** |
| Isolation (Zones) | 07/09 19:34 | 43 | 43 |
| Banished Narrows | 07/09 19:26 | 72 | 72 |

**Les six lecteurs qui exigent ce drapeau** (donc six blocs vides ensemble sur les mêmes matchs) :

| Lecteur | Fichier:ligne | Ce qui s'éteint |
|---|---|---|
| Sources d'arme du kill feed | `platform/duckdb/queries_match.go:501` | icônes d'armes du fil (point 2) |
| Victimes du kill feed | `platform/duckdb/queries_match.go:540` | victime nommée |
| Paires d'assistance (match) | `platform/duckdb/match_view_repo_assist_pairs.go:91` | graphe assistances (point 12) |
| Paires d'assistance (escouade) | `platform/duckdb/queries_squad.go:257,270` | tableau escouade |
| Distances mesurées | `platform/duckdb/kill_measured.go:173` | « Distance par arme » (point 14) |
| Tactique | `platform/duckdb/tactical_repo.go:213,280` | plan tactique (point 21) |

> **À trancher (escalade)** : ce drapeau est aujourd'hui **binaire et par match**. La question
> produit est de savoir si la marge de bijection nulle doit éteindre TOUT (état actuel) ou
> seulement les lignes dont l'attribution est réellement ambiguë. Ce n'est pas un bug, c'est une
> décision d'architecture — elle t'appartient.

---

## HAUTE PRIORITÉ

### 1. Médailles absentes de la frise sous le Replay — **l'anneau est un mauvais encodage : décision utilisateur du 2026-09-08**

**Verdict : confirmé. La donnée est là, la forme est à refaire.**

> **Décision utilisateur du 2026-09-08, qui tranche et remplace les décisions 9 et 14 du plan
> `.ai/PLAN_FRISE_POINT_DE_VUE_2026-09-06.md` :** *« Pourquoi une médaille serait un anneau ? Une
> médaille est une image, qui a un titre et une description en tooltip. »*
>
> Le dépôt sait déjà le faire et le fait ailleurs : `ui/MedalBadges.tsx` affiche les badges EN
> IMAGES, et le kill feed les monte déjà (`ReplayKillFeed.tsx:344` et `:508`). Le document publie
> `medal_image_url`, `medal_label` et `medal_description` par médaille
> (`killFeedLogic.ts:86-92`). La frise est le seul lecteur à avoir choisi une abstraction.
>
> L'argument d'origine — « à trois pixels de large une icône serait une tache » — est un argument
> contre la LARGEUR de la marque, pas contre l'image : c'est la piste qui doit accueillir le
> badge, pas le badge qui doit se réduire à un anneau.

Les médailles n'ont aujourd'hui pas de glyphe : décision 9 du plan
`.ai/PLAN_FRISE_POINT_DE_VUE_2026-09-06.md` — « pas de glyphe supplémentaire quand la médaille est
attachée à un kill », elle décore la marque existante d'un ANNEAU. **Cette décision est annulée.**

Mesure DOM sur le match Origin (`8bc6074f`), point de vue JGtm :

```js
// 48 marques sur la frise, 1 seule porte l'anneau
{ cls: "... top-[5px] h-2 w-[3px] rounded-[2px] ring-1 ring-foreground",
  title: "3:49 — Revirement", w: 3, h: 8 }
```

**Un anneau de 1 px autour d'une barre de 3 × 8 px** (`ReplayMarkTrack.tsx:103`). Il est présent,
il est juste invisible à l'œil.

Second facteur, quantifié : sur ce match JGtm n'a **qu'une seule médaille** (`highlight_events`
event_type=`medal` : 1 ligne ; `medals_earned` : 1). Il n'y a donc qu'un anneau à trouver sur 48
marques.

```bash
./tmp_diag_q.exe data/titles/halo_infinite/warehouse/shared_matches_v2.duckdb "
SELECT xuid, count(*) FROM highlight_events
WHERE match_id='8bc6074f-d001-428b-8d6a-a755f0925572' AND event_type='medal' GROUP BY 1"
```

### 2. Icônes d'armes absentes du killfeed sur certains replays — **cause racine identifiée**

**Verdict : confirmé. Cause = `publishable = false` (cf. section commune).**

Preuve par comparaison des deux matchs du même soir :

| Match | publiables | masques d'icône d'arme dans le DOM |
|---|---|---|
| Origin `8bc6074f` | 0 / 81 | **0** |
| Isolation `81c02726` | 43 / 43 | **49** (`silhouette-00.png`, `Frag.png`, `Repulsor.png`…) |

Le champ `WeaponImageURL` n'est peuplé que si `sourceByKill` contient la ligne
(`service/match_view_killfeed_weapon.go:187` : `src, ok := sourceByKill[key]; if !ok { continue }`),
et `sourceByKill` vient de Q21b, filtrée `AND publishable`.

### 3. Fiches rognées par les longs gamertags — **cause racine trouvée, correctif d'une ligne**

**Verdict : confirmé, prouvé par mutation live.**
Capture : [`02-fiches-rognees-origin.png`](captures/02-fiches-rognees-origin.png), [`08-ecran-fin-decentre.png`](captures/08-ecran-fin-decentre.png)

`apps/web/src/features/match-replay/ui/ReplayTeams.tsx:184` :

```tsx
style={{ gridTemplateColumns: `repeat(${groups.length}, 1fr)` }}
```

`1fr` vaut `minmax(auto, 1fr)` : le minimum `auto` est le **min-content** de la piste, que le
gamertag le plus long impose (le `truncate` du nom pose `white-space: nowrap`, donc son min-content
est le texte ENTIER). La grille dépasse alors son conteneur, que `overflow-hidden` rogne en silence.

Mesure dans le navigateur, puis mutation :

| | `grid-template-columns` calculé | clientWidth | scrollWidth |
|---|---|---|---|
| Tel quel | `279.688px 273.094px` | 480 | **563** (83 px rognés) |
| Après `repeat(2, minmax(0, 1fr))` | `235px 235px` | 480 | **480** |

Après le correctif, les trois gamertags longs (`XXDAEMONGAMERXX`, `BIOLY GOLAB1054`,
`CRUMBLYDONUT617`) passent en `truncate` effectif (`scrollWidth > clientWidth`) — exactement le
comportement demandé.

### 4. Joueurs attribués à la mauvaise équipe (match Origin) — **CONFIRMÉ : deux identités sont interverties sur les 77 premières secondes**

> **Correction du 2026-09-08.** Une première passe avait conclu « réfuté » sur deux preuves
> insuffisantes (absence de frag allié, scoreboard API conforme). Elles ne pouvaient pas voir le
> défaut : il est dans le pont slot → joueur du FILM, pas dans le tableau de score. Le constat de
> l'utilisateur est exact.

**La preuve : les positions de spawn.** Origin est une carte CTF symétrique, bases à `x ≈ −22`
(camp `t0`) et `x ≈ +22` (camp `t1`). À l'image 0 :

| xuid | gamertag | équipe (scoreboard) | x au spawn | base |
|---|---|---|---|---|
| 2533274823110022 | JGtm | `t0` | −22,52 | ouest ✔ |
| 2533274858283686 | Madina97294 | `t0` | −22,01 | ouest ✔ |
| 2533274833178266 | XxDaemonGamerxX | `t0` | −22,03 | ouest ✔ |
| **2535469190789936** | **Chocoboflor** | **`t0`** | **+22,62** | **est ✘** |
| 2535449383340628 | Hanover Cat | `t1` | +23,10 | est ✔ |
| 2535461109438273 | biOly goLab1054 | `t1` | +22,59 | est ✔ |
| 2535450759128461 | CrumblyDonut617 | `t1` | +22,xx | est ✔ |
| **2535452521259564** | **StevenW5318** | **`t1`** | **−22,51** | **ouest ✘** |

**Interversion propre de DEUX joueurs**, un de chaque camp. Chocoboflor est dessiné dans la base
adverse, avec l'équipe adverse — exactement ce que tu décris.

**Et elle se résorbe.** Les vies suivantes retombent du bon côté :

| vie | slot | joueur | image de départ | x | verdict |
|---|---|---|---|---|---|
| 1 | 516 | StevenW5318 (`t1`) | 0 | **−22,51** | ✘ inversé |
| 1 | 518 | Chocoboflor (`t0`) | 1 | **+22,62** | ✘ inversé |
| 2 | 516 | StevenW5318 (`t1`) | 229 | **−22,49** | ✘ inversé |
| 2 | 518 | Chocoboflor (`t0`) | 270 | **+22,62** | ✘ inversé |
| 3 | 525 | StevenW5318 (`t1`) | 874 | **+13,74** | ✔ correct |
| 3 | 526 | Chocoboflor (`t0`) | 875 | **−13,09** | ✔ correct |

L'interversion couvre les **images 0 à ~774, soit les 77 premières secondes** — le début du match,
là où tu l'as vue.

**Ce n'est pas un accident silencieux : le décodeur le SAIT.** C'est très exactement la condition
qui met `publishable = false` sur ce match — `LineByLinePublishable()` rend faux quand
`BijectionMargin == 0`, c'est-à-dire quand **« au moins deux joueurs sont interchangeables »**
(`killsource/kill.go:322`). Le décodeur a signalé qu'il ne savait pas départager deux joueurs, et
c'est bien deux joueurs qui sont intervertis.

**Le défaut de conception est là : la porte n'est appliquée qu'aux lecteurs de TEXTE.**
`replaybuild/kills.go:79` et `replaybuild.go:405` consultent `LineByLinePublishable()` pour taire
le kill feed et les morts sans revendication. **Rien ne la consulte pour la GÉOMÉTRIE** : les
pistes, les pions de la carte et les fiches joueur sont publiés avec les identités douteuses, sans
réserve ni signalement. Le fil se tait pendant que la carte affirme.

**Les métriques de santé du pont ne l'attrapent pas non plus.** `coverage.bridge` de cet artefact
annonce `indexDisagreements: 0`, `slotCollisions: 0`, `unnamedLives: 3` — tout va bien de son point
de vue. Seuls `closedContested: 9` et `deathOffsetRunnerUp: 10` signalent une tension, et aucun
n'est un critère de refus.

**Pourquoi mes deux « preuves » initiales ne valaient rien :**
- l'absence de frag allié se lit sur `match_kill_events`, qui **n'est pas publié du tout** ici
  (`publishable = 0` sur 81 lignes) — je comparais des équipes sur des lignes que le produit
  n'affiche jamais ;
- le scoreboard API est correct **par construction** (il vient de l'API Halo, pas du film) : il ne
  pouvait pas révéler un défaut du film.

**Reproduction :**

```bash
jq -r '[.tracks[]? | select(.xuid != null)
        | {xuid, slot, t0:(.points[0].t), x0:(.points[0].x)}]
       | map(select(.t0 <= 900)) | sort_by(.t0)' \
  data/cache/replays/halo_infinite/8bc6074f.json
```

#### Ampleur mesurée sur le parc — et `publishable` NE PRÉDIT PAS le défaut

Balayage automatique des 64 artefacts (script `captures/../swap.sh`, méthode : médiane de spawn par
camp d'après le scoreboard, puis comptage des joueurs plus proches du camp adverse que du leur ;
**30 artefacts** ont une séparation de spawn suffisante — ≥ 8 m — pour que le test conclue) :

| artefacts testables | avec ≥ 1 joueur mal placé | avec exactement 2 (signature d'interversion) |
|---|---|---|
| 30 | **7** | **4** |

Et voici ce qui change tout — la corrélation avec le drapeau du décodeur :

| artefact | carte | mal placés | `publishable` |
|---|---|---|---|
| `8bc6074f` | Origin | 2 | **0 / 81** |
| `4f77afc1` | Flood Gulch | 1 | 0 / 294 |
| **`a4083bd2`** | **The Pit** | **2** | **95 / 95** |
| **`bf2a9f05`** | **Bazaar** | **2** | **96 / 96** |
| **`d8b13ec2`** | **Goliath** | **2** | **97 / 97** |
| **`e60aaf06`** | **Banished Narrows** | **1** | **72 / 72** |
| `72b0a25e`, `9ffce8ef`, `b0fe12b1`, `c7f94693` | — | **0** | 0 (non publiables) |

**`publishable` n'est ni nécessaire ni suffisant** : trois des quatre interversions propres
surviennent sur des matchs que le décodeur déclare **entièrement publiables**, et quatre matchs
déclarés non publiables ne montrent aucune interversion.

**Vérification manuelle de `d8b13ec2` (Goliath, Team Slayer, 97/97 publiables)** — les spawns
forment deux grappes nettes, à `y ≈ +12,5` et `y ≈ −24` :

| grappe | joueurs (scoreboard) |
|---|---|
| Nord (`y ≈ +12,5`) | `…823110022` (t1) · `…858283686` (t1) · `…469190789936` (t1) · **`…974148269` (t0)** |
| Sud (`y ≈ −24`) | `…873595385` (t0) · `…001807081` (t0) · `…413036598415` (t0) · **`…434888128184` (t1)** |

Interversion propre de deux joueurs, un de chaque camp, sur un match sans la moindre réserve du
décodeur.

> **Conséquence directe sur le correctif envisagé au § « Suite » :** appliquer
> `LineByLinePublishable()` à la géométrie **ne suffirait pas** — le défaut se produit là où ce
> drapeau est vert. Il faut une garde propre à l'attribution : le camp spawne groupé, et cette
> information n'est pas exploitée par le pont. **C'est LA piste, et elle reste à instruire.**

**Second constat, distinct, sur le même panneau :** à `0:00` la colonne COBRA n'affiche que 4
fiches alors que le camp en compte 5. « Ady 01 » n'apparaît qu'en fin de match, à la place de
« biOly goLab1054 ». C'est le **relais de siège** (`model/seatLogic.ts:82`) : un arrivant apparié à
un partant disparaît de la liste des sièges et vit dans celui de son prédécesseur. Intentionnel et
documenté, mais rien à l'écran ne dit qu'un siège est partagé.

### 5. Fond de carte absent sur le dernier Isolation — **cause racine arithmétique, exacte**

**Verdict : confirmé.** Capture : [`03-isolation-sans-fond-de-carte.png`](captures/03-isolation-sans-fond-de-carte.png)

L'API sert bien le fond : sidecar `200` avec calage valide, image `200` / 413 832 octets.
**C'est le client qui l'écarte**, à `hooks/useReplayView.ts:178` :

```ts
return coversPlayedArea(background.calibration, doc.bounds) ? background : null
```

`coversPlayedArea` (`layers/mapBackground.ts:108`) exige un **contenant STRICT** des bornes BRUTES :

| | X | Y |
|---|---|---|
| Emprise de l'image | −61,33 → +11,89 | −49,06 → +3,43 |
| `doc.bounds` (min/max bruts) | **−78,60** → −1,14 | −37,06 → **+46,38** |

Deux dépassements ⇒ `false` ⇒ pas de fond, **et aucun log**.

Or **99 % des positions tiennent dans l'image** (16 064 échantillons) :

| axe | p0 | p1 | p50 | p99 | max |
|---|---|---|---|---|---|
| X | −78,60 | −48,98 | −25,65 | −2,12 | −1,14 |
| Y | −37,06 | −35,40 | −23,10 | −8,27 | **+46,38** |

**Deux points aberrants (une chute hors terrain) suppriment le fond du match entier.** Le garde
teste `min/max` là où il devrait tester une emprise robuste (p1/p99), ou au minimum journaliser.

### 6. Joueur en véhicule non affiché, « Réapparition dans… » pendant des minutes — **CONFIRMÉ : le balayage des véhicules n'a jamais tourné sur ces artefacts**

**Verdict : confirmé, cause racine trouvée. Ce n'est pas « pas de véhicule sur la carte ».**

Les deux derniers Behemoth (`7b0d89c4` 50-44, `f2966f08` 49-50, tous deux **défaites**) n'ont pas
seulement `vehicles: []` — **le bloc `coverage.vehicles` est ENTIÈREMENT ABSENT** de leurs
artefacts. Comparaison avec `1cd3848a` (Behemoth Super Fiesta), qui l'a :
`{"scanned": true, "lives": 40, "published": 21, "rides": 7, …}`.

Absence du bloc = **le balayage n'a pas été exécuté**, pas « il n'a rien trouvé ».

**Ce n'est pas isolé.** Distribution des versions de schéma sur les 64 artefacts du parc local :

| schéma | artefacts | `coverage.vehicles` |
|---|---|---|
| **38** | **15** | **absent sur les 15** |
| 48 | 49 | présent |

**Les 15 artefacts sans balayage véhicules sont exactement les 15 restés au schéma 38** — cuits
avant que le balayage n'existe, et jamais recuits. Les deux Behemoth en font partie, avec
`0891225f`, `0d265ab0`, `1b2d9e08`, `28c9b538`, `30a23d15`, `4ecdf3e7`, `72b0a25e`, `94a28b8b`,
`a03a5e65`, `bfecd02b`, `cde26226`, `f0220a96`, `faff9935`.

Ton souvenir de mongooses et d'un warthog sur ce Behemoth est donc parfaitement compatible avec la
mesure : **les véhicules étaient là, l'artefact ne les a jamais regardés.**

Le mécanisme de l'affichage suit : un occupant attaché **cesse de répliquer la position de son
bipède** (`analysis/replay/vehicle_shots.go`, en-tête). Sans épisode d'occupation publié, la fiche
n'a plus de position et retombe sur « Éliminé + Réapparition dans » (`i18n.ts:295-296`) — pendant
toute la durée de la chevauchée.

**Reproduction :**

```bash
for f in data/cache/replays/halo_infinite/*.json; do case "$f" in *derived*) continue;; esac
  c=$(jq -r 'if (.coverage|has("vehicles")) then 1 else 0 end' $f)
  [ "$c" = "0" ] && echo "$(basename $f .json) schema=$(jq -r .schemaVersion $f)"
done
```

**Correctif : recuire les 15 artefacts au schéma 38.** Ce n'est pas un bug de code — c'est un parc
non convergé.

---

## AUTRES POINTS

### 7. Page Tactique absente du menu déroulant L1 — **confirmé, une entrée manquante**

Capture : [`01-nav-L1-ascension-sans-tactique.png`](captures/01-nav-L1-ascension-sans-tactique.png)

`components/shell/navL1Sections.tsx`, section `ascension`, `tabs` :
`profile` · `objectives` · `coaching` · `realisations` — **pas de `tactique`**.

La route existe (`routes/{-$lang}/t/$titleSlug/players/$playerSlug/ascension/tactique.tsx`) et
l'onglet L2 de la page l'affiche bien (5 onglets : Profil · Objectifs · Entraînement · Réalisations
· **Tactique**). Seul le dropdown L1 en compte 4. Le menu mobile lit la même source et a le même
trou.

### 8. Pas d'effet UI ni sonore au tir en véhicule — **confirmé, et la cause n'est PAS le câblage**

Le câblage est complet : `sound/vehicleShotSound.ts` (table de jointure + sons Wwise reconstruits),
`model/shotFx.ts` (`VehicleShotSource`), `model/vehicleWeaponMounts.ts` (placement au montage).

**Ce qui manque, c'est l'entrée.** `Shot.Vehicle` (`json:"v"`) n'est jamais peuplé : sur les 129
artefacts locaux, **aucun tir ne porte la clé `v`** sauf dans 2 documents.

`coverage.vehicles` du match Isolation (qui a pourtant 3 chevauchées, dont 2 en Ghost) :

```json
{ "rides": 3, "shots": 0, "shotsNoRide": 188, "shotsVehicleWeapon": 0 }
```

Balayage du parc : **2 artefacts sur 129** publient des tirs de véhicule (`4f77afc1` : 155,
`5676a9ba` : 125). Partout ailleurs `shots = 0` avec jusqu'à 2 957 tirs écartés « sans
chevauchée ». La seconde porte des tirs (`attachVehicleShots`) ne rattache quasiment rien.

**Direction du cône de visée en véhicule — ta proposition inverse la décision mesurée.**
Le cône suit aujourd'hui la visée de l'OCCUPANT (schéma 39, `model/vehiclesAim.ts`), avec repli sur
le cap du châssis après 1 s. L'en-tête du fichier documente la mesure du lot V11 : justesse
0,2-0,5° contre référence, et **l'écart au cap du châssis était de 15,7 à 21,8° en médiane (q3
39,6-52,9°)** — c'est l'erreur que faisait l'ancien cône, qui utilisait justement la direction du
véhicule.

> **À trancher** : tu demandes de revenir à la direction du véhicule pour la lecture tactique. C'est
> légitime en tant que choix produit, mais c'est un retour en arrière sur une mesure. Une troisième
> voie existe : **les deux** — cap du châssis en trait plein, visée de l'occupant en secteur. Dis-moi.

### 9. Élargir la barre verticale de lecture — **confirmé**

`ui/ReplayPlayhead.tsx` : `className="absolute inset-y-0 w-px bg-foreground/40"`.
Mesure DOM : **largeur 1 px**, opacité 40 %. Une seule classe à changer.

### 10. Cercle autour du drapeau au sol + tirs en portant le drapeau

**10a — le cercle existe.** `layers/flagReturnZone.ts` est câblé (`useReplayFlagCarries.ts:187`) et
le document Origin publie bien ses paramètres : `flagReturnZone: {radiusM: 1.3, resetSeconds: 30,
soloSeconds: 3.1}`, avec des spans `state: "dropped"` portant `x`/`y`. **La donnée et le calque sont
là.** Si tu ne le vois pas, c'est un défaut de rendu à isoler — donne-moi l'instant exact.

**10b — les tirs pendant le portage : confirmé et quantifié, et TON hypothèse est la bonne.**

Sur Origin, **124 événements de tir tombent pendant un span `carried` du même joueur** — impossible
en jeu. Leur distribution le date :

| position du tir dans le span de portage | part |
|---|---|
| dans les 20 premières frames (2 s) | **0 %** |
| dans les 20 dernières frames (2 s) | **48 %** |
| médiane avant la fin du span | **21 frames (2,1 s)** |

L'asymétrie est décisive : **le span de portage se ferme trop tard**, d'environ 2 s. Le joueur lâche
le drapeau, tire aussitôt, et la lecture le dit encore porteur. Ce n'est pas « on manque des
lectures » : c'est l'instant de lâcher qui est détecté en retard.

Reproduction : `jq` sur `data/cache/replays/halo_infinite/8bc6074f.json` (script dans
[`requetes.md`](requetes.md)).

### 11. Beaucoup de grenades, très peu d'explosions — **partiellement réfuté sur Origin, confirmé ailleurs**

Le film **ne porte aucun événement de détonation** (`layers/explosionFx.ts`, en-tête). L'explosion
est posée à la **dernière position répliquée du projectile**, ce qui exige le lien
lancer → projectile (`grenadeFx.ts:156` : `if (g.proj === undefined || g.proj === null) continue`).

Taux de liaison mesuré :

| match | grenades | liées | taux |
|---|---|---|---|
| **Origin `8bc6074f`** | 76 | 64 | **84 %** |
| Isolation `81c02726` | 48 | 48 | 100 % |
| Domicile `b1ad85eb` | 121 | 114 | 94 % |
| Behemoth `7b0d89c4` | 80 | 19 | **24 %** |
| Absolution `f8efc5ca` | 67 | 14 | **21 %** |
| **Parc entier** | 7 317 | 5 893 | **80 %** |

> **Recadrage utilisateur du 2026-09-08 — et il déplace le défaut :** *« le nombre ne me dérange
> pas, qu'il soit faux ou non est un autre sujet. Qu'on détecte des lancers et qu'on ne sache pas
> dire si les grenades explosent est un problème. »*

Le vrai constat n'est donc pas « trop peu d'explosions » mais **« un lancer sans verdict »** :

- **1 424 lancers sur 7 317 (20 %) n'ont AUCUN sort connu.** `buildGrenadeRestFx` les écarte en
  silence (`grenadeFx.ts:156` : `if (g.proj === undefined || g.proj === null) continue`). L'écran
  montre le lancer, puis **rien** — ni explosion, ni halo, ni « fin de vol inconnue ». Le
  spectateur ne peut pas distinguer « elle n'a pas explosé » de « on ne sait pas ».
- Sur certains matchs c'est **79 % des lancers** qui sont muets (Absolution `f8efc5ca` : 14 liés
  sur 67 ; Behemoth `7b0d89c4` : 19 sur 80).
- Même liée, l'explosion n'affirme pas une détonation : elle est posée à la **dernière position
  répliquée**, et l'en-tête d'`explosionFx.ts` l'assume — « l'écran continue de dire *dernière
  position connue*, jamais *impact* ». Pour une frag, la réplication cesse ~1,4 s après le lancer
  alors que la mèche court jusqu'à ~3 s.

**Ce qui manque est donc un ÉTAT, pas un effet** : le document ne publie nulle part « ce lancer
n'a pas de projectile apparié ». Tant que ce troisième état n'existe pas, l'absence d'explosion
reste ambiguë par construction.

### 12. Graphe des assistances vide + tableau des assistances conservé — **les deux confirmés**

**Le graphe :** sur Origin, l'onglet Joueurs affiche « Antagonistes » puis
« Assistances — *Assistances non disponibles pour ce match (non mesurées ou non publiables)* ».
Mesure : `publishable AND assist_known` = **0 sur 81 lignes**. Sur Isolation (43/43 publiables) le
même match donne 14 paires utiles. Cause = section commune.

**Le tableau :** il est toujours là, mais **sur la page Escouade**, pas sur la vue Match :
« ASSISTANCES DANS L'ESCOUADE » (`features/squad/SquadAssistPairsTable.tsx`, monté par
`SquadSynergiesPage`). C'est probablement lui que tu as vu.

### 13. Médias sans match associé : 106 en local, 17 en prod — **entièrement expliqué**

**104 médias sans association en local** (JGtm), 0 pour Madina97294. Répartition par année de
capture :

| année | sans match |
|---|---|
| 2018 | 49 |
| 2019 | 35 |
| 2025 | 3 |
| **2026** | **17** |

**84 des 104 (2018 + 2019) sont ANTÉRIEURS À HALO INFINITE.** Le registre de matchs commence le
**2021-11-19** (sortie du jeu) : aucun match Infinite ne peut exister pour ces captures.

> **Confirmé par l'utilisateur le 2026-09-08 : ce sont des médias HALO 5**, rangés par erreur dans
> l'arborescence Halo Infinite. Le point est clos côté diagnostic — reste un rangement à faire,
> et éventuellement leur rattachement au titre `halo_5`, qui a son propre parc de matchs.

**Les 17 de 2026 sont exactement les 17 que tu vois en prod** — c'est le même résidu, pas une
divergence. Rien d'inquiétant dans l'écart 106 / 17 : c'est ton archive locale.

Le résidu réel (20 médias, déc. 2025 → janv. 2026, clips de 30 s) : **aucun match du registre ne
couvre leur instant**. Exemple : `Halo Infinite 2026-01-25 17-17-54.mkv` → `capture_start_utc`
16:17:54 ; les matchs du jour vont de 15:09 à 15:52 UTC. Soit les matchs correspondants n'ont jamais
été synchronisés, soit ils relèvent d'un mode absent de `match_registry`.

À noter : les 143 associations existantes datent **toutes** d'une passe unique du **2026-06-24
22:35:06** — le backfill d'association ne tourne plus depuis.

### 14. Graphe « distance des armes » vide — **confirmé, ET le message d'état vide est FAUX**

Sur Origin, le bloc « Distance par arme » affiche :

> « Distances non mesurées sur ce match — elles demandent le décodage du film (positions du tueur et
> de la victime), **qui n'a pas encore été joué ici**. »

**C'est factuellement faux.** Le film A été décodé :

```bash
./tmp_diag_q.exe ... "SELECT match_id, count(*) FROM kill_positions_latest
WHERE match_id IN ('8bc6074f-...','81c02726-...') GROUP BY 1"
```
| match | positions de kill |
|---|---|
| Origin `8bc6074f` | **65** |
| Isolation `81c02726` | 39 |

Origin a **plus** de positions qu'Isolation, où le bloc s'affiche correctement (7 joueurs listés).
La vraie cause est `AND e.publishable` (`kill_measured.go:173`).

C'est l'anti-pattern n°9 du CLAUDE.md (« doc inversée ») : le message envoie l'utilisateur relancer
un décodage déjà fait. Clé i18n `match-view/i18n.ts:399`.

**Note connexe** : `match_weapon_hit_distance_latest` et `weapon_accuracy` sont **vides** (0 ligne)
sur tout le parc, alors que le persister existe (`persist/weapon_hit_distance_persister.go`). À
instruire séparément.

### 15. Bases neutres en bleu — **confirmé, le token neutre EST bleu**

`layers/useZoneStates.ts:51` : encre neutre = token sémantique `divergent-neutral`.
`styles/globals.css:211` : `--ac-divergent-neutral: #60A5FA` → **bleu (Tailwind blue-400)**.

Sur une carte où « allié » se dit déjà en bleu, un neutre bleu ne peut pas se distinguer. Ta demande
(blanc/gris + contour) revient à créer un token dédié — le fichier documente déjà qu'on ne veut pas
d'encre de layout (`--muted-foreground`) pour dire un fait de jeu.

### 16. Message victoire/défaite centré sur le lecteur et non sur la carte — **confirmé, mesuré**

Capture : [`08-ecran-fin-decentre.png`](captures/08-ecran-fin-decentre.png)

`ReplayVictoryOverlay` est monté en frère de `ReplayScoreBanner` + `ReplayCanvas` + la frise, dans
`<section className="relative min-w-0">`, et se pose en `absolute inset-0`
(`ReplayVictoryOverlay.tsx:167`). Il couvre donc **tout le lecteur**.

Mesure DOM à la fin du match Isolation :

| | haut | hauteur | centre vertical |
|---|---|---|---|
| Overlay = `<section>` | 132 | 796 | **530** |
| Canvas (la carte) | 181 | 520 | **441** |
| Bloc « DÉFAITE » | 460 | 140 | 530 |

**89 px sous le centre de la carte.**

### 17. Graphe « Portée des engagements » — **RÉFUTÉ : il est fait, et conforme à la maquette**

Capture : [`06-portee-engagements-implementee.png`](captures/06-portee-engagements-implementee.png)

`features/synthesis/SynthesisWeaponRangeSection.tsx` porte en en-tête : « Transposition de la
maquette validée par l'utilisateur le 2026-09-06
(`.ai/V7.5/MAQUETTE_PORTEE_ENGAGEMENTS_2026-09-06.html`, lot 5) ».

Rendu vérifié à l'écran, **il reprend la maquette élément pour élément** :
4 tuiles de tête (5,4 m / 5,3 m / 7,3 m / −1,7 m) · une carte « Portée par arme — mes frags et mes
morts · 27 armes » · les deux bâtons p10→p90 par ligne avec **losange sur la médiane** · la légende
« Mes frags (bâton du haut) / Mes morts (bâton du bas, l'arme est celle du tueur) » · les barres
empilées de dénivelé · la note de seuil qui NOMME les armes écartées · la note de couverture ·
« Voir en tableau ».

**Où le trouver** : `Solo → Synthèse`, **tout en bas de la page**. C'est probablement ce qui l'a
rendu invisible.

> **Précision utilisateur du 2026-09-08 :** ce qui manque n'est pas la version Solo — c'est sa
> **variante MULTI-JOUEUR sur la page Escouade**, dont la forme n'a jamais été arrêtée.
> **Chantier séparé**, à cadrer sous `plan-review` : la maquette du 2026-09-06 ne couvre que le
> cas solo (une ligne par arme, deux bâtons). À N joueurs il faut choisir entre N séries par arme,
> un petit multiple par joueur, ou une agrégation escouade avec écart individuel — décision de
> forme à prendre avant toute implémentation.

### 18bis. La maquette `2ec1b8eb` — **LUE le 2026-09-08, contenu relevé**

Accès obtenu via Chrome piloté (`claude-in-chrome`), après contournement du blocage de défilement :
les événements de scroll ne traversent pas l'iframe cross-origin de l'artefact ; la parade est
d'allonger l'iframe depuis la page parente (`iframe.style.height = '7000px'` + ancêtres en
`overflow: visible`), ce qui rend la PAGE défilable et pilotable en JS. À réutiliser.

**Titre : « Les formes retenues ».** Session témoin : 9 matchs, 38 joueurs observés, parité
24,2 % / 11,9 %, 3 familles de mode.

**Les deux lentilles** (elles ne répondent pas à la même question) :
- **« Ce que j'ai fait »** — des cadences, normalisées **par dix minutes de jeu**, match par match ;
- **« Où je me situe »** — des parts, **avec l'équipe d'en face pour référence : jamais affichée,
  toujours comptée**.

**Les quatre formes canoniques**, définies en tête de maquette :

| Forme | Définition (verbatim) |
|---|---|
| **ÉCART À LA PARITÉ** | « Le trait ambre est la parité, la barre part de là. Droite : plus que la référence. Gauche : moins. La longueur EST l'écart, en points. » |
| **JAUGE DOUBLE** | « Deux pistes sur un axe commun : ma part de mon équipe, puis du lobby. Chacune son trait de parité et son étendue. » |
| **PISTE DU LOBBY** | « La barre entière vaut le lobby. Coloré : à nous, par joueur ; hachuré : à eux. Trait ambre à la parité. » |
| **GRILLE ET BANDE** | « Lignes alignées, une échelle et un axe PAR colonne. La bande a une case par match, teintée par l'écart. » |

**Règle des grenades — confirmée et explicite** : « **Les grenades sont sorties du bloc** : ce ne
sont pas des équipements. » Familles restantes : camouflage, surbouclier, mur de protection,
grappin, objets lâchés au sol. **Les colonnes sont pilotées par la donnée** — une famille que
personne n'a utilisée n'a pas de colonne. Deux exceptions écrites : le **répulseur** (le film
montre l'objet au sol, aucun canal ne date son activation) et le **propulseur** (mesuré, mais dure
une demi-seconde : il se voit sur la carte du rejeu, pas dans un compteur).

**Les blocs, par domaine et par contexte :**

| Domaine | Contexte SOLO | Contexte ESCOUADE |
|---|---|---|
| **Usages d'équipements** | Ma part, dans mon équipe et dans le lobby (B) · Cadence de gestes, match par match (A) · Étendue et moyenne de la session (B) | Régularité match par match (B) · Ce que mon camp prend du lobby (C) · Cadence de chacun sur la session (A) · Qui porte quel geste dans l'escouade (B) |
| **Contrôle des armes spéciales** (prises de socle) | Écart à la parité, par famille d'arme (A) · Ma part des prises de socle, et sa dispersion (B) · Taux de rafle par arme (B) | Écart à la parité, par famille d'arme (A) · Les deux frises : quand, et qui (B, réunies) |

Vocabulaire posé : « un **socle** » = l'emplacement fixe où une arme de puissance réapparaît ;
« une **prise de socle** » = un ramassage sur cet emplacement, daté à la milliseconde, porteur de
son ramasseur. **Le dénominateur n'est pas le nombre de socles** mais le nombre de prises
NOMMÉES — la réserve (« 102 occupations sans ramasseur nommé ») reste à l'écran.

### 18. Graphes équipement / objectifs manquants — **la maquette est lue ; l'écart reste à mesurer**

Je **n'ai pas pu lire l'artefact `2ec1b8eb-5b4d-4484-b632-c8ee91569825`** : l'outil me répond
« this artifact is served to you as a public (non-member) reader, and reading public artifacts that
way is not enabled yet ». Idem pour `19e7eca1-…` et `4c520da6-…`.

Tentatives (2026-09-08), toutes infructueuses :
- outil `Artifact read` → « public (non-member) reader … not enabled yet » ;
- `https://claude.ai/public/artifacts/<uuid>` → « Page introuvable » ;
- `https://claude.ai/code/artifact/<uuid>` sur un Chrome non authentifié → « Page not found » ;
- `Artifact list` (35 entrées, possédées + partagées) : **les trois UUID n'y figurent pas** — ils
  n'appartiennent donc pas au compte courant et n'y sont pas partagés nommément ;
- bascule vers un Chrome authentifié (`switch_browser`) : aucune extension n'a répondu en 2 min ;
- **voie navigateur abandonnée** : Cloudflare y boucle sur une vérification anti-bot. Contourner
  une détection de bot est hors de question, la question est close de ce côté.

**DÉBLOQUÉ le 2026-09-08 pour `2ec1b8eb`** (cf. § 18bis et 22) : l'utilisateur a ouvert une
instance Chrome et l'extension `claude-in-chrome` a donné l'accès. Deux obstacles à connaître pour
la prochaine fois :
- le contenu vit dans une **iframe cross-origin** (`*.frame.claudeusercontent.com`) — ni
  `get_page_text` ni `fetch` ne l'atteignent, et naviguer directement sur l'URL de frame redirige
  vers l'enveloppe ;
- **les événements de scroll (molette, Page_Down, glisser d'ascenseur) ne la traversent pas.**
  Parade qui marche : allonger l'iframe depuis la page parente
  (`iframe.style.height = '7000px'`, ancêtres en `height:auto; overflow:visible`), ce qui rend la
  PAGE défilable — ensuite `window.scrollTo(0, N)` + capture, par paliers.
- La capture (`Page.captureScreenshot`) expire une fois sur deux sur cette page ; il suffit de la
  relancer seule.

**`4c520da6` LU le 2026-09-08 par la même méthode** (cf. § 20bis). **Reste `19e7eca1`**, sans objet
désormais : la section « Portée des engagements » est implémentée et conforme (§ 17).

**Les trois artefacts sont donc dépouillés** — les points 18, 20 et 22 ne sont plus bloqués.

Ce qui existe, mesuré :
- `features/match-replay/MatchEquipmentUsageSection.tsx` + `model/equipmentUsageChart.ts` /
  `equipmentUsageColumns.ts` / `equipmentZones.ts` — page de rejeu ;
- `features/squad/SquadObjectiveStatsPanel.tsx` + `SquadObjectivesPanel.tsx` — Escouade ;
- bloc « Objectifs » sur la Synthèse (captures de drapeau, retours, vols, temps porteur, zones
  capturées/sécurisées, temps en zone, crâne).

Sur ta consigne « virer les grenades des représentations liées aux équipements » : la répartition
des frags de la Synthèse compte toujours **« Grenade — 5,7 % »** comme une catégorie à part entière
au même niveau que les armes, ce qui est cohérent avec ta règle (grenade = arme), mais je ne peux
pas vérifier qu'elle a été retirée des blocs ÉQUIPEMENT sans la maquette.

> **Ce qu'il me faut** : soit tu me repartages ces trois artefacts (ou tu me connectes ta session
> Chrome, je les ouvre depuis ton navigateur), soit tu me redonnes les décisions en clair.

### 19. Répartition des frags / détail par arme : « armes inconnues » — **confirmé, avec deux causes**

**Cause A — la catégorie « Non attribué ».** Sur la Synthèse (1 147 matchs) :
`Non attribué — 7 %` de 11 232 frags ≈ **786 frags sans arme**. Sur le seul match Origin :
`Non attribué — 14,3 %` (1 frag sur 7). C'est le complément direct de `publishable = false` et des
matchs antérieurs au décodeur.

**Cause B — la lisibilité du sunburst.** Capture
[`05-synthese-pleine-page.png`](captures/05-synthese-pleine-page.png) : les étiquettes de l'anneau
extérieur se **chevauchent au point d'être illisibles** (« Bobine à souffle », « Bo…à plasma »,
« Bo…% choc », « 30…Fusion UNSC »… empilées les unes sur les autres). 20 étiquettes se disputent le
même flanc gauche. C'est un défaut de rendu à part entière.

**Cause C — les identifiants bruts.** Sur la fiche joueur du rejeu Isolation, MADINA97294 affiche
**`0xD7915565`** en lieu et place d'un nom d'arme : un identifiant non résolu par `weaponLabels`
s'affiche tel quel.

Sur ton attente « ça devrait être exactement ce qu'on a dans le killfeed » : les deux lectures
n'ont pas la même porte. Le killfeed exige `publishable` (donc se tait), la répartition des frags
compte l'AGRÉGAT (donc publie, avec un « non attribué »). Elles ne peuvent pas coïncider tant que
ce drapeau est binaire.

### 20bis. La maquette `4c520da6` — **LUE le 2026-09-08 : « L'échange sur la page Escouade »**

Cinq graphes candidats, **chacun bâti sur un wrapper du catalogue existant** et étiqueté par son
composant, sa page cible et son statut :

| # | Bloc | Composant | Page | Statut |
|---|---|---|---|---|
| 1 | Le compte | `KPIStrip` | Escouade + Tactique | **V1** |
| 2 | Combien, et à quelle vitesse | `HistogramChart` | Dynamique | **V1** |
| 3 | **Qui couvre qui** | `Heatmap2DChart` | Synergies | **V1** |
| 4 | **Pourquoi la vengeance ne vient pas** | `ScatterChart` | Synergies | **V1** |
| 5 | Donné et reçu, par coéquipier | `BarGroupedChart` | Synergies | candidat |

**Règle transverse posée dès le bloc 1 :** « Un taux seul ne dit rien, et **un taux seul est même
trompeur** : il se calcule sur tes propres morts. Il est donc **toujours** accompagné du **compte
brut** et d'une **quantité par match**. » — « Brut ET part, toujours ensemble, sans bascule. »

**Bloc 3 — « Qui couvre qui », la heatmap de référence des échanges :**
- matrice **VENGEUR (lignes) × VENGÉ (colonnes)**, noms en chasse fixe, **pastille de couleur du
  joueur à gauche de chaque ligne** ;
- **cases en rectangles ARRONDIS, nettement espacées les unes des autres** — c'est aéré, et c'est
  voulu (retour utilisateur du 2026-09-08) ;
- **valeur en clair au centre de chaque case** ; diagonale = `—` sur une case neutre non colorée ;
- **échelle séquentielle à UNE seule teinte** (bleu, clair = beaucoup) — pas de divergence ici,
  contrairement à la grille de `2ec1b8eb` ;
- **totaux reçus sous chaque colonne** (`reçu 23`, `reçu 37`…), et **deux pastilles de verdict**
  posées sous la colonne concernée : « le plus couvert » (contour vert), « le moins couvert »
  (contour ambre) ;
- **légende d'échelle continue en bas à gauche** : `0 ▬▬▬ 14 échanges` ;
- notes : « Ligne : celui qui venge. Colonne : celui qui est vengé. Même orientation que
  *Assistant / Bénéficiaire* du tableau des assistances, juste à côté. » et « Bandeau de couverture
  obligatoire au-dessus : les films expirent, le manque est définitif et pas un retard. »

**Bloc 4 — « Pourquoi la vengeance ne vient pas », le nuage :**
- X = « morts hors portée de radar → », Y = « ↑ morts vengées », en pourcentages ;
- **quatre libellés de quadrant dans les coins** : *proche et couvert* · *loin, mais on vient* ·
  *proche, et pourtant seul* · *loin et sans secours* ;
- **la zone de danger (bas-droite) est teintée** ; deux lignes de repère en pointillés se croisent
  aux médianes ;
- **deux tailles de point** : petit = une session, gros = un joueur (étiqueté à côté du point) —
  « la taille dit le nombre de morts » ;
- **sous 30 morts, le point est cerclé de pointillés** (échantillon faible) : `Karst · n=24` ;
- **légende de taille en bas** : `● 24 morts` · `● 82 morts` · `· une session` ·
  `⬚ moins de 30 morts` ;
- réserves écrites : « Portée du radar : 18 m en Arène, 24 m en BTB. » · « Le biais du dénominateur
  et comment il est traité […] **Aucun classement de joueurs n'est tiré de ce nuage.** » ·
  « **Pourquoi la session et pas le match** : […] La session est la plus petite maille où un taux
  veut dire quelque chose. Une session sous 5 morts n'entre pas dans le nuage. » ;
- dépendance déclarée : « Demande les positions, donc les matchs cuits : arrive avec la phase 6. »

### 20. Heatmap et scatter des échanges — **ils existent ; l'écart à la maquette est maintenant mesurable**

Capture : [`07-escouade-dynamique-echange.png`](captures/07-escouade-dynamique-echange.png)

Sont rendus sur `Escouade → Synergies` :
- **« Qui échange pour qui » / « VENGEUR × VENGÉ »** (`SquadEchangeMatrixCard.tsx`) — la matrice ;
- **« Isolement et couverture »** (`SquadIsolementNuageCard.tsx`) — le nuage, avec ses planchers
  (5 morts pour qu'un point existe, 30 pour qu'un taux se lise sans réserve) ;
- « CONSTAT DU MOMENT » (`SquadEchangeConstatCard.tsx`) ;
- sur `Escouade → Dynamique` : « Délai d'échange » (`SquadEchangeDelaiCard.tsx`).

**Je ne peux pas les comparer à l'artefact `4c520da6-775b-4fb8-9d6e-dd6aa8d629b9`** (même refus de
lecture qu'au point 18).

### 21. Page Tactique : ouvrir une carte n'affiche rien — **confirmé, DEUX causes, aucune n'est un backfill manquant**

Capture : [`04-tactique-plan-vide.png`](captures/04-tactique-plan-vide.png)

Réponse serveur pour Illusion (`POST /players/JGtm/tactical/9e821f5e-…/raster`, question `morts`) :

```json
{ "matchs_filtres": 56, "matchs_retenus": 38,
  "evenements_journal": 433, "evenements_localises": 115,
  "pas_m": 0.5,
  "bornes": { "min_x":0, "min_y":0, "max_x":0, "max_y":0, "valide": false },
  "cellules": [], "echelle": { "n_cellules": 0 } }
```

**Cause A — le plancher est hors d'atteinte à cette densité.** `PlancherMatchsParCellule = 3`
(`analysis/tactical/merge.go:16`) : une cellule n'existe que si elle est vue dans **3 matchs
DISTINCTS**. Avec une grille de **0,5 m** (`grid.go:41`) et **115 événements localisés**, aucune
cellule n'y arrive. Le front dit alors « Pas assez de matchs mesurés » — le message désigne les
matchs, alors que le blocage est la densité par cellule.

**Cause B — la couverture des positions.** 115 événements localisés sur 433 (**26,6 %**), et au
niveau du parc `kill_positions_latest` ne couvre que **413 matchs sur 1 967 (21 %)**.

Le backfill que tu n'as pas lancé aiderait la cause B, **pas la cause A**.

### 22. Style unifié des heatmaps — **LA RÉFÉRENCE EST RELEVÉE**

La heatmap de référence est **« Régularité match par match »**, sous le bandeau
**« CONTEXTE ESCOUADE — MON CAMP CONTRE LE LEUR »** (étiquetée `LA PART DU LOBBY · ESCOUADE B`)
dans l'artefact `2ec1b8eb`. Sa forme, relevée le 2026-09-08 :

1. **Lignes** = les familles mesurées ; **colonnes** = les matchs de la session, **dans l'ordre**.
2. **Étiquettes de colonne sur DEUX lignes** : l'heure au-dessus (`19:22`), la carte en dessous
   (`Perilous`), en petites capitales. Sous-titre d'axe : « les neuf matchs de la session, dans
   l'ordre ».
3. **Chaque case porte sa VALEUR EN CLAIR** (`19%`, `100%`, `0%`, `86%`…), centrée, sur le
   rectangle plein. Pas de survol obligatoire pour lire un chiffre.
4. **DEUX couleurs divergentes, pas une rampe continue** : bleu « Plus que l'équipe adverse »,
   orange « Moins ». L'intensité module la teinte, **et sature à trente points** (limite assumée).
5. **L'absence a sa propre forme** : case **hachurée** portant un tiret, légendée « Aucune mesure
   sur cet axe ». Jamais une case vide, jamais un zéro à la place d'un inconnu.
6. **PADDING ENTRE LES CASES — c'est aéré, et c'est le point qui fait la qualité du rendu**
   (insistance utilisateur du 2026-09-08). Les cases sont des rectangles **arrondis, nettement
   détachés les uns des autres** par un espacement franc — jamais une grille jointive, jamais une
   bordure. La même respiration se retrouve sur la heatmap « Qui couvre qui » de `4c520da6`
   (cf. § 20bis) : c'est donc une **propriété transverse de la forme**, pas un détail d'une
   maquette. Une reprise qui colle les cases perd l'essentiel de ce qu'on cherche à reproduire.
7. **Légende horizontale sous la grille**, trois entrées à pastille, sur une ligne.
8. **Note de lecture en pied**, qui dit comment lire la forme et non ce que valent les données :
   « Un écart systématique se voit à la couleur d'une ligne entière, un accident se voit à une case
   isolée. »
9. **Elle nomme sa propre réutilisation** : « Cette disposition est réutilisable telle quelle par
   *Performance par joueur × carte* (page Escouade) : c'est le même objet — des lignes, des
   colonnes, une case colorée — et **sa sécurité daltonienne tient à ses jetons `perf-tier-*` et à
   ses libellés de palier**, qui survivent au changement de contenant. » Puis, honnêtement :
   « **Ce qu'elle abandonne** : le volume, et l'intensité sature à trente points. »

Le point 9 est décisif : **la maquette se désigne elle-même comme la grille canonique du dépôt**,
et nomme le mécanisme qui rend la reprise sûre (`perf-tier-*` + libellés de palier). C'est
exactement la doctrine que tu demandes d'appliquer à toutes les heatmaps.

**L'écart avec l'existant** : les heatmaps du dépôt passent par des sources distinctes —
`components/charts/heatmapColors.ts` (échelles), `features/squad/SquadMapHeatmapChart.tsx`,
`features/match-view/MatchPositionsHeatmap.tsx`, `features/match-replay/layers/heatmapLayer` et
`features/tactical/TacticalMapTile.tsx`. **Il n'existe aucun composant de grille canonique
partagé.** Le chantier consiste donc à en créer un d'après cette maquette, puis à migrer les
lecteurs — avec le garde-rail que la règle n°6 du dépôt exige (une factorisation sans garde-rail
re-diverge).

**La forme voisine, à ne pas confondre** : « Ce que mon camp prend du lobby » (`ESCOUADE C`) est
une barre empilée à 100 %, segmentée **par joueur nommé** (JGtm, Madina97294, Chocoboflor,
« Coéquipier hors escouade »), avec l'**équipe adverse en hachuré — comptée, jamais nommée** — et
un trait ambre de parité. Elle répond à *qui*, la grille répond à *quand*.

---

## Ce que le chantier « identités » sur `feat/v2-decodeur-e2` / `wt/orchestration-0907` couvre

Tu demandais si le dernier chantier règle les points 1 à 6. Sur pièces (registre
`.ai/AUDIT_LECTEURS_VIES_ANONYMES_2026-09-06.md`, mis à jour le 2026-09-07) :

| Ton point | Couvert par le chantier identités ? |
|---|---|
| 1 médailles frise | **Non** — c'est l'encodage visuel, pas l'identité |
| 2 icônes killfeed | **Non** — c'est `publishable`, hors périmètre |
| 3 fiches rognées | **Non** — CSS grid, hors périmètre |
| 4 équipes interverties | **NON, et c'est le point le plus sérieux.** Le chantier a traité les vies ANONYMES (une vie sans nom) ; ici les vies sont NOMMÉES, et **mal**. Aucun de ses 14 constats ne couvre une interversion entre deux joueurs nommés de camps opposés, et `coverage.bridge` la déclare saine (`indexDisagreements: 0`) |
| 5 fond de carte | **Non** |
| 6 véhicules absents | **Non** — P1-5 « occupant de véhicule » a corrigé la LECTURE (`OwnerReport.xuidAt`), mais 15 artefacts au schéma 38 n'ont jamais subi le balayage. C'est un problème de parc, pas de code |

---

## Suite proposée

**P0 — identités interverties (point 4)**

Le seul constat qui fait AFFIRMER UN FAUX à l'écran, et le seul qui reste OUVERT après mesure :
**7 artefacts sur 30 testables** portent au moins un joueur mal placé, dont **4 avec la signature
d'interversion propre** — et **3 de ces 4 sont déclarés entièrement publiables** par le décodeur.
La porte `LineByLinePublishable()` ne prédit donc pas le défaut, et l'appliquer à la géométrie ne
le corrigerait pas. **La piste reste à instruire** : le camp spawne groupé, et le pont n'exploite
pas cette information.

Outil de mesure conservé : [`swap.sh`](swap.sh) (à lancer avec l'export
`match_id / xuid / team_id` décrit dans [`requetes.md`](requetes.md)).

**Recuisson du parc (point 6)**

15 artefacts au schéma 38, sans balayage véhicules. `cmd/replay-build` sur ces 15 match_id.

**Correctifs courts, isolés, prouvés (candidats à un lot unique)**

1. `ReplayTeams.tsx:184` → `minmax(0, 1fr)` (point 3) — 1 ligne, prouvé par mutation.
2. `navL1Sections.tsx` → ajouter l'onglet `tactique` (point 7) — 1 entrée.
3. `ReplayPlayhead.tsx` → largeur du trait (point 9) — 1 classe.
4. `ReplayVictoryOverlay` → ancrer sur le canvas et non sur la section (point 16).
5. `match-view/i18n.ts:399` → message d'état vide honnête (point 14) : dire `publishable`, pas
   « décodage non joué ».
6. Token neutre des zones (point 15) — nouveau token, pas `divergent-neutral`.
7. **Médailles de la frise en IMAGES** (point 1) — `MedalBadges` existe et le kill feed l'emploie
   déjà ; il faut élargir la piste pour l'accueillir.

**À instruire (chantiers)**

8. `coversPlayedArea` : emprise robuste (p1/p99) + log (point 5).
9. Fermeture des spans de portage de drapeau, ~2 s trop tardive (point 10b).
10. Troisième état « sort du lancer inconnu » pour les 20 % de grenades sans projectile apparié
    (point 11) — c'est un manque de DONNÉE publiée, pas un effet à ajouter.
11. `attachVehicleShots` : 2 artefacts sur 64 publient des tirs (point 8).
12. Plancher/pas de la grille tactique face à la densité réelle (point 21).
13. `match_weapon_hit_distance` / `weapon_accuracy` vides sur tout le parc (point 14, note).
14. Lisibilité des étiquettes du sunburst de répartition des frags (point 19B).
15. **Variante multi-joueur de « Portée des engagements » sur Escouade** (point 17) — forme à
    arrêter AVANT toute implémentation.

**Escalades — décision utilisateur, sans proposition d'action**

- La sémantique binaire de `publishable` (section commune) — d'autant plus après le point 4 : le
  drapeau protège le texte et pas la géométrie.
- La direction du cône de visée en véhicule (point 8) — mesure contre intuition tactique.
- Les trois artefacts de maquette illisibles (points 18, 20, 22) — accès à obtenir.

---

# Seconde passe — 2026-09-08 (variantes par contexte, et clôture de 3 des 7 points ouverts)

## La taxonomie des maquettes — SCOPE × CONTEXTE × VARIANTE

Chaque bloc des deux maquettes porte une étiquette en haut à droite, et elle n'est pas décorative :

- **SCOPE** — `UNE SOIRÉE` (des **cadences**, normalisées par dix minutes de jeu) ou
  `LA PART DU LOBBY` (des **parts**, avec l'équipe d'en face pour dénominateur) ;
- **CONTEXTE** — `SOLO` ou `ESCOUADE` ;
- **VARIANTE** — `A`, `B`, `C` (formes concurrentes d'une même question).

Relevé complet :

| Bloc | Étiquette |
|---|---|
| Ma part, dans mon équipe et dans le lobby | `LA PART DU LOBBY · SOLO B` |
| Cadence de gestes, match par match | `UNE SOIRÉE · SOLO A` |
| Étendue et moyenne de la session | `UNE SOIRÉE · SOLO B` |
| Régularité match par match | `LA PART DU LOBBY · ESCOUADE B` |
| Ce que mon camp prend du lobby | `LA PART DU LOBBY · ESCOUADE C` |
| Cadence de chacun sur la session | `UNE SOIRÉE · ESCOUADE A` |
| Qui porte quel geste dans l'escouade | `UNE SOIRÉE · ESCOUADE B` |
| Écart à la parité, par famille d'arme | `LA PART DU LOBBY · SOLO A` puis `· ESCOUADE A` |
| Ma part des prises de socle, et sa dispersion | `LA PART DU LOBBY · SOLO B` |
| Taux de rafle par arme | `UNE SOIRÉE · SOLO B` |
| Les deux frises : quand, et qui | `LA PART DU LOBBY · ESCOUADE B, RÉUNIES` |
| Emprise de l'escouade, match par match | `UNE SOIRÉE · ESCOUADE B` |
| Taux de rafle de chaque coéquipier | `UNE SOIRÉE · ESCOUADE C` |

> **Contrainte utilisateur du 2026-09-08 : la page Sessions ne prend que les graphes NORMALISÉS,
> et seulement ceux qui PEUVENT l'être.** Vérifié — elle la respecte déjà :
> `session-detail/usageLogic.ts:5` porte en en-tête « **TOUT AXE EST NORMALISÉ (doctrine §1) :
> parts en %, cadences par dix minutes.** »

## Les variantes par contexte SONT implémentées — mais sur une seule page

`SessionUsageSection.tsx` porte **les trois blocs** de la maquette, et **résout le contexte
Solo/Escouade CÔTÉ SERVEUR** : « le bloc porte `squad_players` et des lignes `squad` par grandeur —
vides en solo. À l'écran, l'escouade ajoute une ligne par coéquipier dans les grilles et découpe la
piste du lobby par joueur ; le solo garde moi + les agrégats. »

`SessionUsageForms.tsx` implémente trois des quatre formes canoniques — « écart à la parité / jauge
double avec étendue », « piste du lobby », « bande de régularité » — les grilles alignées passant
par la primitive partagée `components/charts/ValueGrid`. Les jetons y sont conformes
(`team-ally` pour nous, `squad-player-*` par joueur, `warning` pour la parité, **hachure neutre
anonyme pour eux**, gamme `divergent-*` pour la bande).

## Tableau d'écart — point 18

| Page | Usages d'équipement | Contrôle des armes spéciales | Objectifs |
|---|---|---|---|
| **Sessions** | OUI — `SessionUsageSection` (Solo **et** Escouade) | OUI | OUI |
| **Match view** | OUI — `MatchEquipmentUsageSection` (onglet Chronologie) | OUI — `MatchPadControlSection` | OUI — `MatchObjectivesSection` |
| **Escouade** | **ABSENT** | **ABSENT** | OUI — `SquadObjectivesPanel` + `SquadObjectiveStatsPanel` |
| **Solo / Synthèse** | **ABSENT** | **ABSENT** | Partiel — bloc de compteurs, pas les formes de la maquette |

Preuve de l'absence : le bloc `Usage` n'existe que sur `domain/session_page.go:145`
(`Usage *SessionUsageBlock`). Aucun équivalent sur la page Escouade ni sur la Synthèse, et zéro
occurrence de `pad_control` / `equipment_usage` dans `features/squad/` ou `features/synthesis/`.

**Conclusion du point 18 : ce ne sont pas les VARIANTES qui manquent, ce sont deux PAGES.** Les
variantes Solo et Escouade coexistent correctement là où le bloc existe.

## Point 20 — le nuage d'isolement : trois écarts à la maquette

> **Constat utilisateur du 2026-09-08 : « je ne voyais que 4 points, soit le nombre de joueurs,
> alors que normalement on a un nuage de points par joueur, avec un gros point pour la moyenne. »**
> La maquette lui donne raison mot pour mot : « **Un petit point par session, un gros point par
> joueur** — la taille dit le nombre de morts », et sa légende sépare `· une session` des gros
> points étiquetés (`Karst · n=24`).

1. **LE GROS POINT PAR JOUEUR N'EXISTE PAS.** `SquadIsolementNuageCard.tsx:4` déclare « Un point
   par (joueur, session) », et `seriesParJoueur` ne construit qu'une série de points de session par
   joueur. **Aucun agrégat par joueur n'est tracé** — ni moyenne, ni médiane, ni étiquette de nom.
   C'est l'élément le plus lisible de la maquette, et il manque.
   *(La maquette ne tranche pas moyenne vs médiane : elle dit « un gros point par joueur » et
   « la taille dit le nombre de morts ». À trancher — la médiane est cohérente avec le reste de la
   page, qui médiane déjà ses lignes de repère.)*
2. **Les quatre libellés de quadrant ne sont affichés NULLE PART.** `quadrantDuPoint`
   (`squadIsolement.logic.ts:66`) est exporté et testé, les quatre clés existent
   (`squadIsolementStrings.ts:20-23`, manifeste i18n généré) — et **rien ne les consomme** :

   ```bash
   grep -rn "quadrantDuPoint\|isolement.quadrant" apps/web/src --include=*.ts --include=*.tsx \
     | grep -v "\.test\.\|generated/squad.ts"
   # -> seulement la definition et la table de chaines. Aucun appelant.
   ```

   C'est l'anti-pattern **n°1 du CLAUDE.md, « dead code museum »** — du code mort maintenu en vie
   par des tests verts, et la cause directe des coins non étiquetés.
3. **L'échantillon faible se dit en OPACITÉ, la maquette le dit en CERCLE POINTILLÉ**
   (`pointAttenue` vs la légende `moins de 30 morts` à cercle pointillé). Divergence mineure mais
   réelle.

Ce qui EST conforme : `markLine` (les deux médianes) et `markArea` (la zone de danger) sont bien
implémentés, et la taille du point suit `morts_examinees`.

**Note de lecture** : la page était filtrée sur **1 session** au moment du constat. À ce périmètre,
un point par (joueur, session) donne exactement 4 points — le nuage dégénère par construction. La
maquette, elle, tourne sur 62 matchs d'escouade.

## Point 14 bis — CLOS : les tables vides sont VOULUES

`match_weapon_hit_distance` et `weapon_accuracy` sont vides **par décision datée**, pas par panne.
`games/halo_infinite/adapter_data.go:200` :

```go
games.CapWeaponAccuracy: games.CapNotExposed,
```

Commentaire attaché : « Précision par arme : REMISÉE le 2026-09-01 (`not_exposed`). Le numérateur
film s'est révélé NON FIABLE au recalage : […] les armes automatiques ressortent à 0,9-3,3 % vs
~40 % côté API (`RECALAGE_WEAPON_ACCURACY_FILM_2026-09-01`, commit `945c9fdb7`). […] **Cette clé
data-level gate le numérateur film (`collectHits`) : `not_exposed` implique que la passe film ne
s'exécute pas pour Infinite.** »

Le persister est bien câblé (`sync/killcollector/hits.go:198`) : c'est la passe qui ne tourne pas.
**Et ça ne change rien au point 14** : « Distance par arme » ne lit pas ces tables — elle lit
`kill_positions` + `publishable` (`kill_measured.go:173`). Le diagnostic du point 14 tient.

## Point 19 C — CLOS : `0xD7915565` est un inconnu ASSUMÉ

`layers/useReplayWeaponPads.ts:84` documente la cascade de nommage et conclut : « sinon
l'identifiant lui-même — et **c'est VOULU** pour une arme hors catalogue du titre (l'hexadécimal
est alors la seule chose vraie qu'on puisse écrire, cf. la famille `0xD7915565` du registre des
reports) ». Le test le nomme `INCONNUE`.

Les `weaponLabels` du document comptent 18 entrées, indexées à la fois en 8 et en 16 hexa
(`0x0A1992BC` et `0x0A1992BC42C9679F`) ; `0xD7915565` n'y est pas. **Ce n'est donc pas un défaut de
résolution mais un trou de catalogue, connu et inscrit au registre des reports.**

Reste valide en revanche : la catégorie **« Non attribué — 7 % »** du sunburst (environ 786 frags),
et l'illisibilité des étiquettes de l'anneau extérieur (points 19 A et 19 B).

## Point 8 — CLOS pour l'essentiel : `shotsNoRide` ne compte pas ce que je croyais

`analysis/replay/vehicle_shots.go` définit le verdict : « `vehicleShotNoRide` : aucun épisode de ce
tireur ne couvre l'instant. **C'est le cas nominal d'un tir à pied que le pont n'a pas su placer —
il n'a rien à voir avec un véhicule.** »

Les 188 (Isolation) à 2 957 (Flood Gulch) « écartés » sont donc des tirs À PIED non placés, pas des
tirs de véhicule perdus. Sur Isolation, `shotsAmbiguous: 0` et `shotsUnplaced: 0` : **aucun
orphelin ne tombe dans une fenêtre de chevauchée**, ce qui est cohérent avec les 3 chevauchées du
match (un Mongoose — sans arme — et deux Ghost courts).

**Ce qui reste vrai** : seuls 2 artefacts sur 64 publient des tirs de véhicule, et le vrai facteur
limitant est le nombre de chevauchées ARMÉES dans le parc, pas un défaut de rattachement. À
recroiser après la recuisson des 15 artefacts du schéma 38 (point 6).

---

# Troisième passe — 2026-09-08 : cause racine du point 4, et coordonnées pour le point 19 C

## Point 4 — LA CAUSE EST TROUVÉE : deux vies qui se terminent au même instant

**Le mécanisme, établi sur 4 matchs sur 4.** Les deux joueurs intervertis ont leur première vie
réelle qui se termine **à la même image** (ou à une image près) :

| artefact | carte | slots intervertis | fin de leur 1re vie | `publishable` |
|---|---|---|---|---|
| `8bc6074f` | Origin | 516 / 518 | **774 / 774** | 0 / 81 |
| `d8b13ec2` | Goliath | 516 / 518 | **462 / 462** | 97 / 97 |
| `a4083bd2` | The Pit | 512 / 518 | **775 / 776** | 95 / 95 |
| `bf2a9f05` | Bazaar | 516 / 512 | **176 / 176** | 96 / 96 |

**Deux morts simultanées, deux corps, et le pont doit décider lequel nomme quelle vie.** Quand les
deux morts appartiennent à des camps opposés, se tromper d'appariement produit exactement
l'interversion propre observée : un joueur de chaque camp dans la base adverse.

**Et c'est ce qui explique la résorption.** Les vies suivantes se terminent à des instants
distincts : l'appariement redevient univoque, et l'identité se corrige toute seule. Sur Origin la
bascule tombe à l'image 874, sur Goliath à l'image 543 — dans les deux cas, dès la vie suivante.

**Pourquoi `publishable` ne le voit pas — et c'est structurel.** `LineByLinePublishable()` mesure la
**marge de bijection GLOBALE** (indice de film → joueur, une fois pour le match). Le défaut, lui,
est dans l'appariement **PAR VIE** de la fin de vie à une mort du fil. **Deux mécanismes distincts,
deux marges distinctes** : la première peut être franche pendant que la seconde est nulle. D'où
3 interversions sur 4 dans des matchs déclarés entièrement publiables.

**Les compteurs qui devraient l'attraper, et pourquoi ils n'y arrivent pas.** `coverage.bridge` de
`8bc6074f` publie `closedContested: 9` et `deathOffsetRunnerUp: 10` — donc la tension EST comptée.
Mais `ClosedContested` est documenté comme « déductions ABANDONNÉES faute d'unicité — deux corps
possibles pour un » : une fermeture contestée est **abandonnée**, pas arbitrée. L'interversion
n'emprunte donc pas ce chemin-là : elle passe par un appariement qui, lui, **tranche** malgré
l'égalité. Et `indexDisagreements: 0` / `slotCollisions: 0` déclarent le pont sain.

**Ce qu'il reste à faire — et c'est un chantier, pas un audit.** Localiser l'appariement fin de
vie ↔ mort qui tranche à égalité (`analysis/replay/closures.go`, `closures_respawn.go`,
`identity.go`, `owners.go`), et lui donner soit un refus (laisser la vie non nommée plutôt que mal
nommée), soit un départage. **Le départage existe et il est gratuit : le camp spawne groupé.** Deux
morts simultanées de camps opposés se départagent par la position de la vie suivante du même slot,
ou par la grappe de spawn. Aucune de ces deux informations n'est aujourd'hui consultée.

**Instrument de non-régression prêt à l'emploi** : [`swap.sh`](swap.sh) — il rejoue le test sur
tout le parc et doit rendre `malplaces = 0` partout après correctif.

> **Réserve honnête** : je n'ai pas ouvert le décodeur pour nommer la ligne exacte. La doctrine
> §0.6 le gèle, et l'audit ne corrige pas. Ce qui est établi ici, c'est la SIGNATURE (4/4), la
> résorption, et la non-corrélation avec `publishable` — assez pour cadrer le chantier.

## Point 19 C — CE QU'IL TE FAUT POUR INVESTIGUER

> Demande utilisateur du 2026-09-08 : « il me faut le joueur et la victime, date et heure et nom de
> la map ».

**Match** — `81c02726-3f03-4a9e-9c05-3e286460752c`
**Carte** — **Isolation** (« Bases sur Isolement »), mode **Strongholds:Arena**, partie rapide
**Début** — **2026-09-07 19:34:52 UTC**, soit **21:34 heure locale** — durée 5:21

**Identifiant d'arme non résolu** — `0xD791556542C9679F` (forme longue)
soit `0xD7915565` en forme courte, celle qui s'affiche à l'écran.

**Qui la porte** — elle apparaît dans **12 loadouts**, sur 4 vies appartenant à **deux joueurs** :

| slot | xuid | gamertag | équipe | F/M du match |
|---|---|---|---|---|
| 519, 528 | `2533274846254457` | **BrunoArtl** | 1 | 6 / 2 |
| 535, 549 | `2533274858283686` | **Madina97294** | 0 | 6 / 8 |

**Quand elle tire** — **3 tirs seulement**, images **1636 à 1935**, soit environ
**2:20 → 2:50** sur l'horloge du lecteur (l'horloge affichée est décalée d'environ −23 s par
rapport aux images : `frame × 0,1 s − 23 s`).

**IL N'Y A PAS DE VICTIME.** Cette arme n'est créditée d'**aucune élimination** sur ce match : elle
n'apparaît ni dans `killEffects`, ni comme source d'un kill. Elle n'existe que comme **arme PORTÉE**
(12 loadouts) et **3 tirs**. C'est donc un trou de catalogue sur une arme tenue, pas une mort mal
attribuée — et ça explique pourquoi tu la vois sur une FICHE JOUEUR et pas dans le fil.

**Reproduction :**

```bash
jq -r '[.shots[] | select(.w=="0xD791556542C9679F")] | {n: length, slots: ([.[].slot]|unique), t_min: ([.[].t]|min), t_max: ([.[].t]|max)}' \
  data/cache/replays/halo_infinite/81c02726.json
jq -r '[.loadouts[] | select(.w[0]=="0xD7915565")] | {n: length, slots: ([.[].slot]|unique)}' \
  data/cache/replays/halo_infinite/81c02726.json
jq -r '[.tracks[]? | select(.xuid != null) | {slot, xuid}] | unique_by(.slot)
       | map(select(.slot==519 or .slot==528 or .slot==535 or .slot==549))' \
  data/cache/replays/halo_infinite/81c02726.json
```

**Ce que je NE peux pas te dire** : quelle arme c'est. L'identifiant n'est ni dans les
`weaponLabels` du document (18 entrées), ni dans le catalogue du titre — c'est précisément ce que
`useReplayWeaponPads.ts:84` assume en affichant l'hexadécimal. À toi de jouer : le film est celui
du 7 septembre 21:34 sur Isolement, et l'arme est portée par BrunoArtl et Madina97294 aux instants
ci-dessus.

---

# Quatrième passe — 2026-09-08 : clôture des trois derniers points ouverts

## Point 10 a — RÉFUTÉ : le cercle du drapeau au sol EST dessiné

**Vu à l'écran.** Rejeu du match Origin (`8bc6074f`), **image 790, soit 0:56** : deux anneaux
rouges entourent les deux drapeaux posés au sol, glyphe au centre. Reproduction en trois gestes :

```
/t/halo_infinite/players/JGtm/matches/8bc6074f-d001-428b-8d6a-a755f0925572/replay
puis, en console :  document.querySelector('input[type=range]').value = 790  (+ event input/change)
```

La chaîne est complète et vérifiée de bout en bout :

| maillon | état |
|---|---|
| Règle publiée par le document | `flagReturnZone: {radiusM: 1.3, resetSeconds: 30, soloSeconds: 3.1}` |
| Lâchers présents | spans `state: "dropped"` dans `flagCarries` (le premier : images 757 → 826) |
| Construction | `buildFlagReturnDrops` (`flagReturnZone.ts:143`) — ses trois gardes sont satisfaites |
| Dessin | `drawFlagReturnZones` appelé dans `useReplayFlagCarries.ts:234`, **avant** le glyphe (« la zone passe SOUS le glyphe : c'est un décor de lieu, le drapeau est le sujet ») |

**Je ne reproduis donc pas l'absence.** Si tu la constates encore, il me faut le match et l'instant :
soit le réglage de calque était coupé, soit tu regardais un match sans drapeau au sol, soit l'anneau
est trop discret à ton niveau de zoom — et ce dernier cas serait un constat de lisibilité, pas
d'absence.

## Point 13 résidu — les 20 médias récents : les matchs ne sont PAS dans le registre

Contrôle sur la journée du **2026-01-25** (média orphelin à 16:17:54 UTC) : le registre ne contient
que **trois matchs**, de 15:09 à 15:52 UTC, et **aucun autre joueur suivi** n'a de match plus tard
ce jour-là.

```
2026-01-25 15:09:57  Catalyst   1 joueur suivi
2026-01-25 15:20:29  Prism      1 joueur suivi
2026-01-25 15:44:38  Chasm      1 joueur suivi
```

Le clip de 16:17 tombe **25 minutes après le dernier match connu**. Ce n'est donc pas un défaut
d'association : **il n'y a rien à associer**. Deux explications possibles, que je ne peux pas
départager sans interroger l'API Halo :

1. la synchronisation s'est arrêtée en cours de soirée et les matchs suivants n'ont jamais été
   récupérés ;
2. ces parties ont été jouées hors des profils suivis (autre compte, partie personnalisée, mode
   absent de `match_registry`).

`shared_pve` ne les explique pas : la table ne compte que **20 lignes** au total sur tout le parc.

**Portée : mineure.** 20 médias sur 247, tous antérieurs à février 2026, et le mécanisme
d'association lui-même est sain (143 associations correctes, delta médian 295 s).

## Point 22 — CORRECTION : le composant canonique EXISTE

> **Je me suis trompé dans la première passe** en écrivant « il n'existe aucun composant de grille
> canonique partagé ». `components/charts/Heatmap2DChart.tsx` **est** ce composant, et c'est
> exactement celui que la maquette `4c520da6` désigne par son étiquette `Heatmap2DChart`.

**Qui l'utilise déjà** : `SquadEchangeMatrixCard` (« Qui échange pour qui »),
`squad/charts/heatmapChart.ts` (« Performance par joueur × carte »), `MatchPositionsHeatmap`,
`timeseries/seriesAdapters`.

**Audit du wrapper contre la doctrine relevée** :

| Trait de la doctrine | `Heatmap2DChart` | Verdict |
|---|---|---|
| Valeur en clair dans la case | `label: { show: true }` (l. 202-203) | **conforme** |
| Rampe sûre daltonisme | `heatmapColors` + mode `frequency` mono-teinte pour une intensité sans jugement | **conforme, et bien argumenté** |
| Case impossible émise, jamais omise | `value: null` — « une case impossible doit donc être ÉMISE, et dite vide » | **conforme sur le principe** |
| **Padding entre les cases** | **aucun `itemStyle.borderWidth` / `borderRadius`** sur la série heatmap | **ABSENT** — les cases sont jointives |
| **Absence = hachure + tiret** | la case vide est « non peinte, hors échelle, sans étiquette » | **DIVERGENT** — l'absence est INVISIBLE au lieu d'être dite |
| Étiquettes de colonne sur deux lignes | axes de catégories à une ligne | absent |
| Intensité saturée à 30 points | `valueRange` existe mais aucun plafond par défaut | à décider |

**Les deux écarts qui comptent** sont précisément ceux que tu as pointés et que la maquette insiste
à porter :

1. **le padding** — en ECharts il s'obtient par `itemStyle: { borderWidth, borderColor: <fond> }`
   (+ `borderRadius` pour l'arrondi). Une ligne de plus dans le wrapper, et **les quatre
   consommateurs en profitent d'un coup** ;
2. **l'absence dite** — la maquette exige une case **hachurée portant un tiret**, légendée
   « Aucune mesure sur cet axe ». Le wrapper la laisse invisible, ce qui contredit la doctrine
   propre du dépôt (« l'absence a sa propre forme, jamais un vide »).

**Les implémentations qui NE passent PAS par le wrapper** — c'est là qu'est la vraie dette :

| Implémentation | Nature | Passe par le wrapper ? |
|---|---|---|
| `SquadEchangeMatrixCard` | grille catégorielle | **oui** |
| `squad/charts/heatmapChart.ts` (joueur × carte) | grille catégorielle | **oui** |
| `MatchPositionsHeatmap` | raster de densité sur carte | oui (axes catégoriels) |
| **`SynthesisHeatmapChart`** (heure × jour) | grille catégorielle | **NON — option ECharts construite à la main** |
| **`SessionUsageForms.UsageRegularityBand`** | bande de régularité | **NON — DOM/CSS** |
| `match-replay/layers/heatmapLayer` | raster canvas sur la carte du rejeu | non (objet différent, hors doctrine) |

Deux remarques sur ces deux-là :

- **`SynthesisHeatmapChart` est une seconde implémentation** de la même forme. Elle a ses raisons
  (rampe divergente autour de 50 %), mais elle ne bénéficiera d'aucune correction portée au
  wrapper — c'est la définition d'une factorisation abandonnée (anti-pattern n°8).
- **`UsageRegularityBand` a le padding** (`gap-[3px]`) mais des cases de **14 × 14 px sans valeur
  en clair** — la valeur n'y tient pas. C'est une MINIATURE de la forme, légitime dans son
  contexte, mais qui doit être reconnue comme une variante et non comme la grille.

**Ce qui rend le chantier réaliste** : il ne s'agit plus de créer un composant, mais d'**ajouter
deux traits au wrapper existant** (padding, absence hachurée) puis de **migrer les deux
implémentations hors wrapper**. Avec le garde-rail qu'exige la règle n°6 du dépôt — sans quoi une
troisième implémentation réapparaîtra.

### Point 10 a bis — l'anneau n'est PAS centré sur le pied du drapeau (constat utilisateur, confirmé)

> « Pourquoi le cercle n'a pas le pied du drapeau comme centre ? » — 2026-09-08.

**Deux ancrages différents pour le même point du monde**, et l'écart est écrit en clair dans le
code :

| Objet | Ancrage | Fichier |
|---|---|---|
| **L'anneau** de zone de retour | position projetée **BRUTE**, sans décalage | `flagReturnZone.ts:324` — `drawOne(ctx, project({ x: now.x, y: now.y }), …)` |
| **Le glyphe** de drapeau | position **+ 6 px en x, − 2 px en y** | `flagCarriesLayer.ts:384-385` — `const x = at.x + FLAG_OFFSET_X` / `const foot = at.y - FLAG_OFFSET_Y` |

```ts
/** Décalage du glyphe par rapport au point qu'il qualifie (au-dessus et à droite du marqueur). */
const FLAG_OFFSET_X = 6
const FLAG_OFFSET_Y = 2
```

**Le pied de la hampe tombe donc 6 px à droite et 2 px au-dessus du centre de l'anneau.** C'est
exactement le décalage visible à l'écran.

**Pourquoi ce décalage existe — et pourquoi il ne vaut plus ici.** Le commentaire le dit :
le glyphe se pose « au-dessus et à droite **du marqueur** ». Il a été conçu pour qualifier la
position d'un PORTEUR sans recouvrir son pion. Mais un drapeau **`dropped` n'a aucun marqueur à
éviter** : le drapeau EST à cet endroit, et l'anneau marque ce même endroit. Le décalage devient
alors une incohérence entre deux calques qui parlent du même point.

**Effet de bord confirmant la lecture** : le rayon de survol reprend le décalage ET la demi-hampe
(`flagCarriesLayer.ts:443-444` — `cx = c.x + FLAG_OFFSET_X`, `cy = c.y - FLAG_OFFSET_Y -
FLAG_POLE_H / 2`). Le survol vise donc le MILIEU du glyphe décalé, pas le centre de l'anneau : les
trois objets — anneau, glyphe, cible de survol — ont trois ancrages distincts.

**Deux correctifs possibles, et le second est le bon :**

1. donner le même décalage à l'anneau — les deux coïncideraient, mais l'anneau ne serait plus
   centré sur la position réelle du drapeau, ce qui est faux pour une zone de retour dont le RAYON
   (1,3 m) est une distance de jeu mesurée ;
2. **retirer le décalage du glyphe dans l'état `dropped`** — le pied de la hampe redevient la
   position publiée, l'anneau garde son centre juste, et le survol suit. C'est la seule option qui
   garde les deux objets VRAIS. Le décalage reste légitime pour `carried` (là où un pion est à
   éviter) et pour `home`.

Reproduction : rejeu Origin (`8bc6074f`), image **790** (0:56) — deux drapeaux au sol, deux anneaux,
et le décalage visible sur les deux.

#### Et le POINT DE LIVRAISON a le même écart (constat utilisateur, confirmé)

Le décalage n'est pas propre à la zone de retour : **il désynchronise le glyphe de TOUTES les
décorations d'objectif posées au même endroit**, parce que celles-ci ancrent toutes sur la position
BRUTE.

| Décoration | Ancrage | Fichier |
|---|---|---|
| Anneau de zone de retour | brut | `flagReturnZone.ts:324` |
| **Marqueur de LIVRAISON** (losange + anneau) | **brut** — `const c = px(e)` | `objectivesLayer.ts:196` |
| Marqueur d'apparition / socle | brut, même fonction | `objectivesLayer.ts:196` |
| **Glyphe de drapeau** | **+6 px / −2 px** | `flagCarriesLayer.ts:384-385` |

`FLAG_OFFSET_X/Y` n'est appliqué **qu'à un seul endroit du dépôt** — le glyphe de drapeau
(`grep -n "FLAG_OFFSET" layers/*.ts` : 4 occurrences, toutes dans `flagCarriesLayer.ts`). Tout le
reste de la couche objectifs est à l'ancrage brut. **Le glyphe est donc le seul objet décalé, et il
l'est partout** : sur sa base, sur le point de livraison, dans la zone de retour.

Ça conforte le correctif n°2 : le décalage n'a de sens **que pour l'état `carried`**, où il évite
de recouvrir le pion du porteur — ce que dit son propre commentaire (« au-dessus et à droite **du
marqueur** »). Dans tous les autres états, il n'y a aucun marqueur à éviter et il fait mentir la
position. Un seul `if` sur l'état, et les trois écarts se referment ensemble.

---

## Point 5 — CLOS À LA SOURCE : un échantillon aberrant définissait les bornes

**Correctif livré le 2026-09-08**, branche `wt/bornes-aberrantes`, commit `490dc595e`.

`boundsOf` était un min/max BRUT. Sur `81c02726` (Isolement, Bases), **un point sur 16 064** —
`(x=-78.60, y=+46.38, z=-325.4)` quand le sol joué est à 117,8 en médiane — fixait à lui seul
`MinX`, `MaxY` et `MinZ`, et faisait échouer `coversPlayedArea` sur une image qui couvre 99,99 %
des positions réelles.

Règle retenue : rejet par centiles (étendue centrale p1..p99, seuil 12 étendues), planchers à
200 échantillons et 0,5 m. Calibré sur le parc — écarts artefactuels à partir de 17,7 étendues,
plus grand écart légitime mesuré à 9,5.

| Contrôle | Résultat |
|---|---|
| Innocuité sur le parc | **10 points écartés sur 2 131 593** (0,000469 %), 7 artefacts touchés |
| Effet sur `81c02726` | X [−78.60, −1.14] → [−49.19, −1.14] · Y [−37.06, 46.38] → [−37.06, −6.46] |
| `coversPlayedArea` | **avant = faux, après = vrai** |

**Reste la recuisson** des 7 artefacts : les documents déjà cuits gardent leurs bornes. Portée par
la tâche hors lot du plan de vague C.

---

## Point 18 ter — LE BLOC D'ÉQUIPEMENT : la mesure tranche la forme (2026-09-09)

Relevé sur les 128 artefacts du cache. Ce qui suit ferme le débat « faut-il fusionner les colonnes
*déployé* et *lâché* ».

**1. Ce sont deux issues du même événement, pas deux grandeurs.** Les deux sortent du canal
`equipmentPlacements`, discriminées par le seul champ `origin` (`equipmentUsageLogic.ts:306`).

**2. Un lâcher est une MORT, jamais un geste.** `equipmentOrigin`
(`analysis/replay/equipment_placements.go:275`) ne classe en `dropped` qu'à moins de **200 ms ET
1,5 m** de la dernière position du porteur. Les populations mesurées sont séparées par trois ordres
de grandeur : lâchers à 20-38 ms et 0,63 m ; déploiements à **14-42 secondes** et 5,6-21,3 m.
Confirmé par un second canal : un mur lâché porte l'identifiant de l'APPAREIL, jamais celui des
panneaux que le déploiement produit.

**3. Une mort lâche EXACTEMENT UN objet, jamais un par charge restante.** Poses `dropped` groupées
par (poseur, famille, instant) :

```
wall           {1:249, 2:1}  moy 1.00     sensor        {1:302}  moy 1.00
shroud_screen  {1:60}        moy 1.00     repair_field  {1:13}   moy 1.00
threat_seeker  {1:16}        moy 1.00     translocator  {1:8}    moy 1.00
grenade_frag   {1:812, 2:2044, 3:13, 4:20}  moy 1.74   <- la PILE PORTÉE, pas des charges
```

**4. Le canal des charges ne couvre pas ces familles.** `abilityCharges` ne porte que `grapple` et
`thruster` (valeurs 0..4, 900 et 538 lectures). Et comme rien n'est transmis au ramassage, même le
maximum n'est pas établissable. Les charges annoncées par l'utilisateur (écran occultant 2,
capteur 4-5, champ de réparation 1-2, répulseur 3) sont **hors de portée de l'artefact**.
Décision du 2026-09-09 : on ne compte pas les charges.

**5. `spent` ne nomme jamais l'objet.** Les 458 `spent` du parc portent tous `r = -1`
(`AbilitySetNoRank`). Le canal dit « ce joueur ne porte plus d'équipement », jamais lequel — une
mesure « utilisés par famille » fondée sur lui exigerait une inférence, pas une lecture.

**6. Le défaut de forme actuel, chiffré.** `ValueGrid` donne à chaque colonne SA propre échelle.
Sur le match `4f77afc1`, colonne « Mur » : la barre *déployé* de `Dafar8423` (3) et la barre
*lâché* de `nerdpuncher` (2) sont **toutes deux pleines**. Deux barres identiques à l'œil pour deux
valeurs différentes. Et colonne « Écran occultant », la colonne *déployé* est vide : que **personne
de la partie ne s'en soit jamais servi** est aujourd'hui invisible.

**Forme retenue (décision D9 du plan de vague C, validée sur maquette le 2026-09-09) :** une
colonne par famille, barre empilée *utilisé / mort en le portant*, échelle commune par colonne.
Power-ups exclus (leur usage est un ÉPISODE, pas une pose). Le total n'est **pas** nommé
« ramassés » : un usage est une charge, un lâcher est un objet.

**Part utilisée par famille, sur le parc :**

| Famille | utilisé | lâché | part utilisée |
|---|---|---|---|
| Mur | 295 * | 251 | 54 % |
| Capteur | 49 | 302 | **14 %** |
| Écran occultant | 9 | 60 | 13 % |
| Traqueur de menaces | 3 | 16 | 16 % |
| Champ de réparation | 3 | 13 | 19 % |
| Balise de translocation | 0 | 8 | 0 % |

\* poses brutes, panneaux non filtrés — le graphe en comptera moins.

**Découverte non traitée** : le capteur est à **1:6** quand le mur est à 1:1. Vrai comportement de
jeu ou défaut de classement d'origine — à vérifier dans un chantier à part.

---

## Point 19 C — RÉOUVERT : l'utilisateur nomme un candidat, et les pièces le soutiennent

**Hypothèse utilisateur du 2026-09-09** : « l'arme inconnue sur ce match a l'air d'être le
mutilateur » (dernier match sur Isolement).

Le match est `81c02726`. Son catalogue `weaponLabels` porte **18 entrées, et aucun mutilateur**.
L'arme non cataloguée qui y apparaît est **`0xD7915565`** — exactement l'identifiant que ce
registre avait classé « inconnu ASSUMÉ ».

**Ce que les canaux en disent, et qui va dans le sens de l'hypothèse :**

| Canal | Occurrences de `0xD7915565` |
|---|---|
| `weaponPads` | **1** — l'arme apparaît sur un SOCLE, donc elle est posée par la carte, pas par un équipement de départ |
| `shots` | 3 (variante `0xD791556542C9679F`) |
| `groundWeapons` | 3 |
| `weaponChanges` | 3 |

Le fait décisif est le **socle** : `0xD7915565` n'est dans aucun `loadout`, il naît d'un point
d'apparition de la carte et n'est ramassé que trois fois. C'est le profil d'une arme de ramassage
peu utilisée — cohérent avec un mutilateur sur Isolement, sans que ça l'établisse.

**Ce qui trancherait, et que je n'ai pas fait :** comparer la POSITION du socle porteur de
`0xD7915565` sur `81c02726` avec l'emplacement connu du mutilateur sur Isolement
(`data/titles/halo_infinite/reference/map_weapon_pads.json`), ou une vérification Theater sur l'un
des trois ramassages. Une correspondance de position vaut identification ; l'hypothèse seule, non.

**Statut** : le point n'est plus « inconnu assumé » mais « candidat nommé, à confirmer par la
position du socle ». Ce n'est PAS un item de vague A (question de donnée, pas de forme).
