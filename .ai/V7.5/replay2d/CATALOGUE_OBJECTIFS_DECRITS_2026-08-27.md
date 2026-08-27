# CATALOGUE DES OBJECTIFS DÉCRITS PAR LA GRAMMAIRE DU LECTEUR

> Livrable RE — replay 2D v7.5. Daté 2026-08-27. Branche `feat/v75`.
> Portée : décrire, mode par mode et archétype ECS par archétype, la grammaire de
> désérialisation que le lecteur de film Halo Infinite applique aux composants d'objectif,
> l'état produit à HEAD, et la recommandation (PORTER / PORTER-APRÈS-TEST-CADRE /
> DOCUMENTER) — verdicts d'adversarial de vérification intégrés.
> Travail **LECTURE SEULE** : aucun fichier écrit, aucune compilation, aucun `go build/test/run`.
> Ghidra HaloInfinite.exe (image base `0x140000000`, 311103 fonctions) en lecture seule ;
> chaque fait porte une citation `fichier:ligne` OU une adresse Ghidra.

---

## 0. Résumé exécutif

Les composants d'objectif du film Halo Infinite se répartissent en **deux mondes de lecture**
qui décident, à eux seuls, ce qui est portable aujourd'hui :

1. **Le flux d'enregistrement** — chemin `apps/go-api/internal/analysis/filmdec/traverse.go`,
   qui FONCTIONNE en production. Les archétypes qui y vivent (ti=6 statborg, ti=9
   managed-player, ti=10 boundary, ti=12 navpoint, ti=23 zones, ti=30 tacmap-poi, ti=32
   areaofinterest, ti=42 object, ti=47 splash) sont **désérialisables maintenant**.
2. **Le corps d'image-clé** — en-tête 108 octets, boucle d'entité `FUN_142e2bfd0`, boucle de
   composant `FUN_142e2c690` (table 64 entrées, dispatch `vtable[0x28]`). C'est là que
   vivent **les 34 composants de l'archétype ti=11 `managed-objective`** — le suivi
   d'objectif du HUD. Ce corps d'image-clé **n'est pas encore reproduit de façon fiable** :
   0/34 dispatch ti=11 câblé, et le lecteur de corps d'image-clé plafonne à **0,51 %
   bit-exact** sur le bipède ti=35 (le seul archétype complexe déjà tenté). Les feuilles
   ti=11 sont **triviales et leurs adresses Ghidra sont prêtes**, mais leur gain n'arrive
   **qu'une fois le cadre 108b prouvé** — d'où le classement PORTER-APRÈS-TEST-CADRE.

**Trivial ≠ porté.** `ecs_table.tsv:268-301` marque encore les 34 feuilles ti=11
`non_porte` (0/34 dispatch), alors même que 5 d'entre elles sont des lectures R(n) simples
dont l'adresse Ghidra a été établie ce chantier. La grammaire est acquise ; le portage
attend le cadre.

**Résultat des vérifications adversariales de ce chantier** (décompilation + désassemblage
sur pièces, image base `0x140000000`) :

| Cible testée | Classe annoncée | Verdict de vérification | Conséquence |
|---|---|---|---|
| ti=12 i0 managed-navpoint-sub-type `FUN_1410e0cac` | PM (feuille triviale R(32)) | **CONFIRMÉ** — R(32) pur bit-exact, `FUN_14080dec4` largeur fixe 32b, aucun drapeau | **reste PORTER** (grammaire) — sémantique du sous-type NON établie |
| ti=6 i57 statborg-entry-index-and-type `FUN_1410be614` | PM (R(32)+R(8)) | **CONFIRMÉ** — 40 bits fixes, aucun gate, écritures +0x1ed0 / +0x1ecc | **reste PORTER** (déjà porté, enhancement de cadrage) |
| ti=47 i0 splash-message-static `FUN_141085d50` | PM (feuille triviale, « Drapeau capturé! ») | **RÉFUTÉ** — drapeaux d'encodage R(1) multiples, dispatch tagué R(3), boucles à compte variable, références de chaîne `string_id` | **PASSE de PORTER à DOCUMENTER** |
| ti=11 i3 object-reference `FUN_142ed5550` (+0x40) | PTC (R(32) inline) | **CONFIRMÉ** — R(32) inline pur, écriture unique `[RBX+0x40]`, aucun vec3 | **reste PTC** (dépend du cadre 108b) |

**Liste PORTER-MAINTENANT nette (après vérification)** : deux feuilles seulement, coût
quasi nul (grammaire déjà décodée et consommée dans `traverse.go`) :
- **ti=12 i0** managed-navpoint-sub-type (`traverse.go:490`) — sonder l'identité de marqueur.
- **ti=6 i57** statborg-entry-index-and-type (`traverse.go:456`) — brancher le cadrage typé
  des compteurs de score i0.

Le candidat splash-message (ti=47) **sort de la liste PORTER** : sa désérialisation n'est
pas une feuille triviale et le libellé humain exige une résolution `string_id` → chaîne
localisée dont la faisabilité hors-ligne reste ouverte.

**Le `[!]` qui compte.** Cinq campagnes (D4→D10) ont échoué à nommer LE PORTEUR du crâne
Oddball (gate 79,8/73,8/64,9 % pour 80 exigé ; porteur principal 0/3 ;
`HANDOFF_ODDBALL_PORTAGE_2026-08-27.md:58`). Or **ti=11 i3 object-reference nomme
DIRECTEMENT l'entité qui porte** (`FUN_142ed5550`, champ +0x40, confirmé trivial) — c'est
exactement le `[!]` que cinq campagnes n'ont pas su établir. **Priorité produit n°1 dès que
le cadre 108b est prouvé.** Idem drapeau libre CTF (piste réfutée à 75,6 %).

**Les `[!]` définitifs v7.5** restent définitifs : Total Control (attribution 38,2 %,
cardinal 0-27,8 %, 77 désignations simultanées ; `objective_roles.toml:187-188` **RETIRÉ**),
VIP (bit côté script, aucun sérialiseur film).

---

## 1. Méthode et sources

### 1.1 Sources primaires (vérifiées sur pièces ce chantier)

