/*
A binding of SDL and SDL_image.

The binding works in pretty much the same way as it does in C, although
some of the functions have been altered to give them an object-oriented
flavor (eg. Rather than sdl.Flip(surface) it's surface.Flip() )
*/
package sdl

// #cgo pkg-config: sdl2 SDL2_image
//
// struct private_hwdata{};
// struct SDL_BlitMap{};
// #define map _map
//
// #include <SDL2/SDL.h>
// #include <SDL2/SDL_image.h>
import "C"

import (
	"fmt"
	"os"
	"reflect"
	"runtime"
	"sync"
	"time"
	"unsafe"
)

type cast unsafe.Pointer

// Mutex for serialization of access to certain SDL functions.
//
// There is no need to use this in application code, the mutex is a public variable
// just because it needs to be accessible from other parts of Go-SDL (such as package "sdl/ttf").
//
// Surface-level functions (such as 'Surface.Blit') are not using this mutex,
// so it is possible to modify multiple surfaces concurrently.
// There is no dependency between 'Surface.Lock' and the global mutex.
var GlobalMutex sync.Mutex

type Surface struct {
	cSurface *C.SDL_Surface
	mutex    sync.RWMutex

	Flags  uint32
	Format *PixelFormat
	W      int32
	H      int32
	Pitch  uint16
	Pixels unsafe.Pointer

	gcPixels interface{} // Prevents garbage collection of pixels passed to func CreateRGBSurfaceFrom
}

type Window struct {
	cWindow *C.SDL_Window
	mutex   sync.RWMutex

	Flags uint32
	X     int32
	Y     int32
	W     int32
	H     int32
}

type Renderer struct {
	cRenderer *C.SDL_Renderer
}

type Texture struct {
	cTexture *C.SDL_Texture
}

func wrapSurface(cSurface *C.SDL_Surface) *Surface {
	var s *Surface

	if cSurface != nil {
		var surface Surface
		surface.SetCSurface(unsafe.Pointer(cSurface))
		s = &surface
	} else {
		s = nil
	}

	return s
}

func wrapWindow(cWindow *C.SDL_Window) *Window {
	var w *Window

	if cWindow != nil {
		var window Window
		window.SetCWindow(unsafe.Pointer(cWindow))
		w = &window
	} else {
		w = nil
	}

	return w
}

func wrapRenderer(cRenderer *C.SDL_Renderer) *Renderer {
	var r *Renderer

	if cRenderer != nil {
		var renderer Renderer
		renderer.cRenderer = (*C.SDL_Renderer)(unsafe.Pointer(cRenderer))
		r = &renderer
	} else {
		r = nil
	}

	return r
}

func wrapTexture(cTexture *C.SDL_Texture) *Texture {
	var t *Texture

	if cTexture != nil {
		var texture Texture
		texture.cTexture = (*C.SDL_Texture)(unsafe.Pointer(cTexture))
		t = &texture
	} else {
		t = nil
	}

	return t
}

// FIXME: Ideally, this should NOT be a public function, but it is needed in the package "ttf" ...
func (s *Surface) SetCSurface(cSurface unsafe.Pointer) {
	s.cSurface = (*C.SDL_Surface)(cSurface)
	s.reload()
}

func (s *Window) SetCWindow(cWindow unsafe.Pointer) {
	s.cWindow = (*C.SDL_Window)(cWindow)
	s.reload()
}

// Pull data from C.SDL_Window.
// Make sure to use this when the C surface might have been changed.
//
// TODO: Incomplete
func (s *Window) reload() {
	/*
		s.Flags = uint32(s.cWindow.flags)
		s.W = int32(s.cWindow.w)
		s.H = int32(s.cWindow.h)
		s.X = int32(s.cWindow.x)
		s.Y = int32(s.cWindow.y)
	*/
}

// Pull data from C.SDL_Surface.
// Make sure to use this when the C surface might have been changed.
func (s *Surface) reload() {
	s.Flags = uint32(s.cSurface.flags)
	s.Format = (*PixelFormat)(cast(s.cSurface.format))
	s.W = int32(s.cSurface.w)
	s.H = int32(s.cSurface.h)
	s.Pitch = uint16(s.cSurface.pitch)
	s.Pixels = s.cSurface.pixels
}

