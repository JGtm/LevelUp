package grammar

// generations_vivantes.go — QUELLES GENERATIONS DU HANDLE D UN SLOT BIPEDE SONT VIVANTES (lot J5.2
// du PLAN_SUITE_AUDIT_DECODEUR_FILM_2026-09-25, decision DT-8, constat GB-1 de l audit du
// 2026-09-24).
//
// # LE DEFAUT QUE CE FICHIER FERME
//
// Les positions du bipede et ses huit canaux delta n acceptaient un record que si les deux bits de
// generation de son handle valaient 1 (`ScanFilmOptions.RequireTag1`, et trois copies en dur du
// meme filtre). Or ces deux bits sont la GENERATION du handle (`readRecordID` : « the tag is
// generation ») : quand le pool de slots bipedes reboucle — un BTB long —, le slot d un joueur mort
// est rendu a un autre corps sous la generation 2, puis 3. Ces corps n avaient AUCUNE position ni
// aucune lecture delta. Mesure du lot J5.0 (`.ai/V7.5/film_re/MESURE_GB1_2026-09-27.md`) : 123 vies
// sur 379 sans une position sur `084a804d`, 330 sur 586 sur `1c4c63c2`, dont le rejeu s arretait a
// 826 s pour un film de 1 334 s.
//
// # LE PREDICAT, ET D OU IL VIENT
//
// Une generation est VIVANTE pour un slot quand une lecture de production l a vue designer un corps :
// un record de CREATION de bipede ([ScanBipedCreations], le handle de l en-tete NEW) ou un record
// `ti=35` d IMAGE-CLE (la marche de l image-cle du contexte), sur tout le film. Un record delta est
// lu quand sa generation est vivante pour son slot. Le filtre garde le pouvoir discriminant des deux
// bits — un record d une generation qu aucune lecture n a vue reste ecarte (bruit d ancrage bit a
// bit) — sans perdre les corps de generation >= 2. La mesure J5.0 le chiffre : quelques dizaines
// d en-tetes « orphelins » par film, contre des centaines de milliers de positions recuperees, et
// AUCUNE position de production dont la vie (slot, 1) serait inconnue (19 films).
//
// # LE REPLI NOMME : `repli_generation_vivante_inconnue_tag1`
//
// Un slot dont AUCUNE generation n est connue (un corps cree puis detruit entre deux images-cles,
// sans creation acceptee) retombe sur l ancien filtre : generation 1 seulement. Il est inscrit au
// registre (`facts/fallback/registre_filmdec.go`), et COMPTE par la cuisson depuis les positions
// publiees ([GenerationsVivantes.SlotsEnRepli]) ; cette couche le nomme et ne compte rien (ADR 0034
// D-4). Un ensemble nil ou vide — le coeur pur [ScanBipedRecords] quand l appelant ne fournit rien,
// le detecteur de decoupage — met TOUS les slots dans ce repli : c est bit a bit l ancien filtre.
//
// # UNE GENERATION VIVANTE N EST PAS VIVANTE A TOUT INSTANT (lot R2, 2026-09-28, constat C2 du
// G-corpus J11.1)
//
// Le masque ci-dessus est ATEMPOREL : il dit qu un corps (slot, generation) a existe, pas QUAND. Or
// le moteur cree l entite (record NEW) PUIS replique ses positions (lot E2 : 10 a 30 ms apres) ; un
// record delta dont le handle designe un corps AVANT le record de creation de ce corps n est donc la
// replication d aucun corps — a cet instant le slot porte le corps precedent. Mesure sur les quatre
// films a slot reboucle (instrument `r2_positions_aberrantes_research_test.go`) : 32, 30, 20 et 102
// en-tetes ancres de generation >= 2 AVANT la creation de leur corps, dont 14, 6, 5 et 28 publies en
// positions de production — TOUS aberrants (z de -937 a -946 m sur Fortitude, borne de carte
// (-981,5 ; -688,76 ; -108,12) sur `a349fea8` slot 532, premier point de la piste slot 516 de
// `084a804d` 3,3 s avant sa creation). C est la cause des bornes de scene faussees et des sauts
// impossibles publies depuis le lot J5.2.
//
// D ou la garde DATEE : un record lu A UN INSTANT ([GenerationsVivantes.A]) est refuse quand son
// corps a un record de creation lu POSTERIEUR a cet instant et que le slot porte alors un AUTRE
// corps (ou aucun). EXCEPTION, et c est la frontiere avec `replay` : le PREMIER corps d un slot, de
// generation 1, reste regi par la regle R-B1 de `replay/positions_porte.go` (positions anterieures
// ecartees ET comptees dans `coverage.tracks.avantCreation`) — cette garde ne rejoue pas la regle
// que le rejeu applique et compte deja, elle la porte la ou le rejeu ne la voit pas (la position
// publiee n a pas de generation). Un corps connu par les seules images-cles (aucune creation lue)
// n a pas de date : il n est pas garde. Un balayage non date (instruments, coeur pur) garde le
// filtre atemporel.
//
// # TOUS LES LECTEURS DE RECORDS DELTA BIPEDES, PAS LES SEULES POSITIONS (lot R2-bis, 2026-09-29)
//
// Le lot R2 n avait date que le balayage des positions. Le MEME marcheur servait les huit canaux delta
// (`walkDeltaBipedRecords` : rang et charges de capacite, impulsions, camouflage, grappin, arme tenue,
// inventaire, equipement de l unite) avec le masque atemporel, et deux lecteurs voisins aussi (la
// recuperation d equipement, `equipment_recovery.go`, et la visee seule, `offline_aim_only.go`) :
// 32, 30, 20 et 102 en-tetes anterieurs a la creation de leur corps y etaient encore acceptes sur
// `084a804d`, `a349fea8`, `4f77afc1`, `1c4c63c2` (decouverte 2 du lot R2). Ils prennent desormais leur
// filtre par [FilmContext.GenerationsVivantesA], a l instant du paquet porteur — la MEME garde, par la
// MEME fonction [GenerationsVivantes.A]. Garde-rail : `generations_vivantes_datees_ratchet_test.go`
// (l acces atemporel [FilmContext.GenerationsVivantes] n a plus que deux appelants de production
// nommes, qui le datent ou ne jugent aucun record).
//
// # LES OBJETS DU MONDE : TOUTES LES GENERATIONS, EXPLICITEMENT
//
// Un vehicule ou un objet du monde emploie les quatre valeurs du tag. [ToutesLesGenerations] le dit
// en toutes lettres a l appelant (`replay/build_vehicles.go`) ; il n y a plus de booleen a desarmer.

