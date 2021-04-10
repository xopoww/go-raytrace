#version 460 core


// ===== Global compute shader params

layout(binding = 0, rgba32f) uniform image2D framebuffer;
layout (local_size_x = 1, local_size_y = 1) in;

ivec2 pix = ivec2(gl_GlobalInvocationID.xy);
ivec2 size = imageSize(framebuffer);

uniform float u_time;
uniform uint u_frame_i;

uniform uint MAX_DEPTH;
uniform uint ANTI_ALIASING;
uniform uint MONTE_CARLO_FRAME_COUNT;


// ===== Helper structs and functions

#define FLOAT_DELTA 0.0001

// floating point equality test
bool fleq(const float f1, const float f2) {
  return abs(f1 - f2) < FLOAT_DELTA;
}

// element-wise minimum of the vector
float elmin(vec3 a) {
  return min(a.x, min(a.y, a.z));
}

int argmin(vec3 a) {
  float m = elmin(a);
  if (a.x == m) {
    return 0;
  }
  if (a.y == m) {
    return 1;
  }
  return 2;
}

vec2 solve_quadratic(float a, float b, float c) {
  if (a == 0.0) {
    float k = -b / c;
    return vec2(k, k);
  }
  float D2 = b * b - 4 * a * c;
  if (D2 < 0.0) {
    return vec2(1.0 / 0.0, 1.0 / 0.0);
  }
  float x = -b / 2.0 / a;
  if (D2 == 0.0) {
    return vec2(x, x);
  }
  float delta = sqrt(D2) / 2.0 / a;
  return vec2(x - delta, x + delta);
}

// for debug
vec3 color_from_normal(vec3 normal) {
  vec3 sc = normalize(normal);
  return abs(sc);
}

struct ray3 {
  vec3 origin;
  vec3 dir;
};


// ===== Pseudo-random numbers generation

// these functions are copied from
// https://stackoverflow.com/questions/4200224/random-noise-functions-for-glsl
// vvvvvvvvvvvvvvvv

  // A single iteration of Bob Jenkins' One-At-A-Time hashing algorithm.
  uint hash( uint x ) {
      x += ( x << 10u );
      x ^= ( x >>  6u );
      x += ( x <<  3u );
      x ^= ( x >> 11u );
      x += ( x << 15u );
      return x;
  }

  // Compound versions of the hashing algorithm I whipped together.
  uint hash( uvec2 v ) { return hash( v.x ^ hash(v.y)                         ); }
  uint hash( uvec3 v ) { return hash( v.x ^ hash(v.y) ^ hash(v.z)             ); }
  uint hash( uvec4 v ) { return hash( v.x ^ hash(v.y) ^ hash(v.z) ^ hash(v.w) ); }

  // Construct a float with half-open range [0:1] using low 23 bits.
  // All zeroes yields 0.0, all ones yields the next smallest representable value below 1.0.
  float floatConstruct( uint m ) {
      const uint ieeeMantissa = 0x007FFFFFu; // binary32 mantissa bitmask
      const uint ieeeOne      = 0x3F800000u; // 1.0 in IEEE binary32

      m &= ieeeMantissa;                     // Keep only mantissa bits (fractional part)
      m |= ieeeOne;                          // Add fractional part to 1.0

      float  f = uintBitsToFloat( m );       // Range [1:2]
      return f - 1.0;                        // Range [0:1]
  }

// ^^^^^^^^^^^^^^^^
uint _hash_seed = 0;

float random() {
  float f = floatConstruct(hash(uvec4(_hash_seed, pix.xy, floatBitsToUint(u_time))));
  _hash_seed++;
  return f;
}

vec3 random_in_unit_sphere() {
  vec3 v;
  do {
    v = vec3(random(), random(), random()) * 2.0 - 1.0;
  } while (length(v) > 1.0);
  return v;
}

vec2 random_in_unit_disk() {
  vec2 v;
  do {
    v = vec2(random(), random()) * 2.0 - 1.0;
  } while (length(v) > 1.0);
  return v;
}


// ===== The camera specification

uniform vec3 eye;
uniform vec3 ray00;
uniform vec3 ray10;
uniform vec3 ray01;
uniform vec3 ray11;

uniform float lens_radius;


// ===== Body structs definition

struct box {
  vec3 min;
  vec3 max;
};

struct ball {
  vec3 center;
  float radius;
};


// ===== Body instances declaration

