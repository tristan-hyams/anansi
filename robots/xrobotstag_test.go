package robots_test

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/tristan-hyams/anansi/robots"
)

func TestParseXRobotsTag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		headerVal  string
		wantIndex  bool
		wantFollow bool
	}{
		{"empty header", "", true, true},
		{"noindex only", "noindex", false, true},
		{"nofollow only", "nofollow", true, false},
		{"noindex and follow", "noindex, follow", false, true},
		{"noindex and nofollow", "noindex, nofollow", false, false},
		{"none", "none", false, false},
		{"follow only", "follow", true, true},
		{"mixed case", "NoIndex, NoFollow", false, false},
		{"extra whitespace", "  noindex ,  nofollow  ", false, false},
		{"unrecognised directive", "noarchive, nosnippet", true, true},
		{"nofollow with unrecognised", "nofollow, noarchive", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			header := http.Header{}
			if tt.headerVal != "" {
				header.Set("X-Robots-Tag", tt.headerVal)
			}

			d := robots.ParseXRobotsTag(header)
			assert.Equal(t, !tt.wantIndex, d.NoIndex, "NoIndex")
			assert.Equal(t, !tt.wantFollow, d.NoFollow, "NoFollow")
		})
	}
}

func TestDirectives_ShouldFollow(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		headerVal string
		want      bool
	}{
		{"no header", "", true},
		{"follow", "follow", true},
		{"noindex follow", "noindex, follow", true},
		{"nofollow", "nofollow", false},
		{"none", "none", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			header := http.Header{}
			if tt.headerVal != "" {
				header.Set("X-Robots-Tag", tt.headerVal)
			}

			d := robots.ParseXRobotsTag(header)
			assert.Equal(t, tt.want, d.ShouldFollow())
		})
	}
}
