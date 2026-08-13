# PLAN — les capacités d'armure : les nommer toutes, puis les montrer actives

> Écrit le 2026-07-31. Contrat d'exécution : skill `plan-execution`.
> Deux chantiers distincts que ce plan tient ensemble parce qu'ils partagent leur donnée :
> **savoir QUELLE capacité** (table partielle) et **savoir QUAND elle est active** (`i57`).

---

## OÙ ON EN EST, EXACTEMENT

### Ce qui marche

L'index de capacité est lu à chaque image-clé, sur 132 lectures, et le contrôle terrain donne
**8/8 sur le nom de la capacité**. Le décodage est bon.

### Ce qui ne marche pas — et pourquoi c'est probablement le même défaut

| | |
|---|---|
| ce qu'on lit | l'index sur **3 bits** → 8 valeurs possibles |
| ce que le binaire décrit | **6 bits précédés d'une porte** → 64 valeurs |
| ce que le jeu contient | **11 capacités** |
| ce que notre table nomme | **4** — mur portatif, grappin, propulseur, capteur de menace |

**Trois bits ne peuvent pas coder onze capacités.** C'est l'explication la plus simple de notre
table partielle, et elle n'a jamais été testée.

Le contrôle croisé le disait déjà : la palette `sofd` donne mur = rang **2** et répulseur =
rang **6** ; notre table dit mur = **3** et capteur = **6**. **2 confirmés sur 4** — et le motif
est net : les deux confirmés s'appuient sur un **triplet** de joueurs, les deux contredits sur
une **observation unique**.

### L'état actif : rien n'est lu

Le composant `i57` est **absent du switch** du décodeur. L'ADDENDUM du 2026-07-26 le mesure à
**2 bits** sur 990 lectures. Le POC avait tenté d'exploiter `i57` bit 0 : il valait **1 sur 386
occurrences sur 386** — un interrupteur qui ne bascule jamais n'informe de rien, d'où le retrait
du badge d'état.

### Ce que l'écran doit montrer, une fois la donnée là

Le POC l'a dessiné (Notion 21.1) : **surbouclier** en encadré doré, **camouflage** en effet de
verre, **translocateur** en bordure animée du bleu électrique au jaune orangé.

---

## ÉTAPE 1 — RELIRE L'INDEX SUR 6 BITS  *(aucune capture nécessaire)*

C'est la mesure **déclarée prioritaire depuis le 2026-07-27 et jamais faite**. Le document est
explicite : « le problème des deux films est contourné sans nouvelle capture ».

- [ ] 1.1 Relire les **mêmes records** — ceux qui donnent aujourd'hui 4 index — en lisant
      **6 bits** au lieu de 3, après la porte décrite par le binaire.
- [ ] 1.2 **Le contrôle qui peut échouer, énoncé AVANT** : si la lecture 6 bits est la bonne,
      alors le **mur portatif doit ressortir au rang 2** et le **capteur de menace au rang 1**,
      conformément à la palette `sofd`. Si les rangs ne bougent pas, l'hypothèse est fausse et
      il faut chercher ailleurs.
- [ ] 1.3 Mesurer la **distribution des index** sur les deux films. Une lecture correcte doit
      donner des valeurs **dans [0, 11)** ; des valeurs éparpillées sur [0, 64) signeraient une
      lecture hors position.
- [ ] 1.4 Statuer : combien de capacités distinctes sur les deux films ? Le film de Fiesta
      devrait en montrer plus qu'un Slayer standard, où tout le monde a la même.

**GATE 1** : les rangs de la palette `sofd` sont reproduits, ou l'hypothèse est réfutée par écrit.

---

## ÉTAPE 2 — COMPLÉTER LA TABLE, SANS JAMAIS DEVINER

- [ ] 2.1 Étendre `abilityIndexLabels` aux index nouvellement observés — **et à eux seuls**.
      Un index vu mais non identifié garde son numéro : la règle du dépôt est qu'un nom
      approchant se lit comme une certitude.
- [ ] 2.2 **Sortir la table du code Go vers les mappings de titre**, bilingue. Aujourd'hui les
      noms sont **en français dans du Go**, ce qui interdit l'anglais autant que l'ajout d'un
      titre. Cible : `config/titles/halo_infinite/mappings/`, comme `weapon_names.toml`.
- [ ] 2.3 Croiser avec la capture d'un match à surbouclier et camouflage
      (cf. `SESSION_CAPTURE_AVANT_PC.md`) : ces deux équipements manquent à la table, et une
      capture les nommera par le relevé terrain.

**GATE 2** : la table couvre les capacités observées sur les films disponibles ; les autres sont
absentes plutôt que devinées ; aucun libellé n'est en dur dans du Go.

---

## ÉTAPE 3 — L'ÉTAT ACTIF (`i57`)

- [ ] 3.1 Brancher `i57` dans le switch du décodeur et publier ses **2 bits bruts**, **sans les
      interpréter**. Publier une valeur non interprétée est honnête ; publier une interprétation
      non mesurée ne l'est pas.
- [ ] 3.2 Mesurer la **distribution des 2 bits** : combien de fois chaque combinaison, et
      **est-ce qu'ils bougent** ? Le POC s'est cassé sur un bit constant — c'est le premier
      contrôle à refaire.
- [ ] 3.3 **Le témoin décisif** : croiser avec les événements déjà connus. Un joueur qui vient
      de lancer une grenade, de mourir, ou dont le bouclier remonte brutalement — l'état actif
      d'un surbouclier doit corréler avec quelque chose. Sans corrélation, on ne publie pas.
- [ ] 3.4 Confronter au **relevé terrain** de la capture Catalyst : « à telle seconde, ce joueur
      a pris le surbouclier ». C'est le seul contrôle qui ne partage aucune pièce avec le
      décodage.

**GATE 3** : les 2 bits varient, et leur variation s'explique par un fait indépendant. Sinon,
on documente l'échec et on ne montre rien.

---

## ÉTAPE 4 — L'AFFICHAGE

Rien de tout ceci ne s'affiche avant que l'étape 3 soit passée.

- [ ] 4.1 Le badge de capacité porte son état : actif / inactif / **non lu**. Trois états, et le
      troisième existe — il ne se confond avec aucun des deux autres.
- [ ] 4.2 Les trois rendus dessinés par le POC (surbouclier doré, camouflage vitreux,
      translocateur en bordure animée) — **sur la fiche joueur**, et pas sur la carte : c'est un
      état de joueur, pas un événement de terrain.
- [ ] 4.3 i18n FR + EN, tokens sémantiques, aucune couleur en dur.
- [ ] 4.4 Le compteur d'utilisations **reste absent** : il n'est pas localisé (36 006 positions
      testées, aucune ne reproduit le relevé). Le « ? » est universel, pas occasionnel.

**GATE 4** : revue visuelle, et chaque état affiché est adossé à une mesure citée.

---

## CE QUI PEUT FAIRE ÉCHOUER CE PLAN

- **L'hypothèse des 6 bits peut être fausse.** Elle est plausible et jamais testée ; l'étape 1
  est conçue pour la réfuter vite si elle l'est.
- **Les 2 bits d'`i57` peuvent être constants**, comme le bit 0 l'était. Dans ce cas le chantier
  s'arrête à l'étape 3 et on l'écrit — c'est une réponse, pas un échec.
- **La table peut rester partielle** même après tout cela : 11 capacités existent, un match n'en
  montre que ce que les joueurs portent. Compléter exigerait de balayer beaucoup de films.
