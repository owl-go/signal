package timing

import (
	"strings"
)

func GetPixelsByResolution(resolution string) int64 {
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

func TransformResolutionFromPixels(pixels int64) string {
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
