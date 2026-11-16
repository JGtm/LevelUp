package replaydoc

// hits.go — LES TOUCHES ET LE BILAN DE TIRS PAR JOUEUR, LUS DANS LE FILM : UNE ESTIMATION. Jumeau de
// `replay.Hit`, `replay.ShotStat` et `replay.HitCoverage`.
//
// Le film n'ecrit pas le nombre de touches que l'API compte : ce sont des estimations, et
// `coverage.hits.estimated` vaut toujours vrai. Les tirs, touches et degats d'un joueur ne se
// montrent que sous `shotStats[].sufficient` : faux, le consommateur n'affiche aucun chiffre.

// Hit est UNE touche estimee : la frame, le tireur, le joueur touche (xuid decimal), l'effet de degat
// (huit chiffres hexadecimaux) et l'arme du dernier tir du tireur dans les deux secondes qui precedent
// (cle de metadata.weapon_labels.weapon_id ; absente quand le tireur n'a pas tire a ce moment).
type Hit struct {
	T       int    `json:"t"`
	Shooter string `json:"shooter"`
	Victim  string `json:"victim"`
	Effect  string `json:"effect"`
	Weapon  string `json:"w,omitempty"`
}

// ShotStat est le bilan de tirs d'un joueur sur tout le film et le verdict de suffisance de ses donnees.
// `shots` vient de la numerotation des tirs du jeu, `shotEvents` des evenements de tir ecrits ;
// `coverage` vaut shotEvents / shots et `duplicates` messages / hits. `damageDealt` et `damageTaken`
// couvrent TOUT le film, alors que `damage[]` ne porte que l'axe. `sufficient` faux : aucun chiffre de tir,
// de touche ni de degat ne s'affiche pour ce joueur.
type ShotStat struct {
	XUID        string  `json:"xuid"`
	Shots       int     `json:"shots"`
	ShotEvents  int     `json:"shotEvents"`
	Messages    int     `json:"messages"`
	Hits        int     `json:"hits"`
	DamageDealt float64 `json:"damageDealt"`
	DamageTaken float64 `json:"damageTaken"`
	Coverage    float64 `json:"coverage"`
	Duplicates  float64 `json:"duplicates"`
	Sufficient  bool    `json:"sufficient"`
}

// HitCoverage dit ce que le calque des touches a lu, pose et ecarte. `minCoverage`, `maxDuplicates` et
// `maxStoppedShare` sont les seuils de suffisance appliques aux joueurs ; `frames`, `stoppedFrames` et
// `vitalsUnread` la part de lecture ; `published` les touches posees sur l'axe, `noShooter`, `noVictim`,
// `beforeOrigin` et `afterEnd` celles qui ne le sont pas ; `selfHits` et `selfDamages` les touches et
// degats qu'un joueur se fait a lui-meme, comptes a part.
type HitCoverage struct {
	Estimated       bool    `json:"estimated"`
	MinCoverage     float64 `json:"minCoverage"`
	MaxDuplicates   float64 `json:"maxDuplicates"`
	Frames          int     `json:"frames"`
	StoppedFrames   int     `json:"stoppedFrames"`
	VitalsUnread    int     `json:"vitalsUnread"`
	MaxStoppedShare float64 `json:"maxStoppedShare"`
	Messages        int     `json:"messages"`
	WeaponMessages  int     `json:"weaponMessages"`
	Hits            int     `json:"hits"`
	Published       int     `json:"published"`
	NoShooter       int     `json:"noShooter"`
	NoVictim        int     `json:"noVictim"`
	BeforeOrigin    int     `json:"beforeOrigin"`
	AfterEnd        int     `json:"afterEnd"`
	Sufficient      int     `json:"sufficient"`
	Damages         int     `json:"damages"`
	DamagesMeasured int     `json:"damagesMeasured"`
	DamagesOffUnit  int     `json:"damagesOffUnit"`
	SelfHits        int     `json:"selfHits"`
	SelfDamages     int     `json:"selfDamages"`
	Players         int     `json:"players"`
}
