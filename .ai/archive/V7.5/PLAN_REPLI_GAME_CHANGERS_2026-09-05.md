# PLAN — Repli « game changers » dans les rendus d'usage (2026-09-05)

> Commandé par l'utilisateur le 2026-09-04 (« replier ceux qui ne sont pas game changer »),
> liste TRANCHÉE PAR VOTE le 2026-09-05 (artefact « Game changers », base de l'artefact,
> 10 élus relus par le superviseur). Exécution PILOTÉE : agent d'implémentation,
> revue adversariale à contexte frais, consolidation par le superviseur.
> Worktree `LevelUp-wt-game-changers`, branche `wt/game-changers` (base feat/v75
> `ca55f0ed7`). Contrat : skill `plan-execution`.
>
> **Doctrine.** Le repli est une HIÉRARCHIE D'AFFICHAGE, jamais une suppression : tout
> reste accessible en un clic, les dénominateurs et notes de couverture ne bougent pas.
> La liste est un JUGEMENT PRODUIT voté — elle vit côté web (constante TS + garde-rails
> contre les TOML), pas dans les données de titre (précédent exact : `POWER_PAD_KEYS`,
> arbitrage utilisateur du 2026-08-18, weaponPadFamilies.ts).

## Verdict du vote (2026-09-05, source : base de l'artefact, collection `votes`)

- Équipements EN AVANT (5) : `powerup_camo`, `powerup_overshield`, `sensor`,
  `threat_seeker`, `shroud_screen`. REPLIÉS (6) : `grapple`, `thruster`, `repulsor`,
  `wall`, `repair_field`, `translocator_beacon`.
- Armes EN AVANT (weapon_key, 6+1 variante) : `hinf_s7_sniper`, `hinf_m41_spnkr`
  (+ `hinf_fuel_rod_spnkr`, variante du même socle), `hinf_energy_sword`,
  `hinf_gravity_hammer`, `hinf_skewer`, et `hinf_cindershot` — voté non au matin du
  05/09, PROMU par décision utilisateur le 05/09 même (« le cindershot peut être un game
  changer ») ; la coïncidence avec `POWER_PAD_KEYS` redevient un fait, figé par test.
  DÉCISION du même échange : PAS de réalignement du rejeu 2D sur le vote —
  `POWER_PAD_KEYS` reste une liste indépendante.

## Décisions tranchées AVANT exécution (non rediscutables par l'agent)

- D1 La liste du repli est une constante TS NOUVELLE et sémantiquement DISTINCTE de
  `POWER_PAD_KEYS` (qui gouverne l'échelle des socles du rejeu 2D et reste INTOUCHÉ —
  divergence cindershot DITE au plan et à l'utilisateur, réalignement éventuel = décision
  utilisateur future). Ce n'est pas une 3e copie : autre jugement, autre surface.
