package scenery

import (
	"fmt"
	"image/color"

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

func NewGlasss(c color.RGBA, fuzz, eta float32) Material {
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
