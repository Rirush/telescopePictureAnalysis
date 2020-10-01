package main

import (
	"fmt"
	"golang.org/x/image/tiff"
	"image"
	"image/color"
	"image/png"
	"os"
	"sort"
)

const (
	Threshold = 5
)

type Maximum struct {
	Index int
	Value float64
}

func RGBtoGray(color color.Color) float64 {
	r, g, b, _ := color.RGBA()
	return 0.3*float64(r) + 0.59*float64(g) + 0.11*float64(b)
}

func GetColumn(img image.Image, x int) (res []float64) {
	if x >= img.Bounds().Max.X {
		panic("trying to read out of bounds")
	}
	y := img.Bounds().Max.Y
	for i := 0; i < y; i++ {
		c := img.At(x, i)
		res = append(res, RGBtoGray(c))
	}
	return
}

func GetRow(img image.Image, y int) (res []float64) {
	if y >= img.Bounds().Max.Y {
		panic("trying to read out of bounds")
	}
	x := img.Bounds().Max.X
	for i := 0; i < x; i++ {
		c := img.At(i, y)
		res = append(res, RGBtoGray(c))
	}
	return
}

func ComputeAverages(points []float64, n int) (res []float64) {
	p := make([]float64, len(points))
	copy(p, points)
	for len(p) % n != 0 {
		p = append(p, p[len(p)-1])
	}
	for i := 0; i < len(p); i += n {
		v := 0.0
		for j := i; j < i + n; j++ {
			v += p[j]
		}
		res = append(res, v / float64(n))
	}
	return
}

func ComputeDeltas(points []float64) (deltas []float64) {
	p := points[0]
	for i := 1; i < len(points); i++ {
		deltas = append(deltas, points[i]-p)
		p = points[i]
	}
	return
}

func DetermineMaximums(deltas []float64) (maximums []Maximum) {
	for i := 1; i < len(deltas)-1; i++ {
		if deltas[i] > deltas[i-1] && deltas[i] > deltas[i+1] {
			maximums = append(maximums, Maximum{Index: i, Value: deltas[i]})
		}
	}
	return
}

func SortMaximums(maximums []Maximum) {
	sort.Slice(maximums, func(i, j int) bool {
		return maximums[i].Value > maximums[j].Value
	})
}

func OrderMaximums(maximums []Maximum) {
	sort.Slice(maximums, func(i, j int) bool {
		return maximums[i].Index < maximums[j].Index
	})
}

func FindZeroPoint(deltas []float64, i int) int {
	for ; i > 0; i-- {
		if deltas[i] < 0 {
			return i + 1
		}
	}
	return i
}

func FindLowestPoint(values []float64, i int, limit int) int {
	min, j := values[i], i
	limit = i - limit
	i--
	for ; i > limit && i >= 0; i-- {
		if values[i] < min {
			min = values[i]
			j = i
		}
	}
	return j
}

func DetectEdges(points []float64, n int, limit int) (edge1, edge2, edge3, edge4 int) {
	avg := ComputeAverages(points, n)
	deltas := ComputeDeltas(avg)
	max := DetermineMaximums(deltas)
	SortMaximums(max)
	m := max[:3]
	OrderMaximums(m)
	// Блок с левой границей
	b1 := FindZeroPoint(deltas, m[0].Index)
	edge1 = FindLowestPoint(points, b1 * 3, limit) + Threshold
	// Блок содержащий среднюю границу, нужно разбить на два
	b2 := FindZeroPoint(deltas, m[1].Index)
	_edge2 := FindLowestPoint(points, b2 * 3, limit)
	edge2, edge3 = _edge2 - Threshold, _edge2 + Threshold
	// Блок с правой границей
	b3 := FindZeroPoint(deltas, m[2].Index)
	edge4 = FindLowestPoint(points, b3 * 3, limit) - Threshold
	return
}

func Draw(img *image.RGBA, bounds image.Rectangle) {
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		img.Set(x, bounds.Min.Y, color.RGBA{G: 255, A: 255})
		img.Set(x, bounds.Max.Y, color.RGBA{G: 255, A: 255})
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		img.Set(bounds.Min.X, y, color.RGBA{G: 255, A: 255})
		img.Set(bounds.Max.X, y, color.RGBA{G: 255, A: 255})
	}
}

func main() {
	if len(os.Args) != 2 {
		fmt.Println("Usage: tool picture")
		return
	}
	f, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Println(err)
		return
	}
	img, err := tiff.Decode(f)
	if err != nil {
		fmt.Println("Decode failure:", err)
		return
	}
	column := GetColumn(img, 140)
	cedge1, cedge2, cedge3, cedge4 := DetectEdges(column, 3, 10)
	fmt.Println(cedge1, cedge2, cedge3, cedge4)
	row := GetRow(img, 145)
	vedge1, vedge2, vedge3, vedge4 := DetectEdges(row, 3, 10)
	fmt.Println(vedge1, vedge2, vedge3, vedge4)
	region1 := image.Rect(vedge1, cedge1, vedge2, cedge2)
	region2 := image.Rect(vedge3, cedge1, vedge4, cedge2)
	region3 := image.Rect(vedge1, cedge3, vedge2, cedge4)
	region4 := image.Rect(vedge3, cedge3, vedge4, cedge4)
	fmt.Println(region1.Size(), region2.Size(), region3.Size(), region4.Size())

	out := image.NewRGBA(img.Bounds())
	for x := 0; x < img.Bounds().Max.X; x++ {
		for y := 0; y < img.Bounds().Max.Y; y++ {
			out.Set(x, y, img.At(x, y))
		}
	}

	Draw(out, region1)
	Draw(out, region2)
	Draw(out, region3)
	Draw(out, region4)

	f, err = os.OpenFile("test.png", os.O_CREATE | os.O_TRUNC, 0755)
	if err != nil {
		fmt.Println(err)
		return
	}
	err = png.Encode(f, out)
	if err != nil {
		fmt.Println(err)
		return
	}
}
