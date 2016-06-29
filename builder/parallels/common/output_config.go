package common

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/mitchellh/packer/common"
	"github.com/mitchellh/packer/template/interpolate"
)

type OutputConfig struct {
	OutputDir string `mapstructure:"output_directory"`
}

func (c *OutputConfig) Prepare(ctx *interpolate.Context, pc *common.PackerConfig) []error {
	if c.OutputDir == "" {
		c.OutputDir = fmt.Sprintf("output-%s", pc.PackerBuildName)
	}

	var errs []error
	fmt.Println("OutputDirBefore=", c.OutputDir)

	if !filepath.IsAbs(c.OutputDir) {
		outputDir, err := filepath.Abs(c.OutputDir)
		if err != nil {
			errs = append(errs, err)
			return errs
		} else {
			c.OutputDir = outputDir
		}
	}

	if !pc.PackerForce {
		fmt.Println("OutputDirAfter=", c.OutputDir)

		if _, err := os.Stat(c.OutputDir); err == nil {
			errs = append(errs, fmt.Errorf(
				"Output directory '%s' already exists. It must not exist.", c.OutputDir))
		}
	}

	return errs
}
