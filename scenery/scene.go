package scenery

import (
	"encoding/json"
	"fmt"
	"image/color"
	"math"
	"math/rand"

	mgl "github.com/go-gl/mathgl/mgl32"
)

type Scene struct {
	Data [2]struct {
		Num       uint32
		Descs     []string
		Colors    []string
		Materials []MaterialKind
		Fuzzs     []float32
		Etas      []float32
	}
}

func (s *Scene) GetObjectDesription(index int32) string {
	if index == -1 {
		return "nothing"
	}

	for body, data := range s.Data {
		if index >= int32(data.Num) {
			index -= int32(data.Num)
			continue
		}

		bodyS := [...]string{"box", "ball"}[body]
		materialS := [...]string{"mirror", "lambertian", "glass"}[data.Materials[index]]

		result := fmt.Sprintf("a %s %s with (color = %s", materialS, bodyS, data.Colors[index])
		if data.Materials[index] != Lambertian {
			result += fmt.Sprintf(
				", eta = %f, fuzz = %f",
				data.Etas[index],
				data.Fuzzs[index],
			)
		}
		return result + ")"
	}

	return "[invalid object index]"
}

func (s *Scene) UnmarshalJSON(data []byte) error {
	objects := make([]Object, 0)
	err := json.Unmarshal(data, &objects)
	if err != nil {
		return err
	}

	for _, obj := range objects {
		s.AddObject(obj)
	}

	return nil
}

func NewScene() *Scene {
	return &Scene{}
}

func (s *Scene) AddObject(o Object) {
	s.Data[o.Body_.kind].Num++
	s.Data[o.Body_.kind].Descs = append(s.Data[o.Body_.kind].Descs, o.Body_.desc)
	s.Data[o.Body_.kind].Colors = append(s.Data[o.Body_.kind].Colors, colorToString(o.Material_.color))
	s.Data[o.Body_.kind].Materials = append(s.Data[o.Body_.kind].Materials, o.Material_.kind)
	s.Data[o.Body_.kind].Fuzzs = append(s.Data[o.Body_.kind].Fuzzs, o.Material_.fuzz)
	s.Data[o.Body_.kind].Etas = append(s.Data[o.Body_.kind].Etas, o.Material_.eta)
}

// TODO: fix field/type naming
type Object struct {
	Body_     Body     `json:"body"`
	Material_ Material `json:"material"`
}

func NewObject(body Body, material Material) Object {
	return Object{
		Body_:     body,
		Material_: material,
	}
}

// Materials (optical properties of the object)

type MaterialKind int

const (
	Mirror MaterialKind = iota
	Lambertian
	Glass
)

func (mk MaterialKind) String() string {
	return [...]string{"Mirror", "Lambertian", "Glass"}[mk] + "Material"
}

type Material struct {
	kind  MaterialKind
	color color.RGBA
	fuzz  float32
	eta   float32
}

func (m *Material) UnmarshalJSON(data []byte) error {
	dict := make(map[string]interface{})
	err := json.Unmarshal(data, &dict)
	if err != nil {
		return err
	}

	kindI, found := dict["kind"]
	if !found {
		return fmt.Errorf("kind not specified")
	}
	kindS, ok := kindI.(string)
	if !ok {
		return fmt.Errorf("invalid kind type")
	}
	switch kindS {
	case "mirror":
		m.kind = Mirror
	case "lambertian":
		m.kind = Lambertian
	case "glass":
		m.kind = Glass
	default:
		return fmt.Errorf("unknown kind: %s", kindS)
	}

	clrI, found := dict["color"]
	if !found {
		return fmt.Errorf("color not specified")
	}
	clrS, ok := clrI.(string)
	if !ok {
		return fmt.Errorf("invalid color type")
	}
	_, err = fmt.Sscanf(clrS, "%02x%02x%02x", &m.color.R, &m.color.G, &m.color.B)
	if err != nil {
		return fmt.Errorf("failed to parse color: %w", err)
	}

	if m.kind != Lambertian {
		fzI, found := dict["fuzz"]
		if !found {
			return fmt.Errorf("fuzz not specified")
		}
		fzF, ok := fzI.(float64)
		if !ok {
			return fmt.Errorf("invalid fuzz type")
		}
		if fzF < 0.0 || fzF > 1.0 {
			return fmt.Errorf("fuzz must be in range [0, 1]")
		}
		m.fuzz = float32(fzF)

		etaI, found := dict["eta"]
		if !found {
			return fmt.Errorf("eta not specified")
		}
		etaF, ok := etaI.(float64)
		if !ok {
			return fmt.Errorf("invalid eta type")
		}
		if etaF < 0.0 {
			return fmt.Errorf("eta must be positive")
		}
		m.eta = float32(etaF)
	}

	return nil
}

func NewMirror(c color.RGBA, fuzz, eta float32) Material {
	return Material{
		kind:  Mirror,
		color: c,
		fuzz:  fuzz,
		eta:   eta,
	}
}

func NewLambertian(c color.RGBA) Material {
	return Material{
		kind:  Lambertian,
		color: c,
		fuzz:  0,
		eta:   0.0,
	}
}

func NewGlass(c color.RGBA, fuzz, eta float32) Material {
	return Material{
		kind:  Glass,
		color: c,
		fuzz:  fuzz,
		eta:   eta,
	}
}

func uiToF(i uint8) float32 {
	return float32(i) / float32(0xff)
}

func colorToString(c color.RGBA) string {
	return fmt.Sprintf("{%f, %f, %f}", uiToF(c.R), uiToF(c.G), uiToF(c.B))
}

// Bodies (geometry of the object)

type BodyKind int

const (
	Box BodyKind = iota
	Ball
)

