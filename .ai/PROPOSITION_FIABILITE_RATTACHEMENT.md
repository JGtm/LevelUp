# PROPOSITION — fiabiliser le rattachement des événements aux joueurs

> Écrit le 2026-07-28, à la demande de l'utilisateur : *« les votes et le concept de permutation ou
> de tri alphabétique n'ont rien à faire là »*.
>
> Il a raison, et les trois défauts n'en font qu'un : **on a remplacé des lectures par des
> inférences**, puis on a comparé nos inférences entre elles.

---

## CE QUE L'EXÉCUTION A CONFIRMÉ, ET CE QU'ELLE A CORRIGÉ — 2026-07-28 (soir)

> Ce document a servi : il a produit `PLAN_REJEU_2D_FIABILISATION.md`, dont les étapes 1 à 5 sont
> closes. Il est conservé tel quel, avec cet encart, plutôt que réécrit — un diagnostic qu'on
> réécrit après coup ne s'évalue plus.

**CONFIRMÉ.** Le vote était bien le goulot, et la direction « on lit avant d'inférer » était la
bonne. Le remplacement porte le rattachement à **496 tirs publiés sur 519 disponibles (95,6 %)**,
avec un contrôle par source disjointe à **96,9 % contre 3,7 %** de témoin.

**CORRIGÉ — un chiffre était périmé.** « 147 tirs publiés sur 519, soit 28 % » datait d'avant
l'ajout de la source « lancers de grenade » au vote. Le point de départ réel était **398 tirs,
soit 77 %**. Le gain est 398 → 496, pas 147 → 443. L'ampleur du défaut était surestimée d'un
facteur trois ; sa nature, non.

**RÉFUTÉ — la CORRECTION 0 n'est pas réalisable en l'état.** Décoder
`player-representation-component` (archétype 5, `i21`) devait « supprimer la raison du vote ». Le
composant existe, il est au registre, et le lien EST sérialisé — tout cela est vérifié. Mais il est
**inatteignable** : le parcours séquentiel du flux delta ne rencontre que 125 records ti=5 sur tout
le film et en désynchronise 47 %, et l'ancrage par signature ne trouve aucun handle de biped dans
les 832 records d'image-clé. Ce qui est réfuté n'est pas le diagnostic, c'est son **estimation de
coût** : « un problème de chaîne de composants » est en réalité un chantier de décodeur.

**C'est donc la CORRECTION 1, présentée ici comme un repli, qui porte le résultat.** Elle n'est pas
une inférence au sens où ce document le craignait : le fil des morts est **lu dans le film**
(chunk highlight), et il nomme par XUID — une identité, pas un ordre. Ce qui reste une résolution,
c'est le pont de l'index de tir vers cette identité ; sa marge est publiée
(`Coverage.bridge.margin`) parce qu'elle est étroite.

---

## LE DIAGNOSTIC EN UNE PAGE

### Ce que le code fait aujourd'hui

```
votes[slot][evenement.PlayerIndex]++     // owners.go — les lancers de grenade VOTENT
out[slot] = le plus vote                 // le gagnant devient « le propriétaire du slot »
...
uniqueSlotFor(...)  ->  (slot, ok)       // shots.go — exige UN seul slot correspondant
if !ok { continue }                      // sinon l'événement disparaît, sans trace
```

### Les trois conséquences, toutes mesurées

| conséquence | mesure |
|---|---|
| La carte de propriété est incomplète | **26 slots couverts sur 99** — 70 lancers de grenade doivent nommer 99 vies |
| Les événements sont jetés en silence | **53 records perdus** entre 26,1 s et 66,0 s, dont ceux des cinq premières morts |
| Le plafond de rattachement est bas | **147 tirs publiés sur 519 disponibles**, soit 28 % |

### Pourquoi c'est structurel et pas accidentel

Un vote a besoin d'électeurs. Les lancers de grenade sont rares (70) et **arrivent tard** — le
premier à 73,1 s. Aucune vie antérieure ne peut donc être nommée, quelle que soit la qualité du
décodage. **Le défaut n'est pas dans les données, il est dans le choix de la méthode.**

