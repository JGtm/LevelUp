# Skill : adversarial-audit — Audit adversarial du code existant

Invoquer ce skill pour faire relire du code DÉJÀ EN PLACE — pas un diff — par des contextes
frais et hostiles, et en tirer un registre de constats exploitable.

C'est le pendant hors-diff de `adversarial-review`. La différence n'est pas cosmétique :

| | `adversarial-review` | `adversarial-audit` (ce skill) |
|---|---|---|
| Objet | un diff / une branche | un sous-système existant |
| Contrat de référence | ce que le lot devait faire | la doctrine du projet (CLAUDE.md, ADRs) |
| Sortie | corriger avant merge | un registre daté qui devient un plan |
| Boucle | 2 rondes puis merge | aucune boucle — l'audit ne corrige rien |

---

## 0. La règle qui rend l'audit utile : il ne corrige pas

Un audit qui corrige au fil de l'eau produit un diff énorme, non relu, sur du code qui
marchait. **L'audit produit un registre. Point.** Les corrections sont un chantier
ultérieur, planifié, exécuté sous `plan-execution`, relu sous `adversarial-review`.

Corollaire : pendant un audit, zéro `Edit`, zéro `Write` en dehors du fichier de registre.

## 1. Cadrage — jamais « toute l'app »

« Audite l'application » produit une bouillie générique où tout est signalé et rien n'est
actionnable. Un audit se cadre sur **un périmètre × un axe**.

**Périmètre** — un ensemble de fichiers nommable et fini. Exemples valides :
`internal/persist/` + `internal/sync/` · `internal/api/handlers/` ·
`apps/web/src/features/explorer/` · les chemins listés dans un ADR · les fichiers touchés
par un chantier des 3 derniers mois. Exemple invalide : `apps/go-api/`.

**Axe** — une question fermée, une seule (catalogue au §3).

Si le besoin est large, ce n'est pas un audit mais **une campagne** : plusieurs audits
`périmètre × axe`, lancés en parallèle, dont les registres fusionnent à la fin. C'est le
mode normal pour « revoir l'app ». Cadrer explicitement la campagne avant de lancer quoi
que ce soit, et l'écrire.

## 2. Le contrat de référence — c'est la doctrine

En l'absence de diff, ce contre quoi on audite doit être écrit, sinon l'auditeur invente
ses propres standards et remonte des préférences. Le contrat d'un audit LevelUp est
toujours un extrait de :

- `CLAUDE.md` — règles 1 à 16, section « Règles critiques écritures DuckDB », section
  « Diagnostic de revue de code : anti-patterns interdits » ;
- l'ADR concerné (`docs/adr/`) — en particulier 0006, 0008, 0013, 0019, 0023, 0025, 0026, 0030 ;
- le SKILL.md du domaine (`arch-rules`, `db-schema`, `canonical-types`,
  `frontend-patterns`, `color-tokens`).

L'auditeur reçoit l'extrait pertinent **collé dans son prompt**, pas une référence à aller
lire. Une règle non collée est une règle non appliquée.

Coller aussi la contrepartie, qui évite la moitié des faux positifs :

```
DETTE CONNUE ET ASSUMÉE — NE PAS REMONTER
- Baseline lint golangci-lint : la dette existante est gelée, pas un constat.
- Fallbacks legacy auth (sync_meta.*, SPNKR_OAUTH_REFRESH_TOKEN_*) : en retrait planifié
  (ADR 0023 Phase 5), connus.
- Les exemptions de seuil portant un commentaire de justification sont valides.
- Player DBs legacy sans contraintes : le pattern SELECT-then-UPDATE-or-INSERT est la
  solution retenue, pas un défaut.
```

## 3. Catalogue d'axes

Un axe par auditeur. Ne jamais en confier deux au même contexte : il traite bien le
premier et survole le second.

**A1 — Code mort.** Exports jamais référencés, fonctions sans appelant, branches
toujours fausses, flags dont les deux côtés ne sont plus atteignables, tests qui
maintiennent en vie du code que plus rien n'appelle. Le cas le plus coûteux : le module
débranché du routing dont les tests restent verts. Preuve exigée : la commande de recherche
qui montre zéro appelant hors tests.

