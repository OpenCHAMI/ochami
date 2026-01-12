// SPDX-FileCopyrightText: © 2024-2025 Triad National Security, LLC. All rights reserved.
// SPDX-FileCopyrightText: © 2025 OpenCHAMI a Series of LF Projects, LLC
//
// SPDX-License-Identifier: MIT

package transition

import (
	"encoding/json"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"

	"github.com/OpenCHAMI/ochami/internal/cli"
	"github.com/OpenCHAMI/ochami/internal/log"

	pcs_lib "github.com/OpenCHAMI/ochami/internal/cli/pcs"
)

var pollInterval int = 1

// Possible transition states
const (
	transitionStatusNew           = "new"
	transitionStatusInProgress    = "in-progress"
	transitionStatusCompleted     = "completed"
	transitionStatusAborted       = "aborted"
	transitionStatusAbortSignaled = "abort-signaled"
)

// Possible transition task states
const (
	transitionTaskStateNew        = "new"
	transitionTaskStateInProgress = "in-progress"
	transitionTaskStateFailed     = "failed"
	transitionTaskStateSucceeded  = "succeeded"
)

// transitionTaskCounts represents the counts of tasks in a PCS transition
type transitionTaskCounts struct {
	Total       int `json:"total" yaml:"total"`
	New         int `json:"new" yaml:"new"`
	InProgress  int `json:"in-progress" yaml:"in-progress"`
	Failed      int `json:"failed" yaml:"failed"`
	Succeeded   int `json:"succeeded" yaml:"succeeded"`
	Unsupported int `json:"un-supported" yaml:"un-supported"`
}

// transitionProgress represents the progress of a PCS transition
type transitionProgress struct {
	Status     string               `json:"transitionStatus" yaml:"transitionStatus"`
	TaskCounts transitionTaskCounts `json:"taskCounts" yaml:"taskCounts"`
}

// Create and style a progress bar
func createBar(p *mpb.Progress, name string) *mpb.Bar {
	return p.AddBar(0, mpb.PrependDecorators(
		decor.Name(name, decor.WC{W: 12, C: decor.DindentRight}),
	),
		mpb.AppendDecorators(
			decor.Percentage(),
		),
	)
}

func newCmdTransitionMonitor() *cobra.Command {
	// transitionMonitorCmd represents the "pcs transition monitor" command
	var transitionMonitorCmd = &cobra.Command{
		Use:   "monitor <transition_id>",
		Args:  cobra.ExactArgs(1),
		Short: "Monitor a PCS transition",
		Long: `Abort a PCS transition.

See ochami-pcs(1) for more details.`,
		Example: `  # Monitor the progress of a transition
  ochami pcs transition monitor 8f252166-c53c-435e-8354-e69649537a0f`,
		Run: func(cmd *cobra.Command, args []string) {
			transitionID := args[0]

			// Create client to use for requests
			pcsClient := pcs_lib.GetClient(cmd)

			// Handle token for this command
			cli.HandleToken(cmd)

			p := mpb.New(mpb.WithWidth(64))

			newBar := createBar(p, transitionTaskStateNew)
			inProgressBar := createBar(p, transitionTaskStateInProgress)
			succeededBar := createBar(p, transitionTaskStateSucceeded)
			failedBar := createBar(p, transitionTaskStateFailed)

			// Poll transition state until it is complete or aborted
			for {
				transitionHttpEnv, err := pcsClient.GetTransition(transitionID, cli.Token)
				if err != nil {
					log.Logger.Error().Err(err).Msg("failed to get transition")
					cli.LogHelpError(cmd)
					os.Exit(1)
				}

				// Unmarshal the progress information
				var progress transitionProgress
				if err := json.Unmarshal(transitionHttpEnv.Body, &progress); err != nil {
					log.Logger.Error().Err(err).Msg("failed to unmarshal transition")
					cli.LogHelpError(cmd)
					os.Exit(1)
				}

				// Set the totals for each bar
				for _, bar := range []*mpb.Bar{
					succeededBar,
					failedBar,
					inProgressBar,
					newBar,
				} {
					bar.SetTotal(int64(progress.TaskCounts.Total), false)
				}

				// Update the progress bars
				newBar.SetCurrent(int64(progress.TaskCounts.New))
				succeededBar.SetCurrent(int64(progress.TaskCounts.Succeeded))
				failedBar.SetCurrent(int64(progress.TaskCounts.Failed))
				inProgressBar.SetCurrent(int64(progress.TaskCounts.InProgress))

				// Check if the transition is complete
				if progress.Status == transitionStatusCompleted || progress.Status == transitionStatusAborted {
					break
				}

				// Sleep poll interval
				time.Sleep(time.Duration(pollInterval) * time.Second)
			}

			p.Shutdown()
		},
	}

	// Create flags
	transitionMonitorCmd.Flags().IntVarP(&pollInterval, "poll-interval", "p", 1, "The interval at which to poll the transition status")

	return transitionMonitorCmd
}
