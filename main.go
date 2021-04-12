package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"image/png"
	"io/ioutil"
	"log"
	"os"
	"runtime"
	"time"

	"github.com/go-gl/gl/v4.6-core/gl"
	"github.com/go-gl/glfw/v3.3/glfw"

	"github.com/xopoww/go-raytrace/app"
	"github.com/xopoww/go-raytrace/glutils"
	"github.com/xopoww/go-raytrace/scenery"
)

func init() {
	// This is needed to arrange that main() runs on main thread.
	// See documentation for functions that are only allowed to be called from the main thread.
	runtime.LockOSThread()
}

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

	// Get command line arguments
	SCENE := flag.String("scene", "", "path to json file with scene description (if not set, a random scene will be generated)")
	SEED := flag.Int64("seed", -1, "seed for random scene generation (if negative, current UNIX time is used)")
	WIDTH := flag.Int("width", 640, "screen width in pixels")
	HEIGHT := flag.Int("height", 480, "scene height in pixels")
	RESOLUTION := flag.String("resolution", "", "if set, must be one of \"hd\" (1080x720) or \"fullhd\" (1920x1080); overrides width and height options")
	MONTE_CARLO_FRAME_COUNT := flag.Uint("mcfc", 20, "number of frames for monte carlo denoising")
	ANTI_ALIASING := flag.Uint("alias", 4, "anti-aliasing parameter")
	MAX_DEPTH := flag.Uint("depth", 10, "maximum recursion depth for ray tracing")

	flag.Parse()

	switch *RESOLUTION {
	case "":
		break
	case "hd":
		*WIDTH = 1080
		*HEIGHT = 720
	case "fullhd":
		*WIDTH = 1920
		*HEIGHT = 1080
	default:
		log.Fatalf("unknown resolution option: %s", *RESOLUTION)
	}

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

	window, err := glfw.CreateWindow(*WIDTH, *HEIGHT, "Go Ray Tracer", nil, nil)
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

	// Init the scene
	var scene *scenery.Scene

	if *SCENE == "" {
		if *SEED < 0 {
			*SEED = time.Now().Unix()
		}
		log.Printf("Generating random scene with seed %d", *SEED)
		scene = scenery.RandomScene(*SEED)
	} else {
		scene = scenery.NewScene()
		file, err := os.Open("scene.json")
		if err != nil {
			log.Fatalf("Failed to open scene file: %s", err)
		}
		data, err := ioutil.ReadAll(file)
		if err != nil {
			log.Fatalf("Failed to read scene file: %s", err)
		}
		err = json.Unmarshal(data, scene)
		if err != nil {
			log.Fatalf("Failed to parse scene file: %s", err)
		}
	}

	// Create the program with single compute shader
	// TODO: make paths relative from glsl_scripts folder and automatic prefix generation
	compShaderSrc, err := glutils.NewShaderSourceFromTemplate("../glsl_scripts/raytrace_template.glsl", gl.COMPUTE_SHADER, scene)
	if err != nil {
		log.Fatalf("Failed to load compute shader source: %s", err)
	}
	compProgram, err := glutils.CreateProgram(compShaderSrc)
	if err != nil {
		log.Fatalf("Failed to create comp program: %s", err)
	}

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

	// Init the event handler
	eventHandler := app.NewEventHandler()
	window.SetKeyCallback(eventHandler.KeyCallback())

	screenshotRequested := false
	eventHandler.AddOption(glfw.KeyF3, &screenshotRequested, app.Switch)

	lowGraphics := false
	eventHandler.AddOption(glfw.KeyP, &lowGraphics, app.Switch)

	infoRequested := false
	eventHandler.AddOption(glfw.KeyI, &infoRequested, app.Switch)

	focusRequested := false
	eventHandler.AddOption(glfw.KeyF, &focusRequested, app.Switch)

	// Init the camera
	camera := scenery.NewCamera(*WIDTH, *HEIGHT)
	camera.AttachToEventHandler(eventHandler)

	// Init OpenGL objects
	vao := glutils.MakeVao(quad)
	texture := glutils.MakeEmptyTexture(*WIDTH, *HEIGHT)

	var (
		ssbo         uint32
		lookatIndex  int32
		distToLookat float32
	)

	gl.GenBuffers(1, &ssbo)
	gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, ssbo)
	gl.BufferData(gl.SHADER_STORAGE_BUFFER, 8, nil, gl.DYNAMIC_READ)
	gl.BindBufferBase(gl.SHADER_STORAGE_BUFFER, 5, ssbo)
	gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, 0)

	// Get uniform locations from programs
	gl.UseProgram(compProgram)
	uniformTime := glutils.GetUniformLocation(compProgram, "u_time")
	uniformFrameI := glutils.MustGetUniformLocation(compProgram, "u_frame_i")
	uniformMCFC := glutils.MustGetUniformLocation(compProgram, "MONTE_CARLO_FRAME_COUNT")
	uniformAntiAliasing := glutils.MustGetUniformLocation(compProgram, "ANTI_ALIASING")
	uniformMaxDepth := glutils.MustGetUniformLocation(compProgram, "MAX_DEPTH")
	camera.GetUniformLocations(compProgram)

	gl.UseProgram(quadProgram)
	gl.Uniform1i(glutils.MustGetUniformLocation(quadProgram, "tex"), 0)
	gl.UseProgram(0)

	glfw.SetTime(0.0)
	frame_i := uint32(0)
	// Main loop
	for !window.ShouldClose() {

		// Dispatch compute shader program
		gl.UseProgram(compProgram)
		// set time and frame index
		if uniformTime != -1 {
			gl.Uniform1f(uniformTime, float32(glfw.GetTime()))
		}
		if uniformFrameI != -1 {
			gl.Uniform1ui(uniformFrameI, frame_i)
		}
		// update camera uniforms
		camera.SetUniforms()
		// set graphics options
		if lowGraphics {
			gl.Uniform1ui(uniformMCFC, 1)
			gl.Uniform1ui(uniformAntiAliasing, 1)
			gl.Uniform1ui(uniformMaxDepth, 2)
		} else {
			gl.Uniform1ui(uniformMCFC, uint32(*MONTE_CARLO_FRAME_COUNT))
			gl.Uniform1ui(uniformAntiAliasing, uint32(*ANTI_ALIASING))
			gl.Uniform1ui(uniformMaxDepth, uint32(*MAX_DEPTH))
		}
		gl.BindTexture(gl.TEXTURE_2D, texture)
		gl.BindImageTexture(0, texture, 0, false, 0, gl.READ_WRITE, gl.RGBA32F)
		gl.DispatchCompute(uint32(*WIDTH), uint32(*HEIGHT), 1) // TODO: add support of other workgroup sizes
		gl.BindImageTexture(0, 0, 0, false, 0, gl.READ_WRITE, gl.RGBA32F)
		gl.MemoryBarrier(gl.SHADER_IMAGE_ACCESS_BARRIER_BIT | gl.BUFFER_UPDATE_BARRIER_BIT)
		gl.BindTexture(gl.TEXTURE_2D, 0)

		drawNow := uint(frame_i)%(*MONTE_CARLO_FRAME_COUNT) == 0 || lowGraphics

		// Run fullscreen quad rendering program
		if drawNow {
			gl.UseProgram(quadProgram)
			gl.BindVertexArray(vao)
			gl.BindTexture(gl.TEXTURE_2D, texture)
			gl.DrawArrays(gl.TRIANGLES, 0, int32(len(quad)/3))
			gl.BindTexture(gl.TEXTURE_2D, 0)
			gl.UseProgram(0)
		}

		// Check for errors
		if err := glutils.CheckError(); err != nil {
			log.Fatalf("Fatal error occured: %s", err)
		}

		// Handle screenshot request
		if screenshotRequested && drawNow {
			screenshotRequested = false

			img, err := glutils.GetImage(texture, *WIDTH, *HEIGHT)
			if err != nil {
				log.Printf("Failed to take a screenshot: %s", err)
			} else {
				log.Println("Took a screenshot")
				go func() {
					flippedImg := glutils.FlipImage(img)

					filename := fmt.Sprintf(
						"screenshot_%s.png",
						time.Now().Format("02-01-2006_15:04:05"),
					)

					file, err := os.Create(filename)
					if err != nil {
						log.Printf("Failed to save a screenshot: %s", err)
						return
					}
					defer file.Close()

					err = png.Encode(file, flippedImg)
					if err != nil {
						log.Printf("Failed to save a screenshot: %s", err)
						return
					}

					log.Printf("Saved a screenshot as %q", filename)
				}()
			}
		}

		// Handle info request
		if infoRequested {
			gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, ssbo)
			gl.GetBufferSubData(gl.SHADER_STORAGE_BUFFER, 0, 4, gl.Ptr(&lookatIndex))
			gl.GetBufferSubData(gl.SHADER_STORAGE_BUFFER, 4, 4, gl.Ptr(&distToLookat))
			gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, 0)

			log.Printf("You are looking at %s", scene.GetObjectDesription(lookatIndex))

			infoRequested = false
		}

		// Handle autofocus
		if focusRequested {
			gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, ssbo)
			gl.GetBufferSubData(gl.SHADER_STORAGE_BUFFER, 0, 4, gl.Ptr(&lookatIndex))
			gl.GetBufferSubData(gl.SHADER_STORAGE_BUFFER, 4, 4, gl.Ptr(&distToLookat))
			gl.BindBuffer(gl.SHADER_STORAGE_BUFFER, 0)

			if lookatIndex != -1 {
				camera.FocalDist = distToLookat
				log.Printf("Focused at the object you are looking at (F = %f)", distToLookat)
			} else {
				log.Println("Cannot autofocus on nothing")
			}

			focusRequested = false
		}

		window.SwapBuffers()
		glfw.PollEvents()

		camera.Update()

		frame_i++
	}

}
