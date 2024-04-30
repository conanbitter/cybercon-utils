package main

import (
	"encoding/binary"
	"image"
	"image/png"
	"log"
	"math"
	"os"

	_ "image/jpeg"

	_ "golang.org/x/image/tiff"
)

type ImageData struct {
	Width  int
	Height int
	Data   []float64
}

func clipColor(data float64) uint8 {
	if data < 0.0 {
		data = 0.0
	}
	if data > 1.0 {
		data = 1.0
	}
	return uint8(data * 255)
}

func (img *ImageData) Save(filename string) {
	output := image.NewGray(image.Rectangle{image.Point{0, 0}, image.Point{img.Width, img.Height}})
	for y := 0; y < img.Height; y++ {
		for x := 0; x < img.Width; x++ {
			output.Pix[output.PixOffset(x, y)] = clipColor(img.Get(x, y))
		}
	}
	outf, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer outf.Close()
	if err = png.Encode(outf, output); err != nil {
		log.Printf("failed to encode: %v", err)
	}
}

func (img *ImageData) SaveRaw(filename string) {
	fo, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer fo.Close()

	binary.Write(fo, binary.LittleEndian, uint32(img.Width))
	binary.Write(fo, binary.LittleEndian, uint32(img.Height))
	binary.Write(fo, binary.LittleEndian, img.Data)
}

func (img *ImageData) GetBytes() [][]byte {
	cols := img.Width
	rows := img.Height / 8
	result := make([][]byte, rows)
	for y := 0; y < rows; y++ {
		result[y] = make([]byte, cols)
		for x := 0; x < cols; x++ {
			color := 0
			for s := 0; s < 8; s++ {
				if img.Get(x, y*8+s) > 0.5 {
					color = color | 1<<s
				}
			}
			result[y][x] = byte(color)
		}
	}
	return result
}

func (img *ImageData) Set(x int, y int, color float64) {
	img.Data[x+y*img.Width] = color
}

func (img *ImageData) Get(x int, y int) float64 {
	return img.Data[x+y*img.Width]
}

func NewImage(width int, height int) *ImageData {
	result := &ImageData{
		Width:  width,
		Height: height,
		Data:   make([]float64, width*height),
	}
	clear(result.Data)
	return result
}

func ImageFrom(other *ImageData) *ImageData {
	res := &ImageData{
		Width:  other.Width,
		Height: other.Height,
		Data:   make([]float64, other.Width*other.Height),
	}
	copy(res.Data, other.Data)
	return res
}

func ColorConvert(r uint32, g uint32, b uint32, gamma float64) float64 {
	rf := float64(r) / 65535.0
	gf := float64(g) / 65535.0
	bf := float64(b) / 65535.0

	c := 0.299*rf + 0.587*gf + 0.114*bf

	if gamma < 0 {
		return c
	}
	// Gamma correction
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, gamma)
}

func ImageLoad(filename string, gamma float64) *ImageData {
	imgFile, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer imgFile.Close()
	img, _, err := image.Decode(imgFile)
	if err != nil {
		panic(err)
	}
	bounds := img.Bounds()
	result := NewImage((bounds.Max.X - bounds.Min.X), (bounds.Max.Y - bounds.Min.Y))
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, _ := img.At(x, y).RGBA()
			result.Set(x-bounds.Min.X, y-bounds.Min.Y, ColorConvert(r, g, b, gamma))
		}
	}
	return result
}

func ImageLoadRaw(filename string) *ImageData {
	fo, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer fo.Close()
	result := &ImageData{}

	var width uint32
	var height uint32
	binary.Read(fo, binary.LittleEndian, &width)
	binary.Read(fo, binary.LittleEndian, &height)
	result.Width = int(width)
	result.Height = int(height)
	result.Data = make([]float64, result.Width*result.Height)
	binary.Read(fo, binary.LittleEndian, result.Data)

	return result
}
