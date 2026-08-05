package killcollector

// shots.go — LE PRODUCTEUR DE `shared.match_weapon_shots` : la ventilation des tirs par arme,
// produite par LA MEME PASSE DE DECODAGE que les morts.
//
// # POURQUOI DANS LA MEME PASSE, ET PAS DANS UNE AUTRE
//
// Le decodage d un film est LA passe chere du chantier (de 1 s a 50 s par film, 949 films en
// cache). Un producteur separe re-lirait, re-decompresserait et re-scannerait les memes chunks :
// le guide mesure la passe de tirs a 1,65 s CPU par film en autonome contre ~0,5 s greffee sur un
// decodage qui tient deja les chunks. Le vrai argument n est pourtant pas la : c est qu un film
// decode DEUX FOIS a DEUX occasions de sortir deux etats differents de la meme base.
//
// # CE QUE CE FICHIER NE FAIT PAS, ET NE FERA JAMAIS
//
//	AUCUN TAUX PAR ARME.       La table STOCKE ; elle ne publie pas. Le taux par arme calcule sur
//	                           le corpus entier INVERSE l ordre MA40 / Sidekick par rapport a la
//	                           reference de l API — ce n est pas une imprecision, c est une
//	                           reponse fausse (GUIDE_WEAPON_SHOTS §3bis.0). Quatre armes seulement
//	                           sont publiables, et sur une population restreinte.
//	AUCUN VERDICT DE PORTE.    `EvaluateShotsGate` vit dans le persister et nulle part ailleurs.
//	                           Ici on fournit la REFERENCE (`shots_fired` de l API), jamais le
//	                           verdict : c est ce qui rend impossible un `publishable = true`
//	                           publie sans reference.
//	AUCUNE TOUCHE.             `HitLikely` du scanner est MORT (il annonce 75-79 % de touches pour
//	                           une precision reelle de mediane 0,446). La table n a pas la colonne.
//	AUCUN COEFFICIENT.         Un fire-event est un tir, k = 1. Un tableau de coefficients par arme
//	                           appris sur des films disjoints fait DEUX FOIS PIRE que k = 1.

import (
	"context"
	"encoding/binary"
	"log/slog"
	"sort"
	"strconv"

	"levelup/go-api/internal/analysis"
	"levelup/go-api/internal/analysis/weaponv3"
	"levelup/go-api/internal/observability"
	"levelup/go-api/internal/persist"
)

// WeaponShotsDecoderRev — la version du producteur de tirs, ecrite sur CHAQUE ligne.
//
// Elle est DISTINCTE de [KillSourceDecoderRev] bien que les deux passes soient simultanees :
// les deux lectures n ont ni le meme code, ni le meme espace d identifiants (filmshell 64 bits
// ici, tag jpt! 32 bits la-bas), et un changement de l une ne demande pas de redecoder l autre.
// Les confondre couterait un redecodage complet a chaque changement de l un des deux.
const WeaponShotsDecoderRev = "filmshots-2026-08-01"

// shotsMetric* : les compteurs de sante de la passe de tirs (ADR 0009 — entiers, snake_case).
const (
	metricShotsMatches   = "killsource_tirs_matchs_ventiles"
	metricShotsRows      = "killsource_tirs_lignes_ecrites"
	metricShotsNoIndex   = "killsource_tirs_indices_non_resolus"
	metricShotsWriteFail = "killsource_tirs_erreurs_ecriture"
)

// zeroEstimator : la passe de tirs N A PAS BESOIN D HORODATAGE.
//
// `ScanFireEventsB5` exige un estimateur parce que son autre consommateur (la correlation
// kill -> arme) apparie dans le TEMPS. Ici on COMPTE par (joueur, arme) : ni la population
// scannee, ni la deduplication (qui porte sur la position en octets) ne dependent de l instant.
// Lui passer un estimateur constant evite le balayage des marqueurs de frame — pour un resultat
// identique au bit pres sur ce que la table stocke.
func zeroEstimator(int) float64 { return 0 }

// BuildWeaponShotsBatch : la ventilation d un film, prete a ecrire.
//
// EXPORTEE pour la meme raison que [BuildKillSourceBatch] : le backfill doit emprunter
// EXACTEMENT ce chemin. Une seconde traduction serait une seconde chance de se tromper d indice.
//
// `chunks` sont les chunks REPLICATION_DATA DECOMPRESSES ; `shotsFired` la reference de l API
// par xuid (absente = la porte refusera, elle ne suppose pas).
func BuildWeaponShotsBatch(
	matchID string, chunks [][]byte, rosterXUIDs []string, shotsFired map[string]int,
) persist.WeaponShotsBatch {
	piToXUID := resolvePlayerIndices(rosterXUIDs, chunks)

	// Comptage (indice de replication x arme). L indice est celui du FILM, jamais un rang de
	// base : c est la seule quantite qui ne depende d aucune resolution.
	counts := map[int]map[uint64]int{}
	for _, data := range chunks {
		for _, ev := range analysis.ScanFireEventsB5(data, zeroEstimator) {
			pi := ev.PlayerIndex5
			if pi < 0 || pi > maxReplicationIndex {
				continue
			}
			id := binary.BigEndian.Uint64(ev.WeaponBytes[:])
			// Les sentinelles grenade/melee/vehicule d `analysis` ne sont PAS des identifiants
			// filmshell : les ecrire fabriquerait une jointure fausse avec
			// `metadata.weapon_labels`. Le persister les refuse — mais il refuse la PASSE
			// ENTIERE, alors qu ici une sentinelle isolee ne doit couter que sa propre ligne.
			// On lit la liste chez son proprietaire plutot que d en recopier la borne.
			if analysis.SentinelIDs[id] {
				continue
			}
			if counts[pi] == nil {
				counts[pi] = map[uint64]int{}
			}
			counts[pi][id]++
		}
	}

	return assembleShotsBatch(matchID, counts, piToXUID, shotsFired)
}

