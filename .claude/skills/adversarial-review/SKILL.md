# Skill : adversarial-review — Revue adversariale d'un diff avant merge

Invoquer ce skill quand un lot de code est écrit et qu'il faut le faire relire par un
contexte qui ne l'a pas écrit, AVANT le merge.

Complémentaire, pas redondant :
- `plan-review` relit un PLAN (avant de coder).
- `adversarial-review` (ce skill) relit un DIFF (après avoir codé, avant de merger).
- `delivery-checklist` vérifie une LIVRAISON contre une liste fermée (gates, thought_log).
- `adversarial-audit` audite du code EXISTANT hors de tout diff.

---

## 0. Le problème que ce skill résout

Le contexte qui a écrit le code défend ses choix. Il relit son propre diff en confirmant
ses intentions au lieu de lire ce qui est réellement écrit. Aucune formulation
(« relis attentivement », « sois critique ») ne corrige ça, parce que le biais n'est pas
dans le ton : il est dans le fait que l'auteur SAIT ce qu'il voulait faire et lit cette
intention par-dessus le code.

Le seul correctif fiable : donner la relecture à un contexte **qui n'a pas écrit le code
et ne sait pas qui l'a écrit**.

## 1. Calibrage — n'armer que ce qui le mérite

Une revue adversariale sur un diff trivial produit du bruit : faute de vraie prise, le
relecteur se rabat sur le nommage et le style. Décider AVANT de lancer.

| Nature du diff | Revue |
|---|---|
| Écritures DuckDB, `persist/`, `sync/`, `migration/`, dblease, append-only | OBLIGATOIRE, 2 relecteurs |
| Auth / tokens / ownership joueur / contrôle d'accès | OBLIGATOIRE, 2 relecteurs |
| Nouvelle capability, nouvel adapter, frontière multi-titre | OBLIGATOIRE, 1 relecteur |
| Algo d'analyse (KPI, rating, agrégat) avec formule | OBLIGATOIRE, 1 relecteur |
| Feature front (page, chart, hook) | 1 relecteur |
| Renommage, doc, i18n, déplacement de fichier, bump de version | AUCUNE — `delivery-checklist` suffit |

Si le diff mélange les deux (gros refactor + une page), scoper la revue au sous-ensemble
risqué et le dire.

## 2. Règle non négociable — contexte frais

Le relecteur tourne dans un **sous-agent** (`Agent`, type `general-purpose` ou `Explore`),
jamais dans le contexte qui a codé. Le prompt du sous-agent NE DOIT PAS contenir :

- qui a écrit le code, ni que c'est « notre » code ;
- le raisonnement qui a mené à l'implémentation ;
- une justification anticipée d'un choix (« on a fait X parce que Y »).

Le sous-agent reçoit : le contrat (§3), le diff (ou de quoi le reconstituer), les lentilles
(§5), et les règles de recevabilité (§4). Rien d'autre.

Si le lot est gros, lancer plusieurs relecteurs **en parallèle, aveugles les uns aux
autres**, un par lentille (§5). Deux relecteurs qui trouvent le même défaut
indépendamment, c'est un signal fort ; un relecteur seul qui trouve un défaut unique
demande vérification.

## 3. Le contrat — l'entrée la plus importante

Sans contrat, le relecteur ne peut pas distinguer « pas implémenté » de « hors périmètre »,
et il remonte les deux. Résultat : une liste de faux positifs qui te fait perdre confiance
dans l'outil.

Le contrat tient en 6 lignes et se rédige AVANT de lancer le relecteur :

```
CONTRAT DE CE LOT
Objectif        : <ce que le lot devait accomplir, en une phrase>
Périmètre INCLUS : <fichiers / comportements que ce lot devait changer>
Périmètre EXCLU  : <ce qui a été volontairement laissé de côté, et pourquoi>
Invariants à ne pas casser : <ce qui devait continuer de marcher>
Décisions déjà tranchées : <choix arbitrés en amont, non rediscutables ici>
Dette pré-existante assumée : <ce qui était déjà cassé avant ce diff>
```

La dernière ligne évite la dérive la plus coûteuse : le relecteur qui remonte comme
défaut du lot une dette qui existait avant lui.

## 4. Recevabilité d'un constat — le filtre anti-théâtre

Le relecteur reçoit ces règles **littéralement**, dans son prompt :

