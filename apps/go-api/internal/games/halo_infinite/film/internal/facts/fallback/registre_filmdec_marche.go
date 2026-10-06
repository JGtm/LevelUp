package fallback

// registre_filmdec_marche.go — les replis de la MARCHE des images-cles et des trames de `grammar`
// (election d ancre, generation vivante inconnue, liaison par anticipation, debut de liste ferme
// au bit).
//
// SCINDE DE `registre_filmdec.go` AU LOT J8.7 (2026-09-27) PAR DEPLACEMENT PUR : le fichier passait
// 500 lignes (seuil du depot) avec les sites de compte du sous-lot `grammar`. La coupe suit la
// responsabilite — la-bas les INFERENCES de largeurs, de bandes et de profil ; ici les replis que
// les lots M3, M4b, J5 et J8.1 ont poses sur la MARCHE du flux —, et les trois gestes d une famille
// neuve sont faits : [Tranches], `famillesAttendues` et `plancherTranches` (registre_test.go).

var registreFilmdecMarche = []Repli{
	{
		Nom:  "repli_ancre_d_image_cle_par_election",
		Fait: "le record SUIVANT de la table d image-cle, quand aucun voisin immediat (slot+1, generation 1) ne suit et qu aucun en-tete exact de bipede ne precede le candidat retenu",
		Mecanisme: "election sur la fenetre de 120 000 bits : consecutif d abord, puis generation basse, puis SLOT BAS, puis bit bas (`kfCand.betterThan`) ; " +
			"depuis le lot D-fix (2026-09-24), l elu qu un record PROUVE par la grammaire du film contredit (ordre des bits et des slots inverse) est refuse et l election reprend sans lui (`grammar/keyframe_world_preuve.go`, compte `coverage.keyframes.refutations`)",
		// LECTURE NON PORTEE : le film ECRIT la table comme une chaine (`FUN_142e2bfd0` enchaine
		// les entrees, une par entite vivante) et la marche deterministe qui la suivrait
		// (`WalkKeyframeRecords`) ne ferme pas encore tous les archetypes. C est une dette nommee.
		Condition: CondLectureNonPortee,
		// APRES LECTURE : le voisin immediat et le recalage sur l en-tete exact d un bipede sont
		// tentes d abord ; l election n entre que si les deux se taisent.
		Ordre: OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgFilmdec + "keyframe_world.go",
			Ancre:   "iss.dec = kfElection // repli nomme `repli_ancre_d_image_cle_par_election`",
		}, {
			Fichier: "internal/games/halo_infinite/film/replay/film_scan.go",
			Ancre:   "s.opt.Fallbacks.DeclencheN(fallback.NomAncreDImageCleParElection, marche.Elections)",
		}},
		DatePose: "2026-09-23",
		// POSE PAR LE LOT M3.1 DE LA CAMPAGNE « RETOURS REJEU » : l election etait la regle
		// UNIQUE du balayeur, muette ; elle devient le repli d une lecture (voisin, recalage) et se
		// compte. Son defaut est MESURE (sonde P2, 81c02726 morceau 9 et a0c36016 morceau 2 : une
		// fausse ancre de slot bas, prise dans le corps du dernier bipede, elue devant les vrais
		// bipedes) ; le recalage le ferme pour les bipedes, pas pour les autres archetypes
		// (minibobine bcb6d393, image-cle 0 : le record ti=9 slot 1297 perd contre la fausse
		// ancre 192/ti 1 de son propre corps). LE LOT D-fix (2026-09-24) FERME CE CAS SANS SEUIL :
		// la table est a slots croissants, donc un candidat que la grammaire du film PROUVE (sa
		// marche d etat complet, contenu compris, ferme sur l en-tete valide suivant) interdit
		// tout elu qui contredit cet ordre avec lui — la fausse ancre 192 (dans le corps du
		// record 1298) est refusee devant les records 1280..1298 prouves, et l election reprend.
		// L election reste le repli : elle decide encore la ou aucun record prouve ne la contredit.
		CibleRetrait:    "la marche deterministe (`WalkKeyframeRecords`, cadre d etat complet de l ecrivain) fermant tous les archetypes des bobines par build",
		CritereRetrait:  "`KeyframeClosure` a 100 % sur les sept bobines par build ET 0 election comptee sur le corpus du gate de rejeu",
		CompteurBranche: true,
	},
	{
		Nom:  "repli_generation_vivante_inconnue_tag1",
		Fait: "la generation du handle sous laquelle un record delta bipede est lu (positions, huit canaux delta, visee seule, recuperation d equipement), sur un slot dont aucune generation n est connue",
		Mecanisme: "le slot n est designe ni par un record de creation de bipede ni par un record ti=35 d image-cle : seule la generation 1 est acceptee (le filtre `RequireTag1` d avant le lot J5.2) ; " +
			"compte = slots distincts des positions publiees dans ce cas",
		// NON RESOLU : les deux lectures qui designent une vie ont tourne sur tout le film et n ont
		// rien rendu pour ce slot (un corps cree puis detruit entre deux images-cles, dont la
		// creation n a pas ete acceptee). Mesure J5.0 (19 films, 2026-09-27) : aucune position de
		// production dont la vie (slot, 1) serait inconnue — le repli est attendu a zero sur le parc.
		Condition: CondNonResolu,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgFilmdec + "generations_vivantes.go",
			Ancre:   "return h.Gen == generationDuRepli",
		}, {
			Fichier: pkgFilmdec + "generations_vivantes.go",
			Ancre:   "func (g *GenerationsVivantes) SlotsEnRepli(pos []BipedPosition) int {",
		}, {
			Fichier: "internal/games/halo_infinite/film/replay/film_scan.go",
			Ancre:   "s.opt.Fallbacks.DeclencheN(fallback.NomGenerationVivanteInconnueTag1,",
		}},
		DatePose:        date0927,
		CibleRetrait:    "J11 (gates de corpus du plan PLAN_SUITE_AUDIT_DECODEUR_FILM) : retrait si le compte est a zero sur le corpus, sinon lecture des vies manquantes (creations refusees par la signature)",
		CritereRetrait:  "0 slot compte sur le corpus du gate de rejeu ET sur le parc re-decode",
		CompteurBranche: true,
	},
	{
		// LOT J8.1 DU PLAN DE SUITE D AUDIT (2026-09-27), CONSTAT GA1-2 : le repli du lot 5.23
		// (2026-09-22) decidait hors registre et ne se comptait nulle part — sur `bfecd02b`, les
		// stances passent de 616 a 841 par lui sans que l artefact le dise. Ses identifiants
		// portent desormais `Repli`, pour que le ratchet de vocabulaire les voie.
		//
		// LA TABLE ANTICIPEE A UN SECOND USAGE QUI N EST PAS UN REPLI : `SlotDeLArchetype`
		// (`debut_de_liste.go`, reprise M4b) lit la BANDE d un archetype dans les images-cles pour
		// proposer un candidat de debut de liste, et ce candidat n est retenu que si la chaine de
		// records qu il ouvre finit AU BIT PRES sur le debut localise. C est une lecture prouvee
		// par le flux, pas une decision a la place d une lecture : elle n entre pas ici.
		Nom:  "repli_liaison_par_anticipation",
		Fait: "l archetype d un slot jamais lie dont un record delta arrive (une entite nee en milieu de chunk, que ni l image-cle de son chunk ni la table de datums ne declarent)",
		Mecanisme: "la table anticipee des images-cles du film rend l archetype que la PREMIERE image-cle STRICTEMENT POSTERIEURE au chunk courant donne a l eid entier (slot et tete) ; " +
			"le slot est lie comme une liaison de datum (Soft, generation et vue inconnues, sans position) et son corps est lu ; compte = liaisons posees",
		// LECTURE NON PORTEE : le film ecrit la naissance (un record NEW), et le decodeur ne le lit
		// pas la ou l entite nait (0,0 % dans l image-cle du chunk, 0 sur 23 325 rejets dans le
		// bloc de type 1 — lots 5.20.2 et 5.21.2).
		Condition: CondLectureNonPortee,
		Ordre:     OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgFilmdec + "world.go",
			Ancre:   "func (w *World) LierParRepliDAnticipation(id uint32) (uint32, bool) {",
		}, {
			// LE MONDE NE COMPTE PLUS (lot J8.7, 2026-09-27) : son accesseur
			// `LiaisonsDuRepliDAnticipation` et sa table `anticipations` comptaient le MEME fait que
			// l observation ci-dessous ; la source unique est l observation de la marche.
			Fichier: pkgFilmdec + "frame_infer.go",
			Ancre:   "ti, anticipe := w.LierParRepliDAnticipation(id)",
		}, {
			Fichier: pkgFilmdec + "observateur.go",
			Ancre:   "LiaisonsParRepliDAnticipation map[uint32]int",
		}, {
			Fichier: pkgFilmdec + "observateur.go",
			Ancre:   "func (o *Observation) compterLiaisonParRepliDAnticipation(ti uint32) {",
		}, {
			// L ANCRE EST LA TRANSMISSION, PAS LA DECLARATION : elle nomme le champ et la methode
			// (le ratchet de vocabulaire les voit couverts), et l effacer fait rougir la direction
			// (B) — le seul maillon que le test du paquet ne tient pas.
			Fichier: pkgFilmdec + "movement_states.go",
			Ancre:   "m.LiaisonsParRepliDAnticipation = sc.liaisonsDuRepliDAnticipation()",
		}, {
			Fichier: "internal/games/halo_infinite/film/replay/film_scan_mouvement.go",
			Ancre:   "s.opt.Fallbacks.DeclencheN(fallback.NomLiaisonParAnticipation, m.LiaisonsParRepliDAnticipation)",
		}},
		DatePose:        date0927,
		CibleRetrait:    "la lecture du record de naissance d une entite nee en milieu de chunk ; a defaut, retrait au jalon suivant si le compte est nul au corpus gate de J11 (regle 4 de D-10, 2026-09-27)",
		CritereRetrait:  "record de naissance lu ET 0 liaison par anticipation comptee sur le corpus du gate de rejeu",
		CompteurBranche: true,
	},
	{
		Nom:  "repli_debut_de_liste_ferme_au_bit",
		Fait: "le debut de la liste d evenements d un paquet delta que le localisateur strict ne localise pas, quand aucun candidat NEW de tete ne FERME le paquet",
		Mecanisme: "le premier candidat d ou la marche complete ferme le paquet AU BIT PRES seulement (reste nul, une regle de l ecrivain contredite) est pris comme debut : " +
			"les records sont lus sans modifier le monde (aucun NEW lie, aucun DEL applique), le paquet reste non ferme (trou du tir continu) ; compte = listes dont le debut est pris a ce rang",
		// LECTURE NON PORTEE : le film ecrit la liste depuis un record NEW dont l en-tete est lu
		// juste et le corps mal lu (masque au-dela de l archetype) ; la lecture de ce corps ferait
		// fermer le paquet au premier rang.
		Condition: CondLectureNonPortee,
		// APRES LECTURE : le premier rang (un candidat d ou le paquet ferme, regles de l ecrivain
		// tenues) est essaye sur tous les candidats avant que ce rang entre.
		Ordre: OrdreApresLecture,
		Sites: []Site{{
			Fichier: pkgFilmdec + "debut_de_liste.go",
			Ancre:   "cfg.Obs.compterDebutDeListeParRepliFermeAuBit()",
		}, {
			Fichier: pkgFilmdec + "observateur.go",
			Ancre:   "DebutsDeListeParRepliFermeAuBit int",
		}, {
			Fichier: pkgFilmdec + "observateur.go",
			Ancre:   "func (o *Observation) compterDebutDeListeParRepliFermeAuBit() {",
		}, {
			Fichier: pkgFilmdec + "movement_states.go",
			Ancre:   "m.DebutsDeListeParRepliFermeAuBit = sc.obs.DebutsDeListeParRepliFermeAuBit",
		}, {
			Fichier: "internal/games/halo_infinite/film/replay/film_scan_mouvement.go",
			Ancre:   "s.opt.Fallbacks.DeclencheN(fallback.NomDebutDeListeFermeAuBit, m.DebutsDeListeParRepliFermeAuBit)",
		}},
		DatePose:        "2026-10-02",
		CibleRetrait:    "la lecture du corps des records NEW de tete que ce rang garde (decouverte D-L0-2 de la campagne de grammaire), qui fait fermer ces paquets au premier rang",
		CritereRetrait:  "0 liste prise a ce rang comptee sur le corpus du gate de rejeu ET aucun paquet sain perdu a la carte de fermeture sans ce rang",
		CompteurBranche: true,
	},
}
