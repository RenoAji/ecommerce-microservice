package consulclient

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/consul/api"
)

type ConsulClient struct {
	Client *api.Client
}

func NewConsulClient(address string) (*ConsulClient, error) {
	config := api.DefaultConfig()
	config.Address = address

	client, err := api.NewClient(config)
	if err != nil {
		return nil, err
	}

	return &ConsulClient{Client: client}, nil
}

func (c *ConsulClient) RegisterService(serviceID, serviceName, serviceAddress, port string) error {
	intPort, err := strconv.Atoi(port)
	if err != nil {
		return fmt.Errorf("invalid port number: %w", err)
	}

	serviceAddress = strings.TrimSpace(serviceAddress)
	if serviceAddress == "" {
		serviceAddress = serviceName
	}

    registration := &api.AgentServiceRegistration{
        ID:   serviceID,
        Name: serviceName,
        Address: serviceAddress,
        Port: intPort,
        Check: &api.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("http://%s:8081/api/v1/health", serviceAddress),
            Interval:                       "30s",
            DeregisterCriticalServiceAfter: "1m",
        },
    }

	if err := c.Client.Agent().ServiceRegister(registration); err != nil {
		return fmt.Errorf("failed to register service with Consul: %w", err)
	}

    return nil
}

func (c *ConsulClient) DeregisterService(serviceID string) error {
	return c.Client.Agent().ServiceDeregister(serviceID)
}