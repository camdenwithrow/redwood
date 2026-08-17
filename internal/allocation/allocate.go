package allocation

import (
	"fmt"
	"strings"
)

func (state *State) Assign(branch string) (int, error) {
	if strings.TrimSpace(branch) == "" {
		return 0, fmt.Errorf("branch name must not be empty")
	}
	if err := state.validate(); err != nil {
		return 0, err
	}
	if state.Slots == nil {
		state.Slots = make(map[string]int)
	}

	if slot, exists := state.Slots[branch]; exists {
		return slot, nil
	}

	usedSlots := make(map[int]struct{}, len(state.Slots))
	for _, slot := range state.Slots {
		usedSlots[slot] = struct{}{}
	}

	for slot := 0; ; slot++ {
		if _, used := usedSlots[slot]; used {
			continue
		}

		state.Slots[branch] = slot
		return slot, nil
	}
}
