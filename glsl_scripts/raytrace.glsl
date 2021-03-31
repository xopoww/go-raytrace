#version 460 core

layout(binding = 0, rgba32f) uniform image2D framebuffer;

// The camera specification
uniform vec3 eye;
uniform vec3 ray00;
uniform vec3 ray10;
uniform vec3 ray01;
uniform vec3 ray11;

// body structs definition
struct box {
  vec3 min;
  vec3 max;
};

struct ball {
  vec3 center;
  float radius;
};

// body instances declaration
#define NUM_BOXES 2
const box boxes[] = {
  {vec3(-5.0, -0.1, -5.0), vec3(5.0, 0.0, 5.0)}, // floor
  {vec3(-0.5, 0.0, -0.5), vec3(0.5, 1.0, 0.5)}   // cube
};

#define NUM_BALLS 1
const ball balls[] = {
  {vec3(-2.0, 0.7, 1), 0.7}
};

#define NUM_OBJECTS NUM_BOXES + NUM_BALLS


// body intersection functions
vec2 intersectBox(vec3 origin, vec3 dir, const box b) {
  vec3 tMin = (b.min - origin) / dir;
  vec3 tMax = (b.max - origin) / dir;
  vec3 t1 = min(tMin, tMax);
  vec3 t2 = max(tMin, tMax);
  float tNear = max(max(t1.x, t1.y), t1.z);
  float tFar = min(min(t2.x, t2.y), t2.z);
  return vec2(tNear, tFar);
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

vec2 intersectBall(vec3 origin, vec3 dir, const ball b) {
  float c1 = pow(length(dir), 2);
  float c2 = 2.0 * dot(origin - b.center, dir);
  float c3 = pow(length(origin - b.center), 2) - pow(b.radius, 2);
  return solve_quadratic(c1, c2, c3);
}


// global intersection function
#define MAX_SCENE_BOUNDS 1000.0

struct hitinfo {
  vec2 lambda;
  int oi;
};

bool intersectObjects(vec3 origin, vec3 dir, out hitinfo info) {
  float smallest = MAX_SCENE_BOUNDS;
  bool found = false;
  for (int i = 0; i < NUM_BOXES; i++) {
    vec2 lambda = intersectBox(origin, dir, boxes[i]);
    if (lambda.x > 0.0 && lambda.x < lambda.y && lambda.x < smallest) {
      info.lambda = lambda;
      info.oi = i;
      smallest = lambda.x;
      found = true;
    }
  }
  for (int i = 0; i < NUM_BALLS; i++) {
    vec2 lambda = intersectBall(origin, dir, balls[i]);
    if (lambda.x > 0.0 && lambda.x < lambda.y && lambda.x < smallest) {
      info.lambda = lambda;
      info.oi = i + NUM_BOXES;
      smallest = lambda.x;
      found = true;
    }
  }
  return found;
}


// materials
const vec3 colors[NUM_OBJECTS] = {
  {1.0, 1.0, 1.0},
  {1.0, 0.2, 0.2},
  {0.3, 1.0, 0.3}
};


// main tracing function
vec4 bg_color(vec3 origin, vec3 dir) {
  float brightness = dir.y / length(dir);
  return vec4(vec3(brightness).xyz, 1.0);
}

vec4 trace(vec3 origin, vec3 dir) {
  hitinfo i;
  if (intersectObjects(origin, dir, i)) {
    vec3 color = colors[i.oi];
    return vec4(color.xyz, 1.0);
  }
  return bg_color(origin, dir);
}


layout (local_size_x = 8, local_size_y = 8) in;

void main(void) {
  ivec2 pix = ivec2(gl_GlobalInvocationID.xy);
  ivec2 size = imageSize(framebuffer);
  if (pix.x >= size.x || pix.y >= size.y) {
    return;
  }
  vec2 pos = vec2(pix) / vec2(size.x, size.y);
  vec3 dir = mix(mix(ray00, ray01, pos.y), mix(ray10, ray11, pos.y), pos.x);
  vec4 color = trace(eye, dir);
  imageStore(framebuffer, pix, color);
}
