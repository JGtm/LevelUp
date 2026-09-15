package replay

// vehicle_families.go — LA TABLE D IDENTITE DES CHASSIS : `MPPWord32` -> famille de sprite.
//
// D OU VIENT LA CLE. Le record de CREATION d un vehicule (`ti=40`) porte, dans son default-state,
// le bloc `object-multiplayer-properties` dont le mot de 32 bits INCONDITIONNEL est l identite du
// chassis (`filmdec/default_state_ti40.go`, feuille 2). Ce mot se lit AVANT toute position et
// toute porte optionnelle. Trois gates l ont valide (V1.5, deux films) : constance 100 % par vie,
// <= 8 valeurs distinctes par film, valeurs decouplees du nombre de vies. Et surtout : CINQ des
// sept valeurs d un film Behemoth reapparaissent sur un film Launch Site d une AUTRE build — une
// valeur qui survit au changement de build et de carte n est pas un artefact de film, c est un
// GlobalID de tag (`RE_DEFAULTSTATE_TI40_2026-08-31.md` § 8.4).
//
// POURQUOI UNE TABLE STATIQUE, ET PAS UNE LECTURE DES `.module`. Decision de cadrage du plan
// d integration (2026-09-02), et elle est ferme : le serveur ne lit AUCUN fichier de jeu. Les
// modules pesent des giga-octets, ne sont pas versionnes dans le depot, et changent a chaque
// mise a jour du jeu. La resolution vit donc ici, valeur par valeur, avec sa source.
//
// DEUX SOURCES INDEPENDANTES, ET ELLES SE RECOUPENT. La premiere est la classification par NOMS
// DE MAILLAGE internes du tag `vehi` (`V4_RAPPORT_SPRITES_2026-08-31.md` § 4 : `warthog_p_rf_ma`,
// `ghost_b_f`, `scorpion_frt_bf`...), qui identifie le chassis DANS son propre tag. La seconde
// est la table de nommage des tags de degat du depot
// (`internal/games/halo_infinite/film/damagetag/data/labels.tsv`), qui nomme la BANQUE DE SONS du
// porteur (`sb_010_veh_cv_ghost`, `sb_010_veh_cv_banshee`, `sb_010_veh_bt_chopper`...). Les deux
// s accordent sur les cinq chassis ou elles se croisent — c est ce recoupement qui autorise a
// completer l une par l autre.
//
// CE QUE LA TABLE NE FAIT PAS : distinguer les VARIANTES d un meme chassis. Rockethog, Razorback
// et Warthog Gauss partagent le `render_model` du Warthog (`0x561f2ca7`) ; le Gungoose partage
// celui du Mongoose (`0x9e581380`) — leur difference vit dans une PERMUTATION du modele ou dans
// les refs d armement du `vehi`, pas dans le chassis (V4 § 6, `REWORK_WARTHOG_GUNGOOSE_2026-09-01`
// § 5). Le lot A sert bien un sprite par variante, mais aucun `MPPWord32` observe n a ete resolu
// vers l une d elles : les entrees `rockethog` / `razorback` / `warthog_gauss` / `gungoose` de
// l index de sprites restent donc SANS cle ici, plutot que devinees.
//
// UN MEME VEHICULE PORTE PLUSIEURS `vehi`, ET CE N EST PAS UNE VARIANTE : C EST LE MODULE.
//
// Le jeu range ses tags dans des MODULES (`pc/globals`, `any/globals`, `any/globals/common`...),
// et un meme vehicule y est declare PLUSIEURS FOIS, sous un GlobalID different par module. Le
// manifeste de la chaine de destruction l ecrit noir sur blanc pour le Scorpion
// (`.ai/V7.5/film_re/sons_v3_reconstruits/manifeste_v3.json`, entree « Scorpion (M808) ») :
// « f6f54e56 (any/globals) = chassis 0000d3db (pc/globals), meme hlmt ».
//
// CONSEQUENCE MESUREE, ET ELLE A COUTE DEUX FAMILLES (lot 1.9.9, 2026-09-16). Les releves qui
// ont peuple cette table (V4_RAPPORT_SPRITES, ASSEMBLAGE_*, REWORK_WARTHOG_GUNGOOSE) balayaient
// `pc/globals` ; les FILMS, eux, ecrivent l identifiant du module que la partie charge. Sur les
// 76 artefacts du parc, AUCUN identifiant de base d une famille Covenant ou UNSC n apparait
// jamais (`00002705`, `000025aa`, `0000d3db`, `0000d3dc`, `000026ed`, `00002706`, `002ba902`) —
// seul le SECOND identifiant est observe (`fe32c0f4` 63 vies, `af31ab1a` 60, `5b80c406` 45,
// `c6e79dcc` 16, `3d4a8a5a` 13). Les deux familles qui n avaient QUE leur identifiant de base —
// Wraith et Scorpion — ne se resolvaient donc JAMAIS : leurs vies sortaient sans sprite ET sans
// occupant (cf. `vehicleFamilyIsRideable`, qui refuse tout episode sur une famille vide).
//
// LA REGLE QUI EN DECOULE : une famille n est completement nommee que lorsque TOUS les `vehi`
// que le manifeste lui rattache sont en table. Chaque entree ci-dessous cite sa piece.
//
// CE QUI N ENTRE PAS ICI : les tags `vehi` ENFANTS (tourelles et canons montes — `0000d4ff`,
// `0000d500`, `64b925eb`, `bcfb852f`, `dd7f9102`), que le manifeste range sous « Falcon
// (tourelle LMG) et autres objets-enfants » avec le verdict « PAS DE SON DE DESTRUCTION
// PROPRE ». Un enfant n est pas un chassis : lui donner une famille ferait dessiner un vehicule
// la ou il n y a qu une piece d armement. Aucun n a d ailleurs ete observe au parc.
//
// VALEUR INCONNUE = FAMILLE VIDE, ET C EST UN REPLI NOMME (D14 du plan decodeur). Le vehicule
// reste publie (sa trajectoire est vraie), sans sprite : le client dessine un marqueur neutre.
// Emprunter la famille d un voisin donnerait un Warthog dessine en Banshee, ce qui est pire
// qu un marqueur. Depuis le lot 1.9.9 (2026-09-16) ce refus porte son nom au registre des
// replis — `repli_chassis_vehicule_marqueur_neutre`, condition `chassis_absent_de_la_table`
// (le film N EST PAS muet : il ecrit le mot d identite, c est NOTRE table qui ne le nomme pas),
// ordre `apres_lecture` — et son compteur est CABLE par cuisson (`tallyVehicleCoverage`), en
// plus du compteur publie `Coverage.Vehicles.UnknownChassis` et du journal du calque.
//
// DECISION UTILISATEUR DU 2026-09-14 : le parc d assets vehicules est COMPLET. Un chassis
// absent de cette table n est donc PAS un vehicule dont l image manquerait — c est un MISMATCH
// a nommer, et le nommer est le seul geste qui retire le repli.

