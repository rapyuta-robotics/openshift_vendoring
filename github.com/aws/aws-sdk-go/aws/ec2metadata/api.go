package ec2metadata

import (
	"path"

	"github.com/openshift/github.com/aws/aws-sdk-go/aws/request"
)

// GetMetadata uses the path provided to request
func (c *EC2Metadata) GetMetadata(p string) (string, error) {
	op := &request.Operation{
		Name:       "GetMetadata",
		HTTPMethod: "GET",
		HTTPPath:   path.Join("/", "meta-data", p),
	}

	output := &metadataOutput{}
	req := c.NewRequest(op, nil, output)

	return output.Content, req.Send()
}

// Region returns the region the instance is running in.
func (c *EC2Metadata) Region() (string, error) {
	resp, err := c.GetMetadata("placement/availability-zone")
	if err != nil {
		return "", err
	}

	// returns region without the suffix. Eg: us-west-2a becomes us-west-2
	return resp[:len(resp)-1], nil
}

// Available returns if the application has access to the EC2 Metadata service.
// Can be used to determine if application is running within an EC2 Instance and
// the metadata service is available.
func (c *EC2Metadata) Available() bool {
	if _, err := c.GetMetadata("instance-id"); err != nil {
		return false
	}

	return true
}
