//go:build research

package main

// gb1.go — LE MODE `gb1` : LA MESURE PREALABLE DU CONSTAT GB-1 (lot J5.0 du plan de suite de
// l audit du decodeur). GB-1 : les positions et les canaux delta du bipede n etaient lus que pour la
// generation 1 du handle d un slot (`ScanFilmOptions.RequireTag1`, supprime au lot J5.2 au profit du
// filtre des generations VIVANTES, `grammar.GenerationsVivantes`). L instrument mesure le filtre de
// PRODUCTION en vigueur : avant J5.2 la generation 1, depuis J5.2 les generations vivantes.
//
// # LES DEFINITIONS, EXACTES
//
//	VIE             un couple (slot, generation) du handle d un bipede (`ti=35`), CONNU par au
//	                moins une des deux lectures de production qui le designent : un record de
//	                creation (`grammar.ScanBipedCreations`, generation = les 2 bits du handle de
//	                l en-tete NEW) ou un record d image-cle (`MarcheDImageCle().Records`, champ
//	                `Gen`), sur TOUS les paquets d image-cle du film.
//	POSITION PROD   un echantillon rendu par `grammar.ScanBipedPositions` sous
//	                `DefaultScanFilmOptions` (filtre de generation du film, saturation ecartee,
//	                isolement 15 s), en quanta seuls (aucune borne de carte : le filtre de vitesse
//	                n est PAS joue). Il est rattache a SA vie (slot, tag relu du handle par
//	                `grammar.LireHandleDelta`, cf. `mesurerProd`).
//	EN-TETE BRUT    un record que le MEME marcheur ancre filtre LEVE (`ToutesLesGenerations`,
//	                saturation ecartee, aucun isolement) : en-tete bipede valide (prefixe, slot
//	                dans la bande, bits nuls, masque croissant depuis 0, i0 absolu de la region).
//	                Son tag est lu par l observateur (cf. `balayerBrut`).
//	POSITION        un en-tete brut dont (slot, tag) est une vie connue ET qui survit a
//	VIVANTE         `DropIsolated` (15 s) joue PAR (slot, tag) : l estimation de ce qu un filtre
//	                « generation vivante » (DT-8) publierait.
//	ORPHELIN        un en-tete brut dont (slot, tag) n est AUCUNE vie connue : FAUX POSITIF
//	                POTENTIEL d un filtre « generation vivante » (bruit d ancrage bit a bit, ou vie
//	                qu aucune des deux lectures n a vue). « Garde » s il survit a l isolement.
//	ORPHELIN PROD   une position prod dont la vie (slot, tag) n est pas connue.
//
// # LES DUREES
//
// `durationMs` publie est reconstitue, sans cuisson, par la formule de `replay` (`build.go`
// `ouvrir` : `frameSpan` x pas de 100 ms) sur les positions prod. Le film ne porte aucune duree
// de match explicite ; deux durees en sont lues : celle des paquets DELTA (premier au dernier
// horodatage des paquets de replication) et celle de TOUS les paquets des chunks de donnees (le
// chunk highlight de fin de match compris, quand le film le porte). La duree VIVANTE est la
// formule de `replay` sur les positions vivantes. La troncature de fin est l ecart entre le
// dernier paquet delta et le dernier echantillon retenu (prod, ou vivant).

import (
	"errors"
	"fmt"
	"time"

	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar"
	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
)

// nombreDeTags : le tag du handle tient sur deux bits.
const nombreDeTags = 4

// pasDeFrameUS : le pas de la grille du rejeu (`replay.DefaultFrameIntervalMS` = 100 ms), en
// microsecondes. Seconde copie de la valeur, pour reconstituer `durationMs` sans cuisson.
const pasDeFrameUS = 100 * 1000

// cleDeVie : une vie, (slot, generation du handle).
type cleDeVie struct{ slot, gen uint32 }

// vieGB1 : ce que le film dit d UNE vie.
type vieGB1 struct {
	creations          int
	premiereCreationUS uint64
	imagesCles         int
	horsBande          bool
	prod, brut, vives  int
	// premierUS / dernierUS : bornes des positions vivantes de la vie (0 : aucune).
	premierUS, dernierUS uint64
}

// orphelinGB1 : les en-tetes bruts d un (slot, tag) qui n est aucune vie connue.
type orphelinGB1 struct {
	entetes, gardes      int
	premierUS, dernierUS uint64
}

// bornes : premier et dernier horodatage d un ensemble d echantillons (0, 0 : vide).
type bornes struct{ debut, fin uint64 }

// etendre ajoute un horodatage aux bornes.
func (b *bornes) etendre(ts uint64) {
	if b.debut == 0 || ts < b.debut {
		b.debut = ts
	}
	b.fin = max(b.fin, ts)
}

