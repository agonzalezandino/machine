package rackspace

import (
	"errors"

	"github.com/rackspace/gophercloud/openstack/compute/v2/servers"
	"github.com/rackspace/gophercloud/openstack/networking/v2/networks"
	"github.com/rackspace/gophercloud/pagination"
)

const (
	PublicID    string = "00000000-0000-0000-0000-000000000000"
	PublicName  string = "public"
	PrivateID   string = "11111111-1111-1111-1111-111111111111"
	PrivateName string = "private"
)

type raxNetwork struct {
	UUID string
	Name string
}

var (
	PublicNetwork  raxNetwork = raxNetwork{UUID: PublicID, Name: PublicName}
	PrivateNetwork raxNetwork = raxNetwork{UUID: PrivateID, Name: PrivateName}
)

type raxNetworks struct {
	networks []raxNetwork
}

func (rn *raxNetworks) getName(id string) (name string, err error) {
	for _, _network := range rn.networks {
		if _network.UUID == id {
			return _network.Name, nil
		}
	}
	if name == "" {
		err = errors.New("Network not found")
	}
	return name, err
}

func (rn *raxNetworks) getIP(id string, s *servers.Server) (ip string, err error) {
	_name, err := rn.getName(id)

	if err != nil {
		return "", err
	}

	if s.Addresses[_name] != nil {
		networkAddresses := s.Addresses[_name]
		for _, element := range networkAddresses.([]interface{}) {
			address := element.(map[string]interface{})
			if ok := address["version"].(float64) == 4; ok {
				ip = address["addr"].(string)
			}
		}
	}
	return ip, err
}

func getNetworkList(c *Client) (raxNetworks, error) {
	_networkList, err := getRaxNetworks(c)
	if err != nil {
		return raxNetworks{}, err
	}

	_networkList.networks = append(_networkList.networks, PublicNetwork, PrivateNetwork)
	return _networkList, err
}

func getRaxNetworks(c *Client) (raxNetworks, error) {
	opts := networks.ListOpts{}
	pager := networks.List(c.Network, opts)
	var _raxNetworks raxNetworks

	err := pager.EachPage(func(page pagination.Page) (bool, error) {
		_networks, err := networks.ExtractNetworks(page)
		if err != nil {
			return false, err
		}

		for _, n := range _networks {
			_raxNetworks.networks = append(_raxNetworks.networks, raxNetwork{UUID: n.ID, Name: n.Name})
		}

		return true, nil
	})

	return _raxNetworks, err
}
