# Mesures de cloture J11.3 — 2026-10-01

> Plan : `.ai/V7.5/PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25.md`, lot J11.3. Code mesure : tete de
> `feat/suite-audit-decodeur` (`97fbe8f55`), worktree `LevelUp-wt-suite-audit`. Lecture seule des films
> (`data/cache/film_chunks`, jonctions du worktree) ; aucune base DuckDB ; sorties sous `$TEMP/j113/`.
> « Avant » = la reference d'avant le plan : GB-1 `MESURE_GB1_2026-09-27.md`, carte de fermeture
> `carte_fermeture_2026-09-26/` (J4.0.5). Instrument : `film/research/cmd_fermeture -mode fermeture,gb1`
> (20 films, 0 echec, un film a la fois, plafond 4 Gio, pics 55 Mio a 1,3 Gio, 3 min 19 au total).
> Corpus : les 19 temoins de `config/replay_corpus.toml` (dont `f75e7053`, present sur ce poste, absent
> de la reference) + `1c4c63c2`.

## 1. GB-1 — vies de generation >= 2 sans position (avant -> tete)

Filtre de production en vigueur a la tete (generations vivantes, J5.2), en quanta, vitesse non jouee
(memes definitions que la mesure d'avant).

| Film | Build | Vies (gen >= 2) | Vies sans position, avant -> tete | dont gen >= 2 | Positions, avant -> tete | Duree publiee (s), avant -> tete | Cible |
|---|---|---|---|---|---|---|---|
| `084a804d` | HI_1_10_0 | 379 (123) | 123 -> **0** | 123 -> 0 | 330 867 -> 487 652 | 972,4 -> **1 055,8** | 0 : tenue ; 1 055,8 s : tenue |
| `1c4c63c2` | HI_1_10_0 | 586 (330) | 330 -> **0** | 330 -> 0 | 557 685 -> 1 165 457 | 825,9 -> **1 288,7** | 0 : tenue ; 1 288,7 s : tenue |
| `4f77afc1` | HI_1_13_0 | 333 (77) | 77 -> **0** | 77 -> 0 | 380 222 -> 460 449 | 1 111,1 -> 1 111,1 | 0 : tenue |
| `a349fea8` | version-33 | 323 (67) | 74 -> **8** | 67 -> 1 | 335 188 -> 403 427 | 933,1 -> **948,0** | <= 8 : tenue ; 948,0 s : tenue |
| 15 autres temoins | tous builds | 57 a 223 (0) | inchange (0 a 9) | 0 -> 0 | inchange | inchangee | aucun changement : tenu |

- Les 15 autres temoins (dont `f75e7053`, absent de la reference : 0 vie sans position, 458,7 s) :
  vies sans position et positions IDENTIQUES a la reference, ligne a ligne (0797ce72 2, 396cfc92 1,
  50247b26 9, 60ae07c4 4, a521164d 4, e5adf7b2 7, fb1a1a72 1, les autres 0) ; durees publiees identiques.
- Orphelins de production (position dont la vie n'est connue d'aucune lecture) : **0** sur les 20 films.
- La duree publiee de `4f77afc1` ne bouge pas (la fin du film est deja couverte par la generation 1) : les
  +80 227 positions sont des positions de generation >= 2 ajoutees, sans effet sur `durationMs`.
- Verification croisee sur artefact reel (cuisson de tete des temoins, section 3) : `durationMs` publie =
  1 055 800 (`084a804d`), 948 000 (`a349fea8`), 1 111 100 (`4f77afc1`), soit exactement la valeur de
  l'instrument. `1c4c63c2` n'est pas un temoin du manifeste : pas d'artefact cuit (carte non declaree).
- Les 8 vies de `a349fea8` restantes (7 de generation 1, 1 de generation >= 2) ne relevent pas de GB-1
  (vues en creation seule, sans en-tete ; meme lecture qu'a la mesure d'avant).

## 2. Carte de fermeture rejouee et etat du declencheur

Reference (18 films, 2026-09-26) -> tete (memes 18 films), puis tete a 20 films. Les paquets fermes et les
records utiles fermes sont quasi identiques : l'ecart total sur les 18 films est de +1 a +11 records utiles
sur quatre films, aucune baisse (le plan J3-J10 n'avait pas pour objet la fermeture ; la carte confirme
qu'elle n'a pas bouge).

| Film | Build | Paquets fermes (ref -> tete) | Records utiles fermes (ref -> tete) | Entrees de controle fermees (ref -> tete) |
|---|---|---|---|---|
| `084a804d` | HI_1_10_0 | 5233/32457 -> 5235/32457 | 73458/457937 -> 73469/457948 | 91905 -> 91940 |
| `111fa685` | HI_1_10_0 | 4152/17362 -> 4152/17362 | 39215/235348 -> 39215/235348 | 55013 -> 55013 |
| `1c4c63c2` | HI_1_10_0 | (absent de la reference) -> 19354/79550 | (absent) -> 145934/704566 | -> 273731 |
| `e5adf7b2` | HI_1_11_0 | 4285/16824 -> 4285/16824 | 65967/246477 -> 65967/246477 | 76224 -> 76224 |
| `bcb6d393` | HI_1_12_0 | 5835/21864 -> 5835/21864 | 31993/117732 -> 31993/117732 | 30137 -> 30137 |
| `0797ce72` | HI_1_13_0 | 19205/26740 -> 19205/26740 | 142893/184273 -> 142894/184274 | 117143 -> 117143 |
| `396cfc92` | HI_1_13_0 | 22828/32052 -> 22828/32052 | 147791/202126 -> 147791/202126 | 148761 -> 148761 |
| `4f77afc1` | HI_1_13_0 | 22909/35499 -> 22910/35499 | 424884/579628 -> 424886/579630 | 408151 -> 408162 |
| `51ebbc0f` | HI_1_13_0 | 9847/30666 -> 9847/30666 | 54502/83598 -> 54502/83598 | 55558 -> 55558 |
| `bf15f7ab` | HI_1_13_0 | 28476/31053 -> 28476/31053 | 197434/210027 -> 197434/210027 | 180861 -> 180861 |
| `bfecd02b` | HI_1_13_0 | 26403/31232 -> 26403/31232 | 192884/218327 -> 192884/218327 | 169159 -> 169159 |
| `c75f33b8` | HI_1_13_0 | 22944/27800 -> 22944/27800 | 130330/140567 -> 130330/140567 | 120528 -> 120528 |
| `d9781168` | HI_1_13_0 | 26464/43645 -> 26464/43645 | 162981/224658 -> 162981/224658 | 150730 -> 150730 |
| `f75e7053` | HI_1_13_0 | (absent de la reference) -> 23423/28502 | (absent) -> 153244/180067 | -> 139891 |
| `fb1a1a72` | HI_1_13_0 | 22318/48771 -> 22318/48771 | 147888/164303 -> 147888/164303 | 128721 -> 128721 |
| `a521164d` | HI_1_4_1 | 701/11130 -> 701/11130 | 1/141879 -> 1/141879 | 5 -> 5 |
| `60ae07c4` | HI_1_8_0 | 13969/49696 -> 13969/49696 | 78247/257296 -> 78247/257296 | 79294 -> 79294 |
| `11de8353` | HI_1_9_0 | 5738/17629 -> 5739/17629 | 57875/225121 -> 57876/225122 | 75327 -> 75327 |
| `50247b26` | version-31 | 153/17919 -> 153/17919 | 10/246018 -> 10/246018 | 3 -> 3 |
| `a349fea8` | version-33 | 463/28751 -> 463/28751 | 2302/378095 -> 2302/378095 | 6 -> 6 |

### Declencheur de la representation intermediaire (spec §9) — par build

Declencheur : **>= 95 % des records (et entrees de controle) qui PORTENT une donnee utile au produit sont
fermes au bit pres**, sur chaque build. Mesure : `utiles_fermes / utiles` par film, agrege par build
(colonnes de l'instrument).

| Build | Films (tete a 20) | Records utiles fermes, reference (18 films) | Tete, memes films | Tete a 20 films | Paquets fermes (tete a 20) | Declencheur |
|---|---|---|---|---|---|---|
| HI_1_13_0 | 10 (ref 9) | 79,8 % | 79,8 % (1 601 590 / 2 007 510) | **80,2 %** (1 754 834 / 2 187 577) | 66,9 % | **non atteint** |
| HI_1_10_0 | 3 (ref 2) | 16,3 % | 16,3 % (112 684 / 693 296) | 18,5 % (258 618 / 1 397 862) | 22,2 % | **non atteint** |
| HI_1_8_0 | 1 | 30,4 % | 30,4 % | 30,4 % (78 247 / 257 296) | 28,1 % | **non atteint** |
| HI_1_12_0 | 1 | 27,2 % | 27,2 % | 27,2 % (31 993 / 117 732) | 26,7 % | **non atteint** |
| HI_1_11_0 | 1 | 26,8 % | 26,8 % | 26,8 % (65 967 / 246 477) | 25,5 % | **non atteint** |
| HI_1_9_0 | 1 | 25,7 % | 25,7 % | 25,7 % (57 876 / 225 122) | 32,6 % | **non atteint** |
| version-33 | 1 | 0,6 % | 0,6 % | 0,6 % (2 302 / 378 095) | 1,6 % | **non atteint** |
| HI_1_4_1 | 1 | 0,0 % | 0,0 % | 0,0 % (1 / 141 879) | 6,3 % | **non atteint** |
| version-31 | 1 | 0,0 % | 0,0 % | 0,0 % (10 / 246 018) | 0,9 % | **non atteint** |

**Etat : le declencheur n'est atteint sur AUCUN des 9 builds du corpus.** Le meilleur, HI_1_13_0, est a
80,2 % (il manque environ 15 points). Par film sur HI_1_13_0 : `bf15f7ab` 94,0 %, `c75f33b8` 92,7 %,
`fb1a1a72` 90,0 %, `bfecd02b` 88,3 %, `f75e7053` 85,1 %, `0797ce72` 77,5 %, `4f77afc1` 73,3 %,
`396cfc92` 73,1 %, `d9781168` 72,6 %, `51ebbc0f` 65,2 % : aucun film n'atteint 95 %. Les builds HI_1_4_1,
version-31 et version-33 ferment pratiquement rien d'utile (0 a 0,6 %).

Limites de la lecture (a ne pas arrondir en faveur du declencheur) :
- `utiles` compte les records utiles **LUS** ; ceux situes apres un arret de marche ne sont pas lus du
  tout. La part reelle de records utiles fermes est donc **inferieure ou egale** aux pourcentages ci-dessus.
- L'instrument donne les entrees de controle **fermees** par film (colonne `entrees_controle_fermees`) mais
  pas leur denominateur « entrees que le produit lit » : la moitie « entrees » du critere ne se calcule pas
  avec lui. Seule la moitie « records » est chiffree ici.
- Cause n°1 inchangee (`fermeture_resume.md`) : « vue C : terminateur hors cadre » (264 757 paquets, borne
  haute du gain 2 745 875 records utiles sur 20 films), puis « liste d'evenements non localisee » (48 720
  paquets, gain nul). Le levier de la fermeture reste la fin de la vue de controle, pas le portage de
  composants.

## 3. Compte par repli sur le corpus

Mesure : artefact de rejeu cuit a la tete (schema 76) pour chacun des 19 temoins (`replay-build`, faits du
match de `replay/testdata/equivalence/<id>.facts.json` ; 8 films decodes depuis le film, 11 rejoues depuis les
faits persistes frais, equivalents d'apres l'ADR 0034), puis lecture de `coverage.fallbacks[]`. Le registre
`facts/fallback` compte **119 replis** ; 53 se declenchent sur au moins un temoin, **66 ont un compte nul**.
Aucun nom publie n'est hors registre.

| Repli | Temoins (sur 19) | Total |
|---|---|---|
| `repli_ancre_d_image_cle_par_election` | 19 | 46394 |
| `repli_ancre_sans_vie_delta_ecartee` | 19 | 28258 |
| `repli_chaine_evenement_code_non_modelise` | 19 | 158604 |
| `repli_chunk_du_pied_par_argmax` | 19 | 19 |
| `repli_enregistrement_statborg_abandonne` | 19 | 42678 |
| `repli_largeur_mot_de_poignee_inferee` | 19 | 19 |
| `repli_liaison_par_anticipation` | 19 | 6587 |
| `repli_libelle_de_source_autres` | 19 | 876 |
| `repli_localisation_largeur_libre` | 19 | 28748 |
| `repli_origine_au_sol_lachee_par_fenetre` | 19 | 4832 |
| `repli_plafond_grenade_par_defaut` | 19 | 19 |
| `repli_type_de_chunk_perdu_du_manifeste` | 19 | 19 |
| `repli_composants_statborg_arretes` | 18 | 240 |
| `repli_identite_premier_occupant_du_siege` | 17 | 1199 |
| `repli_bande_bipede_comblee` | 16 | 106 |
| `repli_deadstate_indice_hors_roster` | 15 | 221 |
| `repli_deadstate_hors_bande_bipede` | 13 | 113 |
| `repli_record_desynchronise_jete` | 13 | 177 |
| `repli_mort_non_revendiquee_la_plus_proche` | 11 | 18 |
| `repli_position_hors_emprise_ecartee` | 11 | 420 |
| `repli_sonde_non_lancee_porte_relachee` | 11 | 11 |
| `repli_appariement_par_fenetre_temporelle` | 9 | 540 |
| `repli_cap_vehicule_vitesse_insuffisante` | 9 | 87945 |
| `repli_emission_hors_domaine_jetee` | 9 | 92 |
| `repli_impulsion_fusionnee_dans_le_geste` | 9 | 74 |
| `repli_chassis_vehicule_marqueur_neutre` | 8 | 70 |
| `repli_largeurs_mpp_calibrees_sur_le_film` | 8 | 32 |
| `repli_registre_inconnu_sans_lecteur_de_troncature` | 8 | 8 |
| `repli_tourelle_porteur_voisin_de_slot` | 8 | 183 |
| `repli_episode_occupation_par_trou_de_position` | 7 | 56 |
| `repli_physique_de_type_de_vehicule_supposee` | 7 | 33193 |
| `repli_place_du_remplacant_par_chainage_d_equipe` | 7 | 22 |
| `repli_couple_recolle_sur_le_voisin` | 6 | 50 |
| `repli_bijection_hongroise_du_feed` | 4 | 54 |
| `repli_episode_borne_par_la_vie_suivante` | 4 | 24 |
| `repli_deadstate_categorie_hors_enum` | 3 | 4 |
| `repli_mort_de_bot_premier_candidat` | 3 | 10 |
| `repli_piece_engendree_sans_evenement` | 3 | 28 |
| `repli_vie_coupee_au_trou_de_replication` | 3 | 240 |
| `repli_controle_corruption_section_absente` | 2 | 4 |
| `repli_index_de_tireur_hors_place` | 2 | 1562 |
| `repli_lien_prise_arme_abandonne` | 2 | 2 |
| `repli_place_ouverte_sous_la_capacite_estimee` | 2 | 2 |
| `repli_assistant_non_resolu_abandonne` | 1 | 5 |
| `repli_borne_de_presence_differee_sur_doute` | 1 | 2 |
| `repli_drapeau_seul_en_jeu` | 1 | 2 |
| `repli_echantillon_vehicule_au_travers_d_un_silence_ecarte` | 1 | 2 |
| `repli_nom_piste_par_le_pont` | 1 | 8 |
| `repli_piste_drapeau_sans_pont_ecartee` | 1 | 4 |
| `repli_position_lacher_prend_la_prise` | 1 | 2 |
| `repli_presence_d_une_entree_par_ses_vies` | 1 | 1 |
| `repli_relais_de_bot_abandonne` | 1 | 3 |
| `repli_roster_nom_invente` | 1 | 1 |

(Source de la table : `coverage.fallbacks` des artefacts ; une entree absente de la liste vaut 0.)

### Liste de retrait soumise a DU-7 : replis a compte NUL sur les 19 temoins

DU-7 (retenu : hors plan) : J11 publie la liste, le retrait suit la regle 4 de D-10 (« un repli dont le
compte est a zero sur le corpus a la cloture d'un jalon est supprime au jalon suivant, avec ses tests »)
sur decision. Les 66 replis nuls se repartissent selon leur `CibleRetrait` du registre.

> **Suite du 2026-10-02 (DU-7 tranchee).** Le zero des 19 temoins a ete remplace par le compte du parc :
> `.ai/MESURES_PARC_REPLIS_NULS_2026-10-02.md` (1 227 artefacts au schema 76, passes killsource et
> usage-summary de la vague J11.4, journal du serveur). **28 des 66 se declenchent sur le parc** et
> sortent de la liste de retrait ; les 10 replis des couches grammaire et profil relevent de la
> campagne de grammaire. Sur decision de l utilisateur, 20 replis ont ete soumis au retrait le
> 2026-10-02 ; **19 sont retires** (code, constante, entree du registre, ligne de versement, tests) :
> `repli_bombe_porteur_sans_vie_nommee`, `repli_camp_inconnu_retire_de_la_table`,
> `repli_crane_porteur_sans_vie_nommee`, `repli_gamertag_par_xuid_brut`,
> `repli_gamertag_premier_xuid_gagne`, `repli_homonymes_sans_xuid`,
> `repli_identite_piste_meilleur_recouvrement`, `repli_identite_pont_par_morts`,
> `repli_instant_sur_la_premiere_manche`, `repli_mort_ecartee_hors_equipe_de_base`,
> `repli_mort_neutre_sans_xuid_abandonnee`, `repli_mort_sans_xuid_ignoree`,
> `repli_participant_sans_xuid_retire`, `repli_roster_indice_hors_bijection`,
> `repli_traction_vie_du_tir`, `repli_traction_vie_la_plus_proche`, `repli_zone_camp_sans_roster`
> (classe A) ; `repli_famille_arme_identifiant_brut`, `repli_zone_proprietaire_sans_roster` (classe C).
> `repli_porteur_anonyme_sans_fin_par_mort` (classe A) reste en place : la seule voie honnete est une
> fin de portage « inconnue » que l election du poseur de bombe (`bomb_arms.go`) devrait traiter, une
> conversion non triviale renvoyee a l utilisateur.

**A. Cible = la regle 4 de D-10 (compte nul au gate de J11) : 45 replis, candidats directs.**

| Repli | Condition | Compteur branche | Cible de retrait (abregee) |
|---|---|---|---|
| `repli_bombe_porteur_sans_vie_nommee` | non_resolu | oui | (regle 4 seule) |
| `repli_cadre_de_marche_par_defaut_conserve` | non_resolu | oui | le cadre de la boucle de records devient une donnee de PROFIL par build, comme les largeurs du bloc MPP |
| `repli_camp_inconnu_retire_de_la_table` | section_absente | oui | l equipe lue dans le film sert aux zones, la table de la base n est plus qu un controle |
| `repli_carte_premier_nom_resolu` | non_resolu | oui | un arbitrage du registre des matchs entre `asset_translations` et `match_registry.map_name`, ou un profil par carte qui  |
| `repli_chunk_de_replication_saute` | section_absente | oui | la table de chunk_00 est le lien direct partout |
| `repli_coequipier_hors_de_vue_par_defaut` | non_resolu | oui | la conversion du contexte d isolement (l etat du coequipier lu dans le film) |
| `repli_colline_votes_periode_entiere` | non_resolu | oui | la conversion du calque des collines |
| `repli_crane_porteur_sans_vie_nommee` | non_resolu | oui | le porteur du crane lu au canal des armes tenues |
| `repli_debut_de_manche_au_minimum` | non_resolu | oui | la chaine des bornes de manche lue au consensus |
| `repli_fraicheur_des_derivations_par_taille` | film_muet | oui | la recuisson selective par couche : une revision par calque remplace la taille |
| `repli_gamertag_par_xuid_brut` | section_absente | oui | la table du film nomme les joueurs a zero mort |
| `repli_gamertag_premier_xuid_gagne` | contradiction | oui | un gamertag a deux xuids devient une contradiction comptee, plus tranchee |
| `repli_garde_equipement_negatif_a_zero` | contradiction | oui | la reconciliation des trois canaux prises / utilises / laches (le compte est MESURE NON NUL : pas de retrait sec) |
| `repli_geste_dernier_occupant_du_match` | non_resolu | oui | le compte est deja nul sur les 8 builds, et le journal des passes le confirme ou l infirme sur le parc |
| `repli_geste_premiere_vie_du_slot` | non_resolu | oui | meme canal et meme mesure que repli_geste_dernier_occupant_du_match |
| `repli_homonymes_sans_xuid` | contradiction | oui | la table du film donne l index, pas le nom : l homonymie cesse d etre un obstacle |
| `repli_i0_porte_et_region_par_defaut` | carte_absente_du_catalogue | oui | aucun chemin de production n appelle `DetectI0LayoutOf` (profil par carte : restent les appelants du contexte et du bala |
| `repli_identite_de_slot_par_residu_de_manche` | non_resolu | oui | (regle 4 seule) |
| `repli_identite_piste_meilleur_recouvrement` | non_resolu | oui | abstention explicite des deux candidats, puis retrait de l entree (compte nul depuis le 2026-09-15) |
| `repli_identite_pont_par_morts` | section_absente | oui | le registre d identite prend la table du film comme lien direct |
| `repli_index_de_region_largeur_un` | non_resolu | oui | la largeur d index de region lue au profil de la carte |
| `repli_index_drapeau_zero_pour_tous` | section_absente | oui | la completion du catalogue de socles |
| `repli_indice_en_collision_jete` | contradiction | oui | l indice des tirs et des touches vient de la table du film (cette voie sert `match_weapon_shots` et `match_weapon_accura |
| `repli_instant_sur_la_premiere_manche` | non_resolu | oui | la chaine des bornes de manche lue au consensus |
| `repli_invariant_propre_drapeau_muet` | non_resolu | oui | l equipe du porteur est lue dans le film : le silence doit disparaitre |
| `repli_largeurs_axe_par_defaut_conservees` | section_absente | oui | les largeurs par carte et par build, donnees de profil |
| `repli_largeurs_monde_par_defaut_conservees` | non_resolu | oui | les largeurs, donnee de la carte et du build |
| `repli_largeurs_mpp_par_defaut` | inconditionnel | oui | les largeurs MPP lues au profil du build |
| `repli_manche_zero_decretee` | film_muet | oui | la revision a zero difference de la chaine des manches |
| `repli_mort_ecartee_hors_equipe_de_base` | section_absente | oui | l equipe lue dans le film portee jusqu a ce calque |
| `repli_mort_neutre_sans_xuid_abandonnee` | non_resolu | oui | la table du film portee jusqu au constructeur |
| `repli_mort_sans_xuid_ignoree` | non_resolu | oui | la table du film portee jusqu a ce calque |
| `repli_participant_sans_xuid_retire` | section_absente | oui | le tableau des participants derive du roster HORS LIGNE, deja complet, au lieu de la feuille de match |
| `repli_portage_ferme_a_la_prise_suivante` | film_muet | oui | le porteur du crane lu au canal des armes tenues |
| `repli_porteur_anonyme_sans_fin_par_mort` | non_resolu | oui | le porteur du crane lu au canal des armes tenues, et le registre d identite par la table du film |
| `repli_premiere_occurrence_sans_concordance` | non_resolu | oui | l indice des tirs et des touches vient de la table du film |
| `repli_rang_capacite_vie_elargie` | non_resolu | oui | mesurer le residu du compteur desormais cable |
| `repli_roster_indice_hors_bijection` | non_resolu | oui | la table du film portee jusqu a l ecriture en base |
| `repli_slot_abandonne_au_premier_arrive` | non_resolu | oui | la table du film portee a ce calque |
| `repli_table_identite_vide` | section_absente | oui | la table du film donne le lien direct a ce calque |
| `repli_tourelle_montee_loin_du_porteur` | lecture_non_portee | oui | la lecture de la montee a bord d un artilleur (i10 sur la tourelle) ou du parent d une piece montee |
| `repli_traction_vie_du_tir` | non_resolu | oui | mesurer le compte desormais cable |
| `repli_traction_vie_la_plus_proche` | non_resolu | oui | mesurer le compte desormais cable |
| `repli_xuid_vide_pour_nom_inconnu` | non_resolu | oui | la table du film nomme les joueurs a zero mort, jusqu a l ecriture |
| `repli_zone_camp_sans_roster` | section_absente | oui | l equipe lue dans le film portee au calque des zones (la table de la base n est plus qu un controle) |

**B. Cible = le gate de J11 nommement : 2 replis** (retrait si zero sur le corpus, sinon lecture des vies manquantes).

| Repli | Condition | Compteur branche | Cible de retrait (abregee) |
|---|---|---|---|
| `repli_generation_vivante_inconnue_tag1` | non_resolu | oui | J11 (gates de corpus du plan PLAN_SUITE_AUDIT_DECODEUR_FILM) : retrait si le compte est a zero sur le corpus, sinon lect |
| `repli_identite_vie_par_occupation_du_corps` | non_resolu | oui | J11 du plan PLAN_SUITE_AUDIT_DECODEUR_FILM (gates de corpus) : lecture de l identite des vies restantes (fin de vie sans |

**C. Cible differente (conversion, donnee de profil/catalogue, degradation gracieuse) : 19 replis.** Compte
nul mais ils ne relevent pas de la regle 4 : la cible propre au registre prime (retrait sec a la conversion,
ou « retrait sans objet » pour les degradations gracieuses). Listes pour memoire, pas candidats directs.

| Repli | Condition | Compteur branche | Cible de retrait (abregee) |
|---|---|---|---|
| `repli_amorce_grenade_profil_de_reference` | non_resolu | oui | retrait sec des que la table du profil couvre toutes les clefs du parc : le prochain patch du jeu ajoute sa clef d amorc |
| `repli_armement_bombe_debut_a_zero` | non_resolu | oui | lot de conversion de l'origine du rejeu (coverage.originResolved) |
| `repli_catalogue_de_zones_absent` | section_absente | oui | aucune (degradation gracieuse multi-titre, `ErrCapabilityNotSupported`) ; le COMPTE est ce qui manque |
| `repli_chunks_apres_trou_abandonnes` | non_resolu | oui | lot de conversion du manifeste (le manifeste dit quels chunks existent ; un trou est une donnee, pas une borne) |
| `repli_coequipiers_partis_constante_nulle` | inconditionnel | oui | retrait de la colonne (append-only, ADR 0026 : la colonne reste, c'est son ECRITURE qui doit devenir nulle explicite) |
| `repli_colline_dernier_intervalle_ouvert` | film_muet | oui | aucune tant que le canal reste un ETAT sans emission de fin ; le COMPTE est ce qui manque |
| `repli_decor_carte_sans_zone_affiche` | carte_absente_du_catalogue | oui | toute carte jouee a un fond publie (campagne de cuisson des fonds, cmd/mapfond-build) |
| `repli_decor_sous_le_sol_foule_du_match` | carte_absente_du_catalogue | oui | le fond de carte publie porte l altitude de sa matiere (cuisson cmd/mapfond-build), lue par la regle a la place du match |
| `repli_distances_de_touche_desactivees` | section_absente | oui | le lot qui rallume la precision par arme : tant que la passe ne tourne pas, ses trois compteurs restent a zero par const |
| `repli_famille_arme_identifiant_brut` | film_muet | oui | aucune tant que le catalogue d'armes est incomplet ; retrait sec des que le compte est nul sur le parc |
| `repli_famille_objectif_vide` | non_resolu | NON | question NE13 de la table (D) : le classement par `strings.Contains` sur le nom de variante est hors doctrine multi-titr |
| `repli_nombre_drapeaux_hors_catalogue_sans_passage` | carte_absente_du_catalogue | oui | une lecture du nombre de drapeaux dans le film (vies libres de l'objet drapeau, non mesuree a ce jour), ou l'ajout de la |
| `repli_precision_par_arme_passe_sautee` | section_absente | oui | aucune (configuration d'exploitation) ; le COMPTE est ce qui manque |
| `repli_presence_par_enveloppe_des_vies` | lecture_non_portee | oui | chaque chemin qui publie un roster porte les entites ti=9 (le collecteur de sync y compris) ; ce qui reste est un film s |
| `repli_repere_neutre_generique_conserve` | section_absente | oui | completion de la table d'assets de morts neutres |
| `repli_temps_forts_dernier_numero` | section_absente | oui | le refus des films sans manifeste par la cuisson (replaybuild.jugerFilmSansManifeste), le jour ou replay-build et les in |
| `repli_vie_de_bot_par_relais_de_la_base` | film_muet | oui | le lot qui datera la presence d'un bot de moins d'une image-cle par le film (BOT_METADATA + creation du corps) sans pass |
| `repli_zone_camp_de_capture_deduit_de_l_issue` | non_resolu | oui | un critere d'election qui n'exige pas deux rampes abouties — par exemple l'election du POUSSEUR une fois pour le film  |
| `repli_zone_proprietaire_sans_roster` | section_absente | oui | meme cible que repli_zone_camp_sans_roster |

### Reserves a lever avant de retirer (le zero n'est pas toujours une mesure)

- `repli_famille_objectif_vide` : **compteur non branche** (`CompteurBranche: false`), donc zero par
  absence d'instrument ; seul repli nul dans ce cas. Ne pas retirer sur ce zero.
- `repli_garde_equipement_negatif_a_zero` (classe A) : sa cible dit « compte MESURE NON NUL, pas de retrait
  sec » (30 ecrasements sur 8 builds le 2026-09-16, passe `usage-summary`). Un zero dans
  `coverage.fallbacks` de l'artefact de rejeu ne couvre pas cette passe : a confirmer par la ligne
  `replis de la passe` du backfill avant tout retrait.
- `repli_distances_de_touche_desactivees` et `repli_precision_par_arme_passe_sautee` (classe C) : zero par
  construction (passe de precision par arme non rallumee / configuration d'exploitation), « ne prouvent
  rien » selon le registre.
- Un repli compte uniquement hors cuisson de rejeu (sync, base, passes de backfill) n'est pas vu par cette
  mesure : le zero ne vaut que pour ce que la cuisson verse a `coverage.fallbacks` (table de versement, J8.7).
- Corpus de 19 films ; un zero sur 19 temoins n'est pas un zero sur le parc : la regle 4 vise le corpus
  gate, la decision de retrait reste celle de l'utilisateur (DU-7).

## 4. Assistants `?N` et bots epingles

Mesure : `cmd/killsource json <film> -carte <carte>` (decodeur de la source de degat, hors ligne, aucun
acces base) sur les 19 temoins, un film a la fois, a la tete. Les assistants sont ceux de la table des
morts (`assistant.joueur`) ; l'epinglage des bots se lit dans l'avertissement `bot(s) NON EPINGLE(S)`
(`Coverage.BotsNonEpingles`, FK-1) et dans les noms `[bot]` publies.

**Assistants `?N` : 0 sur les 19 temoins** (aucune valeur commencant par `?` dans aucun champ des 19 JSON :
tueurs, victimes et assistants confondus ; 2 747 morts decodees au total, dont 0 a 110 assistants nommes
selon le film).

| Film | Morts | Assistants nommes | `?N` | Bots publies (noms `[bot]`) | Bots non epingles (borne humains) |
|---|---|---|---|---|---|
| `4f77afc1` | 296 | 89 | 0 | `343 Hollis [bot]` (9 comme tueur, 3 comme victime), `343 Doomfruit [bot]` (1 victime) ; 4 morts de bot, 5 morts infligees par un bot | **1** (borne 24) |
| `bcb6d393` | 49 | 10 | 0 | `343 Hollis [bot]` (1 victime) ; 1 mort de bot | **1** (borne 8) |
| `c75f33b8` | 61 | 19 | 0 | `343 Robot Hoida [bot]` (1 victime) ; 1 mort de bot | 0 |
| 16 autres temoins | 67 a 351 | 0 a 110 | 0 | aucun | 0 |

- Les 16 autres (morts / assistants nommes) : `0797ce72` 92/26, `084a804d` 351/110, `111fa685` 191/56,
  `11de8353` 164/40, `396cfc92` 67/26, `50247b26` 140/0, `51ebbc0f` 70/35, `60ae07c4` 158/60, `a349fea8`
  289/0, `a521164d` 101/0, `bf15f7ab` 75/26, `bfecd02b` 82/41, `d9781168` 142/63, `e5adf7b2` 197/44,
  `f75e7053` 82/39, `fb1a1a72` 140/45. `50247b26`, `a349fea8` et `a521164d` n'ont aucun assistant nomme
  (anciens builds ; `a349fea8` journalise « table des joueurs du film NON LUE »).
- Bots non epingles : 2 films sur 19 (`4f77afc1`, `bcb6d393`), 1 bot chacun ; leurs morts et celles qu'ils
  infligent ne sont pas publiees, et c'est maintenant DIT (avertissement + `Coverage.BotsNonEpingles`).
  Les 2 cas sont des bots dont le slot tombe sur un siege que la table du film nomme (un humain le tient).
- Limite : l'outil n'affiche pas le roster complet publie (liste des bots epingles non vacants) ; le releve
  ne couvre que les bots qui apparaissent dans la table des morts ou dans le compteur de bots non epingles.

## Ce qui n'a pas ete fait

- `1c4c63c2` n'est pas cuit en artefact (hors manifeste, carte inconnue) : il n'est ni dans la table des
  replis ni dans le releve des assistants/bots ; il figure en GB-1 et en fermeture (instrument seul).
- Le denominateur des entrees de controle « utiles » n'est pas fourni par l'instrument : critere du
  declencheur calcule sur les records seuls (section 2).
- Aucun retrait de repli, aucune modification de code, aucun commit hors ce rapport.