- **Table ECS de référence** : `apps/go-api/internal/analysis/filmdec/testdata/ecs_table.tsv`
  (1082 lignes ; colonnes `ti · archetype · i · component · level · status · deser_addr ·
  grammar · bits_typ · code_source · doc_field · product_use · meaning_fr · exploitable_fr
  · confidence · notes`). C'est le registre HEAD des composants, de leur statut de portage
  et de leur grammaire connue. Trois copies périmées existent sous `.claude/worktrees/*` —
  **ne pas les citer** ; seule la copie ci-dessus fait foi.
- **Binaire** : Ghidra sur `HaloInfinite.exe`, image base `0x140000000`. Chaîne du lecteur
  (assertion de la tâche, corroborée en partie — voir 1.3) :
  `FUN_1428e2a04 → FUN_1428e2a9c → FUN_142e2bfd0` (boucle d'entité ; registre d'archétypes
  `DAT_144e61d88` ; en-tête 108 octets) `→ FUN_1428e2b68 → FUN_142e2c690` (boucle de
  composant ; table 64 entrées, descripteurs `0x104` octets ; dispatch `(**(*p+0x28))(...)`
  = `vtable[0x28]` ; thunk `FUN_14076ce9c`). La boucle 64 entrées de `FUN_142e2c690`
  (`uVar14 < 0x40`) a été **re-confirmée indépendamment** lors de la vérification de ti=11 i3.
- **Port Go du flux d'enregistrement** : `apps/go-api/internal/analysis/filmdec/traverse.go`
  et les `components_batch*.go` / `components_managed_object.go` / `components_world.go` /
  `components_game_engine.go` — c'est le lecteur qui FONCTIONNE.
- **Config produit** : `config/titles/halo_infinite/mappings/objective_roles.toml`
  (rôles de zone, entrées de mode), `.../replay_labels.toml` (identités d'objet).
- **Handoff Oddball** : `.ai/V7.5/HANDOFF_ODDBALL_PORTAGE_2026-08-27.md` (5 campagnes,
  gate raté, signal spatial).

### 1.2 Réserve de source — document d'entrée ABSENT

Le document RE imposé par la tâche,
`.ai/V7.5/replay2d/RE_LECTEUR_IMAGE_CLE_SCOUT_2026-08-27.md`, **n'existe pas dans le dépôt**
(recherche par nom et par glob exhaustive, ce jour). Il n'est donc **pas citable**. Toute
la chaîne du lecteur et les adresses de feuilles ont été **re-vérifiées directement sous
Ghidra** (et sur `traverse.go` pour le port), indépendamment de ce document manquant. Les
verdicts de ce catalogue ne dépendent d'aucune assertion de ce fichier absent.

### 1.3 Réserve d'attribution — nommage des feuilles ti=11

La boucle de composant `FUN_142e2c690` **dispatche par NOM à l'exécution** (chaîne de
composant appariée), pas par un index statique. Nommer une feuille « ti=11 i3 » plutôt que
« ti=11 i12 » exige l'identité de classe (RTTI) et n'a **pas été re-dérivée
indépendamment** : l'attribution i-index repose sur l'assertion de la tâche. Elle est
**corroborée** feuille par feuille par la coïncidence exacte entre le champ de destination
annoncé et l'écriture observée (ex. ti=11 i3 → écriture unique `[RBX+0x40]`, champ +0x40
annoncé — concordance). Ce catalogue tient donc pour acquis : (a) la **grammaire** de
chaque feuille adressée (vérifiée bit-exact) ; (b) le **champ de destination** ; il porte en
réserve l'**étiquette i-index** exacte.

### 1.4 Méthode de vérification adversariale

Chaque cible dite « triviale » a été soumise à : décompilation + désassemblage ;
`get_function_callees` (feuille pure = aucun sous-read caché) ; `get_xrefs_to` (dispatch par
table DATA = vrai désérialiseur) ; contrôle des largeurs de bits dans **les deux** branches
de re-remplissage (une largeur constante dans `if` ET `else` = R(n) pur, pas de branche
data-dépendante) ; recherche de MUL/DIV/CVT/float (déquantification), de min/max, de
compteur RAM pilotant la largeur, de drapeau d'encodage. Une seule de ces marques présente
= feuille NON triviale → RÉFUTÉ.

### 1.5 Légende des recommandations

- **PM** — PORTER-MAINTENANT : feuille triviale, dans le flux d'enregistrement qui marche,
  orpheline/inexploitée, gain clair, **aucune dépendance au cadre image-clé ti=11**.
- **PTC** — PORTER-APRÈS-TEST-CADRE : feuille connue, mais dépend de reproduire le cadre
  108b / le corps d'image-clé ti=11, OU d'une validation de largeur de position quantifiée.
- **DOC** — DOCUMENTER-SEULEMENT : raison précise donnée (complexe / hors corpus / feuille
  non adressée / vivant côté script / décision produit).

### 1.6 Statuts produit « PORTÉ / PUBLIÉ »

Les colonnes « Statut produit HEAD » de la matrice sont reportées de l'état HEAD du chantier
objectifs (schémas de rejeu, `objective_stats.go`, `zone_states*.go`, `flag_carries.go`,
`build_objective_objects.go`, etc.). Les faits **contestés ou décisifs** ont été re-vérifiés
sur pièces ce chantier (RE_LECTEUR absent, ti=11 `non_porte`, ti=12 i0 trivial, ti=47
réfuté, Total Control retiré, Firefight non servi, chiffres Oddball). Les faits de grammaire
Ghidra et les quatre verdicts adversariaux sont des vérifications de première main.

---

## 2. La matrice complète — mode × archétype × grammaire × statut × recommandation

### Section A — Archétypes TRANSVERSES (décrivent TOUS les modes à objectif)

