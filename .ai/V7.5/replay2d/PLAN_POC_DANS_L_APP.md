# PLAN — le POC entre dans l'app

> Écrit le 2026-07-28. Branche `feat/filmdec-continuation` (celle en cours, un seul sujet).
> Contrat d'exécution : skill `plan-execution`. Ordre strict, une étape close avant la
> suivante, aucun report d'une action faisable, chaque item statué à la clôture.
>
> Documents qui font foi : `../../SUIVI_REPLAY_2D.md` (avancement), `../../CAHIER_DES_CHARGES_POC.md`
> (ce que l'écran doit montrer), `../../PLAN_REJEU_2D_FIABILISATION.md` (étapes 1 à 6, closes).

---

## CE QUE CE PLAN FINIT

L'étape 6 du plan de fiabilisation a fait sortir le rejeu du POC : `features/match-replay`
sert la carte, les tirs, les lancers et la couverture. **Elle n'a pas porté l'écran.** Le POC
est un poste de travail à quatre colonnes ; la production en a une.

### L'ÉCART, MESURÉ — pas estimé

Relevé le 2026-07-28 sur l'artefact réel (`data/cache/replays/halo_infinite/000d5950.json`,
2,17 Mo) et sur le POC (`eb7b8af2`, bloc de données de 2,9 Mo).

| calque du POC | l'artefact le porte ? | la production le rend ? |
|---|---|---|
| fond de carte tramé (sol reconstruit) | oui — 10 223 emprises | **non** : 10 223 rectangles translucides |
| cône de visée | oui — 15 315 points sur 29 221 | **non** |
| bouclier au-dessus du marqueur | oui — 4 620 points | **non** |
| anneaux d'étage | oui — z sur tous les points | **non** (filtre d'étage seulement) |
| apparition / mort | oui — startFrame/endFrame | **non** |
| trajectoires de projectile | oui — 439 | **non** |
| armes portées | oui — 150 loadouts | **non** — le type TS ne porte même pas le champ |
| **identité des joueurs** | **NON — 0 nom, 0 équipe sur 99 traces** | non |
| **fil des éliminations** | **NON** | non |
| **inventaire (grenades, capacité, munitions)** | **NON** — vit dans `cmd/tmp_kfinv` | non |
| effets de tir par famille | dépend du fil | non |
| zones nommées, objectifs, dispositifs | **NON** | non |

**Le fait qui commande l'ordre des étapes** : les 99 traces de l'artefact portent `team: -1`
et aucun nom, alors que le pont en nomme 90. Le pont existe (`owners.go`, `lives.go`), son
résultat n'est simplement **pas écrit dans le document**. Tant qu'il ne l'est pas, ni le
roster, ni les colonnes d'équipe, ni le fil, ni les couleurs d'équipe ne peuvent exister.

### DEUX ÉCARTS DE TYPAGE, à corriger avant tout rendu

1. `ReplayPoint` (TS) ignore `h`, `sh`, `hp` — trois champs que l'artefact publie.
2. `ReplayProjectile` (TS) déclare `{slot, gen, pts}` ; l'artefact publie `{t0, p, rest}`.
   **Le type est faux**, il décrit une forme qui n'existe pas. Aucun code ne le lisait, d'où
   le silence.

---

## DÉCISIONS TRANCHÉES — ne pas re-litiger

1. **Le POC est la référence de l'ÉCRAN, pas des données.** Son bloc de données est antérieur
   à la fiabilisation (147 tirs, 27 lancers) ; la production en a 475 et 70. On porte la mise
   en scène, on garde les données de production.
2. **Aucune couleur en dur côté web** (règle 12 + skill `color-tokens`). Le POC emploie des
   `hsl()` littéraux : ils deviennent des tokens sémantiques résolus.
3. **Rien ne s'affiche sans lecture.** Un champ non décodé s'affiche comme lacune (pointillé),
   jamais comme une valeur par défaut. C'est la règle qui a fondé la suppression du vote.
4. **Ce qui est décodé hors ligne vit dans `internal/analysis/replay`**, pas dans un `cmd/tmp_*`.
   Le portage d'un décodeur éprouvé n'est pas une réécriture : on déplace, on teste, on cite.
