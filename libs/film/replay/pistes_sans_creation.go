package replay

// pistes_sans_creation.go — UN SLOT QUE AUCUN RECORD DE CREATION N'OUVRE N'EST PAS UN CORPS DE
// JOUEUR (schema 108).
//
// # LE DEFAUT, MESURE
//
// Les balayages ancres lisent des positions de bipede sur des slots qui n'en sont pas : `7aa132d1`
// publiait deux pistes sans identite (slots 36 et 292, quatre echantillons chacune, un objet qui
// parcourt une vingtaine de metres en 1,6 s) alors que ses 379 creations de bipede tombent toutes
// dans la bande de slots 512..767. Au parc, sur 1 440 films, aucune creation hors de la bande du
// film et des positions sans creation dans deux films seulement.
//
// # CE QUE LE FILM DIT
//
// Le moteur CREE l'entite (record `ti=35`), puis replique ses positions : tout corps de joueur a
// son record (positions_porte.go). Sur un film dont les records de creation sont lus, un slot qui
// n'en porte aucun et dont les vies restent sans nom apres toutes les lectures du registre n'a pas
// de corps a publier. Ce n'est pas une lecture : c'est l'absence d'une lecture que le film
// ecrit toujours, et elle se compte (`repli_position_de_bipede_sans_creation_ecartee`).
//
// # CE QUE LA REGLE EXIGE, ET RIEN D'AUTRE
//
//	des records de creation LUS    un film sans aucun record lu reste inchange (le registre n'a alors
//	                               aucune lecture directe, et c'est le fil des morts qui nomme)
//	le slot SANS RECORD            aucun record de creation, indexe ou non, pour ce slot
//	AUCUNE LECTURE NE LE NOMME     le registre d'identite, execute en entier (lecture directe,
//	                               fermetures, tableau de l'API, elimination, exclusion), laisse
//	                               TOUTES les vies du slot sans identite et sans cause autre que
//	                               `sans_record` ; un slot qu'une seule vie nommee touche reste entier
//	LE FILM EST COMPLET            le morceau des temps forts est lu (`coverage.bridge.deathsFeed` =
//	                               `read` ou `empty`) : la cuisson refuse en amont un film dont le
//	                               manifeste n'est pas finalise ou dont des morceaux sortent du
//	                               manifeste (`replaybuild`), donc un record ne peut manquer faute de
//	                               chunk. Le fait voyage dans les faits persistes : aucune montee de
//	                               leur schema
//
// # COMMENT
//
// Les positions du slot sortent AVANT le registre, qui est alors relance : une piste ecartee en
// aval laisserait au registre un slot sans nom que l'elimination ou la fermeture peuvent confondre
// avec un joueur reel, et la couverture du pont publierait des vies que le document ne porte pas.
// Le premier passage n'est qu'un essai (compteur jetable) ; seul le second compte et journalise.
// L'origine, la duree et le nombre de frames restent ceux des paquets de position du film.

import (
	"log/slog"

	"github.com/JGtm/halo-infinite-film-decoder/identity"
	"github.com/JGtm/halo-infinite-film-decoder/internal/facts/fallback"
	"github.com/JGtm/halo-infinite-film-decoder/internal/grammar"
)

// slotsSansRecordDeCreation rend les slots qui portent des positions et aucun record de creation,
// sur un film dont des records de creation sont LUS (au moins un porte son index de participant).
// nil quand le film n'en lit aucun.
func slotsSansRecordDeCreation(pos []grammar.BipedPosition, creations []grammar.BipedCreation) map[uint32]struct{} {
	avecRecord := make(map[uint32]struct{}, len(creations))
	lus := false
	for _, c := range creations {
		avecRecord[c.Slot] = struct{}{}
		lus = lus || c.HasIndex
	}
	if !lus {
		return nil
	}
	var out map[uint32]struct{}
	for _, p := range pos {
		if _, ouvert := avecRecord[p.Slot]; ouvert {
			continue
		}
		if out == nil {
			out = map[uint32]struct{}{}
		}
		out[p.Slot] = struct{}{}
	}
	return out
}

// filmDesTempsFortsLu dit que le morceau des temps forts du film est lu : sa lecture a rendu des
// morts, ou un fil vide sans erreur. C'est la preuve que porte le document de la completude des
// morceaux (cf. l'en-tete) ; faute de verdict, la regle se tait.
func filmDesTempsFortsLu(opt Options) bool {
	v := deathsFeedPublie(opt.DeathsFeed, len(opt.Deaths))
	return v == DeathsFeedRead || v == DeathsFeedEmpty
}

// slotsSansRecordNiNom rend, parmi `candidats`, les slots dont TOUTES les vies du registre sont
// sans record de creation et restent sans identite. Un slot sans aucune vie au registre (table
// d'index vide, film sans pont) n'en fait pas partie : rien ne prouve qu'il n'est pas un joueur.
func (r IdentityRegistry) slotsSansRecordNiNom(candidats map[uint32]struct{}) map[uint32]struct{} {
	vies := map[uint32]int{}
	sansNom := map[uint32]int{}
	for i, l := range r.Vies() {
		if _, candidat := candidats[l.slot]; !candidat {
			continue
		}
		vies[l.slot]++
		if l.xuid == 0 && l.bid == "" && r.CauseNonResolue(i) == identity.MethodNoCreationRecord {
			sansNom[l.slot]++
		}
	}
	out := map[uint32]struct{}{}
	for s, n := range vies {
		if sansNom[s] == n {
			out[s] = struct{}{}
		}
	}
	return out
}

// sansLesSlots rend les positions des autres slots, dans le meme ordre. L'entree n'est pas modifiee.
func sansLesSlots(pos []grammar.BipedPosition, slots map[uint32]struct{}) ([]grammar.BipedPosition, int) {
	out := make([]grammar.BipedPosition, 0, len(pos))
	for _, p := range pos {
		if _, ecarte := slots[p.Slot]; !ecarte {
			out = append(out, p)
		}
	}
	return out, len(pos) - len(out)
}

// construireLeRegistre construit le registre d'identite du film, apres avoir ecarte les positions
// des slots que AUCUN record de creation n'ouvre et qu'aucune lecture ne nomme (cf. l'en-tete).
//
// SANS CANDIDAT, UN SEUL PASSAGE, le compteur de la cuisson en main : c'est le cas de presque tous
// les films, et leur document ne change pas d'un octet.
func (a *assemblage) construireLeRegistre(entree func(*fallback.Compteur) IdentityInput) IdentityRegistry {
	var candidats map[uint32]struct{}
	if filmDesTempsFortsLu(a.opt) {
		candidats = slotsSansRecordDeCreation(a.sorted, a.opt.BipedCreations)
	}
	if len(candidats) == 0 {
		return BuildIdentityRegistry(a.ctx, entree(a.opt.Fallbacks))
	}
	essai := BuildIdentityRegistry(a.ctx, entree(fallback.NouveauCompteur()))
	ecartes := essai.slotsSansRecordNiNom(candidats)
	if len(ecartes) > 0 {
		var positions int
		a.sorted, positions = sansLesSlots(a.sorted, ecartes)
		a.opt.Fallbacks.DeclencheN(fallback.NomPositionDeBipedeSansCreationEcartee, positions)
		// JOURNALISE, JAMAIS AVALE : ces positions ne se publient pas.
		slog.WarnContext(a.ctx, "rejeu : slots sans record de creation ni identite ecartes — aucune piste publiee",
			"match_id", a.matchID, "slots", len(ecartes), "positions", positions)
	}
	return BuildIdentityRegistry(a.ctx, entree(a.opt.Fallbacks))
}
