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
	t.Log(getPixelsByResolution("2k"))
	t.Log(getPixelsByResolution("4k"))
}

func TestCalcPixelsToResolution(t *testing.T) {
	t.Log(CalcPixelsToResolution((getPixelsByResolution("240") + getPixelsByResolution("360")*3)))
	t.Log(CalcPixelsToResolution((getPixelsByResolution("240") + getPixelsByResolution("360")*6)))
	t.Log(CalcPixelsToResolution((getPixelsByResolution("240") + getPixelsByResolution("360")*12)))
	t.Log(CalcPixelsToResolution((getPixelsByResolution("240") + getPixelsByResolution("360")*24)))
	t.Log(CalcPixelsToResolution((getPixelsByResolution("240") + getPixelsByResolution("360")*32)))
	t.Log(CalcPixelsToResolution((getPixelsByResolution("240") + getPixelsByResolution("360")*68)))
}

func TestTransformResolution(t *testing.T) {
	t.Log(TransformResolution("240"))
	t.Log(TransformResolution("360"))
	t.Log(TransformResolution("480p"))
	t.Log(TransformResolution("720p"))
	t.Log(TransformResolution("1080p"))
	t.Log(TransformResolution("2k"))
	t.Log(TransformResolution("4k"))
}
