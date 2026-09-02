package filmdec

// zone_state_scan.go — LE BALAYAGE DES PROPRIETES RESEAU DE ti=13, en production.
//
// # CE QUE CE BALAYAGE FAIT SORTIR
//
// L'archetype 13 est un sac de PROPRIETES NOMMEES attachees a un objet gere par le mode
// (cf. components_managed_property.go). Ses trois desers etaient portes et publiaient deja par
// hook ; personne ne les APPELAIT en dehors des vecteurs figes. Ce fichier est l'appelant : il
// ancre les records delta de l'archetype, marche leur masque avec les desers DE PRODUCTION, et
// rend les valeurs datees sur l'HORLOGE MOTEUR du film — la meme que les positions de bipede.
//
// # LA REGLE QUI GOUVERNE : LE DESERIALISEUR PUBLIE, JAMAIS UN SECOND LECTEUR
//
// Meme regle qu'`equipment_state.go`, et elle a une histoire ici. La phase 2a du lot C-bis a
// mesure `ti=13` depuis le paquet `replay` en RECOPIANT dans des fichiers de test l'ancrage, la
// table de largeurs et la bande de slots — la seule voie honnete a l'epoque, parce que la mesure
// avait besoin du pont geometrique qui vit dans `replay` et que `filmdec` ne peut pas importer
// (cycle). La recopie a coute deux gardes permanentes (registre, chainage) pour prouver qu'elle
// lisait les memes bits. Ce fichier la SUPPRIME : le balayage vit desormais du bon cote, avec
// `matchWorldObjectRecord`, `consumeByName` et `worldObjectSlotBand` — zero copie de grammaire.
//
// # UN SEUL HOOK : LE VARIANT DE LA PROPRIETE, PAR SES DEUX VOIES
//
//	i1        le variant en mode A (valeur SCALAIRE de la propriete) ;
//	i2..i33   le variant en mode B (une valeur PAR JOUEUR), dont l'index de joueur se
//	          reconstitue chez l'appelant — ici — par `ManagedPropertyFilmIndex`.
//
// Le composant i0 (`managed-object-property-name-component`, le NOM de la propriete reseau) est
// MARCHE mais pas recolte : personne ne le consomme — la jointure des zones se fait par le slot,
// et le tag 5 (cle de nommage) est une valeur du variant comme les autres. Le balayage en a
// publie un extrait (nom dominant par slot) jusqu'a la revue R1 de la phase 2b, qui l'a retire
// comme sortie morte : une carte de plus a remplir, un second hook global a poser et restaurer,
// pour une donnee que rien ne lisait.
//
// # CE QU'IL NE DIT PAS
//
// La SEMANTIQUE des tags. Ce balayage rend des couples (tag, quantum) bruts ; ce que le tag 3
// (jauge de capture), le tag 4 (proprietaire) ou le tag 5 (cle de nommage) veulent dire a ete
// etabli ailleurs, par la mesure, et c'est `analysis/replay` qui l'exploite. Un decodeur qui
// nommerait ses propres canaux figerait une interpretation dans le decodage.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requete.

import (
	"fmt"

	"levelup/go-api/internal/analysis/filmsource"
)

// ManagedPropertyTypeIndex est l'index d'archetype des proprietes reseau d'objet gere.
const ManagedPropertyTypeIndex = 13