func (s *Surface) destroy() {
	s.cSurface = nil
	s.Format = nil
	s.Pixels = nil
	s.gcPixels = nil
}

// =======
// Renderer
// =======

func CreateRenderer(w *Window, index int, flags uint32) *Renderer {
	GlobalMutex.Lock()
	defer GlobalMutex.Unlock()

	renderer := C.SDL_CreateRenderer(w.cWindow, C.int(index), C.Uint32(flags))

	return wrapRenderer(renderer)
}

func (r *Renderer) Clear() {
	GlobalMutex.Lock()
	defer GlobalMutex.Unlock()

	C.SDL_RenderClear(r.cRenderer)
}

func (r *Renderer) Copy(t *Texture, src, dst *Rect) {
	GlobalMutex.Lock()
	defer GlobalMutex.Unlock()

	C.SDL_RenderCopy(r.cRenderer, t.cTexture,
		(*C.SDL_Rect)(cast(src)), (*C.SDL_Rect)(cast(dst)))
}

func (r *Renderer) Present() {
	GlobalMutex.Lock()
	defer GlobalMutex.Unlock()

	C.SDL_RenderPresent(r.cRenderer)
}

func (r *Renderer) SetDrawColor(c Color) {
	GlobalMutex.Lock()
	defer GlobalMutex.Unlock()

	C.SDL_SetRenderDrawColor(r.cRenderer, C.Uint8(c.R),
		C.Uint8(c.G), C.Uint8(c.B), C.Uint8(c.Alpha))
}

func (r *Renderer) FillRect(rect *Rect) {
	GlobalMutex.Lock()
	defer GlobalMutex.Unlock()

	fmt.Printf("null? %t\n", r.cRenderer == nil)
	C.SDL_RenderFillRect(r.cRenderer, (*C.SDL_Rect)(cast(rect)))
}

func (r *Renderer) Destroy() {
	GlobalMutex.Lock()
	defer GlobalMutex.Unlock()

	C.SDL_DestroyRenderer(r.cRenderer)
}

// =======
// Texture
// =======

func CreateTextureFromSurface(r *Renderer, s *Surface) *Texture {
	GlobalMutex.Lock()
	defer GlobalMutex.Unlock()

	texture := C.SDL_CreateTextureFromSurface(r.cRenderer, s.cSurface)
	return wrapTexture(texture)
}

func (t *Texture) Destroy() {
	GlobalMutex.Lock()
	defer GlobalMutex.Unlock()

	C.SDL_DestroyTexture(t.cTexture)
}

// =======
// General
// =======

// The version of Go-SDL bindings.
// The version descriptor changes into a new unique string
// after a semantically incompatible Go-SDL update.
//
// The returned value can be checked by users of this package
// to make sure they are using a version with the expected semantics.
//
// If Go adds some kind of support for package versioning, this function will go away.
func GoSdlVersion() string {
	return "⚛SDL bindings 1.0"
}

// Initializes SDL.
func Init(flags uint32) int {
	GlobalMutex.Lock()
	status := int(C.SDL_Init(C.Uint32(flags)))
	if (status != 0) && (runtime.GOOS == "darwin") && (flags&INIT_VIDEO != 0) {
		if os.Getenv("SDL_VIDEODRIVER") == "" {
			os.Setenv("SDL_VIDEODRIVER", "x11")
			status = int(C.SDL_Init(C.Uint32(flags)))
			if status != 0 {
				os.Setenv("SDL_VIDEODRIVER", "")
			}
		}
	}

	GlobalMutex.Unlock()
	return status
}

// Shuts down SDL
func Quit() {
	GlobalMutex.Lock()

	if currentVideoSurface != nil {
		currentVideoSurface.destroy()
		currentVideoSurface = nil
	}

	C.SDL_Quit()

	GlobalMutex.Unlock()
}