---

## LA VRAIE QUESTION, POSÉE PAR L'UTILISATEUR — *pourquoi on parle de vote ?*

Elle est meilleure que la proposition qui suivait. Je proposais de **remplacer** le vote par une
inférence meilleure. La bonne question est : **pourquoi vote-t-on, alors que le format est
déterministe ?**

Réponse, trouvée en la posant : **le lien est écrit dans le film, et personne n'est allé le lire.**

L'archétype **5 est le JOUEUR** (à ne pas confondre avec l'archétype 35, le biped). Il porte
27 composants, dont deux qui n'ont jamais été décodés :

    i10  player-primary-respawn-object-component
    i21  player-representation-component      <- l'entite qui REPRESENTE ce joueur

`player-representation-component` est, par son nom même, l'objet qui représente le joueur dans le
monde — c'est-à-dire son biped, donc son slot. **Le rattachement joueur → entité n'a pas à être
deviné : il est sérialisé.**

Le traverseur atteint déjà l'archétype 5 (il consomme `player-vehicle-entrance-ban-component` en
`i16`). Aller jusqu'à `i21` est un problème de **chaîne de composants**, qui a un chemin de
solution connu — l'ancrage par signature, mesuré à 98,4 % de rappel et **zéro faux positif sur
650 641 positions**. Ce n'est pas un problème qui appelle une heuristique.

**Ce que ça remet à sa place** : les trois défauts relevés ne sont pas trois maladresses, ce sont
trois **symptômes d'un décodage inachevé**. On a comblé un trou de décodage par une inférence, puis
construit dessus. La réparation n'est pas d'améliorer l'inférence, c'est de finir la lecture.

**CORRECTION 0, qui précède toutes les autres** : décoder `player-representation-component`
(archétype 5, `i21`), et brancher le rattachement dessus. Coût : une chaîne de composants à tenir
jusqu'au rang 21. Valeur : le vote, la carte de propriété partielle et le rejet silencieux
disparaissent tous les trois d'un coup, parce qu'ils n'auront plus de raison d'être.

Les corrections ci-dessous restent valables — mais 1 devient un **repli** si la lecture directe
échoue, et non plus la cible.

---

## CORRECTION 1 (repli) — nommer les vies par la mort qui les termine

**Le principe** : chaque vie se termine par une mort, et le fil des morts donne la victime, datée
et attribuée. On nomme donc chaque vie par **la mort qui la termine**, au lieu de la faire élire.

**Ce n'est plus une inférence, c'est une jointure sur un fait.**

Mesures déjà faites :

| | valeur |
|---|---|
| vies nommées | **91 sur 99** |
| écart médian entre fin de vie et mort | **0,0 image** |
| témoin : instants tirés au hasard appariés à moins de 0,5 s | 19 sur 99 |
| accord avec l'attribution actuelle | 86 · désaccord 5, à arbitrer |

**Gain mesuré sur le rattachement des tirs** : 147 → 443 (×3,0), et 0 → 31 avant 66 s.

**Trois contrôles déjà passés** :
1. **Non-régression** — sur les 125 tirs que les deux méthodes rattachent, le slot est identique
   **125 fois sur 125**.
2. **Source disjointe** — l'arme d'un tir nouvellement rattaché appartient au loadout du même slot,
   lu dans les images-clés par une chaîne sans pièce commune : **190 sur 216 (88 %)**, contre
   **8 sur 263 (3 %)** pour une autre vie vivante au même instant.
3. Entièrement hors ligne : le fil des morts se décode du film seul.

---

## CORRECTION 2 — un ordre n'est pas une identité

**Le défaut** : le `pi` du roster est un **tri alphabétique ASCII** que nous imposons. Il a été
utilisé comme s'il était l'index du jeu, et comparé à l'index interne du film. L'écart entre les
deux a même été publié comme une découverte sur le format ; ce n'en était pas une.

