package scenery

import (
	"fmt"
	"math"

	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"
	mgl "github.com/go-gl/mathgl/mgl32"
)

type Camera struct {
	Position mgl.Vec3
	Lookat   mgl.Vec3
	Up       mgl.Vec3
	FOV      float32
	// width / height
	Ratio float32

	UniformEye  int32
	UniformRays [2][2]int32

	deltaX float32
	deltaY float32
	deltaZ float32
}

func NewCamera(width, height int) Camera {
	cam := Camera{
		Position: mgl.Vec3{3.0, 2.0, 7.0},
		Lookat:   mgl.Vec3{0.0, 0.5, 0.0},
		Up:       mgl.Vec3{0.0, 1.0, 0.0},
		FOV:      120.0,
		Ratio:    float32(width) / float32(height),

		UniformEye:  -1,
		UniformRays: [2][2]int32{{-1, -1}, {-1, -1}},

		deltaX: 0,
		deltaY: 0,
		deltaZ: 0,
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

func (cam *Camera) GetUniformLocations(program uint32) {
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			cam.UniformRays[i][j] = gl.GetUniformLocation(
				program,
				gl.Str(fmt.Sprintf("ray%d%d\x00", i, j)),
			)
		}
	}
	cam.UniformEye = gl.GetUniformLocation(program, gl.Str("eye\x00"))
}

func (cam *Camera) checkUniforms() error {
	for i := 0; i < 2; i++ {
		for j := 0; j < 2; j++ {
			if cam.UniformRays[i][j] == -1 {
				return fmt.Errorf("uniform location for ray%d%d is -1", i, j)
			}
		}
	}
	if cam.UniformEye == -1 {
		return fmt.Errorf("uniform location for eye is -1")
	}
	return nil
}

func (cam *Camera) SetUniforms() error {
	if err := cam.checkUniforms(); err != nil {
		return err
	}

	var cameraRays [2][2]mgl.Vec3
	forward := cam.forward()
	right := cam.right()
	tanPhi := float32(math.Tan(float64(mgl.DegToRad(cam.FOV / 2))))
	deltaX := right.Mul(forward.Len() * tanPhi)
	deltaY := cam.Up.Normalize().Mul(forward.Len() / cam.Ratio * tanPhi)

	cameraRays[0][0] = forward.Sub(deltaX).Sub(deltaY)
	cameraRays[1][0] = forward.Add(deltaX).Sub(deltaY)
	cameraRays[0][1] = forward.Sub(deltaX).Add(deltaY)
	cameraRays[1][1] = forward.Add(deltaX).Add(deltaY)

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

	return nil
}

func (cam *Camera) KeyCallback() glfw.KeyCallback {
	return func(w *glfw.Window, key glfw.Key, scancode int, action glfw.Action, mods glfw.ModifierKey) {
		var delta float32
		if action == glfw.Press {
			delta = 0.2
		} else {
			delta = 0
		}

		switch key {
		case glfw.KeyW:
			cam.deltaX = +delta
		case glfw.KeyS:
			cam.deltaX = -delta
		case glfw.KeyD:
			cam.deltaZ = +delta
		case glfw.KeyA:
			cam.deltaZ = -delta
		}
	}
}

func (cam *Camera) Update() {
	deltaVec := cam.forward().Mul(cam.deltaX).Add(cam.right().Mul(cam.deltaZ)).Add(cam.Up.Mul(cam.deltaY))
	cam.Position = cam.Position.Add(deltaVec)
	cam.Lookat = cam.Lookat.Add(deltaVec)
}