| Mode | Archétype / composant | Grammaire + preuve | Statut produit HEAD | Recommandation |
|---|---|---|---|---|
| TOUS | **ti=11 i3** managed-objective-object-reference | TRIVIAL — R(32) inline, `FUN_142ed5550`, écriture unique champ +0x40 (Ghidra **vérifié ce jour**, bit-exact) | NON PORTÉ — `ecs_table.tsv:271` `non_porte`, 0/34 dispatch | **PTC** — débloque LE PORTEUR (crâne Oddball, drapeau libre CTF). Ordre 1 dès cadre prouvé |
| TOUS | **ti=11 i14** managed-objective-state | TRIVIAL — R(3) via `FUN_1424d9a30` octet bas, `FUN_142ed5948`, champ +0x194 (adresse chantier) | NON PORTÉ — `ecs_table.tsv:282` | **PTC** — état contesté/neutre/tenu image par image, toutes zones. Ordre 2 |
| TOUS | **ti=11 i5** managed-objective-type | TRIVIAL — R(32) **nommé** `"objective-type"`, `FUN_1410fc4a4` = `FUN_14080dec4(br,"objective-type", *(p+0x10)+0x150)`, champ +0x150 | NON PORTÉ — `ecs_table.tsv:273` | **PTC** — type actif sans nom de variante. Ordre 3 |
| TOUS | **ti=11 i12 / i13** progress + required-progress | TRIVIAL — 2× R(32) inline, `FUN_142ed575c` +0x18c / `FUN_142ed5844` +0x190 | NON PORTÉ — `ecs_table.tsv:280-281` | **PTC** — barre de capture en fraction. Ordre 4 |
| TOUS | **ti=11 i2 / i9** formatted-text (primaire/secondaire) | Semi — R(1) présence [+R(32)+R(3) count+liste taguée], `consumeObjectiveFormattedText` | DÉSER ÉCRIT NON CÂBLÉ, **pas d'adresse EXE établie** — `ecs_table.tsv:270,277` (`deser sans adresse EXE`), `components_batch3.go:22` | **DOC** — libellé HUD ; recouper à un `FUN_` avant tout câblage |
| TOUS | **ti=11 i1** managed-objective-color (camp/propriétaire) | Feuille **NON adressée** en Ghidra | NON PORTÉ — `ecs_table.tsv:269` | **DOC** — RE de feuille d'abord ; utile Strongholds/KOTH (fiabilité), seule voie propriétaire hors désignateur |
| TOUS (bases) | **ti=11 i16-i31** sub-objective-entities (16 slots) | Feuille **NON adressée** en Ghidra | NON PORTÉ — `ecs_table.tsv:284-299` | **DOC** — RE de feuille d'abord ; SEULE piste identité de zone A/B/C, chemin de reprise Total Control |
| TOUS | **ti=11 i0/i4/i6/i7/i8/i10/i11/i15/i32/i33** (timers / interaction-filter / enabled / priority / message-type / is-new / is-only-one / parent / outro / forced-update) | Non adressées | NON PORTÉ — `ecs_table.tsv:268,272,274-279,283,300-301` | **DOC** — faible valeur objectif |
| TOUS | **ti=0** game-engine (état de mode) — i8 round-condition-flags R(10) `FUN_141132dc0` ; i2 current-state R(3) ; i3 finished R(1) ; i4 round-gate | TRIVIAL — feuilles R(n) simples (Ghidra) | **PORTÉ** — `ecs_table.tsv:24` (`FUN_141132dc0`, `components_game_engine.go:138`) | déjà porté — N/A |
| TOUS | **ti=6** statborg (courbe de score 2 équipes) — i56 round-outcomes 32×R(2) `FUN_142ed71a4` ; i57 entry-index-and-type R(32)+R(8) `FUN_1410be614` | TRIVIAL — getters natifs + R(n) | **PORTÉ**, exploité par `internal/analysis/objectiveevents` — `ecs_table.tsv:162-163` | **i57 = PM** (câbler le cadrage typé des compteurs i0) |
| TOUS | **ti=9** managed-player (équipe du joueur) — i0 team-designator R(4) `FUN_140f581e8` | TRIVIAL | **PORTÉ** — `ecs_table.tsv:228` (`traverse.go:478`) | déjà porté — N/A |

> Note de fond ti=6 : la matrice initiale attribuait `FUN_1410be614` à i56 ; la table
> canonique le donne à **i57** (`ecs_table.tsv:163`) et attribue i56 à `FUN_142ed71a4`
> (`ecs_table.tsv:162`). Valeurs corrigées ci-dessus.

### Section B — Par mode à objectif (archétype spécifique du mode)

