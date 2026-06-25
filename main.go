package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strings"
)

const (
	sampleRate = 44100
	channels   = 2
)

var noteFrequencies = map[string]float64{
	"G2":  98.00,
	"A2":  110.00,
	"C3":  130.81,
	"D3":  146.83,
	"E3":  164.81,
	"F3":  174.61,
	"G3":  196.00,
	"Ab3": 207.65,
	"A3":  220.00,
	"B3":  246.94,
	"C4":  261.63,
	"D4":  293.66,
	"E4":  329.63,
	"F4":  349.23,
	"G4":  392.00,
	"A4":  440.00,
	"B4":  493.88,
	"C5":  523.25,
	"D5":  587.33,
	"E5":  659.25,
	"F5":  698.46,
}

func main() {
	player, args, err := selectAudioPlayer()
	if err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		os.Exit(1)
	}

	fmt.Println("1. Passacaglia      - inspired by Gibran Alcocer")
	fmt.Println("2. Waltz in A minor - inspired by Hijo de la Luna (Piano)")
	fmt.Print("\nChoose (1 or 2): ")

	var choice string
	fmt.Scanln(&choice)
	choice = strings.TrimSpace(choice)

	var pcm []byte
	switch choice {
	case "2":
		fmt.Println("Playing: Waltz in A minor - inspired by Hijo de la Luna")
		pcm = synthesizeWaltz()
	default:
		fmt.Println("Playing: Passacaglia - inspired by Gibran Alcocer")
		pcm = synthesizePassacaglia()
	}

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

