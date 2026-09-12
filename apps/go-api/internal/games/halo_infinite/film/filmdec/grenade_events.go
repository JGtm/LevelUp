package filmdec

import "levelup/go-api/internal/analysis/filmsource"

// grenade_events.go — LANCERS DE GRENADE, lus par balayage d'un marqueur dans les paquets
// delta (type-0).
//
// POURQUOI CE DÉCODEUR EXISTE, ET CE QU'IL EST SEUL À DONNER. Il a été écrit quand la marche
// de composants NON ancrée (DecodeFrameRecords) rendait 91,19 % de comptes i22 violant une
// borne du jeu (2 types au plus, 2 unités de chaque) — on y lisait jusqu'à 255. Le remède
// retenu alors fut celui de keyframe_loadout.go : ancrer sur une CONSTANTE cherchée bit à bit.
//
// CETTE JUSTIFICATION-LÀ N'EST PLUS LA BONNE, et le dire évite de reproduire le raisonnement.
// La chaîne de composants MARCHE désormais, sur le chemin ANCRÉ (matchBipedHeader +
// walkRecordTo) : sur 000d5950, i22 y rend 120 lectures sur 120 avec compteur == 4 et valeurs
// dans {0, 1, 2} (étude du 2026-08-24,
// .ai/V7.5/replay2d/FAISABILITE_SUIVI_DELTA_INVENTAIRE_2026-08-24.md §1.3), et c'est ce chemin
// que ScanFilmInventoryDeltas (inventory_delta.go) exploite pour SUIVRE les compteurs.
//
// CE DÉCODEUR RESTE, pour ce qu'i22 ne donne pas : le TYPE lancé et son AUTEUR à l'instant du
// lancer. i22 donne un ÉTAT (combien il en reste, par rang) ; ce fichier donne un ÉVÉNEMENT.
// Les deux sont complémentaires, ils ne se remplacent pas.
//
// GRAMMAIRE (méthode acurtis166 ; cf. .ai/GRENADE_MELEE_DETECTION.md et
// .ai/V7.5/killweapon/FIRE_MELEE_GRENADE_EVENTS.md §8, qui la classe FIABLE : 70 événements, joueurs 0-7
// tous présents sur 000d5950) :
//
//	[marqueur 24 bits = 0x4C0C00] [identifiant 32 bits] [47 bits] [index joueur 5 bits]
//
// L'index joueur est donc à +103 bits du marqueur. La source de référence décrit « sauter
// 47 bits » sans dire depuis quoi ; les deux lectures possibles (+71 depuis le marqueur,
// +103 depuis la fin de l'identifiant) ont été mesurées côté sonde et c'est +103 qui rend
// des index dans 0..7. LA VALIDATION NE VIENT PAS DU MARQUEUR SEUL : c'est l'appartenance
// de l'identifiant 32 bits à la liste blanche des quatre grenades qui fait la sélectivité.
//
// CE QUE CE DÉCODEUR NE DONNE PAS : ni le compte de grenades en réserve (c'est i22, lu par
// ScanFilmInventoryDeltas), ni la trajectoire du projectile, ni l'impact. Un lancer, son
// type, son auteur.

