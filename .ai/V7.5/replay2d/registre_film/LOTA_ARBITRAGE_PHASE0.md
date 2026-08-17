# Lot A — arbitrage superviseur apres la phase 0 (2026-08-17, soir)

> Lu sur pieces : `LOTA_PHASE0.md`, `LOTA_gates.log` (EXIT_* = 0, KILLED_OVER_3GB=False x15),
> `oracle_lotA.tsv`, `lotA/24dbb67d.tsv`, `lotA_d1_chaine.tsv`, commit `4a9f72f29`.

## Verdict tenu, cause requalifiee

Le GATE 0 est NON ATTEINT tel qu'ecrit (A.0.1 7/12, A.0.2 10/12, A.0.3 79/96) — les seuils ne
bougent pas. Mais la lecture des ecarts dit que le DECODEUR n'est pas en cause la ou l'oracle est
comparable, et que quatre des cinq ecarts ont une cause identifiable qui n'a pas ete mesuree :

1. `24dbb67d` (Oddball) : les emissions des slots 6/8 S'ARRETENT a 290 683 ms sur un match de
   519 s (`lotA/24dbb67d.tsv`) ; 100/78 est la MANCHE 1 (team 6 gagne 100-78 vers 290 s), la
   manche 2 (100 / 43 -> total 200/121) est INVISIBLE a la grammaire d'ancrage. Les frags
   « faux » (somme 48 vs 88) sont ceux d'UNE manche. Ce n'est pas un negatif de mode : c'est une
   grammaire qui ne voit pas la 2e manche (forme d'en-tete, slots/generation, ou composants
   `finalized-rounds` i28-i55 rejetes). A MESURER, pas a conclure.
2. `606d9844`, `8076f97f` (KOTH) : le film compte des collines (0/3, 3/0), coherent avec le
   vainqueur ; `CoreStats.Score` de l'API vaut 105/8 et 78/105 sur ces deux matchs et 3/2, 4/2
   sur les deux autres — l'ORACLE n'est pas comparable, le decodeur n'est pas juge par lui.
3. `64e8adfa` : film tronque de 24,6 s (couverture 0,97) — un film incomplet ne peut pas etre
   compare a un score FINAL ; la couverture est une qualification du corpus, a publier.
4. `7344d24f` (Strongholds 200/126 vs 193/112) : le seul ecart sans cause — a instruire.

Corpus : `06dfe6d9` est un BTB:Fiesta CTF (26 participants), pas un Slayer (erreur du plan §1.2,
a corriger au plan maitre) ; Slayer n'a qu'un film, Oddball un seul : aucun taux par mode n'est
significatif a n=1. Cout machine mesure : 0,4-2,8 s et <= 17,2 Mo par film — etendre le corpus
ne coute rien.

## Decision : phase 0-bis (memes seuils, oracle et corpus rendus comparables)

- A.0b.1 **Oddball, manche 2** : ou sont les records statborg apres 290 s ? (a) images-cles :
  slots ti=6 du World avant/apres 290 s (memes slots ? generation ?) ; (b) chaine, paquets PROPRES
  apres 290 s : slots, composants annonces, forme d'en-tete ; (c) selon la cause, ETENDRE la
  grammaire d'ancrage DANS L'INSTRUMENT (test) : forme d'en-tete != 0 et/ou composants
  `statborg-finalized-rounds-values-stat-component` (i28-i55, grammaire de la chaine
  `components_batch3.go:65-77` : `R(32)` masque + par bit `2 x {R(1)[si 0 : varW]}`), et
  reconstruire score = somme des manches finalisees + manche courante. Seuil : 200/121 exact et
  frags 88 exacts sur `24dbb67d`. Le champ `game-engine-current-round` (ti=0 i4) sera lisible via
  le hook du lot 0 (item 0.6) — pas encore fusionne : ne l'attends pas, la borne de manche se lit
  aussi a la premiere emission finalisee.
- A.0b.2 **KOTH** : oracle = (1) VAINQUEUR (outcome des participants) : le slot au score le plus
  haut est l'equipe gagnante — 4/4 attendu ; (2) exactitude deja acquise sur les 2 films en
  collines ; (3) pour `606d9844`/`8076f97f` : timeline des +1 vs evenements de colline `th=10`
  (approx) s'ils existent ; ecrire ce que vaut `CoreStats.Score` de l'API pour ces 2 matchs
  (decouverte, pas notre bug).
- A.0b.3 **`7344d24f`** : instant ou le film atteint 200 vs fin du film (596 s) ; `time_played`
  et `outcome` des participants ; hypothese « API figee a la sortie du dernier joueur suivi ». Si
  200 est atteint avant la fin et le vainqueur concorde : decodage juste, oracle a ecrire.
- A.0b.4 **Corpus n >= 3 par mode** : +2 Slayer (Arena, pas Fiesta si possible), +2 Oddball,
  +1 Strongholds, +1 KOTH, choisis dans le registre parmi les films du cache, avec la COUVERTURE
  du film calculee AVANT (>= 0,99 pour entrer dans la comparaison de score final ; les films
  < 0,99 restent dans la table avec leur couverture, hors denominateur du score final, DANS le
  denominateur des compteurs joueurs « a l'instant de fin du film » si l'oracle le permet — sinon
  ecrits a part). Un film par processus, plafond RAM, avant-plan.
- A.0b.5 **D1** : caracteriser les 42 records de plus vus par la chaine sur `000d5950` (slots,
  composants, forme d'en-tete) — le lien avec A.0b.1 est probable (records rejetes par
  l'ancrage). D1 reste : l'ancrage est la source (la chaine desynchronise 61-68 % des paquets).
- A.0b.6 **Re-verdict du GATE 0** avec les MEMES seuils (A.0.1 >= 90 % ET >= 4 modes sur 5 tenus ;
  A.0.2 >= 90 % ; A.0.3 >= 90 % par mode) sur le corpus comparable ; table par film avec la
  colonne couverture ; negatifs ecrits par mode si un mode ne tient toujours pas.

Rien de la phase 1 ne s'ouvre avant ce re-verdict. `filmdec/statborg.go` reste en place.

## Decouvertes recues, a porter par le superviseur

- plan §1.2 : `06dfe6d9` = BTB:Fiesta CTF (26 participants) ; le statborg n'a que 8 slots
  joueurs (plafond structurel en BTB, a ecrire dans `coverage`) ; `match_registry.player_count`
  incoherent (0 sur `530820e5`, `696a9d7c`) ; la chaine `filmdec` desynchronise sur 61-68 % des
  paquets delta (utile aux lots B/C/D/P : un hook dans `consumeByName` ne voit qu'un tiers des
  paquets — les scanners par bande de slots restent la voie).
