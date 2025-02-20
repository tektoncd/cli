// Code generated by smithy-go-codegen DO NOT EDIT.

package endpoints

import (
	"github.com/aws/aws-sdk-go-v2/aws"
	endpoints "github.com/aws/aws-sdk-go-v2/internal/endpoints/v2"
	"github.com/aws/smithy-go/logging"
	"regexp"
)

// Options is the endpoint resolver configuration options
type Options struct {
	// Logger is a logging implementation that log events should be sent to.
	Logger logging.Logger

	// LogDeprecated indicates that deprecated endpoints should be logged to the
	// provided logger.
	LogDeprecated bool

	// ResolvedRegion is used to override the region to be resolved, rather then the
	// using the value passed to the ResolveEndpoint method. This value is used by the
	// SDK to translate regions like fips-us-east-1 or us-east-1-fips to an alternative
	// name. You must not set this value directly in your application.
	ResolvedRegion string

	// DisableHTTPS informs the resolver to return an endpoint that does not use the
	// HTTPS scheme.
	DisableHTTPS bool

	// UseDualStackEndpoint specifies the resolver must resolve a dual-stack endpoint.
	UseDualStackEndpoint aws.DualStackEndpointState

	// UseFIPSEndpoint specifies the resolver must resolve a FIPS endpoint.
	UseFIPSEndpoint aws.FIPSEndpointState
}

func (o Options) GetResolvedRegion() string {
	return o.ResolvedRegion
}

func (o Options) GetDisableHTTPS() bool {
	return o.DisableHTTPS
}

func (o Options) GetUseDualStackEndpoint() aws.DualStackEndpointState {
	return o.UseDualStackEndpoint
}

func (o Options) GetUseFIPSEndpoint() aws.FIPSEndpointState {
	return o.UseFIPSEndpoint
}

func transformToSharedOptions(options Options) endpoints.Options {
	return endpoints.Options{
		Logger:               options.Logger,
		LogDeprecated:        options.LogDeprecated,
		ResolvedRegion:       options.ResolvedRegion,
		DisableHTTPS:         options.DisableHTTPS,
		UseDualStackEndpoint: options.UseDualStackEndpoint,
		UseFIPSEndpoint:      options.UseFIPSEndpoint,
	}
}

// Resolver KMS endpoint resolver
type Resolver struct {
	partitions endpoints.Partitions
}

// ResolveEndpoint resolves the service endpoint for the given region and options
func (r *Resolver) ResolveEndpoint(region string, options Options) (endpoint aws.Endpoint, err error) {
	if len(region) == 0 {
		return endpoint, &aws.MissingRegionError{}
	}

	opt := transformToSharedOptions(options)
	return r.partitions.ResolveEndpoint(region, opt)
}

// New returns a new Resolver
func New() *Resolver {
	return &Resolver{
		partitions: defaultPartitions,
	}
}

var partitionRegexp = struct {
	Aws      *regexp.Regexp
	AwsCn    *regexp.Regexp
	AwsIso   *regexp.Regexp
	AwsIsoB  *regexp.Regexp
	AwsIsoE  *regexp.Regexp
	AwsIsoF  *regexp.Regexp
	AwsUsGov *regexp.Regexp
}{

	Aws:      regexp.MustCompile("^(us|eu|ap|sa|ca|me|af|il|mx)\\-\\w+\\-\\d+$"),
	AwsCn:    regexp.MustCompile("^cn\\-\\w+\\-\\d+$"),
	AwsIso:   regexp.MustCompile("^us\\-iso\\-\\w+\\-\\d+$"),
	AwsIsoB:  regexp.MustCompile("^us\\-isob\\-\\w+\\-\\d+$"),
	AwsIsoE:  regexp.MustCompile("^eu\\-isoe\\-\\w+\\-\\d+$"),
	AwsIsoF:  regexp.MustCompile("^us\\-isof\\-\\w+\\-\\d+$"),
	AwsUsGov: regexp.MustCompile("^us\\-gov\\-\\w+\\-\\d+$"),
}

