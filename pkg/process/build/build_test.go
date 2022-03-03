package process

import (
	"testing"
)

func TestBuild(t *testing.T) {

	t.Run("Overwrite should NOT be allowed for semanticVersion", func(t *testing.T) {
		isSnapshot := false
		tagWith := ""
		tags := []string{"1.3.1"}
		semanticVersion := "1.3.1"
		completeVersion := ""
		err := checkTagsForOverwrite(isSnapshot, tags, tagWith, semanticVersion, completeVersion)

		if err == nil {
			t.Fatal("Overwrite should not be allowed")
		}
	})

	t.Run("Overwrite should NOT be allowed for completeVersion", func(t *testing.T) {
		isSnapshot := false
		tagWith := ""
		tags := []string{"1.3.1"}
		semanticVersion := ""
		completeVersion := "1.3.1"
		err := checkTagsForOverwrite(isSnapshot, tags, tagWith, semanticVersion, completeVersion)

		if err == nil {
			t.Fatal("Overwrite should not be allowed")
		}
	})

	t.Run("Overwrite should NOT be allowed for tagWith", func(t *testing.T) {
		isSnapshot := false
		tagWith := "1.3.1"
		tags := []string{"1.3.1"}
		semanticVersion := ""
		completeVersion := ""
		err := checkTagsForOverwrite(isSnapshot, tags, tagWith, semanticVersion, completeVersion)

		if err == nil {
			t.Fatal("Overwrite should be allowed for tagWith")
		}
	})

	t.Run("Overwrite should be allowed for snapshot", func(t *testing.T) {
		isSnapshot := true
		tagWith := ""
		tags := []string{"1.3.1"}
		semanticVersion := "1.3.1"
		completeVersion := ""
		err := checkTagsForOverwrite(isSnapshot, tags, tagWith, semanticVersion, completeVersion)

		if err != nil {
			t.Fatal("Overwrite should be allowed for snapshot")
		}
	})

	t.Run("Overwrite should be allowed for tagWith-snapshot", func(t *testing.T) {
		isSnapshot := false
		tagWith := "1.3.1-SNAPSHOT"
		tags := []string{"1.3.1"}
		semanticVersion := ""
		completeVersion := ""
		err := checkTagsForOverwrite(isSnapshot, tags, tagWith, semanticVersion, completeVersion)

		if err != nil {
			t.Fatal("Overwrite should be allowed for tagWith-snapshot")
		}
	})

}
