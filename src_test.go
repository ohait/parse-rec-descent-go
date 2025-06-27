package parse

import "testing"

func TestSrcLine(t *testing.T) {
	src := Src{bytes: []byte("a\nb\nc")}
	tests := []struct {
		off  int
		line int
	}{
		{0, 1},
		{1, 1},
		{2, 2},
		{3, 2},
		{4, 3},
		{5, 3},
	}
	for _, tt := range tests {
		got := src.Line(tt.off)
		if got != tt.line {
			t.Errorf("offset %d: expected line %d got %d", tt.off, tt.line, got)
		}
	}
}

func TestSrcLineTrailingNewline(t *testing.T) {
	src := Src{bytes: []byte("a\nb\n")}
	tests := []struct {
		off  int
		line int
	}{
		{0, 1},
		{1, 1},
		{2, 2},
		{3, 2},
		{4, 3},
	}
	for _, tt := range tests {
		if got := src.Line(tt.off); got != tt.line {
			t.Errorf("offset %d: expected line %d got %d", tt.off, tt.line, got)
		}
	}
}
