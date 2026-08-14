# PLAN — les capacités d'armure : les nommer toutes, puis les montrer actives

> Écrit le 2026-07-31. **ACTUALISÉ le 2026-08-14** (voir « CE QUI A CHANGÉ DEPUIS LE
> 2026-07-31 » ci-dessous : `i57` réfuté, `i54` mesuré, l'hypothèse de travail déplacée).
> Contrat d'exécution : skill `plan-execution`.
> Deux chantiers distincts que ce plan tient ensemble parce qu'ils partagent leur donnée :
> **savoir QUELLE capacité** (table partielle) et **savoir QUAND elle est active**.

---

## CE QUI A CHANGÉ DEPUIS LE 2026-07-31 — actualisation du 2026-08-14

Trois faits mesurés entre-temps invalident des prémisses de la rédaction d'origine. Ils sont
consignés ici AVANT les étapes, parce qu'ils en déplacent une entièrement.

**1. `i57` n'est plus « absent du switch », et il n'est plus le candidat.** Le composant est
porté depuis (`consumeBipedSpartanAbility`, `traverse.go:822`, largeurs 2 / 28 / 2 / inconnue
selon le tag). Mais sa piste comme INTERRUPTEUR est **réfutée** : le POC avait exploité son
bit 0, qui valait **1 sur 386 lectures sur 386**, puis l'avait retiré. L'étape 3 d'origine
(« brancher `i57`, mesurer ses 2 bits ») est donc **caduque dans sa lettre** : le branchement
est fait, et la mesure qu'elle prescrivait a déjà rendu son verdict.

**2. `i54` « biped-mobility-action » porte un ÉVÉNEMENT DATÉ.** Investigation du 2026-08-13
(lot 1.4 du `PLAN_PARITE_REJEU_2D`, instrument versionné
`apps/go-api/internal/analysis/filmdec/i54_research_test.go`, garde `I54_FILM`) : sur le film
000d5950, 171 851 records delta biped, 2 819 portent `i54` au masque, `flag1==1` sur 2 753 —
soit **67 épisodes discrets** d'environ 0,6 s, 1 à 3 par vie, sur 39 vies sur 99, et 3
seulement à moins de 2 s d'un spawn. Signal d'événement en cours de vie, ni bruit continu ni
action d'apparition. L'investigation a conclu « **non décidable** » faute d'IDENTITÉ
d'équipement voyageant avec l'événement.

**3. Cette clôture est CONTESTÉE, et c'est l'hypothèse de travail de l'étape 3 actualisée.**
L'identité n'a pas besoin de voyager avec l'événement : elle est **déjà connue par ailleurs**.
La capacité ÉQUIPÉE se lit à l'image-clé (`Inventory.A`, cf.
`replay/inventory_decode.go`, règle R1) et s'affiche déjà sur la fiche. Croiser « ce joueur
porte X » × « il a un épisode d'usage à l'instant t » donne l'effet demandé sans rien inventer.
**RESTE À PROUVER** : que l'épisode est bien un usage d'ÉQUIPEMENT et non une escalade ou une
glissade — le corps d'`i54` porte des positions et des vecteurs, compatible avec les deux. Le
témoin indépendant est l'**énergie de capacité `i56`** (masque R(3), 7 bits par charge armée) :
une utilisation doit la faire **CHUTER**. Si la corrélation ne sort pas, on le dit, on s'arrête,
et **aucun effet n'est simulé** — règle absolue du chantier : on ne montre jamais une donnée
qu'on n'a pas mesurée.

**Ce qui ne change pas** : les étapes 1 et 2 valent indépendamment de l'état actif. Une capacité
mieux nommée améliore déjà la fiche.

---

## OÙ ON EN EST, EXACTEMENT

### Ce qui marche

L'index de capacité est lu à chaque image-clé, et le contrôle terrain donne **8/8 sur le nom
de la capacité** (relevé Theater du 2026-07-27, `VERITE_TERRAIN_INVENTAIRE_2026-07-27.md`). Le
décodage est bon.

### Ce qui ne marchait pas — et l'explication qui était proposée

