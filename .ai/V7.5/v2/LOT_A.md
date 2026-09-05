# Lot A — Faits du film (Go) — journal d'exécution

> Plan : `.ai/PLAN_V2_REJEU_FILM_2026-09-05.md`, lot A. Branche `feat/v2-faits`, worktree
> `LevelUp-wt-v2-faits`. Contrat : skill `plan-execution`.
> Statuts : `[x]` fait et vérifié · `[~]` couvert ailleurs (réf.) · `[!]` non traité (justification).

## Tâche A-I

### [x] A.1 (P0-1) — révision du décodeur bumpée, garde-rail d'empreinte posé

**Vérification sur pièces avant de coder.** `internal/sync/killcollector/collector.go:65` portait
bien `KillSourceDecoderRev = "killsource-2026-07-31"`. Mesure du constat rejouée sur la branche :

```
git log --oneline v7.3.0..HEAD -- apps/go-api/internal/games/halo_infinite/film/killsource/ | wc -l
  -> 14
git log -G'KillSourceDecoderRev = ' --oneline v7.3.0..HEAD -- apps/go-api/internal/sync/killcollector/collector.go
  -> 36fc76835 refactor(sync): J4 — le collecteur va dans son sous-paquet  (déplacement de paquet, PAS un bump)
```

**Fait.** `KillSourceDecoderRev = "killsource-2026-09-05"`. Constante `killSourceDecoderFingerprint`
figée juste en dessous (sha256 des 20 sources non-test du paquet décodeur). Nouveau test
`internal/sync/killcollector/decoder_rev_fingerprint_test.go` :

