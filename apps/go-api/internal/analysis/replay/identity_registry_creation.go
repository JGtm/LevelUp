package replay

// identity_registry_creation.go — LE LIEN DIRECT CORPS -> JOUEUR, LU DANS LE FILM.
//
// # CE QUE LE FILM DIT, ET QU'ON NE LUI DEMANDAIT PAS
//
// Le record de CREATION d'un bipede (`ti=35`) porte, dans son default-state, l'index de
// participant de son proprietaire — `filmdec.ScanBipedCreations` le lit (specification
// bit-exacte : `.ai/V7.5/film_re/SONDAGE_E2_BIPEDE_INDEX_2026-09-08.md` §2.2). C'est une
// LECTURE : le film ECRIT a qui appartient ce corps. Jusqu'au lot E2, le registre l'ignorait et
// devinait la meme chose par le fil des morts.
//
// # POURQUOI LA PROPAGATION N'EST PAS UNE DEDUCTION
//
// Mesure du lot E2 sur cinq films (538 vies, 499 slots) : **un seul record de creation par
// slot**, generation invariante a 1, et l'ensemble des slots qui portent des VIES egale
// exactement l'ensemble des slots qui portent une LECTURE. Dans un film, un slot de bipede est
// donc UN CORPS du debut a la fin — le pool de handles ne reboucle pas a cette echelle.
//
// Une vie sans record propre est alors le MEME corps qu'une decoupe a `lifeGapUS` (5 s) a separe
// d'un sejour continu : embarquement en vehicule, occultation, trou de replication, ou le trou
// d'ouverture entre l'apparition du corps et le coup d'envoi. Lui appliquer le record de son
// slot, ce n'est pas deviner un occupant : c'est lire le MEME record. D'ou `LinkDirect` et une
// voie qui le NOMME (`creation_bipede_propagee`), plutot qu'un `LinkInferred` qui la ferait
// passer pour une supposition.
//
// # ET CE QU'ELLE REFUSE, MESURE A ZERO AUJOURD'HUI
//
// Si deux records du MEME slot portaient des index DIFFERENTS, la propagation deviendrait un
// choix : seules les vies qu un record OUVRE sont alors nommees, par ce record-la, et les autres
// passent `non_resolu` cause `lectures_divergentes`. Ce refus n'a jamais eu a jouer (0 slot a
// deux records sur les cinq films), et c'est precisement pour cela qu'il doit exister : le jour
// ou le pool rebouclerait dans un film plus long, la propagation par slot deviendrait fausse
// sans lui.
//
// # L'INDEX QUI N'EST PAS DANS LA TABLE (verdict I0 du lot E2)
//
// L'espace d'index est PARTAGE entre humains et bots. Un index lu que `PlayerIndexTable` ne
// nomme pas designe un participant que l'artefact ne publie pas : la vie reste `non_resolu`,
// cause `index_hors_table`, JAMAIS rattachee. Quand le film DECLARE un bot a cet index
// (`BOT_METADATA`), la lecture se compte a part (`IndexBot`) et n'alarme pas — on sait alors qui
// occupe le corps, on n'a simplement pas de xuid a poser.

import (
	"log/slog"
	"sort"

	"levelup/go-api/internal/analysis/filmdec"
	"levelup/go-api/internal/games/canonical"
)

// Les DEUX voies de nommage par le record de creation. Elles repondent a « comment sait-on a qui
// cette vie appartient », et toutes deux disent : LE FILM L'ECRIT.
//
//	NomParCreation           le record de creation du corps OUVRE cette vie — c est le sejour
//	                         qu'il inaugure (cf. appliquerAuCorps pour la regle exacte).
//	NomParCreationPropagee   le record du MEME corps ouvre un AUTRE de ses sejours — la decoupe a
//	                         `lifeGapUS` les a separes. Meme record, meme corps, meme lecture.
const (
	NomParCreation         = "biped_creation"
	NomParCreationPropagee = "biped_creation_propagee"
)