**A2 — Garde-rails et invariants ART.** Écritures per-match hors `persist.BatchBuilder` ·
UPSERT concurrent sur une table critique · lecture brute d'une table append-only au lieu de
`<table>_latest` · ouverture DuckDB hors provider/lease · `OpenReadOnly` forcé sur une DB
tenue RW · écriture `shared_social` sans `CHECKPOINT` · entrée d'allowlist sans
justification datée. Axe prioritaire : c'est celui dont les défauts corrompent des données
en production.

**A3 — Multi-titre.** Comparaisons `slug == "..."` résiduelles · `filepath.Join` sur
`data/` · libellés FR/EN en dur côté Go · capability absente qui panique ou sert les
données d'un autre titre · champ présent dans le code mais absent des TOML d'un titre.

**A4 — Frontières de couches.** Logique métier dans un handler HTTP ou un composant React ·
SQL inline dans un service · accès DuckDB direct hors adapter/repo · type title-specific
qui traverse la frontière canonique · import cross-feature côté web.

**A5 — Duplication et factorisations abandonnées.** Même littéral ou même prédicat en 3+
endroits · helper canonique existant que des copies n'utilisent pas · garde-rail
(test grep) absent sur une factorisation faite. Preuve exigée : la liste des occurrences.

**A6 — Erreurs avalées.** `_ = f()` · `continue` sur erreur sans log ni compteur · erreur
loggée après la dégradation au lieu d'avant · `err` remonté sans contexte sur plusieurs
niveaux · `slog` absent là où une dégradation best-effort se produit.

**A7 — Flags et compatibilité.** Kill-switch sans date de basculement, sans date cible de
retrait, ou sans critère mesurable · feature laissée OFF « pour plus tard » · commentaire
qui décrit l'ancien défaut d'un flag déjà basculé (doc inversée) · branche legacy dont la
date d'expiration est passée.

**A8 — Ce que les tests ne couvrent pas.** Pour chaque comportement critique du périmètre :
quel test échoue si on l'inverse ? Tests qui passent avec et sans le code qu'ils prétendent
couvrir · `t.Skip` sans justification · assertions sur des champs non calculés par le code
testé · gates dont l'équivalent CI n'existe pas.

**A9 — Correction des données.** Formule KDA/KDR conforme à l'ADR 0006 · timezone
canonique dans tout filtre et tout tri temporel · lecteurs de rating sur `_latest` ·
agrégats qui recalculent au lieu de réutiliser les KPI dérivés · unités (0..1 vs 0..100)
cohérentes de la DB à l'UI.

**A10 — Front.** Hex ou classe Tailwind couleur dans `features/`/`components/` · string UI
présente dans une seule langue · query key inline · anglicisme dans l'UI FR · logique dans
un composant au lieu d'un hook ou d'un `*_logic.ts`.

## 4. Exécution — fan-out aveugle

1. Un **sous-agent par axe**, lancés en parallèle, chacun ignorant les autres et ignorant
   qui a écrit le code. Ne pas auditer depuis le contexte principal : il connaît
   l'historique du projet et pardonne.
2. Chaque auditeur reçoit : le périmètre (liste de fichiers ou glob), son axe unique,
   l'extrait de doctrine correspondant, le bloc « dette assumée » (§2), et les règles de
   recevabilité (§5).
3. Les auditeurs **lisent le code, ils ne le modifient pas**. Leur sortie est une liste
   structurée, pas un rapport en prose.

## 5. Recevabilité — le filtre anti-théâtre

À coller littéralement dans chaque prompt d'auditeur :

