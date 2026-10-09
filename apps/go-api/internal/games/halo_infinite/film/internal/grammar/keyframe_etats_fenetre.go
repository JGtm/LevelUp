package grammar

// keyframe_etats_fenetre.go — LES FENETRES DE BITS DERRIERE LA LECTURE (plan
// `.ai/PLAN_RI_ETAT_COMPLET_IMAGES_CLES_2026-10-08.md`, D1.2 ; ADR 0037 IR-6, option A de
// l utilisateur du 2026-10-04).
//
// Un record bipede d image-cle que la regle d admission refuse (ou dont la phase n a pas parcouru le
// corps) passe aux fenetres de bits : les familles d arme et la marque de portage par la fenetre
// glissante de 32 bits ([motsParRecord], UN passage du payload pour les deux jeux de mots),
// l inventaire par les regles d ancrage ([keyframeInventoriesDe]) sur la SEULE emprise du record.
// Un record admis ne leur est jamais donne : la fenetre ne voit que les records non admis, les
// records admis lui sont rendus muets (archetype non resolu) sans deplacer les bornes des autres.
//
// Ce qui en sort est MARQUE recupere ([RecordRecupere], la `PreuveRecupere` de la structure de
// lecture) et COMPTE au registre des replis, une fois par record donne a chaque fenetre :
// `repli_fenetre_armes_image_cle`, `repli_fenetre_inventaire_image_cle`,
// `repli_fenetre_marque_de_portage` ([ComptesDesReplis], verses par `replay`).

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// renduDuRecord est ce qu UN record du paquet publie, lu par la grammaire ou rendu par la fenetre.
type renduDuRecord struct {
	// bipede : le record est un bipede ; admis : sa valeur vient de la grammaire.
	bipede, admis bool
	armes         *types.KeyframeLoadout
	inventaire    *types.KeyframeInventory
	marque        bool
	// recupere : ce que la fenetre a rendu d un record non admis.
	recupere RecordRecupere
}

// RecordRecupere marque un record bipede d image-cle NON ADMIS dont une fenetre de bits a rendu une
// valeur : la `PreuveRecupere` de la structure de lecture (ADR 0037 IR-6), avec ses methodes.
type RecordRecupere struct {
	// TimestampUS, Slot : l image-cle et le bipede.
	TimestampUS uint64
	Slot        uint32
	// Armes, Inventaire, Marque : les fenetres qui ont rendu une valeur (`repli_fenetre_armes_image_cle`,
	// `repli_fenetre_inventaire_image_cle`, `repli_fenetre_marque_de_portage`).
	Armes, Inventaire, Marque bool
}

// fenetresDerriereLaLecture donne aux fenetres les records bipedes non admis du paquet, et seulement
// eux, et compte chaque record donne.
func (c *canalDeLEtatCompletBipede) fenetresDerriereLaLecture(p *lecture.Paquet, rendus []renduDuRecord) {
	recs := append([]lecture.Record(nil), p.Records...)
	nonAdmis := 0
	for i := range rendus {
		switch {
		case rendus[i].admis:
			recs[i].TI = lecture.TINonResolu // muet pour la fenetre, ses bornes restent
		case rendus[i].bipede:
			nonAdmis++
		}
	}
	if nonAdmis == 0 {
		return
	}
	a := &c.out.Admission
	a.FenetresArmes += nonAdmis
	// ARMES SEULES ([ScanArmesDesImagesCles]) : ni le jeu de mots de la marque ni les regles
	// d inventaire ne tournent, et leurs replis ne voient aucun record.
	jeux := []map[uint32]bool{c.known}
	if !c.armesSeules {
		a.FenetresInventaire += nonAdmis
		a.FenetresMarque += nonAdmis
		jeux = append(jeux, carrierMarkViews)
	}
	mots := motsParRecord(p.Payload, recs, keyframeBipedTI, jeux...)
	familles, marques := map[uint32][]uint32{}, map[uint32]bool{}
	for _, rf := range mots[0] {
		familles[rf.Rec.Debut] = rf.Families
	}
	var emprises []invRecordSpan
	if !c.armesSeules {
		for _, rf := range mots[1] {
			marques[rf.Rec.Debut] = true
		}
		emprises = invRecordSpansDe(p.Payload, recs)
	}
	for i := range rendus {
		if !rendus[i].bipede || rendus[i].admis {
			continue
		}
		var emprise invRecordSpan
		if emprises != nil {
			emprise = emprises[i]
		}
		c.rendreParLaFenetre(p, &p.Records[i], emprise, familles, marques, &rendus[i])
	}
}

