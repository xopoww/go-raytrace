package scenery

import (
	"fmt"
	"math"

	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	mgl "github.com/go-gl/mathgl/mgl32"

	"github.com/xopoww/go-raytrace/app"
	"github.com/xopoww/go-raytrace/glutils"
)

type Camera struct {
	Position mgl.Vec3
	Lookat   mgl.Vec3
	Up       mgl.Vec3
	FOV      float32
	// width / height
	Ratio float32

	Aperture  float32
	FocalDist float32

	UniformEye        int32
	UniformRays       [2][2]int32
	UniformLensRadius int32

	moveUp    bool
	moveDown  bool
	moveLeft  bool
	moveRight bool
	moveFor   bool
	moveBack  bool

	rotUp    bool
	rotDown  bool
	rotLeft  bool
	rotRight bool
	rotFor   bool // rotFor and rotBack are kind of misnomers as it is unclear what a "rotation forward" would be
	rotBack  bool // they're used just for similarity with moveFor and moveBack

	zoomIn     bool
	zoomOut    bool
	lensWide   bool
	lensShrink bool
	fovUp      bool
	fovDown    bool
}

func NewCamera(width, height int) Camera {
	cam := Camera{
		Position: mgl.Vec3{3.0, 2.0, 7.0},
		Lookat:   mgl.Vec3{-2.0, 0.5, 0.0},
		Up:       mgl.Vec3{0.0, 1.0, 0.0},
		FOV:      90.0,
		Ratio:    float32(width) / float32(height),

		Aperture:  0.5,
		FocalDist: 3.0,

		UniformEye:  -1,
		UniformRays: [2][2]int32{{-1, -1}, {-1, -1}},
	}
	cam.fixValues()
	return cam
}

// check if all fields of the struct are valid and fix if not
// TODO: maybe fix this somehow else
func (cam *Camera) fixValues() {
	forward := cam.forward()
	// remove Up's projection on forward so they're prependicular
	cam.Up = cam.Up.Sub(forward.Mul(forward.Dot(cam.Up)))
}

// Camera.Forward(), Camera.Up and Camera.Right() create the right
// orthonormal basis associated with the camera
func (cam *Camera) forward() mgl.Vec3 {
	return cam.Lookat.Sub(cam.Position).Normalize()
}

// Camera.Forward(), Camera.Up and Camera.Right() create the right
// orthonormal basis associated with the camera
func (cam *Camera) right() mgl.Vec3 {
	return cam.forward().Cross(cam.Up).Normalize()
}

func (cam *Camera) transformMatrix() mgl.Mat3 {
	return mgl.Mat3FromCols(cam.forward(), cam.Up, cam.right())
}

func (cam *Camera) GetUniformLocations(program uint32) {
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			cam.UniformRays[i][j] = glutils.MustGetUniformLocation(program, fmt.Sprintf("ray%d%d", i, j))
		}
	}
	cam.UniformEye = glutils.MustGetUniformLocation(program, "eye")
	cam.UniformLensRadius = glutils.MustGetUniformLocation(program, "lens_radius")
}

func (cam *Camera) SetUniforms() {
	var cameraRays [2][2]mgl.Vec3
	forward := cam.forward()
	right := cam.right()
	tanPhi := float32(math.Tan(float64(mgl.DegToRad(cam.FOV / 2))))
	deltaX := right.Mul(forward.Len() * tanPhi)
	deltaY := cam.Up.Normalize().Mul(forward.Len() / cam.Ratio * tanPhi)

	// cosPhi := float32(math.Cos(float64(mgl.DegToRad(cam.FOV / 2))))
	// rayLength := cam.FocalDist * float32(math.Sqrt(float64(1.0+cosPhi*cosPhi*(1.0+1.0/cam.Ratio/cam.Ratio))))

	cameraRays[0][0] = forward.Sub(deltaX).Sub(deltaY).Normalize().Mul(cam.FocalDist)
	cameraRays[1][0] = forward.Add(deltaX).Sub(deltaY).Normalize().Mul(cam.FocalDist)
	cameraRays[0][1] = forward.Sub(deltaX).Add(deltaY).Normalize().Mul(cam.FocalDist)
	cameraRays[1][1] = forward.Add(deltaX).Add(deltaY).Normalize().Mul(cam.FocalDist)

	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			gl.Uniform3f(
				cam.UniformRays[i][j],
				cameraRays[i][j].X(),
				cameraRays[i][j].Y(),
				cameraRays[i][j].Z(),
			)
		}
	}
	gl.Uniform3f(
		cam.UniformEye,
		cam.Position.X(),
		cam.Position.Y(),
		cam.Position.Z(),
	)

	gl.Uniform1f(cam.UniformLensRadius, cam.Aperture/2.0)
}

