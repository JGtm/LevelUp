package fallback

// registre_killsource_collecteur.go — les replis du COLLECTEUR qui ecrit le kill-feed en base
// (`internal/sync/killcollector/`) : resolution des identites, ventilation des tirs, precision par
// arme, pont d identite des positions, faits d isolement.
//
// SCINDE DE `registre_killsource.go` AU LOT J8.7 (2026-09-27) PAR DEPLACEMENT PUR (decision 6 du
// superviseur) : le fichier portait 523 lignes, au-dela du seuil de 500 du depot. La coupe suit le
// PROCESSUS qui execute le repli — la-bas le decodeur (`film/facts/killsource`), dont les comptes
// voyagent avec le resultat jusqu a la cuisson ; ici la passe de synchronisation, hors de toute
// cuisson, dont le compteur est celui de la passe (expvar par nom et journal par film). Les trois
// gestes d une famille neuve sont faits : [Tranches], `famillesAttendues`, `plancherTranches`.
//
// LE COMPTE (sous-lot collecteur du lot J8.7, meme jour) : chaque site `Declenche` sur le compteur de
// la passe du film (`killcollector/replis_de_la_passe.go`), publie en expvar `killsource_<nom>` et
// au journal du film — le collecteur n ecrit aucun document ou `coverage.fallbacks` les porterait.

var registreKillsourceCollecteur = []Repli{
	{
		Nom:       "repli_xuid_vide_pour_nom_inconnu",
		Fait:      "le xuid de la victime d'une mort ecrite en base",
		Mecanisme: "le nom ne se resout dans aucune table : un xuid VIDE est rendu et la mort est ecrite sans xuid de victime",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{Fichier: pkgKillcollector + "identities.go", Ancre: "m.replis.Declenche(decfilm.NomXuidVidePourNomInconnu)"}, {
			Fichier: pkgKillcollector + "identities.go",
			Ancre:   "return m.ParNom[nom], nom",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "la table du film nomme les joueurs a zero mort, jusqu a l ecriture ; a defaut, " + retraitRegle4,
		CritereRetrait:  "0 ligne de `match_deaths` sans xuid de victime sur le parc",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_indice_en_collision_jete",
		Fait:      "le joueur d'un indice de replication, pour les tirs et les touches",
		Mecanisme: "deux xuids sur le meme indice : les DEUX sont jetes",
		Condition: CondContradiction,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{Fichier: pkgKillcollector + "shots.go", Ancre: "fb.DeclencheN(decfilm.NomIndiceEnCollisionJete, collisions)"}, {
			Fichier: pkgKillcollector + "shots.go",
			Ancre:   "out[pi] = \"\"",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "l indice des tirs et des touches vient de la table du film (cette voie sert `match_weapon_shots` et `match_weapon_accuracy`) ; a defaut, " + retraitRegle4,
		CritereRetrait:  "l'indice vient de la table du film ; 0 collision sur le parc",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_premiere_occurrence_sans_concordance",
		Fait:      "quel motif de xuid retenir quand un chunk en porte plusieurs",
		Mecanisme: "la PREMIERE occurrence gagne, sans exiger que les suivantes concordent",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{Fichier: pkgKillcollector + "shots.go", Ancre: "*discordances++"}, {Fichier: pkgKillcollector + "shots.go", Ancre: "fb.DeclencheN(decfilm.NomPremiereOccurrenceSansConcordance, discordances)"}, {
			Fichier: pkgKillcollector + "shots.go",
			Ancre:   "continue // premiere occurrence gagnante",
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "l indice des tirs et des touches vient de la table du film ; a defaut, " + retraitRegle4,
		// CONTRASTE MESURÉ par l'audit 0.E : `replay/player_index.go` REFUSE de publier sur
		// désaccord, ce chemin-ci retient la première valeur vue.
		CritereRetrait:  "l'indice vient de la table du film ; la voie par motifs est retiree avec ses tests",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_precision_par_arme_passe_sautee",
		Fait:      "la precision par arme d'un match",
		Mecanisme: "cache de films non configure : toute la passe est sautee, en best-effort silencieux",
		Condition: CondSectionAbsente,
		Ordre:     OrdreSansLecture,
		Sites: []Site{{Fichier: pkgKillcollector + "hits.go", Ancre: "replisDeLaPasse(ctx).Declenche(decfilm.NomPrecisionParArmePasseSautee)"}, {
			Fichier: pkgKillcollector + "hits.go",
			Ancre:   "numerateur non configure (chemin live sans cache disque)",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "aucune (configuration d'exploitation) ; le COMPTE est ce qui manque",
		CritereRetrait:  "un compteur expvar dit combien de matchs passent sans numerateur film ; retrait sans objet",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_coequipiers_partis_constante_nulle",
		Fait:      "le nombre de coequipiers PARTIS a l'instant d'une mort (`teammates_left`)",
		Mecanisme: "la colonne est ecrite a 0 sur toutes les lignes depuis le retrait de son producteur, indiscernable d'une mesure",
		Condition: CondInconditionnel,
		Ordre:     OrdreSansLecture,
		Sites: []Site{{Fichier: pkgKillcollector + "isolation_facts.go", Ancre: "ids.replis.DeclencheN(decfilm.NomCoequipiersPartisConstanteNulle, len(out))"}, {
			Fichier: pkgKillcollector + "isolation_facts.go",
			Ancre:   "TeammatesLeft:  0,",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "retrait de la colonne (append-only, ADR 0026 : la colonne reste, c'est son ECRITURE qui doit devenir nulle explicite)",
		CritereRetrait:  "la colonne cesse d'etre ecrite, ou un producteur la remplit ; un zero constant ne se distingue d'une mesure par rien",
		CompteurBranche: true,
	},
}
