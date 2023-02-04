// Darek Slusarczyk alias marines marinesovitch 2021, 2022

package e2e_test

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// call flag.Parse() here if TestMain uses flags
	result := m.Run()
	os.Exit(result)
}
