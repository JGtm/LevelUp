# PLAN — les capacités d'armure : les nommer toutes, puis montrer leurs usages actifs

> **ÉTAT AU 2026-08-25 — PHASE 0 CLOSE (items 0.1 à 0.7). VERDICT PARTIEL. PHASES 1 ET 2 NON
> EXÉCUTÉES, y compris sur le résultat positif.**
>
> - **RÉPULSEUR (rang 6) — CLASSÉ SUR MESURES.** Aucun canal, sur quatre voies indépendantes :
>   `i56` (ses chutes suivent le grappin), `i51` (jamais transmis), `i59` tags 0/2 (état
>   générique), `i59` tag 1 (2 transitions pour 72 vies). **Conséquence produit : pas de son
>   ni d'effet de répulseur, faute de canal — et non faute de fichier son.**
> - **PROPULSEUR (rangs 5 / 21) — CLASSÉ SUR MESURES à son tour** (Phase 0-bis, 6 films,
>   2026-08-25). La piste `i59 tag==1` ouverte par l'étape 0.7 a été confirmée sous les
>   quatre seuils écrits d'avance : **le seuil de REPRODUCTIBILITÉ est tombé** — 3 films sur
>   6 échouent le contraste (masse à 46-69 % pour 75 % requis), quand le seuil n'autorise que
>   2 échecs sur 8. Le seuil éliminatoire de DATATION était pourtant tenu (0,0 % au spawn sur
>   5 films sur 6) : le signal existe, il n'est pas régulier. **Aucun seuil n'a été
>   renégocié.**
>
> **CONSÉQUENCE PRODUIT, assumée : ni son ni effet de répulseur ou de propulseur — faute de
> CANAL, pas faute de fichier son. L'archive sonore de l'utilisateur n'a jamais été
> sollicitée.**
>
> Le gate de la Phase 0 est PASSÉ (un verdict publié par capacité EST le livrable attendu).
> **Zéro ligne de production, aucun effet simulé, aucun son téléchargé.** Détail, options et
> coûts pour le superviseur : `.ai/V7.5/replay2d/ARBITRAGE_USAGES_EQUIPEMENT_2026-08-25.md`.
> Branche `wt/usages-equipement`, non poussée.

> **LIRE D'ABORD LA SECTION SUIVANTE** (« ACTUALISATION DU 2026-08-24 »). Le reste du
> fichier est un historique daté, encore VRAI pour l'identité des capacités (étapes 1 et
> 2 : la table `replay_labels.toml` est correcte et à jour), mais SUPERSEDED pour l'état
> actif (étapes 3 et 4 : l'hypothèse qu'elles testaient a été rejouée par une mesure plus
> forte ailleurs, et le verdict a changé). Les phases exécutables de ce plan sont
> désormais **PHASE 0 / PHASE 1 / PHASE 2**, plus bas.

---

## ACTUALISATION DU 2026-08-24 — RÉOUVERTURE V7.5, RECADRAGE, DÉCISIONS À TRANCHER

### Décision produit et origine

L'utilisateur a tranché le 2026-08-24 : le chantier « effet d'équipement actif » **reste
DANS la v7.5** (il était en question « rouvrir ou classer post-v7.5 » —
`PLAN_CLOTURE_V75.md` §A1, question ouverte n°1). Objectif produit reformulé : **détecter
et afficher les USAGES d'équipement** (répulseur, propulseur — le camouflage est déjà
livré, voir plus bas) dans le rejeu 2D, et déclencher leurs sons. Ce n'est pas exactement
l'objectif de 2026-07-31 (« montrer l'état ACTIF », pensé pour des états SOUTENUS comme le
surbouclier) : un usage de répulseur ou de propulseur est un ÉVÉNEMENT COURT
(~0,6-0,7 s à l'échelle d'`i54`, éventuellement une autre échelle si le canal retenu est
différent), pas un intervalle actif/inactif. Cette différence de nature guide la
proposition par défaut de rendu (décision D3 ci-dessous).

### Ce qui est déjà livré depuis la rédaction d'origine — NE PAS RETOUCHER

| capacité | canal | statut | où |
|---|---|---|---|
| **Camouflage actif** (rang 8) | `i28` queue[1], interrupteur binaire 0/4095 | **LIVRÉ le 2026-08-16** | `internal/analysis/replay/equipment_episodes.go` (`EquipFamilyCamo`), effet plein-fiche « vitreux » (`backdrop-filter` + voile), son Activate/Deactivate |
| **Surbouclier** (rang 9) | `i5` non clampé, règle `q > 64` | **LIVRÉ le 2026-08-16** | même fichier (`EquipFamilyOvershield`), effet plein-fiche « cadre doré », son Activate/Deactivate |
| **Grappin — usage** (rang 4/20) | `i59 tag==3`, croisé à `i48` (115/117 sur vies rang 20) | **LIVRÉ** (`wt/son-grappin`, fusionné) | `doc.grappleLines[].t0`, `grapple_fire.wav` (archive utilisateur nov. 2021, normalisé -16 LUFS/-1 dBTP gain linéaire, pcm_s16le 48 kHz), catégorie « équipements » de `replaySound.ts` |
| Équipement LÂCHÉ à la mort (tous types, rendu hors Fiesta) | poses `origin=dropped` | **LIVRÉ le 2026-08-19** (`wt/dropped-hors-fiesta`) | `placementDropped.ts` — un objet TROUVÉ au sol, pas un usage par un joueur vivant |

Ces quatre lots répondent chacun à une partie de ce que la rédaction de 2026-07-31
demandait. **Le périmètre réel de cette actualisation se réduit donc à : répulseur (rang
6) et propulseur (rang 5 famille A / 21 famille B) — leurs USAGES, pas leur pose ni leur
identité (déjà nommés, voir étapes 1-2 ci-dessous, toujours vraies).** Translocateur
(rang 11) reste hors périmètre — voir Décision D2.

### Ce qui reste mort — ne pas ré-essayer sans dire pourquoi on le retente

1. **`equipment-charges-remaining` (`i27`) sur les OBJETS DU MONDE (archétype `ti=37`)**
   est RÉFUTÉ (`registre_film/LOTD_PHASE0.md`, 2026-08-15) : 0/501 appariements avec les
   événements de grappin, et généralisé à zéro décroissance sur les 405 vies d'objet des
   cinq familles nommées (répulseur compris : 122 vies, 0 décroissance). **`i27` est un
   champ sur l'OBJET POSÉ dans le monde — une entité différente du `i56` ci-dessous, qui
   est un champ sur le BIPÈDE (le joueur).** Les deux se sont fait appeler « compteur de
   charges » dans des documents différents ; ne pas les confondre est la raison pour
   laquelle ce négatif ne s'applique PAS à `i56`.
2. **`i57` comme interrupteur d'état actif** est RÉFUTÉ deux fois (bit 0 = 1 sur 386/386
   au POC ; ses transitions ne battent leurs témoins qu'à 72,2 % contre 34,2/32,9 % —
   « une erreur sur quatre »). Vivant comme candidat de NOMMAGE de direction
   (translocateur, non instrumenté), mort comme signal d'état.
3. **L'INTERSECTION `i54` (mobility-action) × `i56` (énergie) PAR FENÊTRE TEMPORELLE
   (±1 s / ±5 s)** — c'est la piste que `PLAN_CLOTURE_V75.md` §A1 (écrit le 2026-08-14)
   propose de reprendre. **Elle est morte, et plus profondément qu'en 2026-08-14.** Voir
   Décision D1 : c'est le changement le plus important de cette actualisation par rapport
   au brief qui l'a demandée.

### Ce qui est nouveau et change la donne : la marche hors ligne à haute couverture

Étude du jour (2026-08-24, non commitée, `.ai/V7.5/replay2d/FAISABILITE_SUIVI_DELTA_INVENTAIRE_2026-08-24.md`
dans le dépôt principal — pas encore portée dans ce worktree) : le mécanisme d'ancrage
(`matchBipedHeader`, `offline_biped.go:305`) + marche de curseur (`walkRecordTo`,
`ability_rank.go:188`, qui consomme via `consumeByName`, les désers de PRODUCTION
eux-mêmes) — le même qui sert déjà `i48` en production — atteint **100 % des annonces au
masque pour `i54`** (2 819/2 819) et **97,4 % pour `i57`** (1 253/1 286), contre le
détecteur historique limité aux masques `<= 7` composants qui plafonnait `i56` à
**0,10 %** (176 lectures sur 171 851 records). **`i56` n'a PAS encore été mesuré sous ce
nouveau mécanisme** — c'est le premier item de la Phase 0 ci-dessous. Si sa couverture
suit celle des 14 autres composants bipède déjà recensés (100 % chacun, sauf `i57` à
97,4 % — census du 2026-08-24 : `i22`, `i25`, `i28`, `i30-34`, `i42-44`, `i47`, `i48`,
`i54` à 100 %, `i57` seul sous ce niveau), le plafond de 41,8 % qui a tué la tentative du
2026-08-14 (176 lectures pour 67 épisodes) disparaît.

### Exécution : branche, statuts, clôture