type Body struct {
	kind BodyKind
	desc string
}

func parseBox(dict map[string]interface{}) (string, error) {
	minI, found := dict["min"]
	if !found {
		return "", fmt.Errorf("min not specified")
	}
	min, err := vec3FromInterface(minI)
	if err != nil {
		return "", fmt.Errorf("min: %w", err)
	}

	maxI, found := dict["max"]
	if !found {
		return "", fmt.Errorf("max not specified")
	}
	max, err := vec3FromInterface(maxI)
	if err != nil {
		return "", fmt.Errorf("max: %w", err)
	}

	return fmt.Sprintf(
		"{{%f, %f, %f},{%f, %f, %f}}",
		min[0], min[1], min[2],
		max[0], max[1], max[2],
	), nil
}

func vec3FromInterface(i interface{}) ([]float32, error) {
	arr, ok := i.([]interface{})
	if !ok || len(arr) != 3 {
		return nil, fmt.Errorf("not array of length 3")
	}
	vec := make([]float32, 3)
	for j := 0; j < 3; j++ {
		f, ok := arr[j].(float64)
		if !ok {
			return nil, fmt.Errorf("wrong type")
		}
		vec[j] = float32(f)
	}
	return vec, nil
}

func parseBall(dict map[string]interface{}) (string, error) {
	cI, found := dict["center"]
	if !found {
		return "", fmt.Errorf("center not specified")
	}
	c, err := vec3FromInterface(cI)
	if err != nil {
		return "", fmt.Errorf("center: %w", err)
	}

	rI, found := dict["radius"]
	if !found {
		return "", fmt.Errorf("radius not specified")
	}
	r, ok := rI.(float64)
	if !ok {
		return "", fmt.Errorf("invalid radius type")
	}
	if r < 0.0 {
		return "", fmt.Errorf("radius must be positive")
	}

	return fmt.Sprintf(
		"{{%f, %f, %f}, %f}",
		c[0], c[1], c[2],
		float32(r),
	), nil
}

func (b *Body) UnmarshalJSON(data []byte) error {
	dict := make(map[string]interface{})
	err := json.Unmarshal(data, &dict)
	if err != nil {
		return err
	}

	kindI, found := dict["kind"]
	if !found {
		return fmt.Errorf("kind not specified")
	}
	kindS, ok := kindI.(string)
	if !ok {
		return fmt.Errorf("invalid kind type")
	}
	switch kindS {
	case "box":
		b.kind = Box
		b.desc, err = parseBox(dict)
	case "ball":
		b.kind = Ball
		b.desc, err = parseBall(dict)
	default:
		return fmt.Errorf("unknown kind: %s", kindS)
	}

	return err
}

func NewBox(min, max mgl.Vec3) Body {
	return Body{
		kind: Box,
		desc: fmt.Sprintf(
			"{{%f, %f, %f},{%f, %f, %f}}",
			min.X(), min.Y(), min.Z(),
			max.X(), max.Y(), max.Z(),
		),
	}
}

func NewBall(center mgl.Vec3, radius float32) Body {
	return Body{
		kind: Ball,
		desc: fmt.Sprintf(
			"{{%f, %f, %f}, %f}",
			center.X(), center.Y(), center.Z(),
			radius,
		),
	}
}

const (
	nObjects = 35
	maxSize  = 5.0
	minSize  = 0.5
	maxDist  = 70.0
)

func RandomScene(seed int64) *Scene {
	rand.Seed(seed)
	s := NewScene()
	s.AddObject(NewObject(
		NewBox(
			mgl.Vec3{-maxDist, -1.0, -maxDist},
			mgl.Vec3{maxDist, 0.0, maxDist},
		),
		NewLambertian(
			color.RGBA{0x66, 0x66, 0x66, 0xFF},
		),
	))
	s.AddObject(NewObject(
		NewBox(
			mgl.Vec3{-maxDist, 0.0, -maxDist},
			mgl.Vec3{-maxDist + 1.0, maxDist / 2.0, maxDist},
		),
		NewLambertian(
			color.RGBA{0xDD, 0xDD, 0xDD, 0xFF},
		),
	))
	s.AddObject(NewObject(
		NewBox(
			mgl.Vec3{-maxDist, 0.0, maxDist - 1.0},
			mgl.Vec3{maxDist, maxDist / 2.0, maxDist},
		),
		NewLambertian(
			color.RGBA{0xDD, 0xDD, 0xDD, 0xFF},
		),
	))
	for i := 0; i < nObjects; i++ {
		pos := mgl.Vec2{rand.Float32() - 0.5, rand.Float32() - 0.5}.Mul(maxDist * 2.0)
		size := minSize + rand.Float32()*(maxSize-minSize)

		var body Body
		if rand.Int()%2 == 0 {
			body = NewBox(
				mgl.Vec3{pos.X() - size, 0.0, pos.Y() - size},
				mgl.Vec3{pos.X() + size, 2.0 * size, pos.Y() + size},
			)
		} else {
			body = NewBall(
				mgl.Vec3{pos.X(), size, pos.Y()}, size,
			)
		}

		clr := color.RGBA{
			R: uint8(rand.Float32() * 0xFF),
			G: uint8(rand.Float32() * 0xFF),
			B: uint8(rand.Float32() * 0xFF),
			A: 0xFF,
		}
		var material Material
		switch rand.Int() % 3 {
		case 0:
			material = NewMirror(clr, rand.Float32(), rand.Float32())
		case 1:
			material = NewLambertian(clr)
		case 2:
			material = NewGlass(clr, float32(math.Pow(rand.Float64()*0.5, 3.0)), rand.Float32()/1.5+1.1)
		}

		s.AddObject(NewObject(body, material))
	}
	return s
}