| Mode | Archétype / composant | Grammaire + preuve | Statut produit HEAD | Recommandation |
|---|---|---|---|---|
| **CTF / Neutral Flag** | ti=42 i9 object-multiplayer-properties (id drapeau `0x2a392328`, family=flag) | TRIVIAL — mot 32b du record de création, `FUN_1407d4c94` (`ecs_table.tsv:923`) ; polarité OK à HEAD `components_batch7.go:32-38` (lit le bloc quand bit==0) | **PORTÉ** identité (`replay_labels.toml:623-627` ; `traverse.go:774`) | déjà porté (statique) — N/A |
| **CTF** | flagCarries (vie du drapeau : carried/dropped/home + porteur xuid) | Heuristique — events nommés statborg + pont morts, PAS ti=11 | **PUBLIÉ** schéma 14/15 (`document_objectives_live.go:87-148` ; `flag_carries.go:164-203`) | déjà publié — N/A ; **ti=11 i3+i14 le remplaceraient** (PTC) |
| **CTF** | piste OBJET drapeau LIBRE (family `flag`) | ti=42 i0 position `FUN_14076e29c` — 55 % positions fantômes | **[!] RÉFUTÉ** : contrôle 3 = 149/197 = **75,6 %** (< 90 % écrit d'avance ; témoin arme 12,8 %) ; `'flag'` exclue, seule `'ball'` publiable (`flag_objects.go:33-42` ; `build_objective_objects.go:41-53`) | **DOC** — non publiable ; débloqué seulement par ti=11 i3 (PTC) |
| **CTF** | API CaptureTheFlagStats (post-match) | — | **PUBLIÉ** (`objective_stats.go:87-100`) | déjà porté — N/A |
| **Strongholds / Bastion** | ti=13 masked-property tag4 (propriétaire) + appariement slot→zone par captures nommées | Semi-complexe (sous-champ) — `FUN_140ce593c` | **PUBLIÉ** schéma 16/18 : propriétaire 93,1 % / jauge 98,4 % (`zone_states.go:118-152` ; `replaybuild/zones.go:72`). Canal jauge delta = ti=12 i14 radial-progress `FUN_140fc8d14` (`ecs_table.tsv:316`, 74,7/93,1 %) | déjà publié — N/A |
| **Strongholds** | `'contested'` ; zones ownerUnpaired | familles disjointes | **[!]** contested RÉFUTÉ (question vide sur corpus) ; ownerUnpaired comptées non publiées | **PTC** — ti=11 i14 state donnerait le contesté fiable |
| **KOTH** | ti=13 tag5 désignateur (colline active) + tag4 slot voisin (propriétaire) | Semi-complexe — `FUN_140ce5554` t5 id constant | **PUBLIÉ** : colline active (schéma 18) ; propriétaire **88-89 %** (< seuil 90 % que le plan s'était fixé, décision utilisateur ; témoin 56 %) (`zone_states_hill.go:26-32,358-513` ; `document.go:173-185`) | déjà publié — N/A ; **ti=11 i14+i1 amélioreraient >89 %** (PTC) |
| **KOTH** | repli par RAMPES (film sans désignateur) | pas d'objet de mode → pas de slot voisin | **[!]** aucun propriétaire sur ce repli (`zone_states_hill.go:34-35,251-254`) | **PTC** — ti=11 i1 color est la seule voie hors désignateur |
| **KOTH + Strongholds** | API ZonesStats (post-match) | — | **PUBLIÉ** (`objective_stats.go:102-110`) | déjà porté — N/A |
| **Total Control** | rôle statique totalcontrol_zone | — | **RETIRÉ 2026-08-27** (`objective_roles.toml:187-188` en commentaire) — absence propre, plus aucune zone affichée | conforme — N/A |
| **Total Control** | ti=13 tag5 désignateur (état vivant) | Semi-complexe mais ne survit pas au BTB | **[!] DÉFINITIF v7.5** : attribution 38,2 % (seuil 80), cardinal-3-zones 0-27,8 % (KOTH positif 98,4-100 %), jusqu'à 77 désignations simultanées croissant avec le film = défaut d'ANCRAGE (`document.go:190-193` ; `objective_roles.toml:138-157`) | **DOC** — reprise = ancrage restreint à l'objet de mode ; contournement propre = ti=11 i16-31+i14 (feuille i16-31 non adressée) |
| **Oddball** | ti=42 i9 object-multiplayer-properties (id crâne `0x0017592C`, family=ball) | TRIVIAL — `FUN_1407d4c94` | **PUBLIÉ** vies LIBRES du crâne, schéma 21/contrat 39 (`build_objective_objects.go:53-97` ; `replay_labels.toml:629-656`) | déjà publié (crâne libre) — N/A |
| **Oddball** | LE PORTEUR du crâne (qui tient, quand) | pas de composant film fiable trouvé | **[!] FINAL** : 5 campagnes D4-D10, protocole commité avant mesure, toutes négatives ; signal SPATIAL établi (décalage 12 m → 0-3,3 % contre 64,9-79,8 %) mais gate 79,8/73,8/64,9 % < 80 et porteur principal 0/3 ; les longs portages se fragmentent (`HANDOFF_ODDBALL_PORTAGE_2026-08-27.md:39-41,58,63-66`) | **PTC** — **ti=11 i3 object-reference nomme DIRECTEMENT le porteur = exactement le `[!]` de 5 campagnes. Priorité produit n°1 dès cadre prouvé** |
| **Oddball** | API OddballStats (post-match) | — | **PUBLIÉ** (`objective_stats.go:112-120`) | déjà porté — N/A |
| **Stockpile** | ti=42 objet noyau (identité) + rôles stockpile_socket/navpoint (points) | Recette identité crâne/drapeau applicable si film | **[!]** vivant : `ObjectiveTypeOf` ignore Stockpile (`extract.go:129-143` rend `''`) → 0 event nommé, 0 oracle ; corpus 2 films | **PTC** — ti=11 i5+i14+i3 fourniraient un état là où aucun event nommé n'existe ; **DOC** en complément : mesure limitée par corpus (2 films) |
| **Extraction** | rôle extraction_zone (neutre, exclu de heldZoneRoles) | — | **[!]** vivant : `ObjectiveTypeOf` ignore Extraction (`extract.go:129-143`) → ni event ni oracle ni appariement ; corpus 2 films (`objective_roles.toml:100-103` ; `replaybuild/zones.go:71-74`) | **PTC** — ti=11 i5+i14+i3 idem Stockpile ; **DOC** : corpus marginal |
| **Extraction** | API ExtractionStats (post-match) | — | **PUBLIÉ** (`objective_stats.go:132-139`) | déjà porté — N/A |
| **Assaut (Bomb)** | ti=42 identité bombe ; ti=13 ancrage ; statborg comp 0 | statborg RÉPLIQUE les points de mode PAR JOUEUR (candidat 4/4) | **[!]** identité bombe (0 candidat 7/7, ancrage site indisponible) ; ti=13 au hasard structurel 8/8 arène ; MAIS statborg nomme le POSEUR (commits `cb3887215` / `c8a107339`) | **PTC** — ti=11 i5+i14+i3 donneraient l'état ; le poseur est déjà atteignable via statborg (déjà porté côté score) |
| **Stockpile** | API StockpileStats (post-match) | — | **PUBLIÉ** (`objective_stats.go:122-130`) | déjà porté — N/A |
| **VIP** | bit VIP (côté SCRIPT, `Player:IsVIP()`) | pas de sérialiseur ; `ApplyVIPPlayerFX` absent de l'EXE | **[!] DÉFINITIF** : inatteignable par les deux voies du film (delta réfuté, image-clé bloquée) (plan §2.10:175-203) | **DOC** — pas de sérialiseur ; ti=11 ne débloque PAS forcément le VIP (vivant côté script) |
| **VIP** | API VipStats (post-match) | — | **PUBLIÉ** (`objective_stats.go:141-150`) | déjà porté (post-match) — N/A |
| **Land Grab** | classé `'zone'` par `ObjectiveTypeOf` (`extract.go:134`) mais AUCUN rôle servi | hashs landgrab_zone dans les cartes sans rôle | **NON SERVI** — ni statique ni vivant (plan §2.8) | **DOC** — décision produit + ajout rôle décodeur mapvar + entrée de table ; hors périmètre v7.5 |
| **Firefight (PvE)** | rôle firefight_objective au catalogue, servi par AUCUN mode | 5 volumes/carte, libellé de manche activateur inconnu | **NON SERVI** (`objective_roles.toml:66-71`) | **DOC** — libellé de manche non établi ; le servir « au cas où » afficherait 5 zones sans savoir laquelle est active |
| **Elimination** | bloc EliminationStats | — | **NON IMPLÉMENTÉ** — aucun payload réel en base (`objective_stats.go:36-38`) | **DOC** — aucun payload en base |
| **Infection** | bloc InfectionStats | — | **NON IMPLÉMENTÉ** — aucun payload réel en base (`objective_stats.go:36-38`) | **DOC** — aucun payload en base |

### Section C — Archétypes ORPHELINS (grammaire lisible, décodés, aucun mode branché)

| Archétype / composant | Grammaire + preuve | Statut produit HEAD | Recommandation |
|---|---|---|---|
| **ti=12 i0** managed-navpoint-sub-type | TRIVIAL — R(32), `FUN_1410e0cac`, déjà lu `traverse.go:490` ; **CONFIRMÉ bit-exact ce jour** (`FUN_14080dec4` largeur fixe 32b, aucun drapeau) | **PORTÉ (grammaire), JAMAIS SONDÉ, aucun mode** — `ecs_table.tsv:302` | **PM ordre 1** — meilleur candidat RÉEL pour « ce marqueur désigne le drapeau bleu / la zone » en contournant le mur ti=11 ; coût quasi nul. **Réserve : sémantique du sous-type NON établie** (valeur lue et rangée, jamais décodée en identité — à échantillonner sur films réels) |
| **ti=47 i0** splash-message-static | **RÉFUTÉ comme trivial** — `FUN_141085d50` : drapeaux R(1) multiples (jeu-ref, corps, custom-header, final), dispatch tagué R(3) `FUN_1407f0ebc`, boucles imbriquées R(3)/R(2), références de chaîne `"message-static-realization-str"` / `string_id`. Seul champ trivial = R(24) @+0x28, **sens INCONNU**, déjà publié | **PORTÉ mais JETÉ, aucun mode** — `ecs_table.tsv:1043` (`porte`) ; port Go `components_batch7.go:150-212` | **DOC** (déplacé de PM) — le libellé humain vit dans une chaîne conditionnelle exigeant une résolution `string_id` → localisé, faisabilité hors-ligne ouverte. Câbler une frise d'événements datés ≠ portage de feuille triviale |
| **ti=23 i0-31** zones (selectable-zone-data, 32 slots) | Semi-complexe — R(32)+POSITION+R(1)[+R(5)], `FUN_142ed6cec` | **DÉSER ÉCRIT NON CÂBLÉ** — `ecs_table.tsv:465-496` (`deser_non_cable`), `components_world.go:106` | **PTC** — SEUL archétype de zone objectif restant à câbler ; la POSITION quantifiée exige la même validation de largeur que le bipède/ti=13 → câbler puis mesurer. Ordre PTC 5 |
| **ti=30 i0** tacmap-poiicon (icône+texte+POSITION) | Lisible mais **contient un vec3 quantifié** — `FUN_142ed8418`, `R(32) icône + … + vec3 quantifié + …` (~230 bits), `traverse.go:517` | **PORTÉ mais JETÉ, aucun mode** — `ecs_table.tsv:656` | **DOC** — 2e voie pour poser des marqueurs d'objectif ; à instrumenter si un mode l'exige |
| **ti=32 i0** tacmap-areaofinterest | Lisible, `FUN_142ed3c50`, `traverse.go:350` | **PORTÉ, aucun mode** — `ecs_table.tsv:670` | **DOC** — « zone d'intérêt déclarée sur la carte » ; comparer aux zones containment |
| **ti=10 i1** managed-object-boundary-color (RGBA 8b) | PORTÉ — `FUN_142ed52b4`, 4×R(8) dequant [0,1], 10 347 annonces/10 films | **PORTÉ, non câblé à un mode** — `ecs_table.tsv:239` (`components_managed_object.go:93`) | **DOC** — rendu « teinter une zone par propriétaire » ; 1er octet à 4 niveaux (55/119/183/247) non expliqué |
| **ti=6 i56** statborg round-outcomes 32×R(2) `FUN_142ed71a4` ; **ti=0 i8** round-condition-flags R(10) `FUN_141132dc0` | TRIVIAL | **PORTÉ** — `ecs_table.tsv:162,24` | **DOC** / enhancement — issue de manche + POURQUOI une manche finit (objective-adjacent), déjà lisibles |
| **ti=35** biped-spartan default-state (contraste, NON objectif) | COMPLEXE — vec3 quantifié aux largeurs de carte, `consumeBipedDefaultState` ; plafond bit-exact **0,51 %** après 5 variables (borne d'arrêt R7-e) | référence de complexité | **DOC** — étalon du cadre image-clé COMPLEXE ; c'est ce plafond qui gèle l'accessibilité de ti=11 |

### Note de méthode sur le cadre — pourquoi ti=11 = PTC et non PM

Les feuilles ti=12 / ti=23 / ti=30 / ti=32 / ti=47 / ti=10 vivent dans le **flux
d'enregistrement** (chemin `traverse.go` qui FONCTIONNE) → portables **maintenant** (sous
réserve, pour ti=23/30, de la validation de largeur des positions quantifiées). Les 34
composants ti=11 vivent dans le **corps d'image-clé** (en-tête 108b, boucle d'entité
`FUN_142e2bfd0`) qui n'est **pas encore reproduit de façon fiable** : 0/34 dispatch ti=11,
et le lecteur de corps d'image-clé plafonne à 0,51 % bit-exact sur le bipède ti=35 (R7-e).
Les 5 feuilles ti=11 triviales ont leurs adresses prêtes, mais **le gain n'arrive qu'une
fois le cadre 108b prouvé** — d'où PORTER-APRÈS-TEST-CADRE, pas PORTER-MAINTENANT.

---

## 3. Archétypes À PORTER — ordre et coût estimé

### 3.1 PORTER-MAINTENANT (PM) — flux d'enregistrement, aucune dépendance au cadre ti=11

| Ordre | Cible | Grammaire (vérifiée) | Où | Coût | Gain |
|---|---|---|---|---|---|
| **PM 1** | ti=12 i0 managed-navpoint-sub-type | R(32) pur — `FUN_1410e0cac` → `FUN_14080dec4` largeur fixe 32b (**CONFIRMÉ bit-exact**) | `traverse.go:490` consomme déjà les 32 bits | **Quasi nul** (déjà décodé & consommé) pour la déser ; **échantillonnage sur films réels** pour la sémantique | Identité de marqueur d'objectif (drapeau bleu / zone) en contournant le mur ti=11. **RÉSERVE** : la valeur 32b → identité n'est PAS un fait RE, c'est une hypothèse à valider |
| **PM 2** | ti=6 i57 statborg-entry-index-and-type | R(32)+R(8) = 40 bits fixes, aucun gate — `FUN_1410be614`, écritures +0x1ed0 (u32) / +0x1ecc (u8) (**CONFIRMÉ**) | `traverse.go:456` | **Quasi nul** (déjà porté) | Cadrage typé des compteurs de score i0 : sans i57, on ne sait pas quel compteur on lit. Enhancement du parser statborg |

> **Retiré de PM** : ti=47 i0 splash-message-static — **RÉFUTÉ comme feuille triviale**
> (voir §2 Section C et §4). Le portage de la déser existe mais ne délivre pas le libellé
> d'événement sans une RE sémantique neuve (résolution `string_id` → chaîne localisée).

### 3.2 PORTER-APRÈS-TEST-CADRE (PTC) — gain réel, bloqué par le cadre 108b / la position quantifiée

L'ordre est le classement de valeur produit. **Le préalable commun aux ordres 1-4 est de
prouver le cadre du corps d'image-clé ti=11** (en-tête 108b, `FUN_142e2bfd0` /
`FUN_142e2c690`). Une fois ce cadre tenu, les feuilles suivantes sont des R(n) triviaux dont
les adresses sont prêtes — coût de portage quasi nul par feuille, coût réel = **prouver le
cadre**.

