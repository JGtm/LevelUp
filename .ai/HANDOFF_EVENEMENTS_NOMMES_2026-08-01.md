# HANDOFF — les evenements de match, nommes et dates a la milliseconde

> Ecrit le 2026-08-01 en fin de session. Branche `feat/re-mode-score` (worktree dedie),
> commits `c75ecfabf` -> `f305032f3`. Etat de l'art : `ETAT_DE_L_ART_MODE_SCORE_EVENEMENTS.md`,
> **sections 15 a 17**. Le handoff du matin (`HANDOFF_MODE_SCORE_CHAINE_2026-08-01.md`) porte
> le volet score ; celui-ci porte le volet **nommage**.
>
> **L'OBJECTIF** : un suivi tres fin, dans le temps, des evenements et du score des matchs a
> objectifs — de quoi le coller aux positions du rejeu 2D.

---

## 1. CE QUI EST ACQUIS

| ce qu'on peut dessiner | statut |
|---|---|
| Courbe de score d'equipe, a la ms, reemise a chaque changement | **ETABLI** (283/284 et 6/6 contre capture CE) |
| Evenements d'objectif dates + acteur (zones, captures CTF) | **ETABLI** |
| **Evenements de joueur NOMMES** : `flag_returned`, `flag_stolen`, `zone_captured`... | **ETABLI pour CTF et zones** |
| Identite slot -> joueur (gamertag ET xuid), sans jointure DB | **ETABLI** |
| Une seule horloge pour tout (manifeste = paquets = footer, ecart < 4 ms sur 573 s) | **ETABLI** |

**Le mecanisme, en une phrase** : chaque entite du match est serialisee dans les paquets
FRAME avec une liste creuse de composants ; l'archetype 6 (statborg) porte **28 emplacements
de statistique** ; **une emission d'un emplacement EST un evenement**, date a la ms et
attribuable au joueur par son slot. On ne lit pas la valeur du score, on lit **quel
emplacement a bouge**.

## 2. LA TABLE — et seulement ce qui a passe le controle

`.ai/refs/TABLE_STATS_STATBORG.tsv`. Ajustee separement sur les films de rang pair et impair
(aucun film partage) ; ne sont gardees que les cles sur lesquelles **les deux moities sont
d'accord**.

| mode | composant | recompense |
|---|---|---|
| CTF | comp 23 A | `flag_returned` |
| CTF | comp 24 A | `flag_stolen` |
| CTF | comp 21 B | `runner_stopped` |
| CTF | comp 22 A | `flag_taken` |
| CTF | comp 20 B | `flag_capture_assist` |
| CTF | comp 0 A · 21 A | `flag_captured` |
| zones | comp 20 B | `zone_captured` |
| zones | comp 21 A | `zone_secured` |
| les deux | comp 2 A · 12 A | `killed_player` |
| les deux | comp 3 A · 12 B | `kill_assist` |

**Le sens d'un emplacement DEPEND DU MODE** : `comp 21 A` vaut `zone_secured` en zones et
`flag_captured` en CTF. Un balayage tous modes confondus rend des noms contradictoires — il a
ete fait, il etait faux, c'est lui qui a appris la regle.

## 3. CE QUI N'EST PAS FAIT, ET IL FAUT LE LIRE AVANT DE CONSTRUIRE

1. **KOTH et Oddball ne sont pas nommes.** Seuls `flag` et `zone` ont ete partitionnes et
   controles. Les recompenses existent (`hill_control`, `hill_scored`, `ball_control`,
   `ball_taken`), le corpus est plus mince ; c'est le meme balayage a rejouer avec
   `MODE=hill` puis `MODE=ball`.
2. **L'oracle ne couvre que les 4 joueurs suivis** (`personal_score_awards` des bases
   joueur) : 588 couples ecartes sur 1 531. L'oracle sur les 8 joueurs est
   `match_objective_stats_latest`, dans `shared_matches_v2.duckdb` — indisponible pendant
   cette session (backfill prioritaire en cours cote utilisateur). **A rejouer quand la base
   se libere** : cela devrait lever une partie des doublons.
3. **Des doublons subsistent**, meme confirmes : `comp 12 A` reproduit `comp 2 A`. Une partie
   est reelle (le statborg duplique des statistiques), le reste demande un depart.
