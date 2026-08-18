package allocation

import (
	"strings"
	"testing"

	"github.com/camdenwithrow/redwood/internal/config"
)

func TestCalculatePorts(t *testing.T) {
	configuration := config.Config{
		PortStride: 100,
		Ports: map[string]int{
			"frontend":  3000,
			"backend":   8080,
			"simulator": 8081,
		},
	}

	ports, err := CalculatePorts(configuration, 2)
	if err != nil {
		t.Fatalf("CalculatePorts() error = %v", err)
	}
	want := map[string]int{
		"frontend":  3200,
		"backend":   8280,
		"simulator": 8281,
	}
	for label, wantPort := range want {
		if ports[label] != wantPort {
			t.Fatalf("CalculatePorts() %s port = %d, want %d", label, ports[label], wantPort)
		}
	}
}

func TestCalculatePortsAtSlotZeroUsesBasePorts(t *testing.T) {
	configuration := config.Config{
		PortStride: 100,
		Ports:      map[string]int{"docs": 4000},
	}

	ports, err := CalculatePorts(configuration, 0)
	if err != nil {
		t.Fatalf("CalculatePorts() error = %v", err)
	}
	if ports["docs"] != 4000 {
		t.Fatalf("CalculatePorts() docs port = %d, want 4000", ports["docs"])
	}
}

func TestCalculatePortsAllowsEmptyConfiguration(t *testing.T) {
	ports, err := CalculatePorts(config.Config{}, 2)
	if err != nil {
		t.Fatalf("CalculatePorts() error = %v", err)
	}
	if len(ports) != 0 {
		t.Fatalf("CalculatePorts() = %v, want no ports", ports)
	}
}

func TestCalculatePortsRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name          string
		configuration config.Config
		slot          int
		want          string
	}{
		{name: "negative slot", configuration: config.Config{PortStride: 100}, slot: -1, want: "slot must not be negative"},
		{name: "invalid stride", configuration: config.Config{Ports: map[string]int{"web": 3000}}, want: "port_stride must be greater than zero"},
		{name: "invalid base port", configuration: config.Config{PortStride: 100, Ports: map[string]int{"web": 70000}}, want: `base port for "web" must be between 1 and 65535`},
		{name: "calculated port too high", configuration: config.Config{PortStride: 100, Ports: map[string]int{"web": 65500}}, slot: 1, want: `calculated port for "web" exceeds 65535 at slot 1`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := CalculatePorts(test.configuration, test.slot)
			if err == nil {
				t.Fatal("CalculatePorts() error = nil, want validation error")
			}
			if !strings.Contains(err.Error(), test.want) {
				t.Fatalf("CalculatePorts() error = %q, want it to contain %q", err, test.want)
			}
		})
	}
}