{{with index .Data 0}}
#define NUM_BOXES {{.Num}}
{{if .Num}}
const box boxes[NUM_BOXES] = {
  {{range .Descs}}
  {{.}},
  {{end}}
};
{{end}}
{{end}}

{{with index .Data 1}}
#define NUM_BALLS {{.Num}}
{{if .Num}}
const ball balls[NUM_BALLS] = {
  {{range .Descs}}
  {{.}},
  {{end}}
};
{{end}}
{{end}}

#define NUM_OBJECTS NUM_BOXES + NUM_BALLS


// ===== Body intersection functions

vec2 _intersectBox(vec3 origin, vec3 dir, const box b) {
  vec3 tMin = (b.min - origin) / dir;
  vec3 tMax = (b.max - origin) / dir;
  vec3 t1 = min(tMin, tMax);
  vec3 t2 = max(tMin, tMax);
  float tNear = max(max(t1.x, t1.y), t1.z);
  float tFar = min(min(t2.x, t2.y), t2.z);
  return vec2(tNear, tFar);
}

vec3 _normalBox(vec3 point, const box b) {
  vec3 dMin = abs(point - b.min);
  vec3 dMax = abs(point - b.max);
  vec3 norm, mask = vec3(0.0);
  vec3 d;
  if (elmin(dMin) < elmin(dMax)) {
    norm = vec3(-1.0);
    d = dMin;
  } else {
    norm = vec3(1.0);
    d = dMax;
  }
  switch (argmin(d)) {
  case 0:
    mask.x = 1.0;
    break;
  case 1:
    mask.y = 1.0;
    break;
  case 2:
    mask.z = 1.0;
    break;
  }
  return norm * mask;
}

vec2 _intersectBall(vec3 origin, vec3 dir, const ball b) {
  float c1 = pow(length(dir), 2);
  float c2 = 2.0 * dot(origin - b.center, dir);
  float c3 = pow(length(origin - b.center), 2) - pow(b.radius, 2);
  return solve_quadratic(c1, c2, c3);
}

vec3 _normalBall(vec3 point, const ball b) {
  return normalize(point - b.center);
}


// ==== Global intersection function

#define MAX_SCENE_BOUNDS 1000.0

struct hitinfo {
  vec2 lambda;
  int oi;
};

bool intersectObjects(vec3 origin, vec3 dir, out hitinfo info) {
  float smallest = MAX_SCENE_BOUNDS;
  bool found = false;
  
  {{with index .Data 0}}
  {{if .Num}}
  // handle boxes
  for (int i = 0; i < NUM_BOXES; i++) {
    vec2 lambda = _intersectBox(origin, dir, boxes[i]);
    if (lambda.x > 0.0 && lambda.x < lambda.y && lambda.x < smallest) {
      info.lambda = lambda;
      info.oi = i;
      smallest = lambda.x;
      found = true;
    }
  }
  {{end}}
  {{end}}

  {{with index .Data 1}}
  {{if .Num}}
  // handle balls
  for (int i = 0; i < NUM_BALLS; i++) {
    vec2 lambda = _intersectBall(origin, dir, balls[i]);
    if (lambda.x > 0.0 && lambda.x < lambda.y && lambda.x < smallest) {
      info.lambda = lambda;
      info.oi = i + NUM_BOXES;
      smallest = lambda.x;
      found = true;
    }
  }
  {{end}}
  {{end}}
  return found;
}

vec3 normalObject(vec3 point, int oi) {
  if (oi < NUM_BOXES) {
    {{with index .Data 0}}{{if .Num}}
    return _normalBox(point, boxes[oi]);
    {{end}}{{end}}
  } else {
    {{with index .Data 1}}{{if .Num}}
    return _normalBall(point, balls[oi - NUM_BOXES]);
    {{end}}{{end}}
  }
}


// ==== Materials

const vec3 colors[NUM_OBJECTS] = {
  {{range .Data}}
  {{range .Colors}}
  {{.}},
  {{end}}
  {{end}}
};

const float fuzzs[NUM_OBJECTS] = {
  {{range .Data}}
  {{range .Fuzzs}}
  {{.}},
  {{end}}
  {{end}}
};

const float etas[NUM_OBJECTS] = {
  {{range .Data}}
  {{range .Etas}}
  {{.}},
  {{end}}
  {{end}}
};

const uint MirrorMaterial     = 0x00000001u;
const uint LambertianMaterial = 0x00000002u;
const uint GlassMaterial      = 0x00000003u;

