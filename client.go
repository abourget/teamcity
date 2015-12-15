package teamcity

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
)

type Client struct {
	HTTPClient *http.Client
	username   string
	password   string
	host       string
}

func New(host, username, password string) *Client {
	return &Client{
		HTTPClient: http.DefaultClient,
		username:   username,
		password:   password,
		host:       host,
	}
}

func (c *Client) QueueBuild(buildTypeID string, branchName string, properties map[string]string) (*Build, error) {
	jsonQuery := struct {
		BuildTypeID string `json:"buildTypeId,omitempty"`
		Properties  struct {
			Property []oneProperty `json:"property,omitempty"`
		} `json:"properties"`
		BranchName string `json:"branchName,omitempty"`
	}{}
	jsonQuery.BuildTypeID = buildTypeID
	if branchName != "" {
		jsonQuery.BranchName = fmt.Sprintf("refs/heads/%s", branchName)
	}
	for k, v := range properties {
		jsonQuery.Properties.Property = append(jsonQuery.Properties.Property, oneProperty{k, v})
	}

	build := &Build{}
	err := c.doRequest("POST", "/httpAuth/app/rest/buildQueue", jsonQuery, &build)
	if err != nil {
		return nil, err
	}

	build.convertInputs()

	return build, nil
}

func (c *Client) SearchBuild(locator string) ([]*Build, error) {
	path := fmt.Sprintf("/httpAuth/app/rest/builds/?locator=%s&fields=count,build(*,tags(tag),triggered(*),properties(property))", locator)

	respStruct := struct {
		Count int
		Build []*Build
	}{}
	err := c.doRequest("GET", path, nil, &respStruct)
	if err != nil {
		return nil, err
	}

	for _, build := range respStruct.Build {
		build.convertInputs()
	}

	return respStruct.Build, nil
}

func (c *Client) GetBuild(buildID string) (*Build, error) {
	path := fmt.Sprintf("/httpAuth/app/rest/builds/id:%s", buildID)
	var build *Build
	err := c.doRequest("GET", path, nil, &build)
	if err != nil {
		return nil, err
	}

	if build == nil {
		return nil, errors.New("build not found")
	}

	return build, nil
}

func (c *Client) GetBuildProperties(buildID string) (map[string]string, error) {
	path := fmt.Sprintf("/httpAuth/app/rest/builds/id:%s/resulting-properties", buildID)

	var response struct {
		Properties []struct {
			Name  string `json:"name"`
			Value string `json:"value"`
		} `json:"property"`
	}
	err := c.doRequest("GET", path, nil, &response)
	if err != nil {
		return nil, err
	}

	m := make(map[string]string)
	for _, prop := range response.Properties {
		m[prop.Name] = prop.Value
	}
	return m, nil
}

func (c *Client) GetChanges(path string) ([]Change, error) {
	var changes struct {
		Change []Change
	}

	err := c.doRequest("GET", path, nil, &changes)
	if err != nil {
		return nil, err
	}

	if changes.Change == nil {
		return nil, errors.New("changes not found")
	}

	return changes.Change, nil
}

func (c *Client) CancelBuild(buildID int64, comment string) error {
	body := map[string]interface{}{
		"buildCancelRequest": map[string]interface{}{
			"comment":       comment,
			"readIntoQueue": true,
		},
	}
	return c.doRequest("POST", fmt.Sprintf("/httpAuth/app/rest/id:%d", buildID), body, nil)
}

func (c *Client) doRequest(method string, path string, data interface{}, v interface{}) error {
	authlessUrl := fmt.Sprintf("%s%s", c.host, path)

	fmt.Printf("Sending request to https://%s\n", authlessUrl)

	var body io.Reader
	if data != nil {
		jsonReq, err := json.Marshal(data)
		if err != nil {
			return fmt.Errorf("marshaling data: %s", err)
		}

		body = bytes.NewBuffer(jsonReq)
	}

	req, _ := http.NewRequest(method, fmt.Sprintf("https://%s:%s@%s", c.username, c.password, authlessUrl), body)
	req.Header.Add("Accept", "application/json")

	if body != nil {
		req.Header.Add("Content-Type", "application/json")
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	jsonCnt, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	//ioutil.WriteFile(fmt.Sprintf("/tmp/mama-%s.json", time.Now().Format("15:04:05.000")), jsonCnt, 0644)

	if v != nil {
		err = json.Unmarshal(jsonCnt, &v)
		if err != nil {
			return fmt.Errorf("json unmarshal: %s (%q)", err, truncate(string(jsonCnt), 1000))
		}
	}

	return nil
}

func truncate(s string, l int) string {
	if len(s) > l {
		return s[:l]
	}
	return s
}
