package crawler

import (
	"reflect"
	"testing"
)

func TestPhotoURLs(t *testing.T) {
	in := []string{
		"https://s.c24.media/images/default/certified/certified-filled.svg",
		"https://img.chrono24.com/images/uhren/1-a-Square480.jpg",
		"https://example.com/icon.SVG?v=2",
		"data:image/gif;base64,R0lGOD",
		"https://example.com/watch.webp#x",
	}
	want := []string{"https://img.chrono24.com/images/uhren/1-a-Square480.jpg", "https://example.com/watch.webp#x"}
	if got := PhotoURLs(in); !reflect.DeepEqual(got, want) {
		t.Errorf("PhotoURLs = %v, want %v", got, want)
	}
	if got := PhotoURLs(nil); len(got) != 0 {
		t.Errorf("PhotoURLs(nil) = %v", got)
	}
}
