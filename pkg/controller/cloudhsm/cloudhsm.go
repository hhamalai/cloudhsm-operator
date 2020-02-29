package cloudhsm

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudhsmv2"
)

type Context struct {
	s  *session.Session
	ch *cloudhsmv2.CloudHSMV2
}

func newContext(s *session.Session) *Context {
	return &Context{
		s: s,
	}
}

func (c *Context) GetHSMIPs(clusterId string) ([]*string, error) {
	if c.s == nil {
		c.s = session.Must(session.NewSession())
	}

	if c.ch == nil {
		c.ch = cloudhsmv2.New(c.s)
	}

	getCloudHSMInput := &cloudhsmv2.DescribeClustersInput{
		Filters: map[string][]*string {
			"clusterIds": aws.StringSlice([]string{clusterId}),
		},
	}


	output, err := c.ch.DescribeClusters(getCloudHSMInput)
	if err != nil {
		return nil, err
	}

	var hsm_ips []*string

	for c := range output.Clusters {
		hsms := output.Clusters[c].Hsms
		for h := range hsms {
			hsm_ips = append(hsm_ips, hsms[h].EniIp)
		}
	}

	fmt.Println("List of HSM IPs", hsm_ips)

	return hsm_ips, nil
}
