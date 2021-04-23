# go-raytrace


**go-raytrace** is a simple ray tracing engine using OpenGL 4.6 written in Go. It was created as a study project for the computer science course at Moscow Institute of Physics and Technology.


![Demo image](/assets/demo_small.png)


## Features

* supported geometry: spheres and axes-aligned boxes
* lambertian, reflective and transparent materials
* dynamic camera with depth of field effect
* loading scene data from JSON and random scene generation


## Acknowlegments

* [Ray Tracing in One Weekend](https://raytracing.github.io/books/RayTracingInOneWeekend.html) by Peter Shirley: a great book on the basics of raytracing that drove the project from the very start
* [Awesome tutorial](https://github.com/LWJGL/lwjgl3-wiki/wiki/2.6.1.-Ray-tracing-with-OpenGL-Compute-Shaders-%28Part-I%29) by Kai Burjack which helped to rewrite the original C++ engine for OpenGL (and also served as a partial inspiration for the scene displayed above)


## Requirements and installation

* Go 1.6+
* OpenGL 4.6 Core
* additional dependencies for the [go-gl/glfw](https://github.com/go-gl/glfw) package

After cloning the repository and installing all dependencies just navigate to the root folder of the repo and run `go build`. Help on how to use the app can be found in `controls.txt` file and via `./go-raytrace -help`.

*The application has been tested only on Linux Mint, so any feedback on compatability is appreciated.*
