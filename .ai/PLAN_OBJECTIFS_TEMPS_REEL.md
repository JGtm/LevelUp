# PLAN — les objectifs au tick : qui capture, qui porte, où en est la manche

> Écrit le 2026-07-31. Contrat d'exécution : skill `plan-execution`.
> Objectif : afficher, **à l'instant du rejeu**, l'état vivant des objectifs — une zone en cours
> de capture, un drapeau porté, un crâne tenu.

---

## LE CONSTAT QUI CHANGE L'ORDRE DU TRAVAIL

**Le dépôt en a déjà beaucoup plus qu'il n'en montre.**

| ce qui existe | ce qui en sort |
|---|---|
| un endpoint câblé de bout en bout | — |
| **8 140 événements d'objectif** en base, sur **237 matchs** | le front en consomme **577, soit 7 %** |
| un décodeur de film complet | son seul producteur est un **CLI de diagnostic** qu'aucune synchronisation ne déclenche |
| `map_objectives.json`, 385 Ko, 34 variantes, les 4 socles de Catalyst | **aucun lecteur à l'exécution** |

Autrement dit : **le chemin court ne passe pas par du décodage.** Il passe par brancher ce qui
dort déjà.

### Et le blocage n'était pas celui qu'on croyait

Le SUIVI accusait `interaction-filter` en `i4`, polymorphe à 6 sous-types. **C'est faux.**
Aucun des 162 désérialiseurs portés ne sait lire **un seul** composant d'objectif : la traversée
bute **dès `i0`**. Le mur n'est pas au quatrième composant, il est au premier.

### Le témoin, mesuré

| film | mode | événements d'objectif | entités d'objectif par image-clé |
|---|---|---|---|
| `64e8adfa` | Catalyst, CTF | **68** | **5** |
| `000d5950` | Cliffhanger, Slayer | **0** | **0** |

---

## ÉTAPE 1 — BRANCHER CE QUI DORT  *(aucun décodage, aucune capture)*

C'est le meilleur rapport valeur / risque de tout ce chantier.

- [ ] 1.1 **Faire produire les événements par la synchronisation**, pas par un CLI de
      diagnostic. Aujourd'hui les 8 140 lignes existent parce que quelqu'un a lancé un outil à
      la main : rien ne les maintient.
- [ ] 1.2 **Consommer les 93 % restants** côté front. Un endpoint qui sert 8 140 événements dont
      577 arrivent à l'écran, c'est une fuite silencieuse.
- [ ] 1.3 **Lire `map_objectives.json`** : il contient déjà les socles de livraison de Catalyst,
      et personne ne l'ouvre. C'est ce qui permettra de placer les objectifs **sur la carte**.
- [ ] 1.4 **Corriger deux défauts connus avant de montrer quoi que ce soit** :
      - le décodeur **sur-compte le crâne d'un facteur 20** ;
      - les libellés de capture de colline et de port du crâne **ne sont validés par rien** —
        les retirer ou les marquer comme non vérifiés.

**GATE 1** : le nombre d'événements affichés égale le nombre décodé ; aucun libellé non validé
n'est présenté comme un fait.

---

## ÉTAPE 2 — LA SÉMANTIQUE DÉJÀ ÉTABLIE, ET SES LIMITES

Une chose est mesurée et solide : en mode à zones, **le nombre d'événements d'un joueur égale
`zone_captures + zone_secures`** — exact sur **418 paires sur 503**.

- [ ] 2.1 S'appuyer dessus pour un premier affichage : « ce joueur a pris N zones » **sur la
      ligne de temps du rejeu**, sans prétendre dire laquelle.
- [ ] 2.2 Comprendre les **85 paires qui ne tombent pas**. Un écart de 17 % n'est pas du bruit :
      c'est soit un mode mal classé, soit un événement qu'on compte deux fois.
