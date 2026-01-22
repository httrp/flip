package commands

import "testing"

func TestSetIconTheme(t *testing.T) {
	// Test emoji theme
	SetIconTheme("emoji")
	if IconCheck != "✅" {
		t.Errorf("Emoji theme: IconCheck should be ✅, got %s", IconCheck)
	}
	if IconBrain != "🧠" {
		t.Errorf("Emoji theme: IconBrain should be 🧠, got %s", IconBrain)
	}

	// Test mixed theme
	SetIconTheme("mixed")
	if IconCheck != "✅" {
		t.Errorf("Mixed theme: IconCheck should be ✅, got %s", IconCheck)
	}
	if IconActive != "*" {
		t.Errorf("Mixed theme: IconActive should be *, got %s", IconActive)
	}

	// Test ascii theme (default)
	SetIconTheme("ascii")
	if IconCheck != "[OK]" {
		t.Errorf("ASCII theme: IconCheck should be [OK], got %s", IconCheck)
	}
	if IconBrain != "+" {
		t.Errorf("ASCII theme: IconBrain should be +, got %s", IconBrain)
	}

	// Test unknown theme defaults to ascii
	SetIconTheme("unknown")
	if IconCheck != "[OK]" {
		t.Errorf("Unknown theme: should default to ASCII, got %s", IconCheck)
	}
}

func TestIconsAreNotEmpty(t *testing.T) {
	// Ensure all icons have values after setting any theme
	SetIconTheme("ascii")
	
	icons := map[string]string{
		"IconCheck":      IconCheck,
		"IconError":      IconError,
		"IconActive":     IconActive,
		"IconDefault":    IconDefault,
		"IconBrain":      IconBrain,
		"IconWorkspace":  IconWorkspace,
		"IconInfo":       IconInfo,
		"IconWarning":    IconWarning,
		"IconQuestion":   IconQuestion,
		"IconArrow":      IconArrow,
		"IconSuccess":    IconSuccess,
		"IconTree":       IconTree,
		"IconTreeBranch": IconTreeBranch,
		"IconTreeLast":   IconTreeLast,
		"IconTreeSpace":  IconTreeSpace,
		"IconTreeLine":   IconTreeLine,
	}

	for name, value := range icons {
		if value == "" {
			t.Errorf("%s should not be empty", name)
		}
	}
}
