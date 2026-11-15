# Mesure 3.1 — morts sans type : lâcher de commande (S1) et compte à rebours (S2)

Date : 2026-11-15. Lecture seule. Décodeur à la tête `origin/main` 558af5f (schéma 121), worktree détaché
`C:\t\bl1115-mesure\decodeur`, sonde hors suivi `replay/bl1115_morts_research_test.go` (`//go:build research`,
jamais commitée). Films lus un par un, un processus par film, pic mémoire maximal 2,26 Go (BTB de 24 sièges).
Oracle : copies à froid de `shared_matches_v2.duckdb` (courante et `avant-purge-20261026`) sous `base\`.

## 1. Réponse courte

- **S1 tient le critère du plan.** Aucune vie que le jeu supprime encore commandée (fin de manche, remise en
  place) n'est classée mort : 653 vies de ce genre sans compte à rebours, 0 mort ; ni aucun des 21 lâchers de fin
  de match. Accord avec la feuille, morts non comptées
  retirées : 99,6 % (plusieurs manches), 100 % (une manche), 97,9 % (BTB), contre 89,1 %, 98,5 % et 88,1 % avec
  les seules morts typées d'aujourd'hui.
- **S2 confirme S1 à la frame près.** Quand les deux canaux sont lus, le lâcher et le début du compte à rebours
  tombent sur la même frame : 14 190 sur 14 204 morts fraggées, 481 sur 538 vies sans frag.
- **Recommandation : S1 seul pour la croix**, avec une tolérance de 6 frames entre le dernier échantillon de la vie
  et le lâcher (tolérance mesurée, section 3.1). S2 reste une contre-épreuve de mesure, pas une condition : il
  manque aux grands salons (305 morts fraggées sans compte en BTB, 6,5 %), là où S1 voit le lâcher.
- **Pour l'accord avec la feuille, il faut retirer les morts que le tableau ne compte pas.** Sans ce retrait,
  S1 compte 124 morts de trop sur les films à plusieurs manches : ce sont de vraies morts (lâcher et compte à la
  même frame), survenues de 0 à 7 s avant le début de la manche suivante ou de 3 à 7 s avant la fin du film. C'est
  la nuance de Den, vérifiée ici. Le retrait se fait par le fil des temps forts du film (le jeu y écrit toute mort
  que le tableau compte) ; voir 4.3.

## 2. Méthode

### 2.1 Échantillon

| Catégorie | Films décodés | Joueurs confrontés à la feuille | Attribution ligne par ligne refusée |
|---|---|---|---|
| Plusieurs manches (`multi.txt` de la phase C, 59 films) | 54 | 504 | 0 |
| Une manche hors BTB (Slayer, Fiesta, CTF, Bases, Colline, classées) | 16 | 130 | 0 |
| BTB (14 tirés au sort, puis 15 de plus pour trouver des attributions refusées) | 29 | 673 | 5 |

Non décodés : cinq films à plusieurs manches absents des deux bases (pas de carte, pas d'oracle : `014fbf5c`,
`1acc197a`, `46d06dac`, `d9fc8a26`, `fc0d4e31`) et un BTB dont le dossier est vide (`33b9fbe9`).

### 2.2 Ce que la sonde fait, par film

1. `killsource.Decode` sous l'entrée de catalogue de la carte (nom de carte de la base), puis la cuisson du
   décodeur (`scanFilmInputs` puis assemblage, sous le profil calibré par `killsource`, comme `replaybuild`) :
   pistes, roster, `respawnCountdowns`, et les entrées du balayage (`UnitControls`, fil des morts).
2. Population : la règle du web, recopiée. Une vie est une piste d'humain nommé (xuid) ; close si sa fin précède
   l'horizon moins 3 s (`SURVIVOR_GRACE_MS`) ; revendiquée par un frag si un kill de la même victime tombe à moins
   de 2 s (`KILL_MATCH_TOL_MS`) ; veto d'un kill sans victime à moins de 4 s. Les frags sont les kills de
   `killsource`, victime résolue par le fil des morts (même table gamertag -> xuid que `replaybuild.killRefs`).
   Mort typée : une mort non revendiquée de `killsource`, même xuid à moins de 2 s, publiée seulement si
   l'attribution ligne par ligne passe et que `killicon.NeutralDeath` donne un pictogramme (règle de
   `replaybuild.neutralDeaths` et `attachDeathKinds`).
3. **S1** : lectures `unit-control` du slot de la vie, frames calculées comme le document (`frameOf`, origine = premier
   paquet de position). La vie commence à la première lecture commandée après la fin de la vie précédente du même
   slot ; elle finit au premier `Controller = -1` (lâcher), ou à une lecture d'un autre joueur sur le slot (corps
   supprimé puis recréé). Classes : lâcher (écart au dernier échantillon de -2 à 6 frames), lâcher tardif (plus de
   6 frames), supprimé encore commandé (aucun lâcher avant la fin des lectures ou la reprise du slot), non lu
   (aucune lecture du slot).
4. **S2** : début (`from`) du compte à rebours du même joueur (`filmIndex` du roster) le plus proche de la fin de vie ;
   on retient aussi s'il tombe sur la frame même du lâcher.
5. Témoins descriptifs (pas des critères) : présence d'une mort du joueur dans le fil des temps forts à moins de 2 s
   (noté « fil ») ; distance au prochain début de manche (début de manche réelle lu dans le tableau des scores, ou
   remise en place collective : au moins deux vies supprimées encore commandées à moins d'une seconde) et à la fin
   du film.

Frame = 33 ms sur les 99 films.

### 2.3 Oracle

Morts par joueur de `match_participants.deaths`. Calcul confronté : frags subis + morts typées + morts sans type
prouvées par le signal. Les morts que le tableau ne compte pas sont retirées en ne gardant, parmi les morts
prouvées, que celles que le fil des temps forts enregistre (variante notée « c »). Accord = part des joueurs dont
le calcul égale exactement la feuille.

## 3. Comptes par signal

### 3.1 Sur les morts certaines (vies revendiquées par un frag)

| Catégorie | Vies fraggées | S1 lâcher | S1 supprimé commandé | S1 non lu | S2 compte | S2 aucun |
|---|---|---|---|---|---|---|
| Plusieurs manches | 9 274 | 9 058 (97,7 %) | 56 | 160 | 9 096 (98,1 %) | 178 |
| Une manche | 1 341 | 1 273 (94,9 %) | 3 | 65 | 1 317 (98,2 %) | 24 |
| BTB | 4 675 | 4 340 (92,8 %) | 57 | 277 | 4 370 (93,5 %) | 305 |

Écart lâcher - dernier échantillon sur les vies fraggées (frames) : -2 : 11, -1 : 60, 0 : 3 977, 1 : 9 330,
2 : 693, 3 : 184, 4 : 208, 5 : 187, 6 : 21, au-delà : 0. Les écarts de 4 à 6 frames viennent des films BTB
(`0862dce4`, `084a804d`, `6b5de1d7`, `41722c72`, `43880965`), où les positions sont plus espacées. D'où la
tolérance de 6 frames (200 ms) : c'est le maximum observé sur 14 671 morts certaines lâchées, pas une fenêtre
choisie.

Sur les morts certaines, S1 ne se trompe que par omission (corps ou lâcher non lu : 2,3 % à 7,2 % selon la
catégorie). Les lâchers qui ne sont pas des morts existent (fin du match qui lâche les vies, classe D ci-dessous) :
ils tombent tous au-delà de la tolérance.

### 3.2 Sur les vies closes sans frag (population de la croix)

| Classe | Plusieurs manches | Une manche | BTB | Lecture |
|---|---|---|---|---|
| A. lâcher (<= 6 frames) et compte à la même frame | 240 (118 fil) | 23 (4 fil) | 162 (132 fil) | mort, deux canaux |
| B. lâcher (<= 6 frames), aucun compte | 7 (3 fil) | 1 (1 fil) | 49 (46 fil) | mort ; joueur sans compte lu (grands salons) |
| C. lâcher tardif et compte à la même frame | 41 | 5 | 10 | mort non comptée (positions arrêtées avant) |
| D. lâcher tardif, aucun compte | 3 | 0 | 18 | fin du match qui lâche les vies (18 sur 21 à moins de 3 s de la fin du film) |
| E1. supprimé encore commandé, aucun compte | 525 | 16 | 112 | fin de manche, remise en place |
| E2. supprimé encore commandé, compte | 2 (1 fil) | 0 | 6 (6 fil) | mort dont le lâcher n'est pas lu |
| F1. non lu, aucun compte | 10 | 0 | 23 (3 fil) | indéterminé |
| F2. non lu, compte | 2 (2 fil) | 0 | 14 (14 fil) | mort dont le corps n'est pas lu |
| Total | 830 | 45 | 394 | |

Signal retenu (S1, tolérance 6 frames) : morts = A + B. Fins de manche (E1, 653 vies) classées mort : **0**.
Lâchers de fin de match (D, 21) : **0**. Vraies morts manquées : E2 + F2 (24 vies avec compte, dont 23 au fil) et
F1 « fil » (3) : elles restent sans croix et comptées non placées, comme aujourd'hui ; 23 des 26 morts comptées
manquées sont en BTB.

### 3.3 Les deux canaux ensemble

Lâcher et début de compte à la même frame quand les deux sont lus : 14 190 / 14 204 vies fraggées, 481 / 538 vies
sans frag. Les 57 écarts sans frag ne sont pas expliqués par cette mesure (la sonde prend le compte le plus
proche de la fin de vie, qui peut être celui d'une autre mort du joueur).

## 4. Accord avec la feuille

### 4.1 Par catégorie (part des joueurs à égalité exacte ; entre parenthèses, écart absolu cumulé en morts)

| Calcul | Plusieurs manches (504) | Une manche (130) | BTB (673) | dont BTB refusés (107) |
|---|---|---|---|---|
| Frags + morts typées (aujourd'hui) | 89,1 % (96) | 98,5 % (2) | 88,1 % (164) | 57,0 % |
| + S1, toutes morts prouvées | 78,0 % (126) | 85,4 % (19) | 93,2 % (48) | 88,8 % |
| **+ S1, morts non comptées retirées (S1c)** | **99,6 % (2)** | **100 % (0)** | **97,9 % (15)** | **91,6 %** |
| + S2, morts non comptées retirées (S2c) | 99,2 % (4) | 100 % (0) | 97,8 % (47) | 92,5 % |
| + S1 ou S2, morts non comptées retirées | 99,8 % (1) | 100 % (0) | 99,3 % (5) | |
| Témoin : + morts du fil des temps forts (S0) | 99,8 % (1) | 100 % (0) | 99,4 % (4) | 99,1 % |

Critère « accord au moins égal à celui des morts typées » : tenu par S1c dans les trois catégories.

### 4.2 Les morts que le tableau ne compte pas

178 vies sans frag closes par un lâcher (<= 6 frames) n'ont pas de mort au fil des temps forts. 173 tombent dans
l'intervalle que Den nomme : 83 de 0 à 7 s avant le début d'une manche (0 s : 8, 1 s : 13, 2 s : 6, 3 s : 12,
4 s : 13, 5 s : 14, 6 s : 13, 7 s : 4), 90 de 3 à 7 s avant la fin du film. Toutes ont un compte à rebours qui
commence à la frame du lâcher, sauf 7 (section 5). Onze d'entre elles précèdent de moins d'une seconde la remise
en place collective : ce sont des morts (deux canaux), pas la remise en place, qui supprime les corps sans lâcher.

### 4.3 Ce que le retrait suppose

Le retrait des morts non comptées s'appuie sur le fil des temps forts : sans lui, aucune lecture de cette mesure
ne date la décision d'une manche (l'horloge de manche s'arrête aussi en cours de jeu, en Oddball par exemple, et
ne sépare pas la décision). Ce retrait ne sert qu'à l'oracle : la croix, elle, se pose sur toute mort prouvée, comptée
ou non.

## 5. Cas contraires nommés

Joueurs où S1c diffère de la feuille (16, tous des morts comptées que S1 ne voit pas, sauf un) :
- BTB, attribution refusée : `0d0bc019` (3 joueurs, dont le seul excès : +1), `5dcdc3ca` (2), `93fcc545` (4) ;
- BTB : `04023f8a`, `06a883f7`, `145908d1`, `43880965`, `a521164d` (-2) ;
- plusieurs manches : `c75f33b8`, `f3622d94`.
Cause : corps non lu (F) ou lâcher non lu (E2), concentrés sur les grands salons.

Morts S1 sans compte ni mort au fil (aucune seconde preuve) : 7 vies. Quatre à moins de 7 s de la fin du film
(`00ba2e1c`, `036c102a`, `0d0bc019`, `983feabb`), une 7 s avant une manche (`443426df`), une en plein jeu sur un
joueur sans compte lu (`983feabb`, film de 15 sièges), une avant le coup d'envoi (`eba1e63f`, frame 150).
Morts non comptées hors de l'intervalle de Den : les trois de `f7b74a65` (CTF classé, environ 80 s avant la fin,
lâcher et compte à la même frame), et les deux ci-dessus de `983feabb` et `eba1e63f`.

## 6. Contre-épreuve libfilmshell

Sonde Rust hors dépôt (`C:\t\bl1115-mesure\lfs`, `CARGO_TARGET_DIR` sous `C:\t\bl1115-mesure\lfs\target`, aucun
fichier écrit dans la copie de libfilmshell). 73 films lus sur 89 essayés ; 16 refusés (« chunk 0 is not a film
header this library reads »), presque tous les BTB. 68 films confrontés, 599 joueurs humains :
- morts comptées par libfilmshell (kills de la victime moins `is_scored() == false`) = feuille : 100 % ;
- morts humaines sans événement de kill (`Kill::inferred`) : 239, toutes sur des films à plusieurs manches, dont
  **0 comptée** par la feuille : la même population que nos morts non comptées. Par film, les deux comptes ne
  coïncident pas (239 contre 108 à 149 chez nous selon qu'on compte les lâchers tardifs ; 9 films égaux sur 51) :
  notre population exclut les vies des 3 dernières secondes et les vies sans piste publiée. Écart non résolu.
- Den marque une mort au lâcher seulement quand son tableau en compte une (`motion.rs` : « a release the scoreboard
  counts no death at is a bot's removal or the end of the match ») ; ici la tolérance de 6 frames au dernier
  échantillon écarte les lâchers de fin de match (classe D) sans recourir au tableau.

## 7. Recommandation

**S1 seul**, publié par le décodeur. Chiffres qui la fondent : 0 fin de manche classée mort sur 653 (E1), 0 lâcher
de fin de match sur 21 (D) ; accord S1c 99,6 / 100 / 97,9 % contre 89,1 / 98,5 / 88,1 % pour les morts typées ;
deux canaux à la même frame sur 99,9 % des morts fraggées. S2 seul fait presque aussi bien sur la feuille mais
manque 6,5 % des morts fraggées en BTB et toute la classe B (56 morts) ; « S1 confirmé par S2 » perdrait la même
classe B. « S1 ou S2 » gagne 1,4 point en BTB (classes E2 et F2 : 24 vies dont le corps ou le lâcher n'est pas lu
mais dont le compte l'est, une seule sans mort au fil) ; à trancher en 3.2, au prix d'une règle à deux sources.

À trancher aussi en 3.2 : la classe C (56 vraies morts non comptées dont le lâcher suit le dernier échantillon de
plus de 6 frames) n'a de croix qu'en acceptant le compte à rebours comme seconde preuve ; sans lui, elles restent
non placées (sans effet sur la feuille).

## 8. Ce que S1 exige comme publication

Le document n'en porte rien aujourd'hui (la façade lit `unit-control`, aucun champ ne le rapproche des pistes).
Proposition, par piste (`tracks[]`, champs optionnels) :

```
endReason   string  "released" : le joueur lâche la commande du corps ; "deleted" : corps supprimé encore
                    commandé (ou slot repris par un autre joueur) ; absent : aucune lecture du corps.
