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

// global
package lacp

import (
	"fmt"
	"net"
	//"github.com/google/gopacket/layers"
	"time"
)

var LacpStartTime time.Time

type TxCallback func(port uint16, data interface{})

type PortIdKey struct {
	Name string
	Id   uint16
}

type AggIdKey struct {
	Name string
	Id   int
}

type LacpSysGlobalInfo struct {
	LacpEnabled                bool
	PortMap                    map[PortIdKey]*LaAggPort
	PortList                   []*LaAggPort
	AggMap                     map[AggIdKey]*LaAggregator
	AggList                    []*LaAggregator
	SystemDefaultParams        LacpSystem
	PartnerSystemDefaultParams LacpSystem
	ActorStateDefaultParams    LacpPortInfo
	PartnerStateDefaultParams  LacpPortInfo
	SysKey                     LacpSystem

	// mux machine coupling
	// false == NOT COUPLING, true == COUPLING
	muxCoupling bool

	// list of tx function which should be called for a given port
	TxCallbacks map[string][]TxCallback
}

// holds default lacp State info
var gLacpSysGlobalInfo map[LacpSystem]*LacpSysGlobalInfo
var gLacpSysGlobalInfoList []*LacpSysGlobalInfo

func convertNetHwAddressToSysIdKey(mac net.HardwareAddr) [6]uint8 {
	var macArr [6]uint8
	macArr[0] = mac[0]
	macArr[1] = mac[1]
	macArr[2] = mac[2]
	macArr[3] = mac[3]
	macArr[4] = mac[4]
	macArr[5] = mac[5]
	return macArr
}

func convertSysIdKeyToNetHwAddress(mac [6]uint8) net.HardwareAddr {

	x := make(net.HardwareAddr, 6)
	x[0] = mac[0]
	x[1] = mac[1]
	x[2] = mac[2]
	x[3] = mac[3]
	x[4] = mac[4]
	x[5] = mac[5]
	return x
}

// NewLacpSysGlobalInfo will create a port map, agg map
// as well as set some default parameters to be used
// to setup each new port.
//
// NOTE: Only one instance should exist on live System
func LacpSysGlobalInfoInit(sysId LacpSystem) *LacpSysGlobalInfo {

	if gLacpSysGlobalInfo == nil {
		gLacpSysGlobalInfo = make(map[LacpSystem]*LacpSysGlobalInfo)
	}

	sysKey := sysId

	if _, ok := gLacpSysGlobalInfo[sysKey]; !ok {
		fmt.Println("LASYS: global vars init sysId", sysId)
		gLacpSysGlobalInfo[sysKey] = &LacpSysGlobalInfo{
			LacpEnabled:                true,
			PortMap:                    make(map[PortIdKey]*LaAggPort),
			PortList:                   make([]*LaAggPort, 0),
			AggMap:                     make(map[AggIdKey]*LaAggregator),
			AggList:                    make([]*LaAggregator, 0),
			SystemDefaultParams:        LacpSystem{Actor_System_priority: 0x8000},
			PartnerSystemDefaultParams: LacpSystem{Actor_System_priority: 0x0},
			TxCallbacks:                make(map[string][]TxCallback),
			SysKey:                     sysKey,
		}

		gLacpSysGlobalInfoList = append(gLacpSysGlobalInfoList, gLacpSysGlobalInfo[sysKey])

		gLacpSysGlobalInfo[sysKey].SystemDefaultParams.LacpSystemActorSystemIdSet(convertSysIdKeyToNetHwAddress(sysId.actor_System))

		// Partner is brought up as aggregatible
		LacpStateSet(&gLacpSysGlobalInfo[sysKey].PartnerStateDefaultParams.State, LacpStateAggregatibleUp)

		// Actor is brought up as individual
		LacpStateSet(&gLacpSysGlobalInfo[sysKey].ActorStateDefaultParams.State, LacpStateIndividual)
	}
	return gLacpSysGlobalInfo[sysKey]
}

func LacpSysGlobalInfoGet() []*LacpSysGlobalInfo {
	return gLacpSysGlobalInfoList
}

func LacpSysGlobalInfoByIdGet(sysId LacpSystem) *LacpSysGlobalInfo {
	return LacpSysGlobalInfoInit(sysId)
}

func LacpSysGlobalDefaultSystemGet(sysId LacpSystem) *LacpSystem {
	return &gLacpSysGlobalInfo[sysId].SystemDefaultParams
}

func LacpSysGlobalDefaultPartnerSystemGet(sysId LacpSystem) *LacpSystem {
	return &gLacpSysGlobalInfo[sysId].PartnerSystemDefaultParams
}

func LacpSysGlobalDefaultPartnerInfoGet(sysId LacpSystem) *LacpPortInfo {
	return &gLacpSysGlobalInfo[sysId].PartnerStateDefaultParams
}

func LacpSysGlobalDefaultActorSystemGet(sysId LacpSystem) *LacpPortInfo {
	return &gLacpSysGlobalInfo[sysId].ActorStateDefaultParams
}

func (g *LacpSysGlobalInfo) LacpSysGlobalAggListGet() []*LaAggregator {
	return g.AggList
}

func (g *LacpSysGlobalInfo) LacpSysGlobalAggPortListGet() []*LaAggPort {
	return g.PortList
}

func (g *LacpSysGlobalInfo) LaSysGlobalRegisterTxCallback(intf string, f TxCallback) {
	g.TxCallbacks[intf] = append(g.TxCallbacks[intf], f)
}

func (g *LacpSysGlobalInfo) LaSysGlobalDeRegisterTxCallback(intf string) {
	delete(g.TxCallbacks, intf)
}

func LaSysGlobalTxCallbackListGet(p *LaAggPort) []TxCallback {

	var a *LaAggregator
	var sysId LacpSystem
	if LaFindAggById(p.AggId, &a) {
		mac, _ := net.ParseMAC(a.Config.SystemIdMac)
		sysId.actor_System = convertNetHwAddressToSysIdKey(mac)
		sysId.Actor_System_priority = a.Config.SystemPriority
	}
	if s, sok := gLacpSysGlobalInfo[sysId]; sok {
		if fList, pok := s.TxCallbacks[p.IntfNum]; pok {
			return fList
		}
	}

	// temporary function
	x := func(port uint16, data interface{}) {
		p.LacpDebug.logger.Info(fmt.Sprintf("TX not registered for port\n", p.IntfNum, p.portId))
		//lacp := data.(*layers.LACP)
		//fmt.Printf("%#v\n", *lacp)
	}

	debugTxList := make([]TxCallback, 0)
	debugTxList = append(debugTxList, x)
	return debugTxList
}
