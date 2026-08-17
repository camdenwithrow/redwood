package allocation

import (
	"fmt"
	"strings"
)

func (store Store) Assign(branch string) (int, error) {
	state, err := store.Load()
	if err != nil {
		return 0, err
	}

	if slot, exists := state.Slots[branch]; exists {
		return slot, nil
	}

	slot, err := state.Assign(branch)
	if err != nil {
		return 0, fmt.Errorf("assign slot for branch %q: %w", branch, err)
	}
	if err := store.Save(state); err != nil {
		return 0, err
	}

	return slot, nil
}

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
