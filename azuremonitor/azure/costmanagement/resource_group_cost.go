package costmanagement

import (
	"encoding/json"
	"fmt"
	"github.com/Go/azuremonitor/azure/batch"
	"github.com/Go/azuremonitor/azure/oauth2"
	"github.com/Go/azuremonitor/common/csv"
	"github.com/Go/azuremonitor/common/errors"
	"github.com/Go/azuremonitor/common/filesystem"
	"github.com/Go/azuremonitor/common/httpclient"
	c "github.com/Go/azuremonitor/config"
	"net/http"
	"strings"
)

type ResourceGroupCost struct {
	ID                string      `json:"id"`
	Name              string      `json:"name"`
	ResourceGroupName string      `json:"resourcegroupname"`
	Type              string      `json:"type"`
	Location          interface{} `json:"location"`
	Sku               interface{} `json:"sku"`
	ETag              interface{} `json:"eTag"`
	Properties        struct {
		NextLink interface{} `json:"nextLink"`
		Columns  []struct {
			Name string `json:"name"`
			Type string `json:"type"`
		} `json:"columns"`
		Rows [][]interface{} `json:"rows"`
	} `json:"properties"`
}
type ResourceGroupCosts []ResourceGroupCost

var (
	configuration    c.CmdConfig
	StartDate        string
	EndDate          string
	SaveCsv bool
	IgnoreZeroCost bool
	csvRgcReportName = "resource_group_cost.csv"
)

func init(){
	configuration, _ = c.GetCmdConfig()
}

func (rgc *ResourceGroupCost) getRequests() httpclient.Requests {
	requests := httpclient.Requests{}
	rgl := batch.ResourceGroupList{}
	rgl.ExecuteRequest(&rgl)

	for _, item := range rgl.ToList() {
		rgc.Name = item
		rgc.ResourceGroupName = item
		request := httpclient.Request{}
		request.Name = item
		request.Header = rgc.GetHeader()
		request.Payload = rgc.GetPayload()
		request.Url = rgc.GetUrl()
		request.Method = rgc.GetMethod()
		request.IsCache = true
		requests = append(requests, request)
	}
	return requests
}


func (rgc *ResourceGroupCost) ExecuteRequest(r httpclient.IRequest) {

	requests := rgc.getRequests()
	errorItems := requests.Execute()
	errors.IfErrorsPrintThem(errorItems)

	if SaveCsv {
		filesystem.RemoveFile(csvRgcReportName)
		rgc.PrintHeader()
	}

	for _, item := range requests {
		if len(item.GetResponse()) > 0 {
			bData := item.GetResponse()
			//fmt.Printf("%s\n %s\n\n", item.Name, string(bData))
			if len(bData) > 0 {
				_ = json.Unmarshal(bData, rgc)
				rgc.ResourceGroupName = item.Name
				rgc.Print()
				rgc.writeCSV()
			}
		}
	}
}