| Ordre | Cible | Grammaire | Champ | Débloque |
|---|---|---|---|---|
| **PTC 1** | ti=11 i3 object-reference | R(32) inline (**CONFIRMÉ ce jour**) | +0x40 | **LE PORTEUR** — crâne Oddball (le `[!]` de 5 campagnes), drapeau libre CTF. Priorité produit n°1 |
| **PTC 2** | ti=11 i14 state | R(3) (`FUN_1424d9a30` octet bas) | +0x194 | État contesté/neutre/tenu image par image, toutes zones (résout Strongholds `contested` réfuté, KOTH >89 %) |
| **PTC 3** | ti=11 i5 type | R(32) nommé `"objective-type"` | +0x150 | Type d'objectif actif — état vivant Extraction/Assaut/Stockpile là où aucun event nommé n'existe (`extract.go:129-143`) |
| **PTC 4** | ti=11 i12 / i13 progress + required | 2× R(32) inline | +0x18c / +0x190 | Barre de capture en fraction |
| **PTC 5** | ti=23 selectable-zone-data | R(32)+POSITION+R(1)[+R(5)], `FUN_142ed6cec` | — | Seul archétype de zone objectif restant à câbler. **Préalable distinct** : valider la largeur de la position quantifiée (même cadre que le bipède ti=35 / ti=13 t3), pas le cadre 108b — câbler puis mesurer |

