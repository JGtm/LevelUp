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
- [!] 2 Vies libres -> `flagObjects`, datation du lacher, `dropped` repositionne : NON PUBLIE. Le
      CONTROLE 3, ecrit avant la mesure, REFUSE la piste — 149/197 = 75,6 % (seuil 90 %). La
      decision 3 du plan s applique telle qu ecrite. Mesure, diagnostic et ancrage : journal.
- [x] 3 Registre/journal (textes au CR), plan statue.

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
- 2026-08-18 — PHASE 2 : LE CONTROLE 3 REFUSE LA PISTE. `[!]`, et la decision 3 s'applique telle
  qu'elle a ete ecrite — negatif ecrit, `flagObjects` NON PUBLIE.
  LA MESURE, sur les trois films CTF du corpus (`drapeau_objet_controle_test.go`, garde
  `OBJ_FILM` + `OBJ_REPO`) : **149/197 = 75,6 %** des vies libres naissent a moins de 1,5 m d'un
  `flag_spawn` ou du porteur qui vient de finir, contre **>= 90 %** exige. Par film :
  `64e8adfa` 79/110 = 71,8 % (29 au socle, 55 au porteur) · `530820e5` 33/41 = 80,5 % (15/22) ·
  `53ce4390` 37/46 = 80,4 % (16/22).
  LE TEMOIN TIENT, ET C'EST CE QUI REND LE NEGATIF INTERPRETABLE : les creations `ti=42` d'ARMES
  ORDINAIRES, passees a la MEME regle, ne font que 122/950 = **12,8 %** (seuil <= 20 %). La piste
  discrimine donc d'un facteur six — elle n'est pas du bruit — mais un quart des vies reste
  inexplique, et publier ces 48 vies-la ferait dessiner comme drapeau des objets dont rien ne dit
  qu'ils le sont.
  LE DIAGNOSTIC ECARTE LA PISTE FACILE : sur les 48 non expliquees, **3** seulement naissent la ou
  l'objet reposait deja (re-creation d'un drapeau au sol). Le residu n'est donc pas un artefact de
  re-creation sur place ; sa cause reste ouverte.
  DEUX DEFAUTS D'INSTRUMENT ONT ETE CORRIGES AVANT DE CONCLURE, aucun ne touche un seuil :
  (a) la reference « porteur » ne retenait que la DERNIERE frame d'un portage, ce qui excluait par
  construction le LACHER VOLONTAIRE — le phenomene meme que le lot existe pour dater (48,2 % ->
  71,8 % sur `64e8adfa`) ; (b) le jeu de socles reutilisait le filtre de PRODUCTION, qui ecarte le
  socle neutre a bon droit pour publier mais pas pour mesurer, quand le plan et son acquis parlent
  du ROLE `flag_spawn` (+1 vie). Le temoin a suivi les deux corrections (12,5 % -> 15,1 % sur
  `64e8adfa`, 12,8 % au total) : elles ne flattent pas le drapeau.
  CE QUI A ETE MESURE MALGRE TOUT, sur la chaine de publication ecrite puis RETIREE — les chiffres
  sont au CR et valent pour la reprise : 108/39/44 vies libres publiables, **2 portages
  `carried_open` fermes par une vie libre** (film `530820e5`, les 2 qu'il portait), 31/17/4
  `dropped` repositionnes sur la piste libre.
  ANCRAGE TENU : 78/30/29 portages publies, identiques a l'item 1.3 et aux artefacts en cache ;
  socles d'arme et couverture `groundWeapons` inchanges (retenues 352/239/359).
  TEMOINS NON RE-CUITS `[~]` : plus rien de nouveau a publier (schema 14 inchange), et les deux
  ancrages qu'ils devaient servir sont verifies contre les artefacts EXISTANTS par la mesure
  ci-dessus.
