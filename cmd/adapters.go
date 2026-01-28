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

package cmd

import (
	"fmt"

	"github.com/cloudygreybeard/favs/pkg/adapter"
	"github.com/spf13/cobra"
)

var adaptersCmd = &cobra.Command{
	Use:   "adapters",
	Short: "List registered input and output adapters",
	Long:  `Lists all registered input (source) and output (renderer) adapters.`,
	RunE:  runAdapters,
}

func runAdapters(cmd *cobra.Command, args []string) error {
	fmt.Println("Input Adapters (bookmark sources):")
	fmt.Println()

	for _, name := range adapter.ListInputs() {
		inp, _ := adapter.GetInput(name)
		status := "not available"
		if inp.Available() {
			status = "available"
		}
		fmt.Printf("  %-12s %-20s [%s]\n", name, inp.DisplayName(), status)
	}

	fmt.Println()
	fmt.Println("Output Adapters (renderers):")
	fmt.Println()

	for _, name := range adapter.ListOutputs() {
		out, _ := adapter.GetOutput(name)
		fmt.Printf("  %-12s %-20s %v\n", name, out.DisplayName(), out.Extensions())
	}

	return nil
}
