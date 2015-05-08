package rackspace

import (
	"fmt"

	log "github.com/Sirupsen/logrus"
	"github.com/docker/machine/drivers/openstack"
	"github.com/docker/machine/version"
	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/compute/v2/extensions/keypairs"
	"github.com/rackspace/gophercloud/openstack/compute/v2/servers"

	"github.com/rackspace/gophercloud/rackspace"
)

func unsupportedOpErr(operation string) error {
	return fmt.Errorf("Rackspace does not currently support the %s operation", operation)
}

// Client is a Rackspace specialization of the generic OpenStack driver.
type Client struct {
	openstack.GenericClient

	driver *Driver
}

// Authenticate creates a Rackspace-specific Gophercloud client.
func (c *Client) Authenticate(d *openstack.Driver) error {
	if c.Provider != nil {
		return nil
	}

	log.WithFields(log.Fields{
		"Username": d.Username,
	}).Debug("Authenticating to Rackspace.")

	apiKey := c.driver.APIKey
	opts := gophercloud.AuthOptions{
		Username: d.Username,
		APIKey:   apiKey,
	}

	provider, err := rackspace.NewClient(rackspace.RackspaceUSIdentity)
	if err != nil {
		return err
	}

	provider.UserAgent.Prepend(fmt.Sprintf("docker-machine/v%s", version.VERSION))

	err = rackspace.Authenticate(provider, opts)
	if err != nil {
		return err
	}

	c.Provider = provider

	return nil
}

// StartInstance is unfortunately not supported on Rackspace at this time.
func (c *Client) StartInstance(d *openstack.Driver) error {
	return unsupportedOpErr("start")
}

// StopInstance is unfortunately not support on Rackspace at this time.
func (c *Client) StopInstance(d *openstack.Driver) error {
	return unsupportedOpErr("stop")
}

// GetInstanceIpAddresses can be short-circuited with the server's AccessIPv4Addr on Rackspace.
func (c *Client) GetInstanceIpAddresses(d *openstack.Driver) ([]openstack.IpAddress, error) {
	server, err := c.GetServerDetail(d)
	if err != nil {
		return nil, err
	}

	if err := c.InitNetworkClient(d); err != nil {
		return nil, err
	}

	networkList, err := getNetworkList(c)
	if err != nil {
		return nil, err
	}

	networkName, err := networkList.getName(d.NetworkId)
	if err != nil {
		return nil, err
	}

	ipAddress, err := networkList.getIP(d.NetworkId, server)
	if err != nil {
		return nil, err
	}

	return []openstack.IpAddress{{
		Network:     networkName,
		Address:     ipAddress,
		AddressType: openstack.Fixed,
	}}, nil

}

func (c *Client) InitNetworkClient(d *openstack.Driver) error {
	if c.Network != nil {
		return nil
	}

	network, err := rackspace.NewNetworkV2(c.Provider, gophercloud.EndpointOpts{
		Region:       d.Region,
		Availability: c.getEndpointType(d),
	})
	if err != nil {
		return err
	}
	c.Network = network
	return nil
}

func (c *Client) getEndpointType(d *openstack.Driver) gophercloud.Availability {
	switch d.EndpointType {
	case "internalURL":
		return gophercloud.AvailabilityInternal
	case "adminURL":
		return gophercloud.AvailabilityAdmin
	}
	return gophercloud.AvailabilityPublic
}

func (c *Client) CreateInstance(d *openstack.Driver) (string, error) {
	serverOpts := servers.CreateOpts{
		Name:           d.MachineName,
		FlavorRef:      d.FlavorId,
		ImageRef:       d.ImageId,
		SecurityGroups: d.SecurityGroups,
	}

	if d.NetworkId != "" {
		serverOpts.Networks = c.serverNetworks()
	}

	log.Info("Creating machine...")

	server, err := servers.Create(c.Compute, keypairs.CreateOptsExt{
		serverOpts,
		d.KeyPairName,
	}).Extract()
	if err != nil {
		return "", err
	}
	return server.ID, nil
}

func (c *Client) serverNetworks() []servers.Network {
	var _networks []servers.Network = []servers.Network{}
	_networks = append(_networks,
		servers.Network{UUID: PublicID},
		servers.Network{UUID: PrivateID})
	if c.driver.NetworkId != PublicID && c.driver.NetworkId != PrivateID {
		_networks = append(_networks, servers.Network{UUID: c.driver.NetworkId})
	}
	return _networks

}
