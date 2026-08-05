package duckdb

// kill_events_source.go — LE SUBSTRAT DE LECTURE DU KILL-FEED, en un seul endroit.
//
// Bascule du 2026-08-03 (phase 2 de `.ai/PLAN_BRANCHEMENT_KILLSOURCE.md`) : les lecteurs de
// ce paquet lisaient `killer_victim_pairs`, qui porte 46,5 % de doublons exacts — le flux
// primaire INSERT pendant que la complétion fait DELETE-then-INSERT, sur une table sans PK.
// Les agrégats carrière en étaient gonflés d'un facteur mesuré à 1,879 en moyenne : sur le
// duel de contrôle `Luigi Numba 01 → JGtm`, l'écran affichait **101 frags là où il y en a 29**.
//
// Ils lisent désormais la table canonique, dédoublonnée à l'identité et enrichie par le film.
//
// ─── LA TRADUCTION, COLONNE PAR COLONNE ──────────────────────────────────────────────────
//
//	killer_victim_pairs          match_kill_events_latest
//	───────────────────────      ────────────────────────────────────────────────────────
//	killer_xuid                  feed_killer_xuid       (le CRÉDIT du kill-feed)
//	killer_gamertag              feed_killer_gamertag
//	victim_xuid / _gamertag      victim_xuid / _gamertag
//	kill_count                   — la table est un JOURNAL : 1 ligne = 1 mort.
//	                               `SUM(kill_count)` devient `COUNT(*)`, ce qu'il calculait
//	                               déjà sans le savoir (kill_count vaut 1 sur toutes les lignes).
//	time_ms                      time_ms
//	created_at                   written_at
//
// ─── DEUX PIÈGES MESURÉS LE 2026-08-03, À NE PAS RÉOUVRIR ────────────────────────────────
//
//  1. **LES BOTS SONT DES `xuid` NULL, PAS DES `bid(...)`.** La table canonique porte
//     **0 ligne** en forme `bid(` : les filtres `NOT LIKE 'bid(%'` de Q27/Q28 restent des
//     NO-OP après la bascule, exactement comme avant. Ce qui exclut les bots des agrégats
//     carrière (arbitrage (A) du DDL `steps_shared_kill_events.go`), c'est la sémantique NULL :
//     `NULL <> ?` ne vaut jamais VRAI, donc le groupe des bots tombe. Mesure sur JGtm :
//     71 frags et 12 morts de bot correctement écartés.
//
//     ⚠ COROLLAIRE — **NE JAMAIS NORMALISER UN xuid ABSENT EN CHAÎNE VIDE.** Un
//     `COALESCE(xuid, ”)` fusionnerait les 973 lignes de bot d'Infinite en UN joueur fantôme
//     de chaîne vide, qui frappe et meurt en permanence. Le cas n'est pas théorique : côté
//     **Halo 5**, `killer_victim_pairs` code déjà ses bots en chaîne vide, et ce fantôme entrait
//     dans le top-10 némésis / souffre-douleur sous un libellé masqué (161 frags, 127 morts sur
//     le joueur le plus actif). La bascule le retire. Un lecteur qui remet un `COALESCE(…, ”)`
//     le fait revenir — c'est ce qu'interdit `TestPasDeXuidNormaliseEnChaineVide`
//     (`kill_events_source_guard_test.go`).
//
//  2. **`publishable` NE FILTRE PAS LES LIGNES.** C'est un attribut de la PASSE de décodage :
//     il dit si le film est fiable ligne par ligne sur ce match (faux en BTB, marge de
//     bijection nulle) — 366 matchs sont dans ce cas. Il ne qualifie PAS l'existence de la
//     mort : 99,3 % des lignes de voie film de ces 366 matchs ont leur identité présente dans
//     la base crédit. Filtrer dessus coûterait **47 037 morts** et viderait 27 % des matchs à
//     l'écran (arbitrage superviseur du 2026-08-03). La colonne se lira le jour où une surface
//     affichera l'ARME d'une mort : c'est cet affichage-là qu'elle conditionne.
const KillEventsCanonicalTable = "match_kill_events_latest"

// QKillsBetweenPlayers : frags échangés entre DEUX joueurs, tous matchs confondus.
//
// UNE SEULE COPIE DANS LE DÉPÔT, et c'est délibéré : Q19b (`queries_match.go`, comparaison de
// deux joueurs) et `CompareRepo.GetEncounterStats` en portaient deux versions textuellement
// identiques. Les faire diverger, c'est afficher deux nombres différents pour le même duel sur
// deux pages. Le garde-rail `TestUneSeuleRequeteDeFragsEntreDeuxJoueurs`
// (`kill_events_source_guard_test.go`) échoue si le littéral
// réapparaît ailleurs.
//
// Pas de `COALESCE` autour des `COUNT(*)` : un `COUNT` sur une sous-requête sans ligne rend 0,
// jamais NULL. L'ancienne forme en portait un parce qu'elle faisait `SUM(kill_count)`, qui,
// lui, rend NULL sur l'ensemble vide.
//
// Paramètres : ?1 = tueur A, ?2 = victime B, ?3 = tueur B, ?4 = victime A.
// Retourne 2 colonnes : kills_dealt, deaths_suffered.
const QKillsBetweenPlayers = `
SELECT
    (SELECT COUNT(*) FROM ` + KillEventsCanonicalTable + `
      WHERE feed_killer_xuid = ? AND victim_xuid = ?) AS kills_dealt,
    (SELECT COUNT(*) FROM ` + KillEventsCanonicalTable + `
      WHERE feed_killer_xuid = ? AND victim_xuid = ?) AS deaths_suffered`
