package notifications

import (
	"context"
	"testing"
)

type countingForwarder struct{ calls int }

func (c *countingForwarder) Forward(context.Context, *Notification) { c.calls++ }

func TestFanout(t *testing.T) {
	if Fanout() != nil || Fanout(nil, nil) != nil {
		t.Fatal("sans relais, Fanout rend nil : le service n'appelle rien")
	}
	a, b := &countingForwarder{}, &countingForwarder{}
	if Fanout(nil, a) != ExternalForwarder(a) {
		t.Fatal("un seul relais : rendu tel quel")
	}
	f := Fanout(a, nil, b)
	f.Forward(context.Background(), &Notification{Category: CategoryMatchSynced})
	if a.calls != 1 || b.calls != 1 {
		t.Fatalf("chaque relais est appelé une fois : a=%d b=%d", a.calls, b.calls)
	}
}