// Initializes subsystems.
func InitSubSystem(flags uint32) int {
	GlobalMutex.Lock()
	status := int(C.SDL_InitSubSystem(C.Uint32(flags)))
	if (status != 0) && (runtime.GOOS == "darwin") && (flags&INIT_VIDEO != 0) {
		if os.Getenv("SDL_VIDEODRIVER") == "" {
			os.Setenv("SDL_VIDEODRIVER", "x11")
			status = int(C.SDL_InitSubSystem(C.Uint32(flags)))
			if status != 0 {
				os.Setenv("SDL_VIDEODRIVER", "")
			}
		}
	}
	GlobalMutex.Unlock()
	return status
}

// Shuts down a subsystem.
func QuitSubSystem(flags uint32) {
	GlobalMutex.Lock()
	C.SDL_QuitSubSystem(C.Uint32(flags))
	GlobalMutex.Unlock()
}

// Checks which subsystems are initialized.
func WasInit(flags uint32) int {
	GlobalMutex.Lock()
	status := int(C.SDL_WasInit(C.Uint32(flags)))
	GlobalMutex.Unlock()
	return status
}

func NumDisplayModes(index int) int {
	GlobalMutex.Lock()
	defer GlobalMutex.Unlock()

	return int(C.SDL_GetNumDisplayModes(C.int(index)))
}

// ==============
// Error Handling
// ==============

// Gets SDL error string
func GetError() string {
	GlobalMutex.Lock()
	s := C.GoString(C.SDL_GetError())
	GlobalMutex.Unlock()
	return s
}

// Clear the current SDL error
func ClearError() {
	GlobalMutex.Lock()
	C.SDL_ClearError()
	GlobalMutex.Unlock()
}

// ======
// Window
// ======

func CreateWindow(title string, x, y int, h, w int, flags uint32) *Window {
	GlobalMutex.Lock()
	defer GlobalMutex.Unlock()

	window := C.SDL_CreateWindow(C.CString(title), C.int(x), C.int(y),
		C.int(h), C.int(w), C.Uint32(flags))

	return wrapWindow(window)
}

// TODO: Not working currently
func CreateWindowAndRenderer(h, w int, flags uint32) (*Window, *Renderer) {
	var win *Window
	var rend *Renderer

	GlobalMutex.Lock()
	defer GlobalMutex.Unlock()

	C.SDL_CreateWindowAndRenderer(C.int(h), C.int(w), C.Uint32(flags),
		(**C.SDL_Window)(cast(&win.cWindow)), (**C.SDL_Renderer)(cast(&rend.cRenderer)))

	return win, rend
}

func (w *Window) GetTitle() string {
	GlobalMutex.Lock()
	defer GlobalMutex.Unlock()

	ctitle := C.SDL_GetWindowTitle(w.cWindow)

	return C.GoString(ctitle)
}

func (w *Window) SetTitle(title string) {
	GlobalMutex.Lock()
	defer GlobalMutex.Unlock()

	ctitle := C.CString(title)
	C.SDL_SetWindowTitle(w.cWindow, ctitle)

	C.free(unsafe.Pointer(ctitle))
}

func (w *Window) SetIcon(s *Surface) {
	GlobalMutex.Lock()
	defer GlobalMutex.Unlock()

	C.SDL_SetWindowIcon(w.cWindow, s.cSurface)
}

func (w *Window) Destroy() {
	GlobalMutex.Lock()
	defer GlobalMutex.Unlock()

	C.SDL_DestroyWindow(w.cWindow)
}

// ======
// Video
// ======

var currentVideoSurface *Surface = nil

// Returns a pointer to the current display surface.
func GetVideoSurface() *Surface {
	GlobalMutex.Lock()
	surface := currentVideoSurface
	GlobalMutex.Unlock()
	return surface
}

// Swaps OpenGL framebuffers/Update Display.
func (w *Window) GL_SwapWindow() {
	GlobalMutex.Lock()
	C.SDL_GL_SwapWindow(w.cWindow)
	GlobalMutex.Unlock()
}

