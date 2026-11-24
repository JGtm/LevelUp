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
  `<poste>\Downloads\Scripts\LevelUp-wt-marche`. Caches Go dédiés `go-build-marche` et
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

- [x] A2.P Positions lues aux largeurs de la région (BSP) que le flux désigne, et non à celles de la
      seule région jouée : catalogue de carte étendu à toutes les régions (`cmd/mapquant-build`,
      `map_quant_bounds.json`), `lireE524Sur` sur la ligne de l'index lu, puis `consumeFlockPosition`,
      `consumeObjectPositionMonde` et les positions de la vue A (genres 5 et 6) sur le même lecteur.
      Enjeu : 13 106 paquets en HI_1_13_0 (1,58 %). Jeu : `FUN_14076e524`.
- [x] A2.T Type d'une entité lié à son identifiant complet (slot et génération), délié à la
      suppression. Enjeu : 1 799 paquets et 908 fermetures fausses. Jeu : `FUN_1406cbaa0`.
- [x] A2.A `biped-spartan-ability` (ti=35 composant 57), tag 3 : branche sous `flags & 1` lue (le
      commentaire qui la dit inatteignable est contredit par trois occurrences relues). Enjeu : 846
      paquets. Jeu : `FUN_142f262d4`, `FUN_14297ea84`. Après A2.P (largeur des positions).
