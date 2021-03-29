package glutils

import (
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/go-gl/gl/v4.6-core/gl"
)

// Simple struct to hold information needed to compile a shader
// TODO: add template support for code injection
type ShaderSource struct {
	source     string
	shaderType uint32
}

// Load shader source code from text file
func NewShaderSource(path string, shaderType uint32) (ShaderSource, error) {
	file, err := os.Open(path)
	if err != nil {
		return ShaderSource{}, fmt.Errorf("source file %q not found on disk: %w", path, err)
	}

	data, err := ioutil.ReadAll(file)
	if err != nil {
		return ShaderSource{}, fmt.Errorf("failed to read from the file: %w", err)
	}

	return ShaderSource{
		source:     string(data) + "\x00",
		shaderType: shaderType,
	}, nil
}

func (ss ShaderSource) compile() (uint32, error) {
	shader := gl.CreateShader(ss.shaderType)

	csources, free := gl.Strs(ss.source)
	gl.ShaderSource(shader, 1, csources, nil)
	free()
	gl.CompileShader(shader)

	var status int32
	gl.GetShaderiv(shader, gl.COMPILE_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetShaderiv(shader, gl.INFO_LOG_LENGTH, &logLength)

		gllog := strings.Repeat("\x00", int(logLength+1))
		gl.GetShaderInfoLog(shader, logLength, nil, gl.Str(gllog))

		return 0, fmt.Errorf("failed to compile \n%v\n--------\n%v", ss.source, gllog)
	}

	return shader, nil
}

// Compile and link OpenGL program with shaders compiled from shaderSrcs
func CreateProgram(shaderSrcs ...ShaderSource) (uint32, error) {
	program := gl.CreateProgram()

	var shaders []uint32
	for i, shaderSrc := range shaderSrcs {
		shader, err := shaderSrc.compile()
		if err != nil {
			return 0, fmt.Errorf("shader #%d: %w", i, err)
		}
		gl.AttachShader(program, shader)
		shaders = append(shaders, shader)
	}

	gl.LinkProgram(program)

	var status int32
	gl.GetProgramiv(program, gl.LINK_STATUS, &status)
	if status == gl.FALSE {
		var logLength int32
		gl.GetProgramiv(program, gl.INFO_LOG_LENGTH, &logLength)

		gllog := strings.Repeat("\x00", int(logLength+1))
		gl.GetProgramInfoLog(program, logLength, nil, gl.Str(gllog))

		return 0, fmt.Errorf("link error: %s", gllog)
	}

	for _, shader := range shaders {
		gl.DeleteShader(shader)
	}

	return program, nil
}
