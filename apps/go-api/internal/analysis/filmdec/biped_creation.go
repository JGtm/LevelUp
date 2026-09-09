package filmdec

// biped_creation.go — LE RECORD DE CRÉATION D'UN BIPÈDE (ti=35), ET L'INDEX QU'IL PORTE.
//
// # CE QUE CE LECTEUR FAIT SORTIR, ET CE QUE LE FILM DIT VRAIMENT
//
// Le default-state du bipède (`consumeBipedDefaultState`, default_state.go, port bit-exact de
// FUN_140F44C38) consommait ses TROIS premières feuilles pour rester aligné, et les jetait :
//
//	g0    = R(1) ; si 1 -> version = R(8)                       [140f44c5c / 140f44c85]
//	gRep  = R(1) ; si 1 -> R(32) « player-representation-name »  [140f44cad / 140f44cd6]
//	si version > 10 : ECS_ReadEntityRefIndex5 (FUN_1407f2058)     [140f44cdb / 140f44ce7]
//	                  R(1) porte INVERSÉE ; si le bit vaut 0 -> R(5)
//
// La troisième est **l'index de participant absolu du PROPRIÉTAIRE du corps**. Ce n'est pas une
// hypothèse : c'est la MÊME primitive que `EnumA`/`EnumB` du dead-state, que la production lit
// déjà comme des index de participant (`killsource/walk.go:224`, validés `< nPlay`), et le
// sondage E2 du 2026-09-08 l'a mesurée sur cinq films —
// `.ai/V7.5/film_re/SONDAGE_E2_BIPEDE_INDEX_2026-09-08.md` :
//
//	porte OUVERTE 529 fois sur 529 · valeur dans le roster du film 527 fois sur 529
//	couverture des vies nommées DIRECTEMENT : 90,3 % / 86,8 % / 95,0 % (372 vies sur 408)
//	témoin fantôme (bande de slots qu'aucun bipède n'occupe) : 0 lecture sur les 5 films
//
// Les deux exceptions sont un index de participant que l'artefact ne publie pas, sur le seul
// film à bot — cf. le verdict I0 du lot E2 : l'espace d'index est PARTAGÉ entre humains et bots,
// et un index absent de la table publiée reste `non_resolu`, jamais rattaché.
//
// # POURQUOI CE BALAYAGE NE PEUT PAS EMPRUNTER `runCreationWalk`
//
// Les trois archétypes de création déjà portés (équipement `ti=37`, arme au sol `ti=42`,
// véhicule `ti=40`) valident leur ancre par le COMPOSANT i0 décodé après le default-state. Sur
// le bipède c'est impossible par construction : son default-state n'est porté qu'à ~120 bits sur
// ~380 (cf. l'en-tête de default_state.go), donc l'offset d'i0 calculé après lui est FAUX. La
// mesure du sondage (§2.3) le chiffre : 1 847 ancres, 181 acceptées, dont 170 à plus de
// 10 000 quanta de la trajectoire réelle de leur propre slot — 90 % de bruit.
//
// LE GATE EST DONC LA SIGNATURE, et elle est bien plus sévère : `g0 = 1`, `version = 13`,
// `gRep = 1`, mot de représentation = [BipedRepresentationName], porte du ref OUVERTE. Cela fait
// **43 bits déterminés** à l'intérieur d'une ancre déjà contrainte par trois constantes de format
// et par la bande de slots.
//
// # LA LIMITE DU GATE PAR SIGNATURE, ÉCRITE ICI PARCE QU'ELLE EST RÉELLE
//
// Un bipède dont la représentation différerait (autre corps que le Spartan multijoueur) ne
// serait pas lu. Le lecteur COMPTE donc, dans [BipedCreationStats.SignatureMismatch], les ancres
// qui ont toute la FORME d'une création de bipède mais un autre mot de 32 bits, et publie le mot
// alternatif le plus fréquent ([BipedCreationStats.OtherWord]). Ce compteur a un plancher de
// bruit — un ancrage bit à bit accroche par hasard — : il se juge en ORDRE DE GRANDEUR contre
// `Accepted`, et l'alarme que l'appelant arme est `Anchors > 0 && Accepted == 0`, le cas d'un
// film dont les bipèdes ne portent jamais la constante.
//
// # LE LECTEUR N'A BESOIN D'AUCUNE HORLOGE
//
// La création DÉSIGNE sa vie par le couple `(slot, génération)` du handle — la même clé que
// `EquipmentLifeKey` pour les objets du monde. L'horodatage publié sert au rattachement et aux
// journaux, jamais à une décision de lecture.
//
// HORS LIGNE (parcours de tous les paquets delta du film) — jamais depuis un chemin de requête.

