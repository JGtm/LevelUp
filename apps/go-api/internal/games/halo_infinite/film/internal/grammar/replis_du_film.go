package grammar

// replis_du_film.go — LE RAPPORT DES REPLIS DE `grammar` ET DE `profile`, PORTE PAR LE CONTEXTE DU
// FILM (lot J8.7 du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, decision 1 du superviseur).
//
// # POURQUOI UN RAPPORT EN DONNEES, ET PAS `Declenche` AU SITE
//
// Le registre des replis vit dans `facts/fallback`, AU-DESSUS de cette couche : `grammar` et
// `profile` ne peuvent pas l importer (ratchet `archlint/film_layers_deps_test.go`, ordre
// `source -> profile -> grammar -> facts -> replay`). Leurs replis se comptent donc ICI, en entiers
// NOMMES, sur le contexte du film ; `replay` les verse au compteur de la cuisson par UNE fonction en
// table (`replay/versement_des_replis.go`), entree du registre -> champ de ce type.
//
// # UN CHAMP PAR REPLI, ET LE NOM DU REPLI EN COMMENTAIRE
//
// Le champ est la SOURCE que la table de versement lit ; son commentaire nomme l entree du registre
// qu il alimente. Un champ neuf sans ligne de versement ne se publie nulle part : le test
// `replay.TestChaqueChampDuRapportDeGrammaireEstVerse` le fait rougir.
//
// # CE QUE LE RAPPORT N EST PAS
//
// Il ne DECIDE rien et ne change aucune consommation de bits : un decodage avec ou sans lui rend les
// memes octets. Il n est pas persiste avec les faits : la cuisson le verse au compteur AVANT de
// capturer le rapport de replis du balayage, et c est ce rapport-la qui voyage (`FilmFactsFile`).
// Un contexte vit dans UNE goroutine, comme toutes ses derivations memorisees : pas de verrou.

// ComptesDesReplis compte, pour UN film, les declenchements des replis que `grammar` et `profile`
// decident. Zero = le repli ne s est pas declenche SUR CE CONTEXTE.
type ComptesDesReplis struct {
	// ChunksDeReplicationSautes : `repli_chunk_de_replication_saute` — chunks de replication
	// illisibles sautes par la table d index de joueur (`player_index.go`).
	ChunksDeReplicationSautes int
	// TempsFortsAuDernierNumero : `repli_temps_forts_dernier_numero` — fil des morts lu au morceau de
	// plus grand numero faute de manifeste type (`deaths_source.go`).
	TempsFortsAuDernierNumero int
	// LargeursMPPCalibrees : `repli_largeurs_mpp_calibrees_sur_le_film` — decoupage MPP installe
	// depuis la calibration du film faute de largeur relue au profil (`equipment_placements.go`,
	// `replay/build_ground_weapons.go`).
	LargeursMPPCalibrees int
	// I0PorteEtRegionParDefaut : `repli_i0_porte_et_region_par_defaut` — decoupage d i0 auto-detecte
	// (porte 5 bits, region 0), UNE fois par contexte (`film_context.go`, `offline_biped_band.go`).
	I0PorteEtRegionParDefaut int
	// SlotsBipedesComblees : `repli_bande_bipede_comblee` — slots AJOUTES a la bande bipede par le
	// comblement de [min, max], sommes sur les releves du contexte (`offline_biped_band.go`).
	SlotsBipedesComblees int
	// LargeursMondeParDefaut : `repli_largeurs_monde_par_defaut_conservees` — decoupage sans largeur
	// d axe : le defaut des objets du monde est conserve (`profil_balayage.go`).
	LargeursMondeParDefaut int
	// IndexDeRegionLargeurUn : `repli_index_de_region_largeur_un` — porte trop courte pour un index
	// de region : la largeur d index en place est conservee (`profil_balayage.go`).
	IndexDeRegionLargeurUn int
	// LargeursMPPParDefaut : `repli_largeurs_mpp_par_defaut` — balayage de socles ou de vehicules
	// fait aux largeurs MPP de l invariant (9/5), ni relue ni calibree n ayant ete posee
	// (`profile/mpp_widths.go`, compte par `replay/build_ground_weapons.go`).
	LargeursMPPParDefaut int
	// ChunksApresTrouAbandonnes : `repli_chunks_apres_trou_abandonnes` — chunks de donnees du
	// manifeste abandonnes derriere un trou de numerotation (`film_chunks.go`).
	ChunksApresTrouAbandonnes int
	// AncresSansVieDelta : `repli_ancre_sans_vie_delta_ecartee` — ancres de creation d equipement
	// ecartees de la calibration MPP faute de vie delta (`equipment_creation_width.go`).
	AncresSansVieDelta int
	// RegistreInconnu : `repli_registre_inconnu_sans_lecteur_de_troncature` — registre `chunk_00`
	// dont l empreinte n est pas celle du binaire de reference, signale sous une cause unique
	// (`registry_fingerprint.go`), UNE fois par contexte.
	RegistreInconnu int
	// AmorceGrenadeDeReference : `repli_amorce_grenade_profil_de_reference` — lancers lus sous
	// l amorce du build de reference, la cle du film n ayant pas de ligne (`grenade_events.go`).
	AmorceGrenadeDeReference int
	// ControleDeCorruptionNonDeclare : `repli_controle_corruption_section_absente` — film sans
	// section d identification : la grammaire garde son invariant, UNE fois par contexte
	// (`controle_corruption_du_film.go`).
	ControleDeCorruptionNonDeclare int
	// LocalisationsALargeurLibre : `repli_localisation_largeur_libre`, site de la marche des morts
	// d objet — paquets a evenements localises par la seconde passe a largeur libre
	// ([LocaliserBoucleDeRecords], `localisateur.go`). Le site de `killsource` se compte chez lui.
	LocalisationsALargeurLibre int
}

