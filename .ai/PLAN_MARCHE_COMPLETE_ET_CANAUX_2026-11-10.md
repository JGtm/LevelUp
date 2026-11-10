# Plan : lecture complète des paquets, retrait des balayages, vitesse, lisibilité, infos de match (2026-11-10)

Statut : **go du user le 2026-11-10** (builds récents visés, logique étendue aux builds anciens
quand elle est gratuite, pas de sur-ingénierie, rendus dans un chantier séparé). Exécution sous le
skill `plan-execution`.
Origine : analyse de `OpenSpartan/libfilmshell` (décodeur de Dend, Rust, MIT, publié le 2026-11-09) et
critique reçue sur le décodeur (« very hacky », « inefficient », « odd heuristics »).
Annexes : `.ai/annexes_marche_complete/` (corpus mesuré, sondes hors dépôt).

## 0. Objectif et critères de succès

Le décodeur lit chaque paquet de tick de bout en bout par sa grammaire, sans recherche de motif ;
ce qui reste de récupération est nommé, compté et borné aux builds qui en ont besoin ; le décodage
d'un film se compte en secondes ; un lecteur extérieur comprend le format en lisant un document ;
et les infos de match que le film porte (variante de partie, carte, heure) sont lues et rangées.

| # | Critère | Aujourd'hui | Cible |
|---|---|---|---|
| C1 | Paquets delta fermés au bit près, builds HI_1_12_0 et HI_1_13_0, corpus de l'annexe | **97,1 % et 96,9 % (mesure A0 du 2026-11-10)** ; les 66,9 % et 80,2 % de l'ADR 0037 dataient du 2026-10-01 | **>= 99 % des paquets, par build** (cible relevée, validée par le user le 2026-11-10 ; 97 % était tenu dès A0) |
| C2 | Balayages bit à bit des paquets delta sur le chemin de production, mêmes builds | 22 au 2026-10-01 (ADR 0037), relevé en A3.1 | 0 |
| C3 | Durée de décodage d'un film (film vers faits), 10 témoins de A0 | **médiane 7,8 s, pire cas 37,9 s (mesure A0, options minimales, sans killsource ni objectifs)** | médiane <= 5 s, pire cas <= 30 s |
| C4 | Replis du registre dont le compte est nul sur le corpus | 128 entrées | toutes retirées, avec leur code et leurs tests |
| C5 | Document de format et glossaire | absents | `libs/film/FORMAT.md` (EN) et `libs/film/GLOSSARY.md` (FR-EN) |
| C6 | Infos de match lues dans le film | 3 champs lus | variante complète, carte, heure : lues et rangées dans les faits |

Aucun build lu aujourd'hui ne doit régresser (7 builds, HI_1_4_1 à HI_1_13_0).

## 1. Constats sur pièces (mesures du 2026-11-10)

Mesures faites avec libfilmshell compilé hors dépôt sur 151 films de l'archive locale
(`annexes_marche_complete/corpus_151.tsv`), puis comparées à la base partagée (copie en lecture).

- **Fermeture.** La marche de libfilmshell lit 97,6 à 100 % des paquets de tick jusqu'à leur dernier
  octet (événements, entités, vue de contrôle) sur 8 films, dont un BTB à 33 joueurs et deux films
  HI_1_12_0. Elle n'emploie que deux aides : le type d'une entité non créée dans le film est pris
  dans l'instantané du chunk, et une entrée illisible fait reprendre la lecture au message de tick
  du paquet (reprise comptée).
- **Cause de notre plafond.** L'ADR 0037 attribue 92,5 % des records utiles non fermés à « vue C :
  terminateur hors cadre », c'est-à-dire une largeur fausse à une position inconnue en amont. Les
  `kind` 1 et 2 de la vue de contrôle, non portés (`frame_vue_controle.go`, `ArretVueCKindNonPorte`),
  ne sont qu'une des causes typées. La cause n'est pas localisée ; libfilmshell donne le moyen de
  la localiser (première divergence de position de bit, paquet par paquet).
- **La conclusion « Recovery is necessary » de l'ADR 0037** reposait sur un portage tiers qui
  fermait 5,5 % des trames sans récupération. libfilmshell montre qu'une lecture canonique avec
  deux aides nommées dépasse 97 %.
- **Builds.** libfilmshell refuse 24 films sur 151 : tous ceux d'avant HI_1_12_0 (registre de 37 à
  40 types, décalages fixes). Il ne sert pas de témoin sur les cinq builds anciens.
