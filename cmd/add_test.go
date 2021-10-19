package cmd

import "testing"

func TestCase(t *testing.T) {
	truncated := TruncatePath("/media/choiway/garynnon/freenas/google_photos/Photos from 2007/IMG_0493.jpg")

	if truncated != "Photos from 2007/IMG_0493.jpg" {
		t.Log(truncated)
		t.Fatalf("truncated should be %s", truncated)
	}
}
