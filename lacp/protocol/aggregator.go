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

// aggregator
package lacp

import (
	//"fmt"
	//"log/syslog"
	"net"
	"time"
)

// Indicates on a port what State
// the aggSelected is in
const (
	LacpAggSelected = iota + 1
	LacpAggStandby
	LacpAggUnSelected
)

type LacpAggregatorStats struct {
	// does not include lacp or marker pdu
	octetsTx              int
	octetsRx              int
	framesTx              int
	framesRx              int
	mcFramesTxOk          int
	mcFramesRxOk          int
	bcFramesTxOk          int
	bcFramesRxOk          int
	framesDiscardedonTx   int
	framesDiscardedonRx   int
	framesWithTxErrors    int
	framesWithRxErrors    int
	unknownProtocolFrames int
}

// 802.1.AX-2014 7.3.1.1 Aggregator attributes GET-SET
type AggregatorObject struct {
	// GET
	AggId int
	// GET
	AggDescription string
	// GET-SET
	AggName string
	// GET-SET
	AggActorSystemID [6]uint8
	// GET-SET
	AggActorSystemPriority uint16
	// GET
	AggAggregateOrIndividual bool
	// GET-SET
	AggActorAdminKey uint16
	// GET
	AggActorOperKey uint16
	// GET
	AggMACAddress [6]uint8
	// GET
	AggPartnerSystemID [6]uint8
	// GET
	AggPartnerSystemPriority uint16
	// GET
	AggPartnerOperKey uint16
	// GET-SET   up/down enum
	AggAdminState bool
	// GET
	AggOperState bool
	// GET
	AggTimeLastOperChange int
	// GET  sum of data rate of each link
	AggDataRate int
	// GET
	AggStats LacpAggregatorStats
	// GET-SET  enable/disable enum
	AggLinkUpDownNotificationEnable bool
	// NOTIFICATION
	AggLinkUpNotification bool
	// NOTIFICATION
	AggLinkDownNotification bool
	// GET  list of AggPortID
	AggPortList []int
	// GET-SET 10s of microseconds
	AggCollectorMaxDelay uint16
	// GET-SET
	AggPortAlgorithm [3]uint8
	// GET-SET
	AggPartnerAdminPortAlgorithm [3]uint8
	// GET-SET up to 4096 values conversationids
	AggConversationAdminLink []int
	// GET-SET
	AggPartnerAdminPortConverstaionListDigest [16]uint8
	// GET-SET
	AggAdminDiscardWrongConversation bool
	// GET-SET 4096 values
	AggAdminServiceConversationMap []int
	// GET-SET
	AggPartnerAdminConvServiceMappingDigest [16]uint8
}

// 802.1ax-2014 Section 6.4.6 Variables associated with each Aggregator
// Section 7.3.1.1

type LaAggregator struct {
	// 802.1ax Section 7.3.1.1 && 6.3.2
	// Aggregator_Identifier
	AggId          int
	HwAggId        int32
	AggDescription string // 255 max chars
	AggName        string // 255 max chars
	AggType        uint32 // LACP/STATIC
	AggMinLinks    uint16

	// lacp configuration info
	Config LacpConfigInfo

	// aggregation capability
	// TRUE - port attached to this aggregetor is not capable
	//        of aggregation to any other aggregator
	// FALSE - port attached to this aggregator is able of
	//         aggregation to any other aggregator
	// Individual_Aggregator
	aggOrIndividual bool
	// Actor_Admin_Aggregator_Key
	actorAdminKey uint16
	// Actor_Oper_Aggregator_Key
	ActorOperKey uint16
	//Aggregator_MAC_address
	aggMacAddr [6]uint8
	// Partner_System
	partnerSystemId [6]uint8
	// Partner_System_Priority
	partnerSystemPriority int
	// Partner_Oper_Aggregator_Key
	PartnerOperKey int

	//		1 : string 	NameKey
	//	    2 : i32 	Interval
	// 	    3 : i32 	LacpMode
	//	    4 : string 	SystemIdMac
	//	    5 : i16 	SystemPriority

	// UP/DOWN
	AdminState bool
	OperState  bool

	// date of last oper change
	timeOfLastOperChange time.Time

	// aggrigator stats
	stats LacpAggregatorStats

	// Receive_State
	rxState bool
	// Transmit_State
	txState bool

	// sum of data rate of each link in aggregation (read-only)
	dataRate int

	// LAG is ready to add a port in the ReadyN State
	ready bool

	// Port number from LaAggPort
	// LAG_Ports
	PortNumList []uint16

	// Ports in Distributed State
	DistributedPortNumList []string

	// For now this value assumes the value of the linux modes
	// 0 - L2
	// 1 - L2+L3
	// 2 - L3+L4
	// 3 - ENCAP
	// 4 - ENCAP2
	LagHash uint32

	LacpDebug *LacpDebug
	log       chan string
}