import (
	"fmt"
	"strings"
)

// LES NOMS DE FAMILLE, une constante chacun — et pas seulement pour faire taire `goconst`.
//
// Une famille n est PAS un libelle (rien n en est traduit : ce sont des noms propres du jeu) :
// c est la CLE qui joint trois choses independantes — le sprite servi
// (`static/vehicles-assets/{slug}/replay/{famille}.png`), la ligne de l index du lot A, et,
// depuis le 2026-09-05, les stems sonores du rejeu (`vehicle_{famille}_{clip}.wav`). La table
// ci-dessous ecrit certaines d entre elles jusqu a douze fois (un chassis par variante observee) :
// une faute de frappe dans UNE de ces douze produirait une famille fantome — sprite 404, moteur
// muet — sans erreur ni compteur. La constante rend cette faute impossible a la compilation.
const (
	familleWarthog  = "warthog"
	familleMongoose = "mongoose"
	familleScorpion = "scorpion"
	familleWasp     = "wasp"
	familleGhost    = "ghost"
	familleBanshee  = "banshee"
	familleWraith   = "wraith"
	familleChopper  = "chopper"
	famillePhantom  = "phantom"
	famillePelican  = "pelican"
	familleSkiff    = "skiff"
	familleShade    = "shade"
	familleFalcon   = "falcon"
	// familleTourelleAutoBannie n est PAS un vehicule : c est un ELEMENT DE CARTE (lot 1.9.9,
	// decision utilisateur du 2026-09-14). Voir la table ci-dessous pour la preuve, et
	// `config/titles/{slug}/mappings/replay_labels.toml` pour son libelle et sa nature publies.
	familleTourelleAutoBannie = "tourelle_auto_bannie"
)