// ManagedPropertyRead est UNE valeur de propriete lue dans un paquet delta.
type ManagedPropertyRead struct {
	// Slot identifie l'objet gere qui porte la propriete. C'est la cle de regroupement : un
	// slot = une propriete nommee (mesure de la phase 2a, coherence du tag par slot 99,9 %).
	Slot uint32
	// TimestampUS est l'horodatage MOTEUR du paquet — la meme horloge que
	// `BipedPosition.TimestampUS`, donc directement posable sur la grille de frames du rejeu.
	// C'est ce qui evite a l'appelant la correction d'origine que paient les calques dates
	// depuis le premier paquet du film.
	TimestampUS uint64
	// Field dit par quelle voie la valeur est venue : scalaire (i1) ou par joueur (i2..i33).
	Field ManagedPropertyField
	// FilmIndex est l'index de joueur porte par le masque, tel que LE FILM numerote les joueurs :
	// 0..31 en mode B (une valeur par joueur), -1 en mode A (valeur scalaire).
	//
	// LE NOM DIT SON STATUT, ET C'EST DELIBERE (meme regle que `FireEvent.FilmIndex`, garde-rail
	// `archlint/no_player_index_identity_test.go`). Il s'appelait `PlayerIndex`, ce qui laissait
	// croire a une identite de joueur. C'est une NUMEROTATION INTERNE AU FILM, valable seulement a
	// l'interieur de ce film : elle ne coincide avec aucun ordre que nous fabriquons, et surtout pas
	// avec le tri alphabetique du roster. L'IDENTITE D'UN JOUEUR EST SON XUID — toute jointure
	// entre joueurs passe par lui ; cet index ne sert qu'a distinguer les valeurs par joueur d'un
	// meme slot A L'INTERIEUR du film.
	FilmIndex int
	// Tag est l'alternative du variant (cf. ManagedPropertyTag*) : le TYPE de la propriete.
	Tag int
	// Value est le quantum BRUT, et HasValue dit si la branche du tag a lu quelque chose.
	//
	// UNE BRANCHE MUETTE EST UNE INFORMATION, pas un manque : elle dit que la propriete existe
	// et que ce record n'en porte pas la valeur (cf. le hook).
	Value    uint64
	HasValue bool
	// Chained dit que le RECORD qui porte cette lecture est CHAINE : sa position de fin porte un
	// en-tete de record valide (cf. ManagedPropertyScan.Chained, le meme temoin, compte par
	// record). C'est le seul temoin de fiabilite PAR LECTURE que le balayage possede, et il est
	// publie pour que l'appelant puisse ecarter la contamination d'ancrage — le canal par joueur
	// (i2..i33) chaine a 33 % sur un KOTH de reference contre 97 % pour le canal scalaire
	// (mesure du lot C-bis phase 2a). Faux pour le dernier record d'un paquet (rien ne peut le
	// suivre) : le filtre coute ~3 % de lectures reelles, il ne les invente jamais. Ajoute par
	// le lot C-ter volet 1 (2026-08-19), sans effet sur les consommateurs existants, qui ne le
	// lisent pas.
	//
	// IL RESTE UTILE APRES LE PASSAGE A LA BANDE OBSERVEE (2026-09-01) : la bande d'ancrage ne
	// COMBLE plus les trous de slots — c'etait la source principale de faux records — mais le
	// chainage plafonne encore a 77 % sur le meilleur film du corpus. Un quart des marches
	// abouties ne retombe pas sur un en-tete.
	Chained bool
}

// ManagedPropertyScan est ce qu'un balayage rend : les lectures, et de quoi juger.
type ManagedPropertyScan struct {
	Reads []ManagedPropertyRead
	// Slots est la taille de la bande d'ancrage : le denominateur de tout ce qui suit.
	Slots int
	// Records / Walked / Broken : records ancres, records dont la marche a abouti, et ceux
	// dont elle s'est arretee (composant non porte, debordement du payload).
	Records, Walked, Broken int
	// Chained compte les marches abouties dont la position de fin porte un EN-TETE DE RECORD
	// valide. C'est le temoin de largeur du balayage, et il est publie plutot que garde pour
	// les journaux : une grammaire fausse le fait s'effondrer. L'appelant le compare a `Walked`.
	//
	// L'ECHELLE DE LECTURE, ET ELLE A ETE CORRIGEE (2026-09-01). Le plancher est de 3 % —
	// `worldObjectHeaderAt` accepte cette part de positions arbitraires — et le repere de
	// « 87 a 99 % » que ce champ affichait n'etait pas reproductible : passe sur ses propres
	// modes, ce balayage plafonne a 77 % (KOTH) et rend 39 a 56 % (Bastion). Sur un mode qui
	// n'a pas d'objet gere a lire (CTF), la bande comblee tombait a 2,6 %, c'est-a-dire AU
	// PLANCHER : ses 5 843 records chaines etaient tous explicables par le hasard.
	Chained int
}

