# V13 — LE DEAD-STATE DES VÉHICULES EST DANS LE FILM, ET IL EST LISIBLE

> Mesure du 2026-09-05, branche `wt/vehicule-deadstate` (partie de `origin/feat/v75-vehicules-sons`).
> Gate écrit AVANT la mesure : `.ai/V7.5/GATE_V13_DEADSTATE_MARCHE_2026-09-05.md`.
> Instrument : `internal/analysis/filmdec/vehicules_v13_{deadstate,marche_helpers}_test.go`,
> garde `V13_FILMS`, lecture seule, aucun code de production touché.

## 1. Le résultat

La piste §7.2 du handoff (« `i11` dead-state est sous-instrumenté, pas réfuté ») est **tranchée
par l'affirmative**. Sur 6 films à véhicules, la mesure rend **21 à 27 dead-states d'archétype 40
par film**, la plupart dans la bande véhicule dérivée des images-clés, et **le champ tueur
(`EnumB`) y est renseigné** comme chez le bipède.

Deux verrous ont dû tomber, dans cet ordre. Le premier était connu, le second ne l'était pas.

## 2. Verrou 1 — l'ancre (connu) : la marche la bat 60 contre 0

Le handoff l'avait diagnostiqué : `ScanFilmBipedPositions` n'accepte qu'un record dont le masque
commence par un `i0` absolu, donc n'atteint jamais `i11`. Mesuré ici, sur le bipède, dont les
morts sont certaines :

| film | ancre : records `ti=35` acceptés | dont portant `i11` | LA MARCHE : dead-states `ti=35` |
|---|---:|---:|---:|
| `0d76e8f1` | 207 808 | **1** | **66** |
| `51d3ab9f` | 231 817 | **0** | **60** |
| `e1bdb97f` | 184 653 | **0** | **59** |
| `b232e02d` | 240 115 | **0** | **65** |
| `21468645` | 153 535 | **0** | **47** |
| `4898d586` | 213 939 | **0** | **56** |

