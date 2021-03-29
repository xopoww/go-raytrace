package main

import (
	"log"
	"runtime"

	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/xopoww/go-raytrace/glutils"
	"github.com/xopoww/go-raytrace/scenery"
)

func init() {
	// This is needed to arrange that main() runs on main thread.
	// See documentation for functions that are only allowed to be called from the main thread.
	runtime.LockOSThread()
}

const (
	WIDTH  = 640
	HEIGHT = 480
)

var (
	quad = []float32{
		-1, -1, 0, // top
		1, -1, 0, // left
		-1, 1, 0, // right
		1, -1, 0, // top
		-1, 1, 0, // left
		1, 1, 0, // right
	}
)

func main() {

	// Initialize GLFW and GL, create window
	err := glfw.Init()
	if err != nil {
		panic(err)
	}
	defer glfw.Terminate()

	glfw.WindowHint(glfw.Resizable, glfw.False)
	glfw.WindowHint(glfw.ContextVersionMajor, 4)
	glfw.WindowHint(glfw.ContextVersionMinor, 6)
	glfw.WindowHint(glfw.OpenGLProfile, glfw.OpenGLCoreProfile)
	glfw.WindowHint(glfw.OpenGLForwardCompatible, glfw.True)

	window, err := glfw.CreateWindow(WIDTH, HEIGHT, "Go Ray Tracer", nil, nil)
	if err != nil {
		panic(err)
	}
	window.MakeContextCurrent()
	glfw.SwapInterval(1)

	// Initialize Glow
	if err := gl.Init(); err != nil {
		panic(err)
	}

	version := gl.GoStr(gl.GetString(gl.VERSION))
	log.Println("OpenGL version", version)

	// Create the program with single compute shader
	// TODO: make paths relative from glsl_scripts folder and automatic prefix generation
	compShaderSrc, err := glutils.NewShaderSource("../glsl_scripts/raytrace.glsl", gl.COMPUTE_SHADER)
	if err != nil {
		log.Fatalf("Failed to load compute shader source: %s", err)
	}
	compProgram, err := glutils.CreateProgram(compShaderSrc)
	if err != nil {
		log.Fatalf("Failed to create comp program: %s", err)
	}
	log.Println("Created the comp program")

	// Do the same for the quad shaders
	vertShaderSrc, err := glutils.NewShaderSource("../glsl_scripts/vert.glsl", gl.VERTEX_SHADER)
	if err != nil {
		log.Fatalf("Failed to load vertex shader source: %s", err)
	}
	fragShaderSrc, err := glutils.NewShaderSource("../glsl_scripts/frag.glsl", gl.FRAGMENT_SHADER)
	if err != nil {
		log.Fatalf("Failed to load vertex shader source: %s", err)
	}
	quadProgram, err := glutils.CreateProgram(vertShaderSrc, fragShaderSrc)
	if err != nil {
		log.Fatalf("Failed to create quad program: %s", err)
	}
	log.Println("Created the quad program")

	// Init the camera
	camera := scenery.NewCamera(WIDTH, HEIGHT)
	window.SetKeyCallback(camera.KeyCallback())

	// Init OpenGL objects
	vao := glutils.MakeVao(quad)
	texture := glutils.MakeEmptyTexture(WIDTH, HEIGHT)

	// Get uniform locations from programs
	gl.UseProgram(compProgram)
	uniformTime := gl.GetUniformLocation(compProgram, gl.Str("u_time\x00"))
	camera.GetUniformLocations(compProgram)

	gl.UseProgram(quadProgram)
	gl.Uniform1i(gl.GetUniformLocation(quadProgram, gl.Str("tex\x00")), 0)
	gl.UseProgram(0)

	glfw.SetTime(0.0)
	// Main loop
	for !window.ShouldClose() {

		// Dispatch compute shader program
		gl.UseProgram(compProgram)
		// set time
		if uniformTime != -1 {
			gl.Uniform1f(uniformTime, float32(glfw.GetTime()))
		}
		// update camera uniforms
		if err := camera.SetUniforms(); err != nil {
			log.Fatalf("Failed to set camera uniforms: %s", err)
		}
		gl.BindTexture(gl.TEXTURE_2D, texture)
		gl.BindImageTexture(0, texture, 0, false, 0, gl.READ_WRITE, gl.RGBA32F)
		gl.DispatchCompute(WIDTH, HEIGHT, 1) // TODO: add support of other workgroup sizes
		gl.BindImageTexture(0, 0, 0, false, 0, gl.READ_WRITE, gl.RGBA32F)
		gl.MemoryBarrier(gl.SHADER_IMAGE_ACCESS_BARRIER_BIT)
		gl.BindTexture(gl.TEXTURE_2D, 0)

		// Run fullscreen quad rendering program
		gl.UseProgram(quadProgram)
		gl.BindVertexArray(vao)
		gl.BindTexture(gl.TEXTURE_2D, texture)
		gl.DrawArrays(gl.TRIANGLES, 0, int32(len(quad)/3))
		gl.BindTexture(gl.TEXTURE_2D, 0)
		gl.UseProgram(0)

		window.SwapBuffers()
		glfw.PollEvents()

		camera.Update()
	}
}
