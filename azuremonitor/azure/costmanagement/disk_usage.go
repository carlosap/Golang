package costmanagement

import (
	"encoding/json"
	"fmt"
	"github.com/Go/azuremonitor/azure"
	"github.com/Go/azuremonitor/azure/oauth2"
	"github.com/Go/azuremonitor/azure/subscription"
	"github.com/Go/azuremonitor/common/csv"
	"github.com/Go/azuremonitor/common/httpclient"
	"net/http"
	"strings"
)

type StorageDiskResponse struct {
	Responses []Responses `json:"responses"`
}

type StorageDisk struct {
	Resource azure.Resource `json:"resource"`
	DiskReads     float64 `json:"disk_reads"`
	DiskWrite     float64 `json:"disk_write"`
	DiskReadOperations         float64 `json:"disk_read_operations"`
	DiskWriteOperations     float64 `json:"disk_write_operations"`
	QueueDepth float64 `json:"queue_depth"`
	Responses []Responses `json:"responses"`
}

type StorageDisks []StorageDisk

var (
	mapStorageDisks = make(map[string]StorageDisk)
	Storage_Disks = StorageDisks{}
)


func (sd *StorageDisk) ExecuteRequest(r httpclient.IRequest) {

	//1-Filters Storage Disk only
	requests := sd.getRequests()
	requests.Execute()

	//2-Serializes All Storage Disks and Sets Metrics
	Storage_Disks = sd.parseRequests(requests)

}

func (sd *StorageDisk) GetUrl() string {

	url := azure.QueryUrl
	return url
}
func (sd *StorageDisk) GetMethod() string {
	return httpclient.Methods.POST
}
func (sd *StorageDisk) GetPayload() string {

	resource := subscription.ResourceSubscription{}
	resource.ExecuteRequest(&resource)
	resource.GetManageByResourceId(sd.Resource.ResourceID)
	vmsourceid := resource.GetManageByResourceId(sd.Resource.ResourceID)
	payload := azure.StorageDiskUsagePayload
	payload = strings.ReplaceAll(payload, "{{startdate}}", StartDate)
	payload = strings.ReplaceAll(payload, "{{enddate}}", EndDate)
	payload = strings.ReplaceAll(payload, "{{subscriptionid}}", configuration.AccessToken.SubscriptionID)
	payload = strings.ReplaceAll(payload, "{{resourcegroup}}", sd.Resource.ResourceGroup)
	payload = strings.ReplaceAll(payload, "{{resourceid}}",vmsourceid )
	return payload
}
func (sd *StorageDisk) GetHeader() http.Header {
	at := oauth2.AccessToken{}
	at.ExecuteRequest(&at)
	token := fmt.Sprintf("Bearer %s", at.AccessToken)
	var header = http.Header{}
	header.Add("Authorization", token)
	header.Add("Accept", "application/json")
	header.Add("Content-Type", "application/json")
	return header
}
func (sd *StorageDisk) Print() {

	if len(Storage_Disks) > 0 {
		fmt.Printf("Usage Report Storage Disk:\n")
		fmt.Println("---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------")
		fmt.Println("Resource Group,ResourceID,Service Name,Resource Type,Resource Location,Location Prefix,Consumption Type,Meter,Cost," +
			"OS Disk Read Bytes/sec Avg,OS Disk Write Bytes/sec Avg,OS Disk Read Operations/Sec Avg,OS Disk Write Operations/Sec Avg,OS Disk Queue Depth")
		fmt.Println("---------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------")
		for _, item := range Storage_Disks {
			fmt.Printf("%s,%s,%s,%s,%s,%s,%s,%s,$%f,%f,%f,%f,%f,%f\n",item.Resource.ResourceGroup, item.Resource.ResourceID, item.Resource.Service,
				item.Resource.ServiceType, item.Resource.Location,item.Resource.LocationPrefix, item.Resource.ChargeType, item.Resource.Meter, item.Resource.Cost,
				item.DiskReads, item.DiskWrite,item.DiskReadOperations, item.DiskWriteOperations, item.QueueDepth)
		}
	} else {
		fmt.Printf("-\n\n\n")
	}
}