**353 dead-states bipèdes contre 1.** Le portage de la marche de `killsource` (timeline
chronologique + localisateur d'events + 8 vues + snapshot/restore) est donc validé sur pièces,
et le gate G1b passe largement.

Gain annexe, mesuré : le filtre de bande dérivé des images-clés aurait **perdu 2 à 5 dead-states
bipèdes par film** (le film lie aussi des entités par records NEW en cours de flux). La marche les
garde parce qu'elle range par ARCHÉTYPE (`FrameRecord.TypeIndex`), jamais par bande.

## 3. Verrou 2 — LE FILTRE `DesyncAt == -1` JETAIT LES MORTS DE VÉHICULE (neuf)

Avec le seul verrou 1 levé, `ti=40` rendait **0 dead-state**. Le contrôle décisif — compter les
records dont le **MASQUE DÉCLARE** le composant, que la traversée ait rompu ou non — a montré que
l'absence était un artefact du filtre :

| archétype | records déclarant le dead-state | dont dans un record désynchronisé |
|---|---:|---:|
| `ti=35` biped (64 comp.) | 234 | 53 (23 %) |
| `ti=36` item (20 comp.) | 97 | **0** |
| `ti=37` equipment (31 comp.) | 335 | **0** |
| `ti=38` generic (20 comp.) | 166 | **0** |
| `ti=41` projectile (22 comp.) | 18 | **0** |
| `ti=42` item+weapon (21 comp.) | 191 | **0** |
| **`ti=40` VÉHICULE (48 comp.)** | **69** | **65 (94 %)** |
| `ti=43` device (41 comp.) | 58 | 57 (98 %) |

*(film `0d76e8f1` ; même profil sur `b232e02d` et `51d3ab9f`.)*

Le véhicule déclarait donc bien 69 dead-states — on n'en lisait aucun. **Où rompt la traversée ?**

	rupture a DesyncAt=30  dans 23 records · vehicle-auto-turret-triggers-component
	rupture a DesyncAt=32  dans 15 records · vehicle-transformed-or-desired-open-state-changed
	rupture a DesyncAt=31  dans 14 records · vehicle-auto-turret-aiming-vector-component
	rupture a DesyncAt=33  dans  3 records · vehicle-type-state-component
	rupture a DesyncAt=35  dans  3 records · vehicle-auto-turret-target-component

**Toutes les ruptures sont à `i30`..`i36` — APRÈS le dead-state, qui est à `i11`.** Or `DesyncAt`
est l'index du PREMIER composant présent non porté : tout ce qui précède a été consommé dans
l'ordre. Les bits du dead-state avaient donc été lus au bon endroit ; seule la QUEUE du record
était inconnue. Le filtre `DesyncAt == -1`, hérité de `killsource` (où le risque de rupture est en
AMONT, chez le bipède), jetait des morts de véhicule parfaitement lues.

**Correctif de l'instrument** : accepter un dead-state quand `DesyncAt > index(dead-state)`, et
compter cette classe À PART (`tailDesync`) pour ne jamais mélanger les deux qualités.

Même pathologie sur `ti=43` (device) : rupture à `i19`..`i23` (`device-position-animation-name`,
`device-power`). Les deux archétypes « lourds » non-bipèdes sont concernés ; les légers (20-31
composants) ne le sont pas.

## 4. Après correctif — la mesure

| film | dead-states `ti=35` | dead-states `ti=40` | dont dans la bande véhicule |
|---|---:|---:|---|
| `0d76e8f1` (Behemoth) | 84 | **21** | majorité (slots 768, 782, 811…) |
| `b232e02d` (Behemoth) | 81 | **27** | majorité (slots 768, 769…) |

Échantillons, avec le champ tueur :

	t=2146769095 us · slot 768 · EnumA= 8 EnumB= 7 cat=0 hasRef=true gid=0x79119a25
	t= 864355092 us · slot 768 · EnumA=12 EnumB=18 cat=0 hasRef=false
	t= 870361597 us · slot 769 · EnumA= 7 EnumB=17 cat=8 hasRef=false

## 5. Réserves — ce qui n'est PAS prouvé

1. **Les valeurs ne sont pas validées, seulement les positions.** Tout ce qui précède `i11` a été
   consommé dans l'ordre, mais « porté » n'est pas « juste ». La confrontation à la vérité
   terrain reste à faire (§6).
2. **Sur-comptage certain.** 21-27 par film dépasse le nombre de destructions d'un match : un
   véhicule mort re-réplique son dead-state sur plusieurs ticks (le bipède a le même profil :
   84 dead-states pour ~60 morts réelles). Il faut grouper les dead-states consécutifs d'un même
   slot en UN épisode de mort. Non fait.
3. **Quelques entrées hors bande** (slots 116, 1449, 5507) : slots liés à `ti=40` en cours de flux,
   candidats artefacts. À écarter par la bande ou par recoupement.
4. **`ti=43` (device) n'a pas été instruit** — même pathologie, hors périmètre de ce lot.

## 6. Vérité terrain disponible, non encore utilisée

1. **Les 7 candidates Theater du handoff §3.1** portent des minutes en temps de match : les films
   `0d76e8f1` (Wasp 6:51) et `b232e02d` (Banshee 6:09) sont dans l'échantillon mesuré ici. Le
   recoupement date-à-date est LA validation, et il ne demande plus de visionnage si les
   horodatages tombent juste.
2. **`VehicleDestroys` de l'API** est parsé (`internal/openspartan/halo_api_payload.go:134`) mais
   ne semble persisté nulle part. C'est un compte PAR JOUEUR et PAR MATCH : il borne les faux
   positifs après groupement en épisodes (§5.2). Il ne date rien.

## 7. Élément de rétro-ingénierie apporté au passage (Ghidra, 2026-09-05)

Le pipeline de mort du moteur, relu par l'autre bout, corrobore le lot V9 :

- `FUN_1404d9828` applique le dégât `jpt!` sur **n'importe quel objet** et, sur le coup fatal,
  appelle `FUN_140adefbc` → `FUN_142c4e850`, **le classifieur de mort**. Celui-ci choisit
  `enemy_vehicle_kill` quand la victime n'a pas d'index joueur (`+0x484 == -1`) et que c'est un
  véhicule ; sinon `enemy_kill`, trahison, suicide. `FUN_142c4dcf8` calcule l'assistance
  (`vehicle_destroy_assist`) en parcourant la liste des contributeurs de dégât de l'objet.
  **Une destruction de véhicule passe par le MÊME code de mort qu'un kill de joueur.**
- Il n'existe **aucun** composant ECS « véhicule détruit » : le binaire ne déclare que des
  composants d'OBJET (`object-body-vitality`, `object-shield-vitality`, `object-damage-sections`,
  `object-dead-state`). Le dead-state est le seul signal d'état gravé à la mort, et il est
  générique — ce que la mesure confirme.
- Les événements NOMMÉS du moteur (`vehicle_death`, `enemy_vehicle_kill`, `vehicle_destroy_assist`,
  `vehicle_enter/exit/board/flip`) sont enregistrés par `FUN_140748a74` = **murmur3 x86_32 seed 0**
  après normalisation (minuscules, `-`/espace → `_`), le même hachage qui a craqué les noms `vehi`
  de la palette Forge. Ids calculés : `vehicle_death` = `0x5c1d8575`, `enemy_vehicle_kill` =
  `0xd12acf08`, `vehicle_destroy_assist` = `0x4ef6ef77`. **Piste distincte des 28 TYPES d'événements
  réfutés par V7** : elle porte sur le VOCABULAIRE des propriétés nommées (`PlayerGameEventSmall`
  0xE9 porte un `nom R(32)`). Notée, non traitée — le dead-state suffit désormais.

## 8. Suite

1. **Grouper en épisodes de mort** (dead-states consécutifs d'un même slot) et écarter le hors-bande.
2. **Recouper avec les 2 candidates Theater** présentes dans l'échantillon (`0d76e8f1`, `b232e02d`)
   — conversion horloge film → temps de match, le piège du décalage ~1 870 s est documenté au
   handoff §3.1.
3. Si le recoupement tient : le Go peut publier `end = "destroyed"` + `tEnd`, et l'effet
   d'explosion + les sons déjà câblés (commits `ad811de64`, `f4c3ed417`) s'allument sans
   re-livraison.
4. Hors périmètre, noté : `ti=43` (device) souffre de la même rupture de queue ; et les composants
   `vehicle-auto-turret-*` / `vehicle-transformed-or-desired-open-state-changed` ne sont pas
   modélisés — c'est la cible naturelle d'un lot de grammaire.
