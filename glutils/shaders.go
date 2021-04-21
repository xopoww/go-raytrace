package glutils

import (
	"fmt"
	"strings"
	"text/template"

	"github.com/go-gl/gl/v4.6-core/gl"
)

// Simple struct to hold information needed to compile a shader
// TODO: add template support for code injection
type ShaderSource struct {
	source     string
	shaderType uint32
}

func NewShaderSource(source string, shaderType uint32) ShaderSource {
	return ShaderSource{
		source:     source + "\x00",
		shaderType: shaderType,
	}
}

// Create ShaderSource from go template source by injecting data into it
func NewShaderSourceFromTemplate(name, source string, shaderType uint32, data interface{}) (ShaderSource, error) {
	tmpl, err := template.New(name).Parse(source)
	if err != nil {
		return ShaderSource{}, fmt.Errorf("parse template %q: %w", name, err)
	}
	bldr := strings.Builder{}
	err = tmpl.Execute(&bldr, data)
	if err != nil {
		return ShaderSource{}, fmt.Errorf("execute template: %w", err)
	}
	return ShaderSource{
		source:     bldr.String() + "\x00",
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
