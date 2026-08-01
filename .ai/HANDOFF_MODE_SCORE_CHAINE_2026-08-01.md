# HANDOFF — score par mode, evenements d'objectif, et le parcours de chaine

> Ecrit le 2026-08-01 en fin de session. Branche `feat/re-mode-score` (worktree dedie),
> 8 commits, de `74b4048c6` a `cd0ec554a`. Etat de l'art complet :
> `.ai/ETAT_DE_L_ART_MODE_SCORE_EVENEMENTS.md` (14 sections).
>
> **L'OBJECTIF, a garder en tete** : rejouer le deroule d'un match — la progression du score
> et qui prend une base / attrape un drapeau, a l'instant pres. Tout ce qui suit se lit sous
> cet angle, pas sous l'angle « ou est tel octet ».

---

## 1. CE QUI EST LIVRABLE AUJOURD'HUI, SANS AUCUN DUMP

| ce qu'on peut dessiner | statut | sur quels films |
|---|---|---|
| **Strongholds : qui prend/securise une base, a la ms, avec le gamertag** | **ETABLI** | tous les films en cache |
| CTF : l'instant de chaque capture | disponible | detecteur `tiers==6` **deja dans le depot** (`objectiveevents/film.go`), ms-precis, 0 manque / 0 faux positif sur 4 matchs |
| CTF : qui a pris / rendu / capture | **NON** | l'identite avec `flag_captures+grabs+returns` est refutee |
| KOTH : qui tient la colline | a 5 s pres | tick d'occupation, pas les captures |
| **La courbe de score** | **NON** (voir §3) | sauf sur les 2 films a capture CE |

**La preuve du premier point** : l'identite « 1 evenement de mode du footer type-3 = 1 prise
OU 1 securisation de zone, par joueur » tient sur **8 films Strongholds sur 9**, dont 8 **sans
capture CE**, une fois les joueurs a zero exclus (ils n'emettent rien par construction). Sur
`696a9d7c` : 8/8 exact, total 77 = 77, et le premier evenement decode tombe a **48,90 s sur
FlyGuy8773** quand le releve terrain, ecrit avant tout decodage, dit « 0:48 flyguy8773 capture
la base B ».

Horloge : octets 48-51 du bloc d'evenement, **BIG-endian** (une lecture LE rend des heures).
Meme base que le `start_ms` du manifeste -> replacable directement sur la ligne de temps.

---

## 2. LA COURBE DE SCORE : RESOLUE SUR LE PRINCIPE, PAS ENCORE UNIVERSELLE

**Ou vit le score** : composant **0** de l'**archetype 6** (le statborg — ses 58 composants
pointent tous sur `FUN_140C18794`), dans les paquets **type-0**, jamais dans les images-cles.
L'archetype porte **2 entites d'equipe + 8 entites de joueur**.

| composant | contenu | verifie |
|---|---|---|
| **0** | **score de mode** | Strongholds 200-94 · CTF 3-0 (l'equipe a 0 n'emet jamais) + par joueur = `flag_captures` |
| 1 | score personnel | 8 420 / 7 420 + les 8 joueurs (somme 15 840 = fermeture exacte) |
| 2 | frags / morts | 54/48 et 48/55 · 53/40 et 39/54 |

Toutes exactes a l'unite contre l'API.

**Resolution temporelle** : le composant n'est reemis **que lorsqu'il change** — 190 emissions
monotones pour l'equipe qui mene en Strongholds, 3 emissions pour 3 captures en CTF. C'est
structurellement plus fin que le tick de 5 s du footer.

**La limite** : ce decodage utilise la capture CE pour LOCALISER les lectures. Le pont est
arithmetique et sans parametre libre — la signature de 16 octets est a
`paquet.Start + 8*floor(c/64) + 8`, donc **`bitpos = 8*M - 64 + (c mod 64)`**. Rendement
2 708/2 716 et 2 331/2 340, **zero signature absente**.

---

## 3. LE PARCOURS DE CHAINE — l'etat exact, et le point de reprise

### 3.1 La grammaire, etablie par Ghidra et validee

```
[1 bit presence][2 bits type][W bits slot][2 bits generation]
[1 bit forme][3 bits compte N][N x 6 bits index de composant]
puis les composants presents, dans l'ordre des index
```

- `FUN_1406CD128` : la boucle de trame (types : 1 = NEW, 2 = ?, 3 = DELTA)
- `FUN_1406D3140` : le slot — `[W bits valeur][2 bits gen]`, `id = (gen<<30) | (base+valeur)`
- `FUN_141F86B58` : le delta -> appelle `FUN_14076CB60`
- `FUN_14076CB60` : la boucle de composants ; deserialiseur = `descripteur[i] + 0x28` ;
  nombre de composants a `descripteur + 0x4320`
- **`FUN_1406D7610` : LE MECANISME DE SAUT** — `[1 bit forme]` puis, en forme 0,
  `[3 bits N][N x 6 bits index]` (liste CREUSE), ou en forme 1 un masque plein de 64 bits

**Validations, chiffrees** :
- liste creuse reconnue et premier index = composant observe : **1 078 / 1 090 = 99 %**
- repartition de N : `1:235 · 2:596 · 3:47 · 4:32 · 5:70 · 6:70 · 7:28` -> **un enregistrement
  porte 1 a 7 composants, mediane 2**, jamais les 58 de l'archetype
- champ d'identite : **purete 100 %, 10 valeurs distinctes pour 10 entites**, a 2 bits avant
  le masque
- correspondance mesuree : **`slot_flux = 2 x (eid - 0x40000000)`** — equipes aux slots **6 et 8**,
  joueurs a 10, 12, 14, 16, 18, 20, 22, 24
