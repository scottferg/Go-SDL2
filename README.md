# Introduction

This a fork of [Go-SDL](http://github.com/banthar/Go-SDL) which supports SDL version 2.0.

Differences from Banthar's original library are:

* SDL functions (except for SDL-mixer) can be safely called from concurrently
  running goroutines
* All SDL events are delivered via a Go channel
* Support for low-level SDL sound functions

* Can be installed in parallel to Banthar's Go-SDL
* The import paths are "github.com/scottferg/Go-SDL2/..."


# Installation

Make sure you have SDL2, SDL2-image, SDL2-mixer and SDL2-ttf (all in -dev version).

Installing libraries and examples:

    go get -v github.com/scottferg/Go-SDL2/sdl


# Credits

Music to test SDL2-mixer is by Kevin MacLeod.
