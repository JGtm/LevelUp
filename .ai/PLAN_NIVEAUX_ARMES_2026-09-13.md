# PLAN — Trois niveaux d'armes : base, terrain, puissance (2026-09-13)

> Demande utilisateur (13/09) : « pour les graphes “contrôle des armes”, faire la distinction
> entre les armes de base (celles avec lesquelles on spawn en général, hors mode Fiesta), les
> armes de terrain et les armes spéciales […] sur les râteliers ce sont des armes de terrain,
> sur les socles des armes de puissance. Ça permettrait d'afficher le bon niveau d'information. »
> Décision : « planifie-le, tu le traiteras quand tu auras de la bande passante ».
>
> Statut : PLANIFIÉ, non démarré. À ouvrir APRÈS l'intégration de
> `PLAN_AJUSTEMENTS_PRE_V75_2026-09-13.md`. Contrat : skill `plan-execution`. Branche :
> `feat/niveaux-armes` depuis `feat/v75`, worktree dédié `LevelUp-wt-niveaux-armes`.
> Effort : moyen (Go à la requête + agrégats + 3 pages web). Aucune recuisson d'artefact.

## 1. Objectif et critère de succès

Chaque prise d'arme mesurée par le film se lit à l'un de trois niveaux, et les blocs
« Contrôle des armes spéciales » (Match view, Sessions, Escouade, Timeseries) s'organisent
par niveau :

| Niveau | Définition mesurée | Source |
|---|---|---|
| **Base** | arme présente dans l'équipement de DÉPART d'une vie du match (canal `loadouts` du film, `Loadout.W`) | film, par match — donc juste en classé (BR75 / Bandit), en partie rapide (AR + Sidekick), quel que soit le mode |
| **Terrain** | arme apparue sur un RÂTELIER de la carte | référence des emplacements de la carte (`map_weapon_pads`, famille `rack`, dérivée du `type_id` du fichier Forge `0x6253CFC0`), croisée au socle du match à < 1 m (`BuildMapWeaponPads`, déjà en place) |
| **Puissance** | arme apparue sur un SOCLE DE PUISSANCE | même référence, famille `power` (`0x5F379533`) |
| Non classé | socle du match sans emplacement confirmé (carte absente de la référence, ou socle hors rayon) | affiché comme tel, jamais fondu dans un autre niveau |

Critère de succès : sur le match témoin `7fce3219` (CTF Takamanohara) et sur une session de
5 matchs, chaque arme du bloc porte son niveau, les totaux par niveau sont égaux au total du
bloc actuel, et « Non classé » est ≤ 5 % des prises sur le parc local (sinon la référence des
cartes se complète AVANT de livrer — étape 0).

Fiesta / Super Fiesta / Husky Raid (catégories `mode_category`, `halo_infinite/mode_category.go`) :
le niveau « Base » n'est pas publié (équipement de départ aléatoire) ; les deux autres niveaux
restent lisibles. Le bloc l'écrit en une ligne (« Départs aléatoires : pas de niveau de base »).

Contrôle croisé (garde-rail, pas une source) : le rôle du registre canonique
(`internal/games/weapons/registry.go`, `RolesByKey`) — une arme `power`/`sniper`/`special` sur
un râtelier, ou `automatic`/`sidearm` sur un socle de puissance, est compté et journalisé
(`slog.WarnContext`) ; au-delà d'un seuil mesuré à l'étape 0, c'est la jointure qui est fausse.

## 2. Décisions tranchées (ne pas rouvrir)

- D1 La nature d'un emplacement vient de la CARTE (référence Forge), jamais du nom de l'arme.
  Une même arme peut être de terrain sur une carte et de puissance sur une autre.
- D2 « Base » est mesuré par match depuis les équipements de départ du film, jamais depuis une
  liste figée par mode.
- D3 Résolution À LA REQUÊTE (comme `mapWeaponPads`, `mapObjectives`) : aucun champ nouveau
  dans l'artefact, `SchemaVersion` inchangé, aucune recuisson.