// rendreParLaFenetre rend ce que les fenetres trouvent d UN record non admis (ses armes seules sous
// [ScanArmesDesImagesCles]).
func (c *canalDeLEtatCompletBipede) rendreParLaFenetre(p *lecture.Paquet, r *lecture.Record, emprise invRecordSpan,
	familles map[uint32][]uint32, marques map[uint32]bool, rendu *renduDuRecord) {
	rendu.recupere = RecordRecupere{TimestampUS: p.TS, Slot: r.Vie.Slot}
	if fs := familles[r.Debut]; len(fs) > 0 {
		rendu.armes = &types.KeyframeLoadout{Slot: r.Vie.Slot, Families: fs}
		rendu.recupere.Armes = true
	}
	if c.armesSeules {
		return
	}
	if invs := keyframeInventoriesDe(p.Payload, []invRecordSpan{emprise}, c.known, c.grenMax); len(invs) == 1 {
		rendu.inventaire = &invs[0]
		rendu.recupere.Inventaire = porteUneValeur(invs[0])
		switch {
		case !invs[0].GrenadesRead:
		case invs[0].GrenadesByPosition:
			c.out.StatsInventaire.GrenadesByPosition++
		default:
			c.out.StatsInventaire.GrenadesByAnchor++
		}
	}
	rendu.marque = marques[r.Debut]
	rendu.recupere.Marque = rendu.marque
}

// porteUneValeur dit si une lecture d inventaire de la fenetre porte au moins une valeur (compteurs,
// munitions, capacite, emplacement desire ou selection) : les regles d ancrage rendent une lecture
// pour chaque record qu on leur donne, meme vide, et une lecture vide n est pas une recuperation.
func porteUneValeur(inv types.KeyframeInventory) bool {
	return inv.GrenadesRead || inv.AmmoRead || inv.AbilityRank >= 0 || inv.DrawnSlot >= 0 || inv.SelectedGrenadeRank >= 0
}

// emettre publie, dans l ordre des records du paquet, ce que chacun rend, et marque les records
// recuperes.
func (c *canalDeLEtatCompletBipede) emettre(p *lecture.Paquet, rendus []renduDuRecord) {
	for i := range rendus {
		rd := &rendus[i]
		slot := p.Records[i].Vie.Slot
		if rd.armes != nil {
			rd.armes.TimestampUS, rd.armes.Chunk, rd.armes.PacketIndex = p.TS, p.Chunk, p.Index
			c.out.Loadouts = append(c.out.Loadouts, *rd.armes)
		}
		if rd.inventaire != nil {
			rd.inventaire.TimestampUS, rd.inventaire.Chunk, rd.inventaire.PacketIndex = p.TS, p.Chunk, p.Index
			c.out.Inventaire = append(c.out.Inventaire, *rd.inventaire)
			c.out.StatsInventaire.Records++
		}
		if rd.marque {
			c.out.Marques.Marks = append(c.out.Marques.Marks, CarrierMark{TimestampUS: p.TS, Slot: slot})
		}
		if k := rd.recupere; k.Armes || k.Inventaire || k.Marque {
			c.out.Recuperes = append(c.out.Recuperes, k)
		}
	}
}
