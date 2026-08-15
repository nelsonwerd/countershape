package main

import (
	"fmt"
	"image"
	_ "image/png"
	"os"
	"path/filepath"
)

var states = []string{
	"empty", "preparing", "active", "partial", "error", "unstable", "uncomparable", "discovered",
	"decision-ready", "predicate-editing", "identity-reveal", "resolved", "reject-all-resolved", "deferred",
	"stale", "invalidated", "incomplete", "completed",
}

func main() {
	if len(os.Args) != 5 {
		fail("usage: compare_matrices <baseline-dir> <baseline-prefix> <candidate-dir> <candidate-prefix>")
	}
	baselineDir, baselinePrefix, candidateDir, candidatePrefix := os.Args[1], os.Args[2], os.Args[3], os.Args[4]
	changedImages := 0
	changedPixels := 0
	totalPixels := 0
	for _, state := range states {
		for _, viewport := range []string{"desktop", "mobile"} {
			baseline := decode(filepath.Join(baselineDir, baselinePrefix+"-"+state+"-"+viewport+".png"))
			candidate := decode(filepath.Join(candidateDir, candidatePrefix+"-"+state+"-"+viewport+".png"))
			if baseline.Bounds() != candidate.Bounds() {
				fail("%s/%s bounds changed: %v != %v", state, viewport, baseline.Bounds(), candidate.Bounds())
			}
			imageChanged := 0
			for y := baseline.Bounds().Min.Y; y < baseline.Bounds().Max.Y; y++ {
				for x := baseline.Bounds().Min.X; x < baseline.Bounds().Max.X; x++ {
					a := rgba8(baseline.At(x, y).RGBA())
					b := rgba8(candidate.At(x, y).RGBA())
					if a == b {
						continue
					}
					for channel := range a {
						if abs(int(a[channel])-int(b[channel])) > 2 {
							fail("%s/%s pixel %d,%d channel %d changed by more than 2/255", state, viewport, x, y, channel)
						}
					}
					imageChanged++
				}
			}
			pixels := baseline.Bounds().Dx() * baseline.Bounds().Dy()
			if imageChanged*1000 > pixels {
				fail("%s/%s changed %d/%d pixels, exceeding the 0.1%% raster tolerance", state, viewport, imageChanged, pixels)
			}
			if imageChanged > 0 {
				changedImages++
				changedPixels += imageChanged
			}
			totalPixels += pixels
		}
	}
	fmt.Printf("P10_MATRIX_PIXEL_STABLE images=36 changed_images=%d changed_pixels=%d total_pixels=%d max_channel_delta=2 max_changed_ratio=0.001\n", changedImages, changedPixels, totalPixels)
}

func decode(path string) image.Image {
	file, err := os.Open(path)
	if err != nil {
		fail("open %s: %v", path, err)
	}
	defer file.Close()
	value, format, err := image.Decode(file)
	if err != nil || format != "png" {
		fail("decode %s as strict PNG: format=%q err=%v", path, format, err)
	}
	return value
}

func rgba8(r, g, b, a uint32) [4]uint8 {
	return [4]uint8{uint8(r >> 8), uint8(g >> 8), uint8(b >> 8), uint8(a >> 8)}
}

func abs(value int) int {
	if value < 0 {
		return -value
	}
	return value
}

func fail(format string, arguments ...any) {
	fmt.Fprintf(os.Stderr, format+"\n", arguments...)
	os.Exit(1)
}
