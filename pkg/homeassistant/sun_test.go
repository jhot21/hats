package homeassistant

import (
	"testing"
)

func TestDirectnessToWindow(t *testing.T) {
	s := Sun{
		Elevation: 10.0,
		Azimuth:   100.0,
	}

	// 30 degrees between azimuths
	test1 := s.DirectnessToWindow(70)
	want1 := (90.0 - 30.0) / 90.0
	if test1 != want1 {
		t.Fatalf("Got %f, Expected %f", test1, want1)
	}

	// 10 degrees between azimuths
	test2 := s.DirectnessToWindow(90)
	want2 := (90.0 - 10.0) / 90.0
	if test2 != want2 {
		t.Fatalf("Got %f, Expected %f", test2, want2)
	}

	// 90 degress between azimuths
	test3 := s.DirectnessToWindow(10)
	want3 := 0.0
	if test3 != want3 {
		t.Fatalf("Got %f, Expected %f", test3, want3)
	}

	// > 90 degrees
	test4 := s.DirectnessToWindow(5)
	want4 := -1.0
	if test4 != want4 {
		t.Fatalf("Got %f, Expected %f", test4, want4)
	}

	// Elevation > azimuth
	test5 := s.DirectnessToWindow(95)
	want5 := (90.0 - 10.0) / 90.0
	if test5 != want5 {
		t.Fatalf("Got %f, Expected %f", test5, want5)
	}
}
