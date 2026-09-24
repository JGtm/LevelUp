package filmshell

// unarmed.go — L OBJET « MAINS NUES » DU JEU, et la regle NOMMEE qui l ecarte des dotations
// (retours du rejeu, lot M6.3, 2026-09-24).
//
// L IDENTITE (sonde CA9 du 2026-09-23, `.ai/V7.5/retours_rejeu_2026-09-23/SONDE_CA9_objet_00007CA9.md`,
// et instrument `internal/himodule/m6_bobine_mains_nues_research_test.go`) : `00007CA9` est un
// tag `weap` du module `globals` (8 796 octets, sans projectile, sans chargeur, sans son de tir)
// que le script Lua global nomme `WeaponTags.unarmed`. Sa liste de variantes porte `42C9679F`,
// comme les armes de l arsenal : l identifiant d arme du film vaut donc `00007CA942C9679F`.
//
// CE QUE LE JEU EN FAIT, MESURE AU PARC (111 documents, 2026-09-24) : il le REMET a chaque bipede
// au debut de chaque vie — 590 ramassages natifs de classe ARME dans 79 documents, 487 au coup
// d envoi (t <= 5 frames), 75 a moins de 2 s d une reapparition, 28 a la bascule de manche d un
// match a manches. Aucun n est une prise dans le monde. Un joueur ne le TIENT en main que s il a
// jete toutes ses armes — cas « quasi impossible » selon l utilisateur (1 changement d arme sur le
// parc).
//
// DECISIONS DE L UTILISATEUR (2026-09-24) : exclu de la dotation affichee par une regle NOMMEE ;
// sa remise classee comme telle, hors des ramassages, avec un compteur dedie ; nomme « Mains
// nues » / « Unarmed » s il apparait en cours de partie (le libelle vit dans
// `config/titles/halo_infinite/mappings/weapon_names.toml`, cle `hinf_unarmed`).
//
// UNE SEULE ECRITURE : ce fichier est le seul de production a porter le litteral (garde-rail
// `internal/archlint/unarmed_family_literal_test.go`). Les dotations de naissance du lot M3
// (vague D) se rebranchent sur `IsUnarmedFamily`.

// UnarmedFamily est le GlobalID du tag `weap` « mains nues » — la famille d arme (32 bits hauts
// de l identifiant d arme du film) sous laquelle le film ecrit l objet.
const UnarmedFamily uint32 = 0x00007ca9

// unarmedVariant est la premiere variante de la liste du tag, lue dans ses octets (instrument M6).
const unarmedVariant uint64 = 0x42c9679f

// UnarmedWeaponID est l identifiant d arme 64 bits de l objet « mains nues » (famille puis
// variante), tel que le registre d armes du titre le range.
const UnarmedWeaponID = uint64(UnarmedFamily)<<32 | unarmedVariant

// IsUnarmedFamily dit si une famille d arme est l objet « mains nues ». C EST LA REGLE NOMMEE :
// une famille pour laquelle elle repond vrai n entre dans AUCUNE dotation publiee, et son
// ramassage natif est une remise du jeu, pas une prise.
func IsUnarmedFamily(family uint32) bool {
	return family == UnarmedFamily
}
