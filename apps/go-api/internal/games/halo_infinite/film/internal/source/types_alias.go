package source

// types_alias.go — LES ALIAS DATES DES TYPES DE CONTRAT DE CETTE COUCHE (lot 2.6.2, 2026-09-16).
//
// # POURQUOI DES ALIAS, ALORS QUE LE DEPOT INTERDIT LE CODE MORT
//
// CLAUDE.md regle 7 : ce qu on debranche, on le supprime. Ces alias ne sont PAS debranches — ils
// sont la SEULE facon de faire le deplacement en gardant la frontiere de fichiers du lot. Les
// declarations ont descendu dans `film/types` ; leurs consommateurs vivent pour partie dans
// `film/grammar`, `film/replay` et `replaybuild`, que d autres executeurs tiennent en ce moment
// meme (lots 2.5.b et suivants). Les re-pointer ici aurait voulu dire ecrire dans des fichiers
// qu un autre est en train de deplacer.
//
// UN ALIAS DE TYPE N EST PAS UNE COPIE : pour le compilateur, `source.Packet` et `types.Packet`
// sont LE MEME type. Rien ne peut diverger entre les deux, et c est ce qui rend le sursis sans
// risque.
//
// RETRAIT : volet grammaire / rejeu du lot 2.6, apres 2.5.e — le commit qui re-pointe les
// consommateurs supprime ce fichier. Critere mesurable : plus aucun `source.ChunkMeta` ni
// `source.Packet` hors de ce paquet (`grep -rn 'source\.\(ChunkMeta\|Packet\)' internal cmd`).

import "levelup/go-api/internal/games/halo_infinite/film/types"

type (
	// ChunkMeta : voir [types.ChunkMeta].
	ChunkMeta = types.ChunkMeta
	// Packet : voir [types.Packet].
	Packet = types.Packet
)
