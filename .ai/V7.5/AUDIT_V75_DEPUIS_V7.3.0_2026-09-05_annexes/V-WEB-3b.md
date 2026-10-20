# Vérification adverse V-WEB-3b

Dépôt `C:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration`, branche `feat/v75`, HEAD `736ccf3c3`.
Lecture seule. Constats source : `scratchpad/audit/W4.md` (§ Constats) + W3 P1-6 (`W3.md:309-336`).

## Constat 1 — Cinq tables Halo Infinite en dur dans le TS du rejeu : RÉFUTÉ

- **Ce que j'ai vérifié** — recompte intégral, les cinq comptes de l'auditeur sont EXACTS :
  `replaySound.ts:220-248` = 26 entrées (`grep -c "^  hinf_"` sur 220-250 → 26) ·
  `weaponSoundVariations.ts:12-22` = 9 · `weaponPadFamilies.ts:76-93` = 7 `hinf_*` + 2 `powerup_*` ·
  `gameChangers.ts:55-63` = 7 · `vehicleWeaponMounts.ts:133-142` = 8 `vehicleWeapTag(...)`.
  Puis j'ai ouvert les cinq en-têtes et cherché le champ équivalent dans les manifestes.
- **Ce que l'auditeur n'a pas vu** — quatre choses, dont deux détruisent la règle invoquée :

  1. **AUCUNE des cinq tables ne porte un libellé.** La règle citée (CLAUDE.md « Libellés/assets/outcomes
     via `TitleSemanticAdapter` + TOML — jamais de label FR/EN en dur » ; frontend-patterns « labels de
     stats via `useFieldLabel()`/`useAssetLabel()` ») vise des chaînes affichées. Ce que ces tables
     portent est une RÈGLE DE RENDU :
     - `WEAPON_SOUND_STEMS` = quel FICHIER `.wav` jouer. En-tête `replaySound.ts:19-24` : « Le manifeste
       — les tables ci-dessous […] — est la liste EXACTE des fichiers livrés ; le garde-rail
       `replaySoundAssets.guard.test.ts` le rejoue contre le dossier : un stem sans fichier ou un fichier
       sans stem casse le test ». C'est un inventaire d'assets livrés, pas un dictionnaire.
     - `WEAPON_SOUND_VARIATIONS` = **fichier GÉNÉRÉ** (`weaponSoundVariations.ts:2-4` : « GENERE PAR
       `_outils/livraison.py` […] NE PAS EDITER A LA MAIN : toute reprise rejoue la recette
       (`.ai/V7.5/RECETTE_SONS_ARMES.md`) »). Contenu = paramètres moteur Wwise extraits du jeu
       (`volume_db`, `pitch_cents`). Recette présente au dépôt (`.ai/V7.5/RECETTE_SONS_ARMES.md`, 8 014 o).
     - `POWER_PAD_KEYS` = taille et teinte d'une icône de socle. `weaponPadFamilies.ts:4-9` cite la
       demande utilisateur du 2026-08-18 mot pour mot (« des icônes trop petites seraient inutiles mais
       des trop grosses risquent de polluer ») et `:63` la commande « liste EXPLICITE ».
     - `GAME_CHANGER_WEAPON_KEYS` = hiérarchie d'affichage (visible d'emblée vs replié sous « Voir plus »).
       `gameChangers.ts:4-16` : « PROVENANCE : vote du 2026-09-05 […] commandé par l'utilisateur le
       2026-09-04 » et « décision D1 du plan 2026-09-05 » qui la distingue explicitement de `POWER_PAD_KEYS`.
     - `VEHICLE_WEAPON_MOUNTS` = géométrie 2D (`ax`, `ay`, classe `fixe`/`tourelle`) du point d'où part
       l'éclair de tir sur un sprite. `vehicleWeaponMounts.ts:3-5` cite la demande utilisateur du
       2026-09-03 et `:33-38` la provenance RE (`V3F_TIRS_COVENANT_2026-09-02.md`).

  2. **Les manifestes ne portent PAS déjà ces objets**, contrairement à l'affirmation centrale du constat
     (« Les tables sont pourtant déjà décrites dans les manifestes du titre (`weapon_names.toml`,
     `replay_labels.toml`) »). `grep -n "sound\|fx" config/titles/halo_infinite/mappings/replay_labels.toml`
     (751 L) ne rend que des mots français (« son », « sonore », « sont ») : **aucun champ `sound`, aucun
     champ `fx`**. `weapon_names.toml` ne porte qu'un couple `{en, fr}` par `weapon_key` — ni stem, ni
     fourchette de pitch, ni échelle d'icône, ni vote, ni ancre de montage. Porter ces tables au TOML =
     créer cinq champs qui n'existent nulle part, pas « les y remettre ».

  3. **La jointure au titre est DÉJÀ faite par la clé canonique**, et garde-raillée. Les tables sont
     indexées par le `weapon_key` du titre servi par l'artefact (`doc.weaponLabels[hex].key`,
     cf. `weaponPadFamilies.ts:12-17`, `gameChangers.ts:26-29`) — exactement le modèle que l'auditeur
     désigne comme « bon élève » (`catalogLabel.ts`). Et trois garde-rails rejouent ces clés contre les
     manifestes : `weaponPadFamilies.test.ts:56-58` (`expect(toml).toContain(key)` sur
     `config/titles/halo_infinite/mappings/weapon_names.toml`), `gameChangers.test.ts:68-70` (idem) et
     `:43-47` (familles contre `replay_labels.toml`). Une clé inventée casse le test.

  4. **Le repli est déjà title-safe.** Chaque table documente le comportement sur clé inconnue :
     « Une clé absente de cette table = silence propre » (`replaySound.ts:218`), « un socle qu'on ne sait
     pas nommer reste `classic`, jamais promu » (`weaponPadFamilies.ts:50-52`), « Un label sans `key` […]
     est NON game changer » (`gameChangers.ts:27-29`), « `null` = repli sur le centre du véhicule […]
     JAMAIS une position inventée » (`vehicleWeaponMounts.ts:145-148`). Un second titre ne verrait donc
     pas les données d'Halo Infinite : il verrait la dégradation prévue.
