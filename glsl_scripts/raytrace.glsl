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
  {vec3(-5.0, -0.5, -5.0), vec3(5.0, 0.0, 5.0)}, // floor
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

vec3 normalBox(vec3 point, const box b) {
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

vec3 normalBall(vec3 point, const ball b) {
  return normalize(point - b.center);
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

vec3 normalObject(vec3 point, int oi) {
  if (oi < NUM_BOXES) {
    return normalBox(point, boxes[oi]);
  } else {
    return normalBall(point, balls[oi - NUM_BOXES]);
  }
}


// for debug
vec3 color_from_normal(vec3 normal) {
  vec3 sc = normalize(normal);
  return abs(sc);
}

// materials
const vec3 colors[] = {
  {0.8, 0.8, 0.4},
  {1.0, 0.2, 0.2},
  {0.3, 1.0, 0.3}
};

const uint MirrorMaterial = 0x00000001u;

vec3 scatterMirror(vec3 incident, vec3 normal) {
  return incident + 2.0 * normal * abs(dot(incident, normal));
}

// main tracing function
vec3 bg_color(vec3 origin, vec3 dir) {
  float brightness = (dir.y / length(dir) + 1.0) / 2.0;
  return vec3(brightness);
}

#define MAX_DEPTH 7

struct ray3 {
  vec3 origin;
  vec3 dir;
};

ray3 trace_step(ray3 r, out vec3 color) {
  hitinfo i;
  if (intersectObjects(r.origin, r.dir, i)) {
    vec3 point = r.origin + r.dir * i.lambda.x;
    vec3 normal = normalObject(point, i.oi);
    color = colors[i.oi];
    return ray3(point, scatterMirror(r.dir, normal));
  }
  color = bg_color(r.origin, r.dir);
  return ray3(vec3(0.0, 0.0, 0.0), vec3(0.0, 0.0, 0.0));
}

vec3 trace(ray3 ray) {
  vec3 resulting_color = vec3(1.0);
  for (int i = 0; i <= MAX_DEPTH; i++) {
    if (i == MAX_DEPTH) {
      return vec3(0.0, 0.0, 0.0);
    }
    vec3 color;
    ray = trace_step(ray, color);
    resulting_color *= color;
    if (length(ray.dir) < 0.001) {
      break;
    }
  }
  return resulting_color;
}


layout (local_size_x = 1, local_size_y = 1) in;

void main(void) {
  ivec2 pix = ivec2(gl_GlobalInvocationID.xy);
  ivec2 size = imageSize(framebuffer);
  if (pix.x >= size.x || pix.y >= size.y) {
    return;
  }
  vec2 pos = vec2(pix) / vec2(size.x, size.y);
  vec3 dir = mix(mix(ray00, ray01, pos.y), mix(ray10, ray11, pos.y), pos.x);
  vec3 resulting_color = vec3(1.0);
  ray3 ray = {eye, dir};
  vec3 color = trace(ray);
  imageStore(framebuffer, pix, vec4(color.xyz, 1.0));
}