> Un constat n'est recevable que s'il porte les TROIS éléments suivants :
> 1. `fichier:ligne` précis ;
> 2. la règle enfreinte, citée depuis la doctrine qui t'a été fournie — pas ton opinion
>    sur ce qui serait mieux ;
> 3. la conséquence concrète : ce qui casse, se corrompt, s'affiche faux, ou ce qui sera
>    plus coûteux et pourquoi.
>
> Sont irrecevables : « pourrait poser problème », « manque de robustesse », « il serait
> préférable », toute suggestion de réécriture sans défaut identifié, toute remarque de
> style non couverte par une règle écrite, et tout élément figurant dans la dette assumée.
>
> Tu as le droit de conclure « aucun constat recevable sur cet axe ». C'est un résultat
> valide et utile. Ne complète pas ta liste pour la remplir : un auditeur qui trouve
> toujours quelque chose ne sert à rien.
>
> Pour chaque constat, fournis la commande (grep, test, requête) qui permet à un tiers de
> le reproduire en une seule invocation. Un constat non reproductible est jeté.

## 6. Vérification adverse des constats

Les constats de la passe 1 ne sont pas des faits. Avant d'écrire le registre, chaque
constat P0/P1 passe devant un **second sous-agent frais dont la consigne est de le
réfuter** :

> Voici un constat produit par un auditeur. Ta tâche est d'établir qu'il est FAUX. Ouvre
> le fichier et la ligne. Cherche ce que l'auditeur n'a pas vu : un appelant, un test, une
> contrainte de schéma, une garantie en amont, une entrée d'allowlist, un commentaire de
> justification. Conclus « réfuté » ou « tient », avec la preuve. En cas de doute, conclus
> « réfuté ».

Le « en cas de doute, réfuté » est délibéré : il inverse le biais de l'auditeur. Un constat
qui survit à ça vaut la peine d'être traité.

## 7. Le registre

Sortie unique : `.ai/AUDIT_<PERIMETRE>_<AAAA-MM-JJ>.md`.

```markdown
# Audit <périmètre> — <date>

## Cadrage
Périmètre : <fichiers / globs exacts>
Axes passés : <A1, A4, A6...>  |  Axes NON passés : <lesquels, et pourquoi>
Doctrine de référence : <extraits utilisés>

## Constats retenus

### [P0] <titre court>
- Où : `fichier:ligne`
- Règle enfreinte : <citation de la doctrine>
- Conséquence : <ce qui casse>
- Reproduction : `<commande>`
- Vérification adverse : tient / réfuté (résumé)
- Traitement proposé : <une phrase>  |  Décision : à trancher / planifié / refusé

### [P1] ...
### [P2] ...

## Constats écartés
| Constat | Axe | Motif d'écart |
|---|---|---|
| ... | A5 | réfuté en vérification adverse : le helper est bien utilisé, l'auditeur a lu une copie de test |
| ... | A7 | dette assumée (baseline lint) |

## Axes sans constat
<liste — c'est une information, l'écrire>

## Suite
<ce qui devient un plan, ce qui est escaladé à l'utilisateur, ce qui est classé>
```

La section « Constats écartés » n'est pas facultative. C'est elle qui rend le registre
crédible la fois d'après : elle prouve que l'auditeur pouvait dire non.

## 8. Gravité et qui décide

- **P0** — corruption de données, perte d'écriture, faille d'accès, résultat faux servi à
  l'UI. Devient un chantier immédiat.
- **P1** — violation d'une règle écrite du projet, sans conséquence de données. Devient un
  lot planifié.
- **P2** — dette réelle, coût futur, pas de conséquence actuelle. Classé au backlog.
- **Escalade utilisateur, sans proposition d'action** : tout ce qui touche une décision
  d'architecture, un contrat d'API, un schéma, un ADR, ou la doctrine elle-même.

Le nombre de constats n'est pas une métrique de qualité de l'audit. Un audit qui remonte
3 P0 reproductibles vaut mieux qu'un audit qui remonte 40 remarques.

## 9. Après l'audit

- Le registre est commité (il n'y a rien d'autre à commiter : l'audit ne modifie pas de
  code).
- Entrée `.ai/thought_log.md` : périmètre, axes passés, axes non passés, nombre de constats
  retenus par gravité, nombre écartés.
- Si un chantier de correction en découle : il se cadre sous `plan-review`, s'exécute sous
  `plan-execution`, se relit sous `adversarial-review`. L'audit ne se prolonge pas en
  correction dans la même session.
