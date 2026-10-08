# Mesure du parc — replis nuls de J11 (DU-7) — 2026-10-02

Objet : remplacer le zéro sur 19 témoins (`.ai/V7.5/film_re/MESURES_CLOTURE_J11_2026-10-01.md` §3)
par le compte du parc, SANS cuisson, avant de décider le retrait (DU-7, règle 4 de D-10 ;
`.ai/ANALYSE_MISE_EN_OEUVRE_REPRESENTATION_INTERMEDIAIRE_2026-10-01.md` §5.2).

## 1. Sources (lecture seule)

| Source | Population | Ce qu'elle voit |
|---|---|---|
| `coverage.fallbacks` des artefacts `data/cache/replays/halo_infinite/<id8>.json` | 1 227 artefacts, tous au schéma 76, tous écrits par la vague J11.4 (2026-10-01 17 h 47 → 2026-10-02 3 h 37) | les replis que la cuisson verse (balayage, assemblage, table de versement J8.7 : killsource, objectifs, grammaire) |
| Journal de `levelup backfill-killsource` de la vague (`06_killsource.log`) | 1 222 films décodés (62 écartés : carte non résolue) | ligne `killsource: replis de la passe du film` (collecteur) + lignes `rejeu : repli declenche` des étages de `replay` rejoués par la passe |
| Journal de `levelup backfill-usage-summary` de la vague (`04_usage_summary.log`) | 1 227 matchs | ligne par match `replis declenches par la projection` + ligne de corpus |
| Journal du serveur `logs/general.log` depuis le redémarrage (2026-10-02 5 h 18 → 15 h 45) | cycles de sync | lignes `replis` des passes killsource, usage-summary et du jugement de fraîcheur des dérivés |

