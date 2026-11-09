package grammar

// keyframe_liaison_instruments_test.go — LA FORME INSTRUMENT DE LA LIAISON DES IMAGES-CLES.
//
// La marche des trames lit ce que les images-cles declarent dans la phase des images-cles et le
// pose chunk par chunk ([liaisonDesImagesCles], [lierLesImagesClesDuChunk]). Les instruments qui
// rejouent la marche hors d elle lisent les images-cles d un chunk et les posent aussitot : c est
// cette forme-ci, qui partage la lecture ([declarerLImageCle]) et la pose de la production.

// lierLeChunkAuMonde pose sur le monde ce que les images-cles du chunk `pks` declarent, lues par la
// marche d ancres `marche`.
func lierLeChunkAuMonde(w *World, marche MarcheDImageCle, data []byte, pks []FilmPacket,
	obs *Observation) LiaisonDUnChunk {
	var decls []declarationDImageCle
	for _, pk := range pks {
		if pk.Type != PacketTypeKeyframe {
			continue
		}
		pay := pk.Payload(data)
		decls = append(decls, declarerLImageCle(pay, marche.Marcher(pay)))
	}
	return lierLesImagesClesDuChunk(w, decls, obs)
}

// AjouterChunk verse dans la table ce que les images-cles d un chunk declarent, lues par la marche
// SANS preuve des instruments ([WalkKeyframeWorld]). La production construit la table dans la phase
// des images-cles ([ConstruireTableAnticipee]).
func (t *TableAnticipee) AjouterChunk(num int, data []byte, pks []FilmPacket) {
	for _, pk := range pks {
		if pk.Type != PacketTypeKeyframe {
			continue
		}
		for _, r := range WalkKeyframeWorld(pk.Payload(data)) {
			if r.Slot < 0 || r.TI < 0 {
				continue
			}
			t.declarer(num, uint32(r.Slot), uint8(r.Gen&3), uint32(r.TI)) //nolint:gosec // bornes par le walker
		}
	}
}
