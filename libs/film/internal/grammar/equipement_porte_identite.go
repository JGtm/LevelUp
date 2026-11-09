package grammar

// equipement_porte_identite.go — L IDENTITE D UN OBJET PORTE, LUE SUR L OBJET (canal de l equipement porte).
//
// Le record de creation d un objet ti=37 porte, dans le bloc `object-multiplayer-properties` de son etat par
// defaut, un mot de 32 bits inconditionnel : le GlobalID du tag `eqip` de l objet ([MPPWord32]). C est l identite
// de l objet, ecrite par le jeu a sa creation, quelle que soit la facon dont le joueur l a obtenu (ramasse au
// sol, equipement de reapparition, equipement de depart de match). Aucune jointure avec une annonce du porteur
// n est necessaire, et aucune n est devinee.
//
// LE MOT SE RELIT, il ne se capture pas a la marche : la marche des trames lit l etat par defaut d un record
// NEW sans observateur, et poser un crochet sur son observation commune ferait dependre l identite de ceux qui
// la partagent. Le canal reprend donc le record a son en-tete (`HeaderBit` + [woNewHeaderBits], le meme
// positionnement que la lecture de la position i0 d une creation) avec un lecteur a lui, charge du profil du
// film, et deroule le MEME deserialiseur ([consumeDefaultStateTI37]) : aucun second lecteur de l etat.

import (
	"github.com/JGtm/LevelUp/libs/film/internal/grammar/lecture"
)

// lecteurDIdentite relit l identite d une creation ti=37 : l observation qui retient le mot, et le contexte de
// lecture du film. Un seul par canal ; chaque lecture le remet a zero.
type lecteurDIdentite struct {
	ctx     ContexteDeLecture
	mot     uint64
	present bool
}

// nouveauLecteurDIdentite prepare la relecture des identites pour le film `fc`.
func nouveauLecteurDIdentite(fc *FilmContext) *lecteurDIdentite {
	l := &lecteurDIdentite{}
	obs := NouvelleObservation()
	obs.MppHook = func(f MPPField, v uint64, present bool) {
		if f == MPPWord32 {
			l.mot, l.present = v, present
		}
	}
	l.ctx = ContexteDeLecture{Profil: fc.ProfilDeBalayage(), Obs: obs}
	return l
}

// identite rend le GlobalID de l objet que le record NEW `r` du paquet `p` cree, et faux quand le deserialiseur
// ne le livre pas (bloc illisible).
func (l *lecteurDIdentite) identite(p *lecture.Paquet, r *FrameRecord) (uint32, bool) {
	l.mot, l.present = 0, false
	br := LecteurSur(p.Payload)
	br.PoserContexte(l.ctx)
	br.SetBitPos(r.HeaderBit + woNewHeaderBits)
	consumeDefaultStateTI37(br)
	return uint32(l.mot), l.present //nolint:gosec // mot de 32 bits (R(32))
}
