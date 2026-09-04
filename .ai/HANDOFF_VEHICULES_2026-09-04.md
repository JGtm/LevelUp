# HANDOFF — Chantier VÉHICULES ET TOURELLES (v7.5)

> Écrit le 2026-09-04. Branche `wt/vehicules-tourelles`, worktree
> `C:\Users\Guillaume\Projects\LevelUp-wt-vehicules`. 21 commits depuis `feat/v75`.
> **JAMAIS POUSSÉE : aucun run CI n'a jamais tourné sur ce chantier.**
> Rapports détaillés : `.ai/V7.5/film_re/` (62 fichiers). Journal : `.ai/thought_log.md`.

## 1. La demande d'origine, et où elle en est

| Objectif utilisateur | État |
|---|---|
| 1. Décoder les films pour trouver les véhicules (Behemoth / Launch Site, Super Fiesta) | **FAIT**, en production |
| 2. Emplacements, spawns, cooldown, état (détruit ou non) | **FAIT sauf la destruction** — 7 candidates datées attendent une vérification en Theater |
| 3. Sons : déplacement, tirs du véhicule, tirs de tourelle | **Tirs FAITS** (9 armes reconstruites, 4 en attente d'écoute) · **Moteurs = CAPTURE EN JEU** (impossibilité prouvée) · Destructions reconstruites |
| 4. Images vue de dessus, teintables par équipe | **FAIT et validé par l'utilisateur**, en production |

Le chantier a débordé — volontairement — sur **l'intégration complète dans le rejeu 2D**, qui
n'était pas demandée à l'origine mais qui est la finalité réelle.

## 2. Ce qui tourne aujourd'hui dans le rejeu

Document `ReplayDocument` **schéma 31**, champ `vehicles[]`, route inchangée
(`GET /players/{slug}/matches/{id}/replay`). Le calque web dessine :

- les véhicules à leurs **proportions réelles** (18 sprites à 10 mm/px, ancrés sur la taille du
  pion : Mongoose ≈ 1,75 pion), **teintés à la couleur d'équipe** du conducteur (teinte
  `multiply`, les sprites sont des silhouettes à traits noirs) ;
