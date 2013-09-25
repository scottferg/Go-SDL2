package mixer

// #cgo pkg-config: SDL2_mixer
// #include <SDL2/SDL_mixer.h>
import "C"
import "unsafe"

// A Chunk file.
type Chunk struct {
	cchunk *C.Mix_Chunk
}

// Loads a sound file to use.
func LoadWAV(file string) *Chunk {
	cfile := C.CString(file)
	rb := C.CString("rb")

	cchunk := C.Mix_LoadWAV_RW(C.SDL_RWFromFile(cfile, rb), 1)
	C.free(unsafe.Pointer(cfile))
	C.free(unsafe.Pointer(rb))

	if cchunk == nil {
		return nil
	}
	return &Chunk{cchunk}
}

// Frees the loaded sound file.
func (c *Chunk) Free() {
	C.Mix_FreeChunk(c.cchunk)
}

func (c *Chunk) Volume(volume int) int {
	return int(C.Mix_VolumeChunk(c.cchunk, C.int(volume)))
}

func (c *Chunk) PlayChannel(channel, loops int) int {
	return c.PlayChannelTimed(channel, loops, -1)
}

func (c *Chunk) PlayChannelTimed(channel, loops, ticks int) int {
	return int(C.Mix_PlayChannelTimed(C.int(channel), c.cchunk, C.int(loops), C.int(ticks)))
}

func (c *Chunk) FadeInChannel(channel, loops, ms int) int {
	return c.FadeInChannelTimed(channel, loops, ms, -1)
}

func (c *Chunk) FadeInChannelTimed(channel, loops, ms, ticks int) int {
	return int(C.Mix_FadeInChannelTimed(C.int(channel), c.cchunk, C.int(loops), C.int(ms), C.int(ticks)))
}

func GetChunk(channel int) *Chunk {
	out := C.Mix_GetChunk(C.int(channel))
	if out == nil {
		return nil
	}
	return &Chunk{out}
}