const uint materials[NUM_OBJECTS] = {
  {{range .Data}}
  {{range .Materials}}
  {{.}},
  {{end}}
  {{end}}
};

vec3 _scatterLambertian(vec3 normal) {
  vec3 scattered = normal + random_in_unit_sphere();
  if (fleq(length(scattered), 0.0)) {
    scattered = normal;
  }
  return scattered;
}

vec3 _scatterMirror(vec3 incident, vec3 normal, float fuzz, float eta) {
  if (random() <= eta) {
    return reflect(incident, normal) + random_in_unit_sphere() * fuzz;
  } else {
    return _scatterLambertian(normal);
  }
}

vec3 _scatterGlass(vec3 incident, vec3 normal, float fuzz, float eta) {
  vec3 scattered = refract(normalize(incident), normal, 1.0 / eta);
  if (fleq(length(scattered), 0.0)) {
    scattered = reflect(incident, normal);
  }
  return scattered + random_in_unit_sphere() * fuzz;
}

vec3 scatter(vec3 incident, vec3 normal, int oi) {
  switch (materials[oi]) {
  case MirrorMaterial:
    return _scatterMirror(incident, normal, fuzzs[oi], etas[oi]);
  case LambertianMaterial:
    return _scatterLambertian(normal);
  case GlassMaterial:
    float eta = etas[oi];
    if (dot(incident, normal) > 0.0) {
      normal *= -1.0;
      eta = 1.0 / eta;
    }
    return _scatterGlass(incident, normal, fuzzs[oi], eta);
  default:
    return vec3(0.0);
  }
}


// ===== Main tracing functions

vec3 bg_color(vec3 dir) {
  float brightness = (dir.y / length(dir) + 1.0) / 2.0;
  return vec3(brightness);
}

ray3 trace_step(ray3 r, out vec3 color) {
  hitinfo i;
  if (intersectObjects(r.origin, r.dir, i)) {
    vec3 point = r.origin + r.dir * i.lambda.x;
    vec3 normal = normalObject(point, i.oi);
    color = colors[i.oi];
    vec3 scattered = scatter(r.dir, normal, i.oi);
    if (fleq(length(scattered), 0.0)) {
      color = vec3(0.0);
      return ray3(vec3(0.0), vec3(0.0));
    }
    return ray3(point, normalize(scattered));
  }
  color = bg_color(r.dir);
  return ray3(vec3(0.0), vec3(0.0));
}

vec3 trace_ray(ray3 ray) {
  vec3 resulting_color = vec3(1.0);
  for (int i = 0; i <= MAX_DEPTH; i++) {
    if (i == MAX_DEPTH) {
      return vec3(0.0, 0.0, 0.0);
    }
    vec3 color;
    ray = trace_step(ray, color);
    resulting_color *= color;
    if (fleq(length(ray.dir), 0.0)) {
      break;
    }
  }
  return resulting_color;
}

ray3 get_ray(vec2 pos) {
  vec2 eye_shift = random_in_unit_disk().xy * lens_radius;
  vec3 h = normalize(ray11 - ray01);
  vec3 v = normalize(ray11 - ray10);
  vec3 offset = eye_shift.x * h + eye_shift.y * v;
  vec3 dir = mix(mix(ray00 - offset, ray01- offset, pos.y), mix(ray10 - offset, ray11 - offset, pos.y), pos.x);
  return ray3(eye + offset, dir);
}

// ===== Main

void main(void) {
  if (pix.x >= size.x || pix.y >= size.y) {
    return;
  }

  vec3 resulting_color = vec3(0.0);
  for (int i = 0; i < ANTI_ALIASING; i++) {
    vec2 shift = vec2(random(), random());
    vec2 pos = (vec2(pix) + shift) / vec2(size.x, size.y);
    ray3 ray = get_ray(pos);
    vec3 color = trace_ray(ray);
    resulting_color += color / float(ANTI_ALIASING);
  }
  vec4 prev_color;
  if (MONTE_CARLO_FRAME_COUNT == 1 || u_frame_i % MONTE_CARLO_FRAME_COUNT == 1) {
    prev_color = vec4(0.0);
  } else {
    prev_color = imageLoad(framebuffer, pix);
  }
  prev_color += vec4(resulting_color.xyz, 1.0) / float(MONTE_CARLO_FRAME_COUNT);

  imageStore(framebuffer, pix, prev_color);
}
