# RAPPORT R2 — Les émissions i48 manquées : perte du film ou perte du scanner ?

> Lot R2 du `PLAN_LECTURE_FIABLE_EQUIPEMENT_2026-09-03.md`. Agent de recherche, 2026-09-03.
> LECTURE SEULE sur la production ; instruments : deux `*_research_test.go` env-gatés.
> Code mesuré : branche `feat/v75` (état de `LevelUp-go-migration` au 03/09).

## Verdict en une phrase

**Perte du SCANNER dans ~3 cas sur 4, mesuré** : les octets des émissions manquées existent
dans le film, dans deux formes de record que le balayage de production rejette par
construction — le record **sans i0** (le masque ne commence pas au composant 0) et le record
à **masque dense R(64)** (bit de porte du masque à 1). Le quart restant est une perte du
FILM (fenêtres balayées candidat par candidat, zéro plausible). Mais le correctif n'est PAS
un simple relâchement des gardes : appliqué inconditionnellement il est réfuté par la mesure
(+800 fausses acceptations sur 10 films) ; la récupération viable est **gatée par le témoin
de compteur** (§6).

## 1. Question et méthode

Le canal i48 (`biped-desired-ability-set-component`) s'auto-mesure par son compteur de
rotation R(3) : entre deux émissions d'une même vie, un pas différent de 1 (modulo 8) compte
les émissions manquées (`equipment_changes.go`, `countEquipmentCounterStep`). Parc : ~5 % de
manques. Question : les octets sont-ils dans le film (défaut de scanner, corrigeable) ou
absents (perte de réplication, incompressible) ?

Méthode : sur une fenêtre bornée, deux balayages du même flux —

1. **STRICT** — réplique bit à bit du balayage de production (`walkAbilityEmissions` :
   `matchBipedHeader` avec tag=1, bits 16-17 nuls, masque à comptage 2..7 commençant à 0,
   i0 absolu de la bonne région, marche des composants de production jusqu'à i48, saut
   post-match). Il note en plus les records i48 dont la marche échoue (Unread).
2. **RELÂCHÉ** — même flux, position de bit par position de bit, sans saut post-match,
   chaque garde relâchée et ÉTIQUETÉE sur le candidat (`tag`, `bit16`, `mc=1`, `noI0`,
   `dense`, `pregate`, `region`). Marche de production tentée quand le point de départ est
   sûr (i0 absolu + bonne région, ou masque sans i0 : départ juste après les indices).

**La prédiction précède le verdict** : le compteur de l'émission manquante est annoncé avant
le balayage relâché (compteur précédent + 1 modulo 8). Un candidat marché, au bon slot, dans
la fenêtre, au compteur prédit = octets retrouvés.

Instruments (rejouables, skip par défaut, `CGO_ENABLED=0`, `LockProcessDecode`, balayages
bornés par fenêtre/paquets) :

- `apps/go-api/internal/analysis/filmdec/i48_manques_research_test.go` — `TestI48ManquesFenetre` (cas index)
- `apps/go-api/internal/analysis/filmdec/i48_manques_parc_research_test.go` — `TestI48ManquesParc` (échantillon) et `TestI48ManquesAvantApres` (contrôle négatif du correctif naïf)

