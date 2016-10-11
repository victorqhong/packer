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

type StepMountGuestAdditions struct {
	GuestAdditionsMode string
	GuestAdditionsPath string
	ControllerLocation string
	ControllerNumber   string
	Generation         uint
}

func (s *StepMountGuestAdditions) Run(state multistep.StateBag) multistep.StepAction {
	ui := state.Get("ui").(packer.Ui)

	if s.GuestAdditionsMode != "attach" {
		ui.Say("Skipping mounting Integration Services Setup Disk...")
		return multistep.ActionContinue
	}

	driver := state.Get("driver").(Driver)
	ui.Say("Mounting Integration Services Setup Disk...")

	vmName := state.Get("vmName").(string)

	// should be able to mount up to 60 additional iso images using SCSI
	// but Windows would only allow a max of 22 due to available drive letters
	// Will Windows assign DVD drives to A: and B: ?

	// For IDE, there are only 2 controllers (0,1) with 2 locations each (0,1)

	var controllerNumber uint
	var controllerLocation uint
	var err error

	if s.ControllerLocation == "" || s.ControllerNumber == "" {
		controllerNumber, controllerLocation, err = driver.CreateDvdDrive(vmName, s.GuestAdditionsPath, s.Generation)
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

		err = driver.CreateDvdDriveAt(vmName, controllerNumber, controllerLocation, s.GuestAdditionsPath, s.Generation)
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
	state.Put("guest.dvd.properties", dvdControllerProperties)

	ui.Say(fmt.Sprintf("Mounting Integration Services dvd drive %s ...", s.GuestAdditionsPath))
	err = driver.MountDvdDrive(vmName, s.GuestAdditionsPath, controllerNumber, controllerLocation)
	if err != nil {
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	log.Println(fmt.Sprintf("ISO %s mounted on DVD controller %v, location %v", s.GuestAdditionsPath, controllerNumber, controllerLocation))

	return multistep.ActionContinue
}

func (s *StepMountGuestAdditions) Cleanup(state multistep.StateBag) {
	if s.GuestAdditionsMode != "attach" {
		return
	}

	dvdControllerState := state.Get("guest.dvd.properties")

	if dvdControllerState == nil {
		return
	}

	dvdController := dvdControllerState.(DvdControllerProperties)
	ui := state.Get("ui").(packer.Ui)
	driver := state.Get("driver").(Driver)
	vmName := state.Get("vmName").(string)
	errorMsg := "Error unmounting Integration Services dvd drive: %s"

	ui.Say("Cleanup Integration Services dvd drive...")

	if dvdController.Existing {
		err := driver.UnmountDvdDrive(vmName, dvdController.ControllerNumber, dvdController.ControllerLocation)
		if err != nil {
			log.Print(fmt.Sprintf(errorMsg, err))
		}
	} else {
		err := driver.DeleteDvdDrive(vmName, dvdController.ControllerNumber, dvdController.ControllerLocation)
		if err != nil {
			log.Print(fmt.Sprintf(errorMsg, err))
		}
	}
}
