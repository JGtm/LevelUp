# PLAN — Kill feed du rejeu : victimes (bug bots), médailles (identité perdue), glyphe ami

> Date : 2026-09-02. Pilote : session principale (ce plan est exécuté par des agents
> exécuteurs pilotés — Opus pour les lots Go, Sonnet possible pour le lot web).
> Contrat d'exécution : skill `plan-execution` (ordre strict, aucun report d'étape
> exécutable, statuts obligatoires `[x]`/`[~]`/`[!]`, zéro fix hors périmètre —
> les découvertes vont en fin de fichier, section « Découvertes »).
>
> Diagnostic source : entrée `.ai/thought_log.md` du 2026-09-02 « Diagnostic kill feed
> du rejeu ». Témoin : match `1b2d9e08-4c0c-430c-9760-a245d48b222e` (JGtm, 01/09,
> 82 kills feed, 76 paires humaines + 6 kills sur « 343 Razzle [bot] » + 1 kill par le bot).

## Objectif et critère de succès

1. **Lot A — victimes** : sur un match à bot, la Match View et le rejeu retrouvent
   100 % de leurs surfaces dérivées des paires (victimes du fil, Dominance/tug, KD
   timeline, antagonistes, némésis Équipe) ; les victimes bot sont NOMMÉES au fil
   (« 343 Razzle [bot] »). Zéro WARN `kv_pairs indisponible` sur le témoin.
2. **Lot B — glyphe** : plus aucun glyphe « ami » dans le fil des éliminations
   (rejeu ET Match View si partagé) ; la carte garde sa grammaire de marques.
3. **Lot C — médailles** : un match synchronisé aujourd'hui affiche ses médailles au
   fil ; les matchs post-avril existants sont rattrapés par backfill des films en cache.

Branche : `feat/v75` (mode branche unique — lots = commits). Exécution en worktrees
dédiés : `wt/killfeed-bots` (lot A), `wt/feed-glyphe` (lot B), `wt/medailles-feed`
(lot C). Jamais deux builds/tests Go en parallèle (corruption de cache mesurée) :
lot A puis lot C séquentiels ; lot B (web) peut courir en parallèle du lot A.

## Décisions tranchées (avant exécution — ne pas rouvrir en cours de lot)

- **D1** : le fix victimes est côté SCANNER (`sql.NullString`), JAMAIS un
  `COALESCE(xuid, '')` au SQL — piège documenté du joueur fantôme
  (`kill_events_source.go` en-tête, garde `TestPasDeXuidNormaliseEnChaineVide`).
