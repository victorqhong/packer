// Copyright (c) Microsoft Open Technologies, Inc.
// All Rights Reserved.
// Licensed under the Apache License, Version 2.0.
// See License.txt in the project root for license information.
package common

import (
	"fmt"
	"log"
	"strconv"

	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/packer"
)

type StepMountDvdDrive struct {
	ControllerNumber   string
	ControllerLocation string
	Generation         uint
}

func (s *StepMountDvdDrive) Run(state multistep.StateBag) multistep.StepAction {
	driver := state.Get("driver").(Driver)
	ui := state.Get("ui").(packer.Ui)

	errorMsg := "Error mounting dvd drive: %s"
	vmName := state.Get("vmName").(string)
	isoPath := state.Get("iso_path").(string)

	// should be able to mount up to 60 additional iso images using SCSI
	// but Windows would only allow a max of 22 due to available drive letters
	// Will Windows assign DVD drives to A: and B: ?

	// For IDE, there are only 2 controllers (0,1) with 2 locations each (0,1)

	var controllerNumber uint
	var controllerLocation uint
	var err error

	if s.ControllerLocation == "" || s.ControllerNumber == "" {
		controllerNumber, controllerLocation, err = driver.CreateDvdDrive(vmName, isoPath, s.Generation)
		if err != nil {
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	} else {
		number, err := strconv.ParseUint(s.ControllerNumber, 10, 32)
		if err != nil {
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		location, err := strconv.ParseUint(s.ControllerLocation, 10, 32)
		if err != nil {
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		controllerNumber = uint(number)
		controllerLocation = uint(location)

		err = driver.CreateDvdDriveAt(vmName, controllerNumber, controllerLocation, isoPath, s.Generation)
		if err != nil {
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}

	var dvdControllerProperties DvdControllerProperties
	dvdControllerProperties.ControllerNumber = controllerNumber
	dvdControllerProperties.ControllerLocation = controllerLocation
	dvdControllerProperties.Existing = false

	state.Put("os.dvd.properties", dvdControllerProperties)

	ui.Say(fmt.Sprintf("Setting boot drive to os dvd drive %s ...", isoPath))
	err = driver.SetBootDvdDrive(vmName, controllerNumber, controllerLocation, s.Generation)
	if err != nil {
		err := fmt.Errorf(errorMsg, err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	ui.Say(fmt.Sprintf("Mounting os dvd drive %s ...", isoPath))
	err = driver.MountDvdDrive(vmName, isoPath, controllerNumber, controllerLocation)
	if err != nil {
		err := fmt.Errorf(errorMsg, err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

func (s *StepMountDvdDrive) Cleanup(state multistep.StateBag) {
	dvdControllerState := state.Get("os.dvd.properties")

	if dvdControllerState == nil {
		return
	}

	dvdController := dvdControllerState.(DvdControllerProperties)
	driver := state.Get("driver").(Driver)
	vmName := state.Get("vmName").(string)
	ui := state.Get("ui").(packer.Ui)
	errorMsg := "Error unmounting os dvd drive: %s"

	ui.Say("Clean up os dvd drive...")

	if dvdController.Existing {
		err := driver.UnmountDvdDrive(vmName, dvdController.ControllerNumber, dvdController.ControllerLocation)
		if err != nil {
			err := fmt.Errorf("Error unmounting dvd drive: %s", err)
			log.Print(fmt.Sprintf(errorMsg, err))
		}
	} else {
		err := driver.DeleteDvdDrive(vmName, dvdController.ControllerNumber, dvdController.ControllerLocation)
		if err != nil {
			err := fmt.Errorf("Error deleting dvd drive: %s", err)
			log.Print(fmt.Sprintf(errorMsg, err))
		}
	}
}
