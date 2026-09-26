package grammar

import (
	"context"
	"log/slog"

	"levelup/go-api/internal/games/halo_infinite/film/internal/profile"
	"levelup/go-api/internal/games/halo_infinite/film/internal/source"
)

// grenade_events.go — LANCERS DE GRENADE, lus par balayage d'un motif d'amorce dans les paquets
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
// # LA GRAMMAIRE, ET LE FAIT QU'ELLE EST UNE DONNÉE DE PROFIL DEPUIS LE LOT 3.3.1
//
// Un lancer est la NAISSANCE d'une entité de l'archétype projectile :
//
//	[typeIndex : 6 bits][amorce de l'état par défaut][identifiant de tag : 32 bits]…[index : 5]
//
// Le « marqueur 24 bits = 0x4C0C00 » que ce fichier cherchait n'était pas un motif de recherche :
// ce sont les CINQ bits bas de `typeIndex = 41` suivis de 19 bits d'amorce constante
// (`projectiles.go`, point 2). Il ne valait donc que pour UN découpage et UN registre.
//
// CE QUE LE VOLET RECHERCHE DU LOT 3.3 A MESURÉ (note
// `.ai/V7.5/film_re/NOTE_3_3_IDENTIFIANTS_GRENADE_2026-09-16.md`, découvertes D3 et D5 (3.3r),
// puis les mesures du lot 3.3.1 sur dix films) : ni le nombre, ni l'ordre, ni les valeurs des
// identifiants n'ont changé depuis la sortie du jeu — [GrenadeTypeIDsByRank] reste une constante
// du TITRE. Ce qui a changé est la grammaire, en trois nombres : la largeur du motif d'amorce
// (24 bits jusqu'à `HI_1_12_0`, 23 avant), sa VALEUR (`0x20400` sur la majeure 31 là où toutes
// les autres clés anciennes portent `0x20600`), et la position du champ d'index de l'auteur
// (+103, +100 ou +99). Les trois vivent au profil (`profile/grenade.go`), keyés par le build de
// la section 2 ou, pour les films qui n'en portent pas, par la version majeure.
//
// LE MOTIF SE DÉRIVE DU FILM, IL NE SE CÂBLE PLUS : le `typeIndex` de l'archétype projectile se
// résout par le NOM de ses composants dans le registre du film (mesure 0 du volet recherche :
// 41 sur les sept builds du cache, résolu 4/4 noms à chaque fois), puis
// [profile.AmorceGrenade.MarqueurDe] compose le motif. `0x4C0C00` et `0x4C0C01` ne sont pas deux
// constantes : c'est LA MÊME règle lue à deux largeurs.
//
// CE QUE LA SÉLECTIVITÉ DOIT AU SIXIÈME BIT D'INDEX. Le motif ne porte que CINQ bits du
// typeIndex : sur un registre de 49 ou 50 blocs, `ti=41` et `ti=9` (`managed-player`,
// `player_teams.go`) produisent le MÊME motif (découverte D2 (3.3r)). Le sixième bit est à
// `motif - 1` (`traverse.go` : `t.TypeIndex = uint32(br.ReadBits(6))`), et il est LU : une
// naissance de `managed-player` est écartée et comptée, jamais soumise à la liste blanche.
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
// ILS SONT LES MÊMES SUR LES SEPT BUILDS DU CACHE (verdict du volet recherche, 2026-09-17) :
// 1 282 lancers retrouvés sur cinq films qui en publiaient ZÉRO, les quatre rangs présents, sans
// qu'aucune valeur ait à être ajoutée ici. L'entrée de profil `Grenade.TypeIDsByRank` que la
// note M3 §2.4 esquissait aurait été un ordre DEVINÉ : elle n'est pas écrite, et ne doit pas
// l'être.
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
// ELLE EST UNE CONSTANTE DU TITRE, PAS UNE ENTRÉE DE PROFIL (lot 3.3.1) : la question « les
// identifiants ont-ils changé entre builds ? » a été posée et MESURÉE, et la réponse est non.
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
//
// IL N'EXISTE PAS DE COMPTEUR `grenadeThrowsUnranked` (M3-Q5 = B), ET C'EST MESURÉ, PAS OUBLIÉ.
// La décision V17 en prévoyait un pour les lancers dont le rang ne serait pas prouvé sur les
// builds anciens. Le volet recherche a établi que l'ordre des rangs n'a jamais bougé : un lancer
// est reconnu PARCE QUE son identifiant est dans la liste ci-dessus, donc son rang est connu par
// construction et ce compteur vaudrait zéro sur tout film. Le créer serait un compteur mort
// (règle 7 du dépôt) ; la garantie est ici, en clair.
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
	// FireEvent.TimestampUS et types.KeyframeLoadout.TimestampUS.
	TimestampUS uint64
	// Chunk / PacketIndex localisent le lancer dans le film (traçabilité).
	Chunk, PacketIndex int
	// BitPos est la position du motif dans le payload, en bits (traçabilité).
	BitPos int
	// FilmIndex est l'index joueur 5 bits de l'auteur, tel que LE FILM l'écrit (0..7 sur un
	// match à 8 joueurs, 0..23 en BTB). Comme FireEvent.FilmIndex : c'est un ORDRE interne au
	// film, pas une identité. L'identité est le XUID.
	// C'est l'indexation « acurtis », à ne pas confondre avec le slot d'entité des
	// trajectoires : le pont entre les deux appartient à la couche appelante.
	FilmIndex int
	// TypeID est l'identifiant 32 bits du type de grenade.
	TypeID uint32
}

