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
- Armes EN AVANT (weapon_key, 5+1 variante) : `hinf_s7_sniper`, `hinf_m41_spnkr`
  (+ `hinf_fuel_rod_spnkr`, variante du même socle), `hinf_energy_sword`,
  `hinf_gravity_hammer`, `hinf_skewer`. REPLIÉ notamment : `hinf_cindershot` (voté non).

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
