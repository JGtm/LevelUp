package grammar

// keyframe_anticipe.go — LA TABLE ANTICIPEE DES ARCHETYPES (lot 5.23.1, 2026-09-22).
//
// # LE FAIT QUI LA JUSTIFIE, ET IL EST MESURE
//
// Sur un film dense (`bfecd02b`, 8 joueurs, Snowbound) 23 325 trames delta sur 30 387 sont
// ABANDONNEES a mi-lecture parce qu un record designe un slot que le monde hors ligne ne
// connait pas encore : l entite est nee en MILIEU de chunk (respawn, vehicule, grenade,
// equipement) et le decodeur ne lit aucun record `NEW` la ou elle nait. Le lot 5.20.2 a mesure
// la BORNE HAUTE de ce que l image-cle du chunk peut declarer — **0,0 %** des rejets, a quelque
// position de bit que ce soit — et le lot 5.21.2 la meme borne sur le bloc de type 1 —
// **0 sur 23 325**. Les deux sources du chunk COURANT sont donc fermees.
//
// Mais **74,7 % des slots rejetes sont declares, avec leur archetype, par l image-cle du chunk
// SUIVANT** (5.20.2, `TestNaissance520`). L entite nait entre deux images-cles ; la premiere
// image-cle qui la voit est celle d APRES.
//
// # CE QUE CE FICHIER EST, ET CE QU IL N EST PAS
//
// C est un REPLI, pas une grammaire : **le record de naissance n est toujours pas lu**. Une
// premiere passe sur les images-cles de TOUS les chunks construit la table
// `(slot, tete) -> archetype` du film entier ; la marche des trames, au point de rejet,
// consulte cette table AVANT de rejeter, et lie l entite PAR ANTICIPATION sur la foi d une
// image-cle ULTERIEURE. Le repli est NOMME (`LiaisonsParAnticipation`), DATE (2026-09-22) et
// COMPTE. Il ne remplace pas la lecture de la naissance : il la rend inutile pour la SUITE du
// flux.
//
// # LA CLE EST CELLE QUE LE JEU COMPARE, ET ELLE EST LUE CHEZ L ECRIVAIN
//
// `FUN_1406caad8`, la porte que tout corps de delta franchit (lecture du 2026-09-22) :
//
//	uVar21 = param_2 & 0x3fffffff                       ; le SLOT — 30 bits bas de l eid
//	si param_2 == 0xffffffff              -> return 3   ; sentinelle
//	lVar19 = *(longlong *)(param_1 + 0x20)              ; la base de la table, pas 200
//	si (fin - base)/200 <= uVar21         -> return 3   ; slot hors cardinal
//	si *(uint *)(uVar21 * 200 + lVar19) != param_2 -> return 3   ; <- LA CLE
//	puVar18[1]                                          ; l ARCHETYPE, en +0x04
//
// La table est INDEXEE par le slot, et son entree porte l eid **ENTIER** : les deux bits de
// tete comptent, et un delta dont la tete ne vaut pas celle de l entree ne rend AUCUN bit. La
// cle est donc le couple `(slot, tete)`, c est-a-dire le mot de 32 bits lui-meme.
//
// **ET LES DEUX BITS DE TETE D UNE IMAGE-CLE SONT CEUX QUE LE DELTA DOIT PRESENTER**, parce que
// c est la MEME table : l entree de 200 octets que `FUN_1406caad8` teste est celle que
// `FUN_142e2bfd0` remplit (`e[0x00] = R(32)` l eid, `e[0x04] = R(32)` l archetype — lot 5.20.1).
// Ce que ces deux bits SIGNIFIENT reste ce que le lot 5.13.1 a etabli — le rang de la vue chez
// `FUN_142f2e174`, la generation du datum chez `FUN_1408f1730` — et les deux films temoins ne
// les departagent pas : ils valent 1 partout, des deux cotes (mesure du lot, `TestTable523`).
// LA CLE, ELLE, N EST PAS AMBIGUE : c est le mot compare, et ce fichier cle dessus.
//
// # LA TABLE EST DATEE PAR CHUNK, ET CE N EST PAS UN ORNEMENT
//
// Un slot se reutilise. Une entree ne vaut donc pas « pour tout le film » mais « a partir du
// chunk qui la declare » : la consultation prend la PREMIERE declaration STRICTEMENT
// POSTERIEURE au chunk du rejet. Anticiper, c est lire l avenir du slot, jamais son passe.
//
// HORS LIGNE — jamais depuis un chemin de requete.

import "sort"

// cleAnticipee est la cle que `FUN_1406caad8` compare : le slot (30 bits bas de l eid) et sa
// TETE (les deux bits 30-31). Les deux ensemble sont le mot de 32 bits.
type cleAnticipee struct {
	slot uint32
	tete uint8
}

// declarationAnticipee : un archetype declare par l image-cle d un chunk.
type declarationAnticipee struct {
	chunk int
	ti    uint32
}