- D4 Un socle non confirmé reste « Non classé » et visible (règle « l'absence a sa forme »).
- D5 Multi-titre : tout passe par capability (`film.replay_artifact` / clé fine à créer
  `film.weapon_tiers` dans `capabilities.toml`), jamais par slug ; Halo 5 = niveau absent, pas
  d'erreur.

## 3. Étapes

### Étape 0 — Mesure de couverture (diagnostic, aucun code de production)
- [x] 0.1 Test de recherche jetable (`*_research_test.go`, précédent lot 6.3) sur les artefacts
  locaux `data/cache/replays/halo_infinite/*.json` : pour chaque socle du match, l'emplacement
  confirmé à < 1 m et sa famille ; compter par carte : confirmés `rack` / `power` / `powerup`,
  non confirmés. Sortie brute au rapport.
- [x] 0.2 Contrôle croisé rôle × famille (tableau arme × niveau) ; lister les incohérences.
- [x] 0.3 Cartes absentes de `map_weapon_pads.json` (Forge, cartes récentes) : les nommer avec
  leur nombre de matchs ; décider (utilisateur) si on complète la référence avant de livrer.
- Gate : rapport `.ai/V7.5/RAPPORT_NIVEAUX_ARMES_2026-09-14.md` (`24f3c813c`). VERDICT : couverture suffisante — 76 artefacts, 76 cartes toutes au catalogue, 669 socles confirmés (466 râteliers, 144 puissance, 48 bonus), 11 non confirmés (1,64 %), « Non classé » = 2,95 % des prises (85/2 881, dont 52 sur Flood Gulch ; 1,28 % hors cette carte). 17 matchs sans aucun socle publié (mode qui n'allume rien) = absence de mesure à dire. Croisé rôle × famille : 70 écarts nominaux (Hydra, Needler, Sentinel Beam, Shock Rifle sur râteliers → D1 confirmée), 3 inversions réelles (0,45 %). « Base » se lit du film : Assassin 94,4 % sur AR/Sidekick/BR75, BTB 92,6 %, Super Fiesta plat — lire la PREMIÈRE émission `loadouts` par slot (tout le canal dilue à 85,6 %).

### Étape 1 — Go : la famille voyage avec l'emplacement (requête)
- [x] 1.1 `film/replay/map_weapon_pads.go` : `MapWeaponPadDTO.Family` (`rack` | `power` |
  `powerup`), posé depuis `MapWeaponPadsEntry.Pads[].Family` (déjà dérivée du `type_id`).
  Test unitaire `BuildMapWeaponPads`.
- [x] 1.2 `film/replay/…` : fonction pure `WeaponTierOf(pad WeaponPad, cross *MapWeaponPads,
  loadouts []Loadout) Tier` dans `internal/analysis/` (paquet pur, testé sur fixtures) :
  base > terrain > puissance > non classé, avec la règle d'exclusion Fiesta reçue en paramètre
  (jamais lue du slug).
- [x] 1.3 Contrat : `openapi.yaml` régénéré, `make generate-types`.
- Gate : `go build`, `go vet`, `go test ./internal/games/halo_infinite/film/replay/...
  ./internal/analysis/...`.

### Étape 2 — Match view : le bloc « Contrôle des armes » par niveau
- [x] 2.1 `features/match-replay/model/padControlLogic.ts` : chaque ligne d'arme porte son
  niveau (depuis `mapWeaponPads[].family` + `loadouts`) ; regroupement Base / Terrain /
  Puissance / Non classé, sous-totaux par niveau, ordre fixe.
- [x] 2.2 `MatchPadControlSection.tsx` : intertitres de niveau, ligne « Départs aléatoires »
  en mode Fiesta (catégorie de mode lue depuis la réponse de la match view, pas du nom).
  i18n FR/EN (`match-replay/i18n`), tests de rendu, capture lue sur le témoin `7fce3219`.
- Gate : tsc, eslint, vitest `features/match-replay`, capture.

### Étape 3 — Agrégats : Sessions, Escouade, Timeseries
- [x] 3.1 `internal/analysis/sessionusage/` : les prises de socle (`pad`) se ventilent par
  niveau (remplace ou complète la ventilation par famille d'arme `usage_families.go`) ; DTO
  `session_usage.go` + `equipment_usage.go` ; tests Go sur fixtures.
- [x] 3.2 Web : `features/_shared/usage/` — barres et parts par niveau (mêmes formes qu'aujourd'hui,
  une ligne par niveau, détail par arme au survol), sur Sessions, Escouade, Timeseries.
- Gate : suites Go et vitest complètes, captures lues des trois pages sur données réelles.

### Étape 4 — Clôture
- [x] 4.1 Garde-rail : test qui interdit toute liste d'armes « de base » en dur côté Go ou TS
  (grep sur les clés `hinf_*` dans `features/` et `internal/analysis/`).
- [x] 4.2 thought_log, référence équipement (`.ai/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md`)
  amendée d'une section « niveaux d'armes », plan statué, CI verte, fusion dans `feat/v75`.

## 4. Découvertes

- **D-a (étape 1)** — Le niveau est implémenté DEUX FOIS, et c'est assumé : en Go
  (`internal/analysis/weapontier`) pour les agrégats, en TypeScript (`padControlLogic`)
  pour la vue match. Une résolution serveur unique aurait demandé que le service de rejeu
  connaisse la catégorie de mode, qu'il ne lit pas aujourd'hui (elle vit dans
  `MatchViewHeader.mode_category`, côté vue match). Les deux copies partagent les mêmes
  constantes écrites et les mêmes témoins de test.
- **D-b (étape 1)** — Le ratchet de forme du document (`document_shape_test.go`) refusait
  toute régénération sans montée de `SchemaVersion`, y compris pour un calque que la
  CUISSON N'ÉCRIT JAMAIS (`mapObjectives`, `mapWeaponPads`, résolus à la requête). Monter
  la version aurait fait lire « à re-cuire » les 77 artefacts du parc pour un champ qu'aucun
  ne porte — contraire à D3. Le ratchet porte désormais une QUATRIÈME empreinte,
  `empreinte-cuite`, et c'est elle qui gouverne le refus ; l'empreinte entière reste figée
  (donc toute modification reste visible), et un nouveau test interdit que ces deux calques
  entrent un jour dans la cuisson sans qu'on le voie.
- **D-d (étape 2)** — AUCUN match Super Fiesta du parc local ne publie de socle : les 13
  artefacts de cette catégorie sont à ZÉRO `weaponPads` (le mode n'allume aucun emplacement,
  comportement déjà documenté en tête de `map_weapon_pads.go`). La ligne « Départs aléatoires »
  n'est donc visible sur AUCUNE donnée réelle du poste : la capture la montre avec
  `header.mode_category` forcé sur le match témoin. Même raison pour `family` : le serveur de
  :8000 tourne sur `feat/v75` et ne le publie pas encore — la fixture est la VRAIE réponse du
  serveur, la famille ajoutée en appariant chaque emplacement à la référence versionnée
  (16 sur 16 à moins d'un centimètre).
- **D-e (étape 3, BLOQUANT)** — L'hypothèse de l'étape 3 ne tient pas : les agrégats
  (Sessions, Escouade, Timeseries) ne lisent PAS les artefacts, ils lisent la table cuite
  `match_usage_players_latest`, dont `pad_pickups_json` ventile les prises par FAMILLE D'ARME
  et a donc PERDU l'index du socle. Le niveau, lui, se lit sur le SOCLE (sa position, croisée à
  la référence de la carte) : il ne peut pas se reconstituer côté lecture. Le rendre disponible
  demande (1) une révision de la projection `UsageSummaryRev` us6 -> us7, (2) une colonne de
  plus dans `match_usage_players` (migration), (3) de faire entrer la référence des cartes, le
  `map_id` et la catégorie de mode dans `BuildUsageSummary(doc)` — qui ne reçoit aujourd'hui
  QUE l'artefact — côté sync ET côté backfill, et (4) une passe `levelup backfill-usage-summary`
  sur tout le parc, locale puis PROD. Or cette passe exige le SERVEUR ARRÊTÉ (écrit dit en tête
  de `cmd_backfill_usage_summary.go` : « OpenReadWrite echoue si le lock est tenu »), ce que le
  cadre de ce lot interdit. Deux décisions utilisateur : accepter la révision de projection + le
  backfill prod, ou se contenter des axes déjà persistés (rôle/classe du registre) — ce dernier
  contredit D1 et produirait 10 % de faux niveaux.
- **D-c (étape 0, hors périmètre)** — Une seule arme du parc (59 matchs, 2 881 prises) porte
  deux niveaux dans le même match : 5 prises, 0,17 %, et c'est `terrain` + `non classé`,
  jamais `terrain` + `puissance`. Les lignes du bloc restent donc keyées par arme.

## 5. Journal
- 2026-09-13 — Plan écrit sur demande utilisateur.
- 2026-09-14 — Décision utilisateur « c'est à faire » ; étape 0 exécutée et fusionnée ; étapes 1-4 lancées.
- 2026-09-14 — Étape 1 CLOSE : `MapWeaponPadDTO.Family` (stocké + servi + openapi + types TS),
  paquet pur `internal/analysis/weapontier` (niveaux + contrôle croisé, 10 tests), garde-rail
  journalisé au service, ratchet de forme amendé (D-b). Gate : `go build`, `go vet ./internal/...`,
  `go test ./internal/games/halo_infinite/film/replay/... ./internal/analysis/... ./internal/service/...
  ./internal/domain/...` verts, `openapi-gen -check` à jour.
- 2026-09-14 — Étape 2 CLOSE : `model/weaponTier.ts` (jumeau TS du paquet Go, 9 tests),
  `padControlLogic` porte le niveau de chaque arme + les sous-totaux + les deux drapeaux,
  `MatchPadControlSection` écrit un intertitre par niveau avec son sous-total, la note
  « Départs aléatoires » et la note « niveaux non établis », i18n FR/EN, `modeCategory`
  câblé depuis `header.mode_category`. Gate : `tsc -b --force` 0 erreur, eslint 0 erreur
  sur les fichiers touchés, `vitest run` complet 713 fichiers / 7 647 tests verts.
  Captures LUES (stub assumé, cf. D-d).
- 2026-09-14 — **Étape 3 BLOQUÉE, décision utilisateur requise** (cf. D-e). Étapes 1 et 2
  livrées et vertes ; rien de poussé.
- 2026-09-14 — **Étape 3 CLOSE** par le MOTIF DES PRISES NETTES (décision utilisateur : voie 1,
  D1 ferme ; ni révision de projection du résumé d'usage, ni recuisson). Livré : table
  append-only `match_pad_pickups_by_tier` + vue `_latest` (enrôlée dans les TROIS garde-rails),
  `persist.PadTiersPersister` (INSERT-only), projection au fil de l'eau
  `sync/replayartifacts/padtiers.go` sous capability `film.weapon_tiers` + règle de titre
  `[weapon_tiers].random_start_mode_prefixes`, CLI reprenable `levelup backfill-pad-tiers`
  (`--dry-run` : une ligne par match), lecture `sessionusage.ComputePadTiers` branchée sur les
  DEUX chemins (page Sessions et bloc d'équipement Escouade/Timeseries), DTO, et la rangée web
  dans `features/_shared/usage/` + la section de la page Sessions. Tests de câblage qui
  rougissent si la porte de capability ou la lecture produit est débranchée
  (`padtiers_test.go`, `service/pad_tiers_wiring_test.go`).
- 2026-09-14 — **Étape 4 CLOSE** : garde-rail `archlint/no_hardcoded_base_weapons_test.go`
  (aucune collection de clés `hinf_*`/`h5_*` dans `internal/analysis/` ni `apps/web/src/features/`),
  section « niveaux d'armes » ajoutée à `.ai/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md`,
  `docs/COMMANDS.md` + `docs/FR/COMMANDS.md` documentent le rattrapage.
- 2026-09-14 — **RESTE À FAIRE, HORS DE CE LOT** : (1) le rattrapage LOCAL
  `levelup backfill-pad-tiers` — il exige le SERVEUR ARRÊTÉ, il revient au superviseur après
  fusion ; (2) le rattrapage PROD, à porter à la liste de release v7.5 à côté de
  `backfill-flag-grabs-net` ; (3) la passe visuelle sur données réelles des trois pages
  d'agrégat, impossible tant que (1) n'a pas tourné (la table est vide sur le poste).