// vehicleFamilyByChassis associe le `MPPWord32` d un record de creation `ti=40` a la FAMILLE de
// chassis, c est-a-dire au nom de fichier du sprite servi par
// `static/vehicles-assets/{slug}/replay/{famille}.png` (index du lot A).
//
// Chaque entree porte sa source. « OBSERVE » signale les valeurs effectivement rencontrees dans
// les films du corpus (`RE_DEFAULTSTATE_TI40_2026-08-31.md` § 8.4, films `0d76e8f1` Behemoth
// Super Fiesta et `fccc61cd` Launch Site Super Fiesta) : ce sont celles dont la resolution est
// verifiee de bout en bout, des bits du film au fichier PNG.
var vehicleFamilyByChassis = map[uint32]string{
	// --- Chassis nommes par leurs maillages internes (V4_RAPPORT_SPRITES_2026-08-31 § 4) ---
	0x00002705: familleWarthog,  // maillages `warthog_p_rf_ma` ; mode 0x561f2ca7
	0x000025aa: familleMongoose, // maillages `mongoose_p` ; recoupe REWORK_WARTHOG_GUNGOOSE § 1
	0x0000d3db: familleScorpion, // maillages `scorpion_frt_bf` ; mode 0x39918211
	0xb65b3b4a: familleWasp,     // recoupe labels.tsv : `vehi b65b3b4a, sb_010_veh_un_wasp`
	0x0000d3dc: familleGhost,    // recoupe labels.tsv : `vehi 0000d3dc, sb_010_veh_cv_ghost`
	0x000026ed: familleBanshee,  // recoupe labels.tsv : `vehi 000026ed, sb_010_veh_cv_banshee`
	0x00002706: familleWraith,   // mode 0x3a98ee2d
	0x002ba902: familleChopper,  // recoupe labels.tsv : `vehi 002ba902, sb_010_veh_bt_chopper`
	0x000026f2: famillePhantom,  // transport, non pilotable — sprite servi, jamais conduit
	0x000026f0: famillePelican,  // transport, non pilotable
	0x86799cb6: familleSkiff,    // mode 0xa3aaa279
	0x000df0c4: familleShade,    // recoupe labels.tsv : `vehi 000df0c4, sb_010_tur_cv_shadeturret`
	// Falcon : V4 § 4 le classe par ses maillages (mode 0xa0ca8a6f, 76 sections). CONFLIT NOTE,
	// non tranche ici : `labels.tsv` porte `vehi 0000254b +1, sb_010_veh_un_pelican` — mais le
	// « +1 » dit que le tag de degat a DEUX porteurs, et la banque citee peut etre celle de
	// l autre. La classification par maillage porte sur le tag `0000254b` LUI-MEME : elle est
	// plus directe, elle l emporte. OBSERVE en film.
	0x0000254b: familleFalcon,

	// --- Variantes de chassis rencontrees EN FILM, nommees par une seconde source ---
	// OBSERVE (0d76e8f1 + fccc61cd). `labels.tsv` : `vehi 5b80c406, sb_010_veh_cv_ghost`.
	0x5b80c406: familleGhost,
	// OBSERVE (0d76e8f1 + fccc61cd). `labels.tsv` : `vehi c6e79dcc, sb_010_veh_cv_banshee`.
	0xc6e79dcc: familleBanshee,
	// OBSERVE (0d76e8f1 + fccc61cd). REWORK_WARTHOG_GUNGOOSE_2026-09-01 § 1 et
	// CONTACT_ARMES_GUNGOOSE_2026-09-02 § : « 3 vehi Mongoose (0x000025aa, 0xaf31ab1a,
	// 0xde26e3d7), tous -> mode 0x9e581380 » — le mode du chassis mongoose.
	0xaf31ab1a: familleMongoose,
	0xde26e3d7: familleMongoose, // meme source, meme mode ; non observe en film a ce jour
	// OBSERVE le 2026-09-02 sur `0d76e8f1` (3 vies), par le chemin de PRODUCTION lui-meme
	// (`replay-build`, journal « chassis de vehicule NON RESOLU »). `labels.tsv` le nomme sur ses
	// DEUX entrees a porteur UNIQUE — `vehi 3d4a8a5a, sb_010_veh_bt_chopper` (jpt 661b2987 et
	// 72230737). Ses deux autres entrees ont plusieurs porteurs (« +2 ») et ne tranchent rien :
	// ce sont celles-la qui sont ecartees, pas la resolution.
	0x3d4a8a5a: familleChopper,
	// OBSERVE (0d76e8f1 : 9 vies ; fccc61cd : 4 vies) — le chassis le plus frequent du corpus
	// (143/441 naissances, V2_SPAWNS_COOLDOWNS_2026-09-01 § 1.2), longtemps irresolu parce que
	// `labels.tsv` porte son `vehi` SANS banque de sons. Resolu par la CHAINE DE DESTRUCTION
	// (V3D_DESTRUCTION_SONS_2026-09-02, table des verdicts) : `vehi fe32c0f4 -> hlmt daf7f543`,
	// le hlmt du Warthog (celui de 0x00002705), qui pose la banque d explosion
	// `exp_vehicle_med_unsc`. Un vehi qui partage le hlmt du Warthog est un chassis Warthog —
	// variante indistinguable, cf. la note d en-tete : la famille suffit.
	0xfe32c0f4: familleWarthog,
	// Meme preuve, meme hlmt daf7f543 (V3D, meme table) ; non observe en film a ce jour.
	0xcb96ca07: familleWarthog,

	// --- ELEMENT DE CARTE, pas un vehicule (lot 1.9.9, 2026-09-16) ---
	//
	// LA TOURELLE AUTOMATIQUE BANNIE. `labels.tsv` la nomme sur TROIS tags de degat a porteur
	// `vehi 038df01a` (3b3b3d40, aed08680, e1ea6e65) et deux d entre eux ne citent qu UNE banque,
	// `sb_003_lvl_moments_ge_shared_autoturret_banished` (+ son `_fire`) : le prefixe `sb_003_lvl`
	// dit deja la NATURE — banque de NIVEAU (`lvl`), pas banque de vehicule (`sb_010_veh`), et
	// « moments_ge_shared » est le vocabulaire des mises en scene de carte.
	//
	// LA MESURE LE CONFIRME (instrument `TestInventaireChassisDesArtefacts`, 76 artefacts du
	// parc) : 18 vies sur deux films (`bfecd02b` 9, `2cf24f30` 9), 17 des 18 SANS AUCUN
	// echantillon de trajectoire, et les 18 avec une fenetre qui couvre LE MATCH ENTIER. Un
	// vehicule pilotable n a ni l une ni l autre de ces signatures.
	//
	// DECISION UTILISATEUR DU 2026-09-14 : « ce sont des elements de la map » — un objet qui
	// interdit la sortie de la zone de jeu, PAS un vehicule jouable. Il entre donc en table sous
	// une famille NOMMEE, il est publie, il est dessine par un pictogramme dedie cote client, et
	// il ne porte JAMAIS d occupant (cf. `vehicleFamillesNonPilotables`).
	0x038df01a: familleTourelleAutoBannie,

	// --- LES SECONDS `vehi` DU MANIFESTE, un module plus loin (lot 1.9.9, 2026-09-16) ---
	//
	// TOUS viennent de la MEME piece, celle que la table cite deja pour `fe32c0f4` et
	// `cb96ca07` : `.ai/V7.5/film_re/sons_v3_reconstruits/manifeste_v3.json`, section
	// `par_vehicule`, recoupee par `V3D_DESTRUCTION_SONS_2026-09-02.md` § 2. Chaque ligne du
	// manifeste donne le `hlmt` de la famille et la banque d explosion qu il atteint ; les
	// identifiants entre parentheses de sa colonne `vehi` sont les declarations du MEME
	// vehicule dans un autre module. Aucun n est devine.
	//
	// RESERVE ECRITE : la BANQUE seule ne separe pas Wraith, Banshee et Phantom (ils partagent
	// `2eaae6d7 large_covenant` et l evenement `1bf6fdde`, dit par le manifeste lui-meme). Ce
	// qui les separe est le `hlmt`, et c est sur lui que chaque entree ci-dessous est rangee —
	// exactement le critere que la table applique deja a `fe32c0f4` (« un vehi qui partage le
	// hlmt du Warthog est un chassis Warthog »).

	// Manifeste, « Wraith » : `vehi` = « 00002706 (+ ae845375) », hlmt `5b5c960d`, foot
	// `48669cd9`, banque `2eaae6d7 (large_covenant)`, event `1bf6fdde`, confiance HAUTE.
	// OBSERVE : 18 vies MOBILES sur quatre films du parc (`4f77afc1` 11, `5676a9ba` 4,
	// `0a44c6cc` 2, `8a485699` 1), 376 a 2 352 echantillons de trajectoire, naissances en
	// relais sur deux socles par carte. C est le chassis Wraith des films ; `00002706` n a
	// JAMAIS ete observe.
	0xae845375: familleWraith,

	// Manifeste, « Scorpion (M808) » : `vehi` = « f6f54e56 (any/globals) = chassis 0000d3db
	// (pc/globals), meme hlmt » — l egalite est ECRITE. hlmt `e7fe7564`, deja documente comme
	// le hlmt du chassis Scorpion (`ASSEMBLAGE_ENFANTS_2026-09-01.md` § « Chassis seul »),
	// banque `94d43e95 (large_unsc)`. OBSERVE : 3 vies mobiles sur trois films.
	0xf6f54e56: familleScorpion,

	// Manifeste, « Ghost » : `vehi` = « 0000d3dc (+ 5b80c406, 9af9e693) », hlmt `3b3038e6`,
	// banque `b1f8608b (small_covenant)`. `5b80c406` est deja en table (45 vies observees) ;
	// `9af9e693` est son troisieme module. NON OBSERVE au parc a ce jour — il entre parce que
	// la piece le nomme, pas parce qu on l a vu.
	0x9af9e693: familleGhost,

	// Manifeste, « Banshee » : `vehi` = « 000026ed (+ 0001530a, c6e79dcc) », hlmt `df38bc96`,
	// banque `2eaae6d7 (large_covenant)`. `c6e79dcc` est deja en table (16 vies observees).
	// NON OBSERVE au parc.
	0x0001530a: familleBanshee,

	// Manifeste, « Warthog (toute la famille : warthog, razorback, rockethog) » : `vehi` =
	// « 00002705, cb96ca07, fe32c0f4 (+ 5159c8ef, 75312e51, 7617ff6e dans any/globals/common) »,
	// hlmt `daf7f543`, banque `c468fb55 (med_unsc)`. Les trois derniers sont les declarations
	// du module `any/globals/common`. NON OBSERVES au parc — meme raison d entrer : la piece.
	// Ils partagent le hlmt du Warthog, donc la famille (cf. la note de `fe32c0f4`).
	0x5159c8ef: familleWarthog,
	0x75312e51: familleWarthog,
	0x7617ff6e: familleWarthog,
}

// vehicleFamilyOf rend la famille de chassis d un `MPPWord32`, ou la chaine VIDE quand la table
// ne le connait pas.
//
// UNE VALEUR INCONNUE N EST PAS UNE ERREUR : c est un chassis que le chantier n a pas encore
// resolu (le cas `0xfe32c0f4`, longtemps dans cette situation, a ete tranche le 2026-09-02 par
// la chaine de destruction — voir la table). Elle se compte et se journalise, elle ne se
// devine pas.
func vehicleFamilyOf(chassis uint32) string {
	return vehicleFamilyByChassis[chassis]
}

// formatChassisID rend le mot d identite en hexadecimal 8 chiffres MINUSCULES, la meme convention
// que les familles d arme du document (`Loadout.W`, `GroundWeapon.W`) : un entier brut ne se
// publie pas, et deux conventions d ecriture d un identifiant dans un meme document rendraient
// toute jointure cote client impossible.
func formatChassisID(chassis uint32) string {
	return strings.ToLower(fmt.Sprintf("%08x", chassis))
}
