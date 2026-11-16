# A3.3 — Les 46 entrées du registre à zéro sur le parc régénéré (2026-11-14)

Mesure : 1 455 documents de rejeu au schéma 116, régénérés le 14/11 par la cuisson de LevelUp (décodeur
v1.3.7, chemin de l'hôte complet). Comptes par repli et par build : `C:\t\marche\A3\parc116\comptes.tsv`,
liste des 46 : `zero.tsv`. Classement sur pièces par un agent en lecture seule ; les deux retraits
re-vérifiés à la main (code des poseurs, `MapQuantEntry.Layout`, `DetectI0LayoutOf`, `I0Layout.Valid`).

Un zéro ne vaut retrait que si (1) le compteur de l'entrée finit dans `coverage.fallbacks` d'un document
et (2) l'entrée est un contournement qu'une lecture a remplacé. Sinon elle reste.

## Retirées (2)

| Entrée | Raison |
|---|---|
| `repli_largeurs_monde_par_defaut_conservees` | inatteignable : la pose de la carte exige trois largeurs non nulles, `DetectI0LayoutOf` ne rend qu'un découpage valide (8 à 26 bits par axe) |
| `repli_index_de_region_largeur_un` | inatteignable : `Layout` rend une porte de 3 + 1 + au moins 1 bits, `DetectI0LayoutOf` une porte de 5 |

Commit du décodeur b84ef68 (branche `feat/fin-phase-a`). Révisions constantes, empreintes `grammar` et
`killsource` refigées.

## Gardées : zéro non mesuré par les documents (17)

Compteur publié ailleurs : le collecteur de sync (expvar et journal de la passe), le résumé d'usage, une
règle de service ou le carnet des dérivations, ou une entrée hors production.

- Collecteur (9) : `carte_premier_nom_resolu`, `coequipier_hors_de_vue_par_defaut`,
  `coequipiers_partis_constante_nulle`, `distances_de_touche_desactivees`, `indice_en_collision_jete`,
  `position_posthume_du_tueur`, `precision_par_arme_passe_sautee`, `premiere_occurrence_sans_concordance`,
  `xuid_vide_pour_nom_inconnu`.
- Service du décor (3) : `decor_carte_sans_zone_affiche`, `decor_sous_le_sol_foule_du_match`,
  `decor_tenu_en_l_air_a_vide`.
- Résumé d'usage (3) : `garde_equipement_negatif_a_zero`, `geste_dernier_occupant_du_match`,
  `geste_premiere_vie_du_slot` (le registre les dit non nulles).
- Carnet des dérivations (1) : `fraicheur_des_derivations_par_taille`.
- Hors production (1) : `famille_objectif_vide`.

## Gardées : protections contre une entrée dégradée ou inconnue (26)

Mesurables, à zéro parce que le parc local ne contient pas le cas : build futur sans clé d'amorce, chunk
illisible, trou de numérotation, catégorie hors énumération, record désynchronisé, carte sans largeur,
titre sans table de zones, nom de jauge ou de désignateur hors vocabulaire, match à colline sans point,
pont d'identité muet, film sans manifeste, film sans `ti=9`, etc. Les retirer rendrait ces cas
silencieux ou faux.

`amorce_grenade_profil_de_reference`, `armement_bombe_debut_a_zero`, `catalogue_de_zones_absent`,
`chunk_de_replication_saute`, `chunks_apres_trou_abandonnes`, `colline_designateur_par_voisinage`,
`colline_proprietaire_voisin_du_designateur`, `colline_votes_periode_entiere`, `colline_votes_sans_garde`,
`deadstate_categorie_hors_enum`, `fin_de_vie_au_dernier_point`, `fin_de_vie_mort_manquante`,
`i0_porte_et_region_par_defaut` (atteint seulement par les instruments en production : candidate à un
reclassement hors production), `largeurs_axe_par_defaut_conservees`, `largeurs_mpp_par_defaut`,
`porteur_anonyme_sans_fin_par_mort`, `presence_par_enveloppe_des_vies`, `record_desynchronise_jete`,
`seuil_garde_table_contredite`, `seuil_garde_table_de_variante`, `socle_du_film_au_centre_des_vols`,
`temps_forts_dernier_numero`, `tourelle_montee_loin_du_porteur`, `vie_de_bot_par_relais_de_la_base`,
`zone_pousseur_par_election`, `zone_proprietaire_par_vote`.

## Gardée, redondante au pire (1)

`position_de_bipede_sans_creation_ecartee` : écrite pour un film du parc qui ne la déclenche plus. Soit
une autre porte (l'emprise de la carte, qui part 2 fois sur ce film) écarte déjà ces positions, soit la
garde ne tourne plus. Dans les deux cas la garder ne change aucune sortie ; son retrait demanderait de
prouver la première lecture sur la piste des deux slots concernés. Non retirée.