// Plus rend la somme champ a champ des deux rapports.
//
// ELLE NOMME CHAQUE CHAMP, ET C EST TENU : `TestPlusSommeChaqueChampDuRapport` parcourt le type par
// reflexion — un champ ajoute au type et oublie ici rougit.
func (r ComptesDesReplis) Plus(d ComptesDesReplis) ComptesDesReplis {
	return ComptesDesReplis{
		ChunksDeReplicationSautes:      r.ChunksDeReplicationSautes + d.ChunksDeReplicationSautes,
		TempsFortsAuDernierNumero:      r.TempsFortsAuDernierNumero + d.TempsFortsAuDernierNumero,
		LargeursMPPCalibrees:           r.LargeursMPPCalibrees + d.LargeursMPPCalibrees,
		I0PorteEtRegionParDefaut:       r.I0PorteEtRegionParDefaut + d.I0PorteEtRegionParDefaut,
		SlotsBipedesComblees:           r.SlotsBipedesComblees + d.SlotsBipedesComblees,
		LargeursMondeParDefaut:         r.LargeursMondeParDefaut + d.LargeursMondeParDefaut,
		IndexDeRegionLargeurUn:         r.IndexDeRegionLargeurUn + d.IndexDeRegionLargeurUn,
		LargeursMPPParDefaut:           r.LargeursMPPParDefaut + d.LargeursMPPParDefaut,
		ChunksApresTrouAbandonnes:      r.ChunksApresTrouAbandonnes + d.ChunksApresTrouAbandonnes,
		AncresSansVieDelta:             r.AncresSansVieDelta + d.AncresSansVieDelta,
		RegistreInconnu:                r.RegistreInconnu + d.RegistreInconnu,
		AmorceGrenadeDeReference:       r.AmorceGrenadeDeReference + d.AmorceGrenadeDeReference,
		ControleDeCorruptionNonDeclare: r.ControleDeCorruptionNonDeclare + d.ControleDeCorruptionNonDeclare,
		LocalisationsALargeurLibre:     r.LocalisationsALargeurLibre + d.LocalisationsALargeurLibre,
	}
}

// replisDuContexte est l etat que le contexte porte pour son rapport : les comptes, et les deux
// verdicts qui ne se comptent qu UNE fois par film (decoupage d i0 auto-detecte, registre inconnu).
type replisDuContexte struct {
	comptes ComptesDesReplis
	i0Note  bool
	regNote bool
}

// ComptesDesReplis rend le rapport des replis de ce contexte, PAR VALEUR. Contexte nil : rapport vide.
func (c *FilmContext) ComptesDesReplis() ComptesDesReplis {
	if c == nil {
		return ComptesDesReplis{}
	}
	return c.replis.comptes
}

// NoterReplis ajoute `d` au rapport de ce contexte. Sur sur un contexte nil (ne compte rien).
//
// ELLE EST EXPORTEE POUR LES REGLES DE `grammar` QUE `replay` EXECUTE HORS DU CONTEXTE : la pose des
// largeurs de la carte sur une COPIE du profil de balayage (l installateur de `replay`), et le
// choix des largeurs MPP des socles (`gwWidthsForFilm`, `gwInstallMPPWidths`). Ces sites decident
// un fait de `grammar` ; leur compte rejoint le rapport du film plutot qu un second canal.
func (c *FilmContext) NoterReplis(d ComptesDesReplis) {
	if c == nil {
		return
	}
	c.replis.comptes = c.replis.comptes.Plus(d)
}

// unSi rend 1 quand un repli a decide, 0 sinon — la forme d un verdict par film dans le rapport.
func unSi(decide bool) int {
	if decide {
		return 1
	}
	return 0
}

// noterI0ParDefaut compte `repli_i0_porte_et_region_par_defaut` UNE fois par contexte, quand le
// decoupage d i0 vient de l auto-detection (err nil). Les deux sites qui detectent (le contexte et
// le balayage des positions) decident du MEME fait pour le MEME film : un seul declenchement.
func (c *FilmContext) noterI0ParDefaut(err error) {
	if c == nil || err != nil || c.replis.i0Note {
		return
	}
	c.replis.i0Note = true
	c.NoterReplis(ComptesDesReplis{I0PorteEtRegionParDefaut: 1})
}

// noterRegistre compte `repli_registre_inconnu_sans_lecteur_de_troncature` UNE fois par contexte :
// le registre lu ne porte pas l empreinte du binaire de reference, et le seul signal qui sort
// (`DiagnosticRegistreInconnu`) ne distingue pas un autre build d un tampon tronque.
func (c *FilmContext) noterRegistre(reg *Registry) {
	if c == nil || reg == nil || c.replis.regNote || reg.fingerprint == KnownRegistryFingerprint {
		return
	}
	c.replis.regNote = true
	c.NoterReplis(ComptesDesReplis{RegistreInconnu: 1})
}