4. **L'ancre d'identite est circulaire pour elle-meme** : le slot d'un joueur est trouve par
   `comp 2 A = son nombre de frags`, donc le nommage de `comp 2 A` ne prouve rien. Les autres,
   si.
5. **Le code du depot lit encore la VALEUR, pas le composant.**
   `objectiveevents.LabelPersonalScore` etiquette par valeur + quota et rend une liste de
   candidates quand la valeur ne discrimine pas. Il fonctionne et il est teste, mais **le
   passage au composant le rendra secondaire**. C'est le prochain lot de code.

## 4. LE PROCHAIN LOT, DANS L'ORDRE

1. **Coder la lecture par composant** : `NamedEvents(src, mode) []NamedEvent` dans
   `internal/analysis/objectiveevents`, adossee a la table du §2, avec des tests de verite
   terrain sur les deux films de reference. Cela remplace `LabelPersonalScore` comme chemin
   principal.
2. **Nommer `hill` et `ball`** (balayage identique, controle identique).
3. **Rejouer le balayage avec l'oracle 8 joueurs** quand la base partagee se libere.
4. **Coller aux positions du rejeu 2D** : chaque evenement porte son `TimeMS` sur l'horloge
   du manifeste, la meme que les positions — superposable sans recalage.

## 5. UNE QUESTION OUVERTE, POSEE PAR L'UTILISATEUR

**« Combien de bases une equipe controle-t-elle a l'instant T ? »** Deux routes existent,
aucune n'est verifiee :

- **par les evenements** : integrer `zone_captured` / `zone_secured` des deux equipes. Cela
  suppose que TOUTE bascule de propriete emet un evenement — a prouver, pas a supposer.
- **par la cadence du score**, et c'est la piste la plus prometteuse. Mesure sur `696a9d7c`,
  equipe slot 6 : **162 emissions a +1 point par seconde**, **8 a +2 points par seconde**, et
  des intervalles plus longs (2, 4, 6, 7, 9, 12, 15, 22, 31 s) a +1. La cadence n'est donc pas
  constante — elle porte de l'information. Reste a etablir la correspondance
  `cadence -> nombre de zones tenues`, en la confrontant aux instants de capture.

**Ce que ca ne dispense PAS de faire** : le releve terrain reste l'oracle de derniere
instance. C'est lui qui a demasque un candidat parfait sur le papier (§5, regle 1) et c'est
lui qui a valide la courbe de score sur quatre ancres. Une derivation, meme elegante, se
verifie contre le terrain.

## 6. LES OUTILS (tous `CGO_ENABLED=0`, non suivis par git)

| outil | role |
|---|---|
| `cmd/tmp_archetype` | les NOMS des composants d'un archetype, lus dans le registre du film |
| `cmd/tmp_statdump` | recensement des composants d'un film ; `COMP=n DELTAS=1`, `HISTO=1` |
| `cmd/tmp_statnames` | **le solveur** ; `MODE=flag\|zone\|hill\|ball`, `HOLDOUT=1 PARITY=0\|1` = le controle |
| `cmd/tmp_scorenames` | bareme des increments + identite des slots par les instants de frag |
| `cmd/tmp_awards` | etiquetage bout en bout par valeur + quota (chemin secondaire) |
| `cmd/tmp_timeline` | ligne de temps fusionnee objectifs + score |
| `cmd/tmp_scoreverify` · `tmp_filmclock` · `tmp_chainhdr` | les mises a l'epreuve du volet score |

## 7. LES REGLES QUE CETTE SESSION A PAYEES

1. **Un label ambigu est inexploitable.** « L'un de trois noms » n'est pas un resultat. La
   bonne lecture n'etait pas la valeur mais le composant — l'objection de l'utilisateur a
   change le chemin.
2. **Toujours partitionner avant d'intersecter.** Le premier balayage melangeait les modes et
   rendait des noms faux avec des chiffres d'apparence solide.
3. **Le controle sur moities disjointes n'est pas une formalite** : il a rejete 8 des 19 cles
   CTF que le balayage donnait pour resolues.
4. **Chercher la bibliotheque avant d'inferer.** `personal_score_awards` et le registre du
   film portaient les noms ; deux tours d'inference statistique auraient pu etre evites.
