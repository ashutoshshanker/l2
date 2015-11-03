// lacp tests
// go test
// go test -coverageprofile lacpcov.out
// go tool cover -html=lacpcov.out
package lacp

import (
	"fmt"
	"testing"
	"time"
	"utils/fsm"
)

func InvalidStateCheck(p *LaAggPort, invalidStates []fsm.Event, prevState fsm.State, currState fsm.State) (string, bool) {

	var s string
	rc := true

	portchan := p.PortChannelGet()

	// force what state transition should have been
	p.RxMachineFsm.Machine.Curr.SetState(prevState)
	p.RxMachineFsm.Machine.Curr.SetState(currState)

	for _, e := range invalidStates {
		// send PORT MOVED event to Rx Machine
		p.RxMachineFsm.RxmEvents <- LacpMachineEvent{
			e:            e,
			responseChan: portchan,
			src:          "TEST"}

		// wait for response
		if msg := <-portchan; msg != RxMachineModuleStr {
			s = fmt.Sprintf("Expected response from", RxMachineModuleStr)
			rc = false
			return s, rc
		}

		// PORT MOVED
		if p.RxMachineFsm.Machine.Curr.PreviousState() != prevState &&
			p.RxMachineFsm.Machine.Curr.CurrentState() != currState {
			s = fmt.Sprintf("ERROR RX Machine state incorrect expected (prev/curr)",
				prevState,
				currState,
				"actual",
				p.RxMachineFsm.Machine.Curr.PreviousState(),
				p.RxMachineFsm.Machine.Curr.CurrentState())
			rc = false
			return s, rc
		}
	}

	return "", rc
}

func TestLaAggPortCreateAndBeginEvent(t *testing.T) {

	var p *LaAggPort

	// must be called to initialize the global
	sysId := [6]uint8{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}
	LacpSysGlobalInfoInit(sysId)

	pconf := &LaAggPortConfig{
		Id:     1,
		Prio:   0x80,
		Key:    100,
		AggId:  2000,
		Enable: false,
		Mode:   LacpModeActive,
		Properties: PortProperties{
			Mac:    [6]uint8{0x00, 0x01, 0xDE, 0xAD, 0xBE, 0xEF},
			speed:  1000000000,
			duplex: LacpPortDuplexFull,
			mtu:    1500,
		},
		IntfId:   "eth1.1",
		traceEna: false,
		sysId:    sysId,
	}

	// lets create a port and start the machines
	CreateLaAggPort(pconf)

	// if the port is found verify the initial state after begin event
	// which was called as part of create
	if LaFindPortById(pconf.Id, &p) {

		//	fmt.Println("Rx:", p.RxMachineFsm.Machine.Curr.CurrentState(),
		//		"Ptx:", p.PtxMachineFsm.Machine.Curr.CurrentState(),
		//		"Cd:", p.CdMachineFsm.Machine.Curr.CurrentState(),
		//		"Mux:", p.MuxMachineFsm.Machine.Curr.CurrentState(),
		//		"Tx:", p.TxMachineFsm.Machine.Curr.CurrentState())

		// lets test the states, after initialization port moves to Disabled State
		// Rx Machine
		if p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStatePortDisabled {
			t.Error("ERROR RX Machine state incorrect expected",
				LacpRxmStatePortDisabled, "actual",
				p.RxMachineFsm.Machine.Curr.CurrentState())
		}
		// Periodic Tx Machine
		if p.PtxMachineFsm.Machine.Curr.CurrentState() != LacpPtxmStateNoPeriodic {
			t.Error("ERROR PTX Machine state incorrect expected",
				LacpPtxmStateNoPeriodic, "actual",
				p.PtxMachineFsm.Machine.Curr.CurrentState())
		}
		// Churn Detection Machine
		if p.CdMachineFsm.Machine.Curr.CurrentState() != LacpCdmStateActorChurnMonitor {
			t.Error("ERROR CD Machine state incorrect expected",
				LacpCdmStateActorChurnMonitor, "actual",
				p.CdMachineFsm.Machine.Curr.CurrentState())
		}
		// Mux Machine
		if p.MuxMachineFsm.Machine.Curr.CurrentState() != LacpMuxmStateDetached {
			t.Error("ERROR MUX Machine state incorrect expected",
				LacpMuxmStateDetached, "actual",
				p.MuxMachineFsm.Machine.Curr.CurrentState())
		}
		// Tx Machine
		if p.TxMachineFsm.Machine.Curr.CurrentState() != LacpTxmStateOff {
			t.Error("ERROR TX Machine state incorrect expected",
				LacpTxmStateOff, "actual",
				p.TxMachineFsm.Machine.Curr.CurrentState())
		}
	}
	DeleteLaAggPort(pconf.Id)
}

