package killsource

// replis_du_decodage.go — LES COMPTES DES REPLIS DU DECODEUR DE MORTS, EN DONNEES (lot J8.7 du
// PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, decision 1 du superviseur, 2026-09-27).
//
// # POURQUOI CE PAQUET NE DECLENCHE PAS LE COMPTEUR LUI-MEME
//
// Il le POURRAIT (le registre `facts/fallback` est a son etage) mais il ne le DOIT PAS : l empreinte
// de cette couche (`TestKillsourceRevSuitLaSortie`) hache ses sources, et un import du registre y
// ferait entrer chaque modification du registre — toute entree neuve rouvrirait le backlog de
// redecodage du parc (ADR 0034 amende, DU-2 (c)). Les replis se comptent donc ICI, en entiers
// nommes portes par [Stats.Replis], et la table de versement de `replay` leur donne leur nom de
// registre. Les comptes que le decodeur tenait DEJA (`Couples.Recolles`, `Appariement.Fenetre`,
// `Appariement.BotFenetre`, `Appariement.NonRevendiqueeFenetre`, `Assist.ParLaFenetre`,
// `Roster.FilmTable.Inferred`) sont versees tels quels : ils ne sont pas recopies ici.
//
// # CE QUE CES COMPTES NE CHANGENT PAS
//
// Aucune sortie de decodage : ni une mort, ni une source, ni un nom. Ils s ajoutent au resultat et
// voyagent avec lui — jusqu au fichier de faits persiste, d ou une republication les relit.

// ReplisDuDecodage compte, pour UN decodage, les declenchements des replis que les autres
// statistiques ne comptaient pas encore. Le commentaire de chaque champ nomme son entree.
type ReplisDuDecodage struct {
	// RecordsDesynchronises : `repli_record_desynchronise_jete` — dead-states Mort=1 d un record
	// dont la marche a desynchronise, jetes (`walk.go`).
	RecordsDesynchronises int
	// IndicesHorsRoster : `repli_deadstate_indice_hors_roster` — dead-states dont l indice de
	// victime OU de tueur sort du roster (`walk.go`).
	IndicesHorsRoster int
	// CategoriesHorsEnum : `repli_deadstate_categorie_hors_enum` — dead-states dont la categorie
	// sort de l enumeration (`walk.go`).
	CategoriesHorsEnum int
	// LocalisationsALargeurLibre : `repli_localisation_largeur_libre`, site de la marche des morts
	// — paquets a evenements localises par la seconde passe a largeur libre (`walk.go`).
	LocalisationsALargeurLibre int
	// NomsInventes : `repli_roster_nom_invente` — noms `?N` fabriques pour rendre l affectation
	// carree (`roster.go`).
	NomsInventes int
	// PiedParArgmax : `repli_chunk_du_pied_par_argmax` — le chunk du pied designe par le plus grand
	// nombre de kills, UNE fois par decodage (`feed.go`).
	PiedParArgmax int
	// ChainesArretees : `repli_chaine_evenement_code_non_modelise` — chaines d evenements ouvertes
	// par un kill-event plausible du rattrapage, arretees sur un code non modelise ou un cfgIdx non
	// resolu (`grammar/chaine_d_evenements*.go`), gardees ou non.
	ChainesArretees int
	// KillsRattrapes : `repli_kill_rattrape_hors_vue_a` — kill-events d une trame dont la lecture de la
	// vue A n est pas etablie (la vue B ne commence pas a sa fin), retrouves par le rattrapage de la
	// grammaire (`grammar/kills_rattrapes.go`).
	KillsRattrapes int
	// TypeDeChunkPerdu : `repli_type_de_chunk_perdu_du_manifeste` — la traduction du film jette le
	// type du manifeste, UNE fois par decodage (`chunks.go`).
	TypeDeChunkPerdu int
	// SondeNonLancee : `repli_sonde_non_lancee_porte_relachee` — couverture jugee complete, la sonde
	// a porte relachee n est pas lancee, UNE fois par decodage (`decode.go`).
	SondeNonLancee int
	// LibellesAutres : `repli_libelle_de_source_autres` — morts publiees (revendiquees ou non) dont
	// la source porte le libelle « Autres » (`label.go`).
	LibellesAutres int
	// ControleDeCorruptionNonDeclare : `repli_controle_corruption_section_absente`, site de la
	// calibration — film sans section d identification, UNE fois par decodage (`decode.go`).
	ControleDeCorruptionNonDeclare int
	// MotDePoigneeInfere : `repli_largeur_mot_de_poignee_inferee` — la largeur du mot de poignee
	// decidee par le balayage (retenue ou invariant faute de discrimination), UNE fois par
	// decodage calibre (`calibrate.go`).
	MotDePoigneeInfere int
}

// unSi rend 1 quand un repli a decide, 0 sinon — la forme d un verdict par decodage.
func unSi(decide bool) int {
	if decide {
		return 1
	}
	return 0
}

// replisDuResultat compose les comptes que la passe a accumules dans ses etages et ceux qui se
// lisent sur ce qu elle publie (libelles « Autres »). Appelee UNE fois, par [decodeCtx.finish].
func (c *decodeCtx) replisDuResultat(kills []Kill, unclaimed []UnclaimedDeath, sondeLancee bool) ReplisDuDecodage {
	r := ReplisDuDecodage{
		RecordsDesynchronises:          c.walkRes.desync,
		IndicesHorsRoster:              c.walkRes.horsRoster,
		CategoriesHorsEnum:             c.walkRes.horsEnum,
		LocalisationsALargeurLibre:     c.walkRes.largeurLibre,
		NomsInventes:                   c.roster.nomsInventes,
		PiedParArgmax:                  1, // `loadKillFeed` designe le pied par argmax a chaque decodage
		ChainesArretees:                c.killEvents.chainesArretees,
		KillsRattrapes:                 c.killEvents.rattrapes,
		TypeDeChunkPerdu:               1, // `loadFilm` jette le type du manifeste a chaque decodage
		SondeNonLancee:                 unSi(!sondeLancee),
		ControleDeCorruptionNonDeclare: unSi(!c.calib.ControleDeCorruptionLu),
		MotDePoigneeInfere:             unSi(c.calib.PoigneeDecidee),
	}
	for i := range kills {
		r.LibellesAutres += unSi(!kills[i].Source.Named)
	}
	for i := range unclaimed {
		r.LibellesAutres += unSi(!unclaimed[i].Source.Named)
	}
	return r
}