// ScanFilmManagedProperties balaye les paquets delta du film de dir et rend les valeurs des
// proprietes reseau de ti=13.
//
// LA BANDE D'ANCRAGE EST CELLE DES SLOTS OBSERVES, PAS LA BANDE COMBLEE (`observedSlotBand`,
// mesure du 2026-09-01). Une propriete d'objet gere est portee par un objet du MODE — zone de
// Bastion, objet de colline : peu nombreux, vivants toute la partie, presents a CHAQUE
// image-cle. Le comblement, qui existe pour rattraper les vies invisibles des objets ephemeres
// (cf. `slot_band_observed.go`), n'a donc rien a rattraper ici et ne fait qu'elargir la fenetre
// d'ancrage. Ce qu'il coutait, mesure sur les quatre films temoins :
//
//	Strongholds (x2)   26 slots -> 26 : la bande comblee n'ajoutait RIEN, rien ne change ;
//	KOTH arene         52 slots -> 20, chainage 47,6 % -> 77,0 % ;
//	CTF arene         914 slots -> 20, chainage  2,6 % -> 32,2 % (mode hors production : sans
//	                   role de zone TENUE, `replaybuild` ne fournit aucun catalogue et ce
//	                   balayage n'a pas lieu).
//
// LE CONSOMMATEUR NE PERD RIEN, ET C'EST LA MESURE QUI DECIDE : sur les SIX films du corpus qui
// s'assemblent (4 KOTH, 2 Bastion), `analysis/replay` publie des etats de zone IDENTIQUES —
// memes zones, memes intervalles, memes proprietaires, meme designateur de colline, meme nombre
// de frames actives — pour 28 a 42 % de lectures en moins sur les KOTH et 0 sur les Bastion
// (ou la bande comblee n'ajoutait aucun slot). Voir `replay.TestZoneBandeTi13Consommateur`.
//
// UN SEUL DECODAGE filmdec A LA FOIS PAR PROCESS : ce balayage installe un hook global de
// paquet. Il est restaure a la sortie, y compris en cas d'erreur.
//
// ScanFilmManagedProperties est l'ENVELOPPE D2, HORS PRODUCTION ; la cuisson appelle
// [ScanManagedProperties].
func ScanFilmManagedProperties(dir string) (ManagedPropertyScan, error) {
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return ManagedPropertyScan{}, err
	}
	return ScanManagedProperties(film)
}

// ScanManagedProperties décode les propriétés réseau ti=13 d'un film DEJA CHARGE.
func ScanManagedProperties(film *filmsource.Film) (ManagedPropertyScan, error) {
	sc := ManagedPropertyScan{}
	nums := FilmChunkNumbers(film)
	if len(nums) == 0 {
		return sc, ErrNoFilmChunk
	}
	band := observedSlotBand(film, ManagedPropertyTypeIndex)
	if len(band) == 0 {
		return sc, fmt.Errorf("aucun slot d'archetype ti=%d dans les keyframes du film",
			ManagedPropertyTypeIndex)
	}
	sc.Slots = len(band)
	arch, err := managedPropertyArchetype(film)
	if err != nil {
		return sc, err
	}
	w := managedPropertyWalk{arch: arch}
	defer w.install()()
	for _, c := range nums {
		data, pks, ok := FilmChunkAt(film, c)
		if !ok {
			continue
		}
		for _, pk := range pks {
			if pk.Type != PacketTypeDelta {
				continue
			}
			w.scanPayload(pk.Payload(data), band, pk.TimestampUS, &sc)
		}
	}
	return sc, nil
}

// managedPropertyArchetype charge l'archetype des proprietes reseau (ti=13) du registre.
//
// LE DECOUPAGE DU REGISTRE CHANGE AVEC LE BUILD (mesure du lot 0) : les noms sont lus du film,
// jamais supposes aux index attendus — c'est `consumeByName` qui route, et un archetype dont les
// noms ne sont pas ceux de ti=13 rend simplement zero lecture.
func managedPropertyArchetype(film *filmsource.Film) (Archetype, error) {
	reg, err := filmRegistry(film)
	if err != nil {
		return Archetype{}, err
	}
	arch, ok := reg.Archetype(ManagedPropertyTypeIndex)
	if !ok {
		return Archetype{}, fmt.Errorf("archetype ti=%d absent du registre", ManagedPropertyTypeIndex)
	}
	return arch, nil
}

// managedPropertyWalk porte ce que la marche d'un record doit connaitre, et l'etat que le hook
// y depose (regle des 5 parametres).
type managedPropertyWalk struct {
	arch Archetype
	// cur est la lecture en cours : le hook n'a pas le contexte du record, l'appelant si.
	cur ManagedPropertyRead
	// got dit que le hook a publie pour le composant courant.
	got bool
}

