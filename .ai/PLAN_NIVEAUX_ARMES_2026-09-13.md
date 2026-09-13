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
- [ ] 0.1 Test de recherche jetable (`*_research_test.go`, précédent lot 6.3) sur les artefacts
  locaux `data/cache/replays/halo_infinite/*.json` : pour chaque socle du match, l'emplacement
  confirmé à < 1 m et sa famille ; compter par carte : confirmés `rack` / `power` / `powerup`,
  non confirmés. Sortie brute au rapport.
- [ ] 0.2 Contrôle croisé rôle × famille (tableau arme × niveau) ; lister les incohérences.
- [ ] 0.3 Cartes absentes de `map_weapon_pads.json` (Forge, cartes récentes) : les nommer avec
  leur nombre de matchs ; décider (utilisateur) si on complète la référence avant de livrer.
- Gate : rapport `.ai/V7.5/RAPPORT_NIVEAUX_ARMES_<date>.md` ; verdict « couverture suffisante »
  ou « référence à compléter d'abord ».

### Étape 1 — Go : la famille voyage avec l'emplacement (requête)
- [ ] 1.1 `film/replay/map_weapon_pads.go` : `MapWeaponPadDTO.Family` (`rack` | `power` |
  `powerup`), posé depuis `MapWeaponPadsEntry.Pads[].Family` (déjà dérivée du `type_id`).
  Test unitaire `BuildMapWeaponPads`.
- [ ] 1.2 `film/replay/…` : fonction pure `WeaponTierOf(pad WeaponPad, cross *MapWeaponPads,
  loadouts []Loadout) Tier` dans `internal/analysis/` (paquet pur, testé sur fixtures) :
  base > terrain > puissance > non classé, avec la règle d'exclusion Fiesta reçue en paramètre
  (jamais lue du slug).
- [ ] 1.3 Contrat : `openapi.yaml` régénéré, `make generate-types`.
- Gate : `go build`, `go vet`, `go test ./internal/games/halo_infinite/film/replay/...
  ./internal/analysis/...`.

### Étape 2 — Match view : le bloc « Contrôle des armes » par niveau
- [ ] 2.1 `features/match-replay/model/padControlLogic.ts` : chaque ligne d'arme porte son
  niveau (depuis `mapWeaponPads[].family` + `loadouts`) ; regroupement Base / Terrain /
  Puissance / Non classé, sous-totaux par niveau, ordre fixe.
- [ ] 2.2 `MatchPadControlSection.tsx` : intertitres de niveau, ligne « Départs aléatoires »
  en mode Fiesta (catégorie de mode lue depuis la réponse de la match view, pas du nom).
  i18n FR/EN (`match-replay/i18n`), tests de rendu, capture lue sur le témoin `7fce3219`.
- Gate : tsc, eslint, vitest `features/match-replay`, capture.

### Étape 3 — Agrégats : Sessions, Escouade, Timeseries
- [ ] 3.1 `internal/analysis/sessionusage/` : les prises de socle (`pad`) se ventilent par
  niveau (remplace ou complète la ventilation par famille d'arme `usage_families.go`) ; DTO
  `session_usage.go` + `equipment_usage.go` ; tests Go sur fixtures.
- [ ] 3.2 Web : `features/_shared/usage/` — barres et parts par niveau (mêmes formes qu'aujourd'hui,
  une ligne par niveau, détail par arme au survol), sur Sessions, Escouade, Timeseries.
- Gate : suites Go et vitest complètes, captures lues des trois pages sur données réelles.

### Étape 4 — Clôture
- [ ] 4.1 Garde-rail : test qui interdit toute liste d'armes « de base » en dur côté Go ou TS
  (grep sur les clés `hinf_*` dans `features/` et `internal/analysis/`).
- [ ] 4.2 thought_log, référence équipement (`.ai/REFERENCE_CANAUX_EQUIPEMENT_2026-09-09.md`)
  amendée d'une section « niveaux d'armes », plan statué, CI verte, fusion dans `feat/v75`.

## 4. Découvertes

## 5. Journal
- 2026-09-13 — Plan écrit sur demande utilisateur ; non démarré.