Même contrat que tous les plans de ce dossier : **worktree dédié, branche neuve depuis
`feat/v75`** (proposition de nom : `wt/usages-equipement` — à confirmer au lancement,
aucune branche de ce nom n'existe encore au 2026-08-24), commits par phase, PAS de push.
Statuts d'item : `[x]` fait / `[~]` couvert ailleurs (avec référence) / `[!]` non traité
(avec justification écrite) — **aucune case vide à la clôture d'une phase**. Ordre strict :
Phase 0 close (gate passé, verdict publié pour les deux capacités) avant tout code de
Phase 1 ; Phase 1 close avant tout code de Phase 2 ; une capacité sans canal mesuré en
Phase 0 ne voit ni Phase 1 ni Phase 2. Contrat d'exécution complet : skill
`plan-execution`.

### DÉCISIONS À TRANCHER AVANT EXÉCUTION (proposition par défaut)

**D1 — Le pivot de méthode. LE PLUS IMPORTANT.** `PLAN_CLOTURE_V75.md` §A1 (2026-08-14)
propose de reprendre le croisement `i54` (épisode daté) × `i56` (chute d'énergie) avec une
fenêtre temporelle, en s'appuyant sur la nouvelle couverture. **Un test PLUS FORT que
celui-là a déjà tranché contre `i54` le 2026-08-16** (`PLAN_ETAT_ACTIF_EQUIPEMENT.md`,
phase D, gate PASSÉ) : croisement `i54` × `i48` **PAR VIE** (pas par fenêtre) sur 12
films, 2 519 242 records — les vies portant une capacité de MOBILITÉ (grappin, propulseur,
répulseur) portent **0,55 épisode `i54`/vie**, les vies portant une AUTRE capacité en
portent **0,45** — ratio 1,2, et les autres rangs DÉPASSENT la mobilité sur 4 films sur 12.
Une vie SANS AUCUNE identité `i48` porte quand même 631 épisodes `flag1==1`. Verdict écrit :
« `i54` est une action de mobilité GÉNÉRIQUE (glissade/escalade), pas l'événement d'usage
d'un équipement ». Ce test ne dépend PAS de la couverture d'`i56` — il ne pouvait donc pas
être réparé par la marche à haute couverture. **Une chute de plafond sur `i56` ne
ressuscite pas `i54` comme signal.**
**PROPOSITION PAR DÉFAUT (retenue dans la Phase 0 ci-dessous) : ne PAS rejouer
`i54`×`i56` par fenêtre. Appliquer à répulseur et propulseur la méthode qui a MARCHÉ pour
camo et surbouclier — un canal dédié, cherché PAR RANG, jugé sur son EXCLUSIVITÉ aux vies
du rang cible — en utilisant `i56` (haute couverture, self-référentiel) et `i51` comme
candidats, pas `i54`.**
Si le superviseur veut malgré tout épuiser une troisième fois la piste `i54`×`i56` par
fenêtre (par exemple parce que la Phase 0 ci-dessous échoue), c'est faisable, mais
l'exécutant doit savoir qu'il rejoue un test déjà tranché négativement par une mesure plus
robuste que celle qui le propose — pas une piste vierge.

**D2 — Périmètre exact.** Le brief cite « répulseur, propulseur, camo ». Le camouflage est
livré (tableau ci-dessus). **PROPOSITION PAR DÉFAUT : le périmètre réel de cette
actualisation = répulseur (rang 6) + propulseur (rang 5/21) uniquement.** Translocateur
(rang 11) reste HORS périmètre : toujours bloqué (3 lectures d'identité `i48` dans tout le
corpus mesuré), aucune piste neuve depuis le 2026-08-16, et ni la Phase 0 ci-dessous ni la
nouvelle marche ne changent son problème (rareté du porteur, pas couverture du décodeur).
Mur et capteur (déployables) restent HORS périmètre : leur pose ne se date par aucun canal
mesuré (Phase C, gate PASSÉ comme négatif) — un lot séparé s'il rouvre.

**D3 — Rendu visuel d'un usage COURT.** Aucun patron produit n'existe pour un événement
bref sur la fiche (contrairement à camo/surbouclier, des ÉTATS soutenus avec un début et
une fin). **PROPOSITION PAR DÉFAUT : réutiliser le patron du grappin / du `fireMark` — un
pulse géométrique court (~600 ms) sur le marqueur du joueur, icône de la capacité déjà
publiée dans `replay_labels.toml` (`hud/Repulsor`, `hud/Thruster`)** — PAS l'effet
plein-fiche doré/vitreux (réservé aux états soutenus). Décision utilisateur nécessaire
seulement si un rendu plein-fiche est quand même voulu pour un événement de moins d'une
seconde ; sinon la proposition par défaut s'applique sans aller-retour.

**D4 — Source des sons.** Le brief indique que les fichiers répulseur/propulseur
« existent déjà dans l'archive de l'utilisateur ». **PROPOSITION PAR DÉFAUT : même source
et même traitement que `grapple_fire.wav`** (archive personnelle, PAS les banques `.wem`
du jeu). Une banque `.wem` du JEU existe bien pour le répulseur (`7bd0883c`, 33 fichiers,
10 gestes, `REGISTRE_REPORTS.md` ligne « Son du répulseur », 2026-08-19) mais elle est
réservée au rendu de PLACEMENT (répulseur lâché au sol), un consommateur différent — ne
pas la réutiliser pour l'usage sans décision explicite. Chemins des fichiers d'archive à
obtenir de l'utilisateur avant la Phase 2, comme cela a été fait pour le grappin.

**D5 — Si la Phase 0 ne trouve rien.** Règle absolue du chantier, rappelée pour qu'elle ne
se discute pas en cours d'exécution : **aucun effet simulé sans donnée mesurée.** Si aucun
canal dédié n'est trouvé pour répulseur et/ou propulseur, la Phase 1/2 ne s'exécute PAS
pour la ou les capacités concernées — on documente le négatif (registre + ce plan), on
n'affiche rien. Un verdict partiel (une capacité oui, l'autre non) est un résultat valide,
pas un échec du plan — c'est exactement ce qui s'est produit pour camo/surbouclier
(oui) vs déployables/translocateur (non).

**D6 — Schéma du document.** `SchemaVersion` (constante, `internal/analysis/replay/document.go`)
vaut **18** au 2026-08-24. Ne pas supposer le prochain numéro : au moins deux lots voisins
en vol (`wt/ti47-annonces`, `wt/qualite-score`) peuvent le consommer avant ce lot-ci. La
Phase 1 revérifie la valeur courante sur `feat/v75` fraîchement rebasée avant de coder
(item 1.1).

### Périmètre voisin vérifié — aucun recouvrement

Trois plans ont été committés le 2026-08-24 à la racine de `feat/v75` (`b16ba17e5`),
chacun sur son propre worktree : `PLAN_QUALITE_SCORE_EQUIPE_SYNC.md` (score d'équipe
affichable, `Teams[].Stats`), `PLAN_TI47_ANNONCES_ZONE.md` (annonces de zone, `ti=47 i2`,
rendu en effet UI à la capture), `PLAN_TEMPS_MORT_WEB.md` (temps mort par joueur, calcul
web pur sans re-cuisson). Aucun des trois ne touche `i48`, `i54`, `i56`, `i51`,
`equipment_episodes.go`, `replay_labels.toml` ni la catégorie son « équipements » — vérifié
par lecture de leur objectif et de leur périmètre déclaré. Le seul point de contact
MÉCANIQUE possible est `SchemaVersion` (D6, déjà couvert).

---

## CE QUI A CHANGÉ DEPUIS LE 2026-07-31 — actualisation du 2026-08-14 *(historique)*

> ⚠️ RELECTURE DU SUPERVISEUR, 2026-08-14 — CE PLAN ET LE LOT DU 14/08 ONT CHERCHÉ CE QUI
> ÉTAIT DÉJÀ TROUVÉ. `RECETTE_LOADOUT_2026-07-27.md` §13 (« qui fait foi ») porte la
> PALETTE COMPLÈTE, obtenue par croisement de chaînes murmur3 avec l'énumération
> d'équipement de l'exécutable : rang 1 détecteur de menaces, 2 mur de protection,
> 4 grappin, 5 propulseur, 6 répulseur, **8 camouflage actif, 9 surbouclier,
> 11 translocateur quantique**, 12 traqueur, 23 champ de réparation. Les TROIS capacités
> que l'utilisateur veut rendre actives À L'ÉCRAN sont donc IDENTIFIÉES. Et l'état actif
> est LOCALISÉ : `i57` à `0x12E4` R(2), compteur d'utilisations à `0x140FC1410`
> (masque R(3) puis 7 bits par emplacement armé, `0x7F` = plein). Le travail de
> rétro-ingénierie est FAIT ; il ne reste que du câblage. TROIS OBSTACLES RÉELS, tous
> écrits, aucun ne demande de nouveau reverse :
>
> 1. **LA TABLE DE PRODUCTION EST À MOITIÉ FAUSSE — bug à l'écran, aujourd'hui.**
>    `config/titles/halo_infinite/mappings/replay_labels.toml` déclare `3 = Drop Wall` et
>    `6 = Threat Sensor` ; le §13 dit `2 = mur de protection`, `1 = détecteur de menaces`
>    et `6 = répulseur`. Le §2 de la recette le signale lui-même : « cette table est à
>    moitié fausse, le `sofd` confirme 4 et 5, il CONTREDIT 3 et 6 ». Les fiches joueur
>    affichent donc probablement un mauvais nom de capacité dans deux cas sur quatre.
> 2. **LA PALETTE N'EST PAS GLOBALE** : sur 46 équipements présents dans au moins deux
>    palettes, **20 changent de rang** (grappin rang 4 ici, rang 8 ailleurs ; surbouclier
>    rang 9 ici, rang 15 ailleurs). La table doit être INDEXÉE PAR LE `sofd` DU MATCH, et
>    « déterminer quel `sofd` s'applique à un film donné est la question ouverte qui
>    reste ». C'est LE verrou, et il est de décodage, pas de RE.
> 3. **Le décodeur n'atteint `i56`/`i57` que sur 0,10 % des enregistrements** (mesure du
>    14/08) : les records à masque dense (> 7 composants) ne sont pas traversés. Position
>    connue, traversée manquante.
>
> ORDRE UTILE : (1) corriger la table fausse — gain immédiat, aucune dépendance ;
> (2) résoudre le choix du `sofd` par match ; (3) traverser les masques denses. L'effet
> plein-fiche (doré / verre / bordure animée) se code ensuite, sans rien inventer.

