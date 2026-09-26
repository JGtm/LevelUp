# Sonde P2 — la marche d'image-clé qui perd un préfixe de table (2026-09-23)

Plan : `.ai/V7.5/PLAN_RETOURS_REJEU_2026-09-23.md` §4.3 (P2), prépare M3.1. Annexe de départ :
`RAPPORT_fiche_armes.md` §1.3, §2.1, §4 (S1).
Légende : **MESURÉ** (sortie chiffrée de l'instrument), **DÉDUIT**, **HYPOTHÈSE**.

## Cadre

- Payloads lus (un film à la fois, sous la voie film, lecture seule, aucun artefact) :
  81c02726 morceau 9 (image-clé ts 600 581 ms = t 1494, 1 392 720 bits) ; a0c36016 morceaux 1 et 2
  (1 101 056 et 1 229 264 bits). `chunk_00` et le morceau sont copiés dans un répertoire temporaire
  du test : le contexte de film ne charge rien d'autre.
- Instruments (paquet `grammar`, `//go:build research`, variables `P2_FILM`, `P2_CHUNKS`,
  `P2_DETAIL=81c02726-9` pour les fenêtres propres au cas rapporté) :
  - `p2_marche_imagecle_research_test.go` — A recensement des en-têtes exacts `[(gen<<30)|slot][35]`
    + `n1` (mot à +108) ; B trace du balayeur (copie instrumentée de `walkKeyframeWorldFenetre`) ;
    C croisement ; D marche déterministe (cadre de l'écrivain, 108 bits).
  - `p2_marche_imagecle_trous_research_test.go` — E suites d'entrées sans archétype dans les trous ;
    F prototype « SA » ; G ordre réel (en-têtes crédibles) ; H voisinage ; J fermeture de l'état
    complet ; K en-têtes bruts d'une fenêtre.
  - `p2_marche_imagecle_variantes_research_test.go` — V variantes de la règle d'élection.
- **Étalonnage** : la trace B et la variante PROD rejouent `WalkKeyframeWorld` à l'identique
  (523 / 123 / 352 ancres, égalité bit à bit vérifiée par le test, sinon échec). Témoin positif du
  recensement A : le bipède 536, seule ancre bipède de production à t 1494, est retrouvé par A au
  même bit (206 882).

Commande (depuis `apps/go-api`, sous la voie film) :
`P2_FILM=<data>/cache/film_chunks/81c02726 P2_CHUNKS=9 P2_DETAIL=81c02726-9 go test -tags research -run '^TestP2MarcheImageCle$' -count=1 -v ./internal/games/halo_infinite/film/internal/grammar/`

## Verdict

### 1. Les records sont-ils là ? OUI (MESURÉ)

| payload | en-têtes exacts `[id][35]` | `n1` | positions (bit : slot) | ancrés par la production |
|---|---|---|---|---|
| 81c02726 m9 (t 1494) | 8 | 152 × 8 | 187 719 : 519 · 190 986 : 528 · 193 499 : 530 · 196 323 : 532 · 198 481 : 533 · 201 142 : 534 · 203 803 : 535 · 206 882 : 536 | **1 / 8** (536) |
| a0c36016 m2 | 8 | 152 × 8 | 205 565 : 512 · 208 263 : 513 · 210 961 : 514 · 213 659 : 515 · 216 357 : 516 · 219 055 : 517 · 221 753 : 518 · 224 451 : 519 | **0 / 8** |
| a0c36016 m1 | 0 | — | aucun bipède dans cette image-clé | — |

Les huit en-têtes de chaque image-clé sont à slots croissants, espacés de 2 158 à 3 267 bits (largeur
d'un record bipède), `n1` identique à celui du bipède que la production ancre. Le bipède 532 (non cité
par le rapport) est présent lui aussi : **huit bipèdes, pas sept**.

### 2. Sous-mécanisme de la perte

**Ni départ refusé, ni entrées sans archétype** (MESURÉ) :
- départ ACCEPTÉ au bit 1 dans les trois payloads (`0x40000000`, archétype 22, slot 0) ;
- **0** entrée sans archétype (`[id][0xFFFFFFFF]` chaînée à 108 bits) dans les trous des élections,
  sur les trois payloads ; le prototype F qui les enchaîne rend exactement les ancres de la production
  (523 = 523, 352 = 352). Réserve : négatif par recensement EXHAUSTIF avec le lecteur d'en-tête de
  production (`readKeyframeHeader`), sans témoin positif d'entrée SA dans ces payloads (aucune n'y
  existe).

**Mécanisme A — élection lointaine d'une fausse ancre de slot bas** (81c02726 m9, a0c36016 m2).
MESURÉ :
1. Juste avant les bipèdes, un record `ti 21` (`flock-emitting-component`) — slot 125 (bit 143 972)
   à 81c02726, slot 141 (bit 164 302) à a0c36016 — est suivi d'une zone de **~43,6 k / ~41,2 k bits
   sans aucun en-tête valide** (recensement K : rien entre 144 161 et 187 719 à 81c02726).
2. Les slots sont épars (125 → 519, 141 → 512) : il n'y a jamais de voisin immédiat (`slot+1`). Le
   balayeur élit donc sur sa fenêtre de 120 000 bits par (génération basse, **slot bas**).
3. Les vrais bipèdes sont bien candidats — ce sont les PREMIERS candidats vus (519 à 43 683 bits,
   512 à 41 199 bits) — mais ils perdent contre une ancre de génération 1 et de slot PLUS BAS située
   plus loin : **slot 256 `ti 0` au bit 206 654** (+62 618 bits) à 81c02726, **slot 385 `ti 19` au bit
   225 276** (+60 910 bits) à a0c36016.
4. Tout ce qui est entre est sauté : 7 bipèdes sur 8, puis 8 sur 8. Depuis la fausse ancre, la chaîne
   repart (536 à 81c02726 ; 1280 à a0c36016). C'est la **perte de préfixe** du rapport.

Pourquoi ces deux ancres sont fausses (DÉDUIT, quatre indices convergents) : elles violent l'ordre des
slots (256 < 535, 385 < 519) ; elles tombent DANS le corps du dernier bipède de la suite (535 + 2 851
bits, entre deux en-têtes bipèdes consécutifs ; 519 + 825 bits) ; `0x40000100 00000000` est un motif
de zone quasi nulle (archétype 0 = mot nul, la classe de faux positifs la plus fréquente : le
recensement K voit un « slot 0, gen 1, arch 0 » dans chaque corps de bipède) ; l'état complet lancé
depuis 256 ne ferme pas sur 536 (fin 206 826 contre 206 882).

**Mécanisme B — arrêt par la fenêtre** (a0c36016 m1). MESURÉ : après le slot 122 (`ti 45`,
`matchflow-sequence-data-component`, bit 139 778), **aucun candidat** dans la fenêtre de 120 000 bits ;
le premier en-tête crédible suivant est à 265 048 (+125 270 bits). La marche s'arrête : perte de
SUFFIXE (120 ancres, objets `ti 38 / 9 / 47` à partir du slot 1280). Cette image-clé n'a aucun
bipède (0 en-tête exact), cohérent avec le §6-4 du rapport (image-clé antérieure à l'origine).

**La marche déterministe ne peut pas prendre le relais aujourd'hui** (MESURÉ) : elle s'arrête au
record n° 1 (slot 1, `ti 34`, composant non porté n° 10) dans les trois payloads ; l'état complet d'un
bipède désynchronise au composant 60 (8/8 à 81c02726 ; 7/8 à a0c36016, le slot 513 au composant 59) ; l'état complet des `ti 18..27, 45`
ne ferme pas (J : désync 0) et celui de `ti 21` se termine 537 bits après l’en-tête sans désync alors que
la frontière suivante est 766 bits plus loin (slot 124) ou 43 210 bits plus loin (slot 125).
HYPOTHÈSE : la zone de ~43 k bits est un corps de longueur variable du record `flock-emitting`
(les nuées), que la grammaire portée ne lit pas.

### 3. Variantes de la règle d'élection (prototypes de mesure, pas de production)

Tout est identique à la production (voisin immédiat, saut de largeur, fenêtre) sauf la règle :
V2 = génération 1 d'abord, puis le candidat le PLUS PROCHE ; V3 = le plus proche des candidats dont
`n1` égale le `n1` modal non nul de leur archétype ; V4 = V2 sans fenêtre.

| payload | règle | ancres | élections | ancres de production perdues | gagnées | bipèdes exacts atteints |
|---|---|---:|---:|---|---:|---|
| 81c02726 m9 | PROD | 523 | 43 | — | — | 1 / 8 |
| | **V2** | 529 | 45 | 1 (256/ti0, la fausse) | 7 | **8 / 8** |
| | V3 | 527 | 43 | 3 (256/ti0 + 768/ti40, 770/ti40 vraisemblablement vraies) | 7 | 8 / 8 |
| | **V4** | 529 | 45 | 1 (256/ti0) | 7 | **8 / 8** |
| a0c36016 m2 | PROD | 352 | 4 | — | — | 0 / 8 |
| | V2 = V3 = V4 | 359 | 4 | 1 (385/ti19, la fausse) | 8 | **8 / 8** |
| a0c36016 m1 | PROD = V2 = V3 | 123 | 0 | — | — | (0 bipède) |
| | **V4** | 243 | 1 | 0 | **120** | (0 bipède) |

## Impact sur le plan (M3.1)

1. **La cause écrite dans M3.1 est réfutée pour ces cas** : les entrées sans archétype ne sont pas
   dans les trous (0 mesuré). La règle des 108 bits reste celle de l'écrivain et peut être portée, mais
   elle ne répare rien ici ; ce n'est pas elle que le gate mesurera.
2. **Ce qu'il faut changer — la règle d'élection** (`kfCand.betterThan`) : remplacer « génération puis
   SLOT le plus bas » par « génération 1 puis le candidat le PLUS PROCHE » (V2). Mesuré : 8/8 + 8/8
   bipèdes, seules les deux fausses ancres perdues. Garder l'élection comme repli NOMMÉ et COMPTÉ.
3. **Retirer la fenêtre de 120 000 bits** (V4) : récupère le suffixe perdu d'a0c36016 m1 (+120
   ancres, 0 perdue). Réserve : le lot 5.20.1 avait mesuré un déraillement sans fenêtre MAIS avec la
   règle « slot le plus bas » (dad793c7, chunks 2 à 5 : 187 → 127) ; avec V2 ce déraillement doit être
   re-mesuré (film dense + parc) avant de trancher.
4. **Ne pas faire de la crédibilité par `n1` la règle principale** (V3 perd des ancres vraisemblablement
   vraies, `ti 40`) ; elle reste utilisable pour départager, et la resynchronisation sur l'en-tête
   EXACT d'un eid connu (`[(1<<30)|slot][35]`, `n1` 152) est confirmée comme repère fiable.
5. Tests rouges à écrire : payload synthétique « record, zone sans en-tête, vrais records de slots
   croissants, fausse ancre de slot plus bas dans le corps du dernier » (la production saute tout,
   V2 non) ; payload à trou > 120 000 bits (la production s'arrête).
6. Gate à ajouter à M3.1 : ce payload (81c02726 m9) et a0c36016 m2 à 8/8 bipèdes ; a0c36016 m1 à
   243 ancres ; puis le gate parc prévu (images-clés touchées 213/2 868 → 0, absorptions 6 → 0).
   La zone de ~43 k bits après `flock-emitting` (HYPOTHÈSE : corps variable des nuées) n'est pas
   bloquante pour V2 ; elle reste un inconnu de la grammaire (§8 du plan).
7. Même balayeur que le monde de la marche et `killsource` : révision de grammaire montée, re-décodage
   (déjà prévu en vague D).

## Découvertes hors périmètre (non traitées)

- Le bipède 532 est présent à t 1494 (le rapport en listait sept) : huit vivants lus.
- Le recensement G voit des en-têtes répétés « slot 32, `ti 2` » à décalage fixe (+1 636 / +1 414
  bits) dans chaque record `ti 43` : référence interne à un record, pas une entrée de table.
- `imagecle_fermeture_research_test.go` et `keyframe_closure_research_test.go` n'ont pas l'étiquette
  `//go:build research` (gardés par variable d'environnement seulement).