// creationReport porte ce que la lecture directe a pose et ce qu'elle a refuse. Les refus sont
// ventiles par CAUSE : « non resolu » sans sa cause ne se corrige pas, il se contemple.
type creationReport struct {
	// Records / Slots : le denominateur de la lecture — records recus, slots couverts.
	Records, Slots int
	// Direct / Propagated : les vies nommees, selon que le record OUVRE cette vie ou un AUTRE
	// sejour du meme corps. `Propagated` est un SOUS-COMPTE de la couverture directe.
	Direct, Propagated int
	// IndexBot : lectures dont l'index est celui d'un bot DECLARE. Comptees a part, jamais
	// alarmees (verdict I0) : le corps est identifie, il n'a simplement pas de xuid.
	IndexBot int
	// causes : indice de vie -> cause de non-resolution. Une vie absente de cette table est
	// nommee ; une vie presente ne l'est pas, et la table DIT pourquoi.
	causes map[int]canonical.LinkMethod
}

// Lues rend le nombre de vies que la lecture directe a nommees, propagation comprise.
func (r creationReport) Lues() int { return r.Direct + r.Propagated }

// CauseNonResolue rend la CAUSE prouvee pour laquelle la vie d'indice `i` n'est pas nommee, ou
// [canonical.MethodNone] quand elle l'est.
//
// ELLE N'EST PAS TOUJOURS RENSEIGNEE, ET C'EST EXACT : une vie que la lecture directe a laissee
// sans nom mais qu'une DEDUCTION a ensuite nommee (fermeture, elimination, exclusion) porte une
// cause qui ne vaut plus. Le producteur ne la lit que pour les vies restees anonymes.
func (r IdentityRegistry) CauseNonResolue(i int) canonical.LinkMethod {
	return r.creation.causes[i]
}

// nommerViesParCreations pose le lien DIRECT sur les vies, AVANT toute autre voie.
//
// Elle n'ecrase jamais rien : elle est la PREMIERE etape de [BuildIdentityRegistry], les vies y
// arrivent anonymes. Elle rend le rapport, jamais une decision — le registre en tire la
// couverture et les alarmes.
func nommerViesParCreations(lives []lifeSpan, creations []filmdec.BipedCreation,
	idx PlayerIndexTable, bots []BotIdentity) creationReport {
	rep := creationReport{Records: len(creations), causes: map[int]canonical.LinkMethod{}}
	if len(lives) == 0 {
		return rep
	}
	corps := corpsParSlot(creations)
	rep.Slots = len(corps)
	if len(corps) == 0 {
		// AUCUNE LECTURE DIRECTE : toutes les vies sont `sans_record`, et c'est la SEULE
		// situation ou le pont par morts garde un role de nommage (cf. identity_registry.go).
		for i := range lives {
			rep.causes[i] = canonical.MethodNoCreationRecord
		}
		return rep
	}
	versXUID, versBot := indexToXUIDOf(idx.ByXUID), indexDesBotsDeclares(bots)
	for slot, vies := range indicesDeViesParSlot(lives) {
		c, ok := corps[slot]
		if !ok {
			for _, i := range vies {
				rep.causes[i] = canonical.MethodNoCreationRecord
			}
			continue
		}
		rep.appliquerAuCorps(lives, vies, c, resolutionDIndex{versXUID: versXUID, versBot: versBot})
	}
	return rep
}

// resolutionDIndex porte les deux resolutions d un index de participant. Une structure plutot que
// parametres de plus : le depot borne a cinq, et les deux vont toujours ensemble.
type resolutionDIndex struct {
	versXUID map[int]uint64
	versBot  map[int]bool
}

// indicesDeViesParSlot groupe les INDICES des vies par slot, dans l'ordre chronologique.
//
// L'ordre vient de `buildLifeSpans`, qui trie par slot puis par instant : le groupement le
// conserve. C'est ce qui permet de dire QUELLE vie un record OUVRE.
func indicesDeViesParSlot(lives []lifeSpan) map[uint32][]int {
	out := map[uint32][]int{}
	for i := range lives {
		out[lives[i].slot] = append(out[lives[i].slot], i)
	}
	return out
}

