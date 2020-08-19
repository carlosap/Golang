package cmd

import (
	"encoding/json"
	"fmt"
	"github.com/Go/azuremonitor/db/cache"
	"io/ioutil"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
)

type Request struct {
	Name         string
	Url          string
	Method       string
	Payload      string
	Header       http.Header
	IsCache      bool
	ValueType    interface{}
}
type Requests []Request

type RequestMethods struct {
	POST string
	GET string
}

func (r Requests) Execute() []string {
	var errorLock sync.Mutex
	var updateLock sync.Mutex
	errors := make([]string, 0)
	var wg sync.WaitGroup
	semaphore := make(chan int, parallel)
	c := &cache.Cache{}
	for _, request := range r {
		wg.Add(1)
		go func(r Request) {
			defer wg.Done()
			semaphore <- 1
			c.Delete(r.Name)
			if r.IsCache {
				//1- fresh request
				cKey := fmt.Sprintf("%s_%s_%s_%s_%s", configuration.AccessToken.SubscriptionID, r.Name,r.Url, startDate, endDate)
				//fmt.Printf("the key is: %s\n", cKey)
				cHashVal := c.Get(cKey)
				if len(cHashVal) <= 0 {
					body, err := makeRequest(r)
					if err != nil {
						errorLock.Lock()
						defer errorLock.Unlock()
						errors = append(errors,
							fmt.Sprintf("%s error: %s", r.Url, err))
					} else {
						updateLock.Lock()
						defer updateLock.Unlock()
						c.Set(r.Name, string(body))
						_ = json.Unmarshal(body, r.ValueType)
						_ = saveCache(cKey, r.ValueType)
					}
				} else {
					//2- corrupted files
					//fmt.Printf("the hashvalue: %s\n", cHashVal)
					err := LoadFromCache(cKey, r.ValueType)
					if err != nil {
						body, err := makeRequest(r)
						if err != nil {
							errorLock.Lock()
							defer errorLock.Unlock()
							errors = append(errors,
								fmt.Sprintf("%s error: %s", r.Url, err))
						} else {
							updateLock.Lock()
							defer updateLock.Unlock()
							c.Set(r.Name, string(body))
							_ = json.Unmarshal(body, r.ValueType)
							_ = saveCache(cKey, r.ValueType)
						}
					}
					path := filepath.Join("cache", cHashVal)
					body, _ := loadFile(path)
					c.Set(r.Name, string(body))
				}
			} else {
				// 3- no cached
				body, err := makeRequest(r)
				if err != nil {
					errorLock.Lock()
					defer errorLock.Unlock()
					errors = append(errors,
						fmt.Sprintf("%s error: %s", r.Url, err))
				} else {
					updateLock.Lock()
					defer updateLock.Unlock()
					c.Set(r.Name, string(body))
				}
			}

			<-semaphore
		}(request)
	}
	wg.Wait()

	return errors
}

func makeRequest(r Request) ([]byte, error) {
	client := &http.Client{}
	var body []byte
	payload := strings.NewReader(r.Payload)
	req, err := http.NewRequest(r.Method, r.Url, payload)
	if err != nil {
		return body,err
	}

	//need header
	if r.Header != nil {
		req.Header = r.Header
	}

	res, err := client.Do(req)
	if err != nil {
		return body, err
	}

	defer res.Body.Close()
	body, err = ioutil.ReadAll(res.Body)
	return body, err
}

func (r Request) GetResponse() []byte {
	c := &cache.Cache{}
	body := c.Get(r.Name)
	return []byte(body)
}

//func (r Request) GetValue() interface{} {
//	c := &cache.Cache{}
//	body := c.Get(r.Name)
//	err := json.Unmarshal([]byte(body), &r.Value)
//	if err != nil {
//		fmt.Println("unmarshal body response: ", err)
//	}
//	return r.Value
//}


func (r Request) Execute() []string {
	var requests = Requests{}
	requests = append(requests, r)
	errors := requests.Execute()
	return errors
}


