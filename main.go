package main

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"golang.org/x/exp/slices"
)

var (
	lightName = flag.String("p", "", "name of the light to use, same name as in path /sys/class/backlight")
)

func main() {
	flag.Parse()

	lights, err := listBacklights()
	if err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}

	if *lightName == "" {
		flag.Usage()
		fmt.Fprintf(os.Stderr, "Available backlights: %s\n", strings.Join(lights, " "))
		os.Exit(1)
	} else if _, contains := slices.BinarySearch(lights, *lightName); !contains {
		fmt.Fprintf(os.Stderr, "Did not find backlight %q. Available backlights: %s\n", *lightName, strings.Join(lights, " "))
		os.Exit(1)
	}

	brightness, err := readLight(*lightName)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get current status of light: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Current brightness is %.0f%%\n", brightness.Percent())

	args := flag.Args()
	if len(args) == 0 {
		return
	}

	var percent float64
	_, err = fmt.Sscanf(args[0], "%f", &percent)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Expected argument %q to be a number: %v\n", args[0], err)
		os.Exit(1)
	}
	brightness.SetPercent(percent)
	if err := writeLight(*lightName, brightness); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to set brightness: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Set brightness to %.0f%%\n", brightness.Percent())
}

func listBacklights() ([]string, error) {
	subdirs, err := ioutil.ReadDir("/sys/class/backlight")
	if err != nil {
		return nil, fmt.Errorf("failed to read /sys/class/backlight: %w", err)
	}
	var result []string
	for _, d := range subdirs {
		result = append(result, d.Name())
	}

	if len(result) == 0 {
		return nil, errors.New("no backlights found")
	}
	return result, nil
}

type Brightness struct {
	Current, Max int
}

func (b Brightness) Percent() float64 {
	return 100. * float64(b.Current) / float64(b.Max)
}

// SetPercent sets the Current brightness to p percent of the Max.
// p must be between 0 and 100.
func (b *Brightness) SetPercent(p float64) {
	b.Current = int((p / 100.) * float64(b.Max))
}

func readLight(name string) (Brightness, error) {
	currentPath := fmt.Sprintf("/sys/class/backlight/%s/brightness", name)
	maxPath := fmt.Sprintf("/sys/class/backlight/%s/max_brightness", name)
	current, err := readIntFile(currentPath)
	if err != nil {
		return Brightness{}, err
	}
	max, err := readIntFile(maxPath)
	if err != nil {
		return Brightness{}, err
	}

	return Brightness{
		Current: current, Max: max,
	}, nil
}

func writeLight(name string, b Brightness) error {
	currentPath := fmt.Sprintf("/sys/class/backlight/%s/brightness", name)
	f, err := os.Create(currentPath)
	if err != nil {
		return err
	}
	defer f.Close()

	_, err = fmt.Fprintf(f, "%d", b.Current)
	return err
}

func readIntFile(path string) (int, error) {
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	var result int
	_, err = fmt.Fscanf(f, "%d", &result)
	if err != nil {
		return 0, err
	}
	return result, nil
}
