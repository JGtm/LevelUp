package port

// data_health.go — ce que les tâches de fond du scheduler lisent dans les bases : les sondes du contrôle de
// santé des données (compteurs d'anomalies, trous LUSR, résumés du journal des morts, jauge des passes) et
// les dates des matchs pour la purge des artefacts de rejeu. Aucune de ces lectures n'écrit.

import (
	"context"
	"errors"
	"io"
	"time"

	"levelup/go-api/internal/migration"
)

// ErrDataHealthUnavailable : une base à sonder n'a pas pu être ouverte (absente, ou tenue en écriture par un
// autre détenteur). L'appelant décide si la sonde est sautée ou comptée non mesurée ; ce n'est pas un échec
// de requête.
var ErrDataHealthUnavailable = errors.New("data health: base inouvrable")

// HealthProbe nomme une sonde de comptage ; sa valeur sert d'étiquette dans les journaux.
type HealthProbe string

// Sondes de la base partagée d'un titre, puis de la base d'un joueur.
const (
	// ProbeUUIDsMapName et ProbeUUIDsPairName : noms de carte ou de paire restés à l'état d'UUID brut.
	ProbeUUIDsMapName  HealthProbe = "uuids_map_name"
	ProbeUUIDsPairName HealthProbe = "uuids_pair_name"
	// ProbeLyingBitsEvents et ProbeLyingBitsWeapons : matchs dont le bit « événements » ou « détail des
	// armes » est posé sans donnée derrière. La seconde ne s'applique que si la base contient la table qui
	// porte la preuve des armes.
	ProbeLyingBitsEvents  HealthProbe = "lying_bits_events"
	ProbeLyingBitsWeapons HealthProbe = "lying_bits_weapons"
	// ProbeOrphanXUIDs : joueurs humains sans alias.
	ProbeOrphanXUIDs HealthProbe = "orphan_xuids"
	// ProbeGarbageBannerURLs : URLs d'images de profil résiduelles d'un ancien chemin (base du joueur).
	ProbeGarbageBannerURLs HealthProbe = "garbage_banner_urls"
)

// LUSRGaps : les trous LUSR d'un joueur.
type LUSRGaps struct {
	// InteriorGaps : matchs éligibles non notés sous le watermark (notes définitivement absentes).
	InteriorGaps int
	// PendingRecent : matchs éligibles non notés au-dessus du watermark (simple attente).
	PendingRecent int
}

// DataHealthShared est la base partagée d'un titre ouverte en lecture seule pour un audit ; la fermer rend
// le handle (compté par référence).
type DataHealthShared interface {
	io.Closer
	// CountProbe exécute une sonde de la base partagée. applicable est faux quand la sonde ne s'applique pas à
	// cette base : n vaut alors 0, sans erreur.
	CountProbe(ctx context.Context, probe HealthProbe) (n int, applicable bool, err error)
	// PendingKillSummaries compte les matchs dont la passe courante du journal des morts n'a pas de résumés à
	// jour (ADR 0039).
	PendingKillSummaries(ctx context.Context) (int, error)
	// ScanLUSRGaps croise les matchs de la base partagée et les notes LUSR de la base du joueur. Une base du
	// joueur inouvrable rend ErrDataHealthUnavailable ; toute autre erreur est un échec de la sonde.
	ScanLUSRGaps(ctx context.Context, playerDBPath, xuid string) (LUSRGaps, error)
}

// DataHealthStore ouvre les bases de l'audit de santé des données.
type DataHealthStore interface {
	// OpenShared ouvre la base partagée du chemin donné en lecture seule.
	OpenShared(path string) (DataHealthShared, error)
	// CountPlayerProbe exécute une sonde de la base d'un joueur (ProbeGarbageBannerURLs). Une base inouvrable
	// rend ErrDataHealthUnavailable.
	CountPlayerProbe(ctx context.Context, probe HealthProbe, playerDBPath string) (int, error)
	// MeasurePasses mesure la jauge « passes par match » d'une table de la base partagée du titre, par une
	// lecture courte. present est faux quand la table n'existe pas dans la base.
	MeasurePasses(ctx context.Context, titleSlug, sharedDBPath, table string) (gauge migration.JaugePasses, present bool, err error)
}

// MatchDatesReader lit les dates des matchs du registre partagé.
type MatchDatesReader interface {
	// MatchDatesByShortID rend, par forme courte d'identifiant de match, son début canonique. Le registre est
	// lu en entier ; la base partagée peut être tenue par le serveur, son handle est réutilisé.
	MatchDatesByShortID(ctx context.Context, sharedDBPath string) (map[string]time.Time, error)
}