- **Conséquence réelle reformulée** : les cinq tables sont des règles de rendu et des arbitrages produit
  datés (deux d'entre elles générées ou votées), déjà jointes au titre par sa clé canonique et vérifiées
  contre ses manifestes par garde-rail — la doctrine « libellés via TOML » ne s'y applique pas, et le
  reproche « les manifestes les portent déjà » est faux sur pièces ; ce qui subsiste est qu'un futur titre
  à rejeu devra écrire ses propres règles de rendu, ce qui n'est pas la même chose et pas un P1.

## Constat 2 — Sons du rejeu servis depuis `/static/sounds/halo_infinite/` : TIENT (gravité → P2)

- **Ce que j'ai vérifié** — les deux échappatoires proposées par la consigne sont fermées :
  1. Le 4e argument n'a pas de défaut « slug du store » : `lib/staticAssets.ts:38` `export const
     DEFAULT_TITLE_SLUG = 'halo_infinite'` et `:54` `titleSlug: string = DEFAULT_TITLE_SLUG` — un
     **littéral**, pas une lecture de contexte. `replayAudioMix.ts:60-62` appelle
     `staticAssetURL('sound', stem, '.wav')` à trois arguments.
  2. Les sons SONT rangés par titre côté serveur : `ls -d static/sounds/*` → **`static/sounds/halo_infinite/`**
     (seul enfant, donc segment de slug réel), et `server_apiv1.go:1371`
     `http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir)))` sert l'arborescence telle quelle.
     `staticAssets.ts:4` promet bien `/static/{folder}/{titleSlug}/{id}{ext}`.
  3. Les trois autres URL d'asset du rejeu passent le slug — vérifié par
     `grep -rn "staticAssetURL(" apps/web/src | grep -v "\.test\."` : `ReplayAssistMark.tsx:52`,
     `useReplayVehicles.ts:168`, `useReplayWeaponPads.ts:127` portent tous `, titleSlug)`.
- **Ce qui confirme** — le fichier se contredit lui-même : `replaySound.ts:16` écrit « Tout vit sous
  `static/sounds/halo_infinite/` » tandis que `:206` documente la même table comme « stem de fichier sous
  `static/sounds/{slug}/` ». Aucune décision écrite ne justifie le slug figé ; `soundUrlOf` est né d'une
  centralisation de littéral (règle n°6, `replayAudioMix.ts:54-58`) qui a recopié l'appel à 3 arguments.
  Il n'y a donc bien pas de choix, mais un oubli — le constat est exact.
- **Pourquoi j'abaisse la gravité** : l'effet est strictement nul aujourd'hui et le restera tant qu'un
  second titre n'aura pas SON pack de sons livré sous `static/sounds/{slug}/` (les assets sont extraits des
  banks Wwise du jeu, cf. `RECETTE_SONS_ARMES.md`) — c'est-à-dire que le slug ne peut pas devenir faux
  avant qu'un travail bien plus lourd n'ait eu lieu. C'est exactement la classe du P2 que l'audit accorde
  lui-même à `teamLabel.ts` (« l'effet est nul aujourd'hui […] mais un titre qui ne sert pas `team_name`
  recevra les noms d'équipe d'Halo Infinite »). Deux défauts latents de même nature doivent porter la même
  gravité ; les autres P1 de W4 sont, eux, visibles à l'écran sur halo_5 aujourd'hui.
- **Conséquence réelle reformulée** : un oubli d'argument réel et non documenté, sans effet observable et
  sans effet possible avant qu'un second titre n'ait un pack de sons livré — P2, pas P1.

## Constat 3 — « Heatmap des positions » et dictionnaire inline hors garde-rail : RÉFUTÉ

- **Ce que j'ai vérifié** — `MatchPositionsHeatmap.tsx:40-58` porte bien un `TEXT = {fr, en}` inline et
  `:42` `title: 'Heatmap des positions'`. Le fichier est bien dans le périmètre (`git ls-tree a2719a68c`
  sur ce chemin → vide ; `git diff --stat a2719a68c HEAD` → 187 insertions : fichier créé après la base).
  Puis j'ai ouvert le garde-rail invoqué en entier.
- **Ce que l'auditeur n'a pas vu** — trois faits qui vident le constat :
  1. **« heatmap » n'est PAS un mot interdit du dépôt.** `no-anglicisms.guard.test.ts:92-99`,
     `FORBIDDEN_PATTERNS` = exactement six entrées : `PB`, `kill(s)`, `assist(s)`, `streak`, `win rate`,
     `leaderboard`. Ni « heatmap », ni « lobby », ni « power-up ».
  2. **Deux manifestes que le garde-rail SCANNE servent « Heatmap » en français, et passent.**
     `explorer.toml:54` `fr = "Heatmap d'activité commune"` et `timeseries.toml:194`
     `fr = "Heatmap d'intensité (jour × heure)"` — or `explorer.toml` et `timeseries.toml` sont
     explicitement dans la liste scannée (`no-anglicisms.guard.test.ts:233-234`). Le garde-rail les lit et
     les accepte. **Verser le dictionnaire inline, `REPLAY_TEXT` et `USAGE_TEXT` à la liste scannée — le
     traitement que le constat propose — ne changerait donc RIEN** : le mot n'est pas dans les patterns.
  3. **Le garde-rail se déclare non exhaustif, et documente une doctrine de mots assimilés.** En-tête
     `:11-14` : « PÉRIMÈTRE — fichiers traités par I15 + complément I15-bis […] pas un audit exhaustif du
     corpus i18n (dette pré-existante plus large hors périmètre) » ; `:66-68` : « `badge` et `playlist` ne
     sont volontairement PAS dans la liste des patterns interdits (mots jugés assimilés) ». N'être pas
     scanné n'est donc pas une infraction : c'est l'état documenté de l'outil.
  4. La seule décision écrite contre le mot est **de portée rejeu** : `match-replay/i18nContract.ts:499-500`
     « JAMAIS « heatmap » à l'écran (règle FR sans anglicismes) — « carte de chaleur » partout ». Elle vaut
     pour le contrat du rejeu, pas pour la fiche de match ni pour l'Explorer (qui dit « Heatmap » depuis
     avant le périmètre).
  5. Le dictionnaire inline satisfait le seul garde-rail i18n réellement en CI :
     `@levelup/no-hardcoded-strings` en `error` (`eslint.config.js:45`), joué par `npm run lint`
     (`ci.yml:83-84`). Le couple `{fr, en}` typé est la façon de le satisfaire.
- **Conséquence réelle reformulée** : le mot « heatmap » est admis de fait par le dépôt (deux manifestes FR
  sous garde-rail le servent et passent), le garde-rail invoqué ne l'aurait jamais attrapé où qu'on range
  le dictionnaire, et il ne reste qu'une inconstance de vocabulaire antérieure au composant — pas un P1.

## Constat 4 — Anglicisme « lobby » × 7 dans `usageI18n.ts` : RÉFUTÉ

- **Ce que j'ai vérifié** — recompte ligne à ligne (`grep -n "obby" usageI18n.ts`, bloc `fr` = `:100-172`) :
  `:114` « Piste du lobby », `:118` « Mon camp / lobby », `:120` « Joueur / lobby », `:123` « Lobby »,
  `:130` « ${team} (équipe) · ${lobby} (lobby) », `:147` « … · lobby ${l} ». Puis j'ai ouvert la liste
  interdite et le fichier cité comme arbitrage.
- **Ce que l'auditeur n'a pas vu** — trois choses :
  1. **Le compte est 6, pas 7.** La 7e est le jeton `${lobby}` de `:130` — un NOM DE PARAMÈTRE, jamais
     rendu. Le dépôt a tranché ce point de méthode : `no-anglicisms.guard.test.ts:44-51` + la fonction
     `stripInterpolationTokens` (`:121-129`) retirent les jetons ICU avant tout test, précisément « pour
     ne pas produire un faux positif sur son propre nom de variable ». Compter `${lobby}` contredit la
     méthode du dépôt.
  2. **« lobby » n'est pas dans `FORBIDDEN_PATTERNS`** (les six patterns listés au constat 3 ci-dessus).
  3. **L'arbitrage invoqué n'en est pas un.** `engagement.toml:23-25` sert bien `fr = "Partie"` /
     `en = "Lobby"` — mais **pour un seul libellé de série de graphe** (`engagement.trace.lobby`), et le
     MÊME fichier sert « lobby » en prose FRANÇAISE deux fois : `:211` « Part historique du joueur dans
     l'action totale du lobby » et `:257` « L'attendu est ancre sur le lobby : … ». Les « deux vocabulaires
     FR concurrents » dénoncés existent donc déjà À L'INTÉRIEUR du fichier présenté comme la décision. Le
     dépôt n'a pas tranché le mot ; il a nommé une courbe.
  4. Le correctif proposé casserait le sens : ici « lobby » est le DÉNOMINATEUR (les 8 ou 12 joueurs du
     match) opposé à « mon camp » et « mon équipe » — `gaugeTeamOfLobby: 'Mon camp / lobby'`,
     `gaugePlayerOfLobby: 'Joueur / lobby'`. « Mon camp / partie » désignerait le match, pas son effectif.
     Les exemples de la règle CLAUDE.md n°1 (« série » pas « streak », « Taux de victoire » pas « WR »)
     sont des mots à équivalent FR net ; celui-ci n'en a pas dans ce rôle.
- **Conséquence réelle reformulée** : six occurrences (pas sept) d'un mot que le dépôt n'a jamais interdit
  et qu'il sert lui-même en FR dans le manifeste cité comme arbitrage — le point demande une décision
  utilisateur de vocabulaire, il ne constate pas une règle enfreinte.

## Constat 5 — Familles de mode en dur au lieu de `useAssetLabel` : RÉFUTÉ

- **Ce que j'ai vérifié** — `usageI18n.ts:163-169` (FR) / `:233-239` (EN) et le `switch familyLabel`
  `:262-281` existent bien tels que décrits. Puis j'ai vérifié que le mécanisme proposé existe.
- **Ce que l'auditeur n'a pas vu** — le mécanisme invoqué **ne couvre pas cette notion** :
  1. `useAssetLabel(kind, id)` (`lib/i18n/fieldMappings.ts:179-183`) lit `data.assets[kind]?.[id]?.label`,
     alimenté par `config/titles/{slug}/mappings/assets.toml`. Les kinds déclarés côté halo_infinite
     (`grep -o "^\[assets\.[a-z_]*" | sort -u`) sont : `cadence`, `challenge_status`, `challenge_tier`,
     `map`, `medal_tier`, `mode`, `prestige_level`, `season`. **Aucun `mode_family`.**
  2. **Aucune des sept clés n'existe dans `assets.toml`** :
     `grep -in "ctf\|oddball\|koth\|stronghold\|stockpile\|extraction\|vip"` sur le fichier → **0 ligne**.
     Le kind `mode` qui existe porte `Assassin`, `Fiesta`, `Super Fiesta`, `Husky Raid`, `BTB`, `Ranked`,
     `Firefight`, `Other` : des catégories de playlist, une autre notion. `useAssetLabel('mode', 'ctf')`
     rendrait donc `'ctf'` brut (repli documenté `:177` « id absent du kind → retourne `id` ») — soit
     exactement le `familyUnknownFmt` que le constat présente comme le risque à éviter.
  3. **Ces clés ne sont pas un vocabulaire d'Halo Infinite** : `usageI18n.ts:262` les nomme « clés
     narrative du contrat », et elles viennent de l'énumération canonique inter-titres
     `apps/go-api/internal/analysis/narrative/objective_participation.go:35` (`ObjectiveFamily`,
     `FamilyCTF = "ctf"`). Traduire une énumération canonique en TS est le patron des DEUX fonctions
     voisines du même fichier — `roleLabel` (`take`/`defend`/`hold` → Prendre/Défendre/Tenir, `:246-259`)
     et `powerupLabel` (`:284-293`) — que l'audit ne signale pas. Le constat isole une des trois.
  4. Adopter le correctif proposé revient donc à **créer** le mécanisme (nouveau kind `mode_family`
     + peuplement dans les manifestes des deux titres), pas à l'employer. La règle « le mécanisme existe
     et est employé ailleurs (`MediaMatchPicker.tsx:78-79`) » porte sur `useAssetLabel('mode', …)`, un
     autre kind et un autre référentiel.
- **Conséquence réelle reformulée** : la porte invoquée n'existe pas pour cette notion et les sept clés
  sont une énumération d'analyse canonique inter-titres, traduite comme ses deux voisines dans le même
  fichier — la règle « jamais hardcodé » n'est pas applicable telle quelle ici.

## Constat 6 (W3 P1-6) — Le lint couleur canonique n'est pas en CI : TIENT

- **Ce que j'ai vérifié** — les quatre échappatoires de la consigne sont fermées une à une :
  1. `tools/lint-no-hardcoded-colors.mjs:189` `const RATCHET_THRESHOLD = 0`, périmètre
     `apps/web/src/{features,components}/` (en-tête `:5-7`). Il n'apparaît que dans `lefthook.yml:75-77`,
     stage `pre-push` (contournable par `LEFTHOOK=0`).
  2. **Aucun step CI ne l'appelle** : `grep -rn "color" .github/workflows/` ne rend que
     `deploy.yml:217` et `test-deploy-precheck.yml:63` (`./actionlint -color`). Le job `frontend`
     (`ci.yml:61-131`) enchaîne : `npm run typecheck`, `npm run lint`, `npm run lint:fields`, ratchet knip,
     garde feedback-drawer, `npm run build`, `npm run test:coverage`.
  3. **Pas sous un autre nom** : `apps/web/package.json:6-23` — `dev`, `build`, `typecheck`,
     `lint` (= `eslint .`), `lint:fields` (= `lint-no-hardcoded-fields.mjs`, un AUTRE lint), `preview`,
     `test*`, `generate-types`, `analyze`, `knip`. Pas de `lint:colors`.
  4. **Pas de règle ESLint couleur** : `eslint.config.js` ne déclare que deux règles maison
     (`:35-36`, `:45`, `:48`) — `@levelup/no-hardcoded-strings` et `@levelup/no-title-slug-literal` ;
     `grep -n "color\|tailwind\|hex\|no-restricted-syntax"` sur le fichier → 0 ligne.
  5. `make gate-push` ne le rejoue pas non plus (`Makefile:302-310` : golangci-lint, `npm run typecheck`,
     `npm run lint`, `check_test_baseline.sh`).
  6. Les 9 copies partielles existent et **divergent bien** : `fxInk.guard.test.ts:78-82` ne teste que
     `/#[0-9a-fA-F]{6}\b/` et `/oklch\(/` sur 3 fichiers nommés — **aucune classe Tailwind** ;
     `heatmapLayer.guard.test.ts:19-24` porte une regex Tailwind à 14 préfixes sur 2 fichiers ;
     `weaponPads.guard.test.ts` une à 8 préfixes sur 5 fichiers.
- **Ce qui confirme / la seule nuance à porter** : les 9 copies, elles, TOURNENT en CI (vitest,
  `ci.yml:127-131`). La règle n'est donc pas absente du verdict d'autorité — elle y est appliquée à une
  quinzaine de fichiers nommés à la main, avec trois regex inégales, au lieu des ~200 du dossier. Cela
  précise le constat sans le renverser : le gate global reste hors CI et contournable.
- **Conséquence réelle reformulée** : le seul contrôle couleur exhaustif du dépôt ne s'exécute qu'en
  pre-push contournable ; ce qui subsiste en CI est un patchwork de trois regex divergentes sur une
  quinzaine de fichiers choisis à la main.

## Bilan : 1 tient, 4 réfutés, 1 requalifié

- **Tient** : constat 6 (lint couleur hors CI) — avec la nuance que les copies, elles, tournent en CI.
- **Réfutés** : constat 1 (tables = règles de rendu et arbitrages datés, pas des libellés ; les manifestes
  ne portent AUCUN champ équivalent) · constat 3 (« heatmap » n'est pas un mot interdit, et deux manifestes
  FR sous garde-rail le servent déjà en passant) · constat 4 (6 occurrences et non 7 ; le fichier cité comme
  arbitrage sert lui-même « lobby » en FR deux fois) · constat 5 (le kind `mode_family` n'existe dans aucun
  `assets.toml` ; les clés sont une énumération canonique inter-titres, traduite comme ses deux voisines).
- **Requalifié** : constat 2 (sons non title-scopés) — exact sur les faits, mais strictement latent et de
  même classe que le P2 accordé à `teamLabel.ts` : **P1 → P2**.
