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
//	          reconstitue chez l'appelant — ici — par `ManagedPropertyPlayerIndex`.
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

import "fmt"

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
	// PlayerIndex vaut 0..31 en mode B, et -1 en mode A.
	PlayerIndex int
	// Tag est l'alternative du variant (cf. ManagedPropertyTag*) : le TYPE de la propriete.
	Tag int
	// Value est le quantum BRUT, et HasValue dit si la branche du tag a lu quelque chose.
	//
	// UNE BRANCHE MUETTE EST UNE INFORMATION, pas un manque : elle dit que la propriete existe
	// et que ce record n'en porte pas la valeur (cf. le hook).
	Value    uint64
	HasValue bool
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
	// les journaux : une grammaire fausse le fait s'effondrer (2-3 % sur une bande fantome,
	// contre 87 a 99 % mesures sur le corpus). L'appelant le compare a `Walked`.
	Chained int
}

// ScanFilmManagedProperties balaye les paquets delta du film de dir et rend les valeurs des
// proprietes reseau de ti=13.
//
// UN SEUL DECODAGE filmdec A LA FOIS PAR PROCESS : ce balayage installe un hook global de
// paquet. Il est restaure a la sortie, y compris en cas d'erreur.
func ScanFilmManagedProperties(dir string) (ManagedPropertyScan, error) {
	sc := ManagedPropertyScan{}
	n := CountFilmChunks(dir)
	if n == 0 {
		return sc, fmt.Errorf("aucun chunk film dans %s", dir)
	}
	band := worldObjectSlotBand(dir, n, ManagedPropertyTypeIndex)
	if len(band) == 0 {
		return sc, fmt.Errorf("aucun slot d'archetype ti=%d dans les keyframes de %s",
			ManagedPropertyTypeIndex, dir)
	}
	sc.Slots = len(band)
	arch, err := managedPropertyArchetype(dir)
	if err != nil {
		return sc, err
	}
	w := managedPropertyWalk{arch: arch}
	defer w.install()()
	for c := 1; c <= n; c++ {
		data, err := ReadFilmChunk(dir, c)
		if err != nil {
			continue
		}
		for _, pk := range WalkPackets(data) {
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
func managedPropertyArchetype(dir string) (Archetype, error) {
	raw, err := ReadFilmChunk(dir, 0)
	if err != nil {
		return Archetype{}, fmt.Errorf("chunk_00 (registre) illisible dans %s : %w", dir, err)
	}
	reg, err := ParseRegistryChunk(raw)
	if err != nil {
		return Archetype{}, fmt.Errorf("registre illisible dans %s : %w", dir, err)
	}
	arch, ok := reg.Archetype(ManagedPropertyTypeIndex)
	if !ok {
		return Archetype{}, fmt.Errorf("archetype ti=%d absent du registre de %s",
			ManagedPropertyTypeIndex, dir)
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
		end, done := w.walk(pay, rec, ts, sc)
		switch {
		case !done:
			sc.Broken++
		default:
			sc.Walked++
			if worldObjectHeaderAt(pay, end) {
				sc.Chained++
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
			w.cur.PlayerIndex = ManagedPropertyPlayerIndex(id)
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
