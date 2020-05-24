package generators

import (
	"github.com/fogleman/gg"
	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font/gofont/goregular"
)

// function that generates JPG and PNG images with the provided text
// and save them into "/templates" directory
func GenerateJPGAndPNG(ssrfToken string) {
	const W = 1024
	const H = 768

	dc := gg.NewContext(W, H)
	dc.SetRGB(0, 0, 0)
	dc.Clear()
	dc.SetRGB(1, 1, 1)
	font, err := truetype.Parse(goregular.TTF)
	if err != nil {
		panic("")
	}
	face := truetype.NewFace(font, &truetype.Options{
		Size: 14,
	})
	dc.SetFontFace(face)
	dc.DrawStringAnchored(ssrfToken,  W/2, H/2, 0.5, 0.5)


	dc.SaveJPG("./templates/jpeg.jpg", 80)
	dc.SavePNG("./templates/png.png")
}