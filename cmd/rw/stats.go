package main

import (
	"fmt"
	"time"

	"github.com/spf13/cobra"
)

func statsCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "stats",
		Short: "Show repository statistics",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			r, err := loadRepo()
			if err != nil {
				return err
			}

			projectRoot := parentDir(r.cfg.RewindDir)
			branch, _ := r.engine.Index.CurrentBranch()
			headCP, hasHead := r.engine.Index.CurrentCheckpoint()

			objectCount, totalBytes, _ := r.store.Stats()

			// First and latest checkpoint times.
			var firstTime, latestTime time.Time
			for _, cp := range r.engine.Index.Checkpoints {
				if firstTime.IsZero() || cp.CreatedAt.Before(firstTime) {
					firstTime = cp.CreatedAt
				}
				if latestTime.IsZero() || cp.CreatedAt.After(latestTime) {
					latestTime = cp.CreatedAt
				}
			}

			sectionTitle("nimbi stats")
			fmt.Println()
			kv("repository", projectRoot)
			kv("branch",     colorPurple+branch.Name+colorReset)
			if hasHead {
				kv("head", colorCyan+shortID(headCP.ID)+colorReset+"  "+dimP.Sprint(fmt.Sprintf("%q", headCP.Message)))
			}
			fmt.Println()
			hrule(40)
			fmt.Println()

			boldP.Println("  timeline")
			hrule(40)
			kv("checkpoints", fmt.Sprintf("%s%d%s", colorBold, len(r.engine.Index.Checkpoints), colorReset))
			kv("branches",    fmt.Sprintf("%s%d%s", colorBold, len(r.engine.Index.Branches), colorReset))
			fmt.Println()

			boldP.Println("  storage")
			hrule(40)
			kv("objects",     fmt.Sprintf("%s%d%s", colorBold, objectCount, colorReset))
			kv("size",        fmt.Sprintf("%s%s%s", colorBold, formatBytes(totalBytes), colorReset))
			kv("compression", dimP.Sprint("gzip compressed"))
			fmt.Println()

			boldP.Println("  activity")
			hrule(40)
			if !firstTime.IsZero() {
				elapsed := int64(time.Since(firstTime.Local()).Seconds())
				kv("first save", dimP.Sprint(humanTime(elapsed)))
			} else {
				kv("first save", dimP.Sprint("none"))
			}
			if !latestTime.IsZero() {
				elapsed := int64(time.Since(latestTime.Local()).Seconds())
				kv("last save", dimP.Sprint(humanTime(elapsed)))
			} else {
				kv("last save", dimP.Sprint("none"))
			}
			fmt.Println()

			return nil
		},
	}
}
