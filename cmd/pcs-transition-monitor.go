// This source code is licensed under the license found in the LICENSE file at
// the root directory of this source tree.
package cmd

import (
	"encoding/json"
	"os"
	"time"

	"github.com/spf13/cobra"
	"github.com/vbauerster/mpb/v8"
	"github.com/vbauerster/mpb/v8/decor"

	"github.com/OpenCHAMI/ochami/internal/log"
	"github.com/OpenCHAMI/ochami/pkg/client/pcs"
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
	Total       int `json:"total"`
	New         int `json:"new"`
	InProgress  int `json:"in-progress"`
	Failed      int `json:"failed"`
	Succeeded   int `json:"succeeded"`
	Unsupported int `json:"un-supported"`
}

// transitionProgress represents the progress of a PCS transition
type transitionProgress struct {
	Status     string               `json:"transitionStatus"`
	TaskCounts transitionTaskCounts `json:"taskCounts"`
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

// pcsTransitionMonitorCmd represents the "pcs transition monitor" command
var pcsTransitionMonitorCmd = &cobra.Command{
	Use:   "monitor <transition_id>",
	Args:  cobra.ExactArgs(1),
	Short: "Monitor a PCS transition",
	Long: `Abort a PCS transition.

See ochami-pcs(1) for more details.`,
	Example: `  # Monitor the progress of a transition
  ochami pcs transition monitor 8f252166-c53c-435e-8354-e69649537a0f`,
	Run: func(cmd *cobra.Command, args []string) {
		transitionID := args[0]

		// Without a base URI, we cannot do anything
		pcsBaseURI, err := getBaseURIPCS(cmd)
		if err != nil {
			log.Logger.Error().Err(err).Msg("failed to get base URI for PCS")
			logHelpError(cmd)
			os.Exit(1)
		}

		// This endpoint requires authentication, so a token is needed
		setTokenFromEnvVar(cmd)
		checkToken(cmd)

		// Create client to make request to PCS
		pcsClient, err := pcs.NewClient(pcsBaseURI, insecure)
		if err != nil {
			log.Logger.Fatal().Err(err).Msg("error creating new PCS client")
		}

		// Check if a CA certificate was passed and load it into client if valid
		useCACert(pcsClient.OchamiClient)

		p := mpb.New(mpb.WithWidth(64))

		newBar := createBar(p, transitionTaskStateNew)
		inProgressBar := createBar(p, transitionTaskStateInProgress)
		succeededBar := createBar(p, transitionTaskStateSucceeded)
		failedBar := createBar(p, transitionTaskStateFailed)

		// Poll transition state until it is complete or aborted
		for {
			transitionHttpEnv, err := pcsClient.GetTransition(transitionID, token)
			if err != nil {
				log.Logger.Fatal().Err(err).Msg("failed to get transition")
			}

			// Unmarshal the progress information
			var progress transitionProgress
			if err := json.Unmarshal(transitionHttpEnv.Body, &progress); err != nil {
				log.Logger.Fatal().Err(err).Msg("failed to unmarshal transition")
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

func init() {
	pcsTransitionMonitorCmd.Flags().IntVarP(&pollInterval, "poll-interval", "p", 1, "The interval at which to poll the transition status")
	pcsTransitionCmd.AddCommand(pcsTransitionMonitorCmd)
}
