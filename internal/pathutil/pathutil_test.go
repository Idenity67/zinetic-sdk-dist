package pathutil

import "testing"

func TestSegment_EscapesSpecialCharacters(t *testing.T) {
	got, err := Segment("id", "agent #1?")
	if err != nil {
		t.Fatalf("Segment returned error: %v", err)
	}
	if got != "agent%20%231%3F" {
		t.Fatalf("unexpected escaped segment: %s", got)
	}
}

func TestSegment_RejectsSlash(t *testing.T) {
	if _, err := Segment("id", "a/b"); err == nil {
		t.Fatal("expected slash to be rejected for a single segment")
	}
}

func TestSlashPath_EscapesEachSegment(t *testing.T) {
	got, err := SlashPath("secret path", "app config/db#primary?rw")
	if err != nil {
		t.Fatalf("SlashPath returned error: %v", err)
	}
	if got != "app%20config/db%23primary%3Frw" {
		t.Fatalf("unexpected escaped slash path: %s", got)
	}
}

func TestSlashPath_RejectsTraversal(t *testing.T) {
	if _, err := SlashPath("secret path", "app/../db"); err == nil {
		t.Fatal("expected traversal segment to be rejected")
	}
}
