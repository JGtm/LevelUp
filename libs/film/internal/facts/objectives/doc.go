// Package objectives — les timelines d'evenements d'objectif (CTF captures, Strongholds/KOTH zones,
// Oddball crane) et ce que les enregistrements d'entite du statborg veulent dire (manches, score,
// identite par slot), vers des []objectiveevent.Event (mode-agnostique, cf. plan
// PLAN_WEAPON_ATTRIBUTION_V3 §10, retire du depot le 2026-07-13 par ca6b1864a).
//
// Algos PURS : zero acces DB, et AUCUNE LECTURE D'OCTET (ADR 0037, D-2 amende). Les lectures du
// film vivent dans la grammaire — le statborg (`signaux.LireLeStatborg`), le pied de film
// (`signaux.FooterEvents`), les rafales de capture (`signaux.CaptureBurstTimes`) — et les points
// d'entree de ce paquet qui recoivent un `*source.Film` DEJA CHARGE le leur passent tel quel. La
// resolution xuid->team reste fournie par l'appelant (Roster, issu de match_participants), en
// controle de l'equipe que le pied ecrit.
package objectives
