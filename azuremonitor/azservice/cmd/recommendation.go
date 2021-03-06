package cmd

import (
	"fmt"
	"github.com/Go/azuremonitor/azure/advisor"
	"github.com/Go/azuremonitor/common/terminal"
	c "github.com/Go/azuremonitor/config"
	"github.com/spf13/cobra"
	"os"
)

func init() {

	r, err := setRecommendationCommand()
	if err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	rootCmd.AddCommand(r)
}

func setRecommendationCommand() (*cobra.Command, error) {

	configuration, _ = c.GetCmdConfig()
	description := fmt.Sprintf("%s\n%s\n%s",
		configuration.Recommendation.DescriptionLine1,
		configuration.Recommendation.DescriptionLine2,
		configuration.Recommendation.DescriptionLine3)

	cmd := &cobra.Command{
		Use:   configuration.Recommendation.Command,
		Short: configuration.Recommendation.CommandComments,
		Long:  description}

	cmd.RunE = func(*cobra.Command, []string) error {
		terminal.Clear()
		recommendations := advisor.Recommendations{}
		recommendations.ExecuteRequest(&recommendations)
		recommendations.Print()
		return nil
	}
	return cmd, nil
}
