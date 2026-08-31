package replay

// visee_etiquettes_keyframe_test.go — LOT G3 : LA CORRELATION SUR LES IMAGES-CLES.
//
// POURQUOI C'EST LA MOITIE LA PLUS PROMETTEUSE, ET POURQUOI ELLE N'AVAIT JAMAIS ETE FAITE.
//
// Un paquet delta ne transmet que ce qui CHANGE : un etat stable peut n'y apparaitre qu'a la
// bascule, puis jamais plus — c'est exactement ce qui rendrait un negatif delta peu concluant
// sur un champ d'etat. Une IMAGE-CLE, elle, porte l'etat COMPLET de chaque entite : si l'etat de
// lunette est un etat replique, il y est, a chaque image-cle, sans exception. C'est aussi le
// geste que l'objection architecturale de l'utilisateur suggere — un decodeur robuste se
// resynchronise sur un etat complet periodique plutot que d'accumuler des bascules.
//
// DEUX CHEMINS EXISTENT DANS LE DEPOT, ET LES DEUX SONT MESURES ICI PLUTOT QUE L'UN CHOISI.
//
//  1. LE MARCHEUR DETERMINISTE (`filmdec.WalkKeyframeRecords`) : en-tete de 64 bits, puis corps
//     par le lecteur de record NEW de production, puis enchainement sur la position atteinte.
//     Il ne balaie rien — donc il est aussi un test de la grammaire, et son arret dit ou elle
//     lache. Son rendement est publie meme quand il est nul : un chemin qui echoue est une
//     piece, pas une gene (meme regle qu'au lot F pour le chemin sequentiel).
//  2. LE BALAYEUR DE PRODUCTION (`filmdec.WalkKeyframeWorld`), celui dont `WorldFromKeyframe`
//     se sert pour binder le monde a chaque image-cle. Il ANCRE les records sur la grammaire
//     d'en-tete au lieu de les enchainer, puis `filmdec.TraverseKeyframeBipedAt` — la sonde de
//     production du harnais loadout — rejoue le corps du bipede et rend ses composants.
//
// RESERVE DU CHEMIN 2, DITE D'AVANCE : le balayeur applique un FILTRE FORT (les 26 bits de
// `field` doivent etre nuls), donc il saute les records dont ce champ ne l'est pas. La
// couverture est partielle et le nombre de records ancres est publie a cote du nombre attendu.
//
// UNE VERIFICATION CROISEE EST FAITE ET PUBLIEE : la fin de corps rendue par la traversee doit
// tomber avant l'ancre du record suivant. Les records qui la depassent sont ECARTES et comptes —
// un debordement signifierait que les offsets lus ne designent pas ce qu'on croit.
//
// LE SOUS-DIMENSIONNEMENT EST UN RESULTAT, PAS UN ECHEC. Les images-cles sont ESPACEES : il y en
// a quelques dizaines par film la ou il y a des dizaines de milliers de paquets delta. Le mandat
// l'exige explicitement — on compte d'abord combien de records d'image-cle tombent dans une
// periode ETIQUETEE, et si une classe n'atteint pas le seuil de recevabilite, on PUBLIE le
// sous-dimensionnement au lieu de fabriquer un faux negatif.

import (
	"levelup/go-api/internal/analysis/filmdec"
)

// vgKFStat porte la couverture de la marche d'image-cle, publiee AVANT tout score.
type vgKFStat struct {
	// paquets d'image-cle deroules ; records ancres par le balayeur ; records d'archetype bipede.
	paquets, records, bipeds int
	// detRecords / detBipeds : rendement du marcheur DETERMINISTE, publie comme piece.
	detRecords, detBipeds int
	// detArrets : cause d'arret du marcheur deterministe -> nombre de paquets.
	detArrets map[string]int
	// retenus : records bipedes d'un slot etiquete, traverses sans debordement.
	retenus int
	// deborde : traversees dont la fin depasse l'ancre du record suivant (ECARTEES).
	deborde int
	// desync : records dont un composant present n'est pas porte (marche partielle).
	desync int
	// comps : composants effectivement consommes et mesures.
	comps int
	// recMin / recMax : records ancres par paquet, minimum et maximum observes.
	recMin, recMax int
}

// vgCollecteKF deroule les images-cles du film et rend les records bipedes des slots etiquetes,
// decoupes en composants — meme forme que la collecte delta, donc meme suite de mesure.
//
// L'appelant detient `LockProcessDecode` : la marche joue les deserialiseurs de production, dont
// les bascules de grammaire sont globales au process.
func vgCollecteKF(dir string, reg *filmdec.Registry, cibles map[uint32]bool) ([]vfRecord, vgKFStat) {
	st := vgKFStat{detArrets: map[string]int{}, recMin: -1}
	var out []vfRecord
	for c := 1; c <= filmdec.CountFilmChunks(dir); c++ {
		data, err := filmdec.ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, p := range filmdec.WalkPackets(data) {
			if p.Type != filmdec.PacketTypeKeyframe {
				continue
			}
			st.paquets++
			pay := p.Payload(data)
			vgMesureDeterministe(&st, pay, reg)
			out = append(out, vgVerseKeyframe(&st, pay, reg, cibles,
				int64(p.TimestampUS/1000))...)
		}
	}
	return out, st
}

