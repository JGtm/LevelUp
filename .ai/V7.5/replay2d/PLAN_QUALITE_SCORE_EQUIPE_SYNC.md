# PLAN — Qualité de données : le « score d'équipe » de l'API n'est pas le score affiché

Date : 2026-08-24. Origine : REGISTRE_REPORTS ligne « L'API Teams[].Stats.CoreStats.Score
n'est pas le score affiché » (lot A phases 0-bis/0-ter, mesures dans
`.ai/V7.5/replay2d/registre_film/LOTA_PHASE0.md`) + backlog Notion « Rejeu 2D — bilan du
18/08/2026 », § À savoir. Go utilisateur : 2026-08-24.

Branche : `wt/qualite-score` (worktree dédié, base feat/v75). Exécution sous le contrat du
skill `plan-execution` (ordre strict, gates, statuts `[x]`/`[~]`/`[!]`, zéro fix hors
périmètre — les découvertes se consignent en §Découvertes, jamais ne se corrigent).

## Objectif et critère de succès

Établir, preuves à l'appui, D'OÙ peut venir un « score d'équipe » AFFICHABLE pour les modes
où `match_registry.team_0_score/team_1_score` ne le porte pas (Strongholds = ticks,
KOTH:Arena = tantôt secondes tantôt collines), et livrer un VERDICT + OPTIONS chiffrées au
superviseur. Ce lot S'ARRÊTE au verdict : AUCUNE écriture DB, AUCUN changement de sync ni
d'affichage sans arbitrage explicite du superviseur (phase 2 = gate d'arbitrage).