- largeur derivee du binaire : `FUN_1406D310C` = `ceil(log2)` ; `FUN_140D10BB0` remplit la
  table des bornes ; index 7 (celui de la boucle de trame) -> base 0, borne `0x1FFF` -> **13**

### 3.2 LE POINT DE REPRISE — une mesure, pas une recherche

Cadrage mesure sur les 1 078 en-tetes : les 3 bits precedant le champ de 13 bits valent
`010` (926 fois) ou `110` (151 fois). **Les 2 bits adjacents au slot sont donc `10` dans
1 077 / 1 078.**

Or `10` lu MSB-first vaut **2**, quand le type DELTA vaut **3**. Incoherence a lever.

**L'hypothese la plus probable, et deux mesures la soutiennent deja** : le champ d'identite
fait **14 bits, pas 13**. L'exploration de purete donnait 100 % pour les largeurs 11 a 14 au
meme decalage, et le facteur 2 de `slot_flux = 2 x (...)` est exactement le bit de poids faible
supplementaire qu'une largeur de 14 capte.

**LE CONTROLE A FAIRE EN PREMIER, il tient en quelques lignes** :
fixer la largeur a **14** dans `cmd/tmp_chainhdr` (variable `w`, bloc `FRAME`) et verifier que
les 3 bits precedents deviennent `[1 bit presence][2 bits = 11]`.

- **si `11` apparait** : l'hypothese est etablie. L'ancrage de `cmd/tmp_scorechain` devient
  sur-contraint (presence + type + 14 bits de slot + forme + compte + index croissants bornes,
  soit > 30 bits de contrainte dure), le bruit tombe, et la courbe de score bascule hors ligne
  **sur les 951 films du cache** ;
- **sinon** : il y a un champ intermediaire entre le type et le slot ; le mesurer de la meme
  facon (par soustraction sur les positions connues), ne pas le deviner.

### 3.3 Ce qui a deja ete tente et qui NE marche PAS (ne pas y revenir)

| tentative | resultat mesure |
|---|---|
| ancrage sur 13 bits de slot, avec contrainte presence/type | **5 ancrages** (au lieu de 190) |
| le meme, sans la contrainte | **506 ancrages** bruites, valeurs incoherentes |
| localisateur par motif « 10 bits nuls + 2 valeurs varwidth » | **103 560 candidats** sur un film — sature |
| balayage a largeur fixe ancre sur le jeton `0x7B6` dans les images-cles | mauvais substrat : le score n'est pas la |
| offset fixe consistant sur 46 films KOTH | 0 triplet ; le hasard atteint deja 46 % |

---

## 4. LES OUTILS (tous en `CGO_ENABLED=0`, non suivis par git — regle `.gitignore` de J3)

| outil | role |
|---|---|
| `cmd/tmp_modeticks` | evenements de mode : compte par equipe/joueur, gamertag, horloge BE ; `VERBOSE=1` sort la timeline, `DUMPX=1` les comptes par xuid |
| `cmd/tmp_statborgfilm` | **le pont capture CE -> film** ; decode les stats d'equipe a la position exacte ; `BYEID=1` separe par entite, `PFX=1` sort les prefixes |
| `cmd/tmp_chainhdr` | grammaire d'en-tete ; `MAP=1` sort la correspondance eid -> slot, `FRAME=1` les bits de cadrage |
| `cmd/tmp_scorechain` | **l'ancrage direct hors ligne** — c'est LUI qu'il faut faire passer |
| `cmd/tmp_modescore` · `tmp_modestate` · `tmp_scorefind` · `tmp_scoreread` · `tmp_scoreanchor` · `tmp_scoresweep` · `tmp_statscan` · `tmp_scoreoffline` · `tmp_pickupmap` | mesures des passes precedentes (cartographie, refutations, controles negatifs) |

**Verite terrain disponible** : 5 039 positions de bit exactes avec leurs valeurs attendues
(2 708 sur `696a9d7c`, 2 331 sur `530820e5`), produites par `tmp_statborgfilm`. C'est ce qui
rend le parcours de chaine enfin verifiable enregistrement par enregistrement, au lieu d'etre
pilote par un gradient — le mur historique du chantier killweapon.

Captures CE : `E:/LevelUp_rejeu2D/captures_2026-07-31/` (2 CSV + `deser_table.tsv` +
`archetype_vtables.tsv`). Manifestes de film : `data/cache/film_manifests/<id>.json`
(`start_ms` par chunk — c'est lui qui donne l'alignement d'horloge, il n'etait pas utilise).

---

## 5. LES REGLES QUE CETTE SESSION A PAYEES

1. **Une valeur finale juste ne prouve rien.** Un candidat Strongholds parfait sur le papier
   (30 colonnes, leurres 201/202 a zero, 200-94 monotone) etait FAUX : sa courbe restait a zero
   jusqu'a 400 s quand le releve terrain atteste 21 points a 1:30. Contraint par les ancres :
   0 colonne.
2. **Publier le compte de faux positifs, toujours.** 103 560 candidats, 98 544 colonnes, un
   hasard a 46 % : ces chiffres sont ce qui separe une mesure d'une impression.
3. **Une source externe credible se traite comme une hypothese.** Le depot
   `davidhouweling/guilty-spark` (PR 752-757) parse le meme format ; deux de ses trois
   revendications sont refutees par controle negatif (le mode SANS objectif porte le PLUS de
   transitions de son « signal d'objectif universel »). Sa table mode -> statistique, elle, a
   servi de chaine independante.
4. **Ne pas conclure sur une correspondance visuelle.** Les bascules `ci=6` semblaient coller
   aux evenements du footer en fin de match ; mise a l'epreuve, 22/43 contre **20/43** pour le
   controle decale. Abandonnee.
