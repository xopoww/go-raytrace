package shaders

//
// Embedded GLSL shader sources
//

import _ "embed"

//go:embed vert.glsl
var Vert string

//go:embed frag.glsl
var Frag string

//go:embed raytrace_template.glsl
var Comp string