// render mixes tracks into stereo 16-bit PCM. beatSec converts beat positions to seconds.
func render(tracks []Track, totalBeats, beatSec float64) []byte {
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
				rel := timeSec - start
				sample := pianoSample(note.Freq * rel)
				amp := pianoEnvelope(rel, dur)
				v := sample * track.Volume * amp
				left += v * (0.5 - track.Pan*0.5)
				right += v * (0.5 + track.Pan*0.5)
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

// synthesizeWaltz composes an original piano waltz in A minor (3/4, 80 BPM)
// inspired by the Spanish romantic style. The E major chord uses Ab3 (G#3)
// for the characteristic Phrygian colour.
func synthesizeWaltz() []byte {
	const bpm = 80.0
	beatSec := 60.0 / bpm
	p := noteFrequencies

	// Build left-hand waltz pattern from a chord list.
	// Each bar: root on beat 1 (sustained ~bar), chord tones on beats 2 and 3.
	type chord struct {
		bar  float64
		root string
		mid  string
		top  string
	}
	chords := []chord{
		// Verse A: Am – F – Am – E
		{0, "A2", "E3", "A3"}, {3, "A2", "E3", "A3"},
		{6, "F3", "C4", "A3"}, {9, "F3", "C4", "A3"},
		{12, "A2", "E3", "A3"}, {15, "A2", "E3", "A3"},
		{18, "E3", "Ab3", "B3"}, {21, "E3", "Ab3", "B3"},
		// Verse B: Am – F – C – E
		{24, "A2", "E3", "A3"}, {27, "A2", "E3", "A3"},
		{30, "F3", "C4", "A3"}, {33, "F3", "C4", "A3"},
		{36, "C3", "G3", "E4"}, {39, "C3", "G3", "E4"},
		{42, "E3", "Ab3", "B3"}, {45, "E3", "Ab3", "B3"},
		// Chorus: Am – F – Am – E
		{48, "A2", "E3", "A3"}, {51, "A2", "E3", "A3"},
		{54, "F3", "C4", "A3"}, {57, "F3", "C4", "A3"},
		{60, "A2", "E3", "A3"}, {63, "A2", "E3", "A3"},
		{66, "E3", "Ab3", "B3"}, {69, "E3", "Ab3", "B3"},
		// Coda: Am – F – E – Am
		{72, "A2", "E3", "A3"},
		{75, "F3", "C4", "A3"},
		{78, "E3", "Ab3", "B3"},
		{81, "A2", "E3", "A3"},
	}

	var lhNotes []Note
	for _, c := range chords {
		lhNotes = append(lhNotes,
			Note{p[c.root], c.bar + 0, 2.8},
			Note{p[c.mid], c.bar + 1, 1.8},
			Note{p[c.top], c.bar + 2, 0.9},
		)
	}
	// Let the final chord ring out
	lhNotes[len(lhNotes)-3].DurationBeats = 6.0
	lhNotes[len(lhNotes)-2].DurationBeats = 5.0
	lhNotes[len(lhNotes)-1].DurationBeats = 4.0

	tracks := []Track{
		// Right hand — lyrical melody (3/4 waltz)
		{
			Volume: 0.65,
			Pan:    0.1,
			Notes: []Note{
				// ── Verse A ─────────────────────────────────────────────
				// Bar 1-2 (Am): gentle pickup, rise and fall
				{p["E4"], 0.5, 0.5}, {p["F4"], 1, 0.5}, {p["G4"], 1.5, 0.5}, {p["A4"], 2, 1},
				{p["G4"], 3, 1.5}, {p["F4"], 4.5, 0.5}, {p["E4"], 5, 1},
				// Bar 3-4 (F): step up, hold
				{p["F4"], 6, 1}, {p["G4"], 7, 1}, {p["A4"], 8, 1},
				{p["G4"], 9, 2}, {p["F4"], 11, 1},
				// Bar 5-6 (Am): lyric phrase with breath
				{p["E4"], 12, 1.5}, {p["F4"], 13.5, 0.5}, {p["E4"], 14, 1},
				{p["D4"], 15, 1.5}, {p["C4"], 16.5, 0.5}, {p["D4"], 17, 1},
				// Bar 7-8 (E): tension — rising to A4
				{p["E4"], 18, 1}, {p["F4"], 19, 1}, {p["G4"], 20, 1},
				{p["A4"], 21, 3},

				// ── Verse B (development) ───────────────────────────────
				// Bar 9-10 (Am): start higher, descend
				{p["C5"], 24, 1}, {p["B4"], 25, 1}, {p["A4"], 26, 1},
				{p["G4"], 27, 2}, {p["A4"], 29, 1},
				// Bar 11-12 (F): rising phrase
				{p["F4"], 30, 1.5}, {p["G4"], 31.5, 0.5}, {p["A4"], 32, 1},
				{p["B4"], 33, 2}, {p["A4"], 35, 1},
				// Bar 13-14 (C): climb toward climax
				{p["G4"], 36, 1}, {p["A4"], 37, 1}, {p["B4"], 38, 1},
				{p["C5"], 39, 2}, {p["B4"], 41, 1},
				// Bar 15-16 (E): build tension
				{p["A4"], 42, 1}, {p["B4"], 43, 1}, {p["C5"], 44, 1},
				{p["B4"], 45, 3},

				// ── Chorus (high and passionate) ────────────────────────
				// Bar 17-18 (Am): peak register
				{p["E5"], 48, 1}, {p["D5"], 49, 1}, {p["C5"], 50, 1},
				{p["B4"], 51, 2}, {p["A4"], 53, 1},
				// Bar 19-20 (F): descend with expression
				{p["C5"], 54, 1.5}, {p["B4"], 55.5, 0.5}, {p["A4"], 56, 1},
				{p["G4"], 57, 3},
				// Bar 21-22 (Am): back to earth
				{p["A4"], 60, 1}, {p["G4"], 61, 1}, {p["F4"], 62, 1},
				{p["E4"], 63, 3},
				// Bar 23-24 (E): longing, unresolved
				{p["F4"], 66, 1}, {p["E4"], 67, 1}, {p["D4"], 68, 1},
				{p["E4"], 69, 3},

				// ── Coda (fading to silence) ─────────────────────────────
				// Bar 25 (Am)
				{p["A4"], 72, 2}, {p["G4"], 74, 1},
				// Bar 26 (F)
				{p["F4"], 75, 1.5}, {p["E4"], 76.5, 0.5}, {p["D4"], 77, 1},
				// Bar 27 (E)
				{p["E4"], 78, 3},
				// Bar 28 (Am) — final
				{p["A4"], 81, 2}, {p["A3"], 83, 4},
			},
		},
		{Volume: 0.20, Pan: -0.3, Notes: lhNotes},
	}

	return render(tracks, 88, beatSec)
}

// synthesizePassacaglia composes a piano passacaglia in A minor (4/4, 60 BPM).
// Ground: Am–F–C–G repeating every 16 beats across 3 cycles and a coda.
func synthesizePassacaglia() []byte {
	const bpm = 60.0
	beatSec := 60.0 / bpm
	p := noteFrequencies

	tracks := []Track{
		{
			Volume: 0.65,
			Pan:    0.1,
			Notes: []Note{
				// ── Cycle 1: Awakening ──────────────────────────────────
				{p["A4"], 1, 0.5}, {p["C5"], 1.5, 0.5},
				{p["B4"], 2, 1}, {p["A4"], 3, 1},
				{p["G4"], 4, 1.5}, {p["A4"], 5.5, 0.5},
				{p["G4"], 6, 1}, {p["F4"], 7, 1},
				{p["E4"], 8, 0.5}, {p["F4"], 8.5, 0.5},
				{p["G4"], 9, 1}, {p["A4"], 10, 2},
				{p["B4"], 12, 2},
				{p["A4"], 14, 0.5}, {p["G4"], 14.5, 0.5}, {p["B4"], 15, 1},
				// ── Cycle 2: Development ────────────────────────────────
				{p["A4"], 16, 0.5}, {p["C5"], 16.5, 0.5},
				{p["E5"], 17, 1},
				{p["D5"], 18, 0.5}, {p["C5"], 18.5, 0.5}, {p["B4"], 19, 1},
				{p["A4"], 20, 1.5}, {p["C5"], 21.5, 0.5},
				{p["A4"], 22, 1}, {p["G4"], 23, 1},
				{p["F4"], 24, 0.5}, {p["G4"], 24.5, 0.5},
				{p["A4"], 25, 1}, {p["G4"], 26, 1}, {p["E4"], 27, 1},
				{p["D4"], 28, 0.5}, {p["F4"], 28.5, 0.5},
				{p["G4"], 29, 1.5},
				{p["B4"], 30.5, 0.5}, {p["A4"], 31, 0.5}, {p["G4"], 31.5, 0.5},
				// ── Cycle 3: Climax ─────────────────────────────────────
				{p["E5"], 32, 1},
				{p["D5"], 33, 0.5}, {p["C5"], 33.5, 0.5},
				{p["A4"], 34, 0.5}, {p["C5"], 34.5, 0.5}, {p["E5"], 35, 1},
				{p["F5"], 36, 0.5}, {p["E5"], 36.5, 0.5},
				{p["D5"], 37, 0.5}, {p["C5"], 37.5, 0.5},
				{p["A4"], 38, 2},
				{p["G4"], 40, 0.5}, {p["A4"], 40.5, 0.5},
				{p["C5"], 41, 1.5}, {p["B4"], 42.5, 0.5}, {p["A4"], 43, 1},
				{p["B4"], 44, 1},
				{p["C5"], 45, 0.5}, {p["B4"], 45.5, 0.5},
				{p["A4"], 46, 1}, {p["G4"], 47, 1},
				// ── Coda ────────────────────────────────────────────────
				{p["A4"], 48, 2},
				{p["G4"], 50, 0.5}, {p["A4"], 50.5, 0.5}, {p["C5"], 51, 1},
				{p["A4"], 52, 1.5}, {p["G4"], 53.5, 0.5},
				{p["F4"], 54, 1}, {p["E4"], 55, 1},
				{p["A4"], 56, 4.5},
			},
		},
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
		{
			Volume: 0.13,
			Pan:    -0.15,
			Notes: []Note{
				{p["E3"], 0.5, 3.3}, {p["A3"], 1.0, 2.8},
				{p["E3"], 16.5, 3.3}, {p["A3"], 17.0, 2.8},
				{p["E3"], 32.5, 3.3}, {p["A3"], 33.0, 2.8},
				{p["E3"], 48.5, 3.3}, {p["A3"], 49.0, 2.8},
				{p["A3"], 4.5, 3.3}, {p["C4"], 5.0, 2.8},
				{p["A3"], 20.5, 3.3}, {p["C4"], 21.0, 2.8},
				{p["A3"], 36.5, 3.3}, {p["C4"], 37.0, 2.8},
				{p["A3"], 52.5, 3.3}, {p["C4"], 53.0, 2.8},
				{p["G3"], 8.5, 3.3}, {p["E4"], 9.0, 2.8},
				{p["G3"], 24.5, 3.3}, {p["E4"], 25.0, 2.8},
				{p["G3"], 40.5, 3.3}, {p["E4"], 41.0, 2.8},
				{p["D3"], 12.5, 3.3}, {p["B3"], 13.0, 2.8},
				{p["D3"], 28.5, 3.3}, {p["B3"], 29.0, 2.8},
				{p["D3"], 44.5, 3.3}, {p["B3"], 45.0, 2.8},
				{p["E3"], 56.5, 4.0}, {p["A3"], 57.0, 3.5},
			},
		},
	}

	return render(tracks, 61.5, beatSec)
}

func pianoSample(phase float64) float64 {
	phase = phase - math.Floor(phase)
	return math.Sin(2*math.Pi*phase)*0.55 +
		math.Sin(4*math.Pi*phase)*0.25 +
		math.Sin(6*math.Pi*phase)*0.12 +
		math.Sin(8*math.Pi*phase)*0.05 +
		math.Sin(10*math.Pi*phase)*0.03
}

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