import (
	"errors"
	"fmt"
	"sort"
)

// BipedRepresentationName est le mot de 32 bits « player-representation-name » (FUN_14080dec4)
// que porte le record de création d'un bipède de multijoueur.
//
// CONSTANTE MESURÉE, pas devinée : `0x1876BDA0` sur les 529 records certains des cinq films du
// sondage E2 (Oddball, Slayer, CTF, Assaut, et le film à bot), sans une seule exception. C'est
// elle qui sert de gate de sélectivité, et c'est elle dont l'absence se compte.
const BipedRepresentationName uint32 = 0x1876BDA0

// bipedCreationVersion est la valeur du sélecteur `uVar10` (version du default-state) qu'un
// record de création de bipède porte : 13 sur 529 records sur 529. C'est aussi la valeur par
// DÉFAUT du désérialiseur (`uVar10 := uint32(13)` dans consumeBipedDefaultState) — mais les
// records mesurés l'écrivent EXPLICITEMENT, porte `g0` ouverte, et le gate l'exige sous cette
// forme : une porte fermée n'est pas la même chose qu'une valeur écrite.
const bipedCreationVersion uint32 = 13

// bipedCreationIndexBits est la largeur de l'index de participant rendu par
// `ECS_ReadEntityRefIndex5` (FUN_1407f2058) : cinq bits, domaine 0..31.
const bipedCreationIndexBits = 5

// bipedCreationPrologueBits est le nombre de bits que le prologue consomme quand toutes ses
// portes sont ouvertes : g0(1) + version(8) + gRep(1) + representation(32) + porte(1) + index(5).
const bipedCreationPrologueBits = 1 + 8 + 1 + 32 + 1 + bipedCreationIndexBits

// BipedCreation est UN record de création de bipède, lu jusqu'à son index de participant.
type BipedCreation struct {
	// Slot et Generation identifient LA VIE du corps — la paire, pas le slot seul : le pool de
	// slots reboucle et la génération ne fait que deux bits. C'est la même clé de vie que
	// `EquipmentLifeKey` emploie pour les objets du monde.
	Slot, Generation uint32
	// ParticipantIndex est l'index de participant ABSOLU du propriétaire du corps, dans l'espace
	// que `PlayerIndexTable` publie pour les humains et que `BOT_METADATA` emploie pour les bots
	// (verdict I0 du lot E2 : c'est le MÊME espace). Valide seulement si HasIndex.
	ParticipantIndex uint32
	// HasIndex dit que la porte INVERSÉE d'`ECS_ReadEntityRefIndex5` a transmis la valeur. Une
	// porte fermée n'est PAS un index nul.
	HasIndex bool
	// Chunk / PacketIndex / TimestampUS localisent la lecture — même horloge que BipedPosition.
	Chunk, PacketIndex int
	TimestampUS        uint64
	// BitPos est la position de l'en-tête du record dans le payload, en bits (traçabilité).
	BitPos int
	// Version est le sélecteur `uVar10` lu ; Representation le mot de 32 bits. Publiés parce
	// qu'un gate ne se juge pas sans les valeurs qu'il a laissées passer.
	Version        uint32
	Representation uint32
}

// LifeKey rend la clé de vie du corps — le couple `(slot, génération)`, sous la forme que les
// consommateurs comparent. Un slot seul désignerait toutes les vies du corps à la fois.
func (c BipedCreation) LifeKey() uint32 { return c.Slot | (c.Generation << 16) }