// appliquerAuCorps pose (ou refuse) le lien d'UN corps sur TOUTES ses vies.
//
// # « DIRECT » EST LA VIE QUE LE RECORD OUVRE, PAS CELLE QUI CONTIENT SA DATE
//
// Mesure du lot E2 : sur les cinq films, AUCUN record de creation ne tombe dans l'intervalle
// d'une vie — le moteur cree l'entite, puis replique ses positions dans un paquet ULTERIEUR. Un
// partage par containment rendrait donc « propage » partout, ce qui serait exact au sens litteral
// et faux au sens utile : le record ouvre bien UNE vie, celle du sejour qu'il inaugure. La regle
// retenue est donc « la premiere vie du corps qui n'est pas deja terminee a la date du record »,
// et il y a exactement un `direct` par record.
func (r *creationReport) appliquerAuCorps(lives []lifeSpan, vies []int, c corpsLu, t resolutionDIndex) {
	ouvertes := c.viesOuvertes(lives, vies)
	for _, i := range vies {
		pi, ouverte := ouvertes[i]
		if !ouverte {
			if len(c.index) != 1 {
				// LECTURES DIVERGENTES : la propagation deviendrait un choix. On se tait.
				r.causes[i] = canonical.MethodDivergentReadings
				continue
			}
			pi = c.index[0]
		}
		if !r.poser(lives, i, int(pi), t) {
			continue
		}
		if ouverte {
			lives[i].nomPar = NomParCreation
			r.Direct++
			continue
		}
		lives[i].nomPar = NomParCreationPropagee
		r.Propagated++
	}
}

// poser resout l'index en xuid et l'ecrit sur la vie. Rend false — avec sa cause — quand l'index
// n'est pas dans la table publiee : un participant que l'artefact ne nomme pas ne se rattache
// JAMAIS au premier venu (verdict I0).
func (r *creationReport) poser(lives []lifeSpan, i, pi int, t resolutionDIndex) bool {
	if t.versBot[pi] {
		// BOT DECLARE : le corps est identifie, il n'y a pas de xuid a poser. La lecture se
		// compte, la vie garde ses autres voies, et rien n'alarme (verdict I0).
		r.IndexBot++
		r.causes[i] = canonical.MethodIndexOutOfTable
		return false
	}
	x, connu := t.versXUID[pi]
	if !connu {
		r.causes[i] = canonical.MethodIndexOutOfTable
		return false
	}
	lives[i].xuid = x
	return true
}

// viesOuvertes apparie chaque record du corps a LA VIE QU'IL OUVRE : la premiere vie du slot que
// le record ne trouve pas deja terminee, et qu'aucun record anterieur n'a deja ouverte. Rend
// l'index de participant par indice de vie.
func (c corpsLu) viesOuvertes(lives []lifeSpan, vies []int) map[int]uint32 {
	out := map[int]uint32{}
	for _, d := range c.dates {
		for _, i := range vies {
			if _, deja := out[i]; deja || lives[i].to < d.tUS {
				continue
			}
			out[i] = d.index
			break
		}
	}
	return out
}

// corpsLu est ce que les records d'UN slot etablissent : les index lus, et leurs dates.
type corpsLu struct {
	// index est l'ensemble des index de participant lus sur ce slot, en ordre croissant. Plus
	// d'un = les lectures divergent.
	index []uint32
	// dates est, pour chaque record, son horodatage et l'index qu'il porte.
	dates []dateDeCreation
}

// dateDeCreation associe l'instant d'un record a l'index qu'il porte.
type dateDeCreation struct {
	tUS   int64
	index uint32
}