// install pose le hook du variant et rend sa restauration (defer).
func (w *managedPropertyWalk) install() func() {
	prev := managedPropertyHook
	SetManagedPropertyHook(func(f ManagedPropertyField, values []uint64) {
		if len(values) == 0 {
			return
		}
		w.cur.Field, w.cur.Tag = f, int(values[0])
		w.cur.Value, w.cur.HasValue = 0, false
		if len(values) > 1 {
			w.cur.Value, w.cur.HasValue = values[1], true
		}
		w.got = true
	})
	return func() { SetManagedPropertyHook(prev) }
}

// scanPayload balaye UN payload delta : ancre les records de la bande, marche leur masque, et
// compte le chainage.
func (w *managedPropertyWalk) scanPayload(pay []byte, band map[uint32]bool, ts uint64,
	sc *ManagedPropertyScan,
) {
	limit := len(pay)*8 - (worldObjectHeaderBits + worldObjectIndexBits)
	for p := 0; p <= limit; p++ {
		rec, ok := matchWorldObjectRecord(pay, p, band)
		if !ok {
			continue
		}
		sc.Records++
		first := len(sc.Reads) // les lectures de CE record commencent ici
		end, done := w.walk(pay, rec, ts, sc)
		switch {
		case !done:
			sc.Broken++
		default:
			sc.Walked++
			if worldObjectHeaderAt(pay, end) {
				sc.Chained++
				for k := first; k < len(sc.Reads); k++ {
					sc.Reads[k].Chained = true
				}
			}
		}
		p = rec.After // un record reconnu n'est pas re-balaye
	}
}

// walk marche les composants du masque avec les desers DE PRODUCTION et recolte ce que le hook
// publie. Rend la position de fin et l'aboutissement.
//
// ELLE S'ARRETE DES QU'UN COMPOSANT N'EST PAS PORTE ou que la marche deborde : au-dela, la
// position du curseur ne serait plus digne de confiance, et lire du bruit vaut moins que ne rien
// lire (meme regle que la marche des equipements).
func (w *managedPropertyWalk) walk(pay []byte, rec WorldObjectRecord, ts uint64,
	sc *ManagedPropertyScan,
) (int, bool) {
	total := len(pay) * 8
	at := rec.After
	for _, id := range rec.Idx {
		name := w.arch.component(id)
		if name == "" || at > total {
			return at, false
		}
		br := NewBitReader(pay)
		br.SetBitPos(at)
		w.got = false
		_, _, ported := consumeByName(br, name, ManagedPropertyTypeIndex, w.arch.Level(id))
		if !ported || br.BitPos() > total {
			return at, false
		}
		at = br.BitPos()
		if w.got {
			w.cur.Slot, w.cur.TimestampUS = rec.Slot, ts
			w.cur.FilmIndex = ManagedPropertyFilmIndex(id)
			sc.Reads = append(sc.Reads, w.cur)
		}
	}
	return at, true
}

// worldObjectHeaderAt dit si un en-tete de record d'objet du monde commence a la position p,
// SANS exiger que son slot appartienne a une bande.
//
// LA DEFINITION EST VOLONTAIREMENT STRUCTURELLE : le record qui SUIT un record ti=13 dans un
// paquet appartient le plus souvent a un autre archetype. Exiger la bande ferait chuter le taux
// sans rien dire de la grammaire — le chainage mesure si la LARGEUR LUE tombe juste, pas qui
// parle ensuite.
func worldObjectHeaderAt(pay []byte, p int) bool {
	total := len(pay) * 8
	if p < 0 || p+worldObjectHeaderBits+worldObjectIndexBits > total {
		return false
	}
	if PeekBits(pay, p, 1) != 1 || PeekBits(pay, p+16, 2) != 0 {
		return false
	}
	mc := int(PeekBits(pay, p+18, 3))
	if mc < 1 || mc > worldObjectMaxMaskCnt || p+worldObjectHeaderBits+worldObjectIndexBits*mc > total {
		return false
	}
	_, ok := ascendingComponents(pay, p+worldObjectHeaderBits, mc)
	return ok
}