// maxReplicationIndex : la borne du champ 5 bits. Au-dela ce ne serait pas une lecture du champ.
const maxReplicationIndex = 31

// resolvePlayerIndices : `indice de replication -> xuid`, LU DANS LE FILM.
//
// LE POINT LE PLUS FACILE A RATER DE TOUT CE FICHIER. L indice de replication n est PAS l ordre
// des participants en base : cette derivation (`getXuidToPI`) est INDISCERNABLE d une permutation
// tiree au sort — elle designe le bon joueur 11,769 % du temps contre 11,137 % attendus au pur
// hasard (949 films). La resolution juste cherche le motif du xuid AU BIT PRES dans le flux et
// lit les 5 bits qui le precedent ; contre l oracle `killsource`, elle rend 77,0 % d accord
// contre 22,8 % pour l ordre de la base (239 films, 16 411 kills, McNemar z = 92,8).
//
// Un xuid non resolu n a PAS de ligne : mieux vaut un joueur absent de la table qu un joueur
// dont les tirs sont attribues a un autre.
func resolvePlayerIndices(rosterXUIDs []string, chunks [][]byte) map[int]string {
	motifs := make(map[uint64]string, len(rosterXUIDs))
	for _, s := range rosterXUIDs {
		v, err := strconv.ParseUint(s, 10, 64)
		if err != nil || v == 0 {
			continue // bot (`bid(...)`) ou xuid non numerique : pas de motif a chercher
		}
		motifs[motifDuXUID(v)] = s
	}
	out := map[int]string{}
	for x, pi := range chercherMotifs(motifs, chunks) {
		if pi < 0 || pi > maxReplicationIndex {
			continue
		}
		// Deux xuids sur le meme indice : on n en garde AUCUN. Trancher au hasard publierait
		// les tirs d un joueur sous le nom d un autre, ce qui est pire que de n en publier aucun.
		if _, deja := out[pi]; deja {
			out[pi] = ""
			continue
		}
		out[pi] = x
	}
	return out
}

// motifDuXUID : le xuid encode en 8 octets LITTLE-ENDIAN puis relu en BIG-ENDIAN. C est sous
// cette forme qu il apparait dans le flux de replication (methode `weaponv3.ResolveXuidToPI`).
func motifDuXUID(xuid uint64) uint64 {
	var le [8]byte
	binary.LittleEndian.PutUint64(le[:], xuid)
	return binary.BigEndian.Uint64(le[:])
}

// chercherMotifs : LA MEME RECHERCHE QUE `weaponv3.ResolveBest`, EN UNE SEULE PASSE.
//
// POURQUOI ELLE EXISTE — LA VERSION NAIVE REND LE BACKFILL IMPRATICABLE. `ResolveBest` balaie
// le film UNE FOIS PAR XUID, et chaque position y coute une relecture de 64 bits : sur un roster
// de 10 joueurs, c est 640 lectures de bit par position de bit du film. Mesure du 2026-08-01 :
// 24 s par film sur des films de 8 a 16 chunks, la ou le decodage des morts en coute 1 a 3 —
// la resolution des indices coutait DIX FOIS le decodage qu elle accompagne.
//
// Ici, une fenetre glissante de 64 bits avance d un bit a la fois (aucune relecture) et un
// PREFILTRE de 16 bits ecarte la quasi-totalite des positions avant la moindre consultation de
// table. La recherche s arrete des que tous les xuids sont resolus.
//
// L EQUIVALENCE EST EXACTE, ET ELLE EST TESTEE : chunks dans l ordre, positions croissantes,
// premiere occurrence gagnante — les trois proprietes de `ResolveBest`. Le test
// `TestRechercheDeMotifsEquivautALaVersionNaive` confronte les deux implementations.
func chercherMotifs(motifs map[uint64]string, chunks [][]byte) map[string]int {
	out := make(map[string]int, len(motifs))
	if len(motifs) == 0 {
		return out
	}
	// Prefiltre : les 16 bits de poids fort de chaque motif. Une position dont les 16 premiers
	// bits ne sont ceux d aucun motif ne peut pas etre un motif — le test coute un acces tableau.
	var prefiltre [1 << 16]bool
	for m := range motifs {
		prefiltre[m>>48] = true
	}

	for _, data := range chunks {
		if len(out) == len(motifs) {
			break // tout est resolu : le reste du film n apprendrait rien
		}
		chercherDansChunk(motifs, &prefiltre, data, out)
	}
	return out
}