Coût estimé PTC 1-4 : **dominé par un chantier unique de décodage** (prouver le corps
d'image-clé 108b) ; une fois tenu, chaque feuille ≈ 1 fonction R(n) à porter, adresse
connue. PTC 5 est indépendant (flux d'enregistrement) mais gardé par la validation de
largeur de position quantifiée.

---

## 4. Archétypes DOCUMENTÉS-SEULEMENT — raison précise

### 4.1 Réfuté comme trivial (déplacé de PORTER)

- **ti=47 i0 splash-message-static** (`FUN_141085d50`) — **RÉFUTÉ**. La désérialisation
  porte des drapeaux d'encodage R(1) multiples (jeu-ref présent, corps présent,
  custom-header présent, drapeau final), un dispatch tagué R(3) (`FUN_1407f0ebc` : union
  taguée handle/`string_id`/…), des boucles imbriquées à longueur variable (R(3)/R(2)/count
  refElem) et des références de chaîne nommées (`"message-static-realization-str"`,
  `"message-static-custom-header"`). Le seul champ trivial (R(24) @+0x28) est déjà porté et
  publié, mais son **sens est inconnu** — il ne porte PAS « Drapeau capturé! / Mort subite ».
  Le contenu sémantique exige une **résolution `string_id` → libellé localisé** (table de
  chaînes externe, absente du corpus offline). Câbler une frise d'événements datés est une
  **RE sémantique neuve**, pas un portage de feuille. Port Go fidèle à la structure :
  `components_batch7.go:150-212`.

### 4.2 Feuille non adressée en Ghidra (RE de feuille d'abord)

- **ti=11 i1 managed-objective-color** (`ecs_table.tsv:269`) — seule voie propriétaire hors
  désignateur pour le repli KOTH par rampes ; RE de feuille requise avant tout câblage.
- **ti=11 i16-i31 sub-objective-entities** (16 slots, `ecs_table.tsv:284-299`) — SEULE piste
  d'identité de zone A/B/C, chemin de reprise Total Control (contourne l'ancrage ti=13
  cassé) ; RE de feuille requise.
- **ti=11 i2 / i9 formatted-text** — déser écrit mais **pas d'adresse EXE établie**
  (`components_batch3.go:22`, `ecs_table.tsv:270,277`) ; recouper à un `FUN_` avant câblage ;
  nommage HUD gratuit.

### 4.3 Complexe (non portable en R(n) simple)

- **ti=35 biped-spartan** — vec3 quantifié aux largeurs de carte, `consumeBipedDefaultState` ;
  plafond 0,51 % bit-exact après 5 variables (R7-e). Étalon du cadre image-clé COMPLEXE qui
  gèle l'accessibilité de ti=11.
