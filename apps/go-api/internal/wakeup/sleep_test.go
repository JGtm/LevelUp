package wakeup

import (
	"testing"
	"time"
)

// fakeClocks : deux horloges pilotées à la main. advance fait avancer les deux (temps ordinaire) ;
// sleep fait avancer la seule horloge murale (veille : le compteur monotone ne compte pas).
type fakeClocks struct {
	wall time.Time
	mono time.Duration
}

func newFakeClocks() *fakeClocks {
	return &fakeClocks{wall: time.Date(2026, 11, 2, 9, 0, 0, 0, time.UTC)}
}

func (c *fakeClocks) Wall() time.Time     { return c.wall }
func (c *fakeClocks) Mono() time.Duration { return c.mono }

func (c *fakeClocks) advance(d time.Duration) {
	c.wall = c.wall.Add(d)
	c.mono += d
}

func (c *fakeClocks) sleep(d time.Duration) { c.wall = c.wall.Add(d) }

// TestSleepDetector : l'écart murale/monotone depuis le tic précédent, contre le seuil de 2 minutes.
func TestSleepDetector(t *testing.T) {
	cas := []struct {
		name     string
		step     func(c *fakeClocks)
		wantWoke bool
		wantGap  time.Duration
	}{
		{"tic ordinaire de 30 s", func(c *fakeClocks) { c.advance(SleepTick) }, false, 0},
		{"tic en retard de 20 s (charge)", func(c *fakeClocks) { c.advance(SleepTick + 20*time.Second) }, false, 0},
		{"veille de 90 s : sous le seuil", func(c *fakeClocks) { c.advance(SleepTick); c.sleep(90 * time.Second) }, false, 90 * time.Second},
		{"veille de 2 minutes pile : pas au-delà", func(c *fakeClocks) { c.advance(SleepTick); c.sleep(SleepGap) }, false, SleepGap},
		{"veille de 8 heures", func(c *fakeClocks) { c.advance(SleepTick); c.sleep(8 * time.Hour) }, true, 8 * time.Hour},
		{"horloge murale reculée d'une heure (réglage manuel)", func(c *fakeClocks) { c.advance(SleepTick); c.sleep(-time.Hour) }, false, -time.Hour},
	}
	for _, c := range cas {
		t.Run(c.name, func(t *testing.T) {
			clocks := newFakeClocks()
			d := NewSleepDetector(clocks)
			c.step(clocks)
			gap, woke := d.Check()
			if woke != c.wantWoke {
				t.Errorf("woke = %v, attendu %v (écart %v)", woke, c.wantWoke, gap)
			}
			if gap != c.wantGap {
				t.Errorf("écart = %v, attendu %v", gap, c.wantGap)
			}
		})
	}
}

// TestSleepDetector_CompareAuTicPrecedent : une veille n'est comptée qu'une fois, au tic qui la suit.
func TestSleepDetector_CompareAuTicPrecedent(t *testing.T) {
	clocks := newFakeClocks()
	d := NewSleepDetector(clocks)
	clocks.advance(SleepTick)
	clocks.sleep(time.Hour)
	if _, woke := d.Check(); !woke {
		t.Fatal("veille d'une heure non détectée")
	}
	clocks.advance(SleepTick)
	if gap, woke := d.Check(); woke || gap != 0 {
		t.Errorf("le tic suivant la veille voit encore un écart : %v (woke=%v)", gap, woke)
	}
}

// TestSystemClocks_WallSansLectureMonotone : l'heure murale de production ne porte pas de lecture
// monotone — sans cela, deux heures murales se compareraient en monotone et l'écart de veille
// disparaîtrait (ADR 0045 A24).
func TestSystemClocks_WallSansLectureMonotone(t *testing.T) {
	c := NewSystemClocks()
	w := c.Wall()
	if w.Round(0) != w {
		t.Error("Wall() porte une lecture monotone")
	}
	time.Sleep(5 * time.Millisecond)
	if c.Mono() <= 0 {
		t.Error("Mono() n'avance pas")
	}
}
