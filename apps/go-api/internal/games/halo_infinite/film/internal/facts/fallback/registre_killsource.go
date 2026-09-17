package fallback

// registre_killsource.go — les replis du décodeur de morts (`film/facts/killsource/`) et du
// collecteur qui l'écrit en base (`internal/sync/killcollector/`).

const (
	pkgKillsource    = "internal/games/halo_infinite/film/internal/facts/killsource/"
	pkgKillcollector = "internal/sync/killcollector/"
)

var registreKillsource = []Repli{
	{
		Nom:       "repli_record_desynchronise_jete",
		Fait:      "quels records d'un paquet entrent dans la marche des morts",
		Mecanisme: "un record dont la marche s'est desynchronisee est jete, et la marche du paquet s'arrete la",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgKillsource + "walk.go",
			Ancre:   "if r.DesyncAt != -1 {",
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "lot 3.x (registre ECS par build) : une desynchronisation est une grammaire fausse, pas une donnee",
		// PIÈGE CONNU (mémoire du chantier véhicules, 2026-09-05) : un filtre `DesyncAt == -1`
		// JETAIT des morts de véhicule réellement lues.
		CritereRetrait:  "0 record desynchronise sur les 8 builds une fois le registre ECS resolu par build",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_deadstate_hors_bande_bipede",
		Fait:      "un dead-state lu est-il credible",
		Mecanisme: "slot hors de la bande de slots bipede du film : le dead-state est rejete par un `continue` nu",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgKillsource + "walk.go",
			Ancre:   "if d.slot < res.bipLo || d.slot > res.bipHi {",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 3.5 (la bande de slots bipede par build)",
		CritereRetrait:  "la bande vient du profil du build ; 0 rejet hors bande sur les 8 builds",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_deadstate_indice_hors_roster",
		Fait:      "un dead-state lu designe-t-il un joueur du roster",
		Mecanisme: "indice de victime ou de tueur hors [0, nPlay) : rejete par un `continue` nu",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgKillsource + "walk.go",
			Ancre:   "if d.dead.EnumA < 0 || int(d.dead.EnumA) >= r.nPlay {",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.8 (le kill feed prend la table du film) porte jusqu'au rejet",
		CritereRetrait:  "0 rejet pour indice hors roster sur les 8 builds ; un indice hors domaine est un defaut de largeur, pas une donnee",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_deadstate_categorie_hors_enum",
		Fait:      "la categorie de mort d'un dead-state",
		Mecanisme: "valeur hors de l'enumeration connue (> 9) : le dead-state est rejete par un `continue` nu",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgKillsource + "walk.go",
			Ancre:   "if d.dead.Val0c > 9 {",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 3.x (enumeration par build)",
		CritereRetrait:  "0 categorie hors enumeration sur les 8 builds",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_localisation_largeur_libre",
		Fait:      "ou commencent les records d'un paquet dont la signature de largeur a echoue",
		Mecanisme: "seconde passe a LARGEUR LIBRE, essayee seulement apres l'echec de la signature",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgKillsource + "walk.go",
			Ancre:   "func locateFallback(pl []byte, w *grammar.World, cfg grammar.FrameConfig) int {",
		}, {
			// SECOND SITE, POSE LE 2026-09-16 (lot 1.9.10) : la marche des morts d'objet porte
			// le MEME localisateur, donc le MEME repli — une seule entrée pour un seul fait.
			// Les deux marches se rejoignent au pas 4 de M2 (« une seule porte aux octets ») ;
			// ce jour-là ce site redeviendra unique.
			Fichier: pkgFilmdec + "object_deaths_march.go",
			Ancre:   "func marchLocateFallback(pay []byte, w *World, cfg FrameConfig) int {",
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "lot 3.4 (largeurs calibrees par la carte et le build)",
		// Ce repli-ci porte DÉJÀ son nom dans le code (`locateFallback`, `marchLocateFallback`) :
		// c'est ce que la convention du garde-rail exige, et il entre au registre pour cette
		// raison.
		CritereRetrait:  "0 recours a la largeur libre sur les 8 builds une fois les largeurs prises au profil",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_roster_nom_invente",
		Fait:      "la liste de noms sur laquelle la bijection indice -> joueur se resout",
		Mecanisme: "moins de noms que de sieges : des noms ?N sont fabriques pour rendre le probleme d'affectation carre",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgKillsource + "roster.go",
			Ancre:   "r.names = append(r.names, fmt.Sprintf(\"?%d\", len(r.names)))",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.8 (la table du film donne le lien indice -> joueur) : un nom fabrique ne doit plus servir",
		CritereRetrait:  "0 nom ?N fabrique sur les 8 builds quand la table du film est lue",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_roster_indice_hors_bijection",
		Fait:      "le nom porte par un indice que la bijection ne resout pas",
		Mecanisme: "la chaine « ? » est rendue ; pour la victime et le tueur elle part en base telle quelle",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgKillsource + "roster.go",
			Ancre:   "return \"?\"",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.8 porte jusqu'a l'ecriture en base",
		CritereRetrait:  "0 ligne de mort ecrite avec un nom « ? » sur le parc",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_bijection_hongroise_du_feed",
		Fait:      "le lien indice de replication -> joueur, pour les indices que la table du film ne donne pas",
		Mecanisme: "affectation hongroise sur les votes du kill feed, puis raffinement local",
		Condition: CondSectionAbsente,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgKillsource + "bijection.go",
			Ancre:   "r.table.Inferred = len(free)",
		}},
		DatePose:     "2026-09-14",
		CibleRetrait: "cloture de M1 puis recuisson : la table du film (lot 1.8) doit couvrir 100 % des indices",
		// Repli POSÉ ET COMPTÉ par le lot 1.8 : `RosterTable.Inferred` est son compteur, publié
		// dans les statistiques de collecte. Il entre au registre pour que sa condition de
		// retrait soit lisible au même endroit que les autres.
		CritereRetrait:  "RosterTable.Inferred a 0 sur le parc apres la recuisson de cloture M1",
		CompteurBranche: false,
		// CIBLE REECRITE LE 2026-09-16 (revue de jalon M1) : elle nommait le lot 1.9.3, fusionne
		// le 2026-09-15 sans avoir publie ce compte sous son nom de registre.
		CibleComptage: "cloture de M1, avec la recuisson (le compte existe deja sous RosterTable.Inferred ; il reste a le publier sous ce nom)",
	},
	{
		Nom:       "repli_couple_recolle_sur_le_voisin",
		Fait:      "la VICTIME d'un kill que le kill feed porte sans mort en face",
		Mecanisme: "la mort d'un instant VOISIN (deux instants au plus) est prise pour victime de ce kill",
		// LE FILM ECRIT CE COUPLE (kill-event 85, `victime(E5) tueur(E5)`) et le lot 1.9.3 le LIT.
		// Le repli ne reprend la main que sur un silence de cette lecture : aucun kill-event de la
		// fenetre ne nomme ce tueur par deux indices EPINGLES (table des joueurs du film ou
		// BOT_METADATA), ou deux enregistrements en nomment des victimes differentes.
		Condition: CondSectionAbsente,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgKillsource + "feed_couples.go",
			Ancre:   "func (res *resolveurDeCouples) repliRecollageSurLeVoisin(i int) {",
		}},
		DatePose:     dateVague2,
		CibleRetrait: "cloture de M1 puis lot 3.6 : la table du film doit couvrir 100 % des indices, et la chaine d'evenements ne doit plus s'arreter",
		// MESURE DU LOT 1.9.3 (21 films entiers, 8 builds, 14 temoins) : 281 kills sans mort en
		// face, 198 decides par la lecture (198 accords, 0 contradiction), 1 victime BOT nommee,
		// 1 ambigu, 81 muets. Les 81 muets sont le compte a faire tomber ; 46 d'entre eux
		// viennent des trois films sans table de joueurs exploitable (`a349fea8`, `a521164d`,
		// `50247b26`), les autres d'une chaine d'evenements qui s'arrete avant le kill-event.
		CritereRetrait:  "CoupleStats.Recolles a 0 sur les 8 builds et sur le corpus gate",
		CompteurBranche: false,
		CibleComptage: "cloture M1 ou pas 2 de M2 : le compte EXISTE deja sous `CoupleStats.Recolles` " +
			"et sort en `killsource_couple_recolle` ; il reste a le publier sous ce nom dans " +
			"`coverage.fallbacks[]`, ce qui demande au compteur de traverser `replaybuild/kills.go`",
	},
	{
		Nom:       "repli_gamertag_par_xuid_brut",
		Fait:      "le nom affiche d'un joueur du kill feed dont le gamertag manque",
		Mecanisme: "la forme xuid:<N> remplace le nom",
		Condition: CondSectionAbsente,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgKillsource + "feed.go",
			Ancre:   "name := gt[e.XUID]",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.8 porte : la table du film nomme les joueurs a zero mort",
		CritereRetrait:  "0 nom sous la forme xuid:<N> sur les 8 builds",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_chunk_du_pied_par_argmax",
		Fait:      "quel chunk d'un film est le PIED (kill feed, recompenses, evenements de mode)",
		Mecanisme: "chaque chunk est parse en evenements de highlight et celui qui porte le PLUS de kills gagne",
		Condition: CondNonResolu,
		Ordre:     OrdreDevantLaLecture,
		Sites: []Site{{
			Fichier: pkgKillsource + "feed.go",
			Ancre:   "if nk > bestN {",
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "lot 1.9.8 (le chunk du pied pris au type du manifeste)",
		// ORDRE `devant_la_lecture` : le TYPE du chunk est porté par `source.Film.Meta()` et
		// déjà lu par ce patron (`objectives/extract.go`) ; l'argmax décide sans le
		// consulter. RÉSERVE du plan : le manifeste est un descripteur EXTERNE, l'argmax restera
		// donc en repli COMPTÉ après la conversion.
		CritereRetrait:  "le type du manifeste decide ; l'argmax ne se declenche que sur un film sans manifeste, et son compte le dit",
		CompteurBranche: false,
		CibleComptage:   "lot 1.9.8",
	},
	{
		Nom:       "repli_calibration_paquet_exclu",
		Fait:      "sur quels paquets la calibration des largeurs de la marche des morts s'appuie",
		Mecanisme: "un paquet trop court, d'un autre type ou porteur d'evenements est exclu de l'echantillon",
		Condition: CondNonResolu,
		Ordre:     OrdreSansLecture,
		Sites: []Site{{
			Fichier: pkgKillsource + "calibrate.go",
			Ancre:   "if p.typ != packetType0 || hasEvents(p) || len(p.payload) < 400 {",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 3.4 (les largeurs deviennent une donnee de profil, la calibration disparait)",
		CritereRetrait:  "la calibration entiere est retiree avec ses tests",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_calibration_paquet_non_localise",
		Fait:      "le parametre d'etat de record retenu pour le film",
		Mecanisme: "un paquet dont les records ne se localisent pas est ignore ; le meilleur score des autres gagne",
		Condition: CondNonResolu,
		Ordre:     OrdreSansLecture,
		Sites: []Site{{
			Fichier: pkgKillsource + "calibrate.go",
			Ancre:   "best, bestN = r, n",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 3.4",
		CritereRetrait:  "la calibration entiere est retiree avec ses tests",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_chaine_evenement_code_non_modelise",
		Fait:      "la longueur du corps d'un evenement, donc la suite de la chaine",
		Mecanisme: "code hors des 28 modelises (95 codes sur 123) ou cfgIdx non resolu : la chaine s'arrete",
		Condition: CondLectureNonPortee,
		Ordre:     OrdreApresLecture,
		Sites: []Site{
			{Fichier: pkgKillsource + "eventbody.go", Ancre: "func evBody(r *curseurEv, code int, gate15 bool) bool {"},
			{Fichier: pkgKillsource + "eventchain.go", Ancre: "if c < 0 || c >= len(presRange) {"},
		},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 3.6 (les composants manquants, archetype par archetype)",
		CritereRetrait:  "les 95 codes non modelises portes, ou leur longueur lue chez l'ecrivain ; 0 arret de chaine sur les 8 builds",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_type_de_chunk_perdu_du_manifeste",
		Fait:      "le type de chaque chunk, dans le vocabulaire du decodeur de morts",
		Mecanisme: "la traduction du film ne recopie PAS Meta() : le type du manifeste est perdu, et l'argmax le remplace en aval",
		Condition: CondInconditionnel,
		Ordre:     OrdreDevantLaLecture,
		Sites: []Site{{
			Fichier: pkgKillsource + "chunks.go",
			// L ANCRE EST LA TRADUCTION ELLE-MEME depuis le lot 2.1.4 : c est LA ligne ou
			// `Meta()` est perdu, donc celle que ce repli decrit. Elle pointait jusque-la sur
			// la lecture de la version majeure, voisine de hasard, qui a bouge quand cette
			// version est passee au profil du film.
			Ancre: "f := &film{src: src, packets: packetsOf(src)}",
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "lot 1.9.8 (cause racine de l'argmax du pied)",
		// DÉFAUT DÉJÀ MESURÉ (audit 0.E) : c'est la cause racine de la ligne A7. Le type existe
		// à l'entrée du décodeur et il est jeté à la traduction.
		CritereRetrait:  "Meta() voyage jusqu'au decodeur ; repli_chunk_du_pied_par_argmax se declenche alors seulement sans manifeste",
		CompteurBranche: false,
		CibleComptage:   "lot 1.9.8",
	},
	{
		Nom:       "repli_mort_non_revendiquee_la_plus_proche",
		Fait:      "a quel couple du kill feed rattacher une mort que personne ne revendique",
		Mecanisme: "aucune identite de paquet en face : le candidat le plus proche EN TEMPS dans la fenetre de 2,5 s gagne",
		// LE FILM ECRIT L IDENTITE DE PAQUET quand un kill-event 85 nomme l instant, et le lot
		// 1.9.7 la LIT d'abord ([pass.choisirNonRevendiquee]). Une mort que PERSONNE ne
		// revendique ne porte, par definition, aucun kill au feed — donc aucun kill-event 85 a
		// associer : le repli y est la voie normale, et c'est un negatif MESURE.
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{
			{
				Fichier: pkgKillsource + "hybrid.go",
				Ancre:   "func (p *pass) choisirNonRevendiquee(e feedEvent) (sourcedCandidate, bool, bool) {",
			},
			{
				Fichier: pkgKillsource + "hybrid.go",
				Ancre:   "if d := absMS(dt); d < bestDT {",
			},
		},
		DatePose:     dateAudit0E,
		CibleRetrait: "lot 3.6 (composants manquants) : une mort non revendiquee n'aura de lecture que si le film nomme son instant autrement que par le kill feed",
		// MESURE DU LOT 1.9.7 (21 films entiers, 8 builds) : 17 morts non revendiquees, dont
		// 17 SANS aucune identite en face. Le repli les sert toutes, et le compte le dit.
		CritereRetrait:  "ApparStats.NonRevendiqueeFenetre a 0 sur les 8 builds et sur le corpus gate",
		CompteurBranche: false,
		CibleComptage: "cloture M1 ou pas 2 de M2 : le compte EXISTE sous `ApparStats." +
			"NonRevendiqueeFenetre` et sort en `killsource_appariement_non_revendiquee_fenetre` ; " +
			"il reste a le publier sous ce nom dans `coverage.fallbacks[]`, ce qui demande au " +
			"compteur de traverser `replaybuild/kills.go`",
	},
	{
		Nom:       "repli_mort_de_bot_premier_candidat",
		Fait:      "quel dead-state correspond a la mort d'un bot, ou a une mort causee par un bot",
		Mecanisme: "aucune identite de paquet en face : le PREMIER candidat de la fenetre de 2,5 s gagne, l'unicite n'est pas verifiee",
		// LE KILL-EVENT 85 QUI NOMME UN BOT EN VICTIME PORTE SON PAQUET, et le lot 1.9.7 le LIT
		// d'abord ([decodeCtx.apparierMortDeBot], [decodeCtx.resolveBotKillerDeaths]). Mais le
		// kill feed est HUMAIN-SEUL : un instant qui ne porte pas de kill humain n'a aucun
		// kill-event a associer, et le repli reste la voie normale de ces deux populations.
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{
			{
				Fichier: pkgKillsource + "match.go",
				Ancre:   "func (c *decodeCtx) resolveBotDeaths() []botMatch {",
			},
			{
				Fichier: pkgKillsource + "match.go",
				Ancre:   "func (c *decodeCtx) apparierMortDeBot(m *botMatch) {",
			},
			{
				Fichier: pkgKillsource + "match.go",
				Ancre:   "func (c *decodeCtx) resolveBotKillerDeaths(all []sourcedCandidate) []botKillerMatch {",
			},
		},
		DatePose:     dateAudit0E,
		CibleRetrait: "lot 3.6 (composants manquants) : la chaine d'evenements doit atteindre le kill-event 85 des morts de bot",
		// MESURE DU LOT 1.9.7 (21 films entiers, 8 builds) : 6 morts DE bot et 4 morts PAR un
		// bot apparieees, dont UNE SEULE par l'identite de paquet — les 9 autres n'ont aucune
		// identite en face.
		CritereRetrait:  "ApparStats.BotFenetre a 0 sur les 8 builds et sur le corpus gate",
		CompteurBranche: false,
		CibleComptage: "cloture M1 ou pas 2 de M2 : le compte EXISTE sous `ApparStats.BotFenetre` " +
			"et sort en `killsource_appariement_bot_fenetre` ; il reste a le publier sous ce nom " +
			"dans `coverage.fallbacks[]`, ce qui demande au compteur de traverser `replaybuild/kills.go`",
	},
	{
		Nom:       "repli_appariement_par_fenetre_temporelle",
		Fait:      "quel instant du kill feed un dead-state lu decrit",
		Mecanisme: "demi-fenetre de 2,5 s (`tolMS`) : le premier instant du feed portant le meme couple gagne, sans aucun lien ecrit",
		// LE FILM ECRIT LE LIEN, ET LE LOT 1.9.7 LE LIT : le dead-state et le kill-event 85 de la
		// meme mort vivent dans le MEME paquet de replication, et l'instant du feed porte
		// desormais cette identite ([feedEvent.paquet], posee par [killFeed.resoudreCouples]).
		// L'ordre est fixe dans [choisirParIdentitePuisFenetre] : lecture d'abord, fenetre
		// ensuite. Le repli n'entre que sur le silence — aucun kill-event associe a cet instant,
		// ou paquet designe qui ne porte aucun dead-state satisfaisant la contrainte de couple.
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{
			{
				Fichier: pkgKillsource + "paquet_identite.go",
				Ancre:   "func choisirParIdentitePuisFenetre(n int, identite, fenetre, couple func(int) bool) (int, bool) {",
			},
			{
				Fichier: pkgKillsource + "match.go",
				Ancre:   "func (c *decodeCtx) matchExact(cd candidate) (*feedEvent, bool) {",
			},
			{
				Fichier: pkgKillsource + "match.go",
				Ancre:   "func (c *decodeCtx) matchVictim(cd candidate) (*feedEvent, bool) {",
			},
			{
				// L ASSISTANT ET LES DEUX PARTS DE DEGATS : quel kill-event 85 decrit la ligne
				// publiee. Le repli y vaut DEJA ZERO sur les 21 films entiers (2 342 memes
				// enregistrements, 1 divergence, 0 fois ou l identite se tait alors que la
				// fenetre trouvait) : c'est le site que D14 (d) rend eligible au retrait sec.
				Fichier: pkgKillsource + "assist.go",
				Ancre:   "func (c *decodeCtx) killEventsFor(k *Kill, s *assistScan, used []bool) ([]int, bool) {",
			},
		},
		DatePose:     "2026-09-16",
		CibleRetrait: "cloture de M1 puis lot 3.6 : tout instant du kill feed doit porter un kill-event 85 aux deux indices EPINGLES",
		// MESURE DU LOT 1.9.7 (21 films entiers, 8 builds, 14 temoins du corpus gate) : 2 899
		// appariements, 2 205 a identite EGALE des deux cotes, 2 a identite DIFFERENTE, 692 SANS
		// identite du cote feed. L'appariement par identite seule rend 2 204 accords et ZERO
		// desaccord : la lecture ne contredit jamais la fenetre, elle la remplace. Les 692 sans
		// identite sont le compte a faire tomber, et ils ont la meme cause que les 81 muets du
		// lot 1.9.3 (chaine d'evenements qui s'arrete avant le kill-event, table de joueurs
		// refusee sur les films sans section d'identification).
		CritereRetrait: "ApparStats.Fenetre et AssistStats.ParLaFenetre a 0 sur les 8 builds et " +
			"sur le corpus gate — le site de l'assistant y est DEJA a zero",
		CompteurBranche: false,
		CibleComptage: "cloture M1 ou pas 2 de M2 : le compte EXISTE sous `ApparStats.Fenetre` et " +
			"`AssistStats.ParLaFenetre`, et sort en `killsource_appariement_fenetre` et " +
			"`killsource_assistant_fenetre` ; il reste a le publier sous ces noms dans " +
			"`coverage.fallbacks[]`, ce qui demande au compteur de traverser `replaybuild/kills.go`",
	},
	{
		Nom:       "repli_sonde_non_lancee_porte_relachee",
		Fait:      "le diagnostic de couverture publie avec le resultat du decodage",
		Mecanisme: "couverture jugee complete : la sonde a porte relachee n'est pas lancee et Probe reste nil — indiscernable d'une sonde a zero",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgKillsource + "decode.go",
			Ancre:   "if cov.Covered < cov.RealPairs {",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot de cloture du diagnostic killsource (hors famille 1.9)",
		CritereRetrait:  "Probe distingue « non lancee » de « lancee, zero trouve » ; le repli disparait avec l'ambiguite",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_libelle_de_source_autres",
		Fait:      "le libelle publie d'une source de mort",
		Mecanisme: "nom vide ou non publiable : la chaine « Autres » est publiee a sa place",
		Condition: CondSectionAbsente,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgKillsource + "label.go",
			Ancre:   "if l.Name != \"\" && l.Publishable() {",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "completion du catalogue de sources (206 tags sur 468 concernes, audit 0.E)",
		CritereRetrait:  "0 libelle « Autres » publie sur le parc",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_xuid_vide_pour_nom_inconnu",
		Fait:      "le xuid de la victime d'une mort ecrite en base",
		Mecanisme: "le nom ne se resout dans aucune table : un xuid VIDE est rendu et la mort est ecrite sans xuid de victime",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgKillcollector + "identities.go",
			Ancre:   "return \"\", nom",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.8 porte jusqu'a l'ecriture (la table du film nomme les joueurs a zero mort)",
		CritereRetrait:  "0 ligne de `match_deaths` sans xuid de victime sur le parc",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_homonymes_sans_xuid",
		Fait:      "le xuid des participants qui portent le MEME gamertag dans un match",
		Mecanisme: "aucun des deux n'est retenu — ecrire les morts de l'un sous le xuid de l'autre serait pire",
		Condition: CondContradiction,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgKillcollector + "roster.go",
			Ancre:   "ambigus := map[string]bool{}",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.8 porte : la table du film donne l'index, pas le nom, donc l'homonymie cesse d'etre un obstacle",
		CritereRetrait:  "0 match a homonymes non resolus sur le parc ; le repli est SAIN, c'est son silence qui ne l'est pas",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_indice_en_collision_jete",
		Fait:      "le joueur d'un indice de replication, pour les tirs et les touches",
		Mecanisme: "deux xuids sur le meme indice : les DEUX sont jetes",
		Condition: CondContradiction,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgKillcollector + "shots.go",
			Ancre:   "out[pi] = \"\"",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "lot 1.8 porte aux tirs et aux touches (D3 (1.8) : cette voie sert `match_weapon_shots` et `match_weapon_accuracy`)",
		CritereRetrait:  "l'indice vient de la table du film ; 0 collision sur le parc",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_premiere_occurrence_sans_concordance",
		Fait:      "quel motif de xuid retenir quand un chunk en porte plusieurs",
		Mecanisme: "la PREMIERE occurrence gagne, sans exiger que les suivantes concordent",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgKillcollector + "shots.go",
			Ancre:   "continue // premiere occurrence gagnante",
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "lot 1.8 porte aux tirs et aux touches",
		// CONTRASTE MESURÉ par l'audit 0.E : `replay/player_index.go` REFUSE de publier sur
		// désaccord, ce chemin-ci retient la première valeur vue.
		CritereRetrait:  "l'indice vient de la table du film ; la voie par motifs est retiree avec ses tests",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_precision_par_arme_passe_sautee",
		Fait:      "la precision par arme d'un match",
		Mecanisme: "cache de films non configure : toute la passe est sautee, en best-effort silencieux",
		Condition: CondSectionAbsente,
		Ordre:     OrdreSansLecture,
		Sites: []Site{{
			Fichier: pkgKillcollector + "hits.go",
			Ancre:   "numerateur non configure (chemin live sans cache disque)",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "aucune (configuration d'exploitation) ; le COMPTE est ce qui manque",
		CritereRetrait:  "un compteur expvar dit combien de matchs passent sans numerateur film ; retrait sans objet",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_identite_pont_par_morts",
		Fait:      "le nom d'un corps de bipede, quand le lien direct n'a pas pu etre lu",
		Mecanisme: "les creations de bipede sont illisibles (TOUTE erreur, y compris transitoire) : le registre retombe sur le pont par le fil des morts",
		Condition: CondSectionAbsente,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgKillcollector + "positions.go",
			Ancre:   "creations de bipede illisibles",
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "lot 1.6 (le registre d'identite prend la table du film comme lien direct)",
		// DÉFAUT DÉJÀ MESURÉ (audit 0.E) : deux `slog.Warn`, aucun expvar. Une erreur
		// transitoire dégrade silencieusement l'identité de TOUT un match.
		CritereRetrait:  "0 degradation sur le pont par morts dans les journaux du parc ; le lien direct couvre 100 % des corps",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
	{
		Nom:       "repli_coequipiers_partis_constante_nulle",
		Fait:      "le nombre de coequipiers PARTIS a l'instant d'une mort (`teammates_left`)",
		Mecanisme: "la colonne est ecrite a 0 sur toutes les lignes depuis le retrait de son producteur, indiscernable d'une mesure",
		Condition: CondInconditionnel,
		Ordre:     OrdreSansLecture,
		Sites: []Site{{
			Fichier: pkgKillcollector + "isolation_facts.go",
			Ancre:   "TeammatesLeft:  0,",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "retrait de la colonne (append-only, ADR 0026 : la colonne reste, c'est son ECRITURE qui doit devenir nulle explicite)",
		CritereRetrait:  "la colonne cesse d'etre ecrite, ou un producteur la remplit ; un zero constant ne se distingue d'une mesure par rien",
		CompteurBranche: false,
		CibleComptage:   comptageFamille19,
	},
}