> Un constat n'est recevable que s'il porte les TROIS éléments suivants :
> 1. `fichier:ligne` — l'endroit exact, pas « quelque part dans le service » ;
> 2. la condition de déclenchement — des entrées, un état, ou une séquence concrète qui
>    produit le problème ;
> 3. la conséquence observable — ce qui casse, se corrompt, s'affiche faux, ou plante.
>
> Un constat sans les trois est jeté sans discussion. Sont explicitement irrecevables :
> « ceci pourrait poser problème », « ce n'est pas très robuste », « il vaudrait mieux »,
> « et si un jour », toute préférence de style non couverte par une règle écrite du projet.
>
> Tu as le droit — et il est attendu de toi — de conclure « aucun défaut recevable trouvé ».
> Un relecteur qui ne peut jamais rien valider n'a aucune valeur : ses alarmes ne veulent
> plus rien dire. Ne fabrique pas de constat pour remplir la liste.

## 5. La formulation qui marche mieux que « trouve les défauts »

L'injonction hostile pure (« assume que c'est cassé ») produit du volume. L'inversion
produit de la précision. Le prompt du relecteur pose la question dans ce sens :

> Pour chaque changement du diff : **énonce ce qui devrait être vrai pour que ce code soit
> correct** — les préconditions, les invariants, ce que l'appelant garantit, ce que le
> schéma garantit. Puis va VÉRIFIER chacune de ces conditions dans le code réel (ouvre les
> appelants, ouvre le schéma, ouvre les tests). Remonte celles qui ne tiennent pas.

C'est la même énergie critique, mais orientée sur une liste finie et vérifiable au lieu
d'une chasse ouverte. Et elle peut se terminer sur « les 7 conditions tiennent » — ce que
la formulation hostile ne permet jamais.

## 6. Lentilles LevelUp — donner au relecteur les invariants du projet

Le relecteur générique rate les pièges maison. Lui passer les lentilles pertinentes au
diff (une par relecteur si fan-out). Chaque lentille est autoportante.

**L1 — Écritures DuckDB / anti-ART.** Toute écriture per-match sur une DB partagée passe
par `persist.BatchBuilder.Submit()` (INSERT-only). Aucun UPSERT / `ON CONFLICT DO UPDATE`
concurrent sur les tables critiques. Toute lecture d'une table append-only passe par la
vue `<table>_latest` — une lecture brute sert des lignes périmées. Un seul process writer
par DB : pas de RO et RW sur le même fichier. Lire une DB potentiellement tenue RW :
`OpenReadForQuery`, jamais `OpenReadOnly`. `shared_social` : `SharedSocialPersister` +
`CHECKPOINT`. Un garde-rail élargi (allowlist, regex assouplie) sans justification datée
dans le diff est un constat P0.

**L2 — Multi-titre.** Aucun `slug == "halo_infinite"` : brancher sur `HasCapability` /
`CapabilityMap`. Aucun `filepath.Join(..., "data", ...)` : tout passe par `PathResolver`.
Aucun libellé FR/EN en dur côté Go : `TitleSemanticAdapter` + TOML. La dégradation d'une
capability absente doit produire `ErrCapabilityNotSupported` géré, jamais un panic, jamais
les données d'un autre titre.

**L3 — Les 10 anti-patterns de CLAUDE.md.** Code mort conservé « au cas où » (surtout avec
des tests verts qui entretiennent l'illusion) · flag legacy sans date d'expiration · fichier
> 500 L / fonction > 80 L · même littéral en 3+ endroits · ouverture DuckDB hors
provider/lease · nombre magique · logique métier dans un handler HTTP ou un composant React ·
helper canonique créé sans migrer les copies ni poser le garde-rail · commentaire qui décrit
l'ancien défaut d'un flag basculé · `_ = f()` ou `continue` sur erreur sans log ni compteur.

**L4 — Correction des données.** KDA n'est jamais le quotient (per-match = valeur API
native ; agrégat = `((frags + assists/3) - morts) / nb_matchs`). Tout filtre ou tri temporel
passe par `COALESCE(x.start_time_utc, x.start_time AT TIME ZONE 'UTC')`, jamais
`start_time` brut. Les lecteurs de rating lisent `match_skill_rank_latest`. Vérifier que
les agrégats réutilisent les KPI dérivés existants au lieu de recalculer ad hoc.

**L5 — Front.** Aucune valeur hex ni classe Tailwind couleur dans `features/` ou
`components/` : tokens sémantiques. Toute string UI présente en FR **et** EN. Query keys
dans `lib/query/keys.ts`, jamais inline. `routeTree.gen.ts` jamais édité à la main.
FR sans anglicismes.

**L6 — Ce que les tests ne couvrent pas.** Lentille distincte, à confier à son propre
relecteur : pour chaque chemin de code ajouté, quel test échouerait si on inversait la
condition ? Un test qui passe avec ET sans le correctif ne teste rien. Un `t.Skip` sans
justification est un constat. Un gate rejoué localement ne couvre pas les jobs CI.

## 7. Triage — c'est toi qui décides, pas le relecteur

