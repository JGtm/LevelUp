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
// VALEUR INCONNUE = FAMILLE VIDE. Le vehicule reste publie (sa trajectoire est vraie), sans
// sprite : le client dessine un marqueur neutre. Emprunter la famille d un voisin donnerait un
// Warthog dessine en Banshee, ce qui est pire qu un marqueur. Le compteur
// `Coverage.Vehicles.UnknownChassis` et le journal du calque rompent le silence.

import (
	"fmt"
	"strings"
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
	0x00002705: "warthog",  // maillages `warthog_p_rf_ma` ; mode 0x561f2ca7
	0x000025aa: "mongoose", // maillages `mongoose_p` ; recoupe REWORK_WARTHOG_GUNGOOSE § 1
	0x0000d3db: "scorpion", // maillages `scorpion_frt_bf` ; mode 0x39918211
	0xb65b3b4a: "wasp",     // recoupe labels.tsv : `vehi b65b3b4a, sb_010_veh_un_wasp`
	0x0000d3dc: "ghost",    // recoupe labels.tsv : `vehi 0000d3dc, sb_010_veh_cv_ghost`
	0x000026ed: "banshee",  // recoupe labels.tsv : `vehi 000026ed, sb_010_veh_cv_banshee`
	0x00002706: "wraith",   // mode 0x3a98ee2d
	0x002ba902: "chopper",  // recoupe labels.tsv : `vehi 002ba902, sb_010_veh_bt_chopper`
	0x000026f2: "phantom",  // transport, non pilotable — sprite servi, jamais conduit
	0x000026f0: "pelican",  // transport, non pilotable
	0x86799cb6: "skiff",    // mode 0xa3aaa279
	0x000df0c4: "shade",    // recoupe labels.tsv : `vehi 000df0c4, sb_010_tur_cv_shadeturret`
	// Falcon : V4 § 4 le classe par ses maillages (mode 0xa0ca8a6f, 76 sections). CONFLIT NOTE,
	// non tranche ici : `labels.tsv` porte `vehi 0000254b +1, sb_010_veh_un_pelican` — mais le
	// « +1 » dit que le tag de degat a DEUX porteurs, et la banque citee peut etre celle de
	// l autre. La classification par maillage porte sur le tag `0000254b` LUI-MEME : elle est
	// plus directe, elle l emporte. OBSERVE en film.
	0x0000254b: "falcon",

	// --- Variantes de chassis rencontrees EN FILM, nommees par une seconde source ---
	// OBSERVE (0d76e8f1 + fccc61cd). `labels.tsv` : `vehi 5b80c406, sb_010_veh_cv_ghost`.
	0x5b80c406: "ghost",
	// OBSERVE (0d76e8f1 + fccc61cd). `labels.tsv` : `vehi c6e79dcc, sb_010_veh_cv_banshee`.
	0xc6e79dcc: "banshee",
	// OBSERVE (0d76e8f1 + fccc61cd). REWORK_WARTHOG_GUNGOOSE_2026-09-01 § 1 et
	// CONTACT_ARMES_GUNGOOSE_2026-09-02 § : « 3 vehi Mongoose (0x000025aa, 0xaf31ab1a,
	// 0xde26e3d7), tous -> mode 0x9e581380 » — le mode du chassis mongoose.
	0xaf31ab1a: "mongoose",
	0xde26e3d7: "mongoose", // meme source, meme mode ; non observe en film a ce jour
	// OBSERVE le 2026-09-02 sur `0d76e8f1` (3 vies), par le chemin de PRODUCTION lui-meme
	// (`replay-build`, journal « chassis de vehicule NON RESOLU »). `labels.tsv` le nomme sur ses
	// DEUX entrees a porteur UNIQUE — `vehi 3d4a8a5a, sb_010_veh_bt_chopper` (jpt 661b2987 et
	// 72230737). Ses deux autres entrees ont plusieurs porteurs (« +2 ») et ne tranchent rien :
	// ce sont celles-la qui sont ecartees, pas la resolution.
	0x3d4a8a5a: "chopper",
	// OBSERVE (0d76e8f1 : 9 vies ; fccc61cd : 4 vies) — le chassis le plus frequent du corpus
	// (143/441 naissances, V2_SPAWNS_COOLDOWNS_2026-09-01 § 1.2), longtemps irresolu parce que
	// `labels.tsv` porte son `vehi` SANS banque de sons. Resolu par la CHAINE DE DESTRUCTION
	// (V3D_DESTRUCTION_SONS_2026-09-02, table des verdicts) : `vehi fe32c0f4 -> hlmt daf7f543`,
	// le hlmt du Warthog (celui de 0x00002705), qui pose la banque d explosion
	// `exp_vehicle_med_unsc`. Un vehi qui partage le hlmt du Warthog est un chassis Warthog —
	// variante indistinguable, cf. la note d en-tete : la famille suffit.
	0xfe32c0f4: "warthog",
	// Meme preuve, meme hlmt daf7f543 (V3D, meme table) ; non observe en film a ce jour.
	0xcb96ca07: "warthog",
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