- **orientés** par la direction de déplacement (dernier cap connu à l'arrêt) ;
- les **occupants** : pion + nom + traînée escamotés pendant l'épisode, noms empilés sur le
  véhicule (conducteur en premier), reprise indépendante à chaque sortie ;
- **un cône de visée par occupant** (conducteur, artilleur, passager), à sa couleur, à sa vraie
  visée, longueur portée par son élévation ;
- les **tirs en véhicule**, partant de leur monture réelle (plateau arrière du Warthog, canons de
  nez des Covenant), directionnels pour une arme fixe, non directionnels pour une tourelle ;
- toggle « Véhicules » FR/EN, décor non pilotable filtré, fantômes fusionnés.

**Effet d'explosion (plasma / conventionnelle)** : codé, testé, **inerte** — il attend que le Go
publie `end = "destroyed"` + `tEnd`. Zéro pixel changé tant que ce n'est pas le cas.

## 3. Les trois choses qui attendent l'UTILISATEUR

### 3.1 Vérifier des destructions dans le Theater (débloque le plus)

Le film porte la **vitalité** des véhicules (lisible depuis la correction Ghidra, cf. §5). Le
**palier d'épave existe** : 18 vies finissent sous 0,25 d'intégrité, durée médiane 2,6 s, max
6,2 s, et **aucune vie ne porte de palier bas long** — l'état bas est toujours bref et terminal.
Mais le témoin n'est pas un témoin d'intactes (un véhicule détruit **à vide** y tombe), donc les
faux positifs ne sont pas bornables et **rien n'est publié**. La vérité terrain manque, et elle
est à quelques minutes de visionnage.

**5 confirmations sur 7 rendent la publication immédiate** (l'effet UI est déjà là).

| Match (identifiant complet) | Carte | Date | Véhicule | Minute |
|---|---|---|---|---|
| `51d3ab9f-0c70-4c49-ae6d-06a1ed6c5765` | Launch Site | 21 janv. 2026 | Chopper | **7:19** |
| *(même match)* | | | Warthog | **7:35** |
| `e1bdb97f-5cb6-4ea9-b0ac-ad1a1573f049` | Behemoth | 17 févr. 2026 | Warthog | **2:30** |
| `b232e02d-6222-4914-ac12-5f64bf04d770` | Behemoth | 19 mars 2026 | Banshee | **6:09** |
| `21468645-be17-4020-a203-0b6b1e429cdd` | Behemoth | 17 févr. 2026 | Falcon (décor) | **6:01** |
| `4898d586-a097-4de7-9c8e-d85b01c5f884` | Behemoth | 19 févr. 2026 | châssis inconnu | **8:35** |
| `0d76e8f1-729a-4f7d-be33-32fb02f90b7d` | Behemoth | 13 oct. 2025 | Wasp | **6:51** |

Conversion validée sur repère indépendant (`originMs` recalculé retombe sur les 10 667 ms de
l'artefact). PIÈGE ÉVITÉ : les rapports internes datent en **horloge film** (décalage ~1 870 s) ;
ces minutes-ci sont en **temps de match**. Le Falcon est du décor filtré : il ne prouve que le
critère, il ne produira aucun effet.

### 3.2 Écouter les 4 tirs Covenant reconstruits

Artefact : **Tirs des véhicules — écoute comparée**
(`https://claude.ai/code/artifact/ca9d05dc-c1b8-4006-a8c5-d899cc4da0d7`), 136 clips, chaque arme
avec sa reconstruction Wwise à côté du brut du dossier utilisateur.
Déjà validés : Scorpion, Rockethog, Wasp, Gungoose, tourelle LMG. **À valider : Ghost, Banshee
(2 modes), Wraith, Chopper.** Leur validation déclenche le câblage des sons de tir dans le rejeu
(le moteur sonore existe déjà et joue une variante aléatoire par coup).
Trou connu : les **missiles du Wasp** n'ont pas encore de reconstruction (méthode V3F applicable).

### 3.3 Capturer les moteurs en jeu

**Piste des banques définitivement fermée** (voir §5). Protocole convenu : ~10 s de conduite à
vitesse moyenne par véhicule (musique et VO coupées), 2-3 s de boost pour Ghost/Wraith/Banshee,
une explosion par véhicule. N'importe quel format ; le découpage début/boucle/fin est fait ensuite.

### 3.4 Et : le feu vert pour pousser

21 commits jamais poussés, **CI jamais exécutée**. C'est le seul vrai risque en suspens du
chantier. `make gate-push` local n'a pas non plus été joué en clôture.

## 4. Où tester

L'app tourne (ou se relance) depuis **`C:\Users\Guillaume\Projects\LevelUp-wt-capture-rejeu`**,
qui détient `feat/v75` (mergée en avance rapide, en local uniquement) et **contient les données**.
Le worktree du chantier n'a pas de base joueur : ne pas essayer d'y lancer l'app.

```
cd C:\Users\Guillaume\Projects\LevelUp-wt-capture-rejeu
make dev
```
Puis `http://localhost:5173/t/halo_infinite/players/JGtm/matches/0d76e8f1/replay`
(ou `fccc61cd`). Les deux artefacts de démo (schéma 31) y sont déjà copiés.

Prérequis déjà réglés dans ce worktree : `npm install` (dépendance `mediabunny` manquante),
copie de `db_profiles.json` / `app_settings.json` / `.env.local` depuis le checkout principal,
copie de la base joueur `players/JGtm/stats.duckdb`.

## 5. Les acquis de rétro-ingénierie qui dépassent le chantier

1. **La table composant→désérialiseur est STATIQUE** (le dépôt la croyait dynamique, extraite à
   chaud). Chaîne : chaîne ASCII du nom → xref DATA → `vtable+0x08` → deser en `vtable[0x28]`,
   ou `[0x30]` si c'est le thunk `FUN_14076ce9c`. **Validée 6/6.** C'est la clé qui ouvre
   n'importe quel composant du film.
2. **`ti=40` porte les variantes `-dynamic-precision-`** des composants d'orientation
   (`FUN_140c5f7ec`, `FUN_140d87740`), là où le dépôt utilisait les désérialiseurs du bipède. Un
   correctif de 2026-07 avait réparé `ti=35` **en cassant `ti=40`** sans que rien ne le signale.
   Conséquence : la vitalité des véhicules est passée de bruit pur à lisible.
3. **Le décodeur bipède avait un point aveugle** : `ScanBipedRecords` exigeait un `i0` absolu, or
   un joueur embarqué émet des records de **visée sans position** (22 963 lectures sur un seul
   film). **Toute conclusion antérieure tirée d'une « absence dans le flux » est à relire.**
4. **Le domaine 1 des références d'événements = les UNITÉS** (bipèdes ET véhicules), 99,6-100 %.
   V6 avait cherché le véhicule en domaine 7 et l'y avait réfuté à raison.
5. **La tourelle est une entité `ti=40` à part** : muette (aucun record), slot = (slot véhicule
   − 1), et elle n'existe **que** pour les familles à siège d'artilleur. L'événement de sortie la
   nomme. Son orientation n'existe pas — c'est l'artilleur qui porte sa visée.
6. **Le « switch de régime moteur » est l'ESPACE DE L'AUDITEUR** (réverbération : extérieur
   ouvert/petit/moyen/grand, intérieur étroit/…). Les « moteurs » des lots V3B/V3C/V3E étaient
   des **sons d'armes sous différentes réverbérations** — d'où l'échec répété à l'oreille.
7. **Les véhicules sont des assemblages parent/enfant** : le châssis n'a pas d'arme, la tourelle
   est un `vehi` (ou `scen`) séparé, lié par nom de famille. Et **+X = AVANT pour tous les
   véhicules** (l'exception Scorpion n'existait pas).

## 6. Négatifs mesurés — ne pas re-creuser sans élément neuf

| Question | Verdict | Où |
|---|---|---|
| État « qui est à bord » sérialisé dans le film | **N'existe pas.** 6 endroits fouillés (états par joueur des images-clés, composants inconnus, entités-enfants, flux delta, bloc +89 bits, liste d'événements) | V5, V5B, V6 |
| Embarquements « cachés » dans la suite de la liste d'événements | **N'existent pas.** Un événement véhicule termine sa liste ; balayage de 296 595 paquets × 400 décalages : le réel est SOUS ses 4 témoins | V6 |
| Destruction datée par un type d'événement | **Aucun des 28 types.** 6 angles, contrôle positif validé (r=0,973) | V7 |
| Destruction par mort du conducteur | **Réfutée** : le conducteur sort vivant (3,8 % vs 21,3 % au témoin) | V3 |
| Destruction par `i4` → 0 | **Mauvais modèle** (le véhicule disparaît). Le bon signal est le palier d'épave → §3.1 | V9, V10 |
| Orientation par `i2` | **Réfutée même sous la grammaire corrigée** (médianes 47,9-122,7°) | V11 |
| Bouclier de véhicule `i5` | **0 record** sur 37 677 | V10 |
| Moteurs reconstructibles depuis les banques | **Non** (§5.6) | V3B/C/E, V3F |
| Orientation propre de la tourelle | **Aucun record émis** | V11 |

## 7. Registre des pistes ouvertes (chiffrées, non traitées)

1. **180 tirs orphelins d'arme de véhicule** hors épisode sur `0d76e8f1`, chacun nommant son
   tireur — probablement des artilleurs de tourelles. Manque : résoudre leur véhicule.
2. **`i11` dead-state est sous-instrumenté, pas réfuté** : l'ancre est aveugle au dead-state y
   compris sur le bipède (où les morts sont certaines). Verrou = désynchronisation de
   `frame_records.go` (2,4 % de couverture `ti=40` contre 24,5 % pour le bipède).
3. **La borne de fin des épisodes à fin ouverte est suspecte** : un épisode publié de 90,7 s ne
   porte que 2 lectures de visée alors que le slot en émet 5-46/s. La série de visée est le
   premier instrument capable de la contredire.
4. **Aucun épisode du corpus n'a de siège > 0** : les cônes multiples sur un même véhicule sont
   établis par construction et par les tests, **pas par l'image**.
5. **`dropper` des armes au sol NON DÉTERMINISTE** (`ground_weapon_rules.go:347` parcourt une map
   et rend le premier match) — hors chantier, mais c'est un vrai défaut.
6. **`TestReplayDocumentFieldCountIsFrozen` (contracttest) est rouge** depuis le schéma 29
   (46 champs contre 44 gelés). À statuer avant merge.
7. **Missiles du Wasp** : pas de reconstruction sonore (méthode V3F applicable).
8. Cooldown des véhicules : **~58 s résolu hors Super Fiesta** ; en SF les socles sont une mêlée,
   pas de cycle propre. Pas de catalogue de pads par carte (le `.mvar` n'a jamais été extrait).

## 8. Méthode qui a marché, à reconduire

- **Le gate écrit AVANT la mesure**, avec témoin (permutation, décalage temporel, population
  contrôle). Plusieurs lots ont été sauvés par leur témoin — et une « découverte » à 89 % de
  véhicules détruits s'est révélée être une lecture désynchronisée.
- **Faire pointer l'utilisateur sur planche-contact** plutôt que deviner : les trois échecs
  successifs sur les tourelles Warthog se sont réglés en une fois avec 27 candidats numérotés.
- **Vérifier la prémisse du mandat sur pièces** : deux agents ont réfuté une affirmation de leur
  propre consigne (l'espace d'identifiants des armes, la position dans le record de tir) — sans
  quoi le code livré n'aurait jamais fonctionné.
- **Un seul module de jeu (7 Go) en RAM à la fois**, GOCACHE dédié par agent, agents en
  avant-plan, fichiers disjoints entre agents concurrents.

## 9. État du dépôt

- Branche `wt/vehicules-tourelles`, 21 commits depuis `feat/v75`, **jamais poussée**.
- `feat/v75` (worktree `LevelUp-wt-capture-rejeu`) est **avancée en local** sur le même commit.
- Non commité volontairement : les **WAV de sons** (transitoires, ~280 Mo) et 2 planches de
  brouillon. Le manifeste `manifeste_v3.json` (rev12), lui, est versionné.
- Gates au dernier commit : gofmt/vet verts, `CGO_ENABLED=0 go test filmdec+replay` vert sans
  variable d'environnement, service en CGO=1, web typecheck/lint/5 725 tests verts.
- **CI jamais exécutée.**