// corpsParSlot groupe les records de creation par slot. Les records SANS index ne sont pas des
// lectures : ils ne fondent aucun lien et n'entrent pas dans le groupe.
func corpsParSlot(creations []filmdec.BipedCreation) map[uint32]corpsLu {
	if len(creations) == 0 {
		return nil
	}
	out := map[uint32]corpsLu{}
	vus := map[uint32]map[uint32]bool{}
	for _, c := range creations {
		if !c.HasIndex {
			continue
		}
		e := out[c.Slot]
		e.dates = append(e.dates, dateDeCreation{tUS: int64(c.TimestampUS), index: c.ParticipantIndex})
		if vus[c.Slot] == nil {
			vus[c.Slot] = map[uint32]bool{}
		}
		if !vus[c.Slot][c.ParticipantIndex] {
			vus[c.Slot][c.ParticipantIndex] = true
			e.index = append(e.index, c.ParticipantIndex)
		}
		out[c.Slot] = e
	}
	for s, e := range out {
		// L'ORDRE EST IMPOSE : l'artefact doit etre reproductible a l'octet, et l'ordre des
		// records d'un chunk n'est pas garanti stable entre deux lectures.
		sort.Slice(e.dates, func(i, j int) bool { return e.dates[i].tUS < e.dates[j].tUS })
		sort.Slice(e.index, func(i, j int) bool { return e.index[i] < e.index[j] })
		out[s] = e
	}
	return out
}

// ownersFromCreations rend le pont slot -> index de joueur que les CREATIONS etablissent, pour
// les seuls slots dont les records concordent.
//
// POURQUOI IL EXISTE A COTE DE `ownersFromLives` : celui-ci passe par les vies NOMMEES, donc par
// un xuid, donc il ignore les corps de bot. Le record de creation, lui, porte l'index quel que
// soit le participant — c'est ce qui donne au nommage des pistes de bot un pont DIRECT plutot
// qu'une fermeture.
func ownersFromCreations(creations []filmdec.BipedCreation) map[uint32]int {
	out := map[uint32]int{}
	for s, c := range corpsParSlot(creations) {
		if len(c.index) == 1 {
			out[s] = int(c.index[0])
		}
	}
	return out
}

// indexDesBotsDeclares rend les index de participant que `BOT_METADATA` declare.
func indexDesBotsDeclares(bots []BotIdentity) map[int]bool {
	out := make(map[int]bool, len(bots))
	for _, b := range bots {
		out[b.FilmIndex] = true
	}
	return out
}

// alarmerSurLesRefus journalise ce que la lecture directe n'a pas su poser. Un refus muet est
// exactement ce que la doctrine §0.2 interdit ; un refus SANS CAUSE ne se corrige pas.
func (r creationReport) alarmerSurLesRefus(matchID string, causes canonical.UnresolvedCauses) {
	slog.Info("rejeu : lien direct corps -> joueur",
		"match_id", matchID, "records", r.Records, "corps", r.Slots,
		"direct", r.Direct, "propage", r.Propagated, "indexBot", r.IndexBot)
	if causes.IndexOutOfTable > r.IndexBot {
		slog.Warn("rejeu : index de participant LU mais absent de la table publiee — vies NON "+
			"rattachees (verdict I0 : participant que PlayerIndexTable ne nomme pas)",
			"match_id", matchID, "vies", causes.IndexOutOfTable-r.IndexBot,
			"dontBotsDeclares", r.IndexBot)
	}
	if causes.DivergentReadings > 0 {
		slog.Warn("rejeu : deux records de creation du MEME corps portent des index differents — "+
			"la propagation se tait", "match_id", matchID, "vies", causes.DivergentReadings)
	}
	if r.Slots > 0 && causes.NoCreationRecord > 0 {
		slog.Warn("rejeu : vies de bipede sans record de creation sur un film qui en porte — "+
			"classe attendue VIDE (lot E2 : bijection slots<->records mesuree sur 5 films)",
			"match_id", matchID, "vies", causes.NoCreationRecord, "corps", r.Slots)
	}
}
