package cloudhsm

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudhsmv2"
	"github.com/aws/aws-sdk-go/service/cloudhsmv2/cloudhsmv2iface"
	"testing"
)


type mockedDescribeClusters struct {
	cloudhsmv2iface.CloudHSMV2API
	Resp *cloudhsmv2.DescribeClustersOutput
	Error error
}

func (m mockedDescribeClusters) DescribeClusters(in *cloudhsmv2.DescribeClustersInput) (*cloudhsmv2.DescribeClustersOutput, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.Resp, nil
}

func TestGetHSMIPs(t *testing.T) {
	cases := []struct {
		Description string
		Resp     *cloudhsmv2.DescribeClustersOutput
		Error error
		Expected []*string
	}{
		{
			Description: "Lists HSM device IPs",
			Resp: &cloudhsmv2.DescribeClustersOutput{
				Clusters: []*cloudhsmv2.Cluster{
					{
						Hsms: []*cloudhsmv2.Hsm{
							{EniIp: aws.String("10.10.10.1")},
							{EniIp: aws.String("10.10.10.2")},
						},
					},
				},
			},
			Error: nil,
			Expected: []*string{
				aws.String("10.10.10.1"),
				aws.String("10.10.10.2"),
			},
		},
		{
			Description: "Returns empty list when now HSM devices present",
			Resp: &cloudhsmv2.DescribeClustersOutput{
				Clusters: []*cloudhsmv2.Cluster{},
			},
			Error: nil,
			Expected: []*string{},
		},
		{
			Description: "Errors are returned",
			Error: errors.New("No results"),
			Expected: []*string{},
		},
	}

	for i, c := range cases {
		s, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")},)
		q := newContext(s)
		q.Client = &mockedDescribeClusters{Resp: c.Resp, Error: c.Error}

		msgs, err := q.GetHSMIPs("foo")
		if err != nil && c.Error == nil {
			t.Fatalf("%d, unexpected error", err)
		}
		if a, e := len(msgs), len(c.Expected); a != e {
			t.Fatalf("%d, expected %d messages, got %d", i, e, a)
		}
		for j, msg := range msgs {
			if a, e := msg, c.Expected[j]; aws.StringValue(a) != aws.StringValue(e) {
				t.Errorf("%d, expected %s message, got %s", i, aws.StringValue(e), aws.StringValue(a))
			}
		}
	}
}