// Identifiants 32 bits des grenades.
//
// CE QU'ILS SONT VRAIMENT (établi le 2026-07-26, corrige la ligne suivante qui les disait
// « 32 bits hauts d'un identifiant d'arme 64 bits ») : ce sont les **identifiants globaux de
// tag du groupe `proj`**, DÉCALÉS D'UN BIT À GAUCHE.
//
//	0x580B8831 << 1 == 0xB0171062  (Fragmentation)   0x6071A622 << 1 == 0xC0E34C44 (Plasma)
//	0x1D92B3EA << 1 == 0x3B2567D4  (Dynamo/Shock)    0x49097214 << 1 == 0x9212E428 (Spike)
//
// Le champ d'identifiant commence donc à +23 bits du marqueur, pas à +24. MESURE : le bit à
// +23 vaut 0 dans les 1 416 marqueurs du film, sans exception — les deux lectures reconnaissent
// exactement les mêmes 70 lancers, elles sont informationnellement équivalentes ICI. On garde
// +24 parce que c'est ce qui est mesuré et validé.
//
// ATTENTION — L'INDEX JOUEUR NE SE DÉCALE PAS. Il reste à +103. Mesuré : à +102 les 70 valeurs
// tombent toutes entre 16 et 19 (le bit de poids fort est à 1), à +103 elles sont toutes dans
// 0..7. Propager mécaniquement le décalage d'un bit aurait cassé un décodeur qui marche.
//
// CE QUE CETTE IDENTIFICATION OUVRE : la liste blanche de quatre valeurs peut devenir une
// résolution par CATALOGUE de tags `proj` (3 086 tags dans les 132 archives .module). Lus de
// cette façon, 19 valeurs récurrentes sur 19 sont des tags `proj` — contre 1,18 % attendus par
// hasard. Cela nommerait TOUS les projectiles (roquettes, plasma, etc.), pas seulement les
// quatre grenades. Chantier à part : il faut embarquer l'index des tags.
const (
	GrenadeFragmentation uint32 = 0xB0171062
	GrenadePlasma        uint32 = 0xC0E34C44
	// GrenadeDynamo — le tag du rang 2. Il s'est appelé `GrenadeShock` jusqu'au
	// 2026-08-02 : le décodeur nommait « Shock » ce que les compteurs d'inventaire
	// nommaient « Dynamo », pour le MÊME rang et sur la MÊME fiche (constaté sur le
	// film 000d5950). Le nom du jeu est Dynamo ; le décodeur ne nomme plus rien.
	GrenadeDynamo uint32 = 0x3B2567D4
	GrenadeSpike  uint32 = 0x9212E428
)

// grenadeMarker est la constante 24 bits qui précède un lancer dans un paquet delta.
const grenadeMarker uint64 = 0x4C0C00

// grenadeThrowBits est la taille de la structure exploitée : marqueur + identifiant +
// bourrage + index joueur.
const grenadeThrowBits = 24 + 32 + 47 + 5

// GrenadeTypeIDsByRank est la liste blanche des types de grenade confirmés, DANS L'ORDRE
// DES RANGS (rang = typeId - 1). Seuls ces quatre identifiants ont été observés ; un
// cinquième type ferait un faux négatif, PAS un faux positif — la liste est donc
// conservatrice par construction.
//
// ELLE NE PORTE AUCUN NOM, et c'est le correctif du lot 3.1. Un décodeur mesure des
// identifiants ; le NOM d'un rang est un libellé de titre, il vit dans
// `config/titles/{slug}/mappings/replay_labels.toml`. Tant que le décodeur nommait
// aussi, deux tables coexistaient et elles avaient fini par diverger.
//
// L'ORDRE EST LA DONNÉE : il est établi par deux chaînes indépendantes (35 lancers
// appariés aux décréments unitaires du compteur porté, et la table `grenade_types` lue
// dans le binaire du jeu). C'est lui qui relie un lancer au compteur d'inventaire du
// même type — le réordonner désaccorderait les deux calques.
var GrenadeTypeIDsByRank = [...]uint32{
	GrenadeFragmentation,
	GrenadePlasma,
	GrenadeDynamo,
	GrenadeSpike,
}

// GrenadeRankOf rend le rang d'un tag de grenade, et s'il est connu. Un tag inconnu n'a
// PAS de rang 0 par défaut : le second retour existe pour que l'appelant ne confonde
// jamais « fragmentation » avec « pas reconnu ».
func GrenadeRankOf(typeID uint32) (int, bool) {
	for rank, id := range GrenadeTypeIDsByRank {
		if id == typeID {
			return rank, true
		}
	}
	return 0, false
}

// GrenadeThrow est un lancer de grenade attribué à son auteur.
type GrenadeThrow struct {
	// TimestampUS est l'horodatage du paquet — MÊME horloge que BipedPosition.TimestampUS,
	// FireEvent.TimestampUS et KeyframeLoadout.TimestampUS.
	TimestampUS uint64
	// Chunk / PacketIndex localisent le lancer dans le film (traçabilité).
	Chunk, PacketIndex int
	// BitPos est la position du marqueur dans le payload, en bits (traçabilité).
	BitPos int
	// FilmIndex est l'index joueur 5 bits de l'auteur, tel que LE FILM l'écrit (0..7 sur un
	// match à 8 joueurs). Comme FireEvent.FilmIndex : c'est un ORDRE interne au film, pas une
	// identité. L'identité est le XUID.
	// C'est l'indexation « acurtis », à ne pas confondre avec le slot d'entité des
	// trajectoires : le pont entre les deux appartient à la couche appelante.
	FilmIndex int
	// TypeID est l'identifiant 32 bits du type de grenade.
	TypeID uint32
}

