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
	// NomPlafondGrenadeParDefaut : applique par `grammar/inventory_decode.go` (`ScanKeyframeInventory`),
	// compte par `replay/film_scan.go` (`balayerInventaire`) depuis le lot J4.2.
	NomPlafondGrenadeParDefaut Nom = "repli_plafond_grenade_par_defaut"
	// NomLargeursAxeParDefautConservees : `replay/world_object_precision.go`.
	NomLargeursAxeParDefautConservees Nom = "repli_largeurs_axe_par_defaut_conservees"
	// NomIdentitePisteMeilleurRecouvrement : `replay/identity.go`, `nameTracksByLives`.
	NomIdentitePisteMeilleurRecouvrement Nom = "repli_identite_piste_meilleur_recouvrement"
	// NomPositionLacherPrendLaPrise : `replay/flag_carries.go`, `attachFlagCarryPositions`.
	NomPositionLacherPrendLaPrise Nom = "repli_position_lacher_prend_la_prise"
	// NomNombreDrapeauxHorsCatalogueSansPassage : `replay/flag_carries_handoff.go`,
	// `countFlagCountUnread` (lot J9.2, constat RB1-5).
	NomNombreDrapeauxHorsCatalogueSansPassage Nom = "repli_nombre_drapeaux_hors_catalogue_sans_passage"
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
	// NomGenerationVivanteInconnueTag1 : `grammar/generations_vivantes.go` (`Accepte`) ; compte par
	// `replay/film_scan.go`, `balayerPositions` (lot J5.2).
	NomGenerationVivanteInconnueTag1 Nom = "repli_generation_vivante_inconnue_tag1"
	// NomIdentiteVieParOccupationDuCorps : `replay/unnamed_lives.go`, `nameRemainingLives` (lot J5.4).
	NomIdentiteVieParOccupationDuCorps Nom = "repli_identite_vie_par_occupation_du_corps"
	// NomLiaisonParAnticipation : `grammar/world.go`, `World.LierParRepliDAnticipation` ; compte par
	// `replay/film_scan_mouvement.go`, `balayerEtatsDeMouvement` (lot J8.1, constat GA1-2).
	NomLiaisonParAnticipation Nom = "repli_liaison_par_anticipation"
	// NomOrigineAuSolLacheeParFenetre : `replay/ground_weapon_rules.go`, `gwPadsClass` ; compte par
	// `replay/ground_weapon_pads.go`, `buildWeaponPads` (lot J8.3, constat RB2-8).
	NomOrigineAuSolLacheeParFenetre Nom = "repli_origine_au_sol_lachee_par_fenetre"
	// NomIdentiteDeSlotParResiduDeManche : `objectives/slotidentity_residue.go`,
	// `CompletedByRoundResidue` ; compte par `replay/identity_registry_section.go`,
	// `compterLesSlotsParResidu` (lot J8.4, constats FO-1 / RA2-4).
	NomIdentiteDeSlotParResiduDeManche Nom = "repli_identite_de_slot_par_residu_de_manche"
	// NomCapVehiculeVitesseInsuffisante : compte par `build_vehicles.go` (lot J8.7).
	NomCapVehiculeVitesseInsuffisante Nom = "repli_cap_vehicule_vitesse_insuffisante"
	// NomEpisodeOccupationParTrouDePosition : compte par `build_vehicles.go` (lot J8.7).
	NomEpisodeOccupationParTrouDePosition Nom = "repli_episode_occupation_par_trou_de_position"
	// NomMortEcarteeHorsEquipeDeBase : compte par `death_context.go` (lot J8.7).
	NomMortEcarteeHorsEquipeDeBase Nom = "repli_mort_ecartee_hors_equipe_de_base"
	// NomCoequipierHorsDeVueParDefaut : compte par `death_context.go` (lot J8.7).
	NomCoequipierHorsDeVueParDefaut Nom = "repli_coequipier_hors_de_vue_par_defaut"
	// NomCranePorteurSansVieNommee : compte par `skull_carries.go` (lot J8.7, nom passe par le calque
	// du crane depuis le lot J8.7-bis).
	NomCranePorteurSansVieNommee Nom = "repli_crane_porteur_sans_vie_nommee"
	// NomBombePorteurSansVieNommee : compte par `bomb_carries.go` (lot J8.7-bis, 2026-09-28) — la
	// MEME porte de presence que le crane (`skull_carries.go`), sous le nom de la bombe.
	NomBombePorteurSansVieNommee Nom = "repli_bombe_porteur_sans_vie_nommee"
	// NomIndexDrapeauZeroPourTous : compte par `flag_assign.go` (lot J8.7).
	NomIndexDrapeauZeroPourTous Nom = "repli_index_drapeau_zero_pour_tous"
	// NomInvariantPropreDrapeauMuet : compte par `flag_assign.go` (lot J8.7).
	NomInvariantPropreDrapeauMuet Nom = "repli_invariant_propre_drapeau_muet"
	// NomDrapeauSeulEnJeu : compte par `flag_assign.go` (lot J8.7).
	NomDrapeauSeulEnJeu Nom = "repli_drapeau_seul_en_jeu"
	// NomFamilleArmeIdentifiantBrut : compte par `ground_weapon_pads.go` (lot J8.7).
	NomFamilleArmeIdentifiantBrut Nom = "repli_famille_arme_identifiant_brut"
	// NomIdentitePremierOccupantDuSiege : compte par `identity_registry.go` (lot J8.7).
	NomIdentitePremierOccupantDuSiege Nom = "repli_identite_premier_occupant_du_siege"
	// NomImpulsionFusionneeDansLeGeste : compte par `document_ability_impulses.go` (lot J8.7).
	NomImpulsionFusionneeDansLeGeste Nom = "repli_impulsion_fusionnee_dans_le_geste"
	// NomRangCapaciteVieElargie : compte par `document_ability_impulses.go` (lot J8.7).
	NomRangCapaciteVieElargie Nom = "repli_rang_capacite_vie_elargie"
	// NomLienPriseArmeAbandonne : compte par `document_ground_weapon_items.go` (lot J8.7).
	NomLienPriseArmeAbandonne Nom = "repli_lien_prise_arme_abandonne"
	// NomNomPisteParLePont : compte par `build_inventaire.go` (lot J8.7).
	NomNomPisteParLePont Nom = "repli_nom_piste_par_le_pont"
	// NomPortageFermeALaPriseSuivante : compte par `bomb_carries.go` (lot J8.7).
	NomPortageFermeALaPriseSuivante Nom = "repli_portage_ferme_a_la_prise_suivante"
	// NomPorteurAnonymeSansFinParMort : compte par `bomb_carries.go` (lot J8.7).
	NomPorteurAnonymeSansFinParMort Nom = "repli_porteur_anonyme_sans_fin_par_mort"
	// NomTractionVieDuTir : compte par `build_calques.go` (lot J8.7).
	NomTractionVieDuTir Nom = "repli_traction_vie_du_tir"
	// NomTractionVieLaPlusProche : compte par `build_calques.go` (lot J8.7).
	NomTractionVieLaPlusProche Nom = "repli_traction_vie_la_plus_proche"
	// NomZoneCampSansRoster : compte par `zone_states_owner.go` (lot J8.7).
	NomZoneCampSansRoster Nom = "repli_zone_camp_sans_roster"
	// NomZoneProprietaireSansRoster : compte par `zone_states_owner.go` (lot J8.7).
	NomZoneProprietaireSansRoster Nom = "repli_zone_proprietaire_sans_roster"

	// LES REPLIS DE `grammar` ET `profile` (sous-lot grammar du lot J8.7, 2026-09-27) : comptes en
	// DONNEES au rapport du contexte de film (`grammar.ComptesDesReplis`), et verses au compteur de
	// la cuisson par la table de `replay/versement_des_replis.go`, qui cite ces noms.

	// NomChunkDeReplicationSaute : `grammar/player_index.go`.
	NomChunkDeReplicationSaute Nom = "repli_chunk_de_replication_saute"
	// NomTempsFortsDernierNumero : `grammar/deaths_source.go`.
	NomTempsFortsDernierNumero Nom = "repli_temps_forts_dernier_numero"
	// NomLargeursMppCalibreesSurLeFilm : `grammar/equipment_placements.go`, `replay/build_ground_weapons.go`.
	NomLargeursMppCalibreesSurLeFilm Nom = "repli_largeurs_mpp_calibrees_sur_le_film"
	// NomI0PorteEtRegionParDefaut : `grammar/film_context.go`, `grammar/offline_biped_band.go`.
	NomI0PorteEtRegionParDefaut Nom = "repli_i0_porte_et_region_par_defaut"
	// NomBandeBipedeComblee : `grammar/offline_biped_band.go`.
	NomBandeBipedeComblee Nom = "repli_bande_bipede_comblee"
	// NomLargeursMondeParDefautConservees : `grammar/profil_balayage.go`.
	NomLargeursMondeParDefautConservees Nom = "repli_largeurs_monde_par_defaut_conservees"
	// NomIndexDeRegionLargeurUn : `grammar/profil_balayage.go`.
	NomIndexDeRegionLargeurUn Nom = "repli_index_de_region_largeur_un"
	// NomLargeursMppParDefaut : `profile/mpp_widths.go`, compte par `replay/build_ground_weapons.go`.
	NomLargeursMppParDefaut Nom = "repli_largeurs_mpp_par_defaut"
	// NomChunksApresTrouAbandonnes : `grammar/film_chunks.go`.
	NomChunksApresTrouAbandonnes Nom = "repli_chunks_apres_trou_abandonnes"
	// NomAncreSansVieDeltaEcartee : `grammar/equipment_creation_width.go`.
	NomAncreSansVieDeltaEcartee Nom = "repli_ancre_sans_vie_delta_ecartee"
	// NomRegistreInconnuSansLecteurDeTroncature : `grammar/registry_fingerprint.go`, compte par
	// `grammar/replis_du_film.go`.
	NomRegistreInconnuSansLecteurDeTroncature Nom = "repli_registre_inconnu_sans_lecteur_de_troncature"
	// NomAmorceGrenadeProfilDeReference : `profile/grenade.go`, `grammar/grenade_events.go`.
	NomAmorceGrenadeProfilDeReference Nom = "repli_amorce_grenade_profil_de_reference"
	// NomControleCorruptionSectionAbsente : `grammar/controle_corruption_du_film.go` (et la calibration
	// de `killsource`).
	NomControleCorruptionSectionAbsente Nom = "repli_controle_corruption_section_absente"
	// NomLocalisationLargeurLibre : `grammar/object_deaths_march.go` (et la marche de `killsource`).
	NomLocalisationLargeurLibre Nom = "repli_localisation_largeur_libre"

	// LES REPLIS DE `killsource` (sous-lot killsource du lot J8.7, 2026-09-27) : comptes en DONNEES
	// dans `killsource.Stats` (`Replis`, et les comptes que le decodeur tenait deja), verses par la
	// table de `replay/versement_des_replis.go` a l assemblage.

	// NomRecordDesynchroniseJete : `killsource/walk.go`.
	NomRecordDesynchroniseJete Nom = "repli_record_desynchronise_jete"
	// NomDeadstateHorsBandeBipede : `killsource/walk.go`.
	NomDeadstateHorsBandeBipede Nom = "repli_deadstate_hors_bande_bipede"
	// NomDeadstateIndiceHorsRoster : `killsource/walk.go`.
	NomDeadstateIndiceHorsRoster Nom = "repli_deadstate_indice_hors_roster"
	// NomDeadstateCategorieHorsEnum : `killsource/walk.go`.
	NomDeadstateCategorieHorsEnum Nom = "repli_deadstate_categorie_hors_enum"
	// NomRosterNomInvente : `killsource/roster.go`.
	NomRosterNomInvente Nom = "repli_roster_nom_invente"
	// NomRosterIndiceHorsBijection : `killsource/roster.go`.
	NomRosterIndiceHorsBijection Nom = "repli_roster_indice_hors_bijection"
	// NomBijectionHongroiseDuFeed : `killsource/bijection.go` (`RosterTable.Inferred`).
	NomBijectionHongroiseDuFeed Nom = "repli_bijection_hongroise_du_feed"
	// NomCoupleRecolleSurLeVoisin : `killsource/feed_couples.go` (`CoupleStats.Recolles`).
	NomCoupleRecolleSurLeVoisin Nom = "repli_couple_recolle_sur_le_voisin"
	// NomGamertagParXuidBrut : `killsource/feed.go`.
	NomGamertagParXuidBrut Nom = "repli_gamertag_par_xuid_brut"
	// NomChunkDuPiedParArgmax : `killsource/feed.go`.
	NomChunkDuPiedParArgmax Nom = "repli_chunk_du_pied_par_argmax"
	// NomChaineEvenementCodeNonModelise : `killsource/eventbody.go`, `killsource/eventchain.go`.
	NomChaineEvenementCodeNonModelise Nom = "repli_chaine_evenement_code_non_modelise"
	// NomTypeDeChunkPerduDuManifeste : `killsource/chunks.go`.
	NomTypeDeChunkPerduDuManifeste Nom = "repli_type_de_chunk_perdu_du_manifeste"
	// NomMortNonRevendiqueeLaPlusProche : `killsource/hybrid.go` (`ApparStats.NonRevendiqueeFenetre`).
	NomMortNonRevendiqueeLaPlusProche Nom = "repli_mort_non_revendiquee_la_plus_proche"
	// NomMortDeBotPremierCandidat : `killsource/match.go` (`ApparStats.BotFenetre`).
	NomMortDeBotPremierCandidat Nom = "repli_mort_de_bot_premier_candidat"
	// NomAppariementParFenetreTemporelle : `killsource/paquet_identite.go`, `match.go`, `assist.go`
	// (`ApparStats.Fenetre` + `AssistStats.ParLaFenetre`).
	NomAppariementParFenetreTemporelle Nom = "repli_appariement_par_fenetre_temporelle"
	// NomSondeNonLanceePorteRelachee : `killsource/decode.go`.
	NomSondeNonLanceePorteRelachee Nom = "repli_sonde_non_lancee_porte_relachee"
	// NomLibelleDeSourceAutres : `killsource/label.go`.
	NomLibelleDeSourceAutres Nom = "repli_libelle_de_source_autres"
	// NomLargeurMotDePoigneeInferee : `killsource/calibrate.go`.
	NomLargeurMotDePoigneeInferee Nom = "repli_largeur_mot_de_poignee_inferee"

	// LES REPLIS D `objectives` (sous-lot objectives du lot J8.7, 2026-09-27) : comptes en DONNEES
	// par le balayage du statborg et par le resolveur d identite par manche
	// (`objectives.ComptesDesReplis`), verses par la table de `replay` a l assemblage.

	// NomEnregistrementStatborgAbandonne : `objectives/statborg.go`.
	NomEnregistrementStatborgAbandonne Nom = "repli_enregistrement_statborg_abandonne"
	// NomComposantsStatborgArretes : `objectives/statborg.go`.
	NomComposantsStatborgArretes Nom = "repli_composants_statborg_arretes"
	// NomTableIdentiteVide : `objectives/slotidentity_deaths.go`.
	NomTableIdentiteVide Nom = "repli_table_identite_vide"
	// NomMortSansXuidIgnoree : `objectives/slotidentity_deaths.go`.
	NomMortSansXuidIgnoree Nom = "repli_mort_sans_xuid_ignoree"
	// NomDebutDeMancheAuMinimum : `objectives/slotidentity_rounds.go`.
	NomDebutDeMancheAuMinimum Nom = "repli_debut_de_manche_au_minimum"
	// NomSlotAbandonneAuPremierArrive : `objectives/slotidentity_rounds.go`.
	NomSlotAbandonneAuPremierArrive Nom = "repli_slot_abandonne_au_premier_arrive"
	// NomEmissionHorsDomaineJetee : `objectives/named_series.go`, releve a la CONSULTATION par
	// evenement distinct (`objectives.ReplisALaConsultation`, lot J8.7-bis, 2026-09-28).
	NomEmissionHorsDomaineJetee Nom = "repli_emission_hors_domaine_jetee"
	// NomInstantSurLaPremiereManche : `objectives/slotidentity_rounds.go`, releve a la CONSULTATION
	// par instant distinct (lot J8.7-bis, 2026-09-28).
	NomInstantSurLaPremiereManche Nom = "repli_instant_sur_la_premiere_manche"

	// LES REPLIS DE LA CONSTRUCTION (`internal/replaybuild`, sous-lot replaybuild du lot J8.7,
	// 2026-09-27) : `Declenche` au site, sur le compteur de la construction de la cuisson, dont le
	// rapport voyage dans `replay.Options.ReplisHorsBalayage` ; cites par `decfilm`.

	// NomAssistantNonResoluAbandonne : `replaybuild/kills.go`.
	NomAssistantNonResoluAbandonne Nom = "repli_assistant_non_resolu_abandonne"
	// NomGamertagPremierXuidGagne : `replaybuild/kills.go`.
	NomGamertagPremierXuidGagne Nom = "repli_gamertag_premier_xuid_gagne"
	// NomMortNeutreSansXuidAbandonnee : `replaybuild/replaybuild.go`.
	NomMortNeutreSansXuidAbandonnee Nom = "repli_mort_neutre_sans_xuid_abandonnee"
	// NomRepereNeutreGeneriqueConserve : `replaybuild/replaybuild.go`.
	NomRepereNeutreGeneriqueConserve Nom = "repli_repere_neutre_generique_conserve"
	// NomRelaisDeBotAbandonne : `replaybuild/replaybuild.go`.
	NomRelaisDeBotAbandonne Nom = "repli_relais_de_bot_abandonne"
	// NomParticipantSansXuidRetire : `replaybuild/matchfacts_feuille.go`.
	NomParticipantSansXuidRetire Nom = "repli_participant_sans_xuid_retire"
	// NomCampInconnuRetireDeLaTable : `replaybuild/matchfacts_feuille.go`, compte par `options.go`.
	NomCampInconnuRetireDeLaTable Nom = "repli_camp_inconnu_retire_de_la_table"
	// NomCatalogueDeZonesAbsent : `replaybuild/zones.go`.
	NomCatalogueDeZonesAbsent Nom = "repli_catalogue_de_zones_absent"
	// NomFraicheurDesDerivationsParTaille : `replaybuild/derivations_index.go`, compte par la passe
	// des derivations de `sync/replayartifacts` (hors cuisson : expvar par nom et journal du cycle).
	NomFraicheurDesDerivationsParTaille Nom = "repli_fraicheur_des_derivations_par_taille"

	// LES REPLIS DU COLLECTEUR (`internal/sync/killcollector`, sous-lot collecteur du lot J8.7,
	// 2026-09-27) : `Declenche` au site, sur le compteur de la passe du film, publie en expvar par
	// nom et au journal du film (`killcollector/replis_de_la_passe.go`) ; cites par `decfilm`.

	// NomXuidVidePourNomInconnu : `killcollector/identities.go`.
	NomXuidVidePourNomInconnu Nom = "repli_xuid_vide_pour_nom_inconnu"
	// NomHomonymesSansXuid : `killcollector/roster.go`.
	NomHomonymesSansXuid Nom = "repli_homonymes_sans_xuid"
	// NomIndiceEnCollisionJete : `killcollector/shots.go`.
	NomIndiceEnCollisionJete Nom = "repli_indice_en_collision_jete"
	// NomPremiereOccurrenceSansConcordance : `killcollector/shots.go`.
	NomPremiereOccurrenceSansConcordance Nom = "repli_premiere_occurrence_sans_concordance"
	// NomPrecisionParArmePasseSautee : `killcollector/hits.go`.
	NomPrecisionParArmePasseSautee Nom = "repli_precision_par_arme_passe_sautee"
	// NomIdentitePontParMorts : `killcollector/positions.go`.
	NomIdentitePontParMorts Nom = "repli_identite_pont_par_morts"
	// NomCoequipiersPartisConstanteNulle : `killcollector/isolation_facts.go`.
	NomCoequipiersPartisConstanteNulle Nom = "repli_coequipiers_partis_constante_nulle"
	// NomDistancesDeToucheDesactivees : `killcollector/hits.go`.
	NomDistancesDeToucheDesactivees Nom = "repli_distances_de_touche_desactivees"
	// NomCartePremierNomResolu : `killcollector/map_identity.go` (et `hits.go`).
	NomCartePremierNomResolu Nom = "repli_carte_premier_nom_resolu"
)
