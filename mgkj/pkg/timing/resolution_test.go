package timing

import (
	"testing"
)

func TestGetPixelsByResolution(t *testing.T) {
	t.Log(getPixelsByResolution("240"))
	t.Log(getPixelsByResolution("360"))
	t.Log(getPixelsByResolution("480p"))
	t.Log(getPixelsByResolution("720p"))
	t.Log(getPixelsByResolution("1080p"))
}

func TestCalcPixelsToResolution(t *testing.T) {
	t.Log(CalcPixelsToResolution((getPixelsByResolution("240") + getPixelsByResolution("360")*3)))
}