Note d'horloge : les fenêtres `I48M_US_MIN/MAX` sont en **horloge MOTEUR des paquets**
(`FilmPacket.TimestampUS`), pas en timeline artefact. Sur `1b2d9e08` : origine moteur de la
frame 0 = 5 548 533 831 µs (déduite de l'émission taken : TS 5 686 333 831 − 1378×100 000) ;
conversion : `frame = (TS_moteur − 5 548 533 831) / 100 000`.

## 2. Cas index — film `1b2d9e08`, slot 535 (JGtm) : SCANNER, prouvé

Commande exacte (depuis `apps/go-api`) :

```
CGO_ENABLED=0 I48M_FILM=<depot>/data/cache/film_chunks/1b2d9e08 \
  I48M_SLOT=535 I48M_US_MIN=5685000000 I48M_US_MAX=5735000000 \
  go test ./internal/analysis/filmdec/ -run '^TestI48ManquesFenetre$' -v -timeout 30m
```

Le strict retrouve les deux émissions connues et prédit le manque :

| Émission | TS moteur | Localisation | Compteur | Rang |
|---|---|---|---|---|
| taken (t=1378) | 5 686 333 831 | chunk 8, paquet 836, bit 650 | c5 | 4 (grappin) |
| **PRÉDITE** | entre les deux | — | **c6** | — |
| spent (t=1851) | 5 733 614 360 | chunk 10, paquet 1708, bit 157 | c7 | aucun |

Le relâché retrouve l'émission manquante — **les octets existent** :

```
CANDIDAT slot 535 @5700481685us (chunk 9, paquet 134, offset bit 3055) variante count
gardes[noI0] masque [17 25 48 56] — MARCHE OK : c6 rang 11
```

- **Compteur 6 = exactement la prédiction** ; **rang 11 = le translocateur**, dont la prise
  avait été confirmée par un autre chemin (téléportation de 22 m à t=1762). Frame ≈ 1519
  (t artefact ≈ 161,0 s), soit 24 s avant le saut spatial — dans la fenêtre attendue
  [1378, 1762] de l'ancre A1 du plan.
- **La garde qui le rejette est identifiée et unique** : `ascendingFromZero` exige que le
  premier index du masque soit 0 (i0 présent). Ce record N'A PAS d'i0 : masque
  [17 25 48 56] = `object-frame-configuration` + `unit-command-tick` +
  `biped-desired-ability-set` + `biped-spartan-ability-energy` — un ramassage d'équipement
  sans rafraîchissement de position, énergie comprise. Tout le reste de l'en-tête est
  conforme production (tag=1, bits 16-17 nuls, comptage 4, indices croissants) et la marche
  des désers de production lit c6/r11 sans désync.
- Dénominateurs de la fenêtre : 2 997 paquets delta, 658 883 octets balayés bit à bit,
  3 922 candidats généralisés, un SEUL au slot 535 avec profil propre — et il porte le
  compteur prédit. Unread strict = 0 (la marche n'est pas le mécanisme de perte ; c'est
  l'ancrage).

## 3. Généralisation — 12 sauts échantillonnés + le cas index

Sélection : films à `coverage.equipmentChanges.counterJumps > 0` (41 sur 106 artefacts,
films tous présents sur disque). Échantillon : 8 films variés, 12 sauts (plafond
`I48M_MAXJUMPS=12` atteint avant `f0220a96`). Commande exacte :

```
CGO_ENABLED=0 I48M_PARC="<b>/01e1f945,<b>/0a44c6cc,<b>/0d265ab0,<b>/28c9b538,<b>/3923bede,<b>/72b0a25e,<b>/a4083bd2,<b>/c75f33b8,<b>/f0220a96" \
  I48M_MAXJUMPS=12 go test ./internal/analysis/filmdec/ -run '^TestI48ManquesParc$' -v -timeout 60m
```

(`<b>` = `<depot>/data/cache/film_chunks`.) Verdicts, cas par cas :

| # | Film | Slot | Saut | Candidat retrouvé (marche OK, compteur prédit) | Verdict |
|---|---|---|---|---|---|
| 1 | 1b2d9e08 (index) | 535 | c5→c7 | count sans-i0 [17 25 48 56] c6 r11 | SCANNER |
| 2 | 01e1f945 | 530 | c5→c7 | count sans-i0 [17 48] c6 r10 | SCANNER |
| 3 | 0a44c6cc | 548 | c6→c1 (2 manques) | dense [0 1 17 21 25 28 48 56] c7 spent ; count [48] tag=0 c0 spent | SCANNER (2/2) |
| 4 | 0d265ab0 | 521 | c7→c1 | aucun (fenêtre 4 s, 238 paquets, 0 candidat slot) | FILM |
| 5 | 0d265ab0 | 521 | c2→c4 | count sans-i0 [17 48] c3 r10 | SCANNER |
| 6 | 28c9b538 | 603 | c5→c7 | dense [0 1 5 17 21 25 26 48] c6 spent | SCANNER |
| 7 | 3923bede | 523 | c5→c7 | dense [0 1 17 21 25 48 57 59] c6 r9 (fenêtre 254 paquets, zéro bruit) | SCANNER |
| 8 | 3923bede | 544 | c5→c7 | dense [0 1 5 17 21 25 26 48] c6 spent | SCANNER |
| 9 | 72b0a25e | 514 | c7→c1 | aucun (560 paquets, 44 candidats slot, 0 au compteur prédit) | FILM |
| 10 | a4083bd2 | 518 | c5→c7 | count sans-i0 [17 26 48] c6 spent | SCANNER |
| 11 | a4083bd2 | 576 | c7→c1 | count [48] tag=0 mc=1 c0 spent (profil bruité) | SCANNER (faible) |
| 12 | c75f33b8 | 535 | c7→c1 | aucun (fenêtre 0,45 s, 28 paquets, 0 candidat) | FILM |
| 13 | c75f33b8 | 543 | c7→c1 | dense tag=0 msb=false c0 spent (profil bruité) | SCANNER (faible) |

**Taux mesurés** :

- Lecture LIBÉRALE (tout candidat marché au compteur prédit) : **10 sauts SCANNER / 13**
  (77 %), **12 émissions retrouvées / 14** (86 %).
- Lecture CONSERVATRICE (profils aux signatures structurelles récurrentes seulement,
  cf. §5 — lignes 11 et 13 requalifiées bruit) : **8 sauts / 13** (62 %),
  **8 émissions / 14** (57 %).
- Pertes du FILM certaines : 3 sauts / 13 (23 %) — fenêtres courtes balayées candidat par
  candidat, zéro octets plausibles (dénominateurs dans le log : paquets, candidats du slot,
  marches hors prédiction).

## 4. Les deux formes de record que la production rejette

1. **Sans i0** (`noI0`) — le masque ne contient pas le composant 0. La position n'a pas
   changé dans ce record ; l'équipement, si. Masques observés sur les vrais manques :
   [17 48], [17 25 48 56], [17 26 48] — toujours i17 (frame-configuration), souvent i25
   (command-tick), i26 (unit-equipment), i56 (ability-energy). Garde qui rejette :
   `ascendingFromZero` (premier index = 0) dans `matchBipedHeaderRaw`.
2. **Masque dense R(64)** (`dense`) — bit de porte du masque à 1 (grammaire
   FUN_1406d7610 : porte=1 → masque dense de 64 bits au lieu de comptage + indices ; c'est
   la seule forme possible à 8 composants et plus). Ordre de bits mesuré sur les 4 hits
   propres : **bit k du flux = composant 63−k** (`msb=true`). Masques observés :
   [0 1 (5) 17 21 25 (26|28) 48 (56|57|59)] — position + vélocité + les mêmes compagnons.
   Garde qui rejette : `readBitsAt(pay, p+16, 2) != 0` (le bit de porte à 1) dans
   `matchBipedHeaderRaw`. Ces records PORTENT un i0 absolu de la bonne région (vérifié
   candidat par candidat), l'ancre anti-bruit reste donc disponible pour eux.

## 5. Contrôle négatif : le relâchement inconditionnel est RÉFUTÉ

Accepter ces deux formes film-entier, sans autre témoin (`TestI48ManquesAvantApres`,
profils « propres » uniquement : `noI0` seul, ou `dense` seul avec i0 absolu + région) :

```
CGO_ENABLED=0 I48M_PARC="<b>/1b2d9e08,<b>/01e1f945,<b>/0a44c6cc,<b>/0d265ab0,<b>/28c9b538,<b>/3923bede,<b>/72b0a25e,<b>/a4083bd2,<b>/c75f33b8,<b>/f0220a96" \
  go test ./internal/analysis/filmdec/ -run '^TestI48ManquesAvantApres$' -v -timeout 90m
```

Résultat total (10 films) : AVANT 305 émissions, 14 sauts, 0 répétition | **+800
acceptées** (~moitié sans-i0, moitié dense) | APRÈS 1 105 émissions, **310 sauts,
122 répétitions**. Les chaînes de compteur — le témoin du canal — explosent : l'écrasante
majorité des +800 sont des faux positifs. Cause : sans i0, le motif d'ancrage perd ses
~17 bits de zéros obligatoires (spine + useDefault + région), et la « marche jusqu'à i48 »
d'un masque court ([17 48] : lectures plates puis 4-10 bits) réussit sur du bruit. Le même
balayage en fenêtre de 50 s montre l'ampleur : ~140 candidats « marche OK » tous slots,
dont un seul est l'émission réellement manquante.

**Conséquence** : le walk seul ne certifie rien ; la certification vient du témoin de
compteur (prédiction) — c'est exactement ce que le correctif doit encoder.

## 6. Correctif proposé (R2.3 — décision D1, PAS appliqué)

**Récupération gatée par le témoin**, en post-passe de `ScanFilmEquipmentChanges` :

1. Balayage strict inchangé (aucune garde de production affaiblie).
2. Pour chaque vie dont la chaîne porte un saut (pas k>1 entre deux émissions), re-balayer
   la SEULE fenêtre [émission d'avant, émission d'après] avec les deux formes du §4
   (en-tête de production intact par ailleurs : tag=1, bits 16-17 nuls sauf porte dense,
   comptage 2..7 et indices croissants pour la forme sans-i0 ; i0 absolu + région pour la
   forme dense msb=true), marche de production jusqu'à i48 obligatoire.
3. N'accepter que les candidats dont le compteur comble exactement le saut (au plus k−1,
   dans l'ordre), et étiqueter l'émission récupérée (`recovered: true` ou équivalent) —
   le `from` d'une émission suivante redevient fiable, mais la provenance reste dite.
4. Ne rien accepter hors fenêtre de saut : le contrôle du §5 prouve que c'est la seule
   politique sûre.

Dénominateurs mesurés sur les films témoins (§3) :

- Rappel : 8/13 sauts comblés sous profils conservateurs (10/13 en libéral) ; sur le canal
  entier des films témoins : 15 émissions manquées annoncées avant, ~7 restantes après
  (conservateur).
- Fausses émissions : vérification par la fermeture des chaînes — un candidat accepté doit
  transformer le saut en pas de 1 sans créer ni répétition ni nouveau saut (une insertion
  aléatoire a 7 chances sur 8 d'en créer un). Plancher de bruit mesuré : 1 seul candidat de
  profil conservateur hors prédiction sur les 12 fenêtres balayées (~8 % par fenêtre), soit
  une probabilité de comblement erroné de l'ordre de 1-2 % par saut (il faudrait en plus
  que son compteur tombe sur la valeur prédite).
- Reste ouvert avant application : (a) figer l'ordre de bits dense sur plus de 4 témoins ;
  (b) le cas `livesFirstOffSpec` (manques AVANT la première émission d'une vie : fenêtre =
  [naissance, première émission], même mécanique, non mesuré ici) ; (c) golden + schéma si
  l'étiquette `recovered` est publiée.

## 7. Ce que ce lot NE tranche pas

- Les 3 pertes FILM sont acquises sur leurs fenêtres avec dénominateurs, mais un négatif
  reste une absence : une forme de record encore inconnue (en-tête non biped-conforme)
  n'est pas exclue — seulement rendue improbable par la structure des positifs (13/13 des
  positifs tombent dans les deux formes du §4).
- Le sens des émissions récupérées `spent` proches de la mort (le lot R4 publiera le `gap`
  résiduel pour les manques incompressibles).
- Le canal weaponChanges (hors périmètre, noté au plan).

## Annexe — environnement de mesure

Les instruments vivent dans le package `filmdec` et se compilent avec l'arbre `feat/v75`.
Worktree de l'agent basé sur un commit antérieur au package (les fichiers y sont livrés
pour consolidation) ; compilation, `go vet ./internal/analysis/filmdec/` (vert) et toutes
les exécutions faites sur une copie de l'arbre courant de `LevelUp-go-migration` dans le
scratchpad de session (`.../scratchpad/v75/apps/go-api`). Films lus en lecture seule depuis
`LevelUp-go-migration/data/cache/film_chunks/`. Aucune DB ouverte, aucun commit.
