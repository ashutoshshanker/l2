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

// lacppdu
package lacp

import ()

/*
SlowProtocolDmac := [6]uint8{0x01, 0x80, 0xC2, 0x00, 0x00, 0x02}
const SlowProtocolDmacByte0 uint8 = 0x01
const SlowProtocolDmacByte1 uint8 = 0x80
const SlowProtocolDmacByte2 uint8 = 0xC2
const SlowProtocolDmacByte3 uint8 = 0x00
const SlowProtocolDmacByte4 uint8 = 0x00
const SlowProtocolDmacByte5 uint8 = 0x02
const SlowProtocolEtherType uint16 = 0x8809
const LacpSubType uint8 = 1
const LampSubType uint8 = 2
const LampMarkerInformation uint8 = 1

type LacpPduInfoTlv struct {
	tlv_type uint8
	len      uint8
	info     LacpPortInfo
	reserved [3]uint8
}

// 6.4.3.2
type LacpPduCollectorInfoTlv struct {
	tlv_type uint8 // 0x03
	len      uint8 // 0x10
	maxDelay uint16
	reserved [12]uint8
}

// 6.4.2.4 Version 2 TLV
// 6.4.2.4.1  Port Algorithm TLV 0x04
//
//
//  Algorithm         Value
//  Unspecified         0
//  C-VID               1
//  S-VID               2
//  I-SID               3
//  TE-SID              4
//  ECMP Flow Hash      5
//  Reserved            6-255
type LacpPduPortAlgorithmTlv struct {
	tlv_type             uint8
	len                  uint8 // 6
	actor_port_algorithm uint32
}

// 6.4.2.4 Version 2 TLV
// 6.4.2.4.2 Port Conversation ID digest TLV 0x05
type LacpPduPortConversationIdDigestTlv struct {
	tlv_type                           uint8
	len                                uint8 // 0x14
	link_number_id                     uint16
	actor_conversation_linklist_digest [16]uint8
}

// 6.4.2.4 Version 2 TLV
// 6.4.2.4.3 Port Conversation Mask 1 TLV 0x06
type LacpPduPortConversationMask1Tlv struct {
	tlv_type   uint8
	len        uint8 // 131
	mask_State uint8
	mask_1     [128]uint8
}

// 6.4.2.4 Version 2 TLV
// 6.4.2.4.3 Port Conversation Mask 2 TLV 0x07
type LacpPduPortConversationMask2Tlv struct {
	tlv_type uint8
	mask_len uint8 // 130
	mask_2   [128]uint8
}

// 6.4.2.4 Version 2 TLV
// 6.4.2.4.3 Port Conversation Mask 3 TLV 0x08
type LacpPduPortConversationMask3Tlv struct {
	tlv_type uint8
	mask_len uint8 // 130
	mask_3   [128]uint8
}

// 6.4.2.4 Version 2 TLV
// 6.4.2.4.3 Port Conversation Mask 4 TLV 0x09
type LacpPortConversationMask4Tlv struct {
	tlv_type uint8
	mask_len uint8 // 130
	mask_4   [128]uint8
}

// 6.4.2.4 Version 2 TLV
// 6.4.2.4.4 Port Conversation Service Mapping TLV 0x0A
type LacpPduPortConversationServiceMappingTlv struct {
	tlv_type uint8
	len      uint8 // 18
	actor    [16]uint8
}

// 6.4.2.3
// format of data below is conforms to
// version 1 && 2, but version 2 allows
// for additional TLV's
type LacpPdu struct {
	subType uint8
	version uint8
	// tlv 0x01, len 0x14
	actor LacpPduInfoTlv
	// tlv 0x02, len 0x14
	partner LacpPduInfoTlv
	// tlv 0x03, len 0x10
	collector LacpPduCollectorInfoTlv
	// Version 2 TLV follow but not included in
	// this structure as they are optional and
	// variable
}
*/
