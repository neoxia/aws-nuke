package resources

import (
	"context"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/acm"

	"github.com/ekristen/libnuke/pkg/resource"
	"github.com/ekristen/libnuke/pkg/types"

	"github.com/ekristen/aws-nuke/pkg/nuke"
)

const ACMCertificateResource = "ACMCertificate"

func init() {
	resource.Register(resource.Registration{
		Name:   ACMCertificateResource,
		Scope:  nuke.Account,
		Lister: &ACMCertificateLister{},
	})
}

type ACMCertificateLister struct{}

func (l *ACMCertificateLister) List(_ context.Context, o interface{}) ([]resource.Resource, error) {
	opts := o.(*nuke.ListerOpts)

	svc := acm.New(opts.Session)
	var resources []resource.Resource

	params := &acm.ListCertificatesInput{
		MaxItems: aws.Int64(100),
		Includes: &acm.Filters{
			KeyTypes: aws.StringSlice([]string{
				acm.KeyAlgorithmEcPrime256v1,
				acm.KeyAlgorithmEcSecp384r1,
				acm.KeyAlgorithmEcSecp521r1,
				acm.KeyAlgorithmRsa1024,
				acm.KeyAlgorithmRsa2048,
				acm.KeyAlgorithmRsa4096,
			})},
	}

	for {
		resp, err := svc.ListCertificates(params)
		if err != nil {
			return nil, err
		}

		for _, certificate := range resp.CertificateSummaryList {
			// Unfortunately the ACM API doesn't provide the certificate details when listing, so we
			// have to describe each certificate separately.
			certificateDescribe, err := svc.DescribeCertificate(&acm.DescribeCertificateInput{
				CertificateArn: certificate.CertificateArn,
			})
			if err != nil {
				return nil, err
			}

			tagParams := &acm.ListTagsForCertificateInput{
				CertificateArn: certificate.CertificateArn,
			}

			tagResp, tagErr := svc.ListTagsForCertificate(tagParams)
			if tagErr != nil {
				return nil, tagErr
			}

			resources = append(resources, &ACMCertificate{
				svc:               svc,
				certificateARN:    certificate.CertificateArn,
				certificateDetail: certificateDescribe.Certificate,
				tags:              tagResp.Tags,
			})
		}

		if resp.NextToken == nil {
			break
		}

		params.NextToken = resp.NextToken
	}

	return resources, nil
}

type ACMCertificate struct {
	svc               *acm.ACM
	certificateARN    *string
	certificateDetail *acm.CertificateDetail
	tags              []*acm.Tag
}

func (f *ACMCertificate) Remove(_ context.Context) error {
	_, err := f.svc.DeleteCertificate(&acm.DeleteCertificateInput{
		CertificateArn: f.certificateARN,
	})

	return err
}

func (f *ACMCertificate) Properties() types.Properties {
	properties := types.NewProperties()
	for _, tag := range f.tags {
		properties.SetTag(tag.Key, tag.Value)
	}
	properties.Set("DomainName", f.certificateDetail.DomainName)
	return properties
}

func (f *ACMCertificate) String() string {
	return *f.certificateARN
}
