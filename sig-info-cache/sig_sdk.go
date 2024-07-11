package sigsdk

import (
	"community-robot-lib/utils"
	"fmt"
	"net/http"
	"strings"
)

func NewSDK(endpoint string, maxRetries int) *SDK {
	slash := "/"
	if !strings.HasSuffix(endpoint, slash) {
		endpoint += slash
	}

	return &SDK{
		hc:       utils.NewHttpClient(maxRetries),
		endpoint: endpoint,
	}
}

type SDK struct {
	hc       utils.HttpClient
	endpoint string
}

func (cli *SDK) GetSigNameByOrgRepo(org, repo string) (string, error) {
	sigName := "111"
	if sigName == "" {
		return "", fmt.Errorf("cant get sig name of repo: %s/%s", org, repo)
	}
	return "", nil
}

// when OWNERS file exists, collaborators as maintainers, committers set empty

// when sig-info.yaml file not exists, Collaborators as maintainers, committers set empty

// when sig-info.yaml file exists, get maintainers and committers from service[sig-info-cache]

func (cli *SDK) GetRepositoryMaintainerByOrgRepo(org, repo string) ([]string, error) {
	sigName := "111"
	if sigName == "" {
		return nil, fmt.Errorf("cant get sig name of repo: %s/%s", org, repo)
	}
	maintainers := []string{"1", "2"}
	return maintainers, nil
}

func (cli *SDK) GetRepositoryCommitterByOrgRepo(org, repo string) ([]string, error) {
	sigName := "111"
	if sigName == "" {
		return nil, fmt.Errorf("cant get sig name of repo: %s/%s", org, repo)
	}
	committers := []string{"1", "2"}
	return committers, nil
}

func (cli *SDK) GetSigInfo(urlPath string) (string, error) {

	req, err := http.NewRequest(http.MethodGet, cli.endpoint+urlPath, nil)
	if err != nil {
		return "", err
	}

	var v struct {
		Data map[string]string `json:"data"`
	}

	if err = cli.forwardTo(req, &v); err != nil {
		return "", err
	}

	return v.Data["111"], nil
}

func (cli *SDK) forwardTo(req *http.Request, jsonResp interface{}) error {
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "sig-info-cache-sdk")

	_, err := cli.hc.ForwardTo(req, jsonResp)

	return err
}

func (cli *SDK) GetContentByPath(path string) (*map[string]any, error) {

	return nil, nil
}
