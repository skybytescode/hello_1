package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/exec"
)

const (
	sampleRate = 44100
	bpm        = 60 // slow, contemplative
	channels   = 2
)

var noteFrequencies = map[string]float64{
	"G2": 98.00,
	"A2": 110.00,
	"C3": 130.81,
	"D3": 146.83,
	"E3": 164.81,
	"F3": 174.61,
	"G3": 196.00,
	"A3": 220.00,
	"B3": 246.94,
	"C4": 261.63,
	"D4": 293.66,
	"E4": 329.63,
	"F4": 349.23,
	"G4": 392.00,
	"A4": 440.00,
	"B4": 493.88,
	"C5": 523.25,
	"D5": 587.33,
	"E5": 659.25,
	"F5": 698.46,
}

func main() {
	player, args, err := selectAudioPlayer()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}
	fmt.Println("Passacaglia (Piano) - inspired by Gibran Alcocer")
	pcm := synthesizePassacaglia()
	playRawPCM(player, args, pcm)
}

func selectAudioPlayer() (string, []string, error) {
	if path, err := exec.LookPath("paplay"); err == nil {
		return path, []string{"--raw", "--format=s16le", "--channels=2", "--rate=44100"}, nil
	}
	if path, err := exec.LookPath("aplay"); err == nil {
		return path, []string{"-q", "-t", "raw", "-f", "S16_LE", "-c", "2", "-r", fmt.Sprint(sampleRate), "-"}, nil
	}
	return "", nil, fmt.Errorf("no supported audio player found; install paplay or aplay")
}

type Note struct {
	Freq          float64
	StartBeats    float64
	DurationBeats float64
}

type Track struct {
	Notes  []Note
	Volume float64
	Pan    float64
}

