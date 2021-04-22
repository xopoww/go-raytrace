package glutils

import (
	"fmt"
	"image"
	"log"

	"github.com/go-gl/gl/v4.6-core/gl"
)

func CheckError() error {
	if errCode := gl.GetError(); errCode != 0 {
		return fmt.Errorf("OpenGL error %d", errCode)
	}
	return nil
}

// Initialize and return a vertex array from the points provided
func MakeVao(points []float32) uint32 {
	var vbo uint32
	gl.GenBuffers(1, &vbo)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.BufferData(gl.ARRAY_BUFFER, 4*len(points), gl.Ptr(points), gl.STATIC_DRAW)

	var vao uint32
	gl.GenVertexArrays(1, &vao)
	gl.BindVertexArray(vao)
	gl.EnableVertexAttribArray(0)
	gl.BindBuffer(gl.ARRAY_BUFFER, vbo)
	gl.VertexAttribPointer(0, 3, gl.FLOAT, false, 0, nil)
	return vao
}

// Convenience wrapper for gl.GetUniformLocation
func GetUniformLocation(program uint32, name string) int32 {
	return gl.GetUniformLocation(program, gl.Str(name+"\x00"))
}

// Same as GetUniformLocation, but calls log.Fatalf if the returned value is -1
func MustGetUniformLocation(program uint32, name string) int32 {
	ul := GetUniformLocation(program, name)
	if ul == -1 {
		log.Fatalf("Uniform location for %s is -1", name)
	}
	return ul
}

// Create new empty OpenGL texture with given size
// The texture is created with the type of RGBA32F
func MakeEmptyTexture(width, height int) uint32 {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	var texture uint32
	gl.GenTextures(1, &texture)
	gl.ActiveTexture(gl.TEXTURE0)
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.TexImage2D(
		gl.TEXTURE_2D,
		0,
		gl.RGBA32F,
		int32(width),
		int32(height),
		0,
		gl.RGBA,
		gl.UNSIGNED_BYTE,
		gl.Ptr(img.Pix),
	)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MIN_FILTER, gl.NEAREST)
	gl.TexParameteri(gl.TEXTURE_2D, gl.TEXTURE_MAG_FILTER, gl.NEAREST)
	gl.GenerateMipmap(gl.TEXTURE_2D)
	gl.BindTexture(gl.TEXTURE_2D, 0)

	return texture
}

// Copy pixel data from a texture to an image
// Safe to use only with textures created by MakeEmptyTexture
func GetImage(texture uint32, width, height int) (image.Image, error) {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	gl.BindTexture(gl.TEXTURE_2D, texture)
	gl.GetTexImage(gl.TEXTURE_2D, 0, gl.RGBA, gl.UNSIGNED_BYTE, gl.Ptr(img.Pix))
	gl.BindTexture(gl.TEXTURE_2D, 0)
	if err := CheckError(); err != nil {
		return nil, err
	}
	return img, nil
}

// Create a copy of src reflected along the vertical axis
func FlipImage(src image.Image) image.Image {
	dst := image.NewRGBA(src.Bounds())
	for i := 0; i < src.Bounds().Dx(); i++ {
		for j := 0; j < src.Bounds().Dy(); j++ {
			dst.Set(i, j, src.At(i, src.Bounds().Dy()-1-j))
		}
	}
	return dst
}
