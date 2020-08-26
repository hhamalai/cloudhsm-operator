/*


Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"errors"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudhsmv2"
	"github.com/aws/aws-sdk-go/service/cloudhsmv2/cloudhsmv2iface"
	g "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	"testing"
	// +kubebuilder:scaffold:imports
)

// These tests use Ginkgo (BDD-style Go testing framework). Refer to
// http://onsi.github.io/ginkgo/ to learn more about Ginkgo.

var cfg *rest.Config
var k8sClient client.Client
var testEnv *envtest.Environment

func TestAPIs(t *testing.T) {
	RegisterFailHandler(g.Fail)
	cases := []struct {
		Description string
		Resp        *cloudhsmv2.DescribeClustersOutput
		Error       error
		Expected    []*string
	}{
		{
			Description: "Lists HSM device IPs",
			Resp: &cloudhsmv2.DescribeClustersOutput{
				Clusters: []*cloudhsmv2.Cluster{
					{
						Hsms: []*cloudhsmv2.Hsm{
							{
								EniIp: aws.String("10.10.10.1"),
								State: aws.String(cloudhsmv2.HsmStateActive),
							},
							{
								EniIp: aws.String("10.10.10.2"),
								State: aws.String(cloudhsmv2.HsmStateActive),
							},
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
			Description: "Lists only active HSM device IPs",
			Resp: &cloudhsmv2.DescribeClustersOutput{
				Clusters: []*cloudhsmv2.Cluster{
					{
						Hsms: []*cloudhsmv2.Hsm{
							{
								EniIp: aws.String("10.10.10.1"),
								State: aws.String(cloudhsmv2.HsmStateActive),
							},
							{
								EniIp: aws.String("10.10.10.2"),
								State: aws.String(cloudhsmv2.HsmStateCreateInProgress),
							},
							{
								EniIp: aws.String("10.10.10.3"),
								State: aws.String(cloudhsmv2.HsmStateDegraded),
							},
						},
					},
				},
			},
			Error: nil,
			Expected: []*string{
				aws.String("10.10.10.1"),
			},
		},
		{
			Description: "Returns empty list when now HSM devices present",
			Resp: &cloudhsmv2.DescribeClustersOutput{
				Clusters: []*cloudhsmv2.Cluster{},
			},
			Error:    nil,
			Expected: []*string{},
		},
		{
			Description: "Errors are returned",
			Error:       errors.New("No results"),
			Expected:    []*string{},
		},
	}

	for i, c := range cases {
		s, err := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})
		q := newContext(s)
		q.Client = &mockedDescribeClusters{Resp: c.Resp, Error: c.Error}

		msgs, err := q.GetHSMIPs("foo")
		if err != nil && c.Error == nil {
			t.Errorf("%d, unexpected error", err)
		}
		if a, e := len(msgs), len(c.Expected); a != e {
			t.Errorf("%d, expected %d messages, got %d", i, e, a)
		}
		for j, msg := range msgs {
			if a, e := msg, c.Expected[j]; aws.StringValue(a) != aws.StringValue(e) {
				t.Errorf("%s: expected %s message, got %s", c.Description, aws.StringValue(e), aws.StringValue(a))
			}
		}
	}
}

type mockedDescribeClusters struct {
	cloudhsmv2iface.CloudHSMV2API
	Resp  *cloudhsmv2.DescribeClustersOutput
	Error error
}

func (m mockedDescribeClusters) DescribeClusters(_ *cloudhsmv2.DescribeClustersInput) (*cloudhsmv2.DescribeClustersOutput, error) {
	if m.Error != nil {
		return nil, m.Error
	}
	return m.Resp, nil
}

var _ = g.AfterSuite(func() {
	g.By("tearing down the test environment")
	err := testEnv.Stop()
	Expect(err).ToNot(HaveOccurred())
})