- [x] A2.V Vue A en HI_1_12_0 : isoler le lecteur trop court (candidat : table des poignées par genre),
      puis le corriger. Enjeu : 780 paquets directs, 4 644 rejets en cascade (tout l'écart de ce build).
- [x] A2.H Entités nées dans un chunk précédent et absentes de la table à l'ouverture du chunk
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
- [x] A2.z Révision de la couche `grammar` montée ; ADR 0037 amendé (tableau des faits, section
      « An independent confirmation », IR-12) ; ligne de `NOTICE.md` (D-2).

Porte A2 : C1 tenu (`cmd_fermeture -mode v2` sur le corpus, >= 99 % par build HI_1_12_0 et
HI_1_13_0) ; aucun compte `fermés` en baisse sur les builds anciens, gains écrits par build ;
`make libs-test` vert ; `make replay-corpus-gate` (référence base) : zéro perte NON EXPLIQUÉE, chaque
écart classé au journal (faux retiré, vraie perte, indécidable). (Rédaction d'origine : « zéro perte » ;
amendée le 2026-11-10 : une lecture devenue juste retire des données fausses que le gate compte comme
des baisses.)

### A3 — Retirer les balayages delta et les replis à compte nul (après la fusion du 30 images/s)

- [x] A3.1 Rejouer A0.4 après A2 : compteurs de replis et de récupérations des dix témoins ;
      relevé des balayages bit à bit encore appelés sur les paquets delta (liste des sites).
- [x] A3.2 Récupérations de la marche, une par item, retirée pour les builds où son compte est nul
      sur le corpus, sinon gardée avec son compte écrit :
      `repli_ancrage_bipede_apres_la_marche`, `repli_localisation_largeur_libre`, signature du slot
      123 (`marchLocateStrict`), pistes et créations d'objets du monde, `repli_kill_rattrape_hors_vue_a`,
      début de liste fermé au bit, liaison par anticipation, génération vivante inconnue.
- [x] A3.3 Registre des replis (`internal/facts/fallback/registre_*.go`) : pour chacune des 128
      entrées, compte sur le corpus ; retrait de celles à zéro avec leur code, leurs tests et leur
      ligne de versement ; les autres gardent leur entrée, la cible de retrait mise à jour.
- [~] A3.4 Les deux aides gardées de la lecture canonique (type pris dans l'instantané, reprise au
      message de tick), si A2 les a introduites, entrent au registre comme récupérations nommées
      et comptées.
- [x] A3.5 Garde-rail : test `archlint` qui refuse tout appel à `source.PremierMotif64` depuis un
      lecteur de paquet delta, hors liste datée des builds anciens.

- [!] A3.6 **Amendement du 2026-11-13 (constat A3.1, accord du user du 2026-11-13 sur la forme).** Les
      balayages delta ne sont pas des replis : onze sites tournent sur le chemin de production, dont
      quatre sur CHAQUE paquet (`biped_creation.go`, `creations_du_monde.go`, `grenade_events.go`,
      `pistes_du_monde.go` : environ 2 250 positions de bit par paquet en HI_1_13_0). Forme cible : la
      marche est la seule lecture d'un paquet delta ; chaque canal lit ce qu'elle a produit (abonnement,
      modèle des états de mouvement et du tir continu) ; un balayage ne tourne plus que sur un paquet que
      la marche n'a PAS fermé, comme récupération nommée et comptée au registre. Un sous-item par site,
      dans cet ordre (du plus coûteux au moins coûteux, les quatre de chaque paquet d'abord) :
      - [x] A3.6.a créations d'objets du monde (`creations_du_monde.go`)
      - [x] A3.6.b pistes des objets du monde (`pistes_du_monde.go`)
      - [x] A3.6.c créations de bipède (`biped_creation.go`) : par la marche, passe derrière elle, journal
        des records rejoué à la clôture (identique à l'octet sur dix témoins)
      - [x] A3.6.d lancers de grenade (`grenade_events.go`) : par la marche, passe derrière elle (identique
        à l'octet)
      - [x] A3.6.e marche des records de bipède à pas de 1 bit après un rejet (`delta_biped_walk.go`) :
        ancrage bipède borné par la preuve (identique à l'octet) ; positions de véhicule par la marche,
        ancrage derrière elle (le décodage change, voir journal du 2026-11-13)
      - [x] A3.6.f visée seule du bipède (`offline_aim_only.go`) : par la marche, passe derrière elle (le
        décodage change, voir journal du 2026-11-13)
      - [x] A3.6.i (ajouté le 2026-11-13) motif de xuid du pont d'identité : un seul parcours par chunk
        pour tous les joueurs (`source.ChercherMotifs64`), identique à l'octet
      - [!] A3.6.g sonde de calibration et récupération d'équipement (`equipment_creation_width.go`,
        `equipment_recovery.go`)
      - [!] A3.6.h les sept sites de la voie hôte (objectifs, zones, états de mort, points de navigation,
        projectiles, état d'équipement, mise en page de i0), après mesure avec les options de l'hôte
      Preuve de chaque sous-item : document de rejeu identique sur les 20 films de `replay-equiv` (aucun
      refigeage attendu : même contenu, autre chemin), compteur de la récupération écrit avant et après,
      durée des dix témoins avant et après. Un écart de contenu arrête le sous-item et s'instruit.
      A3.5 est remplacé par : un test qui refuse toute boucle de balayage de payload delta hors de la
      liste datée des récupérations (le garde-rail sur `PremierMotif64` est déjà vrai et ne protège rien).
      Les items A3.6 recouvrent A4.2 (une seule marche par décodage) : A4.2 se statuera `[~]` sur eux.

Porte A3 : C2 et C4 tenus (A3.1 rejoué : zéro site de balayage delta atteint sur les témoins
HI_1_12_0 et HI_1_13_0 ; registre sans entrée à compte nul) ; `make replay-corpus-gate` et
`make replay-equiv` rejoués, toute différence expliquée ligne à ligne dans le journal ;
`go test ./internal/archlint/...` vert.

### A4 — Vitesse

- [x] A4.1 Rejouer A0.3 après A3 ; profil CPU d'un témoin lent ; classer les passes restantes.
- [x] A4.2 Une seule marche des paquets delta par décodage : les canaux qui relisent le film
      s'abonnent à la marche (modèle déjà en place pour les états de mouvement et le tir continu).
      Un item par canal encore en passe propre, d'après A4.1, tant que C3 n'est pas tenu.
- [x] A4.3 Baseline de `bench_baseline.txt` re-figée.

Porte A4 : C3 tenu sur les dix témoins (TSV de A0.3). Dès que C3 est tenu, A4.2 s'arrête. Si la
cible n'est pas atteinte une fois tous les canaux abonnés, l'item passe `[!]` avec la mesure et le
profil ; pas de travail au-delà.

### A5 — Lisibilité

- [x] A5.1 `libs/film/FORMAT.md` (anglais) : conteneur, paquets, les trois vues d'un paquet de
      tick, entrées d'entité, registre des composants, instantanés, chunk des temps forts ; un
      schéma par niveau ; ce qui est lu, ce qui est récupéré, ce qui n'est pas lu.
- [x] A5.2 `libs/film/GLOSSARY.md` : chaque terme interne (vue A, B, C, `ti`, `iNN`, genre, kind,
      statborg, pont d'identité, repli, récupération, image-clé, trame) avec son équivalent anglais
      et le nom que lui donne libfilmshell.
- [x] A5.3 README de la brique : section « How it reads a film » renvoyant à A5.1, et tableau des
      taux de fermeture par build mesurés en A2.
- [x] A5.4 Liste des fichiers `*_research_test.go` par campagne (préfixe de nom), avec pour chaque
      campagne si sa question est close. Liste remise au user ; la suppression est une décision
      séparée du user.
- [x] A5.5 Commentaires des fichiers touchés par A2 à A4 : contrat seulement, histoire et codes de
      lots déplacés dans la chronique (règle 17).

Porte A5 : les deux documents existent ; `make libs-test` vert ; la liste A5.4 est remise.

### A6 — Infos de match lues dans le film

- [x] A6.1 Lecteur de la variante (Bond du chunk 0) dans `libs/film/internal/grammar` : nom,
      description, score de victoire, durée, délai de réapparition, prolongation, dotation, options ;
      identifiants de la variante de carte et heure de début dans l'en-tête. Sur les 7 builds
      quand la lecture s'y applique sans cas particulier (D-4).
- [x] A6.2 Fait persistant « infos de match du film » dans les faits du film, exposé par la façade
      `decfilm`. Aucun consommateur branché (D-8).
- [x] A6.3 Commentaire de `ScoreTimeline.TargetScore` corrigé (le film porte la valeur) ; la source
      de la valeur ne change pas dans ce plan.
- [x] A6.4 Mesure écrite au journal : sur le corpus, part des films dont le score de victoire lu
      égale celui de `regulation.toml`, et liste des écarts.

Porte A6 : lecteur vert sur tous les films du corpus des builds où il s'applique ; A6.4 écrite ;
`make libs-test`, `cd apps/go-api && go test ./...`, `go test ./internal/archlint/...` verts.

### Clôture

- [x] Z.1 Revue adversariale du diff par un contexte frais (skill `adversarial-review`), une ronde.
- [~] Z.2 `make gate-push` ; CI de la branche verte (`gh run list --branch feat/film-marche-complete`).
- [~] Z.3 Notes de version (FR et EN).
- [x] Z.4 Demande au user : recuisson du parc (D-6), suppression des tests de recherche clos
      (A5.4), fusion dans la release ; lignes de backlog pour la suite (§5).
- [x] Z.5 Purge des caches : `powershell -File scripts/disk-hygiene.ps1 -Lot marche -Apply`.

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
- **Contenu voulu de la fiche du joueur (user, 2026-11-10)** : antagonistes à l'instant, tirs par arme, frags par arme, dégâts par arme, détail des médailles, et plus largement l'essentiel des statistiques tirées du film. État des données : antagonistes (couples tueur-victime), tirs par arme, frags par arme (source de dégât de la mort) et médailles horodatées (chunk des temps forts, absentes du rejeu aujourd'hui) sont déjà lus ; les dégâts par arme demandent la reconstruction des dégâts infligés (ci-dessus).
- Images-clés (D-5) ; travail propre aux cinq builds anciens (D-4).
- Cadence du rejeu et compression (chantier 30 images/s), planches d'icônes (chantier séparé).

## 6. Découvertes

- **D1 (A2.P, `libs/film/replay/grenades.go`, `birthForThrow`)** : sur un match Live Fire, un lancer de
  grenade sans auteur vivant avait une seule naissance de projectile candidate ; la lecture corrigée
  en lit une seconde dans un paquet désormais fermé, et le calcul s'abstient devant deux candidates :
  un lancer n'est plus publié (103 -> 102). Vraie perte de publication, à traiter dans `replay/` après
  la fusion du chantier 30 images/s.
- **D2 (A2.P)** : un paquet d'un témoin ancien (`084a804d`, 26/296) était fermé par recherche du début
  de liste avec un seul record ; la lecture corrigée y lit 32 records mais bute sur un invariant de
  l'écrivain. Indécidable sans témoin sur ce build ; aucun fait publié ne diffère.
- **D3 (A2.P, porte « aucune baisse sur les builds anciens »)** : `f144db25` (HI_1_8_0) perd 3 paquets
  fermés sur 20 210, tous fermés avant par recherche du début de liste (38 paquets de ce type dans le
  film, 35 restent fermés). Indécidable sans témoin ni exécutable de ce build ; accepté sans cas
  particulier par build (D-4).
- **D4 (A2.P)** : 1 382 lectures d'un index de région sans ligne (toujours l'index 1 sur des cartes à
  une seule région) relevées par un balayage ; la marche de la grammaire n'en fait aucune sur trois
  films récents contrôlés : elles viennent d'autres lecteurs (recherches), non localisés.
- **D5 (A2.P)** : `apps/go-api/cmd/mapquant-build/main.go`, déjà au-dessus de 500 lignes, a grossi de
  9 lignes : à ramener à sa taille d'avant dans le prochain item qui touche l'outil, ou à la clôture.
- **D6 (A0, A1)** : les portes de A0 et A1 ne rejouaient pas `archlint` ; deux paquets de recherche y
  étaient hors couche. Corrigé en A2.P ; `go test ./internal/archlint/...` entre dans chaque porte.
- **D7 (A2.T)** : l'hypothèse de A1 (confusion de génération) est réfutée pour les désaccords de type : 1 578
  paquets viennent d'une liaison posée par l'image-clé (type faux dès l'ouverture du morceau), 154 d'une
  création lue avec un autre type que le témoin (41 contre 36). À traiter avec A2.H.
- **D8 (A2.A)** : sur la branche « cible = objet » avec `flags & 0x10`, notre lecture et celle du témoin
  diffèrent de 5 bits ; la nôtre ferme des paquets que la sienne ne ferme pas ; un paquet ferme des deux
  façons : indécidable.
- **D9** : sous charge, `make libs-test` s'est arrêté une fois dans `grammar` sans échec affiché, et des
  tests de `mapcatalog` (concurrence) et `media` (ffprobe) ont échoué puis passé seuls. Non expliqué.
- **D10 (question du user, 11/11)** : entrées et sorties de joueurs et de bots. libfilmshell les tire de
  trois sources du film : table des joueurs du chunk 0, listes de joueurs (paquets de type 8) et de bots
  (type 12) avec l'instant où chacun est listé, et le composant `unit-control` qui nomme le joueur aux
  commandes de chaque corps. Un arrivant peut reprendre l'identifiant d'un partant (`holders.rs`). Chez
  nous `unit-control` est lu sans usage produit. À mesurer contre l'API avant d'en tirer un plan.
- **D11 (A3.6.h, 2026-11-14)** : les propriétés gérées (`ti=13` : états de zone de Bastion et de colline,
  jauge de retour du drapeau) ont deux formes de record dans les trames delta, à masque éparse et à masque
  dense. La passe (`managedPropertyWalk.scanPayload`, `matchWorldObjectRecord`) ne reconnaît que la forme
  éparse ; la marche lit les deux. Passer ces lectures derrière la marche change ce que le rejeu publie :
  sur les 20 films de l'équivalence, la jauge du drapeau change sur 7 films, en hausse ou en baisse
  (396 -> 1 903 lectures sur l'un, 3 945 -> 1 308 sur un autre), les lectures de zone sur 3 (jusqu'à
  15 377 -> 4 555) ; 10 films identiques. Laquelle des deux lectures est juste n'est pas établi : à
  confronter à une vérité (tenue des zones contre le score, remplissage de la jauge contre le temps de
  retour) avant toute bascule. Enjeu de vitesse : la passe pèse 12 à 20 % d'une cuisson sur les modes à
  zones. Changement annulé, rien de livré.

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
- 2026-11-10 : décisions. (1) Entités d'un morceau à l'autre (A2.H) : le user laisse le pilote trancher. Ordre retenu : d'abord garder la table d'un morceau au suivant tant qu'aucune suppression ne la retire, mesurer ; si le gain est insuffisant pour C1, lire la table d'entités de l'instantané (identifiant, génération, type par emplacement, comme le témoin), sans rouvrir le corps des images-clés. D-5 est amendée en ce sens. (2) Ghidra est ouvert : un agent relit en lecture seule les fonctions du jeu des fiches A1 et écrit `C:\t\marche\A2\desassemblage.md` ; le pilote contrôle cette note contre les bits et la fermeture (D-1).
- 2026-11-10 : relecture du jeu faite (lecture seule, API locale du greffon Ghidra, exécutable
  `HaloInfinite.exe` du 2026-04-08, adresses concordantes avec le dépôt) :
  `annexes_marche_complete/A2_desassemblage.md`. Fiches P, A, T, H, K confirmées ; fiche V : les 123
  genres de cet exécutable sont identiques à notre table, l'écart de HI_1_12_0 reste indécidable
  sans son exécutable. Sur la branche « cible = objet » du grappin, le jeu lit 15 bits comme notre
  code et non 13 comme le témoin.
- 2026-11-10, **A2.P : correction faite, item NON clos**. Catalogue de carte étendu à toutes les
  régions (101 cartes, 196 régions, région jouée identique avant et après sur les 101) ; position,
  volées, objets du monde et messages de genre 5 et 6 lus sur la ligne de l'index de région ; index
  sans ligne lu à zéro bit et compté (règle du jeu). Fermeture sous carte, corpus : HI_1_13_0
  97,60 % -> 99,42 % (119 films, +70 715 paquets) ; HI_1_12_0 inchangé à 97,70 % (A2.V) ; HI_1_11_0
  96,07 % -> 98,29 % ; HI_1_10_0 +29 paquets ; HI_1_9_0 +1 ; HI_1_8_0 -3 paquets sur un film (trois
  paquets que la recherche de début de liste fermait avant). Interlock : 88 à 93 % -> 99,6 à
  99,99 %. Désaccord « ti=40 composant 0 » contre le témoin : 12 087 -> 0.
  Restent rouges : (1) `libs/film/replay`, deux tests (la clé de cuisson doit hacher les régions ;
  fixtures de contrat à régénérer) ; (2) `replay-corpus-gate` : 8 témoins ok, 7 PERTE, 3 FAUX, non
  expliqués ; (3) `replay-equiv` : 20 films diffèrent (attendu, références non mises à jour) ;
  (4) `archlint` : les paquets `research/divergence` et `research/duree` de A0 et A1 ne sont dans
  aucune couche (défaut de notre branche, que les portes de A0 et A1 ne regardaient pas).
  `killsource.Rev` à monter (sa sortie change). Branche rebasée sur `origin/release/v8.1.2`.
- 2026-11-10, **A2.P clos**. Portes : `make libs-test` vert ; `cd apps/go-api && go test ./...` vert,
  `archlint` compris ; `libs/film/replay` vert (clé de cuisson hachant les régions, seul changement
  de code sous `replay/` ; fixtures, graines et goldens régénérés : seules les révisions de couche
  bougent) ; `killsource.Rev`, `grammar.Rev`, `profile.Rev` montées. `replay-corpus-gate` : 8 témoins
  identiques, 10 avec des baisses, toutes classées (`annexes_marche_complete/A2P_constats_gate.md`) :
  données fausses retirées par des paquets désormais fermés (changements d'arme attribués à des
  joueurs pas encore nés, postures de 24 à 740 secondes ramenées à leur vraie durée, pistes et
  détonations à des coordonnées impossibles), confirmé par le témoin sur les deux films qu'il sait
  lire ; une vraie perte de publication (D1) ; deux indécidables (D2, D3). `replay-equiv` : 20 films
  diffèrent, familles expliquées par les mêmes causes ; ses références seront mises à jour à la
  porte A2, pas item par item. Fermeture : voir `A2P_fermeture_par_build.txt`.
- 2026-11-10 : coordination. La session qui pilote la v8.1.2 renomme les chemins de module des briques (`libs/*/go.mod` et environ 580 fichiers d'imports) sur `feat/812-renommage`, PR vers la release. Consigne : ne réécrire aucun import de brique ici ; après sa fusion, rebaser et garder `github.com/JGtm/halo-*` ; les paquets neufs de cette branche (`research/divergence`, `research/duree`, fichiers neufs de `grammar`) portent l'ancien chemin et sont à réécrire au rebase. Rebase à faire entre deux items, jamais pendant qu'un agent travaille dans le worktree.
- 2026-11-10 : calendrier de la v8.1.2 (source : sa session pilote). Gel de `libs/` au plus tôt le 2026-11-11 (après la fusion du renommage, celle du lot « seuils en millisecondes », puis les tags posés par le user). Entre le gel et la bascule qui supprime `libs/` et `go.work`, le décodeur n'est modifiable que pour un correctif bloquant. Après la bascule, tout changement du décodeur passe par le dépôt dédié (clone local, espace de travail `GOWORK`, tag par le user, montée du `require` dans LevelUp). Question posée au user : entrée partielle (étape A2 seule) dans la release avant le gel, ou suite du chantier dans le dépôt dédié.
- 2026-11-10 : le user reporte à demain 11/11 la décision d'une entrée partielle dans la v8.1.2 avant le gel (« ok pour ce soir » : préparer la branche comme pour une fusion). Périmètre de cette entrée éventuelle : A2.P, A2.T, A2.A ; A2.H et A2.V restent après. À remettre demain : ce qui est prêt, ce que le user verra changer au rejeu, ce qui reste imparfait.
- 2026-11-10 : **décision du user : OUI**, les corrections de lecture entrent dans la v8.1.2 avant le gel de `libs/` (et, relayé par la session pilote : la recuisson des rejeux n'est pas un frein). Périmètre : A2.P, A2.T, A2.A, chacune seulement si verte et ses écarts classés. Enchaînement : fin de A2.T et A2.A, rebase sur la release après le renommage des modules, mise à jour des références de `replay-equiv`, revue adversariale, `make gate-push`, PR vers `release/v8.1.2`, ligne de notes de version. A2.H, A2.V et les étapes A3 à A6 se poursuivent après, dans le dépôt dédié du décodeur une fois la bascule faite.
- 2026-11-11, **A2.T et A2.A clos**. A2.T (`40dc36e9bd`) : le type d'une entité répond à son
  identifiant complet, comme la garde du jeu ; aucun effet mesuré sur la fermeture (38 films
  identiques), 12 + 7 paquets non fermés s'arrêtent une étape plus tôt au lieu de lire sous un type
  périmé ; gate identique à la fin de A2.P. A2.A (`c95cd8725b`) : branche du grappin lue selon le jeu ;
  aucun film ne baisse ; gains sur les six builds (38 films : HI_1_13_0 +899 paquets, HI_1_12_0 +118,
  HI_1_11_0 +46, HI_1_10_0 +159, HI_1_9_0 +30, HI_1_8_0 +126) ; désaccords sur ce composant 599 -> 3 ;
  gate : un témoin de plus en baisse (`4f77afc1`), classé données fausses retirées, aucune vraie
  perte trouvée (`annexes_marche_complete/A2A_constats.md`). Portes vertes (libs-test au second
  passage, voir D9). `grammar.Rev` = `grammar-2026-11-10.3`.
- 2026-11-11 : décisions du user. (1) Version visée : **v8.2.0** (un hotfix a pris le numéro précédent) ; branche cible de la PR à confirmer avec la session pilote. (2) Les découvertes et enrichissements relevés en route (D7, D10, nouveautés de libfilmshell du 10/11 au soir, fiche du joueur, dégâts, téléporteurs) forment une **phase C**, traitée après les phases en cours, sauf s'il est judicieux d'en avancer une.
- 2026-11-11 : le user confirme que la PR vise `release/v8.1.2` telle quelle ; le numéro 8.2.0 se règle au tag, à la fusion vers `main`.
- 2026-11-11 : la branche d'intégration s'appelle désormais `release/v8.2.0` (annonce de la session pilote ; vérifiée sur le dépôt distant, 91727b0c30). La PR et les rebases de ce chantier la visent ; elle sera rebasée sur `main` après la fusion du correctif 8.1.2. Remplace la ligne précédente (« la PR vise `release/v8.1.2` »).
- 2026-11-13, **mise à jour des statuts sur pièces** (historique du dépôt du décodeur). A2.V et A2.H :
  `e8a2ba1` (v1.1.0). A2.z : révision montée, section de l'ADR 0037 et ligne de `NOTICE.md` présentes.
  A3.1 : mesure du 2026-11-12 (`constats.md` et `sites_balayage.md` de la mesure A3). A5.1 à A5.3 : `d6d1d4c`
  (v1.2.1) ; A5.4 : liste remise. A6.1 et A6.2 : `d9d3a56` (v1.2.0), avancés avant A3 avec la phase C sur
  décision du user du 2026-11-11. A6.3 et A6.4 non vérifiés ce jour : restent ouverts. Décodeur publié
  en v1.3.4 (schéma 113) ; LevelUp monté dessus, PR #203 fusionnée dans `release/v8.2.0` (87c6038c0b).
- 2026-11-13, **A3 amendé (A3.6)**. Constat A3.1 : le gros du temps n'est pas dans les replis mais dans
  des balayages qui tournent en lecture principale. Le user valide la forme cible (une seule lecture,
  les canaux lisent son résultat), dans l'esprit des dépôts de Dend (libfilmshell) et de nuzzles
  (`halo_api`, branche `simbleau/theater-experiments`, lue ce jour : flux d'observations typées puis petit
  moteur d'état ; rien à reprendre côté format). Les optionnels et replis résiduels se retirent au fil
  de A3.3, sur compte nul mesuré, pas par campagne à part.
- 2026-11-13, **A3.6.a et A3.6.b faits** (dépôt du décodeur, branche `perf/balayages-derriere-la-marche`,
  `5d08723`, `e06300c`, `170ed0f`, non fusionnée). Les deux passes fusionnaient déjà derrière la marche par
  `rendParLAncrage` : une position au premier bit prouvé par la fermeture de la trame, ou au-delà, n'était
  jamais rendue. Elles ne la jugent plus (`FilmContext.finDuBalayage`). Une demande de pistes sans archétype
  garde les payloads entiers. Preuve : dix témoins cuits avant et après, documents identiques à l'octet ;
  compteurs de replis identiques par construction. `grammar.Rev` inchangée, empreinte refigée (sortie
  inchangeable). `replay-equiv` sur 20 films : à jouer à la porte du groupe, pas encore fait.
- 2026-11-13, **profil CPU d'un témoin** (`29a3ef4c`, 9,7 s, après A3.6.a) : la marche des trames 30 % ;
  les images-clés 16 % ; puis les balayages : ancrage des records de bipède à pas de 1 bit 17 % (dont
  `ScanBipedRecords` pour le nuage des véhicules 9 %), recherche du motif de xuid du pont d'identité
  (`ChercherMotif64`) 11 %, pistes du monde 8 % (avant A3.6.b), visée seule 8 %, créations de bipède 8 %,
  mots par record d'image-clé 5 %, lancers de grenade 2,5 %. Les balayages pèsent environ 65 % du temps.
  Ordre révisé des sous-items restants, au coût mesuré : e (ancrage bipède), pont d'identité (à ajouter :
  hors paquet delta, donc hors C2, mais second coût), f (visée seule), c (créations de bipède), puis le
  reste. **Constat** : c, e et f ne sont pas des récupérations derrière la marche mais des lectures
  PRINCIPALES ; les créations de bipède nourrissent même les générations vivantes dont la marche se
  sert. Les retirer demande de faire produire ces lectures par la marche elle-même : un sous-item de
  conception chacun, pas une borne de boucle.
- 2026-11-13, **A3.6.e, A3.6.f et A3.6.i faits ; A3.6.c accéléré sans changer de forme** (branche
  `perf/balayages-derriere-la-marche`, commits 964ea22, 4bc23c8, puis visée, véhicules, en-tête NEW,
  b3bc36e ; poussée, sans PR). Relecture du décodeur de Dend à jour (fusion du 12/11) : il garde UN
  balayage (positions et visée seule, une passe mémorisée, pré-tri sur le premier octet, sur un second
  fil) et lit tout le reste par sa marche ; identités par la table du chunk 0 et les paquets 8 et 12.
  **Mesure qui change la lecture du chantier** : en cuisson réelle (contexte calibré), la marche ferme
  presque toutes les trames (26 438 sur 26 439 sur `03b7bbb6`, 40 652 sur 41 019 au pire témoin) ; un
  contexte nu de test n'en ferme que 9 % (fermeture refusée en vue C) : ne jamais mesurer la part de la
  marche hors du contexte de la cuisson. Conséquence : toute passe bornée par la preuve devient presque
  gratuite. Faits : (e) l'ancrage bipède s'arrête où la fermeture prouve, documents identiques à l'octet ;
  (i) les motifs de xuid se cherchent en un parcours par chunk, identiques à l'octet ; (f) la visée seule
  vient des records de la marche, la passe ne rend plus que ce que la marche ne prouve pas : LE DÉCODAGE
  CHANGE, 0 à 259 lectures de moins par film, toutes hors de tout record lu dans une trame fermée (31 sur
  `29a3ef4c`, seize à plus de 90 degrés d'élévation ; une visée à 167,6 degrés disparaît d'un trajet de
  `8b1f15cb`) ; (e, véhicules) positions par la marche, ancrage derrière elle : sur le seul témoin à
  véhicules, 195 échantillons hors de l'emprise de la carte disparaissent, 4 s'ajoutent, un repli sort de
  la liste ; neuf témoins identiques ; (c) l'en-tête NEW se teste en une lecture par bit, identique.
  Durée des dix témoins : médiane 7,4 s -> 3,5 s, pire 31,8 s -> 10,1 s (C3 visait 5 s et 30 s).
  **Reste dû avant fusion** : `grammar.Rev` à monter avec sa chronique (f et véhicules changent le
  décodage ; l'empreinte de la couche est rouge d'ici là), `replay-equiv` 20 films avec instruction des
  écarts, gate du corpus, revue par contexte neuf. Ordre convenu avec la session levelup-7c : elle
  fusionne d'abord son lot (`replay/` seul, schéma 114), cette branche se recale ensuite.
  **A3.6.c reste ouvert** : faire venir les créations de bipède de la marche demande de différer le
  filtre des générations du canal des lectures bipèdes (le masque des générations vivantes est relevé sur
  tout le film) ; non fait. Restent aussi d, g, h. Profil après ces lots (4,3 s sur `29a3ef4c`) :
  images-clés environ 30 % (marchées en plusieurs distributions), marche des trames 11 %, créations de
  bipède 9 %, lancers de grenade 5,5 %.
- 2026-11-13 (suite), **A3.6.c fait** (commit 72c29c8, fusionné dans la branche). Angle retenu avec le
  user : la création est un record NEW que la marche lit ; le filtre des générations vivantes, né pour
  protéger les passes à l'aveugle, ne se joue plus pendant la marche. Le canal porte ses records bipèdes
  à un journal et le rejoue à la clôture sous les générations du film : mêmes gardes, même ordre, donc
  même sortie. Dix témoins identiques à l'octet ; médiane 3,5 s -> environ 2,3 s, pire 10,1 s -> 6,8 s
  (mesure sous charge). Les compteurs de la passe des créations changent de sens (ancres = records lus).
  **`replay-equiv` sur 20 films, branche avant ce lot** : 10 identiques, 10 différents sur la seule
  étape des véhicules (et l'artefact). Instruction sur les dix (documents avant et après comparés) :
  nombre de véhicules inchangé ; 1 à 302 échantillons ajoutés par film, à 1 cm de médiane du trajet déjà
  connu (p90 8 cm au plus, maximum 42 cm) ; 0 ou 1 retiré ; échantillons hors de l'emprise 9 -> 0 et
  3 -> 0 ; lectures de visée brutes en baisse de 38 à 281. Aucune régression relevée. À refiger avec la
  montée de `grammar.Rev`. Restent : d (grenades), g, h, puis les images-clés lues une seule fois.
- 2026-11-13 (soir), **A3.6.d fait ; table d'entités d'image-clé lue une fois ; revue par contexte neuf ;
  équivalence rejouée** (commits 76fc6dd, table, fab1aa4). Lancers de grenade : record NEW de projectile lu
  par la marche, passe derrière elle, dix témoins identiques à l'octet. Table d'entités d'une image-clé :
  lue une fois par contexte au lieu de deux, identique. Revue (un relecteur, une ronde) : 0 P0, 1 P1
  corrigé (seconde marche à la clôture sur un film sans bande bipède), 2 P2 dont 1 corrigé (marche
  demandée pour rien sans projectile au registre) et 1 laissé (motif d'amorce aux 19 premiers bits d'un
  paquet fermé : comportement de l'ancienne passe), 14 conditions vérifiées. `replay-equiv` 20 films avec
  tous les lots : 10 identiques, 10 différents, mêmes empreintes qu'avant les lots c, d et table : ces
  trois lots ne changent rien sur les 20 films.
  **Mesure qui corrige l'annonce de durée** : les durées de l'instrument (2 à 10 s) sont celles du
  décodeur seul, options minimales. La cuisson réelle de LevelUp fait plus : sur dix films de
  l'équivalence, 18 à 116 s avant, 10 à 68 s après (gain de 1,2 à 2). Profil d'une cuisson réelle
  (`1c4c63c2`, BTB) : source de dégât 28 % (sa propre marche des trames 15 %, sonde relâchée 11 %),
  marche des trames de la cuisson 18 %, monde 12 %, propriétés ti=13 5,5 %, statborg 4,6 %, changements
  d'équipement 3 % ; l'observateur d'étapes de l'outil d'équivalence (23 %) n'est pas de la production.
  **Conséquence pour le plan** : le gros de ce qui reste est la voie hôte (A3.6.h), et d'abord la source de
  dégât, qui marche le film une seconde fois. A3.6.g (sonde de calibration d'équipement 1,5 %,
  récupération d'équipement non visible au profil) : `[!]` non traité, coût mesuré négligeable, à revoir avec
  le user. **Découverte hors périmètre** : dans la phase des images-clés, trois lecteurs font chacun leur
  recherche (état complet du bipède, dotations de naissance, équipes) ; la relecture des images-clés
  elle-même ne pèse que 6 % ; chantier distinct, non ouvert. Le gate du corpus se joue avec la montée de
  LevelUp (il compare deux versions du décodeur par le go.mod).
- 2026-11-13 (soir, suite), **voie hôte, premier site : source de dégât** (commit b359cdb). L'appariement
  exact de la sonde relâchée parcourait tout le kill-feed deux fois par candidat : couples indexés par
  (victime, tueur), sorties identiques sur trois films de l'équivalence (mêmes empreintes), sonde relâchée
  11 % -> 3 % du temps d'une cuisson. A3.6.g statué `[!]` (justification à l'entrée précédente).
  **Constat pour A4.2 (une seule marche par décodage)** : la source de dégât ouvre son propre contexte de
  film et marche les trames une seconde fois (`killsource.marcher` : calibration puis
  `grammar.LireLaMarcheDeKillsource`, 17 % du temps d'une cuisson). Les deux marches tournent sous le MÊME
  profil : la cuisson reçoit le profil calibré de la source de dégât (`Result.ProfilCalibre` ->
  `opt.ProfilDeBalayage`) et pose le même découpage MPP. Forme à instruire : une seule distribution qui
  porte les canaux des deux (morts gardées, kills, canaux de la cuisson), rangée dans le contexte, et le
  contexte passé de la source de dégât à la cuisson. Cela demande un champ de plus à la façade et un
  passage de relais dans `replaybuild` côté LevelUp (seul changement de code de l'app de tout le lot) ;
  preuve attendue : documents et lignes de source de dégât identiques sur les 20 films. Non commencé.
  Les mesures de durée au mur de ce soir sont inexploitables (machine chargée par d'autres sessions) :
  seuls les profils CPU et les empreintes comptent.
- 2026-11-13 (nuit), **A4.2, lecture unique partagée entre la source de dégât et la cuisson : instruite, non lancée**. Le user a donné son accord de principe (retouche de LevelUp admise ; tout changement de sortie à écrire dans le message de release du décodeur). Sur pièces, le partage est plus imbriqué qu'un passage de relais : chaque contexte de film tient son rapport de replis, ses diagnostics et son observateur, tous publiés (les replis de la cuisson sont dans le document, les diagnostics de la source de dégât dans son résultat) ; la marche et ses canaux y écrivent pendant qu'ils tournent, donc une marche commune doit rendre à chacun ce qui est à lui. L'ordre est contraint : la calibration de la source de dégât précède la marche de la cuisson, qui hérite de son profil. Gain attendu : 15 à 17 % d'une cuisson. Preuve disponible : `replay-equiv` (étapes `killsource` et document) sur 20 films. Recommandation faite au user : publier d'abord le lot actuel, puis traiter ce partage comme un lot à part.
- 2026-11-13 (nuit), **décodeur v1.3.6 publié** (PR 16, main 5d63c67, tag et release GitHub avec les changements de sortie) : `grammar-2026-11-13`, schéma 115, recalé sur v1.3.5 (schéma 114 de la session levelup-7c). `replay-equiv` contre les références 114 : dix films sur la seule ligne de l'artefact, dix sur véhicules et artefact, rien d'autre ; refigé, passe de contrôle 20 sur 20. CI verte (un rouge de forme corrigé : ordre des imports de `grenade_events.go`). **Montée de LevelUp : portée par levelup-7c** dans son lot (v1.3.4 -> v1.3.6, fixtures 115, équivalence, gate du corpus), à sa demande : son lot porte du code propre à ses corrections ; écarts attendus et chemins des sorties lui ont été transmis ; la recuisson unique suit sa fusion et la revue du pilote. Reste ouvert, à instruire si le gate le relève : `repli_tourelle_porteur_voisin_de_slot` 1 -> 12 sur `e5adf7b2`. A3.6 : a, b, c, d, e, f, i `[x]` ; g `[!]` ; h partiel (source de dégât : appariement indexé fait, marche partagée = A4.2, lot à part). Worktrees et cache du lot retirés.
- 2026-11-13 (nuit), **correction : `repli_tourelle_porteur_voisin_de_slot` ne bouge pas** (12 avant et après sur `e5adf7b2`, identique sur les dix films à véhicules). Le « 1 -> 12 » venait de ma comparaison de `coverage.fallbacks` rang par rang : l'entrée `repli_position_hors_emprise_ecartee` sortant de la liste, les suivantes remontent d'un rang. Relu par nom dans les deux documents. Transmis à levelup-7c ; la ligne du registre des reports posée à tort sur release/v8.2.0 est à retirer (en attente de sa préférence : mon commit de doc ou son lot).
- 2026-11-13 (nuit), **A4.2 conçu sur pièces (inventaire en lecture seule par un agent, deux points clés revérifiés ici)**. Archive des films effacée à 18:30 (incident, restauration par le user) : aucune exécution possible, conception seule.
  **Constats** : (1) les deux contextes sont paramétrés à l'identique en pratique quand `Decode` réussit (même film chargé une fois, même entrée de carte, découpage d'i0 imposé nil des deux côtés, profil calibré transmis, même `ResolutionMPP`) ; ils diffèrent quand `Decode` échoue ou qu'il n'y a pas de kill-feed (la cuisson tourne alors sous `ProfilDeDepart`). (2) Les deux canaux de la source de dégât (`canalDesMorts` à `garder=true`, `canalDesKills`) n'écrivent RIEN dans le contexte (vérifié : aucun `NoterReplis`, aucun diagnostic, aucune mémoire) ; le rapport de replis du contexte de la source de dégât n'est lu par personne ; ses diagnostics ne sont que journalisés. (3) Sur le chemin de calibration, un seul `NoterReplis` existe, dans la pose de la carte, que seule la cuisson appelle. (4) En phase des trames, les intérêts déclarés par un canal ne changent pas ce que les autres voient. (5) Le canal des morts des deux étapes est le même type ; à `garder=true` il sert les deux usages.
  **Forme retenue** : LevelUp ouvre UN contexte (`decfilm.NewFilmContextForMap`, déjà à la façade) et le passe aux deux étapes par un champ de plus dans deux structures existantes (`killsource.Options`, `replay.Options`) : aucun symbole neuf à la façade, rien dans `Result` (haché et persisté). Quand le contexte lui est donné, la source de dégât distribue en UNE marche ses deux canaux ET les canaux de la cuisson, et range le résultat de la cuisson dans la mémoire du contexte avec le profil sous lequel il a été lu ; `ScanMarcheDesTramesAvec` rend cette mémoire si le profil courant est le même, sinon marche comme aujourd'hui. La condition des véhicules, le `NoterReplis` des localisations à largeur libre et la pose de la carte restent du côté de la cuisson, une fois. Pas de partage si `Decode` échoue, s'il n'y a pas de kill-feed, ou si un découpage est forcé : la cuisson garde sa marche. Les autres appelants de la source de dégât (collecteur, outil en ligne de commande) ne changent pas.
  **Étapes** : 1. décodeur, `grammar` : mémoire du résultat de marche gardée par le profil, marche commune ; 2. décodeur, `killsource` et `replay` : le champ de contexte et son emploi ; 3. test sur les mini-bobines du dépôt : `Result` et document identiques entre voie partagée et voie séparée (jouable sans l'archive) ; 4. LevelUp, `replaybuild` : un contexte pour les deux étapes ; 5. preuve : `replay-equiv` 20 films, toutes les étapes identiques, `killsource` comprise, et pic mémoire relevé (les morts gardées et le journal des bipèdes vivent en même temps) ; 6. empreintes de couche refigées à révision CONSTANTE (aucune sortie ne change : pas de recuisson) ; revue par contexte neuf ; publication avec « aucun changement de sortie » au message.
  **Ce qui change hors sortie** : des lignes de journal passent d'une étape à l'autre (diagnostics du contexte), le temps de la marche est compté dans la phase de la source de dégât. **Non établi** : égalité des profils mesurée sur film réel (établie par lecture seulement) ; pic mémoire ; si `ObjectDeathStats.Config` est sérialisé dans les faits. Gain attendu : 11 à 17 % d'une cuisson.
- 2026-11-13 (nuit), **précision du user sur les phases** : la **phase B est celle des RENDUS** (ce que le §5 nomme « chantier des rendus » et « fiche du joueur du rejeu » : tableau des scores vivant, MVP dans le temps, exploitation des infos de match, fiche du joueur, maquettes d'abord), moins prioritaire que le décodeur lui-même pour ne pas se disperser, et plus conséquente que ce que j'en avais retenu. Elle n'est PAS l'étude « B0 » du 11/11 (arrivées, départs et rattachement des vies chez libfilmshell), dont le nom de dossier m'a induit en erreur : cette étude relève du fond du décodeur. Phases : A fond du décodeur (reste A4.2, la marche partagée, voulue par le user bien que C3 soit tenu) ; B rendus, à planifier à part avec ses maquettes ; C enrichissements du décodeur relevés en route. Ordre entre B et C : à confirmer par le user.

## 10. Ordre de la suite (révisé le 2026-11-13, à la demande du user : « tu peux revoir l'ordre, tant que tu ne manques rien »)

Inventaire fait sur le §4 (items non statués), le §5, le §6 (D1 à D10), le journal, le registre des reports et la note sur la fusion du 13/11 de libfilmshell. Chaque item y figure une fois. Rien n'est planifié en détail ici : chaque bloc aura son plan.

**1. Fin de la phase A (fond du décodeur)**
- A4.2 une seule marche pour la source de dégât et la cuisson (conçu, attend le retour des films).
- A3.5 remplacé : garde-rail qui refuse toute boucle de balayage de payload delta hors liste datée.
- A3.2 à A3.4 : retrait des récupérations et replis à compte nul, re-mesurés sur v1.3.6.
- A4.1 et A4.3 : profil et baseline de bancs re-figés ; A5.5 ; A6.3 et A6.4 à vérifier sur pièces ; clôture (Z.1 à Z.5).

**2. Phase C, premier temps : les données dont les rendus dépendent**
- Qui est là, qui arrive, qui part, à qui est chaque vie, lus dans le film (D10 : table du chunk 0, paquets 8 et 12, `unit-control`) ; remplace la recherche de motif de xuid.
- Tableau des scores complet (autres emplacements du statborg, plus de 8 joueurs).
- Tirs et touches, par arme et dans le temps (touches par message de dégât ; nos deux méthodes mesurées ensemble).
- Dégâts infligés et subis en points (total réel par vie, parts du message de mort, estimation par coup pour répartir).
- Médailles horodatées dans le rejeu ; détail du score par action (barème `personal_score_awards`).
- Début de l'horloge de match, compte à rebours de réapparition, tirs alliés (fusion du 13/11 de libfilmshell).
- Propriétés gérées (D11, ajouté le 2026-11-14) : zones de Bastion et de colline, jauge de retour du drapeau. La marche lit deux formes de record, la passe une seule ; confronter les deux lectures à une vérité, puis passer la lecture derrière la marche (12 à 20 % d'une cuisson sur les modes à zones).

**3. Phase B : les rendus (maquettes d'abord, plan à part)**
- Le plan des évolutions de la page de rejeu (`PLAN_REJEU_MAQUETTES_2026-11-10.md`, sept lots : fil filtrable, clic sur une élimination, bouton Vue, fiche de joueur en panneau, graphiques, Mainmise, clôture), tenu par la session du rejeu.
- Tableau des scores vivant dans la vue cockpit ; MVP dans le temps (Timeseries, Sessions, Escouade) ; exploitation des infos de match, dont le score à atteindre.
- Fiche du joueur complète : antagonistes à l'instant, tirs, frags et dégâts par arme, détail des médailles, compteurs à l'instant.
- Ce que le user y range de plus est à reposer avec lui : cette liste est ce qui est écrit, pas forcément tout ce qu'il a en tête.

**4. Phase C, second temps : enrichissements sans rendu en attente**
- Véhicules : santé dans le temps, orientation bâtie comme le jeu, arme et cause d'une destruction (`causeKey` au contrat, vide).
- Portages d'objectif en plages ; munitions, charge et chaleur par arme.
- Entrées de déplacement, santé et bouclier à chaque point, dégâts par coup, voix, messages d'écran, texte des événements ; sprint (sa lecture contredit la nôtre : à confronter).
- Téléporteurs, canons à homme et dispositifs fixes (type 43) lus dans le film.
- Lecture sur deux fils.

**5. Fond différé (décision du user à chaque fois)**
- Images-clés : D-5, et les trois lecteurs qui y font chacun leur recherche (environ 30 % du décodage seul).
- Builds anciens : D-4, genres 12, 14 et 106 sur HI_1_9_0 et HI_1_10_0, création par porte avant HI_1_11_0, D2, D3.
- Sonde de calibration et récupération d'équipement derrière la marche (A3.6.g).
- Découvertes ouvertes : D1 (un lancer de grenade perdu), D4 (index de région sans ligne), D5 (taille de `mapquant-build`), D7 (liaison de type posée par l'image-clé), D8 (5 bits d'écart sur une branche), D9 (aléas de tests sous charge).

**Hors de cette suite** : cadence du rejeu et compression (portées par d'autres sessions), planches d'icônes.
- 2026-11-13 (nuit), **A4.2, côté décodeur : fait sur la branche `perf/marche-partagee`** (commit bc572bf, poussée, sans PR), sur accord du user (« continue », films en cours de restauration). `grammar.LireLaMarcheCommune` distribue en une marche les deux canaux de la source de dégât et ceux de la cuisson ; le contexte garde la marche de la cuisson avec le profil sous lequel elle a tourné ; `ScanMarcheDesTramesAvec` la rend une fois si le profil courant est le même (`reflect.DeepEqual`), sinon marche. Champs neufs : `killsource.Options.Contexte`, `replay.Options.Contexte` ; aucun symbole neuf à la façade. Test `TestLaMarcheCommuneRendCeQueRendentLesDeuxMarches` (mini-bobine, trois jeux de lectures) : résultat de la source de dégât identique au décodage seul ; le contexte partagé GARDE bien une marche sous le profil de la cuisson (profil calibré puis carte et découpage : égalité des profils constatée, plus seulement lue) ; marche gardée et rapport de replis identiques à ceux d'un contexte neuf ; rendue une seule fois. Suite du dépôt verte hors fixtures de contrat (côté hôte) ; empreintes `grammar` et `killsource` refigées à révision constante.
  **Reste** : (4) LevelUp, `replaybuild` : `decodeKillSource` ouvre le contexte (`decfilm.NewFilmContextForMap(film, &entry, nil)`), le donne à `decfilm.Decode` et, si le décodage réussit, aux options de la cuisson ; à écrire APRÈS la fusion du lot de levelup-7c (il retouche `replaybuild`). (5) preuve `replay-equiv` 20 films et pic mémoire : attend la fin de la restauration (à 1 010 films le 13/11 au soir, trois des vingt films du corpus et `data\film_identites\xuids_originaux.txt` manquaient encore) ; pas de jonction vers `data` dans un worktree (incident du 13/11) : passer la racine des données par l'option de l'outil. (6) revue, PR, tag, release « aucun changement de sortie ».
- 2026-11-13 (nuit), **A4.2 prouvé sur les 20 films**. Archive restaurée (1 614 films, identités régénérées par levelup-7c). Retouche de LevelUp écrite dans un worktree jetable sur `release/v8.2.0` (`decodeKillSourceEtContexte` dans `replaybuild/kills.go`, champ `contexte` dans `filmfacts_cuisson.go` et `replaybuild.go`, `Contexte` dans `options.go` ; diff gardé dans `C:\t\marche\A3\levelup_partage.patch`, à reposer après la fusion du lot de levelup-7c). `replay-equiv` avec la marche partagée, références du schéma 115 : **20 identiques sur 20**, étape `killsource` comprise ; la marche gardée est bien rendue à la cuisson (constaté par trace). Durées côte à côte sur les 20 films (machine chargée) : gros matchs 1 min 24 -> 1 min 10, 1 min 32 -> 1 min 10, 43 s -> 36 s, 49 s -> 42 s ; petits matchs à la seconde près. Pic mémoire : +0,1 à +0,25 Gio, maximum 2,54 Gio (plafond souple 3). Méthode sans jonction : le film vient de l'archive du dépôt principal par une variable lue par l'enfant de `replay-equiv` (retouche locale non versionnée ; une option `-archive` de l'outil serait à proposer). Piège rencontré : lancé sans `GOWORK`, l'outil compare aux références du module du `go.mod` (ancienne version) : 20 écarts de pure référence.
- 2026-11-13 (nuit), **A4.2 : revue et preuve finale**. Revue par contexte neuf, une ronde : 0 P0, 0 P1, 4 P2, tous corrigés (commit 109b02b du décodeur ; retouche de LevelUp : `decodeKillSource` rend le contexte, plus de fonction doublon). `replay-equiv` sur le code corrigé : 20 identiques sur 20 ; pic mémoire maximal 2,55 Gio. PR ouverte sur le dépôt du décodeur (CI en cours). Retouche de LevelUp gardée dans `C:\t\marche\A3\levelup_partage.patch` (6 fichiers, 27 lignes ajoutées), à reposer après la fusion du lot de levelup-7c ; elle exigera la version du décodeur qui porte ce lot. Fusion et tag du décodeur : question posée au user (j'avais recommandé d'attendre la recuisson ; ce lot ne change aucune sortie).
- 2026-11-13 (nuit), **PR 17 du dépôt du décodeur (contributeur externe, « Shot direction from aim », schéma 116) : lue et mesurée, non fusionnable en l'état**. Elle remplace la direction publiée d'un tir (champ de visée du record de tir) par la dernière visée i21 du tireur (records de position et visée seule), garde le champ du record en repli nommé et ajoute `shots[].p` et `coverage.shots.aim`. Code soigné et dans les conventions du dépôt. Constats sur pièces : (1) sa justification (« tenir bat interpoler, mesurée, chronique du schéma 110, écart médian sous un degré ») n'a aucun appui : la chronique 110 parle des occupants tués, et la PR n'apporte aucune mesure sur film ; (2) mesure faite ici sur deux témoins (`03b7bbb6`, `221dd082`), 4 246 tirs portant les DEUX visées : écart médian 56 degrés, 28 % à un degré ou moins, 36 % sous 15 degrés, le reste étalé uniformément jusqu'à 180 ; 96 % des lectures tenues sont du même instant que le tir : sur deux tirs sur trois les deux sources ne désignent pas la même direction, et laquelle est juste n'est pas établi ; (3) avec les données de l'hôte, `TestGoldenAssembly` et `TestGoldenBuildsAssembly` sont rouges (les « 0 échec » de la PR viennent des 461 tests sautés sans ces données) ; références d'équivalence non refigées ; (4) la clé est le slot seul, sans génération, avec une tenue de 5 s ; (5) un repli neuf fait un FAUX au banc de vérité (règle R-1) ; (6) côté LevelUp : deux champs neufs à recopier dans le document servi et le contrat, fixtures 116 ; (7) aucune CI n'a tourné (fork). À instruire avant toute décision : confronter les deux visées à une vérité (direction tireur -> victime sur les tirs qui touchent ou tuent). Le témoin `090139ed` manque à l'archive restaurée (signalé).
- 2026-11-13 (nuit), **PR 17 tranchée par la mesure : elle a raison**. 13 films, 36 003 tirs ; sur les 22 327 qui portent les deux directions, accord à 5° pour 24,8 % seulement. Vérité : cap tireur -> ennemi le plus proche en angle à l'image du tir, avec un témoin (visée tenue + 90°) pour mesurer le hasard du « plus proche ». Visée tenue : médiane 1,3°, 67 % à 3°, 85 % à 10° ; champ du record : médiane 37,9°, 8,6 % à 3° (indistinguable du témoin, 48,3°) ; sept conventions d'angle essayées, aucune ne sauve le record. Le rejeu actuel publie une direction fausse sur la plupart des tirs. Goldens d'assemblage refigés localement sur la branche de la PR : seules les lignes attendues changent (schéma 116, caps lisibles 408 -> 504 sur un témoin, un repli neuf à 1). Restes : goldens et chronique (proposés au contributeur), références d'équivalence et fixtures 116, miroir LevelUp de `shots[].p` et `coverage.shots.aim`, repli neuf au banc (R-1). Brouillon de réponse : `C:\t\marche\A3\pr17_commentaire.md`. Mon interprétation non confirmée : un passage « à republier » (assemblage seul) se fond dans la recuisson unique s'il arrive avant elle.
- 2026-11-13 (nuit), **décodeur v1.3.7 publié** (main 35bd870, tag et release GitHub avec les changements de sortie), sur go du user : #18 (marche partagée, aucune sortie ne change) et #17 (direction des tirs par la visée du tireur, schéma 116, contributeur externe). Pour #17 : réponse postée (mesure, compléments), un commit poussé sur la branche du contributeur (goldens d'assemblage et références d'équivalence refigés, chronique portant la mesure), workflows du fork approuvés après lecture du code, CI verte, fusion. Suite de la vérification sur `main` fusionné, données de l'hôte : verte hors fixtures de contrat et fichiers `research/`. levelup-7c prévenue : prendre v1.3.7 dans son lot si possible (miroir de `shots[].p` et `coverage.shots.aim`, fixtures 116, repli neuf en FAUX R-1 au banc) ; sinon montée v1.3.7 dans mon lot après le sien. A4.2 : `[x]` côté décodeur ; la retouche de `replaybuild` reste à poser après la fusion de levelup-7c (`C:\t\marche\A3\levelup_partage.patch`).
- 2026-11-14, **décision du user** : le repli neuf `repli_visee_du_tir_lue_dans_le_record` (PR 17) est accepté ; et « l'idéal serait d'ajuster nos tests, même sur mon ancienne décision ». Mécanisme retenu, à livrer dans le lot de montée v1.3.7 (après la fusion de la PR 207, qui touche aussi le banc) : une section `[[decision]]` dans `config/replay_corpus.toml` (mesure ou repli, témoins ou tous, version du décodeur qui l'introduit, date, raison) ; le gate classe un FAUX qui y correspond « accepté (décision du JJ/MM) » au lieu de FAUX, et sort en code 0 s'il ne reste rien d'autre ; une décision qui ne correspond plus à aucun FAUX (la base l'a rattrapée) est signalée « à retirer », pour que la liste ne grossisse pas. Entrées initiales : la bulle d'écran occultant sur `bcb6d393` (13/11), le repli de visée du tir (14/11), et les deux FAUX de grille acceptés par le user le 14/11 dans le lot de levelup-7c si elle les y veut.
- 2026-11-14, **montée de LevelUp au décodeur v1.3.7 fusionnée dans release/v8.2.0** (e7115865a1, plus un correctif de lint 76da40dc4f ; CI et Release vertes sur 76da40dc4f). Contenu : schéma 116 (direction des tirs), marche partagée activée côté `replaybuild`, décisions du banc (`[[decision]]`, filets compris, nombre de replis). `replay-equiv` 20 sur 20, gate 18 sur 18 présents ok. A4.2 `[x]`. La régénération unique du parc suit le feu vert du pilote (levelup-7c). Worktree `LevelUp-wt-partage` laissé en place (aucune jonction ; à retirer plus tard, en vérifiant le compte de films avant et après).
- 2026-11-14, **reprise après la régénération unique du parc** (signal de levelup-7c : 1 455 rejeux au schéma 116, décodeur v1.3.7). Branche du décodeur `feat/fin-phase-a`. Mesures tirées des 1 455 documents régénérés par la vraie cuisson de LevelUp (chemin de l'hôte complet : source de dégât, objectifs, zones), lus en lecture seule (`C:\t\marche\A3\parc116\`), sans toucher aux bases que tient le serveur local.
  **A3.5 `[x]`** (commit d2c4d52) : `internal/depottest/balayages_test.go`. Relevé AST des sources de production du module : toute boucle à pas d'une position (`v++`, `v += 1`, ou `v++` dans le corps d'une boucle sans pas) qui passe `v` et une tranche d'octets paramètre à un appel. 38 fonctions nommées avec leur raison : 6 derrière la marche, 4 sur paquet non fermé, 9 en pleine passe (sonde et récupération d'équipement, sept sites de la voie de l'hôte), 7 dans les bornes d'un record, 2 atteintes par les seuls tests, 10 hors paquet delta. Mutation (boucle ajoutée à `camo_state.go`) : rouge ; retirée : vert. Limite écrite : un lecteur construit hors de l'appel ou un `range` sur un entier échappe au relevé.
  **A3.2 `[x]`** : aucune récupération n'est à zéro sur le corpus, toutes gardées. Comptes (déclenchements ; documents ; builds récents HI_1_12_0 et HI_1_13_0, 1 345 documents / anciens, 110) : pistes du monde 1 567 886 ; 1 440 ; 186 946 / 1 380 940. Ancrage du bipède 836 569 ; 152 ; 60 089 / 776 480. Localisation à largeur libre 49 454 ; 77 ; 18 / 49 436. Créations du monde 27 415 ; 931 ; 3 640 / 23 775. Liaison par anticipation 6 911 ; 89 ; 96 / 6 815. Début de liste fermé au bit 6 774 ; 59 ; 0 / 6 774. Kill rattrapé hors vue A 5 944 ; 60 ; 2 / 5 942. Génération vivante inconnue 2 ; 2 ; 0 / 2. Signature du slot 123 : sans compteur, gardée (dans la liste de A3.5). Retrait par build non fait pour le début de liste et la génération vivante inconnue (zéro sur les builds récents) : ils n'entrent que sur un paquet non fermé, leur coût est nul, et les couper par build ajouterait une branche par version du jeu sans gain mesurable.
  **A3.4 `[~]`** : sans objet. A2 n'a introduit aucune des deux aides : A2.H garde la table d'entités d'un chunk au suivant au lieu de lire l'instantané (relecture de l'image-clé hors périmètre, D-5), et aucune reprise au message de tick n'existe dans le code (recherche faite).
  **A6.3 `[x]`** : `ScoreTimeline.TargetScore` (`replay/score_timeline.go`) dit que le film porte la valeur, sans en être la source ; le commentaire du document (`document_score.go`) le disait déjà.
  **A6.4 `[x]`** : score de victoire lu dans le film contre la cible publiée (celle de `regulation.toml`) sur les 1 455 documents : 1 420 égaux, 6 différents, 24 sans cible publiée (variante absente de la table ou garde), 5 films sans section d'identité. Les 6 écarts sont tous du même genre : One Flag (3) et Minigame Asymmetrical (3), film 1 contre table 3. Le film y porte le score d'une manche (un drapeau gagne la manche), la table le nombre de manches (ADR 0032) : la table a raison pour la barre de score du match. Sur les variantes où les deux existent, accord 99,6 %. Instrument : `research/infosdematch` (non versionné), sorties dans `C:\t\marche\A6\mesure\`.
  **A3.3 en cours** : 46 entrées du registre (130) n'apparaissent dans aucun des 1 455 documents. Ce zéro ne vaut retrait que si le compteur de l'entrée finit dans les documents et que l'entrée est un contournement remplacé par une lecture ; une garde contre une entrée dégradée (chunk manquant, build non calibré) ou une entrée de l'hôte comptée hors du rejeu reste. Classement sur pièces confié à un agent en lecture seule, puis vérifié ici avant tout retrait.
- 2026-11-14, **A3.3 `[x]`** (commit du décodeur b84ef68). Sur les 46 entrées à zéro dans les 1 455 documents : 2 retirées (`repli_largeurs_monde_par_defaut_conservees`, `repli_index_de_region_largeur_un`, inatteignables depuis toute pose de production : vérifié sur pièces), 17 gardées parce que leur compteur ne passe pas par les documents (collecteur, résumé d'usage, service du décor, carnet des dérivations, hors production), 26 gardées comme protections contre une entrée dégradée ou inconnue absente du parc local, 1 gardée redondante au pire. Détail : `annexes_marche_complete/A3_3_replis_a_zero_2026-11-14.md`. Cibles de retrait des entrées gardées inchangées : la mesure ne change pas ce qui les retirerait. Révisions constantes, empreintes refigées ; suite du dépôt verte avec les données de l'hôte (fixtures de contrat contre le worktree de `release/v8.2.0`). Registre : 128 entrées. Équivalence et profil d'une cuisson réelle en cours (A3.6.h et A4.1).
- 2026-11-14, **A4.1, A3.6.h, A4.3, A5.5**. Équivalence sur les 20 films avec les commits A3.3 à A3.5 (décodeur `feat/fin-phase-a`, LevelUp `release/v8.2.0` par l'espace de travail des briques, films lus dans l'archive du dépôt principal par variable, sans jonction) : **20 identiques sur 20**, pic mémoire 2,52 Gio.
  **A4.1 `[x]`** : profil CPU de la cuisson réelle des 20 films (330 s d'échantillons ; l'observateur d'étapes de l'outil d'équivalence en prend 26 %, hors production). Ce qui reste, en part du total : marche commune de la source de dégât et de la cuisson 21 %, images-clés 9 %, localisation de la liste par la signature du slot 123 et la largeur libre 9 % (sur les seuls builds anciens, HI_1_4_1 à HI_1_10_0 : jusqu'à 12 s sur un film), calibration des largeurs de la source de dégât 6 %, poses d'équipement 5 %, statborg 5 %, propriétés gérées (zones) 4,3 % du total mais 12 à 20 % d'une cuisson sur les modes à zones, états de mort 2,3 %, récupération d'équipement 1,5 % ; points de navigation, objectifs, projectiles, état d'équipement et détecteur de i0 sous 0,5 % chacun.
  **A3.6.h `[!]`** : seul site de la voie de l'hôte qui pèse, les propriétés gérées ont été passées derrière la marche puis le changement annulé : il change ce que le rejeu publie (D11). Les six autres sites pèsent sous 0,5 % (états de mort 2,3 %) : non traités, comme A3.6.g. **A3.6 `[!]`** : a à f et i faits ; g et h non traités, avec leur mesure.
  **Porte A3** : C2 non tenu, et ne le sera pas dans ce plan : 9 fonctions balayent encore en pleine passe (sonde et récupération d'équipement, sept sites de la voie de l'hôte) et la localisation des builds anciens tourne sur les paquets que leur lecture ne ferme pas. Elles sont nommées, avec leur raison, dans le garde-rail de A3.5, qui refuse toute nouvelle. Registre sans entrée retirable à compte nul (A3.3).
  **A4.3 `[x]`** (commit c3def11) : base des bancs re-figée. TraverseEntity -40 %, KeyframeClosure -34 %. BitReaderReadBits +60 % n'est pas du code : le tag v1.0.0 mesuré dans la même minute rend la même médiane (197 contre 195 µs) ; c'est l'état de la machine.
  **A5.5 `[x]`** : les commentaires ajoutés de A2 à A4 hors chroniques ne portent ni code de lot ni histoire ; les mentions « plan de la marche complète, A2.x » sont dans les chroniques de révision, leur place. Hors périmètre (règle 17, pas de réécriture de masse) : les « relevé A0.3 » de `objectives/named.go`, `objectives/statborg.go` et `types/objectives.go`, d'une autre campagne, antérieurs à ce plan.
  **Porte A4** : C3 tenu (mesures du 13/11 ; A4.2 fusionné). **Porte A6** : tenue.
- 2026-11-14, **Z.1 `[x]`** : revue du diff du décodeur par un contexte neuf, une ronde : aucun défaut recevable, 8 conditions vérifiées ; une réserve P2 (la pose des largeurs, exportée par l'alias `decfilm.ProfilDeBalayage`, ne rend plus de rapport : changement d'API écrit dans la PR et la release). PR 19 du dépôt du décodeur ouverte. **Décision du pilote (levelup-dd)** : la 8.2.0 est close côté contenu ; v1.3.8 se tague et se publie après CI verte, mais la montée de LevelUp attend la branche de la release suivante, rien n'est poussé sur `release/v8.2.0`.
- 2026-11-14, **décodeur v1.3.8 publié** (PR 19, main aa25428, tag et release GitHub : « aucun changement de sortie », changement d'API écrit). CI verte sur les six travaux. **Clôture** : Z.1 `[x]` (ci-dessus). Z.2 `[~]` : CI du dépôt du décodeur verte ; le filet de LevelUp (`make gate-push`, CI de la branche) se joue à la montée de LevelUp en v1.3.8, qui attend la branche de la release suivante (décision du pilote). Z.3 `[~]` : notes du décodeur publiées (en français, comme les précédentes) ; les notes de LevelUp, en français et en anglais, se rédigent à cette montée. Z.4 `[x]` : demande faite au user le 14/11 (pas de recuisson : aucune sortie ne change ; suppression des tests de recherche clos de A5.4 ; suite du §10). Z.5 `[x]` : caches du lot purgés (aucun cache `marche` ; `golangci-fina` supprimé). **Plan clos, hors montée de LevelUp.** Suite, dans l'ordre du §10 : phase C premier temps (dont D11), puis phase B (rendus), puis phase C second temps.
- 2026-11-14, **suppression des tests de recherche clos (A5.4), sur accord du user** (« oui »). PR 20 du dépôt du décodeur fusionnée (main fd7303c), CI verte, sans tag (seuls des fichiers sous le tag `research` changent). 116 fichiers retirés parmi les campagnes closes de l'inventaire (mouvement, lot 1.9.1, lot 1, tir continu, campagne de grammaire, chunk_00 et table des joueurs, vue A hors `va_v3`, type 103, profil et registre, fil des morts hors abstention BTB 2025, modes de jeu). Gardés : 25 instruments cités par le code ou un test comme moyen de re-mesurer une règle, et 64 dont un instrument ouvert utilise les aides (fermeture calculée par `go vet -tags=research`). Sept symboles devenus orphelins retirés (`unused` sous le tag : aucun constat neuf par rapport à `main`). Les campagnes « incertaines » ou « non closes » restent.
