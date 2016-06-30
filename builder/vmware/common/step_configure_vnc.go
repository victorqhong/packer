package common

import (
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net"
	"os"
	"runtime"
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
	VNCBindAddress string
	VNCPortMin     uint
	VNCPortMax     uint
}

type VNCAddressFinder interface {
	VNCAddress(string, uint, uint) (string, uint, error)
}

func (StepConfigureVNC) VNCAddress(vncBindAddress string, portMin, portMax uint) (string, uint, error) {
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
		l, err := net.Listen("tcp", fmt.Sprintf("%s:%d", vncBindAddress, vncPort))
		if err == nil {
			defer l.Close()
			break
		}
	}
	return vncBindAddress, vncPort, nil
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

	if len(s.VNCBindAddress) > 0 {
		log.Printf("VNC ip specified: %d", s.VNCBindAddress)
		vmxData["remotedisplay.vnc.ip"] = fmt.Sprintf("%s", s.VNCBindAddress)
		state.Put("vnc_ip", s.VNCBindAddress)
	} else if vncIpString, ok := vmxData["remotedisplay.vnc.ip"]; ok {
		s.VNCBindAddress = vncIpString
		log.Printf("VNC ip specified: %d", s.VNCBindAddress)
		state.Put("vnc_ip", s.VNCBindAddress)
	} else {
		var ipFinder HostIPFinder
		if finder, ok := driver.(HostIPFinder); ok {
			ipFinder = finder
		} else if runtime.GOOS == "windows" {
			ipFinder = new(VMnetNatConfIPFinder)
		} else {
			ipFinder = &IfconfigIPFinder{Device: "vmnet8"}
		}

		if vncIp, err := ipFinder.HostIP(); err == nil {
			s.VNCBindAddress = vncIp
			log.Printf("VNC ip detected: %d", s.VNCBindAddress)
			vmxData["remotedisplay.vnc.ip"] = fmt.Sprintf("%s", s.VNCBindAddress)
			state.Put("vnc_ip", s.VNCBindAddress)
		} else {
			err := fmt.Errorf("Error detecting host IP: %s", err)
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	}

	if vncPortString, ok := vmxData["remotedisplay.vnc.port"]; ok {

		if vncPortInt, err := strconv.Atoi(vncPortString); err == nil {
			vncPort := uint(vncPortInt)

			log.Printf("VNC port specified: %d", vncPort)

			state.Put("vnc_port", vncPort)

		} else {
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}
	} else {
		var vncFinder VNCAddressFinder
		if finder, ok := driver.(VNCAddressFinder); ok {
			vncFinder = finder
		} else {
			vncFinder = s
		}
		log.Printf("Looking for available port between %d and %d", s.VNCPortMin, s.VNCPortMax)
		_, vncPort, err := vncFinder.VNCAddress(s.VNCBindAddress, s.VNCPortMin, s.VNCPortMax)
		if err != nil {
			state.Put("error", err)
			ui.Error(err.Error())
			return multistep.ActionHalt
		}

		log.Printf("Found available VNC port: %d", vncPort)

		vmxData["remotedisplay.vnc.port"] = fmt.Sprintf("%d", vncPort)
		state.Put("vnc_port", vncPort)
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
