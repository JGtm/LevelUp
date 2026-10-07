package grammar

// marche_de_killsource.go — CE QUE LA MARCHE DES TRAMES LIT D UN FILM POUR `killsource` : ses morts
// et ses messages de kill, avec leur position, en une seule marche.
//
// Les morts : le canal des morts ([canalDesMorts]) recolte les records de la vue B que la marche lit,
// et ceux des listes qu elle ne localise pas, qu il recupere (signature, puis largeur libre). Sur
// demande, il garde chaque dead-state `Mort` avec sa trame, la position de son composant et la
// qualite de sa lecture ; killsource y applique sa regle (records entierement portes, archetype
// bipede) et son filtre de credibilite.
//
// Les kills : le canal des kills ([canalDesKills]) recueille les messages de kill que la vue A de
// chaque trame porte, et ceux que le rattrapage retrouve dans les trames ou la lecture de la vue A
// n est pas etablie, avant la vue B (`kills_rattrapes.go`).

import (
	"levelup/go-api/internal/games/halo_infinite/film/internal/grammar/lecture"
	"levelup/go-api/internal/games/halo_infinite/film/types"
)

// EtatDeMortLu est un dead-state `Mort` qu un record lu porte.
type EtatDeMortLu struct {
	// PositionDuChunk et Index situent la trame : la position de son chunk dans la source (celle de
	// `types.Packet.Chunk`), son rang dans le chunk ; TS est son horodatage.
	PositionDuChunk, Index int
	TS                     uint64
	// Slot, Gen et TypeIndex identifient l entite et son archetype, celui que la marche lie.
	Slot, Gen, TypeIndex uint32
	// Bit est le premier bit du dead-state dans le payload, -1 s il n est pas dans la trace.
	Bit int
	// Propre dit que le record est entierement porte (aucune rupture) ; Recupere, que sa liste n a
	// pas ete localisee par la marche et que le canal l a recuperee.
	Propre, Recupere bool
	// Dead est le dead-state lu.
	Dead types.DeadState
}

// KillLu est un message de kill d une trame delta, avec la position de la trame.
type KillLu struct {
	// PositionDuChunk et Index situent la trame, comme pour [EtatDeMortLu] ; TS est son horodatage.
	PositionDuChunk, Index int
	TS                     uint64
	// Kill est le message : sa position (le bit de sa continuation) et ses champs.
	Kill lecture.MessageDeKill
	// Rattrape dit que la vue A ne l a pas lu et que le rattrapage l a retrouve ; Chaine est alors la
	// longueur de la chaine d evenements qui l a valide.
	Rattrape bool
	Chaine   int
}

// BilanDuRattrapage dit ce que le rattrapage des kills a fait sur un film : les messages retenus, les
// chaines arretees sur un code non modelise, et `gate15` — Tranche faux quand aucune trame ne l a
// demande.
type BilanDuRattrapage struct {
	Kills, ChainesArretees int
	Gate15, Tranche        bool
}

// LectureDeKillsource est ce que la marche des trames lit d un film pour `killsource`.
type LectureDeKillsource struct {
	// Morts sont les dead-states `Mort` des records lus et recuperes ; Kills, les messages de kill ;
	// les deux dans l ordre du flux.
	Morts []EtatDeMortLu
	Kills []KillLu
	// Stats sont les denominateurs du canal des morts ; LargeurLibre, les listes recuperees par le
	// repli a largeur libre (`repli_localisation_largeur_libre`).
	Stats        ObjectDeathStats
	LargeurLibre int
	// Rattrapage : ce que le rattrapage des kills a fait.
	Rattrapage BilanDuRattrapage
}

// LireLaMarcheDeKillsource distribue le canal des morts et le canal des kills sur la marche des
// trames du film, en une marche. La cuisson lit les morts par [ScanMarcheDesTramesAvec].
func LireLaMarcheDeKillsource(fc *FilmContext) (LectureDeKillsource, error) {
	reg, err := fc.Registry()
	if err != nil {
		return LectureDeKillsource{}, err
	}
	morts, kills := nouveauCanalDesMorts(reg), nouveauCanalDesKills(fc)
	morts.garder = true
	if err := Distribuer(fc, morts, kills); err != nil {
		return LectureDeKillsource{}, err
	}
	r := &kills.rattrapage
	return LectureDeKillsource{Morts: morts.mortsLues, Kills: kills.kills, Stats: morts.st,
		LargeurLibre: morts.largeurLibre, Rattrapage: BilanDuRattrapage{Kills: kills.rattrapes,
			ChainesArretees: r.chainesArretees, Gate15: r.gate15, Tranche: r.tranche}}, nil
}

// canalDesKills recueille les messages de kill des trames delta d un film.
type canalDesKills struct {
	fc         *FilmContext
	kills      []KillLu
	rattrapage rattrapageDesKills
	rattrapes  int
}

// nouveauCanalDesKills prepare le canal des kills du film de `fc`.
func nouveauCanalDesKills(fc *FilmContext) *canalDesKills {
	return &canalDesKills{fc: fc, rattrapage: rattrapageDesKills{film: fc.Film()}}
}

// Interets : le canal ne lit que la vue A rangee de chaque trame et le debut de sa vue B.
func (c *canalDesKills) Interets() []Interet { return nil }

// Clore : rien — le canal se lit par ses champs.
func (c *canalDesKills) Clore(BilanDeMarche) {}

// Tete recueille les messages de kill de la trame `p`, marchee (la marche des morts l accompagne) :
// ceux de sa vue A, puis ceux du rattrapage.
func (c *canalDesKills) Tete(p *lecture.Paquet) {
	pos := filmChunkPos(c.fc.Film(), p.Chunk)
	for _, k := range p.VueA.Kills {
		c.kills = append(c.kills, KillLu{PositionDuChunk: pos, Index: p.Index, TS: p.TS, Kill: k})
	}
	for _, k := range c.rattrapage.rattraper(p) {
		c.kills = append(c.kills, KillLu{PositionDuChunk: pos, Index: p.Index, TS: p.TS, Kill: k.Kill,
			Rattrape: true, Chaine: k.Chaine})
		c.rattrapes++
	}
}
