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
	// NomCollineDernierIntervalleOuvert : `replay/zone_states_hill_owners.go`, `hillOwnerRuns`.
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
	// NomReplayMancheZeroDecretee : `replay/score_timeline.go`, `attachRoundsCoverage`.
	//
	// LE SITE EST DANS `replay` ALORS QUE LE REPLI SE DECLENCHE DANS `analysis` : c'est
	// l'inversion de dependance annoncee par la note d'architecture de `registre_objectifs.go`
	// (D9 — `internal/analysis/` n'importe JAMAIS `internal/games/{slug}/`). `objectives`
	// rend son verdict en donnee pure ([objectives.RoundsDecision]) et l'appelant, qui
	// porte le compteur de la cuisson, le compte.
	NomReplayMancheZeroDecretee Nom = "repli_manche_zero_decretee"
	// NomPlaceDuRemplacantParChainageDEquipe : `replay/sieges.go`, `poserLesSieges` (lot M2.3 ;
	// remplace l'appariement ordinal du lot 1.9.14).
	NomPlaceDuRemplacantParChainageDEquipe Nom = "repli_place_du_remplacant_par_chainage_d_equipe"
	// NomPlaceOuverteSousLaCapaciteEstimee : `replay/sieges.go`, `poserLesSieges` (lot M2.3).
	NomPlaceOuverteSousLaCapaciteEstimee Nom = "repli_place_ouverte_sous_la_capacite_estimee"
	// NomPresenceParEnveloppeDesVies : `replay/sieges.go`, `poserLesSieges` (lot M2.3).
	NomPresenceParEnveloppeDesVies Nom = "repli_presence_par_enveloppe_des_vies"
	// NomPresenceDUneEntreeParSesVies : `replay/sieges.go`, `poserLesSieges` (revue M2-R5,
	// 2026-09-24).
	NomPresenceDUneEntreeParSesVies Nom = "repli_presence_d_une_entree_par_ses_vies"
	// NomBorneDePresenceDiffereeSurDoute : `replay/sieges.go`, `poserLesSieges` (lot D-fix,
	// constat DFIX-R7 de la revue adverse, 2026-09-24).
	NomBorneDePresenceDiffereeSurDoute Nom = "repli_borne_de_presence_differee_sur_doute"
	// NomVieDeBotParRelaisDeLaBase : `replay/successions.go`, `attributeSuccessions` (lot M2.3).
	NomVieDeBotParRelaisDeLaBase Nom = "repli_vie_de_bot_par_relais_de_la_base"
	// NomChassisVehiculeMarqueurNeutre : `replay/document_vehicles_coverage.go`, `tallyVehicleCoverage`.
	NomChassisVehiculeMarqueurNeutre Nom = "repli_chassis_vehicule_marqueur_neutre"
	// NomTourellePorteurVoisinDeSlot : `replay/vehicle_turrets.go`, `carrierOfTurret`.
	NomTourellePorteurVoisinDeSlot Nom = "repli_tourelle_porteur_voisin_de_slot"
	// NomTourelleMonteeLoinDuPorteur : `replay/vehicle_turrets_boarding.go`,
	// `turretRideBoardsCarrier`.
	NomTourelleMonteeLoinDuPorteur Nom = "repli_tourelle_montee_loin_du_porteur"
	// NomEpisodeBorneParLaVieSuivante : `replay/vehicle_rides_next_life.go`, `cutRidesAtNextLife`.
	NomEpisodeBorneParLaVieSuivante Nom = "repli_episode_borne_par_la_vie_suivante"
	// NomZoneCampDeCaptureDeduitDeLIssue : `replay/zone_states_capturer.go`,
	// `zoneRampCapturerDeduit`.
	NomZoneCampDeCaptureDeduitDeLIssue Nom = "repli_zone_camp_de_capture_deduit_de_l_issue"
	// NomPositionHorsEmpriseEcartee : `replay/positions_porte.go` (points de trace) et
	// `replay/positions_porte_vehicules.go` (echantillons et naissances de vehicule).
	NomPositionHorsEmpriseEcartee Nom = "repli_position_hors_emprise_ecartee"
	// NomEchantillonVehiculeAuTraversDUnSilenceEcarte : `replay/positions_porte_vehicules.go`.
	NomEchantillonVehiculeAuTraversDUnSilenceEcarte Nom = "repli_echantillon_vehicule_au_travers_d_un_silence_ecarte"
	// NomAncreDImageCleParElection : `filmdec/keyframe_world.go`, `kfScanNext` ; compte par
	// `replay/film_scan.go`, `balayerPositions` (lot M3.1).
	NomAncreDImageCleParElection Nom = "repli_ancre_d_image_cle_par_election"
	// NomPhysiqueDeTypeDeVehiculeSupposee : `grammar/composants_vue_b_m4b.go` (ti=40 i34) ; compte
	// par `replay/film_scan_mouvement.go`, `balayerEtatsDeMouvement` (lot M4b).
	NomPhysiqueDeTypeDeVehiculeSupposee Nom = "repli_physique_de_type_de_vehicule_supposee"
	// NomIndexDeTireurHorsPlace : `replay/tirs_index_fiable.go` ; compte par `replay/build_pistes.go`
	// (lot M4b.4, inscrit a la revue du lot).
	NomIndexDeTireurHorsPlace Nom = "repli_index_de_tireur_hors_place"
)