Le relecteur produit des constats. Il ne décide pas de la suite. Classer :

- **P0 — bloquant merge.** Corruption de données, perte d'écriture, faille d'accès,
  régression d'un invariant ART, résultat faux servi à l'UI. Corriger avant merge.
- **P1 — bloquant merge sauf décision explicite.** Violation d'une règle écrite du projet
  (CLAUDE.md, ADR, seuils). Corriger, ou consigner une exemption justifiée en commentaire.
- **P2 — dette consignée.** Réel mais hors périmètre du lot. Va dans la section
  « Découvertes » du plan actif ou dans `.ai/thought_log.md`. **Ne pas corriger dans ce
  diff** (règle 5 de `plan-execution` : zéro fix opportuniste).
- **P3 — jeté.** Ne passe pas le filtre §4, ou relève d'une préférence.

**Escalade obligatoire vers l'utilisateur, sans tenter de corriger :** tout constat qui
implique une décision d'architecture, un changement de contrat d'API, une migration de
schéma, ou une modification de la doctrine (CLAUDE.md / ADR). Le relecteur et l'auteur
n'ont pas autorité là-dessus.

## 8. Bornes de boucle — comment on s'arrête

Sans borne, la boucle correction/relecture ne converge pas : chaque correction ouvre une
nouvelle surface de critique et le relecteur, à qui on demande de trouver, trouve.

1. **2 rondes maximum.** Ronde 1 : relecture, tri, correction des P0/P1. Ronde 2 :
   relecture des seules corrections apportées, par un contexte frais.
2. **Le nombre de P0+P1 doit décroître strictement d'une ronde à l'autre.** S'il stagne ou
   augmente, on arrête et on remonte à l'utilisateur : soit le lot est mal conçu, soit le
   relecteur part en vrille.
3. **Après la ronde 2**, s'il reste des P0/P1 : arrêt, escalade à l'utilisateur avec la
   liste. Pas de ronde 3.
4. La ronde 2 relit **les corrections**, pas tout le diff. Relancer une relecture complète
   à chaque ronde garantit de nouveaux constats à l'infini.

## 9. Modèle de prompt de relecteur (à copier)

```
Tu relis un diff que tu n'as pas écrit. Tu ne sais pas qui l'a écrit et ce n'est pas
pertinent. Ton travail n'est pas d'approuver ni de rassurer : c'est d'établir si ce code
fait ce qu'il prétend faire.

CONTRAT DU LOT
<coller les 6 lignes du §3>

CE QUE TU DOIS FAIRE
1. Lis le diff : `git diff <base>...<branche>` (ou les fichiers listés ci-dessous).
2. Pour chaque changement, énonce ce qui devrait être vrai pour que ce code soit correct :
   préconditions, invariants, garanties de l'appelant, garanties du schéma.
3. Va vérifier chacune de ces conditions dans le code RÉEL — ouvre les appelants, le
   schéma, les tests. Ne conclus rien depuis le seul diff.
4. Remonte les conditions qui ne tiennent pas.

LENTILLE ASSIGNÉE
<coller UNE lentille du §6>

RECEVABILITÉ D'UN CONSTAT
<coller le bloc du §4 intégralement>

FORMAT DE SORTIE
Pour chaque constat : fichier:ligne | condition de déclenchement | conséquence observable |
gravité proposée (P0/P1/P2). Puis une ligne « Conditions vérifiées qui tiennent : N ».
Si rien n'est recevable, écris-le et arrête-toi là.
```

## 10. Après la revue

- Les P0/P1 corrigés le sont dans le même lot, avec leur test de non-régression.
- Les P2 sont consignés (Découvertes / thought_log) avec leur `fichier:ligne`, pas corrigés.
- L'entrée `.ai/thought_log.md` du lot mentionne : nombre de constats recevables, nombre
  jetés, ce qui reste ouvert. Un lot qui a passé une revue adversariale sans constat
  recevable, c'est une information — l'écrire.
- Enchaîner sur `delivery-checklist` : cette revue ne remplace aucun gate.

---

## Variante — relecteur d'un autre modèle

Si un CLI d'un autre fournisseur est installé et authentifié (`codex`, `gemini`,
`copilot`), lui confier la ronde 1 est une amélioration réelle : le biais de défense de
l'auteur disparaît complètement, et pas seulement au niveau du contexte.

Les mêmes règles s'appliquent sans exception — contrat (§3), recevabilité (§4), bornes
(§8) — et le triage (§7) reste de ce côté-ci. Un relecteur externe non calibré sur un
petit lot produit exactement le même bruit, en plus confiant.

Aucun de ces CLI n'est installé sur ce poste à ce jour : la variante est documentée, pas
armée. Ne pas l'invoquer sans avoir vérifié la présence du binaire.