- **D2** : sémantique `domain.KVPairRaw` : xuid vide `""` = identité absente (bot).
  Les AGRÉGATS (tug, KD, antagonistes, némésis, synthèse H5) restent HUMAINS
  SEULEMENT (sauter toute paire à xuid vide — sémantique d'avant le 03/08). Seul le
  KILL FEED apprend à nommer une victime bot par son gamertag.
- **D3** : identité médaille = couple `(type_hint, medal_type)` du film. Mesure du
  2026-09-02 : bijection parfaite sur 44 568 events pré-avril (124 clés ↔ 124 noms,
  0 clé ambiguë). La table de correspondance est GÉNÉRÉE depuis ce corpus, validée
  contre `metadata.medal_definitions.name_en`, et versionnée en Go (clé de
  référentiel anglaise = clé canonique, pas un label UI ; la locale reste résolue
  par `medal_definitions` comme aujourd'hui).
- **D4** : le collector écrit `type_hint` ET `raw_json` (`{"medal_name": ...}`
  résolu via la table D3) sur les events medal — le chemin de lecture existant
  (`medalNameFromRawJSON` → `LookupMedalMetaByName`) ne change PAS, et les vieux
  matchs restent compatibles. Halo 5 (`DetailsJSON`→type_hint numérique) ne bouge pas.
- **D5** : le glyphe « ami » disparaît du FIL uniquement ; l'encre `success` sur les
  noms marqués RESTE (elle dit encore l'identité sans coûter un signe par ligne).
  La carte et sa forme de point ne bougent pas.
- **D6** : backfill médailles = step de migration AU NOM NEUF (jamais élargir un step
  déployé), UPDATE des lignes medal existantes (`raw_json`, `type_hint`) sous accès
  exclusif, SÉRIALISÉ match par match, avec plafond de lot et log de progression.
  Aucun `highlight_events` n'a d'index UNIQUE : l'UPDATE ne peut pas violer de clé.
  Le run réel sur la DB locale exige le serveur ARRÊTÉ (mono-process) et l'accord
  du user au moment du run.

---

## Lot A — Q20 NULL-safe : les victimes reviennent (exécuteur Opus, worktree `wt/killfeed-bots`)

### Étapes

- [ ] A1. `platform/duckdb/match_view_repo_extras.go` (`GetMatchKVPairs`, ~L183-217) :
      scanner `killer_xuid`, `killer_gamertag`, `victim_xuid` en `sql.NullString`
      (`victim_gamertag` et `time_ms` sont NOT NULL au DDL — scan nu conservé),
      mapping `.Valid ? String : ""` vers `KVPairRaw`. Documenter la sémantique D2
      sur le struct (`domain/match_view_raw.go:380-390`).
- [ ] A2. Supprimer l'avalement silencieux du 2e niveau : dans `GetMatchKVPairs`,
      l'erreur de `QueryContext` reste tolérée (table absente, commentaire existant),
      mais une erreur de SCAN doit remonter — elle remonte déjà ; vérifier qu'après
      A1 il ne reste AUCUN chemin qui rende `nil, nil` sur données présentes.
- [ ] A3. `service/match_view_killfeed_weapon.go` : `victimsByKill` (~L74-91) accepte
      une victime à xuid vide si `VictimGT != ""` (clé inchangée tueur+instant ; la
      garde d'unanimité compare le gamertag quand les deux xuids sont vides).
      `decorateVictim` (~L175-186) ne pose `VictimXUID`/`VictimTeamID` que s'ils
      existent, pose toujours le gamertag.
- [ ] A4. Garde manquante mesurée (agent d'audit B) : `buildTugEvents`
      (`match_view_builders_combat.go:220-231`) compte TOUTES les paires — ajouter
      le skip des paires à xuid vide (D2). Vérifier sur pièces que `buildKDEvents`,
      `buildKillerVictimPairs`, `buildNemesisMap` (`match_view_builders_team.go:304-314`)
      et `SynthesizeKillEventsFromKVPairs` sautent déjà les xuids vides — sinon,
      même garde.
- [ ] A5. Tests : (a) test repo DuckDB `:memory:` — fixture `match_kill_events` avec
      1 ligne bot-victime + 1 ligne bot-tueur + 2 humaines → `GetMatchKVPairs` rend
      les 4 paires sans erreur ; (b) test `victimsByKill`/`decorateVictim` cas bot
      (victime nommée sans xuid, pas d'équipe) ; (c) test `buildTugEvents` ignore
      les paires à xuid vide ; (d) test de non-régression `decorateKillFeed` complet
      sur mini-fixture à bot.
- [ ] A6. Vérification frontend SUR PIÈCES (lecture seule, aucun code attendu) :
      `KillLine` rend une victime par `victimGamertag` seul (`ReplayKillFeed.tsx:474-479`)
      et `marks.get('')`/`allyOf` tolèrent le xuid vide. Consigner le constat.

### Gate A (commandes exactes, code de sortie vérifié)

```powershell
$env:Path = "C:\msys64\ucrt64\bin;$env:Path"; $env:CGO_ENABLED = "1"; $env:CC = "C:\msys64\ucrt64\bin\gcc.exe"
go -C apps/go-api test ./internal/service/... ./internal/platform/duckdb/... ./internal/domain/...
go -C apps/go-api vet ./...
```

Puis validation témoin (serveur relancé) : GET de la vue match du témoin → les 82
kills portent 76 victimes humaines + 6 « 343 Razzle [bot] » ; plus aucun
`kv_pairs indisponible` dans `logs/service.log` pour ce match. Filtre d'échecs
ancré `^--- FAIL:` + `$LASTEXITCODE`.

---

## Lot B — retrait du glyphe « ami » du fil (exécuteur Sonnet, worktree `wt/feed-glyphe`, parallèle au lot A)

### Étapes

- [ ] B1. `ReplayFeedName.tsx` : ne plus rendre `PlayerMark` (le fil n'affiche plus
      AUCUN glyphe — `me` était déjà retiré, `friend` part avec D5). L'encre
      `success` (`feedNameInk`) et le `sr-only` du joueur actif restent. Mettre à
      jour l'en-tête du fichier ET celui de `ReplayKillFeed.tsx` (règles C1)
      — pas de doc inversée.
- [ ] B2. Chasse au code mort (règle n°7) : si `PlayerMark` n'a plus d'usage dans le
      FIL, retirer l'import ; vérifier ses autres usages (carte via `ReplayCanvas`,
      légendes) avant toute suppression plus large — la carte le GARDE (D5).
      `marksByGamertag`/`marks` restent nécessaires (encre success). Statuer.
- [ ] B3. Tests : adapter `ReplayKillFeed.test.tsx` (assertions de glyphe) ; ajouter
      l'assertion inverse (aucun `PlayerMark` rendu dans une ligne de fil avec ami).

### Gate B

```powershell
Remove-Item -Recurse -Force apps\web\node_modules\.tmp -ErrorAction SilentlyContinue
cd apps\web; npm run typecheck; npm run lint; npm run test
```

(vitest hors sandbox si nécessaire — piège connu). Gate visuel : la main au user
sur le témoin (le pilote fournit URL + points à vérifier).

---

## Lot C — médailles : identité au sync + backfill (exécuteur Opus, worktree `wt/medailles-feed`, APRÈS lot A mergé)

### Étapes

- [ ] C1. Générer la table de correspondance D3 : requête corpus
      (`(type_hint, medal_value) → medal_name`, 124 entrées, 0 ambiguë — requête du
      diagnostic dans le thought_log), croiser avec `medal_definitions.name_en`
      (noms absents du référentiel = consignés, pas inventés). Livrer en constante
      Go + test de complétude (124 entrées, unicité des clés) + garde-rail : un
      test qui échoue si une clé du corpus de test n'est pas couverte.
      Emplacement : `internal/games/halo_infinite/` (savoir film title-specific),
      exposé au collector via l'adapter existant — PAS de `slug ==` (capability).
- [ ] C2. `sync/collect.go:115-123` : remplir `TypeHint` (manquant pour TOUS les
      events depuis la bascule persist) et, pour les medal, `RawJSON`
      `{"medal_name": "..."}` via C1. `persist.HighlightEventInsert` gagne les
      champs nécessaires ; `persistHighlightEvents` (`shared_persister.go:432-448`)
      écrit `type_hint` ET `raw_json` en colonnes SÉPARÉES — corriger au passage le
      croisement actuel (DetailsJSON versé dans type_hint) SANS casser Halo 5
      (`games/halo_5/ingest/medals.go:62-77` vise type_hint numérique : router
      explicitement DetailsJSON→type_hint pour H5, RawJSON→raw_json pour tous).
      Multi-titre : aucun comportement nouveau pour un titre sans table C1
      (dégradation = raw_json absent, comme aujourd'hui).
- [ ] C3. Tests : unit collector (event medal → insert avec type_hint + raw_json
      correct ; event kill → type_hint seul), unit persister (colonnes séparées,
      NULL propres), non-régression H5 ingest (detail numérique intact).
- [ ] C4. Backfill (D6) : step de migration AU NOM NEUF qui, pour chaque match ayant
      des events medal à `raw_json IS NULL` (mesure : 415 matchs / 22 031 events),
      relit le chunk highlight du film EN CACHE (`killcollector` cache films),
      re-parse (`analysis.ParseHighlightEvents`), apparie par (xuid, time_ms,
      event_type) et UPDATE `raw_json` + `type_hint`. Film absent du cache = match
      consigné et sauté (best-effort compté, jamais silencieux). Sérialisé, plafond
      par run, `slog.InfoContext` de progression (`match_id`, compteurs).
- [ ] C5. Gate intégration OBLIGATOIRE (persist/sync/migration touchés) puis run du
      backfill sur la DB locale : DEMANDER au user (serveur à arrêter). Validation
      témoin : le match `1b2d9e08...` affiche ses 32 médailles au fil ; comptage
      recoupé avec `medals_earned` (ordre de grandeur par joueur).
- [ ] C6. Revue adversariale (skill `adversarial-review`) du diff persist/sync avant
      merge — lot à risque au sens CLAUDE.md.

### Gate C

```powershell
$env:Path = "C:\msys64\ucrt64\bin;$env:Path"; $env:CGO_ENABLED = "1"; $env:CC = "C:\msys64\ucrt64\bin\gcc.exe"
go -C apps/go-api test ./...
go -C apps/go-api test -tags=integration -p 1 ./...   # code de sortie vérifié, filtre ^--- FAIL:
go -C apps/go-api vet ./...
```

---

## Clôture (pilote)

- [ ] Chaque lot : entrée thought_log + commit sur `feat/v75` (demander avant commit),
      worktree nettoyé après merge.
- [ ] CI de branche verte AU NIVEAU JOB après le dernier merge (`gh run list`).
- [ ] Registre des reports : toute découverte non traitée y entre avec sa condition
      de reprise.
- [ ] Mémoire : mettre à jour les mémoires chantier si un invariant nouveau est né
      (ex. table D3).

## Reprise de session

Lire cette section + les cases ci-dessus. L'avancement fait foi ICI (statuts), le
détail dans le thought_log. Prochain geste si tout est vide : lancer lot A et lot B.

## Découvertes (hors périmètre — ne pas traiter)

- (vide)
