— Tâches et TODO centralisés

---

### [citations] « Annexion forcée » compte les captures de tous les joueurs du match

**Relevé le 2026-11-25** (plan `.ai/PLAN_BACKLOG_2026-11-24.md`, piste données, signalé par l'agent et non noté à la clôture). `computeAnnexionForcee` (`apps/go-api/internal/games/halo_infinite/citations_custom.go`) compte les temps forts « mode » de tous les joueurs du match, sans filtrer sur le joueur dont on calcule la citation. Le plan a corrigé le mode (Bases seulement) et le double compte, pas ce filtre. Effet non mesuré : la valeur affichée de la citation serait trop haute pour chaque joueur. **Pour reprendre :** confirmer sur un match de Bases (compte du joueur contre compte du match), filtrer, puis recalculer la citation (`levelup backfill --citations-recompute-all`). **Effort : S. Priorité : moyenne (une valeur affichée est peut-être fausse).**

---

### [divers] Signalements des agents du plan du 24 novembre non repris à la clôture

**Relevé le 2026-11-25.** Signalés par les agents pendant le plan `.ai/PLAN_BACKLOG_2026-11-24.md`, sans effet visible connu, non traités :
- Rejeu des événements d'un match déjà rempli : le statut rendu dit « no_film » alors que rien n'était à insérer.
- Compteur des coups d'envoi de la synchronisation : il compte à la résolution, pas à l'écriture acquittée par la file ; le brancher à l'acquittement demande un point d'accrochage.
- `mapcatalog/store.go` journalise par `slog.Warn` sans contexte (règle 3).
- `internal/platform/auth/pool/pool.go` dépasse toujours 500 lignes (dette sous baseline).
- L'étape de démarrage « comptes » est devenue quasi instantanée : son libellé à l'écran de démarrage ne s'affiche presque plus, à retirer ou à reformuler.
- `scripts/check_test_baseline.sh` et `archlint/no_unbounded_film_loop_test.go` citent encore d'anciens noms (tests à pseudo, `libs/film/...`).
- Deux copies identiques d'un hook de largeur (`_shared/assists/useElementWidth.ts`, `synthesis/WeaponRecordsRuler.tsx`), inscrites sur la liste datée du garde des observateurs de taille.
- Rejeu, répliques : à 4×, une bulle dure 0,65 s réelle ; le bouton épinglé des répliques reste grisé sur un rejeu sans réplique nommée ; la mémoire des voix n'a pas été mesurée dans un navigateur.
- Poste de développement : un exécutable Go bâti par `cmd/chatter-build` a été bloqué par l'antivirus (faux positif), `go run` passe.
- Trois fixtures HTML du rejeu (`ui/__fixtures__/replayTransport*.avant.html`) portent le pseudo du mainteneur (gardé par décision, noté pour mémoire).

**Effort : faible, point par point. Priorité : basse.**

---

### [commentaires] Commentaires du code périmés (regroupés)

**Relevé le 2026-11-22. Mis à jour le 2026-11-25** (plan du 2026-11-24, E.2, `395b4b82e4`) : fait : 23 fichiers corrigés sans effet à l'écran (les marqueurs `color-allow` qui citaient « CLAUDE.md §20 », les tons de `lib/accessibility/semantic-tokens.ts`, `games/canonical/fields.go`, `analysis/modelabel/category.go`, la doc de `replay-corpus-gate` en anglais et en français, l'en-tête de `cmd/killsource`). Reste : le « libellé d'un garde-rail qui décrit l'ancien comportement » (relevé de la relecture finale du 2026-11-23, voir la puce du même nom de la section « 2026-11-23 ») n'a pas été localisé avec certitude : deux messages d'un test de garde ont été corrigés, sans preuve que ce soit celui visé. Pour reprendre : le nom du garde, donné par qui l'a relevé.

**Effort : faible. Priorité basse.**

---

### [décodeur/rejeu] Libellés des médailles du rejeu : champ servi sans jumeau dans le document du décodeur

**Relevé le 2026-11-18** (contrôle complet de la refonte de la page du rejeu). Le document de rejeu SERVI porte `medalLabels` (libellé par langue, description, image, rareté), posé à la requête ; le document STOCKÉ du décodeur n'a pas ce champ, alors que les libellés de véhicules y ont le leur, rempli de la même façon. Les deux garde-rails qui comparent les deux documents lisent donc une liste datée de champs « servis seuls » (`internal/testutil/replay_servis_seuls.go`). À une prochaine version du décodeur : y déclarer le champ, puis vider la liste. Aucun effet à l'écran.

**Effort : faible côté LevelUp, une version du décodeur. Priorité basse.**

---

## 🔄 En cours

- **Backlog du 24 novembre (plan `.ai/PLAN_BACKLOG_2026-11-24.md`)** : plan exécuté et relu (revue finale : aucun P0, deux P1 corrigés), fusionné dans `release/v8.3.0`, CI verte. Restent : les contrôles à l'écran du user (artefact des vérifications, chantier « Backlog du 24 novembre (8.3.0) »), `make gate-push` avec la comparaison complète des tests, puis `main`. Reports : `.ai/REGISTRE_REPORTS.md` ; gestes de production : `.ai/gestes_prod_8_3_0.md` (sections A et B).
- **Version 8.2.0 (branche `release/v8.2.0`, plan `.ai/PLAN_V8_1_2_2026-11-09.md`)** : plan exécuté et relu (relecture
  finale sans P0 ni P1). Restent les gestes du user : contrôles à l'écran (items 5.9 et 7.2 du plan, artefact des
  vérifications), fusion vers `main`, tag et gestes de production (`.ai/V8.3/annexes_v8_1_2/gestes_prod_8_2_0.md`). L'item 7.4
  du plan (numéros de version, notes de version, runbook) n'est pas statué au plan ; ses brouillons sont sous
  `.ai/V8.3/annexes_v8_1_2/brouillon_versions_8_2_0_{FR,EN}.md`.
- **Plans de la racine de `.ai/` dont le statut dit encore « en cours » ou dont des cases restent vides** (relevé du
  2026-11-14, cases non vérifiées une à une) : `PLAN_REJEU_30IPS_COMPRESSION_2026-11-10.md` (compression par la PR #199,
  cuisson à 30 images par seconde par la PR #207 ; cases B4 à E2 à statuer par la session du chantier, C1 à C4 sont des
  gestes du user), `PLAN_REJEU_MAQUETTES_2026-11-10.md` (livré par la PR #200 ; case 7.2 ouverte) et
  `PLAN_MARCHE_COMPLETE_ET_CANAUX_2026-11-10.md` (lecture des paquets par la PR #198, décodeur v1.3.6 consommé par la PR #207 ;
  cases A2 à Z ouvertes, dont la recuisson unique du parc, geste du user). Le chantier application de bureau est clos (plan `.ai/V8/PLAN_DESKTOP_2026-11-01.md`,
  versions 8.1.0 et 8.1.1 publiées).

---

## 📋 Backlog

---

### [match/contrat] Restes de la page du match et du contrat relevés par le plan du 24 novembre

**Relevé le 2026-11-25** (plan du 2026-11-24, piste B). Après le retrait de `nemesis_duels`, des fonctions sans appelant et des champs de médaille sans lecteur, restent :
- `raw_json.medal_name` des événements de médaille est encore écrit (flux principal, complétion, passe et commande de rattrapage) sans lecteur ;
- `MatchViewRepo.LookupMedalMetaByName` n'a aucun appelant de production ; `MatchHistoryService.dataAdapter` est assigné et jamais lu ; `mapMatchSummaries` de Halo 5 est peut-être sans appelant ;
- le pied de la grille de valeurs de la page du match affiche « 0 » pour une colonne dont toutes les valeurs sont absentes ;
- le snapshot de surface du contrat est en retard sur des ajouts antérieurs.

**Pour reprendre :** vérifier chaque nom par recherche (hors tests) avant suppression, puis supprimer avec les tests (règle 7) ; ne pas afficher « 0 » au pied d'une colonne sans valeur (une valeur absente reste « non mesurée »). **Effort : à estimer. Priorité : basse.**

---

### [match/objectifs] Séries temporelles des éliminations liées à l'objectif

**Relevé le 2026-11-25** (plan du 2026-11-24, B.3). La fiche de match affiche maintenant six colonnes d'éliminations liées à l'objectif par mode ; les séries temporelles n'ont pas été faites : ces grandeurs n'ont de sens d'un match à l'autre qu'en cumul par période, et il faut un champ serveur neuf par mode (`features/timeseries/TimeseriesObjectiveCard.tsx` existe).

**Pour reprendre :** dire quelle grandeur a un sens en cumul par période (règle : Sessions, séries temporelles, Escouade), puis ajouter son champ par mode. **Effort : M. Priorité : basse.**

---

### [boot/auth] Pool de jetons : trois restes de la revue du démarrage

**Relevé le 2026-11-25** (plan du 2026-11-24, revue adversariale de la piste serveur, D.2) :
- le plafond du pool n'est pas tenu si un compte arrive pendant le remplissage (sans effet : la production ne pose pas de plafond) ;
- un retrait de consentement fait pendant une reconnexion peut être ignoré jusqu'au re-scan suivant (motif antérieur au plan) ;
- deux tests du pool échouent quand on les répète sous détection de courses (noms non relevés).

**Pour reprendre :** relancer les tests du pool avec `-race -count=N` pour nommer les deux ; traiter le retrait de consentement en premier (ADR 0038). **Effort : à estimer. Priorité : basse.**

---

### [tests/poste] Tests qui échouent ou dépassent leur délai sur le poste du développeur

**Relevé le 2026-11-25** (plan du 2026-11-24, Z.1, suites complètes sur la branche réunie) :
- `TestWeaponLabels_ArtefactReelDuPoste` (`internal/service/replay_weapon_labels_completion_test.go`) échoue sur un poste qui a le rejeu : il lit le fichier compressé comme du texte. Antérieur au plan (test inchangé depuis la 8.2.0) ; invisible en CI et dans les worktrees, où il se saute ;
- deux gardes de confidentialité échouent sur un poste qui porte des fichiers locaux non versionnés (exécutables sous `apps/go-api/bin`, anciennes fixtures) ; la CI ne les voit pas (déduit : elle part d'un clone propre) ; leurs noms ne sont pas relevés dans le plan ;
- `features/_shared/encounters/encountersFormat.guard.test.ts` dépasse son délai sous forte charge (suite web complète : 15 909 tests passés) et passe seul.

**Pour reprendre :** décompresser le rejeu avant lecture ou sauter le test ; limiter les deux gardes aux fichiers suivis par git ; relever le délai du garde du front ou alléger son balayage. **Effort : à estimer. Priorité : basse.**

---

### [rejeu] Restes de la piste rejeu du plan du 24 novembre

**Relevé le 2026-11-25** (plan du 2026-11-24, piste C) : `MatchNemesisCards.logic.ts` dit encore qu'un bot n'est jamais un antagoniste (page du match), alors que les bots sont rouverts dans les antagonistes (C.4) ; deux copies identiques d'un hook de largeur figurent sur la liste datée du garde ; le commentaire « 58 clés » du schéma du document de rejeu est périmé. **Effort : à estimer. Priorité : basse.**

---

### [contrat] `scoreActions[].target` servi sans lecteur

**Relevé le 2026-11-25** (revue finale du plan du 2026-11-24 ; antérieur au plan). Le champ `target` des actions de score du document de rejeu est servi et n'a aucun lecteur de production (règle 18 : une donnée servie a un lecteur). **Pour reprendre :** chercher le lecteur prévu (la désignation de la victime d'une action de score) ou retirer le champ du contrat servi. **Effort : à estimer. Priorité : basse.**

---

### [hygiène/gardes] Gardes figées sur 2026 et processus lancés hors garde

**Relevé le 2026-11-25** (plan du 2026-11-24, piste E). La garde des exemptions datées des jetons est maintenant indifférente à l'année (`ff05b2b2b5`) ; deux autres gardes restent figées sur le préfixe « 2026- » : `checkpoint_single_site_test.go` et `no_inline_per_minute_test.go`. Par ailleurs `pkg/duckdbbackup` lance restic par `exec` nu, hors de la garde « tout enfant naît par childproc » (`pkg/` exclu ; jamais lancé sur le bureau), et `internal/mapcatalog/store.go` journalise par `slog.Warn` sans contexte. **Pour reprendre :** rendre les deux gardes indifférentes à l'année comme celle des jetons ; décider si `pkg/` entre dans la garde des processus enfants. **Effort : à estimer. Priorité : basse.**

---

### [sync/h5] Deux restes de l'abandon du correctif des matchs Halo 5 incomplets

**Relevé le 2026-11-24** (plan du 2026-11-24, 0.4 ; le correctif lui-même est écarté, voir le tableau « Récemment complété »). Deux défauts de code restent : le commentaire de `internal/games/halo_5/capture.go` promet qu'« un futur passage le re-tentera » après un échec de chronologie ou de tableau, ce qui est faux (un match connu est sauté, en synchronisation comme en rattrapage) ; `internal/games/halo_5/livesync/events_backfill_surfaces.go` fait un `DELETE` sur `highlight_events` hors de `persist` (table à index ART ; l'outil tourne serveur arrêté, mais la règle anti-corruption n°1 ne le prévoit pas). **Pour reprendre :** corriger le commentaire ; faire passer la réécriture par la voie d'écriture admise. **Effort : à estimer. Priorité : basse.**

---

### [données/temps de jeu] Temps de jeu écrêté sur les matchs dont la durée au registre est fausse

