# GATE — V13 : le dead-state `i11` des véhicules, lu par la MARCHE et non par l'ancre

> Écrit le 2026-09-05 **AVANT toute mesure** (méthode §8 du handoff véhicules :
> « le gate écrit AVANT la mesure, avec témoin »). Branche `wt/vehicule-deadstate`,
> partie de `origin/feat/v75-vehicules-sons` — worktree dédié, celui du collègue
> (`LevelUp-wt-vehicules`) n'est pas touché.

## 1. Pourquoi cette mesure, et pourquoi maintenant

Le registre des pistes ouvertes du handoff (§7.2) dit exactement ceci :

> `i11` dead-state est **sous-instrumenté, pas réfuté** : l'ancre est aveugle au dead-state y
> compris sur le bipède (où les morts sont certaines). Verrou = désynchronisation de
> `frame_records.go` (2,4 % de couverture `ti=40` contre 24,5 % pour le bipède).

Deux acquis, tirés de deux chantiers qui ne se sont jamais croisés, se combinent ici :

1. **La grammaire du dead-state de `ti=40` est PROUVÉE au désérialiseur** (lot V9/V10, Ghidra) :
   `FUN_140c1dce0` lit `Mort = R(1)`, puis appelle le corps lourd `FUN_140c1dd44` si
   `ti == 0x23` **ou** `ti == 0x28`, et ne lit le bit de queue que si `ti == 0x23`. Pour un
   véhicule, la forme est donc celle du bipède MOINS le bit de queue — le même corps lourd qui
   résout l'arme du kill à 97,6 %. Le dépôt le porte déjà
   (`traverse.go:800`, `components_object.go:182`).
2. **Le verrou n'est pas la grammaire, c'est l'ANCRE.** `ScanFilmBipedPositionsForBand` n'accepte
   qu'un record dont le masque commence par un `i0` ABSOLU : il n'atteint jamais `i11`. C'est le
   témoin `v10ControlBiped` qui l'a établi — l'ancre est aveugle au dead-state **même sur le
   bipède**, dont les morts sont un fait du corpus.
3. **Il existe un lecteur qui ne passe PAS par l'ancre** : la marche du décodeur de source de
   dégât (`killsource` : timeline chronologique + localisateur d'events + 8 vues de réplication
   + `DesyncAt == -1`), portée dans `filmdec` sur `feat/v75`. C'est elle qui a fait passer un
   film de 5 à 26 morts détectées. Elle n'a **jamais** été essayée sur la bande `ti=40`, et elle
   ne l'a jamais été **sous la grammaire `ti=40` corrigée** (§5.2 du handoff : un correctif de
   2026-07 avait réparé `ti=35` en cassant `ti=40`).

La mesure V13 est l'intersection de ces trois faits : **la marche × la grammaire corrigée ×
la bande véhicule.**

## 2. Élément neuf de RE apporté à cette mesure (2026-09-05, Ghidra)

Le pipeline de mort du moteur a été relu ; il corrobore le lot V9 par l'autre bout :

- `FUN_1404d9828` applique le dégât `jpt!` sur **n'importe quel objet**, accumule
  `object-body-vitality` / `object-shield-vitality`, et sur le coup fatal appelle
  `FUN_140adefbc` → `FUN_142c4e850`.
- `FUN_142c4e850` est **le classifieur de mort**. Il choisit l'événement selon la victime : si la
  victime n'a pas d'index joueur (`+0x484 == -1`) et que c'est un véhicule, il émet
  `enemy_vehicle_kill` ; sinon `enemy_kill`, trahison, suicide. `FUN_142c4dcf8` calcule
  l'**assistance** (`vehicle_destroy_assist`) en parcourant la liste des contributeurs de dégât.

**Conséquence** : une destruction de véhicule est traitée par le MÊME code de mort qu'un kill de
joueur. Il n'existe d'ailleurs aucun composant ECS « véhicule détruit » — le binaire ne déclare
que des composants **d'objet** (`object-body-vitality`, `object-shield-vitality`,
`object-damage-sections`, `object-dead-state`). Le dead-state est donc le seul signal d'état
graves à la mort, et il est générique.

Cela ne PROUVE pas que le film réplique ce dead-state pour un véhicule — le film est une capture
de réplication, pas la simulation serveur. C'est précisément ce que la mesure tranche.

## 3. LE GATE — écrit avant la mesure

### G1 — Témoin de non-régression bipède (bloquant)

Le compte de dead-states retenus **en bande bipède** doit être **strictement identique** avec et
sans le filtre de bande.

*Mise en œuvre* : le filtre de bande est **post-hoc par construction** — la marche récolte tous
les dead-states propres, puis les range par archétype. La marche est donc identique bit pour bit
dans les deux cas, et la non-régression est structurelle. Les deux comptes sont **quand même
imprimés** côte à côte : s'ils divergent, l'instrument est faux et la mesure est nulle.

*Second témoin, mesuré celui-là* : le compte bipède de la marche est confronté au compte `i11`
de l'ancre (`v10ControlBiped`) sur le même film. **La marche doit en trouver STRICTEMENT PLUS.**
Si elle en trouve autant ou moins, elle n'apporte rien et la mesure est nulle.

### G2 — Témoin négatif (bloquant s'il est disponible)

Sur un film où **aucune entité `ti=40` n'est déclarée aux images-clés**, le compte de dead-states
en bande véhicule doit être **0**. Assertion dure dans le test.