//---------------Other Functions --------------------------------------------------------------
func (sd *StorageDisk) getRequests() httpclient.Requests {
	requests := httpclient.Requests{}
	if len(Resources) > 0 {
		for index, resource := range Resources {
			if resource.Service == "storage" && resource.ServiceType == "disks" && resource.ChargeType == "usage" && resource.Cost > 0.0 {
				rName := "sd_" + resource.ResourceID + "_" + fmt.Sprintf("%d", index)
				sd.Resource = resource
				request := httpclient.Request{
					Name:    rName,
					Header:  sd.GetHeader(),
					Payload: sd.GetPayload(),
					Url:     sd.GetUrl(),
					Method:  sd.GetMethod(),
					IsCache: false,
				}
				mapStorageDisks[rName] = *sd
				requests = append(requests, request)
			}
		}
	}
	return requests
}
func (sd *StorageDisk) parseRequests(requests httpclient.Requests) StorageDisks {
	sds := StorageDisks{}
	var sdResponse BatchResponse
	for _, item := range requests {
		bData := item.GetResponse()
		if len(bData) > 0 {
			err := json.Unmarshal(bData, &sdResponse)
			if err != nil {
				fmt.Printf("error: failed to unmarshal - %v\n\n", err)
			}
			//fmt.Printf("data: %s\n\n", string(bData))
			sdRef, hasKey := mapStorageDisks[item.Name]
			if hasKey {
				sd.Resource = sdRef.Resource
				sd.Responses = sdResponse.Responses
				sd.setUsageValue()
				sds = append(sds, *sd)
			}
		}
	}
	return sds
}
func (sd *StorageDisk) setUsageValue() {

	if len(sd.Responses) > 0 {
		for _, response := range sd.Responses {
			if len(response.Content.Value) > 0 {
				//fmt.Printf("value: %v\n",response.Content.Value)
				for _, valueItem := range response.Content.Value {
					switch valueItem.Name.Value {
					//Bytes/Sec read from a single disk during monitoring period for OS disk
					case "OS Disk Read Bytes/sec":
						sd.DiskReads = valueItem.Timeseries[0].Data[0].Average
					case "OS Disk Write Bytes/sec":
						sd.DiskWrite = valueItem.Timeseries[0].Data[0].Average
					case "OS Disk Read Operations/Sec":
						sd.DiskReadOperations = valueItem.Timeseries[0].Data[0].Average
					case "OS Disk Write Operations/Sec":
						sd.DiskWriteOperations = valueItem.Timeseries[0].Data[0].Average
					case "OS Disk Queue Depth":
						sd.QueueDepth = valueItem.Timeseries[0].Data[0].Average
					}
				}
			}
		}
	}
}
func (sd *StorageDisk) WriteCSV(filepath string) {

	if len(Storage_Disks) > 0 {
		var matrix [][]string
		rec := []string{"Resource Group","ResourceID","Service Name","Resource Type","Resource Location","Location Prefix","Consumption Type","Meter","Cost",
			"OS Disk Read Bytes/sec Avg","OS Disk Write Bytes/sec Avg","OS Disk Read Operations/Sec Avg","OS Disk Write Operations/Sec Avg","OS Disk Queue Depth"}
		matrix = append(matrix, rec)
		for _, item := range Storage_Disks {
			var rec []string
			rec = append(rec, item.Resource.ResourceGroup)
			rec = append(rec, item.Resource.ResourceID)
			rec = append(rec, item.Resource.Service)
			rec = append(rec, item.Resource.ServiceType)
			rec = append(rec, item.Resource.Location)
			rec = append(rec, item.Resource.LocationPrefix)
			rec = append(rec, item.Resource.ChargeType)
			rec = append(rec, item.Resource.Meter)
			rec = append(rec, fmt.Sprintf("%f",item.Resource.Cost))

			rec = append(rec, fmt.Sprintf("%f",item.DiskReads))
			rec = append(rec, fmt.Sprintf("%f",item.DiskWrite))
			rec = append(rec, fmt.Sprintf("%f",item.DiskReadOperations))
			rec = append(rec, fmt.Sprintf("%f",item.DiskWriteOperations))
			rec = append(rec, fmt.Sprintf("%f",item.QueueDepth))
			matrix = append(matrix, rec)
		}
		csv.SaveMatrixToFile(filepath, matrix)
	}
}
