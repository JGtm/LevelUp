# Lot B — corrections du décodeur reprises du fork ChaseWoodhams (2026-09-11)

> Branche `wt/decodeur-fork`, base `feat/v75` `81f15be30`. Source des défauts :
> `.ai/BILAN_FORK_CHASEWOODHAMS_2026-09-11.md` points 1, 2 et 5. Les idées viennent du fork
> (`1a7946bb3`, `621f69040`) ; **rien n'a été repris comme commit** — 2 570 commits de
> divergence. Toutes les mesures ci-dessous sont les NÔTRES, sur NOTRE parc.

## Ce qui est livré

| Item | Commit | Objet |
|---|---|---|
| B.1 | `0288a35bd` | Banc de mesure de l'écart lancer → lanceur, gardé par environnement |
| B.2 | (commit suivant) | `locateThrow` : l'auteur juge, le slot est publié |
| B.3 | `905c720c4` | Un vol de projectile s'arrête au premier pas impossible, et le dit |
| B.4 | `70edf37b6` | Les props Forge appartiennent à une carte, plus au titre |
| B.5 | `b6b198baf` | `SchemaVersion` 51 → 52, chronique datée, goldens |

---

## B.1 — Mesure : un lancer est-il posé sur son lanceur ?

Banc `internal/analysis/replay/grenade_ecart_research_test.go`, gardé par
`GRENADE_ECART_PARC` + `GRENADE_ECART_FILMS` (il se saute sans films, comme les autres
instruments du paquet). Il cuit le film, reconstruit le pont d'identité depuis les lectures que
l'observateur rend, et mesure la distance entre la position PUBLIÉE du lancer et le biped de son
auteur au même instant.

Le film `36e80b83` du fork n'existe pas dans notre parc ; les trois films retenus sont
`000d5950` (Cliffhanger, film de référence du chantier) et deux films Live Fire.

### Avant correctif

| Film | Carte | n | médiane | p90 | pire cas | > 4 m | fenêtres à 2+ naissances |
|---|---|---|---|---|---|---|---|
| `000d5950` | Cliffhanger | 64 | 0,44 m | 0,48 m | **14,46 m** | 2 (3,1 %) | 15 / 70 (21,4 %) |
| `0797ce72` | Live Fire | 84 | **25,42 m** | 32,17 m | 34,72 m | 84 (**100 %**) | 20 / 103 (19,4 %) |
| `21ece4d8` | Live Fire | 137 | **26,69 m** | 31,45 m | 35,12 m | 137 (**100 %**) | 21 / 156 (13,5 %) |

Témoin : sur la branche BIPED — qui lit la position de l'auteur — l'écart vaut 0,00 m sur les
trois films. La mesure lit bien ce qu'elle croit lire.

### Après correctif

| Film | branche projectile | médiane | pire cas | > 4 m | lancers publiés |
|---|---|---|---|---|---|
| `000d5950` | 65 (inchangé) | 0,44 m | **0,56 m** | **0** | 70 (banc) / 69 (chaîne complète) |
| `0797ce72` | 84 → **1** | 0,00 m (biped) | — | 0 | 103 (inchangé) |
| `21ece4d8` | 137 → **0** | 0,00 m (biped) | — | 0 | 156 (inchangé) |

Deux régimes, deux lectures :

- Sur **Cliffhanger**, le défaut est celui que le fork décrit : quelques lancers reçoivent le
  projectile du voisin. Le pire cas tombe de 14,46 m à 0,56 m, et la médiane ne bouge pas — le
  correctif resserre sans déplacer le signal.
- Sur **Live Fire**, la totalité des naissances est fausse, et **ce n'est pas un défaut
  d'attribution** : c'est le repli de quantum de B.3, qui déplace la naissance d'une demi-étendue
  de carte (31,89 m — à comparer aux médianes de 25 à 27 m mesurées). Le garde-fou d'auteur les
  refuse toutes, et les lancers se lisent sur le biped, à 0,00 m. **Aucun lancer n'est perdu** :
  103 et 156, comme avant.

---

## B.2 — Correctif `locateThrow`

L'auteur est résolu d'abord (`authorBiped` : `slotFor` + biped à `TimestampUS`, tolérance
`shotPosToleranceUS`). Parmi **toutes** les naissances de la fenêtre `grenadeBirthWindowUS`
(`birthsInWindow`, et non plus la plus proche dans le temps), on garde celle qui est à portée de
la main de l'auteur, et aucune au-delà de `grenadeAuthorRadiusM = 4` m. Sans auteur ponté : une
candidate unique reste une lecture, plusieurs sont un refus. Le repli biped est inchangé ;
`Grenade.Proj` suit la naissance retenue ; le tri total est conservé.

