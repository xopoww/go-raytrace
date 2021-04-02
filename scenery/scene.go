package scenery

import (
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

func NewScene() *Scene {
	return &Scene{}
}

func (s *Scene) AddObject(o Object) {
	s.Data[o.body.kind].Num++
	s.Data[o.body.kind].Descs = append(s.Data[o.body.kind].Descs, o.body.desc)
	s.Data[o.body.kind].Colors = append(s.Data[o.body.kind].Colors, colorToString(o.material.color))
	s.Data[o.body.kind].Materials = append(s.Data[o.body.kind].Materials, o.material.kind)
	s.Data[o.body.kind].Fuzzs = append(s.Data[o.body.kind].Fuzzs, o.material.fuzz)
	s.Data[o.body.kind].Etas = append(s.Data[o.body.kind].Etas, o.material.eta)
}

type Object struct {
	body     Body
	material Material
}

func NewObject(body Body, material Material) Object {
	return Object{
		body:     body,
		material: material,
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

func NewMirror(c color.RGBA, fuzz float32) Material {
	return Material{
		kind:  Mirror,
		color: c,
		fuzz:  fuzz,
		eta:   0.0,
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
			color.RGBA{0x44, 0x44, 0x44, 0xFF},
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
			material = NewMirror(clr, rand.Float32())
		case 1:
			material = NewLambertian(clr)
		case 2:
			material = NewGlass(clr, float32(math.Pow(rand.Float64()*0.5, 3.0)), rand.Float32()/1.5+1.1)
		}

		s.AddObject(NewObject(body, material))
	}
	return s
}
