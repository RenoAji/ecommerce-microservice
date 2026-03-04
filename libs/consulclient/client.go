package consulclient

import (
	"fmt"

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

func (c *ConsulClient) RegisterService(serviceID, serviceName string, port int) error {
    registration := &api.AgentServiceRegistration{
        ID:   serviceID,
        Name: serviceName,
        Port: port,
        Check: &api.AgentServiceCheck{
            GRPC:                           fmt.Sprintf(":%d", port), 
            Interval:                       "10s",
            DeregisterCriticalServiceAfter: "1m",
        },
    }

    return c.Client.Agent().ServiceRegister(registration)
}

func (c *ConsulClient) DeregisterService(serviceID string) error {
	return c.Client.Agent().ServiceDeregister(serviceID)
}