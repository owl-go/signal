package timing

import (
	"strings"
)

func getPixelsByResolution(resolution string) uint64 {
	switch strings.ToLower(resolution) {
	case "240":
		return 57600
	case "360":
		return 129600
	case "480p":
		return 407040
	case "720p":
		return 921600
	case "1080p":
		return 2073600
	case "2k":
		return 3686400
	case "4k":
		return 8847360
	default:
		return 0
	}
}

func CalcPixelsToResolution(pixels uint64) string {
	if pixels == 0 {
		return ""
	}
	if pixels >= 129600 && pixels < 407040 {
		return "360"
	}
	if pixels >= 407040 && pixels < 921600 {
		return "480p"
	}
	if pixels >= 921600 && pixels < 2073600 {
		return "720p"
	}
	if pixels >= 2073600 && pixels < 3686400 {
		return "1080p"
	}
	if pixels >= 3686400 && pixels < 8847360 {
		return "2k"
	}
	if pixels >= 8847360 {
		return "4k"
	}
	return "240"
}

func TransformResolution(resolution string) string {
	pixels := getPixelsByResolution(resolution)
	if pixels > 0 && pixels <= 407040 {
		return "SD"
	} else if pixels > 407040 && pixels <= 921600 {
		return "HD"
	} else if pixels > 921600 && pixels <= 2073600 {
		return "FHD"
	} else if pixels > 2073600 && pixels <= 3686400 {
		return "2K"
	} else if pixels > 3686400 && pixels <= 8847360 {
		return "2KP"
	}
	return ""
}
