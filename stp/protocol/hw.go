// hw.go
package stp

import (
	hwconst "asicd/asicdConstDefs"
	"asicd/pluginManager/pluginCommon"
	"asicdServices"
	"encoding/json"
	"fmt"
	"git.apache.org/thrift.git/lib/go/thrift"
	"io/ioutil"
	"net"
	"strconv"
	"strings"
	"utils/ipcutils"
)

type STPClientBase struct {
	Address            string
	Transport          thrift.TTransport
	PtrProtocolFactory *thrift.TBinaryProtocolFactory
	IsConnected        bool
}

type AsicdClient struct {
	STPClientBase
	ClientHdl *asicdServices.ASICDServicesClient
}

type ClientJson struct {
	Name string `json:Name`
	Port int    `json:Port`
}

var asicdclnt AsicdClient

// look up the various other daemons based on c string
func GetClientPort(paramsFile string, c string) int {
	var clientsList []ClientJson

	bytes, err := ioutil.ReadFile(paramsFile)
	if err != nil {
		StpLogger("ERROR", fmt.Sprintf("Error in reading configuration file:%s err:%s\n", paramsFile, err))
		return 0
	}

	err = json.Unmarshal(bytes, &clientsList)
	if err != nil {
		StpLogger("ERROR", "Error in Unmarshalling Json")
		return 0
	}

	for _, client := range clientsList {
		if client.Name == c {
			return client.Port
		}
	}
	return 0
}

// connect the the asic d
func ConnectToClients(paramsFile string) {
	port := GetClientPort(paramsFile, "asicd")
	if port != 0 {
		StpLogger("INFO", fmt.Sprintf("found asicd at port %d\n", port))
		asicdclnt.Address = "localhost:" + strconv.Itoa(port)
		asicdclnt.Transport, asicdclnt.PtrProtocolFactory, _ = ipcutils.CreateIPCHandles(asicdclnt.Address)
		if asicdclnt.Transport != nil && asicdclnt.PtrProtocolFactory != nil {
			fmt.Println("connecting to asicd\n")
			asicdclnt.ClientHdl = asicdServices.NewASICDServicesClientFactory(asicdclnt.Transport, asicdclnt.PtrProtocolFactory)
			asicdclnt.IsConnected = true
			// lets gather all info needed from asicd such as the port
			ConstructPortConfigMap()
		}
	}
}

func ConstructPortConfigMap() {
	currMarker := asicdServices.Int(hwconst.MIN_SYS_PORTS)
	if asicdclnt.IsConnected {
		StpLogger("INFO", "Calling asicd for port config")
		count := asicdServices.Int(10)
		for {
			bulkInfo, err := asicdclnt.ClientHdl.GetBulkPortState(currMarker, count)
			if err != nil {
				StpLogger("INFO", fmt.Sprintf("Error: %s", err))
				return
			}
			bulkCfgInfo, err := asicdclnt.ClientHdl.GetBulkPortConfig(currMarker, count)
			if err != nil {
				StpLogger("INFO", fmt.Sprintf("Error: %s", err))
				return
			}
			objCount := int(bulkInfo.Count)
			more := bool(bulkInfo.More)
			currMarker = asicdServices.Int(bulkInfo.EndIdx)
			for i := 0; i < objCount; i++ {
				portNum := bulkInfo.PortStateList[i].PortNum
				ent := PortConfigMap[portNum]
				ent.Name = bulkInfo.PortStateList[i].Name
				ent.HardwareAddr, _ = net.ParseMAC(bulkCfgInfo.PortConfigList[i].MacAddr)
				PortConfigMap[portNum] = ent
			}
			if more == false {
				return
			}
		}
	}
}

// convert the lacp port names name to asic format string list
func asicDPortBmpFormatGet(distPortList []string) string {
	s := ""
	dLength := len(distPortList)

	for i := 0; i < dLength; i++ {
		num := strings.Split(distPortList[i], "-")[1]
		if i == dLength-1 {
			s += num
		} else {
			s += num + ","
		}
	}
	return s

}

func asicdGetPortLinkStatus(intfNum string) bool {

	if asicdclnt.ClientHdl != nil {
		bulkInfo, err := asicdclnt.ClientHdl.GetBulkPortState(asicdServices.Int(hwconst.MIN_SYS_PORTS), asicdServices.Int(hwconst.MIN_SYS_PORTS))
		if err == nil && bulkInfo.Count != 0 {
			objCount := int64(bulkInfo.Count)
			for i := int64(0); i < objCount; i++ {
				if bulkInfo.PortStateList[i].Name == intfNum {
					return bulkInfo.PortStateList[i].OperState == pluginCommon.UpDownState[1]
				}
			}
		}
		fmt.Printf("asicDGetPortLinkSatus: could not get status for port %s, failure in get method\n", intfNum)
	}
	return true

}