// vgMesureDeterministe republie le rendement du marcheur qui n'enchaine PAS par balayage. Il ne
// sert a rien choisir : il documente si la grammaire tient de bout en bout sur ce film.
func vgMesureDeterministe(st *vgKFStat, pay []byte, reg *filmdec.Registry) {
	recs, stop := filmdec.WalkKeyframeRecords(pay, reg)
	st.detArrets[stop.String()]++
	st.detRecords += len(recs)
	for _, r := range recs {
		if r.TI == filmdec.BipedTypeIndex {
			st.detBipeds++
		}
	}
}

// vgVerseKeyframe ancre les records d'UN paquet d'image-cle et rend ses bipedes mesurables.
func vgVerseKeyframe(st *vgKFStat, pay []byte, reg *filmdec.Registry, cibles map[uint32]bool,
	tMS int64,
) []vfRecord {
	ancres := filmdec.WalkKeyframeWorld(pay)
	st.records += len(ancres)
	if st.recMin < 0 || len(ancres) < st.recMin {
		st.recMin = len(ancres)
	}
	if len(ancres) > st.recMax {
		st.recMax = len(ancres)
	}
	var out []vfRecord
	for i, r := range ancres {
		if r.TI != filmdec.BipedTypeIndex {
			continue
		}
		st.bipeds++
		if !cibles[uint32(r.Slot)] {
			continue
		}
		borne := len(pay) * 8
		if i+1 < len(ancres) {
			borne = ancres[i+1].Bit
		}
		if rec, ok := vgTraverseKF(st, pay, reg, r, borne, tMS); ok {
			out = append(out, rec)
		}
	}
	return out
}

// vgTraverseKF rejoue le corps d'UN bipede d'image-cle et rend son record decoupe en composants.
func vgTraverseKF(st *vgKFStat, pay []byte, reg *filmdec.Registry, r filmdec.KeyframeRec,
	borne int, tMS int64,
) (vfRecord, bool) {
	tr, end := filmdec.TraverseKeyframeBipedAt(pay, r.Bit+64, reg, uint32(r.TI))
	if end > borne {
		st.deborde++
		return vfRecord{}, false
	}
	if tr.DesyncAt >= 0 {
		st.desync++
	}
	comps := vgKFComposants(pay, tr, end)
	if len(comps) == 0 {
		return vfRecord{}, false
	}
	st.retenus++
	st.comps += len(comps)
	return vfRecord{tMS: tMS, slot: uint32(r.Slot), comps: comps}, true
}

// vgKFComposants decoupe le corps d'un record d'image-cle en composants, avec pour chacun sa
// largeur MESUREE (frontiere a frontiere) et le prefixe de sa charge utile.
//
// LA LARGEUR VIENT DES FRONTIERES, PAS D'UNE TABLE : le debut du composant suivant borne le
// precedent, et la fin de corps borne le dernier. Un composant non porte arrete le decoupage —
// au-dela, la position du curseur ne serait plus digne de confiance, exactement comme dans la
// marche delta du lot F.
func vgKFComposants(pay []byte, tr filmdec.EntityTrace, end int) []vfComp {
	out := make([]vfComp, 0, len(tr.Comps))
	for i, c := range tr.Comps {
		if !c.Ported {
			break
		}
		fin := end
		if i+1 < len(tr.Comps) {
			fin = tr.Comps[i+1].StartBit
		}
		larg := fin - c.StartBit
		if larg <= 0 || fin > len(pay)*8 {
			break
		}
		out = append(out, vfComp{idx: c.Index, nom: c.Name, larg: larg,
			bits: vfLitBits(pay, c.StartBit, larg)})
	}
	return out
}

// vgClasses compte les records (delta ou image-cle) par classe d'etiquette a decalage nul. C'est
// le chiffre que le mandat exige AVANT toute correlation : si la classe « zoome » est vide ou
// minuscule, la suite ne mesure rien et il faut le dire.
func vgClasses(recs []vfRecord, g *vgGrille) (un, zero, exclu int) {
	for _, r := range recs {
		switch g.classe(r.slot, r.tMS, 0) {
		case 1:
			un++
		case 0:
			zero++
		default:
			exclu++
		}
	}
	return un, zero, exclu
}
