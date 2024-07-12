package cmd

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	jsonschemaparser "github.com/sighupio/md-gen/internal/json-schema-parser"
	mdgen "github.com/sighupio/md-gen/internal/md-gen"
	"github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

func NewGenCmd() *cobra.Command {
	genCmd := &cobra.Command{
		Use:   "gen",
		Short: "Generates a markdown file from a json schema file",
		Long:  "Generates a markdown file from a json schema file",
		RunE: func(cmd *cobra.Command, args []string) error {
			input, err := cmd.Flags().GetString("input")
			if err != nil {
				panic(err)
			}

			output, err := cmd.Flags().GetString("output")
			if err != nil {
				panic(err)
			}

			overwrite, err := cmd.Flags().GetBool("overwrite")
			if err != nil {
				panic(err)
			}

			banner, err := cmd.Flags().GetString("banner")
			if err != nil {
				panic(err)
			}

			destination := output

			if !overwrite {
				currentFolder, err := os.Getwd()
				if err != nil {
					panic(err)
				}

				timestamp := time.Now().Unix()

				extractedFileName := filepath.Base(output)

				extractedFileName = extractedFileName[:len(extractedFileName)-len(filepath.Ext(extractedFileName))]

				destination = currentFolder + "/" + extractedFileName + "-" + strconv.FormatInt(timestamp, 10) + ".md"
			}

			logrus.Infof("Input file: %s", input)
			logrus.Infof("Output file: %s", destination)
			logrus.Infof("Banner file: %s", banner)

			jsonParser := jsonschemaparser.NewBaseParser(input)

			jsonSchema, err := jsonParser.Parse()
			if err != nil {
				return err
			}

			inputDir := filepath.Dir(input)

			mdGen := mdgen.NewBaseGenerator(destination, jsonSchema, inputDir)

			genOut, err := mdGen.Generate()
			if err != nil {
				return err
			}

			if banner != "" {
				bannerContent, err := os.ReadFile(banner)
				if err != nil {
					return err
				}

				genOut = append(bannerContent, genOut...)
			}

			err = os.WriteFile(destination, genOut, 0644)
			if err != nil {
				return err
			}

			return nil
		},
	}

	genCmd.Flags().StringP("input", "i", "", "Input markdown file")
	genCmd.Flags().StringP("output", "o", "", "Output json file")
	genCmd.Flags().StringP("banner", "b", "", "Banner file")
	genCmd.Flags().BoolP("overwrite", "w", false, "Overwrite output file")

	if err := genCmd.MarkFlagRequired("input"); err != nil {
		panic(err)
	}

	if err := genCmd.MarkFlagRequired("output"); err != nil {
		panic(err)
	}

	return genCmd
}