var defaultPartitions = endpoints.Partitions{
	{
		ID: "aws",
		Defaults: map[endpoints.DefaultKey]endpoints.Endpoint{
			{
				Variant: endpoints.DualStackVariant,
			}: {
				Hostname:          "kms.{region}.api.aws",
				Protocols:         []string{"https"},
				SignatureVersions: []string{"v4"},
			},
			{
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname:          "kms-fips.{region}.amazonaws.com",
				Protocols:         []string{"https"},
				SignatureVersions: []string{"v4"},
			},
			{
				Variant: endpoints.FIPSVariant | endpoints.DualStackVariant,
			}: {
				Hostname:          "kms-fips.{region}.api.aws",
				Protocols:         []string{"https"},
				SignatureVersions: []string{"v4"},
			},
			{
				Variant: 0,
			}: {
				Hostname:          "kms.{region}.amazonaws.com",
				Protocols:         []string{"https"},
				SignatureVersions: []string{"v4"},
			},
		},
		RegionRegex:    partitionRegexp.Aws,
		IsRegionalized: true,
		Endpoints: endpoints.Endpoints{
			endpoints.EndpointKey{
				Region: "ProdFips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.eu-central-2.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "eu-central-2",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "af-south-1",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "af-south-1",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.af-south-1.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "af-south-1-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.af-south-1.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "af-south-1",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "ap-east-1",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "ap-east-1",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.ap-east-1.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "ap-east-1-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.ap-east-1.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "ap-east-1",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "ap-northeast-1",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "ap-northeast-1",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.ap-northeast-1.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "ap-northeast-1-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.ap-northeast-1.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "ap-northeast-1",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "ap-northeast-2",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "ap-northeast-2",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.ap-northeast-2.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "ap-northeast-2-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.ap-northeast-2.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "ap-northeast-2",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "ap-northeast-3",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "ap-northeast-3",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.ap-northeast-3.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "ap-northeast-3-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.ap-northeast-3.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "ap-northeast-3",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "ap-south-1",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "ap-south-1",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.ap-south-1.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "ap-south-1-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.ap-south-1.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "ap-south-1",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "ap-south-2",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "ap-south-2",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.ap-south-2.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "ap-south-2-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.ap-south-2.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "ap-south-2",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "ap-southeast-1",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "ap-southeast-1",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.ap-southeast-1.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "ap-southeast-1-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.ap-southeast-1.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "ap-southeast-1",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "ap-southeast-2",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "ap-southeast-2",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.ap-southeast-2.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "ap-southeast-2-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.ap-southeast-2.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "ap-southeast-2",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "ap-southeast-3",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "ap-southeast-3",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.ap-southeast-3.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "ap-southeast-3-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.ap-southeast-3.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "ap-southeast-3",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "ap-southeast-4",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "ap-southeast-4",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.ap-southeast-4.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "ap-southeast-4-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.ap-southeast-4.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "ap-southeast-4",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "ap-southeast-5",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "ap-southeast-5",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.ap-southeast-5.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "ap-southeast-5-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.ap-southeast-5.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "ap-southeast-5",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "ap-southeast-7",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "ap-southeast-7",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.ap-southeast-7.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "ap-southeast-7-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.ap-southeast-7.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "ap-southeast-7",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "ca-central-1",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "ca-central-1",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.ca-central-1.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "ca-central-1-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.ca-central-1.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "ca-central-1",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "ca-west-1",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "ca-west-1",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.ca-west-1.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "ca-west-1-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.ca-west-1.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "ca-west-1",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "eu-central-1",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "eu-central-1",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.eu-central-1.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "eu-central-1-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.eu-central-1.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "eu-central-1",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "eu-central-2",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "eu-central-2",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.eu-central-2.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "eu-central-2-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.eu-central-2.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "eu-central-2",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "eu-north-1",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "eu-north-1",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.eu-north-1.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "eu-north-1-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.eu-north-1.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "eu-north-1",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "eu-south-1",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "eu-south-1",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.eu-south-1.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "eu-south-1-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.eu-south-1.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "eu-south-1",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "eu-south-2",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "eu-south-2",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.eu-south-2.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "eu-south-2-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.eu-south-2.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "eu-south-2",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "eu-west-1",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "eu-west-1",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.eu-west-1.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "eu-west-1-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.eu-west-1.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "eu-west-1",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "eu-west-2",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "eu-west-2",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.eu-west-2.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "eu-west-2-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.eu-west-2.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "eu-west-2",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "eu-west-3",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "eu-west-3",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.eu-west-3.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "eu-west-3-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.eu-west-3.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "eu-west-3",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "il-central-1",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "il-central-1",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.il-central-1.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "il-central-1-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.il-central-1.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "il-central-1",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "me-central-1",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "me-central-1",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.me-central-1.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "me-central-1-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.me-central-1.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "me-central-1",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "me-south-1",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "me-south-1",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.me-south-1.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "me-south-1-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.me-south-1.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "me-south-1",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "mx-central-1",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "mx-central-1",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.mx-central-1.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "mx-central-1-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.mx-central-1.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "mx-central-1",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "sa-east-1",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "sa-east-1",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.sa-east-1.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "sa-east-1-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.sa-east-1.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "sa-east-1",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "us-east-1",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "us-east-1",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.us-east-1.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "us-east-1-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.us-east-1.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "us-east-1",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "us-east-2",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "us-east-2",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.us-east-2.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "us-east-2-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.us-east-2.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "us-east-2",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "us-west-1",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "us-west-1",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.us-west-1.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "us-west-1-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.us-west-1.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "us-west-1",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "us-west-2",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "us-west-2",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.us-west-2.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "us-west-2-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.us-west-2.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "us-west-2",
				},
				Deprecated: aws.TrueTernary,
			},
		},
	},
	{
		ID: "aws-cn",
		Defaults: map[endpoints.DefaultKey]endpoints.Endpoint{
			{
				Variant: endpoints.DualStackVariant,
			}: {
				Hostname:          "kms.{region}.api.amazonwebservices.com.cn",
				Protocols:         []string{"https"},
				SignatureVersions: []string{"v4"},
			},
			{
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname:          "kms-fips.{region}.amazonaws.com.cn",
				Protocols:         []string{"https"},
				SignatureVersions: []string{"v4"},
			},
			{
				Variant: endpoints.FIPSVariant | endpoints.DualStackVariant,
			}: {
				Hostname:          "kms-fips.{region}.api.amazonwebservices.com.cn",
				Protocols:         []string{"https"},
				SignatureVersions: []string{"v4"},
			},
			{
				Variant: 0,
			}: {
				Hostname:          "kms.{region}.amazonaws.com.cn",
				Protocols:         []string{"https"},
				SignatureVersions: []string{"v4"},
			},
		},
		RegionRegex:    partitionRegexp.AwsCn,
		IsRegionalized: true,
		Endpoints: endpoints.Endpoints{
			endpoints.EndpointKey{
				Region: "cn-north-1",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region: "cn-northwest-1",
			}: endpoints.Endpoint{},
		},
	},
	{
		ID: "aws-iso",
		Defaults: map[endpoints.DefaultKey]endpoints.Endpoint{
			{
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname:          "kms-fips.{region}.c2s.ic.gov",
				Protocols:         []string{"https"},
				SignatureVersions: []string{"v4"},
			},
			{
				Variant: 0,
			}: {
				Hostname:          "kms.{region}.c2s.ic.gov",
				Protocols:         []string{"https"},
				SignatureVersions: []string{"v4"},
			},
		},
		RegionRegex:    partitionRegexp.AwsIso,
		IsRegionalized: true,
		Endpoints: endpoints.Endpoints{
			endpoints.EndpointKey{
				Region: "ProdFips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.us-iso-east-1.c2s.ic.gov",
				CredentialScope: endpoints.CredentialScope{
					Region: "us-iso-east-1",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "us-iso-east-1",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "us-iso-east-1",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.us-iso-east-1.c2s.ic.gov",
			},
			endpoints.EndpointKey{
				Region: "us-iso-east-1-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.us-iso-east-1.c2s.ic.gov",
				CredentialScope: endpoints.CredentialScope{
					Region: "us-iso-east-1",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "us-iso-west-1",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "us-iso-west-1",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.us-iso-west-1.c2s.ic.gov",
			},
			endpoints.EndpointKey{
				Region: "us-iso-west-1-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.us-iso-west-1.c2s.ic.gov",
				CredentialScope: endpoints.CredentialScope{
					Region: "us-iso-west-1",
				},
				Deprecated: aws.TrueTernary,
			},
		},
	},
	{
		ID: "aws-iso-b",
		Defaults: map[endpoints.DefaultKey]endpoints.Endpoint{
			{
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname:          "kms-fips.{region}.sc2s.sgov.gov",
				Protocols:         []string{"https"},
				SignatureVersions: []string{"v4"},
			},
			{
				Variant: 0,
			}: {
				Hostname:          "kms.{region}.sc2s.sgov.gov",
				Protocols:         []string{"https"},
				SignatureVersions: []string{"v4"},
			},
		},
		RegionRegex:    partitionRegexp.AwsIsoB,
		IsRegionalized: true,
		Endpoints: endpoints.Endpoints{
			endpoints.EndpointKey{
				Region: "ProdFips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.us-isob-east-1.sc2s.sgov.gov",
				CredentialScope: endpoints.CredentialScope{
					Region: "us-isob-east-1",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "us-isob-east-1",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "us-isob-east-1",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.us-isob-east-1.sc2s.sgov.gov",
			},
			endpoints.EndpointKey{
				Region: "us-isob-east-1-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.us-isob-east-1.sc2s.sgov.gov",
				CredentialScope: endpoints.CredentialScope{
					Region: "us-isob-east-1",
				},
				Deprecated: aws.TrueTernary,
			},
		},
	},
	{
		ID: "aws-iso-e",
		Defaults: map[endpoints.DefaultKey]endpoints.Endpoint{
			{
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname:          "kms-fips.{region}.cloud.adc-e.uk",
				Protocols:         []string{"https"},
				SignatureVersions: []string{"v4"},
			},
			{
				Variant: 0,
			}: {
				Hostname:          "kms.{region}.cloud.adc-e.uk",
				Protocols:         []string{"https"},
				SignatureVersions: []string{"v4"},
			},
		},
		RegionRegex:    partitionRegexp.AwsIsoE,
		IsRegionalized: true,
	},
	{
		ID: "aws-iso-f",
		Defaults: map[endpoints.DefaultKey]endpoints.Endpoint{
			{
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname:          "kms-fips.{region}.csp.hci.ic.gov",
				Protocols:         []string{"https"},
				SignatureVersions: []string{"v4"},
			},
			{
				Variant: 0,
			}: {
				Hostname:          "kms.{region}.csp.hci.ic.gov",
				Protocols:         []string{"https"},
				SignatureVersions: []string{"v4"},
			},
		},
		RegionRegex:    partitionRegexp.AwsIsoF,
		IsRegionalized: true,
		Endpoints: endpoints.Endpoints{
			endpoints.EndpointKey{
				Region: "ProdFips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.us-isof-east-1.csp.hci.ic.gov",
				CredentialScope: endpoints.CredentialScope{
					Region: "us-isof-east-1",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "us-isof-east-1",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "us-isof-east-1",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.us-isof-east-1.csp.hci.ic.gov",
			},
			endpoints.EndpointKey{
				Region: "us-isof-east-1-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.us-isof-east-1.csp.hci.ic.gov",
				CredentialScope: endpoints.CredentialScope{
					Region: "us-isof-east-1",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "us-isof-south-1",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "us-isof-south-1",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.us-isof-south-1.csp.hci.ic.gov",
			},
			endpoints.EndpointKey{
				Region: "us-isof-south-1-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.us-isof-south-1.csp.hci.ic.gov",
				CredentialScope: endpoints.CredentialScope{
					Region: "us-isof-south-1",
				},
				Deprecated: aws.TrueTernary,
			},
		},
	},
	{
		ID: "aws-us-gov",
		Defaults: map[endpoints.DefaultKey]endpoints.Endpoint{
			{
				Variant: endpoints.DualStackVariant,
			}: {
				Hostname:          "kms.{region}.api.aws",
				Protocols:         []string{"https"},
				SignatureVersions: []string{"v4"},
			},
			{
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname:          "kms-fips.{region}.amazonaws.com",
				Protocols:         []string{"https"},
				SignatureVersions: []string{"v4"},
			},
			{
				Variant: endpoints.FIPSVariant | endpoints.DualStackVariant,
			}: {
				Hostname:          "kms-fips.{region}.api.aws",
				Protocols:         []string{"https"},
				SignatureVersions: []string{"v4"},
			},
			{
				Variant: 0,
			}: {
				Hostname:          "kms.{region}.amazonaws.com",
				Protocols:         []string{"https"},
				SignatureVersions: []string{"v4"},
			},
		},
		RegionRegex:    partitionRegexp.AwsUsGov,
		IsRegionalized: true,
		Endpoints: endpoints.Endpoints{
			endpoints.EndpointKey{
				Region: "ProdFips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.us-gov-west-1.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "us-gov-west-1",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "us-gov-east-1",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "us-gov-east-1",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.us-gov-east-1.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "us-gov-east-1-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.us-gov-east-1.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "us-gov-east-1",
				},
				Deprecated: aws.TrueTernary,
			},
			endpoints.EndpointKey{
				Region: "us-gov-west-1",
			}: endpoints.Endpoint{},
			endpoints.EndpointKey{
				Region:  "us-gov-west-1",
				Variant: endpoints.FIPSVariant,
			}: {
				Hostname: "kms-fips.us-gov-west-1.amazonaws.com",
			},
			endpoints.EndpointKey{
				Region: "us-gov-west-1-fips",
			}: endpoints.Endpoint{
				Hostname: "kms-fips.us-gov-west-1.amazonaws.com",
				CredentialScope: endpoints.CredentialScope{
					Region: "us-gov-west-1",
				},
				Deprecated: aws.TrueTernary,
			},
		},
	},
}
