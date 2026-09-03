# Sonde de fiabilite — precision par arme et par distance depuis le film (2026-08-31)

Instrument : `apps/go-api/internal/analysis/filmdec/lot1_sonde_precision_research_test.go`
(`TestLot1SondePrecisionDistance`, garde `LOT1_TRAME_FILM`, un film par process, borne a
`deltaWitnessChunks` = 12 chunks). Films : `000d5950` (Cliffhanger), `01e1f945` (Catalyst,
carte forcee `LOT1_SONDE_MAP=catalyst`), `00502e52` (Bazaar). Composition PURE des decodeurs
existants — aucune reimplementation : `ScanFilmFireEvents`/`decodeFireEvent`, grammaire de
reference du type 36 (`variant_name`), `lot1DecodeDamageAftermath` + refs d'en-tete
(`lot1RefDom1`), `ScanFilmBipedPositions`.

## But

Feature visee : « precision par arme et par distance » calculee depuis le film. L'API du match
donne la precision GLOBALE (sans distinction) ; le film doit fournir la VENTILATION (par arme,
par distance). Le film SOUS-REPLIQUE les degats : on ne l'utilisera que pour la FORME
(distributions relatives), recalee sur le total API. QUESTION CENTRALE : cette forme est-elle
REPRESENTATIVE, c.-a-d. la sous-replication est-elle NON biaisee par arme / par distance ?

## Les 7 mesures, par film

Base de resolution des slots calibree sur le PIC du balayage de resolvabilite = **512** sur les
trois films (l'argmax structurel lands-on-biped est instable : il a choisi 510 sur `01e1f945`,
d'ou 13,5 % avant recalage, 73 % apres). Le pic est franchement unimodal (cf. sweep ci-dessous),
donc c'est un vrai parametre de decalage de bande, pas un surajustement.

### M1 — Tirs par arme (type 36, `variant_name` R(32))

| Film | records longs | armes distinctes | part | index joueur (FilmIndex) | plausibilite |
|---|---|---|---|---|---|
| 000d5950 | 245 | 27 | 11,0 % | 8 | TENU |
| 01e1f945 | 867 | 11 | 1,3 % | 8 | TENU |
| 00502e52 | 276 | 30 | 10,9 % | 8 | TENU |

L'arme (`variant_name`) est franchement CATEGORIELLE (11-30 distinctes << nb d'evenements).
Cross-check `ScanFilmFireEvents` (WeaponID 64 bits, offsets FIXES) : encore moins de distinctes
(17 / 12 / 19) sur PLUS d'evenements (519 / 2154 / 847, il capte aussi les records courts) —
egalement categoriel, donc les deux cles de tir sont exploitables. Le nombre d'index JOUEUR
(FilmIndex) vaut 8 partout : plausible pour de l'arene 8 joueurs. (Le « handle d'entite tireur »
de la grammaire de reference, 27-41 distincts, est un handle biped x respawns, PAS un index de
joueur — ne pas le confondre avec la plausibilite « 8 joueurs ».)

### M2 — Degats par arme (`damage_aftermath`, source R(32) = tag global)

| Film | evenements | source presente | tags source distincts |
|---|---|---|---|
| 000d5950 | 190 | 190 (100 %) | 11 |
| 01e1f945 | 44 | 44 (100 %) | 8 |
| 00502e52 | 150 | 150 (100 %) | 10 |

La source est presente sur 100 % des degats. Le tag `399949091` domine Cliffhanger (x95) ET
Bazaar (x103) — source de degat commune et stable (arme principale probable) ; `2200066965`,
`4004011046`, `54656` recurrent aussi. A cote, des tags a PETITE valeur entiere (95, 54656,
45624, 10189, 17619, 0xd580...) forment un espace d'id distinct (probablement degat
non-arme / environnemental ou un autre encodage) — a caracteriser avant de les traiter comme des
armes.

### M3 — JOIN des cles d'arme (tir vs degat)

