package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

type ResourceUsageVirtualMachine struct {
	CpuUtilization      string
	MemoryAvailable     string
	DiskLatency         string
	DiskIOPs            string
	DiskBytes           string
	NetworkSentRate     string
	NetworkReceivedRate string
	Tables              []struct {
		Name    string `json:"name"`
		Columns []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"columns"`
		Rows [][]interface{} `json:"rows"`
	} `json:"tables"`
}

func (r *ResourceUsageVirtualMachine) getVmUsage(resourceGroup string, resourceID string) (*ResourceUsageVirtualMachine, error) {

	if resourceID == "" || resourceGroup == "" {
		return nil, fmt.Errorf("resource id and resource group names are required")
	}

	request := Request{
		Name:    "vm" + "_" + resourceGroup + "_" + resourceID,
		Url:     r.getUrl(),
		Method:  Methods.POST,
		Payload: r.getPayload(resourceGroup, resourceID),
		Header:  r.getHeader(),
		IsCache: false,
	}

	errors := request.Execute()
	IfErrorsPrintThem(errors)

	body := request.GetResponse()
	_ = json.Unmarshal(body, r)
	r.setUsageValue()
	return r, nil
}

func (r *ResourceUsageVirtualMachine) getPayload(resourceGroup string, resourceID string) string {
	payload := "{\"query\": \"let startDateTime = datetime('{{startdate}}T08:00:00.000Z');" +
		"let endDateTime = datetime('{{enddate}}T16:00:00.000Z');" +
		"let trendBinSize = 8h;let maxListSize = 1000;" +
		"let cpuMemory = materialize(InsightsMetrics| where TimeGenerated between (startDateTime .. endDateTime)| " +
		"where _ResourceId =~ '/subscriptions/{{subscriptionid}}/resourcegroups/{{resourcegroup}}/providers/microsoft.compute/virtualmachines/{{resourceid}}'| " +
		"where Origin == 'vm.azm.ms'| where (Namespace == 'Processor' and Name == 'UtilizationPercentage') or (Namespace == 'Memory' and Name == 'AvailableMB')| " +
		"project TimeGenerated, Name, Namespace, Val);let networkDisk = materialize(InsightsMetrics| " +
		"where TimeGenerated between (startDateTime .. endDateTime)| " +
		"where _ResourceId =~ '/subscriptions/{{subscriptionid}}/resourcegroups/" +
		"{{resourcegroup}}/providers/microsoft.compute/" +
		"virtualmachines/{{resourceid}}'| " +
		"where Origin == 'vm.azm.ms'| " +
		"where (Namespace == 'Network' and Name in ('WriteBytesPerSecond', 'ReadBytesPerSecond'))    " +
		"or (Namespace == 'LogicalDisk' and Name in ('TransfersPerSecond', 'BytesPerSecond', 'TransferLatencyMs'))| " +
		"extend ComputerId = iff(isempty(_ResourceId), Computer, _ResourceId)| " +
		"summarize Val = sum(Val) by bin(TimeGenerated, 1m), " +
		"ComputerId, Name, Namespace| project TimeGenerated, Name, Namespace, Val);" +
		"let rawDataCached = cpuMemory| union networkDisk| " +
		"extend Val = iif(Name in ('WriteLatencyMs', 'ReadLatencyMs', 'TransferLatencyMs'), Val/1000.0, Val)| " +
		"project TimeGenerated,    cName = case(        Namespace == 'Processor' and Name == 'UtilizationPercentage', '% Processor Time'," +
		"        Namespace == 'Memory' and Name == 'AvailableMB', 'Available MBytes',        " +
		"Namespace == 'LogicalDisk' and Name == 'TransfersPerSecond', 'Disk Transfers/sec',        " +
		"Namespace == 'LogicalDisk' and Name == 'BytesPerSecond', 'Disk Bytes/sec',        " +
		"Namespace == 'LogicalDisk' and Name == 'TransferLatencyMs', 'Avg. Disk sec/Transfer',        " +
		"Namespace == 'Network' and Name == 'WriteBytesPerSecond', 'Bytes Sent/sec',        " +
		"Namespace == 'Network' and Name == 'ReadBytesPerSecond', 'Bytes Received/sec',        " +
		"Name    ),    cValue = case(Val < 0, real(0),Val);rawDataCached| summarize min(cValue),    " +
		"avg(cValue),    max(cValue),    percentiles(cValue, 5, 10, 50, 90, 95) by bin(TimeGenerated, trendBinSize), " +
		"cName| sort by TimeGenerated asc| summarize makelist(TimeGenerated, maxListSize),    makelist(min_cValue, maxListSize)," +
		"    makelist(avg_cValue, maxListSize),    makelist(max_cValue, maxListSize),    makelist(percentile_cValue_5, maxListSize),    " +
		"makelist(percentile_cValue_10, maxListSize),    makelist(percentile_cValue_50, maxListSize),    " +
		"makelist(percentile_cValue_90, maxListSize),    makelist(percentile_cValue_95, maxListSize) by cName| " +
		"join(    rawDataCached    | summarize min(cValue), avg(cValue), max(cValue), " +
		"percentiles(cValue, 5, 10, 50, 90, 95) by cName)on cName\"," +
		"\"timespan\": \"{{startdate}}T08:00:00.000Z/{{enddate}}T16:00:00.000Z\"}"

	payload = strings.ReplaceAll(payload, "{{startdate}}", startDate)
	payload = strings.ReplaceAll(payload, "{{enddate}}", endDate)
	payload = strings.ReplaceAll(payload, "{{subscriptionid}}", configuration.AccessToken.SubscriptionID)
	payload = strings.ReplaceAll(payload, "{{resourcegroup}}", resourceGroup)
	payload = strings.ReplaceAll(payload, "{{resourceid}}", resourceID)

	return payload
}

func (r *ResourceUsageVirtualMachine) getHeader() http.Header {
	var at = &AccessToken{}
	at.ExecuteRequest(at)
	token := fmt.Sprintf("Bearer %s", at.AccessToken)
	var header = http.Header{}
	header.Add("Authorization", token)
	header.Add("Accept", "application/json")
	header.Add("Content-Type", "application/json")
	return header
}

func (r *ResourceUsageVirtualMachine) getUrl() string {
	//TODO:::this may require some location
	url := fmt.Sprintf("https://management.azure.com//subscriptions/%s/resourcegroups/"+
		"defaultresourcegroup-eus/providers/microsoft.operationalinsights/workspaces/"+
		"defaultworkspace-%s-eus/query?api-version=2017-10-01", configuration.AccessToken.SubscriptionID, configuration.AccessToken.SubscriptionID)

	return url
}

func (r *ResourceUsageVirtualMachine) setUsageValue() {

	for i := 0; i < len(r.Tables); i++ {
		for x := 0; x < len(r.Tables[i].Rows); x++ {
			row := r.Tables[i].Rows[x]
			strTile := fmt.Sprintf("%v", row[0])

			//cpu
			if strings.Contains(strTile, "rocessor Time") {
				_, r.CpuUtilization = getCpuUtilization(row)
			}

			switch strTile {
			case "Available MBytes":
				_, _, r.MemoryAvailable = getVmAvailableMemory(row)
			case "Avg. Disk sec/Transfer":
				_, _, r.DiskLatency = getLogicalDiskLatency(row)
			case "Disk Bytes/sec":
				_, _, r.DiskBytes = getDiskBytesPerSeconds(row)
			case "Disk Transfers/sec":
				_, r.DiskIOPs = getLogicalDiskIOPs(row)
			case "Bytes Sent/sec":
				_, _, r.NetworkSentRate = getBytesSentRate(row)
			case "Bytes Received/sec":
				_, _, r.NetworkReceivedRate = getBytesReceivedRate(row)
			}
		}

	}
}

// interface raw is in Kilo Bytes - need to convert to MegaBytes
func getVmAvailableMemory(row []interface{}) (float64, float64, string) {
	m := fmt.Sprintf("%v", row[12])
	kbValue, err := stringToFloat(m)
	if err != nil {
		fmt.Printf("%q\t %g %v\n", m, kbValue, err)
	}

	gbValue := kbValue / GB
	strDisplay := fmt.Sprintf("%v", gbValue)
	strValue := fmt.Sprintf("%sGB", strDisplay[0:3])
	//fmt.Printf("Available Memory Avg: %sGB [%gKB] \n", strDisplay[0:3], kbValue)
	return gbValue, kbValue, strValue
}

func getCpuUtilization(row []interface{}) (float64, string) {
	parsedValue := fmt.Sprintf("%v", row[12])
	value, err := stringToFloat(parsedValue)
	if err != nil {
		fmt.Printf("%q\t %g %v\n", parsedValue, value, err)
	}

	strDisplay := fmt.Sprintf("%v", value)
	strValue := fmt.Sprintf("%s%%", strDisplay[0:4])
	//fmt.Printf("CPU Utilization Avg: %s%% \n", strDisplay[0:4])
	return value, strValue
}

func getLogicalDiskLatency(row []interface{}) (float64, float64, string) {
	//the parsed value is in MS
	parsedValue := fmt.Sprintf("%v", row[12])
	value, err := stringToFloat(parsedValue)
	if err != nil {
		fmt.Printf("%q\t %g %v\n", parsedValue, value, err)
	}
	msValue := value * 1000
	strDisplay := fmt.Sprintf("%v", msValue)
	strValue := fmt.Sprintf("%sms", strDisplay[0:4])
	//fmt.Printf("Logical Disk Latency Avg: %sms [%g] \n", strDisplay[0:4], msValue)
	return msValue, value, strValue
}

func getLogicalDiskIOPs(row []interface{}) (float64, string) {
	//the parsed value is in MS
	parsedValue := fmt.Sprintf("%v", row[12])
	value, err := stringToFloat(parsedValue)
	if err != nil {
		fmt.Printf("%q\t %g %v\n", parsedValue, value, err)
	}

	strDisplay := fmt.Sprintf("%v", value)
	strValue := fmt.Sprintf("%s", strDisplay[0:4])
	//fmt.Printf("Logical Disk IOPs Avg: %s \n", strDisplay[0:4])
	return value, strValue
}

func getDiskBytesPerSeconds(row []interface{}) (float64, float64, string) {

	parsedValue := fmt.Sprintf("%v", row[12])
	value, err := stringToFloat(parsedValue)
	if err != nil {
		fmt.Printf("%q\t %g %v\n", parsedValue, value, err)
	}

	gbValue := value / GB
	strDisplay := fmt.Sprintf("%v", value)
	strValue := fmt.Sprintf("%sGB", strDisplay[0:4])
	//fmt.Printf("Disk Bytes/sec Avg: %sGB [%gKB] \n", strDisplay[0:4], value)
	return gbValue, value, strValue
}

func getBytesSentRate(row []interface{}) (float64, float64, string) {

	parsedValue := fmt.Sprintf("%v", row[12])
	value, err := stringToFloat(parsedValue)
	if err != nil {
		fmt.Printf("%q\t %g %v\n", parsedValue, value, err)
	}

	kbValue := value / KB
	strDisplay := fmt.Sprintf("%v", kbValue)
	//fmt.Printf("Bytes Sent Rate Avg: %sKB [%g] \n", strDisplay[0:4], value)
	strValue := fmt.Sprintf("%sKB", strDisplay[0:4])
	return kbValue, value, strValue
}

func getBytesReceivedRate(row []interface{}) (float64, float64, string) {

	parsedValue := fmt.Sprintf("%v", row[12])
	value, err := stringToFloat(parsedValue)
	if err != nil {
		fmt.Printf("%q\t %g %v\n", parsedValue, value, err)
	}

	kbValue := value / KB
	strDisplay := fmt.Sprintf("%v", kbValue)
	strValue := fmt.Sprintf("%sKB", strDisplay[0:4])
	//fmt.Printf("Bytes Received Rate Avg: %sKB [%g] \n", strDisplay[0:4], value)
	return kbValue, value, strValue
}