func GL_SetAttribute(attr int, value int) int {
	GlobalMutex.Lock()
	status := int(C.SDL_GL_SetAttribute(C.SDL_GLattr(attr), C.int(value)))
	GlobalMutex.Unlock()
	return status
}

// Frees (deletes) a Surface
func (screen *Surface) Free() {
	GlobalMutex.Lock()
	screen.mutex.Lock()

	C.SDL_FreeSurface(screen.cSurface)

	screen.destroy()
	if screen == currentVideoSurface {
		currentVideoSurface = nil
	}

	screen.mutex.Unlock()
	GlobalMutex.Unlock()
}

// Locks a surface for direct access.
func (screen *Surface) Lock() int {
	screen.mutex.Lock()
	status := int(C.SDL_LockSurface(screen.cSurface))
	screen.mutex.Unlock()
	return status
}

// Unlocks a previously locked surface.
func (screen *Surface) Unlock() {
	screen.mutex.Lock()
	C.SDL_UnlockSurface(screen.cSurface)
	screen.mutex.Unlock()
}

// Performs a fast blit from the source surface to the destination surface.
// This is the same as func BlitSurface, but the order of arguments is reversed.
func (dst *Surface) Blit(dstrect *Rect, src *Surface, srcrect *Rect) int {
	GlobalMutex.Lock()
	global := true
	if (src != currentVideoSurface) && (dst != currentVideoSurface) {
		GlobalMutex.Unlock()
		global = false
	}

	// At this point: GlobalMutex is locked only if at least one of 'src' or 'dst'
	//                was identical to 'currentVideoSurface'

	var ret C.int
	{
		src.mutex.RLock()
		dst.mutex.Lock()

		ret = C.SDL_UpperBlit(
			src.cSurface,
			(*C.SDL_Rect)(cast(srcrect)),
			dst.cSurface,
			(*C.SDL_Rect)(cast(dstrect)))

		dst.mutex.Unlock()
		src.mutex.RUnlock()
	}

	if global {
		GlobalMutex.Unlock()
	}

	return int(ret)
}

// Performs a fast blit from the source surface to the destination surface.
func BlitSurface(src *Surface, srcrect *Rect, dst *Surface, dstrect *Rect) int {
	return dst.Blit(dstrect, src, srcrect)
}

// This function performs a fast fill of the given rectangle with some color.
func (dst *Surface) FillRect(dstrect *Rect, color uint32) int {
	dst.mutex.Lock()

	var ret = C.SDL_FillRect(
		dst.cSurface,
		(*C.SDL_Rect)(cast(dstrect)),
		C.Uint32(color))

	dst.mutex.Unlock()

	return int(ret)
}

// Sets the color key (transparent pixel)  in  a  blittable  surface  and
// enables or disables RLE blit acceleration.
func (s *Surface) SetColorKey(flags uint32, ColorKey uint32) int {
	s.mutex.Lock()
	status := int(C.SDL_SetColorKey(s.cSurface, C.int(flags), C.Uint32(ColorKey)))
	s.mutex.Unlock()
	return status
}

// Gets the clipping rectangle for a surface.
func (s *Surface) GetClipRect(r *Rect) {
	s.mutex.RLock()
	C.SDL_GetClipRect(s.cSurface, (*C.SDL_Rect)(cast(r)))
	s.mutex.RUnlock()
}

// Sets the clipping rectangle for a surface.
func (s *Surface) SetClipRect(r *Rect) {
	s.mutex.Lock()
	C.SDL_SetClipRect(s.cSurface, (*C.SDL_Rect)(cast(r)))
	s.mutex.Unlock()
}

// Map a RGBA color value to a pixel format.
func MapRGBA(format *PixelFormat, r, g, b, a uint8) uint32 {
	return (uint32)(C.SDL_MapRGBA((*C.SDL_PixelFormat)(cast(format)), (C.Uint8)(r), (C.Uint8)(g), (C.Uint8)(b), (C.Uint8)(a)))
}

