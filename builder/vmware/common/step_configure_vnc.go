package common

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"strconv"

	"github.com/mitchellh/multistep"
	"github.com/mitchellh/packer/packer"
)

// This step configures the VM to enable the VNC server.
//
// Uses:
//   ui     packer.Ui
//   vmx_path string
//
// Produces:
//   vnc_port uint - The port that VNC is configured to listen on.
type StepConfigureVNC struct {
	VNCPortMin uint
	VNCPortMax uint
}

type VNCAddressFinder interface {
	VNCAddress(uint, uint) (string, uint, error)
}

func (StepConfigureVNC) VNCAddress(portMin, portMax uint) (string, uint, error) {
	// Find an open VNC port. Note that this can still fail later on
	// because we have to release the port at some point. But this does its
	// best.
	var vncPort uint
	portRange := int(portMax - portMin)
	for {
		if portRange > 0 {
			vncPort = uint(rand.Intn(portRange)) + portMin
		} else {
			vncPort = portMin
		}

		log.Printf("Trying port: %d", vncPort)
		l, err := net.Listen("tcp", fmt.Sprintf(":%d", vncPort))
		if err == nil {
			defer l.Close()
			break
		}
	}
	return "127.0.0.1", vncPort, nil
}

func (s *StepConfigureVNC) Run(state multistep.StateBag) multistep.StepAction {
	driver := state.Get("driver").(Driver)
	ui := state.Get("ui").(packer.Ui)
	vmxPath := state.Get("vmx_path").(string)

	f, err := os.Open(vmxPath)
	if err != nil {
		err := fmt.Errorf("Error reading VMX data: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	vmxBytes, err := ioutil.ReadAll(f)
	if err != nil {
		err := fmt.Errorf("Error reading VMX data: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	vmxData := ParseVMX(string(vmxBytes))
	vmxData["remotedisplay.vnc.enabled"] = "TRUE"

	if vncPortString, ok := vmxData["remotedisplay.vnc.port"]; ok {
		vncIp := "127.0.0.1"

		vncPort, err := strconv.ParseUint(vncPortString, 10, 64) 
		if err != nil {
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
				
		log.Printf("VNC port specified: %d", vncPort)

		state.Put("vnc_port", vncPort)
		state.Put("vnc_ip", vncIp)		
	} else {
		var vncFinder VNCAddressFinder
		if finder, ok := driver.(VNCAddressFinder); ok {
			vncFinder = finder
		} else {
			vncFinder = s
		}
		log.Printf("Looking for available port between %d and %d", s.VNCPortMin, s.VNCPortMax)
		vncIp, vncPort, err := vncFinder.VNCAddress(s.VNCPortMin, s.VNCPortMax)
		if err != nil {
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		log.Printf("Found available VNC port: %d", vncPort)
		
		vmxData["remotedisplay.vnc.port"] = fmt.Sprintf("%d", vncPort)
		
		state.Put("vnc_port", vncPort)
		state.Put("vnc_ip", vncIp)
	}

	if err := WriteVMX(vmxPath, vmxData); err != nil {
		err := fmt.Errorf("Error writing VMX data: %s", err)
		state.Put("error", err)
		ui.Error(err.Error())
		return multistep.ActionHalt
	}

	return multistep.ActionContinue
}

func (StepConfigureVNC) Cleanup(multistep.StateBag) {
}