| Film | tags source | inter. `variant_name` | inter. WeaponID bas | inter. WeaponID haut |
|---|---|---|---|---|
| 000d5950 | 11 | 0 (0,0 %) | 0 | 0 |
| 01e1f945 | 8 | 0 (0,0 %) | 0 | 0 |
| 00502e52 | 10 | 0 (0,0 %) | 0 | 0 |

**Intersection NULLE sur les trois films, sur toutes les cles.** Le tag source de degat (32 bits)
et l'id d'arme du tir (`variant_name` / WeaponID) vivent dans des ESPACES D'ID DISJOINTS. Cela
confirme la mesure existante (`components_object.go` : 0/786 des sources resolvaient vers une
famille d'arme). Consequence : **le join tir<->degat par arme n'est PAS direct — il faut une
TABLE `tag de source de degat -> arme`** (le projectile/effet qui applique le degat n'a pas le
tag de l'arme qui l'a tire).

### M4 — Resolvabilite (attaquant ref1 ET victime ref0 -> positions, tolerance 120 ms)

| Film | degats a 2 refs | resolus | taux | slots suivis | echantillons |
|---|---|---|---|---|---|
| 000d5950 | 128 | 123 | **96,1 %** | 47 | 77 858 |
| 01e1f945 | 37 | 27 | **73,0 %** | 49 | 70 508 |
| 00502e52 | 150 | 146 | **97,3 %** | 42 | 73 228 |

Sweep de resolvabilite par base candidate (pic unimodal a 512) :

```
000d5950 : 508=14 510=10 512=123 514=28 516=27
01e1f945 : 508=2  510=5  512=27  514=11 516=2
00502e52 : 508=19 510=25 512=146 514=32 516=10
```

Quand les deux refs d'en-tete sont presentes, les positions sont retrouvees a ~96-97 % sur les
films DENSES. `01e1f945` est plus bas (73 %) ET tres pauvre en degats (44 evts) : la capture des
degats est FORTEMENT film-dependante.

### M5 — Distance par arme (|pos attaquant - pos victime| au degat, metres)

| Film (carte) | mediane globale | histogramme (buckets m) |
|---|---|---|
| 000d5950 (Cliffhanger) | 5,3 | 0-2=2 · 2-5=53 · 5-10=42 · 10-15=22 · 15-25=3 · 25-40=1 |
| 01e1f945 (Catalyst) | 3,7 | 0-2=11 · 2-5=4 · 5-10=3 · 10-15=6 · 15-25=3 |
| 00502e52 (Bazaar) | 5,8 | 0-2=8 · 2-5=64 · 5-10=42 · 10-15=25 · 15-25=7 |

La forme est SENSEE : concentration en combat rapproche (2-10 m), queue vers 15-40 m. Surtout,
l'ORDRE des medianes PAR SOURCE est physiquement coherent (Bazaar) :

| Bazaar — tag source | evts | mediane |
|---|---|---|
| 0x0017d6bd23 (399949091) | 103 | 4,4 m (arme courte portee principale) |
| 0x000000d580 (54656) | 11 | 8,2 m |
| 0x0083225b95 | 8 | 16,3 m (portee plus longue) |

Une source « longue portee » mesure long, une « courte » mesure court -> la distance mesuree est
reelle, pas du bruit.

### M6 — LE TEST CENTRAL, biais de capture

| Film | tag source frequent | total | avecRefs | resolus | capture (resolus/total) |
|---|---|---|---|---|---|
| 000d5950 | 0x0017d6bd23 (399949091) | 95 | 95 | 95 | 100 % |
| 000d5950 | **0x00d9dbf765 (3655071589)** | **62** | **0** | **0** | **0 %** |
| 000d5950 | 0x00eea85c26 | 9 | 9 | 9 | 100 % |
| 01e1f945 | 0x000000005f (95) | 13 | 6 | 0 | 0 % |
| 01e1f945 | 0x00bb1a095b | 9 | 9 | 9 | 100 % |
| 00502e52 | 0x0017d6bd23 (399949091) | 103 | 103 | 103 | 100 % |
| 00502e52 | 0x000000d580 | 11 | 11 | 11 | 100 % |
| 00502e52 | 0x0083225b95 | 8 | 8 | 8 | 100 % |

- **Cote POSITIONS : aucun biais.** Des qu'un degat porte ses deux refs, la position est
  retrouvee ~100 % du temps (colonne `pos %` de l'instrument), quelle que soit l'arme ou la
  distance. La sous-replication n'est PAS biaisee par la densite d'echantillonnage des positions.
- **Cote REFS : biais MARQUE et dependant de la source.** Sur Cliffhanger, la source
  `3655071589` — la 2e plus frequente (62 degats) — ne porte AUCUN handle d'en-tete
  (`avecRefs=0`) : cette arme est INTEGRALEMENT invisible cote geometrie. Meme motif sur Catalyst
  (`95`, 13 degats, 6 refs, 0 resolu). Ecart de capture entre tags frequents = **100 points** sur
  Cliffhanger et Catalyst (uniforme, 0 point, seulement sur Bazaar).

**VERDICT de biais : NON UNIFORME.** Le biais n'est pas au niveau des positions ni de la distance
d'echantillonnage — il est au niveau des REFS d'en-tete : certaines sources de degat n'emettent
pas les references attaquant/victime, et disparaissent alors entierement de la forme distance. La
forme par arme est representative UNIQUEMENT pour le sous-ensemble des sources porteuses de refs.

### M7 — Proxy brut degats/tirs (ordre de grandeur, PAS la precision)

| Film | degats | tirs (longs) | degats/tir |
|---|---|---|---|
| 000d5950 | 190 | 245 | 0,78 |
| 01e1f945 | 44 | 867 | 0,05 |
| 00502e52 | 150 | 276 | 0,54 |

Le ratio varie d'un facteur ~15 entre films (0,05 a 0,78) : la sous-replication des degats est
massivement film-dependante. Ce n'est pas la precision (1 tir -> plusieurs composantes de degat,
et reciproquement) et la ventilation par arme exige le join M3. A RECALER sur le total API.

## Verdict honnete

**(a) « distance des touches par arme » (cote degat seul) — FAISABLE mais BIAISEE.**
La geometrie est propre : resolvabilite 96-97 % sur films denses, positions non biaisees, medianes
par source physiquement ordonnees (courte/longue portee). MAIS certaines sources de degat
frequentes ne portent pas les refs d'en-tete (ex. `3655071589`, 62 degats sur Cliffhanger, 0 %
capture) : ces armes sont ABSENTES de la forme. Avant de livrer, il faut identifier QUELLES
sources sont depourvues de refs et les exclure/annoter — sinon la distribution par arme oublie
silencieusement des armes entieres.

