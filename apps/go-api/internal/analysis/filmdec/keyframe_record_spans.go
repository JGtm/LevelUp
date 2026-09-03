package filmdec

// keyframe_record_spans.go — L'EMPRISE EN BITS des records de la table d'image-clé.
//
// CE QUE CE FICHIER AJOUTE, ET POURQUOI. `WalkKeyframeWorld` rend les ancres de record
// (slot, archétype, position). Il ne rend PAS leur EMPRISE, alors que c'est elle qui a servi
// à tout ce qui lit une image-clé sans en parser le corps : les armes portées
// (`familiesByRecord`) recalculent les bornes pour leur compte, et le lot V5
// (« où est l'état d'occupation ? ») a mesuré que l'emprise EST elle-même un signal.
//
// CE QUE LA MESURE V5 A ÉTABLI (rapport `.ai/V7.5/film_re/V5_ETAT_OCCUPATION_2026-09-02.md`) :
// le record d'image-clé d'un VÉHICULE (`ti=40`) est plus long quand le véhicule est occupé.
// Test apparié véhicule-contre-lui-même, restreint aux records dont le voisin de slot suivant
// n'a pas été sauté par le balayeur : **16 paires sur 17 (94,1 %)**, écart moyen **+151 bits**,
// contre **3/12 (25,0 %)** au témoin par décalage de 37 s. L'écart récurrent est de l'ordre de
// **+89 bits**, la même valeur sur trois films indépendants.
//
// CE QUE CE DÉCODEUR NE DIT PAS : ni QUI occupe (l'identité de l'occupant n'a été trouvée
// nulle part — cf. le rapport), ni le SIÈGE. Il rend une GRANDEUR BRUTE, pas une conclusion :
// c'est à l'appelant de la comparer, et le champ `SlotGap` existe pour qu'il puisse écarter
// les emprises polluées (cf. ci-dessous).
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requête.

import "sort"

// KeyframeRecordSpan est l'emprise d'UN record de la table d'image-clé.
type KeyframeRecordSpan struct {
	// Slot, TI, Gen identifient l'entité et son archétype (mêmes valeurs que KeyframeRec).
	Slot, TI, Gen int
	// BitStart est le premier bit de l'en-tête ; BitEnd le premier bit du record SUIVANT
	// TROUVÉ par le balayeur (donc la fin exclusive de l'emprise).
	BitStart, BitEnd int
	// LengthBits = BitEnd - BitStart.
	LengthBits int
	// SlotGap est l'écart de SLOT jusqu'au record suivant TROUVÉ, ou 0 pour le dernier.
	//
	// C'EST LE GARDE-FOU DE L'EMPRISE, et il n'est pas décoratif. `WalkKeyframeWorld` ne
	// retient une ancre que si les 26 bits de `field` de l'en-tête sont nuls
	// (cf. keyframe_record_walk.go) : un record dont ce champ n'est pas nul est SAUTÉ, et
	// l'emprise rendue couvre alors DEUX records ou plus. Une comparaison de longueurs qui
	// ne filtre pas sur `SlotGap == 1` mesure en partie le nombre de records sautés — la
	// mesure V5 le montre : l'écart apparent tombe de +1 348 à +151 bits une fois le filtre
	// posé, et c'est la valeur filtrée qui est la bonne.
	SlotGap int
	// TimestampUS, Chunk, PacketIndex localisent l'image-clé porteuse.
	TimestampUS        uint64
	Chunk, PacketIndex int
}

// KeyframeRecordSpans rend l'emprise de chaque record d'UN payload d'image-clé, dans l'ordre
// des positions croissantes. PUR (aucune I/O).
func KeyframeRecordSpans(pay []byte) []KeyframeRecordSpan {
	recs := WalkKeyframeWorld(pay)
	if len(recs) == 0 {
		return nil
	}
	sort.Slice(recs, func(i, j int) bool { return recs[i].Bit < recs[j].Bit })
	total := len(pay) * 8
	out := make([]KeyframeRecordSpan, 0, len(recs))
	for i, r := range recs {
		end, gap := total, 0
		if i+1 < len(recs) {
			end, gap = recs[i+1].Bit, recs[i+1].Slot-r.Slot
		}
		out = append(out, KeyframeRecordSpan{
			Slot: r.Slot, TI: r.TI, Gen: r.Gen,
			BitStart: r.Bit, BitEnd: end, LengthBits: end - r.Bit, SlotGap: gap,
		})
	}
	return out
}

// ScanFilmKeyframeRecordSpans rend l'emprise de tous les records de toutes les images-clés du
// film de `dir`, horodatées. Les images-clés illisibles sont ignorées en silence — le compte
// rendu de lisibilité appartient aux balayeurs de position, qui le font déjà.
//
// HORS LIGNE — jamais depuis un chemin de requête.
func ScanFilmKeyframeRecordSpans(dir string) []KeyframeRecordSpan {
	var out []KeyframeRecordSpan
	for c := 1; c <= CountFilmChunks(dir); c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range WalkPackets(data) {
			if p.Type != PacketTypeKeyframe {
				continue
			}
			for _, s := range KeyframeRecordSpans(p.Payload(data)) {
				s.TimestampUS, s.Chunk, s.PacketIndex = p.TimestampUS, c, p.Index
				out = append(out, s)
			}
		}
	}
	return out
}
