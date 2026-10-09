package fallback

// registre_filmdec_images_cles.go — les replis des FENETRES DE BITS des images-cles, passees DERRIERE
// la lecture de l etat complet du bipede par la grammaire (plan
// `.ai/PLAN_RI_ETAT_COMPLET_IMAGES_CLES_2026-10-08.md`, D1.2 ; ADR 0037 IR-6, option A de
// l utilisateur du 2026-10-04).
//
// FAMILLE NEUVE (2026-10-09) : les trois gestes d une famille neuve sont faits — [Tranches],
// `famillesAttendues` et `plancherTranches` (registre_test.go).
//
// LES TROIS REPLIS ONT LE MEME SITE DE DECISION ET LA MEME POPULATION : un record bipede d image-cle
// que la regle d admission refuse (decision de l utilisateur du 2026-10-09, U-1 amendee : record
// ferme OU n(i22) = 4, ET le temoin des armes T1 ET le temoin du jeu de grenades T2), ou dont la
// phase n a pas parcouru le corps (film sans registre), est donne a chaque fenetre ; la grammaire
// compte les records donnes (`keyframe_etats_fenetre.go`), `replay` les verse.

// dateFenetresDerriereLaLecture : le jour ou les fenetres des images-cles passent derriere la lecture.
const dateFenetresDerriereLaLecture = "2026-10-09"

// cibleRetraitFenetresImagesCles et critereRetraitFenetresImagesCles : la date et le critere de
// retrait des trois replis, acceptes par l utilisateur le 2026-10-08 (D1.2.3 du plan).
const (
	cibleRetraitFenetresImagesCles = "2026-12-31 : la lecture de l etat complet du bipede admet les records des formats 20 " +
		"et 21 (valeurs apres i22), de 60ae07c4 (+33 bits, i53), et ceux que des ecarts de fin (multiples de -108, " +
		"en-tetes non vus) ou des arrets (i59, i58) laissent non admis"
	critereRetraitFenetresImagesCles = "part des records bipedes d image-cle non admis <= 5 % sur les 28 films du corpus " +
		"de mesure ET sur le parc a la recuisson ; population a la pose (2026-10-09) : 4 203 non admis sur 10 710 " +
		"(formats 20-21 : 2 006 ; 60ae07c4 ; ecarts de fin multiples de -108 ; autres ecarts non nuls ; arrets i59/i58 ; " +
		"dernier record de paquet sans frontiere ; records sans arme ou a famille hors catalogue refuses par T1)"
)

// siteDesFenetres : le site ou la grammaire compte les trois replis.
var siteDesFenetres = Site{
	Fichier: pkgFilmdec + "keyframe_etats_scan.go",
	Ancre:   "fc.NoterReplis(ComptesDesReplis{FenetresArmesImageCle: a.FenetresArmes,",
}

var registreFilmdecImagesCles = []Repli{
	{
		Nom:  "repli_fenetre_armes_image_cle",
		Fait: "les armes portees par un bipede a une image-cle, quand la grammaire n admet pas sa lecture de l etat complet",
		Mecanisme: "fenetre glissante de 32 bits sur l emprise du record (familles du catalogue, dans l ordre des bits, " +
			"alias non replies), les records admis rendus muets ; compte = records bipedes non admis donnes a la fenetre",
		// LECTURE NON PORTEE : la grammaire lit ce record mais ne prouve pas ses valeurs apres i22.
		Condition: CondLectureNonPortee,
		// APRES LECTURE : la grammaire lit d abord ; la fenetre ne voit que ce qu elle n admet pas.
		Ordre: OrdreApresLecture,
		Sites: []Site{siteDesFenetres, {
			Fichier: pkgFilmdec + "keyframe_etats_fenetre.go",
			Ancre:   "rendu.armes = &types.KeyframeLoadout{Slot: r.Vie.Slot, Families: fs}",
		}, siteDeVersement("NomFenetreArmesImageCle")},
		DatePose:        dateFenetresDerriereLaLecture,
		CibleRetrait:    cibleRetraitFenetresImagesCles,
		CritereRetrait:  critereRetraitFenetresImagesCles,
		CompteurBranche: true,
	},
	{
		Nom:  "repli_fenetre_inventaire_image_cle",
		Fait: "les grenades, la capacite, les munitions, l emplacement degaine et la grenade selectionnee d un bipede a une image-cle, quand la grammaire n admet pas sa lecture de l etat complet",
		Mecanisme: "regles d ancrage R1 a R5 de `inventory_decode.go` (ancre de capacite, motif i22, bloc de munitions " +
			"atterrissant sur la premiere famille, selection bornee par les compteurs) sur la SEULE emprise du record ; " +
			"compte = records bipedes non admis donnes aux regles",
		Condition: CondLectureNonPortee,
		Ordre:     OrdreApresLecture,
		Sites: []Site{siteDesFenetres, {
			Fichier: pkgFilmdec + "keyframe_etats_fenetre.go",
			Ancre:   "if invs := keyframeInventoriesDe(p.Payload, []invRecordSpan{emprise}, c.known, c.grenMax); len(invs) == 1 {",
		}, siteDeVersement("NomFenetreInventaireImageCle")},
		DatePose:        dateFenetresDerriereLaLecture,
		CibleRetrait:    cibleRetraitFenetresImagesCles,
		CritereRetrait:  critereRetraitFenetresImagesCles,
		CompteurBranche: true,
	},
	{
		Nom:  "repli_fenetre_marque_de_portage",
		Fait: "la marque de portage d un bipede a une image-cle, quand la grammaire n admet pas sa lecture de l etat complet",
		Mecanisme: "les quatre vues decalees du motif 0x00010005 dans l emprise du record, du MEME passage de la fenetre " +
			"glissante que les armes ; compte = records bipedes non admis donnes a la fenetre",
		Condition: CondLectureNonPortee,
		Ordre:     OrdreApresLecture,
		Sites: []Site{siteDesFenetres, {
			Fichier: pkgFilmdec + "keyframe_etats_fenetre.go",
			Ancre:   "rendu.marque = marques[r.Debut]",
		}, siteDeVersement("NomFenetreMarqueDePortage")},
		DatePose:        dateFenetresDerriereLaLecture,
		CibleRetrait:    cibleRetraitFenetresImagesCles,
		CritereRetrait:  critereRetraitFenetresImagesCles,
		CompteurBranche: true,
	},
}
