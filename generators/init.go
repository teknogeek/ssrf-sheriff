package generators

// function that run all media files generators with the provided text
func InitMediaGenerators(ssrfToken string)  {
	GenerateJPGAndPNG(ssrfToken)
}