> Écrit le 2026-07-31. **ACTUALISÉ le 2026-08-14** (voir « CE QUI A CHANGÉ DEPUIS LE
> 2026-07-31 » ci-dessous : `i57` réfuté, `i54` mesuré, l'hypothèse de travail déplacée).
> **RÉOUVERT ET RECADRÉ le 2026-08-24** (voir la section en tête de fichier).
> Contrat d'exécution : skill `plan-execution`.
> Deux chantiers distincts que ce plan tient ensemble parce qu'ils partagent leur donnée :
> **savoir QUELLE capacité** (table partielle) et **savoir QUAND elle est active**.

---

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

> **MISE À JOUR 2026-08-16, voir ACTUALISATION DU 2026-08-24 ci-dessus (Décision D1) :**
> le pari de ce point 3 a été rejoué PAR VIE (pas par fenêtre) sur 12 films et a échoué —
> `i54` ne discrimine pas les vies porteuses d'une capacité de mobilité. Le paragraphe
> ci-dessus reste comme trace du raisonnement de l'époque ; il ne fonde plus la Phase 0.

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

> Camouflage et surbouclier : LIVRÉS le 2026-08-16 (voir tableau en tête de fichier).
> Translocateur : hors périmètre de cette actualisation (Décision D2). Répulseur et
> propulseur — absents de cette liste d'origine car regroupés sous « les autres capacités
> restent actives trop peu de temps » — sont exactement l'objet du nouveau périmètre
> (Décision D2) : leur traitement n'est PAS plein-fiche (Décision D3).

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

      > **MISE À JOUR 2026-08-16** : cette dernière phrase est PÉRIMÉE. Le corpus élargi de
      > `PLAN_ETAT_ACTIF_EQUIPEMENT.md` (au-delà des 15 films mesurés le 14/08) trouve des
      > porteurs de rang 8 (camouflage) et 9 (surbouclier) sur `084a804d` (10 et 8 lectures)
      > et `06dfe6d9` (2, 2, plus 3 lectures de rang 11 = translocateur). Les rangs 8 et 9
      > sont donc NOMMÉS depuis le 2026-07-31 (RECETTE_LOADOUT §13) et OBSERVÉS depuis le
      > 2026-08-16 — le trou qui restait ouvert ici est refermé, sans action supplémentaire.
      > Ne reste réellement sans observation dans le corpus mesuré : le translocateur au-delà
      > de ces 3 lectures d'identité (rang 11, toujours trop rare — Décision D2).

**GATE 2 : PASSÉ** — la table est étendue aux index nouvellement observés **et à eux seuls**
(donc pas étendue : le seul nouveau n'est pas identifié) ; les capacités jamais nommées sont
**absentes** plutôt que devinées ; aucun libellé n'est en dur dans du Go.

### Journal de l'étape 2 (2026-08-14)

Voir la section MESURES en fin de plan. Deux limites mesurées, notées et NON traitées :
la règle d'ancrage R1 ne rend AUCUNE lecture sur 8 films sur 14 (l'ancre n'y est jamais
unique dans un record), et l'échantillon reste petit.

---

## ÉTAPE 3 — L'ÉTAT ACTIF  *(historique 2026-08-14 — SUPERSEDED, voir PHASE 0 ci-dessous)*

> **CETTE ÉTAPE EST CLOSE PAR UNE MESURE PLUS FORTE ÉCRITE AILLEURS.** Le gate 3
> ci-dessous (« ÉCHOUÉ, on ne montre rien ») reste correct comme verdict PROVISOIRE du
> 2026-08-14. Il a été confirmé et durci le 2026-08-16 par
> `PLAN_ETAT_ACTIF_EQUIPEMENT.md` (phase D, croisement `i54`×`i48` PAR VIE, 12 films) :
> `i54` n'est PAS un usage d'équipement, point final — voir Décision D1 en tête de
> fichier. Le contenu ci-dessous reste comme trace du raisonnement et des chiffres du
> 2026-08-14 ; il ne fonde plus aucune exécution. Ce qui remplace cette étape pour
> répulseur et propulseur : **PHASE 0**, plus bas.

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

> **CE QUE LA PHASE 0 CI-DESSOUS RÉUTILISE DE CE JOURNAL** : la sémantique d'`i56` (quartet
> fort = charges 0-4, quartet faible = recharge fractionnaire) reste l'acquis le plus solide
> de cette étape. C'est pour ça que la Phase 0 relit `i56` en SÉRIE PROPRE (self-référentiel,
> comme `i28` l'a été pour le camouflage) plutôt que de le croiser à `i54` par fenêtre.

Masques observés : `0` (85 fois — aucune charge transmise), `1` (44), `4` (47). Jamais deux
emplacements armés à la fois.

---

## ÉTAPE 4 — L'AFFICHAGE — *(historique — SUPERSEDED pour camo/surbouclier : LIVRÉ ; reste ouvert pour répulseur/propulseur via PHASE 2)*

> Camouflage et surbouclier sont livrés depuis le 2026-08-16 (tableau en tête de fichier) :
> les items 4.1-4.3 ci-dessous sont donc `[~]` couverts ailleurs POUR CES DEUX FAMILLES.
> Répulseur et propulseur restent bloqués, mais PAS par le GATE 3 de l'étape 3 (qui ne
> portait que sur le trio camo/surbouclier/translocateur de la maquette Notion 21.1) —
> ils sont bloqués par l'absence de canal, question que PHASE 0 rouvre avec une méthode
> différente. Le rendu qui leur serait applicable n'est de toute façon PAS celui décrit
> ici (plein-fiche) : voir Décision D3.

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

**GATE 4 : NON ATTEIGNABLE en l'état pour répulseur/propulseur/translocateur.** Camouflage et
surbouclier l'ont dépassé par une autre voie (voir en tête de fichier).

---

## PHASE 0 — mesurer un canal DÉDIÉ pour répulseur et propulseur *(2026-08-24, remplace l'étape 3 pour ces deux capacités)*

**Ce que cette phase fait, et ce qu'elle ne fait pas.** Elle cherche, PAR RANG, un canal
exclusif aux vies porteuses — la méthode qui a marché deux fois (`i28` pour le
camouflage, `i5` pour le surbouclier) — PAS le croisement `i54`×`i56` par fenêtre
temporelle (mort, Décision D1). Aucune capture Cheat Engine, aucun runtime : offline-pur,
comme tout ce chantier.

- [x] 0.1 **Vérifier sur pièces** l'état d'`i56` dans la marche de production : a-t-il déjà
      un `case` dans `consumeByName` (`traverse.go`) ? Est-il déjà publié par un hook
      appelable depuis `walkRecordTo` (patron `SetAbilitySetHook` pour `i48`) ? L'ancien
      lecteur (`i56_energy_test.go`, étape 3) atteint 176 lectures par UN mécanisme — établir
      lequel avant de coder, pour ne pas dupliquer un lecteur qui existe déjà à moitié.
      Sortie attendue : un court constat écrit (oui/non, ligne de fichier), pas de gate
      chiffré — c'est une lecture, pas une mesure.
      **CONSTAT (2026-08-25) : TOUT EXISTE DÉJÀ, rien à brancher.** `case
      "biped-spartan-ability-energy", "biped-spartan-ability-energy-component"` →
      `consumeBipedSpartanAbilityEnergy(br)`, `traverse.go:810-811` — donc `walkRecordTo` le
      traverse. Hook de publication `SetAbilityEnergyHook` (`ability_energy.go:69`, posé le
      2026-08-15), masque R(3) + 7 bits par charge armée, `AbilityEnergyUnarmed = -1` pour un
      emplacement non armé. Et un instrument utilisant DÉJÀ la marche de production existe :
      `i56_drops_test.go` (garde `I56_DROPS_FILM`). L'item 0.2 n'a donc rien à coder.