const (
	cameraSpeed    = 0.2
	cameraRotSpeed = 0.05
	apertureSpeed  = 0.01
	focalSpeed     = 0.5
	fovSpeed       = 1.0

	minFocalDist = 0.1
)

func (cam *Camera) Update() {
	var dX, dY, dZ float32

	if cam.moveUp {
		dY += cameraSpeed
	}
	if cam.moveDown {
		dY -= cameraSpeed
	}
	if cam.moveRight {
		dZ += cameraSpeed
	}
	if cam.moveLeft {
		dZ -= cameraSpeed
	}
	if cam.moveFor {
		dX += cameraSpeed
	}
	if cam.moveBack {
		dX -= cameraSpeed
	}

	deltaVec := cam.forward().Mul(dX).Add(cam.right().Mul(dZ)).Add(cam.Up.Mul(dY))

	if deltaVec.Len() > 0.0 {
		cam.Position = cam.Position.Add(deltaVec)
		cam.Lookat = cam.Lookat.Add(deltaVec)
	}

	var dPhiY, dPhiZ, dPhiX float32

	if cam.rotRight {
		dPhiY += cameraRotSpeed
	}
	if cam.rotLeft {
		dPhiY -= cameraRotSpeed
	}
	if cam.rotUp {
		dPhiZ -= cameraRotSpeed
	}
	if cam.rotDown {
		dPhiZ += cameraRotSpeed
	}
	if cam.rotFor {
		dPhiX -= cameraRotSpeed
	}
	if cam.rotBack {
		dPhiX += cameraRotSpeed
	}

	A := mgl.Ident3()
	if dPhiY != 0.0 {
		A = A.Mul3(mgl.Rotate3DY(dPhiY))
	}
	if dPhiZ != 0.0 {
		A = A.Mul3(mgl.Rotate3DZ(dPhiZ))
	}
	if dPhiX != 0.0 {
		A = A.Mul3(mgl.Rotate3DX(dPhiX))
	}
	if dPhiX != 0.0 || dPhiY != 0.0 || dPhiZ != 0.0 {
		T := cam.transformMatrix()
		M := T.Mul3(A).Mul3(T.Inv()).Transpose()

		newForward := M.Mul3x1(cam.Lookat.Sub(cam.Position))
		cam.Lookat = cam.Position.Add(newForward)
		cam.Up = M.Mul3x1(cam.Up)
	}

	var dAperture, dFocal float32

	if cam.zoomIn {
		dFocal += focalSpeed
	}
	if cam.zoomOut {
		dFocal -= focalSpeed
	}
	if cam.lensWide {
		dAperture += apertureSpeed
	}
	if cam.lensShrink {
		dAperture -= apertureSpeed
	}

	if cam.FocalDist+dFocal >= minFocalDist {
		cam.FocalDist += dFocal
	}
	if cam.Aperture+dAperture >= 0.0 {
		cam.Aperture += dAperture
	}

	var dFOV float32
	if cam.fovUp {
		dFOV += fovSpeed
	}
	if cam.fovDown {
		dFOV -= fovSpeed
	}
	if 5.0 < cam.FOV+dFOV && cam.FOV+dFOV < 180.0 {
		cam.FOV += dFOV
	}
}

func (cam *Camera) AttachToEventHandler(eh *app.EventHandler) {
	eh.AddOption(glfw.KeyW, &cam.moveFor, app.Hold)
	eh.AddOption(glfw.KeyS, &cam.moveBack, app.Hold)
	eh.AddOption(glfw.KeyD, &cam.moveRight, app.Hold)
	eh.AddOption(glfw.KeyA, &cam.moveLeft, app.Hold)
	eh.AddOption(glfw.KeySpace, &cam.moveUp, app.Hold)
	eh.AddOption(glfw.KeyLeftShift, &cam.moveDown, app.Hold)

	eh.AddOption(glfw.KeyKP8, &cam.rotUp, app.Hold)
	eh.AddOption(glfw.KeyKP2, &cam.rotDown, app.Hold)
	eh.AddOption(glfw.KeyKP6, &cam.rotRight, app.Hold)
	eh.AddOption(glfw.KeyKP4, &cam.rotLeft, app.Hold)
	eh.AddOption(glfw.KeyKP9, &cam.rotFor, app.Hold)
	eh.AddOption(glfw.KeyKP7, &cam.rotBack, app.Hold)

	eh.AddOption(glfw.KeyKPAdd, &cam.zoomIn, app.Hold)
	eh.AddOption(glfw.KeyKPSubtract, &cam.zoomOut, app.Hold)
	eh.AddOption(glfw.KeyX, &cam.lensWide, app.Hold)
	eh.AddOption(glfw.KeyZ, &cam.lensShrink, app.Hold)
	eh.AddOption(glfw.KeyV, &cam.fovUp, app.Hold)
	eh.AddOption(glfw.KeyC, &cam.fovDown, app.Hold)
}
