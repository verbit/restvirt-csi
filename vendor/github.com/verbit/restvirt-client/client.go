package restvirt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"
)

type Client struct {
	Host     string
	Username string
	Password string
}

type ErrorMessage struct {
	Error string
}

type Domain struct {
	UUID      string `json:"uuid,omitempty"`
	Name      string `json:"name"`
	VCPU      int    `json:"vcpu"`
	MemoryMiB int    `json:"memory"`
	PrivateIP string `json:"private_ip"`
	UserData  string `json:"user_data,omitempty"`
}

type PortForwarding struct {
	SourcePort uint16 `json:"source_port"`
	TargetPort uint16 `json:"target_port"`
	TargetIP   string `json:"target_ip"`
}

type VolumeAttachment struct {
	DiskAddress string `json:"disk_address"`
}

func NewClient(host string, username string, password string) (*Client, error) {
	c := Client{
		Host:     host,
		Username: username,
		Password: password,
	}

	return &c, nil
}

func (c *Client) CreateDomain(domain Domain) (string, error) {
	u, err := url.Parse(c.Host)
	_ = err
	u.Path = path.Join(u.Path, "domains")
	domainJSON, err := json.Marshal(domain)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(u.String(), "application/json", bytes.NewBuffer(domainJSON))
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		var errorMessage ErrorMessage
		err := json.NewDecoder(resp.Body).Decode(&errorMessage)
		if err != nil {
			return "", err
		}
		return "", fmt.Errorf("error creating domain: %s", errorMessage.Error)
	}

	var uuidDomain Domain
	err = json.NewDecoder(resp.Body).Decode(&uuidDomain)
	if err != nil {
		return "", err
	}

	return uuidDomain.UUID, nil
}

func (c *Client) GetDomain(id string) (*Domain, error) {
	u, err := url.Parse(c.Host)
	_ = err
	u.Path = path.Join(u.Path, "domains", id)
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(`domain with name "%s" not found`, id)
	}

	var domain Domain
	err = json.NewDecoder(resp.Body).Decode(&domain)
	if err != nil {
		return nil, err
	}

	return &domain, nil
}

func (c *Client) DeleteDomain(id string) error {
	u, err := url.Parse(c.Host)
	_ = err
	u.Path = path.Join(u.Path, "domains", id)
	request, err := http.NewRequest("DELETE", u.String(), nil)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		var errorMessage ErrorMessage
		err := json.NewDecoder(resp.Body).Decode(&errorMessage)
		if err != nil {
			return err
		}
		return fmt.Errorf("error deleting domain: %s", errorMessage.Error)
	}

	return nil
}

func (c *Client) CreatePortForwarding(forwarding PortForwarding) (string, error) {
	u, err := url.Parse(c.Host)
	_ = err
	u.Path = path.Join(u.Path, "forwardings")
	forwardingJSON, err := json.Marshal(forwarding)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(u.String(), "application/json", bytes.NewBuffer(forwardingJSON))
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		var errorMessage ErrorMessage
		err := json.NewDecoder(resp.Body).Decode(&errorMessage)
		if err != nil {
			return "", err
		}
		return "", fmt.Errorf("error creating port forwarding: %s", errorMessage.Error)
	}

	return strconv.Itoa(int(forwarding.SourcePort)), nil
}

func (c *Client) GetPortForwarding(id string) (*PortForwarding, error) {
	u, err := url.Parse(c.Host)
	_ = err
	u.Path = path.Join(u.Path, "forwardings", id)
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(`port forwarding with name "%s" not found`, id)
	}

	var forwarding PortForwarding
	err = json.NewDecoder(resp.Body).Decode(&forwarding)
	if err != nil {
		return nil, err
	}

	return &forwarding, nil
}

func (c *Client) DeletePortForwarding(id string) error {
	u, err := url.Parse(c.Host)
	_ = err
	u.Path = path.Join(u.Path, "forwardings", id)
	request, err := http.NewRequest("DELETE", u.String(), nil)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		var errorMessage ErrorMessage
		err := json.NewDecoder(resp.Body).Decode(&errorMessage)
		if err != nil {
			return err
		}
		return fmt.Errorf("error deleting port forwarding: %s", errorMessage.Error)
	}

	return nil
}

func (c *Client) CreateAttachment(domainID string, volumeID string) (*VolumeAttachment, error) {
	u, err := url.Parse(c.Host)
	_ = err
	u.Path = path.Join(u.Path, "domains", domainID, "volumes", volumeID)
	request, err := http.NewRequest("PUT", u.String(), nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		var errorMessage ErrorMessage
		err := json.NewDecoder(resp.Body).Decode(&errorMessage)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("error attaching volume: %s", errorMessage.Error)
	}

	var attachment VolumeAttachment
	err = json.NewDecoder(resp.Body).Decode(&attachment)
	if err != nil {
		return nil, err
	}

	return &attachment, nil
}

func (c *Client) GetAttachment(domainID string, volumeID string) (*VolumeAttachment, error) {
	u, err := url.Parse(c.Host)
	_ = err
	u.Path = path.Join(u.Path, "domains", domainID, "volumes", volumeID)
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(`volume attachment "%s" -> "%s" not found`, domainID, volumeID)
	}

	var attachment VolumeAttachment
	err = json.NewDecoder(resp.Body).Decode(&attachment)
	if err != nil {
		return nil, err
	}

	return &attachment, nil
}

func (c *Client) DeleteAttachment(domainID string, volumeID string) error {
	u, err := url.Parse(c.Host)
	_ = err
	u.Path = path.Join(u.Path, "domains", domainID, "volumes", volumeID)
	request, err := http.NewRequest("DELETE", u.String(), nil)
	if err != nil {
		return err
	}

	client := &http.Client{}
	resp, err := client.Do(request)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		var errorMessage ErrorMessage
		err := json.NewDecoder(resp.Body).Decode(&errorMessage)
		if err != nil {
			return err
		}
		return fmt.Errorf("error detaching volume: %s", errorMessage.Error)
	}

	return nil
}
