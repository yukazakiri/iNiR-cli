package cmd

import "testing"

func TestTrimForNotification(t *testing.T) {
	t.Parallel()

	short := trimForNotification("hello", 10)
	if short != "hello" {
		t.Fatalf("unexpected short trim result: %q", short)
	}

	trimmed := trimForNotification("abcdefghijklmnopqrstuvwxyz", 8)
	if trimmed != "abcde..." {
		t.Fatalf("unexpected trimmed value: %q", trimmed)
	}
}
