//
//Copyright [2016] [SnapRoute Inc]
//
//Licensed under the Apache License, Version 2.0 (the "License");
//you may not use this file except in compliance with the License.
//You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
//	 Unless required by applicable law or agreed to in writing, software
//	 distributed under the License is distributed on an "AS IS" BASIS,
//	 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//	 See the License for the specific language governing permissions and
//	 limitations under the License.
//
// _______  __       __________   ___      _______.____    __    ____  __  .___________.  ______  __    __  
// |   ____||  |     |   ____\  \ /  /     /       |\   \  /  \  /   / |  | |           | /      ||  |  |  | 
// |  |__   |  |     |  |__   \  V  /     |   (----` \   \/    \/   /  |  | `---|  |----`|  ,----'|  |__|  | 
// |   __|  |  |     |   __|   >   <       \   \      \            /   |  |     |  |     |  |     |   __   | 
// |  |     |  `----.|  |____ /  .  \  .----)   |      \    /\    /    |  |     |  |     |  `----.|  |  |  | 
// |__|     |_______||_______/__/ \__\ |_______/        \__/  \__/     |__|     |__|      \______||__|  |__| 
//                                                                                                           

// markerResponder.go
package lacp

import (
	"fmt"
	"github.com/google/gopacket/layers"
	"strconv"
	"strings"
	"utils/fsm"
)

const MarkerResponderModuleStr = "LAMP Marker Responder"

// Lamp Marker Responder States
const (
	LampMarkerResponderNone = iota + 1
	LampMarkerResponderStateWaitForMarker
	LampMarkerResponderStateRespondToMarker
)

var LampMarkerResponderStateStrMap map[fsm.State]string

func LampMarkerResponderStrStateMapCreate() {
	LampMarkerResponderStateStrMap = make(map[fsm.State]string)
	LampMarkerResponderStateStrMap[LampMarkerResponderNone] = "None"
	LampMarkerResponderStateStrMap[LampMarkerResponderStateWaitForMarker] = "WaitForMarker"
	LampMarkerResponderStateStrMap[LampMarkerResponderStateRespondToMarker] = "RespondToMarker"
}

// lamp responder events
const (
	LampMarkerResponderEventBegin = iota + 1
	LampMarkerResponderEventLampPktRx
	LampMarkerResponderEventIntentionalFallthrough
	LampMarkerResponderEventKillSignal
)

type LampRxLampPdu struct {
	pdu          *layers.LAMP
	src          string
	responseChan chan string
}

// LacpRxMachine holds FSM and current State
// and event channels for State transitions
type LampMarkerResponderMachine struct {
	// for debugging
	PreviousState fsm.State

	Machine *fsm.Machine

	p *LaAggPort

	// debug log
	log chan string

	// machine specific events
	LampMarkerResponderEvents          chan LacpMachineEvent
	LampMarkerResponderPktRxEvent      chan LampRxLampPdu
	LampMarkerResponderKillSignalEvent chan bool
	LampMarkerResponderLogEnableEvent  chan bool
}

func (mr *LampMarkerResponderMachine) PrevState() fsm.State { return mr.PreviousState }

// PrevStateSet will set the previous State
func (mr *LampMarkerResponderMachine) PrevStateSet(s fsm.State) { mr.PreviousState = s }

// Stop should clean up all resources
func (mr *LampMarkerResponderMachine) Stop() {

	// stop the go routine
	mr.LampMarkerResponderKillSignalEvent <- true

	close(mr.LampMarkerResponderEvents)
	close(mr.LampMarkerResponderPktRxEvent)
	close(mr.LampMarkerResponderKillSignalEvent)
	close(mr.LampMarkerResponderLogEnableEvent)

}

