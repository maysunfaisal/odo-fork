package generator

// GeneratorData is a struct that implements the Generator interface
type GeneratorData struct {
}

// New returns an instance of GeneratorImpl
func New() Generator {
	return &GeneratorData{}
}
