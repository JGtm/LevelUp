package fallback

// registre_replay_detonations.go — le repli de L'EXPLOSION DE GRENADE/PROJECTILE publiee au rejeu
// (`replay/projectiles.go`), pose au lot detonations du 2026-10-22.
//
// LA LECTURE QUE CE REPLI REMPLACE QUAND ELLE MANQUE. Le film ecrit la detonation d un projectile
// dans la vue A (`projectile_detonate` genre 5, `projectile_impact_effect` genre 6,
// grammar/vue_a_detonations.go) : un message qui porte sa position, lue et appariee a la piste de
// projectile publiee par proximite spatiale et temporelle (mesure du lot : 100 % des 727 lancers
// lies des cinq films temoins trouvent une detonation a moins de 4 s et sous le centimetre de la
// derniere position repliquee, jamais avant elle). Avant ce lot cette position etait lue par
// [chargeDetonation]/[chargeImpact] puis jetee ; le rejeu posait l explosion a la derniere position
// repliquee de la piste SANS AUCUN repli nomme (une approximation tacite, jamais comptee).
var registreReplayDetonations = []Repli{
	{
		Nom:  NomDetonationDernierePositionConnue,
		Fait: "ou et quand poser l explosion d un projectile (grenade) publiee au rejeu (`Projectile.DetoX/Y/T`)",
		Mecanisme: "la derniere position repliquee de la piste ([types.ProjectileTrack]) sert d explosion quand " +
			"aucune detonation lue (grammar.ScanProjectileDetonations) n est a portee de cette piste dans la " +
			"fenetre mesuree (0,5 s avant sa derniere position, 4 s apres)",
		// LA LECTURE QUI MANQUE N EST PAS LA DETONATION ELLE-MEME (lue a 100 % des detonations a
		// position du parc temoin) : c est l APPARIEMENT a CETTE piste precise qui peut echouer — une
		// piste coupee avant sa vraie fin (projectileMaxStepM), une vie trop courte pour publier un
		// dernier point utile, ou un film dont la table des genres de la vue A ne rend pas le message
		// lisible (classe illisible, grammaireDeLaVueA).
		Condition: CondLectureNonPortee,
		Ordre:     OrdreApresLecture,
		Sites: []Site{
			{Fichier: pkgReplay + "projectiles.go", Ancre: "func attachDetonation("},
		},
		DatePose:     "2026-10-22",
		CibleRetrait: "sans objet pour l instant : la detonation se lit a 100 % sur le parc temoin, le repli ne couvre que la piste coupee ou illisible, pas un manque de grammaire",
		CritereRetrait: "coverage.projectiles.detonationFallback (ou equivalent) mesure a 0 sur le parc complet une fois " +
			"recuit : alors le repli reste un filet, jamais le chemin courant",
		CompteurBranche: true,
	},
}