func (rgc *ResourceGroupCost) GetUrl() string {

	url := strings.Replace(configuration.ResourceGroupCost.URL, "{{subscriptionID}}", configuration.AccessToken.SubscriptionID, 1)
	url = strings.Replace(url, "{{resourceGroup}}", rgc.Name, 1)
	return url
}
func (rgc *ResourceGroupCost) GetMethod() string {
	return httpclient.Methods.POST
}
func (rgc *ResourceGroupCost) GetPayload() string {

	if StartDate == "" || EndDate == "" {
		fmt.Println("StartDate and EndDate are Required in the payload. -", rgc.Name)
		return ""
	}

	url := fmt.Sprintf("{\"type\": \"ActualCost\",\"dataSet\": {\"granularity\": \"None\","+
		"\"aggregation\": {\"totalCost\": {\"name\": \"Cost\",\"function\": \"Sum\"},"+
		"\"totalCostUSD\": {\"name\": \"CostUSD\",\"function\": \"Sum\"}},"+
		"\"grouping\": [{\"type\": \"Dimension\",\"name\": \"ResourceId\"},"+
		" {\"type\": \"Dimension\",\"name\": \"ResourceType\"}, {\"type\": \"Dimension\",\"name\": \"ResourceLocation\"}, "+
		"{\"type\": \"Dimension\",\"name\": \"ChargeType\"}, {\"type\": \"Dimension\",\"name\": \"ResourceGroupName\"}, "+
		"{\"type\": \"Dimension\",\"name\": \"PublisherType\"}, {\"type\": \"Dimension\",\"name\": \"ServiceName\"}, "+
		"{\"type\": \"Dimension\",\"name\": \"Meter\"}],\"include\": [\"Tags\"]},\"timeframe\": \"Custom\","+
		"\"timePeriod\": {"+
		"\"from\": \"%sT00:00:00+00:00\","+
		"\"to\": \"%sT23:59:59+00:00\"}}",
		StartDate,
		EndDate,
	)
	return url
}
func (rgc *ResourceGroupCost) GetHeader() http.Header {
	at := oauth2.AccessToken{}
	at.ExecuteRequest(&at)
	token := fmt.Sprintf("Bearer %s", at.AccessToken)
	var header = http.Header{}
	header.Add("Authorization", token)
	header.Add("Accept", "application/json")
	header.Add("Content-Type", "application/json")
	return header
}
func (rgc *ResourceGroupCost) Print() {
	fmt.Printf("%s\n", rgc.ResourceGroupName)
	for i := 0; i < len(rgc.Properties.Rows); i++ {
		row := rgc.Properties.Rows[i]
		if len(row) > 0 {
			costUSD := fmt.Sprintf("%v", row[1])
			resourceId := fmt.Sprintf("%v", row[2])
			resourceType := fmt.Sprintf("%v", row[3])
			resourceLocation := fmt.Sprintf("%v", row[4])
			chargeType := fmt.Sprintf("%v", row[5])
			serviceName := fmt.Sprintf("%v", row[8])
			meter := fmt.Sprintf("%v", row[9])

			//format cost
			if len(costUSD) > 5 {
				costUSD = costUSD[0:5]
			}

			if IgnoreZeroCost {
				if costUSD == "0" {
					continue
				}
			}

			//remove path
			if strings.Contains(resourceType, "/") {
				pArray := strings.Split(resourceType, "/")
				resourceType = pArray[len(pArray)-1]
			}

			if strings.Contains(resourceId, "/") {
				pArray := strings.Split(resourceId, "/")
				resourceId = pArray[len(pArray)-1]
			}

			fmt.Printf("\t%s,%s,%s,%s,%s,%s,$%s\n", resourceId, serviceName, resourceType, resourceLocation, chargeType, meter, costUSD)
		}
	}
}
func (rgc *ResourceGroupCost) PrintHeader() {
	fmt.Println("Consumption Report:")
	fmt.Println("-------------------------------------------------------------------------------------------------------------------------------")
	fmt.Println("Resource Group,ResourceID,Service Name,Resource Type,Resource Location,Consumption Type,Meter,Cost")
	fmt.Println("-------------------------------------------------------------------------------------------------------------------------------")
	if SaveCsv {
		var matrix [][]string
		rec := []string{"Resource Group", "ResourceID", "Service Name", "Resource Type", "Resource Location", "Consumption Type", "Meter", "Cost"}
		matrix = append(matrix, rec)
		csv.SaveMatrixToFile(csvRgcReportName, matrix)
	}
}
func (rgc ResourceGroupCost) writeCSV() {

	if SaveCsv {
		var matrix [][]string
		for i := 0; i < len(rgc.Properties.Rows); i++ {
			row := rgc.Properties.Rows[i]
			if len(row) > 0 {
				costUSD := fmt.Sprintf("%v", row[1])
				resourceId := fmt.Sprintf("%v", row[2])
				resourceType := fmt.Sprintf("%v", row[3])
				resourceLocation := fmt.Sprintf("%v", row[4])
				chargeType := fmt.Sprintf("%v", row[5])
				serviceName := fmt.Sprintf("%v", row[8])
				meter := fmt.Sprintf("%v", row[9])

				//format cost
				if len(costUSD) > 5 {
					costUSD = costUSD[0:5]

				}

				if IgnoreZeroCost {
					if costUSD == "0" {
						continue
					}
				}

				//remove path
				if strings.Contains(resourceType, "/") {
					pArray := strings.Split(resourceType, "/")
					resourceType = pArray[len(pArray)-1]
				}

				if strings.Contains(resourceId, "/") {
					pArray := strings.Split(resourceId, "/")
					resourceId = pArray[len(pArray)-1]
				}

				var rec []string
				rec = append(rec, rgc.ResourceGroupName)
				rec = append(rec, resourceId)
				rec = append(rec, serviceName)
				rec = append(rec, resourceType)
				rec = append(rec, resourceLocation)
				rec = append(rec, chargeType)
				rec = append(rec, meter)
				rec = append(rec, costUSD)
				matrix = append(matrix, rec)
			}
		}
		csv.SaveMatrixToFile(csvRgcReportName, matrix)
	}
}
