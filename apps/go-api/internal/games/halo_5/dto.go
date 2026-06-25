// Package halo_5 — dto.go : projections Go des reponses JSON internes Halo 5.
//
// Champs calques sur les shapes REELS captures par la sonde live (cmd/probe-h5,
// 2026-06-19, JGtm) et documentes dans .ai/HANDOFF_HALO5_EXPERIMENTAL.md §0-ter.
// On ne projette que les champs consommes en Phase 1 (mapping -> canonical) ; le
// carnage report etendu (scoreboard par-joueur, CSR pre/post) est Phase 2.
package halo_5

// H5MatchesResponse — GET /h5/players/{gamertag}/matches.
type H5MatchesResponse struct {
	Start       int             `json:"Start"`
	Count       int             `json:"Count"`
	ResultCount int             `json:"ResultCount"`
	Results     []H5MatchResult `json:"Results"`
}

// H5MatchResult — un match dans l'historique. Identite GAMERTAG-keyee (Xuid null).
type H5MatchResult struct {
	Id                 H5MatchID       `json:"Id"`
	HopperId           string          `json:"HopperId"`
	MapId              string          `json:"MapId"`
	GameBaseVariantId  string          `json:"GameBaseVariantId"`
	MatchDuration      string          `json:"MatchDuration"`      // ISO8601 "PT..."
	MatchCompletedDate H5ISODate       `json:"MatchCompletedDate"` // date de FIN (pas debut)
	Teams              []H5Team        `json:"Teams"`
	Players            []H5MatchPlayer `json:"Players"`
	IsTeamGame         bool            `json:"IsTeamGame"`
	SeasonId           string          `json:"SeasonId"`
}

// H5MatchID — identifiant + mode du match. GameMode 1 = arena.
type H5MatchID struct {
	MatchId  string `json:"MatchId"`
	GameMode int    `json:"GameMode"`
}

// H5ISODate — wrapper date Halo 5 ({"ISO8601Date":"2023-04-05T00:00:00Z"}).
type H5ISODate struct {
	ISO8601Date string `json:"ISO8601Date"`
}

// H5Team — score + rang d'equipe au niveau match.
type H5Team struct {
	Id    int `json:"Id"`
	Score int `json:"Score"`
	Rank  int `json:"Rank"`
}

// H5MatchPlayer — stats sommaires d'un joueur dans la liste (le detail riche est
// dans le carnage). Player.Xuid est null en Halo 5 -> indexation par Gamertag.
type H5MatchPlayer struct {
	Player       H5PlayerRef `json:"Player"`
	TeamId       int         `json:"TeamId"`
	Rank         int         `json:"Rank"`
	Result       int         `json:"Result"` // 2=win, 3=loss (cf. mapping outcomes)
	TotalKills   int         `json:"TotalKills"`
	TotalDeaths  int         `json:"TotalDeaths"`
	TotalAssists int         `json:"TotalAssists"`
}

// H5PlayerRef — reference joueur. Xuid pointer car TOUJOURS null en Halo 5.
type H5PlayerRef struct {
	Gamertag string  `json:"Gamertag"`
	Xuid     *string `json:"Xuid"`
}

// H5ServiceRecordResponse — GET /h5/servicerecords/{mode}?players={gt}.
type H5ServiceRecordResponse struct {
	Results []H5ServiceRecordResult `json:"Results"`
}

// H5ServiceRecordResult — enveloppe par joueur (Id = gamertag, ResultCode 0 = OK).
type H5ServiceRecordResult struct {
	Id         string              `json:"Id"`
	ResultCode int                 `json:"ResultCode"`
	Result     H5ServiceRecordBody `json:"Result"`
}

// H5ServiceRecordBody — corps du service record. ArenaStats present pour le mode
// arena (warzone = WarzoneStat, Phase 2).
type H5ServiceRecordBody struct {
	ArenaStats *H5ArenaStats `json:"ArenaStats"`
}