**(b) « precision par arme » (avec ancrage API) — PAS ENCORE, cle de join manquante.**
Le join tir<->degat par arme est impossible directement : espaces d'id disjoints (0 % sur 3 films).
Il faut d'abord construire une TABLE `tag de source de degat -> arme` (le tag de degat est le
projectile/effet, pas l'arme). Meme cette table posee, le biais (a) demeure : les armes dont la
source ne porte pas de refs resteront sans geometrie. De plus l'echantillon film-seul est petit et
tres variable (44 a 190 degats sur 12 chunks) : toute distribution DOIT etre recalee sur le total
API, jamais utilisee en absolu.

**Cle d'arme qui joint : NON.** `variant_name`/WeaponID (tir) et tag source (degat) ne partagent
aucun id. Requiert une table de correspondance a batir.

## Reprise / suites possibles (hors perimetre de cette sonde)

- Batir la table `tag source de degat -> arme` (probablement via un balayage corpus croisant, par
  meme paire attaquant/victime et proximite temporelle, un tir type 36 et le degat qui suit).
- Caracteriser les sources sans refs (`3655071589`, `95`, ...) : arme, degat de zone, ou autre ?
- Confirmer la carte de `01e1f945` (Catalyst retenu : FilmIndex=8 => arene, pas BTB Deadlock).
