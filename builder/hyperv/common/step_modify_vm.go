package common

import (
	"fmt"
	"strings"

	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/packer"
	"github.com/mitchellh/packer/powershell"
	"github.com/mitchellh/packer/template/interpolate"
)

type commandTemplate struct {
	Name string
	Path string
}

type StepModifyVM struct {
	ModifyCommands []string
	Ctx            interpolate.Context
}

func (s *StepModifyVM) Run(state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)
	vmName := state.Get("vmName").(string)
	path := state.Get("packerTempDir").(string)

	if len(s.ModifyCommands) == 0 {
		return multistep.ActionContinue
	}

	s.Ctx.Data = &commandTemplate{
		Name: vmName,
		Path: path,
	}

	commandList := []string{}

	for _, commandTemplate := range s.ModifyCommands {
		command, err := interpolate.Render(commandTemplate, &s.Ctx)
		if err != nil {
			err := fmt.Errorf("Error preparing modifyvm command: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		ui.Message(fmt.Sprintf("Adding: %s", command))

		commandList = append(commandList, command)
	}

	commands := strings.Join(commandList, "\n")

	var ps powershell.PowerShellCmd
	cmdOut, err := ps.Output(commands)

	if err != nil {
		err := fmt.Errorf("Error executing modifyvm commands: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	ui.Message(fmt.Sprintf("modifyvm output:\n%s", cmdOut))

	return multistep.ActionContinue
}

func (*StepModifyVM) Cleanup(multistep.StateBag) {}
