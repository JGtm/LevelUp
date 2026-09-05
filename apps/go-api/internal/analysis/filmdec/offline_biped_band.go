// offline_biped_band.go — L'ENTREE PAR BANDE DE SLOTS du decodeur de positions, et la
// plomberie de balayage qu'elle partage avec l'entree bipede.
//
// POURQUOI CE FICHIER EXISTE : DEPLACEMENT, PAS REECRITURE. offline_biped.go franchissait le
// seuil de 500 lignes (559) apres la resolution du chantier vehicules. Ce fichier reprend, SANS
// CHANGER UNE LIGNE DE LOGIQUE, tout ce qui n'est pas la GRAMMAIRE du record :
//
//	ScanFilmBipedPositionsForBand / ScanBipedPositionsForBand   l'entree par bande (D2 + noyau) ;
//	bipedScanChunks / bipedI0Layout / scanBipedChunks           la plomberie partagee ;
//	bipedSlotBand / fillSlotBand / filledSlotMap                le releve de la bande `ti=35`.
//
// offline_biped.go garde ce qui LIT LES BITS — l'en-tete, le masque, i0, la dequantification —
// et son en-tete de fichier reste LA reference de grammaire. La frontiere est celle-la : ici on
// dit QUOI balayer, la-bas on dit COMMENT un record se lit.
//
// La bande est le seul parametre d'archetype du decodeur : `ti=35` (bipede) et `ti=40`
// (vehicule) partagent la meme forme d'i0, et c'est ce fichier qui porte les trois reglages a
// connaitre pour un archetype autre que le bipede (cf. la doc de ScanBipedPositionsForBand).
package filmdec

import (
	"errors"
	"fmt"
	"sort"

	"levelup/go-api/internal/analysis/filmsource"
)

// ScanFilmBipedPositionsForBand est l'ENVELOPPE D2, HORS PRODUCTION, de
// [ScanBipedPositionsForBand] : elle charge le film puis lui passe la bande. La cuisson passe un
// film deja charge (une seule decompression).
func ScanFilmBipedPositionsForBand(dir string, band SlotBand, opt ScanFilmOptions) (
	[]BipedPosition, error) {
	if opt.WorldRange == nil && !opt.QuantaOnly {
		return nil, fmt.Errorf("%w (film %s) : renseigner ScanFilmOptions.WorldRange, ou QuantaOnly pour n'obtenir que les quanta", ErrUnknownMapBounds, dir)
	}
	// LES DEUX REFUS D'ENTREE PRECEDENT LE CHARGEMENT, et c'est l'ordre d'origine : un appelant
	// sans bornes ou sans bande doit recevoir SON erreur, pas une erreur de lecture de repertoire.
	if band.Count() == 0 {
		return nil, errors.New("bande de slots vide")
	}
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return nil, err
	}
	return ScanBipedPositionsForBand(film, band, opt)
}

// ScanBipedPositionsForBand décode les positions absolues des records dont le SLOT tombe
// dans band, avec la grammaire du bipède — celle de la forme `object-position-dynamic-
// precision`, porte de 5 bits.
//
// POURQUOI CETTE ENTRÉE EXISTE. Deux archétypes seulement portent cette forme d'i0 : le bipède
// (`ti=35`) et le VÉHICULE (`ti=40`). Le registre du film le dit, et la mesure du cadrage
// véhicules du 2026-08-31 le confirme sur pièces : sur la même bande de slots et le même film,
// cette grammaire rend 99,4 à 100 % de pas de trajectoire sous 35 m/s, celle des objets du
// monde (porte de 3 bits, `ScanWorldObjects`) 21,2 à 41,8 %. Le décodeur savait donc déjà
// lire un véhicule ; il lui manquait ce point d'entrée, `bipedSlotBand` filtrant en dur
// `ti=35`. Rien de la grammaire n'est paramétré ici : seule la bande l'est.
//
// LA BANDE d'un archétype d'objet du monde se relève par
// `ScanWorldObjectKeyframes(film, ti).Band`, puis `NewSlotBand`. TROIS RÉGLAGES sont à connaître
// pour un archétype autre que le bipède, tous déjà dans ScanFilmOptions — aucune ligne à
// recopier :
//
//   - RequireTag1 est à DÉSARMER : le tag de 2 bits est la génération du handle, et les objets
//     du monde en emploient les quatre valeurs (règle établie par matchWorldObjectRecord).
//   - MaxSpeedMPS et IsolationGapMS sont les deux filtres de post-traitement du bipède ; ils
//     s'appliquent tels quels, et zéro les désarme pour obtenir le flux brut.
//   - Layout à nil laisse DetectI0LayoutOf lire le découpage DANS LE FILM : les largeurs d'axe
//     sont une constante de CARTE, jamais d'archétype ; la cuisson, elle, impose celles du
//     catalogue par le FilmContext (cf. build_from_film.go).
//
// Ce qui ne bouge pas : l'en-tête, le masque, l'exigence d'un i0 absolu de la région attendue,
// et bipedMinMaskCnt — le masque nominal d'un véhicule porte cinq composants (i0/i1/i2/i3/i25),
// donc le minimum de deux est franchi sans réglage.
func ScanBipedPositionsForBand(film *filmsource.Film, band SlotBand, opt ScanFilmOptions) (
	[]BipedPosition, error) {
	chunks, err := bipedScanChunks(film, opt)
	if err != nil {
		return nil, err
	}
	if band.Count() == 0 {
		// Une bande vide n est pas un film illisible : l appelant a releve un archetype absent.
		return nil, errors.New("bande de slots vide")
	}
	lay, err := bipedI0Layout(film, opt)
	if err != nil {
		return nil, err
	}
	out, read := scanBipedChunks(film, chunks, band, lay, opt)
	if read == 0 {
		return nil, ErrNoReadableFilmChunk
	}
	out = DropIsolated(out, opt.IsolationGapMS)
	if opt.WorldRange == nil {
		return out, nil // sans coordonnées monde, un seuil en m/s n'a aucun sens
	}
	return DropTeleportsExcept(out, opt.MaxSpeedMPS, opt.TeleportExemptions), nil
}