- **ti=30 tacmap-poiicon** — déjà porté mais contient un vec3 quantifié
  (`FUN_142ed8418`, `ecs_table.tsv:656`) ; 2e voie de marqueurs, à instrumenter si un mode
  l'exige.

### 4.4 Vivant côté script / inatteignable par le film

- **VIP** — bit côté SCRIPT (`Player:IsVIP()`) sans sérialiseur ; `ApplyVIPPlayerFX` absent
  de l'EXE ; **inatteignable par les deux voies du film** (delta réfuté, image-clé bloquée) ;
  ti=11 ne le débloque pas forcément (plan §2.10:175-203).

### 4.5 `[!]` définitif — reprise = chantier de décodage, pas de config

- **Total Control (vivant)** — ti=13 tag5 ne survit pas au BTB (attribution 38,2 %, cardinal
  0-27,8 %, 77 désignations simultanées) ; `[!]` DÉFINITIF v7.5 ; rôle statique **RETIRÉ**
  (`objective_roles.toml:187-188`). Reprise = ancrage restreint à l'objet de mode, OU ti=11
  i16-31+i14 (feuille i16-31 non adressée).

### 4.6 Hors corpus / non servi / décision produit

- **Stockpile + Extraction (vivant)** — `ObjectiveTypeOf` les ignore (`extract.go:129-143`) ;
  même une feuille ti=11 ne serait mesurable que sur un **corpus de 2 films** (limite de
  mesure, pas de décodage). Corpus marginal.
- **Land Grab** — classé `'zone'` par `ObjectiveTypeOf` (`extract.go:134`) mais AUCUN rôle
  servi (hashs landgrab_zone sans rôle). Décision produit + ajout rôle mapvar + entrée de
  table ; **hors périmètre v7.5**.
- **Firefight (PvE)** — rôle `firefight_objective` au catalogue servi par aucun mode
  (`objective_roles.toml:66-71`) ; libellé de manche activateur inconnu ; le servir
  afficherait 5 zones sans savoir laquelle est active.
- **Elimination + Infection** — blocs `EliminationStats` / `InfectionStats` NON IMPLÉMENTÉS,
  aucun payload réel en base (`objective_stats.go:36-38`).

### 4.7 Faible valeur / rendu sans mode

