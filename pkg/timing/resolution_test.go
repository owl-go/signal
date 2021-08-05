package timing

import (
	"testing"
)

func TestGetPixelsByResolution(t *testing.T) {
	t.Log(GetPixelsByResolution("240"))
	t.Log(GetPixelsByResolution("360"))
	t.Log(GetPixelsByResolution("480p"))
	t.Log(GetPixelsByResolution("720p"))
	t.Log(GetPixelsByResolution("1080p"))
	t.Log(GetPixelsByResolution("2k"))
	t.Log(GetPixelsByResolution("4k"))
}