- [ ] 2.3 **Ne rien étendre au CTF ni à l'Oddball sur cette base** : la relation est établie pour
      les zones, et rien ne dit qu'elle vaut ailleurs.

**GATE 2** : l'écart de 17 % a une cause nommée, ou il est publié comme tel.

---

## ÉTAPE 3 — LIRE L'ENTITÉ D'OBJECTIF  *(exige Cheat Engine, puis Ghidra)*

C'est ici que se trouve le « temps réel » vrai : la **progression d'une capture**, l'état d'un
drapeau, le porteur du crâne.

- [ ] 3.1 **Capturer la table des désérialiseurs** — préalable absolu, 15 minutes, script prêt
      (`SESSION_CAPTURE_AVANT_PC.md`, étape 1). Sans elle, les 34 composants d'objectif ne sont
      pas lisibles du tout.
- [ ] 3.2 Décompiler dans Ghidra les désérialiseurs de l'archétype d'objectif — **hors ligne**,
      le projet est sur la clé.
- [ ] 3.3 Les porter en Go **un par un**, en commençant par ceux dont le nom promet le plus :
      `state`, `progress`, `required-progress`, `object-reference`.
- [ ] 3.4 **Le témoin** : `progress / required-progress` doit être une fraction dans [0, 1] qui
      **monte pendant une capture** et retombe à l'interruption. Une valeur hors bornes ou qui
      ne varie pas signe une lecture hors position.
- [ ] 3.5 Croiser avec les compteurs de l'API : le nombre de captures décodées doit égaler
      `zone_captures` du scoreboard. **Deux sources qui s'ignorent** — c'est le bon contrôle.

**GATE 3** : la progression varie comme une capture, et son compte total tombe sur celui de la
base.

---

## ÉTAPE 4 — L'AFFICHAGE DANS LE REJEU

- [ ] 4.1 **Sur la carte** : les socles d'objectif à leur position (via `map_objectives.json`),
      et l'état de chacun — neutre, en cours de capture, tenu, par quelle équipe.
- [ ] 4.2 **Sur la fiche joueur** : porte le drapeau / porte le crâne. C'est un état de joueur.
- [ ] 4.3 **Sur la ligne de temps** : les instants de capture, comme les éliminations.
- [ ] 4.4 Une valeur non lue reste une **lacune visible**, jamais un état neutre par défaut :
      « on ne sait pas qui tient cette zone » n'est pas « personne ne la tient ».

**GATE 4** : revue visuelle sur le film CTF `64e8adfa` et sur le Strongholds `696a9d7c`.

---

## LES DONNÉES QU'IL FAUT, ET CELLES QU'ON A

| mode | film | statut |
|---|---|---|
| CTF | `64e8adfa` (Catalyst) | **en cache**, 68 événements mesurés |
| Strongholds | `696a9d7c` (Nomad/Vagabond) | **À TÉLÉCHARGER** — absent du cache |
| KOTH | — | **reporté**, noté |
| Oddball | — | **reporté**, noté |

**Attention sur Vagabond** : la carte n'est **ni dans le catalogue de bornes**, ni identifiable
parmi les 31 modules copiés. Son film servira au décodage des objectifs (indépendant de la
carte) mais **pas à construire un rejeu**, tant que son module n'est pas résolu — par la méthode
du `level_id`, qui a déjà réglé 21 niveaux.

---

## CE QUI PEUT FAIRE ÉCHOUER CE PLAN

- **La table des désérialiseurs peut ne pas se lire** si la structure a bougé. Le script imprime
  son propre contrôle (archétype 35 = 64 composants) — s'il ne tombe pas, tout est suspect.
- **Les composants d'objectif peuvent ne pas porter ce qu'on espère.** `progress` est un nom, pas
  une garantie. L'étape 3.4 est faite pour le découvrir tôt.
- **L'étape 1 seule a déjà de la valeur.** Si les étapes 3 et 4 échouent, brancher les 8 140
  événements dormants reste un gain réel.