func NewLaAggregator(ac *LaAggConfig) *LaAggregator {
	netMac, _ := net.ParseMAC(ac.Lacp.SystemIdMac)
	sysId := LacpSystem{
		actor_System:          convertNetHwAddressToSysIdKey(netMac),
		Actor_System_priority: ac.Lacp.SystemPriority,
	}
	sgi := LacpSysGlobalInfoByIdGet(sysId)
	a := &LaAggregator{
		AggName:                ac.Name,
		AggId:                  ac.Id,
		AdminState:             ac.Enabled,
		aggMacAddr:             sysId.actor_System,
		actorAdminKey:          ac.Key,
		AggType:                ac.Type,
		AggMinLinks:            ac.MinLinks,
		Config:                 ac.Lacp,
		partnerSystemId:        [6]uint8{0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		ready:                  true,
		PortNumList:            make([]uint16, 0),
		DistributedPortNumList: make([]string, 0),
		LagHash:                ac.HashMode,
	}

	a.LacpDebugAggEventLogMain()

	// want to ensure that the application can use a string name or id
	// to uniquely identify a lag
	Key := AggIdKey{Id: ac.Id,
		Name: ac.Name}

	// add agg to map
	sgi.AggMap[Key] = a
	sgi.AggList = append(sgi.AggList, a)

	for _, pId := range ac.LagMembers {
		a.PortNumList = append(a.PortNumList, pId)
	}

	return a
}

// warning for each call the map may change
func LaGetAggNext(agg **LaAggregator) bool {
	returnNext := false
	for _, sgi := range LacpSysGlobalInfoGet() {
		for _, a := range sgi.LacpSysGlobalAggListGet() {
			/*
				if *agg == nil {
					fmt.Println("agg map curr %d", a.AggId)
				} else {
					fmt.Println(fmt.Sprintf("agg map prev %d curr %d", (*agg).AggId, a.AggId))
				}
			*/
			if *agg == nil {
				// first agg
				*agg = a
				return true
			} else if (*agg).AggId == a.AggId {
				// found agg
				returnNext = true
			} else if returnNext {
				// next agg
				*agg = a
				return true
			}
		}
	}
	*agg = nil
	return false
}

func LaFindAggById(aggId int, agg **LaAggregator) bool {
	for _, sgi := range LacpSysGlobalInfoGet() {
		for _, a := range sgi.LacpSysGlobalAggListGet() {
			if a.AggId == aggId {
				*agg = a
				return true
			}
		}
	}
	return false
}

func LaFindAggByName(AggName string, agg **LaAggregator) bool {
	for _, sgi := range LacpSysGlobalInfoGet() {
		for _, a := range sgi.LacpSysGlobalAggListGet() {
			if a.AggName == AggName {
				*agg = a
				return true
			}
		}
	}
	return false
}

func LaAggPortNumListPortIdExist(Key uint16, portId uint16) bool {
	var a *LaAggregator
	if LaFindAggByKey(Key, &a) {
		//fmt.Println("Found agg", Key, "PortList", a.PortNumList)
		for _, pId := range a.PortNumList {
			if pId == portId {
				return true
			}
		}
	}
	return false
}

func LaFindAggByKey(Key uint16, agg **LaAggregator) bool {

	for _, sgi := range LacpSysGlobalInfoGet() {
		for _, a := range sgi.LacpSysGlobalAggListGet() {
			if a.actorAdminKey == Key {
				*agg = a
				return true
			}
		}
	}
	return false
}

func (a *LaAggregator) DeleteLaAgg() {
	for _, sgi := range LacpSysGlobalInfoGet() {
		lookupKey := AggIdKey{Id: a.AggId, Name: a.AggName}
		for Key, _ := range sgi.AggMap {
			if Key.Id == lookupKey.Id &&
				Key.Name == lookupKey.Name {
				delete(sgi.AggMap, Key)
				break
			}
		}

		for i, agg := range sgi.LacpSysGlobalAggListGet() {
			if agg.actorAdminKey == a.actorAdminKey {
				sgi.AggList = append(sgi.AggList[:i], sgi.AggList[i+1:]...)
				a.AggId = 0
				a.actorAdminKey = 0
				a.partnerSystemId = [6]uint8{0x00, 0x00, 0x00, 0x00, 0x00, 0x00}
				a.ready = false
				break
			}
		}
	}
}
