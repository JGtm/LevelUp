# Plan — Le drapeau OBJET (mot MPP `0x2A392328`) : sa piste libre a cote des portages

> Ecrit le 2026-08-18 par la session de pilotage, a la suite de la phase 0 du plan attachement
> (registre : « le drapeau de CTF est un objet `ti=42` identifie : mot MPP `0x2A392328` ») et de la
> publication des portages (`flagCarries`, schema 14). Contrat `plan-execution`. Worktree frere.

## Acquis (ne pas re-mesurer)

- `0x2A392328` revient sur les 3 films CTF et 2 cartes : 110/41/46 creations `ti=42`, dont 41/16/18 a
  0,0 m d'un `flag_spawn`, une a 1 ms d'un evenement de l'oracle ; aucun autre mot ecarte du catalogue
  d'armes ne fait les deux (`attachement_phase0_drapeau_test.go`).
- La chaine des socles (`replay/ground_weapon_*`, `filmdec/ground_weapon_creation.go`) ECARTE
  aujourd'hui ce mot (identite hors catalogue) : le drapeau libre n'est ni socle ni piste.
- `flagCarries` (schema 14) publie `carried` / `carried_open` / `dropped` / `home` ; `dropped` = derniere
  position du porteur, faute de piste propre ; le lacher n'est pas date (`carried_open`, registre).
- Un objet du monde replique sa position quand il est LIBRE (chemin delta, `ScanFilmWorldObjects`),
  cesse quand il est porte : la piste du drapeau libre est donc lisible a l'image, ses fins de vie
  (nouvelle creation au socle) aussi.

## Decisions tranchees

1. `0x2A392328` entre au manifeste (`config/titles/halo_infinite/mappings/replay_labels.toml`) comme
   famille `flag` (libelle FR/EN via le TOML, jamais en dur) ; la chaine des socles le RECONNAIT et
   l'EXCLUT des `weaponPads` (un drapeau n'est pas un socle d'arme) — garde-rail.
2. Publication : `flagObjects` [{ team (socle `flag_spawn` le plus proche de la premiere creation),
   spans [{ t0, t1, points [{t, x, y}] }] }] = les vies LIBRES du drapeau (creation -> fin de piste),
   dans le meme fichier `document_objectives_live.go` ; `flagCarries.dropped` prend la position de la
   piste libre quand elle existe (sinon inchangee) et le lacher devient DATE quand une vie libre
   commence pendant un `carried_open` (=> `carried` ferme a t0 de la vie libre : la reprise du registre
   « un canal qui date le lacher »). Schema 15, contrat +1, chronique, OpenAPI, `generated.ts`,
   `NULLABLE_ARRAYS`, goldens, temoins CTF re-cuits ; films non-CTF : vide.
3. Controle (seuils AVANT mesure) : chaque vie libre commence a < 1,5 m d'un `flag_spawn` OU a < 1,5 m
   de la derniere position du porteur qui vient de finir (>= 90 % des vies) ; temoin : creations `ti=42`
   d'armes ordinaires <= 20 % ; sinon negatif ecrit, `flagObjects` non publie.
4. Rendu : phase 3 du plan objectifs vivants (a partir d'`objectivesLayer.ts`), pas ici.

## Phases

- [x] 1 Manifeste + exclusion des socles + garde-rail ; instrument de mesure du controle 3 (garde `OBJ_FILM`).
- [ ] 2 Vies libres -> `flagObjects`, datation du lacher, `dropped` repositionne ; schema 15 + contrat +
      OpenAPI + web + goldens + temoins ; controle 3 publie ; ancrage : portages identiques a 1.3.
- [ ] 3 Registre/journal (textes au CR), plan statue.

**Gate** : controle 3 tenu ; contrat/OpenAPI/goldens/web verts ; portages inchanges ; sinon `[!]`.

## Journal du plan
- 2026-08-18 — plan ecrit, agent lance (worktree frere `../LevelUp-wt-drapeau`, base `9b2aff1c2`).
- 2026-08-18 — PHASE 1 CLOSE. `0x2a392328` entre au manifeste sous une table NEUVE
  (`[[objective_objects]]`, famille `flag`, libelles FR/EN) : ce n est pas `equipment_objects`
  (archetype 37, chaine `sofd -> sofa -> eqip`) mais l archetype 42, celui des armes au sol.
  La chaine des socles le RECONNAIT et l ecarte AVANT la question « est-ce une arme ? » — hier
  il etait ecarte par accident (hors catalogue), ce qui ne se maintient pas. La couverture gagne
  `groundWeapons.objectives` (sous-ensemble NOMME de `rejected`) : sans lui, un drapeau reconnu
  et un octet de bruit sortent par la meme porte. Garde-rail a temoin
  (`ground_weapon_flag_exclusion_test.go`) : une famille DU CATALOGUE D ARMES declaree drapeau
  ne fait plus de socle, et le meme scan sans la table en fait un — sans ce temoin, le test
  vert ne prouverait rien. Instrument du controle 3 ecrit
  (`drapeau_objet_controle_test.go`, garde `OBJ_FILM` + `OBJ_REPO`), seuils 90 %/20 % et
  fenetre de lacher 1 s ECRITS AVANT la mesure ; rayon = `originDropMaxDist`, jamais redeclare.
  Gates verts : go build/vet/test (replay, games, contracttest, replaybuild, archlint),
  golangci 0 issue, tsc, eslint, vitest 1715/1715.