// Rank rend le RANG du type lancé (cf. GrenadeTypeIDsByRank), et s'il est connu. Le
// décodeur ne rend pas de nom : le libellé du rang appartient au titre.
func (g GrenadeThrow) Rank() (int, bool) { return GrenadeRankOf(g.TypeID) }

// grenadeGrammaire est la grammaire des lancers d'UN film : son profil d'amorce, l'archétype de
// projectile résolu dans son registre, et le motif que les deux composent.
type grenadeGrammaire struct {
	amorce   profile.AmorceGrenade
	ti       int
	motif    uint64
	tiParNom bool
}

// grenadeCouverture compte ce que le balayage a vu et ce qu'il a écarté. Elle est JOURNALISÉE et
// ne voyage pas dans l'artefact : ajouter un compteur au document de rejeu changerait sa forme,
// donc son schéma, ce qui est une décision de pilote et pas un effet de bord de ce décodeur.
type grenadeCouverture struct {
	motifs                     int
	naissancesAutresArchetypes int
	// ecartesAvecIdentifiant : parmi les naissances d'un AUTRE archétype, celles que la liste
	// blanche aurait acceptées. C'est la MESURE EXACTE de ce que la lecture du sixième bit
	// d'index coûte, et elle existe pour que « zéro perte » soit un chiffre et non une
	// déduction : avant le lot 3.3.1 ces occurrences étaient publiées comme des lancers.
	ecartesAvecIdentifiant int
	tiIndetermines         int
	publies                int
}

// ScanFilmGrenadeThrows décode tous les lancers de grenade du film de dir.
//
// HORS LIGNE (I/O disque sur tout le film) — jamais depuis un chemin de requête.
//
// ScanFilmGrenadeThrows est l'ENVELOPPE D2, HORS PRODUCTION ; la cuisson appelle
// [ScanGrenadeThrows].
func ScanFilmGrenadeThrows(dir string) ([]GrenadeThrow, error) {
	film, err := source.LoadDir(dir, nil)
	if err != nil {
		return nil, err
	}
	return ScanGrenadeThrows(NewFilmContext(film))
}

// ScanGrenadeThrows décode les lancers de grenade d'un film DEJA CHARGE.
//
// IL PREND LE CONTEXTE ET NON LE FILM DEPUIS LE LOT 3.3.1, et ce n'est pas cosmétique : la
// grammaire des lancers a besoin des CLÉS du film (build, version majeure) et de son REGISTRE
// d'archétypes, que le contexte résout et garde UNE fois par film (lot 2 de PLAN_CUISSON_PERF,
// ratchet `archlint/no_recomputed_film_context_test.go`). Les lire ici rouvrirait `chunk_00` à
// chaque balayage.
func ScanGrenadeThrows(fc *FilmContext) ([]GrenadeThrow, error) {
	g := grammaireDesLancers(fc)
	var out []GrenadeThrow
	var cov grenadeCouverture
	read := 0
	for _, c := range fc.ChunkNumbers() {
		chunk, pks, ok := fc.ChunkAt(c)
		if !ok {
			continue
		}
		read++
		for _, p := range pks {
			if p.Type != PacketTypeDelta {
				continue
			}
			for _, t := range scanGrenadeThrows(p.Payload(chunk), g, &cov) {
				t.TimestampUS, t.Chunk, t.PacketIndex = p.TimestampUS, c, p.Index
				out = append(out, t)
			}
		}
	}
	if read == 0 {
		return nil, ErrNoReadableFilmChunk
	}
	cov.publies = len(out)
	journaliserCouvertureGrenades(g, cov)
	return out, nil
}