5. **Un seul agent compile du Go à la fois** (cache de build Windows).
6. Les fichiers de structure figés n'ont **pas** d'emprise orientée (0 `poly` sur 10 223) : le
   fond est tramé depuis les boîtes alignées. L'emprise orientée exige les fichiers du jeu
   (`cmd/mapstruct-build`) — hors périmètre, et le POC la donnait déjà comme une option.

---

## ÉTAPES

### ÉTAPE 1 — Les types disent ce que l'artefact publie — CLOSE
- [x] 1.1 `ReplayPoint` : `h`, `sh`, `hp` ajoutés, chacun avec ce qu'il garantit et sa
      couverture mesurée (15 315 / 4 620 / 163 points sur 29 221).
- [x] 1.2 `ReplayProjectile` : la forme déclarée (`slot`, `gen`, `pts`) **n'existait pas**.
      Corrigée en `t0` / `p` / `rest`. Aucun code ne la lisait, d'où le silence.
- [x] 1.3 `ReplaySurface.poly` + `ReplayDocument.loadouts`.
- GATE : `make check-types` vert.

### ÉTAPE 2 — Le fond de carte devient un sol — CLOSE (revue visuelle différée)
- [x] 2.1 `mapFloor.ts` + `drawFloorLayer` : trame de 0,25 m, altitude la plus haute par
      cellule, plages horizontales, teinte étalonnée aux centiles 2/98, arêtes aux ruptures
      > 0,45 m. Peint UNE fois hors écran, recopié à chaque image.
- [x] 2.2 15 tests purs sur la trame (exclusions plafond / dalle / mobilier, étalonnage,
      emprise orientée, arêtes marche contre rampe).
- GATE : tests verts. **Revue visuelle DIFFÉRÉE — à la charge de l'utilisateur** (2026-07-28) :
  l'app exige une session et le pilotage navigateur n'était pas exploitable.

### ÉTAPE 3 — Les joueurs se lisent — CLOSE (revue visuelle différée)
- [x] 3.1 Anneaux d'étage, halo, liseré de lisibilité, traînée 7 s.
- [x] 3.2 Cône de visée avec l'âge de la mesure (maintien 5 s, pâlissement 62 %).
- [x] 3.3 Barre de bouclier avec l'âge (maintien 2 s) ; **le zéro est une valeur**, pas une
      absence — trait sous la piste vide.
- [x] 3.4 Anneau d'apparition (0,8 s), croix de mort (1,5 s).
- [x] 3.5 Trajectoires de projectile — le dernier point n'est PAS un impact.
- [x] Lecture avec âge (`heldReading` / `freshness`) sortie en logique pure + 12 tests.
- GATE : tests verts ; revue visuelle différée (même raison qu'à l'étape 2).

### ÉTAPE 4 — L'identité entre dans l'artefact — CLOSE
- [x] 4.1 `Roster` : xuid, index de film, **et gamertag**.
- [x] 4.2 `Track.XUID` renseigné depuis le pont — **90 traces sur 99**, 8 joueurs distincts.
- [x] 4.3 Source statuée. **DÉCOUVERTE : le film porte les gamertags lui-même** (32 octets
      UTF-16LE dans le chunk highlight, à côté du xuid). L'artefact reste donc HORS LIGNE et
      nomme les joueurs sans toucher à la base. L'ÉQUIPE, elle, n'existe que dans la base :
      elle se joint côté client par xuid, et `Track.Team` reste à -1 avec la raison écrite.
- [x] 4.4 `WeaponLabels` : 22/22 familles de loadout et 17/17 armes de tir nommées. Le tag
      brut reste à côté du libellé ; un identifiant hors catalogue garde son hexadécimal.
- **CORROBORATION NON CHERCHÉE** : le roster lu par cette voie reproduit **exactement** la
  bijection index → joueur que le chantier avait établie par deux chaînes sans pièce commune.
  Cela fait une **troisième** route indépendante vers le même résultat.
- GATE : `go test ./internal/analysis/replay/...` vert, artefact reconstruit et vérifié.

### ÉTAPE 6 — Les fiches joueur — CLOSE (revue visuelle différée)
- [x] 6.1 Colonnes d'équipe, K/D/A colorés (frags / morts / assistances), bouclier, état mort,
      compteur de réapparition **lu** (image de départ de la vie suivante), lacune assumée
      quand aucune vie ne suit.