func TestLaAggPortCreateWithInvalidKeySetWithAgg(t *testing.T) {
	var p *LaAggPort

	// must be called to initialize the global
	sysId := [6]uint8{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}
	LacpSysGlobalInfoInit(sysId)

	aconf := &LaAggConfig{
		mac:   [6]uint8{0x00, 0x00, 0x01, 0x02, 0x03, 0x04},
		Id:    2000,
		Key:   50,
		sysId: sysId,
	}

	// Create Aggregation
	CreateLaAgg(aconf)

	pconf := &LaAggPortConfig{
		Id:     2,
		Prio:   0x80,
		Key:    100, // INVALID
		AggId:  2000,
		Enable: true,
		Mode:   LacpModeActive,
		Properties: PortProperties{
			Mac:    [6]uint8{0x00, 0x02, 0xDE, 0xAD, 0xBE, 0xEF},
			speed:  1000000000,
			duplex: LacpPortDuplexFull,
			mtu:    1500,
		},
		IntfId:   "eth1.1",
		traceEna: false,
		sysId:    sysId,
	}

	// lets create a port and start the machines
	CreateLaAggPort(pconf)

	// if the port is found verify the initial state after begin event
	// which was called as part of create
	if LaFindPortById(pconf.Id, &p) {
		if p.aggSelected == LacpAggSelected {
			t.Error("Port is in SELECTED mode")
		}
	}

	// Delete the port and agg
	DeleteLaAggPort(pconf.Id)
	DeleteLaAgg(aconf.Id)
}

func TestLaAggPortCreateWithoutKeySetNoAgg(t *testing.T) {

	var p *LaAggPort
	sysId := [6]uint8{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}
	LacpSysGlobalInfoInit(sysId)

	pconf := &LaAggPortConfig{
		Id:     3,
		Prio:   0x80,
		Key:    100,
		AggId:  2000,
		Enable: true,
		Mode:   LacpModeActive,
		Properties: PortProperties{
			Mac:    [6]uint8{0x00, 0x01, 0xDE, 0xAD, 0xBE, 0xEF},
			speed:  1000000000,
			duplex: LacpPortDuplexFull,
			mtu:    1500,
		},
		IntfId:   "eth1.1",
		traceEna: false,
		sysId:    sysId,
	}

	// lets create a port and start the machines
	CreateLaAggPort(pconf)

	// if the port is found verify the initial state after begin event
	// which was called as part of create
	if LaFindPortById(pconf.Id, &p) {
		if p.aggSelected == LacpAggSelected {
			t.Error("Port is in SELECTED mode")
		}
	}

	// Delete port
	DeleteLaAggPort(pconf.Id)
}

func TestLaAggPortCreateThenCorrectAggCreate(t *testing.T) {

	var p *LaAggPort
	sysId := [6]uint8{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}
	LacpSysGlobalInfoInit(sysId)

	pconf := &LaAggPortConfig{
		Id:     3,
		Prio:   0x80,
		Key:    100,
		AggId:  2000,
		Enable: true,
		Mode:   LacpModeActive,
		Properties: PortProperties{
			Mac:    [6]uint8{0x00, 0x01, 0xDE, 0xAD, 0xBE, 0xEF},
			speed:  1000000000,
			duplex: LacpPortDuplexFull,
			mtu:    1500,
		},
		IntfId:   "eth1.1",
		traceEna: false,
		sysId:    sysId,
	}

	// lets create a port and start the machines
	CreateLaAggPort(pconf)

	// if the port is found verify the initial state after begin event
	// which was called as part of create
	if LaFindPortById(pconf.Id, &p) {
		if p.aggSelected == LacpAggSelected {
			t.Error("Port is in SELECTED mode")
		}
	}

	aconf := &LaAggConfig{
		mac:   [6]uint8{0x00, 0x00, 0x01, 0x02, 0x03, 0x04},
		Id:    2000,
		Key:   100,
		sysId: sysId,
	}

	// Create Aggregation
	CreateLaAgg(aconf)

	// if the port is found verify the initial state after begin event
	// which was called as part of create
	if p.aggSelected == LacpAggSelected {
		t.Error("Port is in SELECTED mode")
	}

	// Add port to agg
	AddLaAggPortToAgg(aconf.Id, pconf.Id)

	if p.aggSelected != LacpAggSelected {
		t.Error("Port is in NOT in SELECTED mode")
	}

	if p.MuxMachineFsm.Machine.Curr.CurrentState() != LacpMuxmStateAttached {
		t.Error("Mux state expected", LacpMuxmStateAttached, "actual", p.MuxMachineFsm.Machine.Curr.CurrentState())
	}

	// TODO Check states of other state machines

	// Delete agg
	DeleteLaAgg(aconf.Id)
}