| | |
|---|---|
| ce qu'on lit | l'index sur **3 bits** → 8 valeurs possibles |
| ce que le binaire décrit pour `i48` | **6 bits précédés d'une porte** → 64 valeurs |
| ce que le jeu contient | **11 capacités** |
| ce que notre table nomme | **4** — mur portatif, grappin, propulseur, capteur de menace |

L'hypothèse d'origine : « trois bits ne peuvent pas coder onze capacités ». **Elle est testée à
l'étape 1, et réfutée** — voir le journal de l'étape 1.

### Ce que l'écran doit montrer, une fois la donnée là

Le POC l'a dessiné (Notion 21.1), et l'utilisateur l'a précisé le 2026-08-14 : l'effet porte
sur **TOUTE LA FICHE**, pas sur un liseré — **surbouclier** = fiche encadrée dorée,
**camouflage** = effet de verre sur la fiche, **translocateur quantique** = bordure animée du
bleu électrique au jaune orangé. Les autres capacités restent actives trop peu de temps pour un
traitement dédié. Durée fixe courte, `prefers-reduced-motion` respecté, tokens sémantiques.

---

## ÉTAPE 1 — RELIRE L'INDEX SUR 6 BITS  *(aucune capture nécessaire)* — CLOSE, HYPOTHÈSE RÉFUTÉE

C'était la mesure **déclarée prioritaire depuis le 2026-07-27 et jamais faite**.

- [x] 1.1 Relire les **mêmes records** — ceux qui donnent aujourd'hui 4 index — en lisant
      **6 bits** au lieu de 3, après la porte décrite par le binaire.
      → instrument versionné `apps/go-api/internal/analysis/replay/i48_ability_width_test.go`
      (garde `ABILITY_FILM_ROOT`, lecture seule). Quatre lectures candidates confrontées à la
      MÊME position : `prod3 [p+20,3]` (production), `large6 [p+17,6]` (le §9 de la recette :
      « trois bits plus tôt »), `suite6 [p+20,6]`, `aval6 [p+24,6]`.
- [x] 1.2 **Le contrôle qui pouvait échouer, énoncé AVANT** : la palette `sofd` famille A
      donne capteur = 1 et mur = 2 ; les huit slots du relevé Theater fixent qui porte quoi.
      **RÉSULTAT : `prod3` 6/8, tous les élargissements 0/8.** L'hypothèse est **RÉFUTÉE**.
- [x] 1.3 Distribution des index sur les deux films : `prod3` donne **100 % des valeurs dans
      [0,11)** ; `large6`, `suite6` et `aval6` donnent **0 %** et s'éparpillent sur [0,64) —
      exactement la signature de « lecture hors position » énoncée par ce plan.
- [x] 1.4 **4 capacités distinctes par film**, et le MÊME jeu {3, 4, 5, 6} sur les deux
      (000d5950 Super Fiesta : 3:24 4:44 5:44 6:20 ; 9e8fb31b : 3:20 4:27 5:25 6:32).

**GATE 1 : PASSÉ — l'hypothèse est réfutée PAR ÉCRIT.**

### Journal de l'étape 1 (2026-08-14)

Mesure : 350 records biped d'image-clé sur les deux films, **236 à ancre unique** (67,4 %),
même règle d'unicité que la production.

| lecture | valeurs distinctes | part dans [0,11) | vérité terrain 8 slots |
|---|---|---|---|
| `prod3 [p+20,3]` | 4 — 3:44 4:71 5:69 6:52 | **100,0 %** | **6/8** |
| `large6 [p+17,6]` | 4 — 19:44 20:71 21:69 22:52 | 0,0 % | 0/8 |
| `suite6 [p+20,6]` | 9 — 25/33/38/39/41/46/47/49/54 | 0,0 % | 0/8 |
| `aval6 [p+24,6]` | 4 — 18:227 32:1 38:3 51:5 | 0,0 % | 0/8 |

