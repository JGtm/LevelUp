# Plan — Aligner les effets d'etat actif sur le cahier des charges Notion

> Ecrit le 2026-08-16. Le lot `d257ba02f` a livre des effets choisis EN SESSION (estompe /
> anneau token `info`) sans relire la spec : le cahier des charges Notion (Backlog LevelUp,
> section REPLAY 2D, item 21.1) prescrivait AUTRE CHOSE. Ecart signale par l'utilisateur.
> Branche `feat/v75`, contrat `plan-execution`.

## La spec, verbatim (Notion, item 21.1)

> etat ACTIF : surbouclier -> **encadre dore** autour de la carte du joueur ; camouflage
> actif -> **effet de verre** ; translocateur quantique -> **bordure animee qui va d'un bleu
> electrique a un jaune orange feu, qui ferait le tour de la bordure**. Les autres restent
> actives trop peu de temps pour justifier un traitement dedie d'emblee.

## Correction semantique a porter partout (enseignement utilisateur du 16/08)

Les 36 episodes camo du film golden (Super Fiesta, aucun porteur d'equipement camo) ne sont
PAS des « power-ups ramasses » : **en Fiesta, un DASH (propulseur) active le camouflage 1 a
2 s — effet propre au mode, rien n'est ramasse**. La doc du lot precedent (golden, registre,
commentaires) dit « power-up » : c'est faux et ca se corrige ici.

## Phases

### Phase 0 — VERIFIER la lecture dash-camo (mesure courte, instruments existants)

- [ ] 0.1 Duree des episodes camo du golden `000d5950` : la prediction falsifiable est
      **1 a 2 s** pour l'essentiel des 36 episodes (les episodes d'equipement du film
      `06dfe6d9` montaient a 16,2 s). Publier la distribution.
- [ ] 0.2 Co-datation episodes camo x episodes `i54` (l'action de mobilite, hook existant)
      sur le golden : si le dash declenche le camo, chaque episode camo suit un episode
      i54 de la MEME vie a moins d'une seconde. Publier le taux et un temoin decale.
      (Bonus si ca tient : en Fiesta, les episodes camo DATENT les dashs — a consigner au
      registre comme piste « effet de deplacement du propulseur », pas a implementer.)

### Phase 1 — LES EFFETS DE LA SPEC

- [ ] 1.1 **Camouflage -> effet de VERRE** sur toute la fiche (remplace l'estompe 0.4) :
      translucidite + flou leger du contenu, lisibilite conservee. Tokens semantiques
      uniquement ; si un token de surface vitree manque, l'AJOUTER au systeme (skill
      `color-tokens`), jamais de hex dans `features/`.
- [ ] 1.2 **Surbouclier -> ENCADRE DORE** autour de la fiche (remplace l'anneau token
      `info`). « Dore » = token : chercher le token or/legendaire existant du systeme
      (rangs, raretes) ; a defaut, en creer un semantique. Pas d'animation obligatoire.
- [ ] 1.3 `prefers-reduced-motion` : les deux effets restent statiques par construction.
- [ ] 1.4 Tests composants mis a jour ; toute string UI en FR ET EN.

### Phase 2 — TRANSLOCATEUR : consigner, ne pas brancher

- [ ] 2.1 La bordure animee bleu electrique -> jaune feu N'EST PAS branchable : aucun canal
      d'etat mesure pour le rang 11 (3 lectures i48 dans tout le corpus, aucun etat).
      Ligne au registre avec sa condition de reprise (un canal d'etat pour le
      translocateur ; candidats non explores : i57 valeurs par rang, entites ti=37 typees
      le jour ou l'identite tombe).
- [ ] 2.2 Piste a consigner au meme endroit : la spec (item 21.2) predisait « i57 porte la
      DIRECTION de la capacite (cubemap 24 bits) » — le R(24) de la branche v==1, aux
      valeurs quasi uniques, COLLE a une direction quantifiee. Non verifie ; c'est la
      reprise naturelle du sujet R(24).

### Phase 3 — CORRIGER LA DOC

- [ ] 3.1 Remplacer « power-up » par la lecture dash-Fiesta partout ou le lot precedent
      l'a ecrit (commentaire du golden, registre, thought_log — entree nouvelle, pas de
      reecriture d'historique), avec les chiffres de la phase 0.

## Gates

- Phase 0 : chiffres publies avec denominateurs.
- Phase 1 : purge `node_modules\.tmp` puis typecheck, lint, vitest — exit 0 ; zero hex ;
  gate VISUEL utilisateur (films temoins : `084a804d` pour l'equipement, `000d5950` pour le
  camo de dash).
- Entree `thought_log.md`, registre, commits sur `feat/v75`, pas de push.

## Regles dures

Aucun effet sans donnee mesuree (translocateur ne se branche pas) ; tokens uniquement ;
zero fix hors perimetre ; JAMAIS de pause d'attente passive.
