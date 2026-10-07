package filetype

import "testing"

func TestDetect(t *testing.T) {
	cases := map[string]string{
		"\x00\x00\x00\x1cftypavif\x00\x00\x00\x00": "image/avif",
		"\x00\x00\x00\x18ftypM4A \x00\x00\x00\x00": "audio/mp4",
		"II*\x00\x08\x00\x00\x00":                  "image/tiff",
		"\x89PNG\r\n\x1a\n\x00\x00\x00\rIHDR":      "image/png",
		"%PDF-1.7\n":                               "application/pdf",
		"name,age\n张三,30\n":                        "text/plain",
		"\x00\x01\x02\x03binary":                   "application/octet-stream",
	}
	for header, want := range cases {
		if got := Detect([]byte(header)); got != want {
			t.Errorf("Detect(%q) = %q, want %q", header, got, want)
		}
	}
}