Succès = les trois questions du §Phase 1 ont chacune une réponse mesurée et sourcée, et le
rapport `RAPPORT_QUALITE_SCORE_EQUIPE.md` propose 2-3 options d'implémentation avec leur
coût (colonne nouvelle + backfill / correction à l'affichage / statu quo documenté).

## Faits établis (ne pas re-mesurer, ne pas re-discuter)

- `7344d24f` (Strongholds:Arena, Vagabond) : le jeu affiche 200/126, l'API donne 193/112
  (= émissions de ticks - 1 ; 200 = plafond = victoire). Détail : LOTA_PHASE0.md §A.0.1.
- `606d9844` et `8076f97f` (KOTH) : le film porte 0/3 et 3/0 (collines/manches), l'API
  105/8 et 78/105 (secondes de garde) — la MÊME colonne change de sémantique selon le match.
- Le film Theater porte le score affiché (`coverage.score.oracle = "displayed"`, schéma 12+),
  mais seuls les matchs avec artefact de rejeu en disposent (39 artefacts aujourd'hui).
- Règle du registre : ne JAMAIS rebaisser le seuil d'un gate sur l'oracle `team_score`.

## Hors périmètre (fermé)

- Toute écriture DuckDB (colonne, backfill, migration) — phase d'arbitrage d'abord.
- Le rejeu 2D et ses artefacts (le score du rejeu est déjà juste, oracle « displayed »).
- La refonte des KPI dérivés (ADR 0006) ; les modes PvE ; Halo 5 (mesurer Infinite d'abord,
  noter en §Découvertes si la même colonne H5 semble suspecte).

## Phase 0 — Recensement des surfaces qui lisent team_0/1_score

- [x] 0.1 Recenser TOUTES les lectures Go de `team_0_score`/`team_1_score` (grep sur
      `apps/go-api/`, colonnes SQL et champs struct qui en dérivent) : fichier:ligne, ce que
      la surface affiche (en-tête de match, media, dominance, historique, autre).
      -> 14 chaînes L1..L14 dans le rapport §0.1, plus 4 familles de faux positifs écartées.
- [x] 0.2 Recenser les surfaces web qui affichent ces valeurs (chaîne handler -> composant).
      -> 7 entrées W1..W7 dans le rapport §0.2.
- [x] 0.3 Dire pour chaque surface si l'anomalie est VISIBLE par l'utilisateur (ex. un
      Strongholds affiché « 193-112 » au lieu de « 200-126 ») ou masquée (outcome seul).
      -> rapport §0.3 : 5 surfaces visibles (W1..W5), Explorer aggravée par un tri
      inter-modes ; rejeu 2D et dominance Infinite non impactés.

Gate 0 : tableau exhaustif dans le rapport (aucun « etc. ») ; commande de contrôle fournie
dans le rapport (le grep exact rejouable). Clore la phase avant d'ouvrir la phase 1.

## Phase 1 — Diagnostic données : existe-t-il une source du score affiché ?

- [ ] 1.1 JSON brut API : re-télécharger les stats des 3 matchs fautifs (`7344d24f`,
      `606d9844`, `8076f97f`) via le wrapper API existant (Grunt/SPNKr ; tokens via
      `MultiUserTokenStore`, JAMAIS de re-capture — un RT valide se rafraîchit) et inspecter
      `Teams[].Stats` EN ENTIER : y a-t-il un autre champ (round scores, objective stats,
      score par manche) qui porte 200/126, 3/2, 105 ? Dumper les JSON dans le dossier du lot.
- [ ] 1.2 Ampleur : sur `match_registry` (lecture seule), mesurer par mode (pair_name /
      famille de modes, jamais un slug de titre) la population concernée : combien de
      Strongholds (score = ticks), combien de KOTH, et pour KOTH la proportion
      secondes vs collines (heuristique à documenter : un score de KOTH affiché est <= 5
      en Arena ; > 20 = secondes). Chiffres exacts, requêtes jointes au rapport.
- [ ] 1.3 Oracle croisé : pour les matchs qui ONT un artefact de rejeu (39), confronter
      colonne API vs score affiché du film — proportion de matchs où l'écart existe, par mode.
- [ ] 1.4 Verdict : une des trois issues, écrite et argumentée —
      (a) l'API porte le score affiché ailleurs (nommer le champ, couverture mesurée) ;
      (b) l'API ne le porte pas, mais il est CALCULABLE par règle de mode (donner la règle
          et son taux de réussite sur l'échantillon) ;
      (c) ni l'un ni l'autre — seul le film le porte (statu quo documenté).

Gate 1 : les 3 matchs fautifs expliqués par le verdict ; toute règle proposée testée sur
l'échantillon 1.2 avec taux chiffré. STOP : rendre le CR au superviseur. La phase 2
(implémentation) n'existe pas dans ce lot — elle sera un lot séparé après arbitrage.

## Garde-rails d'exécution

- DuckDB : lecture SEULE, via `OpenReadForQuery` / CLI existants (`cmd/diag_q`) — le serveur
  air local peut tenir les DB en RW ; ne jamais ouvrir en RW, ne jamais forcer OpenReadOnly.
- Les données vivent dans le dépôt PRINCIPAL (`c:/Users/Guillaume/Downloads/Scripts/LevelUp-go-migration/data/`) ;
  le worktree n'a pas de `data/`.
- Commandes `go` : UNE à la fois (jamais en parallèle), avec un GOCACHE PRIVÉ au lot
  (`$env:GOCACHE` vers un dossier dédié) — le cache partagé se corrompt sous concurrence.
- Réseau : uniquement l'API Halo officielle pour 1.1 (c'est la source de la sync) ; pas de
  re-capture de token ; si l'auth échoue, diagnostiquer et CONSIGNER, ne pas « réparer ».
- Logging Go : slog structuré si du code instrumenté est écrit ; instruments jetables gatés
  par env var (patron `I22_FILM`), jamais actifs en CI.

## Découvertes

(consigner ici tout ce qui dépasse le périmètre — rien corriger)

## CR attendu (à rendre au superviseur)

Rapport `.ai/V7.5/replay2d/RAPPORT_QUALITE_SCORE_EQUIPE.md` : tableaux phase 0 et 1,
verdict 1.4, options d'implémentation chiffrées (avec contraintes anti-ART : INSERT-only,
vues `_latest`, migration déployée élargie = step au nom neuf), gates rejoués (commandes +
sorties). Statut de CHAQUE item ci-dessus. Commits atomiques dans le worktree, sujets
`score-sync(pN): ...`, JAMAIS `git add -A`.
