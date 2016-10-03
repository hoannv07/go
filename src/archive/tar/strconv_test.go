// Copyright 2016 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tar

import (
	"math"
	"strings"
	"testing"
	"time"
)

func TestFitsInBase256(t *testing.T) {
	vectors := []struct {
		in    int64
		width int
		ok    bool
	}{
		{+1, 8, true},
		{0, 8, true},
		{-1, 8, true},
		{1 << 56, 8, false},
		{(1 << 56) - 1, 8, true},
		{-1 << 56, 8, true},
		{(-1 << 56) - 1, 8, false},
		{121654, 8, true},
		{-9849849, 8, true},
		{math.MaxInt64, 9, true},
		{0, 9, true},
		{math.MinInt64, 9, true},
		{math.MaxInt64, 12, true},
		{0, 12, true},
		{math.MinInt64, 12, true},
	}

	for _, v := range vectors {
		ok := fitsInBase256(v.width, v.in)
		if ok != v.ok {
			t.Errorf("fitsInBase256(%d, %d): got %v, want %v", v.in, v.width, ok, v.ok)
		}
	}
}

func TestParseNumeric(t *testing.T) {
	vectors := []struct {
		in   string
		want int64
		ok   bool
	}{
		// Test base-256 (binary) encoded values.
		{"", 0, true},
		{"\x80", 0, true},
		{"\x80\x00", 0, true},
		{"\x80\x00\x00", 0, true},
		{"\xbf", (1 << 6) - 1, true},
		{"\xbf\xff", (1 << 14) - 1, true},
		{"\xbf\xff\xff", (1 << 22) - 1, true},
		{"\xff", -1, true},
		{"\xff\xff", -1, true},
		{"\xff\xff\xff", -1, true},
		{"\xc0", -1 * (1 << 6), true},
		{"\xc0\x00", -1 * (1 << 14), true},
		{"\xc0\x00\x00", -1 * (1 << 22), true},
		{"\x87\x76\xa2\x22\xeb\x8a\x72\x61", 537795476381659745, true},
		{"\x80\x00\x00\x00\x07\x76\xa2\x22\xeb\x8a\x72\x61", 537795476381659745, true},
		{"\xf7\x76\xa2\x22\xeb\x8a\x72\x61", -615126028225187231, true},
		{"\xff\xff\xff\xff\xf7\x76\xa2\x22\xeb\x8a\x72\x61", -615126028225187231, true},
		{"\x80\x7f\xff\xff\xff\xff\xff\xff\xff", math.MaxInt64, true},
		{"\x80\x80\x00\x00\x00\x00\x00\x00\x00", 0, false},
		{"\xff\x80\x00\x00\x00\x00\x00\x00\x00", math.MinInt64, true},
		{"\xff\x7f\xff\xff\xff\xff\xff\xff\xff", 0, false},
		{"\xf5\xec\xd1\xc7\x7e\x5f\x26\x48\x81\x9f\x8f\x9b", 0, false},

		// Test base-8 (octal) encoded values.
		{"0000000\x00", 0, true},
		{" \x0000000\x00", 0, true},
		{" \x0000003\x00", 3, true},
		{"00000000227\x00", 0227, true},
		{"032033\x00 ", 032033, true},
		{"320330\x00 ", 0320330, true},
		{"0000660\x00 ", 0660, true},
		{"\x00 0000660\x00 ", 0660, true},
		{"0123456789abcdef", 0, false},
		{"0123456789\x00abcdef", 0, false},
		{"01234567\x0089abcdef", 342391, true},
		{"0123\x7e\x5f\x264123", 0, false},
	}

	for _, v := range vectors {
		var p parser
		got := p.parseNumeric([]byte(v.in))
		ok := (p.err == nil)
		if ok != v.ok {
			if v.ok {
				t.Errorf("parseNumeric(%q): got parsing failure, want success", v.in)
			} else {
				t.Errorf("parseNumeric(%q): got parsing success, want failure", v.in)
			}
		}
		if ok && got != v.want {
			t.Errorf("parseNumeric(%q): got %d, want %d", v.in, got, v.want)
		}
	}
}

func TestFormatNumeric(t *testing.T) {
	vectors := []struct {
		in   int64
		want string
		ok   bool
	}{
		// Test base-256 (binary) encoded values.
		{-1, "\xff", true},
		{-1, "\xff\xff", true},
		{-1, "\xff\xff\xff", true},
		{(1 << 0), "0", false},
		{(1 << 8) - 1, "\x80\xff", true},
		{(1 << 8), "0\x00", false},
		{(1 << 16) - 1, "\x80\xff\xff", true},
		{(1 << 16), "00\x00", false},
		{-1 * (1 << 0), "\xff", true},
		{-1*(1<<0) - 1, "0", false},
		{-1 * (1 << 8), "\xff\x00", true},
		{-1*(1<<8) - 1, "0\x00", false},
		{-1 * (1 << 16), "\xff\x00\x00", true},
		{-1*(1<<16) - 1, "00\x00", false},
		{537795476381659745, "0000000\x00", false},
		{537795476381659745, "\x80\x00\x00\x00\x07\x76\xa2\x22\xeb\x8a\x72\x61", true},
		{-615126028225187231, "0000000\x00", false},
		{-615126028225187231, "\xff\xff\xff\xff\xf7\x76\xa2\x22\xeb\x8a\x72\x61", true},
		{math.MaxInt64, "0000000\x00", false},
		{math.MaxInt64, "\x80\x00\x00\x00\x7f\xff\xff\xff\xff\xff\xff\xff", true},
		{math.MinInt64, "0000000\x00", false},
		{math.MinInt64, "\xff\xff\xff\xff\x80\x00\x00\x00\x00\x00\x00\x00", true},
		{math.MaxInt64, "\x80\x7f\xff\xff\xff\xff\xff\xff\xff", true},
		{math.MinInt64, "\xff\x80\x00\x00\x00\x00\x00\x00\x00", true},
	}

	for _, v := range vectors {
		var f formatter
		got := make([]byte, len(v.want))
		f.formatNumeric(got, v.in)
		ok := (f.err == nil)
		if ok != v.ok {
			if v.ok {
				t.Errorf("formatNumeric(%d): got formatting failure, want success", v.in)
			} else {
				t.Errorf("formatNumeric(%d): got formatting success, want failure", v.in)
			}
		}
		if string(got) != v.want {
			t.Errorf("formatNumeric(%d): got %q, want %q", v.in, got, v.want)
		}
	}
}

