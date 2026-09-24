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
// CE QUE LE JEU EN FAIT, MESURE AU PARC (107 documents reconstruits depuis les faits persistes a la
// tete de la vague C, 2026-09-24 ; instrument `replaybuild/m1_parc_depuis_faits_research_test.go`) :
// il le REMET a chaque bipede au debut de chaque vie. Le film l ecrit sur DEUX canaux, chaque fois
// comme une prise : 562 ramassages natifs de classe ARME dans 77 documents — 481 au coup d envoi
// (t <= 5 frames), 457 a moins de 2 s du premier point d une vie du meme slot, 105 AVANT le premier
// point publie de la vie de leur slot, aucun au milieu d une vie —, et 1 prise (`taken`) du canal
// `weaponChanges`, au premier point d une vie elle aussi. Aucune n est une prise dans le monde. Un
// joueur ne le TIENT en main que s il a jete toutes ses armes — cas « quasi impossible » selon
// l utilisateur, jamais observe au parc : il apparaitrait comme un ECHANGE (`swapped`) vers
// l objet, publie et nomme.
//
// DECISIONS DE L UTILISATEUR (2026-09-24) : exclu de la dotation affichee par une regle NOMMEE ;
// sa remise classee comme telle, hors des ramassages, avec un compteur dedie ; nomme « Mains
// nues » / « Unarmed » s il apparait en cours de partie (le libelle vit dans
// `config/titles/halo_infinite/mappings/weapon_names.toml`, cle `hinf_unarmed`).
//
// OU LA REGLE S APPLIQUE (trois chemins de publication) : `pickups` et `weaponChanges` (la remise
// sort des publies, comptee dans leur `unarmedGrants`), et les dotations, par le passage unique
// `dotationWeaponName` (`film/replay/loadouts.go`). UNE SEULE ECRITURE du litteral : ce fichier
// (garde-rail `internal/archlint/unarmed_family_literal_test.go`) ; UN SEUL passage des dotations
// vers le catalogue d armes (garde-rail `internal/archlint/unarmed_dotation_gate_test.go`) — les
// dotations de naissance du lot M3 (vague D) devront l emprunter pour passer ce garde-rail.

// UnarmedFamily est le GlobalID du tag `weap` « mains nues » — la famille d arme (32 bits hauts
// de l identifiant d arme du film) sous laquelle le film ecrit l objet.
const UnarmedFamily uint32 = 0x00007ca9

// unarmedVariant est la premiere variante de la liste du tag, lue dans ses octets (instrument M6).
const unarmedVariant uint64 = 0x42c9679f

// UnarmedWeaponID est l identifiant d arme 64 bits de l objet « mains nues » (famille puis
// variante), tel que le registre d armes du titre le range.
const UnarmedWeaponID = uint64(UnarmedFamily)<<32 | unarmedVariant

// IsUnarmedFamily dit si une famille d arme est l objet « mains nues ». C EST LA REGLE NOMMEE :
// une famille pour laquelle elle repond vrai n entre dans AUCUNE dotation publiee, et sa PRISE
// (ramassage natif de classe arme, `taken` du canal des changements d arme) est une remise du
// jeu, pas un ramassage.
func IsUnarmedFamily(family uint32) bool {
	return family == UnarmedFamily
}
