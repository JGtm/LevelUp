# RAPPORT — lot V2b (vehicules : occupant par attachement, destruction par vitalite, cooldown par socles)

> Execute le 2026-09-01 dans le worktree `LevelUp-wt-vehicules` (branche `wt/vehicules-tourelles`).
> Aucun commit, aucun `git add`, aucune ecriture DuckDB, AUCUNE modification de code de production
> (seuls trois `*_test.go` neufs `vehicules_v2b_*`). Mesures en AVANT-PLAN, `CGO_ENABLED=0`, GOCACHE
> isole (`scratchpad/gocache_v2b`), donnees reelles du checkout principal (`.../LevelUp/data`, RO).
> event_list*.go, himap, cmd/* et les instruments d'autres agents : NON touches (lus / appeles seulement).

## 0. LE POINT CLE EN TROIS REPONSES

1. **i10 (object-parent-state) donne-t-il l'occupant DIRECTEMENT ? NON — refute, et la derniere
   condition de reprise est epuisee.** Sur les records bipedes (ti=35) d'un Behemoth SF, les handles
   i10 attaches resolvent vers un slot vehicule ti=40 dans **0 / 19** cas (et vers TOUT vivant dans
   0 %, SOUS le temoin decorrele a 5,3 %). Le **balayage de `param_4` {0,2,3,4}** — condition de
   reprise #1 du GATE 0, JAMAIS tentee — **ne sauve rien** (0 % vers ti=40 a chaque valeur, part de
   records propres inchangee ~76 %). i10 est clos comme il l'etait pour le drapeau.
   **CE QUI PREND LE RELAIS, et c'est deja porte** : `filmdec.ScanFilmVehicleEvents` (event_list.go).
   La SORTIE (exit) resout l'occupant a la ms (10/10 en-bande sur `0d76e8f1`), et le **recoupement
   V1a.4 est PARFAIT** : **10/10 = 100 %** des sorties ferment un trou du flux de position a l'instant
   exact, **temoin decale 0/10**. Le modele « l'enfant attache ne replique plus » (V1a.4) est confirme,
   et l'occupant se lit par l'event-list, pas par i10.

2. **i4->0 date-t-il la destruction ? NON — la VALEUR d'i4 est du BRUIT pour ti=40.** i4 est bien
   emis et sa grammaire TRAVERSE, mais la valeur capturee ne suit pas : histogramme des quanta
   quasi-UNIFORME et **51 % de pas decroissants (= pile ou face)**, la ou le MEME code sur la bande
   BIPEDE (sante validee en prod) rend un histogramme concentre pres du plein et **13 % de pas
   decroissants**. La desynchronisation vient de la grammaire d'i2/i3 pour ti=40 (i2 deja refute en
   V1a) : **1247 / 1249** records i4 portent i2 ou i3 AVANT i4, donc la contamination est totale.
   Meme sort que i2/i3 : le composant traverse sans capturer sa valeur.

3. **Le cooldown se resserre-t-il par la methode des socles ? OUI, et il se RESOUT hors Super
   Fiesta.** Sur Behemoth **non-SF** (7 films), l'ancrage census donne un cooldown de **~58 s, CV
   inter-pad 0,17 (< 0,20), 4/6 pads etablis** — RESOLU. En **Super Fiesta**, aucun ancrage ne
   resout (CV 0,40-0,72), mais l'ancrage POSITION date (depart du pad, ~0,5 s) **bat** l'ancrage
   census (±20 s) : CV median 0,40 vs 0,72 (Behemoth), 0,57 vs 2,23 (Launch Site), et etablit plus
   de pads. **L'echec du lot V2 item 3 (IQR/med 0,87) etait un artefact de la DENSITE Super Fiesta
   (vehicules concurrents par pad), pas un defaut de methode.**

---

## 1. SIGNAL 1 — OCCUPANT PAR L'ATTACHEMENT i10

Instrument : `apps/go-api/internal/analysis/replay/vehicules_v2b_occupant_test.go`
(`TestV2bOccupantI10`, garde `V2B_OCC_ROOT`). Reutilise `attScanI10` (marche stateful
`DecodeFrameViews` + sonde `SetObjectParentStateHook` + resolution du parent dans le World +
temoin decorrele) et `filmdec.ScanFilmVehicleEvents` (le relais, non modifie).

### 1.1 Seuils, ecrits AVANT mesure

- i10 designe l'occupant SI, parmi les lectures i10 ATTACHEES sur des records ti=35, la part qui
  resout vers ti=40 depasse de **> 10 pts** le temoin (slot decorrele resolvant vers du vivant).
- Balayage `param_4` : une valeur qui re-synchronise i10 doit augmenter conjointement la part de
  records propres ET la resolution vers ti=40.
- Recoupement V1a.4 : l'occupant d'une SORTIE doit coincider avec la FERMETURE d'un trou du flux de
  position de son slot ; temoin = instant de sortie decale de 37 s.

### 1.2 Mesure (`0d76e8f1`, Behemoth SF, 8 joueurs)

| param_4 | lectures i10 ti=35 (attachees) | records propres | parent -> ti=40 | parent vivant (tout ti) | temoin vivant | verdict |
|---|---|---|---|---|---|---|
| defaut (1) | 50 (19) | 76,0 % | **0/19 = 0 %** | 0 % | 5,3 % | REFUTE |
| 0 | 51 (20) | 76,0 % | 0/20 = 0 % | 0 % | 5,0 % | REFUTE |
| 2 | 51 (18) | 75,9 % | 0/18 = 0 % | 5,6 % | 5,6 % | REFUTE |
| 3 | 48 (17) | 76,1 % | 0/17 = 0 % | 5,9 % | 5,9 % | REFUTE |
| 4 | 48 (18) | 76,1 % | 0/18 = 0 % | 5,6 % | 5,6 % | REFUTE |

**Aucune valeur de `param_4` ne change le verdict** : la part de records propres reste ~76 % (le
balayage ne re-synchronise rien), et la resolution vers ti=40 reste NULLE. La condition de reprise #1
du GATE 0 (jamais tentee jusqu'ici) est donc epuisee : i10 ne porte pas le lien bipede->vehicule sur
le chemin delta. C'est coherent avec le negatif du drapeau (2026-08-18) et avec la porte i10 ouverte
a ~1/3 uniformement sur tous les archetypes.

### 1.3 Le RELAIS — event-list board/exit (et le recoupement V1a.4)

`ScanFilmVehicleEvents(0d76e8f1)` : **board = 1, exit = 10, occupant en-bande 10/10 = 100 %.**

**RECOUPEMENT V1a.4 (le fait le plus fort du lot)** : pour chaque sortie, on cherche un trou (>= 3 s)
du flux de position du slot occupant qui se ferme a moins de 2 s de l'instant de sortie.

| | reel | temoin (decale 37 s) |
|---|---|---|
| sorties dont un trou se ferme a l'instant | **10 / 10 = 100 %** | **0 / 10 = 0 %** |

L'occupant que le relais nomme est EXACTEMENT celui dont le trou de position se ferme a la sortie :
100 % contre 0 % au temoin. Cela valide d'un coup (a) le relais event-list comme source de
l'occupant, et (b) le modele V1a.4 « l'enfant attache ne replique plus, puis re-emet en descendant ».
**L'occupant est donc RESOLU — par l'event-list, pas par i10.**

### 1.4 Ce qui reste sur l'occupant

L'EMBARQUEMENT (board) reste partiel (cf. `PORT_LISTE_EVENEMENTS_2026-09-01.md` § 4 : sa ref 0 n'est
pas l'occupant mais probablement le vehicule ; 1 seul board sur `0d76e8f1`). Le temps passe en
vehicule par occupant est deja lisible par les SORTIES datees. L'appariement board->exit attend la
resolution de l'occupant d'embarquement (Ghidra sur le deser board, ou plus d'echantillons).

---

## 2. SIGNAL 2 — DESTRUCTION PAR LA VITALITE i4/i5

Instrument : `apps/go-api/internal/analysis/filmdec/vehicules_v2b_vitalite_test.go`
(`TestV2bVitalite`, garde `V2B_FILMS`). La valeur i4/i5 est capturee par le balayage EXISTANT
(`ScanFilmBipedPositionsForBand` + `CaptureDirs` -> `componentVitals`, `HealthAt()`/`ShieldAt()`) :
aucune capture de production a ajouter, ti=40 portant la meme grammaire dyn.-prec. que le bipede.

### 2.1 Seuils, ecrits AVANT mesure

- i4 « a zero » = `HealthFraction(i4) <= 0`. Classement d'une vie : DETRUIT (>= 1 i4 a zero),
  DESPAWN (i4 sans zero), SANS_I4 (aucun i4).
- GATE datation : le premier i4->0 doit PRECEDER/COINCIDER avec la fin bornee, et etre TERMINAL
  (aucune image-cle ne recense la vie apres lui).
- TEMOIN decisif : la MEME lecture i4 sur la bande BIPEDE (ti=35), dont la sante est VALIDEE en
  production ; monotonicite (une vraie sante ne remonte pas).

### 2.2 Le negatif est decisif — le controle bipede tranche

| grandeur | VEHICULE (ti=40) | BIPEDE (ti=35, controle valide) |
|---|---|---|
| histogramme quanta i4 (buckets de 32) Behemoth | `[475 181 254 199 278 200 360 157]` (UNIFORME) | ex. `[0 0 0 0 48 128 294 562]` (concentre pres du plein) |
| histogramme quanta i4 Launch Site | `[187 169 201 104 213 111 209 102]` (UNIFORME) | idem concentre |
| pas i4 decroissants | **52 % (Beh.) / 50 % (LS) = BRUIT** | **13-26 % = signal structure** |

Le MEME code lit une sante reelle (concentree, monotone) sur le joueur et du bruit uniforme
(pile ou face) sur le vehicule. La valeur i4 du vehicule n'est donc PAS capturee correctement.

### 2.3 La cause : i2/i3 desynchronise le curseur avant i4

Diagnostic de masque (`0d76e8f1`) : parmi les records portant i4, ceux dont le masque ne porte NI
i2 NI i3 avant i4 sont **n = 2** ; ceux qui en portent, **n = 1247**. i1 est valide par V1a
(direction 99 %), mais i2 y est REFUTE et i3 non decode : leur consommation de bits pour ti=40 n'est
pas etablie, et elle contamine la quasi-totalite des lectures i4. Meme famille de faute que i2/i3.

### 2.4 Ce que la mesure dit quand meme

- **i5 (bouclier) = 0 echantillon** sur tout le corpus : les vehicules n'ont pas de composant
  bouclier dans le flux (attendu — vehicules Halo = integrite, pas bouclier).
- **SANS_I4 = 71-76 %** des vies : la plupart des vehicules n'emettent jamais i4 dans leur fenetre
  (pas assez endommages) — i4 ne serait de toute facon pas un signal dense.
- Classement « DETRUIT » (i4->0) : 19-26 % des vies, mais TERMINALITE **11-19 %** seulement (un i4->0
  mi-vie, le vehicule reste recense apres) — confirme que ces zeros ne sont pas la destruction.

### 2.5 Ce qui reste sur la destruction

Dater la destruction par i4->0 exige d'abord d'etablir la grammaire (consommation de bits) d'i2 et
i3 pour ti=40 — une session de retro-ingenierie (Ghidra), hors perimetre. i14/i11 restent sous le
plancher (cadrage V0, lot V2). En attendant, la fin de vie DATEE la plus fine disponible est la trace
de position (~0,5 s, exploitee au signal 3), et l'event-list donne les entrees/sorties.

---

## 3. SIGNAL 3 — COOLDOWN PAR LA METHODE DES SOCLES

Instrument : `apps/go-api/internal/analysis/replay/vehicules_v2b_cooldown_test.go`
(`TestV2bCooldown`, garde `V2B_CD_FILMS`). Reutilise `ScanFilmVehicleCreations` (naissances +
position monde), le recensement, la trace de position, `indexBySlot`/`slotTrack`, et surtout
`gwPadsCycleFromGaps` + `gwPadCycleMaxCV` (la REGLE DE PRODUCTION des socles d'arme, seuil CV 0,20).

### 3.1 Trois ancrages de fin de vie, ecrits AVANT mesure

1. naissance->naissance (horloge d'apparition) ; 2. census goneBy->naissance (±20 s, l'ancrage du
lot V2 item 3 qui a echoue) ; 3. depart du pad->naissance (dernier echantillon de position ENCORE
dans le rayon du pad, ~0,5 s, trace VALIDEE V1a). GATE : l'ancrage 3 doit etablir plus de pads que
l'ancrage 2.

### 3.2 Super Fiesta — la methode resserre mais ne resout pas (densite)

| carte (SF) | ancrage | pads >=2 ecarts | pads etablis (CV<=0,20) | CV median inter-pad |
|---|---|---|---|---|
| Behemoth (3 films, 32 pads) | 1 naissance | 16 | 0 | 2,41 |
| | 2 census | 11 | 1 | 0,72 |
| | 3 depart pad | 11 | **3** | **0,40** |
| Launch Site (3 films, 16 pads) | 1 naissance | 11 | 0 | 3,06 |
| | 2 census | 4 | 0 | 2,23 |
| | 3 depart pad | 5 | **1** | **0,57** |

**Gate tenu** : l'ancrage POSITION date (3) bat l'ancrage census (2) partout (CV 0,40 vs 0,72 ;
0,57 vs 2,23) et etablit plus de pads. Mais CV 0,40 > 0,20 : le cooldown reste NON RESOLU en Super
Fiesta — coherent avec la densite (plusieurs vehicules concurrents par pad, note du lot V2).

### 3.3 Behemoth NON-SF — le cooldown est RESOLU

| carte (7 films non-SF, 49 pads) | ancrage | pads >=2 ecarts | pads etablis | CV median | cooldown median des etablis |
|---|---|---|---|---|---|
| Behemoth non-SF | 1 naissance | 19 | 0 | 2,81 | — |
| | **2 census** | 6 | **4** | **0,17** | **58 s** |
| | 3 depart pad | 12 | 4 | 0,45 | 117 s |

Sur un regime a faible densite (Slayer / Team Slayer / Fiesta : un vehicule a la fois par pad),
l'ancrage census suffit : **CV 0,17 (< 0,20), 4/6 pads etablis, cooldown ~58 s.** La methode des
socles resout donc le cooldown vehicule — l'echec du lot V2 (0,87) venait du CORPUS Super Fiesta,
pas de la methode. (En non-SF l'ancrage census bat meme l'ancrage depart : goneBy y est un bon
marqueur de fin, et le depart inclut la conduite avant disparition, d'ou 117 s, moins propre.)

---

## 4. INSTRUMENTS ECRITS (neufs, aucun code de production modifie)

| fichier | paquet | test | garde |
|---|---|---|---|
| `internal/analysis/filmdec/vehicules_v2b_vitalite_test.go` | filmdec | `TestV2bVitalite` | `V2B_FILMS` (+ `V2B_CONTROL`, `V2B_MASK`, `V2B_DUMP`) |
| `internal/analysis/replay/vehicules_v2b_occupant_test.go` | replay | `TestV2bOccupantI10` | `V2B_OCC_ROOT` (`V2B_OCC_FILM`) |
| `internal/analysis/replay/vehicules_v2b_cooldown_test.go` | replay | `TestV2bCooldown` | `V2B_CD_FILMS` (`V2B_CD_BOUNDS`) |

`go vet` propre sur les deux paquets. Reutilisation stricte (aucune grammaire ni calage recopie) :
`attScanI10`, `ScanFilmVehicleEvents`, `ScanFilmVehicleCreations`, `ScanFilmWorldObjectKeyframes`,
`ScanFilmBipedPositionsForBand`+`CaptureDirs`, `indexBySlot`/`slotTrack`, `gwPadsCycleFromGaps`.

Commandes de rejeu (avant-plan, GOCACHE isole `scratchpad/gocache_v2b`, CGO_ENABLED=0) :

```
# Signal 2 (vitalite) + controle bipede + diagnostic de masque
V2B_FILMS="0d76e8f1:behemoth,...:launch site" V2B_CONTROL=1 V2B_MASK=1 \
  go test ./internal/analysis/filmdec/ -run '^TestV2bVitalite$' -v -timeout 60m

# Signal 1 (i10 + balayage param_4 + relais event-list + recoupement V1a.4)
V2B_OCC_ROOT=<data>/cache V2B_OCC_FILM=0d76e8f1 \
  go test ./internal/analysis/replay/ -run '^TestV2bOccupantI10$' -v -timeout 40m

# Signal 3 (cooldown par socles ; comparer SF vs non-SF)
V2B_CD_FILMS="e232ffce:behemoth,b232e02d:behemoth,..." \
  go test ./internal/analysis/replay/ -run '^TestV2bCooldown$' -v -timeout 60m
```

## 5. STATUT DES SIGNAUX

| signal | statut | resultat |
|---|---|---|
| 1 occupant par i10 | `[x]` mesure, **REFUTE** | 0/19 vers ti=40 ; balayage `param_4` (reprise #1) epuise. RELAIS event-list resout l'occupant (exit 10/10) ; recoupement V1a.4 = 100 % vs 0 %. |
| 2 destruction par i4->0 | `[x]` mesure, **REFUTE (valeur)** | i4 vehicule = bruit (histo uniforme, 51 % down) vs bipede valide (concentre, 13-26 % down) ; i2/i3 contamine 1247/1249. i5 absent (pas de bouclier vehicule). Reprise = RE grammaire i2/i3 ti=40. |
| 3 cooldown par socles | `[x]` mesure, **RESOLU hors SF** | Behemoth non-SF : ~58 s, CV 0,17, 4/6 pads (census). SF : non resolu (densite) mais ancrage position date > census. Methode validee ; echec V2 = artefact de corpus. |

## 6. CONDITIONS DE REPRISE (registre des reports — superviseur)

- **Occupant** : industrialiser le relais `ScanFilmVehicleEvents` (schema `attachments` du plan
  parent-state, decision 5) ; resoudre l'occupant d'EMBARQUEMENT (Ghidra deser board, ou marcher la
  liste entiere). i10 est CLOS (ne pas le rouvrir sans nouvelle grammaire).
- **Destruction datee** : etablir la consommation de bits d'i2 et i3 pour ti=40 (Ghidra) — sans quoi
  i4/i5 restent illisibles apres i2/i3. i14/i11 sous plancher (inchange).
- **Cooldown** : re-mesurer sur un corpus non-SF elargi pour confirmer ~58 s et publier le cooldown
  par carte ; en SF, ne pas promettre un cooldown (densite).