*Réserve honnête* : si l'échantillon mesuré ne contient aucun film sans véhicule, G2 est déclaré
**non évalué** — on ne l'invente pas.

### G3 — Couverture publiée à côté de tout compte (bloquant)

Aucun compte de dead-states véhicule n'est publié seul. Chaque compte est accompagné de :

- le nombre de records `ti=40` **décodés proprement** (`DesyncAt == -1`) et le total marché ;
- le même couple pour `ti=35` ;
- le nombre de slots distincts `ti=40` **déclarés aux images-clés** et le nombre **atteints** par
  la marche.

**Règle de lecture, posée d'avance : un compte faible avec une couverture faible ne conclut PAS à
l'absence.** Il conclut « toujours sous-instrumenté », et le verrou reste la couverture.

### G4 — Vérité terrain

Le compte de destructions se confronte à `VehicleDestroys` de l'API, **joueur par joueur**, pas au
seul total. Le champ est parsé (`internal/openspartan/halo_api_payload.go:134`) mais ne semble
persisté nulle part : si la vérité terrain n'est pas récupérable dans cette passe, elle est
déclarée **hors passe**, pas contournée.

*Note* : `VehicleDestroys` donne un COMPTE par joueur et par match, pas une DATE. Il borne les
faux positifs d'un détecteur ; il ne date pas une destruction. La datation reste au Theater
(§3.1 du handoff) ou au dead-state si G1..G3 passent.

### Décision — écrite d'avance

| Résultat | Verdict |
|---|---|
| G1 vert, G3 publié, taux de résolution du champ tueur (`EnumB`) sur les dead-states véhicule **du même ordre que chez le bipède** | **Dead-state véhicule EXPLOITABLE** — le Go peut publier `end = "destroyed"` + `tEnd`, et l'effet d'explosion déjà câblé s'allume |
| G1 vert, dead-states véhicule **présents mais champ tueur non résolu** | **Datation seule** — `end`/`tEnd` publiables, pas d'attribution |
| G1 vert, **0 dead-state véhicule** avec couverture `ti=40` **haute** | **Négatif MESURÉ** — le film ne réplique pas la mort d'un véhicule ; la piste §7.2 se ferme, le Theater reste le chemin |
| Couverture `ti=40` **basse** | **Toujours sous-instrumenté** — on ne conclut rien, le verrou reste la couverture |

## 4. Ce que cette mesure ne fait PAS

- Elle ne rouvre pas les négatifs mesurés du §6 du handoff. En particulier :
  **`i4 → 0` est un mauvais modèle** (le véhicule disparaît), le bon signal candidat est le
  palier d'épave ; et la datation par **type d'événement** est réfutée sur les 28 types (V7).
- Elle ne touche à **aucun code de production** : instrument de recherche sous garde d'env,
  lecture seule, un film par process (`LockProcessDecode`).
- Elle ne préjuge pas de la piste des **événements nommés** (`vehicle_death`,
  `enemy_vehicle_kill`, `vehicle_destroy_assist`, hachés en `murmur3 x86_32 seed 0` par
  `FUN_140748a74`). Cette piste est distincte des 28 TYPES d'événements réfutés par V7 : elle
  porte sur le VOCABULAIRE des propriétés nommées. Elle est notée, non traitée ici.

---

## 5. VERDICT (ajouté après la mesure — 2026-09-05)

Résultats complets : `.ai/V7.5/film_re/NOTE_V13_DEADSTATE_VEHICULE_2026-09-05.md`.

**Verdict : dead-state véhicule LISIBLE.** 21-27 dead-states d'archétype 40 par film, la plupart
dans la bande véhicule, champ tueur renseigné. La ligne « 0 dead-state avec couverture haute =
négatif MESURÉ » du tableau de décision n'a PAS été atteinte : elle l'a semblé pendant une
itération, et c'est le contrôle de masque (G3) qui l'a démentie.

**Deux décisions prises APRÈS l'écriture du gate, consignées ici pour qu'elles restent visibles :**

1. **G1a est passé d'assertion à rapport.** Écrit comme une assertion, il a échoué — en disant
   vrai : la bande dérivée des images-clés EST incomplète (le film lie des entités par records
   NEW en cours de flux). Comme l'instrument range par ARCHÉTYPE et non par bande, la conclusion
   n'en dépend pas ; le chiffre est devenu ce qu'il mesure vraiment, à savoir ce que le filtre de
   bande aurait perdu (2 à 5 dead-states bipèdes par film).
2. **Le filtre `DesyncAt == -1` a été assoupli** : un dead-state est accepté quand la rupture est
   POSTÉRIEURE à l'index du composant dead-state, et cette classe est comptée à part. Justification
   mesurée, pas de confort : 65 des 69 records `ti=40` qui déclarent le dead-state rompent à
   `i30`..`i36` (`vehicle-auto-turret-*`), tous après `i11`. Le gate G3 tenait — c'est lui qui a
   imposé de regarder les masques plutôt que de conclure à l'absence.

**Le critère de décision du gate reste à instruire** : « taux de résolution du champ tueur du même
ordre que chez le bipède » est plausible sur les échantillons mais n'est pas encore chiffré après
groupement en épisodes de mort. Tant qu'il ne l'est pas, la datation (`end`/`tEnd`) est le
livrable candidat, pas l'attribution.
