package replay

// film_scan_ti13.go — LE BALAYAGE PARTAGE DE `ti=13` (zones et jauge de retour du drapeau).
//
// SORTI DE `film_scan.go` le 2026-09-24 (integration de la vague D des retours du rejeu) : les
// lots M2, M3 et la vague C y avaient chacun ajoute des lignes, et le fichier passait a 512 lignes
// (regle 5 de CLAUDE.md). DEPLACEMENT PUR : aucune ligne de logique ne change. La methode est
// appelee par `balayerMonde` (cf. `film_scan.go`).

// balayerProprietesTi13 sert les DEUX consommateurs de `ti=13` — l etat des zones et la jauge de
// retour du drapeau — a partir d UNE SEULE lecture du film.
//
// POURQUOI UNE SEULE LECTURE : le balayage est une marche bit a bit de tous les paquets delta —
// le meme ordre de grandeur que celui des positions. Le payer deux fois sur un match qui
// declencherait les deux gardes serait doubler la cuisson pour les memes octets. `ti13Partage` le
// garantit : le premier appelant lit, le second recoit.
//
// POURQUOI DEUX ETAPES OBSERVEES MALGRE TOUT : ce sont DEUX ENTREES du document, gardees par deux
// modes differents et consommees par deux calques. `ZoneScanned` et `GaugeScanned` sont publies,
// et ils disent « ce calque a ete LU », pas « ces octets ont ete lus » — un CTF qui heriterait de
// `ZoneScanned = true` ferait mentir `coverage.zones`.
func (s *filmScan) balayerProprietesTi13() {
	// LES MEMES PREDICATS QUE [GardesDe] (lot J3.4) : les faits disent exactement ce qui a ete lu.
	zones := zonesBalayables(s.opt.Zone)
	jauge := drapeauBalayable(s.opt.Flag, s.opt.consultations())
	partage := ti13Partage{fc: s.fc, matchID: s.matchID}
	s.in.ZoneReads, s.in.ZoneKeyReads = decodeFilmZoneReads(s.ctx, &partage, zones)
	s.in.ZoneScanned = zones
	s.opt.observe(s.ctx, "zoneReads", s.in.ZoneReads)
	s.opt.observe(s.ctx, "zoneKeyReads", s.in.ZoneKeyReads)
	s.in.FlagGauge = decodeFilmFlagReturnGauge(s.ctx, &partage, jauge)
	s.in.FlagGaugeScanned = jauge
	s.opt.observe(s.ctx, "flagGauge", s.in.FlagGauge)
}