`grenades[].slot` est désormais publié sur les deux branches. Il sortait à zéro sur la branche
projectile — et zéro RESSEMBLE à un slot.

**Le jumeau côté client n'existe pas.** Vérifié le 2026-09-11 : `apps/web/src/features/
match-replay/` ne porte ni `grenadeArcs.ts` ni `ARC_ORIGIN_RADIUS_M` — les arcs de lancer ne sont
pas dessinés chez nous. Le commentaire de la constante le dit, pour que le jour où ce seuil
naîtra côté web il vaille ce nombre et cite celui-ci.

**Tests** : fenêtre à une naissance et auteur connu (slot publié) ; deux lanceurs dans la même
fenêtre, chacun reçoit la sienne ; naissance à 31 m refusée → repli biped, lien `proj` nil ;
sans biped et deux candidates → refus compté en `noSlot`.

**Golden `000d5950`** : 70 → 69 lancers posés, dénominateur inchangé à 70, répartition par source
65/5 → 63/6. Le lancer perdu (`ts=5012458806`, index de joueur 1) n'a **pas** d'auteur ponté et sa
fenêtre porte **dix** naissances, dont huit au même instant à moins d'un mètre les unes des
autres — une rafale de sous-projectiles. L'ancien code en prenait une par proximité temporelle,
soit un tirage au sort sur dix.

---

## B.3 — Garde-fou du pas de projectile, et la forme du défaut

`buildProjectiles` coupe au dernier point lisible dès qu'un pas dépasse `projectileMaxStepM = 10`
m, sans recoudre. `Rest` tombe à `false` sur un vol coupé : il CERTIFIE une fin de vol, et un vol
coupé n'a pas la sienne. Le compte remonte à `coverage.projectiles`
(`tracks` / `published` / `truncated`, champ optionnel) et au journal.

### La forme, par film — et elle contredit le fork

Mesuré sur les 76 artefacts du parc (schéma 51, donc AVANT la coupure) : **947 trajectoires sur
15 735 (6,0 %)** portent au moins un pas > 10 m, soit **4 901 pas**.

Le fork décrivait « le quantum Y repasse d'un bord à l'autre, le saut vaut l'étendue Y ». **Ce
n'est pas ce que notre parc montre.** Le saut vaut l'étendue de la carte sur un axe **divisée par
une puissance de deux** — le poids d'**UN bit** du champ quantifié :

| Film | Carte (module) | pas coupés | \|Δy\| médian | \|Δx\| médian | forme |
|---|---|---|---|---|---|
| `0797ce72` | Live Fire (`sgh_interlock`) | 1 326 | **31,89 m** | 0,17 m | 90 % à 2⁻ᵏ de l'étendue Y |
| `21ece4d8` | Live Fire (`sgh_interlock`) | 1 149 | **31,88 m** | 0,24 m | 99 % à 2⁻ᵏ de l'étendue Y |
| `30724141` | Live Fire (bornes, hors registre) | 943 | **31,89 m** | 0,23 m | idem |
| `c88ec007` | Live Fire (`sgh_interlock`) | 489 | **31,89 m** | 0,17 m | 100 % à 2⁻ᵏ de l'étendue Y |

L'étendue Y de `sgh_interlock` vaut **63,775 m**, sa largeur d'axe Y est de **12 bits**. La
médiane 31,89 m est **exactement la moitié** : c'est le **bit de poids fort de Y** qui bascule, et
X ne bouge pas. Cette forme couvre **3 907 des 4 901 pas** du parc.

Les 994 pas restants sont sur d'autres cartes, et **l'axe touché s'inverse** : |Δx| médian
21,69 m, |Δy| médian 0,00 m, sur les grandes cartes Forge (`banished narrows` 489 pas, `the pit`
82, `isolation` 57…). Le rapport à l'étendue y est majoritairement **2⁻⁷** — un bit de poids plus
faible. Sur 3 907 + 994 pas, **97 % ont un saut égal à étendue / 2ᵏ pour un k de 1 à 7**.

> **Découverte, non traitée dans ce lot** : la cause est un **basculement d'un seul bit du champ
> quantifié** dans la déquantification (`filmdec`) — pas un repli de plage, pas un bit de signe.
> Sa concentration sur `sgh_interlock` (Live Fire) au bit de poids fort de Y, et son déplacement
> vers X et un bit plus bas sur les cartes Forge, orientent vers un **décalage de lecture d'un
> bit** dépendant du découpage d'axes de la carte (Live Fire est justement la carte qui a imposé
> le découpage d'i0 par catalogue, cf. `build_from_film.go` — elle a plus de deux régions de
> compression). C'est la piste à instruire, avec le corpus de témoins que le garde-fou compte
> désormais dans chaque artefact.

**Golden `000d5950`** (Cliffhanger, carte peu touchée) : 439 → 436 trajectoires publiées,
2 732 → 2 725 points, **3 coupures**. Les trois tombent dès le deuxième point de grille, et une
trajectoire d'un seul point ne se dessine pas — d'où l'écart de 3 trajectoires pour 7 points
seulement.

---

## B.4 — Props Forge par carte, et l'attribution du CSV

`MapGeometryDir(titleSlug, module)` rend un répertoire par module — la même clé que
`map_quant_bounds.json`, `MapStructurePath` et `MapBackgroundPath`. Module vide = le répertoire du
titre, celui du **catalogue des types** (`forge_object_types.csv`), qui vaut pour toutes les
cartes. `LoadGeometry(mapDir, typesDir)` prend donc deux répertoires. `replaybuild` charge les
props de LA carte du match, avec cache par module (même motif que `structures`).

Une carte sans fichier de props rend **zéro prop et aucune erreur** — c'est le cas nominal
(79 cartes au catalogue de bornes, une seule extraite), journalisé en **Debug**. Un catalogue de
types illisible reste un **Warn** : c'est une panne du titre, pas de la carte.

### Le CSV est attribué à `ridgeline` (Cliffhanger), pas laissé sous `UNATTRIBUTED/`

Deux lignes de preuve indépendantes, écrites dans
`data/titles/halo_infinite/reference/map_geometry/ridgeline/README.md` :

1. **Emprise contre aire jouée.** Les 453 props couvrent X [-10,56 ; 44,00], Y [-24,65 ; 39,02].
   L'aire jouée de Cliffhanger, relevée sur les artefacts du parc, est X [-7,7 ; 44,2],
   Y [-25,7 ; 36,4] : **90,8 %** de l'emprise des props y tombe, pour un **rapport de surfaces de
   1,078**. Aucune autre carte du parc ne tient les deux critères à la fois — les suivantes
   couvrent une fraction des props (Catalyst 45 %, Aquarius 36 %) ou sont si grandes qu'elles
   avalent n'importe quelle emprise (Fortitude : 98 % de couverture pour une aire 25 fois trop
   vaste, où 382 props occuperaient 4 % du terrain).
2. **Provenance.** Le commit qui a introduit le CSV (`2044b7139`) ne porte que deux `.mvar` dans
   son dump de RE — `cliffhanger_map.mvar` et `cliffhanger_ridgeline.mvar` — et Cliffhanger
   (`000d5950`) est le film de référence de tout le chantier rejeu.

Le README dit que c'est une attribution **déduite** et donne son test de falsification : cuire
`000d5950` et vérifier que les props tombent sur la géométrie plutôt qu'à côté. En cas d'échec,
le fichier retourne sous `map_geometry/UNATTRIBUTED/` — la donnée est peut-être juste, c'est son
attribution qui est raisonnée.

**Tests** : `MapGeometryDir` par carte et par titre, module vide = répertoire du titre ; carte
sans fichier = nil sans erreur ; catalogue de types absent = erreur.

---

## B.5 — Schéma 52, goldens, gate corpus

`SchemaVersion` 51 → 52. Le bump est **exigé par la règle** (le contenu cuit change), pas posé
pour déclencher la recuisson : un artefact v51 porte des lancers posés sur le mauvais joueur, des
vols traversant la carte, et le décor d'une autre carte. Chronique datée dans
`document_chronicle.go`, raison dans le garde de `structure_test.go`, au format des précédentes.
`coverage.projectiles` s'ajoute au passage — un champ optionnel, qui ne l'aurait pas exigé seul.

Contrat et types régénérés : `api/openapi.yaml` (+ `ProjectileCoverage`), `generated.ts`, jumeau
`domain/replaydoc` et projection `replayview`. Le garde de parité et le contrat ont tous deux
attrapé l'écart avant moi.

### Gate corpus — verdict ligne par ligne

`go run ./cmd/replay-corpus-gate --reference=base --base=origin/feat/v75
--parc-root=<LevelUp-go-migration>` : les 7 témoins cuits deux fois (code HEAD et code
`origin/feat/v75`), aucun ABSENT, **aucune écriture dans le parc** (racine de travail jetable).
Sortie 1, **toutes les pertes sont voulues et attendues**.

| Axe | Témoins | Différence | Verdict |
|---|---|---|---|
| `geometry.*` et `geometryBounds` | **les 7** | 382 props → absents | **VOULU (B.4)** — aucune de ces 7 cartes n'a d'extraction de props ; elles recevaient celle de Cliffhanger |
| `projectiles.p/n` et `/total` | les 7 | 408→388, 2969→2692, 2519→2505, 1144→1141, 2188→2183, 2012→1862, 9923→9816 | **VOULU (B.3)** — points retirés après un pas impossible |
| `projectiles/n`, `projectiles.t0/presents` | 5 sur 7 | 283→258, 194→191, 135→134, 238→228, 898→873 | **VOULU (B.3)** — trajectoires coupées si tôt qu'elles n'ont plus deux points de grille |
| `projectiles.rest/presents` | `bf15f7ab` 9→7, `084a804d` 60→52 | — | **VOULU (B.3)** — un vol coupé ne certifie plus sa fin |
| `grenades/n` et ses champs | `c75f33b8` 67→65, `51ebbc0f` 135→134 | — | **VOULU (B.2)** — abstentions sur fenêtre ambiguë sans auteur ponté |
| `coverage.grenades.attached` / `noSlot` | `c75f33b8` 67→65 / 1→3, `51ebbc0f` 135→134 / 0→1 | — | **VOULU (B.2)** — l'invariant tient : les refus sont COMPTÉS, pas avalés |

Gains : **5 sur chacun des 7 témoins** (l'outil n'imprime que les pertes ; les trois métriques
neuves `coverage.projectiles.{tracks,published,truncated}` en font partie).

**Aucune perte inattendue** : pas un seul axe touché hors des trois calques du lot — ni tirs, ni
vies, ni identité, ni score, ni objectifs, ni véhicules, ni zones.

**La recuisson du parc (`backfill-replay`) n'est PAS lancée** : c'est le pilote qui la joue après
fusion, serveur arrêté.

---

## Gates

| Gate | Résultat |
|---|---|
| `go test ./...` (module complet, CGO) | vert |
| `go vet ./...` | vert |
| `make go-api-lint` (ratchet `--new-from-merge-base=origin/main`) | **0 issue** |
| `no_slug_comparison_test`, `archlint` | verts |
| Bancs sans films | se sautent |
| Fichier > 500 L / fonction > 80 L introduits | aucun |
| `fmt.Println` / `log.Printf` | aucun |

Dette de taille de fichier : `build.go` 586 → 604, `replaybuild.go` 532 → 569, `registry.go`
1033 → 1055, `coverage.go` 383 → 410. Les trois premiers étaient DÉJÀ au-dessus du seuil
(baseline lint) et grossissent de 18 à 37 lignes, presque entièrement en commentaires de
justification. `coverage.go` reste sous le seuil. **Découverte à traiter ailleurs** : `build.go`
et `replaybuild.go` méritent une extraction ; elle est hors périmètre de ce lot.

---

## Découvertes (non traitées)

1. **La cause racine du pas impossible est un basculement d'UN BIT**, pas un repli de plage : le
   saut vaut étendue / 2ᵏ (97 % des 4 901 pas du parc, k de 1 à 7). Concentré sur
   `sgh_interlock` au bit de poids fort de Y (k=1, 3 907 pas) ; sur les cartes Forge, l'axe
   touché est X et k vaut plutôt 7. Chantier `filmdec`, à instruire avec le compteur
   `coverage.projectiles.truncated` comme témoin.
2. **`30724141` n'est pas dans l'instantané du registre partagé** (snapshot v93, 2026-09-09) alors
   que son artefact est cuit : 17 des 78 films du parc sont dans ce cas. Sa carte a été
   identifiée par ses bornes (Live Fire), pas par la base.
3. **`build.go` (604 L) et `replaybuild.go` (569 L)** dépassent le seuil de 500 L depuis avant ce
   lot et continuent de grossir. Une extraction est due.
4. **Le fork chiffrait le défaut des grenades sur son parc, pas sur le nôtre** : sa mesure
   (« 171 lancers sur 247 sans aucun joueur à moins de 4 m, médiane 7,95 m ») ne se retrouve chez
   nous sous cette forme sur aucun des trois films. Chez nous le défaut est soit marginal
   (Cliffhanger, 2 lancers sur 64) soit total et d'une AUTRE cause (Live Fire, le repli de
   quantum). Les deux appellent le même correctif, mais le diagnostic diffère.

## Non fait, et pourquoi

- **Cause racine de la déquantification** : hors périmètre explicite du lot (« tu la caractérises
  et tu la consignes »). Caractérisée ci-dessus.
- **Recuisson du parc** : réservée au pilote, après fusion, serveur arrêté.
- **Attribution des props des 78 autres cartes** : une seule extraction existe.
- **`--reference=parc`** : non joué. Le parc est plus ancien que HEAD, la comparaison serait
  informative seulement ; le mode `base` est le gate d'autorité avant fusion.
