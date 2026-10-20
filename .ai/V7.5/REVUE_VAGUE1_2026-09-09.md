# Revue adversariale unique — vague 1 du plan master (2026-09-09)

Diff relu : `feat/v75...feat/vague1-integration` (escouade A0-A3, frise medailles C4, ETag/304
etape 1, grille canonique C1-C2 + migration Synthese, nuage d'isolement C3). Un relecteur
Sonnet en contexte frais, lentilles L5 (front), L6 (tests), L3 (anti-patterns), contrat en six
lignes, filtre de recevabilite du skill `adversarial-review`.

## Constats recevables : 2 — corriges dans le meme lot (commit de cloture de la vague)

| # | Ou | Constat | Triage superviseur | Correctif |
|---|---|---|---|---|
| 1 | `handlers/cache_http.go` (`servirBlobAvecETag`) | `_, _ = w.Write(blob)` : la coupure client (pipe casse) n'est plus journalisee, alors que `replay.go` et `tactical.go` la loggaient avant la centralisation | Propose P2 par le relecteur, **requalifie P1** (regle 3 CLAUDE.md, anti-pattern n 10) | `slog.WarnContext` avec `err`, `path`, `content_type`, `octets` ; test `TestServirBlobAvecETag_EcritureInterrompue_Journalisee` (ecrivain casse, journal capture) |
| 2 | `service/teammates/teammates_service_composition_sessions.go` (`groupExcludedBySession`) + contrat `domain.CompositionExcludedMatch` | Un match du roster sans AUCUNE ligne d'allie chargee est ecarte avec `ExtraGamertags = []`, alors que le contrat ecrit « jamais vide sous exclusion » ; aucun test Go ne couvrait la couverture partielle | P1 (contrat faux) | Le comportement est le bon (rien a inventer) : le CONTRAT est corrige (vide uniquement si l'equipe alliee est inconnue, le web rend « coequipier inconnu »), test `TestGetPage_ExactComposition_RosterMatchSansEquipeConnue` |

Constats jetes : 0. Conditions verifiees qui tiennent : 8 (ETag fort, analyse `If-None-Match`,
304 sans corps, garde-rail grep discriminant, `MatchCountRoster >= MatchCount`, repli
« Joueur <4 derniers> », allowlist heatmap a 4 sites, rampe divergente centree a 50 %,
zero couleur en dur, parite FR/EN).

## Ronde 2 : non jouee (decision superviseur)

Les corrections tiennent en une ligne de journalisation, deux commentaires de contrat et deux
tests ; une seconde relecture en contexte frais couterait plus qu'elle ne couvre (sobriete des
quotas, consigne utilisateur du 09-09). P0+P1 : 2 -> 0.