// grammaireDesLancers résout, pour CE film, le profil d'amorce et l'archétype de projectile.
//
// AUCUNE DES DEUX RÉSOLUTIONS NE MET LE FILM DE CÔTÉ (D-4 d'ADR 0034) : une clé absente de la
// table applique le profil de RÉFÉRENCE, un registre illisible garde l'archétype de référence, et
// les deux cas sont journalisés. Le refus serait pire que le repli : il éteindrait les lancers de
// tout le parc au premier build que ce dépôt ne connaît pas encore.
func grammaireDesLancers(fc *FilmContext) grenadeGrammaire {
	g := grammaireDeReference()
	p := fc.Profile()
	if a, connue := profile.AmorceGrenadePour(p.Build(), p.Highlight().MajorVersion); connue {
		g.amorce = a
	}
	if reg, err := fc.Registry(); err == nil && reg != nil {
		if ti, vus := tiProjectileParNom(reg); vus > 0 {
			g.ti, g.tiParNom = ti, true
		}
	}
	g.motif = g.amorce.MarqueurDe(g.ti)
	return g
}

// grammaireDeReference compose la grammaire du build de RÉFÉRENCE, appliquée à un film dont la
// clé est absente de la table ou dont le registre ne se lit pas. Elle est aussi ce sur quoi les
// bancs d'essai construisent leurs records.
func grammaireDeReference() grenadeGrammaire {
	a := profile.AmorceGrenadeDeReference()
	return grenadeGrammaire{amorce: a, ti: ProjectileTypeIndex, motif: a.MarqueurDe(ProjectileTypeIndex)}
}

// composantsProjectile : les noms qui identifient l'archétype projectile dans un registre, quel
// que soit son rang (`projectiles.go`, point 1). C'est la règle « par NOM, jamais par rang » :
// l'index de l'archétype est une donnée de build, pas une constante du format.
var composantsProjectile = [...]string{
	"projectile-at-rest-state",
	"projectile-tether-state",
	"projectile-command_tick",
	"projectile-deceleration-disabled-state",
}

// tiProjectileParNom rend l'index d'archétype qui porte le plus de noms de projectile, et ce
// compte. Un bloc qui n'en porte AUCUN ne peut pas gagner : le second retour vaut alors zéro, et
// l'appelant garde l'archétype de référence plutôt que de retenir le bloc 0.
func tiProjectileParNom(reg *Registry) (int, int) {
	meilleur, vus := -1, 0
	for _, a := range reg.Archetypes {
		n := 0
		for _, nom := range composantsProjectile {
			for _, c := range a.Components {
				if c == nom {
					n++
					break
				}
			}
		}
		if n > vus {
			meilleur, vus = a.Index, n
		}
	}
	return meilleur, vus
}