- **Vitesse.** 1,1 s par film en moyenne chez lui (5,7 s au pire) sur 127 films.
- **Taille.** Lignes de code hors tests, commentaires et tables : 60 500 chez nous (grammaire
  28 100, rejeu 30 500, faits 8 900), 15 900 chez lui. Commentaires : 46 % de nos lignes, 17 % des
  siennes.
- **Ses limites, à ne pas reprendre.** Positions trouvées par balayage à tous les décalages de bit
  (`replication.rs`), attributions par fenêtre et plus proche voisin (grenade, équipement posé,
  arme du kill, tir en tête), sans registre ni compteur.
- **Variante de partie.** Nom, score de victoire, durée, délai de réapparition, dotation, options,
  identifiants de la variante de carte et heure de début sont dans le chunk 0 (Bond), lus par
  `variant.rs` et `header.rs`. Vérifié sur 122 films (exemples : Ranked Strongholds 250, Ranked
  Capture the Flag 5 et 3, Neutral Flag 5). Chez nous `ScoreTimeline.TargetScore` vient de
  `regulation.toml [score_target]`, table tenue à la main, et le commentaire dit que le film ne
  porte pas cette valeur : c'est faux.
- **Mesures gardées pour la suite (hors de ce plan)** : tableau des scores du film égal à l'API pour
  99,7 % des joueurs à 8 ou moins (frags, morts, assistances, tirs en tête), 87 à 94 % au-delà ;
  touches comptées par message de dégât à +-10 % pour 69 à 75 % des joueurs (médiane 1,000) ; tirs
  de libfilmshell à +-10 % pour 54 à 60 % seulement (notre `match_weapon_shots` : 84 %) ;
  l'emplacement 13 du statborg ne vaut pas `shots_fired`.

## 2. Décisions

- **D-1 — La vérité reste le lecteur du jeu.** libfilmshell est une piste et un témoin. Tout
  lecteur porté cite la fonction du jeu (convention `FUN_…` existante). En cas de désaccord entre
  libfilmshell et le désassemblage, le désassemblage tranche ; s'il manque, la fermeture au bit
  près sur le corpus tranche.
- **D-2 — Reprise de code.** Licence MIT : reprise permise dans `libs/film` (AGPL) ; une ligne dans
  `libs/film/NOTICE.md` (auteur, dépôt, commit lu, licence) dans le commit du premier portage.
- **D-3 — Pas de Rust dans le dépôt.** La sonde libfilmshell vit hors dépôt (`C:\t\oracle-filmshell`),
  ses sources sont gardées en `.txt` dans l'annexe. Le dépôt ne lit que ses sorties TSV, par un
  test sous le tag `research`.
- **D-4 — Builds (user, 2026-11-10).** C1 et C2 visent HI_1_12_0 et HI_1_13_0. Chaque lecteur
  corrigé est essayé tel quel sur les builds anciens : si leur fermeture monte sans cas
  particulier, la correction leur est appliquée ; sinon le constat est noté au §6 et rien n'est
  bâti pour eux. Sur les builds anciens les récupérations restent, derrière le profil, comptées.
- **D-5 — Images-clés hors périmètre.** La recherche d'ancres des images-clés (`keyframe_world.go`)
  ne change pas : arrêt décidé par le user (R7-e), et libfilmshell y cherche aussi par en-tête.
- **D-6 — Aucune recuisson en lot dans ce plan.** Une révision de couche qui change marque les
  rejeux « à recuire » ; la recuisson du parc reste un geste du user, demandé à la clôture.
- **D-7 — Pas de renommage de masse.** La lisibilité passe par deux documents et la suppression de
  ce qui ne sert plus ; ni les fichiers ni les identifiants existants ne sont renommés.
- **D-8 — Fond seulement (user, 2026-11-10).** Ce plan ne produit aucun rendu ni aucune table
  d'analyse. Le tableau des scores vivant, la précision (dans le temps, par arme), le MVP dans le
  temps et l'exploitation des infos de match forment un chantier séparé, après maquettes, et la
  précision sera réexaminée une fois le décodeur mis à niveau.
- **D-9 — Pas de sur-ingénierie (user, 2026-11-10).** Aucun outil, abstraction ou garde-rail au-delà
  de ceux que ce plan nomme. Les instruments de mesure sont des tests sous tag `research`, sans
  changement du code de production. Un item qui grossit au-delà du double de son périmètre est
  signalé au user avant d'être poursuivi.