func TestParsePAXTime(t *testing.T) {
	timestamps := map[string]time.Time{
		"1350244992.023960108":  time.Unix(1350244992, 23960108), // The common case
		"1350244992.02396010":   time.Unix(1350244992, 23960100), // Lower precision value
		"1350244992.0239601089": time.Unix(1350244992, 23960108), // Higher precision value
		"1350244992":            time.Unix(1350244992, 0),        // Low precision value
	}

	for input, expected := range timestamps {
		ts, err := parsePAXTime(input)
		if err != nil {
			t.Fatal(err)
		}
		if !ts.Equal(expected) {
			t.Fatalf("Time parsing failure %s %s", ts, expected)
		}
	}
}

func TestParsePAXRecord(t *testing.T) {
	medName := strings.Repeat("CD", 50)
	longName := strings.Repeat("AB", 100)

	vectors := []struct {
		in      string
		wantRes string
		wantKey string
		wantVal string
		ok      bool
	}{
		{"6 k=v\n\n", "\n", "k", "v", true},
		{"19 path=/etc/hosts\n", "", "path", "/etc/hosts", true},
		{"210 path=" + longName + "\nabc", "abc", "path", longName, true},
		{"110 path=" + medName + "\n", "", "path", medName, true},
		{"9 foo=ba\n", "", "foo", "ba", true},
		{"11 foo=bar\n\x00", "\x00", "foo", "bar", true},
		{"18 foo=b=\nar=\n==\x00\n", "", "foo", "b=\nar=\n==\x00", true},
		{"27 foo=hello9 foo=ba\nworld\n", "", "foo", "hello9 foo=ba\nworld", true},
		{"27 ☺☻☹=日a本b語ç\nmeow mix", "meow mix", "☺☻☹", "日a本b語ç", true},
		{"17 \x00hello=\x00world\n", "", "\x00hello", "\x00world", true},
		{"1 k=1\n", "1 k=1\n", "", "", false},
		{"6 k~1\n", "6 k~1\n", "", "", false},
		{"6_k=1\n", "6_k=1\n", "", "", false},
		{"6 k=1 ", "6 k=1 ", "", "", false},
		{"632 k=1\n", "632 k=1\n", "", "", false},
		{"16 longkeyname=hahaha\n", "16 longkeyname=hahaha\n", "", "", false},
		{"3 somelongkey=\n", "3 somelongkey=\n", "", "", false},
		{"50 tooshort=\n", "50 tooshort=\n", "", "", false},
	}

	for _, v := range vectors {
		key, val, res, err := parsePAXRecord(v.in)
		ok := (err == nil)
		if ok != v.ok {
			if v.ok {
				t.Errorf("parsePAXRecord(%q): got parsing failure, want success", v.in)
			} else {
				t.Errorf("parsePAXRecord(%q): got parsing success, want failure", v.in)
			}
		}
		if v.ok && (key != v.wantKey || val != v.wantVal) {
			t.Errorf("parsePAXRecord(%q): got (%q: %q), want (%q: %q)",
				v.in, key, val, v.wantKey, v.wantVal)
		}
		if res != v.wantRes {
			t.Errorf("parsePAXRecord(%q): got residual %q, want residual %q",
				v.in, res, v.wantRes)
		}
	}
}

func TestFormatPAXRecord(t *testing.T) {
	medName := strings.Repeat("CD", 50)
	longName := strings.Repeat("AB", 100)

	vectors := []struct {
		inKey string
		inVal string
		want  string
	}{
		{"k", "v", "6 k=v\n"},
		{"path", "/etc/hosts", "19 path=/etc/hosts\n"},
		{"path", longName, "210 path=" + longName + "\n"},
		{"path", medName, "110 path=" + medName + "\n"},
		{"foo", "ba", "9 foo=ba\n"},
		{"foo", "bar", "11 foo=bar\n"},
		{"foo", "b=\nar=\n==\x00", "18 foo=b=\nar=\n==\x00\n"},
		{"foo", "hello9 foo=ba\nworld", "27 foo=hello9 foo=ba\nworld\n"},
		{"☺☻☹", "日a本b語ç", "27 ☺☻☹=日a本b語ç\n"},
		{"\x00hello", "\x00world", "17 \x00hello=\x00world\n"},
	}

	for _, v := range vectors {
		got := formatPAXRecord(v.inKey, v.inVal)
		if got != v.want {
			t.Errorf("formatPAXRecord(%q, %q): got %q, want %q",
				v.inKey, v.inVal, got, v.want)
		}
	}
}