// Gets RGBA values from a pixel in the specified pixel format.
func GetRGBA(color uint32, format *PixelFormat, r, g, b, a *uint8) {
	C.SDL_GetRGBA(C.Uint32(color), (*C.SDL_PixelFormat)(cast(format)), (*C.Uint8)(r), (*C.Uint8)(g), (*C.Uint8)(b), (*C.Uint8)(a))
}

// Loads Surface from file (using IMG_Load).
func Load(file string) *Surface {
	GlobalMutex.Lock()

	cfile := C.CString(file)
	var screen = C.IMG_Load(cfile)
	C.free(unsafe.Pointer(cfile))

	GlobalMutex.Unlock()

	return wrapSurface(screen)
}

// Creates an empty Surface.
func CreateRGBSurface(flags uint32, width int, height int, bpp int, Rmask uint32, Gmask uint32, Bmask uint32, Amask uint32) *Surface {
	GlobalMutex.Lock()

	p := C.SDL_CreateRGBSurface(C.Uint32(flags), C.int(width), C.int(height), C.int(bpp),
		C.Uint32(Rmask), C.Uint32(Gmask), C.Uint32(Bmask), C.Uint32(Amask))

	GlobalMutex.Unlock()

	return wrapSurface(p)
}

// Creates a Surface from existing pixel data. It expects pixels to be a slice, pointer or unsafe.Pointer.
func CreateRGBSurfaceFrom(pixels interface{}, width, height, bpp, pitch int, Rmask, Gmask, Bmask, Amask uint32) *Surface {
	var ptr unsafe.Pointer
	switch v := reflect.ValueOf(pixels); v.Kind() {
	case reflect.Ptr, reflect.UnsafePointer, reflect.Slice:
		ptr = unsafe.Pointer(v.Pointer())
	default:
		panic("Don't know how to handle type: " + v.Kind().String())
	}

	GlobalMutex.Lock()
	p := C.SDL_CreateRGBSurfaceFrom(ptr, C.int(width), C.int(height), C.int(bpp), C.int(pitch),
		C.Uint32(Rmask), C.Uint32(Gmask), C.Uint32(Bmask), C.Uint32(Amask))
	GlobalMutex.Unlock()

	s := wrapSurface(p)
	s.gcPixels = pixels
	return s
}

// Key
type Key C.int

// Gets the name of an SDL virtual keysym
func GetKeyName(key Key) string {
	GlobalMutex.Lock()
	name := C.GoString(C.SDL_GetKeyName(C.SDL_Keycode(key)))
	GlobalMutex.Unlock()
	return name
}

// ======
// Events
// ======

// Polls for currently pending events
func (event *Event) poll() bool {
	GlobalMutex.Lock()

	var ret = C.SDL_PollEvent((*C.SDL_Event)(cast(event)))

	GlobalMutex.Unlock()

	return ret != 0
}

// =====
// Mouse
// =====

// Retrieves the current state of the mouse.
func GetMouseState(x, y *int) uint8 {
	GlobalMutex.Lock()
	state := uint8(C.SDL_GetMouseState((*C.int)(cast(x)), (*C.int)(cast(y))))
	GlobalMutex.Unlock()
	return state
}

// Retrieves the current state of the mouse relative to the last time this
// function was called.
func GetRelativeMouseState(x, y *int) uint8 {
	GlobalMutex.Lock()
	state := uint8(C.SDL_GetRelativeMouseState((*C.int)(cast(x)), (*C.int)(cast(y))))
	GlobalMutex.Unlock()
	return state
}

// Toggle whether or not the cursor is shown on the screen.
func ShowCursor(toggle int) int {
	GlobalMutex.Lock()
	state := int(C.SDL_ShowCursor((C.int)(toggle)))
	GlobalMutex.Unlock()
	return state
}

// ========
// Joystick
// ========

type Joystick struct {
	cJoystick *C.SDL_Joystick
}

func wrapJoystick(cJoystick *C.SDL_Joystick) *Joystick {
	var j *Joystick
	if cJoystick != nil {
		var joystick Joystick
		joystick.cJoystick = (*C.SDL_Joystick)(unsafe.Pointer(cJoystick))
		j = &joystick
	} else {
		j = nil
	}
	return j
}

