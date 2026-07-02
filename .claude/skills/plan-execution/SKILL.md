# Skill : plan-execution — Contrat d'exécution d'un plan multi-étapes

Invoquer ce skill AVANT de commencer (ou reprendre) l'exécution de tout plan multi-étapes
(`.ai/**/PLAN_*.md` ou plan donné dans la conversation).

Ce contrat existe parce que les dérives d'exécution constatées sont toujours les mêmes :
traiter les étapes dans le désordre, en traiter une partiellement puis passer à la
suivante, DIFFÉRER une étape faisable (« je ferai X ensuite », TODO, « à traiter plus
tard ») au lieu de la faire, et déclarer terminé sans vérifier. Chaque règle ci-dessous
bloque une de ces dérives. Elles sont NON NÉGOCIABLES.

---

## 1. Les 10 règles du contrat

1. **Ordre strict.** Les étapes s'exécutent dans l'ordre du plan. Interdiction de
   commencer l'étape N+1 tant que l'étape N n'est pas CLOSE (règle 6). Seule exception :
   une étape que le plan lui-même déclare différée (soak, dépendance datée).
2. **Une étape commencée est une étape terminée.** On ne picore pas les items faciles de
   plusieurs étapes. Si une étape s'avère plus grosse que prévu, on la finit quand même —
   ou on s'arrête PROPREMENT (règle 9), on n'enjambe pas.
3. **Interdiction de différer une action exécutable maintenant.** Si l'information et
   l'accès existent pour faire l'action dans la session courante, elle se fait dans la
   session courante. Sont des reports INVALIDES : « pour garder le momentum », « ce sera
   plus simple plus tard », « je note un TODO », « l'utilisateur pourra le faire ».
   Sont des reports VALIDES (et documentés au journal) : dépendance explicite du plan,
   décision/donnée que seul l'utilisateur possède, délai d'observation prescrit,
   ressource externe indisponible.
4. **Vérifier sur pièces, deux fois.** Avant de coder : rouvrir le fichier/la ligne cités
   par le plan ou l'audit source (le code a pu bouger — ne jamais corriger de mémoire).
   Avant de cocher : re-vérifier que le résultat est bien en place (le test passe, le grep
   retourne 0, la page s'affiche).
5. **Statuer chaque item, aucune case vide.** Statuts autorisés :
   - `[x]` fait et vérifié ;
   - `[~]` couvert par un autre item — indiquer lequel ;
   - `[!]` non traité — justification écrite au journal, à revoir avec l'utilisateur.
   Un `[!]` silencieux ou une case vide = étape non close.
6. **Clôture d'une étape = 5 actions, dans l'ordre :**
   a. Gate passé (commandes exactes de l'étape, sorties propres — jamais de test
      désactivé/skippé pour passer) ;
   b. Tous les items statués ;
   c. Le fichier plan mis à jour (cases + journal) et inclus dans le commit ;
   d. Entrée `.ai/thought_log.md` ;
   e. Point d'étape à l'utilisateur : fait / skippé+pourquoi / découvert.
7. **Zéro fix opportuniste hors périmètre.** Toute découverte (bug, dette, idée) se note
   dans la section « Découvertes » du plan et N'EST PAS traitée — sauf si elle bloque le
   gate de l'étape courante.
8. **Commits disciplinés.** Au moins 1 commit par étape, message préfixé par la référence
   de l'étape. Jamais `git stash`. Demander avant tout push sur `main` (deploy auto).
9. **Blocage = arrêt propre, pas contournement silencieux.** Si un gate est réellement
   impossible (dépendance externe, décision utilisateur) : statuer `[!]` + justification,
   informer l'utilisateur, et seulement alors passer à la suite SI le plan le permet.
   Ne jamais « adapter » discrètement le périmètre pour que ça passe.
10. **Reprise de session.** Le fichier plan est la source de vérité de l'avancement. À
    chaque reprise : relire ce contrat, puis le journal du plan, reprendre à la première
    case non statuée de l'étape courante. Ne pas re-décider ce qui est déjà tranché
    (décisions validées = fermes).

## 2. Auto-contrôle avant de dire « terminé »

Avant de déclarer une étape (ou le plan) terminé, répondre honnêtement :

- [ ] Ai-je exécuté TOUS les items, ou en ai-je reformulé certains en « à faire ensuite » ?
      (Chercher dans ma propre sortie : « plus tard », « ensuite », « prochaine étape »,
      « TODO », « pourra être » — chaque occurrence est un report à requalifier règle 3.)
- [ ] Chaque commande de gate a-t-elle RÉELLEMENT tourné dans cette session, avec sortie
      verte ? (Pas « devrait passer ».)
- [ ] Ai-je laissé un test skippé, un lint désactivé, une allowlist agrandie sans
      justification datée ?
- [ ] Le fichier plan reflète-t-il l'état réel (cases, journal) ?
- [ ] L'entrée thought_log existe-t-elle ?
- [ ] Ce que j'annonce à l'utilisateur correspond-il à ce que j'ai fait (ni plus, ni moins) ?

## 3. Communication pendant l'exécution

- Point d'étape à chaque clôture d'étape (règle 6e) — pas de marathon silencieux.
- Signaler IMMÉDIATEMENT (sans attendre la fin d'étape) : une décision produit non prévue
  par le plan, un périmètre qui explose (> 2x l'estimation), une découverte qui invalide
  une hypothèse du plan.
- Ne pas déléguer à l'utilisateur des micro-décisions que le plan ou les conventions du
  repo tranchent déjà — décider, consigner, avancer.

## 4. Ce que ce contrat ne dit pas

- Il ne remplace pas `plan-review` (qualité du plan avant exécution) ni
  `delivery-checklist` (go/no-go technique avant commit) — les trois se composent :
  plan-review AVANT, plan-execution PENDANT, delivery-checklist à CHAQUE clôture.
- Si le plan lui-même contient un contrat d'exécution (section dédiée), le plan fait foi
  en cas de divergence — ce skill est le défaut.