// H5ArenaStats — agregat arena : stats par playlist + pic CSR atteint.
type H5ArenaStats struct {
	ArenaPlaylistStats []H5ArenaPlaylistStat `json:"ArenaPlaylistStats"`
	HighestCsrAttained *H5Csr                `json:"HighestCsrAttained"`
}

// H5ArenaPlaylistStat — totaux d'une playlist classee (a agreger sur les buckets).
type H5ArenaPlaylistStat struct {
	PlaylistId          string `json:"PlaylistId"`
	TotalKills          int    `json:"TotalKills"`
	TotalDeaths         int    `json:"TotalDeaths"`
	TotalAssists        int    `json:"TotalAssists"`
	TotalHeadshots      int    `json:"TotalHeadshots"`
	TotalShotsFired     int    `json:"TotalShotsFired"`
	TotalShotsLanded    int    `json:"TotalShotsLanded"`
	TotalGamesCompleted int    `json:"TotalGamesCompleted"`
	TotalGamesWon       int    `json:"TotalGamesWon"`
	TotalGamesLost      int    `json:"TotalGamesLost"`
	TotalGamesTied      int    `json:"TotalGamesTied"`
	TotalTimePlayed     string `json:"TotalTimePlayed"` // ISO8601 "PT..."

	// État placement par playlist (sonde : "MeasurementMatchesLeft":7, Csr/HighestCsr
	// null pendant les matchs de mesure). MeasurementMatchesLeft > 0 = en placement.
	// Nommage Halo 5 = "Left" (vs Halo Infinite "Remaining").
	MeasurementMatchesLeft int    `json:"MeasurementMatchesLeft"`
	Csr                    *H5Csr `json:"Csr"`        // null en placement
	HighestCsr             *H5Csr `json:"HighestCsr"` // null en placement
}

// H5Csr — palier CSR natif Halo 5. DesignationId = palier majeur (0..5 :
// Bronze/Silver/Gold/Platinum/Diamond/Onyx), Tier = sous-palier (1..6 ; Onyx sans
// sous-palier). Csr = valeur brute.
type H5Csr struct {
	Tier              int `json:"Tier"`
	DesignationId     int `json:"DesignationId"`
	Csr               int `json:"Csr"`
	PercentToNextTier int `json:"PercentToNextTier"`
}

// H5Appearance — GET /h5/profiles/{gamertag}/appearance (host haloplayer).
// Shape CONFIRMÉ par la sonde live 2026-06-25 (JGtm, HTTP 200, ~729 o). On ne
// projette que les champs identitaires consommés par le Home (service tag +
// emblème). Le reste (ModelCustomization armure, StanceId, WeaponSkinIds…) n'est
// pas exposé tant qu'aucune surface UI ne le consomme (YAGNI).
type H5Appearance struct {
	Gamertag   string               `json:"Gamertag"`
	ServiceTag string               `json:"ServiceTag"`
	Emblem     H5AppearanceEmblem   `json:"Emblem"`
	Company    *H5AppearanceCompany `json:"Company"`
}

// H5AppearanceEmblem — composition de l'emblème (IDs + couleurs + harmony).
// L'image n'est PAS portée par ce JSON : elle est rendue côté serveur par
// l'endpoint /h5/profiles/{gamertag}/emblem (302 → CDN image.halocdn.com, dont
// le path est emblems/{EmblemId}_{ColorPrimary}_{ColorSecondary}_{ColorTertiary}
// + un hash signé non reproductible côté client). Cf. GetEmblemPNG.
type H5AppearanceEmblem struct {
	ColorPrimary      int `json:"ColorPrimary"`
	ColorSecondary    int `json:"ColorSecondary"`
	ColorTertiary     int `json:"ColorTertiary"`
	EmblemId          int `json:"EmblemId"`
	HarmonyGroupIndex int `json:"HarmonyGroupIndex"`
	HarmonyIndex      int `json:"HarmonyIndex"`
}

// H5AppearanceCompany — compagnie (clan) du joueur, optionnelle.
type H5AppearanceCompany struct {
	Id   string `json:"Id"`
	Name string `json:"Name"`
}