- [~] 0.2 **COUVERT PAR L'INSTRUMENT EXISTANT** `i56_drops_test.go` — aucun lecteur neuf
      écrit (item 0.1 : `i56` est déjà branché ET publié). **SEUIL ATTEINT, ET LA PRÉMISSE DE
      LA PHASE 0 EST RÉFUTÉE AU PASSAGE.** Mesure du 2026-08-25 sur `000d5950` :
      `masque∋i56 176 (0,102 %) · i56 LU 176 · i56 illisible 0` → **couverture de lecture
      176/176 = 100,0 %**, très au-dessus du seuil de 80 % posé avant la mesure.
      **MAIS le plafond de 176 n'a JAMAIS été un problème de couverture** : il est la
      FRÉQUENCE D'ANNONCE. `i56` n'est annoncé au masque que dans 0,102 % des records, et
      tout ce qui est annoncé est lu. La cause supposée (« les records à masque dense
      échappent au détecteur, 176 est un PLANCHER ») est mesurée MARGINALE par
      `TestI57Reach` sur le même film : **records reconnus 171 979 — masque CREUX 171 778
      (99,9 %) · masque DENSE 201 (0,1 %)**, et l'instrument conclut de lui-même « AUCUN
      composant ne casse la marche : la traversée n'est PAS le facteur limitant ».
      La marche à haute couverture ne pouvait donc rien débloquer : il n'y avait rien de
      bloqué. Le contenu original de l'item est conservé ci-dessous pour mémoire.
      *(original)* Si `i56` n'est pas déjà branché sur `walkRecordTo` : l'y brancher (grammaire déjà
      connue et confirmée à l'étape 3 — masque R(3) + 7 bits par charge armée,
      `RECETTE_LOADOUT_2026-07-27.md` §9), hook de publication dédié, instrument de
      recherche neuf `apps/go-api/internal/analysis/filmdec/i56_delta_coverage_test.go`
      (garde `I56D_FILM`, lecture seule, un seul décodage par process — règle D17).
      **Seuil posé AVANT la mesure** : couverture ≥ 80 % des annonces au masque sur
      `000d5950` (rappel : ancien détecteur = 0,10 %, 176/171 851 ; la marche atteint déjà
      100 % sur 14 des 15 autres composants bipède du même census, 97,4 % sur le
      quinzième). **Sous ce
      seuil, la Phase 0 s'ARRÊTE ICI et documente un négatif de couverture** — ne pas
      enchaîner sur 0.4 avec des données trop clairsemées, c'est exactement l'erreur du
      2026-08-14.
      Commande : `cd apps/go-api && CGO_ENABLED=0 I56D_FILM=<repo>/data/cache/film_chunks/000d5950 go test ./internal/analysis/filmdec/ -run '^TestI56DeltaCoverage$' -v`
- [x] 0.3 **GATE PASSÉ, largement** (mesure du 2026-08-25, `TestI48PaletteRank`, un `go test`
      par film, aucun code neuf comme prescrit). Vies par rang, sur les six films du corpus :

      | film | famille | records | i48 lu/annoncé | rang 5 PROP. | rang 6 RÉP. | rang 21 PROP. |
      |---|---|---|---|---|---|---|
      | `084a804d` | A | 330 981 | 143/143 | **9 vies** | **14 vies** | — |
      | `06dfe6d9` | A | 336 212 | 230/230 | **19 vies** | **28 vies** | — |
      | `00ba2e1c` | A | 240 645 | 206/206 | **16 vies** | **31 vies** | — |
      | `000d5950` | B | 171 851 | 92/92 | — | — | **23 vies** |
      | `00502e52` | B | 182 876 | 82/82 | — | — | **8 vies** |
      | `07aa428d` | B | 165 198 | 56/56 | — | — | **12 vies** |

      Seuil demandé (≥ 5 vies répulseur sur un film ET ≥ 5 vies propulseur sur un film) :
      atteint sur **trois** films pour chaque capacité. Élargissement au lot `d4be4ab95`
      INUTILE. `i48` est lu à 100 % de ses annonces sur les six films (759/759, 0 illisible).
      **CORRECTION D'UNE HYPOTHÈSE DU PLAN** : `00ba2e1c` ne peut PAS servir de témoin
      négatif (l'item 0.4 le proposait « à vérifier en 0.3 ») — il porte au contraire **31
      vies de rang 6 et 16 de rang 5**, c'est le film le PLUS riche en répulseurs du corpus.
      Le témoin retenu à la place est INTERNE (les autres rangs du même film) et le groupe
      **sans identité `i48`**, plus sévère encore.
      *(original)* Identifier les vies porteuses de répulseur (rang 6, famille A uniquement — aucun
      rang famille B n'a été établi pour lui) et de propulseur (rang 5 famille A / 21
      famille B) sur un corpus candidat, **par l'instrument EXISTANT, aucun code neuf** :
      `filmdec/i48_rank_test.go` (garde `I48_FILM`). Réutiliser le corpus déjà rassemblé
      plutôt que d'en ouvrir un nouveau : `084a804d`, `06dfe6d9`, `00ba2e1c` (famille A,
      corpus Phase A/B) et `000d5950`, `00502e52`, `07aa428d` (famille B, corpus Phase D —
      `000d5950` porte 3 propulseurs confirmés au relevé Theater du 2026-07-27). Publier un
      tableau vies × rang, même format que celui déjà produit pour camo/surbouclier.
      **Gate** : au moins un film avec ≥ 5 vies répulseur ET un film avec ≥ 5 vies
      propulseur. Si absent sur ces six films, élargir au lot `d4be4ab95` (12 films) avant
      d'abandonner — ne pas conclure sur un corpus qu'on n'a pas vérifié suffisant.