**POURQUOI `large6` NE POUVAIT PAS INFORMER — et pourquoi la prédiction du §9 de la recette
était en partie INFALSIFIABLE.** Le §9 prédisait « le bit immédiatement précédent vaut 0
partout (la porte « présent ») ». Mesure : **0 sur 236 lectures sur 236**. Mais ce bit est
`p+16`, c'est-à-dire un bit **INTÉRIEUR au motif 20 bits** `invAbilityPattern` (0x00012 =
quinze zéros puis `10010`) : il est constant **par construction de l'ancrage**, pas par une
propriété du flux. La prédiction ne pouvait donc pas échouer, et sa réussite n'apprend rien.
Même mécanique pour les trois bits ajoutés : `p+17..p+19` sont les trois derniers bits du
motif, valeur fixe `010` = 2 — d'où `large6 = 16 + prod3` systématiquement (19, 20, 21, 22).
L'élargissement est **arithmétiquement trivial** : il n'ajoute aucune information.

**CE QUE CELA TRANCHE.** Le champ des images-clés **n'est pas `i48`** — la recette §2 le disait
déjà (« elle ne vient PAS d'`i48` »), la mesure le confirme : à cette position il n'y a ni
porte lisible ni champ de 6 bits. La 3e branche ouverte de la recette §13 (« relire sur 6 bits
et vérifier si le mur ressort au rang 2 ») est **fermée par la négative**. Les deux canaux —
champ d'image-clé et `i48` — doivent bien être traités séparément.

**CE QUE CELA NE TRANCHE PAS, et qui reste une DÉCOUVERTE non traitée** (règle 7 du contrat) :
si le champ était malgré tout un rang de palette lu ailleurs, les valeurs 19–22 tombent
exactement dans la plage `sofd` **non cassée** (rangs 13 à 22, recette §13) — auquel cas la
« contradiction 2/4 » entre notre table et `sofd` se dissoudrait au lieu de se résoudre. Non
mesurable sans casser ces rangs. **Sans effet sur la table** : les 4 noms publiés reposent sur
le relevé Theater (partition 3 grappins / 3 propulseurs / 1 capteur / 1 mur reproduite à
l'identique par le décodeur), pas sur `sofd` — la source la plus forte est intacte.

---

## ÉTAPE 2 — COMPLÉTER LA TABLE, SANS JAMAIS DEVINER

- [x] 2.1 Étendre la table aux index nouvellement observés — **et à eux seuls**.
      → balayage de couverture (même instrument). **UN INDEX NOUVEAU EST APPARU : le 7**, sur
      85 lectures du film `00ba2e1c` — et c'est le SEUL index de ce film, les huit joueurs
      portent le même équipement (mode à dotation unique, pas un Fiesta). **AUCUNE ENTRÉE
      N'EST AJOUTÉE** : l'index est observé, il n'est pas identifié, et la règle du dépôt est
      qu'un nom approchant se lit comme une certitude. Il garde donc son numéro à l'écran
      (le rendu le fait déjà, cf. `ReplayInventoryRow.abilityText`). Le nommer demande un
      relevé terrain — c'est l'objet de l'item 2.3, dont la cible est désormais PRÉCISE.
- [x] 2.2 **Sortir la table du code Go vers les mappings de titre**, bilingue.
      → **DÉJÀ FAIT le 2026-08-02** (vérifié sur pièces) :
      `config/titles/halo_infinite/mappings/replay_labels.toml` section `[abilities]`, quatre
      entrées `en` / `fr` / `icon`, chargées par `internal/games/mappings/loader_replay_labels.go`
      et injectées via `replay.NewLabelCatalog`. Le commentaire de tête de
      `replay/catalog.go` acte la migration. **Aucun libellé n'est en dur dans du Go.**
- [!] 2.3 Croiser avec un relevé terrain pour nommer ce qui manque.
      **NON TRAITÉ — report VALIDE** (règle 3 du contrat : donnée que seul l'utilisateur
      possède). Le balayage a rendu la demande PRÉCISE et bon marché : **ouvrir le film
      `00ba2e1c` en Theater et lire l'équipement d'un joueur** suffit à nommer l'index 7, que
      les huit joueurs portent. Surbouclier et camouflage, eux, n'apparaissent sur aucun film
      mesuré. Ligne au `REGISTRE_REPORTS.md` avec sa condition de reprise.

**GATE 2 : PASSÉ** — la table est étendue aux index nouvellement observés **et à eux seuls**
(donc pas étendue : le seul nouveau n'est pas identifié) ; les capacités jamais nommées sont
**absentes** plutôt que devinées ; aucun libellé n'est en dur dans du Go.

### Journal de l'étape 2 (2026-08-14)

Voir la section MESURES en fin de plan. Deux limites mesurées, notées et NON traitées :
la règle d'ancrage R1 ne rend AUCUNE lecture sur 8 films sur 14 (l'ancre n'y est jamais
unique dans un record), et l'échantillon reste petit.

---

## ÉTAPE 3 — L'ÉTAT ACTIF  *(ACTUALISÉE : `i54` × `i56`, plus `i57`)*

L'énoncé d'origine (« brancher `i57`, publier ses 2 bits, chercher une corrélation ») est
remplacé par la mesure que le registre désigne comme reprise n°1. La justification est en tête
de plan.

- [~] 3.1 Brancher `i57` dans le switch du décodeur et publier ses 2 bits bruts.
      **COUVERT AILLEURS** : `i57` est porté (`consumeBipedSpartanAbility`, `traverse.go:822`).
- [~] 3.2 Mesurer la distribution des 2 bits d'`i57`, et **est-ce qu'ils bougent ?**
      **COUVERT AILLEURS ET RÉFUTÉ** : bit 0 = 1 sur 386/386 au POC. Un interrupteur qui ne
      bascule jamais n'informe de rien ; c'est la raison du retrait du badge d'état.
- [!] 3.3 **Le témoin décisif, actualisé** : croiser les **épisodes `i54`** (événement daté,
      67 sur le film témoin) avec l'**énergie de capacité `i56`** du même slot. Une
      utilisation doit faire chuter l'énergie. **Contrôle qui peut échouer, énoncé AVANT** :
      le taux de coïncidence réel (±1 s) est comparé aux mêmes épisodes **décalés de ±5 s**.
      Si le réel n'écrase pas les témoins décalés, la coïncidence n'est pas une relation.
      → instrument versionné `apps/go-api/internal/analysis/filmdec/i56_energy_test.go`
      (garde `I56_FILM`, lecture seule, 8 s). **MESURE : réel 3/67 (4,5 %) · témoin +5 s
      0/67 (0,0 %) · témoin −5 s 2/67 (3,0 %). Le réel n'écrase pas les témoins.
      LA CORRÉLATION N'EST PAS ÉTABLIE.** Voir le journal ci-dessous pour la cause mesurée.
- [!] 3.4 Confronter au relevé terrain d'une capture datée. **NON TRAITÉ — report VALIDE** :
      seul l'utilisateur peut produire un relevé Theater daté avec ses usages d'équipement.

**GATE 3 : ÉCHOUÉ — on documente et ON NE MONTRE RIEN.** Règle absolue du chantier : on
n'affiche jamais une donnée qu'on n'a pas mesurée. Aucun effet n'est simulé.

### Journal de l'étape 3 (2026-08-14) — pourquoi la mesure échoue

Sur le film témoin 000d5950 : **171 851 records delta biped**, dont 2 819 portent `i54` au
masque (67 épisodes, chiffre re-dérivé à l'identique) mais **176 seulement portent `i56`** —
**0,10 %**, répartis sur 78 slots, soit ~2,3 lectures par slot. Il en sort **28 chutes
d'énergie lisibles pour 67 épisodes** : même une relation PARFAITE plafonnerait à 41,8 %.
**La mesure est limitée par la couverture, pas seulement par l'absence de relation** — et
cette limite est structurelle : le détecteur de records ne reconnaît que les masques
explicites de ≤ 7 composants (`bipedMaxMaskCnt`), les records à masque dense lui échappent,
donc `176` est un PLANCHER. Conclusion honnête : **`i56` est transmis trop rarement dans les
deltas pour dater un usage**, et la piste de reprise n°1 du registre est épuisée en l'état.

**CE QUE LA MESURE CONFIRME QUAND MÊME, et qui est un acquis** : les 176 lectures rendent des
valeurs **toutes multiples de 16** — 0, 16, 32, 48, 64, soit un quartet de poids fort de 0 à 4
et un quartet de poids faible toujours nul. C'est **exactement** l'encodage discret décrit par
`RECETTE_LOADOUT_2026-07-27.md` §9 (`(v>>4)&0xF` charges entières + `(v&0xF)` recharge
fractionnaire), confirmé ici par une voie indépendante du décompile. Les 28 chutes portent
**toutes** sur le quartet de poids fort — donc ce champ EST bien un compteur de charges. Ce
qui manque n'est pas la sémantique : c'est la fréquence de transmission.

Masques observés : `0` (85 fois — aucune charge transmise), `1` (44), `4` (47). Jamais deux
emplacements armés à la fois.

---

## ÉTAPE 4 — L'AFFICHAGE — **NON EXÉCUTÉE, le GATE 3 l'interdit**

La première ligne de cette étape, écrite le 2026-07-31, est sa propre condition : « rien de
tout ceci ne s'affiche avant que l'étape 3 soit passée ». Elle n'est pas passée.

- [!] 4.1 Le badge de capacité porte son état : actif / inactif / **non lu**. **BLOQUÉ par le
      GATE 3** : sans source d'état actif mesurée, les trois états se réduiraient à « non lu »
      partout — ce que la fiche dit déjà en n'affichant rien.
- [!] 4.2 Les trois rendus du POC, **sur TOUTE LA FICHE JOUEUR** (précision utilisateur du
      2026-08-14) et pas sur la carte : surbouclier = fiche encadrée dorée, camouflage = effet
      de verre, translocateur = bordure animée bleu électrique → jaune orangé. Durée fixe
      courte. **BLOQUÉ par le GATE 3** — et doublement : aucun de ces trois équipements n'est
      nommé dans la table (l'index 7 mis à part, aucun index hors {3,4,5,6} n'a été observé),
      donc même l'IDENTITÉ requise pour choisir entre les trois rendus manque.
- [!] 4.3 i18n FR + EN, tokens sémantiques, `prefers-reduced-motion` respecté. **BLOQUÉ** :
      sans 4.2, rien à traduire ni à colorer.
- [x] 4.4 Le compteur d'utilisations **reste absent** : il n'est pas localisé. Vérifié sur
      pièces — `ReplayInventoryRow.tsx` n'affiche aucun compteur, et son commentaire de tête
      porte la mesure (36 006 positions testées). Rien à faire, la règle tient.

**GATE 4 : NON ATTEIGNABLE en l'état.** Aucun code d'affichage n'a été écrit — écrire un effet
alimenté par une source non mesurée serait précisément l'interdit du chantier.

---

## CE QUI PEUT FAIRE ÉCHOUER CE PLAN

- ~~L'hypothèse des 6 bits peut être fausse.~~ **ELLE L'EST** (étape 1, 2026-08-14).
- **L'épisode `i54` peut n'être qu'un mouvement** (escalade, glissade). C'est ce que l'étape 3
  mesure ; dans ce cas le chantier s'arrête là et on l'écrit — c'est une réponse, pas un échec.
- **La table peut rester partielle** : 11 capacités existent, un match n'en montre que ce que
  les joueurs portent. C'est un trou de couverture, pas une dérive.

---

## MESURES PUBLIÉES — 2026-08-14

Toutes re-dérivables en une commande ; les instruments sont versionnés et gardés par
variable d'environnement (sautés en CI).

### Largeur du champ de capacité (étape 1) — 2 films de référence, 236 lectures à ancre unique

| lecture | valeurs | part dans [0,11) | vérité terrain 8 slots |
|---|---|---|---|
| `prod3` (production) | 3:44 4:71 5:69 6:52 | **100,0 %** | **6/8** |
| `large6` (hypothèse du plan) | 19:44 20:71 21:69 22:52 | 0,0 % | 0/8 |
| `suite6` | 9 valeurs sur [25,54] | 0,0 % | 0/8 |
| `aval6` | 18:227 32:1 38:3 51:5 | 0,0 % | 0/8 |

### Couverture de la table de noms (étape 2)

|  | avant | après |
|---|---|---|
| index NOMMÉS dans `replay_labels.toml` | 4 (3, 4, 5, 6) | 4 — **inchangé, volontairement** |
| index OBSERVÉS | 4 (2 films de référence) | **5** (échantillon de 14 films : le **7** apparaît) |
| lectures nommées, 2 films de référence | 236/236 = **100 %** | 236/236 = 100 % |
| lectures nommées, échantillon 14 films | — | 657/742 = **88,5 %** |

Détail de l'échantillon (14 premiers films du cache, un `go test` par film) : **6 films rendent
des lectures**, 8 n'en rendent aucune (l'ancre R1 n'y est jamais unique dans un record).

    000d5950  3:24 4:44 5:44 6:20   (132)      00502e52  3:29 4:52 5:21 6:50   (152)
    01b972df  3:21 4:26 5:20 6:28   (95)       022d2e0e  3:25 4:52 5:35 6:26   (138)
    02d39fa0  3:30 4:50 5:30 6:30   (140)      00ba2e1c  7:85                  (85)
    9e8fb31b  3:20 4:27 5:25 6:32   (104, film de référence, hors échantillon)

**SECOND TÉMOIN DE L'INDEX 7, cherché exprès** (deux films BTB tirés du registre de matchs,
hors échantillon) :

    4f77afc1  3:48 4:40   (88)   BTB:CTF sur Flood Gulch, 2026-07-24     -> pas de 7
    9fe88ec4  7:63        (63)   BTB:Fiesta CTF sur Insolence, 2025-11-13 -> QUE du 7

L'index 7 sort donc sur **deux films indépendants**, tous deux **BTB Fiesta**
(`00ba2e1c` BTB:Fiesta Slayer sur Obituary, 2025-07-25 ; `9fe88ec4` BTB:Fiesta CTF sur
Insolence, 2025-11-13), et sur **eux seuls** — un BTB non-Fiesta récent rend {3,4}. Le motif
est noté ; **il ne nomme rien** et n'entre dans aucune table.

### État actif (étape 3) — film 000d5950

    records delta biped        171 851
    masque ∋ i54                 2 819   -> 67 épisodes discrets
    masque ∋ i56                   176   (0,10 %), 78 slots, 176 lus / 0 illisibles
    valeurs i56 transmises       0, 16, 32, 48, 64  (quartet bas TOUJOURS nul)
    chutes d'énergie                28   dont 28 sur le quartet de poids fort
    COÏNCIDENCE ±1 s             3/67 = 4,5 %
    témoin +5 s                  0/67 = 0,0 %
    témoin -5 s                  2/67 = 3,0 %

## DÉCOUVERTES — notées, NON traitées (règle 7)

- **Rangs 19–22 et la plage `sofd` non cassée** : voir le journal de l'étape 1. Ne pas agir
  sans avoir cassé les rangs 13–22 de la palette.
- **L'ancrage R1 n'est pas universel** : 8 films sur 14 ne rendent AUCUNE lecture de capacité
  (l'ancre 28 bits n'y est jamais unique dans un record). Le rejeu de ces matchs affiche donc
  une fiche sans capacité — lacune correcte, mais la cause n'est pas diagnostiquée.
- **Le balayage de couverture ne passe pas à l'échelle** : 120 films en une seule exécution
  n'avaient pas terminé après 16 minutes (2,9 Go), là où 2 films prennent 8 s. Le coût n'est
  pas linéaire en octets — probablement le balayage bit à bit sur des records mal bornés pour
  d'autres cartes. Un `go test` par film contourne, sans expliquer.
- **`i56` mérite un lecteur de production** si la question de l'énergie revient : sa
  sémantique est confirmée (charges entières sur le quartet haut), seule sa fréquence de
  transmission manque.
