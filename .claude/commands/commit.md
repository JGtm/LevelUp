Crée un commit git pour les modifications en cours du projet LevelUp.

Étapes :
1. Lance `git status` et `git diff --staged` (+ `git diff` si rien n'est staged) pour voir ce qui change
2. Lance `git log --oneline -5` pour t'aligner sur le style de commits existants
3. Stage les fichiers pertinents (jamais `.env`, secrets, ni binaires lourds non intentionnels)
4. Rédige un message de commit au format **Conventional Commits** :
   - `feat(scope):` nouvelle fonctionnalité
   - `fix(scope):` correction de bug
   - `refactor(scope):` refactoring sans changement de comportement
   - `test(scope):` ajout/modification de tests
   - `docs(scope):` documentation uniquement
   - `chore(scope):` maintenance, deps, config
   - Corps optionnel si le titre ne suffit pas (max 72 car. pour le titre)
5. Exécute le commit avec ce message + la co-authorship Claude
6. Confirme avec `git log --oneline -1`

Ne push pas sauf demande explicite.
