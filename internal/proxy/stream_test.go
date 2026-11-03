package proxy

import (
	"net/url"
	"strings"
	"testing"
)

func TestImageSizing(t *testing.T) {
	cases := []struct {
		query string
		want  string
	}{
		{"maxWidth=320", "maxWidth=320"},
		{"maxWidth=99999&quality=500", "maxWidth=1920&quality=100"},
		{"maxHeight=-4&quality=abc&maxWidth=", ""},
		{"maxWidth=400&other=1", "maxWidth=400"},
	}
	for _, c := range cases {
		query, _ := url.ParseQuery(c.query)
		if got := imageSizing(query).Encode(); got != c.want {
			t.Errorf("%q: got %q, want %q", c.query, got, c.want)
		}
	}
}

func TestSanitizeManifestRemovesCredentials(t *testing.T) {
	in := "#EXTM3U\n" +
		"#EXT-X-STREAM-INF:BANDWIDTH=6708045,RESOLUTION=1800x1080\n" +
		"main.m3u8?AudioCodec=copy&MediaSourceId=abc&ApiKey=secret&VideoBitrate=6068045\n" +
		"#EXT-X-IMAGE-STREAM-INF:BANDWIDTH=6775,URI=\"Trickplay/320/tiles.m3u8?MediaSourceId=abc&ApiKey=secret\"\n"

	out := string(sanitizeManifest([]byte(in)))
	for _, bad := range []string{"ApiKey", "secret", "EXT-X-IMAGE-STREAM-INF"} {
		if strings.Contains(out, bad) {
			t.Fatalf("sanitized manifest still contains %q:\n%s", bad, out)
		}
	}
	for _, want := range []string{"main.m3u8?", "MediaSourceId=abc", "VideoBitrate=6068045"} {
		if !strings.Contains(out, want) {
			t.Fatalf("sanitized manifest lost %q:\n%s", want, out)
		}
	}
}

func TestSanitizeManifestStripsURICredentials(t *testing.T) {
	in := "#EXTM3U\n#EXT-X-MAP:URI=\"init.mp4?api_key=secret&X-Emby-Token=secret2\"\n"
	out := string(sanitizeManifest([]byte(in)))
	for _, bad := range []string{"secret", "api_key", "X-Emby-Token"} {
		if strings.Contains(out, bad) {
			t.Fatalf("sanitized manifest still contains %q:\n%s", bad, out)
		}
	}
	if !strings.Contains(out, `URI="init.mp4"`) {
		t.Fatalf("sanitized manifest lost the URI:\n%s", out)
	}
}