// BipedCreationStats compte ce que le balayage a rencontré. Sans ces dénominateurs, une
// couverture ne se juge pas — et sans le détail des rejets, on ne sait pas si le balayage rate
// des records ou en invente.
type BipedCreationStats struct {
	// Slots est le nombre de slots de la bande passée au balayage.
	Slots int
	// Anchors est le nombre d'en-têtes NEW `ti=35` reconnus (trois constantes de format + la
	// bande de slots). C'est le DÉNOMINATEUR du gate, pas un nombre de records.
	Anchors int
	// Truncated : le prologue déborde du payload — le curseur n'est plus digne de confiance.
	Truncated int
	// ShapeBad : l'ancre n'a pas la forme d'une création de bipède (porte de version fermée,
	// version ≠ 13, ou porte de représentation fermée). Le rejet ORDINAIRE de l'ancrage bit à
	// bit ; il n'alarme rien.
	ShapeBad int
	// SignatureMismatch : l'ancre a toute la FORME d'une création de bipède, mais son mot de
	// 32 bits n'est pas [BipedRepresentationName]. C'est LE compteur à surveiller — cf.
	// l'en-tête du fichier pour son plancher de bruit et pour l'alarme que l'appelant arme.
	SignatureMismatch int
	// OtherWord est le mot de 32 bits alternatif le plus fréquent parmi les
	// `SignatureMismatch`, et OtherWordCount son compte. Zéro quand il n'y en a aucun.
	OtherWord      uint32
	OtherWordCount int
	// GateClosed : signature reconnue, mais la porte INVERSÉE de l'index s'est FERMÉE — le
	// record ne porte pas d'index. Mesuré à ZÉRO sur 529 records ; s'il devient non nul, c'est
	// un fait à instruire, pas un défaut de lecture.
	GateClosed int
	// Accepted est le nombre de records rendus (signature reconnue ET index transmis).
	Accepted int
}

// ScanBipedCreations décode les records de création de bipède d'un film DÉJÀ CHARGÉ, sur la
// bande de slots `ti=35` relevée aux images-clés — la MÊME que celle des trajectoires
// (`FilmContext.BipedSlots`), pour que les deux lectures parlent des mêmes corps.
//
// HORS LIGNE (parcours de tous les paquets delta du film) — jamais depuis un chemin de requête.
func ScanBipedCreations(fc *FilmContext) ([]BipedCreation, BipedCreationStats, error) {
	if fc == nil {
		return nil, BipedCreationStats{}, errors.New("contexte de film absent")
	}
	if len(fc.ChunkNumbers()) == 0 {
		return nil, BipedCreationStats{}, ErrNoFilmChunk
	}
	band := fc.BipedSlots()
	if band.Count() == 0 {
		return nil, BipedCreationStats{}, fmt.Errorf(
			"aucun slot biped (ti=%d) dans les keyframes du film", BipedTypeIndex)
	}
	return ScanBipedCreationsForBand(fc, band)
}

// ScanBipedCreationsForBand balaye une BANDE DE SLOTS donnée.
//
// LA BANDE EST UN PARAMÈTRE POUR QUE LE TÉMOIN passe par le MÊME code que la mesure : une bande
// FANTÔME de même cardinalité, faite de slots qu'aucun bipède n'occupe, mesure le plancher de
// faux positifs du gate. Sans ce passage par le même décodeur, le témoin ne contrôlerait pas le
// décodeur mais une variante de lui (règle établie par `WorldObjectPositionsForBand`).
func ScanBipedCreationsForBand(
	fc *FilmContext, band SlotBand,
) ([]BipedCreation, BipedCreationStats, error) {
	var st BipedCreationStats
	if fc == nil {
		return nil, st, errors.New("contexte de film absent")
	}
	nums := fc.ChunkNumbers()
	if len(nums) == 0 {
		return nil, st, ErrNoFilmChunk
	}
	if band.Count() == 0 {
		return nil, st, errors.New("bande de slots vide")
	}
	st.Slots = band.Count()
	w := bipedCreationWalk{band: band, st: &st, autres: map[uint32]int{}}
	var out []BipedCreation
	for _, c := range nums {
		data, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			out = append(out, w.scanPayload(pk.Payload(data), pk, c)...)
		}
	}
	st.OtherWord, st.OtherWordCount = motAlternatifModal(w.autres)
	return out, st, nil
}

// bipedCreationWalk porte ce que la marche d'un payload doit connaître (règle des 5 paramètres) :
// la bande, les compteurs, et l'histogramme des mots de représentation refusés.
type bipedCreationWalk struct {
	band   SlotBand
	st     *BipedCreationStats
	autres map[uint32]int
}