Les journaux de la vague sont dans le dossier de travail de la session qui l'a menée
(`%TEMP%\claude\C--Users-Guillaume-Projects-LevelUp\4432ca41-...\scratchpad\vague_j114\`, UTF-16,
lignes repliées à la largeur de la console : le dépouillement recolle les lignes repliées). Le
total de la ligne de corpus d'usage-summary (2 984 / 163 / 5) est retrouvé à l'unité par la somme
des lignes par match.

Contrôle de cohérence : tout nom journalisé par la cuisson (`03_replay.log`) figure dans
`coverage.fallbacks` (aucun écart) ; 71 noms se déclenchent au moins une fois dans les artefacts
(53 sur les 19 témoins).

## 2. Résultat : 28 des 66 « nuls » se déclenchent sur le parc

Colonnes : artefacts = matchs/déclenchements dans `coverage.fallbacks` ; killsource = matchs/
déclenchements dans la passe de la vague ; usage = matchs/déclenchements dans usage-summary.

### 2.1 Actifs sur le parc (sortent de la liste de retrait)

| Classe | Repli | Registre | Artefacts | Killsource | Usage / serveur |
|---|---|---|---|---|---|
| A | `repli_carte_premier_nom_resolu` | killsource_carte | 0 | 15/28 | serveur : 4 matchs |
| A | `repli_coequipier_hors_de_vue_par_defaut` | replay_identites | 0 | 1205/189473 | 0 |
| A | `repli_colline_votes_periode_entiere` | replay_objectifs | 17/90 | 0 | 0 |
| A | `repli_debut_de_manche_au_minimum` | objectifs | 3/3 | 0 | 0 |
| A | `repli_fraicheur_des_derivations_par_taille` | objectifs | 0 | 0 | serveur : 129 cycles / 5 745 |
| A | `repli_garde_equipement_negatif_a_zero` | replay_equipement | 0 | 0 | 847/2984 |
| A | `repli_geste_dernier_occupant_du_match` | replay_equipement | 0 | 0 | 104/163 |
| A | `repli_geste_premiere_vie_du_slot` | replay_equipement | 0 | 0 | 3/5 |
| A | `repli_identite_de_slot_par_residu_de_manche` | objectifs | 14/35 | 0 | 0 |
| A | `repli_index_drapeau_zero_pour_tous` | replay_objectifs | 10/1024 | 0 | 0 |
| A | `repli_indice_en_collision_jete` | killsource_collecteur | 0 | 5/123 | 0 |
| A | `repli_invariant_propre_drapeau_muet` | replay_objectifs | 1/4 | 0 | 0 |
| A | `repli_manche_zero_decretee` | objectifs | 10/10 | 0 | 0 |
| A | `repli_portage_ferme_a_la_prise_suivante` | replay_equipement | 5/5 | 0 | 0 |
| A | `repli_premiere_occurrence_sans_concordance` | killsource_collecteur | 0 | 1212/110135 | 0 |
| A | `repli_rang_capacite_vie_elargie` | replay_equipement | 2/4 | 0 | 0 |
| A | `repli_slot_abandonne_au_premier_arrive` | objectifs | 12/12 | 0 | 0 |
| A | `repli_table_identite_vide` | objectifs | 2/2 | 0 | 0 |
| A | `repli_tourelle_montee_loin_du_porteur` | replay_vehicules | 2/2 | 0 | 0 |
| A | `repli_xuid_vide_pour_nom_inconnu` | killsource_collecteur | 0 | 310/2684 | 0 |
| B | `repli_identite_vie_par_occupation_du_corps` | replay_identites | 3/4 | 0 | 0 |
| C | `repli_coequipiers_partis_constante_nulle` | killsource_collecteur | 0 | 1208/120107 | 0 |
| C | `repli_colline_dernier_intervalle_ouvert` | replay_objectifs | 62/62 | 0 | 0 |
| C | `repli_nombre_drapeaux_hors_catalogue_sans_passage` | replay_objectifs | 10/1031 | 0 | 0 |
| C | `repli_presence_par_enveloppe_des_vies` | replay_places | 0 | 37/247 (étage `replay` de la passe) | 0 |
| C | `repli_repere_neutre_generique_conserve` | objectifs | 14/16 | 0 | 0 |
| C | `repli_vie_de_bot_par_relais_de_la_base` | replay_places | 2/10 | 0 | 0 |
| C | `repli_zone_camp_de_capture_deduit_de_l_issue` | replay_objectifs | 14/168 | 0 | 0 |

Lecture : les sept replis de classe A comptés hors cuisson (killsource, usage-summary, cycle du
serveur) étaient à zéro dans les 19 artefacts témoins parce que la mesure de J11 ne lisait que
`coverage.fallbacks`, pas parce qu'ils ne se déclenchaient pas. Les douze autres actifs de
classe A sont des cas rares absents des 19 témoins. La réserve écrite sur
`repli_garde_equipement_negatif_a_zero` est confirmée (2 984 sur le parc). Celle sur
`repli_fraicheur_des_derivations_par_taille` est nouvelle : il compte le chemin NORMAL du
jugement « à jour » (chaque artefact à jour à chaque cycle).

### 2.2 Couches grammaire / profil — hors de ce chantier (campagne de grammaire)

`repli_cadre_de_marche_par_defaut_conserve`, `repli_largeurs_mpp_par_defaut`,
`repli_largeurs_axe_par_defaut_conservees`, `repli_largeurs_monde_par_defaut_conservees`,
`repli_i0_porte_et_region_par_defaut`, `repli_index_de_region_largeur_un`,
`repli_chunks_apres_trou_abandonnes`, `repli_amorce_grenade_profil_de_reference`,
`repli_generation_vivante_inconnue_tag1` (actif : 3 artefacts / 45),
`repli_chunk_de_replication_saute`. Dix noms : la consigne en citait 9 en comptant « largeurs
MPP/axe/monde » ensemble ; la table de l'analyse §5.2 (9) n'inclut pas les largeurs d'axe,
rangées au registre `replay_equipement`. Les dix sont exclus.

### 2.3 Nuls sur le parc (28)

| Classe | Repli | Registre | Où il se compte | Exposition du parc |
|---|---|---|---|---|
| A | `repli_bombe_porteur_sans_vie_nommee` | replay_objectifs | cuisson | 9 artefacts avec porteurs de bombe |
| A | `repli_camp_inconnu_retire_de_la_table` | objectifs | cuisson (`replaybuild/options.go`) | 1 227 |
| A | `repli_crane_porteur_sans_vie_nommee` | replay_objectifs | cuisson | 23 artefacts avec porteurs de crâne |
| A | `repli_gamertag_par_xuid_brut` | killsource | cuisson (versement) | 1 227 |
| A | `repli_gamertag_premier_xuid_gagne` | objectifs | cuisson (`replaybuild/kills.go`) | 1 227 |
| A | `repli_homonymes_sans_xuid` | killsource_collecteur | passe killsource | 1 222 |
| A | `repli_identite_piste_meilleur_recouvrement` | replay_identites | cuisson | 1 227 |
| A | `repli_identite_pont_par_morts` | killsource_collecteur | passe killsource (pont des positions, exécuté) | 1 222 |
| A | `repli_instant_sur_la_premiere_manche` | objectifs | cuisson (versement) | 1 227 |
| A | `repli_mort_ecartee_hors_equipe_de_base` | replay_identites | cuisson | 1 227 |
| A | `repli_mort_neutre_sans_xuid_abandonnee` | objectifs | cuisson (`replaybuild.go`) | 1 227 |
| A | `repli_mort_sans_xuid_ignoree` | objectifs | cuisson (versement) | 1 227 |
| A | `repli_participant_sans_xuid_retire` | objectifs | cuisson (`matchfacts_feuille.go`) | 1 227 |
| A | `repli_porteur_anonyme_sans_fin_par_mort` | replay_equipement | cuisson | 9 artefacts avec porteurs de bombe |
| A | `repli_roster_indice_hors_bijection` | killsource | cuisson (versement) | 1 227 |
| A | `repli_traction_vie_du_tir` | replay_equipement | cuisson | 1 227 |
| A | `repli_traction_vie_la_plus_proche` | replay_equipement | cuisson | 1 227 |
| A | `repli_zone_camp_sans_roster` | replay_objectifs | cuisson | 135 artefacts avec états de zone |
| C | `repli_armement_bombe_debut_a_zero` | replay_objectifs | cuisson | 9 |
| C | `repli_catalogue_de_zones_absent` | objectifs | cuisson | 1 227 |
| C | `repli_decor_carte_sans_zone_affiche` | replay_vehicules | LECTURE (service, `vehicleScenery.zoneUnknown`) | non mesuré |
| C | `repli_decor_sous_le_sol_foule_du_match` | replay_vehicules | LECTURE (service) | non mesuré |
| C | `repli_distances_de_touche_desactivees` | killsource_carte | passe killsource | nul par construction |
| C | `repli_famille_arme_identifiant_brut` | replay_equipement | cuisson (`ground_weapon_pads.go`) | 1 227 |
| C | `repli_famille_objectif_vide` | objectifs | compteur NON branché | non mesurable |
| C | `repli_precision_par_arme_passe_sautee` | killsource_collecteur | passe killsource | nul par construction |
| C | `repli_temps_forts_dernier_numero` | objectifs | cuisson (versement) | 1 227 ; 0 WARN « film SANS manifeste » sur la vague |
| C | `repli_zone_proprietaire_sans_roster` | replay_objectifs | cuisson | 135 |

## 3. Proposition de retrait (soumise à l'utilisateur)

- **Classe A, 18 candidats** : les 18 nuls du §2.3 (cuisson ou passe killsource, compteur
  branché, zéro sur le parc). Population mince pour trois d'entre eux (bombe 9, crâne 23).
- **Classe B** : aucun. `repli_identite_vie_par_occupation_du_corps` est actif (3 matchs) : sa
  cible dit « sinon lecture des vies manquantes » ; `repli_generation_vivante_inconnue_tag1`
  appartient à la grammaire.
- **Classe C, 2 candidats par leur propre cible** : `repli_famille_arme_identifiant_brut`
  (« retrait sec dès que le compte est nul sur le parc », tenu) et
  `repli_zone_proprietaire_sans_roster` (« même cible que `repli_zone_camp_sans_roster` », suit
  sa décision). Les huit autres restent : conversion à venir (armement de bombe, temps forts —
  critère de mesure tenu mais la cible est le refus des films sans manifeste, un changement de
  comportement), dégradation gracieuse (catalogue de zones), comptés à la lecture et non mesurés
  (décor ×2), nuls par construction (distances, précision), sans compteur (famille d'objectif).

## 4. Constat hors périmètre

Le serveur redécode à chaque cycle les 8 films sans kill-feed (440 décodages « sans kill-feed,
rien à publier » sur 8 match_id entre 5 h 18 et 15 h 45). Signalé en tâche séparée, non traité.
Effet sur cette mesure : aucun, les comptes du serveur sont pris par match distinct.

## 5. Issue (2026-10-02)

Proposition retenue par l'utilisateur (18 A + 2 C). Retirés : 19. Maintenu :
`repli_porteur_anonyme_sans_fin_par_mort` — sa seule voie honnête (fin de portage inconnue)
change l'élection de l'armement de bombe (`replay/bomb_arms.go` lit `!FinParMort`) : c'est une
conversion, pas un retrait. Branche `feat/retrait-replis-nuls` ; `replay-equiv` re-figé (forme des
objets observés `flag`, `killsource`, `skull` ; `artifact` identique sur les 20 films, cf.
CORPUS.txt).