// synthesizePassacaglia composes an original piano passacaglia in A minor.
//
// Ground bass: Am–F–C–G, repeating every 16 beats (4 bars of 4/4 at 60 BPM).
// Structure: 3 cycles build the melody, coda resolves quietly to A minor.
// Left hand: staggered arpeggios (bass root → 5th → 3rd) sustain for warmth.
func synthesizePassacaglia() []byte {
	beatSec := 60.0 / bpm
	p := noteFrequencies // shorthand

	tracks := []Track{
		// Right hand — original flowing melody
		{
			Volume: 0.65,
			Pan:    0.1,
			Notes: []Note{
				// ── Cycle 1: Awakening (sparse, questioning) ─────────────
				// Am
				{p["A4"], 1, 0.5}, {p["C5"], 1.5, 0.5},
				{p["B4"], 2, 1}, {p["A4"], 3, 1},
				// F
				{p["G4"], 4, 1.5}, {p["A4"], 5.5, 0.5},
				{p["G4"], 6, 1}, {p["F4"], 7, 1},
				// C
				{p["E4"], 8, 0.5}, {p["F4"], 8.5, 0.5},
				{p["G4"], 9, 1}, {p["A4"], 10, 2},
				// G
				{p["B4"], 12, 2},
				{p["A4"], 14, 0.5}, {p["G4"], 14.5, 0.5}, {p["B4"], 15, 1},

				// ── Cycle 2: Development (fuller, rising) ─────────────────
				// Am
				{p["A4"], 16, 0.5}, {p["C5"], 16.5, 0.5},
				{p["E5"], 17, 1},
				{p["D5"], 18, 0.5}, {p["C5"], 18.5, 0.5}, {p["B4"], 19, 1},
				// F
				{p["A4"], 20, 1.5}, {p["C5"], 21.5, 0.5},
				{p["A4"], 22, 1}, {p["G4"], 23, 1},
				// C
				{p["F4"], 24, 0.5}, {p["G4"], 24.5, 0.5},
				{p["A4"], 25, 1}, {p["G4"], 26, 1}, {p["E4"], 27, 1},
				// G
				{p["D4"], 28, 0.5}, {p["F4"], 28.5, 0.5},
				{p["G4"], 29, 1.5},
				{p["B4"], 30.5, 0.5}, {p["A4"], 31, 0.5}, {p["G4"], 31.5, 0.5},

				// ── Cycle 3: Climax (high register, passionate) ───────────
				// Am
				{p["E5"], 32, 1},
				{p["D5"], 33, 0.5}, {p["C5"], 33.5, 0.5},
				{p["A4"], 34, 0.5}, {p["C5"], 34.5, 0.5}, {p["E5"], 35, 1},
				// F
				{p["F5"], 36, 0.5}, {p["E5"], 36.5, 0.5},
				{p["D5"], 37, 0.5}, {p["C5"], 37.5, 0.5},
				{p["A4"], 38, 2},
				// C
				{p["G4"], 40, 0.5}, {p["A4"], 40.5, 0.5},
				{p["C5"], 41, 1.5}, {p["B4"], 42.5, 0.5}, {p["A4"], 43, 1},
				// G
				{p["B4"], 44, 1},
				{p["C5"], 45, 0.5}, {p["B4"], 45.5, 0.5},
				{p["A4"], 46, 1}, {p["G4"], 47, 1},

				// ── Coda: Resolution (quiet, slowing to stillness) ────────
				// Am
				{p["A4"], 48, 2},
				{p["G4"], 50, 0.5}, {p["A4"], 50.5, 0.5}, {p["C5"], 51, 1},
				// F
				{p["A4"], 52, 1.5}, {p["G4"], 53.5, 0.5},
				{p["F4"], 54, 1}, {p["E4"], 55, 1},
				// Final Am — hold until silence
				{p["A4"], 56, 4.5},
			},
		},
		// Left hand — bass roots (the passacaglia ostinato)
		{
			Volume: 0.20,
			Pan:    -0.30,
			Notes: []Note{
				{p["A2"], 0, 3.8}, {p["F3"], 4, 3.8}, {p["C3"], 8, 3.8}, {p["G2"], 12, 3.8},
				{p["A2"], 16, 3.8}, {p["F3"], 20, 3.8}, {p["C3"], 24, 3.8}, {p["G2"], 28, 3.8},
				{p["A2"], 32, 3.8}, {p["F3"], 36, 3.8}, {p["C3"], 40, 3.8}, {p["G2"], 44, 3.8},
				{p["A2"], 48, 3.8}, {p["F3"], 52, 3.8},
				{p["A2"], 56, 5.0},
			},
		},
		// Left hand — inner chord voices (staggered entry for warmth)
		{
			Volume: 0.13,
			Pan:    -0.15,
			Notes: []Note{
				// Am: E3 + A3
				{p["E3"], 0.5, 3.3}, {p["A3"], 1.0, 2.8},
				{p["E3"], 16.5, 3.3}, {p["A3"], 17.0, 2.8},
				{p["E3"], 32.5, 3.3}, {p["A3"], 33.0, 2.8},
				{p["E3"], 48.5, 3.3}, {p["A3"], 49.0, 2.8},
				// F: A3 + C4
				{p["A3"], 4.5, 3.3}, {p["C4"], 5.0, 2.8},
				{p["A3"], 20.5, 3.3}, {p["C4"], 21.0, 2.8},
				{p["A3"], 36.5, 3.3}, {p["C4"], 37.0, 2.8},
				{p["A3"], 52.5, 3.3}, {p["C4"], 53.0, 2.8},
				// C: G3 + E4
				{p["G3"], 8.5, 3.3}, {p["E4"], 9.0, 2.8},
				{p["G3"], 24.5, 3.3}, {p["E4"], 25.0, 2.8},
				{p["G3"], 40.5, 3.3}, {p["E4"], 41.0, 2.8},
				// G: D3 + B3
				{p["D3"], 12.5, 3.3}, {p["B3"], 13.0, 2.8},
				{p["D3"], 28.5, 3.3}, {p["B3"], 29.0, 2.8},
				{p["D3"], 44.5, 3.3}, {p["B3"], 45.0, 2.8},
				// Final Am
				{p["E3"], 56.5, 4.0}, {p["A3"], 57.0, 3.5},
			},
		},
	}

	totalBeats := 61.5
	totalSamples := int(totalBeats * beatSec * sampleRate)
	buffer := make([]byte, totalSamples*channels*2)

	for i := 0; i < totalSamples; i++ {
		timeSec := float64(i) / sampleRate
		left, right := 0.0, 0.0

		for _, track := range tracks {
			for _, note := range track.Notes {
				start := note.StartBeats * beatSec
				dur := note.DurationBeats * beatSec
				if timeSec < start || timeSec >= start+dur {
					continue
				}
				relative := timeSec - start
				phase := note.Freq * relative
				sample := pianoSample(phase)
				amp := pianoEnvelope(relative, dur)
				value := sample * track.Volume * amp
				left += value * (0.5 - track.Pan*0.5)
				right += value * (0.5 + track.Pan*0.5)
			}
		}

		left = clamp(left, -1, 1)
		right = clamp(right, -1, 1)
		pos := i * channels * 2
		binary.LittleEndian.PutUint16(buffer[pos:], int16ToUint16(left))
		binary.LittleEndian.PutUint16(buffer[pos+2:], int16ToUint16(right))
	}

	return buffer
}

// pianoSample blends five harmonic sine partials to approximate piano timbre.
func pianoSample(phase float64) float64 {
	phase = phase - math.Floor(phase)
	return math.Sin(2*math.Pi*phase)*0.55 +
		math.Sin(4*math.Pi*phase)*0.25 +
		math.Sin(6*math.Pi*phase)*0.12 +
		math.Sin(8*math.Pi*phase)*0.05 +
		math.Sin(10*math.Pi*phase)*0.03
}

// pianoEnvelope: sharp hammer attack, decay to sustain, natural release.
func pianoEnvelope(position, duration float64) float64 {
	attack := 0.008
	decay := math.Min(duration*0.20, 0.25)
	release := math.Min(duration*0.35, 0.45)
	sustain := 0.68

	switch {
	case position < attack:
		return position / attack
	case position > duration-release:
		return sustain * (duration - position) / release
	case position < attack+decay:
		t := (position - attack) / decay
		return 1.0 - t*(1.0-sustain)
	default:
		return sustain
	}
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

func int16ToUint16(v float64) uint16 {
	return uint16(int16(v * 32767))
}

func playRawPCM(player string, args []string, pcm []byte) {
	cmd := exec.Command(player, args...)
	cmd.Stdin = bytes.NewReader(pcm)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "audio playback failed using %s: %v\n", player, err)
		os.Exit(1)
	}
}