// A helpful function that lets us apply arbitrary rulesets to this
// instances State machine without reallocating the machine.
func (mr *LampMarkerResponderMachine) Apply(r *fsm.Ruleset) *fsm.Machine {
	if mr.Machine == nil {
		mr.Machine = &fsm.Machine{}
	}

	// Assign the ruleset to be used for this machine
	mr.Machine.Rules = r
	mr.Machine.Curr = &LacpStateEvent{
		strStateMap: LampMarkerResponderStateStrMap,
		logEna:      mr.p.logEna,
		logger:      mr.LampMarkerResponderLog,
		owner:       MarkerResponderModuleStr,
	}

	return mr.Machine
}

// NewLacpRxMachine will create a new instance of the LacpRxMachine
func NewLampMarkerResponder(port *LaAggPort) *LampMarkerResponderMachine {
	mr := &LampMarkerResponderMachine{
		p:                                  port,
		log:                                port.LacpDebug.LacpLogChan,
		PreviousState:                      LacpRxmStateNone,
		LampMarkerResponderEvents:          make(chan LacpMachineEvent, 10),
		LampMarkerResponderPktRxEvent:      make(chan LampRxLampPdu, 1000),
		LampMarkerResponderKillSignalEvent: make(chan bool),
		LampMarkerResponderLogEnableEvent:  make(chan bool)}

	port.MarkerResponderFsm = mr

	return mr
}

func (mr *LampMarkerResponderMachine) LampMarkerResponderWaitForMarker(m fsm.Machine, data interface{}) fsm.State {
	return LampMarkerResponderStateWaitForMarker
}

func (mr *LampMarkerResponderMachine) LampMarkerResponderRespondToMarker(m fsm.Machine, data interface{}) fsm.State {
	p := mr.p
	lampPduInfo := data.(*layers.LAMP)

	// validate some of the packet info
	if lampPduInfo.Marker.Length != layers.LAMPMarkerTlvLength {
		mr.LampMarkerResponderLog(fmt.Sprintf("ERROR RX INVALID TLV LENGTH FROM MARKER PDU received %d expected %d", lampPduInfo.Marker.Length, layers.LAMPMarkerTlvLength))
		p.LacpCounter.AggPortStatsIllegalRx += 1
		return LampMarkerResponderStateWaitForMarker
	}

	// we only want to handle marker pdu, not response since we are not
	// generating
	if lampPduInfo.Marker.TlvType != layers.LAMPTLVMarkerInfo {
		if lampPduInfo.Marker.TlvType == layers.LAMPTLVMarkerResponder {
			p.LacpCounter.AggPortStatsMarkerResponsePDUsRx += 1
		} else {
			p.LacpCounter.AggPortStatsIllegalRx += 1
		}
		return LampMarkerResponderStateWaitForMarker
	} else {
		p.LacpCounter.AggPortStatsMarkerPDUsRx += 1

		// info in receive is same as generated just need to change the tlvType
		lampResponsePdu := lampPduInfo
		lampResponsePdu.Marker.TlvType = layers.LAMPTLVMarkerResponder

		for _, ftx := range LaSysGlobalTxCallbackListGet(p) {
			//txm.LacpTxmLog(fmt.Sprintf("Sending Tx packet port %d pkts %d", p.PortNum, txm.txPkts))
			ftx(p.PortNum, lampResponsePdu)
			p.LacpCounter.AggPortStatsMarkerResponsePDUsTx += 1
		}
	}
	return LampMarkerResponderStateRespondToMarker
}