// bipedScanChunks refuse les options qui interdisent toute émission de coordonnée, puis rend la
// liste des chunks à balayer. Les DEUX entrées y passent : c'est ce qui leur garantit la même
// erreur, dans le même ordre, sur des options incomplètes ou un film sans chunk.
func bipedScanChunks(film *filmsource.Film, opt ScanFilmOptions) ([]int, error) {
	if opt.WorldRange == nil && !opt.QuantaOnly {
		return nil, fmt.Errorf("%w : renseigner ScanFilmOptions.WorldRange, ou QuantaOnly pour n'obtenir que les quanta", ErrUnknownMapBounds)
	}
	chunks := opt.Chunks
	if len(chunks) == 0 {
		chunks = FilmChunkNumbers(film)
	}
	if len(chunks) == 0 {
		return nil, ErrNoFilmChunk
	}
	return chunks, nil
}

// bipedI0Layout rend le découpage d'i0 : celui que l'appelant force, sinon celui lu dans le film.
func bipedI0Layout(film *filmsource.Film, opt ScanFilmOptions) (I0Layout, error) {
	if opt.Layout != nil {
		return *opt.Layout, nil
	}
	lay, _, err := DetectI0LayoutOf(film)
	if err != nil {
		return I0Layout{}, fmt.Errorf("découpage i0 illisible : %w", err)
	}
	return lay, nil
}

// scanBipedChunks déroule le balayage sur les chunks demandés et rend les positions ainsi que
// le nombre de chunks effectivement LUS — un film partiel est licite, un film illisible non.
func scanBipedChunks(film *filmsource.Film, chunks []int, band SlotBand, lay I0Layout,
	opt ScanFilmOptions) ([]BipedPosition, int) {
	var out []BipedPosition
	read := 0
	for _, c := range chunks {
		data, pks, ok := FilmChunkAt(film, c)
		if !ok {
			continue
		}
		read++
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			for _, r := range ScanBipedRecords(pk.Payload(data), band, lay, opt) {
				r.Chunk, r.PacketIndex, r.TimestampUS = c, pk.Index, pk.TimestampUS
				out = append(out, r)
			}
		}
	}
	return out, read
}

// bipedSlotBand construit l'ensemble des slots biped plausibles : union des ti=35 des
// keyframes de tous les chunks balayés (+ le suivant, car un biped créé en cours de chunk
// n'apparaît que dans le keyframe d'après), trous comblés entre min et max — les slots
// biped sont alloués dans une bande contiguë, et un biped créé PUIS détruit à l'intérieur
// d'un chunk n'apparaît dans aucun keyframe.
func bipedSlotBand(film *filmsource.Film, chunks []int) SlotBand {
	seen := map[uint32]bool{}
	scan := append(append([]int{}, chunks...), chunks[len(chunks)-1]+1)
	for _, c := range scan {
		data, pks, ok := FilmChunkAt(film, c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeKeyframe {
				continue
			}
			for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
				if r.TI == BipedTypeIndex && r.Slot >= 0 {
					seen[uint32(r.Slot)] = true
				}
			}
			break
		}
	}
	return fillSlotBand(seen)
}

// fillSlotBand comble les trous entre le min et le max de l'ensemble (bande contiguë) et
// rend la bande DENSE consultable par bit candidat.
func fillSlotBand(s map[uint32]bool) SlotBand { return NewSlotBand(filledSlotMap(s)) }

// filledSlotMap est le comblement lui-même, rendu sous forme d'ensemble : `slotBandExcluding`
// a besoin de retirer des slots AVANT de figer la bande dense.
func filledSlotMap(s map[uint32]bool) map[uint32]bool {
	if len(s) == 0 {
		return s
	}
	keys := make([]uint32, 0, len(s))
	for k := range s {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
	out := make(map[uint32]bool, int(keys[len(keys)-1]-keys[0])+1)
	for k := keys[0]; k <= keys[len(keys)-1]; k++ {
		out[k] = true
	}
	return out
}
