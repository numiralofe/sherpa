package status

import (
	"fmt"
	"os"
	"strings"

	"github.com/jrasell/sherpa/cmd/helper"
	"github.com/jrasell/sherpa/pkg/api"
	clientCfg "github.com/jrasell/sherpa/pkg/config/client"
	"github.com/jrasell/sherpa/pkg/config/scale"
	"github.com/sean-/sysexits"
	"github.com/spf13/cobra"
)

const (
	listOutputHeader = "ID|Job:Group|Status|Time"
	infoOutputHeader = "Job:Group|ChangeCount|Direction|Meta"
)

func RegisterCommand(rootCmd *cobra.Command) error {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Display the status output for scaling activities",
		Run: func(cmd *cobra.Command, args []string) {
			runStatus(cmd, args)
		},
	}
	rootCmd.AddCommand(cmd)
	scale.RegisterScaleStatusConfig(cmd)

	return nil
}

func runStatus(_ *cobra.Command, args []string) {
	if len(args) > 1 {
		fmt.Println("Too many arguments, expected 1 or 0, got", len(args))
		os.Exit(sysexits.Usage)
	}

	clientConfig := clientCfg.GetConfig()
	mergedConfig := api.DefaultConfig(&clientConfig)
	latestConfig := scale.GetScaleStatusConfig()

	client, err := api.NewClient(mergedConfig)
	if err != nil {
		fmt.Println("Error setting up Sherpa client:", err)
		os.Exit(sysexits.Software)
	}

	switch len(args) {
	case 0:
		os.Exit(runList(client, latestConfig.Latest))
	case 1:
		os.Exit(runInfo(client, args[0]))
	}
}

func runList(c *api.Client, latest bool) int {
	resp, err := c.Scale().List(latest)
	if err != nil {
		fmt.Println("Error getting scaling list:", err)
		os.Exit(sysexits.Software)
	}

	out := []string{listOutputHeader}

	for id, jobEvents := range resp {
		for jg, event := range jobEvents {
			out = append(out, fmt.Sprintf("%v|%s|%s|%v", id, jg, event.Status, helper.UnixNanoToHumanUTC(event.Time)))
		}
	}

	if len(out) > 1 {
		fmt.Println(helper.FormatList(out))
	}
	return sysexits.OK
}

func runInfo(c *api.Client, id string) int {
	resp, err := c.Scale().Info(id)
	if err != nil {
		fmt.Println("Error getting scaling info:", err)
		os.Exit(sysexits.Software)
	}

	var header []string

	events := []string{infoOutputHeader}
	for jobGroup, event := range resp {
		events = append(events, fmt.Sprintf("%s|%v|%v|%s",
			jobGroup, event.Details.Count, event.Details.Direction, strings.Join(metaToStrings(event.Meta), ",")))

		if len(header) == 0 {
			header = []string{
				fmt.Sprintf("ID|%s", id),
				fmt.Sprintf("EvalID|%v", event.EvalID),
				fmt.Sprintf("Status|%s", event.Status),
				fmt.Sprintf("Source|%v", event.Source),
				fmt.Sprintf("Time|%v", helper.UnixNanoToHumanUTC(event.Time)),
			}
		}
	}

	fmt.Println(helper.FormatKV(header))
	fmt.Println("")
	fmt.Println(helper.FormatList(events))

	return sysexits.OK
}

func metaToStrings(meta map[string]string) []string {
	out := []string{}
	for k, v := range meta {
		out = append(out, k+"="+v)
	}
	return out
}
