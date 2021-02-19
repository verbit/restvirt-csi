package restvirt

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"path"
)

type Volume struct {
	ID   string `json:"id,omitempty"`
	Name string `json:"name"`
	Size int64  `json:"size"`
}

func (c *Client) CreateVolume(volume Volume) (string, error) {
	u, err := url.Parse(c.Host)
	_ = err
	u.Path = path.Join(u.Path, "volumes")
	volumeJSON, err := json.Marshal(volume)
	if err != nil {
		return "", err
	}

	resp, err := http.Post(u.String(), "application/json", bytes.NewBuffer(volumeJSON))
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		var errorMessage ErrorMessage
		err := json.NewDecoder(resp.Body).Decode(&errorMessage)
		if err != nil {
			return "", err
		}
		return "", fmt.Errorf("error creating volume: %s", errorMessage.Error)
	}

	var vol Volume
	err = json.NewDecoder(resp.Body).Decode(&vol)
	if err != nil {
		return "", err
	}

	return vol.ID, nil
}

func (c *Client) GetVolume(id string) (*Volume, error) {
	u, err := url.Parse(c.Host)
	_ = err
	u.Path = path.Join(u.Path, "volumes", id)
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(`volume with id "%s" not found`, id)
	}

	var volume Volume
	err = json.NewDecoder(resp.Body).Decode(&volume)
	if err != nil {
		return nil, err
	}

	return &volume, nil
}

func (c *Client) GetVolumes() ([]Volume, error) {
	u, err := url.Parse(c.Host)
	_ = err
	u.Path = path.Join(u.Path, "volumes")
	resp, err := http.Get(u.String())
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf(`couldn't list volumes`)
	}

	var volumes map[string][]Volume
	err = json.NewDecoder(resp.Body).Decode(&volumes)
	if err != nil {
		return nil, err
	}

	return volumes["volumes"], nil
}

func (c *Client) DeleteVolume(id string) error {
	u, err := url.Parse(c.Host)
	_ = err
	u.Path = path.Join(u.Path, "volumes", id)
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
