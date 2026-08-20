# Damage Efficiency et Enemy Efficiency

> Pour le cadrage produit et les integrations Go/React liees a cette famille de metriques, voir aussi `.ai/go_migration_v2/DAMAGE_EFFICIENCY_INTEGRATION.md`.

Ce document résume la façon dont Spartan Record calcule ces deux statistiques.

## Base du calcul

Le code utilise une constante de 225 dégâts comme quantité de dégâts nécessaire pour un perfect kill.

Source principale : src/Objects/Model/ServiceRecord.tsx

## Damage Efficiency

Formule interne :

Damage Efficiency = 225 / (damage dealt / kills)

La formule peut aussi se lire comme ceci :

Damage Efficiency = (225 x kills) / damage dealt

Interprétation :

- 100 % signifie 225 dégâts en moyenne par kill.
- Moins de 100 % signifie qu'il a fallu plus de dégâts que le minimum théorique pour obtenir un kill.
- Plus de 100 % peut arriver si les kills sont souvent finis sur des ennemis deja affaiblis.

Si damage dealt = 0 ou kills = 0, la valeur renvoyée est 0.

## Enemy Efficiency

Formule interne :

Enemy Efficiency = 225 / (damage taken / deaths)

La formule peut aussi se lire comme ceci :

Enemy Efficiency = (225 x deaths) / damage taken

Interprétation :

- 100 % signifie 225 dégâts encaissés en moyenne par mort.
- Moins de 100 % signifie que l'ennemi a eu besoin de plus de dégâts que le minimum théorique pour te tuer.
- Plus de 100 % peut arriver si certaines morts arrivent alors qu'une partie des dégâts n'est pas créditée de façon "parfaite" dans cette lecture simplifiée.

Si damage taken = 0 ou deaths = 0, la valeur renvoyée est 0.

## Affichage dans l'interface

Dans l'interface, ces valeurs sont affichées comme des pourcentages :

- Damage Efficiency affiche 100 x damageEfficiency
- Enemy Efficiency affiche 100 x enemyDamageEfficiency

Exemple : une valeur interne de 0,8 est affichée comme 80 %.

## Exemple rapide

Si un joueur fait 4500 dégâts pour 20 kills :

- dégâts moyens par kill = 4500 / 20 = 225
- damage efficiency = 225 / 225 = 1,0
- affichage = 100 %

Si un joueur subit 5400 dégâts pour 20 morts :

- dégâts moyens subis par mort = 5400 / 20 = 270
- enemy efficiency = 225 / 270 = 0,833...
- affichage = 83,3 % environ

## Emplacements du code

- Service record global : src/Objects/Model/ServiceRecord.tsx
- Données d'un match joueur : src/Objects/Pieces/PlayerMatchPlayer.ts
- Variante match : src/Objects/Pieces/MatchPlayer.tsx
- Affichage UI : src/Assets/Components/Breakdowns/DamageBreakdown.tsx