releaseFrame int    frame du lâcher (axe du document), présent seulement pour "released".
```

et dans `coverage` : comptes `released`, `deleted`, `unread`, plus les lâchers rejetés par la tolérance.
Règle que le web lirait : mort sans type prouvée si `endReason == "released"` et `releaseFrame - endFrame` entre -2 et
6 frames (constante nommée, source : cette mesure). Le décodeur lit déjà les suppressions dans la table d'entités de
la marche : publier « deleted » comme une lecture plutôt que comme l'absence d'un lâcher fermerait la classe F1.
Crédit à Den (libfilmshell, `owners.rs`, `motion.rs`) dans le code et la note de version, comme le plan le prévoit.

## 9. Ce qui n'a pas été mesuré

- Les cinq films à plusieurs manches absents des bases et `33b9fbe9` (dossier vide).
- La décision d'une manche (instant) : aucune lecture de cette mesure ne la date ; seules la distance au début de
  manche suivant et à la fin du film sont décrites.
- Les suppressions de corps ne sont pas lues directement : « supprimé encore commandé » est l'absence de lâcher
  avant la fin des lectures du slot ou sa reprise.
- Les frags viennent de `killsource`, pas du fil servi au web par la base ; sur les cinq BTB refusés, leur
  attribution ligne par ligne est justement celle que le décodeur refuse.
- Les bots (le web ne pose pas de croix sans xuid) et les joueurs sans `filmIndex` : hors population.
- Incident : à l'arrêt de deux boucles successives, deux processus de sonde orphelins ont tourné en même temps
  pendant moins d'une minute (pics de 0,7 Go chacun) avant d'être tués ; toutes les sorties retenues viennent de la
  troisième passe, séquentielle.

## 10. Fichiers

Sous `C:\t\bl1115-mesure\` : `out3\<id8>.tsv` (sortie brute par film), `vies.tsv`, `joueurs.tsv`, `lachees.tsv`,
`lfs_out\`, scripts `analyse.sh`, `joueurs.sh`, `resets.sh`, `resume.sh`, `run1.ps1`, `loop.sh`, `loop4.sh`.
Ces sorties brutes contiennent des xuids : elles restent sous `C:\t\`.