// scanPayload balaye UN payload delta et rend les records de création reconnus.
//
// Le curseur saute le prologue d'un record ACCEPTÉ : un record reconnu n'est pas re-balayé, et
// c'est ce qui interdit qu'une même création soit rendue deux fois par glissement d'un bit.
func (w bipedCreationWalk) scanPayload(pay []byte, pk FilmPacket, chunk int) []BipedCreation {
	var out []BipedCreation
	total := len(pay) * 8
	limit := total - woNewHeaderBits - bipedCreationPrologueBits
	for p := 0; p <= limit; p++ {
		slot, gen, ok := matchWorldObjectNewHeaderIn(pay, p, w.band.Has, BipedTypeIndex)
		if !ok {
			continue
		}
		w.st.Anchors++
		cre, ok := w.readCreation(pay, p, total)
		if !ok {
			continue
		}
		cre.Slot, cre.Generation, cre.BitPos = slot, gen, p
		cre.Chunk, cre.PacketIndex, cre.TimestampUS = chunk, pk.Index, pk.TimestampUS
		w.st.Accepted++
		out = append(out, cre)
		p += woNewHeaderBits + bipedCreationPrologueBits - 1
	}
	return out
}

// readCreation rejoue les TROIS premières feuilles de `consumeBipedDefaultState` à l'intérieur
// d'une ancre, et applique le gate de signature.
//
// CHAQUE REFUS EST COMPTÉ À PART, et les catégories ne sont pas décoratives : `ShapeBad` est le
// rejet ordinaire d'un ancrage bit à bit, `SignatureMismatch` est le seul qui dise quelque chose
// du FILM, et `GateClosed` est un fait mesuré à zéro dont la remontée serait une découverte.
func (w bipedCreationWalk) readCreation(pay []byte, p, total int) (BipedCreation, bool) {
	var cre BipedCreation
	st := w.st
	start := p + woNewHeaderBits
	if start+bipedCreationPrologueBits > total {
		st.Truncated++
		return cre, false
	}
	br := NewBitReader(pay)
	br.SetBitPos(start)
	// g0 : la porte de version. Les records mesurés l'ouvrent et écrivent 13 EXPLICITEMENT ;
	// une porte fermée laisserait la version implicite, et le gate ne l'accepte pas.
	if !br.ReadBit() {
		st.ShapeBad++
		return cre, false
	}
	cre.Version = uint32(br.ReadBits(8))
	if cre.Version != bipedCreationVersion {
		st.ShapeBad++
		return cre, false
	}
	// gRep : la porte du mot « player-representation-name ». Fermée, il n'y a pas de signature
	// à confronter — l'ancre n'a pas la forme, elle n'est pas un désaccord de signature.
	if !br.ReadBit() {
		st.ShapeBad++
		return cre, false
	}
	cre.Representation = uint32(br.ReadBits(32))
	if cre.Representation != BipedRepresentationName {
		st.SignatureMismatch++
		w.autres[cre.Representation]++
		return cre, false
	}
	// ECS_ReadEntityRefIndex5 : la porte est INVERSÉE — le bit à ZÉRO annonce la valeur.
	if br.ReadBit() {
		st.GateClosed++
		return cre, false
	}
	cre.HasIndex = true
	cre.ParticipantIndex = uint32(br.ReadBits(bipedCreationIndexBits))
	return cre, true
}

// motAlternatifModal rend le mot de 32 bits alternatif le plus fréquent et son compte. Le
// départage par la valeur rend le résultat REPRODUCTIBLE : l'ordre d'itération d'une map Go ne
// l'est pas, et un compteur qui change d'un run à l'autre ne se compare pas.
func motAlternatifModal(m map[uint32]int) (uint32, int) {
	if len(m) == 0 {
		return 0, 0
	}
	mots := make([]uint32, 0, len(m))
	for w := range m {
		mots = append(mots, w)
	}
	sort.Slice(mots, func(i, j int) bool {
		if m[mots[i]] != m[mots[j]] {
			return m[mots[i]] > m[mots[j]]
		}
		return mots[i] < mots[j]
	})
	return mots[0], m[mots[0]]
}
