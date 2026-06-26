package analysis

import (
	"testing"

	"levelup/go-api/internal/games/canonical"
)

// kev construit un event kill/death canonique pour le joueur xuid au timestamp t.
func kev(t int64, eventType canonical.HighlightEventType, xuid string) canonical.HighlightEvent {
	return canonical.HighlightEvent{EventType: string(eventType), TimeMS: t, XUID: xuid}
}

func TestComputeMaxKillingSpree(t *testing.T) {
	t.Parallel()
	const me = "xuid_me"
	const other = "xuid_other"

	cases := []struct {
		name   string
		events []canonical.HighlightEvent
		xuid   string
		want   int
	}{
		{
			name:   "aucun event",
			events: nil,
			xuid:   me,
			want:   0,
		},
		{
			name: "3 kills puis death puis 2 kills => max 3",
			events: []canonical.HighlightEvent{
				kev(1000, canonical.EventKill, me),
				kev(2000, canonical.EventKill, me),
				kev(3000, canonical.EventKill, me),
				kev(4000, canonical.EventDeath, me),
				kev(5000, canonical.EventKill, me),
				kev(6000, canonical.EventKill, me),
			},
			xuid: me,
			want: 3,
		},
		{
			name: "spree finale plus longue => max 2 puis 4",
			events: []canonical.HighlightEvent{
				kev(1000, canonical.EventKill, me),
				kev(2000, canonical.EventKill, me),
				kev(3000, canonical.EventDeath, me),
				kev(4000, canonical.EventKill, me),
				kev(5000, canonical.EventKill, me),
				kev(6000, canonical.EventKill, me),
				kev(7000, canonical.EventKill, me),
			},
			xuid: me,
			want: 4,
		},
		{
			name: "events non tries (ordre TimeMS melange) => tri interne, max 3",
			events: []canonical.HighlightEvent{
				kev(4000, canonical.EventDeath, me),
				kev(2000, canonical.EventKill, me),
				kev(6000, canonical.EventKill, me),
				kev(1000, canonical.EventKill, me),
				kev(3000, canonical.EventKill, me),
				kev(5000, canonical.EventKill, me),
			},
			xuid: me,
			want: 3,
		},
		{
			name: "kills d'un autre joueur ignores (XUID different)",
			events: []canonical.HighlightEvent{
				kev(1000, canonical.EventKill, other),
				kev(2000, canonical.EventKill, other),
				kev(3000, canonical.EventKill, me),
				kev(4000, canonical.EventKill, me),
			},
			xuid: me,
			want: 2,
		},
		{
			name: "morts de l'autre joueur ne resettent pas le compteur du joueur",
			events: []canonical.HighlightEvent{
				kev(1000, canonical.EventKill, me),
				kev(1500, canonical.EventDeath, other), // mort de l'autre = ne touche pas `me`
				kev(2000, canonical.EventKill, me),
			},
			xuid: me,
			want: 2,
		},
		{
			name: "joueur sans kill => 0",
			events: []canonical.HighlightEvent{
				kev(1000, canonical.EventKill, other),
				kev(2000, canonical.EventDeath, me),
			},
			xuid: me,
			want: 0,
		},
		{
			name: "xuid vide => 0",
			events: []canonical.HighlightEvent{
				kev(1000, canonical.EventKill, me),
			},
			xuid: "",
			want: 0,
		},
		{
			name: "types non kill/death (medaille) ignores",
			events: []canonical.HighlightEvent{
				kev(1000, canonical.EventKill, me),
				kev(1500, canonical.EventMedal, me),
				kev(2000, canonical.EventKill, me),
			},
			xuid: me,
			want: 2,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := ComputeMaxKillingSpree(tc.events, tc.xuid); got != tc.want {
				t.Errorf("ComputeMaxKillingSpree() = %d, want %d", got, tc.want)
			}
		})
	}
}

// TestComputeMaxKillingSpree_DoesNotMutateInput garantit que le slice du caller n'est
// pas reordonne (copie locale avant tri).
func TestComputeMaxKillingSpree_DoesNotMutateInput(t *testing.T) {
	t.Parallel()
	const me = "xuid_me"
	events := []canonical.HighlightEvent{
		kev(3000, canonical.EventKill, me),
		kev(1000, canonical.EventKill, me),
		kev(2000, canonical.EventDeath, me),
	}
	_ = ComputeMaxKillingSpree(events, me)
	if events[0].TimeMS != 3000 || events[1].TimeMS != 1000 || events[2].TimeMS != 2000 {
		t.Errorf("l'input a ete mute : %+v", events)
	}
}
