package notify

import (
	"testing"
)

func TestBuildMediaEmbed_Videos(t *testing.T) {
	rows := []mediaRow{
		{FileName: "clip1.mp4", Kind: "video", MatchID: "abc-123-def-456"},
		{FileName: "clip2.mp4", Kind: "video"},
	}
	embed := buildMediaEmbed(rows, "TestPlayer", "fr", nil)
	if embed.Title == "" {
		t.Fatal("expected non-empty title")
	}
	if len(embed.Fields) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(embed.Fields))
	}
}

func TestBuildMediaEmbed_Images(t *testing.T) {
	rows := []mediaRow{
		{FileName: "shot1.png", Kind: "image"},
	}
	embed := buildMediaEmbed(rows, "TestPlayer", "en", nil)
	if len(embed.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(embed.Fields))
	}
}

func TestBuildMediaEmbed_Mixed(t *testing.T) {
	rows := []mediaRow{
		{FileName: "clip.mp4", Kind: "video"},
		{FileName: "shot.png", Kind: "image"},
	}
	embed := buildMediaEmbed(rows, "Player", "fr", nil)
	if embed.Color != colorBlurple {
		t.Fatal("expected blurple color")
	}
}

func TestBuildMediaEmbed_Overflow(t *testing.T) {
	rows := make([]mediaRow, 10)
	for i := range rows {
		rows[i] = mediaRow{FileName: "file.png", Kind: "image"}
	}
	embed := buildMediaEmbed(rows, "Player", "fr", nil)
	// 6 + 1 overflow field = 7
	if len(embed.Fields) != 7 {
		t.Fatalf("expected 7 fields (6+overflow), got %d", len(embed.Fields))
	}
}
