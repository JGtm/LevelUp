package fallback

// registre_killsource.go — les replis du décodeur de morts (`film/facts/killsource/`). Ceux du
// collecteur qui l'écrit en base (`internal/sync/killcollector/`) vivent dans
// `registre_killsource_collecteur.go` depuis le lot J8.7 (2026-09-27, scission de taille).

const (
	pkgKillsource    = "internal/games/halo_infinite/film/internal/facts/killsource/"
	pkgKillcollector = "internal/sync/killcollector/"
)

var registreKillsource = []Repli{
	{
		Nom:       "repli_record_desynchronise_jete",
		Fait:      "quels records d'un paquet entrent dans la marche des morts",
		Mecanisme: "un dead-state de bipede dont le record a rompu dans la marche des trames, meme apres le dead-state, est jete",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{Fichier: pkgKillsource + "walk.go", Ancre: "res.desync++"}, siteDeVersement("NomRecordDesynchroniseJete"), {
			Fichier: pkgKillsource + "walk.go",
			Ancre:   "if !m.Propre {",
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "le registre ECS lu par build : une desynchronisation est une grammaire fausse, pas une donnee ; a defaut, " + retraitRegle4,
		// PIÈGE CONNU (mémoire du chantier véhicules, 2026-09-05) : un filtre `DesyncAt == -1`
		// JETAIT des morts de véhicule réellement lues.
		CritereRetrait:  "0 record desynchronise sur les 8 builds une fois le registre ECS resolu par build",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_deadstate_indice_hors_roster",
		Fait:      "un dead-state lu designe-t-il un joueur du roster",
		Mecanisme: "indice de victime ou de tueur hors [0, nPlay) : rejete par un `continue` nu",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{Fichier: pkgKillsource + "walk.go", Ancre: "res.horsRoster++"}, siteDeVersement("NomDeadstateIndiceHorsRoster"), {
			Fichier: pkgKillsource + "walk.go",
			Ancre:   "if d.dead.EnumA < 0 || int(d.dead.EnumA) >= r.nPlay {",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "le kill feed prend la table du film jusqu au rejet ; a defaut, " + retraitRegle4,
		CritereRetrait:  "0 rejet pour indice hors roster sur les 8 builds ; un indice hors domaine est un defaut de largeur, pas une donnee",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_deadstate_categorie_hors_enum",
		Fait:      "la categorie de mort d'un dead-state",
		Mecanisme: "valeur hors de l'enumeration connue (> 9) : le dead-state est rejete par un `continue` nu",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{Fichier: pkgKillsource + "walk.go", Ancre: "res.horsEnum++"}, siteDeVersement("NomDeadstateCategorieHorsEnum"), {
			Fichier: pkgKillsource + "walk.go",
			Ancre:   "if d.dead.Val0c > 9 {",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "l enumeration des categories lue au profil du build ; a defaut, " + retraitRegle4,
		CritereRetrait:  "0 categorie hors enumeration sur les 8 builds",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_localisation_largeur_libre",
		Fait:      "ou commencent les records d'un paquet dont la signature de largeur a echoue",
		Mecanisme: "seconde passe a LARGEUR LIBRE, essayee seulement apres l'echec de la signature",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{Fichier: pkgKillsource + "walk.go", Ancre: "largeurLibre: lus.LargeurLibre}"}, {
			// Le repli lui-meme : le localisateur unique de `grammar`, appele par les deux sites
			// qui lisent les morts (killsource ci-dessus ; le canal des morts de la marche des
			// trames, sur les listes que la cuisson n a pas localisees, ci-dessous).
			Fichier: pkgFilmdec + "localisateur.go",
			Ancre:   "func marchLocateFallback(pay []byte, w *World, cfg FrameConfig) int {",
		}, {
			Fichier: pkgFilmdec + "canal_des_morts.go",
			Ancre:   "c.largeurLibre += unSi(libre)",
		}, {
			// COMPTE du site de `grammar` au rapport du contexte de film.
			Fichier: pkgFilmdec + "movement_states.go",
			Ancre:   "fc.NoterReplis(ComptesDesReplis{LocalisationsALargeurLibre: morts.largeurLibre})",
		}, siteDeVersement("NomLocalisationLargeurLibre")},
		DatePose:     dateAudit0E,
		CibleRetrait: "les largeurs calibrees par la carte et le build ; a defaut, " + retraitRegle4,
		// Ce repli-ci porte DÉJÀ son nom dans le code (`marchLocateFallback`) :
		// c'est ce que la convention du garde-rail exige, et il entre au registre pour cette
		// raison.
		CritereRetrait:  "0 recours a la largeur libre sur les 8 builds une fois les largeurs prises au profil",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_roster_nom_invente",
		Fait:      "la liste de noms sur laquelle la bijection indice -> joueur se resout",
		Mecanisme: "moins de noms que de sieges : des noms ?N sont fabriques pour rendre le probleme d'affectation carre",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{Fichier: pkgKillsource + "roster.go", Ancre: "r.nomsInventes = max(0, r.nPlay-len(r.names))"}, siteDeVersement("NomRosterNomInvente"), {
			Fichier: pkgKillsource + "roster.go",
			Ancre:   "r.names = append(r.names, fmt.Sprintf(\"?%d\", len(r.names)))",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "la table du film donne le lien indice -> joueur : un nom fabrique ne sert plus ; a defaut, " + retraitRegle4,
		CritereRetrait:  "0 nom ?N fabrique sur les 8 builds quand la table du film est lue",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_bijection_hongroise_du_feed",
		Fait:      "le lien indice de replication -> joueur, pour les indices que la table du film ne donne pas",
		Mecanisme: "affectation hongroise sur les votes du kill feed, puis raffinement local",
		Condition: CondSectionAbsente,
		Ordre:     OrdreApresLecture,
		Sites: []Site{siteDeVersement("NomBijectionHongroiseDuFeed"), {
			Fichier: pkgKillsource + "bijection.go",
			Ancre:   "r.table.Inferred = len(free)",
		}},
		DatePose:     "2026-09-14",
		CibleRetrait: "la table du film couvre 100 % des indices ; a defaut, " + retraitRegle4,
		// Repli POSÉ ET COMPTÉ par le lot 1.8 : `RosterTable.Inferred` est son compteur, publié
		// dans les statistiques de collecte. Il entre au registre pour que sa condition de
		// retrait soit lisible au même endroit que les autres.
		CritereRetrait:  "RosterTable.Inferred a 0 sur le parc apres la recuisson de cloture M1",
		CompteurBranche: true,
		// CIBLE REECRITE LE 2026-09-16 (revue de jalon M1) : elle nommait le lot 1.9.3, fusionne
		// le 2026-09-15 sans avoir publie ce compte sous son nom de registre.
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
		Sites: []Site{siteDeVersement("NomCoupleRecolleSurLeVoisin"), {
			Fichier: pkgKillsource + "feed_couples.go",
			Ancre:   "func (res *resolveurDeCouples) repliRecollageSurLeVoisin(i int) {",
		}},
		DatePose:     dateVague2,
		CibleRetrait: "la table du film couvre 100 % des indices et la chaine d evenements ne s arrete plus ; a defaut, " + retraitRegle4,
		// MESURE DU LOT 1.9.3 (21 films entiers, 8 builds, 14 temoins) : 281 kills sans mort en
		// face, 198 decides par la lecture (198 accords, 0 contradiction), 1 victime BOT nommee,
		// 1 ambigu, 81 muets. Les 81 muets sont le compte a faire tomber ; 46 d'entre eux
		// viennent des trois films sans table de joueurs exploitable (`a349fea8`, `a521164d`,
		// `50247b26`), les autres d'une chaine d'evenements qui s'arrete avant le kill-event.
		CritereRetrait:  "CoupleStats.Recolles a 0 sur les 8 builds et sur le corpus gate",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_chunk_du_pied_par_argmax",
		Fait:      "quel chunk d'un film est le PIED (kill feed, recompenses, evenements de mode)",
		Mecanisme: "chaque chunk est parse en evenements de highlight et celui qui porte le PLUS de kills gagne",
		Condition: CondNonResolu,
		Ordre:     OrdreDevantLaLecture,
		Sites: []Site{{Fichier: pkgKillsource + "replis_du_decodage.go", Ancre: "// `loadKillFeed` designe le pied par argmax a chaque decodage"}, siteDeVersement("NomChunkDuPiedParArgmax"), {
			Fichier: pkgFilmdec + "fil_des_kills.go",
			Ancre:   "if n := compterLesKills(evs); n > plus {",
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "le chunk du pied pris au type du manifeste ; a defaut, " + retraitRegle4,
		// ORDRE `devant_la_lecture` : le TYPE du chunk est porté par `source.Film.Meta()` et
		// déjà lu par ce patron (`objectives/extract.go`) ; l'argmax décide sans le
		// consulter. RÉSERVE du plan : le manifeste est un descripteur EXTERNE, l'argmax restera
		// donc en repli COMPTÉ après la conversion.
		CritereRetrait:  "le type du manifeste decide ; l'argmax ne se declenche que sur un film sans manifeste, et son compte le dit",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_chaine_evenement_code_non_modelise",
		Fait:      "la longueur du corps d'un evenement, donc la suite de la chaine",
		Mecanisme: "code hors des 28 modelises (95 codes sur 123) ou cfgIdx non resolu : la chaine s'arrete",
		Condition: CondLectureNonPortee,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{Fichier: pkgKillsource + "eventchain.go", Ancre: "arretees += unSi(arretee)"}, {Fichier: pkgKillsource + "assist.go", Ancre: "s.chainesArretees += arretees"}, siteDeVersement("NomChaineEvenementCodeNonModelise"),
			{Fichier: pkgKillsource + "eventbody.go", Ancre: "func evBody(r *curseurEv, code int, gate15 bool) bool {"},
			{Fichier: pkgKillsource + "eventchain.go", Ancre: "if c < 0 || c >= len(presRange) {"},
		},
		DatePose:        dateAudit0E,
		CibleRetrait:    "les codes d evenement manquants portes, archetype par archetype, ou leur longueur lue chez l ecrivain ; a defaut, " + retraitRegle4,
		CritereRetrait:  "les 95 codes non modelises portes, ou leur longueur lue chez l'ecrivain ; 0 arret de chaine sur les 8 builds",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_type_de_chunk_perdu_du_manifeste",
		Fait:      "le type de chaque chunk, dans le vocabulaire du decodeur de morts",
		Mecanisme: "la traduction du film ne recopie PAS Meta() : le type du manifeste est perdu, et l'argmax le remplace en aval",
		Condition: CondInconditionnel,
		Ordre:     OrdreDevantLaLecture,
		Sites: []Site{{Fichier: pkgKillsource + "replis_du_decodage.go", Ancre: "// `loadFilm` jette le type du manifeste a chaque decodage"}, siteDeVersement("NomTypeDeChunkPerduDuManifeste"), {
			Fichier: pkgKillsource + "chunks.go",
			// L ANCRE EST LA TRADUCTION ELLE-MEME depuis le lot 2.1.4 : c est LA ligne ou
			// `Meta()` est perdu, donc celle que ce repli decrit. Elle pointait jusque-la sur
			// la lecture de la version majeure, voisine de hasard, qui a bouge quand cette
			// version est passee au profil du film.
			Ancre: "f := &film{src: src, packets: packetsOf(src)}",
		}},
		DatePose:     dateAudit0E,
		CibleRetrait: "le type du manifeste voyage jusqu au decodeur (cause racine de l argmax du pied) ; a defaut, " + retraitRegle4,
		// DÉFAUT DÉJÀ MESURÉ (audit 0.E) : c'est la cause racine de la ligne A7. Le type existe
		// à l'entrée du décodeur et il est jeté à la traduction.
		CritereRetrait:  "Meta() voyage jusqu'au decodeur ; repli_chunk_du_pied_par_argmax se declenche alors seulement sans manifeste",
		CompteurBranche: true,
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
		Sites: []Site{siteDeVersement("NomMortNonRevendiqueeLaPlusProche"),
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
		CibleRetrait: "le film nomme l instant d une mort non revendiquee autrement que par le kill feed (composants manquants portes) ; a defaut, " + retraitRegle4,
		// MESURE DU LOT 1.9.7 (21 films entiers, 8 builds) : 17 morts non revendiquees, dont
		// 17 SANS aucune identite en face. Le repli les sert toutes, et le compte le dit.
		CritereRetrait:  "ApparStats.NonRevendiqueeFenetre a 0 sur les 8 builds et sur le corpus gate",
		CompteurBranche: true,
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
		Sites: []Site{siteDeVersement("NomMortDeBotPremierCandidat"),
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
		CibleRetrait: "la chaine d evenements atteint le kill-event 85 des morts de bot (composants manquants portes) ; a defaut, " + retraitRegle4,
		// MESURE DU LOT 1.9.7 (21 films entiers, 8 builds) : 6 morts DE bot et 4 morts PAR un
		// bot apparieees, dont UNE SEULE par l'identite de paquet — les 9 autres n'ont aucune
		// identite en face.
		CritereRetrait:  "ApparStats.BotFenetre a 0 sur les 8 builds et sur le corpus gate",
		CompteurBranche: true,
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
		Sites: []Site{siteDeVersement("NomAppariementParFenetreTemporelle"),
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
		CibleRetrait: "tout instant du kill feed porte un kill-event 85 aux deux indices EPINGLES ; a defaut, " + retraitRegle4,
		// MESURE DU LOT 1.9.7 (21 films entiers, 8 builds, 14 temoins du corpus gate) : 2 899
		// appariements, 2 205 a identite EGALE des deux cotes, 2 a identite DIFFERENTE, 692 SANS
		// identite du cote feed. L'appariement par identite seule rend 2 204 accords et ZERO
		// desaccord : la lecture ne contredit jamais la fenetre, elle la remplace. Les 692 sans
		// identite sont le compte a faire tomber, et ils ont la meme cause que les 81 muets du
		// lot 1.9.3 (chaine d'evenements qui s'arrete avant le kill-event, table de joueurs
		// refusee sur les films sans section d'identification).
		CritereRetrait: "ApparStats.Fenetre et AssistStats.ParLaFenetre a 0 sur les 8 builds et " +
			"sur le corpus gate — le site de l'assistant y est DEJA a zero",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_sonde_non_lancee_porte_relachee",
		Fait:      "le diagnostic de couverture publie avec le resultat du decodage",
		Mecanisme: "couverture jugee complete : la sonde a porte relachee n'est pas lancee et Probe reste nil — indiscernable d'une sonde a zero",
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{Fichier: pkgKillsource + "replis_du_decodage.go", Ancre: "unSi(!sondeLancee)"}, siteDeVersement("NomSondeNonLanceePorteRelachee"), {
			Fichier: pkgKillsource + "decode.go",
			Ancre:   "if cov.Covered < cov.RealPairs {",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "la cloture du diagnostic killsource (sonde non lancee distinguee de sonde lancee a zero) ; a defaut, " + retraitRegle4,
		CritereRetrait:  "Probe distingue « non lancee » de « lancee, zero trouve » ; le repli disparait avec l'ambiguite",
		CompteurBranche: true,
	},
	{
		Nom:       "repli_libelle_de_source_autres",
		Fait:      "le libelle publie d'une source de mort",
		Mecanisme: "nom vide ou non publiable : la chaine « Autres » est publiee a sa place",
		Condition: CondSectionAbsente,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{Fichier: pkgKillsource + "replis_du_decodage.go", Ancre: "r.LibellesAutres += unSi(!kills[i].Source.Named)"}, siteDeVersement("NomLibelleDeSourceAutres"), {
			Fichier: pkgKillsource + "label.go",
			Ancre:   "if l.Name != \"\" && l.Publishable() {",
		}},
		DatePose:        dateAudit0E,
		CibleRetrait:    "la completion du catalogue de sources (206 tags sur 468 concernes a l audit du 2026-09-13) ; a defaut, " + retraitRegle4,
		CritereRetrait:  "0 libelle « Autres » publie sur le parc",
		CompteurBranche: true,
	},
}