- [x] 6.2 Rangée d'armes portées depuis `loadouts`, estompée par l'âge de la lecture.
- [x] 6.3 Jointure film ↔ base par xuid (`rosterLogic.ts`) + 19 tests.
- [x] 6.4 L'image courante est publiée à 150 ms vers React : les fiches suivent sans re-rendre
      le DOM à la cadence de l'écran.
- GATE : tests verts ; revue visuelle différée.

### ÉTAPE 5 — Le fil des éliminations  [Go puis web] — OUVERTE
- [ ] 5.1 `Kills` dans l'artefact : instant, victime, tueur si lisible, arme si lisible.
      **Ce que la lecture de l'étape 4 a appris** : le chunk highlight porte aussi les
      événements `kill` (avec leur auteur) et les `medal` (avec leur type). Le fil est donc
      décodable DU FILM SEUL — reste à mesurer l'appariement kill ↔ death, qui n'est pas fait.
- [ ] 5.2 Colonne du fil : arme à la place de la croix, horodatage, liseré d'équipe.
- GATE : le compte du fil égale le compte des morts décodées.

### ÉTAPE 7 — L'inventaire complet  [Go puis web]
- [ ] 7.1 Porter `cmd/tmp_kfinv` vers `internal/analysis/replay` avec ses tests.
- [ ] 7.2 Grenades portées, capacité, munitions, emplacement dégainé, fraîcheur.
- GATE : le relevé écran 8/8 du POC se reproduit sur la même image-clé.

### ÉTAPE 8 — Clôture
- [ ] 8.1 Effets de tir par famille (dépend de l'étape 5).
- [ ] 8.2 i18n FR+EN complet, `delivery-checklist`, thought_log, mise à jour du SUIVI.

---

## RÉPARTITION AVEC LE CHANTIER VOISIN — arbitrée par l'utilisateur le 2026-07-29

**Ce qui appartient au voisin (`filmdec-killweapon`, paquet `killsource`), et qui y évolue** :
le **FIL DES ÉLIMINATIONS** — qui a tué qui et comment, l'assistance et sa part de dégâts, le
kill par véhicule. C'est sa source de vérité. Le rejeu doit le **consommer**, jamais le
redécoder : deux décodeurs du même fait divergeraient.

**Ce qui appartient à ce chantier** : tout le reste — trajectoires, armes portées, inventaire,
projectiles, structure de carte, identité.

### Ce que cette répartition impose à l'architecture

L'utilisateur : *« faut que ce soit factorisé pour pouvoir brancher ma version qui évolue sans
soucis […] séparation en couches, des responsabilités, title agnostic »*. Traduit en règles
tenables ici :

1. **Trois couches, jamais mélangées** : le DÉCODAGE (film → données), l'ASSEMBLAGE (données →
   artefact figé), l'AFFICHAGE (artefact → écran). Elles se parlent par des types, pas par des
   appels croisés.
2. **Le fil des éliminations entre par une ENTRÉE DE DONNÉES**, au même titre que `Deaths`,
   `Loadouts` ou `Inventory` — une valeur passée à l'assemblage, produite ailleurs. L'adaptateur
   `killsource` → rejeu s'écrira quand le paquet sera atteignable, et lui seul connaîtra les
   deux formes.
3. **Le décodage propre au rejeu vit avec le rejeu** (`inventory_decode.go`, `deaths_source.go`),
   et ne touche `filmdec` que par sa surface publique. Une liste d'appels qui s'allonge est le
   signal qu'il faut un port explicite, pas une intrusion de plus.

**CE QUI N'EST PAS ÉCRIT, ET POURQUOI** : aucun type `Kill` n'est déclaré tant qu'aucun
producteur ne l'alimente. Un type publié que personne ne remplit est du code mort avec l'air
d'une fonctionnalité — la règle du dépôt l'interdit, et il ferait croire à un fil des
éliminations qui n'existe pas.

### Contrainte mesurée qui pèse sur le rapprochement

