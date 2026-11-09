# RAPPORT R1 — L'activation du translocateur dans le film : entité, recensement, événement

Date : 2026-09-03. Lot R1 du plan `PLAN_LECTURE_FIABLE_EQUIPEMENT_2026-09-03.md`.
Recherche pure, lecture seule : aucun fichier de production touché. Instruments :

- `apps/go-api/internal/analysis/filmdec/faille_activation_research_test.go` (canaux d'entités)
- `apps/go-api/internal/analysis/filmdec/faille_activation_events_research_test.go` (canal des événements)

Gardés par `FAILLE_FILM` (+ `FAILLE_BOUNDS`, `FAILLE_ANCRES`), skip par défaut, `CGO_ENABLED=0`,
`go vet ./internal/analysis/filmdec/` vert. Toutes les commandes rejouables sont en annexe.

## Verdict en deux phrases

**La POSE de la faille ne crée aucune entité lisible et n'émet aucun événement de tête : négatif
mesuré sur les trois canaux d'entités (créations NEW, deltas d'objet du monde, recensement
d'images-clés).** **La TÉLÉPORTATION, elle, est un ÉVÉNEMENT NOMMÉ du film : type 117
`EquipmentTranslocatorTeleportEffects`, un par téléportation exécutée, portant la référence du
bipède qui saute, émis 5 à 80 ms avant la discontinuité de position — précision 18/18 sur les
5 films, y compris une téléportation de 3,24 m invisible au seuil spatial de 4 m.**

## 0. Piège d'horloge (à retenir pour tout instrument futur)

Les paquets du film sont datés sur l'horloge MOTEUR (ex. 5 539 s à 6 101 s sur `1b2d9e08`), pas
sur l'horloge du film. Conversion : `US_moteur = US_film + origine`, où l'origine est
l'horodatage du PREMIER paquet du chunk 1 (même lecture que `replay.ScanFilmClockOrigin`).
L'instrument fait ce décalage lui-même ; `FAILLE_ANCRES` se donne en horloge FILM
(`(originMs + frame × frameIntervalMs) × 1000`). Une première exécution sans ce décalage a rendu
0 paquet fenêtré — le bug est documenté ici pour ne pas être refait.

## 1. Canal des créations (records NEW, tous archétypes lisibles) — NÉGATIF

Balayage `equipCreationWalk` (marche de production, archétype en paramètre) sur les paquets
delta des fenêtres A1 [146 862, 185 262] ms et A2 [328 162, 351 062] ms (+/- 2 s) du film
`1b2d9e08`. Archétypes couverts : ti 36, 37, 38, 39, 42, 43 (default-state porté ET i0 =
`object-position-component`). ti 41 (projectile) exclu des créations (default-state non résolu),
couvert par le canal 2. Critère : position i0 à <= 3 m (2D) d'une ancre.

| ti | paquets fenêtrés | ancres NEW | acceptées | à <= 3 m | rejets (overflow/mask/pos) |
|---|---|---|---|---|---|
| 36 | — | — | — | — | bande vide (aucun slot ti=36 aux images-clés) |
| 37 | 4 154 | 738 | 43 | **0** | 12 / 636 / 47 |
| 38 | 4 154 | 479 | 3 | **0** | 52 / 410 / 14 |
| 39 | — | — | — | — | bande vide |
| 42 | 4 154 | 836 | 43 | **0** | 46 / 728 / 19 |
| 43 | 4 154 | 9 | 1 | **0** | 0 / 7 / 1 |

Les 90 créations acceptées sont toutes à 10-40 m des ancres : ce sont les respawns des socles
d'équipement/d'armes de la carte (positions groupées vers (4-5, 141-145)). On y voit d'ailleurs
la respawn du translocateur RAMASSABLE : ti=37 slot 1789 @151 751 ms, MPP32=0x730dc70f, sur son
socle à (5.25, 143.00) — 14,2 m de l'ancre A1. **Aucune création d'entité au lieu de la faille,
dans aucune des deux fenêtres.**

## 2. Canal des deltas d'objet du monde (bande union ti 36-43 dont 41) — NÉGATIF

Même fenêtres, `scanProjectileRecords` sur la bande union (1 469 slots) : 9 644 échantillons de
position décodés, 13 vies avec au moins un point à <= 3 m d'une ancre — TOUTES pour A1, AUCUNE
pour A2.

Lecture des 13 vies d'A1 : 10 tombent dans la rafale [151 266, 152 101] ms — l'instant exact où
le POSEUR (JGtm, slot 535) traverse (17.2-18.6, 136.0-135.3) à pied (vérifié sur l'artefact,
frames 1424-1432). Les 3 autres sont des points isolés (155,5 s, 160,2 s, 181,1 s, 1 point
chacun). **Entre 152,1 s et 185,2 s — les 33 s où la faille existe et où le poseur est
ailleurs — AUCUNE entité ne vit près de l'ancre.** Pour A2, rien du tout, ni près de l'arrivée
(18.34, 120.19) ni près de l'autre bout du va-et-vient (17.35, 123.20). La rafale d'A1 date donc
très probablement la POSE (~151,5 s) mais ne prouve aucune entité de faille : elle coïncide avec
la présence physique du joueur, et rien ne persiste après son départ.

## 3. Canal du recensement d'images-clés (TOUS les ti 0..49) — NÉGATIF

Marche unique des images-clés (28 images-clés, 938 vies recensées sur le film). Vies recensées
pour la PREMIÈRE fois dans [t0, t1 + 25 s] : 69 pour A1, 54 pour A2. Toutes s'expliquent par le
churn des socles (ti 37/42 par paires), leurs entités de gestion (ti 10/12) et les respawns de
bipèdes (ti 35). Aucune vie au profil « née à la pose, disparue à l'usage » n'est corroborée par
une position proche de l'ancre (canaux 1-2). Le recensement borne mais ne situe pas : ce canal
ne peut PAS à lui seul infirmer une entité muette — c'est la conjonction avec les canaux 1-2
(aucune position, aucune création) qui ferme la voie.

**Conclusion entités : la faille activée n'est pas une entité répliquée lisible du film.** Ni
sous 0x730dc70f, ni sous un autre GlobalID, ni sous un autre archétype à position.

## 4. Canal des ÉVÉNEMENTS — POSITIF : type 117 `EquipmentTranslocatorTeleportEffects`

### 4.1 La découverte

La liste d'événements en tête des paquets delta (modèle M, `NOTE_MODELE_EVENEMENTS_2026-08-30`)
recensée sur le film entier : sur `1b2d9e08`, 33 571 paquets delta, 16 types en tête de liste,
et **le type 117 n'a que 3 occurrences — toutes trois dans les fenêtres des ancres** :

| occurrence | instant film | rapport à la vérité terrain |
|---|---|---|
| @185 184 ms | **77 ms avant** le saut de 22,1 m de JGtm (A1, frame 1762) |
| @335 257 ms | **5 ms avant** un saut de **3,24 m** du slot 560 (frame 3261→3262) — téléportation RÉELLE jamais détectée par le seuil de 4 m, vérifiée sur l'artefact : (18.41,120.14) → (17.35,123.20) |
| @350 989 ms | **72 ms avant** le saut de 7,8 m de SpiffyDart (A2, frame 3420) |

L'annexe A de `GRAMMAIRE_EVENTS_FILM_2026-08-30.md` nomme le type 117
`EquipmentTranslocatorTeleportEffects` (lecteur exe `0x140f04fb8`, réception 28 octets). Le nom
vient de l'exe, pas d'une interprétation.

### 4.2 La forme du record (mesurée sur pièces)

- Premier octet du paquet : `0xFA` = bit config (1) + bit de continuation (1) + les 6 bits hauts
  du type ; le bit de poids faible du type est le 1er bit du 2e octet (ici 1 → type 117).
- **ref0 = l'unité qui se téléporte** : porte 1 bit, index **8 bits**, génération 2 bits —
  domaine 2 (base 512). Fermeture : sur les 3 événements de `1b2d9e08`, l'hypothèse w=8 rend
  exactement les slots 535, 560, 560 des ancres ; les hypothèses w=9 et w=13 rendent des slots
  sans rapport. Sur les 5 films, les 18 événements rendent tous un slot bipède plausible
  (512-640), gen=1.
- refs 1 et 2 : absentes (portes à 0) sur les 18 observations.
- Charge : un mot 32 bits constant `0x42689F84` (aligné octet, présent sur les 18 événements des
  5 films — un identifiant d'effet, à résoudre côté exe), puis ~16 octets variables par
  événement. Une sonde 3 x 16 bits (quantification type `unit_teleported`) n'y a PAS localisé de
  position au-dessus du hasard (2 coups sur ~400 offsets testés, ~7 attendus par hasard au seuil
  3 m) : la position de la faille n'est pas établie dans la charge — layout à sourcer de l'exe
  (`FUN_140f04fb8`) si on la veut.

### 4.3 Rappel sur les 4 films témoins (R1.3)

Recensement film entier des têtes de type 117 (dénominateur : paquets delta), ref0 décodée :

| film | paquets delta | têtes 117 | correspondances |
|---|---|---|---|
| `a0c36016` | 46 534 | 4 | 520@72,8s (spent @89,4s) · 534@183,8s (**-79 ms** du saut 16,25 m) · 555@302,7s (saut 25,4 m VÉRIFIÉ artefact) · 623@659,2s (**-9 ms** du saut 20,9 m) |
| `4577fcc4` | 34 246 | 6 | 514 x3 @77,6/80,9/91,3s (sauts 9,0/8,4/4,5 m vérifiés ; spent @103,9s) · 591 x3 @517,7/523,7/528,4s (= les sauts 8,3/8,5/14,7 m @5113/5172/5220) |
| `f2966f08` | 37 598 | 2 | 555@304,8s (saut 12,2 m vérifié) · 610@614,6s (saut 7,5 m vérifié) — AUCUN pour le spent 523@1014 |
| `faff9935` | 33 042 | 3 | 533@155,1s (saut 18,9 m vérifié ; spent @170,4s) · 551 x2 @245,3/252,6s (sauts 9,8/17,9 m vérifiés ; spent @253,8s) |

Contre-épreuve systématique : pour CHACUN des 18 événements des 5 films, l'artefact du rejeu
montre une discontinuité de position du slot désigné à la frame suivante (3,24 à 25,36 m).
**Précision 18/18. Rappel sur les téléportations mesurées : 8/8** (les 2 sauts de `1b2d9e08`,
2 de `a0c36016`, 3 de `4577fcc4` slot 591, et le 3,24 m découvert par l'événement lui-même
et confirmé après coup).

**Trois `spent` n'ont AUCUN événement 117 de leur slot** (`a0c36016` 581@4484, `4577fcc4`
579@4260, `f2966f08` 523@1014). Deux causes possibles, non départagées ici : (a) épuisement sans
téléportation (l'équipement expire sans avoir été réactivé) ; (b) l'événement existe mais pas en
TÊTE de liste — le scanner ne lit que le premier événement de chaque paquet (limite identique à
`zoom_events.go`, décodage de liste complète = chantier `PLAN_PERCER_TRAME_FILM`). Aucun de ces
trois spent n'a de téléportation CONNUE : ce ne sont pas des rappels manqués mesurés, ce sont
des absences non expliquées.

### 4.4 Ce que l'événement révèle en plus

- La sémantique du translocateur est un VA-ET-VIENT : sur `1b2d9e08` slot 560, le saut de
  3,24 m arrive à (17.35,123.20) puis le saut suivant arrive à (18.34,120.19) — soit le point de
  DÉPART du précédent (0,09 m près). La balise échange sa position avec le joueur à chaque
  usage. Toute lecture « la faille est un point fixe posé une fois » serait fausse.
- Les événements 117 existent pour des slots absents de toute liste d'ancres (555 dans deux
  films, 610) : des usages entiers du translocateur étaient invisibles à la méthode « spent +
  saut > 4 m ».

## 5. Statut des items du plan

- R1.1 (entité à la position de l'ancre) : **fait — négatif mesuré** (sections 1-3, dénominateurs
  inclus ; réserve : ti 36/39 sans bande sur ce film, ti 41 sans lecteur de création).
- R1.2 (événement aux instants des ancres) : **fait — positif** (section 4). Gestes sonores non
  croisés : le canal Wwise date des gestes AUDIO, l'événement 117 date déjà le geste exactement —
  le croisement n'apporterait rien à la question posée.
- R1.3 (rappel sur A3-A6) : **fait** (section 4.3) — précision 18/18, rappel 8/8 sur les
  téléportations mesurées, 7/10 spent couverts (3 absences non expliquées, deux causes candidates
  documentées).

## 6. Suites possibles (hors périmètre R1, à arbitrer)

1. Décodage de la LISTE COMPLÈTE d'événements (pas seulement la tête) : lèverait la seule
   réserve du rappel, et vaut pour tous les types (zoom compris).
2. Charge utile de `FUN_140f04fb8` côté exe : positions de départ/arrivée probablement dedans
   (28 octets de réception, ~16 octets variables observés).
3. Production : un lecteur `ScanFilmTranslocatorTeleports` sur le modèle de `zoom_events.go`
   (même grammaire, ~60 lignes) daterait chaque téléportation sans AUCUNE heuristique spatiale —
   c'est la brique que R3/D4 attendaient.

## Annexe — commandes exactes (depuis `apps/go-api`, chemins à adapter au dépôt)

Film Dynasty `1b2d9e08` (entités puis événements) :

```
CGO_ENABLED=0 FAILLE_FILM=<repo>/data/cache/film_chunks/1b2d9e08 \
  FAILLE_BOUNDS=-11.45,104.54,73.91,19.72,153.51,82.53 \
  FAILLE_ANCRES="A1:535:17.34,135.50:146862000-185262000:185262000;A2:560:18.34,120.19:328162000-351062000:351062000" \
  go test ./internal/analysis/filmdec/ -run '^TestFailleActivationEntites$' -timeout 40m -v

CGO_ENABLED=0 FAILLE_FILM=<repo>/data/cache/film_chunks/1b2d9e08 \
  FAILLE_BOUNDS=-11.45,104.54,73.91,19.72,153.51,82.53 \
  FAILLE_ANCRES="A1:535:17.34,135.50:146862000-185262000:185262000;A2:560:18.34,120.19:328162000-351062000:351062000" \
  go test ./internal/analysis/filmdec/ -run '^TestFailleActivationEvenements$' -timeout 20m -v
```

Témoins (test événements seul ; fenêtres = [spent-60s, saut/spent+2s], horloge FILM) :

```
CGO_ENABLED=0 FAILLE_FILM=<repo>/data/cache/film_chunks/a0c36016 \
  FAILLE_BOUNDS=-34.8,-24.7,-0.67,3.64,27.89,7.43 \
  FAILLE_ANCRES="T520:520:-16.37,6.93:29357000-91357000:89357000;T534:534:-15.60,1.38:123857000-185857000:183857000;T581:581:-26.00,10.00:392157000-454157000:452157000;T623:623:-18.91,-0.54:599257000-677657000:659257000" \
  go test ./internal/analysis/filmdec/ -run '^TestFailleActivationEvenements$' -timeout 20m -v

CGO_ENABLED=0 FAILLE_FILM=<repo>/data/cache/film_chunks/4577fcc4 \
  FAILLE_BOUNDS=-35.55,-200.53,-481.65,217.3,30.71,108.4 \
  FAILLE_ANCRES="T514:514:-23.49,26.47:43946000-105946000:103946000;T579:579:-21.66,2.50:372446000-434446000:432446000;T591:591:-13.96,3.11:468446000-544646000:528446000" \
  go test ./internal/analysis/filmdec/ -run '^TestFailleActivationEvenements$' -timeout 20m -v

CGO_ENABLED=0 FAILLE_FILM=<repo>/data/cache/film_chunks/f2966f08 \
  FAILLE_BOUNDS=-149.53,17.58,-11.17,-99.27,87.81,15.91 \
  FAILLE_ANCRES="T523:523:-124.88,62.78:51925000-113925000:111925000" \
  go test ./internal/analysis/filmdec/ -run '^TestFailleActivationEvenements$' -timeout 20m -v

CGO_ENABLED=0 FAILLE_FILM=<repo>/data/cache/film_chunks/faff9935 \
  FAILLE_BOUNDS=-97.26,-15.03,-453.45,-12.21,203.5,69.71 \
  FAILLE_ANCRES="T533:533:-26.54,10.21:110423000-172423000:170423000;T551:551:-23.56,9.96:193823000-255823000:253823000" \
  go test ./internal/analysis/filmdec/ -run '^TestFailleActivationEvenements$' -timeout 20m -v
```

Contre-épreuve des discontinuités (lecture des artefacts, hors go test) : pour chaque événement
117 (film, slot, frame), lire `.tracks[]` de `data/cache/replays/halo_infinite/<id8>.json` et
mesurer le déplacement 2D entre la frame de l'événement et la suivante — 18/18 entre 3,24 et
25,36 m (script node ad hoc, valeurs reportées section 4.3).