- D2 Périmètre = fiche match, onglet Chronologie : bilan d'équipement
  (`MatchEquipmentUsageSection`) et contrôle des socles (`MatchPadControlSection`).
  EXCLUS (candidats à arbitrage ultérieur, ne pas toucher) : `FragWeaponBreakdown` et
  tous les charts partagés (replier BR/AR dans un graphe de frags cacherait l'essentiel),
  `PlayerDetailPanel`, squad/synthesis/timeseries/session, calques du rejeu 2D.
- D3 REPLIÉ PAR DÉFAUT, état local au montage (non persisté), contrôle « Voir plus (N) /
  Replier » sur le patron `MedalDigest.tsx:281-294`. Deux copies locales du patron sont
  ACCEPTÉES (règle <= 2) ; pas d'extraction `components/ui/` dans ce lot.
- D4 Les GRENADES sont HORS VOTE (l'utilisateur a déjà tranché : ce ne sont pas des
  équipements) : le groupe grenades du bilan reste tel quel, ni en avant ni replié
  autrement qu'aujourd'hui — il suit le bloc replié ? NON : il reste TOUJOURS VISIBLE,
  après les groupes en avant.
- D5 Pont de vocabulaire : `powerup_camo` -> famille d'épisode `camo`,
  `powerup_overshield` -> `overshield` (le piège n°1 du lot — deux vocabulaires pour le
  même objet : socle vs état actif). Le pont est ÉCRIT et testé, jamais un `includes`.
- D6 Armes identifiées par `doc.weaponLabels[hex].key` (weapon_key), JAMAIS par l'hex ni
  par le libellé. Un socle dont le label n'a pas de `key` (artefact ancien) est traité
  comme NON game changer (replié) — dégradation dite, pas devinée. Les socles de bonus
  `powerup_camo`/`powerup_overshield` du contrôle des socles sont en avant (vote).

## Lots (ordre strict, contrat plan-execution)

## CLÔTURE (05/09) — tous lots FAITS, revue passée, commité par le superviseur

Statuts : G0.1 [x], G0.2 [x] (12 garde-rails, divergence cindershot FIGÉE par test),
G1.1 [x], G1.2 [x], G1.3 [x], G2.1 [x], G2.2 [x], G2.3 [x], G3.1 [~] (couvert par
G1.2/G2.2 — 3 clés partagées `collapsedColumns*` FR+EN dans l'i18n du rejeu, existant
vérifié avant ajout), G3.2 [x] (typecheck purgé exit 0, make test-web 575 fichiers /
5 999 tests / 0 échec, eslint 0, garde-rails verts, plafonds tenus — les deux composants
ramenés à 80 L pile par extraction), G3.3 [x] (revue ci-dessous).

INCIDENT D'EXÉCUTION, DIT : une restauration de mutation par `git checkout --` a effacé
les éditions non commitées d'`equipmentUsageColumns.ts`, réappliquées de mémoire — la
revue a relu le fichier en entier avec ce point d'attention imposé : « cohérent au
centime », rien de perdu.

REVUE ADVERSARIALE RONDE 1 (05/09, relecteur frais, mutations sur copies, SHA256 de
restauration vérifiés) : **16 conditions tiennent** — dont D1-D6 une à une (mutations :
replié par défaut, pont D5 retiré = 19 échecs, branche powerup retirée, partition
inversée/.reverse(), bouton à N=0), les invariants « rien ne ment » (totaux/attribués/
footnotes calculés sur TOUT, partition d'affichage seule), le chart replié jugé
AFFICHAGE COHÉRENT (aucune agrégation visible ne sous-compte ; « Voir plus (N) » annoncé
dans l'en-tête). **0 P0, 0 P1, 1 P2 cosmétique CONSIGNÉ** (voir Découvertes — padding
résiduel quand tout est replié). Pas de ronde 2 : aucune correction apportée.

### G0 — La liste et ses garde-rails

- [x] G0.1 `features/match-replay/gameChangers.ts` : `GAME_CHANGER_EQUIPMENT_FAMILIES`
      (les 5 familles), `GAME_CHANGER_WEAPON_KEYS` (les 5+1 clés), pont
      `EPISODE_FAMILY_OF_POWERUP` (D5), prédicats purs `isGameChangerFamily(family)` /
      `isGameChangerWeaponKey(key | undefined)`. En-tête : provenance (vote du 05/09,
      artefact) + la divergence cindershot vs POWER_PAD_KEYS, DITE.
- [x] G0.2 Garde-rails (patron `weaponPadFamilies.test.ts`) : chaque famille existe dans
      `config/titles/halo_infinite/mappings/replay_labels.toml` ; chaque weapon_key existe
      dans `config/titles/halo_infinite/mappings/weapon_names.toml` ; chaque clé du pont
      D5 existe dans `EPISODE_FAMILIES` (`equipmentUsageLogic.ts`).

### G1 — Bilan d'équipement : les groupes se replient

- [x] G1.1 Partition dans la LOGIQUE (pure, `equipmentUsageLogic.ts` /
      `equipmentUsageColumns.ts`, pas dans le composant) : groupes/colonnes de familles
      en avant d'abord, groupes/colonnes repliés ensuite, grenades toujours visibles
      (D4). L'ordre interne existant (PLACEMENT_RENDER, etc.) est conservé DANS chaque
      partition. Épisodes camo/overshield = en avant via le pont D5.
- [x] G1.2 Rendu : les colonnes repliées n'apparaissent qu'après « Voir plus (N) »
      (patron MedalDigest, compte = nombre de colonnes masquées), bouton dans l'en-tête
      de section ; footnotes, dénominateurs de couverture et double porte `hasData`
      INCHANGÉS (une famille repliée compte toujours dans les notes).
- [x] G1.3 Tests logic : partition (mutation « partition inversée » tuée), pont D5
      (mutation « pont retiré » : camo/overshield tombent du bloc en avant), grenades
      toujours visibles, zéro colonne repliée = pas de bouton.

### G2 — Contrôle des socles : les colonnes se replient

- [x] G2.1 `padControlLogic.ts` : partition par `isGameChangerWeaponKey` (via
      `weaponLabels`) AVANT le tri par volume existant (le tri « du plus disputé au
      moins disputé » survit DANS chaque partition) ; socles `powerup_*` en avant (D6).
- [x] G2.2 Rendu `MatchPadControlSection.tsx` : mêmes règles de repli que G1.2 ; la
      colonne Total reste calculée sur TOUTES les armes (repliées comprises) — le total
      ne ment pas.
- [x] G2.3 Tests logic : partition avant tri (mutation tuée), label sans `key` = replié
      (D6), total inchangé par le repli.

### G3 — i18n et clôture

- [~] G3.1 Strings FR ET EN (« Voir plus (N) » / « Replier », et tout libellé
      d'infobulle) — vérifier l'existant AVANT d'ajouter (MedalDigest a peut-être déjà
      les clés génériques réutilisables).
- [x] G3.2 Gates : `rm -rf node_modules/.tmp && npm run typecheck` exit 0 ;
      `make test-web` complet exit 0 ; `npx eslint` exit 0 sur les fichiers touchés ;
      grep couleurs en dur vide ; garde-rails G0.2 verts ; plafonds tenus (aucun fichier
      > 500 L, fonctions <= 80 L).
- [x] G3.3 Revue adversariale à contexte frais AVANT commit (1 relecteur, front) ;
      ronde 2 sur les corrections seules.

## Protocole

- Statuts `[x]`/`[~]`/`[!]`, aucune case vide à la clôture ; lot N clos avant N+1.
- Zéro fix opportuniste : découvertes consignées ci-dessous, jamais traitées.
- Aucun commit par les agents — consolidation par le superviseur. Reprise : ce fichier +
  `.ai/thought_log.md` du worktree.

## Lot H — Extension aux GRAPHES PARTAGÉS (commandé par l'utilisateur le 05/09 :
## « il faut étendre le repli aux graphes partagés, mets un agent sonnet dessus »)

Décisions tranchées :
- H-D1 Périmètre : `FragWeaponBreakdown` (composant PARTAGÉ — consommateurs : fiche match
  `MatchFragCard`, synthesis, timeseries summary, session-detail) ;
  `SquadWeaponKillsChart` ; `SquadWeaponAccuracyBarsChart` ;
  `SynthesisWeaponAccuracyChart`. TOUJOURS EXCLUS : `PlayerDetailPanel` (troncature
  existante), rejeu 2D, tableaux déjà traités (G1/G2).
- H-D2 Même grammaire que G1/G2 : armes élues d'abord, le reste replié derrière
  « Voir plus (N) », replié par défaut, état au montage. AUCUNE valeur ne ment : si un
  agrégat/une moyenne est AFFICHÉ, il se calcule sur toutes les armes.
- H-D3 IDENTIFICATION PAR CLÉ STABLE OBLIGATOIRE : vérifier SUR PIÈCES comment ces
  données agrégées identifient l'arme (weapon_key ? id ? libellé seul ?). Si une surface
  n'expose AUCUNE clé stable côté données (que des libellés localisés), cette surface est
  statuée `[!]` et REMONTÉE — jamais un matching par libellé FR/EN.
- H-D4 Le prédicat est celui de `gameChangers.ts` (réutilisé, pas copié). Si une clé
  d'agrégat diffère du vocabulaire `weapon_names.toml`, le pont est une table ÉCRITE et
  gardée, comme D5.
- H-D5 Ces surfaces vivent hors du rejeu : leurs strings i18n passent par le système de
  la feature concernée (manifests TOML ou dictionnaire local — vérifier l'existant),
  FR ET EN. Le patron « Voir plus (N) » y arrive en TROISIÈME+ copie -> extraction d'un
  composant/hook partagé DEVIENT OBLIGATOIRE (règle n°6 : helper + garde-rail) — à
  poser là où les quatre surfaces peuvent l'importer sans violer les frontières de
  features (vérifier le ratchet lint-cross-feature-imports, AU PLAFOND 7/7 : si
  l'extraction exige un 8e import croisé, la placer dans components/ ou lib/).

- [x] H1 FAIT le 05/09 — reconnaissance sur pièces, tableau par surface au rapport :
      AUCUNE des 4 surfaces n'expose de clé stable côté front (`SynthesisWeaponKillEntry`
      sans id ; `SquadWeaponBar.weapon_id` = id brut opaque, mapping id->key
      plusieurs-vers-un tenu en metadata.duckdb, non gardable côté front ;
      `SquadWeaponAccuracyBar` agrégé par RÔLE ; `SynthesisWeaponAccuracyEntry` libellé
      seul — et son port amont `WeaponAccuracyRow` n'a PAS de champ clé du tout) ;
      `MatchFragCard` jette `weapon_id` avant l'appel. CONTRE-VÉRIFIÉ par la revue H7,
      les cinq affirmations vraies. FAIT SERVEUR (P2 de la revue, décisif pour le lot I) :
      `WeaponKillRow.WeaponKey` (format `hinf_*`) est DÉJÀ résolu dans la même passe
      (`ResolveRoles: true`) pour Squad kills et Synthesis/Match kills — il n'est
      simplement pas sérialisé.
- [x] H2 FAIT — `components/ui/collapsed-items-toggle.tsx` (57 L, agnostique : zéro
      import, libellés en props), les DEUX copies G1/G2 migrées dessus (comportement
      constant : mêmes clés i18n, même chaîne CSS, garde N<=0 déplacée dans le
      composant — appelants jamais négatifs, DOM identique), garde-rail
      `collapsed-items-toggle.guard.test.ts` (glob src/** + assertion anti-glob-mort
      > 200 fichiers).
- [!] H3 NON TRAITÉ, justifié (H-D3 appliquée à la lettre) : aucune clé stable servie —
      débloqué par le lot I (champ API), jamais par un matching de libellé.
- [!] H4 NON TRAITÉ, même justification ; `SquadWeaponAccuracyBar` restera EXCLU même
      après le lot I (agrégation par rôle par conception — décision I-D2).
- [x] H5 FAIT — 6 tests du composant + garde-rail ; mutation « ré-inline fantôme »
      rejouée deux fois (feature migrée ET autre feature) -> ROUGE les deux fois.
- [x] H6 FAIT — typecheck purgé exit 0 ; vitest ciblé 51/51 ; make test-web 577
      fichiers / 6 008 tests / 0 échec ; eslint 0 ; grep couleurs vide ; ratchet
      cross-feature 7/7 inchangé (composant en components/ui/).
- [x] H7 FAIT — revue à contexte frais : 12 conditions tiennent (migration octet pour
      octet, garde-rail mordant dans toute feature, anti-glob-mort, suites
      préexistantes inchangées vertes), 0 P0/P1, 1 P2 = complétude de la remontée
      (le fait serveur ci-dessus), ABSORBÉ dans le cadrage du lot I ; 3 constats jetés
      (conventions établies du dossier). Restauration vérifiée par sha1.

DÉCISION UTILISATEUR (05/09) : PAS DE MERGE feat/v75 à ce stade — clôture = revue + gates
+ commit + thought_log sur `wt/game-changers` uniquement ; le merge attendra sa décision
(gate visuel probablement d'abord).

## Lot I — ABANDONNÉ le 05/09 SUR CLARIFICATION UTILISATEUR, avant toute écriture
## retenue (agent arrêté en vol, ses modifications Go défaites, worktree revenu à
## `8180180f1`)

La commande « étendre le repli aux graphes partagés » reposait sur un MALENTENDU du
superviseur : l'utilisateur a clarifié que SEULS les rendus d'USAGE de ce chantier (les
« nouveaux graphes » — bilan d'équipement, contrôle des socles, déjà repliables G1/G2)
sont concernés — « leur rôle n'est pas le même que les précédents ». Les graphes de
PERFORMANCE existants (« Frags par arme » et ses déclinaisons synthèse/escouade/session)
ne se replient PAS. Conséquence : l'exposition de `weapon_key` dans les DTOs d'agrégats
n'a plus de demandeur — abandonnée avec le lot (la note technique reste vraie et
documentée au H1 si un besoin futur la ressuscite). La section ci-dessous est conservée
comme trace de cadrage, AUCUN item n'est à exécuter.

## (trace, non exécuté) Lot I — Exposer `weapon_key` dans les DTOs d'agrégats, puis
## replier les graphes

Constat fondateur (H1, contre-vérifié) : l'identité existe à chaque étage (agrégation
DuckDB par `effective_weapon_id`, `port.WeaponKillRow.WeaponID` jusqu'au service) et
n'est jetée qu'à la construction des DTOs servis — nés « affichage seul ».

Décisions tranchées :
- I-D1 La clé exposée est `weapon_key` (clé CANONIQUE du registre du titre, celle de
  `weapon_names.toml` et du rejeu) — JAMAIS l'id numérique interne. Résolue côté service
  dans la MÊME passe que Class/Role (le résolveur la connaît déjà), champ additif
  `omitempty` : aucun client existant ne casse. Vide si le registre ne résout pas
  (dégradation existante, conservée).
- I-D2 (AMENDÉE le 05/09 sur info utilisateur) DTOs concernés : `SynthesisWeaponKillEntry`
  (+ le chemin fiche match : `MatchFragCard` cesse de JETER l'identité — vérifier sur
  pièces si `MatchWeaponKill` porte déjà une clé exploitable ou s'il faut l'ajouter) et
  `SquadWeaponBar` — les DEUX où `WeaponKey` est déjà résolu serveur. EXCLUS :
  `SquadWeaponAccuracyBar` (agrégé par RÔLE par conception) ET
  `SynthesisWeaponAccuracyEntry`/`SynthesisWeaponAccuracyChart` — la précision PAR ARME
  n'est PAS supportée par Halo Infinite (Halo 5 seulement, chantier Infinite remisé,
  capability retirée) : le vote (clés `hinf_*`) ne s'y applique pas. VÉRIFICATION DE
  SALUBRITÉ à faire au passage (I1, lecture seule, rapport sans correction) : ces
  surfaces de précision sont-elles bien gatées par capability côté Go ET côté web — si
  Halo Infinite les voit encore, c'est un reliquat à remonter.
- I-D3 Côté web : repli des DEUX graphes (frags par arme partagé + kills d'escouade)
  par le prédicat `isGameChangerWeaponKey` RÉUTILISÉ et le contrôle partagé
  `collapsed-items-toggle` (lot H) — mêmes règles que G1/G2 : élus d'abord, tri
  existant conservé dans chaque bloc, replié par défaut, N=0 = pas de bouton, aucun
  agrégat affiché ne sous-compte, entrée sans `weapon_key` = repliée (dégradation
  dite). Hauteurs dynamiques des charts = éléments VISIBLES. **GARDE MULTI-TITRE
  (ajoutée le 05/09)** : ces graphes servent TOUS les titres — le vote est un jugement
  Halo Infinite (clés `hinf_*`). Si la partition rend ZÉRO élu (Halo 5, titre futur,
  vieux artefacts sans clés), AUCUN repli : tout visible, pas de bouton — jamais un
  test de slug, la donnée décide. Règle TESTÉE par mutation sur les deux graphes.
- I-D4 Surfaces Go : domain + service (résolution), openapi régénéré (`make
  openapi-gen`), `make generate-types`, contracttest si ces schémas y sont tenus —
  vérifier sur pièces. Aucune écriture DuckDB, lecture seule des repos existants.

- [ ] I1 Go : `weapon_key` porté par `port.WeaponKillRow` (résolution existante) jusqu'à
      `SynthesisWeaponKillEntry` / `SynthesisWeaponAccuracyEntry` / `SquadWeaponBar` ;
      tests service (entrée avec/sans résolution).
- [ ] I2 Génération : openapi + generated.ts + frontières éventuelles ; `make
      check-types` exit 0.
- [ ] I3 Web : repli des 2 graphes (logique pure par chart, composants rendent),
      `MatchFragCard` transmet la clé au composant partagé.
- [ ] I4 Tests : par graphe (partition, mutation tuée, sans clé = replié, agrégats
      affichés inchangés, N=0) ; garde-rails existants verts.
- [ ] I5 Gates : Go (go test paquets touchés, vet, gofmt) + web (typecheck purgé,
      make test-web, eslint, grep couleurs) + openapi-check.
- [ ] I6 Revue adversariale à contexte frais AVANT commit. PAS DE MERGE (décision user).

## Découvertes (hors périmètre, non traitées)

- **P2 revue (05/09), cosmétique, NON corrigé dans ce diff** :
  `MatchPadControlSection.tsx:178` / `MatchEquipmentUsageSection.tsx:221` — quand TOUS
  les éléments mesurés sont hors vote (artefact sans catalogue : tout replié), le
  conteneur `px-3 pb-3 pt-3` rend une bande de padding vide (~24 px) entre bandeau et
  note de pied. L'état lui-même est voulu et documenté (le bouton d'en-tête reste la
  porte) ; seul le padding résiduel est un artefact. Correctif candidat : padding
  conditionné au contenu visible.
- `i18n.ts` (818 L) et `i18nContract.ts` (862 L) dépassaient déjà 500 L avant ce lot
  (+5/+14 lignes ici, parité oblige) — candidats à découpe, dette antérieure.
- `weaponPadFamilies.test.ts` vérifie les weapon_keys par sous-chaîne alignée sur les
  espaces du TOML — fragile à un réalignement ; le garde-rail G0.2 réutilise le même
  patron par cohérence. À durcir ensemble le jour venu.
- Réalignement éventuel `POWER_PAD_KEYS` (rejeu 2D, contient cindershot) sur le vote :
  décision utilisateur future, hors de ce lot.
- Repli des surfaces EXCLUES par D2 (FragWeaponBreakdown et charts partagés,
  PlayerDetailPanel, squad/synthesis) : à arbitrer séparément si demandé.