`feat/filmdec-killweapon` **n'a aucun ancêtre commun avec `main`** ; `feat/filmdec-continuation`
en a un (`811be64ec`). Une seule des deux branches peut donc être livrée. Par ailleurs les deux
ont fait diverger `filmdec` : 36 fichiers, +1 472 / −2 972, avec des SUPPRESSIONS de part et
d'autre — les sept fichiers dont le rejeu dépend (`fire_events`, `grenade_events`,
`keyframe_loadout`, `map_bounds`, `vitality`, `i0_layout`, `capture`) n'existent pas chez le
voisin. Le rapprochement est donc un chantier de décodeur, symétrique quel que soit son sens ;
seule la direction « vers la branche connectée à `main` » débouche sur une livraison.

---

## DÉCOUVERTES FAITES EN CHEMIN — à ne pas perdre

### LA CELLULE DE MUNITIONS *k* N'EST PAS L'ARME *k* — mesuré, et il faut le dire à l'écran

Le POC posait les cellules de munitions sous la rangée d'armes, dans le même ordre, en signalant
l'ambiguïté par un simple numéro. La correspondance n'avait jamais été **mesurée**. Elle l'est
maintenant, sur 300 appariements du film de référence (arme du loadout ↔ cellule de même rang) :

| armes | chargeur | jauge | aucune |
|---|---|---|---|
| S7 Sniper, Skewer, Disruptor, Mangler, BR75, Needler… (15 armes sur 22) | **100 %** | 0 | 0 |
| **Gravity Hammer** | 13 | **17** | 8 |
| **Energy Sword** | 3 | **5** | 3 |
| Stalker Rifle | 5 | 9 | 1 |
| Ravager | 3 | 3 | 2 |

**Le marteau et l'épée sont exactement les deux armes que le décodeur documente comme
n'émettant NI chargeur NI jauge** — et elles en portent dans la majorité des cas. La
correspondance tient donc pour 15 armes sur 22 et **échoue mesurablement** pour les autres.

Ce n'est pas une raison de masquer les munitions : elles sont LUES, pas inventées. C'est une
raison de ne pas affirmer à quelle arme elles appartiennent. L'écran affiche le **numéro
d'emplacement du record**, et l'infobulle porte la réserve avec son chiffre.

**Ce qui n'est pas tranché** : est-ce l'ORDRE des emplacements qui diffère de celui du loadout,
ou le parse qui dérive quand une arme n'émet rien ? Les deux expliqueraient la mesure. La
question appartient au décodage, pas à l'affichage, et elle reste ouverte.



### Deux tests du handler de rejeu étaient ROUGES depuis le commit précédent

`fe8912a13` (« étape 6 close ») a posé le garde « servi en local » **sans adapter les tests du
handler**. `httptest.NewRequest` émet depuis `192.0.2.1` — une adresse de documentation, donc
non locale — et le garde répondait 404 à `TestReplayHandler_OK` et `..._ServiceError`, qui
annonçaient un handler cassé. **Réparé** (les requêtes de test partent de la boucle locale) et
le BRANCHEMENT du garde gagne le test qui lui manquait : `TestReplayGate_*` éprouvait la règle,
rien ne vérifiait que la route l'applique.

### `ownersFromLives` : premier arrivé, premier servi — c'est un CHOIX, pas une lecture

Sur collision de slot, le commentaire dit « on ne tranche pas, on ne publie pas » ; le code
garde la PREMIÈRE identité et écarte la contradictoire. Un test l'exige explicitement, donc
c'est délibéré — mais c'est bien un arbitrage, dans un chantier qui les a par ailleurs
supprimés. **Non traité** (hors périmètre, 0 collision sur le film de référence) ; consigné
ici pour que la question soit posée sur le prochain film.

### Le cadrage a changé, et c'est volontaire

Avec un sol reconstruit, la scène se cadre sur la zone JOUÉE (`doc.bounds`) et non sur l'union
avec les props. La structure d'une carte couvre ±250 m là où les joueurs en parcourent 50 :
cadrer sur elle réduirait le terrain à un timbre. C'est aussi le cadrage du POC. Les props
Forge deviennent un REPLI, utilisé seulement quand la carte n'a pas de fichier de structure.

---

## PROTOCOLE DE REPRISE

1. Relire le contrat `plan-execution`.
2. Lire ce fichier, puis `../../SUIVI_REPLAY_2D.md`.
3. Reprendre à la **première case non statuée**.

**Statuts autorisés** : `[x]` fait et vérifié · `[~]` couvert ailleurs, avec la référence ·
`[!]` non traité, avec justification écrite.
