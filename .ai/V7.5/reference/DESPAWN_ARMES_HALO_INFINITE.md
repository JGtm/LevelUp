# Despawn des armes, equipements et objets — regles du jeu (source communautaire)

> Releve le 2026-08-30 depuis le guide Steam « [HOLD] Halo Infinite: Spawn times for Weapons,
> Powerups, Equipment & Vehicles on every map ». La page en ligne renvoyait une erreur ; la
> copie locale fournie par l utilisateur a servi de source. **Donnee COMMUNAUTAIRE, non
> officielle** : elle sert de repere d interpretation, jamais de verite-terrain pour valider un
> decodage.

## Ce que cette source change pour nous

Elle explique un resultat de mesure qui paraissait etre un defaut de decodage et qui n en est
pas un. Mesure du 2026-08-30 : seules **5 a 14 %** des armes au sol recoivent un evenement de
disparition dans le film, avec des durees de vie sans coherence d un match a l autre. On en avait
conclu « on n a pas le despawn ». La regle du jeu dit pourquoi : **il n existe pas de minuterie
de despawn inconditionnelle**.

## La regle de despawn (armes lachees)

Le compte a rebours ne demarre PAS a la chute de l arme. Il depend de la position ET du regard
des joueurs :

- **Zone verte** (~1 longueur de Wraith) : tant qu un joueur y est, aucun compte a rebours, quoi
  qu il regarde. Le compte a rebours ne demarre qu en SORTANT de la zone verte ET en ne regardant
  PAS l arme.
- **Zone orange** (~3 longueurs de Wraith) : rester dedans ne suspend le compte a rebours que si
  l on REGARDE l arme. En sortir le demarre, meme en la regardant.
- Rentrer dans une zone **gele** le compte a rebours. Ressortir le reprend la ou il en etait.
- **Un joueur qui reste dans la zone verte en regardant ailleurs, ou dans la zone orange en
  regardant l arme, empeche indefiniment le despawn.**
- Les murs entre le joueur et l arme ne changent rien. Le scan non plus.

Deux autres facons de faire disparaitre une arme : epuiser ses munitions, ou prendre des
munitions du meme type sur un socle en la tenant.

## Durees de despawn, une fois le compte a rebours lance

| duree | armes |
|---|---|
| **0:30** | Sniper (S7 et Flexfire), Skewer (et Volatile), Rocket Launcher / M41 Tracker, Cindershot (et Backdraft), Gravity Hammer (et Rushdown), Energy Sword (et Duelist), Ravager (et Rebound), Needler (et Pinpoint) |
| **0:20** | Heatwave (et Scatterbound), Hydra (et Pursuit) |
| **0:10** | Shock Rifle, Stalker Rifle, Mangler, Bulldog, Sentinel Beam, Battle Rifle / BR75, Commando, Assault Rifle, Sidekick, Disruptor, Pulse Carbine, Plasma Pistol (et leurs variantes) |

| duree | equipement / objet |
|---|---|
| **0:20** | Grapple Hook, Repulsor, Drop Shield, Thruster, Threat Sensor |
| **3:00** | Power Seed (0:10 s il tombe dans un volume de mort douce) |
| **AUCUN** | Hardlight Coil, Plasma Coil (ne disparaissent jamais d eux-memes) |

## Trois categories de reapparition

- **Armes regulieres** : le compte a rebours de reapparition demarre au RAMASSAGE, meme si le
  joueur garde l arme en main.
- **Armes « munitions/despawn »** : il demarre quand toutes les munitions sont epuisees OU quand
  l arme a disparu de la carte.
- **Armes a reapparition statique** : minuterie lineaire depuis le debut du match, ininterrompue.

## Consequences pour le rejeu — ce qu on peut et ne peut pas afficher

1. **Ne jamais afficher une duree de vie calculee a partir d une minuterie fixe.** Une arme peut
   rester au sol indefiniment ; une autre disparait en 10 s. Ce serait inventer une donnee.
2. **La borne haute par l image-cle reste la seule honnete** (l arme n est plus recensee au
   releve suivant). Sa granularite de ~20 s est du meme ordre que les durees de despawn
   elles-memes, donc elle n est pas grossiere pour cet usage : elle est adaptee.
3. Les durees ci-dessus donnent un **controle de vraisemblance** : une arme au sol que le rejeu
   montrerait presente 5 minutes apres son lacher, alors que des joueurs sont passes loin et sans
   la regarder, signale une erreur de lecture — pas une propriete du jeu.
4. La regle « proximite + regard » est en principe reconstructible : on a les trajectoires de
   tous les joueurs ET leur cap de visee (`Point.H`). C est une piste, pas un acquis, et elle
   demanderait un oracle avant d etre publiee.