// TableAnticipee porte, pour tout le film, ce que chaque image-cle declare : `(slot, tete)` ->
// la suite DATEE des archetypes que les chunks lui donnent.
//
// Construite par une passe unique sur les images-cles ([ConstruireTableAnticipee]), consultee
// au point de rejet de la marche de trames ([TableAnticipee.ArchetypeApres]).
type TableAnticipee struct {
	entrees map[cleAnticipee][]declarationAnticipee
	// declarations est le nombre d entrees d image-cle versees (un slot declare par vingt
	// chunks en compte vingt) : c est le VOLUME de la passe, pas la taille de la table.
	declarations int
	// conflits est le nombre de cles qu au moins DEUX archetypes distincts se disputent —
	// la signature d une reutilisation de slot sous la meme tete. C est ce que la datation
	// par chunk resout, et un compteur muet le cacherait.
	conflits int
	// tetes recense les deux bits de tete rencontres : c est la mesure qui dit si la cle
	// `(slot, tete)` se distingue de la cle `slot` sur ce film.
	tetes map[uint8]int
	// marche : la marche d image-cle qui LIT les declarations — celle du film (preuve comprise, lot
	// D-fix) quand la table est construite depuis son contexte, sans preuve sinon.
	marche MarcheDImageCle
}

// NouvelleTableAnticipee rend une table vide.
func NouvelleTableAnticipee() *TableAnticipee {
	return &TableAnticipee{entrees: map[cleAnticipee][]declarationAnticipee{},
		tetes: map[uint8]int{}}
}

// ConstruireTableAnticipee lit les images-cles de TOUS les chunks du film et rend la table du
// film. UNE passe, aucun decodage de trame.
func ConstruireTableAnticipee(fc *FilmContext) *TableAnticipee {
	t := NouvelleTableAnticipee()
	if fc == nil {
		return t
	}
	t.marche = fc.MarcheDImageCle()
	for _, c := range fc.ChunkNumbers() {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		t.AjouterChunk(c, data, pks)
	}
	t.Clore()
	return t
}

// AjouterChunk verse dans la table ce que les images-cles d un chunk declarent.
//
// La lecture est celle que le monde emprunte deja pour ses liaisons d image-cle
// ([MarcheDImageCle]) : elle rend `(Slot, TI, Gen)`, c est-a-dire exactement le mot de 32 bits
// que `FUN_1406caad8` compare, decompose. Aucune seconde lecture d image-cle n est ecrite ici.
func (t *TableAnticipee) AjouterChunk(num int, data []byte, pks []FilmPacket) {
	for _, pk := range pks {
		if pk.Type != PacketTypeKeyframe {
			continue
		}
		for _, r := range t.marche.Records(pk.Payload(data)) {
			if r.Slot < 0 || r.TI < 0 {
				continue
			}
			//nolint:gosec // Slot, TI et Gen sont bornes par les gardes de kfAnchorFromID
			cle := cleAnticipee{slot: uint32(r.Slot), tete: uint8(r.Gen & 3)}
			t.tetes[cle.tete]++
			t.declarations++
			//nolint:gosec // TI < kfArchMax par la garde
			t.entrees[cle] = append(t.entrees[cle], declarationAnticipee{chunk: num, ti: uint32(r.TI)})
		}
	}
}

// Clore ordonne les declarations de chaque cle par chunk et compte les conflits. A appeler une
// fois la passe finie ; [ConstruireTableAnticipee] s en charge.
func (t *TableAnticipee) Clore() {
	for cle, decls := range t.entrees {
		sort.Slice(decls, func(a, b int) bool { return decls[a].chunk < decls[b].chunk })
		t.entrees[cle] = decls
		for i := 1; i < len(decls); i++ {
			if decls[i].ti != decls[0].ti {
				t.conflits++
				break
			}
		}
	}
}

// ArchetypeApres rend l archetype que la PREMIERE image-cle STRICTEMENT POSTERIEURE au chunk
// `chunk` donne a l eid `id`, et le numero de ce chunk.
//
// `id` est l eid ENTIER du record de delta : son slot et sa tete forment la cle que
// `FUN_1406caad8` compare. Une declaration du chunk courant ou d un chunk anterieur N EST PAS
// une anticipation — elle est deja portee par [LierTableDeDatums] — et ne rend rien ici.
func (t *TableAnticipee) ArchetypeApres(id uint32, chunk int) (ti uint32, declarant int, ok bool) {
	if t == nil {
		return 0, 0, false
	}
	cle := cleAnticipee{slot: id & 0x3fffffff, tete: uint8(id >> 30)}
	for _, d := range t.entrees[cle] {
		if d.chunk > chunk {
			return d.ti, d.chunk, true
		}
	}
	return 0, 0, false
}

// Entrees rend le nombre de cles `(slot, tete)` distinctes que la table porte.
func (t *TableAnticipee) Entrees() int {
	if t == nil {
		return 0
	}
	return len(t.entrees)
}

// Declarations rend le nombre d entrees d image-cle versees (volume de la passe).
func (t *TableAnticipee) Declarations() int {
	if t == nil {
		return 0
	}
	return t.declarations
}

// Conflits rend le nombre de cles qu au moins deux archetypes distincts se disputent — ce que
// la datation par chunk arbitre.
func (t *TableAnticipee) Conflits() int {
	if t == nil {
		return 0
	}
	return t.conflits
}

// Tetes rend le recensement des deux bits de tete rencontres dans les images-cles du film.
func (t *TableAnticipee) Tetes() map[uint8]int {
	if t == nil {
		return nil
	}
	return t.tetes
}
