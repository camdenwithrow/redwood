package allocation

import (
	"fmt"
	"sort"

	"github.com/camdenwithrow/redwood/internal/config"
)

const maximumPort = 65535

func CalculatePorts(configuration config.Config, slot int) (map[string]int, error) {
	if slot < 0 {
		return nil, fmt.Errorf("slot must not be negative")
	}
	if len(configuration.Ports) == 0 {
		return map[string]int{}, nil
	}
	if configuration.PortStride <= 0 {
		return nil, fmt.Errorf("port_stride must be greater than zero")
	}

	labels := make([]string, 0, len(configuration.Ports))
	for label := range configuration.Ports {
		labels = append(labels, label)
	}
	sort.Strings(labels)

	ports := make(map[string]int, len(configuration.Ports))
	for _, label := range labels {
		basePort := configuration.Ports[label]
		if basePort < 1 || basePort > maximumPort {
			return nil, fmt.Errorf("base port for %q must be between 1 and %d", label, maximumPort)
		}
		if slot > (maximumPort-basePort)/configuration.PortStride {
			return nil, fmt.Errorf("calculated port for %q exceeds %d at slot %d", label, maximumPort, slot)
		}

		ports[label] = basePort + slot*configuration.PortStride
	}

	return ports, nil
}