// TestLaAggPortCreateThenCorrectAggCreateThenDetach:
// - create port
// - create lag
// - attach port
// - enable port
func TestLaAggPortCreateThenCorrectAggCreateThenDetach(t *testing.T) {

	var p *LaAggPort
	sysId := [6]uint8{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}
	LacpSysGlobalInfoInit(sysId)

	pconf := &LaAggPortConfig{
		Id:    3,
		Prio:  0x80,
		Key:   100,
		AggId: 2000,
		Mode:  LacpModeActive,
		Properties: PortProperties{
			Mac:    [6]uint8{0x00, 0x01, 0xDE, 0xAD, 0xBE, 0xEF},
			speed:  1000000000,
			duplex: LacpPortDuplexFull,
			mtu:    1500,
		},
		IntfId:   "eth1.1",
		traceEna: false,
		sysId:    sysId,
	}

	// lets create a port and start the machines
	CreateLaAggPort(pconf)

	// if the port is found verify the initial state after begin event
	// which was called as part of create
	if LaFindPortById(pconf.Id, &p) {
		if p.aggSelected == LacpAggSelected {
			t.Error("Port is in SELECTED mode")
		}
	}

	aconf := &LaAggConfig{
		mac:   [6]uint8{0x00, 0x00, 0x01, 0x02, 0x03, 0x04},
		Id:    2000,
		Key:   100,
		sysId: sysId,
	}

	// Create Aggregation
	CreateLaAgg(aconf)

	// if the port is found verify the initial state after begin event
	// which was called as part of create
	if p.aggSelected == LacpAggSelected {
		t.Error("Port is in SELECTED mode")
	}

	// Add port to agg
	AddLaAggPortToAgg(aconf.Id, pconf.Id)

	if p.aggSelected == LacpAggSelected {
		t.Error("Port is in SELECTED mode")
	}

	EnableLaAggPort(pconf.Id)

	if p.aggSelected != LacpAggSelected {
		t.Error("Port is in NOT in SELECTED mode")
	}

	if p.MuxMachineFsm.Machine.Curr.CurrentState() != LacpMuxmStateAttached {
		t.Error("Mux state expected", LacpMuxmStateAttached, "actual", p.MuxMachineFsm.Machine.Curr.CurrentState())
	}
	// Delete port
	DeleteLaAggPortFromAgg(pconf.AggId, pconf.Id)
	DeleteLaAggPort(pconf.Id)
}

// Enable port post creation
func TestLaAggPortEnable(t *testing.T) {
	var p *LaAggPort
	sysId := [6]uint8{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}
	LacpSysGlobalInfoInit(sysId)

	pconf := &LaAggPortConfig{
		Id:    3,
		Prio:  0x80,
		Key:   100,
		AggId: 2000,
		Mode:  LacpModeActive,
		Properties: PortProperties{
			Mac:    [6]uint8{0x00, 0x01, 0xDE, 0xAD, 0xBE, 0xEF},
			speed:  1000000000,
			duplex: LacpPortDuplexFull,
			mtu:    1500,
		},
		IntfId:   "eth1.1",
		traceEna: false,
		sysId:    sysId,
	}

	// lets create a port and start the machines
	CreateLaAggPort(pconf)

	// if the port is found verify the initial state after begin event
	// which was called as part of create
	if LaFindPortById(pconf.Id, &p) {
		if p.aggSelected == LacpAggSelected {
			t.Error("Port is in SELECTED mode")
		}
	}

	aconf := &LaAggConfig{
		mac:   [6]uint8{0x00, 0x00, 0x01, 0x02, 0x03, 0x04},
		Id:    2000,
		Key:   100,
		sysId: sysId,
	}

	// Create Aggregation
	CreateLaAgg(aconf)

	// if the port is found verify the initial state after begin event
	// which was called as part of create
	if p.aggSelected == LacpAggSelected {
		t.Error("Port is in SELECTED mode")
	}

	// Add port to agg
	AddLaAggPortToAgg(aconf.Id, pconf.Id)

	if p.aggSelected == LacpAggSelected {
		t.Error("Port is in SELECTED mode")
	}

	EnableLaAggPort(pconf.Id)

	if p.aggSelected != LacpAggSelected {
		t.Error("Port is in NOT in SELECTED mode")
	}

	if p.MuxMachineFsm.Machine.Curr.CurrentState() != LacpMuxmStateAttached {
		t.Error("Mux state expected", LacpMuxmStateAttached, "actual", p.MuxMachineFsm.Machine.Curr.CurrentState())
	}
	// Delete port
	DeleteLaAggPortFromAgg(pconf.AggId, pconf.Id)
	DeleteLaAggPort(pconf.Id)

}

