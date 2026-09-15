package fallback

// noms.go — LES NOMS QUE LE CODE CITE.
//
// # POURQUOI DES CONSTANTES, ET PAS LA CHAÎNE AU SITE
//
// Un site de repli appelle `fb.Declenche(fallback.NomXxx)`. Avec une chaîne littérale, une faute
// de frappe passerait la compilation et compterait sous un nom que le registre ne connaît pas —
// le compteur le journaliserait en erreur, mais seulement à l'exécution, et seulement si le
// repli se déclenche. La constante rend la faute impossible à la compilation.
//
// # CE QUI LES TIENT
//
// `TestChaqueNomConstantEstAuRegistre` (registre_test.go) parse CE fichier et exige que chaque
// constante déclarée ici corresponde à une entrée du registre. Une constante orpheline — parce
// qu'un lot de conversion a retiré l'entrée sans retirer la constante — fait rougir.
//
// N'entrent ici que les replis dont le compteur est CÂBLÉ ([Repli.CompteurBranche] vrai). Les
// autres n'ont pas de site qui les cite : leur nom vit dans le registre, et il y suffit.

const (
	// NomPieceEngendreeSansEvenement : `replay/equipment_origin.go`, `origineDeLaPose`.
	NomPieceEngendreeSansEvenement Nom = "repli_piece_engendree_sans_evenement"
	// NomCadreDeMarcheParDefautConserve : `filmdec/object_deaths_calibrate.go`, `calibrateFrameConfig`.
	NomCadreDeMarcheParDefautConserve Nom = "repli_cadre_de_marche_par_defaut_conserve"
	// NomPlafondGrenadeParDefaut : `replay/inventory_decode.go`, `ScanKeyframeInventory`.
	NomPlafondGrenadeParDefaut Nom = "repli_plafond_grenade_par_defaut"
	// NomLargeursAxeParDefautConservees : `replay/world_object_precision.go`.
	NomLargeursAxeParDefautConservees Nom = "repli_largeurs_axe_par_defaut_conservees"
	// NomIdentitePisteMeilleurRecouvrement : `replay/identity.go`, `nameTracksByLives`.
	NomIdentitePisteMeilleurRecouvrement Nom = "repli_identite_piste_meilleur_recouvrement"
	// NomPositionLacherPrendLaPrise : `replay/flag_carries.go`, `attachFlagCarryPositions`.
	NomPositionLacherPrendLaPrise Nom = "repli_position_lacher_prend_la_prise"
	// NomPisteDrapeauSansPontEcartee : `replay/flag_carrier_tracks.go`, `tracksByXUID`.
	NomPisteDrapeauSansPontEcartee Nom = "repli_piste_drapeau_sans_pont_ecartee"
	// NomCollineVotesPeriodeEntiere : `replay/zone_states_hill.go`, `buildDesignatedHills`.
	NomCollineVotesPeriodeEntiere Nom = "repli_colline_votes_periode_entiere"
	// NomCollineDernierIntervalleOuvert : `replay/zone_states_hill.go`, `hillOwnerRuns`.
	NomCollineDernierIntervalleOuvert Nom = "repli_colline_dernier_intervalle_ouvert"
	// NomVieCoupeeAuTrouDeReplication : DEUX sites — `replay/lives_decoupe.go`,
	// `causeDeLaCoupure` (film muet sur les morts du joueur) et `replay/tracks_publication.go`,
	// `bornesDesVies` (table d'index des joueurs vide). Le registre porte la condition de chacun.
	NomVieCoupeeAuTrouDeReplication Nom = "repli_vie_coupee_au_trou_de_replication"
	// NomArmementBombeDebutAZero : `replay/bomb_armings.go`, `buildBombArmings`.
	NomArmementBombeDebutAZero Nom = "repli_armement_bombe_debut_a_zero"
	// NomGesteDernierOccupantDuMatch : `replay/usage_summary_owners.go`, `usageOwners.at` et
	// `usageOwners.atOrJustBefore`.
	NomGesteDernierOccupantDuMatch Nom = "repli_geste_dernier_occupant_du_match"
	// NomGestePremiereVieDuSlot : `replay/usage_summary_owners.go`, `usageOwners.atOrJustBefore`.
	NomGestePremiereVieDuSlot Nom = "repli_geste_premiere_vie_du_slot"
	// NomGardeEquipementNegatifAZero : `replay/usage_summary_outcomes.go`, `deriveUsageKept`.
	NomGardeEquipementNegatifAZero Nom = "repli_garde_equipement_negatif_a_zero"
)