func LampMarkerResponderFSMBuild(p *LaAggPort) *LampMarkerResponderMachine {

	LampMarkerResponderStrStateMapCreate()

	rules := fsm.Ruleset{}

	// Instantiate a new LacpRxMachine
	// Initial State will be a psuedo State known as "begin" so that
	// we can transition to the initalize State
	mr := NewLampMarkerResponder(p)

	//BEGIN -> WAIT FOR MARKER
	rules.AddRule(LacpRxmStateNone, LampMarkerResponderEventBegin, mr.LampMarkerResponderWaitForMarker)
	rules.AddRule(LampMarkerResponderStateWaitForMarker, LacpRxmEventBegin, mr.LampMarkerResponderWaitForMarker)
	rules.AddRule(LampMarkerResponderStateRespondToMarker, LacpRxmEventBegin, mr.LampMarkerResponderWaitForMarker)

	// PKT RX -> RESPOND TO MARKER
	rules.AddRule(LampMarkerResponderStateWaitForMarker, LampMarkerResponderEventLampPktRx, mr.LampMarkerResponderRespondToMarker)

	// INTENTIONAL FALLTHROUGH ->  WAIT FOR MARKER
	rules.AddRule(LampMarkerResponderStateWaitForMarker, LampMarkerResponderEventIntentionalFallthrough, mr.LampMarkerResponderWaitForMarker)

	// Create a new FSM and apply the rules
	mr.Apply(&rules)

	return mr
}

// LampMarkerResponderMain:  802.1ax-2014 Table 6-28
// Creation of Marker Responder State Machine State transitions and callbacks
// and create go routine to pend on events
func (p *LaAggPort) LampMarkerResponderMain() {

	// Build the State machine for Lacp Receive Machine according to
	// 802.1ax Section 6.4.12 Receive Machine
	mr := LampMarkerResponderFSMBuild(p)

	// set the inital State
	mr.Machine.Start(mr.PrevState())

	// lets create a go routing which will wait for the specific events
	// that the RxMachine should handle.
	go func(m *LampMarkerResponderMachine) {
		m.LampMarkerResponderLog("Machine Start")
		defer m.p.wg.Done()
		for {
			select {
			case <-m.LampMarkerResponderKillSignalEvent:
				m.LampMarkerResponderLog("Machine End")
				return

			case event := <-m.LampMarkerResponderEvents:
				rv := m.Machine.ProcessEvent(event.src, event.e, nil)

				if rv != nil {
					m.LampMarkerResponderLog(strings.Join([]string{error.Error(rv), event.src, LampMarkerResponderStateStrMap[m.Machine.Curr.CurrentState()], strconv.Itoa(int(event.e))}, ":"))
				}

				// respond to caller if necessary so that we don't have a deadlock
				if event.responseChan != nil {
					SendResponse(RxMachineModuleStr, event.responseChan)
				}
			case rx := <-m.LampMarkerResponderPktRxEvent:
				//m.LacpRxmLog(fmt.Sprintf("RXM: received packet %d %s", m.p.PortNum, rx.src))
				// lets check if the port has moved
				p.LacpCounter.AggPortStatsMarkerPDUsRx += 1

				rv := m.Machine.ProcessEvent(MarkerResponderModuleStr, LampMarkerResponderEventLampPktRx, rx.pdu)
				if rv != nil {
					m.LampMarkerResponderLog(strings.Join([]string{error.Error(rv), rx.src, LampMarkerResponderStateStrMap[m.Machine.Curr.CurrentState()], strconv.Itoa(int(LampMarkerResponderEventLampPktRx))}, ":"))
				}
				// processed the packet, now lets send a response
				if m.Machine.Curr.CurrentState() == LampMarkerResponderStateRespondToMarker {
					rv = m.Machine.ProcessEvent(MarkerResponderModuleStr, LampMarkerResponderEventIntentionalFallthrough, rx.pdu)
					if rv != nil {
						m.LampMarkerResponderLog(strings.Join([]string{error.Error(rv), MarkerResponderModuleStr, LampMarkerResponderStateStrMap[m.Machine.Curr.CurrentState()], strconv.Itoa(int(LampMarkerResponderEventIntentionalFallthrough))}, ":"))
					}
				}

				// respond to caller if necessary so that we don't have a deadlock
				if rx.responseChan != nil {
					SendResponse(RxMachineModuleStr, rx.responseChan)
				}

			case ena := <-m.LampMarkerResponderLogEnableEvent:
				m.Machine.Curr.EnableLogging(ena)

			}
		}
	}(mr)
}