// chercherDansChunk : la fenetre glissante sur UN chunk.
//
// `pos` designe le bit qui vient d entrer dans la fenetre ; le motif commence donc a
// `pos-63`, et l indice se lit sur les 5 bits qui PRECEDENT — d ou la garde `pos >= 68`.
func chercherDansChunk(
	motifs map[uint64]string, prefiltre *[1 << 16]bool, data []byte, out map[string]int,
) {
	var fenetre uint64
	total := len(data) * 8
	for pos := 0; pos < total; pos++ {
		fenetre = fenetre<<1 | uint64((data[pos>>3]>>uint(7-(pos&7)))&1)
		if pos < 63 || !prefiltre[fenetre>>48] {
			continue
		}
		xuid, ok := motifs[fenetre]
		if !ok {
			continue
		}
		if _, deja := out[xuid]; deja {
			continue // premiere occurrence gagnante, comme `ResolveBest`
		}
		debut := pos - 63
		if debut < weaponv3.PIBits {
			continue // pas assez de bits AVANT le motif pour porter un indice
		}
		out[xuid] = lireIndiceAvant(data, debut)
		if len(out) == len(motifs) {
			return
		}
	}
}

// lireIndiceAvant : les 5 bits qui precedent immediatement le motif, MSB-first.
func lireIndiceAvant(data []byte, debutMotif int) int {
	v := 0
	for i := debutMotif - weaponv3.PIBits; i < debutMotif; i++ {
		v = v<<1 | int((data[i>>3]>>uint(7-(i&7)))&1)
	}
	return v
}

// assembleShotsBatch : les comptes deviennent des lignes, dans un ordre STABLE.
//
// L ordre est celui de l indice de replication puis de l identifiant d arme — pas celui d une
// map. Une passe doit etre reproductible : deux executions sur le meme film ecrivent les memes
// lignes dans le meme ordre, sinon un diff de controle ne veut rien dire.
func assembleShotsBatch(
	matchID string, counts map[int]map[uint64]int, piToXUID map[int]string, shotsFired map[string]int,
) persist.WeaponShotsBatch {
	batch := persist.WeaponShotsBatch{MatchID: matchID, DecoderRev: WeaponShotsDecoderRev}
	for pi := 0; pi <= maxReplicationIndex; pi++ {
		byWeapon := counts[pi]
		if len(byWeapon) == 0 {
			continue
		}
		pl := persist.WeaponShotsPlayer{PlayerIndex: pi, XUID: piToXUID[pi]}
		if pl.XUID != "" {
			if n, ok := shotsFired[pl.XUID]; ok {
				// La reference est un POINTEUR : nil = « pas de reference », et c est different
				// de 0 = « l API dit zero tir ». Le second est un verdict, le premier une absence.
				ref := n
				pl.ShotsFired = &ref
			}
		}
		for _, id := range sortedWeaponIDs(byWeapon) {
			pl.Weapons = append(pl.Weapons, persist.WeaponShotCount{WeaponID: id, Shots: byWeapon[id]})
		}
		batch.Players = append(batch.Players, pl)
	}
	return batch
}

// sortedWeaponIDs : les identifiants d armes d un joueur, en ordre croissant.
func sortedWeaponIDs(byWeapon map[uint64]int) []uint64 {
	ids := make([]uint64, 0, len(byWeapon))
	for id := range byWeapon {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}

// publishShotsPass : les compteurs de sante (ADR 0009) et la trace de la passe.
//
// `killsource_tirs_indices_non_resolus` est celui qui informe : un joueur dont l indice n a pas
// ete trouve dans le flux n a AUCUNE ligne, et sans ce compteur son absence serait
// indistinguable d un joueur qui n a pas tire (GUIDE_WEAPON_SHOTS §3.2 — un echec total est
// silencieux DANS la table ; il ne doit pas l etre dans la telemetrie).
func publishShotsPass(ctx context.Context, batch persist.WeaponShotsBatch, rosterSize int) {
	rows, named, sansReference := 0, 0, 0
	for i := range batch.Players {
		rows += len(batch.Players[i].Weapons)
		if batch.Players[i].XUID != "" {
			named++
		}
		if batch.Players[i].ShotsFired == nil {
			sansReference++
		}
	}
	observability.AddInt(metricShotsMatches, 1)
	observability.AddInt(metricShotsRows, int64(rows))
	if manquants := rosterSize - named; manquants > 0 {
		observability.AddInt(metricShotsNoIndex, int64(manquants))
	}
	slog.InfoContext(ctx, "killsource: ventilation des tirs",
		"match_id", batch.MatchID, "joueurs_ventiles", len(batch.Players), "lignes", rows,
		"roster", rosterSize, "sans_reference_api", sansReference)
}
