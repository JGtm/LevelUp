# Rapport bl-div (E.6, E.2, F.3, F.6, F.4, F.5)

Outils hors depot : `run.sh` (replay-build de LevelUp-wt-dec160 lie au worktree par un modfile), `build/replay-build-{avant,apres}.exe`,
`avant_docs/`, `apres_docs/` (documents de 27 films, avec identifiants de joueurs : ne pas verser), `exp/` (copie de la base de verite et
faits des 3 films VIP), scripts `ana.py amb.py cmp.py f6_cas.js`. Sondes `research` retirees du worktree.
Insertion dans un fichier reserve : aucune.

## E.6 : FAIT (code). Victime absente quand plusieurs candidates restent, compteur `victimAmbiguous`.
- Code : `replay/eliminations.go` (`victimsOf` et quatre aides), `replay/document_eliminations.go` (champ et contrat), tests
  `replay/eliminations_ambigues_test.go` (action reclamee par deux images, deux noms differents, cas sans ambiguite).
- Mesure (verifie) : 27 films, avant/apres : listes d'eliminations identiques, ambiguite = 0, 332/332 joueurs egaux a l'API. Effet nul
  sur ces films : la regle retire l'appariement arbitraire, elle ne corrige pas un defaut observe.

## E.2 : FAIT (code). Temps en VIP, plus long temps, meilleure serie, en courbes par joueur.
- Existence dans le film (verifie) : comp 23 B = TimeAsVip, comp 23 A = LongestTimeAsVip (dixiemes), comp 22 B = BestKillingSpreeAsVip.
- Arbitre : la copie de l'API CONTIENT les trois colonnes (`match_objective_stats_latest`), renseignees pour les 3 films VIP du parc
  (99553e4a, 9903b1c5, 00761d27), contrairement a l'enonce. Documents cuits : temps 24/24 exacts, serie 24/24, plus long temps 22/24 + 2 a 0,1 s.
- Code : `internal/facts/objectives/scoreboard_stats.go` (`ScoreboardStatsOf`, `StatTimeAsVip`, `StatLongestTimeAsVip`, `StatMaxSpreeAsVip`),
  `named_kill_bound.go`, `replis_a_la_consultation.go`, `replay/score_timeline{,_stats,_players}.go`, `build_score.go` (famille posee par
  `opt.Vip.Scanned`). Tests : `replay/score_timeline_vip_test.go`, `scoreboard_stats_test.go`.
- Cout : +3,1 / +3,2 / +4,1 % de poids compresse sur les 3 films VIP (points du tableau 829 -> 13 504 sur l'un). Hors compteurs VIP, le tableau
  des 3 films est identique ; aucun film non VIP ne change.

## F.3 : VERDICT, entree du registre CONSERVEE, aucun code.
- Mesure (verifie) : 13 films a vehicules (builds HI_1_4_1 a HI_1_13_0), 146 pieces montees toutes lues par le film,
  `repli_tourelle_porteur_voisin_de_slot` = 0, `turretCarrierBirthMismatch` = 0.
- FORMAT.md dit que le format de film 31 (le plus ancien) ne porte pas le parent et garde ce repli ; le seul film de ce format du lot
  (50247b26) n'a aucune piece montee : le zero ne couvre pas le cas pour lequel le repli existe. Et retirer l'entree sans retirer le code
  (qui appelle `Declenche`) produirait un diagnostic `repli_hors_registre` en erreur. A reprendre sur le parc regenere, comme le dit le backlog.

## F.6 : VERDICT [!], pas de defaut, aucun code.
- Les 4 cas (parc de 1 455 documents, lecture seule, `f6_cas.js`) : deux films, ecarts de 2 et 4 images, grenades a fragmentation ; dans les 4 la
  vie « repreneuse » du slot est du MEME joueur que la vie achevee, donc le lacher n'est credite a aucun autre joueur. Aucun fait a chercher,
  aucune tolerance a poser. Sur les 27 films neufs : 0 cas.

## F.4 : VERDICT, aucun code.
- Les huit noms d'action de la v1.5.0 sont des `objectives[]` (statborg), pas des `scoreActions`. Sur 8 films recents (HI_1_13_0, CTF et Bases) :
  409 des 410 temps forts « mode » ont, pour le meme joueur a 2 s, un objectif ET une action de score ; 1 n'a que l'action ; aucun sans rien.
- Les temps forts sans correspondance viennent des films ou rien ne se publie : 8 films de builds HI_1_4_1 a HI_1_11_0 (dont Heavies CTF,
  Total Control) annoncent 54 a 138 evenements `available` tous `refusedByRoster`, et `scoreActions` est vide (`groupMissing` ou vue A illisible).
  Ce n'est pas un nom manquant. A ouvrir ailleurs (deduit) : refus par le roster sur 8 films sur 8 avant HI_1_12_0, 0 sur 8 apres.
- Extraction : aucun film dans le lot, non mesure. KOTH : 63 temps forts sur 66 couverts par `onHill*Response`.

## F.5 : CAUSE TROUVEE (verifie sur 7 films a 8 joueurs presents), aucun code decodeur.
- `duration_seconds` de l'API compte la periode d'AVANT le coup d'envoi : `real_start_time - start_time` (fuseau corrige) = 26,5 / 41,1 / 41,3 / 33,5 /
  33,6 / 26,6 / 28,8 s, egal a l'instant de coup d'envoi du film (`t0FilmMs`) a 0,4 s pres sur les 7 films.
- L'API s'arrete environ 6,5 s AVANT la fin du film (6 films : 6,0 a 6,6 s ; un : 13,8 s) : le film porte une queue de cet ordre.
- API moins film (du coup d'envoi a la fin) = avant coup d'envoi - 6,5 s : de 19,4 a 35,1 s, c'est le decalage commun aux partants d'un match.
- `time_played_seconds` d'un joueur vaut tantot le jeu depuis le coup d'envoi (5 films, ecart -1,6 a -4,9 s), tantot la duree API entiere (2 films, +35,1 et
  +21,4 s) : il compte le decompte d'avant le coup d'envoi pour certains matchs ; a recouper avec `present_at_beginning` (non fait).
- `real_start_time` de la copie est decale d'un nombre entier d'heures par rapport a `start_time` (fuseau) ; corrige ici modulo 1 h.

## Preuve
- gofmt vide ; `go vet ./...` et `go vet -tags=research ./...` (CGO=1) propres.
- `go test -count=1` : replay, internal/grammar, internal/facts/{objectives,killsource,fallback}, decfilm, filmcache, types. Rouges : voir fragments.md
  (golden de forme, revision objectives, 4 tests de fixtures d'environnement).
- Un fil / quatre fils : NON rejoue. L'hote (`replaybuild`, LevelUp) ne passe pas de nombre de fils, donc `replay-build` tourne a un fil ; mes changements
  sont des fonctions pures apres le decodage, sans goroutine ni ordre de map observable. A rejouer par le pilote a l'integration.