// Le statut d une mesure GB-1 : mesuree, ou sans bipede aux images-cles (rien a mesurer).
const (
	statutMesure     = "mesure"
	statutSansBipede = "aucun-slot-bipede"
)

// mesureGB1 est ce qu UN film a rendu en mode `gb1`.
type mesureGB1 struct {
	id, build, statut string
	pic               uint64
	duree             time.Duration

	creations, creationsRepetees int
	creationsParGen              [nombreDeTags]int
	slotsCrees, slotsCreesGen2   int

	vies      map[cleDeVie]*vieGB1
	orphelins map[cleDeVie]*orphelinGB1

	positionsProd, orphelinsProd  int
	entetesParTag, vivesParTag    [nombreDeTags]int
	orphelinsParTag, gardesParTag [nombreDeTags]int

	film, delta, prod, vivant bornes
}

// mesurerGB1 fait la mesure d un film deja ouvert. Toutes les lectures sont celles de production
// (creations, images-cles, marcheur des positions) ; seul le tag de l en-tete brut est relu, par
// l observateur, a la position que le marcheur vient d ancrer.
func mesurerGB1(fc *grammar.FilmContext) (mesureGB1, error) {
	m := mesureGB1{statut: statutMesure, vies: map[cleDeVie]*vieGB1{}, orphelins: map[cleDeVie]*orphelinGB1{}}
	band := fc.BipedSlots()
	m.parcourirPaquets(fc, band)
	if band.Count() == 0 {
		// Aucun bipede aux images-cles (bobine anterieure a la premiere apparition, film partiel) : il
		// n y a rien a mesurer, et ce n est pas un echec du film — la ligne le dit.
		m.statut = statutSansBipede
		return m, nil
	}
	lay, err := fc.I0Layout()
	if err != nil {
		return m, fmt.Errorf("decoupage i0 illisible : %w", err)
	}
	cre, _, err := fc.CreationsDeBipede() // memorisees : le filtre de production les relit
	if err != nil {
		return m, fmt.Errorf("creations : %w", err)
	}
	m.compterCreations(cre)
	if err := m.mesurerProd(fc, lay); err != nil {
		return m, err
	}
	brut, tags, err := balayerBrut(fc, lay)
	if err != nil {
		return m, err
	}
	m.rattacherBrut(brut, tags)
	return m, nil
}

// compterCreations range les records de creation par vie et par generation.
func (m *mesureGB1) compterCreations(cre []grammar.BipedCreation) {
	slots, slotsGen2 := map[uint32]bool{}, map[uint32]bool{}
	for _, c := range cre {
		m.creations++
		m.creationsParGen[c.Generation&(nombreDeTags-1)]++
		slots[c.Slot] = true
		if c.Generation >= 2 {
			slotsGen2[c.Slot] = true
		}
		v := m.vie(cleDeVie{c.Slot, c.Generation})
		if v.creations > 0 {
			m.creationsRepetees++ // meme (slot, generation) cree deux fois : generation rebouclee ?
		}
		if v.creations == 0 || c.TimestampUS < v.premiereCreationUS {
			v.premiereCreationUS = c.TimestampUS
		}
		v.creations++
	}
	m.slotsCrees, m.slotsCreesGen2 = len(slots), len(slotsGen2)
}

// parcourirPaquets releve, en un passage sur les paquets de tous les chunks de donnees, les
// bornes d horodatage du film et les vies bipedes des images-cles.
func (m *mesureGB1) parcourirPaquets(fc *grammar.FilmContext, band grammar.SlotBand) {
	marche := fc.MarcheDImageCle()
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.TimestampUS > 0 {
				m.film.etendre(pk.TimestampUS)
			}
			switch pk.Type {
			case grammar.PacketTypeDelta:
				if pk.TimestampUS > 0 {
					m.delta.etendre(pk.TimestampUS)
				}
			case grammar.PacketTypeKeyframe:
				m.releverImageCle(marche.Records(pk.Payload(data)), band)
			}
		}
	}
}

// releverImageCle range les records bipedes d UNE image-cle par vie.
func (m *mesureGB1) releverImageCle(recs []grammar.KeyframeRec, band grammar.SlotBand) {
	for _, r := range recs {
		if r.TI != grammar.BipedTypeIndex || r.Slot < 0 || r.Gen < 0 {
			continue
		}
		v := m.vie(cleDeVie{uint32(r.Slot), uint32(r.Gen)}) //nolint:gosec // bornes verifiees
		v.imagesCles++
		v.horsBande = !band.Has(uint32(r.Slot)) //nolint:gosec // borne verifiee
	}
}

