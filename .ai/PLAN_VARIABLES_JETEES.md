# PLAN — les variables décodées puis jetées

> Écrit le 2026-07-31. Contrat d'exécution : skill `plan-execution`.
> Arbitrage validé par l'utilisateur le 2026-07-31.

---

## LE CONSTAT

Le décodeur de film est un **sauteur de bits par construction**. Le dispatch porte environ
**200 grammaires de composants**, et **quatre seulement rendent une valeur** — les deux
vitalités, le compteur de réapparition et l'horloge de manche. Tout le reste est lu bit-exact
pour avancer le curseur, puis abandonné.

Ce n'est pas un défaut : c'était le bon choix pour arriver aux positions. Mais une trentaine de
grandeurs sont désormais **à portée de main**, et il faut trier — parce que tout brancher
coûterait cher pour rien.

---

## L'ARBITRAGE

### À BRANCHER

| variable | ce qu'elle apporte | coût |
|---|---|---|
| **`i57`, état actif des capacités** | la seule qui serve une fonctionnalité **déjà dessinée** (surbouclier doré, camouflage vitreux) | plan dédié : `PLAN_CAPACITES_ACTIVES.md` |
| **index de capacité sur 6 bits** | explique probablement la table partielle (4 noms sur 11) | idem |
| **compteur de réapparition** | **remplace une déduction par une lecture** | ce plan, étape 1 |
| **horloge de manche** | l'état de la partie, absent du rejeu | ce plan, étape 1 |
| **pitch de visée, orientation du corps, vélocité** | viser le sol ou une passerelle ; reculer en tirant | ce plan, étape 2 |

### À NE PAS BRANCHER — et pourquoi

| variable | raison |
|---|---|
| **le code de médaille** du chunk highlight | **L'API le lit très bien**, et les visuels sont un mapping à reprendre des pages existantes. Le film le porte et le perd à l'écriture : c'est une **redondance, pas un manque**. *(Un audit avait surestimé ce point ; corrigé par l'utilisateur.)* |
| **le dead-state `i11`** — victime, tueur, catégorie mêlée/lancé, source du dégât | **C'est exactement ce que décode le chantier voisin** : sa « source du dégât » est lue dans l'état de mort de la victime. Deux décodeurs du même fait divergeraient — et c'est précisément ce que la séparation en couches évite. |
| parties du corps touchées, camouflage, accroupissement, vies restantes, dernier traître, loadout moteur, visée de contrôle | **aucune destination dans le produit**. On les laisse sautées. Les rouvrir demanderait une raison, pas une possibilité. |
| la **surchauffe** d'arme | lue mais non interprétée, et elle **borne le parse** des munitions — elle sert déjà, autrement. |

---

## ÉTAPE 1 — LE COMPTEUR DE RÉAPPARITION ET L'HORLOGE DE MANCHE

Ces deux-là sont **déjà décodés et capturés** — ils font partie des quatre composants qui
rendent une valeur. Ils ne sortent pourtant jamais de `filmdec` : leurs seuls appelants sont une
sonde jetable.

Or le rejeu affiche aujourd'hui un compteur de réapparition **déduit** de l'image de départ de
la vie suivante. Le film porte le compteur **réel**. C'est exactement le geste que ce chantier
répète : remplacer une inférence par une lecture.

- [ ] 1.1 Publier `player-respawn-timer` dans l'artefact, sur la grille du rejeu.
- [ ] 1.2 **Le contrôle, énoncé avant** : le compteur lu doit décroître régulièrement et
      atteindre zéro à l'image où la vie suivante commence. L'écart avec la déduction actuelle
      **se mesure** — 90 épisodes de mort, dont 82 avec un retour lisible, médiane 8,0 s.
- [ ] 1.3 Faire du compteur lu la **source**, et de la déduction un **repli explicite et
      compté** — comme le rejeu le fait déjà ailleurs. Les 8 épisodes sans retour lisible
      trouveront peut-être ici leur réponse.
- [ ] 1.4 Publier `game-engine-round-timer` : le rejeu ne sait pas aujourd'hui où en est la
      manche. Sur un mode à manches (KOTH), c'est une information de premier plan.

**GATE 1** : le compteur lu et la déduction s'accordent sur les 82 épisodes mesurables, ou leur
écart a une cause nommée.

---

## ÉTAPE 2 — LE PITCH, L'ORIENTATION DU CORPS, LA VÉLOCITÉ

Tous trois sont décodés dans le **même record** que la position — donc au même instant, sur le
même joueur — et jamais publiés. Seul le lacet de visée sort, sous `Point.H`.

**Mesurer avant de publier**, parce que le lacet n'est présent que sur ~52 % des points et que
rien ne dit que les autres champs le soient davantage.

- [ ] 2.1 Mesurer la **couverture** de chacun : sur combien de points sur les 29 221 ?
- [ ] 2.2 Mesurer leur **plausibilité** : un pitch doit tenir dans [−90, +90] degrés ; une
      vélocité doit rester sous la vitesse de course du jeu. Des valeurs hors bornes signent une
      lecture hors position — c'est le même témoin de FORME qui a validé le bouclier
      (27 404 quanta sur 27 404 dans la plage attendue).
- [ ] 2.3 **Le témoin croisé** : l'orientation du corps doit être proche du **sens de
      déplacement** la plupart du temps, et s'en écarter quand le joueur recule. Le lacet de
      visée a été validé exactement ainsi — écart médian de 4° contre 90° pour un témoin
      aléatoire.
- [ ] 2.4 Publier **seulement ce qui passe** 2.2 et 2.3. Un champ à faible couverture peut être
      publié s'il porte son âge, comme le bouclier ; un champ implausible ne se publie pas.

**GATE 2** : chaque champ publié a sa couverture et son témoin écrits à côté.

---

## ÉTAPE 3 — L'AFFICHAGE, S'IL Y A LIEU

- [ ] 3.1 Le **pitch** enrichit le cône de visée : un cône court quand le joueur vise le sol,
      long quand il vise loin. À décider à l'écran — ce n'est pas évident et ça peut nuire.
- [ ] 3.2 L'**orientation du corps** distinguée du regard : deux marques, jamais confondues.
      C'est la même leçon que la primaire et l'arme en main, où les confondre avait coûté cher.
- [ ] 3.3 Le **compteur de manche** au bandeau, à côté du chronomètre.
- [ ] 3.4 i18n FR + EN, tokens sémantiques.

**GATE 3** : revue visuelle. Si un ajout charge l'écran sans rien apprendre, on le retire —
la carte a déjà huit joueurs, des tirs, des lancers et des projectiles.

---

## CE QUI PEUT FAIRE ÉCHOUER CE PLAN

- **La couverture peut être trop faible.** Le champ le mieux couvert du rejeu, le lacet, est à
  52 % ; la vie est à 0,6 % et n'est **pas** affichée pour cette raison. Un champ à 5 % ne
  s'affiche pas, même bien décodé.
- **Le compteur de réapparition lu peut contredire la déduction.** Ce serait une bonne nouvelle
  déguisée : cela voudrait dire qu'on affiche aujourd'hui quelque chose de faux, et il vaut
  mieux le savoir.
- **L'écran peut se charger.** C'est le risque le moins technique et le plus réel.