**Relevé le 2026-11-24** (revue du plan `.ai/PLAN_BACKLOG_2026-11-24.md`, piste données). Sur la base du poste, 35 matchs ont une durée au registre plus courte que le jeu réel (par exemple 201 s pour un match d'environ 1 130 s). Sur ces matchs, le temps de jeu rangé d'environ 550 joueurs humains a été écrêté à cette durée fausse par une ancienne passe de rattrapage ; la sauvegarde du 2026-11-22 porte déjà ces valeurs. La 8.3.0 empêche que cela recommence mais ne répare pas : la valeur d'origine de l'API n'est gardée nulle part en local. Effets : temps de jeu faux dans les notes de ces matchs, et fenêtre de la Mainmise parfois trop courte. Production non regardée.

**Pour reprendre :** écrire une commande de reprise qui relit les statistiques de ces matchs auprès de Halo et réécrit le temps de jeu par la voie admise (`persist.KickoffPersister`, ADR 0050), puis rejouer la Mainmise sur ces matchs (`levelup backfill-lead-share --force --match …`). Écartée : la déduction depuis la durée de jeu du match, juste pour les seuls joueurs restés jusqu'au bout (environ 150 joueurs partis en cours de match resteraient faux). La durée du registre elle-même reste fausse sur ces matchs : à corriger dans le même geste. **Décision du user du 2026-11-24 : noté au backlog, hors du plan.** **Effort : M. Priorité : basse.**

---


### [décodeur] Comparaison avec les parseurs voisins : ce qui reste après la v1.6.0

**Mis à jour le 2026-11-24** (clôture du plan du décodeur v1.6.0 ; relevé d'origine du 2026-11-22 sur `nuzzles/halo_api` 0.6.1 et libfilmshell de Den). L'apparence des Spartans est faite : publiée en codes bruts (`roster[].appearance`), sans table de noms. Fait : le parallélisme du décodeur est intégré (un film récent passe de 1,12 s'en v1.5.0 à 0,80 s à un fil, 0,53 s à deux, 0,48 s à quatre, 0,46 s à huit, sortie identique à l'octet). Abandonné : décompression sur plusieurs fils (gain de 0,1 à 0,8 %), lecture de plusieurs chunks à la fois (gain plafonné à 10-15 %), lecture au choix des canaux (abandonnée par le user le 2026-11-23).

Reste ouvert :
- **Nom des options de variante** : le film ne porte que le hachage (vérifié sur deux films, tous les décalages de bit) ; les archives lisibles du jeu ne rendent aucun des 30 hachages testés ; les noms sont probablement dans l'exécutable du jeu, non lu. Reprise : établir cette source (item E.3).
- **Crédit à nuzzles** pour les méthodes reprises : fait des deux côtés le 2026-11-24 (décodeur : fichier de crédits, README, release v1.6.0 ; LevelUp : `docs/ACKNOWLEDGMENTS.md`).

**Effort : S à M. Priorité : basse.**


---

### [décodeur/drapeau] Grande bataille : 45 films sans calque de portages du drapeau

**Relevé le 2026-11-24** (mesures après la régénération du parc, étape L.7 du plan du décodeur v1.6.0). 45 films de capture de drapeau à 16 joueurs ou plus ont des captures et des vols lus, mais aucun calque de portages, avant comme après la régénération. Deux causes :
- 3 films sont bloqués par la règle de reconnaissance du mode (`captures <= rafales`, `flagfilm.go:79` du décodeur). L'un d'eux a PERDU son calque à la v1.6.0 : la lecture de tous les sièges y fait passer les captures lues de 1 à 5, pour 3 rafales (qui sont ici une retransmission périodique du tableau des scores, pas des captures). Règle levée en essai : les 7 vrais porteurs sont trouvés sur 7, erreur cumulée du temps de porteur contre l'API de 114,9 s à 34,1 s. Les deux autres n'avaient pas de calque avant non plus.
- 42 films n'ont aucune rafale : la règle les écarte, comme avant. Ils demandent une autre condition de reconnaissance, non étudiée (pistes : accord des captures lues avec la courbe d'équipes, rapport vols sur captures pour écarter l'Oddball).

**Pour reprendre :** remplacer la règle par une condition qui ne suppose pas que les rafales comptent des captures, la mesurer sur les 45 films et sur les films hors drapeau (zéro faux positif exigé). Demande une version du décodeur et la régénération des seuls films concernés. Pièce : `.ai/V8.3/annexes_decodeur_v160/l7_drapeau_calque_perdu.md`. **Effort : M. Priorité : moyenne.**

---

### [déploiement] Rejeux du site hébergé à l'ancien schéma, sans film

**Relevé le 2026-11-24, instruit le même jour (priorité abaissée : non bloquant). Mis à jour le 2026-11-25** (plan du 2026-11-24, C.6, `2d9a7367ef`) : fait : le panneau de la vue d'ensemble d'administration compte les rejeux par schéma, avec la date du décompte ; reste : la reconstitution des films. Environ 2 080 rejeux du serveur n'ont pas d'équivalent local, et l'archive des films du serveur a été vidée le 2026-11-23 : ces rejeux ne peuvent être ni copiés ni régénérés. Lecture du code (vérifié) : la 8.3.0 SERT un rejeu à l'ancien schéma tel quel (`apps/go-api/internal/service/replay_service.go`, `GetReplay`, aucune comparaison de schéma, désérialisation tolérante, calques récents simplement absents) ; la régénération à la demande est réservée au bureau et au développement (`apps/go-api/internal/config/deployment.go`, capacité `GenerateReplays`).

**Ce qui reste :** ces rejeux n'auront jamais les calques neufs (répliques, gestes, vitalités, apparence, collages). Pour reprendre : reconstituer les films si l'archive est un jour rétablie. Voir `.ai/gestes_prod_8_3_0.md` (section du décodeur v1.6.1). **Effort : à estimer. Priorité : basse.**

---

### [décodeur/doc] README du décodeur : commande d'installation à `@v1.6.0`

**Relevé le 2026-11-24.** Le README de la brique cite encore `@v1.6.0` à la v1.6.1. À corriger à la prochaine version de la brique (le dépôt de la brique ne se modifie pas d'ici). **Effort : trivial. Priorité : basse.**

---

### [décodeur] Limites des calques neufs de la v1.6.x

**Relevé le 2026-11-24** (mesures après la régénération, étape L.7). Ce que les calques neufs ne couvrent pas, mesuré sur le parc de 1 455 rejeux :
- Répliques : aucun nom de réplique sur les builds 1.9 à 1.11 (52 734 répliques lues, aucune nommée) ; 88 % nommées en 1.12, 97 % en 1.13. Aucun geste publié sur les builds 1.9 à 1.11 (règle de validation).
- Accroupis et glissades : sur 7 films seulement (C.1 n'a pas élargi ces deux genres).
- Lettres des zones : le compteur d'accord avec le rang (B.6) est absent de `coverage.zones`.

**Pour reprendre :** décider, build par build, si une table de réplique ou une règle de geste est à établir pour les builds 1.9 à 1.11, et où publier le compteur des lettres. Pièce : `.ai/V8.3/annexes_decodeur_v160/l7_mesures_apres_regeneration.md` (sections 1 et 5). **Effort : moyen. Priorité : basse.**

---

### [rejeu/maquette] Écarts avec les maquettes à juger à l'écran après livraison

**Relevé le 2026-11-18, reformulé le 2026-11-23. Mis à jour le 2026-11-25** (plan du 2026-11-24, 0.5) : fait : les trois points sont inscrits à l'artefact des vérifications (trois lignes : où regarder, ce qui est à juger, ce qui est attendu) ; reste : le jugement du user à l'écran. Les trois points, sans dessin : les pictogrammes et le marquage actif des icônes de cadre ; le curseur de volume gardé dans la barre d'une ligne (la maquette n'a qu'une icône) ; le fond d'accent du bouton caméra quand un joueur est suivi. Huit écarts sur onze avaient été réglés ou décidés (plan `.ai/V8.3/PLAN_BACKLOG_2026-11-18.md`, Q.1 et Q.2 ; grille `.ai/V8.3/annexes_refonte_ui_rejeu/conformite_maquettes.md`). Aucun des trois n'est un défaut constaté. **Effort : faible. Priorité basse.**

---

### [match/équipement] Équipements « lâchés en mourant » sous-comptés dans la grille d'usage

**Relevé le 2026-11-17** (refonte de la page du rejeu, lecture de l'équipement à l'instant). Un lâcher est daté à la fin de la vie plus une image, mais il est attribué par le propriétaire de la place À CETTE IMAGE, lecture stricte : quand la place n'a plus de propriétaire à cet instant, le lâcher n'est attribué à personne. Sur le seul document de test complet, 19 lâchers sur 29 sont attribués ; avec la lecture « dernier propriétaire connu » (`ownerAtFrameOrLast`, prévue pour ce cas dans `rosterLogic`), les 29 le seraient. Les sept autres documents de test, amincis, vont de 0 sur 4 à 21 sur 89. La grille « Usage d'équipement, par joueur » de la page du match et la future section « Équipement » de la fiche du joueur du rejeu lisent la même fonction : corriger l'une corrige l'autre, et change les totaux affichés (plus de « lâchés », donc plus d'« objets pris »). Aucun rejeu à régénérer. **Décision du user du 2026-11-17 : « ok corrige le sous-compte des équipements lâchés »** ; correction lancée le soir même sur la branche de la refonte de la page du rejeu (attribution au joueur qui vient de mourir, cas limites testés) ; **Corrigé le 2026-11-17** sur la branche de la refonte (`gestureOwner` : occupant de l'image précédente, sinon dernier propriétaire connu) : 79 lâchers attribués sur 338 avant, 338 sur 338 après, sur huit documents réels. Contrôlé le 2026-11-17 par lecture du code : les résumés d'usage des Sessions, de l'Escouade et des Tendances sont calculés par le décodeur (`replay.BuildUsageSummary`, attribution par « la vie qui couvre l'image, sinon la dernière vie terminée avant »), qui n'a PAS ce défaut. Reste un cas de bord côté décodeur, sans test : une place reprise à l'image même du lâcher crédite le nouvel occupant, là où le site crédite le mort ; et, pour les poses déployées, le décodeur lit « ou juste avant » quand le site lit strictement, donc l'« utilisé » du mur peut différer entre la page du match et les Sessions. Fréquence non mesurée ; à verser au chantier décodeur.

**Effort : faible (une lecture, ses tests). Priorité moyenne : un compte affiché est faux par défaut.**

---

### [décodeur] Chantier décodeur à ouvrir (décision D-6 du plan)

**Relevé le 2026-11-17** (plan `.ai/PLAN_BACKLOG_2026-11-17.md`, décision D-6). **Mis à jour le 2026-11-23** : le plan du décodeur v1.6.0 a traité une partie de ces points. Restent, chacun renvoyé à son entrée : verdict « lecture complète » du score en BTB et en Assaut (voir « Actions de score : verdict « lecture complète » faux en BTB et en Assaut ») ; étape manquante de `replay-equiv` pour les joueurs du tableau (voir « `replay-equiv` ne compare ni les joueurs du tableau… ») ; arme d'une touche sur l'index brut (voir « Auteur d'un dégât lu sur l'index brut du film ») ; quatre compteurs nommés faux (voir « Écarts restants du gate du corpus ») ; lettres des zones (voir « Nommer les états des marqueurs d'objectifs »). Traités par la v1.6.0 : porteurs du drapeau au-delà de 8 sièges et commandes et gestes (voir leurs entrées). **Effort : à estimer.**


---

### [décodeur/score] Points de bombe en Assaut : explosion reconnue, action sans nom

**Relevé le 2026-11-23 (plan du décodeur v1.6.0, item A.4), mis au backlog à la demande du user.** Mesure sur 8 films d'Assaut :

- L'explosion de la bombe (+200) se reconnaît dans le film : le compteur `bomb_detonations` monte dans la même émission, 22 fois sur 22. Mais rien dans le film ne donne le NOM de l'action de score : la publier obligerait à inventer un nom.
- Un bonus de +10 tombe avec des messages de rang 349 (80 cas sur 85), sans texte, absents de la table des réponses de jeu, et sans compteur associé.
- D'autres bonus (+25, +50, +60, +75, +150, +250) n'ont aucun motif net.
- Même en publiant le +200, ces matchs resteraient « incomplets ».

Pistes : lire l'arbre des actions du mode dans les fichiers du jeu (graphe du mode) pour nommer l'explosion et les bonus ; ou décider d'un nom publié pour l'explosion seule. Rien n'est publié dans la v1.6.0. Pièces : annexe du plan du décodeur v1.6.0, rapport du groupe « score » (hors dépôt à ce jour).

### [décodeur] Actions de score : verdict « lecture complète » faux en BTB et en Assaut

**Relevé le 2026-11-17** (suites de l'audit des versions 8.2.0 à 8.3.0). **Mis à jour le 2026-11-23** (plan du décodeur v1.6.0, items A.1 à A.4). Trois causes prouvées et corrigées dans `feat/v1.6.0` : quatre genres de message illisibles (cinq Grandes batailles « incomplètes » deviennent complètes), une trame renvoyée par le jeu comptée deux fois, une dernière manche courte refusée. Reste, verdicts du 2026-11-23 :
- **Arbre des actions décalé sur la version précédant `HI_1_11_0`** : lu mais non prouvable (2 990 trames restent arrêtées sur une soixantaine de genres, la somme des actions n'égale l'API que pour 33 joueurs sur 232). Pour reprendre : la table des genres de la vue A de `HI_1_10_0`.
- **34 Grandes batailles à drapeau** : aucun groupe de la table ne porte leur arbre (meilleur accord 13,5 %). Pour reprendre : la source du groupe de scripts dans les fichiers du jeu.
- **Assaut** : voir « Points de bombe en Assaut » ci-dessus.

Tant que le verdict est faux dans ces modes, ne pas afficher le détail daté des actions de score. Mesure : `.ai/V8.3/annexes_suites_audit/sources.md` et `.ai/V8.3/annexes_decodeur_v160/a_score_verdicts_a3_a4_a5_a6.md`. **Effort : à estimer.**


---

### [décodeur] Tirs, touches et dégâts lus dans le film : lecture suffisante pour 12 % des joueurs seulement

**Relevé le 2026-11-17** (suites de l'audit des versions 8.2.0 à 8.3.0, `.ai/PLAN_SUITES_AUDIT_2026-11-16.md`). **Reformulée le 2026-11-17.** Constat sur pièces : l'échantillon de 8 fixtures était dominé par des builds anciens ; mesure du décodeur sur 193 films : lecture suffisante pour 23 % des joueurs, et le parc du poste est à environ 90 % au build 13. Les totaux de l'API restent la référence ; le film ne sert que pour les joueurs à lecture suffisante. Remesuré le 2026-11-24 après la régénération : 2 673 085 tirs lus avant, 3 256 907 après (+21,8 %), voir l'entrée « Bilan des tirs par joueur après la régénération » et `.ai/V8.3/annexes_decodeur_v160/l7_mesures_apres_regeneration.md` ; la part de joueurs à lecture suffisante n'a pas été recomptée. Mesure : `.ai/V8.3/annexes_suites_audit/sources.md`. **Effort : à estimer.**

---

### [divers] Test instable sous charge des suites de l'audit (2026-11-17)

**Relevé le 2026-11-17** (suites de l'audit des versions 8.2.0 à 8.3.0, `.ai/PLAN_SUITES_AUDIT_2026-11-16.md`, section Découvertes). **Reformulée le 2026-11-23** (plan `.ai/V8.3/PLAN_BACKLOG_2026-11-18.md`, K.4, diagnostic seul) : `TestBackfillRegistryNames_BaseTenueParUnAutreProcessus` (`apps/go-api/cmd/levelup`) échoue quand la machine est chargée et passe seul. Aucune course trouvée ; la sortie d'erreur du processus fils, qui était perdue, est maintenant branchée, la lecture du « prêt » a un délai explicite et la sortie du fils est jointe à l'échec. À reprendre s'il échoue de nouveau, avec cette sortie. **Effort : à estimer.**

---

### [décodeur/marqueurs] Nommer les états des marqueurs d'objectifs par la liste de cartographer

Relevé le 2026-11-16 : `NAVPOINT_NAMES` de OpenSpartan/cartographer (MIT) liste environ 65 noms dont l'empreinte murmur3 donne les identifiants des marqueurs. **Reformulée le 2026-11-17** : la prémisse est fausse (aucune des 65 empreintes ne figure dans le décodeur, qui ne publie aucun identifiant de marqueur). **Mis à jour le 2026-11-23** (plan du décodeur v1.6.0, item B.6) : la vraie piste est la lettre réelle des zones lue dans le film ; son calcul comme contrôle est fait et mesuré (231 lettres justes sur 231, 77 films de Bases) mais NON intégré, il reste sur la branche `piste/bl-zones` sans appelant. Pour reprendre : un balayage qui publie l'état et l'équipe des marqueurs, la lecture du composant qui arrête les images-clés, les deux compteurs de couverture. Pièce : `.ai/V8.3/annexes_decodeur_v160/b_zones_verdicts_b3_b4_b6_b7.md`. Créditer la source si la liste est reprise. **Effort : moyen à lourd.**


---

### [rejeu/fiche] Bouclier brisé : badge narratif « os de verre » et frags sans bouclier

**Demande de l'utilisateur (2026-11-16).** La fiche détaillée du rejeu comptera, à l'instant lu, combien de fois le joueur a perdu son bouclier. Ce compte se dérive de la valeur de bouclier publiée à chaque point du film : un passage à zéro = un bouclier brisé. Le film n'écrit pas d'événement « bouclier brisé » (vérifié le 2026-11-16 dans le décodeur LevelUp et dans libfilmshell). Idées à instruire à partir de ce compte : un badge narratif de type « os de verre » dans la matrice d'impact de la page du match (joueur qui perd souvent son bouclier), et une stat « frags réalisés sans bouclier ». **Condition de reprise :** le compte des boucliers brisés livré dans la fiche (refonte de la page du rejeu, 8.3.0) et décision du user. **Effort : à estimer.**

---

### [décodeur/répliques] Répliques des Spartans : affichage (textes, voix, maquette)

**Demande de l'utilisateur (2026-11-16), très basse priorité. Mis à jour le 2026-11-25** (plan du 2026-11-24, piste R, fusion `bfab87cce0`, revue finale `67ee911d1d`). Ce que le décodeur publie (v1.6.1, schéma 127) : `chatter[]` (chaque réplique avec son instant, son joueur et son `name`, nom court prouvé par hachage sur les builds `HI_1_12_0` et `HI_1_13_0`), `announcements[]`, et le paquet public `chattername`. Le film ne dit pas quelle variante audio a joué.

**Décisions de l'utilisateur :** 2026-11-23, textes ET voix, en français et en anglais, versés au dépôt comme les sons d'armes ; 2026-11-24, les trois affichages (bulles sur les joueurs, voix, fil) sont combinables avec des filtres, seules les répliques parlées s'affichent (les cris sans paroles, 36 % des répliques jouées, jamais), et aucun réglage de langue : les répliques suivent la langue de l'interface (français en français, anglais dans toute autre langue, par `config/locales.toml`).

**Fait :** la table des textes (519 répliques parlées, une variante par son, commande `cmd/chatter-build`), l'affichage (bulles, icône de voix, fil, familles, fréquence par phrase, réglages dans le panneau Vue, préréglages, épingle) et les voix françaises (519 fichiers, 7,66 Mo, chargées à la demande, 16 voix en mémoire au plus). Crédit : méthode et table de textes de Den (libfilmshell, MIT).

**Reste :** (1) les voix anglaises : 1 402 fichiers à recevoir du contributeur, liste et règle de nommage prêtes ; une fois déposées, relancer `chatter-build` (procédure dans `docs/COMMANDS.md`) ; (2) le contrôle à l'écran et à l'oreille sur un rejeu en build 1.12 ou 1.13 (artefact des vérifications) ; (3) la mesure réelle de la mémoire des voix préchargées (17 Mo au pire par calcul, non mesuré). Pièces hors dépôt, sur le poste du user, dossier de sorties du chantier du décodeur, sous-dossier `repliques/` : `a_fournir_en.tsv` (liste des voix anglaises) ; le sous-dossier `work/` contient des extraits du jeu qui ne doivent jamais être versés. Noms sur les anciens builds : voir « Nom court des répliques sur les versions du jeu d'avant HI_1_12_0 ». **Effort : moyen. Priorité : très basse.**

---

### [décodeur] Temps forts de type « mode » du film : ce qu'ils recouvrent

Relevé le 2026-11-17 (PR 17 d'OpenSpartan/libfilmshell), instruit le 2026-11-18 (`.ai/V8.3/annexes_decodeur_1118/temps_forts_mode.md`) : un temps fort « mode » est l'écho d'une action de score propre au mode, du même joueur, dans la même image ; le décodeur n'a rien à publier de plus. **Mis à jour le 2026-11-23** (plan du décodeur v1.6.0, item F.4) : sur 8 films récents, 409 temps forts « mode » sur 410 ont un objectif ET une action de score du même joueur à 2 s. **Mis à jour le 2026-11-25** (plan du 2026-11-24, A.2, `f245776240`) : fait : la citation « zone capturée » ne compte plus que les matchs de Bases, une fois par joueur et par instant (aucun écran de la page du match ni du rejeu ne lit un événement « mode », vérifié à l'étape 0 ; le recalcul des citations est un geste de production, `.ai/gestes_prod_8_3_0.md`, B2). Reste : Extraction, non mesuré (aucun film dans le lot). Pour reprendre : un film d'Extraction. **Effort : S.**

---

### [base/temps forts] Temps forts « mode » en double, sans code, sur 76 matchs

Relevé le 2026-11-18 (mesure : `.ai/V8.3/annexes_decodeur_1118/temps_forts_mode.md`, section 5). La table `highlight_events` portait 2 330 événements de type « mode » sans code (`type_hint` NULL, `raw_json` vide) sur 76 matchs ; sur 33 d'entre eux, les 1 284 événements sans code étaient tous des doubles d'un événement de code 10. **Mis à jour le 2026-11-25** (plan du 2026-11-24, A.1 et A.2) : fait : la forme sans code est l'ancien état du flux principal, qui ne posait pas le code avant le 2026-09-02 (déduit de deux indices) ; le rejeu des événements, seul écrivain qui pouvait encore doubler, n'insère plus que le surplus ; la migration `shared_highlight_events_dedupe_v1` a retiré les doubles. Reste : 1 046 événements « mode » sans code sur 43 matchs (joués entre juin et septembre 2026) qui n'ont que cette forme, sans jumeau à code 10. Tous les événements « mode » de la base portent le code 10 ou rien (mesuré le 2026-11-24). Pour reprendre : décider si une étape pose le code 10 sur ces lignes par la voie d'écriture admise (jamais un UPDATE ou un DELETE à la main sur la base partagée), ou si les lecteurs acceptent le code absent. **Effort : S.**

---

### [données/précision] Touches et précision lues par l'ancien appariement des tirs

Relevé le 2026-11-16 (montée du décodeur v1.4.1). **Reformulée le 2026-11-17** (fusionnée avec l'entrée « Retrait de l'ancienne précision par film »). Constat sur pièces : les deux lecteurs existent mais ne s'exécutent plus pour Halo Infinite depuis le 2026-09-01 (capacité `match.weapon.accuracy` non exposée) ; 0 ligne dans `weapon_accuracy` et `match_weapon_hit_distance` sur le poste ; aucun lecteur. Rien de faux n'est persisté. Le killcollector et l'ingestion de la précision lisent l'ancien appariement des tirs, qui retrouve 11 % des touches que compte l'API, contre 81 % pour les `shotStats[]` suffisants du décodeur.

**Retrait de l'ancienne précision par film, reporté au chantier décodeur** (décision du user du 2026-11-17). Le code ne s'exécute plus et n'écrit rien, mais le retirer casse `internal/filmreplis` : le décodeur v1.4.1 déclare encore deux replis tenus par LevelUp (`repli_distances_de_touche_desactivees`, `repli_precision_par_arme_passe_sautee`). Ordre : une version du décodeur retire ces deux replis, puis LevelUp supprime `sync/killcollector/hits.go`, le mapper, le persister et la table `match_weapon_hit_distance`. Inventaire fermé : `.ai/V8.3/annexes_backlog_1117/verification_0_3.md`, section F. `ConfigureFilmAccuracy` n'a aucun appelant de production. Sort : piste F (décision D-5 du plan). **Condition de reprise :** une version du décodeur sans les deux replis ; à défaut, un plan de passage de ces deux lecteurs à `shotStats[]` (changement de données persistées, rétro-remplissage à décider). **Effort : M.**

---

### [rejeu/drapeau] Porteurs du drapeau faux en grande bataille (au-delà de 8 sièges)

Relevé le 2026-11-16 (phase C du décodeur, C7). **Mis à jour le 2026-11-23** (plan du décodeur v1.6.0, items B.1 et B.2, intégrés) : le drapeau lit désormais tous les sièges du film, une base neutre lue par le film fait un seul drapeau, la suite d'un siège se coupe au changement d'occupant, un porteur immobile garde sa position ; sur 218 joueurs de 13 films, justes à 2 s de l'API : 161 avant, 203 après, aucun juste devenu faux. Restent à instruire : un portage dont le porteur précédent est un bot arrivé en cours de match (piste sans identifiant), le premier occupant d'un siège que le pont par les morts ne nomme pas, un portage faux apparu sur un film à 6 socles (deux drapeaux publiés au lieu de six, risque sur les cartes à 6 socles). Mesure après la régénération du parc (2026-11-24) : temps de porteur à 2 s de l'API juste pour 86,07 % des joueurs avant, 92,71 % après (143 devenus justes, 0 devenu faux) ; au-delà de 8 joueurs 83,27 % puis 93,38 % ; reste l'entrée « Grande bataille : 45 films sans calque de portages du drapeau ». Pièce : `.ai/V8.3/annexes_decodeur_v160/b1_causes_des_portages_faux.md`. **Effort : M.**


---

### [rejeu] Écarts restants du gate du corpus à la montée du décodeur v1.3.9

**Demande de l'utilisateur (2026-11-15).** « Un film qu'on lit, y a pas de raisons d'avoir des écarts comme ça. » Le gate du corpus (6 témoins, parc au schéma 116 contre décodeur 123) a été accepté par le user le 2026-11-15 sur recommandation.

**Reformulée le 2026-11-17.** Constat sur pièces : trois écarts réglés ou expliqués ; deux replis restent par construction (films de builds anciens) ; quatre compteurs nommés faux pour 78 joueurs comparés, cause inconnue. Sort : chantier décodeur, recherche (décision D-6 du plan). Le détail de chaque écart d'origine (score personnel, assistances, replis actifs, compteurs nommés) est dans l'historique git de ce fichier.

Règle de travail : pour chaque écart restant, trouver la cause dans le film (lecture de la grammaire, pas d'heuristique) et la corriger par une règle générale validée sur le parc, jusqu'à zéro faux positif et zéro repli actif sur ces témoins. Le décodeur v1.4.1 (phase C) a pu en réduire une partie ; remesurer ces témoins avant de reprendre. **Plan de référence :** `.ai/PLAN_MONTEE_DECODEUR_V1_3_9_2026-11-15.md` (worktree `LevelUp-wt-decodeur139`). **Effort : à estimer après instruction.**

---

### [progression] Jalon « jours avec une précision d'au moins 50 % » atteint trop facilement

Relevé le 2026-11-14 (plan `.ai/PLAN_COMPLEMENTS_V8_2_0_2026-11-14.md`, étape 1.5). `match_participants.accuracy` porte la précision native de l'API en POURCENTAGE, mais le jalon `accuracy_threshold_days` la compare au seuil 0,50 (`internal/platform/duckdb/post_sync_progression_store.go`, agrégat `Aggregates` autour de la ligne 179, et `internal/ops/milestone_dates.go`) : tout match à plus de 0,5 % de précision compte comme un jour réussi. Le test `post_sync_stores_test.go` fige ce comportement en le signalant comme « unité en question ». Corriger (diviser par 100 ou comparer à 50) change des jalons déjà affichés et datés. **Décision du user du 2026-11-15 : ne pas y toucher à ce stade, noté au backlog.** **Effort : S, plus la décision sur les jalons déjà posés.**

---

### [perf/notes] Les lignes d'objectif de l'équipe sont rechargées en entier à chaque passage post-sync

`LoadObjectiveTeamRowsForPlayer` (appelée par `applyPerfEnrichments`, `internal/sync/performance_helpers.go`) relit les lignes de TOUS les joueurs de TOUS les matchs du joueur à chaque passage du batch de notes, y compris en post-sync où presque tous les matchs sont déjà notés et sautés. Le coût n'est pas mesuré (une requête par joueur et par passage, quelques milliers de lignes sur la copie locale). Pistes : ne charger que les matchs à noter et ceux de la fenêtre de leur chaîne, ou mémoriser par joueur jusqu'au prochain sync. **Condition de reprise :** une mesure du temps du passage post-sync sur une base de production qui montre un coût. **Effort : petit à moyen.**

---

### [bureau/auth] Connexion Microsoft dans le navigateur : « redirect_uri » refusé par Azure

**Constaté le 2026-11-11** (poste de l'utilisateur, v8.1.1). Le serveur construit `http://localhost:<port lié>/auth/xbox/callback` (`handlers/auth_browser_login.go`, port mémorisé dans `desktop/port`, 53891 sur ce poste). Microsoft répond `invalid_request … redirect_uri … not valid` : l'enregistrement Azure du client `e1cb35ab` ne porte pas `http://localhost/auth/xbox/callback` sous la plateforme « Applications mobiles et de bureau » (seule plateforme où le port est libre ; sous « Web » le port doit correspondre exactement). Rien dans le code ni dans les bases copiées ne change cette adresse. **Réglé côté Azure le 2026-11-11 par l'utilisateur** (adresse ajoutée à l'inscription) ; reste à vérifier depuis l'application une fois les services Halo revenus. Le code d'appareil avait échoué le même jour pour une autre raison : `settings.svc.halowaypoint.com/spartan-token` en dépassement de délai (service de jetons Halo injoignable ce jour-là, voir l'entrée du journal).

---

### [replay/sons] Fins de partie multi-équipes par couleur — écran + annonceur — BLOQUÉS faute de donnée

Noté le 2026-08-27 (chantier rejeu 2D, plan `.ai/archive/V7.5/PLAN_REPLAY_CADRAGE_VICTOIRE.md`,
branche `wt/replay-cadrage-victoire`). L'écran de victoire et les sons de fin (lots B/C)
couvrent les matchs à 2 équipes + le FFA gagné ; les modes MULTI-ÉQUIPES (3+) n'ont ni
écran (décision D-B1) ni son. Or le jeu porte une famille complète d'annonces dédiées,
identifiée par transcription locale des packs annonceur :

- **8 répliques FR** « Partie terminée, l'équipe X est déclarée vainqueur » — bleue
  `1070034924`, rouge `808622693`, cyan `83592248`, mauve `186794961`, verte `265309140`,
  citron `784566745`, jaune `868146650`, orange `399957729` (ids `.wem` du pack
  `French(France)/sb_001_vo_ai_mp_announcer.pck`) ;
- **8 jumelles EN** « Game over — <color> team wins » — red `1005389916`, blue `101785491`,
  green `1010879786`, purple `100056384`, yellow `927455187`, orange `564119611`,
  lime `92374`, cyan `256805823` (pack `English(US)/…`).

**BLOQUÉ FAUTE DE DONNÉE (plan backlog du 2026-09-26, lot A5, statué `[!]` le 2026-09-27).**
Le plan d'exécution est prêt dans `.ai/archive/V7.5/PLAN_BACKLOG_2026-09-26.md`, section A5 : pièces, items
A5.0 à A5.7, et décision D-9. D-9 garde l'écran du joueur regardé, ajoute « Victoire de l'équipe
<Nom> » en défaite, et fait jouer la réplique de la couleur du gagnant pour tous. Deux faits établis
en lisant les données le 2026-09-26 bloquent le lot :
- **aucun match multi-équipes jouable** : il n'y a qu'un seul match à 3 équipes ou plus, de 2
  joueurs ou plus chacune, dans toute la base partagée locale (2025-08-13, aucun joueur suivi).
  Aucun des 111 artefacts de rejeu ne porte d'équipe au-delà de 1 ;
- **la table équipe → couleur annoncée n'existe nulle part** : les couleurs de `teamNames.ts`
  ne correspondent pas aux 8 couleurs de l'annonceur (magenta, doublon orange-rouge).

**Condition de reprise** : l'utilisateur joue une partie à 3 équipes ou plus (matchmaking ou
partie personnalisée), la synchronise, et note la couleur annoncée pour l'équipe gagnante. Le lot
peut alors partir, en commençant par la recherche hors ligne de la table dans les fichiers du jeu
installé (A5.0). Les 16 répliques sont déjà extraites et converties sous
`Desktop/Halo Infinite - Sons armes/_fin_partie/annonceur_{fr,en}_wav/` (non normalisées ; les
copies de `livraison/` y sont restées en 4 canaux). **Effort : M.**
**Constat du user le 2026-10-09 : aucun mode à plus de deux équipes (hors FFA) n'est disponible en
jeu actuellement.** Le lot reste en attente d'un tel mode.

---

### [data/objectifs] Blocs `EliminationStats` / `InfectionStats` — BLOQUÉS faute de donnée

Réduit le 2026-07-25 (v7.2.1, V721-02). Sur les 4 blocs de mode non extraits, **3 sont
livrés** (Stockpile, Extraction, et `VipStats` qui n'était même pas déclaré dans
`StatsBundle`) — 18 colonnes. Les 2 restants ne sont **pas** un manque de travail mais un
manque de donnée, établi sur payloads réels (P0, 10 matchs interrogés) :

- `EliminationStats` : les 2 seuls matchs Attrition en base ne portent **aucun** bloc de
  mode (`Stats` = `CoreStats` seul). Aucun mode Elimination parmi les 65 modes distincts
  présents en base.
- `InfectionStats` : « Survive The Undead 3.0 » est un Firefight UGC (catégorie 41), pas
  un Infection. Aucun mode Infection en base.

**Déblocage** : jouer une partie matchmaking de chacun de ces modes, puis la synchroniser. **Constat du user le 2026-10-09 : ni Élimination ni Infection ne sont disponibles en jeu actuellement.**
Le patron existe alors en triple exemplaire — **effort : S par bloc**. Interdiction
explicite d'inventer le schéma par analogie.


---

### [décodeur/véhicules] Cinq pièces montées publiées sans porteur, sur trois films

**Relevé le 2026-11-24** (mesure du parc régénéré, plan `.ai/PLAN_BACKLOG_2026-11-24.md`, piste F). Sur 2 619 pièces montées du parc, 5 sont publiées sans porteur (`turretParentUnknown`), sur 3 films : leur parent lu ne désigne aucun châssis connu. Elles ne sont pas dessinées ; aucune n'a de porteur faux. Cause non cherchée. L'ancien défaut « pièces montées à plus de 15 m de leur porteur » est clos : 0 cas sur le parc au schéma 127, le porteur est lu dans le film depuis la v1.5.0 du décodeur. **Effort : S. Priorité : basse.**

---

### [licences/tests] Surnoms dérivés de pseudos réels dans des identifiants de tests

Relevé le 2026-11-10. Noté pendant la v8.1.2, périmètre figé. Après la pseudonymisation des fixtures sous `apps/` (v8.1.2, item 5.4), des surnoms tirés de gamertags réels restent dans des noms de tests et des libellés (plusieurs centaines d'occurrences, liste dans les tables hors dépôt de la pseudonymisation). La table de remplacement ne remplace que des gamertags entiers. **Mis à jour le 2026-11-25** (plan du 2026-11-24, E.4, `389c8731ca`, décision D-14 du user) : fait : trois noms de tests qui portaient un pseudo, renommés avec leurs 13 lignes de la référence des tests ; le périmètre du plan est limité à ces trois noms. Reste : tous les autres surnoms, non bornables sans les tables hors dépôt. **Condition de reprise :** à planifier ; décision du user. **Effort : M.**

---

### [licences] Pseudos réels hors de `apps/`

Relevé le 2026-11-10. Noté pendant la v8.1.2, périmètre figé. `.claude/skills/db-schema/SKILL.md`, `.gitignore`, `config/titles/halo_infinite/constants.toml`, `docs/` et `.ai/` portent encore des gamertags réels. Un chemin Windows personnel reste dans `apps/go-api/cmd/restore_one_player/main.go`. Le pseudo du mainteneur (`JGtm`) est conservé par décision du même jour. **Mis à jour le 2026-11-25** (plan du 2026-11-24, E.4, décision D-14 du user) : rien de cela n'a été traité : le périmètre du plan est limité à trois noms de tests (voir « Surnoms dérivés de pseudos réels dans des identifiants de tests »). La passe a relevé en plus qu'un pseudo reste dans `.gitignore`, dans deux scripts de banc et dans l'ADR 0018. **Condition de reprise :** à planifier ; décision du user. **Effort : M.**

---

### [rejeu/gate] `replay-equiv` ne compare ni les joueurs du tableau des scores ni leurs stats

Relevé le 2026-11-15 par levelup-1f à la montée v1.3.9 : l'étape `score` de `cmd/replay-equiv` ignore `scoreTimeline.players[]` et `players[].stats` ; un défaut d'association ou de manches ne se voit que dans l'empreinte opaque `artifact`. Ajouter leur empreinte (étape neuve ou `score` étendue). **Effort : S.**

---

### [fonds de carte/plan] Plan par mode : découper le plan d'une carte selon le mode joué

**Relevé le 2026-11-18** (décision D-11 de `.ai/V8.3/PLAN_FONDS_RELIEF_2026-11-18.md`). Principe de Cartographer (Den) : découper le plan d'une carte selon le mode joué (volumes mortels et de sécurité, points d'apparition de la variante, voir `libfilmshell` `NavMesh::rings_within`). **Version visée : 8.3.1**, décision du user du 2026-11-18 (« le temps que je comprenne mieux comment il fait »). **Reprise : après la sortie de la 8.3.0.** Recherche faite le 2026-11-18 : `.ai/V8.3/annexes_plan_par_mode/recherche.md` (étiquettes de mode `<mode>_include` / `<mode>_exclude` des objets des variantes, murs invisibles dans les instances physiques sans rendu du `sbsp`, volumes de mort natifs dans le tag `levl` ; fermetures de Den retrouvées sur Live Fire et Streets ; méthode et lots proposés).

**À surveiller (relevé le 2026-11-22).** La demande n° 1 ouverte le 2026-11-17 sur `OpenSpartan/cartographer` (« Initial Minimap and App Startup Flow Improvements », encore ouverte) propose de retoucher à la main les minicartes des cartes les plus jouées en compétition (Aquarius : triangles à 19 m, zone basse inaccessible en bas au milieu). Certaines de ces zones ne sont fermées que dans certains modes : la recherche du 2026-11-18 le montre sur Aquarius (86 Player Blocker `infection_include`, 2 Projectile Blocker `infection_exclude`). Relire la demande et la réponse de Den avant de lancer le plan par mode. État au 2026-11-22 : aucun code ne découpe le plan selon le mode ; une carte affiche le même plan quel que soit le mode joué.

---

### [fonds de carte/Relief] Passages couverts : montrer le sol qui passe sous l'étage affiché

**Relevé le 2026-11-22** (relecture des dépôts de Den depuis le 2026-11-18). `OpenSpartan/films-ui` (MIT), `src/floor.ts`, commits du 18 et du 19/11 (demande de fusion n° 8) : la vue de dessus trace les murs du sol praticable qui passe sous l'étage affiché (tunnel sous la carte, passage sous un pont ou sous l'étage d'un bâtiment). Trait en tirets de 2 px (6 pleins, 4 vides) dans la couleur des ruptures de hauteur, sur un liseré plein de 4 px d'une teinte sombre du sol bas, à 0,85 de leur force, pour rester lisible sur sol bas comme haut. La donnée (`under`) sort du générateur privé de Den (`film-assets`) : chez nous, à calculer depuis le maillage de navigation à la cuisson du fond Relief (brique `halo-infinite-map-images`). Le fond Relief et la coupe par étage de la 8.3.0 n'affichent pas ces passages. Créditer Den si le style est repris. **Reprise :** après la sortie de la 8.3.0, avec ou après le plan par mode. **Effort : moyen (brique et dessin web).**

---

### [unités] Les « mètres » de l'application sont des unités du jeu d'environ 3 m

**Relevé le 2026-11-18** (conception du fond Relief, `.ai/V8.3/annexes_fonds_relief/conception.md` §9, mesure sur 1,55 million de points de rejeux : course à 2,2-2,4 unités par seconde, saut de 0,82 à 0,89 unité). **Mis à jour le 2026-11-25** (plan du 2026-11-24, 0.6, B.6 et B.7 ; décisions D-11, D-14, D-16 du user) : une unité du jeu vaut 3,048 m (10 pieds ; trois mesures la donnent à environ 3 m, 3,0 et 3,048 ne se séparent pas). Fait : toute distance affichée est convertie par un seul helper (web et jumeau Go, avec garde-rails) : 11 familles de distances ; les 22 champs d'API nommés en mètres ne sont pas renommés, ils sont documentés comme unités du jeu ; le radar de la Tactique compare désormais un rayon en vrais mètres à des distances converties (les comptes « isolé » changent : rayon ramené d'environ 55 m à 18 m, à dire dans les notes de version) ; Emprise : la portée du radar est convertie avant d'être transmise au décodeur (`27a9de07e0`).

**Reste :** (1) la part « hors radar » des vies de l'Emprise (`beyond_ms`, table `match_life_placement`) : 304 576 vies sur 1 454 matchs du poste portent encore la valeur sous-estimée ; les valeurs anciennes restent affichées jusqu'à la prochaine passe `levelup backfill-killsource`, qui exige les films (un match sans film garde sa valeur) ; (2) une dizaine de constantes web gardent un nom trompeur (seuils réglés sur des unités mais nommés « M » : étiquettes corrigées, comportement inchangé) ; les noms internes (`metersPerPixel` et voisins) sont laissés tels quels, aucun affichage n'en dépend. Voir `.ai/gestes_prod_8_3_0.md` (B2). **Effort : S. Priorité : basse.**

---

### [fonds de carte/Forge] Deux des quatre types de himap.TypesVolumesDeMort ne sont pas des volumes de mort

**Relevé le 2026-11-18** (recherche du plan par mode, `.ai/V8.3/annexes_plan_par_mode/recherche.md`). « Light probe marker » (type d'objet 29) et « Nav Cutter » (type 28) sont comptés comme volumes de mort par la brique `halo-infinite-map-images` (`himap.TypesVolumesDeMort`, `BoiteDesVolumesDeMort`). Effet possible sur le cadrage des fonds Forge publiés, à mesurer. **Reprise :** à instruire et décider par le user.

---

### [fonds de carte/Forge] L'échelle des primitives Forge n'est pas lue

**Relevé le 2026-11-18** (recherche du plan par mode, `.ai/V8.3/annexes_plan_par_mode/recherche.md`). Le vecteur d'échelle des objets (`#8/#23` de la variante) n'est pas lu par `mapvar`. 94,7 % des 404 578 primitives du parc ne sont pas à l'échelle 1, alors que `InstanceForge` les pose à l'échelle 1. Effet sur les fonds Forge publiés à mesurer avant tout correctif. **Reprise :** à instruire et décider par le user.

---

### [fonds de carte/plan par mode] Fichiers de variante des 20 cartes natives : copies locales vides, à télécharger avant le plan par mode

**Relevé le 2026-11-18, reformulé le 2026-11-24** (recherche du plan par mode, `.ai/V8.3/annexes_plan_par_mode/recherche.md`). Le fichier de variante d'une carte dit quels objets existent dans quel mode (murs, volumes, points d'apparition) : c'est la matière du plan par mode. Le dossier de travail local `.ai/re_dump/mapvar/` (hors dépôt) en porte 20, ceux des cartes natives, tous vides (0 octet). Rien ne les lit aujourd'hui. À télécharger (`mapnav-fetch -fichier map.mvar -nom-cuisson`) quand le plan par mode démarre. **Décision du user du 2026-11-24 :** on garde l'entrée, on ne s'en occupe pas tant que Den n'a pas répondu à la demande n° 1 ouverte sur `OpenSpartan/cartographer` (voir l'entrée « Plan par mode ») ; d'autres priorités passent avant. **Condition de reprise :** réponse de Den à cette demande, puis lancement du plan par mode.

---

### [décodeur/drapeau] Le compteur `carrierTeamUnknown` porte un nom trompeur

**Relevé le 2026-11-23** (plan du décodeur v1.6.0, item B.3). Sur un film à 6 socles (3 par camp), `carrierTeamUnknown` vaut 32 sur 32 portages, alors que l'équipe du porteur est lue pour 25 joueurs sur 25. Le compteur mesure l'abstention des trois règles qui nomment le DRAPEAU pris par l'équipe (passage de main, retour crédité, chute créditée) : sur une carte à 3 drapeaux par camp « rien ne les départage ». Conséquence mesurable nulle (temps de porteur à 2 s de l'API pour 24 joueurs sur 25). À faire : renommer le compteur (nom publié du document, donc schéma) ou le retirer, avec une version du décodeur ; ne pas le lire comme « équipe inconnue ». Pièce : `.ai/V8.3/annexes_decodeur_v160/b_zones_verdicts_b3_b4_b6_b7.md`. **Effort : S. Priorité : basse.**

---

### [décodeur/projectiles] 33 collages de projectile sur un véhicule restent sans cible

**Relevé le 2026-11-23** (plan du décodeur v1.6.0, projectiles collés, chronique du schéma 127). Sur les films mesurés, 33 projectiles collés se posent sur une entité de l'archétype des véhicules que `vehicles[]` ne publie pas sous ce slot : ils restent `unresolved`, aucune résolution n'est forcée (déduit : ce sont des vies de véhicule que le document ne publie pas). Pour reprendre : relier ces slots aux véhicules publiés (recoupement par la position, comme les 7 cas déjà résolus à 0,5 à 1,6 m), ou nommer le repli. **Effort : S à M. Priorité : basse.**

---

### [décodeur/répliques] Nom court des répliques sur les versions du jeu d'avant HI_1_12_0

**Relevé le 2026-11-23** (plan du décodeur v1.6.0, item D.2 ; décision du user du 2026-11-23 : pas dans la 1.6.0). Le nom court d'une réplique est publié pour les films des builds `HI_1_12_0` et `HI_1_13_0`, où il est prouvé par hachage (291 noms, 98 % des répliques nommées sur trois films récents). Sur `HI_1_10_0` et `HI_1_11_0`, la table est décalée par un bloc de 205 entrées inséré plus tôt : le décalage se mesure par la priorité (781 répliques sur 808, 400 sur 426, 489 sur 496) mais n'est pas prouvé par hachage, donc ces films sont publiés sans nom et comptés. Les autres builds ne sont pas mesurés. Restent aussi sans nom : les indices hors de 0-914 et 2732-2956 (ordre des deuxième et troisième tables non établi) et six répliques de ramassage d'arme. Pour reprendre : prouver le décalage par build (table de dialogue d'une ancienne installation, ou accord de priorité à 100 % sur un échantillon large), puis ouvrir la garde de build. **Effort : S à M. Priorité : basse.**

---

### [décodeur/pistes] Une vie sans tracé : un joueur à 18 morts pour 17 tracés de vie

**Relevé le 2026-11-24** (contrôle D.1 des répliques contre le témoin tiers). Sur un film, un joueur compte 18 morts pour 17 tracés de vie : la vie qui se termine à la trame 12 998 n'a pas de tracé, la suivante commence à 13 303. Une réplique tombe dans cette vie et reste publiée sans joueur (1 sur 948 répliques contrôlées). La lecture de la réplique est juste ; c'est une vie manquante dans les pistes. Cause non cherchée. Pour reprendre : lister, sur le parc, les joueurs dont le nombre de morts dépasse le nombre de tracés de vie, puis chercher pourquoi la naissance de cette vie n'ouvre pas de tracé. Pièce : `.ai/V8.3/annexes_decodeur_v160/d1_controle_repliques_contre_temoin.md`. **Effort : S à M. Priorité : basse.**

---

### [décodeur/journal] « Vies publiées sans identité » : une alerte sur 4 films du banc sur 13

**Relevé le 2026-11-24** (relecture finale de la v1.6.0). Le journal du décodeur lève l'alerte « vies publiées sans identité » sur 4 films du banc de 13, à l'identique en v1.5.0 et en v1.6.0 : ce n'est pas une régression de la version. Cause non cherchée (déduit : des vies de bots ou d'unités sans joueur, à confirmer). Pour reprendre : compter ces vies par film et par type d'occupant, et dire si c'est une absence de lecture ou un état normal du jeu. **Effort : S. Priorité : basse.**

---

### [décodeur/tests] Les jeux d'entrées figés ne portent pas les canaux neufs

**Relevé le 2026-11-24** (neuvième intégration de la v1.6.0). Les 8 jeux d'entrées figés des tests du décodeur sont le blob de base ; répliques, gestes, vitalités, collages et apparence vivent dans le complément du fichier de faits, que ce format ne contient pas. Ces canaux ne sont donc tenus que par le rendu complet du document (un compte et une empreinte par champ) et les tests par calque ; régénérés par le pipeline, les 8 jeux sortent identiques sauf un octet. Pour reprendre : étendre le format du fixture pour porter le complément, ce qui change les 8 fichiers et le budget de 11 Mio des fixtures de contrat. **Effort : M. Priorité : basse.**

---

### [décodeur/corpus] Le contrôle du corpus compte les socles toutes familles confondues

**Relevé le 2026-11-24** (gates finals de la v1.6.0). `padDating.uncovered`, `padDating.ambiguous` et `groundWeapons.unknown` additionnent toutes les familles de socle : chaque famille ajoutée (ici les équipements, 15 socles neufs sur un film) fait rougir 15 témoins sur 19 sans qu'aucun socle d'arme ne régresse (vérifié sur 4 films : les socles anciens sont identiques). Il a fallu trois décisions du manifeste. Pour reprendre : lire ces compteurs par famille de socle (armes, power-ups, équipements), ou comparer sur le sous-ensemble ancien. Pièce : `.ai/V8.3/annexes_decodeur_v160/l1b_enquete_corpus_causes.md`. **Effort : S. Priorité : basse.**

---

### [décodeur/drapeau] `gaugeReads` et les compteurs de rejet de la jauge ne sont pas sur la même base

**Relevé le 2026-11-24** (dixième intégration de la v1.6.0). `gaugeReads` ne compte que la passe ; les compteurs de rejet de la jauge du drapeau additionnent la passe et la marche, et les échantillons posés avant allégement ne sont pas dans le document : l'égalité lues = publiées + écartées ne se lit pas sur le seul document. Elle est tenue par un test (mutation vérifiée) et sur 3 films à drapeau par un recoupement des comptes. Pour reprendre : poser les deux sur la même base (passe plus marche) avec une version du schéma, ou nommer la base dans la description publiée. **Effort : S. Priorité : basse.**

---

### [décodeur/vitesse] La borne de fils est `Fils` plus 3 au plus

**Relevé le 2026-11-24** (neuvième intégration de la v1.6.0, relecture du parallélisme). Pour `Fils` demandés, jusqu'à `Fils` + 3 goroutines calculent en même temps (chaîne principale, voie du monde, voie des sièges, ouvriers du pré-calcul) : tenir `Fils` strictement coûtait 4 à 17 % de temps à 4 fils. Un hôte qui dimensionne ses cuissons simultanées sur `Fils` emploie donc quelques cœurs de plus. Écart assumé, dit dans les notes de la release. Pour reprendre : si un hôte multi-cuisson le demande, partager un seul jeu de fils entre les voies. **Effort : M. Priorité : basse.**

---

### [décodeur/grammaire] Genres de message 13, 27, 93 et 117 : charges ouvertes build par build, abstentions à reprendre

**Relevé le 2026-11-24** (correction de la relecture de la v1.6.0, mesure sur 1 000 films avec la carte). Les charges de ces quatre genres ne sont ouvertes que là où la fermeture de la trame au bit est prouvée : genre 13 sur 1.11 ; 27 sur 1.11 et 1.13 ; 93 sur 1.11, 1.12 et 1.13 ; 117 sur 1.12 et 1.13. Restent en abstention : genre 27 en 1.12 (2 messages, moins de cinq), genre 117 en 1.11 (36 fermées sur 37, une trame non fermée) ; les builds 1.10 et 1.9 restent fermés (1.10 : 23 à 34 % d'arrêts au message suivant selon le genre, donc charge non prouvée). Ailleurs le genre arrête la trame, arrêt compté. Pour reprendre : mesurer d'autres films 1.12 et 1.11 pour lever les deux abstentions ; chercher les largeurs exactes en 1.10 (build à des milliers de trames arrêtées). **Effort : M. Priorité : basse.**

---

### [décodeur/reprise du parc] `backfill-killsource` saute un film sur la seule révision de sa couche

**Relevé le 2026-11-24** (régénération du parc, v1.6.1). La commande décide de sauter un film d'après la révision du décodeur rangée dans les vues `_latest` du journal des morts, des vies et du placement ; les révisions de la grammaire et du profil n'y entrent pas. Comparaison sur 30 films copiés (reprise forcée) : journal des morts identique ligne à ligne ; les vies (`match_lives`), le placement et les tirs (`weapon_shots`) n'ont pas été comparés. Pour reprendre : refaire la comparaison sur ces trois tables, puis décider si les révisions de la grammaire et du profil doivent entrer dans la décision de saut. Pièce : `.ai/V8.3/annexes_decodeur_v160/l6_regeneration_reelle_resume.md`. **Effort : S. Priorité : basse.**

---

### [décodeur/profils] Un film écarté du journal des morts : profil de build `HI_1_5_1` absent

**Relevé le 2026-11-24** (régénération du parc). Un film (`58e6f72a`) est écarté avec une clé de profil inconnue : le build `HI_1_5_1` n'est pas dans la table des profils. Pour reprendre : dire si ce build est lisible (ajouter son profil, avec preuve sur le film) ou s'il faut garder l'écart nommé et compté. **Effort : S. Priorité : basse.**

---

### [décodeur/CI] Deux lots de CI de la brique frôlent leur délai

**Relevé le 2026-11-24** (CI du dépôt du décodeur). Les lots `film-grammar-lourd` et `film-replay-reste` durent environ 8 minutes pour un délai de 10. Pour reprendre : si l'un rougit, recouper les lots avant d'allonger le délai. **Effort : S. Priorité : basse.**

---

### [sync] Choisir jusqu'où remonter l'historique d'un joueur : une date ou un nombre de matchs

Relevé le 2026-11-22. Aujourd'hui (8.2.2) la synchronisation rattrape tout l'historique sans limite (décision du user, 2026-11-22).

---

## 🎮 Backlog — Coach proactif × Prestige (post-V2)

Référence : ADR 0020 — Coach proactif : pont vers Prestige. ADR 0021 — Synthèse dynamique de Template et Arc ad-hoc.

> **MàJ 2026-07-17** : les 3 extensions autrefois parkées sont LIVRÉES (train backlog) —
> V2.1 télémétrie `source` + endpoint diag, analyseur de tuning de la grammaire
> (recommandations, validation manuelle), et canal externe Discord webhook (opt-in, OFF
> par défaut). Voir « Récemment complété ».

**Arcs multi-titres** : clos le 2026-07-25 (v7.2.1, Option A, retrait `arc_titles`) — plan archivé
[.ai/archive/V7.2/PLAN_CROSS_TITLE_ARCS_2026-07.md](.ai/archive/V7.2/PLAN_CROSS_TITLE_ARCS_2026-07.md).

Enrichissements possibles (non planifiés) : funnel `coach_proposal.status` dans l'analyseur de tuning.
Réglages Discord sans interrupteur web : relais du coach, rejeu prêt, reconnexion requise, disque.

---

## ✅ Récemment complété (référence)

> Entrées antérieures à 2026-07 (ère Python/Streamlit, bascule vers Go) : `.ai/archive/BACKLOG_COMPLETE_avant_2026-07.md`.
> Les chantiers en cours se suivent dans leurs plans (`.ai/PLAN_*.md`) et dans la section « En cours » ci-dessus.

| Date | Item |
|------|------|
| 2026-11-25 | **[données] Éliminations en double dans `highlight_events`** — traité (plan `.ai/PLAN_BACKLOG_2026-11-24.md`, A.1 et A.2) : le rejeu des événements, seul écrivain qui pouvait encore doubler, n'insère plus que le surplus ; une étape de migration (`shared_highlight_events_dedupe_v1`) a retiré 38 573 lignes en double sur la base du poste, sans toucher une ligne légitime ; un match n'affiche plus chaque élimination deux fois. Avant de déployer, compter la production en lecture seule (`.ai/gestes_prod_8_3_0.md`, B1). |
| 2026-11-25 | **[décodeur/rejeu] Éliminations dont un bot est l'auteur : à rouvrir côté LevelUp à la montée du décodeur v1.6.0** — traité (plan `.ai/PLAN_BACKLOG_2026-11-24.md`, C.4, `0a18f92fb4`) : les éliminations et les actions de score d'un bot sont désignées par place (le `xuid` devient facultatif), les lignes de bot se rattachent à leur place dans le fil et les bots reviennent dans les antagonistes. |
| 2026-11-25 | **[contrat] `combat_tab.nemesis_duels` servi vide et sans lecteur** — traité (plan `.ai/PLAN_BACKLOG_2026-11-24.md`, B.5, `c583d1eca3`) : `nemesis_duels`, son type et le bloc `team_tab.nemesis` du fichier de référence sont retirés ; contrat OpenAPI et types web régénérés. |
| 2026-11-25 | **[code mort] Fonctions d'analyse sans appelant de production** — traité (plan `.ai/PLAN_BACKLOG_2026-11-24.md`, B.5, `c583d1eca3`) : `ComputeAntagonistCounts`, `ComputeModeCategoryAverages` et la méthode `LoadMatchSummaries` (trois titres, plus la source d'historique Halo 5 qui ne servait qu'à elle) sont supprimées avec leurs tests ; `spawn_detection.go` l'était déjà (`87910a4602`). |
| 2026-11-25 | **[médailles du bouclier] Trois défauts P2 de la revue adversariale, non corrigés (consigne de release : P0 et P1 seulement)** — traité (plan `.ai/PLAN_BACKLOG_2026-11-24.md`, A.5, `775068f07c`) : les frags à temps égal se classent par identifiant (même résultat quel que soit l'ordre d'insertion) ; la révision `lu2` n'est posée que si le fil des morts et les équipes ont été lus ; la vue des médailles (v2) décide « dérivé » sur l'existence d'une passe et non sur une ligne de joueur. Relancer `levelup backfill-usage-summary` est un geste de production (`.ai/gestes_prod_8_3_0.md`, B2). |
| 2026-11-25 | **[boot/auth] Le démarrage attend la résolution du pool de jetons avant d'ouvrir le port** — traité (plan `.ai/PLAN_BACKLOG_2026-11-24.md`, D.2, `bf3718ed37`) : le pool est créé vide, les comptes se résolvent en tâche de fond et le port s'ouvre sans attendre ; sur un banc simulé à douze comptes, 665 ms avant et moins de 1 ms après (temps réel à relever au premier déploiement). |
| 2026-11-25 | **[sync/journal] Alerte « zero-insert prolongé » qui sonne à chaque cycle en prod** — traité (plan `.ai/PLAN_BACKLOG_2026-11-24.md`, D.3, `94723378ab`) : l'alerte sonne au franchissement du seuil puis tous les 48 cycles, non plus à chaque cycle ; l'avertissement de jeton de la mise en file n'est journalisé qu'une fois par épisode. |
| 2026-11-25 | **[sync/h5] Match Halo 5 rangé sans participants ni chronologie, jamais repris** — écarté (plan `.ai/PLAN_BACKLOG_2026-11-24.md`, D.4, décisions D-5 et D-13) : au plus 66 matchs sur 3 032 (2,2 %) sont touchés, surtout des parties à un joueur de 2015 et 2016, sur un jeu qui n'est plus joué, et une commande de reprise manuelle existe (`cmd/h5-events-backfill`). Deux défauts de code relevés au passage sont au backlog. |
| 2026-11-25 | **[archlint] Garde « tout enfant naît par childproc » sans objet sur les briques** — traité (plan `.ai/PLAN_BACKLOG_2026-11-24.md`, E.3, `ff05b2b2b5`) : défaut écarté : le balayage `libs/` de la garde est actif, il désigne les briques du cache de modules ; la garde est gardée et son commentaire corrigé. Le lancement de restic de `pkg/duckdbbackup` hors garde reste au backlog. |
| 2026-11-25 | **[match/objectifs] Éliminations liées à l'objectif sur la fiche de match** — traité (plan `.ai/PLAN_BACKLOG_2026-11-24.md`, B.3, `15f9300c3b`) : six colonnes d'éliminations liées à l'objectif s'ajoutent à la section Objectifs de la fiche de match, par mode (web seul : le serveur les servait déjà). Les séries temporelles sont au backlog. |
| 2026-11-25 | **[outillage] disk-hygiene supprime les worktrees enregistrés sous C:/t et les clones git à commits non poussés** — déjà fait (plan `.ai/PLAN_BACKLOG_2026-11-24.md`, E.1) : corrigé le 2026-11-22 (`84ec5ed295`) : le script n'efface jamais un dossier qui est, ou qui contient jusqu'à trois niveaux plus bas, un dépôt ou un worktree git, et journalise ce qu'il épargne (relu dans le script, non rejoué). |
| 2026-11-25 | **[décodeur/objectifs] Ordre de rotation des collines du Roi de la colline** — traité (plan `.ai/PLAN_BACKLOG_2026-11-24.md`, C.2, `83ace65640`) : le rejeu affiche une pastille « Collines A › B › C » (suite des collines jouées) et la prochaine colline quand un cycle complet s'est déjà répété ; emplacement à juger à l'écran. |
| 2026-11-25 | **[décodeur] Commandes et gestes du joueur : notes de bas de page de la fiche du rejeu** — traité (plan `.ai/PLAN_BACKLOG_2026-11-24.md`, C.3, `3a66f1936e`) : la fiche du joueur du rejeu (section « En passant ») montre un groupe « Gestes » : mêlée, lancers amorcés, accroupis, saut au bouton ; le contrat servi porte `gestures`. |
| 2026-11-25 | **[rejeu/interface] Rationaliser les réglages d'affichage du rejeu (calques et informations sur la carte)** — traité (plan `.ai/PLAN_BACKLOG_2026-11-24.md`, M.1 à M.4) : le panneau de réglages est refait sur la maquette validée par le user le 2026-11-24 (variante A, rubriques en onglets) : cinq rubriques, préréglages souples qu'on ajuste librement, « Mes vues », accès rapide épinglable, « Prendre le point de vue » depuis la fiche. Contrôle à l'écran dû (artefact des vérifications). |
| 2026-11-25 | **[rejeu/dégâts] Auteur d'un dégât lu sur l'index brut du film** — déjà fait (plan `.ai/PLAN_BACKLOG_2026-11-24.md`, 0.2) : l'arme d'une touche se cherche sous le tireur publié depuis le décodeur v1.6.0 ; sur le parc régénéré, 86,6 % des touches ont une arme avant et 89,5 % après, et les touches sans tireur passent de 79 457 à 1 419 (annexe de mesures du décodeur, relue, non remesurée). |
| 2026-11-25 | **[rejeu/killfeed] Zone de callout du tueur dans le killfeed** — déjà fait (plan `.ai/PLAN_BACKLOG_2026-11-24.md`, C.1) : livrée en 8.2.0 ; le fil affiche depuis le lieu de la victime, par décision de la refonte de la page du rejeu. |
| 2026-11-25 | **[rejeu/schéma] Deux films du parc avec des faits d'un ancien schéma** — sans objet (plan `.ai/PLAN_BACKLOG_2026-11-24.md`, 0.2) : les deux matchs cités n'ont plus rien sur le poste (ni rejeu, ni film, ni faits) et les 1 455 rejeux du poste sont tous au schéma 127. |
| 2026-11-25 | **[films/prod] Le calcul des frags archive chaque film téléchargé sans lire le réglage « archiver les films »** — traité (plan `.ai/PLAN_BACKLOG_2026-11-24.md`, D.1, `a1990f6270`) : le calcul des frags n'écrit plus rien dans l'archive des films (le « relais » d'archivage, qui n'évitait aucun second téléchargement, est retiré) ; le site hébergé n'archive plus les films par défaut (capacité `ArchiveFilmsByDefault`) et l'application Windows continue d'archiver. Reste au déploiement : lire deux clés du fichier de réglages du serveur, puis retirer la purge horaire (`.ai/gestes_prod_8_3_0.md`, A1 à A3). |
| 2026-11-25 | **[hygiène/tests] Trois noms de tests portent encore un pseudo de joueur** — traité (plan `.ai/PLAN_BACKLOG_2026-11-24.md`, E.4, `389c8731ca`) : trois noms de tests sont renommés avec leurs 13 lignes de la référence des tests. |
| 2026-11-25 | **[backlog] Puces de la section « 2026-11-23 — Découvertes du plan du backlog des 17 et 18 novembre »** — traité (plan `.ai/PLAN_BACKLOG_2026-11-24.md`, pistes A à E) : écriture concurrente du catalogue de cartes (A.6 : un accès refusé sur le verrou est un verrou tenu ; l'instabilité d'origine n'a pas été reproduite, cause déduite) ; ordre instable des « rencontres » (B.2) ; trois points du coup d'envoi (A.4) ; match « film absent » alors que son film est archivé et coup d'envoi après la fin du match (A.3 : marque retirée des films complets, 34 sur 616 ; fenêtre du match bornée) ; avertissement de mise en file (D.3) ; `medal_name` du contexte du match (B.5, l'écriture dans `raw_json` reste au backlog) ; doc de `replay-corpus-gate` (E.2) ; garde des jetons figé sur 2026 (E.3) ; fichiers au-dessus du seuil et quatre mesures de largeur du rejeu (C.5) ; test au nom de fichier à octet nul : **déclaré introuvable** (E.3, non reproduit sur ce poste ni localisé dans le dépôt). |
| 2026-11-25 | **[backlog] Puces des sections « 2026-11-22 — Découvertes du décodeur v1.5.0 » (vague 2 et recherches)** — traité (plan `.ai/PLAN_BACKLOG_2026-11-24.md`, B.4, E.2, F.1) : tableau « Impact des coéquipiers » par session (B.4 : bloc non publié sans équipe alliée lue, colonne de match à 44 px au moins, à juger à l'écran, sept ré-exports sans lecteur retirés) ; en-tête faux de `cmd/killsource` (E.2) ; voisinage des pièces montées (F.1 : sur le parc au schéma 127, 2 619 pièces montées, 2 614 porteurs lus dans le film, 0 à plus de 15 m, repli par voisinage jamais déclenché). |
| 2026-11-25 | **[backlog] Puce de la section « Confidentialité du dépôt public (2026-11-23) »** — traité (plan `.ai/PLAN_BACKLOG_2026-11-24.md`, E.5, `3fe2325280`) : `seed-demo --emit-manifest` n'écrit plus l'identifiant réel du joueur source : champ retiré de l'émission et des deux manifestes, aucun code ne le lisait. |
| 2026-11-23 | **[outillage] `scripts/disk-hygiene.ps1` : contrôle « fusionné » vers une branche disparue** — traité (plan `.ai/V8.3/PLAN_BACKLOG_2026-11-18.md`, K.3) : base résolue (amont de la branche courante, sinon branche par défaut du dépôt), paramètre `-BaseRef`, « base introuvable » journalisé sans arrêt. |
| 2026-11-23 | **[rejeu/cadres] Rangée de fiches et colonne qui oscillent au seuil de largeur** — traité (plan du 2026-11-18, C.1) : marge de 24 px entre « la rangée tient » et « elle ne tient plus », même marge pour l'emplacement du score. |
| 2026-11-23 | **[rejeu/cadres] Plein écran du navigateur laissé actif après le retour au Standard** — traité (plan du 2026-11-18, B.3) : la sortie d'un Plein cadre né du geste rend le plein écran natif par tous les chemins ; un plein écran refusé ou absent ne bloque plus le geste suivant. |
| 2026-11-23 | **[rejeu/fiche] La fenêtre de fiche relit tout le lobby à chaque changement de joueur** — close sur mesure (plan du 2026-11-18, D.6) : la lecture des séries coûte 2,2 à 4,3 ms en médiane, 13,4 ms au pire, sous le seuil de 50 ms ; rien à corriger. |
| 2026-11-23 | **[rejeu/fiche] Clé de liste par nom d'arme dans « Tirs et dégâts »** — traité (plan du 2026-11-18, D.2) : une ligne par nom d'arme, familles de même nom additionnées, clé de la première famille du groupe. |
| 2026-11-23 | **[rejeu] Soupçons de la relecture finale, non démontrés** — les huit sont statués (plan du 2026-11-18) : orientation Auto dépendante du match précédent et pendant un export, démontrée et corrigée (C.2) ; horloge de la barre en pause, test qui passe (C.3) ; lâcher daté à fin de vie + 2, 0 cas sur 28 540 lâchers, non traité (D.5) ; deux scores à l'instant, un seul score (D.7) ; barre d'une ligne sur écran étroit (M.2) ; rareté d'une médaille à l'ordre des langues fixe (G.3) ; réapparitions comptant une vie d'avant le match (D.1). |
| 2026-11-23 | **[rejeu/fiche] Cinq cotes de l'ancien gabarit de fiche sans lecteur** — traité (plan du 2026-11-18, D.4) : `GABARIT_NORMAL` retiré, une cote nommée pour le corps du bloc « À l'instant », garde-rail contre son retour. |
| 2026-11-23 | **[tactique/orientation] La Tactique et « Occupation du terrain » ne suivent pas l'orientation « Auto » du rejeu** — traité (plan du 2026-11-18, C.4, décision D-3) : les deux vues appliquent l'Auto sur leur propre zone ; un choix retenu pour la carte prime. |
| 2026-11-23 | **[rejeu] Point de vue du rejeu gardé d'un match à l'autre** — traité (plan du 2026-11-18, B.2) : le point de vue est rangé avec son match et repart de zéro. |
| 2026-11-23 | **[rejeu/fiche] « Suivre » un joueur et « Seul » sur un autre à la fois** — traité (plan du 2026-11-18, B.1, décision D-2) : « Seul » éteint le suivi d'un autre joueur, et suivre un autre joueur lève « Seul ». |
| 2026-11-23 | **[rejeu] Nom de grenade de repli écrit en français dans la logique** — traité (plan du 2026-11-18, D.3) : « rang N » passe par les dictionnaires, en français et en anglais. |
| 2026-11-23 | **[rejeu/médailles] Événements de médaille du contexte du match : trois champs sans lecteur** — traité (plan du 2026-11-18, G.1, décision D-7) : `medal_label`, `medal_description`, `medal_image_url` et `medal_name_id` retirés, `MedalImageURL` retiré des trois adaptateurs de titre ; reste `medal_name`, voir la section du 2026-11-23. |
| 2026-11-23 | **[rejeu] Médailles du fil du rejeu pour un titre à planches d'images** — traité (plan du 2026-11-18, A.3) : le fil et les pastilles dessinent une médaille depuis une découpe de planche, comme la fiche. |
| 2026-11-23 | **[rejeu] Clé de requête du contexte du rejeu sans la langue** — traité (plan du 2026-11-18, H.4) : `queryKeys.matchReplayContext` porte la langue. |
| 2026-11-23 | **[web/couches] Garde-rail des couches : périmètre et faux négatifs** — traité (plan du 2026-11-18, H.1 et H.2) : le garde-rail balaie `components/`, `lib/` et les `.ts` de `features/`, reconnaît un nom d'événement non littéral et `case 'Escape'` ; treize composants migrés vers le mécanisme commun, sept exemptions datées. |
| 2026-11-23 | **[sécurité] Jetons : `haloTokensForDrain` prend le premier jeton valide** — traité (plan du 2026-11-18, E.1 et E.2) : le jeton vient de la source de lecture publique de la mise en commun ; sans compte consentant, erreur typée journalisée, sans conseil de recapture ; garde-rail dans `internal/api/wire`. |
| 2026-11-23 | **[rejeu] Régénération du bureau : fichier tactique d'un rejeu au film expiré** — traité (plan du 2026-11-18, F.8, décision D-5) : la file reprojette, sans film, le fichier tactique périmé ; le contrôle de péremption ne lit que l'en-tête (0,25 s au lieu de 27 s pour 1 455 fichiers). |
| 2026-11-23 | **[tests/durée] Paquet DuckDB : migrations neuves indirectes** — traité (plan du 2026-11-18, S.4) : les cinq sites qui partaient d'un fichier joueur absent reçoivent la base migrée copiée ; le garde-rail reconnaît `GetOrOpen` et une fonction de migration prise en valeur. |
| 2026-11-23 | **[web/couleurs] Contrôle des couleurs : entrées mortes de la liste des fichiers autorisés** — traité (plan du 2026-11-18, K.1) : trois entrées retirées (et non deux), test « chaque chemin de la liste existe ». |
| 2026-11-23 | **[données] Escouade : tous les comptes d'un coéquipier inconnus** — traité (plan du 2026-11-18, G.2, décision D-4) : les trois taux par minute sont facultatifs, « inconnu » sans match retenu. |
| 2026-11-23 | **[fonds de carte] Limites du rempart contre les chemins du poste** — traité (plan du 2026-11-18, K.2) : racine d'installation connue retirée en plus du motif, dossiers à point et à espaces reconnus, garde-rail étendu aux partages réseau et aux chemins Unix hors dossier personnel. |
| 2026-11-23 | **[rejeu] Compteur des rejeux illisibles** — traité (plan du 2026-11-18, G.4) : un fichier tactique illisible a son compteur et son message. |
| 2026-11-23 | **[rejeu] Rejeu à la demande : restes de la relecture du 2026-11-17** — traité (plan du 2026-11-18, F.1 à F.7 et F.9) : garde des jetons en liste blanche, demande reprise revalidée, contrôles d'état avant un nouvel essai, film retiré après un échec définitif, capacité limitée au bureau et au développement non exposé, bouton absent quand le jeton demande une reconnexion, jeton refusé retenté avec un jeton frais ; verdict « film absent » de la synchronisation mesuré (0 match sur 251 vus moins de 5 minutes après leur fin) : défaut écarté, fenêtre de 5 minutes posée côté clic seulement. |
| 2026-11-23 | **[web/couches] Accessibilité des menus** — traité (plan du 2026-11-18, H.3) : le sélecteur de média écoute Échap par le mécanisme commun ; les menus d'expérience et de vue annoncent leur état ouvert. |
| 2026-11-23 | **[archi] `bootstrap_service.go` au-delà du seuil de taille** — traité (plan du 2026-11-18, K.5) : trois blocs déplacés sans renommage, fichier sous 500 lignes. |
| 2026-11-23 | **[rejeu/horloge] Coup d'envoi estimé depuis l'API : décalé de l'horloge du film** — traité (plan du 2026-11-18, piste T, ADR 0050) : une définition, une résolution (film, puis estimation de l'API, puis inconnu), une écriture, une commande `levelup backfill-kickoff`. Reste à faire au premier démarrage de la version : le rattrapage du parc, voir `.ai/REGISTRE_REPORTS.md`. |
| 2026-11-23 | **[rejeu] Barre de lecture en surimpression en Cinéma et en Plein cadre** — traité (plan du 2026-11-18, M.1 à M.3, choix B + C du user) : barre sur deux lignes quand la frise n'a plus sa largeur minimale, barre fine et translucide posée sur le plan au Plein cadre. Contrôle à l'écran du user dû avant `main`. |
| 2026-11-17 | **[tests/durée] Paquet `internal/platform/duckdb` au-delà de son budget** — traité : base migrée copiée par test, 1 259 tests réussis, 2 sautés, 191 s ; garde-rail contre la migration neuve dans les tests. |
| 2026-11-17 | **[tests/local] `internal/platform/duckdb` dépasse 300 s sous `-tags=integration` en local** — doublon de l'entrée « Paquet `internal/platform/duckdb` au-delà de son budget » (même paquet, même symptôme). |
| 2026-11-17 | **[tactique/disque] Compresser les fichiers tactiques `rasters/*.json`** — traité : écrits et lus en `.json.gz` ; l'ancien `.json` n'est plus lu et est supprimé à la réécriture et par les purges. |
| 2026-11-17 | **[perf/rejeu] Probabilité de réapparition : tous les calques tactiques de la carte relus à chaque ouverture d'un rejeu** — traité : cache par fichier, relu quand la date ou la taille change (39 fichiers : environ 320 ms sans cache, 3 ms avec). |
| 2026-11-17 | **[rejeu] Générer à la demande un rejeu absent** — traité pour l'application de bureau et le poste de développement ; l'instance hébergée garde son action d'administration. |
| 2026-11-17 | **[web/couches] Fermeture des couches par Échap ou clic dehors : migrer les 17 écrans restants vers le hook partagé** — traité : 12 écrans migrés vers le mécanisme commun, 5 exemptés avec une raison datée dans le garde-rail. |
| 2026-11-17 | **[web/couleurs] Reliquats du garde-rail des couleurs** — traité : 19 marqueurs sans effet retirés, dégradés et ombres en couleur nommée refusés, marqueur sans effet refusé. |
| 2026-11-17 | **[rejeu/gate] Témoin `f75e7053` hors d'atteinte du gate du corpus** — traité : remplacé par `0a247154` dans `config/replay_corpus.toml`. |
| 2026-11-17 | **[docs/adr] ADR 0024 (LUSR v2) en partie périmée** — traité : 500 itérations, limites 4 à 6 marquées livrées, liens rétablis, amendement daté. |
| 2026-11-17 | **[fonds de carte] Carte Forge au maillage de navigation illisible : comprendre pourquoi** — close : aucune carte concernée, les 92 maillages du poste se lisent. Thunderhead et Thunderhead Heavies : close, la carte ne publie aucun maillage (vérifié le 2026-11-17) ; textes corrigés. |
| 2026-11-17 | **[décodeur] Règles de la variante : délai de retour du drapeau absent du parc** — close : le champ naît au schéma 125, le parc mesuré était au 124. |
| 2026-11-17 | **[rejeu/postures] Le document de rejeu porte déjà les sprints et les sauts** — textes corrigés le 2026-11-17 ; l'affichage est un item de la refonte de la page du rejeu (section « Gestes » de la fiche du joueur). |
| 2026-11-17 | **[rejeu/médailles] Médailles absentes du rejeu dans 11,9 % des matchs** — rattachée le 2026-11-17 à la refonte de la page du rejeu : les médailles du fil et de la fiche se liront dans le document de rejeu. |
| 2026-11-16 | **[rejeu] Croix des vraies morts sans cause** (8.2.1) : la croix se pose quand le film prouve la mort (lâcher de commande ou compte à rebours du joueur, de -2 à +6 images) ; mesure sur les 1 455 rejeux : 0 croix à une fin de manche, 0 à la fin du match, accord avec la feuille 99,3 / 100 / 99,5 %. |
| 2026-11-16 | **[campagnes] Défis de l'axe Objectif : départ non mesuré, Halo 5 sauté par capability au recalage, étapes notes et campagnes indépendantes** (commit 2b507b8c77, release 8.2.1). |
| 2026-11-15 | **[logs] Avertissements répétés pendant un long recalcul** (plan `.ai/PLAN_COMPLEMENTS_V8_2_0_2026-11-14.md`, étape 1) : un dépassement de seuil du writer ou d'une lecture se dit une fois par épisode (franchissement, second seuil de 30 s, libération avec durée et lectures retenues), repli du solveur LUSR résumé en une ligne par rejeu avec sa cause, pannes 5xx des services Halo en une ligne par passe, 429 absorbés en INFO. |
| 2026-11-15 | **[web/couleurs] Infobulle `info-tooltip` au focus clavier seulement, garde-rail de couleurs étendu** (même plan, étape 5) : couche globale des infobulles (survol et focus clavier, visibles en partage d'écran), `ring-offset`, `outline`, `decoration`, `accent`, `caret` et portée de `color-allow` ; reliquats au backlog. |
| 2026-11-15 | **[kpi/notes] Vie d'un Spartan à 229 points, note de performance des modes à objectif** (plan `.ai/PLAN_COMPLEMENTS_V8_2_0_2026-11-14.md`, étape 7, branche `feat/c820-note`) : mesure sur 1 614 films (compteur « dégâts subis », 229 dans 99,6 % des vies sans recharge de bouclier, 230 dans 0,4 %), `games.DefaultEffectiveHpToKill` et `effective_hp_to_kill` à 229, aide FR/EN ; note à objectif (paquet `internal/sync/objectiveblend` : moyenne symétrique du combat, de l'objectif et du soutien de l'équipe, rangs à mi-rang contre l'historique du joueur dans la famille de mode), métrique `objective_participation` retirée ; `PerformanceScoreRev` et `LUSRRev` montées ensemble (un seul recalcul, test D-14), ADR 0047 et 0006 amendées, notes de version et changelog FR/EN avec les écarts mesurés (médiane +0,6, 90e centile 11,8, 32 % de paliers changés, après le repli de famille du soutien). |
| 2026-11-15 | **[rejeu] « Seul » et « Masquer » sur tous les calques** (plan `.ai/PLAN_BACKLOG_2026-11-15.md`, lot 4, branche `feat/bl1115-killfx`) : règle du user (tout ce qui appartient au joueur suit la fiche, le décor reste) ; quatorze calques ou éléments branchés sur un prédicat unique (`ownershipShown`), dont les effets de mort par tueur ET victime ; inventaire `.ai/V8.3/annexes_backlog_1115/calques_seul_masquer.md`. |
| 2026-11-15 | **[analysis] Cadences par minute par `analysis.PerMinute`** (plan `.ai/PLAN_BACKLOG_2026-11-15.md`, lot 1, branche `feat/bl1115-minute`) : quatorze sites (et non six) migrés dans `analysis`, `service` et `sync`, valeurs identiques au bit (témoins avant/après en assertions), garde-rail AST `archlint/no_inline_per_minute_test.go` à allowlist vide. |
| 2026-11-14 | **[release] LevelUp 8.2.0** (plan `.ai/PLAN_V8_1_2_2026-11-09.md`, branche `release/v8.2.0`) : plan 8.1.2 exécuté (tous les points du backlog du 2026-11-09 sauf 17, 18, 24 et 25 : barre latérale, volet des référentiels, sélecteur de dossier, zone du tueur au killfeed, FDA nette, alertes localisées, notes des matchs non convergents, icônes du jeu, nettoyage de l'installeur, licences et identités fictives ; PR #181, #183, #184, #185, #186, #187, #189, #190), correctif 8.1.2 de synchronisation intégré (#193) ; plan d'architecture en 8 étapes (suppressions, interrupteurs, écritures, couche HTTP, stockage, multi-titre, front, clôture ; PR #191, poussé sur la release pour le reste) ; briques consommées comme modules publiés (`github.com/JGtm/...` en v1.0.1, `libs/` et `go.work` supprimés, paquet `internal/briques` ; PR #194, #201, #202, puis montées en v1.1.0 par la #205 et au décodeur v1.3.6 par la #207) ; rejeu à 30 images par seconde (#190, #207), rejeux compressés (#199), lecture complète des paquets du film (#198), fil filtrable, fiche en panneau et Mainmise (#200), destructions de véhicule et occupants tués (#203), fonds de carte tenus par le maillage de navigation (#205), alertes de sécurité des dépendances (#206). Relecture finale par trois relecteurs sans le plan : 0 P0, 0 P1, 3 P2 corrigés. Restent les gestes du user : contrôles à l'écran, fusion vers `main`, gestes de production. |
| 2026-11-08 | **[archi] Décodeur de film : brique `libs/film` publiable** (plan `.ai/PLAN_FIN_DECOUPLAGE_BRIQUE_FILM_2026-11-02.md`) : réorganisation protocole/exploitation, identifiants fictifs dans les fixtures, README/NOTICE/exemple, ADR 0043 amendée, revue adversariale close (branche `refactor/brique-film-publication`, PR vers `release/v8.1.0`). Reste : tag `libs/film/v0.1.0` (geste du user), lot 5 (jeton Spartan sur redirection, NOTICE gamefiles) et lot 6 (quatre dépôts dédiés) après le bureau et `main`. |
| 2026-11-08 | **[backlog] Lot du 2026-11-08** (plan `.ai/V8/PLAN_BACKLOG_2026-11-08.md`, branche `chore/backlog-2026-11-08`) : matchs exclus retirés du graphe de carrière et des composantes LUSR, rejeu LUSR complet en échec visible si une écriture est tenue ; moteurs servis par le provider de leur titre, profils `auth_only` écartés du recalcul de sessions, ratchet des slugs étendu à `cmd/` ; page Citations sans barre de filtres, noms sur deux lignes, erreur de règles remontée, citations réconciliées à chaque démarrage ; rafraîchissement OAuth par le dernier client ayant réussi (mémoire du processus) ; `reset-bitmasks` sur les constantes canoniques, relance de connexion gardée après succès, deux instruments sortis du corpus `gamefiles`. Clos sur preuve sans code, détail au §0 du plan : `weapon_resolver`, `match_bomb_stats`, `cmd/levelup` et étapes par titre, maillon de nom Halo 5, fixture `psa_index_repro`, lecture du précédent LUSR, filtre CI, `RunForDB`, passes CSR, première vérification de participation, `/tactical/maps`, banc Cliffhanger ; instruments `research` traités par la brique film. Revue indépendante du plan, relecture dédiée de l'authentification, relecture finale Opus. |
| 2026-11-02 | **[qol] Citations `.ai/` retirées des commentaires de code** (PR #134, entrée de backlog ouverte par la PR #127) : nettoyage fait, ratchet à plafond 0. Même PR : moteur des véhicules au mouvement dans le rejeu, tuile seule des sessions récentes. |
| 2026-11-01 | **[archi] Publication des briques : modules, README, tags** (plan `.ai/V8/PLAN_PUBLICATION_BRIQUES_2026-10-30.md`, PR #118 et #119 dans `release/v8.1.0`) : module `libs/haloapi` multi-jeu (`xblauth`, `infinite`, `halo5`, `xboxlive`, `haloapitest` ; clients Halo 5, UGC cartes, GameCMS et succès Xbox coupés en protocole / exploitation sans changement observable), README EN des trois briques, `example_test.go`, ADR 0043 ; tags `libs/{gamefiles,himap,haloapi}/v0.1.0` posés sur validation du user, `go get` vérifié hors dépôt. Reste : dépôts dédiés (entrée du backlog ci-dessus). |
| 2026-10-29 | **[archi] Découplage des briques D1, D2, D3** (audit `.ai/V8/AUDIT_DECOUPLAGE_BRIQUES_2026-10-27.md`, ADR 0041 et 0042) : `libs/gamefiles` + `libs/himap` reliés par `go.work` (PR #111), décodeur de film en module `libs/film` (PR #113), API Halo séparée en protocole et exploitation (PR #114) ; suites (gates locaux, ratchets, robustesse `haloapi`) en PR #117. |
| 2026-10-28 | **[i18n] N langues (ADR 0040)** : source unique `config/locales.toml`, manifests générés côté web et Go, libellés résolus par le serveur dans la langue de la requête, garde-rails des deux côtés ; fusionné dans `release/v8.1.0` avec la revue UI du 2026-10-28 (rangs déduits du code, défis reconnus dans une autre langue). |
| 2026-10-18 | **[données] Indicateur « données de compétence » des 157 matchs synchronisés du 25 au 30 mai** (plan `.ai/archive/V7.5/PLAN_BACKLOG_2026-10-16.md`, après E10, sur décision du user) : la donnée était en base (MMR d'équipe présent ; joueurs suivis concernés : JGtm 137 matchs, joueur_165 87, joueur_117 75, joueur_291 10, joués de nov. 2021 à mai 2026), seul le bit skill (4) manquait. `levelup reset-bitmasks` serveur arrêté, base sauvegardée avant (`data/backups/avant_reset_bitmasks_2026-10-18/`) : 157 lignes `match_registry` et 1 302 lignes `match_participants`, dry-run rejoué = 0, `levelup healthcheck` tout OK. Les bases locales étant copiées vers le VPS à la release, rien à faire en prod. |
| 2026-10-18 | **[déploiement] Recalcul des dérivés Halo 5, LUSR v2 de tous les joueurs, séquence d'après déploiement** (plan `.ai/archive/V7.5/PLAN_BACKLOG_2026-10-16.md`, E9, branche `feat/backlog-2026-10-16`) : (1) `levelup backfill --title <slug>` est honoré par `--engagement-scores`, `--citations`, `--citations-recompute-all`, `--composite-only`, `--lusr` (v2), `--perf`, `--assists-model` et `--compare-formulas` (un pilote et une table de passes remplacent six paires de boucles recopiées ; `--all` prend les profils du titre, `--csr`/`--shared-csr` refusent un autre titre, migrations de la base DU titre) ; sur Halo 5 les passes de citations répondent « non supporte » par la capability `citations.engine`, sans rien ouvrir ; `POST /backfill/start` prend un champ `title` (404 si le joueur n'est pas suivi sur ce titre), construit le moteur et le provider shared DU titre (`config.SharedProviderForTitle`, aussi utilisé par le replay LUSR de l'admin, qui passait le provider d'Infinite à un joueur Halo 5). (2) `POST /settings/sessions/recalculate` recalculait CHAQUE profil sur les bases du titre par défaut (titre vide) : un profil Halo 5 n'était jamais recalculé ; corrigé par profil, test rouge avant. (3) « Dates de paliers » : source `cmd/backfill_milestone_dates` (hors image de prod, décision au runbook). (4) Runbook : section « Post-deploy recomputes (v7.5) » (R1 à R10 : 157 matchs, dérivés Halo 5, LUSR v2 des deux titres, séquence film, sessions, accords du pool, re-seed de la démo), commandes et routes exactes. Rien n'est exécuté : geste du user. |
| 2026-10-18 | **[rejeu] Lot film : positions du coup fatal, score KOTH Lattice, Harvest, vies sans nom** (plan `.ai/archive/V7.5/PLAN_BACKLOG_2026-10-16.md`, E8, branche `feat/backlog-2026-10-16`) : (1) positions : la passe lit le journal fusionné par xuid (joueurs renommés), tient la dernière position dans la même vie, lit le siège du véhicule et place le tueur déjà mort à sa mort ; le contexte de mort lit la victime par la même lecture que `kill_positions` (source unique, tolérance de 0,5 m retirée) : tueurs sans position 1 476 -> 78 et victimes 1 028 -> 85 sur 233 films (22 535 frags), 2 contextes de mort sur 23 326 non placés contre 148 gagnés. (2) vies sans nom : 449 -> 15 sur les 36 films en cause (index des chunks hors table, chaîne de contradictions, corps de bot arrivé avant sa première déclaration), aucune vie nommée perdue sur les 2 660 films comparés. (3) score KOTH Lattice : une émission hors domaine (canal B non nul d'un slot d'équipe) remplaçait le point réel par 0 et le seuil de garde retenait un intervalle fusionné : `7de0b91d` rend 40 sans contradiction, `26602661` sans descente, 6 films changent de série, les autres KOTH inchangés. (4) Harvest : carte Forge, canevas `fo11_blank` prouvé par le level_id de la variante (426470249, unicité 1/1), catalogues à 85 cartes (bornes) et 131 (objectifs), `e449a696` et `99b8ba5c` cuisent (158 et 144 pistes, 0 vie sans nom). Révisions montées une fois : `objectives-2026-10-17`, `isolement-2026-10-17-positions`, schéma 99, `placement-2026-10-17`, `grammar-2026-10-18`, `killsource-2026-10-18`. Gate : `replay-corpus-gate` 19/19, 0 perte. Reste après la fusion (serveur arrêté, puis prod, le user prévenu) : `backfill-replay --only-existing`, `backfill-killsource`, usage-summary, `backfill-pad-tiers --force` ; contrôle visuel du user en lecture sur `7de0b91d` et `e449a696`. |
| 2026-10-17 | **[ui] Indicateur « Analyse en cours » (tuile + vue match)** (maquette validée le 2026-10-16) : pastille sur l'image de carte de la tuile, emplacements en attente (frags assistés, citations, rejeu), bandeau dans la vue match ; état donné par le serveur, rechargement tant que l'analyse court (`components/ui/analysis-pending-*.tsx`, `match-card-analysis-pending.tsx`). Fusionné dans `feat/v75`, publié avec la v8.0.0. |
| 2026-10-17 | **[perf/tactique] Chargement de la page Tactique : de 96 requêtes à 5 hors images** (plan `.ai/archive/V7.5/PLAN_BACKLOG_2026-10-16.md`, E7, branche `feat/backlog-2026-10-16`) : le chargement coûtait 2 requêtes par carte ouvrable (le calage `GET /tactical/{map}/background`, puis l'image `background.png`) en plus de 4 requêtes fixes, soit 96 pour 46 cartes. `/tactical/maps` porte désormais le calage du fond de chaque carte ouvrable (`calage_fond`, absent pour une carte sans fond figé ; même source que la route par carte ; panne journalisée sans arrêter la grille) ; la vignette lit le cadre du fond dans la carte (plus aucune requête de calage par vignette) et ne demande son image qu'une fois entrée dans le champ de vision, et seulement si la carte a un calage (hook `useVisibleUneFois`, même clé de cache que le plan principal). Repli de D-10 : pas de `<img loading="lazy">`, car un `<img>` n'emporte pas `X-LevelUp-Title` et l'API résoudrait le titre de la session. Restent 5 requêtes de données (4 fixes + le calage de la carte du plan) puis les images des vignettes visibles. Mesure établie par le code et un test à 46 cartes ; la mesure en navigateur (onglet Réseau, temps d'affichage) reste au contrôle visuel. |
| 2026-10-17 | **[dette/citations] Premiers frags du profil, capability des citations, maîtrise Carabine à impulsion** (plan `.ai/archive/V7.5/PLAN_BACKLOG_2026-10-16.md`, E6, branche `feat/backlog-2026-10-16`) : (1) `computeFKFD` lisait des colonnes absentes de `highlight_events` et avalait l'erreur : premiers frags et premières morts à 0 pour tous les titres ; il passe par le chargeur unifié des événements (repli sur le journal des frags pour Halo 5), `analysis.FirstBlood` et le nouveau `analysis.FirstDeath`, sur les matchs du joueur, Campagne exclue, erreur journalisée ; tests Infinite et Halo 5 rouges avant, verts après. (2) `citations.engine` passe à `supported` pour Infinite (TOML, repli de l'adapter, parité verte ; seul lecteur : la matrice des fonctionnalités admin). (3) Diagnostic de la maîtrise Carabine à impulsion absente de la page Citations : aucun défaut de code (ligne présente et active au seed et dans les copies à froid de la metadata, chaîne service/page rejouée sur copie, image suivie par git, les 32 maîtrises `weapon_kills:` du seed se résolvent) ; reste à lire l'état de la metadata de l'instance regardée (requête au journal du plan) et, au besoin, `levelup seed citation-mappings`. Storm : aucune citation ajoutée (l'équivalent Infinite est la Carabine à impulsion). |
| 2026-10-17 | **[dette] Dettes Go du plan du 2026-10-15** (plan `.ai/archive/V7.5/PLAN_BACKLOG_2026-10-16.md`, E5, branche `feat/backlog-2026-10-16`) : code mort supprimé avec ses tests et lignes de baseline (`FindMatchesMissingParticipantBits`, `loadAwardsForScoreboard`, `WithAwardsRepo`, `CachedHighlightEventsRepo`, `BatchBuilder.AddLUSRComponents`, paramètre `titleSlug` de `SquadUsualContexts` remplacé par `prestige.Deps.TitleSlug`, 14 réexportations de constantes) ; erreurs avalées journalisées avant la dégradation (libellés d'armes, image de carte, modes connus, nom de paire, assistances attendues) et `ApplyResolutionViews` remonte l'échec de `v_match_full` ; DDL de `v_match_full` en source unique `migration.VMatchFullViewSQL` + garde-rail archlint ; une requête abandonnée ne bascule plus en lecture live ni ne compte ni n'écrit d'ERROR sur la vue match ; garde de la Campagne étendu au registre lu sur fenêtre de temps (2 dispenses d'outils opérateur) ; `reset-bitmasks` sur `matchflags.BackfillParticipants`, mapper OpenSpartan honnête sur les médailles (`EventsLoaded` supprimé), `--title` sur 3 CLI et passage du slug à `backfill-killsource` ; `match_bomb_stats` exporté dans le snapshot (un cut ancien reste lisible) ; un ami suivi sans player DB est une absence et non plus une panne des extras. Reste : le bit skill de 157 matchs (geste de déploiement, runbook). |
| 2026-10-17 | **[data/LUSR] Le recalcul après exclusion d'un match utilisait encore LUSR v1** (plan `.ai/archive/V7.5/PLAN_BACKLOG_2026-10-16.md`, E4, branche `feat/backlog-2026-10-16`) : la v1 du LUSR est supprimée (batch, étape de post-sync, aperçu `levelup backfill --lusr --dry-run`, interrupteurs `LEVELUP_LUSR_V2_ENABLED` et `LEVELUP_LUSR_CANONICAL`, `RecomputeAfterARTRebuild`, tests v1). L'exclusion d'un match rejoue la chaîne v2 du joueur (sentinelle de reset puis rejeu ; l'écrivain partagé est pris par rafales courtes, jamais en continu) et la v2 saute les matchs `is_excluded`. Les 8 composantes (`lusr_component_history`, lues par Prestige, le profil et l'escouade), écrites jusque-là par la seule v1 donc figées depuis la bascule canonique, sont écrites par la v2 : même calcul, parité vérifiée valeur à valeur (275 valeurs identiques à 1e-12). Constat corrigé : la vue `match_skill_rank_latest` ne laissait pas la v1 masquer la v2 (`start_time` avant `written_at`), mais le graphe d'évolution (`match_skill_rank_latest_by_type`) servait bien les lignes v1. Mesure sur une copie à froid de JGtm (sauvegarde du 16/10) : composantes 7 137 lignes pour 1 028 matchs, dernière ligne du 18/07, 125 matchs notés sans composantes, puis après rejeu 8 011 lignes pour 1 153 matchs et 0 note sans composantes (rejeu complet 47 s'en 24 rafales). Reste : le simulateur de formules `levelup backfill --compare-formulas` (TrueSkill v1, n'écrit rien) est conservé. |
| 2026-10-17 | **[ops/arrêt] Séquence d'arrêt du serveur bornée sous le délai de Docker** (plan `.ai/archive/V7.5/PLAN_BACKLOG_2026-10-16.md`, E3, branche `feat/backlog-2026-10-16`) : `stop_grace_period: 45s` sur `levelup` et `levelup-demo` ; budget global de 40 s depuis la demande d'arrêt (5 s gardés pour la fermeture des bases), chaque phase d'attente prend la plus petite de sa borne et du budget restant, y compris `watcherDaemon.Stop`, `BatchQueue.Close`, `workerWG.Wait` et le gate (jusque-là non bornés ou hors chronomètre) ; `slog.WarnContext` nommant la phase qui dépasse ; `duckdb.CloseAll` toujours atteint. Séquence extraite de `main()` dans `cmd/server/shutdown.go` + `shutdown_budget.go`. Tests : budget sous la grâce lue dans `docker-compose.yml` (rouge prouvé sans la clé), phases bloquées, ordre HTTP, recalculs, écrivains, fermeture. Commentaire périmé (`stop_timeout=20s`) supprimé. |
| 2026-10-17 | **[sécurité/suites] Code Xbox redemandé après une connexion réussie, et vérification de participation en 503** (plan `.ai/archive/V7.5/PLAN_BACKLOG_2026-10-16.md`, E2, branche `feat/backlog-2026-10-16`) : (1) `XboxLoginPage` ne relance plus `start` sur `attempt_not_found` une fois la tentative autorisée (la rotation de session au succès rend l'ancienne tentative introuvable) et le sondage `useDeviceFlowStatus` s'arrête dès `authorized`/`provisioned` (prédicat `isDeviceFlowSucceeded`, `setup/queries.ts`) ; test Vitest rouge avant le correctif (5 sondages au lieu de 1). (2) `RequireMatchParticipation` : cache borné (4 096 entrées, le plus ancien sort) des participations CONFIRMÉES seulement, clé titre + xuid + match, ni `false` ni erreur mémorisés, 503 fermé conservé sur échec non caché (`api/wire/participation_cache.go`, câblé dans `MatchParticipation`) ; tests : 2 appels = 1 requête, clé, borne, bout en bout avec le middleware. La première vérification d'un couple reste soumise au budget de lecture. |
| 2026-10-17 | **[ci/durée] CI raccourcie de 49 à ~16 min, mêmes tests** (plan `.ai/archive/V7.5/PLAN_BACKLOG_2026-10-16.md`, E1, branche `feat/backlog-2026-10-16`) : suite Go de couverture en 6 lots sur 6 machines, chacun en `-p 1` (définition unique `.github/ci/go-shards.txt`, dernier lot = complément calculé, `grammar` coupé en deux par une liste nommée de tests lourds `-run`/`-skip`), job de fusion (contrôle 356 paquets sur 356, gate baseline, profils additionnés, ratchet de couverture : 81,0 % identique) ; second passage `internal/sync` supprimé ; Vitest en 3 lots fusionnés ; `concurrency` (runs périmés annulés sur les branches de travail, groupe propre au run sur `main`/`feat/v75`) ; filtre de chemins sur les branches de travail (base `feat/v75`) ; `-timeout` revenu à 600 s ; `make gate-push` à deux voies (film sans DuckDB en parallèle). Garde-rails archlint : lots et partition, contournement du filtre, concurrency, voie film sans DuckDB. Run de preuve 38979844734 : 15 min 39 s. |
| 2026-10-16 | **[sécurité] Durcissement de la surface exposée + restes de la démo** — audit adversarial (9 axes, 12 P0 / 17 P1 / 36 P2, `.ai/archive/V7.5/AUDIT_SECURITE_SURFACE_EXPOSEE_2026-10-15.md`) puis plan `.ai/archive/V7.5/PLAN_SECURITE_2026-10-15.md` (lots L0-L11, deux rondes de revue adversariale, recette, E2E) ; fusionné dans `feat/v75` (c6a72cf02). nginx de prod durci le 2026-10-15/16. Reste au déploiement de la v7.5 : V-3 (`data/demo_films/`), V-5 (unité de l'ouvrier), V-6 (limites mémoire après mesure). |
| 2026-10-16 | **[lot backlog du 2026-10-15] Campagne dans les lecteurs, vue match « lecture en échec », intégrité du registre** (branche `feat/backlog-2026-10-15`, plan `.ai/archive/V7.5/PLAN_BACKLOG_2026-10-15.md`, E1 à E6) : lecteurs de la Campagne de `sync`/`ops`/`cmd` statués et garde étendu (E2), dettes Go (temps canonique, `GET /squads`) (E3), raisons `<source>_unavailable` distinctes de `<source>_empty` (E4), colonnes d'intégrité de `match_registry` (E5), deux rondes de revue adversariale corrigées (E6.2a, E6.2b). Reste : fusion dans `feat/v75` (geste du user) et les entrées ci-dessus. |
| 2026-09-27 | **[lot backlog du 2026-09-26, fusion 3] Dérogations mortes du lint inter-features, musique d'intro du rejeu, niveau des fanfares de fin** (branche `feat/backlog-2026-09-26`, plan `.ai/archive/V7.5/PLAN_BACKLOG_2026-09-26.md`, lots A3 et A4). **Lint (5)** : 16 dérogations mortes retirées ; le script échoue désormais sur toute dérogation qui ne sert plus ; README des graphes corrigé (12 wrappers, `FirstBloodLanes` dans `components/charts/`) ; lint ajouté au job web de la CI (`1ea900034`). **Musique d'intro (7)** : les 3 dernières secondes de la montée de la piste `402178411`, résolution R = 3,19 s calée sur le coup d'envoi, −18 LUFS, jouée au clic sur « Lecture » ou « Recommencer » depuis le préambule, jamais à la reprise, au saut ni dans l'export. Au premier « Lecture » après un rechargement, l'intro attend son décodage et part avec un décalage borné par la durée du préambule (`6bac387b8`, `eb57ec3c2`). **Niveau des fanfares (suite de l'item 11)** : les trois fanfares de fin mesuraient −25 LUFS au lieu des −18 prévus (réduction stéréo du 28/08 faite après la normalisation). Remises à −18 LUFS par un gain linéaire, avec un garde-rail de niveau sur les échantillons PCM (`e93c19ed9`). Fins multi-équipes (6) : `[!]`, faute de donnée (entrée ci-dessus). Écoutes de l'utilisateur après la fusion. |
| 2026-09-27 | **[lot backlog du 2026-09-26] Musique de fin et son du rejeu, classement mondial, découpeur SQL, index ART de `match_skill_rank`, garde-rail d'exclusion de la campagne, mode démo hermétique** (branche `feat/backlog-2026-09-26`, plan `.ai/archive/V7.5/PLAN_BACKLOG_2026-09-26.md`, un exécutant Opus à la fois, deux relectures adversariales puis une ronde 2 sur les corrections : 8 constats en ronde 1, dont 2 P1, et 0 P1 en ronde 2). **Musique de fin (11)** : la conclusion échappe au plafond de 8 voix dans la page, comme dans l'export (`soundOccupiesVoice`, `a373539d9`). **Son au rechargement (12)** : le lecteur s'ouvre au premier clic ou à la première touche, ou dès l'affichage si le document a déjà reçu un geste (`useAudioUnlock`, `89f311191`) ; la cause était la lecture automatique. **Classement mondial (1)** : `ErrDrainTimeout`, 3 essais espacés de 30 s sans nouveau scrape, premier tir à +2 min (`cd47936fc`). **Découpeur SQL (3)** : cœur unique `migration.ExecScriptContext`, 3 copies supprimées, garde-rail AST (`edd0054d5`). **Index de `match_skill_rank` (2)** : les trois `idx_msr_*` sont retirés par une migration et par un soin convergent rejoué à chaque ouverture (MSR et PSA) ; ratchet étendu à `sync/schema.go` ; sonde, `repair_msr_index` et `indexcheck` supprimés ; critère D-4 amendé (relatif, `EXPLAIN ANALYZE`) ; copies réelles vérifiées (`7e9ef7c15`, `0b6c11f73`). **Exclusion de la campagne (4)** : garde AST multi-racines (83 lecteurs, 21 dispenses datées), 12 lecteurs corrigés ; les chiffres Halo 5 qui comptaient la campagne changent (`a9e2192ac`, `c61c60761`). **Mode démo (8)** : tâches de fond coupées et chemins d'état redirigés par le SEUL serveur démo (`config.LoadServer`), 4 mutations en 403 `demo_mode_forbidden`. La preuve de bout en bout sur le vrai checkout donne 0 écriture (`3de419efe`, corrections B-C1 à B-C10). Écoutes de l'utilisateur (11, 12) faites après la fusion. Découvertes versées en 6 entrées ci-dessus. |
| 2026-09-19 | **[hygiène] Lot compare / armes / frontières** (branche `feat/hygiene-compare-armes`, 4 commits `e4238dea6`→`aa1a8dc2c`, exécuté par Opus sous pilotage) — **A** champ `filters` de `CompareRequest` retiré (Go + `types.ts` ; le fragment OpenAPI manuel ne le déclarait déjà pas, `FilterContextInput` conservé : 7 autres consommateurs). **B** repli « échantillon croisé » SUPPRIMÉ — mais la prémisse du backlog était fausse : la branche n'était pas morte, elle était FAUTIVE. `GetCrossMatchSample` n'excluait pas la campagne alors que `GetLocalStats` le fait ; mesuré sur copie du shared Halo 5 : pour un B présent uniquement en coop campagne, `GetLocalStats` rend 0 ligne et l'échantillon croisé rend 1 match (stats de campagne servies sous un service record matchmade). Exclusion alignée ⇒ branche morte par construction ⇒ retrait complet (service, repo, port + noop, `domain.CrossMatchSample`, `IsLocalSample`/`is_local_sample` régénéré par Huma, 2 tests, 8 lignes de baseline). **C** `buildTopWeapons` des séries temporelles délègue à `topWeaponKillRows` (départage sur le libellé) ; garde-rail étendu au motif `WeaponID <` sans propriétaire ; changement assumé : une arme sans libellé résolu n'est plus publiée (barre anonyme avant). **D** `SynthesisCards` → `components/ui/section-primitives.tsx`, `SynthesisWeaponAccuracyChart` → `components/charts/WeaponAccuracyChart.tsx` ; DEUX dérogations retirées (`timeseries=>synthesis` et `session-detail=>synthesis`) ; le ratchet du script compte les violations non déclarées (7/7, inchangé), pas les dérogations. Gates : Go build/vet/test 22 paquets, openapi-gen -check, types frais, tsc -b (cache purgé), eslint 0 erreur, lint inter-features, vitest 7984 tests — tous verts, rejoués par le pilote. Découvertes → 2 items backlog ci-dessus. |
| 2026-08-03 | **[ops/deps] Bump `echarts` 5.6.0 → 6.1.0** (CVE-2026-45249, XSS) — livré par `545b870de` (lot B4 echarts6, diff visuel joint). L'entrée « REPORTÉ » du backlog était restée après la livraison ; retirée le 2026-09-19. |
| 2026-07-26 | **[ops/prod] Écritures `app_settings.json` dans le conteneur (bind-mount fichier → rename EBUSY)** (v7.3, `branche feat/v7.3-notion-batch, lot backlog du 26/07`) — point d'écriture unique `internal/platform/atomicfile.WriteFile` : atomique (temp + rename) d'abord, repli **in-place** (truncate + un seul Write + fsync) quand le rename répond EBUSY ou que le répertoire parent refuse le temporaire. Toute AUTRE erreur de rename reste remontée (le repli couvre une contrainte d'environnement connue, pas un diagnostic manquant). Audit des écritures runtime de settings : **3 call sites**, tous migrés — `settings.Store.Save` (chemin de TOUS les toggles admin `PATCH /settings`, qui faisait un `os.WriteFile` nu, donc jamais atomique), `settings.Store.SaveTitleOverlay`, `notify.writeLastNotifiedVersion` (le bug d'origine : notif Discord « nouvelle version » rejouée à chaque redémarrage). Limite ASSUMÉE et documentée : le repli n'est pas atomique — risque borné (contenu déjà sérialisé en mémoire, un seul Write, fsync, fichiers reconstructibles). Garde-rail `archlint/no_bare_settings_write_test.go` (interdit `os.WriteFile`/`os.Rename` nus dans les packages writers de settings). Tests : rename EBUSY → repli, rename ENOSPC → erreur non masquée, temporaire impossible → repli, troncature, création. |
| 2026-07-26 | **[ops/notifs] Bruit WARN `app_release: emit` pour les comptes auth_only** (v7.3, `branche feat/v7.3-notion-batch, lot backlog du 26/07`) — filtre pur `appReleaseTargets` dans `internal/api/wire/notifications_boot.go` : les profils `auth_only` de `db_profiles.json` (5 en prod, `db_path` vide — ils n'existent que pour le pool de tokens) sont écartés AVANT toute résolution, avec une trace `DebugContext` groupée au lieu de 5 WARN par redémarrage. Découverte au passage, corrigée dans le même filtre : `LoadPlayers()` sans filtre de titre renvoie une entrée par (titre, joueur) alors que la notification est per-JOUEUR → un joueur déclaré sur 2 titres était traité deux fois ; déduplication par slug + ordre d'émission stable. 5 tests unitaires. |
| 2026-07-26 | **[archi/contrat] Reliquats V72-01 — clos** (v7.3, `branche feat/v7.3-notion-batch, lot backlog du 26/07`) — (a) **`securitySchemes`** : le contrat ne déclarait AUCUN mécanisme d'auth ; `sessionCookie` (apiKey/cookie) est désormais posé côté Go (`internal/api/openapi_security.go`) en lisant le nom du cookie à sa source unique `session.CookieName` — aucune exigence `security` par opération n'est ajoutée (une part de la surface est publique par conception ; l'inventaire route→garde reste le ratchet `bare_routes`, qui est exécutable). (b) **UI `/docs`** : `internal/api/openapi_docs.go`, montée sur le routeur RACINE (le `DocsPath` de Huma aurait enregistré la route une fois par sous-routeur, sous son préfixe) et gatée sur `IsProduction() **OU** DemoMode` — la démo est publique et ne pose pas `LEVELUP_ENV`, la gater sur la seule production l'aurait exposée. Sert le document VIVANT (`/docs/openapi.{json,yaml}`), CSP dédiée. (c) **`ApiError.details`** : `huma.SchemaTransformer` sur `humacore.apiError` restaure `oneOf: [object(additionalProperties true), array]`, perdu par le type Go `any` — corps runtime inchangé. **Statués `[!]`, non traités et pourquoi** : validation automatique des inputs + résolveurs cross-champs = CLOS par décision produit (contrat `RawBody`/400 CONSERVÉ, pas de bascule 422) ; les **7 descriptions de schéma racine** restent au fragment manuel — les rapatrier exigerait un `SchemaTransformer` sur 7 types de `internal/domain`, donc d'y importer `huma` alors que ce package n'a **aucune** dépendance externe aujourd'hui : coût architectural disproportionné pour 7 chaînes déjà présentes au contrat publié. |
| 2026-07-25 | **[data/objectifs] Stockpile + Extraction + VIP extraits** (v7.2.1, V721-02) — 18 colonnes ajoutées à `match_objective_stats` par une migration séparée incluant la recréation de `match_objective_stats_latest` (une vue `SELECT *` fige ses colonnes à la création : sans ça, les 18 colonnes restaient invisibles à TOUS les lecteurs). `VipStats` n'était même pas déclaré dans `StatsBundle`. Payloads réels capturés (P0, 10 matchs), fixtures committées, INSERT pur. Vérifié sur copie des vraies bases : 27 → 45 colonnes, 5757 lignes préservées. Restent `EliminationStats`/`InfectionStats`, bloqués faute de donnée (cf. backlog actif). |
| 2026-07-25 | **[feat/citations] 10 citations d'objectifs livrées** (v7.2.1, V721-03) — 10 retenues sur 19 (9 écartées par décision utilisateur). Paliers **calibrés sur données réelles**, ce qui a invalidé les paliers génériques initiaux : 4 de ces citations seraient restées au premier palier à vie (3, 1, 4 et 3 occurrences au total chez le meilleur joueur). Parité EN garantie par un garde-rail neuf (le nom EN n'en avait aucun, seule la description était couverte) + garde-rail d'existence des visuels sur disque. Seed vérifié : 88 → 98, 0 sans nom ni description EN. Visuels de 2 citations : bouche-trous provisoires, l'utilisateur les produit (blocage explicite avant déploiement). |
| 2026-07-25 | **[data/armes] Colonne résiduelle `weapons.name_fr` purgée** (v7.2.1, V721-05) — migration de reconstruction CTAS-swap idempotente (no-op sur une base déjà propre), clé primaire composite recréée, garde anti-perte sur le nombre de lignes. Vérifié sur copie de la vraie metadata : colonne absente, 84 lignes et PK intactes. |
| 2026-07-25 | **[archi/contrat] `DefaultStatus` + routes absentes du contrat** (v7.2.1, V721-04) — portée réelle **23 routes** et non 11 : 7 mentaient en silence (elles publiaient `200` pour un 201/202 sans que le fragment ne les corrige), 1 découverte en plus. Champ `Status` retiré partout où le statut est fixe (une seule source de vérité) + ratchet anti-retour. Et **35 routes** absentes du contrat (et non 38 : 3 étaient un artefact de comptage, deux implémentations déclarant le même contrat) montées via un harnais unique — un second harnais aurait rouvert le canal de dérive contrat/tests fermé en v7.2. Fragment manuel : 3291 → 3237 lignes. |
| 2026-07-25 | **[archi/data] Unifier la SOURCE des noms d'armes** — traductions centralisées dans un fichier TOML par titre keyé `weapon_key` (`config/titles/{halo_infinite,halo_5}/mappings/weapon_names.toml`), résolution via `weapon_key` dans `weapon_resolver.go` (:76-94), colonne `name_fr` retirée du CREATE du registre (`weapon_registry.go` :177-189), garde-rail `weapon_names_completeness_test.go`. Commits `cebe2fed9` (V72-06) + `95dd7b5e3`. |
| 2026-07-25 | **[feat/frags] Sunburst : niveau 2 pour la classe Grenade** — ventilation par type de grenade (`grenadeRoles`, `fragdist.go` :226-258) avec résidu « Autre grenade » et invariant Σ niveau 2 == kills classe respecté, i18n `frags.toml` :124-146. Commit `0eb523bb2` (V72-15.2). |
| 2026-07-25 | **[data/frags] H5 : Σ des classes du sunburst > total (double-comptage mêlée/assassinat)** — `buildGunFragClasses` retranche désormais `MechanicKills` des kills d'arme (`fragdist.go` :93-107), invariant Σ classes == total rétabli. Commit `0eb523bb2` (V72-15.3). |
| 2026-07-25 | **[archi/match-view] Retirer le fallback LIVE du Match view** — `GetMatchMeta` miss renvoie désormais `domain.ErrNotFound` (404 `match_not_found`) au lieu d'un fetch live API ; chaîne `tryCanonicalMatchView`/`WithDataAdapter`/`WithViewerGamertag`/`LoadMatchDetail` H5 entièrement supprimée (`match_view_service.go` :229-244). Commit `468154424` (V72-15.4). |
| 2026-07-18 | **[bug/h5] Fuite d'affichage campagne (287 matchs) — balayage complet** — cause : le mécanisme centralisé d'exclusion (`analysis.campaignExcludedVariantIDs` + fragments) n'était pas appliqué sur `Q5SharedHistory` (LISTE historique + compteur) ni `Q4/Q4MV` (filtres). Fix appliqué à **la source** : liste/filtres + relations/career hub (Q26/Q28/Q28Scoped/QRelationsCoreForm). Un **garde-rail structurel** (scan AST de tous les lecteurs `match_participants`/`mv_player_matches` + `xuid=?`) a révélé la fuite systémique → balayage complet des lecteurs restants (Q10 rencontres, Q19 communs, Q23/Q23b détail-match, Q29Heatmap/Q30Rival moments, Q29TopTeammates/Q30SquadShared/Q31/Q42 escouade). Exempts justifiés (allowlist) : mono-match (Q17/Q17b/Q26), filtré au call site (Q25Template, QRelationsPlayerWinRate), sur-lecture inoffensive (Q25MatchParticipants), code mort (Q30SquadMatches). Nouveau résolveur `resolveCampaignExclusionByMatchID` (sous-requête sans placeholder, sûr avant Sprintf). Tests : structural coverage + comportemental (alias/by-match-id/token). PAS de purge BDD (règle ART). Même session : code mort `Q30SquadMatches` supprimé (règle 7). |
| 2026-07-18 | **[sécurité/autZ] BOLA prestige objet-level clos** — découverte : `WithPlayerSlug` jamais câblé → les routes `{id}` pures (`GetChallenge`/`Update`/`Abandon`/`SuggestNext`/`GetArc`) + `ListMySquads`-avec-escouades échouaient en `ErrPlayerNotResolved`. Fix : middleware `prestigePlayerSlugCtx` stampe le slug du chemin (ownership-gardé) dans le contexte → répare la résolution ET clôt le BOLA par isolation player DB (défis/arcs perso en `stats.duckdb` du joueur du chemin → `{id}` étranger = 404). Défis d'escouade (shared_social, non isolés) : garde `assertMemberUser` ajoutée à `ListSquadChallenges` (`requestedBy` = slug du chemin). Tests service+handler+middleware. Même session : `JoinSquadChallenge` (rejoindre un défi d'escouade par `{id}`) reçoit la même garde d'appartenance. |
| 2026-07-17 | **[data/h5] Classification hors-arsenal + capture mécanique de kill** — 26 IDs classés (véhicule/tourelle/environnement/non-attribué/autres) au donut, exclus de l'insight coach (`16d2a09eb`) ; colonne `kill_kind` persistée à l'ingestion H5 pour cesser de jeter la mécanique (`c13e7f6bc`). Phase 2 (backfill + découpage « Capacités Spartan »/Corps-à-corps/Non-attribué) → backlog actif. Investigations : « Spartan » = bucket d'attribution sans arme (API officielle `weapon_type=Unknown`), concentré dans les modes mêlée. |
| 2026-07-17 | **[ux/relations] Lot G LIVRÉ** (revirement produit) — CSR/tier de la bête noire affiché en dégradation gracieuse (rien si absent). Justification d'abandon initiale corrigée : la donnée EST collectée (`ExtractAllSharedCSRRows` → `match_csrs_latest`) ; couverture nulle en dev car base sociale (1,8 % classé), non représentative des joueurs compétitifs cibles. Backend best-effort + chip front conditionnel + tests. Commit `8570af76a`. |
| 2026-07-17 | **[vérif finale + dettes] passe QA post-train** — audit logging (5 flux best-effort re-routés hors `general.log` vers modules dédiés + erreur avalée `decodeParams` comblée, `e6671ff89`) ; renforcement tests (branches best-effort/gardes nil, `058ba9486`/`d49a1734f`) ; dettes réglées : erreurs avalées coach/prestige/handlers, doc inversée `notifications/types.go`, schéma OpenAPI orphelin `BattlePassResponse` retiré, `SUBTIER_ROMAN` centralisé + garde-rail (3e copie trouvée → seuil franchi). Commits `d48da5912`/`556991069`/`6bb186e9b`/`3c975bde3`. Gates Go unit+intégration `-p 1` (114 ok/0 FAIL) + web (2258 tests) verts. |
| 2026-07-17 | **[data/h5] Inventaire complet des frags hors-arsenal** — `.ai/archive/V7/H5_WEAPON_LONGTAIL_UNMAPPED.md` : 26 IDs non couverts (16 649 frags = 6,2 %), dont bucket « Spartan » 8 812 frags. Base pour une classification produit (catégories véhicule/tourelle/corps-à-corps/environnement à trancher). Commit `fbbfd809a`. |
| 2026-07-17 | **[train backlog] Purges code mort** — `squad/v2` purgé (knip croisé grep, 4 types vivants conservés) ; route `GET /players/{slug}/battlepass` supprimée après preuve de mort (service `GetBattlePass` conservé, consommé par season-pass). Commits `42b317cd8`, `c9bfa0e7d`. |
| 2026-07-17 | **[train backlog] Registre armes H5** — long-tail `v_weapon_kills` : 5 armes tenues mappées (3 grenades + golf club + oddball, ~18,4 k frags) ; véhicules/tourelles/UGC documentés hors-arsenal (item résiduel ouvert). Commit `9e0a8217d`. |
| 2026-07-17 | **[train backlog] i18n tiers + cache défis** — mapping tiers CSR centralisé dans `lib/skillTiers.ts` (+ garde-rail grep) ; défis locale-aware via locale dans les query keys `home`/`seasonPass`. Commits `45200bc4c`, `7fc82f575`. |
| 2026-07-17 | **[train backlog] Release CI (GHCR)** — livré (`7bb6a4257`) puis **RETIRÉ le 2026-07-18** sur décision user (pas d'intérêt) : retour au build local sur le VPS (comportement d'origine). |
| 2026-07-17 | **[train backlog] Prod** — `/health` expose `media_tooling` (sonde ffmpeg au boot, figée) ; `Cache-Control: public, max-age=31536000, immutable` sur les assets Vite hashés (côté Go). Commits `731c3934f`, `5f486b0b1`. |
| 2026-07-17 | **[chantier Relations UX] Plan 2026-07 CLOS** — lots A (tri colonnes), B (duels cliquables → match view via `matchIndexAtX`), C (toggle « jamais affrontés », défaut masqué, migrate store v2), F (CoreCards supprimées + lien Escouade), D (volet « Quoi de neuf » : `is_revived` SQL+DTO+strip front), E (notification `rival_encounter` post-sync par watermark, garde-fous 3/sync et 7 j), H régularisé `[~]`, G `[!]` (0 % couverture CSR rivaux). Plan archivé : `.ai/archive/V7/PLAN_RELATIONS_UX_2026-07.md`. Commits `91649c01d`→`e38592ca9`. |
| 2026-07-17 | **[sécurité] BOLA acteur prestige clos** — 14 routes non-squad réconcilient l'acteur body/query avec la session (`authorizeActor`, 403 `player_forbidden`), garde-rail AST structurel anti-régression, tests 403+verts par endpoint. Commit `b8e97cb43`. Résiduel objet-level → item backlog. |
| 2026-07-17 | **[coach/prestige] V2.1 + V3 livrés** — colonne `source` (challenge + prestige_telemetry, migration `prestige_add_source_columns_v1`), source renseignée aux 4 origines (coach/user/pilot_mode/preset), endpoint diag `GET /_diag/prestige/telemetry/{slug}` ; analyseur `cmd/prestige-tuning-analyze` (recommandations grammaire, seuils 30 %/50, validation manuelle) ; canal Discord webhook pour notifications coach (opt-in `discord_notify_coach`, OFF par défaut, best-effort). Commits `709bac9ad`, `7fa643699`, `02797d4cb`, `2f30e180d`. |

## 2026-11-23 — Découvertes du plan du backlog des 17 et 18 novembre (notées, non traitées)

Source : journal du plan `.ai/V8.3/PLAN_BACKLOG_2026-11-18.md`, section « Découvertes ».

- **Colonnes mortes du registre des matchs** : `t0_quality`, `film_match_start_ms` et `playable_duration_seconds` n'ont plus d'écrivain ni de lecteur depuis la piste du coup d'envoi (la dernière n'est ni lue ni écrite). DuckDB refuse `DROP COLUMN` sur une table à clé primaire et index : le retrait demande de reconstruire la table (recette de `internal/migration/append_only_rebuild.go`), à faire à la prochaine reconstruction. Priorité basse.
- **Scène du rejeu reconstruite à chaque rendu** : les hooks de calques de la page du rejeu rendent des objets non mémoïsés, donc la scène est reconstruite et la toile redessinée à chaque rendu. C'est aussi ce qui garde l'horloge de la barre écrite après une bascule de cadre en pause, par effet de bord. Coût de rendu probable, non mesuré.
- **Corps supprimés sans mort** : 128 corps supprimés appartiennent à un joueur qui a une vie ensuite (ni fin de manche, ni départ, ni bot) ; ils donnent une ligne « mort » non prouvée dans le fil. Cause non instruite (retour d'un joueur parti, réapparition forcée ?).
- **Durée de l'API différente de « fin moins début »** : 437 matchs, de plus d'une seconde ; non instruit.
- **Que font les notes quand le temps de jeu d'un joueur manque** : 600 s supposées pour la note de performance, poids et ordre des joueurs partis pour le LUSR (replis R11 à R14 de l'audit du coup d'envoi, `.ai/V8.3/annexes_backlog_1118/coup_d_envoi.md`). Hors du système du coup d'envoi, validé comme tel par le user.
- **Limite de la révision des notes** : une publication de rejeu qui change un temps de jeu de quelques secondes ne rejoue pas les notes (limite écrite dans l'ADR 0050).
- **Libellé d'un garde-rail qui décrit l'ancien comportement** (P3).

## 2026-11-22 — Découvertes du décodeur v1.5.0, vague 2 (notées, non traitées)

- **Accroupissement et glissade** (mis à jour le 2026-11-23) : l'accroupi est publié comme commande tenue sur les films à 60 Hz (portage des gestes du contributeur, `feat/v1.6.0`) ; la glissade n'est pas publiable (74 % des séries rapides atteignent le pic, aucune vérité indépendante).
- **Bouton de saut** (mis à jour le 2026-11-23) : publié comme commande tenue sur les films à 60 Hz seulement ; les builds 1.9 à 1.11 sont à 30 Hz et ne le publient pas. Sens des quatre booléens d'un rechargement : toujours inconnu, laissés tels quels.

## 2026-11-22 — Découvertes du décodeur v1.5.0, recherches (notées, non traitées)

- **Lettre des zones comme contrôle** : voir l'entrée « Nommer les états des marqueurs d'objectifs » (calcul fait, non intégré).
- **Arbre des actions décalé sur la version précédant `HI_1_11_0`, et Grandes batailles à drapeau des anciennes versions** : verdict du 2026-11-23, voir l'entrée « Actions de score : verdict « lecture complète » faux en BTB et en Assaut ».
- **Drapeau, état « au socle » faux avant un portage juste** : traité par les items B.1 et B.2 du plan v1.6.0 (203 joueurs justes sur 218 contre 161) ; ce qui reste est dans l'entrée « Porteurs du drapeau faux en grande bataille ».
- **Compteur des émissions jetées gonflé hors Drapeau** : la passe du verdict de mode lit la table du Drapeau sur tout film et verse ses rejets au compteur du document (0 → 1 952 sur un film d'un autre mode), sans effet sur les compteurs publiés. Correction proposée : un enregistreur à part pour cette passe.
- **Répliques des Spartans** : voir l'entrée « Répliques des Spartans : affichage » (lecture, publication et nom court faits ; table des textes et maquette côté LevelUp). Pièces de la recherche : `dec-v150-recup/h5/chatter/`.
- **Compteurs de zones sur d'anciens builds** (verdict du 2026-11-23, item B.4) : sur ces films le composant 21 côté A n'est pas un compteur (valeurs congrues à 8 modulo 32) ; la définition des emplacements du tableau n'existe dans le film que pour `HI_1_12_0` et `HI_1_13_0` ; sans effet aujourd'hui (`objectives[]` vide). Pour borner par build : un lecteur Go de ces définitions (Rust de Den en témoin, avec crédit).
- **Coup d'envoi sur trois anciens builds** : environ 0,9 s plus tôt qu'avant, sous la tolérance d'une seconde, non vérifiable à l'API.
- **Valeur de test non remesurée** (2026-11-23) : le test de l'ouvrier réel de la file de génération attend un coup d'envoi par cadence ; la valeur de la cadence de 100 ms n'a pas été remesurée après le passage à l'horloge du jeu (aucun test ne tourne à cette cadence).

## Chantier « précision » (tirs, touches, précision par arme) — section dédiée

Ouverte le 2026-11-22 à la demande du user (« mets ça dans une section dédiée pour pas qu'on confonde avec les autres
travaux de type décodeur »). Le user a annoncé ce chantier le même jour, avec ses propres idées, à recueillir avant
d'ouvrir un plan. Tout ce qui touche à la justesse des tirs, des touches et de la précision par arme se note ICI, pas
dans les découvertes du décodeur. Les écarts que le gate du corpus accepte pour la v1.5.0 du décodeur renvoient à cette
section : à la reprise, relire les décisions datées du 2026-11-22 de `config/replay_corpus.toml`.

- **Tirs du BR75 sur un film de build ancien** : trois joueurs passent hors des 10 % de l'API après la lecture des tirs à tout rang ; des numéros de tir alternent entre deux unités en moins de 500 ms, cause non prouvée. Baisse acceptée par le user le 2026-11-22.
- **Précision des armes à énergie** : le tireur d'un objet projectile est écrit dans l'état par défaut de sa création ; le garder permettrait d'ajouter les projectiles aux événements de tir. Matière du chantier de précision annoncé par le user.
- **Copies des messages de tir de rang 1 et plus** : non mesurées (les rechargements de la même vue sont écrits deux fois dans 26 à 32 % des cas).
- **Parts d'échec en hausse avec les tirs lus à tout rang** : visée absente (0,12 % → 0,23 % des tirs rattachés, un témoin en porte l'essentiel), projectile au tir ambigu (5,1 % → 6,0 %), tirs sans place sur 6 témoins, tirs embarqués sans trajet sur 2 témoins. À instruire avec le chantier « précision ».
- **Baisse acceptée du BR75** (user, 2026-11-22 : « Oui on accepte, j'ai un autre chantier à prévoir avec des idées pour la précision ») : deux joueurs de moins à 10 % de l'API après la règle du numéro de tir en retard.
- **Ancienne précision par film retirée de LevelUp** avec la montée du décodeur v1.5.0 : la table `match_weapon_hit_distance` reste au schéma sans écrivain ni lecteur ; la supprimer ou la réemployer se décide avec ce chantier.
- **Bilan des tirs par joueur après la régénération (2026-11-24)** : il se rapproche de l'API sur 250 films, mais s'en éloigne de plus de 50 tirs sur 151 films, parfois en passant sous l'API (trop haut en 1.10, trop bas en 1.11 et en Super Fiesta : hypothèse, estimation par la couverture des messages, non vérifiée). Pour reprendre : comparer `shotStats.coverage` à la part de messages non lus sur les dix pires films. Pièce : `.ai/V8.3/annexes_decodeur_v160/l7_mesures_apres_regeneration.md`, section 3.1 point 4. Priorité moyenne.

## Confidentialité du dépôt public (2026-11-23)

Constat relevé à la relecture du plan du backlog ; non traité, hors de ce plan.

- Les pseudos de joueurs figurent dans 228 fichiers du dépôt (3 182 lignes) : archives de `.ai/`, journaux de décisions, maquettes, 8 décisions d'architecture, 5 fichiers de tests web.
- Le chemin du poste de travail figure dans 240 fichiers de la branche principale.
- Ces contenus sont antérieurs au plan et déjà publics ; le plan n'en ajoute pas.
- **Décidé le 2026-11-23** : les fichiers actuels sont anonymisés « au maximum en étant prudent » (documents faits en trois passes ; code, tests et configurations en cours). **L'historique git n'est pas purgé** : message du user du 2026-11-23, trop lourd pour un projet amateur (tous les numéros de commit changeraient, 61 étiquettes et 37 releases à repousser, tous les dossiers de travail à recréer, et les forks garderaient l'ancien historique). Ne pas reproposer cette purge sans demande du user. Ce qui reste donc visible dans l'historique : anciens contenus des fichiers, messages de commit, identités d'auteur.
- Passes du code faites le 2026-11-23 (103 fichiers, puis l'identifiant du joueur source dans les deux manifestes de la démo). Restent, parce qu'ils servent : les fichiers d'attribution ; les chemins par défaut de cinq outils (`cmd/diag_weapons_v3`, `cmd/h5-metadata-fetch`, `cmd/probe-h5`, `scripts/check_syncmeta`, `scripts/audit_bp.sh`) et d'un test de repli ; `scripts/bench-pages.*` ; les outils Cheat Engine `tools/ce/filmdec_*.lua` ; deux motifs de `.gitignore` et deux lignes de `.gitleaks.toml` ; la référence ancienne des tests lue par la CI (`.ai/baselines/tests_pre_migration.jsonl`, chemins du poste et trois noms de tests) ; un pseudo de forme générique dans douze tests du rejeu.
- Règle pour la suite : ne rien ajouter de neuf (libellés neutres pour les joueurs, pas de prénom, pas de chemin du poste dans un fichier versionné ni dans un message de commit).