import (
	"cmp"
	"slices"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// generationDuRepli : la seule generation que le repli `repli_generation_vivante_inconnue_tag1`
// accepte sur un slot dont aucune generation n est connue — le filtre d avant le lot J5.2.
const generationDuRepli = 1

// generationDuPremierCorps : la generation du record de creation du PREMIER corps d un slot, celle
// dont la regle R-B1 du rejeu garde les positions anterieures a la creation (meme constante mesuree
// que `replay.premiereGenerationDuCorps`, lot E2).
const generationDuPremierCorps = 1

// GenerationsVivantes dit, slot par slot, quelles generations du handle designent un corps.
// Le zero n est pas utilisable : passer par [NouvellesGenerationsVivantes] ou
// [ToutesLesGenerations] ; nil est accepte partout et vaut « aucune generation connue ».
type GenerationsVivantes struct {
	// toutes : le filtre est leve (objets du monde, instruments).
	toutes bool
	// masques : indexe par le slot ; le bit g vaut 1 quand la generation g est vivante.
	masques []uint8
	// creations : indexe par le slot, les records de creation lus, tries par instant puis
	// generation — les dates des corps successifs du siege (lot R2). Vide : aucune garde datee.
	creations [][]creationDatee
	// instant / date : l instant du paquet porteur, pose par [GenerationsVivantes.A]. Sans date,
	// le filtre est atemporel.
	instant uint64
	date    bool
}

// creationDatee : l instant d un record de creation de bipede et la generation du corps qu il ouvre.
type creationDatee struct {
	tUS uint64
	gen uint32
}

// ToutesLesGenerations rend le filtre LEVE : un record est lu quelle que soit sa generation. C est
// le reglage des objets du monde (vehicules), dont le tag emploie ses quatre valeurs.
func ToutesLesGenerations() *GenerationsVivantes { return &GenerationsVivantes{toutes: true} }

// NouvellesGenerationsVivantes construit l ensemble a partir des vies lues. Une vie hors domaine
// (slot au-dela de 13 bits, generation au-dela de 2) est ignoree : elle ne designe aucun handle.
func NouvellesGenerationsVivantes(vies []types.LifeKey) *GenerationsVivantes {
	g := &GenerationsVivantes{masques: make([]uint8, 1<<handleSlotBits)}
	for _, v := range vies {
		if v.Slot < uint32(len(g.masques)) && v.Gen < 1<<handleGenBits {
			g.masques[v.Slot] |= 1 << v.Gen
		}
	}
	return g
}

// masque rend le masque des generations vivantes d un slot, 0 quand aucune n est connue.
func (g *GenerationsVivantes) masque(slot uint32) uint8 {
	if g == nil || slot >= uint32(len(g.masques)) {
		return 0
	}
	return g.masques[slot]
}

// avecCreations pose les dates des corps (records de creation lus) : la garde datee de
// [GenerationsVivantes.A]. Rend le recepteur.
func (g *GenerationsVivantes) avecCreations(cre []BipedCreation) *GenerationsVivantes {
	if g == nil || g.toutes || len(cre) == 0 {
		return g
	}
	g.creations = make([][]creationDatee, len(g.masques))
	for _, c := range cre {
		if c.Slot < uint32(len(g.creations)) {
			g.creations[c.Slot] = append(g.creations[c.Slot], creationDatee{tUS: c.TimestampUS, gen: c.Generation})
		}
	}
	for _, l := range g.creations {
		slices.SortFunc(l, func(a, b creationDatee) int {
			return cmp.Or(cmp.Compare(a.tUS, b.tUS), cmp.Compare(a.gen, b.gen))
		})
	}
	return g
}

// A rend le filtre DATE a l instant `tUS` (horloge des paquets) : celui d un record lu dans un
// paquet de cet instant. Sans creation connue (ou filtre leve, ou nil), le recepteur lui-meme.
func (g *GenerationsVivantes) A(tUS uint64) *GenerationsVivantes {
	if g == nil || g.toutes || len(g.creations) == 0 {
		return g
	}
	v := *g
	v.instant, v.date = tUS, true
	return &v
}

// Accepte dit si un record de ce handle se lit. Slot sans generation connue : le repli
// `repli_generation_vivante_inconnue_tag1` (generation 1 seulement). Filtre date : un corps n est
// pas lu avant son record de creation (cf. l en-tete, lot R2).
func (g *GenerationsVivantes) Accepte(h types.LifeKey) bool {
	if g != nil && g.toutes {
		return true
	}
	if m := g.masque(h.Slot); m != 0 {
		return m&(1<<h.Gen) != 0 && !g.avantSaCreation(h)
	}
	return h.Gen == generationDuRepli
}

// avantSaCreation dit que le corps `h` n est pas encore cree a l instant du filtre : le slot porte
// alors un autre corps (ou aucun), et un record de creation de `h` est lu PLUS TARD. Le premier
// corps d un slot, de generation 1, en est exempte : la regle R-B1 du rejeu le regit (cf. l en-tete).
func (g *GenerationsVivantes) avantSaCreation(h types.LifeKey) bool {
	if !g.date || h.Slot >= uint32(len(g.creations)) {
		return false
	}
	cs := g.creations[h.Slot]
	tenant := -1
	for i := range cs {
		if cs[i].tUS > g.instant {
			break
		}
		tenant = i
	}
	if tenant >= 0 && cs[tenant].gen == h.Gen {
		return false
	}
	for i := tenant + 1; i < len(cs); i++ {
		if cs[i].gen == h.Gen {
			return i > 0 || h.Gen != generationDuPremierCorps
		}
	}
	return false
}

// Connue dit si au moins une generation du slot est connue (ou si le filtre est leve).
func (g *GenerationsVivantes) Connue(slot uint32) bool {
	return g != nil && (g.toutes || g.masque(slot) != 0)
}

// SlotsEnRepli compte les slots DISTINCTS des positions publiees dont aucune generation n est
// connue : ceux dont les records ont ete lus par le repli `repli_generation_vivante_inconnue_tag1`.
// C est le compte que la cuisson publie dans `coverage.fallbacks`.
func (g *GenerationsVivantes) SlotsEnRepli(pos []BipedPosition) int {
	if g != nil && g.toutes {
		return 0
	}
	vus := map[uint32]bool{}
	for _, p := range pos {
		if !g.Connue(p.Slot) {
			vus[p.Slot] = true
		}
	}
	return len(vus)
}

// memoDesVies : ce que [FilmContext] memorise des vies du bipede. Paresseux, comme les autres
// derivations du contexte (cf. film_context.go) : calcule a la premiere demande.
type memoDesVies struct {
	creations     []BipedCreation
	stats         types.BipedCreationStats
	err           error
	creationsLues bool
	vivantes      *GenerationsVivantes
}

// CreationsDeBipede rend les records de creation de bipede du film ([ScanBipedCreations]), lus UNE
// fois. La tranche rendue est une COPIE : un appelant qui la trie ne touche pas celle du contexte.
func (c *FilmContext) CreationsDeBipede() ([]BipedCreation, types.BipedCreationStats, error) {
	if c == nil {
		return ScanBipedCreations(nil)
	}
	if !c.vies.creationsLues {
		c.vies.creations, c.vies.stats, c.vies.err = ScanBipedCreations(c)
		c.vies.creationsLues = true
	}
	return slices.Clone(c.vies.creations), c.vies.stats, c.vies.err
}

// GenerationsVivantes rend les generations vivantes du handle bipede de ce film, relevees une fois
// sur les records de creation ([FilmContext.CreationsDeBipede]) et les records `ti=35` de TOUTES
// les images-cles. Jamais nil.
func (c *FilmContext) GenerationsVivantes() *GenerationsVivantes {
	if c == nil {
		return NouvellesGenerationsVivantes(nil)
	}
	if c.vies.vivantes == nil {
		// LES DATES DES CORPS : celles des records de creation (lot R2). Une creation illisible ne
		// date rien, et le filtre reste atemporel.
		cre, _, err := c.CreationsDeBipede()
		if err != nil {
			cre = nil
		}
		c.vies.vivantes = NouvellesGenerationsVivantes(viesConnuesDuFilm(c)).avecCreations(cre)
	}
	return c.vies.vivantes
}

// GenerationsVivantesA rend le filtre de generation de ce film DATE a l instant `tUS` d un paquet :
// [FilmContext.GenerationsVivantes] passe par [GenerationsVivantes.A]. C est l acces des lecteurs de
// records delta bipedes (lot R2-bis) : un record lu dans un paquet n est la replication d un corps
// qu apres le record de creation de ce corps. Jamais nil.
func (c *FilmContext) GenerationsVivantesA(tUS uint64) *GenerationsVivantes {
	return c.GenerationsVivantes().A(tUS)
}

// viesConnuesDuFilm rend les vies (slot, generation) que les deux lectures de production designent.
// Une creation illisible n est pas une erreur ici : les images-cles restent, et un slot que rien ne
// designe retombe sur le repli nomme. Les vies des images-cles se relevent dans la phase des
// images-cles ([releveDesViesBipedes]).
func viesConnuesDuFilm(c *FilmContext) []types.LifeKey {
	var out []types.LifeKey
	if cre, _, err := c.CreationsDeBipede(); err == nil {
		for _, x := range cre {
			out = append(out, types.LifeKey{Slot: x.Slot, Gen: x.Generation})
		}
	}
	r := &releveDesViesBipedes{vies: out}
	distribuerLesImagesClesSeules(c, []Canal{r})
	return r.vies
}

// releveDesViesBipedes ajoute aux vies connues celles des records ti=35 de chaque image-cle.
type releveDesViesBipedes struct{ vies []types.LifeKey }

func (*releveDesViesBipedes) Interets() []Interet { return nil }

func (r *releveDesViesBipedes) ImageCle(p *lecture.Paquet, _ *MarcheDistribuee) {
	for _, rec := range p.Records {
		if int(rec.TI) == BipedTypeIndex {
			r.vies = append(r.vies, rec.Vie)
		}
	}
}

func (*releveDesViesBipedes) Clore(BilanDeMarche) {}