- **ti=10 boundary-color RGBA** (teinter une zone par propriétaire ; 1er octet 4 niveaux
  inexpliqué), **ti=32 tacmap-areaofinterest** (zone d'intérêt carte) — grammaire lisible,
  à instrumenter si un mode l'exige.
- **27 autres composants ti=11** (timers i0, interaction-filter i4, enabled i6, priority i7,
  message-type i8, is-new i10, is-only-one i11, parent i15, outro i32, forced-update i33…) —
  non portés, faible valeur objectif.

---

## 5. Archétypes orphelins — synthèse (grammaire lisible, aucun mode branché)

Récapitulatif des archétypes dont la grammaire est **déjà décodée** (souvent déjà consommée
dans `traverse.go`) mais qu'**aucun mode n'exploite** — le gisement de valeur immédiate :

| Archétype | Grammaire | État | Verdict |
|---|---|---|---|
| **ti=12 i0** navpoint-sub-type | R(32) — CONFIRMÉ trivial | consommé, jamais sondé (`traverse.go:490`) | **PM 1** — sonder l'identité de marqueur (sémantique à valider) |
| **ti=6 i57** statborg entry-index-and-type | R(32)+R(8) — CONFIRMÉ trivial | porté, jeté (`traverse.go:456`) | **PM 2** — cadrage typé des compteurs i0 |
| **ti=47 i0** splash-message-static | complexe (RÉFUTÉ trivial) | porté (R(24) seul, sens inconnu) | **DOC** — RE sémantique `string_id` requise |
| **ti=23** selectable-zone-data | R(32)+POSITION+R(1)[+R(5)] | déser écrit non câblé | **PTC 5** — valider la position quantifiée |
| **ti=30 i0** tacmap-poiicon | icône+texte+vec3 quantifié | porté, jeté (`traverse.go:517`) | **DOC** — 2e voie marqueurs |
| **ti=32 i0** tacmap-areaofinterest | lisible | porté, aucun mode (`traverse.go:350`) | **DOC** — zone d'intérêt carte |
| **ti=10 i1** boundary-color | 4×R(8) dequant [0,1] RGBA | porté, non câblé | **DOC** — teinter par propriétaire |
| **ti=6 i56** round-outcomes | 32×R(2) | porté | **DOC** — issue de manche |
| **ti=12 i14** navpoint-radial-progress | R(8) dequant [-1,1] | porté ET exploité (jauge Strongholds delta) | déjà exploité — N/A |

---

## 6. Texte prêt pour `.ai/thought_log.md`

```
[2026-08-27] Catalogue des objectifs décrits par la grammaire du lecteur (replay 2D v7.5) — Complété
Décision technique : cataloguer, mode par mode et archétype ECS par archétype, la grammaire
de désérialisation du lecteur de film (chaîne FUN_1428e2a04 -> FUN_142e2bfd0 boucle entité,
en-tête 108b -> FUN_142e2c690 boucle composant, table 64 entrées, dispatch vtable[0x28]),
avec statut produit HEAD et recommandation PORTER / PORTER-APRES-TEST-CADRE / DOCUMENTER.
Travail LECTURE SEULE (aucune écriture, aucune compilation). Sources : ecs_table.tsv (1082 l,
canonique), Ghidra HaloInfinite.exe (image base 0x140000000), traverse.go, objective_roles.toml,
HANDOFF_ODDBALL_PORTAGE_2026-08-27.md. Le doc d'entrée imposé
(.ai/V7.5/replay2d/RE_LECTEUR_IMAGE_CLE_SCOUT_2026-08-27.md) est ABSENT du dépôt : chaîne et
feuilles re-vérifiées directement sous Ghidra.
Résultats observés : deux mondes de lecture. Flux d'enregistrement (traverse.go, marche) =
portable maintenant ; corps d'image-clé ti=11 (managed-objective, 34 composants) = non
reproduit de façon fiable (0/34 dispatch, ecs_table.tsv:268-301 non_porte ; plafond 0,51 %
bit-exact sur le bipède ti=35). Les 5 feuilles ti=11 utiles (i3 object-reference +0x40, i14
state +0x194, i5 type +0x150, i12/i13 progress +0x18c/+0x190) sont TRIVIALES et adressées en
Ghidra, mais bloquées par le cadre => PORTER-APRES-TEST-CADRE.
4 vérifications adversariales sur pièces : ti=12 i0 navpoint-sub-type (FUN_1410e0cac, R(32))
CONFIRMÉ trivial ; ti=6 i57 statborg entry-index-and-type (FUN_1410be614, R(32)+R(8)) CONFIRMÉ ;
ti=11 i3 object-reference (FUN_142ed5550, +0x40) CONFIRMÉ trivial ; ti=47 i0 splash-message
(FUN_141085d50) RÉFUTÉ comme trivial (drapeaux d'encodage, dispatch tagué R(3), boucles à
compte variable, références string_id) => déplacé de PORTER à DOCUMENTER.
Liste PORTER-MAINTENANT nette = 2 feuilles, coût quasi nul : ti=12 i0 (sonder l'identité de
marqueur, sémantique à valider) et ti=6 i57 (cadrage typé des compteurs de score). Le [!] qui
compte : ti=11 i3 object-reference nomme DIRECTEMENT le porteur du crâne Oddball = exactement
le [!] que 5 campagnes (D4-D10, gate 79,8/73,8/64,9 % pour 80, porteur 0/3) n'ont pas su
établir => priorité produit n°1 dès cadre 108b prouvé. Total Control et VIP restent [!]
DÉFINITIFS.
Conclusion / prochaine étape : livrable = ce catalogue (à commiter dans .ai). Condition de
reprise du gain objectifs vivants = prouver le cadre du corps d'image-clé ti=11 (en-tête 108b),
qui débloque i3/i14/i5/i12/i13 d'un coup ; en parallèle, PM ti=12 i0 et ti=6 i57 sans
dépendance de cadre ; PTC ti=23 gardé par la validation de largeur de position quantifiée.
```

---

## 7. Lignes pour `.ai/V7.5/REGISTRE_REPORTS.md`

```
[2026-08-27] ti=11 managed-objective (5 feuilles triviales : i3 object-reference +0x40, i14
  state +0x194, i5 type +0x150, i12/i13 progress +0x18c/+0x190) — PORTER-APRES-TEST-CADRE.
  Grammaire triviale et adresses Ghidra prêtes (FUN_142ed5550/5948/FUN_1410fc4a4/FUN_142ed575c/
  FUN_142ed5844) MAIS 0/34 dispatch (ecs_table.tsv:268-301) : bloquées par le corps d'image-clé
  108b non reproduit (plafond 0,51 % bit-exact sur ti=35). CONDITION DE REPRISE = prouver le
  cadre du corps d'image-clé (en-tête 108b, FUN_142e2bfd0/FUN_142e2c690). Débloque en priorité
  le PORTEUR Oddball (i3) — le [!] de 5 campagnes D4-D10 — et le drapeau libre CTF.

[2026-08-27] ti=12 i0 managed-navpoint-sub-type (FUN_1410e0cac, R(32), traverse.go:490) —
  PORTER-MAINTENANT (déser), CONFIRMÉ trivial. CONDITION DE REPRISE de l'EXPLOITATION =
  échantillonner la valeur 32b sur films réels pour établir la sémantique (drapeau/zone/
  coéquipier) — la déser publie sans décoder ; l'identité de marqueur reste une hypothèse.

[2026-08-27] ti=6 i57 statborg-entry-index-and-type (FUN_1410be614, R(32)+R(8), traverse.go:456)
  — PORTER-MAINTENANT, CONFIRMÉ trivial. Enhancement : brancher le cadrage typé des compteurs
  de score i0 (sans i57 on ne sait pas quel compteur on lit). Pas de blocage.

[2026-08-27] ti=47 i0 splash-message-static (FUN_141085d50) — RETIRÉ de PORTER, passe à
  DOCUMENTER (vérification adversariale). N'EST PAS une feuille triviale : drapeaux d'encodage
  R(1) multiples, dispatch tagué R(3), boucles à compte variable, références string_id. Le
  libellé humain (« Drapeau capturé! ») exige une résolution string_id -> chaîne localisée,
  faisabilité hors-ligne OUVERTE. CONDITION DE REPRISE = RE sémantique de message-static-
  realization-str + table de chaînes offline.

[2026-08-27] ti=23 selectable-zone-data (FUN_142ed6cec, R(32)+POSITION+R(1)[+R(5)],
  components_world.go:106) — PORTER-APRES-TEST-CADRE. Seul archétype de zone objectif restant à
  câbler. CONDITION DE REPRISE = valider la largeur de la position quantifiée (même cadre que le
  bipède ti=35 / ti=13 t3), puis câbler et mesurer.

[2026-08-27] Feuilles ti=11 i1 (color) et i16-i31 (sub-objective-entities) — DOCUMENTER, feuille
  NON adressée en Ghidra. CONDITION DE REPRISE = RE de feuille d'abord. i1 = seule voie
  propriétaire hors désignateur (repli KOTH par rampes) ; i16-31 = seule piste identité de zone
  A/B/C et chemin de reprise Total Control (contourne l'ancrage ti=13 cassé).
```

---

### Réserves portées par ce livrable (rappel)

1. **RE_LECTEUR doc ABSENT** — chaîne et feuilles vérifiées directement sous Ghidra, sans
   dépendre du fichier manquant.
2. **Attribution i-index ti=11 non re-dérivée** — dispatch par nom à l'exécution ; la
   grammaire et le champ de destination sont vérifiés, l'étiquette i3/i5/i12/i13/i14 repose
   sur l'assertion de la tâche, corroborée par la concordance champ↔écriture.
3. **ti=12 i0 : grammaire CONFIRMÉE, sémantique NON** — la valeur 32b → identité de marqueur
   est une hypothèse à échantillonner, pas un fait RE.
4. **Statuts « PUBLIÉ/PORTÉ »** reportés de l'état HEAD du chantier ; les faits contestés ou
   décisifs (RE_LECTEUR absent, ti=11 non_porté, ti=12 i0 trivial, ti=47 réfuté, Total
   Control retiré, Firefight non servi, chiffres Oddball) ont été re-vérifiés sur pièces.