func TestLaAggPortRxMachineStateTransitions(t *testing.T) {

	var msg string
	var portchan chan string
	sysId := [6]uint8{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}
	LacpSysGlobalInfoInit(sysId)

	pconf := &LaAggPortConfig{
		Id:     1,
		Prio:   0x80,
		IntfId: "eth1.1",
		Key:    100,
		sysId:  sysId,
	}

	// not calling Create because we don't want to launch all state machines
	p := NewLaAggPort(pconf)

	// lets start the Rx Machine only
	p.LacpRxMachineMain()

	// Rx Machine
	if p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStateNone {
		t.Error("ERROR RX Machine state incorrect expected",
			LacpRxmStateNone, "actual",
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	portchan = p.PortChannelGet()
	// send event to Rx Machine
	p.RxMachineFsm.RxmEvents <- LacpMachineEvent{
		e:            LacpRxmEventBegin,
		responseChan: portchan,
		src:          "TEST"}

	// wait for response
	msg = <-portchan
	if msg != RxMachineModuleStr {
		t.Error("Expected response from", RxMachineModuleStr)
	}

	// port is initally disabled and lacp is disabled
	if p.RxMachineFsm.Machine.Curr.PreviousState() != LacpRxmStateInitialize &&
		p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStatePortDisabled {
		t.Error("ERROR RX Machine state incorrect expected (prev/curr)",
			LacpRxmStateInitialize,
			LacpRxmStatePortDisabled,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	// check state info
	if p.aggSelected != LacpAggUnSelected {
		t.Error("expected UNSELECTED", LacpAggUnSelected, "actual", p.aggSelected)
	}
	if LacpStateIsSet(p.actorOper.state, LacpStateExpiredBit) {
		t.Error("expected state Expired to be cleared")
	}
	if p.portMoved != false {
		t.Error("expected port moved to be false")
	}
	// TODO check actor oper state

	p.portMoved = true
	// send PORT MOVED event to Rx Machine
	p.RxMachineFsm.RxmEvents <- LacpMachineEvent{
		e:            LacpRxmEventPortMoved,
		responseChan: portchan,
		src:          "TEST"}

	// wait for response
	msg = <-portchan
	if msg != RxMachineModuleStr {
		t.Error("Expected response from", RxMachineModuleStr)
	}

	// PORT MOVED
	if p.RxMachineFsm.Machine.Curr.PreviousState() != LacpRxmStateInitialize &&
		p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStatePortDisabled {
		t.Error("ERROR RX Machine state incorrect expected (prev/curr)",
			LacpRxmStateInitialize,
			LacpRxmStatePortDisabled,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	p.aggSelected = LacpAggSelected
	p.portMoved = false
	p.portEnabled = true
	p.lacpEnabled = false
	// send PORT ENABLED && LACP DISABLED event to Rx Machine
	p.RxMachineFsm.RxmEvents <- LacpMachineEvent{
		e:            LacpRxmEventPortEnabledAndLacpDisabled,
		responseChan: portchan,
		src:          "TEST"}

	// wait for response
	msg = <-portchan
	if msg != RxMachineModuleStr {
		t.Error("Expected response from", RxMachineModuleStr)
	}

	// port is initally disabled and lacp is disabled
	if p.RxMachineFsm.Machine.Curr.PreviousState() != LacpRxmStatePortDisabled &&
		p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStateLacpDisabled {
		t.Error("ERROR RX Machine state incorrect expected (prev/curr)",
			LacpRxmStatePortDisabled,
			LacpRxmStateLacpDisabled,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	p.lacpEnabled = true
	// send LACP ENABLED event to Rx Machine
	p.RxMachineFsm.RxmEvents <- LacpMachineEvent{
		e:            LacpRxmEventLacpEnabled,
		responseChan: portchan,
		src:          "TEST"}

	// wait for response
	msg = <-portchan
	if msg != RxMachineModuleStr {
		t.Error("Expected response from", RxMachineModuleStr)
	}

	// port was lacp disabled, but then transitioned to port disabled
	// then expired
	if p.RxMachineFsm.Machine.Curr.PreviousState() != LacpRxmStatePortDisabled &&
		p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStateExpired {
		t.Error("ERROR RX Machine state incorrect expected (prev/curr)",
			LacpRxmStatePortDisabled,
			LacpRxmStateExpired,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	if LacpStateIsSet(p.partnerOper.state, LacpStateSyncBit) {
		t.Error("Expected partner Sync Bit to not be set")
	}
	if !LacpStateIsSet(p.partnerOper.state, LacpStateTimeoutBit) {
		t.Error("Expected partner Timeout bit to be set since we are in short timeout")
	}
	if p.RxMachineFsm.currentWhileTimerTimeout != LacpShortTimeoutTime {
		t.Error("Expected timer to be set to short timeout")
	}
	if !LacpStateIsSet(p.actorOper.state, LacpStateExpiredBit) {
		t.Error("Expected actor expired bit to be set")
	}

	p.portEnabled = false
	// send NOT ENABLED AND NOT MOVED event to Rx Machine from Expired State
	p.RxMachineFsm.RxmEvents <- LacpMachineEvent{
		e:            LacpRxmEventNotPortEnabledAndNotPortMoved,
		responseChan: portchan,
		src:          "TEST"}

	// wait for response
	msg = <-portchan
	if msg != RxMachineModuleStr {
		t.Error("Expected response from", RxMachineModuleStr)
	}

	// port was enabled and lacp is disabled
	if p.RxMachineFsm.Machine.Curr.PreviousState() != LacpRxmStateExpired &&
		p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStatePortDisabled {
		t.Error("ERROR RX Machine state incorrect expected (prev/curr)",
			LacpRxmStateExpired,
			LacpRxmStatePortDisabled,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	if LacpStateIsSet(p.partnerOper.state, LacpStateSyncBit) {
		t.Error("Expected partner Sync Bit to not be set")
	}

	p.portEnabled = true
	p.lacpEnabled = false
	// send NOT ENABLED AND NOT MOVED event to Rx Machine from Expired State
	p.RxMachineFsm.RxmEvents <- LacpMachineEvent{
		e:            LacpRxmEventPortEnabledAndLacpDisabled,
		responseChan: portchan,
		src:          "TEST"}

	// wait for response
	msg = <-portchan
	if msg != RxMachineModuleStr {
		t.Error("Expected response from", RxMachineModuleStr)
	}

	p.portEnabled = false
	// send NOT ENABLED AND NOT MOVED event to Rx Machine from LACP DISABLED
	p.RxMachineFsm.RxmEvents <- LacpMachineEvent{
		e:            LacpRxmEventNotPortEnabledAndNotPortMoved,
		responseChan: portchan,
		src:          "TEST"}

	// wait for response
	msg = <-portchan
	if msg != RxMachineModuleStr {
		t.Error("Expected response from", RxMachineModuleStr)
	}

	// port was enabled and lacp is disabled
	if p.RxMachineFsm.Machine.Curr.PreviousState() != LacpRxmStateLacpDisabled &&
		p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStatePortDisabled {
		t.Error("ERROR RX Machine state incorrect expected (prev/curr)",
			LacpRxmStateLacpDisabled,
			LacpRxmStatePortDisabled,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	p.portEnabled = true
	p.lacpEnabled = true
	// send PORT ENABLE LACP ENABLED event to Rx Machine
	p.RxMachineFsm.RxmEvents <- LacpMachineEvent{
		e:            LacpRxmEventPortEnabledAndLacpEnabled,
		responseChan: portchan,
		src:          "TEST"}

	// wait for response
	msg = <-portchan
	if msg != RxMachineModuleStr {
		t.Error("Expected response from", RxMachineModuleStr)
	}

	// send CURRENT WHILE TIMER event to Rx Machine
	p.RxMachineFsm.RxmEvents <- LacpMachineEvent{
		e:            LacpRxmEventCurrentWhileTimerExpired,
		responseChan: portchan,
		src:          "TEST"}

	// wait for response
	msg = <-portchan
	if msg != RxMachineModuleStr {
		t.Error("Expected response from", RxMachineModuleStr)
	}

	// port was enabled and lacp is disabled
	if p.RxMachineFsm.Machine.Curr.PreviousState() != LacpRxmStateExpired &&
		p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStateDefaulted {
		t.Error("ERROR RX Machine state incorrect expected (prev/curr)",
			LacpRxmStateExpired,
			LacpRxmStateDefaulted,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	// TODO check default selected, record default, expired == false

	// LETS GET THE STATE BACK TO EXPIRED

	p.portEnabled = false
	// send NOT PORT ENABLE NOT PORT MOVED event to Rx Machine
	p.RxMachineFsm.RxmEvents <- LacpMachineEvent{
		e:            LacpRxmEventNotPortEnabledAndNotPortMoved,
		responseChan: portchan,
		src:          "TEST"}

	// wait for response
	msg = <-portchan
	if msg != RxMachineModuleStr {
		t.Error("Expected response from", RxMachineModuleStr)
	}

	p.portEnabled = true
	// send PORT ENABLE LACP ENABLED event to Rx Machine
	p.RxMachineFsm.RxmEvents <- LacpMachineEvent{
		e:            LacpRxmEventPortEnabledAndLacpEnabled,
		responseChan: portchan,
		src:          "TEST"}

	// wait for response
	msg = <-portchan
	if msg != RxMachineModuleStr {
		t.Error("Expected response from", RxMachineModuleStr)
	}

	// lets adjust the actorOper timeout state
	// TODO Assume a method was called to adjust this
	LacpStateSet(&p.actorAdmin.state, LacpStateTimeoutBit)
	LacpStateSet(&p.actorOper.state, LacpStateTimeoutBit)

	// send valid pdu
	lacppdu := &LacpPdu{
		subType: LacpSubType,
		version: 1,
		actor: LacpPduInfoTlv{tlv_type: 1,
			len: 0x14,
			info: LacpPortInfo{
				system: LacpSystem{actor_system: [6]uint8{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
					actor_system_priority: 1},
				key:      100,
				port_pri: 0x80,
				port:     10,
				state:    LacpStateActivityBit | LacpStateAggregationBit | LacpStateTimeoutBit},
		},
		partner: LacpPduInfoTlv{tlv_type: 1,
			len: 0x14,
			info: LacpPortInfo{
				system: LacpSystem{actor_system: p.actorOper.system.actor_system,
					actor_system_priority: p.actorOper.system.actor_system_priority},
				key:      p.key,
				port_pri: p.portPriority,
				port:     p.portNum,
				state:    p.actorOper.state},
		},
	}

	rx := LacpRxLacpPdu{
		pdu:          lacppdu,
		responseChan: portchan,
		src:          "TEST"}
	p.RxMachineFsm.RxmPktRxEvent <- rx

	// wait for response
	msg = <-portchan
	if msg != RxMachineModuleStr {
		t.Error("Expected response from", RxMachineModuleStr)
	}

	// port was enabled and lacp is disabled
	if p.RxMachineFsm.Machine.Curr.PreviousState() != LacpRxmStateExpired &&
		p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStateCurrent {
		t.Error("ERROR RX Machine state incorrect expected (prev/curr)",
			LacpRxmStateExpired,
			LacpRxmStateCurrent,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	// allow for current while timer to expire
	time.Sleep(time.Second * 4)

	// port was enabled and lacp is disabled
	if p.RxMachineFsm.Machine.Curr.PreviousState() != LacpRxmStateCurrent &&
		p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStateExpired {
		t.Error("ERROR RX Machine state incorrect expected (prev/curr)",
			LacpRxmStateCurrent,
			LacpRxmStateExpired,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	// allow for current while timer to expire
	time.Sleep(time.Second * 4)

	// port was enabled and lacp is disabled
	if p.RxMachineFsm.Machine.Curr.PreviousState() != LacpRxmStateExpired &&
		p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStateDefaulted {
		t.Error("ERROR RX Machine state incorrect expected (prev/curr)",
			LacpRxmStateExpired,
			LacpRxmStateDefaulted,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	// send valid pdu
	lacppdu = &LacpPdu{
		subType: LacpSubType,
		version: 1,
		actor: LacpPduInfoTlv{tlv_type: 1,
			len: 0x14,
			info: LacpPortInfo{
				system: LacpSystem{actor_system: [6]uint8{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
					actor_system_priority: 1},
				key:      100,
				port_pri: 0x80,
				port:     10,
				state:    LacpStateActivityBit | LacpStateAggregationBit | LacpStateTimeoutBit},
		},
		partner: LacpPduInfoTlv{tlv_type: 1,
			len: 0x14,
			info: LacpPortInfo{
				system: LacpSystem{actor_system: p.actorOper.system.actor_system,
					actor_system_priority: p.actorOper.system.actor_system_priority},
				key:      p.key,
				port_pri: p.portPriority,
				port:     p.portNum,
				state:    p.actorOper.state},
		},
	}

	rx = LacpRxLacpPdu{
		pdu:          lacppdu,
		responseChan: portchan,
		src:          "TEST"}
	p.RxMachineFsm.RxmPktRxEvent <- rx

	// wait for response
	msg = <-portchan
	if msg != RxMachineModuleStr {
		t.Error("Expected response from", RxMachineModuleStr)
	}

	// port was enabled and lacp is disabled
	if p.RxMachineFsm.Machine.Curr.PreviousState() != LacpRxmStateDefaulted &&
		p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStateCurrent {
		t.Error("ERROR RX Machine state incorrect expected (prev/curr)",
			LacpRxmStateExpired,
			LacpRxmStateCurrent,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}

	// send valid pdu
	lacppdu = &LacpPdu{
		subType: LacpSubType,
		version: 1,
		actor: LacpPduInfoTlv{tlv_type: 1,
			len: 0x14,
			info: LacpPortInfo{
				system: LacpSystem{actor_system: [6]uint8{0x01, 0x02, 0x03, 0x04, 0x05, 0x06},
					actor_system_priority: 1},
				key:      100,
				port_pri: 0x80,
				port:     10,
				state:    LacpStateActivityBit | LacpStateAggregationBit | LacpStateTimeoutBit},
		},
		partner: LacpPduInfoTlv{tlv_type: 1,
			len: 0x14,
			info: LacpPortInfo{
				system: LacpSystem{actor_system: p.actorOper.system.actor_system,
					actor_system_priority: p.actorOper.system.actor_system_priority},
				key:      p.key,
				port_pri: p.portPriority,
				port:     p.portNum,
				state:    p.actorOper.state},
		},
	}

	rx = LacpRxLacpPdu{
		pdu:          lacppdu,
		responseChan: portchan,
		src:          "TEST"}
	p.RxMachineFsm.RxmPktRxEvent <- rx

	// wait for response
	msg = <-portchan
	if msg != RxMachineModuleStr {
		t.Error("Expected response from", RxMachineModuleStr)
	}

	// port was enabled and lacp is disabled
	if p.RxMachineFsm.Machine.Curr.PreviousState() != LacpRxmStateCurrent &&
		p.RxMachineFsm.Machine.Curr.CurrentState() != LacpRxmStateCurrent {
		t.Error("ERROR RX Machine state incorrect expected (prev/curr)",
			LacpRxmStateExpired,
			LacpRxmStateCurrent,
			"actual",
			p.RxMachineFsm.Machine.Curr.PreviousState(),
			p.RxMachineFsm.Machine.Curr.CurrentState())
	}
	p.DelLaAggPort()
}

func TestLaAggPortRxMachineInvalidStateTransitions(t *testing.T) {

	// must be called to initialize the global
	sysId := [6]uint8{0x00, 0x01, 0x02, 0x03, 0x04, 0x05}
	LacpSysGlobalInfoInit(sysId)

	pconf := &LaAggPortConfig{
		Id:     1,
		Prio:   0x80,
		IntfId: "eth1.1",
		Key:    100,
		sysId:  sysId,
	}

	// not calling Create because we don't want to launch all state machines
	p := NewLaAggPort(pconf)

	// lets start the Rx Machine only
	p.LacpRxMachineMain()

	p.BEGIN(false)

	// turn timer off so that we do not accidentally transition states
	p.RxMachineFsm.CurrentWhileTimerStop()
	/*
		LacpRxmEventBegin = iota + 1
		LacpRxmEventUnconditionalFallthrough
		LacpRxmEventNotPortEnabledAndNotPortMoved
		LacpRxmEventPortMoved
		LacpRxmEventPortEnabledAndLacpEnabled
		LacpRxmEventPortEnabledAndLacpDisabled
		LacpRxmEventCurrentWhileTimerExpired
		LacpRxmEventLacpEnabled
		LacpRxmEventLacpPktRx
		LacpRxmEventKillSignal
	*/

	// BEGIN -> INITIALIZE automatically falls through to PORT_DISABLED so no
	// need to tests

	// PORT_DISABLED
	portDisableInvalidStates := [4]fsm.Event{LacpRxmEventUnconditionalFallthrough,
		LacpRxmEventCurrentWhileTimerExpired,
		LacpRxmEventLacpEnabled,
		LacpRxmEventLacpPktRx}

	str, ok := InvalidStateCheck(p, portDisableInvalidStates[:], LacpRxmStateInitialize, LacpRxmStatePortDisabled)
	if !ok {
		t.Error(str)
	}

	// EXPIRED - note disabling current while timer so state does not change
	expiredInvalidStates := [5]fsm.Event{LacpRxmEventUnconditionalFallthrough,
		LacpRxmEventPortMoved,
		LacpRxmEventPortEnabledAndLacpEnabled,
		LacpRxmEventPortEnabledAndLacpDisabled,
		LacpRxmEventLacpEnabled}

	str, ok = InvalidStateCheck(p, expiredInvalidStates[:], LacpRxmStatePortDisabled, LacpRxmStateExpired)
	if !ok {
		t.Error(str)
	}

	// LACP_DISABLED
	lacpDisabledInvalidStates := [6]fsm.Event{LacpRxmEventUnconditionalFallthrough,
		LacpRxmEventPortMoved,
		LacpRxmEventPortEnabledAndLacpEnabled,
		LacpRxmEventPortEnabledAndLacpDisabled,
		LacpRxmEventCurrentWhileTimerExpired,
		LacpRxmEventLacpPktRx}

	str, ok = InvalidStateCheck(p, lacpDisabledInvalidStates[:], LacpRxmStatePortDisabled, LacpRxmStateLacpDisabled)
	if !ok {
		t.Error(str)
	}

	// DEFAULTED
	defaultedInvalidStates := [6]fsm.Event{LacpRxmEventUnconditionalFallthrough,
		LacpRxmEventPortMoved,
		LacpRxmEventPortEnabledAndLacpEnabled,
		LacpRxmEventPortEnabledAndLacpDisabled,
		LacpRxmEventCurrentWhileTimerExpired,
		LacpRxmEventLacpEnabled}

	str, ok = InvalidStateCheck(p, defaultedInvalidStates[:], LacpRxmStateExpired, LacpRxmStateDefaulted)
	if !ok {
		t.Error(str)
	}

	// DEFAULTED
	currentInvalidStates := [5]fsm.Event{LacpRxmEventUnconditionalFallthrough,
		LacpRxmEventPortMoved,
		LacpRxmEventPortEnabledAndLacpEnabled,
		LacpRxmEventPortEnabledAndLacpDisabled,
		LacpRxmEventLacpEnabled}

	str, ok = InvalidStateCheck(p, currentInvalidStates[:], LacpRxmStateExpired, LacpRxmStateCurrent)
	if !ok {
		t.Error(str)
	}

	p.DelLaAggPort()
}

func TestTwoAggsBackToBackSinglePort(t *testing.T) {

	const LaAggPortActor = 10
	const LaAggPortPeer = 20
	LaSystemActor := [6]uint8{0x00, 0x00, 0x00, 0x00, 0x00, 0x64}
	LaSystemPeer := [6]uint8{0x00, 0x00, 0x00, 0x00, 0x00, 0xC8}

	bridge := SimulationBridge{
		port1:       LaAggPortActor,
		port2:       LaAggPortPeer,
		rxLacpPort1: make(chan RxPacket),
		rxLacpPort2: make(chan RxPacket),
	}

	ActorSystem := LacpSysGlobalInfoInit(LaSystemActor)
	PeerSystem := LacpSysGlobalInfoInit(LaSystemPeer)
	ActorSystem.LaSysGlobalRegisterTxCallback(LaAggPortActor, bridge.TxViaGoChannel)
	PeerSystem.LaSysGlobalRegisterTxCallback(LaAggPortPeer, bridge.TxViaGoChannel)

	// port 1
	go LaRxMain(bridge.rxLacpPort1)
	// port 2
	go LaRxMain(bridge.rxLacpPort2)

	p1conf := &LaAggPortConfig{
		Id:     LaAggPortActor,
		Prio:   0x80,
		Key:    100,
		AggId:  100,
		Enable: true,
		Mode:   LacpModeActive,
		//Timeout: LacpFastPeriodicTime,
		Properties: PortProperties{
			Mac:    [6]uint8{0x00, LaAggPortActor, 0xDE, 0xAD, 0xBE, 0xEF},
			speed:  1000000000,
			duplex: LacpPortDuplexFull,
			mtu:    1500,
		},
		IntfId:   "eth1.1",
		traceEna: true,
		sysId:    LaSystemActor,
	}

	p2conf := &LaAggPortConfig{
		Id:     LaAggPortPeer,
		Prio:   0x80,
		Key:    200,
		AggId:  200,
		Enable: true,
		Mode:   LacpModeActive,
		Properties: PortProperties{
			Mac:    [6]uint8{0x00, LaAggPortPeer, 0xDE, 0xAD, 0xBE, 0xEF},
			speed:  1000000000,
			duplex: LacpPortDuplexFull,
			mtu:    1500,
		},
		IntfId:   "eth1.2",
		traceEna: false,
		sysId:    LaSystemPeer,
	}

	// lets create a port and start the machines
	CreateLaAggPort(p1conf)
	CreateLaAggPort(p2conf)

	a1conf := &LaAggConfig{
		mac:   [6]uint8{0x00, 0x00, 0x01, 0x01, 0x01, 0x01},
		Id:    100,
		Key:   100,
		sysId: LaSystemActor,
	}

	a2conf := &LaAggConfig{
		mac:   [6]uint8{0x00, 0x00, 0x02, 0x02, 0x02, 0x02},
		Id:    200,
		Key:   200,
		sysId: LaSystemPeer,
	}

	// Create Aggregation
	CreateLaAgg(a1conf)
	CreateLaAgg(a2conf)

	// Add port to agg
	AddLaAggPortToAgg(a1conf.Id, p1conf.Id)
	AddLaAggPortToAgg(a2conf.Id, p2conf.Id)

	//time.Sleep(time.Second * 30)
	stopTest := make(chan bool)

	var p1 *LaAggPort
	var p2 *LaAggPort
	if LaFindPortById(p1conf.Id, &p1) &&
		LaFindPortById(p2conf.Id, &p2) {

		go func() {
			for i := 0; i < 10 &&
				(p1.MuxMachineFsm.Machine.Curr.CurrentState() != LacpMuxmStateDistributing ||
					p2.MuxMachineFsm.Machine.Curr.CurrentState() != LacpMuxmStateDistributing); i++ {
				time.Sleep(time.Second * 1)
			}
			stopTest <- true
		}()

		<-stopTest
		close(stopTest)

		state1 := GetLaAggPortActorOperState(p1conf.Id)
		state2 := GetLaAggPortActorOperState(p2conf.Id)

		const portUpState = LacpStateActivityBit | LacpStateAggregationBit |
			LacpStateSyncBit | LacpStateCollectingBit | LacpStateDistributingBit

		if !LacpStateIsSet(state1, portUpState) {
			t.Error(fmt.Sprintf("Actor Port state 0x%x did not come up properly with peer expected 0x%x", state1, portUpState))
		}
		if !LacpStateIsSet(state2, portUpState) {
			t.Error(fmt.Sprintf("Peer Port state 0x%x did not come up properly with actor expected 0x%x", state2, portUpState))
		}
	} else {
		t.Error("Unable to find port just created")
	}
}

//
func TestLaAggPortPeriodicTxMachineStateTransitions(t *testing.T) {

}

func TestLaAggPortPeriodicTxMachineInvalidStateTransitions(t *testing.T) {

}

func TestLaAggPortMuxMachineStateTransitions(t *testing.T) {

}

func TestLaAggPortMuxMachineInvalidStateTransitions(t *testing.T) {

}

func TestLaAggPortChurnDetectionMachineStateTransitions(t *testing.T) {

}

func TestLaAggPortChurnDetectionMachineInvalidStateTransitions(t *testing.T) {

}

func TestLaAggPortTxMachineStateTransitions(t *testing.T) {

}

func TestLaAggPortTxMachineInvalidStateTransitions(t *testing.T) {

}

// TODO add more tests
// 1) invalid events on stats
// 2) pkt events