- `TestKillSourceDecoderRevSuitLeDecodeur` — hache
  `internal/games/halo_infinite/film/killsource/` (récursif, `_test.go` et `testdata/` exclus,
  chemin relatif haché à côté du contenu, CRLF normalisés en LF pour que le gate soit identique
  sur la CI Linux et sur un poste Windows) et compare à la constante. Message d'échec : les DEUX
  gestes à faire (bumper la révision, recopier l'empreinte mesurée), avec la conséquence de
  chacun.
- `TestEmpreinteSourcesGoMord` — la preuve que le garde-rail mord, sur quatre propriétés : un
  octet de source changé fait bouger l'empreinte ; un renommage aussi ; un `_test.go` ou une
  fixture `testdata/` ne la bouge PAS ; le même contenu en CRLF donne la même valeur.

**Preuve par mutation manuelle** (exigée par le plan) :

```
printf '\n// mutation temporaire...\n' >> internal/games/halo_infinite/film/killsource/doc.go
go test ./internal/sync/killcollector/ -run TestKillSourceDecoderRevSuitLeDecodeur
  -> FAIL   attendue b272f221909247fd6f8e2c1cca01d4136ec9b317fbecd31488e3eda48bc59379
            mesurée  2f29297b39d9e59557efb68e4623bad41afaa6414d52b7390c35cbd52ac68d5c
git checkout -- apps/go-api/internal/games/halo_infinite/film/killsource/doc.go
go test ./internal/sync/killcollector/ -run TestKillSourceDecoderRevSuitLeDecodeur
  -> ok  levelup/go-api/internal/sync/killcollector  0.118s
```

**Question posée par le plan : les familles « tirs » et « positions » partagent-elles cette
révision pour leur REPRISE ?** Réponse mesurée : **elles n'ont aucune reprise du tout**. Le dépôt
ne contient que DEUX prédicats de reprise, et les deux lisent la même colonne :

| Site | Prédicat |
|---|---|
| `internal/sync/killcollector/postsync.go:375` (`conditionBacklog`) | `match_kill_events_latest.decoder_rev = KillSourceDecoderRev AND read_path <> credit-backfill` |
| `cmd/levelup/cmd_backfill_killsource.go:403` (`matchsAJour`) | idem |

Vérification : `grep -rn "decoder_rev" --include=*.go` hors tests ne rend, en contexte SQL, que ces
deux sites plus une migration (`steps_shared_kill_events_credit_base.go:136`, agrégat de rebuild).
Conséquences, toutes documentées ici et **non traitées dans ce lot** (le plan l'interdit
explicitement : « pas de nouvelle révision dans ce lot ») :

- `match_weapon_shots` porte sa propre constante `WeaponShotsDecoderRev = "filmshots-2026-08-01"`
  (`shots.go:49`), écrite sur chaque ligne — mais **aucun prédicat ne la lit**. La passe de tirs
  est une PASSAGÈRE de la passe de morts (un film téléchargé, quatre tables écrites,
  `collector.go:291`). Un changement du seul producteur de tirs ne déclenchera donc jamais de
  redécodage ; inversement le bump ci-dessus rend les tirs de tous les matchs à nouveau candidats.
- `match_weapon_hit_distance` porte `WeaponHitDistanceDecoderRev = "whd-v1"`
  (`internal/migration/steps_shared_weapon_hit_distance.go:75`) — même situation : écrite, jamais
  lue par une reprise.
- `kill_positions` **n'a pas de colonne `decoder_rev` du tout**, et c'est documenté comme un choix
  (`internal/persist/kill_position_persister.go:18` : la clé fonctionnelle + `written_at`
  suffisent). Elle n'a donc ni révision ni reprise propre : passagère elle aussi.

Autrement dit : les trois familles dépendent aujourd'hui de `KillSourceDecoderRev` pour être
redécodées, sans le dire nulle part. Le traitement (« une révision par famille de lignes »,
proposé par l'audit P0-1) reste ouvert — consigné en découverte D-1 ci-dessous.

**Gate A.1** (commandes exactes, dernière ligne de sortie) :

```
go build ./...                                       -> (aucune sortie) EXIT=0
go test ./internal/sync/...                          -> ok levelup/go-api/internal/sync/v2  15.427s   EXIT=0
golangci-lint run ./internal/sync/killcollector/...  -> 0 issues.  EXIT=0
```

### [x] A.2 (G4) — `kill_positions` et `match_weapon_hit_distance` enrôlées dans les deux listes

**Vérification sur pièces avant enrôlement** (les deux tables doivent être append-only AVEC vue
`_latest`, sinon l'enrôlement serait un mensonge) :

| Table | Migration | Forme | Vue `_latest` | Écrivains |
|---|---|---|---|---|
| `kill_positions` | `games/halo_infinite/migrations/steps_appendonly_misc.go:53` (rebuild G.2, 2026-08-30) | id PK `kill_positions_seq` + `written_at` | `kill_positions_latest`, `QUALIFY ROW_NUMBER() … PARTITION BY (match_id, killer_xuid, time_ms)` | `persist/kill_position_persister.go` (film Infinite) et `persist/shared_persister.go` `persistKillPositions` (builder Halo 5) — **INSERT purs tous les deux** |
| `match_weapon_hit_distance` | `migration/steps_shared_weapon_hit_distance.go:104` (créée append-only) | id PK seq + `decode_pass` + `decoder_rev` + `written_at` | `match_weapon_hit_distance_latest`, dernière PASSE par match | `persist/weapon_hit_distance_persister.go` — un seul statement, INSERT pur |

Note de forme : `kill_positions` n'a volontairement pas de `decoder_rev` (`written_at` arbitre,
cf. l'en-tête du persister) ; l'unité de génération y est LA LIGNE, alors que
`match_weapon_hit_distance` supersède par PASSE entière. Les deux formes sont couvertes par les
mêmes garde-rails (aucun n'inspecte la clé de partition).

**Fait.** Les deux noms ajoutés à `tablesProtegees` (`internal/sync/no_art_patterns_test.go`) et à
`appendOnlyStateTables` (`internal/sync/append_only_state_guard_test.go`), avec la justification
et la date en commentaire. **Aucune allowlist agrandie** : `allowlistArtPatterns`,
`allowlistRawDelete` et `allowlistMediaMutation` restent vides. Le `\b` final des motifs ne
déborde pas sur les vues (`_` est un caractère de mot) : les lectures via `_latest` ne sont pas
touchées.

**Preuve que l'enrôlement mord** (deux violations injectées, puis annulées) :

```
# ajout temporaire d'un `DELETE FROM kill_positions …` dans persist/kill_position_persister.go
# et d'un `INSERT INTO match_weapon_hit_distance … ON CONFLICT (…) DO UPDATE …` dans
# persist/weapon_hit_distance_persister.go
go test ./internal/sync/ -run 'TestNoRawDeleteOnAppendOnlyTables|TestNoMutationOnAppendOnlyStateTables|TestNoARTPatternsOnProtectedTables'
  -> FAIL  - DELETE FROM kill_positions dans internal/persist/kill_position_persister.go
           - INSERT ON CONFLICT/REPLACE/IGNORE sur match_weapon_hit_distance dans internal/persist/weapon_hit_distance_persister.go
           - table=match_weapon_hit_distance pattern_detected file=internal/persist/weapon_hit_distance_persister.go
           - table=kill_positions DELETE brut file=internal/persist/kill_position_persister.go
git checkout -- (les deux fichiers)
  -> ok  levelup/go-api/internal/sync  74.215s
```

**Gate A.2** :

```
go test ./internal/sync/ -run 'ART|AppendOnly|Mutation|Allowlist|Delete|Bulk' -v
  -> ok  levelup/go-api/internal/sync  44.897s   (11 tests, 0 FAIL, 0 SKIP)
golangci-lint run --new-from-merge-base=origin/main ./internal/sync/...
  -> 0 issues.  EXIT=0
```

Note : `golangci-lint run ./internal/sync/` NON ratcheté remonte 15 problèmes, tous
**préexistants** (goconst `weapon_kills`/`match_registry`, argument-limit de `citations.go` et
`engine_v2bridge.go`, `unused` de `convergence.go`/`engine_e2e_test.go`, SA4006 de `engine.go`) —
dette gelée par la baseline, aucune sur les deux fichiers modifiés. Le gate d'autorité est le
ratchet `--new-from-merge-base` (Makefile:307), vert.

### [ ] A.3 (A0)

## Gates de la tâche A-I

(à remplir à la clôture)

## Découvertes (hors périmètre, NON traitées)

- **D-1** — Aucune des trois familles passagères de la passe film (`match_weapon_shots`,
  `match_weapon_hit_distance`, `kill_positions`) n'a de prédicat de reprise propre : leur seule
  reprise possible passe par `KillSourceDecoderRev`, alors que deux d'entre elles portent une
  révision distincte qui n'est lue nulle part et que la troisième n'en a pas. Détail et mesures
  ci-dessus (A.1). Le plan interdit d'ajouter une révision dans ce lot.

## Questions ouvertes

(aucune à ce stade)
