package rawpanellib

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"math/rand"
	"testing"
	"time"

	su "github.com/SKAARHOJ/ibeam-lib-utils"
	rwp "github.com/SKAARHOJ/rawpanel-lib/ibeam_rawpanel"
)

// rwpImgToImageV1100 is RwpImgToImage as it was in v1.10.0, which went through every pixel of the image.
func rwpImgToImageV1100(rwpImg *rwp.HWCGfx, width int, height int) image.Image {
	rwpImgW := int(rwpImg.W)
	rwpImgH := int(rwpImg.H)
	outputImage := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.Draw(outputImage, outputImage.Bounds(), &image.Uniform{color.RGBA{0, 0, 0, 255}}, image.ZP, draw.Src)
	wOffset := (width - rwpImgW) / 2
	hOffset := (height - rwpImgH) / 2
	wInBytes := int(math.Ceil(float64(rwpImgW) / 8))
	for y := 0; y < rwpImgH; y++ {
		for x := 0; x < rwpImgW; x++ {
			switch rwpImg.ImageType {
			case rwp.HWCGfx_RGB16bit:
				i := 2 * (rwpImgW*y + x)
				if int(i+1) < len(rwpImg.ImageData) {
					colorWord := (uint16(rwpImg.ImageData[i]) << 8) | uint16(rwpImg.ImageData[i+1])
					colR := su.MapValue(int(colorWord&0b11111), 0, 31, 0, 255)
					colG := su.MapValue(int((colorWord>>5)&0b111111), 0, 63, 0, 255)
					colB := su.MapValue(int((colorWord>>11)&0b11111), 0, 31, 0, 255)
					outputImage.Set(x+wOffset, y+hOffset, color.RGBA{uint8(colR), uint8(colG), uint8(colB), 255})
				}
			case rwp.HWCGfx_Gray4bit:
				i := (rwpImgW*y + x)
				if i/2 < len(rwpImg.ImageData) {
					colorByte := rwpImg.ImageData[i/2]
					if i%2 == 0 {
						colorByte = colorByte >> 4
					}
					gray := su.MapValue(int(colorByte&0xF), 0, 15, 0, 255)
					outputImage.Set(x+wOffset, y+hOffset, color.RGBA{uint8(gray), uint8(gray), uint8(gray), 255})
				}
			case rwp.HWCGfx_MONO:
				index := y*wInBytes + x/8
				if index >= 0 && index < len(rwpImg.ImageData) {
					if rwpImg.ImageData[index]&byte(0b1<<(7-x%8)) > 0 {
						outputImage.Set(x+wOffset, y+hOffset, color.RGBA{255, 255, 255, 255})
					} else {
						outputImage.Set(x+wOffset, y+hOffset, color.RGBA{0, 0, 0, 255})
					}
				}
			}
		}
	}
	return outputImage
}

func TestRwpImgToImageSameAsBefore(t *testing.T) {
	rnd := rand.New(rand.NewSource(1))
	sizes := [][2]int{{72, 72}, {64, 32}, {1, 1}, {0, 0}, {100, 90}, {200, 30}, {30, 200}, {73, 71}, {7, 9}}
	types := []rwp.HWCGfx_ImageTypeE{rwp.HWCGfx_RGB16bit, rwp.HWCGfx_Gray4bit, rwp.HWCGfx_MONO}
	for _, out := range [][2]int{{72, 72}, {96, 96}, {248, 58}, {65, 33}} {
		for _, sz := range sizes {
			for _, typ := range types {
				for _, dataFrac := range []float64{0, 0.3, 1, 1.5} {
					full := sz[0] * sz[1] * 2
					data := make([]byte, int(float64(full)*dataFrac))
					rnd.Read(data)
					img := &rwp.HWCGfx{ImageType: typ, W: uint32(sz[0]), H: uint32(sz[1]), ImageData: data}
					got := RwpImgToImage(img, out[0], out[1]).(*image.RGBA)
					want := rwpImgToImageV1100(img, out[0], out[1]).(*image.RGBA)
					if string(got.Pix) != string(want.Pix) {
						t.Fatalf("%v %dx%d with %d bytes into %dx%d: output differs from before", typ, sz[0], sz[1], len(data), out[0], out[1])
					}
				}
			}
		}
	}
}

func TestRwpImgToImageHugeSizeIsQuick(t *testing.T) {
	for _, typ := range []rwp.HWCGfx_ImageTypeE{rwp.HWCGfx_RGB16bit, rwp.HWCGfx_Gray4bit, rwp.HWCGfx_MONO} {
		img := &rwp.HWCGfx{ImageType: typ, W: 99999, H: 99999, ImageData: make([]byte, 3000)}
		start := time.Now()
		RwpImgToImage(img, 72, 72)
		if el := time.Since(start); el > 100*time.Millisecond {
			t.Errorf("%v 99999x99999: took %v", typ, el)
		}
	}
}
