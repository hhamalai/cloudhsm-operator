package controllers

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudhsmv2"
	"github.com/aws/aws-sdk-go/service/cloudhsmv2/cloudhsmv2iface"
)

type Context struct {
	s      *session.Session
	ch     *cloudhsmv2.CloudHSMV2
	Client cloudhsmv2iface.CloudHSMV2API
}

func newContext(s *session.Session) *Context {
	return &Context{
		s: s,
	}
}

func (c *Context) DescribeClusters(clusterId string) (*cloudhsmv2.DescribeClustersOutput, error) {
	if c.Client == nil {
		c.Client = c.ch
	}
	getCloudHSMInput := &cloudhsmv2.DescribeClustersInput{
		Filters: map[string][]*string{
			"clusterIds": aws.StringSlice([]string{clusterId}),
		},
	}

	output, err := c.Client.DescribeClusters(getCloudHSMInput)
	if err != nil {
		return nil, err
	}
	return output, nil
}

func (c *Context) GetHSMIPs(clusterId string) ([]*string, error) {
	if c.s == nil {
		c.s = session.Must(session.NewSession())
	}

	if c.ch == nil {
		c.ch = cloudhsmv2.New(c.s)
	}

	clusters, err := c.DescribeClusters(clusterId)
	if err != nil {
		return nil, err
	}

	var hsmIps []*string
	for c := range clusters.Clusters {
		hsms := clusters.Clusters[c].Hsms
		for _, hsm := range hsms {
			if aws.StringValue(hsm.State) == cloudhsmv2.HsmStateActive {
				hsmIps = append(hsmIps, hsm.EniIp)
			}
		}
	}

	return hsmIps, nil
}