// vie rend la vie de la cle, creee au besoin.
func (m *mesureGB1) vie(k cleDeVie) *vieGB1 {
	v := m.vies[k]
	if v == nil {
		v = &vieGB1{}
		m.vies[k] = v
	}
	return v
}

// mesurerProd joue le balayage des positions sous le filtre de PRODUCTION (`DefaultScanFilmOptions`,
// quanta seuls) et rattache chaque position a SA vie (slot, tag relu du handle). Le crochet qui relit
// le tag ne tire qu avant les filtres de post-traitement : le balayage se fait donc isolement desarme,
// puis l isolement de production est rejoue (`DropIsolated`, le meme appel que la production, par
// slot) — sa sortie est une sous-suite ORDONNEE de son entree, ce qui rend a chaque position gardee le
// tag de son record. Le balayage est libere au retour, avant le brut.
func (m *mesureGB1) mesurerProd(fc *grammar.FilmContext, lay profile.I0Layout) error {
	opt := grammar.DefaultScanFilmOptions()
	opt.QuantaOnly, opt.Layout = true, &lay
	isolement := opt.IsolationGapMS
	opt.IsolationGapMS = 0
	lus, tags, err := balayerAvecTags(fc, lay, opt)
	if err != nil {
		return fmt.Errorf("positions prod : %w", err)
	}
	prod := grammar.DropIsolated(lus, isolement)
	m.positionsProd = len(prod)
	j := 0
	for i, p := range lus {
		if j >= len(prod) || !memeEchantillon(p, prod[j]) {
			continue
		}
		j++
		m.prod.etendre(p.TimestampUS)
		if v := m.vies[cleDeVie{p.Slot, tags[i]}]; v != nil {
			v.prod++
		} else {
			m.orphelinsProd++
		}
	}
	if j != len(prod) {
		return fmt.Errorf("%w : %d positions gardees par l isolement, %d rattachees", errEnTeteIntrouvable, len(prod), j)
	}
	return nil
}

// memeEchantillon : deux positions sont le meme echantillon du balayage (meme paquet, meme slot,
// meme instant, memes quanta).
func memeEchantillon(a, b grammar.BipedPosition) bool {
	return a.Chunk == b.Chunk && a.PacketIndex == b.PacketIndex && a.Slot == b.Slot &&
		a.TimestampUS == b.TimestampUS && a.Q == b.Q
}

// rattacherBrut range les en-tetes bruts par tag, les rattache a leur vie ou aux orphelins, puis
// joue l isolement par (slot, tag) pour obtenir les positions vivantes et les orphelins gardes.
func (m *mesureGB1) rattacherBrut(brut []grammar.BipedPosition, tags []uint32) {
	var parTag [nombreDeTags][]grammar.BipedPosition
	for i, p := range brut {
		t := tags[i]
		parTag[t] = append(parTag[t], p)
		m.entetesParTag[t]++
		k := cleDeVie{p.Slot, t}
		if v := m.vies[k]; v != nil {
			v.brut++
			continue
		}
		m.orphelinsParTag[t]++
		o := m.orphelins[k]
		if o == nil {
			o = &orphelinGB1{}
			m.orphelins[k] = o
		}
		o.entetes++
	}
	for t := range parTag {
		for _, p := range grammar.DropIsolated(parTag[t], grammar.DefaultIsolationGapMS) {
			m.garder(p, uint32(t)) //nolint:gosec // t < nombreDeTags
		}
		parTag[t] = nil
	}
}

// garder compte UN en-tete brut qui a survecu a l isolement.
func (m *mesureGB1) garder(p grammar.BipedPosition, t uint32) {
	k := cleDeVie{p.Slot, t}
	if v := m.vies[k]; v != nil {
		v.vives++
		m.vivesParTag[t]++
		m.vivant.etendre(p.TimestampUS)
		if v.premierUS == 0 || p.TimestampUS < v.premierUS {
			v.premierUS = p.TimestampUS
		}
		v.dernierUS = max(v.dernierUS, p.TimestampUS)
		return
	}
	m.gardesParTag[t]++
	o := m.orphelins[k]
	o.gardes++
	if o.premierUS == 0 || p.TimestampUS < o.premierUS {
		o.premierUS = p.TimestampUS
	}
	o.dernierUS = max(o.dernierUS, p.TimestampUS)
}

// Reconstitution du DEBUT de l en-tete d un record bipede a partir de ce que l observateur recoit
// (la fin d i0 et la liste des index du masque). Ce sont les largeurs de la grammaire
// (`grammar/offline_biped.go`, en-tete de fichier : `[1 prefixe][13 slot][2 tag][2 nuls][3
// nombre d index]` puis 6 bits par index). Le HANDLE (slot, tag) se relit ensuite par
// `grammar.LireHandleDelta`, le lecteur unique du handle (lot J5.1) : l instrument ne recopie plus
// ses largeurs. La reconstitution est VERIFIEE a chaque record : le slot relu doit etre celui que
// le marcheur a publie, sinon le film echoue (`balayerBrut`).
const (
	enteteBipedeBits     = 21
	indexDeComposantBits = 6
)

