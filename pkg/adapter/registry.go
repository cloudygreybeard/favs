// Copyright 2026 cloudygreybeard
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Package adapter provides the registry for input and output adapters.
package adapter

import (
	"sort"
	"sync"

	"github.com/cloudygreybeard/favs/pkg/input"
	"github.com/cloudygreybeard/favs/pkg/output"
)

var (
	inputsMu  sync.RWMutex
	inputs    = make(map[string]input.Adapter)
	outputsMu sync.RWMutex
	outputs   = make(map[string]output.Adapter)
)

// RegisterInput registers an input adapter.
func RegisterInput(adapter input.Adapter) {
	inputsMu.Lock()
	defer inputsMu.Unlock()
	inputs[adapter.Name()] = adapter
}

// RegisterOutput registers an output adapter.
func RegisterOutput(adapter output.Adapter) {
	outputsMu.Lock()
	defer outputsMu.Unlock()
	outputs[adapter.Name()] = adapter
}

// GetInput returns an input adapter by name.
func GetInput(name string) (input.Adapter, bool) {
	inputsMu.RLock()
	defer inputsMu.RUnlock()
	a, ok := inputs[name]
	return a, ok
}

// GetOutput returns an output adapter by name.
func GetOutput(name string) (output.Adapter, bool) {
	outputsMu.RLock()
	defer outputsMu.RUnlock()
	a, ok := outputs[name]
	return a, ok
}

// ListInputs returns all registered input adapter names.
func ListInputs() []string {
	inputsMu.RLock()
	defer inputsMu.RUnlock()
	names := make([]string, 0, len(inputs))
	for name := range inputs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// ListOutputs returns all registered output adapter names.
func ListOutputs() []string {
	outputsMu.RLock()
	defer outputsMu.RUnlock()
	names := make([]string, 0, len(outputs))
	for name := range outputs {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// AllInputs returns all registered input adapters.
func AllInputs() []input.Adapter {
	inputsMu.RLock()
	defer inputsMu.RUnlock()
	adapters := make([]input.Adapter, 0, len(inputs))
	for _, a := range inputs {
		adapters = append(adapters, a)
	}
	return adapters
}

// AllOutputs returns all registered output adapters.
func AllOutputs() []output.Adapter {
	outputsMu.RLock()
	defer outputsMu.RUnlock()
	adapters := make([]output.Adapter, 0, len(outputs))
	for _, a := range outputs {
		adapters = append(adapters, a)
	}
	return adapters
}

// AvailableInputs returns input adapters that are currently available.
func AvailableInputs() []input.Adapter {
	inputsMu.RLock()
	defer inputsMu.RUnlock()
	var available []input.Adapter
	for _, a := range inputs {
		if a.Available() {
			available = append(available, a)
		}
	}
	return available
}