// scanGrenadeThrows balaye un payload de paquet delta. PUR (aucune I/O).
func scanGrenadeThrows(pay []byte, g grenadeGrammaire, cov *grenadeCouverture) []GrenadeThrow {
	limit := len(pay)*8 - g.amorce.StructureBits()
	var out []GrenadeThrow
	for bp := 0; bp <= limit; bp++ {
		if PeekBits(pay, bp, g.amorce.Bits) != g.motif {
			continue
		}
		cov.motifs++
		switch ti, indetermine := typeIndexDuRecord(pay, bp); {
		case indetermine:
			// Le motif commence au premier bit du payload : le sixième bit d'index est HORS du
			// tampon. Il vaudrait zéro par la tolérance de PeekBits, c'est-à-dire l'archétype
			// `managed-player` une fois sur deux — on ne devine pas, on compte.
			cov.tiIndetermines++
			continue
		case ti != g.ti:
			cov.naissancesAutresArchetypes++
			if _, ok := GrenadeRankOf(uint32(PeekBits(pay, bp+g.amorce.Bits, 32))); ok {
				cov.ecartesAvecIdentifiant++
			}
			continue
		}
		id := uint32(PeekBits(pay, bp+g.amorce.Bits, 32))
		if _, ok := GrenadeRankOf(id); !ok {
			continue
		}
		out = append(out, GrenadeThrow{
			BitPos:    bp,
			FilmIndex: int(PeekBits(pay, bp+g.amorce.IndexAuteurBit, profile.AmorceGrenadeIndexBits)),
			TypeID:    id,
		})
	}
	return out
}

// typeIndexDuRecord rend le typeIndex COMPLET du record dont le motif commence à `bp`, et si son
// bit de poids fort était hors du payload.
//
// LE MOTIF NE PORTE QUE CINQ DES SIX BITS. Le typeIndex d'un record fait six bits
// (`traverse.go` : `t.TypeIndex = uint32(br.ReadBits(6))` ; `keyframe_fullstate_loop.go` :
// `kfReadBits(pay, recBit+keyframeRecordTIBit, 6)`), et le motif commence au DEUXIÈME : le bit de
// poids fort (valeur 32) est donc à `bp - 1`.
//
//	41 = 0b101001  -> bit à bp-1 : 1
//	 9 = 0b001001  -> bit à bp-1 : 0
func typeIndexDuRecord(pay []byte, bp int) (int, bool) {
	bas := int(PeekBits(pay, bp, bitsTypeIndexBasDuMotif))
	if bp < 1 {
		return bas, true
	}
	return int(PeekBits(pay, bp-1, 1))<<bitsTypeIndexBasDuMotif | bas, false
}

// bitsTypeIndexBasDuMotif : les cinq bits bas du typeIndex, en tête du motif d'amorce. Il vaut
// cinq comme la largeur du champ d'index de l'auteur, et ce n'est PAS la même grandeur : nommer
// les deux séparément évite qu'une correction de l'une déplace l'autre.
const bitsTypeIndexBasDuMotif = 5

// journaliserCouvertureGrenades publie ce que le balayage a vu. Une dégradation (clé absente de
// la table, archétype non résolu par le nom) est journalisée AVANT d'être appliquée — règle 3 du
// dépôt : jamais d'erreur avalée en silence.
func journaliserCouvertureGrenades(g grenadeGrammaire, cov grenadeCouverture) {
	ctx := context.Background()
	if !g.amorce.Connue {
		slog.WarnContext(ctx, "lancers de grenade : cle du film ABSENTE de la table d amorce — "+
			"profil de reference applique (repli_amorce_grenade_profil_de_reference)",
			"amorce_bits", g.amorce.Bits, "type_index", g.ti)
	}
	if !g.tiParNom {
		slog.WarnContext(ctx, "lancers de grenade : archetype projectile NON RESOLU par le nom "+
			"de ses composants — archetype de reference applique", "type_index", g.ti)
	}
	slog.InfoContext(ctx, "lancers de grenade : couverture du balayage",
		"amorceBits", g.amorce.Bits, "indexAuteurBit", g.amorce.IndexAuteurBit,
		"typeIndexProjectile", g.ti, "motifs", cov.motifs,
		"naissancesAutresArchetypes", cov.naissancesAutresArchetypes,
		"naissancesAutresArchetypesAvecIdentifiant", cov.ecartesAvecIdentifiant,
		"naissancesTypeIndexIndetermine", cov.tiIndetermines, "lancersPublies", cov.publies)
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
// Lecture par mot ([source.BitsAt]) des que la position de depart est positive et la
// largeur tient sur 64 bits : la primitive rend deja des zeros au-dela de la fin du tampon.
// Un depart NEGATIF ou une largeur > 64 retombent sur la boucle d'origine, seule a porter
// ces deux conventions.
func PeekBits(d []byte, bp, n int) uint64 {
	if bp >= 0 && n >= 0 && n <= 64 {
		return source.BitsAt(d, bp, uint(n))
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