// Rank rend le RANG du type lancé (cf. GrenadeTypeIDsByRank), et s'il est connu. Le
// décodeur ne rend pas de nom : le libellé du rang appartient au titre.
func (g GrenadeThrow) Rank() (int, bool) { return GrenadeRankOf(g.TypeID) }

// ScanFilmGrenadeThrows décode tous les lancers de grenade du film de dir.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requête.
//
// ScanFilmGrenadeThrows est l'ENVELOPPE D2, HORS PRODUCTION ; la cuisson appelle
// [ScanGrenadeThrows].
func ScanFilmGrenadeThrows(dir string) ([]GrenadeThrow, error) {
	film, err := filmsource.LoadDir(dir, nil)
	if err != nil {
		return nil, err
	}
	return ScanGrenadeThrows(film)
}

// ScanGrenadeThrows décode les lancers de grenade d'un film DEJA CHARGE.
func ScanGrenadeThrows(film *filmsource.Film) ([]GrenadeThrow, error) {
	var out []GrenadeThrow
	read := 0
	for _, c := range FilmChunkNumbers(film) {
		chunk, pks, ok := FilmChunkAt(film, c)
		if !ok {
			continue
		}
		read++
		for _, p := range pks {
			if p.Type != PacketTypeDelta {
				continue
			}
			for _, g := range scanGrenadeThrows(p.Payload(chunk)) {
				g.TimestampUS, g.Chunk, g.PacketIndex = p.TimestampUS, c, p.Index
				out = append(out, g)
			}
		}
	}
	if read == 0 {
		return nil, ErrNoReadableFilmChunk
	}
	return out, nil
}

// scanGrenadeThrows balaye un payload de paquet delta. PUR (aucune I/O).
func scanGrenadeThrows(pay []byte) []GrenadeThrow {
	limit := len(pay)*8 - grenadeThrowBits
	var out []GrenadeThrow
	for bp := 0; bp <= limit; bp++ {
		if PeekBits(pay, bp, 24) != grenadeMarker {
			continue
		}
		id := uint32(PeekBits(pay, bp+24, 32))
		if _, ok := GrenadeRankOf(id); !ok {
			continue
		}
		out = append(out, GrenadeThrow{
			BitPos:    bp,
			FilmIndex: int(PeekBits(pay, bp+24+32+47, 5)),
			TypeID:    id,
		})
	}
	return out
}

// PeekBits lit n bits MSB-first à la position bp, sans curseur. Tout bit HORS du buffer vaut
// 0 — l'appelant borne son balayage, cette tolérance n'est là que pour ne jamais paniquer sur
// un payload tronqué.
//
// « Hors du buffer » veut bien dire des DEUX CÔTÉS. Jusqu'au 2026-08-01 la tolérance était à
// SENS UNIQUE : au-delà de la fin elle rendait 0, mais une position NÉGATIVE paniquait
// (`index out of range [-1]`) — le contraire de ce que cette phrase annonce (découverte J2,
// corrigée en J3.4). Aucun appelant de production ne recule sous zéro ; le défaut était que la
// documentation promettait une garantie que le code n'offrait pas, sur une primitive dont
// c'est la SEULE raison d'être.
//
// EXPORTÉ pour les sondes qui balayent un payload à la recherche d'un motif (marqueurs de
// mêlée, de tir, de lancer) sans dérouler la chaîne de composants.
//
// Lecture par mot (cf. bits_word.go) des que la position de depart est positive et la
// largeur tient sur 64 bits : `wordBitsAt` rend deja des zeros au-dela de la fin du tampon.
// Un depart NEGATIF ou une largeur > 64 retombent sur la boucle d'origine, seule a porter
// ces deux conventions.
func PeekBits(d []byte, bp, n int) uint64 {
	if bp >= 0 && n >= 0 && n <= 64 {
		return wordBitsAt(d, bp, uint(n))
	}
	var v uint64
	for i := 0; i < n; i++ {
		p := bp + i
		if p < 0 || p>>3 >= len(d) {
			v <<= 1
			continue
		}
		v = (v << 1) | uint64((d[p>>3]>>uint(7-(p&7)))&1)
	}
	return v
}
