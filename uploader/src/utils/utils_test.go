package utils

import (
	"fmt"
	"testing"
)

func TestExtractTime(t *testing.T) {
	stamp, err := ExtractTime("02-20160920135426-14.jpg")
	t.Log(stamp, err)
	if err != nil || fmt.Sprintf("%s", stamp) != "2016-09-20 13:54:26 +0000 UTC" {
		t.Error("Failed happy path")
	}

	stamp, err = ExtractTime("20160920135426-14.jpg")
	t.Log(stamp, err)
	if err == nil {
		t.Error("Expected an error")
	}

	stamp, err = ExtractTime("")
	t.Log(stamp, err)
	if err == nil {
		t.Error("Expected an error")
	}

	stamp, err = ExtractTime("02-201d0920135426-14.jpg")
	t.Log(stamp, err)
	if err == nil || stamp != nil {
		t.Error("Expected an error")
	}
}