// errEnTeteIntrouvable : la reconstitution de l en-tete ne retombe pas sur le record publie.
var errEnTeteIntrouvable = errors.New("en-tete reconstitue divergent du record publie par le marcheur")

// balayerBrut joue le marcheur des positions filtre LEVE (`grammar.ToutesLesGenerations`, saturation
// ecartee, aucun isolement) et rend, pour chaque record, son tag.
func balayerBrut(fc *grammar.FilmContext, lay profile.I0Layout) ([]grammar.BipedPosition, []uint32, error) {
	opt := grammar.ScanFilmOptions{QuantaOnly: true, Layout: &lay, DropSaturated: true,
		Generations: grammar.ToutesLesGenerations()}
	brut, tags, err := balayerAvecTags(fc, lay, opt)
	if err != nil {
		return nil, nil, fmt.Errorf("positions brutes : %w", err)
	}
	return brut, tags, nil
}

// balayerAvecTags joue le marcheur des positions sous `opt` (qui ne doit porter AUCUN filtre de
// post-traitement : ni isolement, ni bornes de carte) et rend, pour chaque record, son tag.
//
// Le tag n est publie par aucune sortie du marcheur ; il est relu par `RecordMaskHook`, le seul
// crochet de l observateur qui recoive le payload et la position d un record EMIS. Le crochet ne
// tire que sous `CaptureDirs`, AVANT les filtres de post-traitement : c est pourquoi l isolement
// est desarme par l appelant (et rejoue ensuite) et pourquoi aucune borne de carte n est donnee (le
// filtre de vitesse n est pas joue) — la sortie du marcheur et les appels du crochet sont alors en
// bijection, dans le meme ordre, ce qui est verifie.
func balayerAvecTags(fc *grammar.FilmContext, lay profile.I0Layout, opt grammar.ScanFilmOptions) ([]grammar.BipedPosition, []uint32, error) {
	obs := fc.Observation()
	precedent := obs.RecordMaskHook
	defer func() { obs.RecordMaskHook = precedent }()
	i0Bits := lay.TotalBits()
	var tags, slots []uint32
	obs.RecordMaskHook = func(idx []int, pay []byte, apresI0 int) {
		p := apresI0 - i0Bits - enteteBipedeBits - indexDeComposantBits*len(idx)
		if p < 0 {
			tags, slots = append(tags, 0), append(slots, ^uint32(0))
			return
		}
		h := grammar.LireHandleDelta(pay, p)
		tags, slots = append(tags, h.Gen), append(slots, h.Slot)
	}
	opt.CaptureDirs = true
	brut, err := grammar.ScanBipedPositions(fc, opt)
	if err != nil {
		return nil, nil, err
	}
	if len(brut) != len(tags) {
		return nil, nil, fmt.Errorf("%w : %d records, %d appels du crochet", errEnTeteIntrouvable, len(brut), len(tags))
	}
	for i, p := range brut {
		if slots[i] != p.Slot {
			return nil, nil, fmt.Errorf("%w : record %d, slot %d relu %d", errEnTeteIntrouvable, i, p.Slot, slots[i])
		}
	}
	return brut, tags, nil
}

// dureeMS reconstitue `durationMs` a la maniere de `replay` : nombre de frames de 100 ms couvertes
// par les bornes, fois le pas. 0 sans echantillon.
func (b bornes) dureeMS() int64 {
	if b.fin == 0 {
		return 0
	}
	return int64((b.fin-b.debut)/pasDeFrameUS+1) * (pasDeFrameUS / 1000) //nolint:gosec // duree d un film
}

// ecartMS rend `a - b` en millisecondes (signe).
func ecartMS(a, b uint64) int64 { return (int64(a) - int64(b)) / 1000 } //nolint:gosec // horloge du film

// viesSansPositionProd : les vies connues qu aucune position prod ne couvre.
func (m mesureGB1) viesSansPositionProd() int {
	n := 0
	for _, v := range m.vies {
		if v.prod == 0 {
			n++
		}
	}
	return n
}

// viesSansPositionVivante : les vies connues qu aucune position vivante ne couvre.
func (m mesureGB1) viesSansPositionVivante() int {
	n := 0
	for _, v := range m.vies {
		if v.vives == 0 {
			n++
		}
	}
	return n
}

// totalOrphelins : les en-tetes bruts orphelins, tous tags.
func (m mesureGB1) totalOrphelins() int {
	n := 0
	for _, c := range m.orphelinsParTag {
		n += c
	}
	return n
}
