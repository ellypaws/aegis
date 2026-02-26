package utils

import (
	"errors"
	"regexp"
	"strings"
	"testing"
)

// stripANSI removes ANSI color codes so we can assert on the plain text.
var ansiRE = regexp.MustCompile(`\x1b\[[0-9;]*m`)

func stripANSI(s string) string {
	return ansiRE.ReplaceAllString(s, "")
}

func TestStripInvalidName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		in   string
		want string
		name string
	}{
		{name: "lowercase_kept", in: "hello-world_123", want: "hello-world_123"},
		{name: "uppercase_kept", in: "HELLO_WORLD", want: "HELLO_WORLD"},
		{name: "mixed_forced_lower", in: "Hello-World", want: "hello-world"},
		{name: "apostrophe_allowed", in: "Foo'Bar_Baz", want: "foo'bar_baz"},
		{name: "invalids_removed", in: "A*B C!", want: "ABC"},
		{name: "mixed_plus_invalids", in: "A*B c!", want: "abc"},
		{name: "numbers_allowed", in: "User42", want: "user42"},
		{name: "dashes_and_underscores_allowed", in: "Dash-Under_score", want: "dash-under_score"},
		{name: "empty_input", in: "", want: ""},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := StripInvalidName(tc.in)
			if got != tc.want {
				t.Fatalf("StripInvalidName(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestDetectInvalidChars_None(t *testing.T) {
	t.Parallel()

	valids := []string{
		"hello",
		"HELLO_WORLD",
		"foo-bar_baz'qux123",
	}

	for _, v := range valids {
		v := v
		t.Run(v, func(t *testing.T) {
			t.Parallel()
			if err := DetectInvalidChars(v); err != nil {
				t.Fatalf("DetectInvalidChars(%q) unexpected error: %v", v, err)
			}
		})
	}
}

func TestDetectInvalidChars_WithMarkers(t *testing.T) {
	t.Parallel()

	type want struct {
		rawLine     string // original line, no ANSI
		caretLine   string // positions of carets marking invalid chars
		numCarets   int
		containsMsg string
	}

	tests := []struct {
		in   string
		want want
		name string
	}{
		{
			name: "double_bang",
			in:   "He!!o_World",
			want: want{
				rawLine:     "He!!o_World",
				caretLine:   "  ^^        ",
				numCarets:   2,
				containsMsg: "invalid characters detected:",
			},
		},
		{
			name: "spaces_and_symbols",
			in:   "A*B C!",
			want: want{
				rawLine:     "A*B C!",
				caretLine:   " ^ ^ ^",
				numCarets:   3,
				containsMsg: "invalid characters detected:",
			},
		},
		{
			name: "leading_trailing_invalids",
			in:   "#OK?",
			want: want{
				rawLine:     "#OK?",
				caretLine:   "^  ^",
				numCarets:   2,
				containsMsg: "invalid characters detected:",
			},
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := DetectInvalidChars(tc.in)
			if err == nil {
				t.Fatalf("DetectInvalidChars(%q) expected error, got nil", tc.in)
			}

			// LOG the raw output
			t.Logf("DetectInvalidChars(%q) output:\n%s", tc.in, err.Error())

			// Basic message presence
			if !strings.Contains(err.Error(), tc.want.containsMsg) {
				t.Fatalf("error message missing header %q:\n%s", tc.want.containsMsg, err.Error())
			}

			lines := strings.Split(err.Error(), "\n")
			if len(lines) < 3 {
				t.Fatalf("unexpected error formatting, got %d lines:\n%s", len(lines), err.Error())
			}

			coloredLine := lines[len(lines)-2]
			caretLine := lines[len(lines)-1]

			plainLine := stripANSI(coloredLine)
			if plainLine != tc.want.rawLine {
				t.Fatalf("plain colored line = %q, want %q", plainLine, tc.want.rawLine)
			}

			if len([]rune(caretLine)) != len([]rune(tc.in)) {
				t.Fatalf("caret length = %d, want %d", len([]rune(caretLine)), len([]rune(tc.in)))
			}

			if got := strings.Count(caretLine, "^"); got != tc.want.numCarets {
				t.Fatalf("number of carets = %d, want %d\ncaretLine: %q", got, tc.want.numCarets, caretLine)
			}

			mask := make([]rune, len(tc.in))
			for i := range mask {
				mask[i] = ' '
			}
			for _, loc := range invalidChars.FindAllStringIndex(tc.in, -1) {
				for i := loc[0]; i < loc[1]; i++ {
					mask[i] = '^'
				}
			}
			if strings.TrimRight(string(mask), " ") != strings.TrimRight(caretLine, " ") {
				t.Fatalf("caret positions mismatch\n got: %q\nwant: %q", caretLine, string(mask))
			}
		})
	}
}

func TestDetectInvalidChars_ErrorImplements(t *testing.T) {
	t.Parallel()

	err := DetectInvalidChars("bad!")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	t.Logf("Output for 'bad!':\n%s", err.Error())

	var _ error = err

	if errors.Is(err, errors.New("invalid characters detected")) {
		t.Fatal("unexpected errors.Is match")
	}
}

func BenchmarkStripInvalidName(b *testing.B) {
	in := "Hello-World_123* Foo'Bar!? こんにちは"
	b.ReportAllocs()
	for b.Loop() {
		_ = StripInvalidName(in)
	}
}