- [x] 0.4 **FAIT, ET LE CONTRÔLE ÉCHOUE POUR LES DEUX CAPACITÉS.** Instrument neuf
      `apps/go-api/internal/analysis/filmdec/i56_rank_cross_test.go` (garde `I56X_FILM`) —
      **dans `filmdec/` et non dans `replay/`** comme le plan l'annonçait : la mesure a besoin
      du hook non exporté d'`i56`, du détecteur de records et des désers de production, tous
      internes à `filmdec`. Jointure PAR VIE (slot), jamais par fenêtre. Le tableau publie le
      dénominateur qui rend l'exclusivité lisible (vies AYANT UNE LECTURE d'`i56`), sans quoi
      « 0 chute sur ce rang » se lirait comme une exclusivité au lieu d'un trou de
      transmission.

      **Chutes de charge par vie-lue** (quartet fort décroissant, même définition qu'au
      2026-08-15) :

      | film | famille | GRAPPIN (4/20) | **PROPULSEUR (5/21)** | **RÉPULSEUR (6)** | autres rangs | TÉMOIN sans identité |
      |---|---|---|---|---|---|---|
      | `00ba2e1c` | A | **0,59** | 0,30 | 0,17 | 0,09 | 0,60 |
      | `06dfe6d9` | A | **0,83** | 0,14 | 0,16 | 0,27 | 0,91 |
      | `084a804d` | A | **1,15** | 0,27 | 0,27 | 0,69 | 0,02 |
      | `000d5950` | B | 0,52 | **0,69** | — | 0,15 | 0,60 |
      | `00502e52` | B | 0,53 | **1,25** | — | 0,17 | 1,38 |
      | `07aa428d` | B | 1,70 | **2,42** | — | 0,30 | 0,93 |

      **RÉPULSEUR — verdict NON.** 0,17 / 0,16 / 0,27 chute par vie-lue, quand le GRAPPIN du
      MÊME film en porte 0,59 / 0,83 / 1,15 — soit 3,5 à 4,3 fois plus. Le critère échoue
      dans les deux sens : la cible est minoritaire, ET les autres rangs ne sont pas à zéro
      (sur `084a804d` ils font 0,69 contre 0,27 à la cible).
      **PROPULSEUR — verdict NON.** Famille A (rang 5) : 0,30 / 0,14 / 0,27, dominé par le
      grappin sur les trois films. Famille B (rang 21) : 0,69 / 1,25 / 2,42, en tête de son
      film à chaque fois — mais le grappin (rang 20) suit à 0,52 / 0,53 / 1,70 (ratio 1,3 /
      2,4 / 1,4 seulement) et surtout **le témoin « sans identité `i48` » l'ÉGALE ou le
      DÉPASSE sur deux films sur trois** (1,38 contre 1,25 ; 0,60 contre 0,69). Un canal qui
      ne bat pas des vies dont on ne sait rien ne distingue rien.
      **Le bar de comparaison, rappelé** : le camouflage (Phase A) avait donné 39 transitions
      sur les vies du rang cible contre **0 sur 574 autres vies**. C'est cela, une
      exclusivité. Rien ici n'en approche.
      **CE QUE LA MESURE ÉTABLIT QUAND MÊME** : `i56` est bien un compteur de charges — mais
      il suit la capacité la plus CONSOMMATRICE de charges du film, le grappin (rang 4
      famille A, rangs 20/21 famille B), lequel a déjà son canal dédié livré (`i59 tag==3`).
      Ce n'est pas un canal par capacité, c'est un compteur générique de l'équipement porté,
      transmis trop rarement (0,089 à 0,125 % des records) pour dater quoi que ce soit.
      *(original)* **Le test qui remplace le croisement mort.** Sur les vies identifiées en 0.3,
      examiner la série `i56` PAR VIE (self-référentielle — PAS de croisement à `i54`) :
      cherche-t-on des paliers de décrément du quartet fort (0→4, déjà confirmé
      sémantiquement à l'étape 3) suivis d'une remontée — la signature d'un usage puis
      d'une recharge ? **Contrôle EXACTEMENT celui de la Phase A/B, même bar, pas un
      nouveau chiffre inventé** : (a) exclusivité INTERNE — ces paliers tombent-ils
      quasi-exclusivement sur les vies du rang cible du même film, et quasi jamais sur un
      échantillon d'autres rangs ? (b) témoin INTER-FILMS — un film sans porteur du rang
      cible (`00ba2e1c` n'a ni rang 6 ni rang 9 confirmés : à vérifier en 0.3) montre-t-il
      ~0 palier comparable ? Seuil : reprendre tel quel celui de la Phase A (quasi-totalité
      de la masse de transitions sur les vies du rang cible ; 0 ou quasi-0 sur le témoin).
      **Verdict PAR CAPACITÉ** — répulseur peut réussir pendant que propulseur échoue, ou
      l'inverse ; pas de verdict groupé.
      Instrument : `apps/go-api/internal/analysis/replay/i56_capacity_episodes_research_test.go`
      (nouveau, gardé, s'appuie sur 0.2).
- [x] 0.5 **FAIT (0.4 ayant échoué pour les deux capacités, la condition d'ouverture est
      remplie). VERDICT : `i51` NE VOYAGE PAS.** Le déser jetait ses 8 bits ; il les PUBLIE
      désormais (`emp_timer.go` + `publishEmpTimer` dans `traverse.go:495`, largeur
      inchangée, même règle que `i48`/`i56` : c'est le déserialiseur qui publie). Instrument
      `i51_rank_cross_test.go` (garde `I51X_FILM`), même tableau et même bar que 0.4
      (`rank_cross_shared_test.go`, une seule implémentation pour les deux canaux).
      **MESURE, six films, 1 427 763 records delta biped cumulés : `masque∋i51 = 0` sur les
      six.** Le composant est correctement nommé dans l'archétype
      (`"biped-emp-timer-component"`) mais n'est JAMAIS annoncé dans un paquet delta. Aucun
      tableau d'exclusivité n'a pu être produit — il n'y a rien à croiser. Négatif net, sans
      zone grise. *(À noter : sa sémantique documentée — « combien de temps le joueur reste
      neutralisé » — en faisait de toute façon un effet SUBI, pas une action ; la mesure
      tranche sans avoir à en débattre.)*
      *(original)* **Candidat secondaire, coût faible, SEULEMENT si 0.4 échoue pour une des deux
      capacités** (règle d'ordre : ne pas paralléliser sans raison écrite) : `i51
      biped-emp-timer`, jamais interrogé (`PLAN_EQUIPEMENT_TI37.md`, section
      Découvertes). Hook minimal, publier ses valeurs brutes et leur cadence de
      transmission, même contrôle d'exclusivité qu'en 0.4.
- [x] 0.6 Verdict écrit, PAR CAPACITÉ, publié au `REGISTRE_REPORTS.md` : **`[!]` AUCUN CANAL
      pour le répulseur, `[!]` AUCUN CANAL pour le propulseur**, cause mesurée documentée,
      **aucun effet simulé** (Décision D5). Arbitrage détaillé pour le superviseur :
      `.ai/V7.5/replay2d/ARBITRAGE_USAGES_EQUIPEMENT_2026-08-25.md`.

- [x] 0.7 **FAIT (3 films). VERDICT EN TROIS TEMPS : tags 0 et 2 = ÉTAT GÉNÉRIQUE (négatif) ·
      contrôle de validité PASSÉ · TAG 1 = SIGNAL DISCRIMINANT POUR LE PROPULSEUR, à
      confirmer.** Détail et seuils de reprise dans le journal de l'item, plus bas.
      *(énoncé d'origine ci-dessous, inchangé)*
      **SONDE `i59` TAGS 0 ET 2 — ajoutée au plan le 2026-08-25 sur DÉCISION DU
      SUPERVISEUR**, après l'arbitrage de clôture (option A). Périmètre **strictement
      instrumental** : aucun code de production, aucune publication, et **les Phases 1-2 ne
      s'ouvrent PAS même si le verdict est positif** — dans ce cas l'exécutant écrit un
      arbitrage de reprise avec seuils proposés et s'arrête pour validation.
      Croiser les tags **0 et 2** d'`i59` aux rangs `i48` **PAR VIE**, exactement comme
      l'item 0.4, sur **3 films** : `00ba2e1c` (3 234 lectures / 98,8 % mesurées le
      2026-08-25), puis un film à répulseur dominant et un à propulseur dominant tirés du
      tableau de l'item 0.3.
      **BARRE DE DÉCISION, POSÉE AVANT LA MESURE et écrite dans l'instrument** :
      **POSITIF** seulement si les transitions se CONCENTRENT sur les vies du rang cible ET
      sont ~nulles sur les vies des autres rangs ET sur les vies SANS identité `i48` — la
      forme du camouflage (39 transitions sur la cible, 0 sur 574 autres vies).
      **NÉGATIF** si le signal est présent PARTOUT à volume comparable : c'est un ÉTAT
      GÉNÉRIQUE, le défaut exact d'`i57` (bit 0 à 1 sur 386/386), et **on classe sans
      renégocier le seuil**.
      **CONTRÔLE DE VALIDITÉ INTERNE** : le tag **3** est mesuré avec les autres bien qu'on
      connaisse sa réponse (canal du grappin, 115/117 sur les vies de rang 20). Il DOIT
      ressortir concentré sur le groupe GRAPPIN ; sinon c'est la MÉTHODE qui est en cause et
      le verdict sur les tags 0/1/2 ne vaut rien.
      Instrument : `apps/go-api/internal/analysis/filmdec/i59_rank_cross_test.go` (garde
      `I59X_FILM`), réutilisant le balayage existant `i59Scan` et le tableau partagé
      `rank_cross_shared_test.go` — aucun second lecteur du même champ.

### Journal de l'item 0.7 (2026-08-25) — la sonde `i59`

Trois films, un processus par film : `00ba2e1c` (famille A, mixte riche), `06dfe6d9`
(famille A, répulseur dominant : 28 vies de rang 6), `000d5950` (famille B, propulseur
dominant : 23 vies de rang 21). Couverture d'`i59` : **3 234/3 274 · 4 720/4 817 ·
1 309/1 342**, soit **97,5 à 98,8 % des annonces**, sur 0,78 à 1,43 % des records — un canal
dix fois plus bavard qu'`i56`, comme l'arbitrage l'annonçait.

**1. LE CONTRÔLE DE VALIDITÉ EST PASSÉ, et c'est ce qui rend la suite opposable.** Le tag 3
(grappin, canal déjà livré) devait ressortir concentré sur le groupe GRAPPIN. Il le fait, sur
les trois films — transitions par vie-lue :

| film | GRAPPIN (4/20) | RÉPULSEUR (6) | PROPULSEUR (5/21) | autres rangs | sans identité |
|---|---|---|---|---|---|
| `00ba2e1c` | **1,08** | 0,26 | 0,06 | **0,00** | 0,23 |
| `06dfe6d9` | **1,63** | 0,49 | 0,00 | **0,05** | 0,26 |
| `000d5950` | **1,14** | — | 0,04 | **0,00** | 0,22 |

La méthode discrimine donc quand il y a quelque chose à discriminer. Un négatif obtenu avec
elle est une propriété du signal, pas un défaut de la mesure.

**2. TAGS 0 ET 2 — ÉTAT GÉNÉRIQUE, verdict NÉGATIF, on classe sans renégocier le seuil.**
Le tag 0 est porté par **99 à 100 % des vies lues**, le tag 2 par **100 %**. Leurs transitions
sont partout, à volume comparable :

| tag | film | RÉP. | PROP. | GRAPPIN | autres rangs | sans identité |
|---|---|---|---|---|---|---|
| 0 | `00ba2e1c` | 8,51 | 7,75 | 9,00 | 6,79 | 7,35 |
| 0 | `06dfe6d9` | 10,78 | 11,32 | 8,97 | 8,81 | 7,11 |
| 0 | `000d5950` | — | 7,32 | 6,95 | 6,46 | 5,00 |
| 2 | `00ba2e1c` | 7,80 | 6,75 | 7,64 | 5,93 | 6,48 |
| 2 | `06dfe6d9` | 9,89 | 10,11 | 7,67 | 7,95 | 6,14 |
| 2 | `000d5950` | — | 5,36 | 5,14 | 5,46 | 3,70 |

L'écart maximal entre une cible et le témoin « sans identité » est de **1,5×**, là où le tag 3
fait 20× et plus. **C'est exactement le défaut d'`i57`** (bit 0 à 1 sur 386/386) : un état que
tout le monde porte tout le temps ne date personne. La réserve écrite dans l'arbitrage avant
la mesure — « tags 0+2 = 97 % des lectures, risque d'état générique » — est confirmée.

**3. TAG 1 — LE RÉSULTAT INATTENDU : il discrimine LE PROPULSEUR.** Le tag 1 était le parent
pauvre (10, 13 et 43 lectures). Sa forme est celle qu'on cherchait :

| film | RÉP. (6) | **PROP. (5/21)** | GRAPPIN (4/20) | autres rangs | sans identité | événements tag 1 |
|---|---|---|---|---|---|---|
| `00ba2e1c` | 0,03 | **0,50** | **0,00** | 0,00 | 0,03 | 10 |
| `06dfe6d9` | 0,03 | **0,32** | **0,00** | 0,02 | 0,05 | 13 |
| `000d5950` | — | **1,52** | **0,00** | 0,07 | 0,13 | 43 |

**Zéro transition sur les 76 vies de grappin cumulées** — le confondeur naturel, celui qui
avait avalé `i56`. Quatre transitions sur 209 vies d'autres rangs. **52 des 66 transitions
(78,8 %) tombent sur des vies de propulseur**, alors que ces vies ne pèsent que 7,6 % à 25,8 %
des vies lues selon le film — un enrichissement de **3,4× à 9×**. Le rapport cible/autres
rangs va de 16× à l'infini (0,00 au dénominateur), et cible/sans identité de 6,4× à 17×.

**Le répulseur, lui, ne sort sur AUCUN tag** : 0,03 · 0,03 sur le tag 1 (2 transitions pour 72
vies), noyé sur les tags 0 et 2, sous le grappin sur le tag 3. **Classé sur mesures.**

**CE QUI MANQUE AVANT D'APPELER ÇA UN CANAL — seuils proposés pour la reprise, à valider par
le superviseur :**

1. **Volume.** 66 transitions cumulées, c'est peu. Exiger **≥ 150 transitions `tag==1`
   cumulées** sur un corpus élargi à **8-10 films portant du propulseur** dans les DEUX
   familles (rang 5 et rang 21) — le corpus actuel n'en a que trois.
2. **Reproductibilité du contraste.** Sur **au moins 6 films sur 8** : ≥ 75 % de la masse des
   transitions sur les vies de propulseur, **≤ 0,10 par vie-lue sur les vies de grappin** (le
   confondeur), **≤ 0,15 sur les vies sans identité**.
3. **Contrôle de datation, jamais fait pour ce tag.** Les instants `tag==1` doivent tomber EN
   COURS DE VIE et non à l'apparition — le contrôle qui avait sauvé l'interprétation d'`i54`
   (« 3 épisodes seulement à moins de 2 s d'un spawn »). Un tag qui ne se lèverait qu'au
   spawn daterait une dotation, pas un usage.
4. **Sémantique.** Le corps d'`i59` n'est porté que pour `tag==3` ; on ne sait pas ce que
   `tag==1` transporte. Établir s'il a une charge utile avant d'en faire un événement produit.

**Tant que ces quatre points ne sont pas tenus, aucun effet n'est affiché** (Décision D5) —
et conformément à la consigne du superviseur, **les Phases 1 et 2 restent fermées** même sur
ce résultat positif : la suite est un arbitrage de reprise, pas une implémentation.

**GATE PHASE 0 : PASSÉ — verdict publié pour répulseur ET pour propulseur** (négatif pour
les deux, ce que la Décision D5 qualifie explicitement de résultat valide et non d'échec du
plan) ; contraintes machine D17 respectées (voir plus bas) ;
`go vet ./internal/analysis/filmdec/... ./internal/analysis/replay/...` **exit 0** et
`CGO_ENABLED=0 go test ./internal/analysis/filmdec/ ./internal/analysis/replay/` **verts**
(`ok filmdec 10,598s`, `ok replay 17,642s`, 2026-08-25).

**CONSÉQUENCE, appliquée sans discussion (Décision D5 + règle d'ordre de la section
« Exécution ») : les PHASES 1 et 2 NE S'EXÉCUTENT PAS.** Aucune ligne de production, aucun
rendu, aucun son. « Une capacité sans canal mesuré en Phase 0 ne voit ni Phase 1 ni
Phase 2 » — c'est le cas des deux. Les items des deux phases sont statués `[!]` ci-dessous
avec cette justification unique.

### Journal de la Phase 0 (2026-08-25) — ce que la mesure a déplacé

**La prémisse qui a motivé la réouverture était fausse, et c'est le résultat le plus
réutilisable de ce lot.** L'actualisation du 2026-08-24 pariait que le plafond d'`i56` (176
lectures) était un PLANCHER dû au décodeur, et que la marche à haute couverture le ferait
tomber. Trois mesures indépendantes disent le contraire :

1. `i56` était **déjà** branché (`traverse.go:810`) et **déjà** publié
   (`ability_energy.go:69`) — il n'y avait rien à brancher ;
2. tout ce qui est annoncé est lu : **176 annoncées, 176 lues, 0 illisible** (100,0 %) ;
3. la cause supposée — les records à masque dense — pèse **201 records sur 171 979
   (0,1 %)**, et l'instrument de census conclut lui-même que « la traversée n'est PAS le
   facteur limitant ».

`i56` n'est pas mal décodé : **il est rarement transmis** (0,089 à 0,125 % des records
selon le film). Le journal de l'étape 3 le disait déjà en 2026-08-14 (« `i56` est transmis
trop rarement dans les deltas pour dater un usage ») ; ce lot le confirme avec le
dénominateur correct et ferme la reprise n°1 du registre.

**Ce que le croisement par rang apporte de neuf**, au-delà du verdict : `i56` n'est pas
inerte — il chute, et ses chutes ne sont pas réparties au hasard. Elles suivent le
**grappin**, sur les deux familles de palette et sur les six films. C'est cohérent avec sa
sémantique (compteur de charges) et avec le jeu (le grappin est la capacité dont on
consomme le plus de charges). Mais un canal qui répond « grappin » quand on lui demande
« répulseur » n'est pas un canal de répulseur — et le grappin, lui, a déjà son canal dédié
livré (`i59 tag==3`).

**Coût machine (D17), mesuré et non supposé** : 19 décodages de film, un processus par
film, jamais deux vivants à la fois. **Pic RSS observé : 23 MiB** (plafond 3 072 MiB
surveillé par `Start-Process -PassThru` / `WorkingSet64`, jamais approché). Durées : 5 s
(`000d5950`, 21 Mo) à 127 s (`084a804d`, 60 Mo). Le film-bombe `51101d1d` n'a jamais été
ouvert.

---

## PHASE 0-BIS — confirmer ou tuer `i59 tag==1` pour le PROPULSEUR *(2026-08-25, autorisée par le superviseur après l'étape 0.7)*

**Périmètre : instrumental pur.** Aucune ligne de production, aucune publication. Les quatre
seuils ci-dessous ont été écrits à l'étape 0.7, AVANT cette mesure, et validés tels quels par
le superviseur : **aucun n'a été renégocié.** Instrument :
`apps/go-api/internal/analysis/filmdec/i59_tag1_confirm_test.go` (garde `I59T1_FILM`), qui
instruit les quatre en une passe par film.

- [x] 0-bis.1 **SEUIL (1) VOLUME — ≥ 150 transitions `tag==1` cumulées : NON ATTEINT, 147.**
      Six films mesurés (les deux familles), 147 transitions dont 107 sur des vies de
      propulseur. À 3 transitions du seuil — non bloquant en soi, mais non atteint.
- [x] 0-bis.2 **SEUIL (2) REPRODUCTIBILITÉ — TOMBÉ, et c'est lui qui tranche.**

      | film | fam. | masse prop. | total | % prop. (≥ 75 %) | grappin (≤ 0,10) | sans identité (≤ 0,15) | verdict |
      |---|---|---|---|---|---|---|---|
      | `000d5950` | B | 38 | 43 | **88,4 %** | 0,00 | 0,13 | PASSE |
      | `00502e52` | B | 15 | 25 | **60,0 %** | 0,06 | 0,22 | **ÉCHOUE** |
      | `07aa428d` | B | 36 | 52 | **69,2 %** | 0,12 | 0,28 | **ÉCHOUE** |
      | `00ba2e1c` | A | 8 | 10 | **80,0 %** | 0,00 | 0,03 | PASSE |
      | `06dfe6d9` | A | 6 | 13 | **46,2 %** | 0,00 | 0,05 | **ÉCHOUE** |
      | `084a804d` | A | 4 | 4 | **100,0 %** | 0,00 | 0,00 | PASSE |

      **3 échecs sur 6 films.** Le seuil autorise **2 échecs sur 8**. Il est donc
      **arithmétiquement hors d'atteinte** : même en mesurant les deux films manquants et
      même s'ils passaient tous les deux, le compte serait de 5 succès sur 8, sous les 6
      exigés. **Les films restants n'ont pas été décodés — ils ne peuvent pas changer le
      verdict**, et dépenser du temps machine pour un résultat déjà déterminé n'aurait rien
      prouvé.
- [x] 0-bis.3 **SEUIL (3) DATATION (ÉLIMINATOIRE) — TENU, largement.** C'était le risque
      majeur : un tag qui ne se lèverait qu'au spawn daterait une DOTATION, et l'effet
      pulserait à chaque réapparition. Transitions `tag==1` **sur les vies de propulseur**,
      offset au début de la vie :

      | film | n | au spawn (< 2 s) | médiane | témoin tag 0 au spawn | témoin tag 2 au spawn |
      |---|---|---|---|---|---|
      | `000d5950` | 38 | **0,0 %** | 24,1 s | 5,6 % | 2,3 % |
      | `00502e52` | 15 | **0,0 %** | 19,5 s | 8,4 % | 1,5 % |
      | `07aa428d` | 36 | **8,3 %** | 24,2 s | 9,4 % | 3,1 % |
      | `00ba2e1c` | 8 | **0,0 %** | 15,0 s | — | — |
      | `06dfe6d9` | 7 | **0,0 %** | 37,6 s | — | — |
      | `084a804d` | 4 | **0,0 %** | 33,3 s | — | — |

      **Le tag 1 date un événement EN COURS DE VIE**, et il est même MOINS « au spawn » que
      les états génériques qui lui servent de témoins. Ce seuil-là, le canal le passe.
- [x] 0-bis.4 **SEUIL (4) CHARGE UTILE — NON MESURABLE PAR LE FLUX, et il faut le dire ainsi
      plutôt que de le déclarer tenu.** Le test prévu (si `R(2)` était une largeur fausse
      pour le tag 1, la marche AVAL casserait plus souvent sur ces records) s'est révélé
      **inapplicable : sur les six films, AUCUN record n'a de composant après `i59` dans son
      masque** — `i59` est toujours le DERNIER composant annoncé. Il n'y a donc rien en aval
      dont la casse pourrait trahir un décalage. Reste la preuve documentaire, inchangée : le
      déser reproduit `FUN_142f2679c` = `R(2)` plat, et la seule branche à corps est
      `tag==3` → `FUN_142f25e90`. **Conséquence à retenir : la largeur du tag 1 n'est pas
      falsifiable par le flux sur ce corpus.** Ce n'est pas un échec du seuil, c'est une
      impossibilité de mesure — consignée comme telle.

**GATE PHASE 0-BIS : le SEUIL (2) est TOMBÉ → PROPULSEUR CLASSÉ SUR MESURES**, avec la même
netteté que le répulseur, et sans renégocier le seuil (consigne explicite du superviseur).
`go vet` et `go test ./internal/analysis/filmdec/ ./internal/analysis/replay/` verts.

**CONSÉQUENCE PRODUIT, assumée : aucun son ni effet de propulseur ni de répulseur dans le
rejeu 2D — faute de CANAL, pas faute de fichier.** L'archive sonore de l'utilisateur n'a
jamais été sollicitée, et ne doit pas l'être tant qu'aucun canal ne date les usages.

### Journal de la Phase 0-bis — ce que ce négatif dit, et ce qu'il ne dit pas

**Le signal du tag 1 n'est pas une illusion, il est INSUFFISANT.** Trois films sur six
tiennent le contraste, dont l'un à 88,4 % avec zéro transition sur les vies de grappin ; le
seuil éliminatoire de datation est tenu partout. Ce qui manque, c'est la RÉGULARITÉ : sur
trois autres films, la masse tombe à 46-69 % et le témoin « sans identité » monte à 0,22-0,28.

**Une limite de méthode est identifiée — elle est notée, et elle n'excuse rien.** Le témoin
« vies sans identité `i48` » est structurellement gonflé : `i48` n'est transmis qu'environ une
fois par vie, si bien que **4,8 % à 60 % des vies n'ont aucune identité selon le film** (150
vies sur 250 pour `084a804d`, 53 sur 89 pour `07aa428d`). Une vie de propulseur non
identifiée compte donc dans le témoin et pénalise le contraste deux fois. Les deux films où
le témoin dépasse le seuil sont précisément ceux où les vies sans identité sont les plus
nombreuses. **Cela n'annule pas le verdict** — le seuil était posé d'avance en connaissance
de la méthode — mais toute reprise éventuelle devrait d'abord améliorer le taux
d'identification des vies, faute de quoi elle rejouerait le même biais et obtiendrait le
même résultat.

---

## PHASE 1 — port production + publication *(uniquement pour la ou les capacités où la Phase 0 a trouvé un canal)*

> **NON EXÉCUTÉE — tous les items ci-dessous sont `[!]`, avec CETTE justification unique :
> la Phase 0 n'a trouvé de canal pour AUCUNE des deux capacités du périmètre** (item 0.6,
> verdict du 2026-08-25). La condition d'entrée de cette phase — « uniquement pour la ou les
> capacités où la Phase 0 a trouvé un canal » — n'est remplie par aucune capacité. Aucune
> ligne de production n'a été écrite, et c'est la règle D5 qui l'impose, pas un manque de
> temps : **aucun effet simulé sans donnée mesurée.**
> Rien ici n'est périmé pour autant : le jour où un canal sort (voir les options de
> l'arbitrage), ces six items restent le chemin de portage, `SchemaVersion` à revérifier en
> premier (item 1.1, Décision D6) parce que la branche bouge vite.

- [!] 1.1 Revérifier `SchemaVersion` sur `feat/v75` fraîchement rebasée AVANT de coder —
      NE PAS supposer un numéro figé dans ce plan (Décision D6).
      Commande : `grep -n "const SchemaVersion" apps/go-api/internal/analysis/replay/document.go`
- [!] 1.2 Étendre `internal/analysis/replay/equipment_episodes.go` : nouvelle(s) constante(s)
      `EquipFamilyRepulsor` / `EquipFamilyThruster` (identifiants stables, pas des
      libellés — même règle que `EquipFamilyCamo`/`EquipFamilyOvershield`), nouvelle
      fonction `buildXEpisodes` sur le patron exact de `buildCamoEpisodes` /
      `buildOvershieldEpisodes`. **La machine à états `episodeAccum` est déjà générique —
      la RÉUTILISER, ne pas la dupliquer** (règle des ≤ 2 copies du dépôt). Si le signal
      retenu en Phase 0 est un ÉVÉNEMENT PONCTUEL plutôt qu'un intervalle (cas probable
      pour un usage court, contrairement à camo/surbouclier), documenter pourquoi
      `EquipmentEpisode{T0,T1}` convient quand même (T0==T1, ou une fenêtre de charge
      mesurée) plutôt que d'inventer un second type de donnée pour deux lignes de plus.
- [!] 1.3 `EquipmentCoverage` : ajouter les compteurs Lives/Episodes de la ou des nouvelles
      familles, même patron que `CamoLives`/`CamoEpisodes`.
- [!] 1.4 `apps/go-api/contracttest/replay_contract_test.go` : mettre à jour le compte gelé
      des champs de `ReplayDocument` (piège déjà noté au registre pour tout document
      SERVI — un oubli ici casse silencieusement la garde de non-régression).
- [!] 1.5 `make generate-types` (régénération des types web depuis `openapi.yaml` /
      le contrat d'artefact).
- [!] 1.6 i18n FR + EN : vérifier si un libellé NOUVEAU apparaît côté document (probablement
      aucun — les identifiants restent stables, le libellage vit déjà côté client comme
      documenté au registre pour `padEquipmentFamily`, patron à réutiliser si un libellé
      d'équipement doit s'afficher).

**GATE PHASE 1** : `cd apps/go-api && go test ./internal/analysis/replay/... ./contracttest/...`
vert ; `make check-types` vert ; artefact régénéré sur le ou les films témoins de la Phase 0
avec les nouveaux épisodes visibles au JSON produit.

---

## PHASE 2 — rendu web + sons *(uniquement pour la ou les capacités livrées en Phase 1)*

> **NON EXÉCUTÉE — tous les items ci-dessous sont `[!]`, même justification qu'en Phase 1 :
> aucune capacité n'a passé la Phase 0, donc aucune n'a été livrée en Phase 1.** Rien n'a été
> rendu, rien n'a été sonorisé, et **aucun fichier son n'a été téléchargé depuis l'archive de
> l'utilisateur** (item 2.2) : il n'y a rien à déclencher. Le pulse du marqueur (D3) et les
> `.wav` répulseur/propulseur (D4) restent des décisions valides — elles attendent une
> donnée, pas un arbitrage.

- [!] 2.1 **Rendu (Décision D3, proposition par défaut)** : pulse géométrique court sur le
      marqueur du joueur, patron du `fireMark` (~600 ms, même mécanique que le « ! » du
      tireur livré au lot score/épure du 2026-08-24) ou du tracé du grappin —
      PAS l'effet plein-fiche doré/vitreux (réservé aux états soutenus, incohérent avec un
      événement de moins d'une seconde). Icône : celle déjà publiée dans
      `replay_labels.toml` (`hud/Repulsor`, `hud/Thruster`) — aucune nouvelle icône à
      produire. **Couleurs : tokens sémantiques uniquement (skill `color-tokens`)** —
      aucune valeur hex ni classe Tailwind couleur en dur, même règle que le reste de
      `features/match-replay/`.
- [!] 2.2 **Son (Décision D4)** : fichiers depuis l'archive personnelle de l'utilisateur —
      chemins à obtenir avant cet item, comme pour le grappin. Normalisation identique au
      précédent : **-16 LUFS / -1 dBTP gain linéaire, pcm_s16le 48 kHz**. Déclenchement
      par le(s) même(s) canal(aux) que les épisodes publiés en Phase 1 (patron
      `doc.grappleLines[].t0`), catégorie « équipements » de `replaySound.ts`.
- [!] 2.3 Étendre le garde-rail de parité stem↔fichier (`replaySoundAssets.guard.test.ts`),
      même patron que l'extension faite pour le grappin.
- [!] 2.4 Si l'archive contient plusieurs gestes candidats par capacité : arbitrage par
      écoute utilisateur, règle du vote (`RECETTE_SONS_ARMES.md` §5) — ne pas choisir à la
      place de l'utilisateur.
- [!] 2.5 Bascule d'activation : réutiliser la bascule son globale existante (catégorie
      équipements, déjà réglée par défaut comme les autres sons d'équipement) — pas de
      nouveau réglage UI à créer sans raison.

**GATE PHASE 2** : `make test-web` et `make check-types` verts ; gate visuel ET sonore
utilisateur sur le film témoin (même protocole que tous les rendus précédents de ce
chantier — la session ne juge pas son propre rendu).

---

## EFFORT ESTIMÉ PAR PHASE

| phase | effort | pourquoi |
|---|---|---|
| Phase 0 (mesure) | **MOYEN** | 2-3 instruments de recherche neufs, mais réutilise des patrons de hook/marche déjà écrits 4 fois (i28, i5, i54, i59) ; aucune rétro-ingénierie nouvelle — la grammaire d'`i56` est déjà connue |
| Phase 1 (port prod) | **RAPIDE à MOYEN** | copie-adaptation quasi directe d'`equipment_episodes.go` (patron déjà écrit deux fois) ; dépend du nombre de capacités qui passent la Phase 0 (0, 1 ou 2) |
| Phase 2 (web + sons) | **RAPIDE** (rendu) **à MOYEN** (sons, si plusieurs gestes à arbitrer par écoute) | patron du grappin directement réutilisable ; le coût variable est humain (écoute), pas technique |

Un verdict Phase 0 entièrement négatif (aucun canal pour aucune des deux capacités) ramène
tout le lot à un effort MOYEN ponctuel (la Phase 0 seule) et zéro ligne de production —
c'est un résultat valide, pas un échec (Décision D5).

---

## CONTRAINTES MACHINE (D17) — rappel opérationnel pour la Phase 0

Reprise à l'identique de la discipline déjà appliquée par `registre_film/LOTD_PHASE0.md`
§5, pour la même classe d'instruments (décodage de film complet) :

- **Un film par processus** — `go test -c` puis un lancement par film, jamais un balayage
  multi-films dans le même process (le balayage de couverture de l'étape 2 a mesuré un
  coût NON linéaire en octets sur de gros films : 120 films en une exécution n'avaient pas
  terminé après 16 minutes).
- **Avant-plan**, jamais en tâche de fond pendant un autre build.
- **Plafond mémoire surveillé** : `Start-Process -PassThru`, lecture de
  `PeakWorkingSet64`, arrêt au-delà d'un plafond explicite (3 Go dans le précédent LOTD).
- **Coût calibré sur 2-3 films avant le corpus complet** — mesurer, pas supposer.
- **Jamais pendant un `go build`** — les deux processus se disputent la même mémoire.

---

## CE QUI PEUT FAIRE ÉCHOUER CE PLAN

- ~~L'hypothèse des 6 bits peut être fausse.~~ **ELLE L'EST** (étape 1, 2026-08-14).
- ~~L'épisode `i54` peut n'être qu'un mouvement.~~ **IL L'EST** (Phase D, 2026-08-16) — c'est
  précisément pourquoi la Phase 0 ne s'appuie plus sur `i54` (Décision D1).
- **`i56` peut ne pas atteindre une couverture utile même sous la nouvelle marche** — rien
  ne le garantit avant la mesure de l'item 0.2 ; c'est exactement pour ça que 0.2 porte son
  propre seuil d'arrêt.
- **Un canal peut exister mais ne pas être EXCLUSIF au rang cible** (le cas déjà rencontré
  pour `queue[2]` du camouflage, qui oscille partout et n'a pas passé le contrôle) — un
  candidat qui « bouge sur les bonnes vies » sans être ABSENT sur les autres ne suffit pas.
- **La table peut rester partielle** : 11 capacités existent, un match n'en montre que ce que
  les joueurs portent. C'est un trou de couverture, pas une dérive.
- **Le rendu court (Décision D3) peut sembler trop discret à l'usage** — c'est un jugement
  d'oreille/d'œil, pas une mesure ; le gate visuel/sonore utilisateur de la Phase 2 le
  tranchera, comme pour tous les rendus précédents de ce chantier.

---

## PROTOCOLE DE REPRISE

1. Lire ce fichier de haut en bas à partir de la section « ACTUALISATION DU 2026-08-24 » ;
   les cases cochées disent où en est le chantier (Phase 0 → 1 → 2, ordre strict).
2. `.ai/thought_log.md`, entrée la plus récente portant « équipement » ou « capacité ».
3. `.ai/V7.5/REGISTRE_REPORTS.md` pour les reports en cours liés à `i54`/`i56`/`i48`/
   « équipement actif ».
4. **Vérifier sur pièces avant de coder** : rouvrir le fichier et la ligne cible, le code a
   pu bouger — en particulier `SchemaVersion` (Décision D6) et l'état d'`i56` dans
   `consumeByName` (item 0.1), qui peuvent avoir changé entre l'écriture de ce plan et son
   exécution.

---

## MESURES PUBLIÉES — 2026-08-14 *(historique)*

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

### État actif (Phase D, PLAN_ETAT_ACTIF_EQUIPEMENT) — 12 films, 2026-08-16 *(la mesure qui remplace la précédente comme fondement de la Phase 0)*

    records delta biped (12 films)      2 519 242
    masque ∋ i54                           21 188   (21 187 lus, 1 illisible)
    flag1==1                               20 595
    épisodes/vie — vies MOBILITÉ (445 vies)   0,55   (244 épisodes)
    épisodes/vie — AUTRES RANGS (392 vies)    0,45   (177 épisodes)
    ratio mobilité/autres                      1,2   — AUTRES > MOBILITÉ sur 4 films/12
    témoin 0014603f (0 identité i48)         631 lectures flag1==1 quand même

## DÉCOUVERTES — notées, NON traitées (règle 7)

- **2026-08-25 — « masque dense » ne veut PAS dire « plus de 7 composants », et ce n'est pas
  un gisement.** Plusieurs documents (dont le journal de l'étape 3 de ce plan) présentent
  `bipedMaxMaskCnt = 7` comme une limite du détecteur qui laisserait échapper les records
  « à masque dense (> 7 composants) », faisant de toute mesure un PLANCHER. Vérifié sur
  pièces : le champ `maskCount` de l'en-tête fait **3 bits** (`offline_biped.go:334`), donc 7
  est le maximum REPRÉSENTABLE — un masque creux de 8 composants n'existe pas dans la
  grammaire. Le « masque dense » est l'AUTRE encodage (porte à 1 → `R(64)`, cf.
  `consumeMask` / FUN_1406d7610), et il pèse **201 records sur 171 979 (0,1 %)** sur
  `000d5950`. Corriger cette formulation partout où elle sert d'espoir de couverture.
- **2026-08-25 — `i56` suit le GRAPPIN, pas la capacité interrogée.** Ses chutes de charge se
  concentrent sur le rang 4 (famille A) et les rangs 20/21 (famille B), sur les six films.
  C'est un compteur de charges de l'équipement porté, pas un canal par capacité — et le
  grappin a déjà son canal dédié (`i59 tag==3`). Si une capacité à charges DEVAIT un jour se
  dater par `i56`, ce serait celle-là, et elle n'en a pas besoin.
- **2026-08-25 — `084a804d` mélange des rangs des deux familles de palette.** Le film rend
  {1, 4, 5, 6, 8, 9, 10, **19**, 23, **44**} : le 19 appartient à la plage famille B (19-22),
  et le **44** est hors de toute palette connue (1 lecture). Les autres films du corpus sont
  homogènes (famille A pure ou famille B pure). Non diagnostiqué — cela peut être une lecture
  fantôme (cf. l'avertissement réel/fantôme du 2026-07-26) ou un vrai mélange de palettes. À
  regarder avant toute conclusion fondée sur la palette de CE film.
- **2026-08-25 — `i59` tags 0 et 2 : le canal jamais interrogé, et il est GROS.** Mesuré
  pendant l'arbitrage (disponibilité seulement, aucun croisement — hors périmètre de la
  Phase 0) : sur `00ba2e1c`, `masque∋i59 = 3 274 · LUS 3 234` (**98,8 %**), tags
  `0:1572 · 1:10 · 2:1565 · 3:87`. C'est **12,7 fois plus de lectures qu'`i56`** sur le même
  film. `tag==3` est le grappin (livré) ; l'instrument existant compte tous les tags
  (`i59_tag3_test.go:71`) mais ne croise aux rangs QUE le tag 3 (ligne 72). Les tags 0 et 2
  n'ont jamais été confrontés à une identité. **Réserve à ne pas oublier** : ils pèsent ~97 %
  des lectures, ce qui peut signaler un état générique « au repos / en cours » — le défaut
  exact qui a tué `i57`. Option A de l'arbitrage, à ouvrir par le superviseur.
- **2026-08-25 — `i51` a désormais un hook, et zéro donnée dans les deltas.** `emp_timer.go`
  publie ce que le déser lisait déjà. Le composant n'est annoncé dans AUCUN des 1 427 763
  records delta mesurés. Si la durée d'immobilisation par EMP redevient un objectif, la
  donnée devra être cherchée ailleurs que dans le delta bipède (images-clés ? autre `ti` ?) —
  le hook est posé, il ne coûte rien à laisser en place, il ne rapporte rien en l'état.

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
  transmission manque. *(2026-08-24 : c'est désormais l'objet direct de la Phase 0, item 0.2.)*
- **2026-08-24 — le rang de propulseur famille A n'a pas de témoin de comptage cité dans ce
  plan** (contrairement à répulseur, mur, capteur, etc., listés au manifeste
  `equipment_objects` de `replay_labels.toml` avec leurs parts `carried`/`deployed`) : l'item
  0.3 doit l'établir plutôt que de supposer sa présence sur les films de famille A du corpus.
