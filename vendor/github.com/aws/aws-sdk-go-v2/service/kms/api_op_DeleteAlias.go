// Code generated by smithy-go-codegen DO NOT EDIT.

package kms

import (
	"context"
	"fmt"
	awsmiddleware "github.com/aws/aws-sdk-go-v2/aws/middleware"
	"github.com/aws/aws-sdk-go-v2/aws/signer/v4"
	"github.com/aws/smithy-go/middleware"
	smithyhttp "github.com/aws/smithy-go/transport/http"
)

// Deletes the specified alias. Adding, deleting, or updating an alias can allow
// or deny permission to the KMS key. For details, see ABAC for KMS (https://docs.aws.amazon.com/kms/latest/developerguide/abac.html)
// in the Key Management Service Developer Guide. Because an alias is not a
// property of a KMS key, you can delete and change the aliases of a KMS key
// without affecting the KMS key. Also, aliases do not appear in the response from
// the DescribeKey operation. To get the aliases of all KMS keys, use the
// ListAliases operation. Each KMS key can have multiple aliases. To change the
// alias of a KMS key, use DeleteAlias to delete the current alias and CreateAlias
// to create a new alias. To associate an existing alias with a different KMS key,
// call UpdateAlias . Cross-account use: No. You cannot perform this operation on
// an alias in a different Amazon Web Services account. Required permissions
//   - kms:DeleteAlias (https://docs.aws.amazon.com/kms/latest/developerguide/kms-api-permissions-reference.html)
//     on the alias (IAM policy).
//   - kms:DeleteAlias (https://docs.aws.amazon.com/kms/latest/developerguide/kms-api-permissions-reference.html)
//     on the KMS key (key policy).
//
// For details, see Controlling access to aliases (https://docs.aws.amazon.com/kms/latest/developerguide/kms-alias.html#alias-access)
// in the Key Management Service Developer Guide. Related operations:
//   - CreateAlias
//   - ListAliases
//   - UpdateAlias
func (c *Client) DeleteAlias(ctx context.Context, params *DeleteAliasInput, optFns ...func(*Options)) (*DeleteAliasOutput, error) {
	if params == nil {
		params = &DeleteAliasInput{}
	}

	result, metadata, err := c.invokeOperation(ctx, "DeleteAlias", params, optFns, c.addOperationDeleteAliasMiddlewares)
	if err != nil {
		return nil, err
	}

	out := result.(*DeleteAliasOutput)
	out.ResultMetadata = metadata
	return out, nil
}

type DeleteAliasInput struct {

	// The alias to be deleted. The alias name must begin with alias/ followed by the
	// alias name, such as alias/ExampleAlias .
	//
	// This member is required.
	AliasName *string

	noSmithyDocumentSerde
}

type DeleteAliasOutput struct {
	// Metadata pertaining to the operation's result.
	ResultMetadata middleware.Metadata

	noSmithyDocumentSerde
}

func (c *Client) addOperationDeleteAliasMiddlewares(stack *middleware.Stack, options Options) (err error) {
	if err := stack.Serialize.Add(&setOperationInputMiddleware{}, middleware.After); err != nil {
		return err
	}
	err = stack.Serialize.Add(&awsAwsjson11_serializeOpDeleteAlias{}, middleware.After)
	if err != nil {
		return err
	}
	err = stack.Deserialize.Add(&awsAwsjson11_deserializeOpDeleteAlias{}, middleware.After)
	if err != nil {
		return err
	}
	if err := addProtocolFinalizerMiddlewares(stack, options, "DeleteAlias"); err != nil {
		return fmt.Errorf("add protocol finalizers: %v", err)
	}

	if err = addlegacyEndpointContextSetter(stack, options); err != nil {
		return err
	}
	if err = addSetLoggerMiddleware(stack, options); err != nil {
		return err
	}
	if err = awsmiddleware.AddClientRequestIDMiddleware(stack); err != nil {
		return err
	}
	if err = smithyhttp.AddComputeContentLengthMiddleware(stack); err != nil {
		return err
	}
	if err = addResolveEndpointMiddleware(stack, options); err != nil {
		return err
	}
	if err = v4.AddComputePayloadSHA256Middleware(stack); err != nil {
		return err
	}
	if err = addRetryMiddlewares(stack, options); err != nil {
		return err
	}
	if err = awsmiddleware.AddRawResponseToMetadata(stack); err != nil {
		return err
	}
	if err = awsmiddleware.AddRecordResponseTiming(stack); err != nil {
		return err
	}
	if err = addClientUserAgent(stack, options); err != nil {
		return err
	}
	if err = smithyhttp.AddErrorCloseResponseBodyMiddleware(stack); err != nil {
		return err
	}
	if err = smithyhttp.AddCloseResponseBodyMiddleware(stack); err != nil {
		return err
	}
	if err = addSetLegacyContextSigningOptionsMiddleware(stack); err != nil {
		return err
	}
	if err = addOpDeleteAliasValidationMiddleware(stack); err != nil {
		return err
	}
	if err = stack.Initialize.Add(newServiceMetadataMiddleware_opDeleteAlias(options.Region), middleware.Before); err != nil {
		return err
	}
	if err = awsmiddleware.AddRecursionDetection(stack); err != nil {
		return err
	}
	if err = addRequestIDRetrieverMiddleware(stack); err != nil {
		return err
	}
	if err = addResponseErrorMiddleware(stack); err != nil {
		return err
	}
	if err = addRequestResponseLogging(stack, options); err != nil {
		return err
	}
	if err = addDisableHTTPSMiddleware(stack, options); err != nil {
		return err
	}
	return nil
}

func newServiceMetadataMiddleware_opDeleteAlias(region string) *awsmiddleware.RegisterServiceMetadata {
	return &awsmiddleware.RegisterServiceMetadata{
		Region:        region,
		ServiceID:     ServiceID,
		OperationName: "DeleteAlias",
	}
}