// Count the number of joysticks attached to the system
func NumJoysticks() int {
	GlobalMutex.Lock()
	num := int(C.SDL_NumJoysticks())
	GlobalMutex.Unlock()
	return num
}

// Open a joystick for use The index passed as an argument refers to
// the N'th joystick on the system. This index is the value which will
// identify this joystick in future joystick events.  This function
// returns a joystick identifier, or NULL if an error occurred.
func JoystickOpen(deviceIndex int) *Joystick {
	GlobalMutex.Lock()
	joystick := C.SDL_JoystickOpen(C.int(deviceIndex))
	GlobalMutex.Unlock()
	return wrapJoystick(joystick)
}

// Update the current state of the open joysticks. This is called
// automatically by the event loop if any joystick events are enabled.
func JoystickUpdate() {
	GlobalMutex.Lock()
	C.SDL_JoystickUpdate()
	GlobalMutex.Unlock()
}

// Enable/disable joystick event polling. If joystick events are
// disabled, you must call SDL_JoystickUpdate() yourself and check the
// state of the joystick when you want joystick information. The state
// can be one of SDL_QUERY, SDL_ENABLE or SDL_IGNORE.
func JoystickEventState(state int) int {
	GlobalMutex.Lock()
	result := int(C.SDL_JoystickEventState(C.int(state)))
	GlobalMutex.Unlock()
	return result
}

// Close a joystick previously opened with SDL_JoystickOpen()
func (joystick *Joystick) Close() {
	GlobalMutex.Lock()
	C.SDL_JoystickClose(joystick.cJoystick)
	GlobalMutex.Unlock()
}

// Get the number of general axis controls on a joystick
func (joystick *Joystick) NumAxes() int {
	return int(C.SDL_JoystickNumAxes(joystick.cJoystick))
}

// Get the number of buttons on a joystick
func (joystick *Joystick) NumButtons() int {
	return int(C.SDL_JoystickNumButtons(joystick.cJoystick))
}

// Get the number of trackballs on a Joystick trackballs have only
// relative motion events associated with them and their state cannot
// be polled.
func (joystick *Joystick) NumBalls() int {
	return int(C.SDL_JoystickNumBalls(joystick.cJoystick))
}

// Get the number of POV hats on a joystick
func (joystick *Joystick) NumHats() int {
	return int(C.SDL_JoystickNumHats(joystick.cJoystick))
}

// Get the current state of a POV hat on a joystick
// The hat indices start at index 0.
func (joystick *Joystick) GetHat(hat int) uint8 {
	return uint8(C.SDL_JoystickGetHat(joystick.cJoystick, C.int(hat)))
}

// Get the current state of a button on a joystick. The button indices
// start at index 0.
func (joystick *Joystick) GetButton(button int) uint8 {
	return uint8(C.SDL_JoystickGetButton(joystick.cJoystick, C.int(button)))
}

// Get the ball axis change since the last poll. The ball indices
// start at index 0. This returns 0, or -1 if you passed it invalid
// parameters.
func (joystick *Joystick) GetBall(ball int, dx, dy *int) int {
	return int(C.SDL_JoystickGetBall(joystick.cJoystick, C.int(ball), (*C.int)(cast(dx)), (*C.int)(cast(dy))))
}

// Get the current state of an axis control on a joystick. The axis
// indices start at index 0. The state is a value ranging from -32768
// to 32767.
func (joystick *Joystick) GetAxis(axis int) int16 {
	return int16(C.SDL_JoystickGetAxis(joystick.cJoystick, C.int(axis)))
}

// ====
// Time
// ====

// Gets the number of milliseconds since the SDL library initialization.
func GetTicks() uint32 {
	GlobalMutex.Lock()
	t := uint32(C.SDL_GetTicks())
	GlobalMutex.Unlock()
	return t
}

// Waits a specified number of milliseconds before returning.
func Delay(ms uint32) {
	time.Sleep(time.Duration(ms) * time.Millisecond)
}
