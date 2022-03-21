package resources

import (
	"encoding/json"
	"fmt"
	"github.com/MikelAlejoBR/fake-availability-status-producer/config"
	"io"
	"net/http"
)

var HttpClient = &http.Client{}

type ResponseId struct {
	Id string `json:"id"`
}

type ResponseDataIds struct {
	Data []ResponseId `json:"data"`
	Id []string `json:"id"`
}

func SourceExists(sourceId string, xRhIdentity string) (bool, error){
	sourceEndpointPath := fmt.Sprintf("%s/sources/%s", config.SourcesApiUrl, sourceId)

	sourceExistsReq, err := http.NewRequest(http.MethodGet, sourceEndpointPath, nil)
	if err != nil {
		return false, fmt.Errorf(`[source_id: %s][path: %s] could not send request: %s`, sourceId, sourceEndpointPath, err)
	}

	sourceExistsReq.Header.Add("x-rh-identity", xRhIdentity)

	sourceExistsRes, err := HttpClient.Do(sourceExistsReq)
	if err != nil {
		return false, fmt.Errorf(`[source_id: %s][path: %s] could not check if source exists: %s`, sourceId, sourceEndpointPath, err)
	}

	return sourceExistsRes.StatusCode == http.StatusOK, nil
}

func GetApplicationIds(sourceId string, xRhIdentity string) (*ResponseDataIds, error) {
	applicationsEndpointPath := fmt.Sprintf("%s/sources/%s/applications", config.SourcesApiUrl, sourceId)
	getAppsReq, err := http.NewRequest(http.MethodGet, applicationsEndpointPath, nil)
	if err != nil {
		return nil, fmt.Errorf(`could not create the request to fetch the applications of source "%s": %s`, sourceId, err)
	}

	getAppsReq.Header.Add("x-rh-identity", xRhIdentity)

	getAppsRes, err := HttpClient.Do(getAppsReq)
	if err != nil {
		return nil, fmt.Errorf(`could not fetch applications for source "%s": %s`, sourceId, err)
	}

	if getAppsRes != nil && (getAppsRes.StatusCode != http.StatusOK) {
		return nil, fmt.Errorf(`could not fetch applications for source "%s". Expecting status code "%d", got "%d"`, sourceId, http.StatusOK, getAppsRes.StatusCode)
	}

	body, err := io.ReadAll(getAppsRes.Body)
	if err != nil {
		return nil, fmt.Errorf(`could not read request's body. Body: %s, error: %s`, getAppsRes.Body, err)
	}

	err = getAppsRes.Body.Close()
	if err != nil {
		return nil, fmt.Errorf(`could not close request's body: %s`, err)
	}

	var appIds ResponseDataIds
	err = json.Unmarshal(body, &appIds)
	if err != nil {
		return nil, fmt.Errorf(`could not unmarshal application ids: %s`, err)
	}

	return &appIds, nil
}

func GetResourceIds(sourceId string, path string, xRhIdentity string) (*ResponseDataIds, error) {
	resourcesEndpointPath := fmt.Sprintf("%s/sources/%s/%s", config.SourcesApiUrl, sourceId, path)
	getResourcesReq, err := http.NewRequest(http.MethodGet, resourcesEndpointPath, nil)
	if err != nil {
		return nil, fmt.Errorf(`[source_id: %s][path: %s] could not create the request to fetch the resources: %s`, sourceId, resourcesEndpointPath, err)
	}

	getResourcesReq.Header.Add("x-rh-identity", xRhIdentity)

	getResourcesRes, err := HttpClient.Do(getResourcesReq)
	if err != nil {
		return nil, fmt.Errorf(`[source_id: %s][path: %s] could not fetch the resources: %s`, sourceId, resourcesEndpointPath, err)
	}

	body, err := io.ReadAll(getResourcesRes.Body)
	if err != nil {
		return nil, fmt.Errorf(`[source_id: %s][path: %s] could not read the request's body: %s`, sourceId, resourcesEndpointPath, err)
	}

	err  = getResourcesRes.Body.Close()
	if err != nil {
		return nil, fmt.Errorf(`[source_id: %s][path: %s] could not close the request's body: %s`, sourceId, resourcesEndpointPath, err)
	}

	var resourceIds ResponseDataIds
	err = json.Unmarshal(body, &resourceIds)
	if err != nil {
		return nil, fmt.Errorf(`[source_id: %s][path: %s] could not unmarshal resource ids: %s`, sourceId, resourcesEndpointPath, err)
	}

	return &resourceIds, nil
}