## 3. Branche, worktree, dépendances

- Branche `feat/film-marche-complete`, créée depuis `origin/release/v8.1.2` (`76635f08be`, qui porte
  déjà l'étape film de la 8.1.2). Worktree dédié
  `C:\Users\Guillaume\Downloads\Scripts\LevelUp-wt-marche`. Caches Go dédiés `go-build-marche` et
  `golangci-marche` (skill `plan-execution` §5).
- **Dépendance — chantier 30 images/s** (`feat/rejeu-30ips`, en cours, 65 fichiers de `libs/film`
  touchés, surtout `replay/`) : A0 à A2 travaillent dans `internal/grammar`, `internal/profile` et
  `research/`, sans toucher `replay/`. A3 (registre des replis, versements) et A4 (abonnement des
  canaux à la marche) touchent `replay/` : ils ne commencent qu'après la fusion de
  `feat/rejeu-30ips` dans la release, la branche étant rebasée dessus. A6 range ses faits selon la
  forme des artefacts en vigueur à ce moment-là.
- C3 mesure le décodage (film vers faits) ; la publication (faits vers rejeu) appartient au
  chantier 30 images/s.

## 4. Étapes

Règles : une étape est close quand tous ses items sont statués (`[x]` fait, `[~]` couvert ailleurs
avec référence, `[!]` non traité avec justification) et que sa porte est passée ; l'étape N+1 ne
commence pas avant. Aucun correctif hors périmètre : toute trouvaille va au §6. Une entrée dans
`.ai/thought_log.md` par étape close. Dossier de sortie des mesures : `C:\t\marche\<étape>\`,
jamais sous `data/`. Films lus un à la fois.

### A0 — Témoins et instruments (aucun changement du code de production)

- [x] A0.1 Carte de fermeture d'avant, corpus de l'annexe, un film à la fois :
      `cd libs/film && go run -tags=research ./research/cmd_fermeture -racine <parc>/data/cache/film_chunks -films <ids de corpus_151.tsv> -sortie C:\t\marche\A0\fermeture -mode v2 -paquets`.
- [x] A0.2 Dix témoins de durée choisis dans le corpus (2 Slayer, 2 CTF, 2 Bases, 2 KOTH, 1 Oddball,
      1 CTF en BTB, dont 2 films HI_1_12_0) ; liste écrite dans l'annexe (`temoins_duree.txt`).
- [x] A0.3 Test de mesure sous tag `research` (`libs/film/research/duree/`) : pour chaque témoin,
      décodage (film vers faits) puis publication (faits vers rejeu) vers un dossier temporaire,
      un film à la fois sous `memguard` ; sortie TSV (durée de chaque temps, pic mémoire).
- [x] A0.4 Relevé d'avant des dix témoins avec A0.3, et des compteurs de replis
      (`coverage.fallbacks[]`) de leurs dix rejeux.
- [x] A0.5 Empreintes d'avant : `make replay-equiv` et `make replay-corpus-gate` verts sur la base.
- [x] A0.6 Sonde libfilmshell hors dépôt, étendue à partir de
      `annexes_marche_complete/sondes/sonde_fermeture_filmshell.rs.txt` : une ligne par entrée lue
      (film, chunk, paquet, vue, position de bit de début, type d'entrée, index d'entité, type
      d'entité, liste des composants, position de bit de chaque bloc, position de bit de fin).
      Sortie dans `C:\t\marche\A0\oracle\` ; source versée en `.txt` dans l'annexe.
- [x] A0.7 Test sous tag `research` (`libs/film/research/divergence/`) : lit le TSV de A0.6 et la
      marche de production du même film ; rend par paquet la première position de bit où les deux
      lectures diffèrent et le lecteur en cause (vue, genre de message ou composant).

Porte A0 : `make libs-test` vert ; les TSV de A0.1, A0.4 et A0.6 existent pour tout le corpus lu
par libfilmshell ; A0.7 rend zéro divergence sur 100 paquets que les deux lectures ferment
(contrôle de l'instrument) ; `git diff --stat origin/release/v8.1.2 -- libs/film` ne montre que
des fichiers sous `research/`.

### A1 — Localiser les largeurs fausses

- [x] A1.1 Lancer A0.7 sur les films HI_1_12_0 et HI_1_13_0 du corpus.
- [x] A1.2 Classer les premières divergences par lecteur en cause, en paquets touchés.
- [x] A1.3 Écrire la liste fermée des lecteurs à corriger dans ce plan (addendum daté « A1 ») :
      tout lecteur responsable d'au moins 0,1 % des paquets du corpus, par ordre décroissant, avec
      pour chacun la lecture de libfilmshell et la fonction du jeu à relire.
- [x] A1.4 Même classement pour les causes typées d'arrêt déjà connues (composant non porté,
      message de la vue A non porté, `kind` 1 et 2 de la vue C).
- [x] A1.5 Builds anciens : classement des causes d'arrêt par build avec la carte de A0.1 (sans
      libfilmshell), pour savoir quelles corrections de la liste A1.3 y sont candidates (D-4).

Porte A1 : l'addendum existe ; la somme des paquets expliqués par sa liste, ajoutée aux paquets
déjà fermés, atteint 99 % par build récent. Sinon le constat est rendu au user avec le taux
  atteignable, et A2 ne traite que la liste établie ; pas d'arrêt du plan : A3 et A4 ne dépendent pas
  de ce seuil. (Ancienne rédaction : arrêt du plan ; remplacée le 10/11, la fermeture étant déjà à 97 %.)

### A2 — Corriger les lecteurs de la liste A1

**Addendum A1 (2026-11-10) — liste fermée, détail et fiches dans `annexes_marche_complete/A1_addendum.md`.**
Mesure sous catalogue de carte (contexte de production), 30 films. Fermés aujourd'hui : 97,09 %
(HI_1_13_0) et 97,67 % (HI_1_12_0). Estimation après la liste : 99,08 % et 97,82 % sans compter les
rejets en cascade, 99,85 % et 99,99 % au mieux.

- [ ] A2.P Positions lues aux largeurs de la région (BSP) que le flux désigne, et non à celles de la
      seule région jouée : catalogue de carte étendu à toutes les régions (`cmd/mapquant-build`,
      `map_quant_bounds.json`), `lireE524Sur` sur la ligne de l'index lu, puis `consumeFlockPosition`,
      `consumeObjectPositionMonde` et les positions de la vue A (genres 5 et 6) sur le même lecteur.
      Enjeu : 13 106 paquets en HI_1_13_0 (1,58 %). Jeu : `FUN_14076e524`.
- [ ] A2.T Type d'une entité lié à son identifiant complet (slot et génération), délié à la
      suppression. Enjeu : 1 799 paquets et 908 fermetures fausses. Jeu : `FUN_1406cbaa0`.
- [ ] A2.A `biped-spartan-ability` (ti=35 composant 57), tag 3 : branche sous `flags & 1` lue (le
      commentaire qui la dit inatteignable est contredit par trois occurrences relues). Enjeu : 846
      paquets. Jeu : `FUN_142f262d4`, `FUN_14297ea84`. Après A2.P (largeur des positions).
- [ ] A2.V Vue A en HI_1_12_0 : isoler le lecteur trop court (candidat : table des poignées par genre),
      puis le corriger. Enjeu : 780 paquets directs, 4 644 rejets en cascade (tout l'écart de ce build).
- [ ] A2.H Entités nées dans un chunk précédent et absentes de la table à l'ouverture du chunk
      suivant : garder la table d'un chunk au suivant tant qu'aucune suppression ne la retire. La
      relecture de l'image-clé comme le témoin reste hors périmètre (D-5), sauf décision du user.
      Enjeu : 6 165 paquets (0,72 % en HI_1_13_0, surtout les BTB).
- [!] A2.C États de création (ti=3, 21, 47, 13, 10, 4, 22) : sous le seuil de 0,1 % (265 paquets),
      non isolés par lecteur. Non traités dans ce plan ; à reprendre si C1 n'est pas tenu après A2.H.

Règle commune à chaque item : lecture portée avec la référence du jeu (D-1), test synthétique
au bit près, carte de fermeture rejouée sous carte sur le corpus (les 7 builds), application
aux builds anciens selon D-4 (tous les lecteurs de la liste y existent au même composant),
`TestFrameClosureRatchet` régénéré avec une ligne d'historique.

- [!] A2.v Vue de contrôle, `kind` 1 et 2 : non traité. Mesure A1 : premier désaccord dans 0 paquet
      sur 30 films ; le témoin lui-même n'en lit presque aucun (6 messages sur 756 696). Les porter
      serait du travail sans effet mesuré (D-9).
- [ ] A2.z Révision de la couche `grammar` montée ; ADR 0037 amendé (tableau des faits, section
      « An independent confirmation », IR-12) ; ligne de `NOTICE.md` (D-2).

Porte A2 : C1 tenu (`cmd_fermeture -mode v2` sur le corpus, >= 99 % par build HI_1_12_0 et
HI_1_13_0) ; aucun compte `fermés` en baisse sur les builds anciens, gains écrits par build ;
`make libs-test`, `make replay-corpus-gate` (référence base, zéro perte) verts.

### A3 — Retirer les balayages delta et les replis à compte nul (après la fusion du 30 images/s)

- [ ] A3.1 Rejouer A0.4 après A2 : compteurs de replis et de récupérations des dix témoins ;
      relevé des balayages bit à bit encore appelés sur les paquets delta (liste des sites).
- [ ] A3.2 Récupérations de la marche, une par item, retirée pour les builds où son compte est nul
      sur le corpus, sinon gardée avec son compte écrit :
      `repli_ancrage_bipede_apres_la_marche`, `repli_localisation_largeur_libre`, signature du slot
      123 (`marchLocateStrict`), pistes et créations d'objets du monde, `repli_kill_rattrape_hors_vue_a`,
      début de liste fermé au bit, liaison par anticipation, génération vivante inconnue.
- [ ] A3.3 Registre des replis (`internal/facts/fallback/registre_*.go`) : pour chacune des 128
      entrées, compte sur le corpus ; retrait de celles à zéro avec leur code, leurs tests et leur
      ligne de versement ; les autres gardent leur entrée, la cible de retrait mise à jour.
- [ ] A3.4 Les deux aides gardées de la lecture canonique (type pris dans l'instantané, reprise au
      message de tick), si A2 les a introduites, entrent au registre comme récupérations nommées
      et comptées.
- [ ] A3.5 Garde-rail : test `archlint` qui refuse tout appel à `source.PremierMotif64` depuis un
      lecteur de paquet delta, hors liste datée des builds anciens.

Porte A3 : C2 et C4 tenus (A3.1 rejoué : zéro site de balayage delta atteint sur les témoins
HI_1_12_0 et HI_1_13_0 ; registre sans entrée à compte nul) ; `make replay-corpus-gate` et
`make replay-equiv` rejoués, toute différence expliquée ligne à ligne dans le journal ;
`go test ./internal/archlint/...` vert.

### A4 — Vitesse

- [ ] A4.1 Rejouer A0.3 après A3 ; profil CPU d'un témoin lent ; classer les passes restantes.
- [ ] A4.2 Une seule marche des paquets delta par décodage : les canaux qui relisent le film
      s'abonnent à la marche (modèle déjà en place pour les états de mouvement et le tir continu).
      Un item par canal encore en passe propre, d'après A4.1, tant que C3 n'est pas tenu.
- [ ] A4.3 Baseline de `bench_baseline.txt` re-figée.

Porte A4 : C3 tenu sur les dix témoins (TSV de A0.3). Dès que C3 est tenu, A4.2 s'arrête. Si la
cible n'est pas atteinte une fois tous les canaux abonnés, l'item passe `[!]` avec la mesure et le
profil ; pas de travail au-delà.

### A5 — Lisibilité

- [ ] A5.1 `libs/film/FORMAT.md` (anglais) : conteneur, paquets, les trois vues d'un paquet de
      tick, entrées d'entité, registre des composants, instantanés, chunk des temps forts ; un
      schéma par niveau ; ce qui est lu, ce qui est récupéré, ce qui n'est pas lu.
- [ ] A5.2 `libs/film/GLOSSARY.md` : chaque terme interne (vue A, B, C, `ti`, `iNN`, genre, kind,
      statborg, pont d'identité, repli, récupération, image-clé, trame) avec son équivalent anglais
      et le nom que lui donne libfilmshell.
- [ ] A5.3 README de la brique : section « How it reads a film » renvoyant à A5.1, et tableau des
      taux de fermeture par build mesurés en A2.
- [ ] A5.4 Liste des fichiers `*_research_test.go` par campagne (préfixe de nom), avec pour chaque
      campagne si sa question est close. Liste remise au user ; la suppression est une décision
      séparée du user.
- [ ] A5.5 Commentaires des fichiers touchés par A2 à A4 : contrat seulement, histoire et codes de
      lots déplacés dans la chronique (règle 17).

Porte A5 : les deux documents existent ; `make libs-test` vert ; la liste A5.4 est remise.

### A6 — Infos de match lues dans le film

- [ ] A6.1 Lecteur de la variante (Bond du chunk 0) dans `libs/film/internal/grammar` : nom,
      description, score de victoire, durée, délai de réapparition, prolongation, dotation, options ;
      identifiants de la variante de carte et heure de début dans l'en-tête. Sur les 7 builds
      quand la lecture s'y applique sans cas particulier (D-4).
- [ ] A6.2 Fait persistant « infos de match du film » dans les faits du film, exposé par la façade
      `decfilm`. Aucun consommateur branché (D-8).
- [ ] A6.3 Commentaire de `ScoreTimeline.TargetScore` corrigé (le film porte la valeur) ; la source
      de la valeur ne change pas dans ce plan.
- [ ] A6.4 Mesure écrite au journal : sur le corpus, part des films dont le score de victoire lu
      égale celui de `regulation.toml`, et liste des écarts.

Porte A6 : lecteur vert sur tous les films du corpus des builds où il s'applique ; A6.4 écrite ;
`make libs-test`, `cd apps/go-api && go test ./...`, `go test ./internal/archlint/...` verts.

### Clôture

- [ ] Z.1 Revue adversariale du diff par un contexte frais (skill `adversarial-review`), une ronde.
- [ ] Z.2 `make gate-push` ; CI de la branche verte (`gh run list --branch feat/film-marche-complete`).
- [ ] Z.3 Notes de version (FR et EN).
- [ ] Z.4 Demande au user : recuisson du parc (D-6), suppression des tests de recherche clos
      (A5.4), fusion dans la release ; lignes de backlog pour la suite (§5).
- [ ] Z.5 Purge des caches : `powershell -File scripts/disk-hygiene.ps1 -Lot marche -Apply`.

## 5. Hors périmètre — la suite, à planifier après ce plan

- **Chantier des rendus** (maquettes d'abord) : tableau des scores vivant dans la vue cockpit ;
  MVP dans le temps sur un match, c'est-à-dire le MVP existant (le joueur qui a les meilleures
  statistiques sur l'ensemble du tableau, `analysis/scoreboard_extremes.go`) recalculé au fil du
  match, exploité dans Timeseries, Sessions et Escouade ; exploitation des infos de match (dont
  `TargetScore` lu du film).
- **Données à réexaminer une fois le décodeur mis à niveau** : précision dans le temps et par arme
  (touches par message de dégât, table soeur de `match_weapon_shots`), tableau des scores complet
  (autres emplacements du statborg, plus de 8 joueurs).
- **Fiche du joueur du rejeu** (maquette validée le 2026-11-09, panneau à la place du fil) : ses
  compteurs à l'instant demandent trois données que ce plan ne produit pas : tirs et touches
  (précision), dégâts infligés en points (table des effets de dégât tirée des fichiers du jeu,
  mesure du 10/11 : médiane 1,05 de l'API, 72 % des joueurs à +-10 %), dégâts subis (emplacement
  20 du statborg, médiane 0,96 de l'API). Frags, morts, assistances et score existent déjà.
  Le détail du score par action reste à instruire.
  Précisions du user (2026-11-10) : le barème existe déjà (table `personal_score_awards`, récompenses
  de score par match venues de l'API) ; nos deux méthodes de comptage des touches sont à mesurer
  ensemble ; le BTB est une bêta, on y donne ce que le film permet, sans seuil à tenir.
  Mesure du 10/11 sur 3 témoins : le compteur « dégâts subis » du film monte une fois par vie, à la réapparition (481 ticks après la mort en Slayer, 601 en CTF, sur 100 % des hausses ; 229 points pour une vie sans recharge). Croisé avec la part de dégâts du tueur et de l'assistant que porte le message de mort, il donne des dégâts infligés en vrais points, par vie ; à l'instant, entre deux morts, il faut une estimation corrigée à la mort.
  Piste du user (10/11) pour combler les limites (troisième contributeur, joueur qui ne meurt plus, valeur à l'instant) : croiser le total réel par vie et les parts du message de mort avec une estimation par coup tirée des armes (dégât de base tête ou corps, sur bouclier ou non, coup au but ou non), l'estimation servant à répartir et le total réel à recaler.
- Autres canaux que libfilmshell publie : entrées de déplacement, santé et bouclier à chaque point,
  dégâts par coup en points, munitions et chaleur dans le temps, voix, messages d'écran, sprint.
- **Téléporteurs et canons à homme** (abandonnés chez nous ; question du user le 10/11) : libfilmshell les
  tire du film, pas des cartes. Chaque téléportation est un message du film avec le point de départ et
  le point d'arrivée (32 sur un CTF en BTB mesuré). Les dispositifs fixes (type 43) viennent des
  instantanés du film avec position et tag ; sa table publique n'en nomme presque aucun.
- Images-clés (D-5) ; travail propre aux cinq builds anciens (D-4).
- Cadence du rejeu et compression (chantier 30 images/s), planches d'icônes (chantier séparé).

## 6. Découvertes

(vide)

## 7. Reprise de session

Lire dans l'ordre : ce fichier (cases cochées, addendum A1, journal), les dernières entrées de
`.ai/thought_log.md`, `git log --oneline -15` sur `feat/film-marche-complete`, puis
`C:\t\marche\` pour les mesures de l'étape en cours. Reprendre à la première case non statuée.

## 8. Revue du plan (grille `plan-review`, 2026-11-10, rejouée après le go)

- Structure : objectif, critères chiffrés, étapes ordonnées, branche nommée, blocages (dépendance
  30 images/s avant A3, arrêt possible en A1) : présents.
- Couches : tout le travail est dans `libs/film` (grammaire, profil, faits, recherche) ; aucune
  table, aucun service, aucun handler, aucun écran.
- Multi-titre : sans objet (brique propre à Halo Infinite, rien de neuf côté application).
- Tests : synthétiques au bit près par lecteur (A2, A6), ratchet de fermeture, portes sur corpus
  réel, `replay-corpus-gate` et `replay-equiv`.
- Journalisation : aucun chemin d'erreur neuf côté application ; les replis restent comptés.
- Exécutabilité : périmètres fermés, sauf A2.n et A4.2 dont la liste est produite par une étape
  antérieure selon une règle écrite d'avance (seuil de 0,1 % ; arrêt dès C3 tenu) ; portes en
  commandes ; statuts définis ; décisions au §2 ; reprise au §7.
- Risque principal : A1 peut montrer que la divergence ne se réduit pas à une liste courte de
  lecteurs. Le plan s'arrête alors à la porte A1, sans code de production touché.

## 9. Journal d'exécution

- 2026-11-10 : go du user ; plan révisé (rendus et tables d'analyse sortis, infos de match
  ramenées en A6, D-4 et D-9 ajoutées, dépendance au chantier 30 images/s écrite). Worktree créé.
- 2026-11-10, A0 (mesures d'avant) — **constat qui corrige le §1** : la carte de fermeture mesurée
  sur les 151 films donne 97,1 % des paquets delta fermés au bit en HI_1_13_0 (119 films) et 96,9 %
  en HI_1_12_0 (8 films), 98,0 % et 97,8 % des records utiles. Les chiffres du §1 (66,9 %, 80,2 %,
  plafond à 81,3 %) venaient du tableau de l'ADR 0037, daté du 2026-10-01 et antérieur à la
  campagne de grammaire ; ils n'avaient pas été re-mesurés avant la rédaction du plan. Builds
  anciens : 41,3 % (HI_1_8_0), 32,8 % (HI_1_9_0), 26,5 % (HI_1_10_0), 23,3 % (HI_1_11_0).
  Sur les dix témoins, film par film, nous fermons 84,1 à 99,8 % des paquets là où libfilmshell en
  lit 98,97 à 100 % : l'écart restant va de 0,2 à 15 points. Causes d'arrêt sur les builds récents
  du corpus : sortie de la vue B par rejet (90 425 paquets), terminateur de la vue C hors cadre
  (12 078), liste d'événements non localisée (6 048), `kind` de la vue C non porté (3 400).
  Durées : décodage 4,7 à 37,9 s (médiane 7,8 s), publication 0,16 à 0,88 s. Replis déclenchés sur
  les dix rejeux : 220 072 pour 28 replis distincts, dont `repli_pistes_du_monde_apres_la_marche`
  106 573 et `repli_ancrage_bipede_apres_la_marche` 75 719 : les récupérations tournent encore à
  plein alors que les paquets ferment à 97 %. Conséquence : l'enjeu de A1 et A2 sur les builds
  récents est de quelques points ; le gros du travail utile est A3 (retirer les récupérations
  devenues inutiles) et A4 (vitesse). Signalé au user le 10/11.
  A0.5 : `make replay-corpus-gate` 18 témoins sur 19 sans perte (le 19e, `f75e7053`, n'a pas de
  manifeste dans le parc local) ; `make replay-equiv` n'a rien comparé (le harnais cherche les
  films sous `data/cache` du worktree) : à rejouer avec les jonctions `film_chunks` et
  `film_manifests` vers le parc, comme dans les autres worktrees. Les jonctions se retirent par
  `rmdir` avant toute suppression du worktree.
- 2026-11-10 : cible de C1 relevée à 99 % (validée par le user). Porte A1 réécrite : plus d'arrêt du plan si la liste n'explique pas 99 %.
- 2026-11-10, **A0 close**. A0.5 : `make replay-equiv` 20 identiques sur 20 ; `make replay-corpus-gate
  --base origin/release/v8.1.2 --allow-missing` 18 témoins sur 19 sans perte ni changement, le 19e
  (`f75e7053`) absent du parc local, seul manque admis. A0.7 : test `research/divergence/`, contrôle
  sur 100 paquets fermés par les deux lectures : 0 désaccord de position ; aucune conversion de bit
  nécessaire (mêmes conventions des deux côtés). Porte tenue : `make libs-test` vert, seuls des
  fichiers sous `libs/film/research/` diffèrent de la release.
  Constats de A0.7 sur les dix témoins (`annexes_marche_complete/A0_divergence_resume_global.md`) :
  (1) 685 paquets sur 388 569 que nous « fermons » sont lus de travers (0,18 %) : début de la vue B
  retrouvé par signature à un autre bit que le témoin, le paquet fermant quand même ;
  (2) premier désaccord le plus fréquent hors rejet : `ti=40` composant 0, décalage constant de
  19 bits (aussi `ti=41` et `ti=37` composant 0), 6 953 + 2 852 + 633 paquets ; puis `ti=21`
  composant 16 (+37 bits, constant), `ti=35` composant 57 (variable), type de l'entité (1 700
  paquets, dont `ti=20` chez nous contre 35 chez le témoin : 1 182), largeurs des messages de genre
  5, 6, 82, 36 et 0 de la vue A ;
  (3) les 9 077 rejets « hors datum » visent tous un slot absent de notre table d'entités alors que
  le témoin a vu sa création : création sautée ou suppression fantôme dans un paquet mal lu en
  amont, ou entité perdue au changement de chunk. La première divergence n'est donc pas toujours la
  cause de l'arrêt : corriger les lecteurs de tête doit faire tomber une partie des rejets.
- 2026-11-10, **A1 close**. Mesure élargie à 30 films (liste : `annexes_marche_complete/A1_films.txt`).
  L'instrument de A0 lisait sans l'entrée de catalogue de carte que la production pose ; rejoué sous
  carte : 97,6 % (HI_1_13_0) et 97,7 % (HI_1_12_0) sur le corpus. Décision du pilote : les portes se
  mesurent sous carte. Porte A1 : 99 % atteint au mieux sur les deux builds (99,85 % et 99,99 %),
  en estimation prudente seulement en HI_1_13_0 (99,08 %) ; HI_1_12_0 dépend de la vue A (A2.V),
  dont le lecteur fautif n'est pas encore isolé. Constats : une cause domine (positions lues aux
  largeurs de la région jouée quel que soit l'index de région du flux : 19 bits d'écart sur
  Interlock, 7 sur Academy et Wetland) ; les rejets « hors datum » sont à 93 % des entités nées
  dans un chunk précédent (HI_1_13_0) et à 96 % la suite d'une vue A mal lue (HI_1_12_0) ; les
  `kind` 1 et 2 de la vue C ne pèsent rien. Le désassemblage n'a été relu pour aucune fonction
  (aucune instance Ghidra ouverte) : D-1 retombe sur la fermeture au bit près tant que ce n'est pas
  fait. Builds anciens : le registre des composants est identique de HI_1_8_0 à HI_1_13_0, tous les
  lecteurs de la liste y sont candidats ; leur cause dominante (plus de 95 % des rejets) est une
  création non lue.
