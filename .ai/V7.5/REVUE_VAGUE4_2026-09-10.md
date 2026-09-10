# Revue adversariale de la vague 4 — 2026-09-10

Deux relectures en contexte frais, aveugles l'une de l'autre, une ronde chacune, règles du
skill `adversarial-review`. Diff relu : `737e4fe50...HEAD` (4.1 browserslist, 4.2 assets
dédoublonnés, 4.3 identité par le tableau de l'API + table rang → famille, 4.4 véhicule hors
cadre, 4.5 provenance film sur la vue match). Persist touché par 4.3 → deux relecteurs.

## Relecteur A — lentilles L1 (anti-ART) et L4 (données), diff Go + manifestes

- 0 P0, 0 P1. **20 conditions vérifiées qui tiennent** (ordre des étapes du registre, sens
  et unités du calage film/match, bornes exclusives par vie, exclusivité xuid/bid, aucun bot
  en base, symétrie `named_by` + ratchet, aucune écriture DuckDB nouvelle, périmètre du
  bilan identique à l'ancienne table de racines, `FromDamageSource` vrai sur les deux seuls
  producteurs film et faux sur `v_weapon_kills`, goldens et OpenAPI en phase).
- P2 — `cmd/levelup/cmd_backfill_usage_summary.go:285-292` : la reprise ne compare que
  `(rev, schema)` ; jouer `backfill-usage-summary` sur un artefact encore au schéma 50 après
  le bump us5 écrirait un bilan vide marqué à jour `(us5, 50)`, que seule une recuisson
  répare. Sans effet ici (recuisson jouée AVANT le re-résumé), à garder : refuser ou alarmer
  sur `doc.SchemaVersion < 51`. **Consigné.**
- P2 — `internal/analysis/replay/identity_registry_creation.go:374-378` : l'alarme « index
  LU mais absent de la table » soustrait `r.IndexBot` (lecture directe) d'un résidu
  post-tableau ; sur `4f77afc1` (18 − 10 = 8 vies restantes) la condition devient fausse et
  l'alarme ne sort plus, ou sort sous-évaluée. Le résidu reste alarmé par
  `scoreboardReport.alarmer`. **Consigné.**
- P2 (préexistant, non refermé par le lot) — `internal/sync/killcollector/positions.go:484-489`
  (`entreeDuRegistre`) : le collecteur de sync ne passe ni `Bots` ni `Participants` au
  registre ; sur un siège partagé bot → humain, `match_lives` reçoit TOUTES les vies du siège
  sous le xuid de l'humain (`biped_creation`) alors que l'artefact publie `bid(N.0)` avant
  l'arrivée. D11 (« deux producteurs, un seul nommage ») ne tient pas sur ces vies, et la
  voix `tableau_api` n'a aujourd'hui aucun producteur vers `match_lives`. **Consigné.**

## Relecteur B — lentilles L6 (couverture des tests) et L3 (anti-patterns), diff complet `apps/`

- 0 constat recevable. **9 conditions vérifiées qui tiennent**, deux mutations jouées :
  retirer la garde du `bid` dans `poserIdentiteDeVie` fait rougir
  `TestUneDeductionNEcrasePasLeBotDuTableau` ; `TestUsageSummary_JointureSurLaFamillePubliee`
  ne peut passer que si la jointure lit `label.Family`. Vérifiés : `BotID` transporté,
  `FromDamageSource` recopié avec cas réel + cas négatif Halo 5, garde-rail des familles
  étendu, golden borné au schéma + familles, dédoublonnage web déterministe, i18n FR/EN,
  aucune couleur en dur, seuils de taille non aggravés par du code neuf, `generated.ts` en
  phase avec `openapi.yaml`.

## Bilan

P0 = 0, P1 = 0, P2 = 3 consignés (§6 du plan maître). Aucune ronde 2. Suite : `make
gate-push`, push de `feat/v75` = CI de vague.