**La règle à poser** : *un index est un ordre local, jamais une identité.* L'identité d'un joueur
est son **xuid** — stable, global, indépendant de tout tri. Toute jointure passe par lui.

**Concrètement** :
- renommer le champ pour que son statut soit visible dans le code (`rangAffichage` plutôt que
  `playerIndex`, qui laisse croire à une identité du jeu) ;
- interdire son usage comme clé de jointure, avec un garde-rail — le dépôt sait faire, il en a
  déjà pour les comparaisons de slug ;
- là où le film porte son propre index, le nommer **différemment** (`indexFilm`), pour que la
  confusion soit impossible à écrire.

**Coût** : faible. **Valeur** : ce défaut a produit une fausse découverte en une journée, il en
produira d'autres.

---

## CORRECTION 3 — ne jamais jeter en silence

**Le défaut** : `uniqueSlotFor` rend `(slot, ok)`, et l'appelant fait `continue` quand `ok` est
faux. Les 53 records perdus n'apparaissent nulle part. Le décodeur **ne sait pas** qu'il a perdu
quelque chose, donc personne ne peut le savoir.

C'est un anti-patron déjà interdit ailleurs dans le dépôt — l'erreur avalée, `_ = f()` sans log ni
compteur.

**La règle** : tout événement écarté est **compté** et **catégorisé**, et le compte est publié avec
le résultat. Trois compteurs suffisent : slot introuvable, slot ambigu, hors fenêtre.

**Ce que ça change** : le trou de 34 secondes aurait été visible dès le premier jour, au lieu
d'être découvert par l'utilisateur en regardant l'écran.

---

## CORRECTION 4 — publier une couverture, comme le chantier voisin

Le worktree `filmdec-killweapon` porte une **métrique de santé** et une **porte de publication** :
il sait dire « 371 couples sur 371, verdict nominal, publication ligne par ligne autorisée ». Le
rejeu n'a rien de tel : il publie 147 tirs sans dire que 519 existent.

**À reprendre** : une couverture chiffrée accompagne chaque calque — *rattachés / disponibles*, et
la raison des écarts. Elle se lit dans le POC, à côté des autres chiffres du bandeau.

**Et une règle du voisin, à adopter telle quelle** : *on ne stocke jamais une résolution qui peut
s'améliorer.* Le tag brut se garde à côté du libellé. C'est déjà fait pour les armes de kill ;
ça doit l'être partout.

---

## CE QUI RESTE UNE INFÉRENCE, ET QUI DOIT LE DIRE

Toutes les inférences ne sont pas évitables. Celles qui restent doivent être **marquées comme
telles à l'écran**, ce que le rejeu sait déjà faire par ailleurs (pointillé, estompage) :

- le report en arrière d'une position au début d'une vie (le joueur immobile n'émet rien) ;
- la rémanence d'un lancer de grenade, convention d'affichage de 1,4 s ;
- le compte à rebours de réapparition, appuyé sur une médiane mesurée et non sur une constante.

La frontière est simple : **ce qui est lu se peint franchement, ce qui est déduit se peint
autrement.**

---

## ORDRE PROPOSÉ

| rang | correction | coût | ce que ça débloque |
|---|---|---|---|
| **0** | **Décoder `player-representation-component`** (arch. 5, `i21`) | moyen | supprime la RAISON du vote, pas seulement le vote |
| 1 | *(repli)* Nommer les vies par la mort | moyen | 147 → 443 tirs, et le trou de 34 s |
| 2 | Compter les rejets | faible | rend visible ce qu'on perd, partout |
| 3 | Un ordre n'est pas une identité | faible | empêche la prochaine fausse découverte |
| 4 | Publier une couverture | faible | ferme l'écart entre « publié » et « disponible » |

Les trois dernières sont peu coûteuses et se tiennent ensemble : elles rendent le décodeur
**honnête sur lui-même**, ce qui est le préalable à toute industrialisation.
