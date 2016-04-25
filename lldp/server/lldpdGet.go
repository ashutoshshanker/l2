package lldpServer

import (
	"fmt"
	"lldpd"
	"strconv"
)

/*  helper function to convert Mandatory TLV's (chassisID, portID, TTL) from byte
 *  format to string
 */
func (svr *LLDPServer) PopulateMandatoryTLV(ifIndex int32,
	entry *lldpd.LLDPIntfState) bool {
	gblInfo, exists := svr.lldpGblInfo[ifIndex]
	if !exists {
		svr.logger.Err(fmt.Sprintln("Entry not found for", ifIndex))
		return exists
	}
	entry.LocalPort = gblInfo.Name
	if gblInfo.rxFrame != nil {
		entry.PeerMac = gblInfo.GetChassisIdInfo()
		entry.Port = gblInfo.GetPortIdInfo()
		entry.HoldTime = strconv.Itoa(int(gblInfo.rxFrame.TTL))
	}
	entry.IfIndex = gblInfo.IfIndex
	entry.Enable = true
	return exists
}

/*  Server get bulk for lldp up intf state's
 */
func (svr *LLDPServer) GetBulkLLDPIntfState(idx int, cnt int) (int, int,
	[]lldpd.LLDPIntfState) {
	var nextIdx int
	var count int

	if svr.lldpIntfStateSlice == nil {
		svr.logger.Info("No neighbor learned")
		return 0, 0, nil
	}

	length := len(svr.lldpUpIntfStateSlice)
	result := make([]lldpd.LLDPIntfState, cnt)

	var i, j int

	for i, j = 0, idx; i < cnt && j < length; j++ {
		key := svr.lldpUpIntfStateSlice[j]
		succes := svr.PopulateMandatoryTLV(key, &result[i])
		if !succes {
			result = nil
			return 0, 0, nil
		}
		i++
	}

	if j == length {
		nextIdx = 0
	}
	count = i
	return nextIdx, count, result
}
